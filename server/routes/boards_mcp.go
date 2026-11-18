package routes

import (
	sort "sort"
	strconv "strconv"
	strings "strings"
	time "time"

	fiber "github.com/gofiber/fiber/v3"

	access "todoboard/server/access"
	mcp "todoboard/server/mcp"
	models "todoboard/server/models"
)

// The small routes the board tools stand on, which the page does not call:
// a board's notes alone, changing them safely, and searching cards. They
// were MCP-only tools; as routes, a key can call them over HTTP too.

// getBoardTool is get_board: the route, with archived cards left out unless
// the agent asks for them. The page gets them by default; an agent mostly
// wants what is open.
func getBoardTool( call *mcp.Call , in *getBoardRequest , serve func( fiber.Ctx , *getBoardRequest ) ( *fiber.Map , error ) ) ( result any , err error ) {
	if in.IncludeArchived == nil {
		include := false
		in.IncludeArchived = &include
	}
	result , err = mcp.Answer( serve( call.Ctx , in ) )
	return
}

// getBoardNotes is a board's words without its cards.
//
// @route("GET /api/boards/{board_id}/notes", summary="Read a board's purpose and notes",
//     tag="Boards", errors=(404,),
//     description="Just the board's notes and purpose, without its cards. notes holds links, documents and "
//         "reference material. purpose says what the board is for and how to work on it, and is only there "
//         "when purpose_enabled is true. Both are markdown, and may be empty. Read them before working on a "
//         "board. The revs go back to update_board_notes.",
//     mcp="get_board_notes", read_only=True,
// )
func ( handlers *Handlers ) getBoardNotes( c fiber.Ctx , in *boardPath ) ( reply *fiber.Map , err error ) {
	_ , board , err := handlers.requireBoardAt( c , in.BoardID , access.View )
	if err != nil { return }
	payload := boardTextPayload( board )
	reply = &payload
	return
}

type updateBoardNotesRequest struct {
	boardPath
	Purpose    *string `json:"purpose,omitempty" jsonschema_description:"Markdown: what the board is for and how to work on it. The whole new text."`
	PurposeRev *uint64 `json:"purpose_rev,omitempty" jsonschema_description:"Required with purpose: the purpose_rev it was read at."`
	Notes      *string `json:"notes,omitempty" jsonschema_description:"Markdown: links, documents, an index. The whole new text."`
	NotesRev   *uint64 `json:"notes_rev,omitempty" jsonschema_description:"Required with notes: the notes_rev they were read at."`
}

type boardNotesReply struct {
	PurposeRev uint64 `json:"purpose_rev"`
	NotesRev   uint64 `json:"notes_rev"`
}

// updateBoardNotes is updateBoard for the purpose and notes alone, with the
// revs required: a change through a key has no undo, so it must be made on
// top of what is there.
//
// @route("POST /api/boards/{board_id}/notes", summary="Change a board's purpose or notes",
//     tag="Boards", errors=(400, 403, 404, 409),
//     description="Replace a board's notes, its purpose, or both; leave out what should not change. A board's "
//         "purpose can only be written while its Purpose section is on, which a person does in Board settings "
//         "(409 purpose_off otherwise). Each is the whole new markdown text, so to add a link, send the old "
//         "notes with it added. Each needs the rev it was read at, from get_board_notes or get_board: if "
//         "someone edited the text since, the change is refused (409 board_text_stale) rather than written "
//         "over theirs; read it again and redo the change on top. Needs edit rights on the board.",
//     mcp="update_board_notes",
// )
func ( handlers *Handlers ) updateBoardNotes( c fiber.Ctx , in *updateBoardNotesRequest ) ( reply *boardNotesReply , err error ) {
	if in.Purpose == nil && in.Notes == nil {
		err = refuse( fiber.StatusBadRequest , "send purpose, notes, or both" )
		return
	}
	if ( in.Purpose != nil && in.PurposeRev == nil ) || ( in.Notes != nil && in.NotesRev == nil ) {
		err = refuse( fiber.StatusBadRequest , "purpose needs purpose_rev and notes needs notes_rev; read them with get_board_notes" )
		return
	}
	principal , board , err := handlers.requireBoardAt( c , in.BoardID , access.Edit )
	if err != nil { return }
	settings := &models.BoardSettings{ Purpose: in.Purpose , PurposeRev: in.PurposeRev , Notes: in.Notes , NotesRev: in.NotesRev }
	before := *board
	updated , update_err := models.UpdateBoardSettings( handlers.Store , board.ID , settings )
	if update_err != nil {
		err = modelErr( update_err )
		return
	}
	handlers.boardTextActivity( principal , &before , updated )
	reply = &boardNotesReply{ PurposeRev: updated.PurposeRev , NotesRev: updated.NotesRev }
	return
}

type findCardsRequest struct {
	Query           string `json:"query,omitempty" query:"query" jsonschema_description:"Matches title and description, in any case."`
	BoardID         boardRef `json:"board,omitempty" query:"board" jsonschema_description:"Search one board only: its id or URL slug, from list_boards."`
	AssignedToMe    bool   `json:"assigned_to_me,omitempty" query:"assigned_to_me" jsonschema_description:"Only cards assigned to you."`
	DueBefore       string `json:"due_before,omitempty" query:"due_before" jsonschema_description:"Only cards due before this: \"2026-10-30\" or an RFC 3339 timestamp."`
	IncludeDone     bool   `json:"include_done,omitempty" query:"include_done" jsonschema_description:"Include done cards."`
	IncludeArchived bool   `json:"include_archived,omitempty" query:"include_archived" jsonschema_description:"Include archived cards."`
	Limit           int    `json:"limit,omitempty" query:"limit" jsonschema:"minimum=1,maximum=200,default=50" jsonschema_description:"At most this many rows (default 50)."`
}

// findCards walks the boards the caller is on, plus the boards of cards
// shared with them, and keeps the cards access.Card lets them see. It is
// a scan, which is fine at the size a team's boards reach.
//
// @route("GET /api/cards", summary="Find cards",
//     tag="Cards", errors=(400, 404),
//     description="Search the cards you can see, across all your boards or one (board_id). query matches title "
//         "and description (case-insensitive). Done and archived cards are left out unless asked for. Returns "
//         "compact rows, soonest due first, with total and whether the list was cut at limit; use get_card for "
//         "the full card.",
//     mcp="find_cards", read_only=True,
// )
func ( handlers *Handlers ) findCards( c fiber.Ctx , in *findCardsRequest ) ( reply *fiber.Map , err error ) {
	principal := principalFrom( c )
	limit := in.Limit
	if limit <= 0 { limit = 50 }
	if limit > 200 { limit = 200 }
	var due_before *time.Time
	if strings.TrimSpace( in.DueBefore ) != "" {
		parsed , _ , parse_err := models.ParseCardDate( in.DueBefore )
		if parse_err != nil {
			err = refuse( fiber.StatusBadRequest , `due_before must be "2026-10-30" or an RFC 3339 timestamp` )
			return
		}
		due_before = parsed
	}
	query := strings.ToLower( strings.TrimSpace( in.Query ) )

	var boards []*models.Board
	if in.BoardID != "" {
		_ , board , board_err := handlers.boardAt( c , in.BoardID )
		if board_err != nil {
			err = board_err
			return
		}
		boards = append( boards , board )
	} else {
		summaries , shared , list_err := models.ListBoardsForUser( handlers.Store , principal.UserID )
		if list_err != nil {
			err = list_err
			return
		}
		seen := map[uint64]bool{}
		ids := []uint64{}
		for _ , summary := range summaries { ids = append( ids , summary.ID ) }
		for _ , card := range shared { ids = append( ids , card.BoardID ) }
		for _ , id := range ids {
			if seen[ id ] { continue }
			seen[ id ] = true
			if board , refused := handlers.loadBoard( principal , strconv.FormatUint( id , 10 ) ); refused == nil {
				boards = append( boards , board )
			}
		}
	}

	type hit struct {
		row fiber.Map
		due *time.Time
	}
	hits := []hit{}
	for _ , board := range boards {
		cards , list_err := models.ListCards( handlers.Store , board.ID )
		if list_err != nil {
			err = list_err
			return
		}
		places := models.RankPlaces( board , cards )
		for _ , card := range cards {
			if access.Card( principal , board , card ) == access.None { continue }
			if card.Archived && in.IncludeArchived == false { continue }
			if card.Done && in.IncludeDone == false { continue }
			if in.AssignedToMe && containsID( card.Assignees , principal.UserID ) == false { continue }
			if due_before != nil && ( card.DueAt == nil || card.DueAt.Before( *due_before ) == false ) { continue }
			if query != "" && strings.Contains( strings.ToLower( card.Title ) , query ) == false && strings.Contains( strings.ToLower( card.Description ) , query ) == false { continue }
			list_title := ""
			if list := board.List( card.ListID ); list != nil { list_title = list.Title }
			hits = append( hits , hit{ due: card.DueAt , row: fiber.Map{
				"board_id":    board.ID,
				"board_title": board.Title,
				"card_id":     card.ID,
				"serial":      card.Serial,
				"uid":         card.UID,
				"list_id":     card.ListID,
				"list_title":  list_title,
				"title":       card.Title,
				"due_at":      card.DueAt,
				"done":        card.Done,
				"doing":       card.Doing,
				"scheduled":   card.Scheduled,
				"rank_place":  places[ card.ID ],
				"archived":    card.Archived,
				"assignees":   card.Assignees,
			} } )
		}
	}
	// Soonest due first, undated last; stable so equal dates keep board order.
	sort.SliceStable( hits , func( i int , j int ) bool {
		a , b := hits[ i ].due , hits[ j ].due
		if a == nil || b == nil { return a != nil && b == nil }
		return a.Before( *b )
	} )

	rows := []fiber.Map{}
	for index , item := range hits {
		if index >= limit { break }
		rows = append( rows , item.row )
	}
	reply = &fiber.Map{ "cards": rows , "total": len( hits ) , "truncated": len( hits ) > limit }
	return
}

func containsID( ids []uint64 , id uint64 ) ( result bool ) {
	for _ , candidate := range ids {
		if candidate == id {
			result = true
			return
		}
	}
	return
}
