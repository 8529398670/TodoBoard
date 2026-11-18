package routes

import (
	errors "errors"

	fiber "github.com/gofiber/fiber/v3"

	access "todoboard/server/access"
	apidoc "todoboard/server/apidoc"
	models "todoboard/server/models"
)

// Boards, their columns and labels. The routes on a board's own content
// (reading it, its settings, columns, labels, carry-over) are mounted twice:
// under /api/boards for people on the board, and under /api/shared for a
// guest holding a share link (the shared* wrappers). Who may do what is
// server/access's decision either way.

type createBoardRequest struct {
	Title string   `json:"title" jsonschema:"maxLength=120" jsonschema_description:"The new board's title."`
	Lists []string `json:"lists,omitempty" jsonschema_description:"Its columns, in order, for example [\"To do\", \"Doing\", \"Done\"]. A board always gets at least three, so fewer are padded out."`
}

// newBoardReply is a board just made: where to find it.
type newBoardReply struct {
	ID   uint64 `json:"id"`
	Slug string `json:"slug" jsonschema_description:"Its URL name, the part after /b/."`
}

type getBoardRequest struct {
	boardPath
	IncludeArchived *bool `json:"include_archived,omitempty" query:"include_archived" jsonschema_description:"Include archived cards. The page gets them unless this is false; an agent's get_board leaves them out unless it is true."`
}

type boardStampReply struct {
	Rev uint64 `json:"rev" jsonschema_description:"Moves whenever anything the caller can see on the board changes."`
}

// updateBoardRequest is models.BoardSettings, field for field, with words for
// an agent. Every field is optional: what is left out does not change.
type updateBoardRequest struct {
	boardPath
	Title         *string `json:"title,omitempty" jsonschema:"maxLength=120"`
	Slug          *string `json:"slug,omitempty" jsonschema_description:"The board's URL name (the part after /b/). Empty picks one from the title. 409 slug_taken if another board has it."`
	DefaultView   *string `json:"default_view,omitempty" jsonschema:"enum=kanban,enum=todo,enum=timeline" jsonschema_description:"The view the board opens in for someone who has not picked one."`
	Color         *string `json:"color,omitempty" jsonschema:"enum=,enum=blue,enum=green,enum=orange,enum=red,enum=purple,enum=pink,enum=teal,enum=gray,enum=yellow" jsonschema_description:"The board's colour; empty for none."`
	Layout        *string `json:"layout,omitempty" jsonschema:"enum=,enum=three,enum=staged,enum=calendar" jsonschema_description:"\"\" (free: any number of columns), \"three\" (pinned to three columns: to do, doing, done), \"staged\" (each column has its own To do / Doing / Done, with an All view ranked board-wide), or \"calendar\" (cards filed by day; columns titled with dates are filed under those days)."`
	ListTitles    []string `json:"list_titles,omitempty" jsonschema_description:"With layout \"three\": the titles of the columns added if the board has fewer than three open ones."`
	Numbering     *string `json:"numbering,omitempty" jsonschema:"enum=,enum=on,enum=off" jsonschema_description:"Card numbers for everyone on the board: on, off, or \"\" to leave it to each person's account setting."`
	DonePlacement *string `json:"done_placement,omitempty" jsonschema:"enum=,enum=top,enum=bottom" jsonschema_description:"Where finished cards go in a column: top, bottom, or \"\" for each person's account setting."`
	AddPlacement  *string `json:"add_placement,omitempty" jsonschema:"enum=,enum=top,enum=bottom" jsonschema_description:"Where the add-card box sits in a column: top, bottom, or \"\" for each person's account setting."`
	Theme         *models.DisplayTheme `json:"theme,omitempty" jsonschema_description:"The board's own theme, shown to everyone on it."`
	ClearTheme    bool    `json:"clear_theme,omitempty" jsonschema_description:"true drops the board's own theme: everyone sees their own again."`
	Shopping      *bool   `json:"shopping,omitempty" jsonschema_description:"A shopping board: cards carry products and prices."`
	ShowRank      *bool   `json:"show_rank,omitempty" jsonschema_description:"Show each card's place in the board-wide ranking."`
	CenterColumns *bool   `json:"center_columns,omitempty" jsonschema_description:"Centre the columns on wide screens."`
	StageTabs     *bool   `json:"stage_tabs,omitempty" jsonschema_description:"On a staged board, give every column its own To do / Doing / Done buttons."`
	ShowUpdated   *bool   `json:"show_updated,omitempty" jsonschema_description:"Every card face shows when it was last changed, in its bottom-right corner. A column's own show_updated can override it."`
	StageTodo     *string `json:"stage_todo,omitempty" jsonschema_description:"What the To do stage is called on this board; empty for the default."`
	StageDoing    *string `json:"stage_doing,omitempty" jsonschema_description:"What the Doing stage is called on this board; empty for the default."`
	StageScheduled *string `json:"stage_scheduled,omitempty" jsonschema_description:"With stage_set review: what the Scheduled stage is called; empty for the default."`
	StageSet      *string `json:"stage_set,omitempty" jsonschema:"enum=,enum=review" jsonschema_description:"On a staged or calendar board, its stages: \"\" for To do / Doing / Done, or \"review\" for Ready for review / Working / Scheduled / Resolved (cards' doing, scheduled and done flags)."`
	StageDone     *string `json:"stage_done,omitempty" jsonschema_description:"What the Done stage is called on this board; empty for the default."`
	CarryOver     *bool   `json:"carry_over,omitempty" jsonschema_description:"On a calendar board, move unfinished cards from past days to today."`
	AllowQuickShare   *bool `json:"allow_quick_share,omitempty" jsonschema_description:"Let the board's cards be shared by quick link. The owner's, from a browser only: refused through a key."`
	AllowAnonComments *bool `json:"allow_anon_comments,omitempty" jsonschema_description:"Let guests comment without giving a name. The owner's, from a browser only: refused through a key."`
	PurposeEnabled *bool   `json:"purpose_enabled,omitempty" jsonschema_description:"Switch the board's Purpose section on or off. Members only."`
	Purpose        *string `json:"purpose,omitempty" jsonschema_description:"Markdown: what the board is for and how to work on it. Only while the Purpose section is on (409 purpose_off otherwise). Members only."`
	PurposeRev     *uint64 `json:"purpose_rev,omitempty" jsonschema_description:"The purpose_rev the purpose was read at: if it moved since, 409 board_text_stale instead of writing over someone's change."`
	Notes          *string `json:"notes,omitempty" jsonschema_description:"Markdown: links, documents, an index. Members only."`
	NotesRev       *uint64 `json:"notes_rev,omitempty" jsonschema_description:"The notes_rev the notes were read at, as purpose_rev."`
}

func ( in *updateBoardRequest ) settings() ( settings *models.BoardSettings ) {
	settings = &models.BoardSettings{
		Title: in.Title , DefaultView: in.DefaultView , Color: in.Color , Slug: in.Slug , Layout: in.Layout ,
		Numbering: in.Numbering , ListTitles: in.ListTitles , Theme: in.Theme , ClearTheme: in.ClearTheme ,
		Shopping: in.Shopping , ShowRank: in.ShowRank , CenterColumns: in.CenterColumns , StageTabs: in.StageTabs , ShowUpdated: in.ShowUpdated ,
		StageTodo: in.StageTodo , StageDoing: in.StageDoing , StageScheduled: in.StageScheduled , StageSet: in.StageSet , StageDone: in.StageDone ,
		AllowQuickShare: in.AllowQuickShare , AllowAnonComments: in.AllowAnonComments , CarryOver: in.CarryOver ,
		DonePlacement: in.DonePlacement , AddPlacement: in.AddPlacement ,
		PurposeEnabled: in.PurposeEnabled , Purpose: in.Purpose , PurposeRev: in.PurposeRev , Notes: in.Notes , NotesRev: in.NotesRev ,
	}
	return
}

type updateBoardReply struct {
	Rev        uint64 `json:"rev"`
	Slug       string `json:"slug"`
	PurposeRev uint64 `json:"purpose_rev"`
	NotesRev   uint64 `json:"notes_rev"`
}

type duplicateBoardRequest struct {
	boardPath
	Title string `json:"title" jsonschema:"maxLength=120" jsonschema_description:"The copy's title."`
}

// Archive scopes: for everyone on the board (the owner's), or off the
// caller's own home page.
const (
	archiveEveryone = "everyone"
	archiveMe       = "me"
)

type archiveBoardRequest struct {
	boardPath
	Archived *bool  `json:"archived" jsonschema_description:"true archives, false unarchives."`
	Scope    string `json:"scope" jsonschema:"enum=everyone,enum=me" jsonschema_description:"everyone (the owner only) or me (just your own home page)."`
}

// trashBoardPath names a board in the caller's board trash, by its number
// alone, as the old route read it.
type trashBoardPath struct {
	BoardID uint64 `json:"board_id" path:"board_id" jsonschema_description:"The deleted board's id, from GET /api/board-trash."`
}

type boardTrashReply struct {
	Boards []*models.TrashedBoard `json:"boards"`
}

type emptiedTrashReply struct {
	Removed int `json:"removed"`
}

type myViewRequest struct {
	boardPath
	PlainIcons *bool `json:"plain_icons,omitempty" jsonschema_description:"true for greyscale status icons on this board, false for coloured; null or left out follows your account default."`
}

type myViewReply struct {
	PlainIcons *bool `json:"plain_icons"`
}

type myCollapsedRequest struct {
	boardPath
	ListIDs []uint64 `json:"list_ids,omitempty" jsonschema_description:"Every column you keep collapsed on this board (the whole set; ids from get_board's board.lists). Empty expands them all."`
}

type myCollapsedReply struct {
	CollapsedLists []uint64 `json:"collapsed_lists" jsonschema_description:"The ids kept: ones that are not columns of this board are dropped."`
}

type createListRequest struct {
	boardPath
	Title    string   `json:"title" jsonschema:"maxLength=120"`
	Position *float64 `json:"position,omitempty" jsonschema_description:"Sort position among the columns; lower comes first. Leave out to put it at the end."`
}

type updateListRequest struct {
	listPath
	Title     *string  `json:"title,omitempty" jsonschema:"maxLength=120"`
	Position  *float64 `json:"position,omitempty" jsonschema_description:"Sort position among the columns; lower comes first."`
	Archived  *bool    `json:"archived,omitempty" jsonschema_description:"true hides the column; false brings it back."`
	Shopping  *bool    `json:"shopping,omitempty" jsonschema_description:"A shopping column: its cards carry products and prices."`
	Width     *int     `json:"width,omitempty" jsonschema_description:"The column's width in pixels (200-800); 0 puts it back to the standard width."`
	StageView *string  `json:"stage_view,omitempty" jsonschema:"enum=,enum=todo,enum=doing,enum=scheduled,enum=done" jsonschema_description:"On a staged board, the stage this column shows for everyone; empty follows the navbar."`
	StageTabs *string  `json:"stage_tabs,omitempty" jsonschema:"enum=,enum=on,enum=off" jsonschema_description:"On a staged board, this column's own To do / Doing / Done buttons: on, off, or empty to follow the board's setting."`
	ShowUpdated *string `json:"show_updated,omitempty" jsonschema:"enum=,enum=on,enum=off" jsonschema_description:"Whether this column's cards show when they were last changed: on, off, or empty to follow the board's show_updated."`
}

func ( in *updateListRequest ) update() ( update *models.ListUpdate ) {
	update = &models.ListUpdate{ Title: in.Title , Position: in.Position , Archived: in.Archived , Shopping: in.Shopping , Width: in.Width , StageView: in.StageView , StageTabs: in.StageTabs , ShowUpdated: in.ShowUpdated }
	return
}

type createLabelRequest struct {
	boardPath
	Name  *string `json:"name,omitempty" jsonschema:"maxLength=40"`
	Color string  `json:"color" jsonschema_description:"A palette name (blue, green, orange, red, purple, pink, teal, gray, yellow) or a #rrggbb colour."`
}

type updateLabelRequest struct {
	labelPath
	Name  *string `json:"name,omitempty" jsonschema:"maxLength=40"`
	Color *string `json:"color,omitempty" jsonschema_description:"A palette name (blue, green, orange, red, purple, pink, teal, gray, yellow) or a #rrggbb colour."`
}

type carryOverRequest struct {
	boardPath
	Today string `json:"today" jsonschema_description:"Your today, as 2026-11-07: within a day of the server's."`
}

type carryOverReply struct {
	Moved int `json:"moved" jsonschema_description:"How many cards moved to today."`
}

// listMyBoards is the home page: boards I own or am a member of, the folders
// I sort them into, and single cards shared with me from boards I am not on.
//
// @route("GET /api/boards", summary="List my boards",
//     tag="Boards",
//     description="Every board you own or are a member of (id, slug, title, your role, folder, and whether it is "
//         "archived for everyone -- read-only -- or archived_for_me), your home-page folders, and single cards "
//         "shared with you from boards you are not on.",
//     mcp="list_boards", read_only=True,
// )
func ( handlers *Handlers ) listMyBoards( c fiber.Ctx , in *apidoc.None ) ( reply *fiber.Map , err error ) {
	payload , err := handlers.myBoards( principalFrom( c ) )
	if err != nil { return }
	reply = &payload
	return
}

// myBoards is the home page's list. This is a list rather than one board,
// so a key's board limit is applied here by hand. A limited key gets no
// folders: they are home-page settings that span boards.
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

// createBoard makes a board owned by the caller.
//
// @route("POST /api/boards", summary="Create a board",
//     tag="Boards", errors=(403,),
//     description="A new board owned by you, with the given columns in order (for example [\"To do\", \"Doing\", "
//         "\"Done\"]); a board always gets at least three, so fewer are padded out. Returns its id and slug. "
//         "A key held to some boards cannot make one.",
//     mcp="create_board",
// )
func ( handlers *Handlers ) createBoard( c fiber.Ctx , in *createBoardRequest ) ( reply *newBoardReply , err error ) {
	if err = refuseOnlyBoardsKey( c ); err != nil { return }
	board , create_err := models.CreateBoard( handlers.Store , principalFrom( c ).UserID , in.Title , in.Lists )
	if create_err != nil {
		err = modelErr( create_err )
		return
	}
	reply = &newBoardReply{ ID: board.ID , Slug: board.Slug }
	return
}

// getBoard returns everything a board view needs in one response: the board,
// the cards this caller may see, the names behind every user id mentioned,
// and what this caller may do.
//
// @route("GET /api/boards/{board_id}", summary="Read a board",
//     tag="Boards", errors=(404,),
//     description="A board as you can see it: its notes, its purpose if the board has that section on "
//         "(board.purpose says what the board is for and how to work on it, so read it first), columns "
//         "(board.lists), labels, members, and the cards you may see, with each card's column, labels, "
//         "assignees, dates, checklist and rev. people maps user ids to names. access is what you may do.",
//     mcp_note="Archived cards are left out unless include_archived is true.",
//     mcp="get_board", read_only=True, call=getBoardTool,
// )
func ( handlers *Handlers ) getBoard( c fiber.Ctx , in *getBoardRequest ) ( reply *fiber.Map , err error ) {
	principal , board , err := handlers.boardAt( c , in.BoardID )
	if err != nil { return }
	cards , err := models.ListCards( handlers.Store , board.ID )
	if err != nil { return }
	if in.IncludeArchived != nil && *in.IncludeArchived == false {
		open := cards[ :0 ]
		for _ , card := range cards {
			if card.Archived == false { open = append( open , card ) }
		}
		cards = open
	}
	payload := handlers.boardPayload( principal , board , cards )
	reply = &payload
	return
}

// sharedGetBoard is getBoard for a share link's guest.
//
// @route("GET /api/shared/{board_id}", summary="Read a board (share link)",
//     tag="Shared", access=PUBLIC, use=[handlers.SharePrincipal], errors=(404,),
//     description="getBoard for a guest holding a share link in X-Share-Token: what the link shows.",
// )
func ( handlers *Handlers ) sharedGetBoard( c fiber.Ctx , in *getBoardRequest ) ( reply *fiber.Map , err error ) {
	reply , err = handlers.getBoard( c , in )
	return
}

// getBoardStamp is the cheap call a board view polls: the revision only.
//
// @route("GET /api/boards/{board_id}/stamp", summary="A board's revision",
//     tag="Boards", errors=(404,),
//     description="The board's revision alone, which the page polls to know when to read the board again.",
//     mcp_http="The board page's cheap change poll; get_board reads the board itself.",
// )
func ( handlers *Handlers ) getBoardStamp( c fiber.Ctx , in *boardPath ) ( reply *boardStampReply , err error ) {
	principal , board , err := handlers.boardAt( c , in.BoardID )
	if err != nil { return }
	rev := board.Rev
	// The board's revision moves with every other card, which is not a quick
	// link's to know: it hears about its own card.
	if access.QuickLink( principal ) {
		card , get_err := models.GetCard( handlers.Store , board.ID , principal.Share.CardID )
		if get_err != nil {
			err = modelErr( get_err )
			return
		}
		rev = card.Rev
	}
	reply = &boardStampReply{ Rev: rev }
	return
}

// sharedGetBoardStamp is getBoardStamp for a share link's guest.
//
// @route("GET /api/shared/{board_id}/stamp", summary="A board's revision (share link)",
//     tag="Shared", access=PUBLIC, use=[handlers.SharePrincipal], errors=(404,),
//     description="getBoardStamp for a guest holding a share link in X-Share-Token.",
// )
func ( handlers *Handlers ) sharedGetBoardStamp( c fiber.Ctx , in *boardPath ) ( reply *boardStampReply , err error ) {
	reply , err = handlers.getBoardStamp( c , in )
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
			"scheduled":   card.Scheduled,
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
	if columns_only && principal.Share.Mode == models.ModeEdit && board.Archived() == false { reported = access.Edit }

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
			me[ "archived" ] = ref.Archived
			me[ "google_sync" ] = ref.GoogleSync
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
		"stage_tabs":   board.StageTabs,
		"show_updated": board.ShowUpdated,
		"stage_todo":   board.StageTodo,
		"stage_doing":  board.StageDoing,
		"stage_scheduled": board.StageScheduled,
		"stage_set":    board.StageSet,
		"stage_done":   board.StageDone,
		"theme":        board.Theme,
		"rev":          board.Rev,
		"lists":        lists,
		"labels":       board.Labels,
		"members":      members,
		// Archived for everyone: the page shows it read-only, with the
		// owner's Unarchive.
		"archived_at":  board.ArchivedAt,
		"archived_by":  board.ArchivedBy,
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
		"scheduled":   card.Scheduled,
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

// updateBoard changes a board's settings: everything Board settings shows.
//
// @route("POST /api/boards/{board_id}", summary="Change a board's settings",
//     tag="Boards", errors=(403, 404, 409),
//     description="Change any of a board's settings; leave out what should not change: its title, URL name "
//         "(slug), layout (free, three, staged or calendar), default view, colour, theme, card numbering, "
//         "where finished cards and the add-card box go, the stage names and buttons of a staged board, "
//         "carry-over on a calendar board, shopping, ranking, and its purpose and notes. Needs edit rights "
//         "on the whole board. The purpose and notes are for members, not share links; the quick-share and "
//         "anonymous-comment switches are the owner's, from a browser. Answers the board's new revs.",
//     mcp_note="To change only the purpose or notes, update_board_notes is the safer call: it insists on the rev "
//         "you read them at.",
//     mcp="update_board",
// )
func ( handlers *Handlers ) updateBoard( c fiber.Ctx , in *updateBoardRequest ) ( reply *updateBoardReply , err error ) {
	principal , board , err := handlers.requireBoardAt( c , in.BoardID , access.Edit )
	if err != nil { return }
	// The purpose is what agents act on, so it is for the board's members: an
	// edit link is handed around for changing cards, not for instructing them.
	if principal.IsGuest() && ( in.PurposeEnabled != nil || in.Purpose != nil || in.Notes != nil ) {
		err = refuse( fiber.StatusForbidden , "forbidden" )
		return
	}
	// Opening a board's cards to the public is the owner's call, made in a
	// browser: a leaked key cannot switch paused links back on.
	if ( in.AllowQuickShare != nil || in.AllowAnonComments != nil ) && ( access.Board( principal , board ) != access.Manage || principal.Key != nil ) {
		err = refuse( fiber.StatusForbidden , "forbidden" )
		return
	}
	before := *board
	updated , update_err := models.UpdateBoardSettings( handlers.Store , board.ID , in.settings() )
	if errors.Is( update_err , models.ErrSlugTaken ) {
		err = refuse( fiber.StatusConflict , "slug_taken" )
		return
	}
	if update_err != nil {
		err = modelErr( update_err )
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
	reply = &updateBoardReply{ Rev: updated.Rev , Slug: updated.Slug , PurposeRev: updated.PurposeRev , NotesRev: updated.NotesRev }
	return
}

// sharedUpdateBoard is updateBoard for a share link's guest.
//
// @route("POST /api/shared/{board_id}", summary="Change a board's settings (share link)",
//     tag="Shared", access=PUBLIC, use=[handlers.SharePrincipal], errors=(403, 404, 409),
//     description="updateBoard for a guest holding an edit share link in X-Share-Token. Never the purpose or notes.",
// )
func ( handlers *Handlers ) sharedUpdateBoard( c fiber.Ctx , in *updateBoardRequest ) ( reply *updateBoardReply , err error ) {
	reply , err = handlers.updateBoard( c , in )
	return
}

// boardTextActivity reports the board's words changing. Its look is left out:
// a theme or colour is not news.
func ( handlers *Handlers ) boardTextActivity( principal *access.Principal , before *models.Board , after *models.Board ) {
	if before.Title != after.Title { handlers.activity( principal , after.ID , "board_renamed" , map[string]string{ "title": after.Title } ) }
	if before.Notes != after.Notes { handlers.activity( principal , after.ID , "board_notes" , nil ) }
	if before.Purpose != after.Purpose { handlers.activity( principal , after.ID , "board_purpose" , nil ) }
}

// duplicateChunk is how many cards go into the copy per transaction, kept
// under the import limit in server/models/snapshot.go.
const duplicateChunk = 1000

// duplicateBoard is a new board, owned by the caller, holding what they can
// see of this one -- the same things a backup carries (settings, notes,
// labels, lists, cards and their checklists), so no comments, pictures,
// members or share links.
//
// @route("POST /api/boards/{board_id}/duplicate", summary="Duplicate a board",
//     tag="Boards", errors=(403, 404),
//     description="A new board owned by you, copied from one you can open: its settings, notes, labels, columns, "
//         "and the cards you can see with their checklists. Comments, card pictures, members and share links "
//         "are not copied. Returns the copy's id and slug. A key held to some boards cannot make one.",
//     mcp="duplicate_board",
// )
func ( handlers *Handlers ) duplicateBoard( c fiber.Ctx , in *duplicateBoardRequest ) ( reply *newBoardReply , err error ) {
	if err = refuseOnlyBoardsKey( c ); err != nil { return }
	principal , board , err := handlers.requireBoardAt( c , in.BoardID , access.View )
	if err != nil { return }
	copied , refused := handlers.copyBoard( principal , board , in.Title )
	if refused != nil {
		err = refused
		return
	}
	reply = &newBoardReply{ ID: copied.ID , Slug: copied.Slug }
	return
}

// copyBoard is the backup import run straight from a snapshot; a copy that
// fails partway is removed again.
func ( handlers *Handlers ) copyBoard( principal *access.Principal , board *models.Board , title string ) ( copied *models.Board , refused *refusal ) {
	cards , list_err := models.ListCards( handlers.Store , board.ID )
	if list_err != nil {
		refused = internalFailure( list_err )
		return
	}
	visible := func( card *models.Card ) bool { return access.Card( principal , board , card ) != access.None }
	snap := models.SnapshotBoard( board , cards , visible , access.ListOnly( principal , board ) , handlers.personNamer() )
	snap.Title = title
	snap.Slug = ""
	copied , lists , import_err := models.ImportBoardSkeleton( handlers.Store , principal.UserID , snap )
	if import_err != nil {
		refused = modelRefusal( import_err )
		return
	}
	for index , list := range snap.Lists {
		for start := 0 ; start < len( list.Cards ) ; start += duplicateChunk {
			end := min( start + duplicateChunk , len( list.Cards ) )
			if _ , cards_err := models.ImportCards( handlers.Store , copied.ID , lists[ index ] , snap.Labels , list.Cards[ start:end ] , principal.UserID , principal.Name ) ; cards_err != nil {
				models.PurgeBoard( handlers.Store , copied.ID )
				copied = nil
				refused = modelRefusal( cards_err )
				return
			}
		}
	}
	return
}

// deleteBoard moves the board to its owner's trash, archived or not. It
// still needs the owner and a browser session: a leaked API key must not be
// able to take a board away from everyone on it, even for a while.
//
// @route("POST /api/boards/{board_id}/delete", summary="Delete a board",
//     tag="Boards", access=SESSION_ONLY, errors=(403, 404),
//     description="Move one of your own boards to your board trash, where it stays until you restore it or "
//         "delete it for good. The owner only, from a browser.",
// )
func ( handlers *Handlers ) deleteBoard( c fiber.Ctx , in *boardPath ) ( reply *okReply , err error ) {
	principal , board , err := handlers.boardAt( c , in.BoardID )
	if err != nil { return }
	if access.Owner( principal , board ) == false {
		err = refuse( fiber.StatusForbidden , "forbidden" )
		return
	}
	if trash_err := models.TrashBoard( handlers.Store , board.ID , principal.UserID ); trash_err != nil {
		err = modelErr( trash_err )
		return
	}
	reply = &okReply{ Ok: true }
	return
}

// archiveBoard archives or unarchives a board. For everyone it is the
// owner's, and works on an archived board -- it is how one is unarchived.
// For me, anyone on the board, but not a key held to some boards: like
// folders, it is the person's own home page.
//
// @route("POST /api/boards/{board_id}/archive", summary="Archive or unarchive a board",
//     tag="Boards", errors=(400, 403, 404),
//     description="scope \"everyone\" (the owner only): the board leaves every member's home page and becomes "
//         "read-only -- every change is refused with board_archived -- until it is unarchived the same way. "
//         "scope \"me\" (anyone on the board, but not a key held to some boards): it only leaves your own home "
//         "page. list_boards says which applies (archived, archived_for_me).",
//     mcp="archive_board",
// )
func ( handlers *Handlers ) archiveBoard( c fiber.Ctx , in *archiveBoardRequest ) ( reply *okReply , err error ) {
	if in.Archived == nil || ( in.Scope != archiveEveryone && in.Scope != archiveMe ) {
		err = refuse( fiber.StatusBadRequest , "invalid_input" )
		return
	}
	principal , board , err := handlers.boardAt( c , in.BoardID )
	if err != nil { return }
	var model_err error
	switch {
	case principal.IsGuest() || access.Board( principal , board ) == access.None:
		err = refuse( fiber.StatusForbidden , "forbidden" )
	case in.Scope == archiveEveryone && access.Owner( principal , board ) == false:
		err = refuse( fiber.StatusForbidden , "forbidden" )
	case in.Scope == archiveMe && principal.Key.Limited():
		err = refuse( fiber.StatusForbidden , "forbidden" )
	case in.Scope == archiveEveryone:
		_ , model_err = models.SetBoardArchived( handlers.Store , board.ID , *in.Archived , principal.Name )
	default:
		model_err = models.SetBoardArchivedFor( handlers.Store , principal.UserID , board.ID , *in.Archived )
	}
	if model_err != nil { err = modelErr( model_err ) }
	if err != nil { return }
	reply = &okReply{ Ok: true }
	return
}

// The board trash: the caller's own boards they deleted, which stay until
// restored or deleted for good. A browser session only, like delete. Its
// own prefix, not under /api/boards, so no /{board_id} route there can
// match "trash".

// listBoardTrash is the caller's deleted boards.
//
// @route("GET /api/board-trash", summary="Your deleted boards",
//     tag="Trash", access=SESSION_ONLY,
//     description="The boards you deleted, which stay until you restore them or delete them for good.",
// )
func ( handlers *Handlers ) listBoardTrash( c fiber.Ctx , in *apidoc.None ) ( reply *boardTrashReply , err error ) {
	boards , err := models.ListTrashedBoards( handlers.Store , principalFrom( c ).UserID )
	if err != nil { return }
	reply = &boardTrashReply{ Boards: boards }
	return
}

// restoreBoard takes one of the caller's boards back out of their trash.
//
// @route("POST /api/board-trash/{board_id}/restore", summary="Restore a deleted board",
//     tag="Trash", access=SESSION_ONLY, errors=(404,),
//     description="Bring one of your deleted boards back, for everyone who was on it. Answers its id and slug.",
// )
func ( handlers *Handlers ) restoreBoard( c fiber.Ctx , in *trashBoardPath ) ( reply *newBoardReply , err error ) {
	board , restore_err := models.RestoreBoard( handlers.Store , principalFrom( c ).UserID , in.BoardID )
	if restore_err != nil {
		err = modelErr( restore_err )
		return
	}
	reply = &newBoardReply{ ID: board.ID , Slug: board.Slug }
	return
}

// purgeBoard deletes one board in the caller's trash for good.
//
// @route("POST /api/board-trash/{board_id}/delete", summary="Delete a board for good",
//     tag="Trash", access=SESSION_ONLY, errors=(404,),
//     description="Delete one of your deleted boards for good, with its cards, comments and pictures. "
//         "This cannot be undone.",
// )
func ( handlers *Handlers ) purgeBoard( c fiber.Ctx , in *trashBoardPath ) ( reply *okReply , err error ) {
	if purge_err := models.PurgeTrashedBoard( handlers.Store , principalFrom( c ).UserID , in.BoardID ); purge_err != nil {
		err = modelErr( purge_err )
		return
	}
	reply = &okReply{ Ok: true }
	return
}

// emptyBoardTrash deletes every board in the caller's trash for good.
//
// @route("POST /api/board-trash/empty", summary="Empty your board trash",
//     tag="Trash", access=SESSION_ONLY,
//     description="Delete every board in your board trash for good. This cannot be undone. Answers how many went.",
// )
func ( handlers *Handlers ) emptyBoardTrash( c fiber.Ctx , in *apidoc.None ) ( reply *emptiedTrashReply , err error ) {
	removed , empty_err := models.EmptyBoardTrash( handlers.Store , principalFrom( c ).UserID )
	if empty_err != nil {
		err = modelErr( empty_err )
		return
	}
	reply = &emptiedTrashReply{ Removed: removed }
	return
}

// setMyBoardView saves how this board looks to the caller alone -- greyscale
// status icons, for now. Only people on the board have somewhere to keep it.
//
// @route("POST /api/boards/{board_id}/my-view", summary="Your own view of a board",
//     tag="Boards", errors=(403, 404),
//     description="How a board looks to you alone: greyscale status icons (plain_icons), or null to follow "
//         "your account default. Nobody else on the board sees it.",
//     mcp="set_my_board_view",
// )
func ( handlers *Handlers ) setMyBoardView( c fiber.Ctx , in *myViewRequest ) ( reply *myViewReply , err error ) {
	principal , board , err := handlers.requireBoardAt( c , in.BoardID , access.View )
	if err != nil { return }
	if set_err := models.SetBoardPlainIcons( handlers.Store , principal.UserID , board.ID , in.PlainIcons ); set_err != nil {
		err = modelErr( set_err )
		return
	}
	reply = &myViewReply{ PlainIcons: in.PlainIcons }
	return
}

// setMyCollapsedLists saves which columns the caller keeps collapsed: the
// whole list each time. Its own route so that saving it cannot touch the rest
// of how the board looks to them.
//
// @route("POST /api/boards/{board_id}/my-view/collapsed", summary="Your collapsed columns",
//     tag="Boards", errors=(403, 404),
//     description="Which columns of a board you keep collapsed, for you alone: the whole set each time.",
//     mcp="set_my_collapsed_lists",
// )
func ( handlers *Handlers ) setMyCollapsedLists( c fiber.Ctx , in *myCollapsedRequest ) ( reply *myCollapsedReply , err error ) {
	principal , board , err := handlers.requireBoardAt( c , in.BoardID , access.View )
	if err != nil { return }
	kept , set_err := models.SetBoardCollapsedLists( handlers.Store , principal.UserID , board , in.ListIDs )
	if set_err != nil {
		err = modelErr( set_err )
		return
	}
	reply = &myCollapsedReply{ CollapsedLists: kept }
	return
}

// leaveBoard lets a member remove themselves. The owner cannot leave; they
// delete the board or hand it over instead.
//
// @route("POST /api/boards/{board_id}/leave", summary="Leave a board",
//     tag="Boards", errors=(400, 403, 404),
//     description="Take yourself off a board you are a member of; it leaves your home page and you lose access "
//         "until someone adds you again. The owner cannot leave (400 owner_cannot_leave).",
//     mcp="leave_board", destructive=True,
// )
func ( handlers *Handlers ) leaveBoard( c fiber.Ctx , in *boardPath ) ( reply *okReply , err error ) {
	principal , board , err := handlers.requireBoardAt( c , in.BoardID , access.View )
	if err != nil { return }
	if board.OwnerID == principal.UserID {
		err = refuse( fiber.StatusBadRequest , "owner_cannot_leave" )
		return
	}
	if _ , set_err := models.SetMember( handlers.Store , board.ID , principal.UserID , "" ); set_err != nil {
		err = modelErr( set_err )
		return
	}
	handlers.activity( principal , board.ID , "member_left" , nil )
	reply = &okReply{ Ok: true }
	return
}

// createList adds a column.
//
// @route("POST /api/boards/{board_id}/lists", summary="Add a column",
//     tag="Boards", errors=(403, 404, 409),
//     description="Add a column (list) to a board, at the end or at position. Needs edit rights on the whole board.",
//     mcp="create_list",
// )
func ( handlers *Handlers ) createList( c fiber.Ctx , in *createListRequest ) ( reply *models.List , err error ) {
	principal , board , err := handlers.requireBoardAt( c , in.BoardID , access.Edit )
	if err != nil { return }
	_ , update_err := models.MutateBoard( handlers.Store , board.ID , func( target *models.Board ) ( mutate_err error ) {
		reply , mutate_err = models.AddList( target , in.Title , in.Position )
		return
	} )
	if update_err != nil {
		reply = nil
		err = modelErr( update_err )
		return
	}
	handlers.activity( principal , board.ID , "list_added" , map[string]string{ "list": reply.Title } )
	return
}

// sharedCreateList is createList for a share link's guest.
//
// @route("POST /api/shared/{board_id}/lists", summary="Add a column (share link)",
//     tag="Shared", access=PUBLIC, use=[handlers.SharePrincipal], errors=(403, 404, 409),
//     description="createList for a guest holding an edit share link in X-Share-Token.",
// )
func ( handlers *Handlers ) sharedCreateList( c fiber.Ctx , in *createListRequest ) ( reply *models.List , err error ) {
	reply , err = handlers.createList( c , in )
	return
}

// updateList changes a column; the activity feed hears of renames and
// archiving.
//
// @route("POST /api/boards/{board_id}/lists/{list_id}", summary="Rename, move or archive a column",
//     tag="Boards", errors=(403, 404, 409),
//     description="Change a column's title, position or width, archive it (archived: true hides it; false brings "
//         "it back), make it a shopping column, or on a staged board fix the stage it shows for everyone "
//         "(stage_view; empty follows the navbar) or give it its own To do/Doing/Done buttons (stage_tabs: on, "
//         "off, or empty to follow the board's setting). Leave out what should not change. Needs edit rights "
//         "on the whole board.",
//     mcp="update_list",
// )
func ( handlers *Handlers ) updateList( c fiber.Ctx , in *updateListRequest ) ( reply *okReply , err error ) {
	principal , board , err := handlers.requireBoardAt( c , in.BoardID , access.Edit )
	if err != nil { return }
	if in.ListID == 0 {
		err = refuse( fiber.StatusNotFound , "list_not_found" )
		return
	}
	// A copy: the board passed in is not reread, and Apply is what changes.
	before := models.List{}
	if list := board.List( in.ListID ); list != nil { before = *list }
	update := in.update()
	updated , update_err := models.MutateBoard( handlers.Store , board.ID , func( target *models.Board ) error {
		return update.Apply( target , in.ListID )
	} )
	if update_err != nil {
		err = modelErr( update_err )
		return
	}
	reply = &okReply{ Ok: true }
	after := updated.List( in.ListID )
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

// sharedUpdateList is updateList for a share link's guest.
//
// @route("POST /api/shared/{board_id}/lists/{list_id}", summary="Rename, move or archive a column (share link)",
//     tag="Shared", access=PUBLIC, use=[handlers.SharePrincipal], errors=(403, 404, 409),
//     description="updateList for a guest holding an edit share link in X-Share-Token.",
// )
func ( handlers *Handlers ) sharedUpdateList( c fiber.Ctx , in *updateListRequest ) ( reply *okReply , err error ) {
	reply , err = handlers.updateList( c , in )
	return
}

// deleteList removes a column and the cards in it.
//
// @route("POST /api/boards/{board_id}/lists/{list_id}/delete", summary="Delete a column",
//     tag="Boards", errors=(403, 404, 409),
//     description="Delete a column and its share links; its cards go to the board's trash, where they can be restored into another column. Archiving it (update_list, archived: true) "
//         "is the gentler way to put one away. Needs edit rights on the whole board.",
//     mcp="delete_list", destructive=True,
// )
func ( handlers *Handlers ) deleteList( c fiber.Ctx , in *listPath ) ( reply *okReply , err error ) {
	principal , board , err := handlers.requireBoardAt( c , in.BoardID , access.Edit )
	if err != nil { return }
	if in.ListID == 0 {
		err = refuse( fiber.StatusNotFound , "list_not_found" )
		return
	}
	title := ""
	if list := board.List( in.ListID ); list != nil { title = list.Title }
	if delete_err := models.DeleteList( handlers.Store , board.ID , in.ListID , principal.Name ); delete_err != nil {
		err = modelErr( delete_err )
		return
	}
	handlers.activity( principal , board.ID , "list_deleted" , map[string]string{ "list": title } )
	reply = &okReply{ Ok: true }
	return
}

// sharedDeleteList is deleteList for a share link's guest.
//
// @route("POST /api/shared/{board_id}/lists/{list_id}/delete", summary="Delete a column (share link)",
//     tag="Shared", access=PUBLIC, use=[handlers.SharePrincipal], errors=(403, 404, 409),
//     description="deleteList for a guest holding an edit share link in X-Share-Token.",
// )
func ( handlers *Handlers ) sharedDeleteList( c fiber.Ctx , in *listPath ) ( reply *okReply , err error ) {
	reply , err = handlers.deleteList( c , in )
	return
}

// createLabel adds a label to the board.
//
// @route("POST /api/boards/{board_id}/labels", summary="Add a label",
//     tag="Boards", errors=(403, 404, 409),
//     description="Add a label to a board, to put on cards with update_card's label_ids. Needs edit rights on "
//         "the whole board.",
//     mcp="create_label",
// )
func ( handlers *Handlers ) createLabel( c fiber.Ctx , in *createLabelRequest ) ( reply *models.Label , err error ) {
	_ , board , err := handlers.requireBoardAt( c , in.BoardID , access.Edit )
	if err != nil { return }
	name := ""
	if in.Name != nil { name = *in.Name }
	_ , update_err := models.MutateBoard( handlers.Store , board.ID , func( target *models.Board ) ( mutate_err error ) {
		reply , mutate_err = models.AddLabel( target , name , in.Color )
		return
	} )
	if update_err != nil {
		reply = nil
		err = modelErr( update_err )
	}
	return
}

// sharedCreateLabel is createLabel for a share link's guest.
//
// @route("POST /api/shared/{board_id}/labels", summary="Add a label (share link)",
//     tag="Shared", access=PUBLIC, use=[handlers.SharePrincipal], errors=(403, 404, 409),
//     description="createLabel for a guest holding an edit share link in X-Share-Token.",
// )
func ( handlers *Handlers ) sharedCreateLabel( c fiber.Ctx , in *createLabelRequest ) ( reply *models.Label , err error ) {
	reply , err = handlers.createLabel( c , in )
	return
}

// updateLabel renames or recolours a label.
//
// @route("POST /api/boards/{board_id}/labels/{label_id}", summary="Change a label",
//     tag="Boards", errors=(403, 404, 409),
//     description="Rename a label or change its colour; leave out what should not change. Needs edit rights on "
//         "the whole board.",
//     mcp="update_label",
// )
func ( handlers *Handlers ) updateLabel( c fiber.Ctx , in *updateLabelRequest ) ( reply *okReply , err error ) {
	_ , board , err := handlers.requireBoardAt( c , in.BoardID , access.Edit )
	if err != nil { return }
	if in.LabelID == 0 {
		err = refuse( fiber.StatusNotFound , "label_not_found" )
		return
	}
	_ , update_err := models.MutateBoard( handlers.Store , board.ID , func( target *models.Board ) error {
		return models.UpdateLabel( target , in.LabelID , in.Name , in.Color )
	} )
	if update_err != nil {
		err = modelErr( update_err )
		return
	}
	reply = &okReply{ Ok: true }
	return
}

// sharedUpdateLabel is updateLabel for a share link's guest.
//
// @route("POST /api/shared/{board_id}/labels/{label_id}", summary="Change a label (share link)",
//     tag="Shared", access=PUBLIC, use=[handlers.SharePrincipal], errors=(403, 404, 409),
//     description="updateLabel for a guest holding an edit share link in X-Share-Token.",
// )
func ( handlers *Handlers ) sharedUpdateLabel( c fiber.Ctx , in *updateLabelRequest ) ( reply *okReply , err error ) {
	reply , err = handlers.updateLabel( c , in )
	return
}

// deleteLabel removes a label from the board and from every card.
//
// @route("POST /api/boards/{board_id}/labels/{label_id}/delete", summary="Delete a label",
//     tag="Boards", errors=(403, 404, 409),
//     description="Delete a label; it comes off every card it was on. Needs edit rights on the whole board.",
//     mcp="delete_label", destructive=True,
// )
func ( handlers *Handlers ) deleteLabel( c fiber.Ctx , in *labelPath ) ( reply *okReply , err error ) {
	_ , board , err := handlers.requireBoardAt( c , in.BoardID , access.Edit )
	if err != nil { return }
	if in.LabelID == 0 {
		err = refuse( fiber.StatusNotFound , "label_not_found" )
		return
	}
	if delete_err := models.DeleteLabel( handlers.Store , board.ID , in.LabelID ); delete_err != nil {
		err = modelErr( delete_err )
		return
	}
	reply = &okReply{ Ok: true }
	return
}

// sharedDeleteLabel is deleteLabel for a share link's guest.
//
// @route("POST /api/shared/{board_id}/labels/{label_id}/delete", summary="Delete a label (share link)",
//     tag="Shared", access=PUBLIC, use=[handlers.SharePrincipal], errors=(403, 404, 409),
//     description="deleteLabel for a guest holding an edit share link in X-Share-Token.",
// )
func ( handlers *Handlers ) sharedDeleteLabel( c fiber.Ctx , in *labelPath ) ( reply *okReply , err error ) {
	reply , err = handlers.deleteLabel( c , in )
	return
}

// carryOver moves a calendar board's unfinished cards from past days to
// today, the caller's today. The page asks on opening the board, so
// whoever opens it first that day does it for everyone; asking again moves
// nothing. A board with carry-over off, or not a calendar, moves nothing.
//
// @route("POST /api/boards/{board_id}/carry-over", summary="Carry unfinished cards over to today",
//     tag="Boards", errors=(403, 404, 409),
//     description="On a calendar board with carry-over on, move the unfinished cards of past days to today "
//         "(yours, within a day of the server's). Asking again the same day moves nothing; a board with "
//         "carry-over off, or not a calendar, moves nothing. Needs edit rights on the whole board.",
//     mcp="carry_over",
// )
func ( handlers *Handlers ) carryOver( c fiber.Ctx , in *carryOverRequest ) ( reply *carryOverReply , err error ) {
	principal , board , err := handlers.requireBoardAt( c , in.BoardID , access.Edit )
	if err != nil { return }
	moved , carry_err := models.CarryOverCards( handlers.Store , board.ID , in.Today , principal.Name )
	if carry_err != nil {
		err = modelErr( carry_err )
		return
	}
	if moved > 0 { handlers.googleTouch( board.ID ) }
	reply = &carryOverReply{ Moved: moved }
	return
}

// sharedCarryOver is carryOver for a share link's guest.
//
// @route("POST /api/shared/{board_id}/carry-over", summary="Carry unfinished cards over to today (share link)",
//     tag="Shared", access=PUBLIC, use=[handlers.SharePrincipal], errors=(403, 404, 409),
//     description="carryOver for a guest holding an edit share link in X-Share-Token.",
// )
func ( handlers *Handlers ) sharedCarryOver( c fiber.Ctx , in *carryOverRequest ) ( reply *carryOverReply , err error ) {
	reply , err = handlers.carryOver( c , in )
	return
}
