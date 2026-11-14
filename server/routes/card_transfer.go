package routes

import (
	strconv "strconv"

	fiber "github.com/gofiber/fiber/v3"

	access "todoboard/server/access"
	models "todoboard/server/models"
)

// Sending a card to another board, copying it to any board, and the mirror
// links copies can keep (see models/card_link.go). These are for people on
// boards: a share link only ever opens the one board.

type cardTransferRequest struct {
	cardPath
	// The target board: its id or URL slug.
	ToBoard  boardRef `json:"to_board" jsonschema_description:"The board it goes to: id or URL slug, from list_boards."`
	ListID   uint64   `json:"list_id" jsonschema_description:"The column on that board it goes into, from get_board (board.lists)."`
	Top      bool     `json:"top,omitempty" jsonschema_description:"Put it at the top of the column instead of the end."`
	// An exact place in the list, for putting a card back where it was.
	Position *float64 `json:"position,omitempty" jsonschema_description:"An exact place in the column, for putting a card back where it was; wins over top."`
	// Copy only: "", "one_way" or "two_way".
	Mirror   string   `json:"mirror,omitempty" jsonschema:"enum=,enum=one_way,enum=two_way" jsonschema_description:"Copy only. Empty for a plain copy; one_way keeps the copy following the original (the copy's synced fields become read-only); two_way keeps both in step. Synced: title, description, checklist, dates, price, picture, To do/Doing/Done, comments."`
}

type cardLinkPath struct {
	cardPath
	LinkID uint64 `json:"link_id" path:"link_id" jsonschema_description:"A mirror link's id, from list_card_links."`
}

type cardLinksReply struct {
	Links []*models.CardLinkView `json:"links"`
}

// cardTransfer is a send or copy, checked: edit rights on where it lands, and
// for a send, on the whole board it leaves (it leaves the column and the
// board), or for a mirrored copy on the card it mirrors.
func ( handlers *Handlers ) cardTransfer( principal *access.Principal , board *models.Board , card *models.Card , body *cardTransferRequest , send bool ) ( result *models.Card , target *models.Board , refused *refusal ) {
	if principal.Share != nil {
		refused = refuse( fiber.StatusForbidden , "forbidden" )
		return
	}
	target , refused = handlers.loadBoard( principal , string( body.ToBoard ) )
	if refused != nil { return }
	list := target.List( body.ListID )
	if list == nil || list.Archived {
		refused = refuse( fiber.StatusNotFound , "list_not_found" )
		return
	}
	if access.List( principal , target , body.ListID ) < access.Edit {
		refused = writeRefusal( target , access.Edit )
		return
	}
	place := models.TransferPlace{ BoardID: target.ID , ListID: body.ListID , Top: body.Top , Position: body.Position }
	var model_err error
	if send {
		if target.ID == board.ID {
			refused = refuse( fiber.StatusForbidden , "forbidden" )
			return
		}
		if access.Board( principal , board ) < access.Edit {
			refused = writeRefusal( board , access.Edit )
			return
		}
		result , model_err = models.SendCard( handlers.Store , board.ID , card.ID , place , principal.Name )
	} else {
		if models.ValidMirrorMode( body.Mirror ) == false {
			refused = refuse( fiber.StatusBadRequest , "invalid_input" )
			return
		}
		if body.Mirror != models.MirrorNone && access.Card( principal , board , card ) < access.Edit {
			refused = writeRefusal( board , access.Edit )
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

func ( handlers *Handlers ) transferReply( c fiber.Ctx , in *cardTransferRequest , send bool ) ( reply *fiber.Map , err error ) {
	// Seeing the card is enough to copy it; a send is checked in cardTransfer.
	principal , board , card , err := handlers.requireCardAt( c , &in.cardPath , access.View )
	if err != nil { return }
	result , target , refused := handlers.cardTransfer( principal , board , card , in , send )
	if refused != nil {
		err = refused
		return
	}
	payload := cardPayload( result , access.Card( principal , target , result ) , access.Board( principal , target ) )
	payload[ "card_path" ] = models.CardPath( target , result )
	payload[ "board_title" ] = target.Title
	reply = &payload
	return
}

// sendCard moves a card to another board.
//
// @route("POST /api/boards/{board_id}/cards/{card_id}/send", summary="Send a card to another board",
//     tag="Cards", errors=(400, 403, 404),
//     description="Move a card to another board, whole: it keeps its link, comments and picture. Labels "
//         "are matched by name there (made if missing), assignees kept only if they are on that board, and "
//         "per-person shares dropped. Needs edit rights on both boards. Answers the card where it landed, "
//         "with card_path and board_title.",
//     mcp="send_card",
// )
func ( handlers *Handlers ) sendCard( c fiber.Ctx , in *cardTransferRequest ) ( reply *fiber.Map , err error ) {
	reply , err = handlers.transferReply( c , in , true )
	return
}

// copyCard copies a card onto any board, optionally mirrored.
//
// @route("POST /api/boards/{board_id}/cards/{card_id}/copy", summary="Copy a card to a board",
//     tag="Cards", errors=(400, 403, 404),
//     description="Copy a card onto any board (its own included) with its content, comments and picture, "
//         "optionally kept in step with the original (mirror). Needs edit rights where the copy goes, and "
//         "on the card for a mirror. Answers the copy, with card_path and board_title.",
//     mcp="copy_card",
// )
func ( handlers *Handlers ) copyCard( c fiber.Ctx , in *cardTransferRequest ) ( reply *fiber.Map , err error ) {
	reply , err = handlers.transferReply( c , in , false )
	return
}

// listCardLinks: the card's mirrors, for anyone who can see the card. The
// other end's board title is shown only when this person can open it.
//
// @route("GET /api/boards/{board_id}/cards/{card_id}/links", summary="List a card's mirrors",
//     tag="Cards",
//     description="The cards this one is kept in step with: each link's id, mode (one_way / two_way), "
//         "direction (out: this card's changes flow there; in: this card follows that one; both), and the "
//         "other card, named only when you can open it.",
//     mcp="list_card_links", read_only=True,
// )
func ( handlers *Handlers ) listCardLinks( c fiber.Ctx , in *cardPath ) ( reply *cardLinksReply , err error ) {
	principal , board , card , err := handlers.requireCardAt( c , in , access.View )
	if err != nil { return }
	links , list_err := handlers.cardLinks( principal , board , card )
	if list_err != nil {
		err = modelErr( list_err )
		return
	}
	reply = &cardLinksReply{ Links: links }
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

// unlinkCard ends one of the card's mirror links; editing the card is enough.
//
// @route("POST /api/boards/{board_id}/cards/{card_id}/links/{link_id}/delete", summary="Stop mirroring a card",
//     tag="Cards", errors=(404,),
//     description="End one of a card's mirror links (link_id from list_card_links). Both cards stay as "
//         "they are. Needs edit rights on the card.",
//     mcp="unlink_card", destructive=True,
// )
func ( handlers *Handlers ) unlinkCard( c fiber.Ctx , in *cardLinkPath ) ( reply *okReply , err error ) {
	_ , board , card , err := handlers.requireCardAt( c , &in.cardPath , access.Edit )
	if err != nil { return }
	if in.LinkID == 0 {
		err = refuse( fiber.StatusNotFound , "link_not_found" )
		return
	}
	if unlink_err := models.UnlinkCard( handlers.Store , board.ID , card.ID , in.LinkID ); unlink_err != nil {
		err = modelErr( unlink_err )
		return
	}
	reply = &okReply{ Ok: true }
	return
}
