package routes

import (
	http "net/http"
	httptest "net/http/httptest"
	os "os"
	strings "strings"
	testing "testing"
	time "time"

	language "todoboard/server/language"
	models "todoboard/server/models"
	notify "todoboard/server/notify"
)

// The device routes take a browser session; the fixture only has keys, so
// these prove a key is turned away. What a session can do is covered by
// models and notify, which hold the rules.
func TestPushDevicesNeedASession( t *testing.T ) {
	fixture := newMCPFixture( t )
	cases := [][]string{
		{ "GET" , "/api/notify/devices" },
		{ "POST" , "/api/notify/devices" },
		{ "POST" , "/api/notify/devices/abc/delete" },
	}
	for _ , item := range cases {
		request := httptest.NewRequest( item[ 0 ] , item[ 1 ] , strings.NewReader( `{}` ) )
		request.Header.Set( "Content-Type" , "application/json" )
		request.Header.Set( "Authorization" , "Bearer "+fixture.user_key )
		response , err := fixture.app.Test( request )
		if err != nil { t.Fatal( err ) }
		if response.StatusCode < 400 { t.Errorf( "%s %s with a key: %d" , item[ 0 ] , item[ 1 ] , response.StatusCode ) }
		anonymous := httptest.NewRequest( item[ 0 ] , item[ 1 ] , strings.NewReader( `{}` ) )
		if response , err = fixture.app.Test( anonymous ); err != nil { t.Fatal( err ) }
		if response.StatusCode < 400 { t.Errorf( "%s %s signed out: %d" , item[ 0 ] , item[ 1 ] , response.StatusCode ) }
	}
}

// A browser device counts as somewhere to reach someone, so the board panel
// stops asking for a Pushover key.
func TestSubscriptionWarningWithBrowserDevice( t *testing.T ) {
	fixture := newMCPFixture( t )
	cfg := *fixture.cfg
	cfg.WebPush = true
	fixture.handlers.Notify = notify.New( fixture.store , &cfg , nil )
	prefs := models.DefaultUserPrefs()
	if fixture.handlers.canReceive( fixture.owner.ID , prefs ) { t.Errorf( "nothing set up, yet reachable" ) }
	endpoint , p256dh , auth := "https://fcm.googleapis.com/fcm/send/abc" , "BCVxsr7N_eNgVRqvHtD0zTZsEc6-VV-JvLexhqUzORcxaOzi6-AYWXvTBHm4bjyPjs7Vd8pZGH6SRpkNtoIAiw4" , "BTBZMqHH6r4Tts7J_aSIgg"
	if _ , err := models.SavePushSubscription( fixture.store , fixture.owner.ID , endpoint , p256dh , auth , "" ); err != nil { t.Fatal( err ) }
	if fixture.handlers.canReceive( fixture.owner.ID , prefs ) == false { t.Errorf( "a browser did not count" ) }
}

// What an agent does over MCP reaches a board's subscribers like anything
// else: the publishing sits in the helpers both doors share.
func TestActivityPublishedOverMCP( t *testing.T ) {
	fixture := newMCPFixture( t )
	received := make( chan string , 8 )
	pushover := httptest.NewServer( http.HandlerFunc( func( writer http.ResponseWriter , request *http.Request ) {
		request.ParseForm()
		received <- request.PostForm.Get( "title" ) + "\n" + request.PostForm.Get( "message" )
		writer.Write( []byte( `{"status":1}` ) )
	} ) )
	t.Cleanup( pushover.Close )
	raw , err := os.ReadFile( "../../config/language.yaml" )
	if err != nil { t.Fatal( err ) }
	lang , err := language.Parse( raw )
	if err != nil { t.Fatal( err ) }
	cfg := *fixture.cfg
	cfg.PushoverAppToken , cfg.PushoverAPIURL , cfg.DueCheckInterval = "azGDORePK8gMaC0QOYAMyEEuzJnyUi" , pushover.URL , time.Hour
	service := notify.New( fixture.store , &cfg , lang )
	service.Start()
	t.Cleanup( service.Stop )
	fixture.handlers.Notify = service

	board , err := models.CreateBoard( fixture.store , fixture.owner.ID , "Launch" , []string{ "To do" } )
	if err != nil { t.Fatal( err ) }
	if _ , err = models.SetMember( fixture.store , board.ID , fixture.member.ID , models.BoardRoleEditor ); err != nil { t.Fatal( err ) }
	prefs := models.DefaultUserPrefs()
	prefs.PushoverUserKey = "uQiRzpo4DXghDmr9QzzfQu27cmVRsG"
	if err = models.SaveUserPrefs( fixture.store , fixture.member.ID , prefs ); err != nil { t.Fatal( err ) }
	if err = models.SetBoardSubscription( fixture.store , fixture.member.ID , board.ID , &models.EventSet{ Activity: true } ); err != nil { t.Fatal( err ) }

	_ , reply := fixture.post( t , fixture.user_key , callTool( "create_list" , `{"board":`+itoa( board.ID )+`,"title":"Agent column"}` ) )
	toolResult( t , reply )
	select {
	case sent := <-received:
		if strings.Contains( sent , "Launch" ) == false || strings.Contains( sent , "Agent column" ) == false { t.Errorf( "sent: %q" , sent ) }
	case <-time.After( 5 * time.Second ):
		t.Fatal( "nothing reached the subscriber" )
	}
}
