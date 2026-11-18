package gcal

import (
	context "context"
	json "encoding/json"
	fmt "fmt"
	io "io"
	http "net/http"
	httptest "net/http/httptest"
	filepath "path/filepath"
	strings "strings"
	sync "sync"
	testing "testing"
	time "time"

	config "todoboard/server/config"
	db "todoboard/server/db"
	models "todoboard/server/models"
)

// fakeGoogle is the token endpoint and the Calendar API, holding events in
// memory the way Google would, with a change feed for sync tokens.
type fakeGoogle struct {
	mu      sync.Mutex
	events  map[string]*Event
	changed []string // event ids in change order
	next    int
	writes  int // inserts and patches: what an echo would show up as
	revoked bool
}

func ( fake *fakeGoogle ) handler( t *testing.T ) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc( "/token" , func( w http.ResponseWriter , r *http.Request ) {
		r.ParseForm()
		if fake.revoked {
			w.WriteHeader( 400 )
			fmt.Fprint( w , `{"error":"invalid_grant"}` )
			return
		}
		if r.Form.Get( "grant_type" ) == "authorization_code" && ( r.Form.Get( "code" ) != "good-code" || r.Form.Get( "code_verifier" ) == "" ) {
			w.WriteHeader( 400 )
			fmt.Fprint( w , `{"error":"invalid_grant"}` )
			return
		}
		fmt.Fprint( w , `{"access_token":"at","refresh_token":"rt","expires_in":3600}` )
	} )
	mux.HandleFunc( "/revoke" , func( w http.ResponseWriter , r *http.Request ) {} )
	mux.HandleFunc( "/api/users/me/calendarList/primary" , func( w http.ResponseWriter , r *http.Request ) {
		fmt.Fprint( w , `{"id":"person@example.com","summary":"Person","primary":true}` )
	} )
	mux.HandleFunc( "/api/calendars/primary/events" , func( w http.ResponseWriter , r *http.Request ) {
		fake.mu.Lock()
		defer fake.mu.Unlock()
		switch r.Method {
		case http.MethodPost:
			event := &Event{}
			json.NewDecoder( r.Body ).Decode( event )
			fake.next += 1
			event.ID = fmt.Sprintf( "ev%d" , fake.next )
			fake.events[ event.ID ] = event
			fake.changed = append( fake.changed , event.ID )
			fake.writes += 1
			json.NewEncoder( w ).Encode( event )
		case http.MethodGet:
			from := 0
			fmt.Sscanf( r.URL.Query().Get( "syncToken" ) , "%d" , &from )
			page := eventPage{ NextSyncToken: fmt.Sprintf( "%d" , len( fake.changed ) ) }
			seen := map[string]bool{}
			for _ , id := range fake.changed[ from: ] {
				if seen[ id ] { continue }
				seen[ id ] = true
				page.Items = append( page.Items , fake.events[ id ] )
			}
			json.NewEncoder( w ).Encode( page )
		}
	} )
	mux.HandleFunc( "/api/calendars/primary/events/" , func( w http.ResponseWriter , r *http.Request ) {
		fake.mu.Lock()
		defer fake.mu.Unlock()
		id := strings.TrimPrefix( r.URL.Path , "/api/calendars/primary/events/" )
		event , ok := fake.events[ id ]
		if ok == false || event.Status == "cancelled" {
			w.WriteHeader( 404 )
			return
		}
		switch r.Method {
		case http.MethodPatch:
			raw , _ := io.ReadAll( r.Body )
			json.Unmarshal( raw , event )
			fake.changed = append( fake.changed , id )
			fake.writes += 1
		case http.MethodDelete:
			event.Status = "cancelled"
			fake.changed = append( fake.changed , id )
		}
	} )
	return mux
}

// edit is someone changing an event in Google.
func ( fake *fakeGoogle ) edit( id string , change func( event *Event ) ) {
	fake.mu.Lock()
	defer fake.mu.Unlock()
	change( fake.events[ id ] )
	fake.changed = append( fake.changed , id )
}

func ( fake *fakeGoogle ) only( t *testing.T ) ( event *Event ) {
	t.Helper()
	fake.mu.Lock()
	defer fake.mu.Unlock()
	for _ , item := range fake.events {
		if item.Status == "cancelled" { continue }
		if event != nil { t.Fatalf( "more than one live event" ) }
		event = item
	}
	return
}

type setup struct {
	store   *db.Store
	service *Service
	fake    *fakeGoogle
	board   *models.Board
	list    uint64
	owner   *models.User
}

func newSetup( t *testing.T ) ( s *setup ) {
	t.Helper()
	cfg := &config.Config{ DatabasePath: filepath.Join( t.TempDir() , "test.db" ) }
	store , err := db.Open( cfg )
	if err != nil { t.Fatal( err ) }
	t.Cleanup( func() { store.Close() } )
	fake := &fakeGoogle{ events: map[string]*Event{} }
	server := httptest.NewServer( fake.handler( t ) )
	t.Cleanup( server.Close )
	service := New( store , func() Settings { return Settings{ ClientID: "id" , ClientSecret: "secret" , PublicURL: "https://todo.test" } } )
	service.TokenURL , service.RevokeURL , service.APIBase , service.AuthURL = server.URL + "/token" , server.URL + "/revoke" , server.URL + "/api" , server.URL + "/auth"
	owner , _ := models.CreateUser( store , "Owner" , models.RoleAdmin )
	board , err := models.CreateBoard( store , owner.ID , "Plans" , []string{ "Work" } )
	if err != nil { t.Fatal( err ) }
	s = &setup{ store: store , service: service , fake: fake , board: board , list: board.SortedLists()[ 0 ].ID , owner: owner }
	return
}

func ( s *setup ) connect( t *testing.T ) {
	t.Helper()
	state , address := s.service.Begin( s.owner.ID )
	if strings.Contains( address , "code_challenge=" ) == false || strings.Contains( address , "access_type=offline" ) == false {
		t.Fatalf( "auth address: %s" , address )
	}
	user_id , err := s.service.Finish( context.Background() , state , "good-code" )
	if err != nil || user_id != s.owner.ID { t.Fatalf( "finish: %d %v" , user_id , err ) }
	if err = s.service.SetBoardSync( context.Background() , s.owner.ID , s.board.ID , true ); err != nil { t.Fatal( err ) }
}

func ( s *setup ) card( t *testing.T , title string , due string ) ( card *models.Card ) {
	t.Helper()
	card , err := models.CreateCard( s.store , s.board.ID , s.list , title , nil , s.owner.ID , "Owner" )
	if err != nil { t.Fatal( err ) }
	if due != "" {
		if card , _ , err = models.PatchCard( s.store , s.board.ID , card.ID , &models.CardPatch{ DueAt: &due } , "Owner" ); err != nil { t.Fatal( err ) }
	}
	return
}

func TestEventForAndFingerprint( t *testing.T ) {
	board := &models.Board{ ID: 7 , Layout: models.LayoutCalendar }
	due := time.Date( 2026 , 11 , 20 , 15 , 0 , 0 , 0 , time.UTC )
	card := &models.Card{ ID: 3 , Title: "Call" , DueAt: &due }
	event := EventFor( board , card , "https://todo.test/b/plans" )
	if *event.Start.DateTime != "2026-11-20T14:30:00Z" || *event.End.DateTime != "2026-11-20T15:00:00Z" {
		t.Errorf( "a due time with no start is the half hour before: %v %v" , *event.Start.DateTime , *event.End.DateTime )
	}
	if strings.HasSuffix( event.Description , "https://todo.test/b/plans" ) == false { t.Errorf( "description: %q" , event.Description ) }
	// Google writes times back in the calendar's zone: the same instant is
	// the same fingerprint, so our own write is not taken for an edit.
	google := *event
	start , end := "2026-11-20T06:30:00-08:00" , "2026-11-20T07:00:00-08:00"
	google.Start , google.End = dateTime( start ) , dateTime( end )
	if Fingerprint( &google ) != Fingerprint( event ) { t.Error( "the same times in another zone should match" ) }

	card.DueAllDay = true
	all_day := EventFor( board , card , "" )
	if *all_day.Start.Date != "2026-11-20" || *all_day.End.Date != "2026-11-21" { t.Errorf( "all day: %v %v" , *all_day.Start.Date , *all_day.End.Date ) }

	filed := &models.Card{ ID: 4 , Title: "Filed" , Day: "2026-11-22" }
	if day := EventFor( board , filed , "" ); day == nil || *day.Start.Date != "2026-11-22" { t.Errorf( "a calendar board's day is an all-day event: %+v" , day ) }
	if EventFor( &models.Board{ ID: 1 } , filed , "" ) != nil { t.Error( "a day means nothing off a calendar board" ) }
	card.Archived = true
	if EventFor( board , card , "" ) != nil { t.Error( "an archived card has no event" ) }
	done := &models.Card{ ID: 5 , Title: "Shipped" , DueAt: &due , Done: true }
	if EventFor( board , done , "" ).Summary != DoneMark + "Shipped" { t.Error( "a finished card is marked" ) }
}

func TestPatchFrom( t *testing.T ) {
	board := &models.Board{ ID: 7 , Layout: models.LayoutCalendar }
	due := time.Date( 2026 , 11 , 20 , 15 , 0 , 0 , 0 , time.UTC )
	card := &models.Card{ ID: 3 , Title: "Call" , DueAt: &due }
	event := EventFor( board , card , "" )
	if PatchFrom( board , card , event ) != nil { t.Error( "an unchanged event changes nothing" ) }

	moved := *event
	moved.Start , moved.End = dateTime( "2026-11-21T09:30:00Z" ) , dateTime( "2026-11-21T10:00:00Z" )
	patch := PatchFrom( board , card , &moved )
	if patch == nil || patch.DueAt == nil || *patch.DueAt != "2026-11-21T10:00:00Z" || patch.StartAt != nil {
		t.Errorf( "moving a half-hour event moves the due time only: %+v" , patch )
	}
	longer := *event
	longer.Start = dateTime( "2026-11-20T13:00:00Z" )
	if patch = PatchFrom( board , card , &longer ); patch == nil || patch.StartAt == nil || *patch.StartAt != "2026-11-20T13:00:00Z" {
		t.Errorf( "stretching it gives the card a start: %+v" , patch )
	}
	renamed := *event
	renamed.Summary = DoneMark + "Call Sam"
	if patch = PatchFrom( board , card , &renamed ); patch == nil || patch.Title == nil || *patch.Title != "Call Sam" {
		t.Errorf( "renamed: %+v" , patch )
	}
	filed := &models.Card{ ID: 4 , Title: "Filed" , Day: "2026-11-22" }
	day := EventFor( board , filed , "" )
	day.Start , day.End = date( "2026-11-24" ) , date( "2026-11-25" )
	if patch = PatchFrom( board , filed , day ); patch == nil || patch.Day == nil || *patch.Day != "2026-11-24" || patch.DueAt != nil {
		t.Errorf( "a filed card moves to the new day: %+v" , patch )
	}
	day.End = date( "2026-11-27" )
	if patch = PatchFrom( board , filed , day ); patch == nil || patch.DueAt == nil || *patch.DueAt != "2026-11-26" || patch.StartAt == nil || *patch.StartAt != "2026-11-24" {
		t.Errorf( "spread over days it becomes a start and due date: %+v" , patch )
	}
}

func TestStateIsSpentAndChecked( t *testing.T ) {
	s := newSetup( t )
	if _ , err := s.service.Finish( context.Background() , "made-up" , "good-code" ); err != ErrBadState { t.Errorf( "unknown state: %v" , err ) }
	state , _ := s.service.Begin( s.owner.ID )
	if _ , err := s.service.Finish( context.Background() , state , "bad-code" ); err == nil { t.Error( "a bad code should fail" ) }
	if _ , err := s.service.Finish( context.Background() , state , "good-code" ); err != ErrBadState { t.Errorf( "a state works once: %v" , err ) }
}

func TestSyncBothWays( t *testing.T ) {
	s := newSetup( t )
	s.connect( t )
	account , _ := models.GetGoogleAccount( s.store , s.owner.ID )
	if account == nil || account.Email != "person@example.com" || account.CalendarID != "primary" { t.Fatalf( "account: %+v" , account ) }
	card := s.card( t , "Dentist" , "2026-11-20T15:00:00Z" )
	s.card( t , "Someday" , "" )

	ctx := context.Background()
	s.service.SyncBoard( ctx , s.board.ID )
	event := s.fake.only( t )
	if event == nil || event.Summary != "Dentist" { t.Fatalf( "event: %+v" , event ) }
	writes := s.fake.writes

	// Nothing changed: a job run writes nothing, even with our insert in
	// Google's change feed.
	if _ , err := s.service.Run(); err != nil { t.Fatal( err ) }
	if s.fake.writes != writes { t.Errorf( "an unchanged card was written again: %d writes" , s.fake.writes - writes ) }

	// Moved in Google: the card follows, and nothing is written back.
	s.fake.edit( event.ID , func( event *Event ) { event.Start , event.End = dateTime( "2026-11-21T08:30:00-08:00" ) , dateTime( "2026-11-21T09:00:00-08:00" ) } )
	if _ , err := s.service.Run(); err != nil { t.Fatal( err ) }
	moved , _ := models.GetCard( s.store , s.board.ID , card.ID )
	if moved.DueAt == nil || moved.DueAt.UTC().Format( time.RFC3339 ) != "2026-11-21T17:00:00Z" { t.Errorf( "card due: %v" , moved.DueAt ) }
	if moved.UpdatedBy != Actor { t.Errorf( "changed by %q" , moved.UpdatedBy ) }
	if s.fake.writes != writes { t.Errorf( "a change from Google was echoed back: %d writes" , s.fake.writes - writes ) }

	// Changed here: the event follows.
	title := "Dentist (moved)"
	if _ , _ , err := models.PatchCard( s.store , s.board.ID , card.ID , &models.CardPatch{ Title: &title } , "Owner" ); err != nil { t.Fatal( err ) }
	s.service.SyncBoard( ctx , s.board.ID )
	if got := s.fake.only( t ).Summary; got != title { t.Errorf( "event title: %q" , got ) }

	// Deleted in Google: unlinked, the card keeps its date, and no new event
	// until the card changes.
	s.fake.edit( event.ID , func( event *Event ) { event.Status = "cancelled" } )
	if _ , err := s.service.Run(); err != nil { t.Fatal( err ) }
	if s.fake.only( t ) != nil { t.Error( "an event deleted in Google came back" ) }
	kept , _ := models.GetCard( s.store , s.board.ID , card.ID )
	if kept.DueAt == nil { t.Error( "deleting the event cleared the card's date" ) }
	later := "2026-11-25"
	models.PatchCard( s.store , s.board.ID , card.ID , &models.CardPatch{ DueAt: &later } , "Owner" )
	s.service.SyncBoard( ctx , s.board.ID )
	again := s.fake.only( t )
	if again == nil || again.Start.Date == nil || *again.Start.Date != "2026-11-25" { t.Fatalf( "a changed card gets an event again: %+v" , again ) }

	// Deleted here: the event goes.
	if err := models.TrashCard( s.store , s.board.ID , card.ID , "Owner" ); err != nil { t.Fatal( err ) }
	s.service.SyncBoard( ctx , s.board.ID )
	if s.fake.only( t ) != nil { t.Error( "a deleted card's event stayed" ) }
}

func TestStopSyncAndDisconnect( t *testing.T ) {
	s := newSetup( t )
	s.connect( t )
	s.card( t , "Review" , "2026-11-20" )
	ctx := context.Background()
	s.service.SyncBoard( ctx , s.board.ID )
	if s.fake.only( t ) == nil { t.Fatal( "no event" ) }
	if err := s.service.SetBoardSync( ctx , s.owner.ID , s.board.ID , false ); err != nil { t.Fatal( err ) }
	if s.fake.only( t ) != nil { t.Error( "stopping a board's sync takes its events off" ) }

	s.service.SetBoardSync( ctx , s.owner.ID , s.board.ID , true )
	s.service.SyncBoard( ctx , s.board.ID )
	if err := s.service.Disconnect( ctx , s.owner.ID , false ); err != nil { t.Fatal( err ) }
	if s.fake.only( t ) == nil { t.Error( "disconnecting without remove_events leaves events" ) }
	if account , _ := models.GetGoogleAccount( s.store , s.owner.ID ); account != nil { t.Error( "account kept" ) }
	if boards , _ := models.GoogleSyncBoards( s.store , s.owner.ID ); len( boards ) != 0 { t.Errorf( "boards still synced: %v" , boards ) }
}

func TestRevokedAccessIsReported( t *testing.T ) {
	s := newSetup( t )
	s.connect( t )
	s.fake.revoked = true
	s.service.mu.Lock()
	s.service.tokens = map[string]cachedToken{}
	s.service.mu.Unlock()
	s.service.Run()
	account , _ := models.GetGoogleAccount( s.store , s.owner.ID )
	if account.Problem != "revoked" { t.Errorf( "problem: %q" , account.Problem ) }
}
