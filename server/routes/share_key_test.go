package routes

import (
	json "encoding/json"
	os "os"
	filepath "path/filepath"
	strings "strings"
	testing "testing"

	config "todoboard/server/config"
	logs "todoboard/server/logs"
	models "todoboard/server/models"
)

// POST /api/boards/{board_id}/shares from an API key: an admin-role key may
// make a view link to the whole board, and nothing else; a browser session is
// as it was. The owner's admin_key and user_key come from newMCPFixture.

type shareKeys struct {
	*mcpFixture
	board *models.Board
	path  string
	dir   string
	logger *logs.Logger
	closed bool
}

// closeLog flushes the log, once: Logger.Close may not be called twice.
func ( fixture *shareKeys ) closeLog() {
	if fixture.closed { return }
	fixture.closed = true
	fixture.logger.Close()
}

func newShareKeys( t *testing.T ) ( fixture *shareKeys ) {
	t.Helper()
	base := newMCPFixture( t )
	board , err := models.CreateBoard( base.store , base.owner.ID , "Roadmap" , []string{ "To do" , "Done" } )
	if err != nil { t.Fatal( err ) }
	dir := t.TempDir()
	logger := logs.New( config.LogsConfig{ Enabled: true , Dir: dir , RetentionDays: 7 , MaxTotalBytes: 1 << 20 , MaxFileBytes: 1 << 20 } , base.cfg.SecretKey , false )
	logger.Start()
	base.handlers.Logs = logger
	fixture = &shareKeys{ mcpFixture: base , board: board , path: "/api/boards/" + itoa( board.ID ) + "/shares" , dir: dir , logger: logger }
	t.Cleanup( fixture.closeLog )
	return
}

func ( fixture *shareKeys ) make( t *testing.T , key string , cookie string , body string ) ( status int , reply map[ string ]any , text string ) {
	t.Helper()
	keys := &keyBoards{ mcpFixture: fixture.mcpFixture }
	status , text = keys.call( t , "POST" , fixture.path , key , cookie , body )
	json.Unmarshal( []byte( text ) , &reply )
	return
}

// auditLines are the log's lines that mention text, once the log is flushed.
func ( fixture *shareKeys ) auditLines( t *testing.T , text string ) ( found []string ) {
	t.Helper()
	fixture.closeLog()
	files , _ := filepath.Glob( filepath.Join( fixture.dir , "*.jsonl" ) )
	for _ , file := range files {
		raw , _ := os.ReadFile( file )
		for _ , line := range strings.Split( string( raw ) , "\n" ) {
			if strings.Contains( line , text ) { found = append( found , line ) }
		}
	}
	return
}

func TestAdminKeyMakesAViewShareLink( t *testing.T ) {
	fixture := newShareKeys( t )
	status , reply , text := fixture.make( t , fixture.admin_key , "" , `{"mode":"view","label":"Public roadmap","days":7}` )
	if status != 201 { t.Fatalf( "admin key, view: %d %s, want 201" , status , text ) }
	share_path , _ := reply[ "share_path" ].( string )
	if strings.HasPrefix( share_path , "/s/" ) == false || len( share_path ) < 10 { t.Fatalf( "share_path %q" , share_path ) }
	if reply[ "mode" ] != "view" || reply[ "card_id" ] != float64( 0 ) || reply[ "list_id" ] != float64( 0 ) { t.Fatalf( "not a view board link: %s" , text ) }
	link_id := reply[ "id" ].( string )
	links , _ := models.ListShareLinks( fixture.store , fixture.board.ID )
	if len( links ) != 1 || links[ 0 ].ID != link_id || links[ 0 ].Live() == false { t.Fatalf( "links on the board: %+v" , links ) }

	lines := fixture.auditLines( t , link_id )
	if len( lines ) != 1 { t.Fatalf( "audit entries naming the link: %d, want 1" , len( lines ) ) }
	for _ , want := range []string{ `"key_name":"admin"` , `view share link made with API key \"admin\"` , `"kind":"admin"` } {
		if strings.Contains( lines[ 0 ] , want ) == false { t.Fatalf( "audit entry lacks %s: %s" , want , lines[ 0 ] ) }
	}
	if strings.Contains( lines[ 0 ] , strings.TrimPrefix( share_path , "/s/" ) ) { t.Fatal( "the audit entry holds the link's secret" ) }
}

func TestKeysCannotMakeOtherShareLinks( t *testing.T ) {
	fixture := newShareKeys( t )
	list_id := itoa( fixture.board.Lists[ 0 ].ID )
	card , err := models.CreateCard( fixture.store , fixture.board.ID , fixture.board.Lists[ 0 ].ID , "Card" , nil , fixture.owner.ID , "Owner" )
	if err != nil { t.Fatal( err ) }
	cases := []struct{ name , key , body , want string }{
		{ "admin key, edit" , fixture.admin_key , `{"mode":"edit"}` , shareKeyViewOnly },
		{ "admin key, comment" , fixture.admin_key , `{"mode":"comment"}` , shareKeyViewOnly },
		{ "admin key, view of a column" , fixture.admin_key , `{"mode":"view","list_id":` + list_id + `}` , shareKeyViewOnly },
		{ "admin key, view of a card" , fixture.admin_key , `{"mode":"view","card_id":` + itoa( card.ID ) + `}` , shareKeyViewOnly },
		{ "user key, view" , fixture.user_key , `{"mode":"view"}` , shareKeyNotAdmin },
		{ "user key, edit" , fixture.user_key , `{"mode":"edit"}` , shareKeyNotAdmin },
	}
	for _ , each := range cases {
		status , reply , text := fixture.make( t , each.key , "" , each.body )
		if status != 403 || reply[ "error" ] != each.want { t.Fatalf( "%s: %d %s, want 403 %q" , each.name , status , text , each.want ) }
	}
	if links , _ := models.ListShareLinks( fixture.store , fixture.board.ID ); len( links ) != 0 { t.Fatalf( "a refused key made %d links" , len( links ) ) }
}

// The owner check still holds for an admin key: a member's board is not the
// key's to share, whatever its role.
func TestAdminKeyCannotShareSomeoneElsesBoard( t *testing.T ) {
	fixture := newShareKeys( t )
	theirs , err := models.CreateBoard( fixture.store , fixture.member.ID , "Theirs" , []string{ "To do" } )
	if err != nil { t.Fatal( err ) }
	if _ , err = models.SetMember( fixture.store , theirs.ID , fixture.owner.ID , models.BoardRoleEditor ); err != nil { t.Fatal( err ) }
	keys := &keyBoards{ mcpFixture: fixture.mcpFixture }
	status , text := keys.call( t , "POST" , "/api/boards/" + itoa( theirs.ID ) + "/shares" , fixture.admin_key , "" , `{"mode":"view"}` )
	if status != 403 && status != 404 { t.Fatalf( "admin key on an editor's board: %d %s" , status , text ) }
	if links , _ := models.ListShareLinks( fixture.store , theirs.ID ); len( links ) != 0 { t.Fatal( "a key made a link on a board its account does not own" ) }
}

func TestSessionShareLinksUnchanged( t *testing.T ) {
	fixture := newShareKeys( t )
	admins := &adminKeys{ mcpFixture: fixture.mcpFixture }
	cookie , csrf := admins.signIn( t , fixture.owner.ID )
	for _ , mode := range []string{ "view" , "comment" , "edit" } {
		status , reply , text := fixture.make( t , "" , cookie , `{"csrf_token":"` + csrf + `","mode":"` + mode + `","list_id":` + itoa( fixture.board.Lists[ 1 ].ID ) + `}` )
		if status != 201 || strings.HasPrefix( reply[ "share_path" ].( string ) , "/s/" ) == false { t.Fatalf( "session, %s: %d %s" , mode , status , text ) }
	}
	// Still behind the CSRF token, and still the owner's alone.
	if status , _ , text := fixture.make( t , "" , cookie , `{"mode":"view"}` ); status != 403 { t.Fatalf( "session without csrf: %d %s" , status , text ) }
	member_cookie , member_csrf := admins.signIn( t , fixture.member.ID )
	if _ , err := models.SetMember( fixture.store , fixture.board.ID , fixture.member.ID , models.BoardRoleEditor ); err != nil { t.Fatal( err ) }
	if status , _ , text := fixture.make( t , "" , member_cookie , `{"csrf_token":"` + member_csrf + `","mode":"view"}` ); status != 403 && status != 404 { t.Fatalf( "editor's session: %d %s" , status , text ) }
	if lines := fixture.auditLines( t , "share link made with API key" ); len( lines ) != 0 { t.Fatalf( "a session's link was audited as a key's: %v" , lines ) }
}

func TestCreateViewShareTool( t *testing.T ) {
	fixture := newShareKeys( t )
	board := itoa( fixture.board.ID )
	_ , reply := fixture.post( t , fixture.admin_key , callTool( "create_view_share" , `{"board":`+board+`,"mode":"view","label":"Agent"}` ) )
	result := toolResult( t , reply )
	share_url , _ := result[ "share_url" ].( string )
	if strings.Contains( share_url , "/s/" ) == false || strings.HasSuffix( share_url , result[ "share_path" ].( string ) ) == false { t.Fatalf( "share_url %q, result %v" , share_url , result ) }

	_ , reply = fixture.post( t , fixture.admin_key , callTool( "create_view_share" , `{"board":`+board+`,"mode":"edit"}` ) )
	if failed , text := toolFailed( t , reply ); failed == false || strings.Contains( text , shareKeyViewOnly ) == false { t.Fatalf( "edit over MCP: %v %s" , failed , text ) }

	// Not offered to a user-role key, and refused if called anyway.
	_ , listed := fixture.post( t , fixture.user_key , rpc( "tools/list" , `{}` ) )
	for _ , name := range toolNames( listed ) {
		if name == "create_view_share" { t.Fatal( "create_view_share is listed for a user-role key" ) }
	}
	_ , listed = fixture.post( t , fixture.admin_key , rpc( "tools/list" , `{}` ) )
	if strings.Contains( strings.Join( toolNames( listed ) , "," ) , "create_view_share" ) == false { t.Fatal( "create_view_share is not listed for an admin key" ) }
	_ , reply = fixture.post( t , fixture.user_key , callTool( "create_view_share" , `{"board":`+board+`,"mode":"view"}` ) )
	if reply[ "error" ] == nil {
		if failed , _ := toolFailed( t , reply ); failed == false { t.Fatal( "a user-role key made a share link over MCP" ) }
	}
	if links , _ := models.ListShareLinks( fixture.store , fixture.board.ID ); len( links ) != 1 { t.Fatalf( "links: %d, want 1" , len( links ) ) }
}
