// Package notify sends notifications -- through Pushover and to browsers --
// when someone is assigned to a card, when a board or card is shared with
// them, when a card they follow changes, and before (and after) a card is due.
//
// Route handlers only *publish* -- a call here never blocks a request and
// never fails one. A single worker goroutine decides who should hear about
// it (server/models.EffectiveEvents: card subscription, then board, then the
// person's defaults) and sends. Due reminders come from a ticker in
// scheduler.go. Each way of reaching someone is a Channel (channel.go), and
// everything is off when no channel is configured.
//
// The words in a notification come from language.yaml's push: section, like
// every other string a person reads.
package notify

import (
	context "context"
	errors "errors"
	fmt "fmt"
	log "log"
	strings "strings"
	sync "sync"
	time "time"

	config "todoboard/server/config"
	db "todoboard/server/db"
	language "todoboard/server/language"
	models "todoboard/server/models"
)

// How long a push service may hold a notice for a device that is offline.
// Past this, news of a routine change is not worth arriving.
const (
	importantTTL = 24 * time.Hour
	routineTTL   = 6 * time.Hour
)

type eventKind int

const (
	eventChanged eventKind = iota
	eventAssigned
	eventBoardShared
	eventCardShared
	eventActivity
)

type event struct {
	kind      eventKind
	boardID   uint64
	cardID    uint64
	actorID   uint64 // 0 for a guest
	actorName string
	change    models.CardChange
	userID    uint64 // the person a share is for
	// eventActivity: how it is worded, push.activity_<what>. Empty for a card
	// patch, which is worded from change.
	what   string
	values map[string]string
}

// pendingChange collects the edits to one card that one person will hear
// about together once the debounce window closes.
type pendingChange struct {
	boardID uint64
	cardID  uint64
	userID  uint64
	since   time.Time
	actors  []string
	lines   []string
}

type Service struct {
	store    *db.Store
	cfg      *config.Config
	lang     *language.Language
	client   *Pushover
	vapid    *models.VAPIDKeys
	lanes    []*lane
	events   chan event
	mutex    sync.Mutex
	pending  map[string]*pendingChange
	activity map[string]*pendingActivity
	now      func() time.Time
	stopOnce sync.Once
	stop     chan struct{}
}

func New( store *db.Store , cfg *config.Config , lang *language.Language ) ( service *Service ) {
	service = &Service{
		store:   store,
		cfg:     cfg,
		lang:    lang,
		events:  make( chan event , 512 ),
		pending: map[string]*pendingChange{},
		activity: map[string]*pendingActivity{},
		now:     time.Now,
		stop:    make( chan struct{} ),
	}
	if cfg.PushoverAppToken != "" {
		service.client = NewPushover( cfg.PushoverAppToken , cfg.PushoverAPIURL )
		service.lanes = append( service.lanes , newLane( &pushoverChannel{ client: service.client } ) )
	}
	if cfg.WebPush {
		keys , err := models.EnsureVAPIDKeys( store )
		if err != nil {
			log.Printf( "[notify] web push is off, no keys: %v" , err )
			return
		}
		service.vapid = keys
		service.lanes = append( service.lanes , newLane( newWebPushChannel( store , cfg , keys ) ) )
	}
	return
}

// WebPushKey is what a browser subscribes against, or "" with web push off.
func ( service *Service ) WebPushKey() ( result string ) {
	if service != nil && service.vapid != nil { result = service.vapid.Public }
	return
}

// PushoverOn reports whether this server can send through Pushover, which is
// what asking someone for a Pushover key depends on.
func ( service *Service ) PushoverOn() ( result bool ) {
	result = service != nil && service.client != nil
	return
}

// Reachable reports whether anything sent to this person would arrive
// somewhere, leaving aside whether they have paused it.
func ( service *Service ) Reachable( user_id uint64 , prefs *models.UserPrefs ) ( result bool ) {
	result = service != nil && service.reachable( user_id , prefs )
	return
}

func ( service *Service ) Enabled() ( result bool ) {
	result = service != nil && len( service.lanes ) > 0
	return
}

// Start runs the event worker, the sender, the change flusher, and the due
// scheduler. It returns at once; Stop ends them.
func ( service *Service ) Start() {
	if service.Enabled() == false { return }
	go service.eventLoop()
	names := []string{}
	for _ , item := range service.lanes {
		names = append( names , item.channel.Name() )
		go service.sendLoop( item )
	}
	go service.flushLoop()
	go service.scheduleLoop()
	log.Printf( "[notify] on through %s, checking due dates every %s" , strings.Join( names , " and " ) , service.cfg.DueCheckInterval )
}

func ( service *Service ) Stop() {
	service.stopOnce.Do( func() { close( service.stop ) } )
}

func ( service *Service ) publish( item event ) {
	if service.Enabled() == false { return }
	select {
	case service.events <- item:
	default:
		log.Printf( "[notify] event queue full, dropping a notification" )
	}
}

// CardChanged reports whatever a card patch did: assignments go to the new
// assignees, everything else to people following the card.
func ( service *Service ) CardChanged( board_id uint64 , card_id uint64 , change models.CardChange , actor_id uint64 , actor_name string ) {
	if change.Noted() == false { return }
	if len( change.NewAssignees ) > 0 {
		service.publish( event{ kind: eventAssigned , boardID: board_id , cardID: card_id , actorID: actor_id , actorName: actor_name , change: change } )
	}
	if change.Any() {
		service.publish( event{ kind: eventChanged , boardID: board_id , cardID: card_id , actorID: actor_id , actorName: actor_name , change: change } )
	}
	service.publish( event{ kind: eventActivity , boardID: board_id , actorID: actor_id , actorName: actor_name , change: change } )
}

func ( service *Service ) BoardShared( board_id uint64 , user_id uint64 , actor_id uint64 , actor_name string ) {
	service.publish( event{ kind: eventBoardShared , boardID: board_id , userID: user_id , actorID: actor_id , actorName: actor_name } )
}

func ( service *Service ) CardShared( board_id uint64 , card_id uint64 , user_id uint64 , actor_id uint64 , actor_name string ) {
	service.publish( event{ kind: eventCardShared , boardID: board_id , cardID: card_id , userID: user_id , actorID: actor_id , actorName: actor_name } )
}

var ErrDisabled = errors.New( "push notifications are not configured on this server" )
var ErrNoTarget = errors.New( "nowhere to send this person's notifications" )

// Test sends a notification everywhere the person can be reached, straight
// away, so someone setting this up finds out at once whether it works. It
// succeeds if any target took it; otherwise the error is the service's own
// verdict.
func ( service *Service ) Test( user_id uint64 , prefs *models.UserPrefs ) ( err error ) {
	if service.Enabled() == false { return ErrDisabled }
	notice := Notice{
		Title: service.text( "push.test_title" , nil ),
		Body:  service.text( "push.test_body" , nil ),
		URL:   service.cfg.PublicURL,
		Path:  "/",
		URLTitle: service.text( "push.open_app" , nil ),
	}
	sent , tried := false , false
	for _ , item := range service.lanes {
		for _ , target := range item.channel.Targets( user_id , prefs ) {
			tried = true
			ctx , cancel := context.WithTimeout( context.Background() , 15 * time.Second )
			send_err := item.channel.Send( ctx , target , notice )
			cancel()
			if send_err == nil { sent = true } else { err = send_err }
		}
	}
	if tried == false { return ErrNoTarget }
	if sent { err = nil }
	return
}

// text reads a push: string from language.yaml and fills {placeholders}.
func ( service *Service ) text( key string , values map[string]string ) ( result string ) {
	result = service.lang.Get( key )
	for name , value := range values {
		result = strings.ReplaceAll( result , "{" + name + "}" , value )
	}
	return
}

// cardPath is where a board, or a card on it, opens on this site.
func ( service *Service ) cardPath( board_id uint64 , card_id uint64 ) ( result string ) {
	result = fmt.Sprintf( "/b/%d" , board_id )
	board , get_err := models.GetBoard( service.store , board_id )
	if get_err != nil { return }
	if board.Slug != "" { result = "/b/" + board.Slug }
	if card_id == 0 { return }
	card , card_err := models.GetCard( service.store , board_id , card_id )
	if card_err != nil {
		result += fmt.Sprintf( "?card=%d" , card_id )
		return
	}
	result = models.CardPath( board , card )
	return
}

// linked fills in where a notice opens.
func ( service *Service ) linked( notice Notice , board_id uint64 , card_id uint64 ) ( result Notice ) {
	result = notice
	result.Path = service.cardPath( board_id , card_id )
	if service.cfg.PublicURL != "" { result.URL = service.cfg.PublicURL + result.Path }
	return
}

// actorLabel turns a stored "guest:Name" into the worded guest form.
func ( service *Service ) actorLabel( name string ) ( result string ) {
	if guest , ok := strings.CutPrefix( name , "guest:" ); ok {
		if guest == "" {
			result = service.text( "push.anonymous" , nil )
			return
		}
		result = service.text( "push.guest_name" , map[string]string{ "name": guest } )
		return
	}
	result = name
	return
}

func ( service *Service ) eventLoop() {
	for {
		select {
		case <-service.stop:
			return
		case item := <-service.events:
			safely( "event" , func() { service.handle( item ) } )
		}
	}
}

// recipients is everyone with an account who can see the card: the board's
// owner and members, and anyone it is shared with directly.
func recipients( board *models.Board , card *models.Card ) ( user_ids []uint64 ) {
	seen := map[uint64]bool{}
	add := func( id uint64 ) {
		if id != 0 && seen[ id ] == false {
			seen[ id ] = true
			user_ids = append( user_ids , id )
		}
	}
	add( board.OwnerID )
	for id := range board.Members { add( id ) }
	if card != nil {
		for id := range card.SharedWith { add( id ) }
	}
	return
}

func ( service *Service ) handle( item event ) {
	// First, and on its own: the card behind this one may be gone.
	if item.kind == eventActivity {
		service.handleActivity( item )
		return
	}
	board , err := models.GetBoard( service.store , item.boardID )
	if err != nil { return }
	var card *models.Card
	if item.cardID != 0 {
		if card , err = models.GetCard( service.store , item.boardID , item.cardID ); err != nil { return }
	}
	actor := service.actorLabel( item.actorName )

	switch item.kind {
	case eventAssigned:
		for _ , user_id := range item.change.NewAssignees {
			if user_id == item.actorID { continue }
			prefs := service.prefsFor( user_id )
			if prefs == nil { continue }
			if events , _ := models.EffectiveEvents( service.store , prefs , user_id , board.ID , card ); events.Assigned == false { continue }
			service.enqueue( user_id , prefs , service.linked( Notice{
				Title: service.text( "push.assigned_title" , map[string]string{ "card": card.Title } ),
				Body:  service.text( "push.assigned_body" , map[string]string{ "board": board.Title , "actor": actor } ),
				TTL:   importantTTL,
				Urgent: true,
			} , board.ID , card.ID ) )
		}

	case eventChanged:
		lines := service.changeLines( board , item.change )
		if len( lines ) == 0 { return }
		for _ , user_id := range recipients( board , card ) {
			if user_id == item.actorID { continue }
			prefs := service.prefsFor( user_id )
			if prefs == nil { continue }
			if events , _ := models.EffectiveEvents( service.store , prefs , user_id , board.ID , card ); events.Changes == false { continue }
			// This change reaches them in the board's activity already.
			if events , _ := models.EffectiveEvents( service.store , prefs , user_id , board.ID , nil ); events.Activity { continue }
			service.queueChange( board.ID , card.ID , user_id , actor , lines )
		}

	case eventBoardShared , eventCardShared:
		if item.userID == item.actorID { return }
		prefs := service.prefsFor( item.userID )
		if prefs == nil || prefs.SharedWithMe == false { return }
		notice := service.linked( Notice{
			Title: service.text( "push.board_shared_title" , map[string]string{ "board": board.Title } ),
			Body:  service.text( "push.board_shared_body" , map[string]string{ "actor": actor } ),
			TTL:   importantTTL,
		} , board.ID , 0 )
		if item.kind == eventCardShared && card != nil {
			notice = service.linked( Notice{
				Title: service.text( "push.card_shared_title" , map[string]string{ "card": card.Title } ),
				Body:  service.text( "push.card_shared_body" , map[string]string{ "actor": actor , "board": board.Title } ),
				TTL:   importantTTL,
			} , board.ID , card.ID )
		}
		service.enqueue( item.userID , prefs , notice )
	}
}

func ( service *Service ) changeLines( board *models.Board , change models.CardChange ) ( lines []string ) {
	if change.Moved {
		// The card was moved *to* a list; the event only knows where from,
		// so the line is filled in from the card's current list at flush.
		lines = append( lines , "moved" )
	}
	if change.Completed { lines = append( lines , service.text( "push.change_completed" , nil ) ) }
	if change.Reopened { lines = append( lines , service.text( "push.change_reopened" , nil ) ) }
	if change.DueChanged { lines = append( lines , service.text( "push.change_due" , nil ) ) }
	if change.Edited { lines = append( lines , service.text( "push.change_edited" , nil ) ) }
	if change.Commented { lines = append( lines , service.text( "push.change_commented" , map[string]string{ "comment": change.Comment } ) ) }
	return
}

func ( service *Service ) prefsFor( user_id uint64 ) ( prefs *models.UserPrefs ) {
	user , err := models.GetUser( service.store , user_id )
	if err != nil || user.Disabled() { return }
	loaded , err := models.GetUserPrefs( service.store , user_id )
	if err != nil || loaded.Paused || service.reachable( user_id , loaded ) == false { return }
	prefs = loaded
	return
}

func ( service *Service ) reachable( user_id uint64 , prefs *models.UserPrefs ) ( result bool ) {
	for _ , item := range service.lanes {
		if len( item.channel.Targets( user_id , prefs ) ) > 0 { return true }
	}
	return
}

// enqueue hands a notice to every place this person can be reached.
func ( service *Service ) enqueue( user_id uint64 , prefs *models.UserPrefs , notice Notice ) {
	if notice.URL != "" && notice.URLTitle == "" { notice.URLTitle = service.text( "push.open_card" , nil ) }
	for _ , item := range service.lanes {
		for _ , target := range item.channel.Targets( user_id , prefs ) {
			select {
			case item.outbox <- delivery{ target: target , notice: notice }:
			default:
				log.Printf( "[notify] %s queue full, dropping a notification" , item.channel.Name() )
			}
		}
	}
}

func pendingKey( card_id uint64 , user_id uint64 ) ( key string ) {
	key = fmt.Sprintf( "%d/%d" , card_id , user_id )
	return
}

func ( service *Service ) queueChange( board_id uint64 , card_id uint64 , user_id uint64 , actor string , lines []string ) {
	service.mutex.Lock()
	defer service.mutex.Unlock()
	key := pendingKey( card_id , user_id )
	entry := service.pending[ key ]
	if entry == nil {
		entry = &pendingChange{ boardID: board_id , cardID: card_id , userID: user_id , since: service.now() }
		service.pending[ key ] = entry
	}
	if containsString( entry.actors , actor ) == false { entry.actors = append( entry.actors , actor ) }
	for _ , line := range lines {
		if containsString( entry.lines , line ) == false { entry.lines = append( entry.lines , line ) }
	}
	if service.cfg.ChangeDebounce == 0 {
		go safely( "flush" , func() { service.flush( false ) } )
	}
}

func containsString( values []string , target string ) ( result bool ) {
	for _ , value := range values {
		if value == target { return true }
	}
	return
}

func ( service *Service ) flushLoop() {
	ticker := time.NewTicker( 10 * time.Second )
	defer ticker.Stop()
	for {
		select {
		case <-service.stop:
			return
		case <-ticker.C:
			safely( "flush" , func() { service.flush( false ) } )
		}
	}
}

// flush sends every coalesced change whose window has closed (or all of them
// when force is set). The card is re-read here, so the notification reflects
// where the card ended up rather than each step on the way.
func ( service *Service ) flush( force bool ) {
	service.mutex.Lock()
	ready := []*pendingChange{}
	for key , entry := range service.pending {
		if force || service.now().Sub( entry.since ) >= service.cfg.ChangeDebounce {
			ready = append( ready , entry )
			delete( service.pending , key )
		}
	}
	activity := []*pendingActivity{}
	for key , entry := range service.activity {
		if force || service.now().Sub( entry.since ) >= service.cfg.ChangeDebounce {
			activity = append( activity , entry )
			delete( service.activity , key )
		}
	}
	service.mutex.Unlock()

	for _ , entry := range activity { service.sendActivity( entry ) }
	for _ , entry := range ready {
		board , err := models.GetBoard( service.store , entry.boardID )
		if err != nil { continue }
		card , err := models.GetCard( service.store , entry.boardID , entry.cardID )
		if err != nil { continue }
		prefs := service.prefsFor( entry.userID )
		if prefs == nil { continue }
		lines := []string{}
		for _ , line := range entry.lines {
			if line == "moved" {
				list_title := ""
				if list := board.List( card.ListID ); list != nil { list_title = list.Title }
				line = service.text( "push.change_moved" , map[string]string{ "list": list_title } )
			}
			lines = append( lines , line )
		}
		service.enqueue( entry.userID , prefs , service.linked( Notice{
			Title: service.text( "push.changed_title" , map[string]string{ "card": card.Title } ),
			Body: service.text( "push.changed_body" , map[string]string{
				"board":   board.Title,
				"actor":   strings.Join( entry.actors , ", " ),
				"changes": strings.Join( lines , ", " ),
			} ),
			TTL: routineTTL,
		} , board.ID , card.ID ) )
	}
}

// sendLoop delivers one notice at a time, backing off on 429 and 5xx. A
// rejection (bad user key) is logged and dropped: retrying will not fix it.
func ( service *Service ) sendLoop( item *lane ) {
	for {
		select {
		case <-service.stop:
			return
		case next := <-item.outbox:
			safely( item.channel.Name() , func() { service.deliver( item , next ) } )
		}
	}
}

func ( service *Service ) deliver( item *lane , next delivery ) {
	wait := 2 * time.Second
	for attempt := 1; attempt <= 4; attempt += 1 {
		ctx , cancel := context.WithTimeout( context.Background() , 20 * time.Second )
		err := item.channel.Send( ctx , next.target , next.notice )
		cancel()
		if err == nil { return }
		if errors.Is( err , ErrRejected ) {
			log.Printf( "[notify] %v" , err )
			return
		}
		var retry *retryableError
		if errors.As( err , &retry ) && retry.after > 0 { wait = retry.after }
		log.Printf( "[notify] send failed (attempt %d): %v" , attempt , err )
		select {
		case <-service.stop:
			return
		case <-time.After( wait ):
		}
		wait *= 2
	}
}
