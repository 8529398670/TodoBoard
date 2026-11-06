package notify

import (
	context "context"
	log "log"
	debug "runtime/debug"
	time "time"

	models "todoboard/server/models"
)

// Notice is one notification before it is shaped for a way of reaching
// someone. URL is absolute and empty without server.public_url; Path is the
// same place relative to the site, which is all a browser needs.
type Notice struct {
	Title    string
	Body     string
	URL      string
	Path     string
	URLTitle string
	// Topic, when set, lets a newer notice replace one still undelivered.
	Topic string
	// TTL is how long a push service may hold it for an offline device.
	TTL time.Duration
	// Urgent asks a push service not to save it up to spare a phone's battery.
	Urgent bool
}

// Channel is a way of reaching someone. A target is one place a notice goes:
// a Pushover key, one browser. Sends are per target so that a retry never
// repeats a notice to a target that already has it.
type Channel interface {
	Name() string
	// Targets is where this person can be reached this way. None means not
	// at all.
	Targets( user_id uint64 , prefs *models.UserPrefs ) []any
	Send( ctx context.Context , target any , notice Notice ) error
}

type delivery struct {
	target any
	notice Notice
}

// lane is a channel with its own queue and sender, so a channel that is down
// and retrying holds up nothing but itself.
type lane struct {
	channel Channel
	outbox  chan delivery
}

func newLane( channel Channel ) ( result *lane ) {
	result = &lane{ channel: channel , outbox: make( chan delivery , 512 ) }
	return
}

// safely keeps one bad notification from ending a worker: nothing would
// restart it, and every later notification would silently stop.
func safely( name string , work func() ) {
	defer func() {
		if failure := recover(); failure != nil {
			log.Printf( "[notify] %s panicked: %v\n%s" , name , failure , debug.Stack() )
		}
	}()
	work()
}

type pushoverTarget struct {
	user   string
	device string
}

type pushoverChannel struct {
	client *Pushover
}

func ( channel *pushoverChannel ) Name() ( result string ) {
	result = "pushover"
	return
}

func ( channel *pushoverChannel ) Targets( user_id uint64 , prefs *models.UserPrefs ) ( targets []any ) {
	if prefs.PushoverUserKey == "" || prefs.WantsPushover() == false { return }
	targets = []any{ pushoverTarget{ user: prefs.PushoverUserKey , device: prefs.PushoverDevice } }
	return
}

func ( channel *pushoverChannel ) Send( ctx context.Context , target any , notice Notice ) ( err error ) {
	to := target.( pushoverTarget )
	err = channel.client.Send( ctx , Message{
		User: to.user , Device: to.device ,
		Title: notice.Title , Body: notice.Body , URL: notice.URL , URLTitle: notice.URLTitle,
	} )
	return
}
