package notify

import (
	context "context"
	json "encoding/json"
	fmt "fmt"
	io "io"
	log "log"
	net "net"
	http "net/http"
	url "net/url"
	bytes "bytes"
	strings "strings"
	time "time"

	config "todoboard/server/config"
	db "todoboard/server/db"
	encryption "todoboard/server/encryption"
	models "todoboard/server/models"
	product "todoboard/server/product"
)

// webPushHosts are the push services browsers use. An endpoint comes from the
// browser, so without this list a signed-in person could point this server at
// any address: the public-address check stops the local network, and this
// stops the rest of the internet. Each entry matches the host or, with a
// leading dot, anything under it.
var webPushHosts = []string{
	"fcm.googleapis.com",
	".push.services.mozilla.com",
	".push.apple.com",
	".notify.windows.com",
}

// EndpointAllowed reports whether endpoint is somewhere this server will send
// a notification.
func EndpointAllowed( endpoint string ) ( result bool ) {
	parsed , err := url.Parse( endpoint )
	if err != nil || parsed.Scheme != "https" || parsed.User != nil { return }
	if port := parsed.Port(); port != "" && port != "443" { return }
	host := strings.ToLower( parsed.Hostname() )
	if host == "" || net.ParseIP( host ) != nil { return }
	for _ , allowed := range webPushHosts {
		if host == allowed || ( strings.HasPrefix( allowed , "." ) && strings.HasSuffix( host , allowed ) ) { return true }
	}
	return
}

// A notification's size is capped well under the 4 KB a push service takes,
// and its words to what a phone's lock screen can show anyway.
const (
	webPushTitleChars = 120
	webPushBodyChars  = 300
)

type webPushChannel struct {
	store   *db.Store
	keys    *models.VAPIDKeys
	subject string
	client  *http.Client
	allowed func( endpoint string ) bool
	now     func() time.Time
}

// webPushSubject is how a push service can reach whoever runs this server.
func webPushSubject( cfg *config.Config ) ( subject string ) {
	switch {
	case cfg.WebPushContact != "":
		subject = cfg.WebPushContact
	case strings.HasPrefix( cfg.PublicURL , "https://" ):
		subject = cfg.PublicURL
	default:
		log.Printf( "[notify] notify.web_push_contact is not set; push services have no way to reach you" )
		subject = "mailto:admin@todoboard.invalid"
	}
	return
}

func newWebPushChannel( store *db.Store , cfg *config.Config , keys *models.VAPIDKeys ) ( channel *webPushChannel ) {
	channel = &webPushChannel{
		store:   store,
		keys:    keys,
		subject: webPushSubject( cfg ),
		client:  product.NewPublicClient( 15 * time.Second ),
		allowed: EndpointAllowed,
		now:     time.Now,
	}
	return
}

func ( channel *webPushChannel ) Name() ( result string ) {
	result = "web push"
	return
}

func ( channel *webPushChannel ) Targets( user_id uint64 , prefs *models.UserPrefs ) ( targets []any ) {
	if prefs.WantsDevices() == false { return }
	subscriptions , err := models.ListPushSubscriptions( channel.store , user_id )
	if err != nil { return }
	for _ , subscription := range subscriptions { targets = append( targets , subscription ) }
	return
}

func ( channel *webPushChannel ) forget( subscription *models.PushSubscription ) {
	if err := models.DeletePushSubscription( channel.store , subscription.UserID , subscription.ID ); err != nil {
		log.Printf( "[notify] could not remove a dead browser: %v" , err )
	}
}

func ( channel *webPushChannel ) Send( ctx context.Context , target any , notice Notice ) ( err error ) {
	subscription := target.( *models.PushSubscription )
	// Asked again here: the list can have shrunk since this was saved.
	if channel.allowed( subscription.Endpoint ) == false {
		channel.forget( subscription )
		return fmt.Errorf( "%w: push service not allowed" , ErrRejected )
	}
	payload , err := json.Marshal( map[string]string{
		"title": truncate( notice.Title , webPushTitleChars ),
		"body":  truncate( notice.Body , webPushBodyChars ),
		"path":  notice.Path,
		"tag":   notice.Topic,
	} )
	if err != nil { return }
	body , err := encryption.EncryptWebPush( subscription.P256dh , subscription.Auth , payload )
	if err != nil { return }

	parsed , err := url.Parse( subscription.Endpoint )
	if err != nil { return }
	authorization , err := encryption.VAPIDAuthorization( channel.keys.Private , channel.keys.Public , parsed.Scheme + "://" + parsed.Host , channel.subject , channel.now() )
	if err != nil { return }

	request , err := http.NewRequestWithContext( ctx , http.MethodPost , subscription.Endpoint , bytes.NewReader( body ) )
	if err != nil { return }
	ttl := notice.TTL
	if ttl == 0 { ttl = routineTTL }
	request.Header.Set( "Authorization" , authorization )
	request.Header.Set( "Content-Encoding" , "aes128gcm" )
	request.Header.Set( "Content-Type" , "application/octet-stream" )
	request.Header.Set( "TTL" , fmt.Sprint( int( ttl.Seconds() ) ) )
	urgency := "normal"
	if notice.Urgent { urgency = "high" }
	request.Header.Set( "Urgency" , urgency )
	if notice.Topic != "" { request.Header.Set( "Topic" , notice.Topic ) }

	response , err := channel.client.Do( request )
	if err != nil { return }
	defer response.Body.Close()
	raw , _ := io.ReadAll( io.LimitReader( response.Body , 4 * 1024 ) )

	switch {
	case response.StatusCode >= 200 && response.StatusCode < 300:
		if mark_err := models.MarkPushDelivered( channel.store , subscription , channel.now() ); mark_err != nil {
			log.Printf( "[notify] could not note a delivery: %v" , mark_err )
		}
	case response.StatusCode == http.StatusNotFound || response.StatusCode == http.StatusGone:
		// The browser dropped this subscription, and it will not be back.
		channel.forget( subscription )
		err = fmt.Errorf( "%w: browser is gone" , ErrRejected )
	case response.StatusCode == http.StatusTooManyRequests || response.StatusCode >= 500:
		err = &retryableError{ status: response.StatusCode , detail: response.Status , after: retryAfter( response ) }
	default:
		err = fmt.Errorf( "%w: %s %s" , ErrRejected , response.Status , truncate( string( raw ) , 200 ) )
	}
	return
}
