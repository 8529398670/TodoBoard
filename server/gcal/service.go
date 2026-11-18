// Package gcal syncs cards with dates to each person's Google Calendar, both
// ways. Each person connects their own Google account (OAuth, offline
// access) and picks the boards they sync; their dated cards on those boards
// become events on their chosen calendar.
//
// Out: a card change touches its board (Touch), and a worker brings that
// board's events in step a moment later. In: the periodic job reads what
// changed in each calendar since last time (Google's sync token) and moves
// cards whose events were moved or renamed there. Each card's link holds a
// fingerprint of what its event last said, so our own write coming back
// from Google is recognised and not applied again. Deleting an event in
// Google unlinks it; the card keeps its dates, and gets an event again only
// once it changes. Events without our private properties are never touched.
package gcal

import (
	context "context"
	errors "errors"
	fmt "fmt"
	log "log"
	http "net/http"
	url "net/url"
	sync "sync"
	time "time"

	db "todoboard/server/db"
	models "todoboard/server/models"
)

// Actor is the name a card change made from Google is recorded under.
const Actor = "Google Calendar"

// JobInterval is how often every calendar is read for changes and every
// synced board checked.
const JobInterval = 5 * time.Minute

// stateLife is how long a started connection waits for Google's answer.
const stateLife = 10 * time.Minute

// Settings is what the server is configured with: the OAuth client, and the
// public address Google sends people back to.
type Settings struct {
	ClientID     string
	ClientSecret string
	PublicURL    string
}

type cachedToken struct {
	token   string
	expires time.Time
}

type pending struct {
	user_id  uint64
	verifier string
	expires  time.Time
}

type Service struct {
	Store    *db.Store
	Settings func() Settings
	HTTP     *http.Client

	AuthURL   string
	TokenURL  string
	RevokeURL string
	APIBase   string

	mu      sync.Mutex
	tokens  map[string]cachedToken
	states  map[string]pending
	// syncing: one sync at a time, so the worker and the job never both
	// make the same card's event.
	syncing sync.Mutex
	touched chan uint64
	stop    chan struct{}
	done    chan struct{}
}

func New( store *db.Store , settings func() Settings ) ( service *Service ) {
	service = &Service{
		Store: store , Settings: settings , HTTP: &http.Client{ Timeout: 20 * time.Second } ,
		AuthURL: defaultAuthURL , TokenURL: defaultTokenURL , RevokeURL: defaultRevokeURL , APIBase: defaultAPIBase ,
		tokens: map[string]cachedToken{} , states: map[string]pending{} ,
		touched: make( chan uint64 , 256 ) ,
	}
	return
}

// Enabled: an OAuth client is configured and the server knows its public
// address, which Google sends people back to.
func ( service *Service ) Enabled() ( result bool ) {
	if service == nil || service.Settings == nil { return }
	settings := service.Settings()
	result = settings.ClientID != "" && settings.ClientSecret != "" && settings.PublicURL != ""
	return
}

// RedirectURL is where Google sends someone after they answer.
func ( service *Service ) RedirectURL() ( result string ) {
	result = service.Settings().PublicURL + "/api/google/callback"
	return
}

// Begin starts connecting this person's account: the state to hold in their
// browser and the Google address to send them to.
func ( service *Service ) Begin( user_id uint64 ) ( state string , address string ) {
	state , verifier := randomText( 32 ) , randomText( 48 )
	service.mu.Lock()
	now := time.Now()
	for key , item := range service.states {
		if now.After( item.expires ) { delete( service.states , key ) }
	}
	service.states[ state ] = pending{ user_id: user_id , verifier: verifier , expires: now.Add( stateLife ) }
	service.mu.Unlock()
	query := url.Values{
		"client_id": { service.Settings().ClientID },
		"redirect_uri": { service.RedirectURL() },
		"response_type": { "code" },
		"scope": { scopes },
		"access_type": { "offline" },
		// Always ask, so Google always hands over a refresh token.
		"prompt": { "consent" },
		"include_granted_scopes": { "true" },
		"state": { state },
		"code_challenge": { challenge( verifier ) },
		"code_challenge_method": { "S256" },
	}
	address = service.AuthURL + "?" + query.Encode()
	return
}

var ErrBadState = errors.New( "gcal: unknown or expired state" )

// Forget drops a connection under way that will not be finished.
func ( service *Service ) Forget( state string ) {
	service.mu.Lock()
	delete( service.states , state )
	service.mu.Unlock()
}

// Finish takes Google's answer: the state from Begin (spent here, whatever
// happens) and the code. It saves the connection for the person who began
// it, and answers who that was.
func ( service *Service ) Finish( ctx context.Context , state string , code string ) ( user_id uint64 , err error ) {
	service.mu.Lock()
	item , ok := service.states[ state ]
	delete( service.states , state )
	service.mu.Unlock()
	if ok == false || time.Now().After( item.expires ) || state == "" {
		err = ErrBadState
		return
	}
	user_id = item.user_id
	reply , err := service.token( ctx , url.Values{
		"grant_type": { "authorization_code" } , "code": { code } ,
		"redirect_uri": { service.RedirectURL() } , "code_verifier": { item.verifier } ,
	} )
	if err != nil { return }
	if reply.RefreshToken == "" {
		err = errors.New( "gcal: Google sent no refresh token" )
		return
	}
	service.mu.Lock()
	service.tokens[ reply.RefreshToken ] = cachedToken{ token: reply.AccessToken , expires: time.Now().Add( time.Duration( reply.ExpiresIn ) * time.Second - time.Minute ) }
	service.mu.Unlock()
	email , err := service.primaryEmail( ctx , reply.RefreshToken )
	if err != nil { return }
	// Connecting again (another Google account, or the same) starts afresh:
	// the old account's events are not this one's to edit.
	if old , _ := models.GetGoogleAccount( service.Store , user_id ); old != nil { _ = models.DeleteGoogleAccount( service.Store , user_id ) }
	err = models.SaveGoogleAccount( service.Store , user_id , &models.GoogleAccount{
		RefreshToken: reply.RefreshToken , Email: email , CalendarID: "primary" , ConnectedAt: time.Now().UTC() ,
	} )
	return
}

// Calendars is the calendars this person can put events on.
func ( service *Service ) Calendars( ctx context.Context , user_id uint64 ) ( list []Calendar , err error ) {
	account , err := models.GetGoogleAccount( service.Store , user_id )
	if err != nil || account == nil { return }
	list , err = service.calendars( ctx , account.RefreshToken )
	return
}

// SetCalendar moves this person's events to another calendar: the next sync
// takes each one off the old calendar and puts it on the new.
func ( service *Service ) SetCalendar( user_id uint64 , calendar_id string ) ( err error ) {
	err = models.UpdateGoogleAccount( service.Store , user_id , func( account *models.GoogleAccount ) {
		if account.CalendarID == calendar_id { return }
		account.CalendarID , account.SyncToken = calendar_id , ""
	} )
	if err == nil { service.TouchUser( user_id ) }
	return
}

// Disconnect forgets this person's Google account. remove_events takes the
// events this server made off their calendar first.
func ( service *Service ) Disconnect( ctx context.Context , user_id uint64 , remove_events bool ) ( err error ) {
	service.syncing.Lock()
	defer service.syncing.Unlock()
	account , err := models.GetGoogleAccount( service.Store , user_id )
	if err != nil || account == nil { return }
	if remove_events {
		boards , _ := models.GoogleSyncBoards( service.Store , user_id )
		for _ , board_id := range boards { service.removeBoardEvents( ctx , user_id , account , board_id ) }
	}
	service.revoke( ctx , account.RefreshToken )
	service.mu.Lock()
	delete( service.tokens , account.RefreshToken )
	service.mu.Unlock()
	err = models.DeleteGoogleAccount( service.Store , user_id )
	return
}

// SetBoardSync turns syncing one board on or off for this person. Off takes
// the board's events off their calendar.
func ( service *Service ) SetBoardSync( ctx context.Context , user_id uint64 , board_id uint64 , on bool ) ( err error ) {
	if err = models.SetBoardGoogleSync( service.Store , user_id , board_id , on ); err != nil { return }
	if on {
		service.Touch( board_id )
		return
	}
	account , _ := models.GetGoogleAccount( service.Store , user_id )
	if account == nil { return }
	service.syncing.Lock()
	defer service.syncing.Unlock()
	service.removeBoardEvents( ctx , user_id , account , board_id )
	return
}

func ( service *Service ) removeBoardEvents( ctx context.Context , user_id uint64 , account *models.GoogleAccount , board_id uint64 ) {
	links , err := models.GoogleLinks( service.Store , user_id , board_id )
	if err != nil { return }
	for card_id , link := range links {
		if link.EventID != "" { _ = service.deleteEvent( ctx , account.RefreshToken , link.CalendarID , link.EventID ) }
		_ = models.DeleteGoogleLink( service.Store , user_id , board_id , card_id )
	}
}

// Touch says a board's cards changed: its synced events catch up shortly.
// Never blocks; a full queue is caught by the next job run.
func ( service *Service ) Touch( board_id uint64 ) {
	if service == nil || service.touched == nil || service.Enabled() == false { return }
	select {
	case service.touched <- board_id:
	default:
	}
}

// TouchUser touches every board this person syncs.
func ( service *Service ) TouchUser( user_id uint64 ) {
	boards , _ := models.GoogleSyncBoards( service.Store , user_id )
	for _ , board_id := range boards { service.Touch( board_id ) }
}

// Start runs the worker that answers Touch.
func ( service *Service ) Start() {
	service.stop , service.done = make( chan struct{} ) , make( chan struct{} )
	go service.work()
}

func ( service *Service ) Stop() {
	if service.stop == nil { return }
	close( service.stop )
	<-service.done
}

// work gathers touched boards for a moment, so a burst of edits is one sync.
func ( service *Service ) work() {
	defer close( service.done )
	for {
		select {
		case <-service.stop:
			return
		case first := <-service.touched:
			boards := map[uint64]bool{ first: true }
			wait := time.After( 2 * time.Second )
		gather:
			for {
				select {
				case more := <-service.touched:
					boards[ more ] = true
				case <-wait:
					break gather
				case <-service.stop:
					return
				}
			}
			for board_id := range boards { service.SyncBoard( context.Background() , board_id ) }
		}
	}
}

// SyncBoard brings the board's events in step for everyone who syncs it.
func ( service *Service ) SyncBoard( ctx context.Context , board_id uint64 ) {
	board , err := models.GetBoard( service.Store , board_id )
	if err != nil { return }
	people := []uint64{ board.OwnerID }
	for user_id := range board.Members { people = append( people , user_id ) }
	for _ , user_id := range people {
		ref := models.GetBoardRef( service.Store , user_id , board_id )
		if ref == nil || ref.GoogleSync == false { continue }
		account , _ := models.GetGoogleAccount( service.Store , user_id )
		if account == nil { continue }
		service.syncing.Lock()
		err = service.pushBoard( ctx , user_id , account , board )
		service.syncing.Unlock()
		service.note( user_id , err )
	}
}

// Run is the job: every connected calendar read for changes, then every
// synced board brought in step.
func ( service *Service ) Run() ( note string , err error ) {
	if service.Enabled() == false {
		note = "google calendar is not configured"
		return
	}
	users , err := models.GoogleAccountUsers( service.Store )
	if err != nil { return }
	ctx := context.Background()
	pulled , pushed , failed := 0 , 0 , 0
	for _ , user_id := range users {
		account , _ := models.GetGoogleAccount( service.Store , user_id )
		if account == nil { continue }
		service.syncing.Lock()
		applied , sync_err := service.pull( ctx , user_id , account )
		pulled += applied
		if sync_err == nil {
			boards , _ := models.GoogleSyncBoards( service.Store , user_id )
			for _ , board_id := range boards {
				board , board_err := models.GetBoard( service.Store , board_id )
				if board_err != nil { continue }
				if sync_err = service.pushBoard( ctx , user_id , account , board ); sync_err != nil { break }
				pushed += 1
			}
		}
		service.syncing.Unlock()
		if sync_err != nil { failed += 1 }
		service.note( user_id , sync_err )
	}
	note = fmt.Sprintf( "%d calendars, %d cards moved from Google, %d boards pushed, %d failed" , len( users ) , pulled , pushed , failed )
	return
}

// note records how this person's last sync went, for their settings screen.
func ( service *Service ) note( user_id uint64 , err error ) {
	now := time.Now().UTC()
	_ = models.UpdateGoogleAccount( service.Store , user_id , func( account *models.GoogleAccount ) {
		account.LastSync = &now
		account.Problem = ""
		if errors.Is( err , ErrRevoked ) {
			account.Problem = "revoked"
		} else if err != nil {
			account.Problem = "failed"
			log.Printf( "[gcal] sync for user %d: %v" , user_id , err )
		}
	} )
}

// member: still on the board, so still allowed its cards.
func member( board *models.Board , user_id uint64 ) ( result bool ) {
	_ , listed := board.Members[ user_id ]
	result = board.OwnerID == user_id || listed
	return
}

// pushBoard makes the person's calendar match the board's cards.
func ( service *Service ) pushBoard( ctx context.Context , user_id uint64 , account *models.GoogleAccount , board *models.Board ) ( err error ) {
	links , err := models.GoogleLinks( service.Store , user_id , board.ID )
	if err != nil { return }
	cards := []*models.Card{}
	if member( board , user_id ) && board.ArchivedAt == nil {
		if cards , err = models.ListCards( service.Store , board.ID ); err != nil { return }
	}
	calendar := account.CalendarID
	public := service.Settings().PublicURL
	for _ , card := range cards {
		link := links[ card.ID ]
		delete( links , card.ID )
		want := EventFor( board , card , public + models.CardPath( board , card ) )
		if want == nil {
			if link != nil {
				if link.EventID != "" {
					if err = service.deleteEvent( ctx , account.RefreshToken , link.CalendarID , link.EventID ); err != nil { return }
				}
				if err = models.DeleteGoogleLink( service.Store , user_id , board.ID , card.ID ); err != nil { return }
			}
			continue
		}
		print := Fingerprint( want )
		if link != nil && link.Fingerprint == print && link.CalendarID == calendar { continue }
		// Deleted in Google: left off until the card changes.
		if link != nil && link.EventID == "" && link.Fingerprint == print { continue }
		if link != nil && link.EventID != "" && link.CalendarID != calendar {
			if err = service.deleteEvent( ctx , account.RefreshToken , link.CalendarID , link.EventID ); err != nil { return }
			link = nil
		}
		if link != nil && link.EventID != "" {
			patch_err := service.patchEvent( ctx , account.RefreshToken , calendar , link.EventID , want )
			if patch_err == nil {
				link.Fingerprint = print
				if err = models.PutGoogleLink( service.Store , user_id , board.ID , card.ID , link ); err != nil { return }
				continue
			}
			if errors.Is( patch_err , ErrNotFound ) == false { return patch_err }
		}
		made , insert_err := service.insertEvent( ctx , account.RefreshToken , calendar , want )
		if insert_err != nil { return insert_err }
		if err = models.PutGoogleLink( service.Store , user_id , board.ID , card.ID , &models.GoogleLink{ EventID: made.ID , CalendarID: calendar , Fingerprint: print } ); err != nil { return }
	}
	// Cards gone from the board (deleted, or the person left it).
	for card_id , link := range links {
		if link.EventID != "" {
			if err = service.deleteEvent( ctx , account.RefreshToken , link.CalendarID , link.EventID ); err != nil { return }
		}
		if err = models.DeleteGoogleLink( service.Store , user_id , board.ID , card_id ); err != nil { return }
	}
	return
}

// pull applies what changed in the person's calendar since last time to the
// cards its events stand for. applied is how many cards it changed.
func ( service *Service ) pull( ctx context.Context , user_id uint64 , account *models.GoogleAccount ) ( applied int , err error ) {
	events , next , err := service.changedEvents( ctx , account.RefreshToken , account.CalendarID , account.SyncToken )
	if errors.Is( err , ErrSyncExpired ) {
		events , next , err = service.changedEvents( ctx , account.RefreshToken , account.CalendarID , "" )
	}
	if err != nil { return }
	boards := map[uint64]*models.Board{}
	for _ , event := range events {
		board_id , card_id , ours := eventCard( event )
		if ours == false { continue }
		link , _ := models.GetGoogleLink( service.Store , user_id , board_id , card_id )
		if link == nil || link.EventID != event.ID { continue }
		if event.Status == "cancelled" {
			// Deleted in Google: unlink, and leave the card alone.
			link.EventID = ""
			_ = models.PutGoogleLink( service.Store , user_id , board_id , card_id , link )
			continue
		}
		print := Fingerprint( event )
		if print == link.Fingerprint { continue }
		board , ok := boards[ board_id ]
		if ok == false {
			board , _ = models.GetBoard( service.Store , board_id )
			boards[ board_id ] = board
		}
		if board == nil || member( board , user_id ) == false { continue }
		ref := models.GetBoardRef( service.Store , user_id , board_id )
		if ref == nil || ref.GoogleSync == false { continue }
		card , card_err := models.GetCard( service.Store , board_id , card_id )
		if card_err != nil || card.Archived { continue }
		if patch := PatchFrom( board , card , event ); patch != nil {
			if _ , _ , patch_err := models.PatchCard( service.Store , board_id , card_id , patch , Actor ); patch_err == nil { applied += 1 }
		}
		// What Google's copy says now; the push that follows sends the card's
		// own version back if the two still differ (its description, say).
		link.Fingerprint = print
		_ = models.PutGoogleLink( service.Store , user_id , board_id , card_id , link )
	}
	if next != "" || account.SyncToken != "" {
		account.SyncToken = next
		_ = models.UpdateGoogleAccount( service.Store , user_id , func( saved *models.GoogleAccount ) { saved.SyncToken = next } )
	}
	return
}
