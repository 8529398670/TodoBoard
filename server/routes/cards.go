package routes

import (
	fiber "github.com/gofiber/fiber/v3"

	access "todoboard/server/access"
	models "todoboard/server/models"
)

type createCardRequest struct {
	ListID   uint64   `json:"list_id"`
	Title    string   `json:"title"`
	Position *float64 `json:"position"`
}

type checklistAddRequest struct {
	Text string `json:"text"`
}

// CreateCard needs edit rights on the column the card goes into: a board
// editor has them everywhere, an edit link for one column only there.
func ( handlers *Handlers ) CreateCard( c fiber.Ctx ) ( err error ) {
	principal , board , err := handlers.boardFor( c )
	if board == nil { return }
	var body createCardRequest
	if ok , bind_err := bindBody( c , &body ); ok == false { return bind_err }
	if board.List( body.ListID ) == nil {
		err = notFound( c , "list_not_found" )
		return
	}
	if access.List( principal , board , body.ListID ) < access.Edit {
		err = forbidden( c , "forbidden" )
		return
	}
	card , create_err := models.CreateCard( handlers.Store , board.ID , body.ListID , body.Title , body.Position , principal.UserID , principal.Name )
	if create_err != nil {
		err = modelError( c , create_err )
		return
	}
	err = c.JSON( cardPayload( card , access.Card( principal , board , card ) , access.Board( principal , board ) ) )
	return
}

// PatchCard edits a card. Editing its fields needs edit rights on the card;
// reordering or archiving it changes the column around it, so that needs
// edit rights on the column, and moving it to another column needs them on
// the board -- someone with one shared card or column cannot rearrange the
// rest.
func ( handlers *Handlers ) PatchCard( c fiber.Ctx ) ( err error ) {
	principal , board , card , err := handlers.requireCard( c , access.Edit )
	if card == nil { return }
	var body models.CardPatch
	if ok , bind_err := bindBody( c , &body ); ok == false { return bind_err }
	if body.Moves() {
		leaves := body.ListID != nil && *body.ListID != card.ListID
		if ( leaves && access.Board( principal , board ) < access.Edit ) || access.List( principal , board , card.ListID ) < access.Edit {
			err = forbidden( c , "forbidden" )
			return
		}
	}
	updated , change , patch_err := models.PatchCard( handlers.Store , board.ID , card.ID , &body , principal.Name )
	if patch_err != nil {
		if patch_err == models.ErrStale {
			// Hand back the current card so the client can show what changed
			// without a second request.
			current , _ := models.GetCard( handlers.Store , board.ID , card.ID )
			if current != nil {
				err = c.Status( fiber.StatusConflict ).JSON( fiber.Map{ "error": "card_stale" , "card": cardPayload( current , access.Card( principal , board , current ) , access.Board( principal , board ) ) } )
				return
			}
		}
		err = modelError( c , patch_err )
		return
	}
	handlers.Notify.CardChanged( board.ID , card.ID , change , principal.UserID , principal.Name )
	err = c.JSON( cardPayload( updated , access.Card( principal , board , updated ) , access.Board( principal , board ) ) )
	return
}

func ( handlers *Handlers ) DeleteCard( c fiber.Ctx ) ( err error ) {
	principal , board , card , err := handlers.requireCard( c , access.Edit )
	if card == nil { return }
	if access.List( principal , board , card.ListID ) < access.Edit {
		err = forbidden( c , "forbidden" )
		return
	}
	if delete_err := models.DeleteCard( handlers.Store , board.ID , card.ID ); delete_err != nil {
		err = modelError( c , delete_err )
		return
	}
	err = c.JSON( fiber.Map{ "ok": true } )
	return
}

// archiveCardsRequest names the cards to archive; without card_ids, every
// open card in the column goes.
type archiveCardsRequest struct {
	CardIDs []uint64 `json:"card_ids"`
}

// ArchiveListCards archives a column's open cards, all or the ones picked --
// clearing out a Done column, say. Edit rights on that column are enough.
func ( handlers *Handlers ) ArchiveListCards( c fiber.Ctx ) ( err error ) {
	principal , board , err := handlers.boardFor( c )
	if board == nil { return }
	var body archiveCardsRequest
	if ok , bind_err := bindBody( c , &body ); ok == false { return bind_err }
	list_id , ok := paramID( c , "list_id" )
	if ok == false || board.List( list_id ) == nil {
		err = notFound( c , "list_not_found" )
		return
	}
	if access.List( principal , board , list_id ) < access.Edit {
		err = forbidden( c , "forbidden" )
		return
	}
	ids , archive_err := models.ArchiveListCards( handlers.Store , board.ID , list_id , body.CardIDs , principal.Name )
	if archive_err != nil {
		err = modelError( c , archive_err )
		return
	}
	err = c.JSON( fiber.Map{ "archived": ids } )
	return
}

func ( handlers *Handlers ) AddChecklistItem( c fiber.Ctx ) ( err error ) {
	principal , board , card , err := handlers.requireCard( c , access.Edit )
	if card == nil { return }
	var body checklistAddRequest
	if ok , bind_err := bindBody( c , &body ); ok == false { return bind_err }
	updated , mutate_err := models.MutateCard( handlers.Store , board.ID , card.ID , principal.Name , func( _ *models.Board , target *models.Card ) ( inner error ) {
		_ , inner = models.AddChecklistItem( target , body.Text )
		return
	} )
	err = handlers.cardResult( c , principal , board , updated , mutate_err )
	return
}

func ( handlers *Handlers ) UpdateChecklistItem( c fiber.Ctx ) ( err error ) {
	principal , board , card , err := handlers.requireCard( c , access.Edit )
	if card == nil { return }
	item_id , ok := paramID( c , "item_id" )
	if ok == false {
		err = notFound( c , "item_not_found" )
		return
	}
	var body models.ChecklistUpdate
	if ok , bind_err := bindBody( c , &body ); ok == false { return bind_err }
	updated , mutate_err := models.MutateCard( handlers.Store , board.ID , card.ID , principal.Name , func( _ *models.Board , target *models.Card ) error {
		return body.Apply( target , item_id )
	} )
	err = handlers.cardResult( c , principal , board , updated , mutate_err )
	return
}

func ( handlers *Handlers ) DeleteChecklistItem( c fiber.Ctx ) ( err error ) {
	principal , board , card , err := handlers.requireCard( c , access.Edit )
	if card == nil { return }
	item_id , ok := paramID( c , "item_id" )
	if ok == false {
		err = notFound( c , "item_not_found" )
		return
	}
	updated , mutate_err := models.MutateCard( handlers.Store , board.ID , card.ID , principal.Name , func( _ *models.Board , target *models.Card ) error {
		return models.DeleteChecklistItem( target , item_id )
	} )
	err = handlers.cardResult( c , principal , board , updated , mutate_err )
	return
}

func ( handlers *Handlers ) cardResult( c fiber.Ctx , principal *access.Principal , board *models.Board , card *models.Card , model_err error ) ( err error ) {
	if model_err != nil {
		err = modelError( c , model_err )
		return
	}
	err = c.JSON( cardPayload( card , access.Card( principal , board , card ) , access.Board( principal , board ) ) )
	return
}
