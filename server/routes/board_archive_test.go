package routes

import (
	http "net/http"
	testing "testing"

	fiber "github.com/gofiber/fiber/v3"

	models "todoboard/server/models"
)

// Archived for everyone: read-only for the owner and members alike, with
// the few things that still work on it -- unarchiving, the trash, leaving,
// duplicating -- still working. Archived for me: only my home page.
func TestArchiveBoard( t *testing.T ) {
	fixture := newAuthFixture( t )
	store := fixture.store
	board , _ := models.CreateBoard( store , fixture.owner.ID , "Lab" , []string{ "To do" , "Done" } )
	models.SetMember( store , board.ID , fixture.alice.ID , models.BoardRoleEditor )
	card , _ := models.CreateCard( store , board.ID , board.Lists[ 0 ].ID , "Report" , nil , fixture.owner.ID , "Owner" )
	base := "/api/boards/" + itoa( board.ID )
	owner := fixture.signIn( t , fixture.owner.ID )
	alice := fixture.signIn( t , fixture.alice.ID )

	if status , _ := send( t , alice , base + "/archive" , fiber.Map{ "archived": true , "scope": "everyone" } ); status != 403 {
		t.Errorf( "an editor cannot archive for everyone: %d" , status )
	}
	if status , _ := send( t , owner , base + "/archive" , fiber.Map{ "archived": true , "scope": "nobody" } ); status != 400 {
		t.Errorf( "unknown scope: %d" , status )
	}
	if status , reply := send( t , owner , base + "/archive" , fiber.Map{ "archived": true , "scope": "everyone" } ); status != 200 {
		t.Fatalf( "archive for everyone: %d %v" , status , reply )
	}

	// The page is told, and offered nothing to change.
	status , reply := owner.do( t , http.MethodGet , base , nil )
	if status != 200 || reply[ "access" ] != "view" || reply[ "board" ].( map[ string ]any )[ "archived_at" ] == nil {
		t.Errorf( "archived board payload: %d access=%v" , status , reply[ "access" ] )
	}

	for _ , client := range []*browser{ owner , alice } {
		if status , reply := send( t , client , base + "/cards" , fiber.Map{ "title": "New" , "list_id": board.Lists[ 0 ].ID } ); status != 409 || reply[ "error" ] != "board_archived" {
			t.Errorf( "add a card to an archived board: %d %v" , status , reply )
		}
		if status , reply := send( t , client , base + "/cards/" + itoa( card.ID ) , fiber.Map{ "title": "Changed" } ); status != 409 || reply[ "error" ] != "board_archived" {
			t.Errorf( "edit a card on an archived board: %d %v" , status , reply )
		}
		if status , _ := send( t , client , base + "/cards/" + itoa( card.ID ) + "/comments" , fiber.Map{ "text": "hi" } ); status != 409 {
			t.Errorf( "comment on an archived board: %d" , status )
		}
	}
	if status , _ := send( t , owner , base , fiber.Map{ "title": "Renamed" } ); status != 409 {
		t.Errorf( "board settings on an archived board: %d" , status )
	}
	if status , reply := send( t , alice , base + "/duplicate" , fiber.Map{ "title": "Copy" } ); status != 200 {
		t.Errorf( "duplicate still works: %d %v" , status , reply )
	} else if copied , _ := models.GetBoard( store , uint64( reply[ "id" ].( float64 ) ) ); copied.Archived() {
		t.Error( "the copy starts unarchived" )
	}

	// Archive for me, on top: Alice's home page only.
	if status , _ := send( t , alice , base + "/archive" , fiber.Map{ "archived": true , "scope": "me" } ); status != 200 {
		t.Errorf( "archive for me on an archived board: %d" , status )
	}
	if ref := models.GetBoardRef( store , fixture.alice.ID , board.ID ); ref == nil || ref.Archived == false { t.Error( "alice's own archive" ) }
	if ref := models.GetBoardRef( store , fixture.owner.ID , board.ID ); ref.Archived { t.Error( "the owner's row is untouched" ) }

	if status , _ := send( t , owner , base + "/archive" , fiber.Map{ "archived": false , "scope": "everyone" } ); status != 200 {
		t.Fatalf( "unarchive: %d" , status )
	}
	if status , reply := send( t , alice , base + "/cards/" + itoa( card.ID ) , fiber.Map{ "title": "Changed" } ); status != 200 {
		t.Errorf( "editable again once unarchived: %d %v" , status , reply )
	}
}

// Delete moves a board to its owner's trash; restore brings it back for
// everyone; delete forever only from the trash, and only with a session.
func TestBoardTrashRoutes( t *testing.T ) {
	fixture := newAuthFixture( t )
	store := fixture.store
	board , _ := models.CreateBoard( store , fixture.owner.ID , "Lab" , nil )
	models.SetMember( store , board.ID , fixture.alice.ID , models.BoardRoleEditor )
	owner := fixture.signIn( t , fixture.owner.ID )
	alice := fixture.signIn( t , fixture.alice.ID )
	base := "/api/boards/" + itoa( board.ID )

	if status , _ := send( t , alice , base + "/delete" , fiber.Map{} ); status != 403 {
		t.Errorf( "an editor cannot delete: %d" , status )
	}
	// Archived boards can still go to the trash.
	models.SetBoardArchived( store , board.ID , true , "Owner" )
	if status , reply := send( t , owner , base + "/delete" , fiber.Map{} ); status != 200 {
		t.Fatalf( "delete to trash: %d %v" , status , reply )
	}
	if status , _ := alice.do( t , http.MethodGet , base , nil ); status != 404 {
		t.Errorf( "a trashed board is not found: %d" , status )
	}

	status , reply := owner.do( t , http.MethodGet , "/api/board-trash" , nil )
	rows , _ := reply[ "boards" ].( []any )
	if status != 200 || len( rows ) != 1 { t.Fatalf( "owner's trash: %d %v" , status , reply ) }
	if _ , reply = alice.do( t , http.MethodGet , "/api/board-trash" , nil ); len( reply[ "boards" ].( []any ) ) != 0 {
		t.Errorf( "alice has none of it: %v" , reply )
	}
	if status , _ = send( t , alice , "/api/board-trash/" + itoa( board.ID ) + "/restore" , fiber.Map{} ); status != 404 {
		t.Errorf( "only the owner restores: %d" , status )
	}
	if status , reply = send( t , owner , "/api/board-trash/" + itoa( board.ID ) + "/restore" , fiber.Map{} ); status != 200 || reply[ "slug" ] != "lab" {
		t.Fatalf( "restore: %d %v" , status , reply )
	}
	if status , _ = alice.do( t , http.MethodGet , base , nil ); status != 200 {
		t.Errorf( "restored for everyone: %d" , status )
	}

	if status , _ = send( t , owner , "/api/board-trash/" + itoa( board.ID ) + "/delete" , fiber.Map{} ); status != 404 {
		t.Errorf( "delete forever needs it in the trash first: %d" , status )
	}
	send( t , owner , base + "/delete" , fiber.Map{} )
	if status , _ = send( t , owner , "/api/board-trash/" + itoa( board.ID ) + "/delete" , fiber.Map{} ); status != 200 {
		t.Fatalf( "delete forever: %d" , status )
	}
	if _ , reply = owner.do( t , http.MethodGet , "/api/board-trash" , nil ); len( reply[ "boards" ].( []any ) ) != 0 {
		t.Errorf( "gone from the trash: %v" , reply )
	}

	other , _ := models.CreateBoard( store , fixture.owner.ID , "Other" , nil )
	models.TrashBoard( store , other.ID , fixture.owner.ID )
	if status , reply = send( t , owner , "/api/board-trash/empty" , fiber.Map{} ); status != 200 || reply[ "removed" ] != float64( 1 ) {
		t.Errorf( "empty trash: %d %v" , status , reply )
	}
}

// The trash takes a browser session; archiving takes a key, except "me" on
// a key held to some boards.
func TestBoardTrashAndArchiveWithKeys( t *testing.T ) {
	fixture := newMCPFixture( t )
	board , _ := models.CreateBoard( fixture.store , fixture.owner.ID , "Lab" , nil )
	base := "/api/boards/" + itoa( board.ID )

	if status , _ := commentCall( t , fixture , http.MethodGet , "/api/board-trash" , "" , "" , fixture.user_key , "" ); status == 200 {
		t.Errorf( "a key cannot read the board trash: %d" , status )
	}
	if status , _ := commentCall( t , fixture , http.MethodPost , base + "/delete" , "{}" , "" , fixture.user_key , "" ); status == 200 {
		t.Errorf( "a key cannot delete a board: %d" , status )
	}
	if status , reply := commentCall( t , fixture , http.MethodPost , base + "/archive" , `{"archived":true,"scope":"everyone"}` , "" , fixture.user_key , "" ); status != 200 {
		t.Errorf( "a key archives: %d %v" , status , reply )
	}

	_ , reply := fixture.post( t , fixture.user_key , callTool( "archive_board" , `{"board":"lab","archived":false,"scope":"everyone"}` ) )
	if result , _ := reply[ "result" ].( map[ string ]any ); result == nil || result[ "isError" ] == true {
		t.Errorf( "archive_board tool: %v" , reply )
	}
	if fresh , _ := models.GetBoard( fixture.store , board.ID ); fresh.Archived() { t.Error( "the tool unarchived it" ) }
}
