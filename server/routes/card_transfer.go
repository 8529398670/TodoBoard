package routes

import (
	strconv "strconv"

	fiber "github.com/gofiber/fiber/v3"

	access "todoboard/server/access"
	mcp "todoboard/server/mcp"
	models "todoboard/server/models"
)

// Sending a card to another board, copying it to any board, and the mirror
// links copies can keep (see models/card_link.go). These are for people on
// boards: a share link only ever opens the one board.

type cardTransferRequest struct {
	// The target board: its id or URL slug.
	Board    string   `json:"board"`
	ListID   uint64   `json:"list_id"`
	Top      bool     `json:"top"`
	// An exact place in the list, for putting a card back where it was.
	Position *float64 `json:"position"`
	// Copy only: "", "one_way" or "two_way".
	Mirror   string   `json:"mirror"`
}

// cardTransfer is a send or copy, checked: edit rights on where it lands, and
// for a send, on the whole board it leaves (it leaves the column and the
// board), or for a mirrored copy on the card it mirrors.
func ( handlers *Handlers ) cardTransfer( principal *access.Principal , board *models.Board , card *models.Card , body *cardTransferRequest , send bool ) ( result *models.Card , target *models.Board , refused *refusal ) {
	if principal.Share != nil {
		refused = refuse( fiber.StatusForbidden , "forbidden" )
		return
	}
	target , refused = handlers.loadBoard( principal , body.Board )
	if refused != nil { return }
	list := target.List( body.ListID )
	if list == nil || list.Archived {
		refused = refuse( fiber.StatusNotFound , "list_not_found" )
		return
	}
	if access.List( principal , target , body.ListID ) < access.Edit {
		refused = refuse( fiber.StatusForbidden , "forbidden" )
		return
	}
	place := models.TransferPlace{ BoardID: target.ID , ListID: body.ListID , Top: body.Top , Position: body.Position }
	var model_err error
	if send {
		if target.ID == board.ID || access.Board( principal , board ) < access.Edit {
			refused = refuse( fiber.StatusForbidden , "forbidden" )
			return
		}
		result , model_err = models.SendCard( handlers.Store , board.ID , card.ID , place , principal.Name )
	} else {
		if models.ValidMirrorMode( body.Mirror ) == false {
			refused = refuse( fiber.StatusBadRequest , "invalid_input" )
			return
		}
		if body.Mirror != models.MirrorNone && access.Card( principal , board , card ) < access.Edit {
			refused = refuse( fiber.StatusForbidden , "forbidden" )
			return
		}
		result , model_err = models.CopyCard( handlers.Store , board.ID , card.ID , place , body.Mirror , principal.UserID , principal.Name )
	}
	if model_err != nil {
		refused = modelRefusal( model_err )
		return
	}
	// The board as it now is: labels may have been made on it.
	if fresh , get_err := models.GetBoard( handlers.Store , target.ID ); get_err == nil { target = fresh }
	handlers.activity( principal , target.ID , "card_added" , map[string]string{ "card": result.Title , "list": list.Title } )
	if send {
		handlers.activity( principal , board.ID , "card_sent" , map[string]string{ "card": result.Title , "board": target.Title } )
	}
	return
}

func ( handlers *Handlers ) transferReply( c fiber.Ctx , send bool ) ( err error ) {
	// Seeing the card is enough to copy it; a send is checked in cardTransfer.
	principal , board , card , err := handlers.requireCard( c , access.View )
	if card == nil { return }
	var body cardTransferRequest
	if ok , bind_err := bindBody( c , &body ); ok == false { return bind_err }
	result , target , refused := handlers.cardTransfer( principal , board , card , &body , send )
	if refused != nil {
		err = refused.send( c )
		return
	}
	payload := cardPayload( result , access.Card( principal , target , result ) , access.Board( principal , target ) )
	payload[ "card_path" ] = models.CardPath( target , result )
	payload[ "board_title" ] = target.Title
	err = c.JSON( payload )
	return
}

// SendCard: POST /api/boards/:board_id/cards/:card_id/send.
func ( handlers *Handlers ) SendCard( c fiber.Ctx ) ( err error ) {
	err = handlers.transferReply( c , true )
	return
}

// CopyCard: POST /api/boards/:board_id/cards/:card_id/copy.
func ( handlers *Handlers ) CopyCard( c fiber.Ctx ) ( err error ) {
	err = handlers.transferReply( c , false )
	return
}

// ListCardLinks: the card's mirrors, for anyone who can see the card. The
// other end's board title is shown only when this person can open it.
func ( handlers *Handlers ) ListCardLinks( c fiber.Ctx ) ( err error ) {
	principal , board , card , err := handlers.requireCard( c , access.View )
	if card == nil { return }
	links , err := handlers.cardLinks( principal , board , card )
	if err != nil {
		err = modelError( c , err )
		return
	}
	err = c.JSON( fiber.Map{ "links": links } )
	return
}

func ( handlers *Handlers ) cardLinks( principal *access.Principal , board *models.Board , card *models.Card ) ( links []*models.CardLinkView , err error ) {
	links , err = models.ListCardLinks( handlers.Store , board.ID , card.ID )
	if err != nil { return }
	for _ , link := range links {
		other , refused := handlers.loadBoard( principal , strconv.FormatUint( link.BoardID , 10 ) )
		visible := false
		if refused == nil {
			if found , get_err := models.GetCard( handlers.Store , other.ID , link.CardID ); get_err == nil {
				visible = access.Card( principal , other , found ) > access.None
			}
		}
		if visible == false {
			link.BoardTitle , link.CardTitle , link.ListTitle , link.CardPath = "" , "" , "" , ""
		}
	}
	return
}

// UnlinkCard ends one of the card's mirror links; editing the card is enough.
func ( handlers *Handlers ) UnlinkCard( c fiber.Ctx ) ( err error ) {
	_ , board , card , err := handlers.requireCard( c , access.Edit )
	if card == nil { return }
	link_id , ok := paramID( c , "link_id" )
	if ok == false {
		err = notFound( c , "link_not_found" )
		return
	}
	if unlink_err := models.UnlinkCard( handlers.Store , board.ID , card.ID , link_id ); unlink_err != nil {
		err = modelError( c , unlink_err )
		return
	}
	err = c.JSON( fiber.Map{ "ok": true } )
	return
}

// The same, for agents. Each runs the HTTP route's checks.
func ( handlers *Handlers ) registerTransferTools( server *mcp.Server ) {
	transferSchema := func( copy bool ) ( schema fiber.Map ) {
		properties := fiber.Map{
			"board":    boardRefSchema,
			"card_id":  cardIDSchema,
			"to_board": fiber.Map{ "type": []string{ "integer" , "string" } , "description": "The board it goes to: id or URL slug, from list_boards." },
			"list_id":  fiber.Map{ "type": "integer" , "minimum": 1 , "description": "The column on to_board it goes into, from get_board." },
			"top":      fiber.Map{ "type": "boolean" , "default": false , "description": "Put it at the top of the column instead of the end." },
		}
		if copy {
			properties[ "mirror" ] = fiber.Map{ "type": "string" , "enum": []string{ "" , models.MirrorOneWay , models.MirrorTwoWay } , "default": "" ,
				"description": `"" for a plain copy; "one_way" to keep the copy following the original (the copy's synced fields become read-only); "two_way" to keep both in step. Synced: title, description, checklist, dates, price, picture, To do/Doing/Done, comments.` }
		}
		schema = objectSchema( properties , "board" , "card_id" , "to_board" , "list_id" )
		return
	}
	type transferArgs struct {
		Board   boardRef `json:"board"`
		CardID  uint64   `json:"card_id"`
		ToBoard boardRef `json:"to_board"`
		ListID  uint64   `json:"list_id"`
		Top     bool     `json:"top"`
		Mirror  string   `json:"mirror"`
	}
	run := func( call *mcp.Call , send bool ) ( result any , err error ) {
		var args transferArgs
		if err = call.Bind( &args ); err != nil { return }
		principal , board , card , err := handlers.mcpCard( call , args.Board , args.CardID , access.View )
		if err != nil { return }
		body := &cardTransferRequest{ Board: string( args.ToBoard ) , ListID: args.ListID , Top: args.Top , Mirror: args.Mirror }
		moved , target , refused := handlers.cardTransfer( principal , board , card , body , send )
		if refused != nil {
			err = refusalError( refused )
			return
		}
		payload := cardPayload( moved , access.Card( principal , target , moved ) , access.Board( principal , target ) )
		payload[ "card_path" ] = models.CardPath( target , moved )
		result = payload
		return
	}

	server.Register( &mcp.Tool{
		Name:        "send_card",
		Route:       "POST /api/boards/{board_id}/cards/{card_id}/send",
		Title:       "Send a card to another board",
		Description: "Move a card to another board, whole: it keeps its link, comments and picture. Labels are matched by name there (made if missing), assignees kept only if they are on that board, and per-person shares dropped. Needs edit rights on both boards.",
		InputSchema: transferSchema( false ),
		Call: func( call *mcp.Call ) ( any , error ) { return run( call , true ) },
	} )

	server.Register( &mcp.Tool{
		Name:        "copy_card",
		Route:       "POST /api/boards/{board_id}/cards/{card_id}/copy",
		Title:       "Copy a card to a board",
		Description: "Copy a card onto any board (its own included) with its content, comments and picture, optionally kept in step with the original (mirror). Needs edit rights where the copy goes, and on the card for a mirror.",
		InputSchema: transferSchema( true ),
		Call: func( call *mcp.Call ) ( any , error ) { return run( call , false ) },
	} )

	server.Register( &mcp.Tool{
		Name:        "list_card_links",
		Route:       "GET /api/boards/{board_id}/cards/{card_id}/links",
		Title:       "List a card's mirrors",
		Description: "The cards this one is kept in step with: each link's id, mode (one_way / two_way), direction (out: this card's changes flow there; in: this card follows that one; both), and the other card.",
		ReadOnly:    true,
		InputSchema: objectSchema( fiber.Map{ "board": boardRefSchema , "card_id": cardIDSchema } , "board" , "card_id" ),
		Call: func( call *mcp.Call ) ( result any , err error ) {
			var args struct {
				Board  boardRef `json:"board"`
				CardID uint64   `json:"card_id"`
			}
			if err = call.Bind( &args ); err != nil { return }
			principal , board , card , err := handlers.mcpCard( call , args.Board , args.CardID , access.View )
			if err != nil { return }
			links , list_err := handlers.cardLinks( principal , board , card )
			if list_err != nil {
				err = refusalError( modelRefusal( list_err ) )
				return
			}
			result = fiber.Map{ "links": links }
			return
		},
	} )

	server.Register( &mcp.Tool{
		Name:        "unlink_card",
		Route:       "POST /api/boards/{board_id}/cards/{card_id}/links/{link_id}/delete",
		Title:       "Stop mirroring a card",
		Description: "End one of a card's mirror links (link_id from list_card_links). Both cards stay as they are.",
		InputSchema: objectSchema( fiber.Map{ "board": boardRefSchema , "card_id": cardIDSchema , "link_id": fiber.Map{ "type": "integer" , "minimum": 1 } } , "board" , "card_id" , "link_id" ),
		Call: func( call *mcp.Call ) ( result any , err error ) {
			var args struct {
				Board  boardRef `json:"board"`
				CardID uint64   `json:"card_id"`
				LinkID uint64   `json:"link_id"`
			}
			if err = call.Bind( &args ); err != nil { return }
			_ , board , card , err := handlers.mcpCard( call , args.Board , args.CardID , access.Edit )
			if err != nil { return }
			if unlink_err := models.UnlinkCard( handlers.Store , board.ID , card.ID , args.LinkID ); unlink_err != nil {
				err = refusalError( modelRefusal( unlink_err ) )
				return
			}
			result = fiber.Map{ "ok": true }
			return
		},
	} )
}
