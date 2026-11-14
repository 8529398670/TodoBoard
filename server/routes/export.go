package routes

import (
	fmt "fmt"
	sort "sort"
	strconv "strconv"
	strings "strings"
	time "time"

	fiber "github.com/gofiber/fiber/v3"

	access "todoboard/server/access"
	models "todoboard/server/models"
	sheet "todoboard/server/sheet"
)

type exportBoardRequest struct {
	boardPath
	Format   string `json:"format" query:"format" jsonschema_description:"xlsx or csv."`
	Archived string `json:"archived,omitempty" query:"archived" jsonschema_description:"1 adds archived cards and columns, with an Archived column."`
	TZ       string `json:"tz,omitempty" query:"tz" jsonschema_description:"The timezone timed dates are written in (an IANA name); defaults to your own, else UTC."`
}

// exportBoard answers with the board as a spreadsheet: one row per card the
// caller can see, list by list, in board order. archived=1 adds archived
// cards (and archived lists) with an Archived column; tz names the timezone
// timed dates are written in, which defaults to the member's own and
// otherwise UTC. Column headings come from language.yaml (export.col_*),
// like everything else people read.
//
// @route("GET /api/boards/{board_id}/export", summary="Download a board as a spreadsheet",
//     tag="Boards", errors=(400, 403, 404), reply_type="application/octet-stream",
//     description="The board as an xlsx or csv spreadsheet (format): one row per card you can see, column by "
//         "column, in board order. archived=1 adds archived cards and columns; tz is the timezone timed dates "
//         "are written in.",
//     mcp_http="A file (a picture, a spreadsheet or a backup), which travels over HTTP rather than as a tool answer.",
// )
func ( handlers *Handlers ) exportBoard( c fiber.Ctx , in *exportBoardRequest ) ( err error ) {
	principal , board , err := handlers.boardAt( c , in.BoardID )
	if err != nil { return }
	// A spreadsheet names the board and the card's column.
	if access.QuickLink( principal ) {
		err = refuse( fiber.StatusNotFound , "board_not_found" )
		return
	}
	format := in.Format
	if format != "xlsx" && format != "csv" {
		err = refuse( fiber.StatusBadRequest , "invalid_input" )
		return
	}
	cards , list_err := models.ListCards( handlers.Store , board.ID )
	if list_err != nil {
		err = list_err
		return
	}
	table := handlers.exportTable( principal , board , cards , in.Archived == "1" , handlers.exportLocation( principal , in.TZ ) )

	var body []byte
	var build_err error
	content_type := "text/csv; charset=utf-8"
	if format == "xlsx" {
		body , build_err = table.XLSX()
		content_type = "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
	} else {
		body , build_err = table.CSV()
	}
	if build_err != nil {
		err = build_err
		return
	}
	name := board.Slug
	if name == "" { name = "board-" + strconv.FormatUint( board.ID , 10 ) }
	c.Set( fiber.HeaderContentType , content_type )
	c.Set( fiber.HeaderContentDisposition , fmt.Sprintf( `attachment; filename="%s.%s"` , name , format ) )
	c.Set( fiber.HeaderCacheControl , "no-store" )
	err = c.Send( body )
	return
}

// sharedExportBoard is exportBoard for a share link's guest.
//
// @route("GET /api/shared/{board_id}/export", summary="Download a board as a spreadsheet (share link)",
//     tag="Shared", access=PUBLIC, use=[handlers.SharePrincipal], errors=(400, 403, 404),
//     reply_type="application/octet-stream",
//     description="exportBoard for a guest holding a share link in X-Share-Token.",
// )
func ( handlers *Handlers ) sharedExportBoard( c fiber.Ctx , in *exportBoardRequest ) ( err error ) {
	err = handlers.exportBoard( c , in )
	return
}

// exportLocation is the zone timed dates are written in: the one asked for,
// else a member's own, else UTC.
func ( handlers *Handlers ) exportLocation( principal *access.Principal , zone string ) ( location *time.Location ) {
	if zone != "" {
		if loaded , load_err := time.LoadLocation( zone ); load_err == nil { return loaded }
	}
	location = time.UTC
	if principal.IsGuest() == false {
		if prefs , prefs_err := models.GetUserPrefs( handlers.Store , principal.UserID ); prefs_err == nil { location = prefs.Location() }
	}
	return
}

func ( handlers *Handlers ) exportTable( principal *access.Principal , board *models.Board , all_cards []*models.Card , with_archived bool , location *time.Location ) ( table *sheet.Sheet ) {
	text := func( key string ) string { return handlers.Language.Get( "export." + key ) }
	// A staged board's stage and rank; other boards have neither to show.
	staged := board.Layout == models.LayoutStaged
	// A stage's heading is the board's own name for it, if it has one.
	stage_heading := func( key string , own string ) string {
		if staged && own != "" { return own }
		return text( key )
	}
	table = &sheet.Sheet{
		Name: board.Title,
		Header: []string{
			text( "col_list" ) , text( "col_number" ) , text( "col_title" ) , stage_heading( "col_done" , board.StageDone ) , text( "col_labels" ) ,
			text( "col_start" ) , text( "col_due" ) , text( "col_assignees" ) , text( "col_checklist" ) , text( "col_body" ) ,
		},
		Numeric: map[int]bool{ 1: true },
	}
	// The card's lasting ID (its ULID, as in its link), unlike the #
	// column, which is its place.
	table.Header = append( table.Header , text( "col_serial" ) )

	// What the caller is shown on the board is what they can export: a column
	// link its column, a card-only caller their cards.
	list_only := access.ListOnly( principal , board )
	by_list := map[uint64][]*models.Card{}
	priced := board.Shopping
	for _ , list := range board.Lists { priced = priced || list.Shopping }
	for _ , card := range all_cards {
		if access.Card( principal , board , card ) == access.None { continue }
		if card.Archived && with_archived == false { continue }
		by_list[ card.ListID ] = append( by_list[ card.ListID ] , card )
		priced = priced || card.PriceText() != ""
	}

	// Price is free text with its currency in it, so it is not Numeric. Any
	// board can hold a priced card; a shopping board has the column regardless.
	if priced { table.Header = append( table.Header , text( "col_price" ) ) }
	places := models.RankPlaces( board , all_cards )
	if staged {
		table.Numeric[ len( table.Header ) + 1 ] = true
		table.Header = append( table.Header , stage_heading( "col_doing" , board.StageDoing ) , text( "col_rank" ) )
	}
	if with_archived { table.Header = append( table.Header , text( "col_archived" ) ) }

	labels := map[uint64]string{}
	for _ , label := range board.Labels {
		name := label.Name
		if name == "" { name = label.Color }
		labels[ label.ID ] = name
	}
	person := handlers.personNamer()

	for _ , list := range board.SortedLists() {
		if list_only != 0 && list.ID != list_only { continue }
		if list.Archived && with_archived == false { continue }
		cards := by_list[ list.ID ]
		if len( cards ) == 0 { continue }
		// Open cards first, numbered as on the board; archived ones after.
		sort.SliceStable( cards , func( i int , j int ) bool {
			if cards[ i ].Archived != cards[ j ].Archived { return cards[ j ].Archived }
			return cards[ i ].Position < cards[ j ].Position
		} )
		number := 0
		for _ , card := range cards {
			position := ""
			if card.Archived == false {
				number += 1
				position = strconv.Itoa( number )
			}
			names := []string{}
			for _ , id := range card.LabelIDs {
				if name , ok := labels[ id ]; ok { names = append( names , name ) }
			}
			assignees := []string{}
			for _ , id := range card.Assignees { assignees = append( assignees , person( id ) ) }
			checklist := ""
			if len( card.Checklist ) > 0 {
				done := 0
				for _ , item := range card.Checklist {
					if item.Done { done += 1 }
				}
				checklist = fmt.Sprintf( "%d/%d" , done , len( card.Checklist ) )
			}
			row := []string{
				list.Title , position , card.Title , yes( card.Done , text( "yes" ) ) , strings.Join( names , ", " ) ,
				exportDate( card.StartAt , card.StartAt != nil && card.StartTimed == false , location ) ,
				exportDate( card.DueAt , card.DueAllDay , location ) ,
				strings.Join( assignees , ", " ) , checklist , strings.TrimSpace( card.Description ) ,
				card.UID ,
			}
			if priced { row = append( row , card.PriceText() ) }
			if staged {
				rank := ""
				if place := places[ card.ID ]; place > 0 { rank = strconv.Itoa( place ) }
				row = append( row , yes( card.Doing , text( "yes" ) ) , rank )
			}
			if with_archived { row = append( row , yes( card.Archived , text( "yes" ) ) ) }
			table.Rows = append( table.Rows , row )
		}
	}
	return
}

func yes( value bool , word string ) ( result string ) {
	if value { result = word }
	return
}

// exportDate writes an all-day date as 2026-10-30 and a timed one as
// 2026-10-30 14:05 in location -- forms every spreadsheet reads as dates.
func exportDate( value *time.Time , all_day bool , location *time.Location ) ( result string ) {
	if value == nil { return }
	if all_day {
		result = value.UTC().Format( "2006-01-02" )
		return
	}
	result = value.In( location ).Format( "2006-01-02 15:04" )
	return
}
