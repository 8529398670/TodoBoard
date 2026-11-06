package notify

import (
	context "context"
	errors "errors"
	http "net/http"
	httptest "net/http/httptest"
	strings "strings"
	sync "sync"
	testing "testing"
	time "time"

	db "todoboard/server/db"
	models "todoboard/server/models"
)

const (
	browserP256dh = "BCVxsr7N_eNgVRqvHtD0zTZsEc6-VV-JvLexhqUzORcxaOzi6-AYWXvTBHm4bjyPjs7Vd8pZGH6SRpkNtoIAiw4"
	browserAuth   = "BTBZMqHH6r4Tts7J_aSIgg"
)

// mockPushService stands in for a browser's push service.
type mockPushService struct {
	mutex    sync.Mutex
	requests []*http.Request
	status   int
}

func ( mock *mockPushService ) ServeHTTP( writer http.ResponseWriter , request *http.Request ) {
	mock.mutex.Lock()
	defer mock.mutex.Unlock()
	mock.requests = append( mock.requests , request )
	status := mock.status
	if status == 0 { status = http.StatusCreated }
	writer.WriteHeader( status )
}

func ( mock *mockPushService ) count() ( result int ) {
	mock.mutex.Lock()
	defer mock.mutex.Unlock()
	result = len( mock.requests )
	return
}

// withWebPush adds a browser lane talking to a test server, and returns it
// with the endpoint browsers there subscribe to.
func withWebPush( t *testing.T , service *Service , store *db.Store ) ( mock *mockPushService , endpoint string , channel *webPushChannel ) {
	t.Helper()
	mock = &mockPushService{}
	server := httptest.NewServer( mock )
	t.Cleanup( server.Close )
	keys , err := models.EnsureVAPIDKeys( store )
	if err != nil { t.Fatal( err ) }
	channel = &webPushChannel{ store: store , keys: keys , subject: "mailto:ops@example.com" , client: server.Client() , now: time.Now ,
		allowed: func( string ) bool { return true } }
	service.lanes = append( service.lanes , newLane( channel ) )
	endpoint = server.URL + "/push/abc"
	return
}

func TestEndpointAllowlist( t *testing.T ) {
	good := []string{
		"https://fcm.googleapis.com/fcm/send/abc",
		"https://updates.push.services.mozilla.com/wpush/v2/abc",
		"https://web.push.apple.com/abc",
		"https://wns2-par02p.notify.windows.com/w/?token=abc",
		"https://fcm.googleapis.com:443/x",
	}
	for _ , endpoint := range good {
		if EndpointAllowed( endpoint ) == false { t.Errorf( "refused %s" , endpoint ) }
	}
	bad := []string{
		"http://fcm.googleapis.com/x",
		"https://evil.example.com/x",
		"https://fcm.googleapis.com.evil.example.com/x",
		"https://evilpush.apple.com/x",
		"https://user:secret@fcm.googleapis.com/x",
		"https://fcm.googleapis.com:8443/x",
		"https://127.0.0.1/x",
		"https://[::1]/x",
		"https://169.254.169.254/latest/meta-data",
		"not a url",
		"",
	}
	for _ , endpoint := range bad {
		if EndpointAllowed( endpoint ) { t.Errorf( "allowed %s" , endpoint ) }
	}
}

func TestWebPushRequestShape( t *testing.T ) {
	service , _ , store := setup( t )
	mock , endpoint , channel := withWebPush( t , service , store )
	owner , _ := models.CreateUser( store , "Owner" , models.RoleAdmin )
	subscription , err := models.SavePushSubscription( store , owner.ID , endpoint , browserP256dh , browserAuth , "Laptop" )
	if err != nil { t.Fatal( err ) }
	notice := Notice{ Title: strings.Repeat( "t" , 500 ) , Body: strings.Repeat( "b" , 5000 ) , Path: "/b/plans" , Topic: "board-1" , TTL: time.Hour }
	if err = channel.Send( context.Background() , subscription , notice ); err != nil { t.Fatal( err ) }
	if mock.count() != 1 { t.Fatalf( "%d requests" , mock.count() ) }
	sent := mock.requests[ 0 ]
	if sent.Header.Get( "Content-Encoding" ) != "aes128gcm" { t.Errorf( "encoding: %q" , sent.Header.Get( "Content-Encoding" ) ) }
	if sent.Header.Get( "TTL" ) != "3600" || sent.Header.Get( "Topic" ) != "board-1" { t.Errorf( "headers: %v" , sent.Header ) }
	if strings.HasPrefix( sent.Header.Get( "Authorization" ) , "vapid t=" ) == false { t.Errorf( "authorization: %q" , sent.Header.Get( "Authorization" ) ) }
	if sent.Header.Get( "Urgency" ) != "normal" { t.Errorf( "urgency: %q" , sent.Header.Get( "Urgency" ) ) }
	found , _ := models.ListPushSubscriptions( store , owner.ID )
	if found[ 0 ].LastSuccess == nil { t.Errorf( "the delivery was not noted" ) }
}

func TestUrgentNoticeAsksForHighUrgency( t *testing.T ) {
	service , _ , store := setup( t )
	mock , endpoint , channel := withWebPush( t , service , store )
	owner , _ := models.CreateUser( store , "Owner" , models.RoleAdmin )
	subscription , err := models.SavePushSubscription( store , owner.ID , endpoint , browserP256dh , browserAuth , "Phone" )
	if err != nil { t.Fatal( err ) }
	if err = channel.Send( context.Background() , subscription , Notice{ Title: "Due" , Urgent: true } ); err != nil { t.Fatal( err ) }
	if got := mock.requests[ 0 ].Header.Get( "Urgency" ); got != "high" { t.Errorf( "urgency: %q" , got ) }
}

func TestGoneDeviceIsRemoved( t *testing.T ) {
	for _ , status := range []int{ http.StatusGone , http.StatusNotFound } {
		service , _ , store := setup( t )
		mock , endpoint , channel := withWebPush( t , service , store )
		mock.status = status
		owner , _ := models.CreateUser( store , "Owner" , models.RoleAdmin )
		subscription , err := models.SavePushSubscription( store , owner.ID , endpoint , browserP256dh , browserAuth , "" )
		if err != nil { t.Fatal( err ) }
		if err = channel.Send( context.Background() , subscription , Notice{ Title: "t" } ); errors.Is( err , ErrRejected ) == false { t.Errorf( "%d: %v" , status , err ) }
		if found , _ := models.ListPushSubscriptions( store , owner.ID ); len( found ) != 0 { t.Errorf( "%d left the browser in place" , status ) }
	}
}

func TestWebPushRetriesAServerError( t *testing.T ) {
	service , _ , store := setup( t )
	mock , endpoint , channel := withWebPush( t , service , store )
	mock.status = http.StatusServiceUnavailable
	owner , _ := models.CreateUser( store , "Owner" , models.RoleAdmin )
	subscription , err := models.SavePushSubscription( store , owner.ID , endpoint , browserP256dh , browserAuth , "" )
	if err != nil { t.Fatal( err ) }
	err = channel.Send( context.Background() , subscription , Notice{ Title: "t" } )
	var retry *retryableError
	if errors.As( err , &retry ) == false { t.Errorf( "got %v" , err ) }
	if found , _ := models.ListPushSubscriptions( store , owner.ID ); len( found ) != 1 { t.Errorf( "a bad moment removed the browser" ) }
}

func TestRefusedEndpointIsNotSentTo( t *testing.T ) {
	service , _ , store := setup( t )
	mock , endpoint , channel := withWebPush( t , service , store )
	channel.allowed = EndpointAllowed
	owner , _ := models.CreateUser( store , "Owner" , models.RoleAdmin )
	subscription , err := models.SavePushSubscription( store , owner.ID , endpoint , browserP256dh , browserAuth , "" )
	if err != nil { t.Fatal( err ) }
	if err = channel.Send( context.Background() , subscription , Notice{ Title: "t" } ); errors.Is( err , ErrRejected ) == false { t.Errorf( "got %v" , err ) }
	if mock.count() != 0 { t.Errorf( "sent to a refused host" ) }
}

func TestChannelsPushoverWebBothNeither( t *testing.T ) {
	service , pushover , store := setup( t )
	browser , endpoint , _ := withWebPush( t , service , store )
	owner , _ := models.CreateUser( store , "Owner" , models.RoleAdmin )
	both , _ := models.CreateUser( store , "Both" , models.RoleUser )
	web , _ := models.CreateUser( store , "Web" , models.RoleUser )
	only_pushover , _ := models.CreateUser( store , "Pushover" , models.RoleUser )
	neither , _ := models.CreateUser( store , "Neither" , models.RoleUser )
	withKey( t , store , both.ID )
	withKey( t , store , only_pushover.ID )
	if _ , err := models.SavePushSubscription( store , both.ID , endpoint+"/both" , browserP256dh , browserAuth , "" ); err != nil { t.Fatal( err ) }
	if _ , err := models.SavePushSubscription( store , web.ID , endpoint+"/web" , browserP256dh , browserAuth , "" ); err != nil { t.Fatal( err ) }

	board , _ := models.CreateBoard( store , owner.ID , "Plans" , []string{ "To do" } )
	card , _ := models.CreateCard( store , board.ID , board.Lists[ 0 ].ID , "Card" , nil , owner.ID , "Owner" )
	for _ , user := range []*models.User{ both , web , only_pushover , neither } {
		models.SetMember( store , board.ID , user.ID , models.BoardRoleEditor )
	}
	assignees := []uint64{ both.ID , web.ID , only_pushover.ID , neither.ID }
	_ , change , err := models.PatchCard( store , board.ID , card.ID , &models.CardPatch{ Assignees: &assignees } , "Owner" )
	if err != nil { t.Fatal( err ) }
	service.CardChanged( board.ID , card.ID , change , owner.ID , "Owner" )
	drain( service )
	if pushover.count() != 2 { t.Errorf( "%d through Pushover" , pushover.count() ) }
	if browser.count() != 2 { t.Errorf( "%d to browsers" , browser.count() ) }
}

func TestDeliveryChoice( t *testing.T ) {
	cases := []struct {
		via      string
		pushover int
		browser  int
	}{
		{ models.NotifyViaBoth , 1 , 1 },
		{ models.NotifyViaPushover , 1 , 0 },
		{ models.NotifyViaDevices , 0 , 1 },
	}
	for _ , item := range cases {
		t.Run( item.via , func( t *testing.T ) {
			service , pushover , store := setup( t )
			browser , endpoint , _ := withWebPush( t , service , store )
			owner , _ := models.CreateUser( store , "Owner" , models.RoleAdmin )
			if _ , err := models.SavePushSubscription( store , owner.ID , endpoint , browserP256dh , browserAuth , "" ); err != nil { t.Fatal( err ) }
			prefs := models.DefaultUserPrefs()
			prefs.PushoverUserKey = userKey
			prefs.NotifyVia = item.via
			if err := models.SaveUserPrefs( store , owner.ID , prefs ); err != nil { t.Fatal( err ) }
			if err := service.Test( owner.ID , prefs ); err != nil { t.Fatal( err ) }
			if pushover.count() != item.pushover { t.Errorf( "%d through Pushover" , pushover.count() ) }
			if browser.count() != item.browser { t.Errorf( "%d to browsers" , browser.count() ) }
			saved , _ := models.GetUserPrefs( store , owner.ID )
			if saved.PushoverUserKey != userKey { t.Errorf( "the key was not kept" ) }
		} )
	}
}

// Choosing a way that is not set up must not fall back on the other one.
func TestChoiceWithNothingBehindItReachesNobody( t *testing.T ) {
	service , _ , store := setup( t )
	_ , endpoint , _ := withWebPush( t , service , store )
	owner , _ := models.CreateUser( store , "Owner" , models.RoleAdmin )
	if _ , err := models.SavePushSubscription( store , owner.ID , endpoint , browserP256dh , browserAuth , "" ); err != nil { t.Fatal( err ) }
	prefs := models.DefaultUserPrefs()
	prefs.NotifyVia = models.NotifyViaPushover
	if service.Reachable( owner.ID , prefs ) { t.Errorf( "reachable with no key" ) }
	if err := service.Test( owner.ID , prefs ); errors.Is( err , ErrNoTarget ) == false { t.Errorf( "got %v" , err ) }
}

func TestReminderReachesWebPush( t *testing.T ) {
	service , _ , store := setup( t )
	browser , endpoint , _ := withWebPush( t , service , store )
	owner , _ := models.CreateUser( store , "Owner" , models.RoleAdmin )
	if _ , err := models.SavePushSubscription( store , owner.ID , endpoint , browserP256dh , browserAuth , "" ); err != nil { t.Fatal( err ) }
	board , _ := models.CreateBoard( store , owner.ID , "Plans" , []string{ "To do" } )
	card , _ := models.CreateCard( store , board.ID , board.Lists[ 0 ].ID , "Card" , nil , owner.ID , "Owner" )
	assignees := []uint64{ owner.ID }
	due := time.Now().Add( 30 * time.Minute ).UTC().Format( time.RFC3339 )
	if _ , _ , err := models.PatchCard( store , board.ID , card.ID , &models.CardPatch{ Assignees: &assignees , DueAt: &due } , "Owner" ); err != nil { t.Fatal( err ) }
	service.checkDue()
	drain( service )
	if browser.count() != 1 { t.Errorf( "%d reminders reached the browser" , browser.count() ) }
}

func TestTestReportsNoTarget( t *testing.T ) {
	service , _ , store := setup( t )
	owner , _ := models.CreateUser( store , "Owner" , models.RoleAdmin )
	prefs , err := models.GetUserPrefs( store , owner.ID )
	if err != nil { t.Fatal( err ) }
	if err = service.Test( owner.ID , prefs ); errors.Is( err , ErrNoTarget ) == false { t.Errorf( "got %v" , err ) }
}
