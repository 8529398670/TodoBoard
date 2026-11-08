package routes

import (
	errors "errors"

	fiber "github.com/gofiber/fiber/v3"

	access "todoboard/server/access"
	models "todoboard/server/models"
)

type createBoardRequest struct {
	Title string   `json:"title"`
	Lists []string `json:"lists"`
}

type listRequest struct {
	Title    string   `json:"title"`
	Position *float64 `json:"position"`
}

type labelRequest struct {
	Name  *string `json:"name"`
	Color *string `json:"color"`
}

// ListMyBoards is the home page: boards I own or am a member of, the folders
// I sort them into, and single cards shared with me from boards I am not on.
func ( handlers *Handlers ) ListMyBoards( c fiber.Ctx ) ( err error ) {
	payload , list_err := handlers.myBoards( principalFrom( c ) )
	if list_err != nil {
		err = serverError( c )
		return
	}
	err = c.JSON( payload )
	return
}

// myBoards is shared with the MCP list_boards tool. This is a list rather
// than one board, so a key's board limit is applied here by hand. A limited
// key gets no folders: they are home-page settings that span boards.
func ( handlers *Handlers ) myBoards( principal *access.Principal ) ( payload fiber.Map , err error ) {
	boards , cards , err := handlers.boardsFor( principal )
	if err != nil { return }
	folders := []*models.Folder{}
	if principal.Key.Limited() == false {
		folders , err = models.ListFolders( handlers.Store , principal.UserID )
		if err != nil { return }
	}
	payload = fiber.Map{ "boards": boards , "folders": folders , "shared_cards": cards }
	return
}

// boardsFor is models.ListBoardsForUser cut down to what the principal's key
// may reach. Anything listing boards for a member goes through here.
func ( handlers *Handlers ) boardsFor( principal *access.Principal ) ( boards []*models.BoardSummary , cards []*models.SharedCardSummary , err error ) {
	all_boards , all_cards , err := models.ListBoardsForUser( handlers.Store , principal.UserID )
	if err != nil { return }
	boards = []*models.BoardSummary{}
	cards = []*models.SharedCardSummary{}
	for _ , board := range all_boards {
		if access.KeyAllows( principal , board.ID ) { boards = append( boards , board ) }
	}
	for _ , card := range all_cards {
		if access.KeyAllows( principal , card.BoardID ) { cards = append( cards , card ) }
	}
	return
}

func ( handlers *Handlers ) CreateBoard( c fiber.Ctx ) ( err error ) {
	var body createBoardRequest
	if ok , bind_err := bindBody( c , &body ); ok == false { return bind_err }
	principal := principalFrom( c )
	board , create_err := models.CreateBoard( handlers.Store , principal.UserID , body.Title , body.Lists )
	if create_err != nil {
		err = modelError( c , create_err )
		return
	}
	err = c.JSON( fiber.Map{ "id": board.ID , "slug": board.Slug } )
	return
}

// GetBoard returns everything a board view needs in one response: the board,
// the cards this caller may see, the names behind every user id mentioned,
// and what this caller may do.
func ( handlers *Handlers ) GetBoard( c fiber.Ctx ) ( err error ) {
	principal , board , err := handlers.boardFor( c )
	if board == nil { return }
	cards , list_err := models.ListCards( handlers.Store , board.ID )
	if list_err != nil {
		err = serverError( c )
		return
	}
	err = c.JSON( handlers.boardPayload( principal , board , cards ) )
	return
}

// GetBoardStamp is the cheap call a board view polls: the revision only.
func ( handlers *Handlers ) GetBoardStamp( c fiber.Ctx ) ( err error ) {
	principal , board , err := handlers.boardFor( c )
	if board == nil { return }
	rev := board.Rev
	// The board's revision moves with every other card, which is not a quick
	// link's to know: it hears about its own card.
	if access.QuickLink( principal ) {
		card , get_err := models.GetCard( handlers.Store , board.ID , principal.Share.CardID )
		if get_err != nil {
			err = modelError( c , get_err )
			return
		}
		rev = card.Rev
	}
	err = c.JSON( fiber.Map{ "rev": rev } )
	return
}

// quickBoardPayload is all a quick link is shown, and takes its shape from
// boardPayload so the page needs no second path. The fields are named one by
// one rather than cut from the full payload, so that a field added to a card
// or board later stays private until someone puts it here.
func quickBoardPayload( principal *access.Principal , board *models.Board , all_cards []*models.Card ) ( payload fiber.Map ) {
	cards := []fiber.Map{}
	labels := []*models.Label{}
	rev := uint64( 0 )
	for _ , card := range all_cards {
		if card.ID != principal.Share.CardID || access.Card( principal , board , card ) == access.None { continue }
		rev = card.Rev
		for _ , label := range board.Labels {
			if containsLabel( card.LabelIDs , label.ID ) { labels = append( labels , label ) }
		}
		cards = append( cards , fiber.Map{
			"id":          card.ID,
			"list_id":     0,
			"position":    0,
			"title":       card.Title,
			"description": card.Description,
			"label_ids":   card.LabelIDs,
			"assignees":   []uint64{},
			"start_at":    card.StartAt,
			"start_all_day": card.StartAt != nil && card.StartTimed == false,
			"due_at":      card.DueAt,
			"due_all_day": card.DueAllDay,
			"done":        card.Done,
			"doing":       card.Doing,
			"checklist":   card.Checklist,
			"archived":    card.Archived,
			"rev":         card.Rev,
			"access":      access.View.String(),
			"product":     card.Product,
			"price":       card.Price,
			"image_at":    card.ImageAt,
		} )
	}
	payload = fiber.Map{
		"board": fiber.Map{
			"id":      board.ID,
			"title":   "",
			"layout":  board.Layout,
			"rev":     rev,
			"lists":   []*models.List{},
			"labels":  labels,
			"members": []fiber.Map{},
		},
		"cards":     cards,
		"people":    map[uint64]string{},
		"access":    access.None.String(),
		"card_only": true,
		"list_only": 0,
		"columns_only": false,
		"hidden_lists": map[uint64]int{},
		"me":        fiber.Map{ "guest": true },
		"quick":     true,
	}
	return
}

func containsLabel( label_ids []uint64 , label_id uint64 ) ( result bool ) {
	for _ , id := range label_ids {
		if id == label_id { return true }
	}
	return
}

func ( handlers *Handlers ) boardPayload( principal *access.Principal , board *models.Board , all_cards []*models.Card ) ( payload fiber.Map ) {
	if access.QuickLink( principal ) {
		payload = quickBoardPayload( principal , board , all_cards )
		return
	}
	board_level := access.Board( principal , board )
	// A column link sees that one column as a board of its own; a card-only
	// caller sees just their cards. Neither is shown the member list.
	list_only := access.ListOnly( principal , board )
	card_only := board_level == access.None && list_only == 0
	whole_board := board_level != access.None
	reported := board_level
	if list_only != 0 { reported = access.List( principal , board , list_only ) }
	// A link showing only some columns may still edit cards in those; the
	// page offers that, and nothing board-wide (columns_only).
	columns_only := principal.Share != nil && principal.Share.BoardID == board.ID && principal.Share.Restricted()
	if columns_only && principal.Share.Mode == models.ModeEdit { reported = access.Edit }

	visible := []fiber.Map{}
	visible_lists := map[uint64]bool{}
	people_ids := map[uint64]bool{ board.OwnerID: true }
	// Columns a link shows by title only: how many open cards are in each,
	// and nothing else about them.
	hidden_lists := map[uint64]int{}
	places := models.RankPlaces( board , all_cards )
	comment_counts , _ := models.CommentCounts( handlers.Store , board.ID )
	for _ , card := range all_cards {
		if access.HiddenList( principal , board , card.ListID ) {
			if card.Archived == false { hidden_lists[ card.ListID ] += 1 }
			continue
		}
		level := access.Card( principal , board , card )
		if level == access.None { continue }
		visible_lists[ card.ListID ] = true
		for _ , id := range card.Assignees { people_ids[ id ] = true }
		row := cardPayload( card , level , board_level )
		row[ "rank_place" ] = places[ card.ID ]
		row[ "comment_count" ] = comment_counts[ card.ID ]
		row[ "can_comment" ] = access.CanComment( principal , board , card )
		visible = append( visible , row )
	}

	lists := []*models.List{}
	for _ , list := range board.SortedLists() {
		if list_only != 0 && list.ID != list_only { continue }
		if card_only && visible_lists[ list.ID ] == false { continue }
		if columns_only && access.HiddenList( principal , board , list.ID ) { hidden_lists[ list.ID ] += 0 }
		lists = append( lists , list )
	}

	members := []fiber.Map{}
	if whole_board {
		for id := range board.Members { people_ids[ id ] = true }
	}
	people := map[uint64]string{}
	for id := range people_ids {
		if user , get_err := models.GetUser( handlers.Store , id ); get_err == nil {
			people[ id ] = user.DisplayName
		}
	}
	if whole_board {
		members = append( members , fiber.Map{ "user_id": board.OwnerID , "role": models.BoardRoleOwner , "display_name": people[ board.OwnerID ] } )
		for id , role := range board.Members {
			members = append( members , fiber.Map{ "user_id": id , "role": role , "display_name": people[ id ] } )
		}
	}

	me := fiber.Map{ "guest": principal.IsGuest() }
	if principal.IsGuest() == false {
		me[ "user_id" ] = principal.UserID
		me[ "role" ] = board.RoleOf( principal.UserID )
		if ref := models.GetBoardRef( handlers.Store , principal.UserID , board.ID ); ref != nil {
			me[ "plain_icons" ] = ref.PlainIcons
			// Always a list here: having one is how the page knows this person
			// has a row to save theirs in.
			me[ "collapsed_lists" ] = ref.CollapsedLists
			if ref.CollapsedLists == nil { me[ "collapsed_lists" ] = []uint64{} }
		}
	}

	board_payload := fiber.Map{
		"id":           board.ID,
		"title":        board.Title,
		"slug":         board.Slug,
		"layout":       board.Layout,
		"numbering":    board.Numbering,
		"done_placement": board.DonePlacement,
		"add_placement":  board.AddPlacement,
		"owner_id":     board.OwnerID,
		"default_view": board.DefaultView,
		"color":        board.Color,
		"shopping":     board.Shopping,
		"show_rank":    board.ShowRank,
		"center_columns": board.CenterColumns,
		"stage_todo":   board.StageTodo,
		"stage_doing":  board.StageDoing,
		"stage_done":   board.StageDone,
		"theme":        board.Theme,
		"rev":          board.Rev,
		"lists":        lists,
		"labels":       board.Labels,
		"members":      members,
	}
	// The people working on the board decide what to share; a guest has no
	// use for it.
	if principal.IsGuest() == false { board_payload[ "allow_quick_share" ] = board.AllowQuickShare }
	// Everyone needs this one: a guest without a name is asked for one
	// before commenting unless it is on.
	board_payload[ "allow_anon_comments" ] = board.AllowAnonComments
	board_payload[ "carry_over" ] = board.CarriesOver()
	// Left out for a link showing only part of the board: the notes are about
	// all of it, and may point at things those columns were kept apart from.
	if whole_board && columns_only == false {
		for key , value := range boardTextPayload( board ) { board_payload[ key ] = value }
	}

	payload = fiber.Map{
		"board":     board_payload,
		"cards":     visible,
		"people":    people,
		"access":    reported.String(),
		"card_only": card_only,
		"list_only": list_only,
		"columns_only": columns_only,
		"hidden_lists": hidden_lists,
		"me":        me,
	}
	return
}

// boardTextPayload leaves the purpose out while it is switched off, so a
// purpose someone turned off stops steering agents.
func boardTextPayload( board *models.Board ) ( payload fiber.Map ) {
	payload = fiber.Map{
		"purpose_enabled": board.PurposeEnabled,
		"notes":           board.Notes,
		"notes_rev":       board.NotesRev,
	}
	if board.PurposeEnabled {
		payload[ "purpose" ] = board.Purpose
		payload[ "purpose_rev" ] = board.PurposeRev
	}
	return
}

// cardPayload is a card as a caller sees it. Who else a card is shared with
// is the owner's business, so it is only included for someone who manages
// the board.
func cardPayload( card *models.Card , level access.Level , board_level access.Level ) ( payload fiber.Map ) {
	payload = fiber.Map{
		"id":          card.ID,
		"serial":      card.Serial,
		"uid":         card.UID,
		"board_id":    card.BoardID,
		"list_id":     card.ListID,
		"position":    card.Position,
		"title":       card.Title,
		"description": card.Description,
		"label_ids":   card.LabelIDs,
		"assignees":   card.Assignees,
		"start_at":    card.StartAt,
		"start_all_day": card.StartAt != nil && card.StartTimed == false,
		"due_at":      card.DueAt,
		"due_all_day": card.DueAllDay,
		"day":         card.Day,
		"done":        card.Done,
		"done_at":     card.DoneAt,
		"doing":       card.Doing,
		"rank":        card.Rank,
		"checklist":   card.Checklist,
		"archived":    card.Archived,
		"created_at":  card.CreatedAt,
		"updated_at":  card.UpdatedAt,
		"updated_by":  card.UpdatedBy,
		"rev":         card.Rev,
		"access":      level.String(),
		"product":     card.Product,
		"price":       card.Price,
		"image_at":    card.ImageAt,
		"mirror":      card.Mirror,
	}
	if board_level == access.Manage {
		payload[ "shared_with" ] = card.SharedWith
	}
	return
}

func ( handlers *Handlers ) UpdateBoard( c fiber.Ctx ) ( err error ) {
	principal , board , err := handlers.requireBoard( c , access.Edit )
	if board == nil { return }
	var body models.BoardSettings
	if ok , bind_err := bindBody( c , &body ); ok == false { return bind_err }
	// The purpose is what agents act on, so it is for the board's members: an
	// edit link is handed around for changing cards, not for instructing them.
	if principal.IsGuest() && ( body.PurposeEnabled != nil || body.Purpose != nil || body.Notes != nil ) {
		err = c.Status( fiber.StatusForbidden ).JSON( fiber.Map{ "error": "forbidden" } )
		return
	}
	// Opening a board's cards to the public is the owner's call, made in a
	// browser: a leaked key cannot switch paused links back on.
	if ( body.AllowQuickShare != nil || body.AllowAnonComments != nil ) && ( access.Board( principal , board ) != access.Manage || principal.Key != nil ) {
		err = c.Status( fiber.StatusForbidden ).JSON( fiber.Map{ "error": "forbidden" } )
		return
	}
	before := *board
	updated , update_err := models.UpdateBoardSettings( handlers.Store , board.ID , &body )
	if errors.Is( update_err , models.ErrSlugTaken ) {
		err = c.Status( fiber.StatusConflict ).JSON( fiber.Map{ "error": "slug_taken" } )
		return
	}
	if update_err != nil {
		err = modelError( c , update_err )
		return
	}
	// A board becoming a calendar files the cards in date-titled columns
	// (06NOV2026, 2026-11-06) under those days.
	if before.Layout != models.LayoutCalendar && updated.Layout == models.LayoutCalendar {
		if dated , _ := models.DaysFromListTitles( handlers.Store , board.ID ); dated > 0 {
			if again , get_err := models.GetBoard( handlers.Store , board.ID ); get_err == nil { updated = again }
		}
	}
	handlers.boardTextActivity( principal , &before , updated )
	err = c.JSON( fiber.Map{ "rev": updated.Rev , "slug": updated.Slug , "purpose_rev": updated.PurposeRev , "notes_rev": updated.NotesRev } )
	return
}

// boardTextActivity reports the board's words changing. Its look is left out:
// a theme or colour is not news.
func ( handlers *Handlers ) boardTextActivity( principal *access.Principal , before *models.Board , after *models.Board ) {
	if before.Title != after.Title { handlers.activity( principal , after.ID , "board_renamed" , map[string]string{ "title": after.Title } ) }
	if before.Notes != after.Notes { handlers.activity( principal , after.ID , "board_notes" , nil ) }
	if before.Purpose != after.Purpose { handlers.activity( principal , after.ID , "board_purpose" , nil ) }
}

// DeleteBoard needs the owner and a browser session: it cannot be undone,
// so a leaked API key must not be able to do it.
func ( handlers *Handlers ) DeleteBoard( c fiber.Ctx ) ( err error ) {
	_ , board , err := handlers.requireBoard( c , access.Manage )
	if board == nil { return }
	if delete_err := models.DeleteBoard( handlers.Store , board.ID ); delete_err != nil {
		err = modelError( c , delete_err )
		return
	}
	err = c.JSON( fiber.Map{ "ok": true } )
	return
}

// plain_icons null means "use my account default".
type myViewRequest struct {
	PlainIcons *bool `json:"plain_icons"`
}

// SetMyBoardView saves how this board looks to the caller alone -- greyscale
// status icons, for now. Only people on the board have somewhere to keep it.
func ( handlers *Handlers ) SetMyBoardView( c fiber.Ctx ) ( err error ) {
	principal , board , err := handlers.requireBoard( c , access.View )
	if board == nil { return }
	var body myViewRequest
	if ok , bind_err := bindBody( c , &body ); ok == false { return bind_err }
	if set_err := models.SetBoardPlainIcons( handlers.Store , principal.UserID , board.ID , body.PlainIcons ); set_err != nil {
		err = modelError( c , set_err )
		return
	}
	err = c.JSON( fiber.Map{ "plain_icons": body.PlainIcons } )
	return
}

type myCollapsedRequest struct {
	ListIDs []uint64 `json:"list_ids"`
}

// SetMyCollapsedLists saves which columns the caller keeps collapsed: the
// whole list each time. Its own route so that saving it cannot touch the rest
// of how the board looks to them.
func ( handlers *Handlers ) SetMyCollapsedLists( c fiber.Ctx ) ( err error ) {
	principal , board , err := handlers.requireBoard( c , access.View )
	if board == nil { return }
	var body myCollapsedRequest
	if ok , bind_err := bindBody( c , &body ); ok == false { return bind_err }
	kept , set_err := models.SetBoardCollapsedLists( handlers.Store , principal.UserID , board , body.ListIDs )
	if set_err != nil {
		err = modelError( c , set_err )
		return
	}
	err = c.JSON( fiber.Map{ "collapsed_lists": kept } )
	return
}

// LeaveBoard lets a member remove themselves. The owner cannot leave; they
// delete the board or hand it over instead.
func ( handlers *Handlers ) LeaveBoard( c fiber.Ctx ) ( err error ) {
	principal , board , err := handlers.requireBoard( c , access.View )
	if board == nil { return }
	if board.OwnerID == principal.UserID {
		err = badRequest( c , "owner_cannot_leave" )
		return
	}
	if _ , set_err := models.SetMember( handlers.Store , board.ID , principal.UserID , "" ); set_err != nil {
		err = modelError( c , set_err )
		return
	}
	handlers.activity( principal , board.ID , "member_left" , nil )
	err = c.JSON( fiber.Map{ "ok": true } )
	return
}

func ( handlers *Handlers ) CreateList( c fiber.Ctx ) ( err error ) {
	principal , board , err := handlers.requireBoard( c , access.Edit )
	if board == nil { return }
	var body listRequest
	if ok , bind_err := bindBody( c , &body ); ok == false { return bind_err }
	created , refused := handlers.createList( principal , board , body.Title , body.Position )
	if refused != nil {
		err = refused.send( c )
		return
	}
	err = c.JSON( created )
	return
}

// createList, updateList and createLabel are shared with the MCP tools of
// the same name. The board has already passed loadBoardAt at Edit.
func ( handlers *Handlers ) createList( principal *access.Principal , board *models.Board , title string , position *float64 ) ( created *models.List , refused *refusal ) {
	_ , update_err := models.MutateBoard( handlers.Store , board.ID , func( target *models.Board ) ( mutate_err error ) {
		created , mutate_err = models.AddList( target , title , position )
		return
	} )
	if update_err != nil {
		refused = modelRefusal( update_err )
		return
	}
	handlers.activity( principal , board.ID , "list_added" , map[string]string{ "list": created.Title } )
	return
}

func ( handlers *Handlers ) updateList( principal *access.Principal , board *models.Board , list_id uint64 , update *models.ListUpdate ) ( refused *refusal ) {
	// A copy: the board passed in is not reread, and Apply is what changes.
	before := models.List{}
	if list := board.List( list_id ); list != nil { before = *list }
	updated , update_err := models.MutateBoard( handlers.Store , board.ID , func( target *models.Board ) error {
		return update.Apply( target , list_id )
	} )
	if update_err != nil {
		refused = modelRefusal( update_err )
		return
	}
	after := updated.List( list_id )
	if after == nil { return }
	if after.Title != before.Title {
		handlers.activity( principal , board.ID , "list_renamed" , map[string]string{ "from": before.Title , "list": after.Title } )
	}
	if after.Archived != before.Archived {
		what := "list_restored"
		if after.Archived { what = "list_archived" }
		handlers.activity( principal , board.ID , what , map[string]string{ "list": after.Title } )
	}
	return
}

func ( handlers *Handlers ) createLabel( board *models.Board , name string , color string ) ( created *models.Label , refused *refusal ) {
	_ , update_err := models.MutateBoard( handlers.Store , board.ID , func( target *models.Board ) ( mutate_err error ) {
		created , mutate_err = models.AddLabel( target , name , color )
		return
	} )
	if update_err != nil { refused = modelRefusal( update_err ) }
	return
}

func ( handlers *Handlers ) UpdateList( c fiber.Ctx ) ( err error ) {
	principal , board , err := handlers.requireBoard( c , access.Edit )
	if board == nil { return }
	list_id , ok := paramID( c , "list_id" )
	if ok == false {
		err = notFound( c , "list_not_found" )
		return
	}
	var body models.ListUpdate
	if ok , bind_err := bindBody( c , &body ); ok == false { return bind_err }
	if refused := handlers.updateList( principal , board , list_id , &body ); refused != nil {
		err = refused.send( c )
		return
	}
	err = c.JSON( fiber.Map{ "ok": true } )
	return
}

func ( handlers *Handlers ) DeleteList( c fiber.Ctx ) ( err error ) {
	principal , board , err := handlers.requireBoard( c , access.Edit )
	if board == nil { return }
	list_id , ok := paramID( c , "list_id" )
	if ok == false {
		err = notFound( c , "list_not_found" )
		return
	}
	title := ""
	if list := board.List( list_id ); list != nil { title = list.Title }
	if delete_err := models.DeleteList( handlers.Store , board.ID , list_id , principal.Name ); delete_err != nil {
		err = modelError( c , delete_err )
		return
	}
	handlers.activity( principal , board.ID , "list_deleted" , map[string]string{ "list": title } )
	err = c.JSON( fiber.Map{ "ok": true } )
	return
}

func ( handlers *Handlers ) CreateLabel( c fiber.Ctx ) ( err error ) {
	_ , board , err := handlers.requireBoard( c , access.Edit )
	if board == nil { return }
	var body labelRequest
	if ok , bind_err := bindBody( c , &body ); ok == false { return bind_err }
	if body.Color == nil {
		err = badRequest( c , "invalid_input" )
		return
	}
	name := ""
	if body.Name != nil { name = *body.Name }
	created , refused := handlers.createLabel( board , name , *body.Color )
	if refused != nil {
		err = refused.send( c )
		return
	}
	err = c.JSON( created )
	return
}

func ( handlers *Handlers ) UpdateLabel( c fiber.Ctx ) ( err error ) {
	_ , board , err := handlers.requireBoard( c , access.Edit )
	if board == nil { return }
	label_id , ok := paramID( c , "label_id" )
	if ok == false {
		err = notFound( c , "label_not_found" )
		return
	}
	var body labelRequest
	if ok , bind_err := bindBody( c , &body ); ok == false { return bind_err }
	_ , update_err := models.MutateBoard( handlers.Store , board.ID , func( target *models.Board ) error {
		return models.UpdateLabel( target , label_id , body.Name , body.Color )
	} )
	if update_err != nil {
		err = modelError( c , update_err )
		return
	}
	err = c.JSON( fiber.Map{ "ok": true } )
	return
}

func ( handlers *Handlers ) DeleteLabel( c fiber.Ctx ) ( err error ) {
	_ , board , err := handlers.requireBoard( c , access.Edit )
	if board == nil { return }
	label_id , ok := paramID( c , "label_id" )
	if ok == false {
		err = notFound( c , "label_not_found" )
		return
	}
	if delete_err := models.DeleteLabel( handlers.Store , board.ID , label_id ); delete_err != nil {
		err = modelError( c , delete_err )
		return
	}
	err = c.JSON( fiber.Map{ "ok": true } )
	return
}

type carryOverRequest struct {
	Today string `json:"today"`
}

// CarryOver moves a calendar board's unfinished cards from past days to
// today, the caller's today. The page asks on opening the board, so
// whoever opens it first that day does it for everyone; asking again moves
// nothing. A board with carry-over off, or not a calendar, moves nothing.
func ( handlers *Handlers ) CarryOver( c fiber.Ctx ) ( err error ) {
	principal , board , err := handlers.requireBoard( c , access.Edit )
	if board == nil { return }
	var body carryOverRequest
	if ok , bind_err := bindBody( c , &body ); ok == false { return bind_err }
	moved , carry_err := models.CarryOverCards( handlers.Store , board.ID , body.Today , principal.Name )
	if carry_err != nil {
		err = modelError( c , carry_err )
		return
	}
	err = c.JSON( fiber.Map{ "moved": moved } )
	return
}
