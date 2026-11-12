package routes

import (
	testing "testing"

	fiber "github.com/gofiber/fiber/v3"

	models "todoboard/server/models"
)

// Duplicating from the home page: a new board the caller owns with the same
// lists and cards, the original left alone, and nothing for someone who is
// not on the board.
func TestDuplicateBoard( t *testing.T ) {
	fixture := newAuthFixture( t )
	store := fixture.store
	board , _ := models.CreateBoard( store , fixture.owner.ID , "Groceries" , []string{ "To do" , "Done" } )
	models.CreateCard( store , board.ID , board.Lists[ 0 ].ID , "Milk" , nil , fixture.owner.ID , "Owner" )
	models.CreateCard( store , board.ID , board.Lists[ 0 ].ID , "Bread" , nil , fixture.owner.ID , "Owner" )
	models.CreateCard( store , board.ID , board.Lists[ 1 ].ID , "Eggs" , nil , fixture.owner.ID , "Owner" )
	path := "/api/boards/" + itoa( board.ID ) + "/duplicate"

	stranger := fixture.signIn( t , fixture.alice.ID )
	if status , _ := send( t , stranger , path , fiber.Map{ "title": "Mine" } ); status != 404 && status != 403 {
		t.Errorf( "duplicate a board you are not on: %d" , status )
	}

	owner := fixture.signIn( t , fixture.owner.ID )
	if status , _ := send( t , owner , path , fiber.Map{ "title": "" } ); status != 400 {
		t.Errorf( "duplicate with no title: %d" , status )
	}
	status , reply := send( t , owner , path , fiber.Map{ "title": "Groceries (copy)" } )
	if status != 200 { t.Fatalf( "duplicate: %d %v" , status , reply ) }
	copy_id := uint64( reply[ "id" ].( float64 ) )
	if copy_id == board.ID || reply[ "slug" ] == board.Slug { t.Fatalf( "copy reuses the original: %v" , reply ) }

	copied , err := models.GetBoard( store , copy_id )
	if err != nil { t.Fatal( err ) }
	if copied.Title != "Groceries (copy)" || copied.OwnerID != fixture.owner.ID { t.Errorf( "copy: %q owned by %d" , copied.Title , copied.OwnerID ) }
	board , _ = models.GetBoard( store , board.ID )
	lists := copied.SortedLists()
	want := board.SortedLists()
	if len( lists ) != len( want ) { t.Fatalf( "copy has %d lists, original %d" , len( lists ) , len( want ) ) }
	for index := range want {
		if lists[ index ].Title != want[ index ].Title { t.Errorf( "list %d: %q, want %q" , index , lists[ index ].Title , want[ index ].Title ) }
	}
	cards , _ := models.ListCards( store , copy_id )
	per_list := map[ uint64 ]int{}
	for _ , card := range cards { per_list[ card.ListID ]++ }
	if len( cards ) != 3 || per_list[ lists[ 0 ].ID ] != 2 || per_list[ lists[ 1 ].ID ] != 1 { t.Errorf( "copy cards: %d %v" , len( cards ) , per_list ) }

	original , _ := models.ListCards( store , board.ID )
	if len( original ) != 3 { t.Errorf( "original cards: %d" , len( original ) ) }
}
