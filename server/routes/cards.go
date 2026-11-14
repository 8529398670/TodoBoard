package routes

import (
	strconv "strconv"

	fiber "github.com/gofiber/fiber/v3"

	access "todoboard/server/access"
	models "todoboard/server/models"
)

type createCardRequest struct {
	boardPath
	ListID   uint64   `json:"list_id" jsonschema_description:"The column's id, from get_board (board.lists)."`
	Title    string   `json:"title"`
	Position *float64 `json:"position,omitempty" jsonschema_description:"Sort position within the column; lower comes first. Leave out to put it at the end."`
	// The rest are set on the new card in the same call.
	Description *string   `json:"description,omitempty" jsonschema_description:"Markdown."`
	LabelIDs    *[]uint64 `json:"label_ids,omitempty" jsonschema_description:"Label ids, from get_board (board.labels)."`
	Assignees   *[]uint64 `json:"assignees,omitempty" jsonschema_description:"User ids, from get_board's members or list_team."`
	StartAt     *string   `json:"start_at,omitempty" jsonschema_description:"2026-10-30 for an all-day date, an RFC 3339 timestamp for a time, an empty string to clear."`
	DueAt       *string   `json:"due_at,omitempty" jsonschema_description:"2026-10-30 for an all-day date, an RFC 3339 timestamp for a time, an empty string to clear."`
	Done        *bool     `json:"done,omitempty" jsonschema_description:"true makes it complete from the start."`
	// The stage it starts in, on a staged board.
	Doing     *bool `json:"doing,omitempty" jsonschema_description:"On a staged board, the Doing stage. true starts the card (reopening it if done); done: true moves it on to Done."`
	RankPlace *int  `json:"rank_place,omitempty" jsonschema:"minimum=0" jsonschema_description:"Importance across the whole board: 1 is the most important, and other cards shift to make room. 0 takes the card out of the ranking. Needs edit rights on the board."`
	// The day it is filed under, on a calendar board.
	Day *string `json:"day,omitempty" jsonschema_description:"On a calendar board (layout calendar), the day the card is filed under: 2026-11-07, or an empty string for none."`
}

// patch is the fields set after the card is made, the zero patch when none
// are.
func ( in *createCardRequest ) patch() ( patch models.CardPatch ) {
	patch = models.CardPatch{ Description: in.Description , LabelIDs: in.LabelIDs , Assignees: in.Assignees , StartAt: in.StartAt , DueAt: in.DueAt , Done: in.Done , Doing: in.Doing , RankPlace: in.RankPlace , Day: in.Day }
	return
}

// createCard needs edit rights on the column the card goes into: a board
// editor has them everywhere, an edit link for one column only there.
//
// @route("POST /api/boards/{board_id}/cards", summary="Add a card",
//     tag="Cards", errors=(400, 403, 404, 409),
//     description="Add a card to a column. The other fields are optional and set in the same call. "
//         "Needs edit rights on that column.",
//     mcp_note="If the card is made but its other fields are refused, the answer says so and names the "
//         "new card: change it with update_card rather than adding it again.",
//     mcp="create_card", call=handlers.mcpCreateCard,
// )
func ( handlers *Handlers ) createCard( c fiber.Ctx , in *createCardRequest ) ( reply *fiber.Map , err error ) {
	principal , board , err := handlers.boardAt( c , in.BoardID )
	if err != nil { return }
	card , refused := handlers.createCardRule( principal , board , in.ListID , in.Title , in.Position )
	if refused != nil {
		err = refused
		return
	}
	if patch := in.patch(); patch != ( models.CardPatch{} ) {
		updated , _ , patch_refused := handlers.patchCardRule( principal , board , card , &patch )
		if patch_refused != nil {
			// The card is there; card_id says which, so it is changed
			// rather than added twice.
			err = &refusal{ status: patch_refused.status , message: patch_refused.message , cause: patch_refused.cause , extra: fiber.Map{ "card_id": card.ID } }
			return
		}
		card = updated
	}
	handlers.lookUpProductLater( board , card )
	payload := cardPayload( card , access.Card( principal , board , card ) , access.Board( principal , board ) )
	reply = &payload
	return
}

// sharedCreateCard is createCard for a share link's guest.
//
// @route("POST /api/shared/{board_id}/cards", summary="Add a card (share link)",
//     tag="Shared", access=PUBLIC, use=[handlers.SharePrincipal], errors=(400, 403, 404, 409),
//     description="createCard for a guest holding a share link in X-Share-Token.",
// )
func ( handlers *Handlers ) sharedCreateCard( c fiber.Ctx , in *createCardRequest ) ( reply *fiber.Map , err error ) {
	reply , err = handlers.createCard( c , in )
	return
}

// createCardRule is createCard's rule: the column check and the card.
func ( handlers *Handlers ) createCardRule( principal *access.Principal , board *models.Board , list_id uint64 , title string , position *float64 ) ( card *models.Card , refused *refusal ) {
	if board.List( list_id ) == nil {
		refused = refuse( fiber.StatusNotFound , "list_not_found" )
		return
	}
	if access.List( principal , board , list_id ) < access.Edit {
		refused = writeRefusal( board , access.Edit )
		return
	}
	card , create_err := models.CreateCard( handlers.Store , board.ID , list_id , title , position , principal.UserID , principal.Name )
	if create_err != nil {
		refused = modelRefusal( create_err )
		return
	}
	handlers.activity( principal , board.ID , "card_added" , map[string]string{ "card": card.Title , "list": board.List( list_id ).Title } )
	return
}

// activity tells the board's subscribers what happened. Worded here and now
// because the thing it names may not be there by the time anyone is told.
func ( handlers *Handlers ) activity( principal *access.Principal , board_id uint64 , what string , values map[string]string ) {
	handlers.Notify.Activity( board_id , principal.UserID , principal.Name , what , values )
}

func ( handlers *Handlers ) checklistActivity( principal *access.Principal , card *models.Card ) {
	handlers.activity( principal , card.BoardID , "checklist" , map[string]string{ "card": card.Title } )
}

// patchCardRequest is models.CardPatch with the card it is for, spelled out
// so each field can say what it does.
type patchCardRequest struct {
	cardPath
	Rev         *uint64   `json:"rev,omitempty" jsonschema_description:"The rev from get_card / get_board. Pass it when changing title or description: if someone edited the card since, the change is refused rather than written over theirs."`
	Title       *string   `json:"title,omitempty"`
	Description *string   `json:"description,omitempty" jsonschema_description:"Markdown."`
	ListID      *uint64   `json:"list_id,omitempty" jsonschema_description:"The column to move it to, from get_board (board.lists)."`
	Position    *float64  `json:"position,omitempty" jsonschema_description:"Sort position within the column; lower comes first."`
	LabelIDs    *[]uint64 `json:"label_ids,omitempty" jsonschema_description:"The full set of label ids; replaces the current labels."`
	Assignees   *[]uint64 `json:"assignees,omitempty" jsonschema_description:"The full set of user ids; replaces the current assignees."`
	StartAt     *string   `json:"start_at,omitempty" jsonschema_description:"2026-10-30 for an all-day date, an RFC 3339 timestamp for a time, an empty string to clear."`
	DueAt       *string   `json:"due_at,omitempty" jsonschema_description:"2026-10-30 for an all-day date, an RFC 3339 timestamp for a time, an empty string to clear."`
	Done        *bool     `json:"done,omitempty" jsonschema_description:"Marks it complete, or not."`
	Doing       *bool     `json:"doing,omitempty" jsonschema_description:"On a staged board, the Doing stage. true starts the card (reopening it if done); done: true moves it on to Done."`
	Archived    *bool     `json:"archived,omitempty" jsonschema_description:"true hides it from the board (reversible); false brings it back."`
	Price       *string   `json:"price,omitempty" jsonschema_description:"The price shown on the card, as text; an empty string clears it."`
	Day         *string   `json:"day,omitempty" jsonschema_description:"On a calendar board (layout calendar), the day the card is filed under: 2026-11-07, or an empty string for none."`
	RankPlace   *int      `json:"rank_place,omitempty" jsonschema:"minimum=0" jsonschema_description:"Importance across the whole board: 1 is the most important, and other cards shift to make room. 0 takes the card out of the ranking. Needs edit rights on the board."`
	Rank        *float64  `json:"rank,omitempty" jsonschema_description:"The stored rank value itself (the rank from get_card), for putting a card back exactly as it was; rank_place is the usual way. Needs edit rights on the board."`
}

func ( in *patchCardRequest ) patch() ( patch *models.CardPatch ) {
	patch = &models.CardPatch{
		Rev: in.Rev , Title: in.Title , Description: in.Description , ListID: in.ListID , Position: in.Position ,
		LabelIDs: in.LabelIDs , Assignees: in.Assignees , StartAt: in.StartAt , DueAt: in.DueAt , Done: in.Done ,
		Doing: in.Doing , Archived: in.Archived , Price: in.Price , Day: in.Day , RankPlace: in.RankPlace , Rank: in.Rank ,
	}
	return
}

// patchCard edits a card. Editing its fields needs edit rights on the card;
// reordering or archiving it changes the column around it, so that needs
// edit rights on the column, and moving it to another column needs them on
// the board -- someone with one shared card or column cannot rearrange the
// rest. A card's rank is a place on the whole board, so that needs edit
// rights on the board too.
//
// @route("POST /api/boards/{board_id}/cards/{card_id}", summary="Change a card",
//     tag="Cards", errors=(400, 403, 404, 409),
//     description="Change any of a card's fields; leave out what should not change. Moving it (list_id, "
//         "position) or archiving it needs edit rights on its column, and on the whole board to move it to "
//         "another column. done marks it complete; doing and rank_place are the Doing stage and board-wide "
//         "importance of a staged board. archived: true hides it from the board (reversible). Pass the "
//         "card's rev when changing title or description: if someone edited it since, the change is "
//         "refused (409 card_stale, with the card as it now stands) rather than written over theirs.",
//     mcp="update_card", call=handlers.mcpUpdateCard,
// )
func ( handlers *Handlers ) patchCard( c fiber.Ctx , in *patchCardRequest ) ( reply *fiber.Map , err error ) {
	principal , board , card , err := handlers.requireCardAt( c , &in.cardPath , access.Edit )
	if err != nil { return }
	patch := in.patch()
	updated , current , refused := handlers.patchCardRule( principal , board , card , patch )
	if refused == nil && ( patch.Title != nil || patch.Description != nil || patch.ListID != nil ) { handlers.lookUpProductLater( board , updated ) }
	if current != nil {
		// Hand back the current card so the client can show what changed
		// without a second request.
		err = refuse( fiber.StatusConflict , "card_stale" , fiber.Map{ "card": cardPayload( current , access.Card( principal , board , current ) , access.Board( principal , board ) ) } )
		return
	}
	if refused != nil {
		err = refused
		return
	}
	payload := cardPayload( updated , access.Card( principal , board , updated ) , access.Board( principal , board ) )
	reply = &payload
	return
}

// sharedPatchCard is patchCard for a share link's guest.
//
// @route("POST /api/shared/{board_id}/cards/{card_id}", summary="Change a card (share link)",
//     tag="Shared", access=PUBLIC, use=[handlers.SharePrincipal], errors=(400, 403, 404, 409),
//     description="patchCard for a guest holding a share link in X-Share-Token.",
// )
func ( handlers *Handlers ) sharedPatchCard( c fiber.Ctx , in *patchCardRequest ) ( reply *fiber.Map , err error ) {
	reply , err = handlers.patchCard( c , in )
	return
}

// patchCardRule is patchCard's rule. When the patch named a rev that is no
// longer current, current is the card as it now stands, alongside the
// card_stale refusal.
func ( handlers *Handlers ) patchCardRule( principal *access.Principal , board *models.Board , card *models.Card , patch *models.CardPatch ) ( updated *models.Card , current *models.Card , refused *refusal ) {
	if patch.Moves() {
		leaves := patch.ListID != nil && *patch.ListID != card.ListID
		// Leaving its column needs edit rights where it lands too: a board
		// editor has them everywhere, a column link nowhere else, and a link
		// showing some columns only in those.
		if ( leaves && access.List( principal , board , *patch.ListID ) < access.Edit ) || access.List( principal , board , card.ListID ) < access.Edit {
			refused = writeRefusal( board , access.Edit )
			return
		}
	}
	if patch.Ranks() && access.Board( principal , board ) < access.Edit {
		refused = writeRefusal( board , access.Edit )
		return
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

// deleteCard moves the card to the board's trash (see models/trash.go);
// nothing is gone for good until the trash is emptied.
//
// @route("POST /api/boards/{board_id}/cards/{card_id}/delete", summary="Delete a card",
//     tag="Cards", errors=(400, 403, 404, 409),
//     description="Move a card to the board's trash. Not permanent: restore_card brings it back, until "
//         "purge_card or empty_trash removes it for good. Needs edit rights on its column.",
//     mcp="delete_card", destructive=True,
// )
func ( handlers *Handlers ) deleteCard( c fiber.Ctx , in *cardPath ) ( reply *okReply , err error ) {
	principal , board , card , err := handlers.requireCardAt( c , in , access.Edit )
	if err != nil { return }
	if access.List( principal , board , card.ListID ) < access.Edit {
		err = writeRefusal( board , access.Edit )
		return
	}
	if delete_err := models.TrashCard( handlers.Store , board.ID , card.ID , principal.Name ); delete_err != nil {
		err = modelErr( delete_err )
		return
	}
	handlers.activity( principal , board.ID , "card_trashed" , map[string]string{ "card": card.Title } )
	reply = &okReply{ Ok: true }
	return
}

// sharedDeleteCard is deleteCard for a share link's guest.
//
// @route("POST /api/shared/{board_id}/cards/{card_id}/delete", summary="Delete a card (share link)",
//     tag="Shared", access=PUBLIC, use=[handlers.SharePrincipal], errors=(400, 403, 404, 409),
//     description="deleteCard for a guest holding a share link in X-Share-Token.",
// )
func ( handlers *Handlers ) sharedDeleteCard( c fiber.Ctx , in *cardPath ) ( reply *okReply , err error ) {
	reply , err = handlers.deleteCard( c , in )
	return
}

// archiveCardsRequest names the cards to archive; without card_ids, every
// open card in the column goes.
type archiveCardsRequest struct {
	listPath
	CardIDs []uint64 `json:"card_ids,omitempty" jsonschema_description:"The cards to archive; leave out to archive every open card in the column."`
}

type archiveCardsReply struct {
	Archived []uint64 `json:"archived" jsonschema_description:"The ids of the cards archived."`
}

// archiveListCards archives a column's open cards, all or the ones picked --
// clearing out a Done column, say. Edit rights on that column are enough.
//
// @route("POST /api/boards/{board_id}/lists/{list_id}/archive-cards", summary="Archive a column's cards",
//     tag="Cards", errors=(400, 403, 404, 409),
//     description="Archive a column's open cards, all of them or the ones named in card_ids: clearing out "
//         "a Done column, say. Archived cards are hidden, not deleted (update_card with archived: false "
//         "brings one back). Needs edit rights on that column.",
//     mcp="archive_list_cards",
// )
func ( handlers *Handlers ) archiveListCards( c fiber.Ctx , in *archiveCardsRequest ) ( reply *archiveCardsReply , err error ) {
	principal , board , err := handlers.boardAt( c , in.BoardID )
	if err != nil { return }
	if in.ListID == 0 || board.List( in.ListID ) == nil {
		err = refuse( fiber.StatusNotFound , "list_not_found" )
		return
	}
	if access.List( principal , board , in.ListID ) < access.Edit {
		err = writeRefusal( board , access.Edit )
		return
	}
	ids , archive_err := models.ArchiveListCards( handlers.Store , board.ID , in.ListID , in.CardIDs , principal.Name )
	if archive_err != nil {
		err = modelErr( archive_err )
		return
	}
	if len( ids ) > 0 {
		handlers.activity( principal , board.ID , "column_archived" , map[string]string{ "list": board.List( in.ListID ).Title , "n": strconv.Itoa( len( ids ) ) } )
	}
	reply = &archiveCardsReply{ Archived: ids }
	return
}

// sharedArchiveListCards is archiveListCards for a share link's guest.
//
// @route("POST /api/shared/{board_id}/lists/{list_id}/archive-cards", summary="Archive a column's cards (share link)",
//     tag="Shared", access=PUBLIC, use=[handlers.SharePrincipal], errors=(400, 403, 404, 409),
//     description="archiveListCards for a guest holding a share link in X-Share-Token.",
// )
func ( handlers *Handlers ) sharedArchiveListCards( c fiber.Ctx , in *archiveCardsRequest ) ( reply *archiveCardsReply , err error ) {
	reply , err = handlers.archiveListCards( c , in )
	return
}

type checklistAddRequest struct {
	cardPath
	Text string `json:"text"`
}

// addChecklistItem adds to the end of a card's checklist.
//
// @route("POST /api/boards/{board_id}/cards/{card_id}/checklist", summary="Add a checklist item",
//     tag="Cards", errors=(400, 403, 404, 409),
//     description="Add an item to the end of a card's checklist. Returns the card. Needs edit rights on the card.",
//     mcp="add_checklist_item",
// )
func ( handlers *Handlers ) addChecklistItem( c fiber.Ctx , in *checklistAddRequest ) ( reply *fiber.Map , err error ) {
	principal , board , card , err := handlers.requireCardAt( c , &in.cardPath , access.Edit )
	if err != nil { return }
	updated , refused := handlers.addChecklistItemRule( principal , card , in.Text )
	reply , err = cardAnswer( principal , board , updated , refused )
	return
}

// sharedAddChecklistItem is addChecklistItem for a share link's guest.
//
// @route("POST /api/shared/{board_id}/cards/{card_id}/checklist", summary="Add a checklist item (share link)",
//     tag="Shared", access=PUBLIC, use=[handlers.SharePrincipal], errors=(400, 403, 404, 409),
//     description="addChecklistItem for a guest holding a share link in X-Share-Token.",
// )
func ( handlers *Handlers ) sharedAddChecklistItem( c fiber.Ctx , in *checklistAddRequest ) ( reply *fiber.Map , err error ) {
	reply , err = handlers.addChecklistItem( c , in )
	return
}

// addChecklistItemRule and updateChecklistItemRule change a card that has
// already passed loadCard at Edit.
func ( handlers *Handlers ) addChecklistItemRule( principal *access.Principal , card *models.Card , text string ) ( updated *models.Card , refused *refusal ) {
	updated , mutate_err := models.MutateCard( handlers.Store , card.BoardID , card.ID , principal.Name , func( _ *models.Board , target *models.Card ) ( inner error ) {
		_ , inner = models.AddChecklistItem( target , text )
		return
	} )
	if mutate_err != nil {
		refused = modelRefusal( mutate_err )
		return
	}
	handlers.checklistActivity( principal , updated )
	return
}

func ( handlers *Handlers ) updateChecklistItemRule( principal *access.Principal , card *models.Card , item_id uint64 , update *models.ChecklistUpdate ) ( updated *models.Card , refused *refusal ) {
	updated , mutate_err := models.MutateCard( handlers.Store , card.BoardID , card.ID , principal.Name , func( _ *models.Board , target *models.Card ) error {
		return update.Apply( target , item_id )
	} )
	if mutate_err != nil {
		refused = modelRefusal( mutate_err )
		return
	}
	handlers.checklistActivity( principal , updated )
	return
}

type checklistUpdateRequest struct {
	itemPath
	Text     *string  `json:"text,omitempty"`
	Done     *bool    `json:"done,omitempty" jsonschema_description:"true ticks it off, false back on."`
	Position *float64 `json:"position,omitempty" jsonschema_description:"Its place in the checklist; lower comes first."`
}

// updateChecklistItem ticks, edits or moves one checklist item.
//
// @route("POST /api/boards/{board_id}/cards/{card_id}/checklist/{item_id}", summary="Tick or edit a checklist item",
//     tag="Cards", errors=(400, 403, 404, 409),
//     description="Tick an item off (done: true) or back on, change its text, or move it. Item ids are in "
//         "the card's checklist. Returns the card. Needs edit rights on the card.",
//     mcp="update_checklist_item",
// )
func ( handlers *Handlers ) updateChecklistItem( c fiber.Ctx , in *checklistUpdateRequest ) ( reply *fiber.Map , err error ) {
	principal , board , card , err := handlers.requireCardAt( c , &in.cardPath , access.Edit )
	if err != nil { return }
	if in.ItemID == 0 {
		err = refuse( fiber.StatusNotFound , "item_not_found" )
		return
	}
	updated , refused := handlers.updateChecklistItemRule( principal , card , in.ItemID , &models.ChecklistUpdate{ Text: in.Text , Done: in.Done , Position: in.Position } )
	reply , err = cardAnswer( principal , board , updated , refused )
	return
}

// sharedUpdateChecklistItem is updateChecklistItem for a share link's guest.
//
// @route("POST /api/shared/{board_id}/cards/{card_id}/checklist/{item_id}", summary="Tick or edit a checklist item (share link)",
//     tag="Shared", access=PUBLIC, use=[handlers.SharePrincipal], errors=(400, 403, 404, 409),
//     description="updateChecklistItem for a guest holding a share link in X-Share-Token.",
// )
func ( handlers *Handlers ) sharedUpdateChecklistItem( c fiber.Ctx , in *checklistUpdateRequest ) ( reply *fiber.Map , err error ) {
	reply , err = handlers.updateChecklistItem( c , in )
	return
}

// deleteChecklistItem removes one checklist item.
//
// @route("POST /api/boards/{board_id}/cards/{card_id}/checklist/{item_id}/delete", summary="Delete a checklist item",
//     tag="Cards", errors=(400, 403, 404, 409),
//     description="Remove an item from a card's checklist, for good. Returns the card. Needs edit rights on the card.",
//     mcp="delete_checklist_item", destructive=True,
// )
func ( handlers *Handlers ) deleteChecklistItem( c fiber.Ctx , in *itemPath ) ( reply *fiber.Map , err error ) {
	principal , board , card , err := handlers.requireCardAt( c , &in.cardPath , access.Edit )
	if err != nil { return }
	if in.ItemID == 0 {
		err = refuse( fiber.StatusNotFound , "item_not_found" )
		return
	}
	updated , mutate_err := models.MutateCard( handlers.Store , board.ID , card.ID , principal.Name , func( _ *models.Board , target *models.Card ) error {
		return models.DeleteChecklistItem( target , in.ItemID )
	} )
	if mutate_err != nil {
		err = modelErr( mutate_err )
		return
	}
	handlers.checklistActivity( principal , updated )
	reply , err = cardAnswer( principal , board , updated , nil )
	return
}

// sharedDeleteChecklistItem is deleteChecklistItem for a share link's guest.
//
// @route("POST /api/shared/{board_id}/cards/{card_id}/checklist/{item_id}/delete", summary="Delete a checklist item (share link)",
//     tag="Shared", access=PUBLIC, use=[handlers.SharePrincipal], errors=(400, 403, 404, 409),
//     description="deleteChecklistItem for a guest holding a share link in X-Share-Token.",
// )
func ( handlers *Handlers ) sharedDeleteChecklistItem( c fiber.Ctx , in *itemPath ) ( reply *fiber.Map , err error ) {
	reply , err = handlers.deleteChecklistItem( c , in )
	return
}

// cardAnswer is a declared route's card reply: the refusal when there is
// one, else the card as the principal sees it.
func cardAnswer( principal *access.Principal , board *models.Board , card *models.Card , refused *refusal ) ( reply *fiber.Map , err error ) {
	if refused != nil {
		err = refused
		return
	}
	payload := cardPayload( card , access.Card( principal , board , card ) , access.Board( principal , board ) )
	reply = &payload
	return
}

// cardReply is cardAnswer for a handler that sends its own reply.
func ( handlers *Handlers ) cardReply( c fiber.Ctx , principal *access.Principal , board *models.Board , card *models.Card , refused *refusal ) ( err error ) {
	if refused != nil {
		err = refused.send( c )
		return
	}
	err = c.JSON( cardPayload( card , access.Card( principal , board , card ) , access.Board( principal , board ) ) )
	return
}

type trashReply struct {
	Cards []fiber.Map `json:"cards" jsonschema_description:"The cards in the trash, each with deleted_at and deleted_by."`
}

// listTrash: the trash is the board's, not one column's, so listing,
// restoring and purging need edit rights on the whole board.
//
// @route("GET /api/boards/{board_id}/trash", summary="List a board's trash",
//     tag="Trash", errors=(403, 404, 409),
//     description="The board's deleted cards, with when and by whom each was deleted. restore_card brings "
//         "one back. Needs edit rights on the whole board.",
//     mcp="list_trash", read_only=True,
// )
func ( handlers *Handlers ) listTrash( c fiber.Ctx , in *boardPath ) ( reply *trashReply , err error ) {
	principal , board , err := handlers.requireBoardAt( c , in.BoardID , access.Edit )
	if err != nil { return }
	cards , list_err := models.ListTrash( handlers.Store , board.ID )
	if list_err != nil {
		err = modelErr( list_err )
		return
	}
	level := access.Board( principal , board )
	payload := make( []fiber.Map , 0 , len( cards ) )
	for _ , card := range cards {
		item := cardPayload( card , level , level )
		item[ "deleted_at" ] , item[ "deleted_by" ] = card.DeletedAt , card.DeletedBy
		payload = append( payload , item )
	}
	reply = &trashReply{ Cards: payload }
	return
}

// sharedListTrash is listTrash for a share link's guest.
//
// @route("GET /api/shared/{board_id}/trash", summary="List a board's trash (share link)",
//     tag="Shared", access=PUBLIC, use=[handlers.SharePrincipal], errors=(403, 404, 409),
//     description="listTrash for a guest holding a share link in X-Share-Token.",
// )
func ( handlers *Handlers ) sharedListTrash( c fiber.Ctx , in *boardPath ) ( reply *trashReply , err error ) {
	reply , err = handlers.listTrash( c , in )
	return
}

// restoreCard takes a card back out of the trash.
//
// @route("POST /api/boards/{board_id}/trash/{card_id}/restore", summary="Restore a card from the trash",
//     tag="Trash", errors=(400, 403, 404, 409),
//     description="Bring a deleted card back out of the board's trash. Card ids are in list_trash. Needs "
//         "edit rights on the whole board.",
//     mcp="restore_card",
// )
func ( handlers *Handlers ) restoreCard( c fiber.Ctx , in *cardPath ) ( reply *fiber.Map , err error ) {
	principal , board , err := handlers.requireBoardAt( c , in.BoardID , access.Edit )
	if err != nil { return }
	if in.CardID == 0 {
		err = refuse( fiber.StatusNotFound , "card_not_found" )
		return
	}
	card , restore_err := models.RestoreCard( handlers.Store , board.ID , in.CardID , principal.Name )
	if restore_err != nil {
		err = modelErr( restore_err )
		return
	}
	handlers.activity( principal , board.ID , "card_restored" , map[string]string{ "card": card.Title } )
	reply , err = cardAnswer( principal , board , card , nil )
	return
}

// sharedRestoreCard is restoreCard for a share link's guest.
//
// @route("POST /api/shared/{board_id}/trash/{card_id}/restore", summary="Restore a card from the trash (share link)",
//     tag="Shared", access=PUBLIC, use=[handlers.SharePrincipal], errors=(400, 403, 404, 409),
//     description="restoreCard for a guest holding a share link in X-Share-Token.",
// )
func ( handlers *Handlers ) sharedRestoreCard( c fiber.Ctx , in *cardPath ) ( reply *fiber.Map , err error ) {
	reply , err = handlers.restoreCard( c , in )
	return
}

// purgeCard removes one card from the trash for good.
//
// @route("POST /api/boards/{board_id}/trash/{card_id}/delete", summary="Delete a card for good",
//     tag="Trash", errors=(400, 403, 404, 409),
//     description="Permanently delete one card that is in the board's trash. This cannot be undone. Needs "
//         "edit rights on the whole board.",
//     mcp="purge_card", destructive=True,
// )
func ( handlers *Handlers ) purgeCard( c fiber.Ctx , in *cardPath ) ( reply *okReply , err error ) {
	_ , board , err := handlers.requireBoardAt( c , in.BoardID , access.Edit )
	if err != nil { return }
	if in.CardID == 0 {
		err = refuse( fiber.StatusNotFound , "card_not_found" )
		return
	}
	if purge_err := models.PurgeCard( handlers.Store , board.ID , in.CardID ); purge_err != nil {
		err = modelErr( purge_err )
		return
	}
	reply = &okReply{ Ok: true }
	return
}

// sharedPurgeCard is purgeCard for a share link's guest.
//
// @route("POST /api/shared/{board_id}/trash/{card_id}/delete", summary="Delete a card for good (share link)",
//     tag="Shared", access=PUBLIC, use=[handlers.SharePrincipal], errors=(400, 403, 404, 409),
//     description="purgeCard for a guest holding a share link in X-Share-Token.",
// )
func ( handlers *Handlers ) sharedPurgeCard( c fiber.Ctx , in *cardPath ) ( reply *okReply , err error ) {
	reply , err = handlers.purgeCard( c , in )
	return
}

type emptyTrashReply struct {
	Removed int `json:"removed" jsonschema_description:"How many cards were deleted for good."`
}

// emptyTrash removes every card in the trash for good.
//
// @route("POST /api/boards/{board_id}/trash/empty", summary="Empty a board's trash",
//     tag="Trash", errors=(400, 403, 404, 409),
//     description="Permanently delete every card in the board's trash. This cannot be undone. Needs edit "
//         "rights on the whole board.",
//     mcp="empty_trash", destructive=True,
// )
func ( handlers *Handlers ) emptyTrash( c fiber.Ctx , in *boardPath ) ( reply *emptyTrashReply , err error ) {
	_ , board , err := handlers.requireBoardAt( c , in.BoardID , access.Edit )
	if err != nil { return }
	removed , empty_err := models.EmptyTrash( handlers.Store , board.ID )
	if empty_err != nil {
		err = modelErr( empty_err )
		return
	}
	reply = &emptyTrashReply{ Removed: removed }
	return
}

// sharedEmptyTrash is emptyTrash for a share link's guest.
//
// @route("POST /api/shared/{board_id}/trash/empty", summary="Empty a board's trash (share link)",
//     tag="Shared", access=PUBLIC, use=[handlers.SharePrincipal], errors=(400, 403, 404, 409),
//     description="emptyTrash for a guest holding a share link in X-Share-Token.",
// )
func ( handlers *Handlers ) sharedEmptyTrash( c fiber.Ctx , in *boardPath ) ( reply *emptyTrashReply , err error ) {
	reply , err = handlers.emptyTrash( c , in )
	return
}
