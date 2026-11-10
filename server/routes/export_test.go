package routes

import (
	os "os"
	testing "testing"
	time "time"

	access "todoboard/server/access"
	config "todoboard/server/config"
	language "todoboard/server/language"
	models "todoboard/server/models"
	sheet "todoboard/server/sheet"
)

type exportFixture struct {
	*mcpFixture
	handlers  *Handlers
	principal *access.Principal
}

// The real language file, so a column whose heading was never added shows up
// as an empty one here.
func newExportFixture( t *testing.T ) ( fixture *exportFixture ) {
	t.Helper()
	raw , err := os.ReadFile( "../../config/language.yaml" )
	if err != nil { t.Fatal( err ) }
	lang , err := language.Parse( raw )
	if err != nil { t.Fatal( err ) }
	base := newMCPFixture( t )
	fixture = &exportFixture{
		mcpFixture: base ,
		handlers:   New( base.store , config.NewStaticHolder( base.cfg ) , nil , lang , nil , "test" ) ,
		principal:  &access.Principal{ UserID: base.member.ID , Name: "Member" } ,
	}
	return
}

func ( fixture *exportFixture ) board( t *testing.T , mutate func( board *models.Board ) error ) ( board *models.Board ) {
	t.Helper()
	board , err := models.CreateBoard( fixture.store , fixture.member.ID , "Export" , []string{ "To do" , "Later" } )
	if err != nil { t.Fatal( err ) }
	if mutate != nil {
		if board , err = models.MutateBoard( fixture.store , board.ID , mutate ); err != nil { t.Fatal( err ) }
	}
	return
}

func ( fixture *exportFixture ) card( t *testing.T , board *models.Board , title string , price string ) ( card *models.Card ) {
	t.Helper()
	card , err := models.CreateCard( fixture.store , board.ID , board.SortedLists()[ 0 ].ID , title , nil , fixture.member.ID , "Member" )
	if err != nil { t.Fatal( err ) }
	if price != "" {
		if card , _ , err = models.PatchCard( fixture.store , board.ID , card.ID , &models.CardPatch{ Price: &price } , "Member" ); err != nil { t.Fatal( err ) }
	}
	return
}

func ( fixture *exportFixture ) table( t *testing.T , board *models.Board ) ( table *sheet.Sheet ) {
	t.Helper()
	board , err := models.GetBoard( fixture.store , board.ID )
	if err != nil { t.Fatal( err ) }
	cards , err := models.ListCards( fixture.store , board.ID )
	if err != nil { t.Fatal( err ) }
	table = fixture.handlers.exportTable( fixture.principal , board , cards , false , time.UTC )
	for index , heading := range table.Header {
		if heading == "" { t.Errorf( "column %d has no heading: %q" , index , table.Header ) }
	}
	for _ , row := range table.Rows {
		if len( row ) != len( table.Header ) { t.Errorf( "row has %d cells for %d headings: %q" , len( row ) , len( table.Header ) , row ) }
	}
	return
}

func column( table *sheet.Sheet , heading string ) ( index int ) {
	for index , name := range table.Header {
		if name == heading { return index }
	}
	return -1
}

func TestExportNoPriceColumnWithoutPrices( t *testing.T ) {
	fixture := newExportFixture( t )
	board := fixture.board( t , nil )
	fixture.card( t , board , "Plain" , "" )
	table := fixture.table( t , board )
	if column( table , "Price" ) != -1 { t.Errorf( "a board with no prices needs no column: %q" , table.Header ) }
	id := column( table , "Card ID" )
	if id == -1 || len( table.Rows[ 0 ][ id ] ) != 26 || table.Numeric[ id ] {
		t.Errorf( "card ID column should hold the ULID as text: %q %q %v" , table.Header , table.Rows , table.Numeric )
	}
}

func TestExportPriceColumn( t *testing.T ) {
	fixture := newExportFixture( t )

	// A priced card on a board that is not a shopping one still exports it.
	normal := fixture.board( t , nil )
	fixture.card( t , normal , "Priced" , "$24.99" )
	fixture.card( t , normal , "Unpriced" , "" )
	table := fixture.table( t , normal )
	price := column( table , "Price" )
	if price == -1 || table.Rows[ 0 ][ price ] != "$24.99" || table.Rows[ 1 ][ price ] != "" {
		t.Fatalf( "price column: %q %q" , table.Header , table.Rows )
	}
	if table.Numeric[ price ] { t.Error( "a price is free text, not a number" ) }

	// Shopping boards and columns have the column even before any price.
	whole := fixture.board( t , func( board *models.Board ) error { board.Shopping = true ; return nil } )
	fixture.card( t , whole , "Unpriced" , "" )
	if table = fixture.table( t , whole ); column( table , "Price" ) == -1 { t.Errorf( "shopping board: %q" , table.Header ) }
	one := fixture.board( t , func( board *models.Board ) error { board.SortedLists()[ 1 ].Shopping = true ; return nil } )
	fixture.card( t , one , "Unpriced" , "" )
	if table = fixture.table( t , one ); column( table , "Price" ) == -1 { t.Errorf( "shopping column: %q" , table.Header ) }
}

func TestExportStagedRankStaysNumeric( t *testing.T ) {
	fixture := newExportFixture( t )
	board := fixture.board( t , func( board *models.Board ) error { board.Layout = models.LayoutStaged ; return nil } )
	card := fixture.card( t , board , "Ranked" , "$5" )
	_ , err := models.MutateCard( fixture.store , board.ID , card.ID , "Member" , func( board *models.Board , card *models.Card ) error {
		card.Rank = 1024
		return nil
	} )
	if err != nil { t.Fatal( err ) }
	table := fixture.table( t , board )
	rank , doing , price := column( table , "Rank" ) , column( table , "Doing" ) , column( table , "Price" )
	if rank == -1 || table.Numeric[ rank ] == false || table.Numeric[ doing ] || table.Numeric[ price ] {
		t.Errorf( "numeric columns moved: %q %v" , table.Header , table.Numeric )
	}
	if table.Rows[ 0 ][ rank ] != "1" || table.Rows[ 0 ][ price ] != "$5" { t.Errorf( "row: %q %q" , table.Header , table.Rows[ 0 ] ) }
}

// Headings follow a staged board's own names for its stages, and nothing else
// has them: a free board's Done column stays the app's word.
func TestExportUsesStageNames( t *testing.T ) {
	fixture := newExportFixture( t )
	staged := fixture.board( t , func( board *models.Board ) error {
		board.Layout = models.LayoutStaged
		board.StageDoing = "Review"
		board.StageDone = "Shipped"
		return nil
	} )
	table := fixture.table( t , staged )
	if column( table , "Review" ) < 0 || column( table , "Shipped" ) < 0 { t.Errorf( "stage names not used: %v" , table.Header ) }
	if column( table , "Doing" ) >= 0 || column( table , "Done" ) >= 0 { t.Errorf( "app's words left in: %v" , table.Header ) }

	free := fixture.board( t , func( board *models.Board ) error {
		board.Layout = models.LayoutFree
		board.StageDone = "Shipped"
		return nil
	} )
	table = fixture.table( t , free )
	if column( table , "Done" ) < 0 || column( table , "Shipped" ) >= 0 { t.Errorf( "a free board has no stage names: %v" , table.Header ) }
}
