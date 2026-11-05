package routes

import (
	io "io"
	http "net/http"
	strings "strings"
	testing "testing"
	time "time"

	encryption "todoboard/server/encryption"
	models "todoboard/server/models"
)

// A key's board limit from the outside, over both doors. "Secret" is written
// on everything the key should not reach, so a leak shows up as that word
// wherever it lands, whichever route it came through.

type keyBoards struct {
	*mcpFixture
	open    *models.Board
	blocked *models.Board
	sealed  *models.Board // reached only through one shared card
	list_id uint64        // on blocked
	card_id uint64        // on blocked
	item_id uint64        // on blocked
}

func newKeyBoards( t *testing.T ) ( fixture *keyBoards ) {
	t.Helper()
	base := newMCPFixture( t )
	member := base.member.ID
	open , err := models.CreateBoard( base.store , member , "Open" , []string{ "To do" } )
	if err != nil { t.Fatal( err ) }
	if _ , err = models.CreateCard( base.store , open.ID , open.Lists[ 0 ].ID , "Visible" , nil , member , "Member" ); err != nil { t.Fatal( err ) }
	blocked , err := models.CreateBoard( base.store , member , "Secret board" , []string{ "To do" } )
	if err != nil { t.Fatal( err ) }
	card , err := models.CreateCard( base.store , blocked.ID , blocked.Lists[ 0 ].ID , "Secret card" , nil , member , "Member" )
	if err != nil { t.Fatal( err ) }
	sealed , err := models.CreateBoard( base.store , base.owner.ID , "Secret shared board" , []string{ "To do" } )
	if err != nil { t.Fatal( err ) }
	shared_card , err := models.CreateCard( base.store , sealed.ID , sealed.Lists[ 0 ].ID , "Secret shared card" , nil , base.owner.ID , "Owner" )
	if err != nil { t.Fatal( err ) }
	if _ , err = models.ShareCardWithUser( base.store , sealed.ID , shared_card.ID , member , models.ModeEdit , "Owner" ); err != nil { t.Fatal( err ) }
	if _ , err = models.CreateFolder( base.store , member , "Secret folder" ); err != nil { t.Fatal( err ) }
	fixture = &keyBoards{ mcpFixture: base , open: open , blocked: blocked , sealed: sealed , list_id: blocked.Lists[ 0 ].ID , card_id: card.ID , item_id: 1 }
	return
}

func ( fixture *keyBoards ) key( t *testing.T , mode string , ids ...uint64 ) ( credential string , key *models.APIKey ) {
	t.Helper()
	credential , key , err := models.IssueAPIKey( fixture.store , fixture.member.ID , "limited" , models.RoleUser , time.Hour , mode , ids )
	if err != nil { t.Fatal( err ) }
	return
}

// session signs the member in the way a browser ends up signed in.
func ( fixture *keyBoards ) session( t *testing.T ) ( cookie string , csrf string ) {
	t.Helper()
	token , err := models.IssueLoginToken( fixture.store , fixture.member.ID , time.Hour )
	if err != nil { t.Fatal( err ) }
	credential , _ , err := models.RedeemLoginToken( fixture.store , token , time.Hour )
	if err != nil { t.Fatal( err ) }
	session , _ := models.LoadSession( fixture.store , credential )
	sealed , err := encryption.ChaChaEncryptString( fixture.cfg.SecretKey , credential )
	if err != nil { t.Fatal( err ) }
	cookie , csrf = fixture.cfg.CookieName+"="+sealed , session.CSRFToken
	return
}

func ( fixture *keyBoards ) call( t *testing.T , method string , path string , key string , cookie string , body string ) ( status int , text string ) {
	t.Helper()
	request , _ := http.NewRequest( method , path , strings.NewReader( body ) )
	request.Header.Set( "Content-Type" , "application/json" )
	if key != "" { request.Header.Set( "Authorization" , "Bearer "+key ) }
	if cookie != "" { request.Header.Set( "Cookie" , cookie ) }
	response , err := fixture.app.Test( request )
	if err != nil { t.Fatal( err ) }
	raw , _ := io.ReadAll( response.Body )
	status , text = response.StatusCode , string( raw )
	return
}

// Both ways of keeping a key off the same two boards have to behave alike.
func ( fixture *keyBoards ) limitedKeys( t *testing.T ) ( keys map[ string ]string ) {
	t.Helper()
	only , _ := fixture.key( t , models.BoardAccessOnly , fixture.open.ID )
	except , _ := fixture.key( t , models.BoardAccessExcept , fixture.blocked.ID , fixture.sealed.ID )
	keys = map[ string ]string{ "only" : only , "except" : except }
	return
}

func TestLimitedKeyOverREST( t *testing.T ) {
	fixture := newKeyBoards( t )
	open , blocked , sealed := itoa( fixture.open.ID ) , itoa( fixture.blocked.ID ) , itoa( fixture.sealed.ID )
	card := itoa( fixture.card_id )

	for name , key := range fixture.limitedKeys( t ) {
		status , text := fixture.call( t , http.MethodGet , "/api/boards/"+open , key , "" , "" )
		if status != 200 || strings.Contains( text , "Visible" ) == false {
			t.Errorf( "%s: allowed board: %d %s" , name , status , text )
		}

		// By id and by slug, read and write, every one reported as not there.
		denied := [][ 2 ]string{
			{ http.MethodGet , "/api/boards/" + blocked },
			{ http.MethodGet , "/api/boards/" + fixture.blocked.Slug },
			{ http.MethodGet , "/api/boards/" + blocked + "/stamp" },
			{ http.MethodGet , "/api/boards/" + blocked + "/export" },
			{ http.MethodGet , "/api/boards/" + blocked + "/trash" },
			{ http.MethodGet , "/api/boards/" + blocked + "/cards/" + card + "/image" },
			{ http.MethodPost , "/api/boards/" + blocked + "/cards/" + card },
			{ http.MethodGet , "/api/boards/" + sealed },
		}
		for _ , item := range denied {
			status , text = fixture.call( t , item[ 0 ] , item[ 1 ] , key , "" , `{"done":true}` )
			if status != 404 || strings.Contains( text , "board_not_found" ) == false {
				t.Errorf( "%s: %s %s: %d %s" , name , item[ 0 ] , item[ 1 ] , status , text )
			}
		}

		for _ , path := range []string{ "/api/boards" , "/api/backup" } {
			status , text = fixture.call( t , http.MethodGet , path , key , "" , "" )
			if status != 200 || strings.Contains( text , "Open" ) == false || strings.Contains( text , "Secret" ) {
				t.Errorf( "%s: %s: %d %s" , name , path , status , text )
			}
		}

		refused := []string{ "/api/notify/prefs" , "/api/folders" , "/api/folders/order" , "/api/boards/" + open + "/folder" }
		for _ , path := range refused {
			status , text = fixture.call( t , http.MethodPost , path , key , "" , `{"pushover_user_key":"stolen","name":"x"}` )
			if status != 403 {
				t.Errorf( "%s: POST %s: %d %s" , name , path , status , text )
			}
		}
		prefs , _ := models.GetUserPrefs( fixture.store , fixture.member.ID )
		if prefs.PushoverUserKey != "" { t.Errorf( "%s: changed where reminders go" , name ) }
	}
}

func TestOnlyKeysCannotMakeBoards( t *testing.T ) {
	fixture := newKeyBoards( t )
	keys := fixture.limitedKeys( t )
	make_board , import_board := `{"title":"New"}` , `{"board":{"title":"New","lists":[{"title":"A"}]}}`

	if status , text := fixture.call( t , http.MethodPost , "/api/boards" , keys[ "only" ] , "" , make_board ); status != 403 {
		t.Errorf( "only key created a board: %d %s" , status , text )
	}
	if status , text := fixture.call( t , http.MethodPost , "/api/backup/board" , keys[ "only" ] , "" , import_board ); status != 403 {
		t.Errorf( "only key imported a board: %d %s" , status , text )
	}
	_ , reply := fixture.post( t , keys[ "only" ] , callTool( "create_board" , `{"title":"New"}` ) )
	if failed , text := toolFailed( t , reply ); failed == false {
		t.Errorf( "only key created a board over MCP: %s" , text )
	}

	// The board is off nobody's list, so an "all except" key can open it.
	status , text := fixture.call( t , http.MethodPost , "/api/boards" , keys[ "except" ] , "" , make_board )
	if status != 200 { t.Fatalf( "except key: %d %s" , status , text ) }
	_ , reply = fixture.post( t , keys[ "except" ] , callTool( "create_board" , `{"title":"Newer"}` ) )
	created := itoa( uint64( toolResult( t , reply )[ "id" ].( float64 ) ) )
	_ , reply = fixture.post( t , keys[ "except" ] , callTool( "get_board" , `{"board":`+created+`}` ) )
	toolResult( t , reply )
}

func TestLimitedKeyOverMCP( t *testing.T ) {
	fixture := newKeyBoards( t )
	board , list , card := itoa( fixture.blocked.ID ) , itoa( fixture.list_id ) , itoa( fixture.card_id )

	for name , key := range fixture.limitedKeys( t ) {
		tools := map[ string ]string{
			"get_board":             `{"board":`+board+`}`,
			"get_card":              `{"board":`+board+`,"card_id":`+card+`}`,
			"find_cards":            `{"board":`+board+`}`,
			"create_list":           `{"board":`+board+`,"title":"x"}`,
			"update_list":           `{"board":`+board+`,"list_id":`+list+`,"title":"x"}`,
			"create_label":          `{"board":`+board+`,"name":"x","color":"red"}`,
			"create_card":           `{"board":`+board+`,"list_id":`+list+`,"title":"x"}`,
			"update_card":           `{"board":`+board+`,"card_id":`+card+`,"done":true}`,
			"add_checklist_item":    `{"board":`+board+`,"card_id":`+card+`,"text":"x"}`,
			"update_checklist_item": `{"board":`+board+`,"card_id":`+card+`,"item_id":`+itoa( fixture.item_id )+`,"done":true}`,
		}
		for tool , args := range tools {
			_ , reply := fixture.post( t , key , callTool( tool , args ) )
			if failed , text := toolFailed( t , reply ); failed == false || strings.Contains( text , "board_not_found" ) == false {
				t.Errorf( "%s: %s: %s" , name , tool , text )
			}
		}

		_ , reply := fixture.post( t , key , callTool( "list_boards" , `{}` ) )
		if _ , text := toolFailed( t , reply ); strings.Contains( text , "Open" ) == false || strings.Contains( text , "Secret" ) {
			t.Errorf( "%s: list_boards: %s" , name , text )
		}
		_ , reply = fixture.post( t , key , callTool( "find_cards" , `{}` ) )
		if _ , text := toolFailed( t , reply ); strings.Contains( text , "Visible" ) == false || strings.Contains( text , "Secret" ) {
			t.Errorf( "%s: find_cards: %s" , name , text )
		}
	}
}

func TestSessionAndEditingALimit( t *testing.T ) {
	fixture := newKeyBoards( t )
	credential , key := fixture.key( t , models.BoardAccessOnly , fixture.open.ID )
	cookie , csrf := fixture.session( t )
	blocked := "/api/boards/" + itoa( fixture.blocked.ID )
	edit := "/api/keys/" + key.ID + "/boards"
	widen := `{"csrf_token":"` + csrf + `","board_access":"only","board_ids":[` + itoa( fixture.open.ID ) + `,` + itoa( fixture.blocked.ID ) + `]}`

	// The limit is the key's. The same account in a browser is not held to it,
	// even when the request also carries the limited key.
	if status , text := fixture.call( t , http.MethodGet , blocked , "" , cookie , "" ); status != 200 { t.Errorf( "session: %d %s" , status , text ) }
	if status , text := fixture.call( t , http.MethodGet , blocked , credential , cookie , "" ); status != 200 { t.Errorf( "session and key: %d %s" , status , text ) }
	if status , text := fixture.call( t , http.MethodGet , "/api/boards" , "" , cookie , "" ); status != 200 || strings.Contains( text , "Secret folder" ) == false {
		t.Errorf( "session list: %d %s" , status , text )
	}

	// A leaked key cannot widen itself, a browser needs its CSRF token, and
	// someone else's key is not there to edit.
	if status , text := fixture.call( t , http.MethodPost , edit , credential , "" , widen ); status != 403 { t.Errorf( "key edited itself: %d %s" , status , text ) }
	if status , text := fixture.call( t , http.MethodPost , edit , "" , cookie , `{"board_access":""}` ); status != 403 { t.Errorf( "no csrf: %d %s" , status , text ) }
	other , other_key , err := models.IssueAPIKey( fixture.store , fixture.owner.ID , "other" , models.RoleUser , time.Hour , models.BoardAccessAll , nil )
	if err != nil || other == "" { t.Fatal( err ) }
	if status , text := fixture.call( t , http.MethodPost , "/api/keys/"+other_key.ID+"/boards" , "" , cookie , widen ); status != 404 { t.Errorf( "edited another's key: %d %s" , status , text ) }

	if status , text := fixture.call( t , http.MethodGet , blocked , credential , "" , "" ); status != 404 { t.Fatalf( "before edit: %d %s" , status , text ) }
	if status , text := fixture.call( t , http.MethodPost , edit , "" , cookie , widen ); status != 200 { t.Fatalf( "edit: %d %s" , status , text ) }
	if status , text := fixture.call( t , http.MethodGet , blocked , credential , "" , "" ); status != 200 { t.Errorf( "after edit: %d %s" , status , text ) }

	// A board the account is not on stays out of reach of any list.
	bad := `{"csrf_token":"` + csrf + `","board_access":"only","board_ids":[99999]}`
	if status , text := fixture.call( t , http.MethodPost , edit , "" , cookie , bad ); status != 400 { t.Errorf( "bad board: %d %s" , status , text ) }
}

// The owner sees which boards; an admin looking across everyone's keys sees
// only how many, since admin rights grant nothing on boards.
func TestKeyListsAndBoards( t *testing.T ) {
	fixture := newKeyBoards( t )
	_ , key := fixture.key( t , models.BoardAccessOnly , fixture.open.ID )
	own := ownKeyRow( key )
	if own[ "board_access" ] != "only" || len( own[ "board_ids" ].( []uint64 ) ) != 1 {
		t.Errorf( "own row: %v" , own )
	}
	admin := apiKeyRow( key , "Member" )
	if _ , present := admin[ "board_ids" ]; present || admin[ "board_count" ] != 1 {
		t.Errorf( "admin row: %v" , admin )
	}
}
