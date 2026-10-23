package notify

import (
	context "context"
	errors "errors"
	http "net/http"
	httptest "net/http/httptest"
	filepath "path/filepath"
	os "os"
	sync "sync"
	testing "testing"
	time "time"

	config "todoboard/server/config"
	db "todoboard/server/db"
	language "todoboard/server/language"
	models "todoboard/server/models"
)

// mockPushover records every message it is sent.
type mockPushover struct {
	mutex    sync.Mutex
	messages []map[string]string
	status   int
}

func ( mock *mockPushover ) handler( writer http.ResponseWriter , request *http.Request ) {
	request.ParseForm()
	mock.mutex.Lock()
	defer mock.mutex.Unlock()
	if mock.status != 0 {
		writer.WriteHeader( mock.status )
		writer.Write( []byte( `{"status":0,"errors":["user identifier is invalid"]}` ) )
		return
	}
	entry := map[string]string{}
	for key := range request.PostForm { entry[ key ] = request.PostForm.Get( key ) }
	mock.messages = append( mock.messages , entry )
	writer.Write( []byte( `{"status":1}` ) )
}

func ( mock *mockPushover ) count() ( result int ) {
	mock.mutex.Lock()
	defer mock.mutex.Unlock()
	result = len( mock.messages )
	return
}

func ( mock *mockPushover ) last() ( result map[string]string ) {
	mock.mutex.Lock()
	defer mock.mutex.Unlock()
	if len( mock.messages ) > 0 { result = mock.messages[ len( mock.messages ) - 1 ] }
	return
}

const userKey = "uQiRzpo4DXghDmr9QzzfQu27cmVRsG"

func setup( t *testing.T ) ( service *Service , mock *mockPushover , store *db.Store ) {
	t.Helper()
	mock = &mockPushover{}
	server := httptest.NewServer( http.HandlerFunc( mock.handler ) )
	t.Cleanup( server.Close )

	cfg := &config.Config{
		DatabasePath:     filepath.Join( t.TempDir() , "test.db" ),
		PushoverAppToken: "azGDORePK8gMaC0QOYAMyEEuzJnyUi",
		PushoverAPIURL:   server.URL,
		PublicURL:        "https://boards.example.com",
		DueCheckInterval: time.Hour,
		ChangeDebounce:   time.Minute,
	}
	store , err := db.Open( cfg )
	if err != nil { t.Fatal( err ) }
	t.Cleanup( func() { store.Close() } )

	raw , err := os.ReadFile( "../../language.yaml" )
	if err != nil { t.Fatal( err ) }
	lang , err := language.Parse( raw )
	if err != nil { t.Fatal( err ) }

	service = New( store , cfg , lang )
	return
}

// drain processes queued events and messages synchronously, so tests do not
// race the goroutines Start would launch.
func drain( service *Service ) {
	for {
		select {
		case item := <-service.events:
			service.handle( item )
			continue
		default:
		}
		select {
		case message := <-service.outbox:
			service.deliver( message )
			continue
		default:
		}
		return
	}
}

func withKey( t *testing.T , store *db.Store , user_id uint64 ) {
	prefs := models.DefaultUserPrefs()
	prefs.PushoverUserKey = userKey
	prefs.Timezone = "Europe/London"
	if err := models.SaveUserPrefs( store , user_id , prefs ); err != nil { t.Fatal( err ) }
}

func TestPushoverClient( t *testing.T ) {
	service , mock , _ := setup( t )
	err := service.client.Send( context.Background() , Message{ User: userKey , Title: "t" , Body: "b" , URL: "https://x" , URLTitle: "open" } )
	if err != nil { t.Fatal( err ) }
	if got := mock.last(); got[ "token" ] == "" || got[ "user" ] != userKey || got[ "url" ] != "https://x" {
		t.Errorf( "unexpected form: %v" , got )
	}
	mock.status = 400
	err = service.client.Send( context.Background() , Message{ User: userKey , Title: "t" , Body: "b" } )
	if errors.Is( err , ErrRejected ) == false {
		t.Errorf( "a 400 should be ErrRejected, got %v" , err )
	}
}

func TestAssignAndChangeNotifications( t *testing.T ) {
	service , mock , store := setup( t )
	owner , _ := models.CreateUser( store , "Owner" , models.RoleAdmin )
	bob , _ := models.CreateUser( store , "Bob" , models.RoleUser )
	withKey( t , store , owner.ID )
	withKey( t , store , bob.ID )

	board , _ := models.CreateBoard( store , owner.ID , "Lab" , []string{ "To do" , "Done" } )
	models.SetMember( store , board.ID , bob.ID , models.BoardRoleEditor )
	card , _ := models.CreateCard( store , board.ID , board.Lists[ 0 ].ID , "Titrate" , nil , owner.ID , "Owner" )

	// Owner assigns Bob: Bob hears, the owner (the actor) does not.
	assignees := []uint64{ bob.ID }
	_ , change , _ := models.PatchCard( store , board.ID , card.ID , &models.CardPatch{ Assignees: &assignees } , "Owner" )
	service.CardChanged( board.ID , card.ID , change , owner.ID , "Owner" )
	drain( service )
	if mock.count() != 1 || mock.last()[ "title" ] != "Assigned to you: Titrate" {
		t.Fatalf( "expected one assignment push, got %d: %v" , mock.count() , mock.last() )
	}
	if mock.last()[ "url" ] != "https://boards.example.com/b/lab?card=1" {
		t.Errorf( "deep link wrong: %v" , mock.last()[ "url" ] )
	}

	// Two quick edits by a guest coalesce into one push for Bob (assignee,
	// so following by default) and none for the owner (watch defaults).
	done := true
	list := board.Lists[ 1 ].ID
	_ , change , _ = models.PatchCard( store , board.ID , card.ID , &models.CardPatch{ ListID: &list } , "guest:Sam" )
	service.CardChanged( board.ID , card.ID , change , 0 , "guest:Sam" )
	_ , change , _ = models.PatchCard( store , board.ID , card.ID , &models.CardPatch{ Done: &done } , "guest:Sam" )
	service.CardChanged( board.ID , card.ID , change , 0 , "guest:Sam" )
	drain( service )
	if mock.count() != 1 {
		t.Fatalf( "changes must wait for the debounce window, got %d pushes" , mock.count() )
	}
	service.flush( true )
	drain( service )
	if mock.count() != 2 {
		t.Fatalf( "expected one coalesced change push, got %d total" , mock.count() )
	}
	if body := mock.last()[ "message" ]; body != "Sam (guest) on Lab: moved to Done, marked done" {
		t.Errorf( "coalesced body: %q" , body )
	}

	// The owner subscribes to changes on this card and now hears too.
	models.SetCardSubscription( store , owner.ID , board.ID , card.ID , &models.EventSet{ Changes: true } )
	not_done := false
	_ , change , _ = models.PatchCard( store , board.ID , card.ID , &models.CardPatch{ Done: &not_done } , "Bob" )
	service.CardChanged( board.ID , card.ID , change , bob.ID , "Bob" )
	drain( service )
	service.flush( true )
	drain( service )
	if mock.count() != 3 || mock.last()[ "message" ] != "Bob on Lab: reopened" {
		t.Errorf( "owner's card subscription should deliver: %d %v" , mock.count() , mock.last() )
	}
}

func TestDueReminders( t *testing.T ) {
	service , mock , store := setup( t )
	owner , _ := models.CreateUser( store , "Owner" , models.RoleAdmin )
	withKey( t , store , owner.ID )
	board , _ := models.CreateBoard( store , owner.ID , "Lab" , []string{ "To do" } )
	card , _ := models.CreateCard( store , board.ID , board.Lists[ 0 ].ID , "Report" , nil , owner.ID , "Owner" )

	due := time.Date( 2026 , 11 , 2 , 15 , 0 , 0 , 0 , time.UTC )
	due_text := due.Format( time.RFC3339 )
	assignees := []uint64{ owner.ID }
	models.PatchCard( store , board.ID , card.ID , &models.CardPatch{ DueAt: &due_text , Assignees: &assignees } , "Owner" )

	clock := due.Add( -25 * time.Hour )
	service.now = func() time.Time { return clock }

	service.checkDue()
	drain( service )
	if mock.count() != 0 {
		t.Fatalf( "nothing is due 25h out, got %d" , mock.count() )
	}

	clock = due.Add( -23 * time.Hour )
	service.checkDue()
	service.checkDue() // the ledger stops a second send
	drain( service )
	if mock.count() != 1 || mock.last()[ "title" ] != "Due in 1 day: Report" {
		t.Fatalf( "expected the 1-day reminder once, got %d %v" , mock.count() , mock.last() )
	}

	clock = due.Add( -30 * time.Minute )
	service.checkDue()
	drain( service )
	if mock.count() != 2 || mock.last()[ "title" ] != "Due in 1 hour: Report" {
		t.Fatalf( "expected the 1-hour reminder, got %d %v" , mock.count() , mock.last() )
	}

	clock = due.Add( 5 * time.Minute )
	service.checkDue()
	drain( service )
	if mock.count() != 3 || mock.last()[ "title" ] != "Overdue: Report" {
		t.Fatalf( "expected overdue, got %d %v" , mock.count() , mock.last() )
	}

	// Moving the due date re-arms the reminders.
	later := due.Add( 48 * time.Hour ).Format( time.RFC3339 )
	models.PatchCard( store , board.ID , card.ID , &models.CardPatch{ DueAt: &later } , "Owner" )
	clock = due.Add( 47 * time.Hour + 30 * time.Minute )
	service.checkDue()
	drain( service )
	if mock.count() != 4 || mock.last()[ "title" ] != "Due in 1 hour: Report" {
		t.Fatalf( "a moved due date should remind again, got %d %v" , mock.count() , mock.last() )
	}

	// Done cards are left alone.
	done := true
	models.PatchCard( store , board.ID , card.ID , &models.CardPatch{ Done: &done } , "Owner" )
	clock = due.Add( 48 * time.Hour + 5 * time.Minute )
	service.checkDue()
	drain( service )
	if mock.count() != 4 {
		t.Errorf( "a done card must not go overdue, got %d" , mock.count() )
	}
}

func TestAllDayDueUsesRecipientTimezone( t *testing.T ) {
	prefs := models.DefaultUserPrefs()
	prefs.Timezone = "America/New_York"
	prefs.AllDayReminderHour = 9
	day := time.Date( 2026 , 11 , 2 , 0 , 0 , 0 , 0 , time.UTC )
	card := &models.Card{ DueAt: &day , DueAllDay: true }
	due , ok := DueMoment( card , prefs )
	if ok == false || due.UTC().Hour() != 14 || due.Day() != 2 {
		t.Errorf( "all-day due should be 09:00 New York (14:00 UTC), got %v" , due.UTC() )
	}
}

func TestSharedNotificationRespectsPreference( t *testing.T ) {
	service , mock , store := setup( t )
	owner , _ := models.CreateUser( store , "Owner" , models.RoleAdmin )
	bob , _ := models.CreateUser( store , "Bob" , models.RoleUser )
	withKey( t , store , bob.ID )
	board , _ := models.CreateBoard( store , owner.ID , "Lab" , []string{ "L" } )

	service.BoardShared( board.ID , bob.ID , owner.ID , "Owner" )
	drain( service )
	if mock.count() != 1 || mock.last()[ "title" ] != "Board shared with you: Lab" {
		t.Fatalf( "expected a share push, got %d %v" , mock.count() , mock.last() )
	}

	prefs , _ := models.GetUserPrefs( store , bob.ID )
	prefs.SharedWithMe = false
	models.SaveUserPrefs( store , bob.ID , prefs )
	service.BoardShared( board.ID , bob.ID , owner.ID , "Owner" )
	drain( service )
	if mock.count() != 1 {
		t.Errorf( "share pushes were switched off, got %d" , mock.count() )
	}
}
