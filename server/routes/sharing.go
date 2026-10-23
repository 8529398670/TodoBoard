package routes

import (
	fiber "github.com/gofiber/fiber/v3"

	access "todoboard/server/access"
	models "todoboard/server/models"
)

// Sharing is the board owner's business (access.Manage): who is on the
// board, which single cards are shared with whom, and the share links.

type memberRequest struct {
	UserID uint64 `json:"user_id"`
	Role   string `json:"role"` // editor | viewer | "" to remove
}

type cardShareRequest struct {
	UserID uint64 `json:"user_id"`
	Mode   string `json:"mode"` // view | edit | "" to remove
}

type shareLinkRequest struct {
	CardID uint64 `json:"card_id"`
	Mode   string `json:"mode"`
	Label  string `json:"label"`
	Days   int    `json:"days"`
}

func ( handlers *Handlers ) SetBoardMember( c fiber.Ctx ) ( err error ) {
	principal , board , err := handlers.requireBoard( c , access.Manage )
	if board == nil { return }
	var body memberRequest
	if ok , bind_err := bindBody( c , &body ); ok == false { return bind_err }
	if body.UserID == board.OwnerID {
		err = badRequest( c , "invalid_input" )
		return
	}
	was_member := board.RoleOf( body.UserID ) != ""
	if target , get_err := models.GetUser( handlers.Store , body.UserID ); get_err != nil || target.Disabled() {
		if body.Role != "" {
			err = notFound( c , "user_not_found" )
			return
		}
	}
	if _ , set_err := models.SetMember( handlers.Store , board.ID , body.UserID , body.Role ); set_err != nil {
		err = modelError( c , set_err )
		return
	}
	if was_member == false && body.Role != "" {
		handlers.Notify.BoardShared( board.ID , body.UserID , principal.UserID , principal.Name )
	}
	err = c.JSON( fiber.Map{ "ok": true } )
	return
}

func ( handlers *Handlers ) ShareCardWithUser( c fiber.Ctx ) ( err error ) {
	principal , board , err := handlers.requireBoard( c , access.Manage )
	if board == nil { return }
	card_id , ok := paramID( c , "card_id" )
	if ok == false {
		err = notFound( c , "card_not_found" )
		return
	}
	var body cardShareRequest
	if ok , bind_err := bindBody( c , &body ); ok == false { return bind_err }
	before , get_err := models.GetCard( handlers.Store , board.ID , card_id )
	if get_err != nil {
		err = modelError( c , get_err )
		return
	}
	if target , user_err := models.GetUser( handlers.Store , body.UserID ); user_err != nil || ( target.Disabled() && body.Mode != "" ) {
		err = notFound( c , "user_not_found" )
		return
	}
	card , share_err := models.ShareCardWithUser( handlers.Store , board.ID , card_id , body.UserID , body.Mode , principal.Name )
	if share_err != nil {
		err = modelError( c , share_err )
		return
	}
	if body.Mode != "" && before.SharedWith[ body.UserID ] == "" && board.RoleOf( body.UserID ) == "" {
		handlers.Notify.CardShared( board.ID , card.ID , body.UserID , principal.UserID , principal.Name )
	}
	err = c.JSON( cardPayload( card , access.Card( principal , board , card ) , access.Manage ) )
	return
}

// shareLinkPayload never includes the secret: it is shown once, when the
// link is created, and only its hash is kept.
func shareLinkPayload( link *models.ShareLink ) ( payload fiber.Map ) {
	payload = fiber.Map{
		"id":           link.ID,
		"card_id":      link.CardID,
		"mode":         link.Mode,
		"label":        link.Label,
		"created_at":   link.CreatedAt,
		"expires_at":   link.ExpiresAt,
		"revoked_at":   link.RevokedAt,
		"last_used_at": link.LastUsedAt,
		"live":         link.Live(),
	}
	return
}

func ( handlers *Handlers ) ListShareLinks( c fiber.Ctx ) ( err error ) {
	_ , board , err := handlers.requireBoard( c , access.Manage )
	if board == nil { return }
	links , list_err := models.ListShareLinks( handlers.Store , board.ID )
	if list_err != nil {
		err = serverError( c )
		return
	}
	result := []fiber.Map{}
	for _ , link := range links { result = append( result , shareLinkPayload( link ) ) }
	err = c.JSON( result )
	return
}

// CreateShareLink mints a credential, so -- like creating a user or an API
// key -- it needs a browser session. A leaked API key cannot hand the board
// out to the world.
func ( handlers *Handlers ) CreateShareLink( c fiber.Ctx ) ( err error ) {
	principal , board , err := handlers.requireBoard( c , access.Manage )
	if board == nil { return }
	var body shareLinkRequest
	if ok , bind_err := bindBody( c , &body ); ok == false { return bind_err }
	credential , link , issue_err := models.IssueShareLink( handlers.Store , board.ID , body.CardID , body.Mode , body.Label , body.Days , principal.UserID )
	if issue_err != nil {
		err = modelError( c , issue_err )
		return
	}
	payload := shareLinkPayload( link )
	payload[ "share_path" ] = "/s/" + credential
	err = c.JSON( payload )
	return
}

func ( handlers *Handlers ) shareLinkOnBoard( c fiber.Ctx , board *models.Board ) ( link *models.ShareLink , err error ) {
	found , get_err := models.GetShareLink( handlers.Store , c.Params( "link_id" ) )
	if get_err != nil || found.BoardID != board.ID {
		err = notFound( c , "share_link_not_found" )
		return
	}
	link = found
	return
}

func ( handlers *Handlers ) RevokeShareLink( c fiber.Ctx ) ( err error ) {
	_ , board , err := handlers.requireBoard( c , access.Manage )
	if board == nil { return }
	link , err := handlers.shareLinkOnBoard( c , board )
	if link == nil { return }
	if revoke_err := models.RevokeShareLink( handlers.Store , link.ID ); revoke_err != nil {
		err = modelError( c , revoke_err )
		return
	}
	err = c.JSON( fiber.Map{ "ok": true } )
	return
}

func ( handlers *Handlers ) DeleteShareLink( c fiber.Ctx ) ( err error ) {
	_ , board , err := handlers.requireBoard( c , access.Manage )
	if board == nil { return }
	link , err := handlers.shareLinkOnBoard( c , board )
	if link == nil { return }
	if delete_err := models.DeleteShareLink( handlers.Store , link.ID ); delete_err != nil {
		err = modelError( c , delete_err )
		return
	}
	err = c.JSON( fiber.Map{ "ok": true } )
	return
}

// SharedInfo tells a guest page what its link opens. It is the first call a
// share page makes.
func ( handlers *Handlers ) SharedInfo( c fiber.Ctx ) ( err error ) {
	principal := principalFrom( c )
	link := principal.Share
	board , get_err := models.GetBoard( handlers.Store , link.BoardID )
	if get_err != nil {
		err = notFound( c , "share_link_invalid" )
		return
	}
	err = c.JSON( fiber.Map{
		"board_id":    link.BoardID,
		"board_title": board.Title,
		"card_id":     link.CardID,
		"mode":        link.Mode,
		"label":       link.Label,
	} )
	return
}

// ShowSharePage answers /s/<credential> with the guest page. Like a login
// link, the credential is in the URL, so no request the page makes may carry
// it onward in a Referer header.
func ( handlers *Handlers ) ShowSharePage( c fiber.Ctx ) ( err error ) {
	c.Set( fiber.HeaderReferrerPolicy , "no-referrer" )
	err = handlers.Static.ServeFile( c , "board.html" )
	return
}

func ( handlers *Handlers ) ShowBoardPage( c fiber.Ctx ) ( err error ) {
	err = handlers.Static.ServeFile( c , "board.html" )
	return
}

func ( handlers *Handlers ) ShowPrintPage( c fiber.Ctx ) ( err error ) {
	c.Set( fiber.HeaderReferrerPolicy , "no-referrer" )
	err = handlers.Static.ServeFile( c , "print.html" )
	return
}
