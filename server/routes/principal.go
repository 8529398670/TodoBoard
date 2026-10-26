package routes

import (
	json "encoding/json"
	errors "errors"
	strconv "strconv"
	strings "strings"
	utf8 "unicode/utf8"

	fiber "github.com/gofiber/fiber/v3"

	access "todoboard/server/access"
	db "todoboard/server/db"
	models "todoboard/server/models"
	security "todoboard/server/security"
)

// Board and card routes are written once and mounted twice: under
// /api/boards for signed-in people and under /api/shared for guests holding
// a share link. The only difference between the two is the middleware in
// this file, which turns the request into an access.Principal. After that a
// handler cannot tell -- and does not need to know -- which kind of caller it
// has; server/access decides what each may do.

type principalKey string

const keyPrincipal principalKey = "board_principal"

// ShareTokenHeader carries a share link credential. A custom header rather
// than a cookie means a page on another site cannot make a browser send it,
// so guest writes need no CSRF token.
const ShareTokenHeader = "X-Share-Token"
const GuestNameHeader = "X-Guest-Name"

func principalFrom( c fiber.Ctx ) ( principal *access.Principal ) {
	principal , _ = c.Locals( keyPrincipal ).( *access.Principal )
	return
}

// MemberPrincipal runs after RequireLogin. For anything other than a GET it
// also checks the CSRF token, which every write body carries -- once here
// rather than in each of the handlers below.
func ( handlers *Handlers ) MemberPrincipal( c fiber.Ctx ) ( err error ) {
	actor := security.ActorFrom( c )
	if actor == nil {
		err = c.Status( fiber.StatusUnauthorized ).JSON( fiber.Map{ "error": "not authenticated" } )
		return
	}
	if c.Method() != fiber.MethodGet && c.Method() != fiber.MethodHead {
		var body struct {
			CSRFToken string `json:"csrf_token"`
		}
		json.Unmarshal( c.Body() , &body )
		if handlers.Guard.CheckCSRF( c , body.CSRFToken ) == false {
			err = forbidden( c , "invalid csrf token" )
			return
		}
	}
	c.Locals( keyPrincipal , &access.Principal{ UserID: actor.User.ID , Name: actor.User.DisplayName } )
	err = c.Next()
	return
}

// SharePrincipal resolves the share link in X-Share-Token. A bad, expired,
// or revoked link always answers 404 share_link_invalid.
func ( handlers *Handlers ) SharePrincipal( c fiber.Ctx ) ( err error ) {
	link := models.LoadShareLink( handlers.Store , c.Get( ShareTokenHeader ) )
	if link == nil {
		err = notFound( c , "share_link_invalid" )
		return
	}
	name := strings.TrimSpace( c.Get( GuestNameHeader ) )
	if utf8.RuneCountInString( name ) > 40 || utf8.ValidString( name ) == false { name = "" }
	// Stored with a marker; the UI and push text word it ("Sam (guest)").
	c.Locals( keyPrincipal , &access.Principal{ Share: link , Name: "guest:" + name } )
	err = c.Next()
	return
}

func paramID( c fiber.Ctx , name string ) ( id uint64 , ok bool ) {
	id , err := strconv.ParseUint( c.Params( name ) , 10 , 64 )
	ok = err == nil && id > 0
	return
}

// refusal is a rule saying no, apart from how it is delivered: an HTTP
// handler sends it as a status and an error body, an MCP tool hands it to the
// model as a tool error. The checks below return one so both doors refuse
// for the same reasons in the same words.
type refusal struct {
	status  int
	message string
}

func ( refused *refusal ) send( c fiber.Ctx ) ( err error ) {
	if refused.status >= 500 {
		err = serverError( c )
		return
	}
	err = c.Status( refused.status ).JSON( fiber.Map{ "error": refused.message } )
	return
}

// boardFor loads :board_id and refuses with 404 unless the principal can at
// least see some of it.
func ( handlers *Handlers ) boardFor( c fiber.Ctx ) ( principal *access.Principal , board *models.Board , err error ) {
	principal = principalFrom( c )
	board , refused := handlers.loadBoard( principal , c.Params( "board_id" ) )
	if refused != nil {
		err = refused.send( c )
	}
	return
}

// loadBoard is boardFor without the request: ref is the numeric ID or a
// board's URL slug (never all digits, so the two cannot be confused). Either
// way the access check is the same, so a slug confirms nothing a number
// would not. 404 rather than 403, so a board's existence is not confirmed to
// someone who cannot open it.
func ( handlers *Handlers ) loadBoard( principal *access.Principal , ref string ) ( board *models.Board , refused *refusal ) {
	board_id , parse_err := strconv.ParseUint( ref , 10 , 64 )
	if parse_err != nil || board_id == 0 {
		board_id = models.BoardIDForSlug( handlers.Store , ref )
	}
	if board_id == 0 || principal == nil {
		refused = &refusal{ fiber.StatusNotFound , "board_not_found" }
		return
	}
	loaded , get_err := models.GetBoard( handlers.Store , board_id )
	if get_err != nil {
		refused = &refusal{ fiber.StatusNotFound , "board_not_found" }
		return
	}
	if access.Board( principal , loaded ) == access.None && handlers.canSeeAnyCard( principal , loaded ) == false {
		refused = &refusal{ fiber.StatusNotFound , "board_not_found" }
		return
	}
	board = loaded
	return
}

func ( handlers *Handlers ) canSeeAnyCard( principal *access.Principal , board *models.Board ) ( result bool ) {
	if principal.Share != nil {
		result = principal.Share.BoardID == board.ID && ( principal.Share.CardID != 0 || principal.Share.ListID != 0 )
		return
	}
	found := false
	handlers.Store.ForEachPrefix( db.BucketUserCards , db.Key( principal.UserID , board.ID ) ,
		func() any { return &models.CardRef{} } ,
		func( key []byte , item any ) bool {
			found = true
			return false
		} )
	result = found
	return
}

// requireBoard is boardFor plus a minimum board-wide level.
func ( handlers *Handlers ) requireBoard( c fiber.Ctx , minimum access.Level ) ( principal *access.Principal , board *models.Board , err error ) {
	principal = principalFrom( c )
	board , refused := handlers.loadBoardAt( principal , c.Params( "board_id" ) , minimum )
	if refused != nil {
		err = refused.send( c )
	}
	return
}

func ( handlers *Handlers ) loadBoardAt( principal *access.Principal , ref string , minimum access.Level ) ( board *models.Board , refused *refusal ) {
	board , refused = handlers.loadBoard( principal , ref )
	if refused != nil { return }
	if access.Board( principal , board ) < minimum {
		board = nil
		refused = &refusal{ fiber.StatusForbidden , "forbidden" }
	}
	return
}

// requireCard loads :card_id on an already-loaded board and checks the
// principal's level on that card.
func ( handlers *Handlers ) requireCard( c fiber.Ctx , minimum access.Level ) ( principal *access.Principal , board *models.Board , card *models.Card , err error ) {
	principal , board , err = handlers.boardFor( c )
	if board == nil { return }
	card_id , _ := paramID( c , "card_id" )
	card , refused := handlers.loadCard( principal , board , card_id , minimum )
	if refused != nil {
		board = nil
		err = refused.send( c )
	}
	return
}

// loadCard is requireCard on an already-loaded board, without the request.
func ( handlers *Handlers ) loadCard( principal *access.Principal , board *models.Board , card_id uint64 , minimum access.Level ) ( card *models.Card , refused *refusal ) {
	if card_id == 0 {
		refused = &refusal{ fiber.StatusNotFound , "card_not_found" }
		return
	}
	loaded , get_err := models.GetCard( handlers.Store , board.ID , card_id )
	level := access.Card( principal , board , loaded )
	if get_err != nil || level == access.None {
		refused = &refusal{ fiber.StatusNotFound , "card_not_found" }
		return
	}
	if level < minimum {
		refused = &refusal{ fiber.StatusForbidden , "forbidden" }
		return
	}
	card = loaded
	return
}

// bindBody decodes a JSON body, answering 400 itself when it cannot.
func bindBody( c fiber.Ctx , body any ) ( ok bool , err error ) {
	if c.Bind().JSON( body ) != nil {
		err = badRequest( c , "malformed request body" )
		return
	}
	ok = true
	return
}

// modelError maps the errors models return onto responses, so every handler
// answers the same way for the same failure.
func modelError( c fiber.Ctx , model_err error ) ( err error ) {
	err = modelRefusal( model_err ).send( c )
	return
}

func modelRefusal( model_err error ) ( refused *refusal ) {
	switch {
	case errors.Is( model_err , models.ErrBadInput ):
		refused = &refusal{ fiber.StatusBadRequest , "invalid_input" }
	case errors.Is( model_err , models.ErrStale ):
		refused = &refusal{ fiber.StatusConflict , "card_stale" }
	case errors.Is( model_err , models.ErrBoardNotFound ):
		refused = &refusal{ fiber.StatusNotFound , "board_not_found" }
	case errors.Is( model_err , models.ErrCardNotFound ):
		refused = &refusal{ fiber.StatusNotFound , "card_not_found" }
	case errors.Is( model_err , models.ErrListNotFound ):
		refused = &refusal{ fiber.StatusNotFound , "list_not_found" }
	case errors.Is( model_err , models.ErrLabelNotFound ):
		refused = &refusal{ fiber.StatusNotFound , "label_not_found" }
	case errors.Is( model_err , models.ErrItemNotFound ):
		refused = &refusal{ fiber.StatusNotFound , "item_not_found" }
	case errors.Is( model_err , models.ErrShareLinkNotFound ):
		refused = &refusal{ fiber.StatusNotFound , "share_link_not_found" }
	case errors.Is( model_err , models.ErrFolderNotFound ):
		refused = &refusal{ fiber.StatusNotFound , "folder_not_found" }
	case errors.Is( model_err , db.ErrNotFound ):
		refused = &refusal{ fiber.StatusNotFound , "not_found" }
	default:
		refused = &refusal{ fiber.StatusInternalServerError , "internal error" }
	}
	return
}
