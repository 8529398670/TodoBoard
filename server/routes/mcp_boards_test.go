package routes

import (
	json "encoding/json"
	strings "strings"
	testing "testing"
	time "time"

	models "todoboard/server/models"
)

// The board tools go through server/access exactly like the HTTP routes, so
// these check the access rules from the agent's side: what a key cannot see
// it is told does not exist, and what it may only view it cannot change.

func ( fixture *mcpFixture ) memberKey( t *testing.T ) ( key string ) {
	t.Helper()
	key , _ , err := models.IssueAPIKey( fixture.store , fixture.member.ID , "member" , models.RoleUser , time.Hour )
	if err != nil { t.Fatal( err ) }
	return
}

// toolResult decodes a successful tool call's structured result.
func toolResult( t *testing.T , reply map[ string ]any ) ( result map[ string ]any ) {
	t.Helper()
	failed , text := toolFailed( t , reply )
	if failed { t.Fatalf( "tool refused: %s" , text ) }
	if err := json.Unmarshal( []byte( text ) , &result ); err != nil { t.Fatalf( "result %q: %v" , text , err ) }
	return
}

func TestMCPBoardRoundTrip( t *testing.T ) {
	fixture := newMCPFixture( t )
	key := fixture.user_key

	_ , reply := fixture.post( t , key , callTool( "create_board" , `{"title":"Launch","lists":["To do","Doing","Done"]}` ) )
	board_id := itoa( uint64( toolResult( t , reply )[ "id" ].( float64 ) ) )

	_ , reply = fixture.post( t , key , callTool( "get_board" , `{"board":`+board_id+`}` ) )
	lists := toolResult( t , reply )[ "board" ].( map[ string ]any )[ "lists" ].( []any )
	if len( lists ) != 3 { t.Fatalf( "lists: %v" , lists ) }
	todo := itoa( uint64( lists[ 0 ].( map[ string ]any )[ "id" ].( float64 ) ) )
	done := itoa( uint64( lists[ 2 ].( map[ string ]any )[ "id" ].( float64 ) ) )

	// A board id may come as a string too.
	_ , reply = fixture.post( t , key , callTool( "create_card" , `{"board":"`+board_id+`","list_id":`+todo+`,"title":"Write notes","due_at":"2026-11-02","assignees":[`+itoa( fixture.owner.ID )+`]}` ) )
	card := toolResult( t , reply )
	if card[ "due_at" ] == nil || card[ "due_all_day" ] != true {
		t.Errorf( "create_card did not set the due date: %v" , card )
	}
	card_id := itoa( uint64( card[ "id" ].( float64 ) ) )
	rev := itoa( uint64( card[ "rev" ].( float64 ) ) )

	_ , reply = fixture.post( t , key , callTool( "add_checklist_item" , `{"board":`+board_id+`,"card_id":`+card_id+`,"text":"outline"}` ) )
	if items := toolResult( t , reply )[ "checklist" ].( []any ); len( items ) != 1 {
		t.Errorf( "checklist: %v" , items )
	}

	_ , reply = fixture.post( t , key , callTool( "find_cards" , `{"assigned_to_me":true,"due_before":"2026-12-01"}` ) )
	if found := toolResult( t , reply )[ "cards" ].( []any ); len( found ) != 1 {
		t.Errorf( "find_cards: %v" , found )
	}

	// A rev that is no longer current refuses a title change.
	_ , reply = fixture.post( t , key , callTool( "update_card" , `{"board":`+board_id+`,"card_id":`+card_id+`,"rev":`+rev+`,"title":"Late"}` ) )
	if failed , text := toolFailed( t , reply ); failed == false || strings.Contains( text , "card_stale" ) == false {
		t.Errorf( "stale rev: %s" , text )
	}

	_ , reply = fixture.post( t , key , callTool( "update_card" , `{"board":`+board_id+`,"card_id":`+card_id+`,"list_id":`+done+`,"done":true}` ) )
	if moved := toolResult( t , reply ); moved[ "list_id" ].( float64 ) != lists[ 2 ].( map[ string ]any )[ "id" ].( float64 ) || moved[ "done" ] != true {
		t.Errorf( "update_card: %v" , moved )
	}

	// Done cards drop out of find_cards unless asked for.
	_ , reply = fixture.post( t , key , callTool( "find_cards" , `{"query":"notes"}` ) )
	if found := toolResult( t , reply )[ "cards" ].( []any ); len( found ) != 0 {
		t.Errorf( "done card found by default: %v" , found )
	}
}

func TestMCPBoardAccess( t *testing.T ) {
	fixture := newMCPFixture( t )
	member_key := fixture.memberKey( t )

	board , err := models.CreateBoard( fixture.store , fixture.owner.ID , "Private" , []string{ "To do" } )
	if err != nil { t.Fatal( err ) }
	card , err := models.CreateCard( fixture.store , board.ID , board.Lists[ 0 ].ID , "Secret" , nil , fixture.owner.ID , "Owner" )
	if err != nil { t.Fatal( err ) }
	board_id , card_id := itoa( board.ID ) , itoa( card.ID )

	// Not on the board: it does not exist, by id or by search.
	_ , reply := fixture.post( t , member_key , callTool( "get_board" , `{"board":`+board_id+`}` ) )
	if failed , text := toolFailed( t , reply ); failed == false || strings.Contains( text , "board_not_found" ) == false {
		t.Errorf( "outsider read the board: %s" , text )
	}
	_ , reply = fixture.post( t , member_key , callTool( "find_cards" , `{"query":"secret"}` ) )
	if found := toolResult( t , reply )[ "cards" ].( []any ); len( found ) != 0 {
		t.Errorf( "outsider found cards: %v" , found )
	}

	// A viewer can read but not change.
	if _ , err = models.SetMember( fixture.store , board.ID , fixture.member.ID , models.BoardRoleViewer ); err != nil { t.Fatal( err ) }
	_ , reply = fixture.post( t , member_key , callTool( "get_card" , `{"board":`+board_id+`,"card_id":`+card_id+`}` ) )
	toolResult( t , reply )
	_ , reply = fixture.post( t , member_key , callTool( "update_card" , `{"board":`+board_id+`,"card_id":`+card_id+`,"done":true}` ) )
	if failed , text := toolFailed( t , reply ); failed == false || strings.Contains( text , "forbidden" ) == false {
		t.Errorf( "viewer changed a card: %s" , text )
	}
	_ , reply = fixture.post( t , member_key , callTool( "create_list" , `{"board":`+board_id+`,"title":"Mine"}` ) )
	if failed , _ := toolFailed( t , reply ); failed == false {
		t.Errorf( "viewer added a column" )
	}

	// App admin rights grant nothing on someone else's board: the owner's
	// admin key is refused on a board the member made.
	other , _ := models.CreateBoard( fixture.store , fixture.member.ID , "Member's" , []string{ "A" } )
	_ , reply = fixture.post( t , fixture.admin_key , callTool( "get_board" , `{"board":`+itoa( other.ID )+`}` ) )
	if failed , _ := toolFailed( t , reply ); failed == false {
		t.Errorf( "admin key read a board it is not on" )
	}

	// No tool deletes anything.
	_ , reply = fixture.post( t , fixture.admin_key , rpc( "tools/list" , `{}` ) )
	for _ , name := range toolNames( reply ) {
		if strings.Contains( name , "delete" ) { t.Errorf( "deleting tool offered: %s" , name ) }
	}
}
