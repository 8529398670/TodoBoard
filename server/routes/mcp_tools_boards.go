package routes

import (
	bytes "bytes"
	json "encoding/json"
	errors "errors"
	sort "sort"
	strconv "strconv"
	strings "strings"
	time "time"

	fiber "github.com/gofiber/fiber/v3"

	access "todoboard/server/access"
	mcp "todoboard/server/mcp"
	models "todoboard/server/models"
)

// The board and card tools. Each one runs the same checks as its HTTP route
// -- loadBoard / loadBoardAt / loadCard, then the shared helper in boards.go
// or cards.go -- so what a key may do on a board is still server/access's
// decision alone. App admin rights grant nothing here, exactly as on the web.
//
// What is deliberately missing, beyond the template's list: deleting a
// board, column, label, card or checklist item (archiving a card through
// update_card is the reversible way to clear it), share links, board
// membership, and backup import. Deleting a board already needs a browser
// session on the web; the rest either cannot be undone or hands out access.

// boardRef is a board's numeric id or its URL slug. A model is as likely to
// send 12 as "12", so both are accepted.
type boardRef string

func ( ref *boardRef ) UnmarshalJSON( data []byte ) ( err error ) {
	data = bytes.TrimSpace( data )
	if len( data ) > 0 && data[ 0 ] == '"' {
		var text string
		err = json.Unmarshal( data , &text )
		*ref = boardRef( text )
		return
	}
	var number uint64
	if json.Unmarshal( data , &number ) != nil {
		err = errors.New( "board must be a board id or slug" )
		return
	}
	*ref = boardRef( strconv.FormatUint( number , 10 ) )
	return
}

var boardRefSchema = fiber.Map{ "type": []string{ "integer" , "string" } , "description": "The board's id or URL slug, from list_boards." }
var cardIDSchema = fiber.Map{ "type": "integer" , "minimum": 1 , "description": "The card's id, from get_board or find_cards." }
var listIDSchema = fiber.Map{ "type": "integer" , "minimum": 1 , "description": "The column's id, from get_board." }
var positionSchema = fiber.Map{ "type": "number" , "description": "Sort position within the column; lower comes first. Leave out to put it at the end." }
var dateSchema = fiber.Map{ "type": "string" , "description": `"2026-10-30" for an all-day date, an RFC 3339 timestamp for a time, "" to clear.` }
var labelColors = []string{ "blue" , "green" , "orange" , "red" , "purple" , "pink" , "teal" , "gray" , "yellow" }

// mcpPrincipal is MemberPrincipal for a key: the same account, the same name
// on its edits. No CSRF check -- a bearer key is not an ambient credential.
func mcpPrincipal( call *mcp.Call ) ( principal *access.Principal ) {
	principal = &access.Principal{ UserID: call.Actor.User.ID , Name: call.Actor.User.DisplayName }
	return
}

func objectSchema( properties fiber.Map , required ...string ) ( schema fiber.Map ) {
	schema = fiber.Map{ "type": "object" , "properties": properties , "additionalProperties": false }
	if len( required ) > 0 { schema[ "required" ] = required }
	return
}

func ( handlers *Handlers ) registerBoardTools( server *mcp.Server ) {
	server.Register( &mcp.Tool{
		Name:        "list_boards",
		Title:       "List my boards",
		Description: "Every board you own or are a member of (id, slug, title, your role, folder), your home-page folders, and single cards shared with you from boards you are not on.",
		ReadOnly:    true,
		Call: func( call *mcp.Call ) ( result any , err error ) {
			if err = call.Bind( &struct{}{} ); err != nil { return }
			result , err = handlers.myBoards( call.Actor.User.ID )
			return
		},
	} )

	server.Register( &mcp.Tool{
		Name:        "get_board",
		Title:       "Read a board",
		Description: "A board as you can see it: its columns (lists), labels, members, and the cards you may see, with each card's column, labels, assignees, dates, checklist and rev. people maps user ids to names. Archived cards are left out unless include_archived is true.",
		ReadOnly:    true,
		InputSchema: objectSchema( fiber.Map{
			"board":            boardRefSchema,
			"include_archived": fiber.Map{ "type": "boolean" , "default": false },
		} , "board" ),
		Call: func( call *mcp.Call ) ( result any , err error ) {
			var args struct {
				Board           boardRef `json:"board"`
				IncludeArchived bool     `json:"include_archived"`
			}
			if err = call.Bind( &args ); err != nil { return }
			principal := mcpPrincipal( call )
			board , refused := handlers.loadBoard( principal , string( args.Board ) )
			if refused != nil {
				err = refusalError( refused )
				return
			}
			cards , err := models.ListCards( handlers.Store , board.ID )
			if err != nil { return }
			if args.IncludeArchived == false {
				open := cards[ :0 ]
				for _ , card := range cards {
					if card.Archived == false { open = append( open , card ) }
				}
				cards = open
			}
			result = handlers.boardPayload( principal , board , cards )
			return
		},
	} )

	server.Register( &mcp.Tool{
		Name:        "get_card",
		Title:       "Read a card",
		Description: "One card in full: title, description, column, labels, assignees, dates, checklist, and rev (pass rev back to update_card when changing the title or description).",
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
			result = cardPayload( card , access.Card( principal , board , card ) , access.Board( principal , board ) )
			return
		},
	} )

	server.Register( &mcp.Tool{
		Name:        "find_cards",
		Title:       "Find cards",
		Description: "Search the cards you can see, across all your boards or one. query matches title and description (case-insensitive). Done and archived cards are left out unless asked for. Returns compact rows, soonest due first; use get_card for the full card.",
		ReadOnly:    true,
		InputSchema: objectSchema( fiber.Map{
			"query":            fiber.Map{ "type": "string" },
			"board":            boardRefSchema,
			"assigned_to_me":   fiber.Map{ "type": "boolean" , "default": false },
			"due_before":       fiber.Map{ "type": "string" , "description": `Only cards due before this: "2026-10-30" or an RFC 3339 timestamp.` },
			"include_done":     fiber.Map{ "type": "boolean" , "default": false },
			"include_archived": fiber.Map{ "type": "boolean" , "default": false },
			"limit":            fiber.Map{ "type": "integer" , "minimum": 1 , "maximum": 200 , "default": 50 },
		} ),
		Call: func( call *mcp.Call ) ( result any , err error ) {
			var args findCardsArgs
			if err = call.Bind( &args ); err != nil { return }
			result , err = handlers.findCards( mcpPrincipal( call ) , &args )
			return
		},
	} )

	server.Register( &mcp.Tool{
		Name:        "create_board",
		Title:       "Create a board",
		Description: "A new board owned by you, with the given columns in order (for example [\"To do\", \"Doing\", \"Done\"]); a board always gets at least three, so fewer are padded out. Returns its id and slug.",
		InputSchema: objectSchema( fiber.Map{
			"title": fiber.Map{ "type": "string" , "maxLength": 120 },
			"lists": fiber.Map{ "type": "array" , "items": fiber.Map{ "type": "string" } },
		} , "title" ),
		Call: func( call *mcp.Call ) ( result any , err error ) {
			var args struct {
				Title string   `json:"title"`
				Lists []string `json:"lists"`
			}
			if err = call.Bind( &args ); err != nil { return }
			board , create_err := models.CreateBoard( handlers.Store , call.Actor.User.ID , args.Title , args.Lists )
			if create_err != nil {
				err = refusalError( modelRefusal( create_err ) )
				return
			}
			result = fiber.Map{ "id": board.ID , "slug": board.Slug }
			return
		},
	} )

	server.Register( &mcp.Tool{
		Name:        "create_list",
		Title:       "Add a column",
		Description: "Add a column (list) to a board. Needs edit rights on the whole board.",
		InputSchema: objectSchema( fiber.Map{
			"board":    boardRefSchema,
			"title":    fiber.Map{ "type": "string" , "maxLength": 120 },
			"position": positionSchema,
		} , "board" , "title" ),
		Call: func( call *mcp.Call ) ( result any , err error ) {
			var args struct {
				Board    boardRef `json:"board"`
				Title    string   `json:"title"`
				Position *float64 `json:"position"`
			}
			if err = call.Bind( &args ); err != nil { return }
			board , refused := handlers.loadBoardAt( mcpPrincipal( call ) , string( args.Board ) , access.Edit )
			if refused == nil {
				result , refused = handlers.createList( board , args.Title , args.Position )
			}
			if refused != nil { err = refusalError( refused ) }
			return
		},
	} )

	server.Register( &mcp.Tool{
		Name:        "update_list",
		Title:       "Rename, move or archive a column",
		Description: "Change a column's title or position, or archive it (archived: true hides it; false brings it back). Leave out what should not change.",
		InputSchema: objectSchema( fiber.Map{
			"board":    boardRefSchema,
			"list_id":  listIDSchema,
			"title":    fiber.Map{ "type": "string" , "maxLength": 120 },
			"position": positionSchema,
			"archived": fiber.Map{ "type": "boolean" },
		} , "board" , "list_id" ),
		Call: func( call *mcp.Call ) ( result any , err error ) {
			var args struct {
				Board  boardRef `json:"board"`
				ListID uint64   `json:"list_id"`
				models.ListUpdate
			}
			if err = call.Bind( &args ); err != nil { return }
			board , refused := handlers.loadBoardAt( mcpPrincipal( call ) , string( args.Board ) , access.Edit )
			if refused == nil {
				refused = handlers.updateList( board , args.ListID , &args.ListUpdate )
			}
			if refused != nil {
				err = refusalError( refused )
				return
			}
			result = fiber.Map{ "ok": true }
			return
		},
	} )

	server.Register( &mcp.Tool{
		Name:        "create_label",
		Title:       "Add a label",
		Description: "Add a label to a board, to put on cards with update_card's label_ids. Needs edit rights on the whole board.",
		InputSchema: objectSchema( fiber.Map{
			"board": boardRefSchema,
			"name":  fiber.Map{ "type": "string" , "maxLength": 40 },
			"color": fiber.Map{ "type": "string" , "enum": labelColors },
		} , "board" , "color" ),
		Call: func( call *mcp.Call ) ( result any , err error ) {
			var args struct {
				Board boardRef `json:"board"`
				Name  string   `json:"name"`
				Color string   `json:"color"`
			}
			if err = call.Bind( &args ); err != nil { return }
			board , refused := handlers.loadBoardAt( mcpPrincipal( call ) , string( args.Board ) , access.Edit )
			if refused == nil {
				result , refused = handlers.createLabel( board , args.Name , args.Color )
			}
			if refused != nil { err = refusalError( refused ) }
			return
		},
	} )

	server.Register( &mcp.Tool{
		Name:        "create_card",
		Title:       "Add a card",
		Description: "Add a card to a column. The other fields are optional and set in the same call. Needs edit rights on that column.",
		InputSchema: objectSchema( fiber.Map{
			"board":       boardRefSchema,
			"list_id":     listIDSchema,
			"title":       fiber.Map{ "type": "string" },
			"position":    positionSchema,
			"description": fiber.Map{ "type": "string" , "description": "Markdown." },
			"label_ids":   fiber.Map{ "type": "array" , "items": fiber.Map{ "type": "integer" } },
			"assignees":   fiber.Map{ "type": "array" , "items": fiber.Map{ "type": "integer" } , "description": "User ids, from get_board's members or list_team." },
			"start_at":    dateSchema,
			"due_at":      dateSchema,
		} , "board" , "list_id" , "title" ),
		Call: func( call *mcp.Call ) ( result any , err error ) {
			var args struct {
				Board       boardRef  `json:"board"`
				ListID      uint64    `json:"list_id"`
				Title       string    `json:"title"`
				Position    *float64  `json:"position"`
				Description *string   `json:"description"`
				LabelIDs    *[]uint64 `json:"label_ids"`
				Assignees   *[]uint64 `json:"assignees"`
				StartAt     *string   `json:"start_at"`
				DueAt       *string   `json:"due_at"`
			}
			if err = call.Bind( &args ); err != nil { return }
			principal := mcpPrincipal( call )
			board , refused := handlers.loadBoard( principal , string( args.Board ) )
			if refused != nil {
				err = refusalError( refused )
				return
			}
			card , refused := handlers.createCard( principal , board , args.ListID , args.Title , args.Position )
			if refused != nil {
				err = refusalError( refused )
				return
			}
			patch := models.CardPatch{ Description: args.Description , LabelIDs: args.LabelIDs , Assignees: args.Assignees , StartAt: args.StartAt , DueAt: args.DueAt }
			if patch != ( models.CardPatch{} ) {
				updated , _ , patch_refused := handlers.patchCard( principal , board , card , &patch )
				if patch_refused != nil {
					err = mcp.Refuse( "card " + strconv.FormatUint( card.ID , 10 ) + " was created, but its other fields were not set: " + patch_refused.message )
					return
				}
				card = updated
			}
			result = cardPayload( card , access.Card( principal , board , card ) , access.Board( principal , board ) )
			return
		},
	} )

	server.Register( &mcp.Tool{
		Name:        "update_card",
		Title:       "Change a card",
		Description: "Change any of a card's fields; leave out what should not change. Moving it (list_id, position) or archiving it needs edit rights on its column, and on the whole board to move it to another column. done marks it complete. archived: true hides it from the board (reversible). Pass the card's rev when changing title or description: if someone edited it since, the change is refused rather than written over theirs.",
		InputSchema: objectSchema( fiber.Map{
			"board":       boardRefSchema,
			"card_id":     cardIDSchema,
			"rev":         fiber.Map{ "type": "integer" , "description": "The rev from get_card / get_board." },
			"title":       fiber.Map{ "type": "string" },
			"description": fiber.Map{ "type": "string" , "description": "Markdown." },
			"list_id":     listIDSchema,
			"position":    positionSchema,
			"label_ids":   fiber.Map{ "type": "array" , "items": fiber.Map{ "type": "integer" } , "description": "The full set; replaces the current labels." },
			"assignees":   fiber.Map{ "type": "array" , "items": fiber.Map{ "type": "integer" } , "description": "The full set of user ids; replaces the current assignees." },
			"start_at":    dateSchema,
			"due_at":      dateSchema,
			"done":        fiber.Map{ "type": "boolean" },
			"archived":    fiber.Map{ "type": "boolean" },
		} , "board" , "card_id" ),
		Call: func( call *mcp.Call ) ( result any , err error ) {
			var args struct {
				Board  boardRef `json:"board"`
				CardID uint64   `json:"card_id"`
				models.CardPatch
			}
			if err = call.Bind( &args ); err != nil { return }
			principal , board , card , err := handlers.mcpCard( call , args.Board , args.CardID , access.Edit )
			if err != nil { return }
			updated , current , refused := handlers.patchCard( principal , board , card , &args.CardPatch )
			if current != nil {
				err = mcp.Refuse( "card_stale: someone changed this card since that rev (it is now rev " + strconv.FormatUint( current.Rev , 10 ) + "). Read it again with get_card before changing the title or description." )
				return
			}
			if refused != nil {
				err = refusalError( refused )
				return
			}
			result = cardPayload( updated , access.Card( principal , board , updated ) , access.Board( principal , board ) )
			return
		},
	} )

	server.Register( &mcp.Tool{
		Name:        "add_checklist_item",
		Title:       "Add a checklist item",
		Description: "Add an item to the end of a card's checklist. Returns the card.",
		InputSchema: objectSchema( fiber.Map{
			"board":   boardRefSchema,
			"card_id": cardIDSchema,
			"text":    fiber.Map{ "type": "string" },
		} , "board" , "card_id" , "text" ),
		Call: func( call *mcp.Call ) ( result any , err error ) {
			var args struct {
				Board  boardRef `json:"board"`
				CardID uint64   `json:"card_id"`
				Text   string   `json:"text"`
			}
			if err = call.Bind( &args ); err != nil { return }
			principal , board , card , err := handlers.mcpCard( call , args.Board , args.CardID , access.Edit )
			if err != nil { return }
			updated , refused := handlers.addChecklistItem( principal , card , args.Text )
			if refused != nil {
				err = refusalError( refused )
				return
			}
			result = cardPayload( updated , access.Card( principal , board , updated ) , access.Board( principal , board ) )
			return
		},
	} )

	server.Register( &mcp.Tool{
		Name:        "update_checklist_item",
		Title:       "Tick or edit a checklist item",
		Description: "Tick an item off (done: true) or back on, change its text, or move it. Item ids are in the card's checklist. Returns the card.",
		InputSchema: objectSchema( fiber.Map{
			"board":    boardRefSchema,
			"card_id":  cardIDSchema,
			"item_id":  fiber.Map{ "type": "integer" , "minimum": 1 },
			"text":     fiber.Map{ "type": "string" },
			"done":     fiber.Map{ "type": "boolean" },
			"position": fiber.Map{ "type": "number" },
		} , "board" , "card_id" , "item_id" ),
		Call: func( call *mcp.Call ) ( result any , err error ) {
			var args struct {
				Board  boardRef `json:"board"`
				CardID uint64   `json:"card_id"`
				ItemID uint64   `json:"item_id"`
				models.ChecklistUpdate
			}
			if err = call.Bind( &args ); err != nil { return }
			principal , board , card , err := handlers.mcpCard( call , args.Board , args.CardID , access.Edit )
			if err != nil { return }
			updated , refused := handlers.updateChecklistItem( principal , card , args.ItemID , &args.ChecklistUpdate )
			if refused != nil {
				err = refusalError( refused )
				return
			}
			result = cardPayload( updated , access.Card( principal , board , updated ) , access.Board( principal , board ) )
			return
		},
	} )
}

// mcpCard is requireCard for a tool: the board, then the card on it at the
// given level, with either refusal handed back as the tool's error.
func ( handlers *Handlers ) mcpCard( call *mcp.Call , ref boardRef , card_id uint64 , minimum access.Level ) ( principal *access.Principal , board *models.Board , card *models.Card , err error ) {
	principal = mcpPrincipal( call )
	board , refused := handlers.loadBoard( principal , string( ref ) )
	if refused == nil {
		card , refused = handlers.loadCard( principal , board , card_id , minimum )
	}
	if refused != nil {
		err = refusalError( refused )
	}
	return
}

type findCardsArgs struct {
	Query           string    `json:"query"`
	Board           *boardRef `json:"board"`
	AssignedToMe    bool      `json:"assigned_to_me"`
	DueBefore       string    `json:"due_before"`
	IncludeDone     bool      `json:"include_done"`
	IncludeArchived bool      `json:"include_archived"`
	Limit           int       `json:"limit"`
}

// findCards walks the boards the caller is on, plus the boards of cards
// shared with them, and keeps the cards access.Card lets them see. It is
// a scan, which is fine at the size a team's boards reach.
func ( handlers *Handlers ) findCards( principal *access.Principal , args *findCardsArgs ) ( result fiber.Map , err error ) {
	limit := args.Limit
	if limit <= 0 { limit = 50 }
	if limit > 200 { limit = 200 }
	var due_before *time.Time
	if strings.TrimSpace( args.DueBefore ) != "" {
		due_before , _ , err = models.ParseCardDate( args.DueBefore )
		if err != nil {
			err = mcp.Refuse( `due_before must be "2026-10-30" or an RFC 3339 timestamp` )
			return
		}
	}
	query := strings.ToLower( strings.TrimSpace( args.Query ) )

	var boards []*models.Board
	if args.Board != nil {
		board , refused := handlers.loadBoard( principal , string( *args.Board ) )
		if refused != nil {
			err = refusalError( refused )
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
		for _ , card := range cards {
			if access.Card( principal , board , card ) == access.None { continue }
			if card.Archived && args.IncludeArchived == false { continue }
			if card.Done && args.IncludeDone == false { continue }
			if args.AssignedToMe && containsID( card.Assignees , principal.UserID ) == false { continue }
			if due_before != nil && ( card.DueAt == nil || card.DueAt.Before( *due_before ) == false ) { continue }
			if query != "" && strings.Contains( strings.ToLower( card.Title ) , query ) == false && strings.Contains( strings.ToLower( card.Description ) , query ) == false { continue }
			list_title := ""
			if list := board.List( card.ListID ); list != nil { list_title = list.Title }
			hits = append( hits , hit{ due: card.DueAt , row: fiber.Map{
				"board_id":    board.ID,
				"board_title": board.Title,
				"card_id":     card.ID,
				"list_id":     card.ListID,
				"list_title":  list_title,
				"title":       card.Title,
				"due_at":      card.DueAt,
				"done":        card.Done,
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
	result = fiber.Map{ "cards": rows , "total": len( hits ) , "truncated": len( hits ) > limit }
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
