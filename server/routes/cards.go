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
	card , refused := handlers.createCard( principal , board , body.ListID , body.Title , body.Position )
	err = handlers.cardReply( c , principal , board , card , refused )
	if refused == nil { handlers.lookUpProductLater( board , card ) }
	return
}

// createCard is CreateCard's rule, shared with the MCP create_card tool.
func ( handlers *Handlers ) createCard( principal *access.Principal , board *models.Board , list_id uint64 , title string , position *float64 ) ( card *models.Card , refused *refusal ) {
	if board.List( list_id ) == nil {
		refused = &refusal{ fiber.StatusNotFound , "list_not_found" }
		return
	}
	if access.List( principal , board , list_id ) < access.Edit {
		refused = &refusal{ fiber.StatusForbidden , "forbidden" }
		return
	}
	card , create_err := models.CreateCard( handlers.Store , board.ID , list_id , title , position , principal.UserID , principal.Name )
	if create_err != nil {
		refused = modelRefusal( create_err )
	}
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
	updated , current , refused := handlers.patchCard( principal , board , card , &body )
	if refused == nil && ( body.Title != nil || body.Description != nil || body.ListID != nil ) { handlers.lookUpProductLater( board , updated ) }
	if current != nil {
		// Hand back the current card so the client can show what changed
		// without a second request.
		err = c.Status( fiber.StatusConflict ).JSON( fiber.Map{ "error": "card_stale" , "card": cardPayload( current , access.Card( principal , board , current ) , access.Board( principal , board ) ) } )
		return
	}
	err = handlers.cardReply( c , principal , board , updated , refused )
	return
}

// patchCard is PatchCard's rule, shared with the MCP update_card tool. When
// the patch named a rev that is no longer current, current is the card as it
// now stands, alongside the card_stale refusal.
func ( handlers *Handlers ) patchCard( principal *access.Principal , board *models.Board , card *models.Card , patch *models.CardPatch ) ( updated *models.Card , current *models.Card , refused *refusal ) {
	if patch.Moves() {
		leaves := patch.ListID != nil && *patch.ListID != card.ListID
		// Leaving its column needs edit rights where it lands too: a board
		// editor has them everywhere, a column link nowhere else, and a link
		// showing some columns only in those.
		if ( leaves && access.List( principal , board , *patch.ListID ) < access.Edit ) || access.List( principal , board , card.ListID ) < access.Edit {
			refused = &refusal{ fiber.StatusForbidden , "forbidden" }
			return
		}
	}
	updated , change , patch_err := models.PatchCard( handlers.Store , board.ID , card.ID , patch , principal.Name )
	if patch_err != nil {
		refused = modelRefusal( patch_err )
		if patch_err == models.ErrStale {
			current , _ = models.GetCard( handlers.Store , board.ID , card.ID )
		}
		return
	}
	handlers.Notify.CardChanged( board.ID , card.ID , change , principal.UserID , principal.Name )
	return
}

// DeleteCard moves the card to the board's trash (see models/trash.go);
// nothing is gone for good until the trash is emptied.
func ( handlers *Handlers ) DeleteCard( c fiber.Ctx ) ( err error ) {
	principal , board , card , err := handlers.requireCard( c , access.Edit )
	if card == nil { return }
	if access.List( principal , board , card.ListID ) < access.Edit {
		err = forbidden( c , "forbidden" )
		return
	}
	if delete_err := models.TrashCard( handlers.Store , board.ID , card.ID , principal.Name ); delete_err != nil {
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
	updated , refused := handlers.addChecklistItem( principal , card , body.Text )
	err = handlers.cardReply( c , principal , board , updated , refused )
	return
}

// addChecklistItem and updateChecklistItem are shared with the MCP tools of
// the same name. The card has already passed loadCard at Edit.
func ( handlers *Handlers ) addChecklistItem( principal *access.Principal , card *models.Card , text string ) ( updated *models.Card , refused *refusal ) {
	updated , mutate_err := models.MutateCard( handlers.Store , card.BoardID , card.ID , principal.Name , func( _ *models.Board , target *models.Card ) ( inner error ) {
		_ , inner = models.AddChecklistItem( target , text )
		return
	} )
	if mutate_err != nil { refused = modelRefusal( mutate_err ) }
	return
}

func ( handlers *Handlers ) updateChecklistItem( principal *access.Principal , card *models.Card , item_id uint64 , update *models.ChecklistUpdate ) ( updated *models.Card , refused *refusal ) {
	updated , mutate_err := models.MutateCard( handlers.Store , card.BoardID , card.ID , principal.Name , func( _ *models.Board , target *models.Card ) error {
		return update.Apply( target , item_id )
	} )
	if mutate_err != nil { refused = modelRefusal( mutate_err ) }
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
	updated , refused := handlers.updateChecklistItem( principal , card , item_id , &body )
	err = handlers.cardReply( c , principal , board , updated , refused )
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

// cardReply is cardResult for the helpers above, which have already turned
// a model error into a refusal.
func ( handlers *Handlers ) cardReply( c fiber.Ctx , principal *access.Principal , board *models.Board , card *models.Card , refused *refusal ) ( err error ) {
	if refused != nil {
		err = refused.send( c )
		return
	}
	err = c.JSON( cardPayload( card , access.Card( principal , board , card ) , access.Board( principal , board ) ) )
	return
}

// The trash is the board's, not one column's: listing, restoring and purging
// need edit rights on the whole board.
func ( handlers *Handlers ) ListTrash( c fiber.Ctx ) ( err error ) {
	principal , board , err := handlers.requireBoard( c , access.Edit )
	if board == nil { return }
	cards , list_err := models.ListTrash( handlers.Store , board.ID )
	if list_err != nil {
		err = modelError( c , list_err )
		return
	}
	level := access.Board( principal , board )
	payload := make( []fiber.Map , 0 , len( cards ) )
	for _ , card := range cards {
		item := cardPayload( card , level , level )
		item[ "deleted_at" ] , item[ "deleted_by" ] = card.DeletedAt , card.DeletedBy
		payload = append( payload , item )
	}
	err = c.JSON( fiber.Map{ "cards": payload } )
	return
}

func ( handlers *Handlers ) RestoreCard( c fiber.Ctx ) ( err error ) {
	principal , board , err := handlers.requireBoard( c , access.Edit )
	if board == nil { return }
	card_id , ok := paramID( c , "card_id" )
	if ok == false {
		err = notFound( c , "card_not_found" )
		return
	}
	card , restore_err := models.RestoreCard( handlers.Store , board.ID , card_id , principal.Name )
	err = handlers.cardResult( c , principal , board , card , restore_err )
	return
}

func ( handlers *Handlers ) PurgeCard( c fiber.Ctx ) ( err error ) {
	_ , board , err := handlers.requireBoard( c , access.Edit )
	if board == nil { return }
	card_id , ok := paramID( c , "card_id" )
	if ok == false {
		err = notFound( c , "card_not_found" )
		return
	}
	if purge_err := models.PurgeCard( handlers.Store , board.ID , card_id ); purge_err != nil {
		err = modelError( c , purge_err )
		return
	}
	err = c.JSON( fiber.Map{ "ok": true } )
	return
}

func ( handlers *Handlers ) EmptyTrash( c fiber.Ctx ) ( err error ) {
	_ , board , err := handlers.requireBoard( c , access.Edit )
	if board == nil { return }
	removed , empty_err := models.EmptyTrash( handlers.Store , board.ID )
	if empty_err != nil {
		err = modelError( c , empty_err )
		return
	}
	err = c.JSON( fiber.Map{ "removed": removed } )
	return
}
