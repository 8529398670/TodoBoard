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

// ExportBoard answers GET .../export?format=xlsx|csv with the board as a
// spreadsheet: one row per card the caller can see, list by list, in board
// order. archived=1 adds archived cards (and archived lists) with an
// Archived column; tz names the timezone timed dates are written in, which
// defaults to the member's own and otherwise UTC. Column headings come from
// language.yaml (export.col_*), like everything else people read.
func ( handlers *Handlers ) ExportBoard( c fiber.Ctx ) ( err error ) {
	principal , board , err := handlers.boardFor( c )
	if board == nil { return }
	format := c.Query( "format" )
	if format != "xlsx" && format != "csv" {
		err = badRequest( c , "invalid_input" )
		return
	}
	cards , list_err := models.ListCards( handlers.Store , board.ID )
	if list_err != nil {
		err = serverError( c )
		return
	}
	table := handlers.exportTable( principal , board , cards , c.Query( "archived" ) == "1" , handlers.exportLocation( principal , c.Query( "tz" ) ) )

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
		err = serverError( c )
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
	table = &sheet.Sheet{
		Name: board.Title,
		Header: []string{
			text( "col_list" ) , text( "col_number" ) , text( "col_title" ) , text( "col_done" ) , text( "col_labels" ) ,
			text( "col_start" ) , text( "col_due" ) , text( "col_assignees" ) , text( "col_checklist" ) , text( "col_body" ) ,
		},
		Numeric: map[int]bool{ 1: true },
	}
	if with_archived { table.Header = append( table.Header , text( "col_archived" ) ) }

	// What the caller is shown on the board is what they can export: a column
	// link its column, a card-only caller their cards.
	list_only := access.ListOnly( principal , board )
	by_list := map[uint64][]*models.Card{}
	for _ , card := range all_cards {
		if access.Card( principal , board , card ) == access.None { continue }
		if card.Archived && with_archived == false { continue }
		by_list[ card.ListID ] = append( by_list[ card.ListID ] , card )
	}
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
