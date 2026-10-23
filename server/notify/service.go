// Package notify sends Pushover notifications: when someone is assigned to a
// card, when a board or card is shared with them, when a card they follow
// changes, and before (and after) a card is due.
//
// Route handlers only *publish* -- a call here never blocks a request and
// never fails one. A single worker goroutine decides who should hear about
// it (server/models.EffectiveEvents: card subscription, then board, then the
// person's defaults) and sends. Due reminders come from a ticker in
// scheduler.go. Everything is off when pushover.app_token is empty.
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

type eventKind int

const (
	eventChanged eventKind = iota
	eventAssigned
	eventBoardShared
	eventCardShared
)

type event struct {
	kind      eventKind
	boardID   uint64
	cardID    uint64
	actorID   uint64 // 0 for a guest
	actorName string
	change    models.CardChange
	userID    uint64 // the person a share is for
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
	events   chan event
	outbox   chan Message
	mutex    sync.Mutex
	pending  map[string]*pendingChange
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
		outbox:  make( chan Message , 512 ),
		pending: map[string]*pendingChange{},
		now:     time.Now,
		stop:    make( chan struct{} ),
	}
	if cfg.PushoverAppToken != "" {
		service.client = NewPushover( cfg.PushoverAppToken , cfg.PushoverAPIURL )
	}
	return
}

func ( service *Service ) Enabled() ( result bool ) {
	result = service != nil && service.client != nil
	return
}

// Start runs the event worker, the sender, the change flusher, and the due
// scheduler. It returns at once; Stop ends them.
func ( service *Service ) Start() {
	if service.Enabled() == false { return }
	go service.eventLoop()
	go service.sendLoop()
	go service.flushLoop()
	go service.scheduleLoop()
	log.Printf( "[notify] pushover on, checking due dates every %s" , service.cfg.DueCheckInterval )
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
	if change.Any() == false { return }
	if len( change.NewAssignees ) > 0 {
		service.publish( event{ kind: eventAssigned , boardID: board_id , cardID: card_id , actorID: actor_id , actorName: actor_name , change: change } )
	}
	service.publish( event{ kind: eventChanged , boardID: board_id , cardID: card_id , actorID: actor_id , actorName: actor_name , change: change } )
}

func ( service *Service ) BoardShared( board_id uint64 , user_id uint64 , actor_id uint64 , actor_name string ) {
	service.publish( event{ kind: eventBoardShared , boardID: board_id , userID: user_id , actorID: actor_id , actorName: actor_name } )
}

func ( service *Service ) CardShared( board_id uint64 , card_id uint64 , user_id uint64 , actor_id uint64 , actor_name string ) {
	service.publish( event{ kind: eventCardShared , boardID: board_id , cardID: card_id , userID: user_id , actorID: actor_id , actorName: actor_name } )
}

var ErrDisabled = errors.New( "push notifications are not configured on this server" )
var ErrNoUserKey = errors.New( "no pushover user key saved" )

// Test sends a notification straight away and returns Pushover's verdict,
// so a person pasting their key finds out immediately whether it works.
func ( service *Service ) Test( prefs *models.UserPrefs ) ( err error ) {
	if service.Enabled() == false { return ErrDisabled }
	if prefs.PushoverUserKey == "" { return ErrNoUserKey }
	ctx , cancel := context.WithTimeout( context.Background() , 15 * time.Second )
	defer cancel()
	err = service.client.Send( ctx , Message{
		User:  prefs.PushoverUserKey,
		Device: prefs.PushoverDevice,
		Title: service.text( "push.test_title" , nil ),
		Body:  service.text( "push.test_body" , nil ),
		URL:   service.cfg.PublicURL,
		URLTitle: service.text( "push.open_app" , nil ),
	} )
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

func ( service *Service ) cardURL( board_id uint64 , card_id uint64 ) ( result string ) {
	if service.cfg.PublicURL == "" { return }
	result = fmt.Sprintf( "%s/b/%d" , service.cfg.PublicURL , board_id )
	if board , get_err := models.GetBoard( service.store , board_id ); get_err == nil && board.Slug != "" {
		result = service.cfg.PublicURL + "/b/" + board.Slug
	}
	if card_id != 0 { result += fmt.Sprintf( "?card=%d" , card_id ) }
	return
}

// actorLabel turns a stored "guest:Name" into the worded guest form.
func ( service *Service ) actorLabel( name string ) ( result string ) {
	if guest , ok := strings.CutPrefix( name , "guest:" ); ok {
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
			service.handle( item )
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
			service.enqueue( prefs , Message{
				Title: service.text( "push.assigned_title" , map[string]string{ "card": card.Title } ),
				Body:  service.text( "push.assigned_body" , map[string]string{ "board": board.Title , "actor": actor } ),
				URL:   service.cardURL( board.ID , card.ID ),
			} )
		}

	case eventChanged:
		lines := service.changeLines( board , item.change )
		if len( lines ) == 0 { return }
		for _ , user_id := range recipients( board , card ) {
			if user_id == item.actorID { continue }
			prefs := service.prefsFor( user_id )
			if prefs == nil { continue }
			if events , _ := models.EffectiveEvents( service.store , prefs , user_id , board.ID , card ); events.Changes == false { continue }
			service.queueChange( board.ID , card.ID , user_id , actor , lines )
		}

	case eventBoardShared , eventCardShared:
		if item.userID == item.actorID { return }
		prefs := service.prefsFor( item.userID )
		if prefs == nil || prefs.SharedWithMe == false { return }
		message := Message{
			Title: service.text( "push.board_shared_title" , map[string]string{ "board": board.Title } ),
			Body:  service.text( "push.board_shared_body" , map[string]string{ "actor": actor } ),
			URL:   service.cardURL( board.ID , 0 ),
		}
		if item.kind == eventCardShared && card != nil {
			message.Title = service.text( "push.card_shared_title" , map[string]string{ "card": card.Title } )
			message.Body = service.text( "push.card_shared_body" , map[string]string{ "actor": actor , "board": board.Title } )
			message.URL = service.cardURL( board.ID , card.ID )
		}
		service.enqueue( prefs , message )
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
	return
}

func ( service *Service ) prefsFor( user_id uint64 ) ( prefs *models.UserPrefs ) {
	user , err := models.GetUser( service.store , user_id )
	if err != nil || user.Disabled() { return }
	loaded , err := models.GetUserPrefs( service.store , user_id )
	if err != nil || loaded.CanPush() == false { return }
	prefs = loaded
	return
}

func ( service *Service ) enqueue( prefs *models.UserPrefs , message Message ) {
	message.User = prefs.PushoverUserKey
	message.Device = prefs.PushoverDevice
	if message.URL != "" && message.URLTitle == "" { message.URLTitle = service.text( "push.open_card" , nil ) }
	select {
	case service.outbox <- message:
	default:
		log.Printf( "[notify] send queue full, dropping a notification" )
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
		go service.flush( false )
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
			service.flush( false )
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
	service.mutex.Unlock()

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
		service.enqueue( prefs , Message{
			Title: service.text( "push.changed_title" , map[string]string{ "card": card.Title } ),
			Body: service.text( "push.changed_body" , map[string]string{
				"board":   board.Title,
				"actor":   strings.Join( entry.actors , ", " ),
				"changes": strings.Join( lines , ", " ),
			} ),
			URL: service.cardURL( board.ID , card.ID ),
		} )
	}
}

// sendLoop delivers one message at a time, backing off on 429 and 5xx. A
// rejection (bad user key) is logged and dropped: retrying will not fix it.
func ( service *Service ) sendLoop() {
	for {
		select {
		case <-service.stop:
			return
		case message := <-service.outbox:
			service.deliver( message )
		}
	}
}

func ( service *Service ) deliver( message Message ) {
	wait := 2 * time.Second
	for attempt := 1; attempt <= 4; attempt += 1 {
		ctx , cancel := context.WithTimeout( context.Background() , 20 * time.Second )
		err := service.client.Send( ctx , message )
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
