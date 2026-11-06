package routes

import (
	json "encoding/json"
	http "net/http"
	strings "strings"
	testing "testing"
	time "time"

	encryption "todoboard/server/encryption"
	models "todoboard/server/models"
)

type adminKeys struct {
	*mcpFixture
	admin  *models.User
	other  *models.User
	cookie string
	csrf   string
}

// A second admin signed in the way a browser is: the owner is the fixture's
// other admin, and the owner is who this has to be kept away from.
func newAdminKeys( t *testing.T ) ( fixture *adminKeys ) {
	t.Helper()
	base := newMCPFixture( t )
	admin , err := models.CreateUser( base.store , "Second Admin" , models.RoleAdmin )
	if err != nil { t.Fatal( err ) }
	other , err := models.CreateUser( base.store , "Third Admin" , models.RoleAdmin )
	if err != nil { t.Fatal( err ) }
	fixture = &adminKeys{ mcpFixture: base , admin: admin , other: other }
	fixture.cookie , fixture.csrf = fixture.signIn( t , admin.ID )
	return
}

func ( fixture *adminKeys ) signIn( t *testing.T , user_id uint64 ) ( cookie string , csrf string ) {
	t.Helper()
	token , err := models.IssueLoginToken( fixture.store , user_id , time.Hour )
	if err != nil { t.Fatal( err ) }
	credential , _ , err := models.RedeemLoginToken( fixture.store , token , time.Hour )
	if err != nil { t.Fatal( err ) }
	session , _ := models.LoadSession( fixture.store , credential )
	sealed , err := encryption.ChaChaEncryptString( fixture.cfg.SecretKey , credential )
	if err != nil { t.Fatal( err ) }
	cookie , csrf = fixture.cfg.CookieName+"="+sealed , session.CSRFToken
	return
}

func ( fixture *adminKeys ) call( t *testing.T , method string , path string , key string , cookie string , body string ) ( status int , reply string ) {
	t.Helper()
	keys := &keyBoards{ mcpFixture: fixture.mcpFixture }
	status , reply = keys.call( t , method , path , key , cookie , body )
	return
}

func ( fixture *adminKeys ) make( t *testing.T , user_id uint64 , role string ) ( status int , reply string ) {
	t.Helper()
	body := `{"csrf_token":"` + fixture.csrf + `","user_id":` + itoa( user_id ) + `,"name":"bot","role":"` + role + `"}`
	status , reply = fixture.call( t , http.MethodPost , "/api/admin/keys" , "" , fixture.cookie , body )
	return
}

func TestAdminMakesAKeyAsAUser( t *testing.T ) {
	fixture := newAdminKeys( t )
	board , err := models.CreateBoard( fixture.store , fixture.member.ID , "Members only" , []string{ "To do" } )
	if err != nil { t.Fatal( err ) }

	status , reply := fixture.make( t , fixture.member.ID , "" )
	if status != 201 { t.Fatalf( "%d %s" , status , reply ) }
	var made struct {
		Credential    string `json:"credential"`
		ID            string `json:"id"`
		UserID        uint64 `json:"user_id"`
		Role          string `json:"role"`
		CreatedBy     uint64 `json:"created_by"`
		CreatedByName string `json:"created_by_name"`
	}
	if err = json.Unmarshal( []byte( reply ) , &made ); err != nil { t.Fatal( err ) }
	if made.Credential == "" || made.UserID != fixture.member.ID || made.Role != models.RoleUser {
		t.Errorf( "the key should be the member's, at user level: %s" , reply )
	}
	if made.CreatedBy != fixture.admin.ID || made.CreatedByName != "Second Admin" { t.Errorf( "the maker was not kept: %s" , reply ) }

	// It is that person: it reaches their board, which the admin is not on.
	path := "/api/boards/" + itoa( board.ID )
	if status , reply = fixture.call( t , http.MethodGet , path , made.Credential , "" , "" ); status != 200 {
		t.Errorf( "the key should reach its user's board: %d %s" , status , reply )
	}
	if status , _ = fixture.call( t , http.MethodGet , path , "" , fixture.cookie , "" ); status == 200 {
		t.Error( "the admin has no way onto that board themselves" )
	}

	// The member sees it, with the name on it, and the credential nowhere.
	cookie , _ := fixture.signIn( t , fixture.member.ID )
	status , reply = fixture.call( t , http.MethodGet , "/api/keys" , "" , cookie , "" )
	if status != 200 || strings.Contains( reply , `"created_by_name":"Second Admin"` ) == false { t.Errorf( "own list: %d %s" , status , reply ) }
	if strings.Contains( reply , made.Credential ) { t.Error( "the credential is shown once only" ) }
	status , reply = fixture.call( t , http.MethodGet , "/api/admin/keys" , "" , fixture.cookie , "" )
	if status != 200 || strings.Contains( reply , `"created_by_name":"Second Admin"` ) == false { t.Errorf( "admin list: %d %s" , status , reply ) }

	// Once the admin is gone their id stays, for the page to say "an admin".
	if _ , err = models.DeleteUser( fixture.store , fixture.admin.ID , fixture.owner.ID ); err != nil { t.Fatal( err ) }
	status , reply = fixture.call( t , http.MethodGet , "/api/keys" , "" , cookie , "" )
	if status != 200 || strings.Contains( reply , `"created_by":` ) == false || strings.Contains( reply , "created_by_name" ) {
		t.Errorf( "after the admin went: %d %s" , status , reply )
	}
}

func TestAdminKeyLimits( t *testing.T ) {
	fixture := newAdminKeys( t )

	if status , reply := fixture.make( t , fixture.owner.ID , "" ); status != 403 { t.Errorf( "a key for the owner: %d %s" , status , reply ) }
	if status , reply := fixture.make( t , fixture.member.ID , models.RoleAdmin ); status != 403 { t.Errorf( "an admin key for a user: %d %s" , status , reply ) }
	if status , reply := fixture.make( t , fixture.other.ID , models.RoleAdmin ); status != 201 { t.Errorf( "an admin key for an admin: %d %s" , status , reply ) }
	if status , reply := fixture.make( t , 99999 , "" ); status != 404 { t.Errorf( "nobody: %d %s" , status , reply ) }

	// No board limit on offer: which boards they are is not an admin's to see.
	body := `{"csrf_token":"` + fixture.csrf + `","user_id":` + itoa( fixture.member.ID ) + `,"name":"bot","board_access":"only","board_ids":[1]}`
	status , reply := fixture.call( t , http.MethodPost , "/api/admin/keys" , "" , fixture.cookie , body )
	if status != 201 || strings.Contains( reply , `"board_access":""` ) == false { t.Errorf( "a limit was taken from the request: %d %s" , status , reply ) }

	// Who may ask: a browser session of an admin, and nothing less.
	plain := `{"user_id":` + itoa( fixture.member.ID ) + `,"name":"bot"}`
	if status , reply = fixture.call( t , http.MethodPost , "/api/admin/keys" , "" , fixture.cookie , plain ); status != 403 { t.Errorf( "no csrf: %d %s" , status , reply ) }
	if status , reply = fixture.call( t , http.MethodPost , "/api/admin/keys" , fixture.admin_key , "" , plain ); status != 403 { t.Errorf( "an admin key made a key: %d %s" , status , reply ) }
	cookie , csrf := fixture.signIn( t , fixture.member.ID )
	asked := `{"csrf_token":"` + csrf + `","user_id":` + itoa( fixture.other.ID ) + `,"name":"bot"}`
	if status , reply = fixture.call( t , http.MethodPost , "/api/admin/keys" , "" , cookie , asked ); status != 403 { t.Errorf( "a user made a key: %d %s" , status , reply ) }

	// A disabled account cannot be given one.
	if err := models.SetUserDisabled( fixture.store , fixture.member.ID , true ); err != nil { t.Fatal( err ) }
	if status , reply = fixture.make( t , fixture.member.ID , "" ); status != 403 { t.Errorf( "a key for a disabled account: %d %s" , status , reply ) }
}

// The control socket and the CLI keep the owner's way in: only a person
// signed in through the app is held to it.
func TestOwnerKeysStillMadeWithoutAnAdmin( t *testing.T ) {
	fixture := newAdminKeys( t )
	_ , key , err := models.IssueAPIKey( fixture.store , fixture.owner.ID , "cli" , models.RoleAdmin , time.Hour , models.BoardAccessAll , nil )
	if err != nil { t.Fatal( err ) }
	if key.CreatedBy != 0 { t.Errorf( "made by nobody: %d" , key.CreatedBy ) }
}
