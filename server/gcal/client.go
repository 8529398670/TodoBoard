package gcal

import (
	bytes "bytes"
	context "context"
	sha256 "crypto/sha256"
	rand "crypto/rand"
	base64 "encoding/base64"
	json "encoding/json"
	errors "errors"
	fmt "fmt"
	io "io"
	http "net/http"
	url "net/url"
	strings "strings"
	time "time"
)

// The Google endpoints. Fields of Service so tests can point them at a fake.
const (
	defaultAuthURL   = "https://accounts.google.com/o/oauth2/v2/auth"
	defaultTokenURL  = "https://oauth2.googleapis.com/token"
	defaultRevokeURL = "https://oauth2.googleapis.com/revoke"
	defaultAPIBase   = "https://www.googleapis.com/calendar/v3"
)

// Scopes: the events on calendars, and the list of calendars (to show which
// account it is and offer a choice). Nothing else of the person's Google.
const scopes = "https://www.googleapis.com/auth/calendar.events https://www.googleapis.com/auth/calendar.calendarlist.readonly"

var (
	// ErrNotFound: the event (or calendar) is gone -- deleted in Google.
	ErrNotFound = errors.New( "gcal: not found" )
	// ErrSyncExpired: Google wants a full read; the sync token is too old.
	ErrSyncExpired = errors.New( "gcal: sync token expired" )
	// ErrRevoked: the refresh token no longer works; connect again.
	ErrRevoked = errors.New( "gcal: access revoked" )
)

// randomText is n random bytes, base64url without padding.
func randomText( n int ) ( result string ) {
	raw := make( []byte , n )
	if _ , err := rand.Read( raw ); err != nil { panic( err ) }
	result = base64.RawURLEncoding.EncodeToString( raw )
	return
}

// challenge is PKCE's S256 of a verifier.
func challenge( verifier string ) ( result string ) {
	sum := sha256.Sum256( []byte( verifier ) )
	result = base64.RawURLEncoding.EncodeToString( sum[ : ] )
	return
}

type tokenReply struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	Error        string `json:"error"`
}

// token posts a form to the token endpoint.
func ( service *Service ) token( ctx context.Context , form url.Values ) ( reply *tokenReply , err error ) {
	settings := service.Settings()
	form.Set( "client_id" , settings.ClientID )
	form.Set( "client_secret" , settings.ClientSecret )
	request , err := http.NewRequestWithContext( ctx , http.MethodPost , service.TokenURL , strings.NewReader( form.Encode() ) )
	if err != nil { return }
	request.Header.Set( "Content-Type" , "application/x-www-form-urlencoded" )
	response , err := service.HTTP.Do( request )
	if err != nil { return }
	defer response.Body.Close()
	body , _ := io.ReadAll( io.LimitReader( response.Body , 1 << 20 ) )
	found := &tokenReply{}
	_ = json.Unmarshal( body , found )
	if response.StatusCode != http.StatusOK {
		if found.Error == "invalid_grant" {
			err = ErrRevoked
			return
		}
		err = fmt.Errorf( "gcal: token endpoint answered %d %s" , response.StatusCode , found.Error )
		return
	}
	reply = found
	return
}

// accessToken is a fresh access token for this refresh token, cached until
// a minute before it runs out.
func ( service *Service ) accessToken( ctx context.Context , refresh string ) ( token string , err error ) {
	service.mu.Lock()
	cached , ok := service.tokens[ refresh ]
	service.mu.Unlock()
	if ok && time.Now().Before( cached.expires ) {
		token = cached.token
		return
	}
	reply , err := service.token( ctx , url.Values{ "grant_type": { "refresh_token" } , "refresh_token": { refresh } } )
	if err != nil { return }
	token = reply.AccessToken
	life := time.Duration( reply.ExpiresIn ) * time.Second
	if life <= 0 { life = time.Hour }
	service.mu.Lock()
	service.tokens[ refresh ] = cachedToken{ token: token , expires: time.Now().Add( life - time.Minute ) }
	service.mu.Unlock()
	return
}

// call makes one Calendar API request and reads its JSON answer into out.
func ( service *Service ) call( ctx context.Context , refresh string , method string , path string , query url.Values , body any , out any ) ( err error ) {
	token , err := service.accessToken( ctx , refresh )
	if err != nil { return }
	address := service.APIBase + path
	if len( query ) > 0 { address += "?" + query.Encode() }
	var reader io.Reader
	if body != nil {
		raw , marshal_err := json.Marshal( body )
		if marshal_err != nil { return marshal_err }
		reader = bytes.NewReader( raw )
	}
	request , err := http.NewRequestWithContext( ctx , method , address , reader )
	if err != nil { return }
	request.Header.Set( "Authorization" , "Bearer " + token )
	if body != nil { request.Header.Set( "Content-Type" , "application/json" ) }
	response , err := service.HTTP.Do( request )
	if err != nil { return }
	defer response.Body.Close()
	raw , _ := io.ReadAll( io.LimitReader( response.Body , 8 << 20 ) )
	switch {
	case response.StatusCode == http.StatusNotFound || ( response.StatusCode == http.StatusGone && method != http.MethodGet ):
		err = ErrNotFound
	case response.StatusCode == http.StatusGone:
		err = ErrSyncExpired
	case response.StatusCode == http.StatusUnauthorized:
		// A cached access token Google no longer takes: forget it.
		service.mu.Lock()
		delete( service.tokens , refresh )
		service.mu.Unlock()
		err = fmt.Errorf( "gcal: %s %s answered 401" , method , path )
	case response.StatusCode >= 300:
		err = fmt.Errorf( "gcal: %s %s answered %d" , method , path , response.StatusCode )
	case out != nil && len( raw ) > 0:
		err = json.Unmarshal( raw , out )
	}
	return
}

func calendarPath( calendar_id string ) ( path string ) {
	path = "/calendars/" + url.PathEscape( calendar_id ) + "/events"
	return
}

func ( service *Service ) insertEvent( ctx context.Context , refresh string , calendar_id string , event *Event ) ( made *Event , err error ) {
	made = &Event{}
	err = service.call( ctx , refresh , http.MethodPost , calendarPath( calendar_id ) , nil , event , made )
	return
}

func ( service *Service ) patchEvent( ctx context.Context , refresh string , calendar_id string , event_id string , event *Event ) ( err error ) {
	err = service.call( ctx , refresh , http.MethodPatch , calendarPath( calendar_id ) + "/" + url.PathEscape( event_id ) , nil , event , nil )
	return
}

// deleteEvent: an event already gone is fine.
func ( service *Service ) deleteEvent( ctx context.Context , refresh string , calendar_id string , event_id string ) ( err error ) {
	err = service.call( ctx , refresh , http.MethodDelete , calendarPath( calendar_id ) + "/" + url.PathEscape( event_id ) , nil , nil , nil )
	if errors.Is( err , ErrNotFound ) { err = nil }
	return
}

type eventPage struct {
	Items         []*Event `json:"items"`
	NextPageToken string   `json:"nextPageToken"`
	NextSyncToken string   `json:"nextSyncToken"`
}

// changedEvents is every event changed since sync_token ("" reads them
// all), and the token for next time.
func ( service *Service ) changedEvents( ctx context.Context , refresh string , calendar_id string , sync_token string ) ( events []*Event , next string , err error ) {
	page_token := ""
	for {
		query := url.Values{ "maxResults": { "2500" } }
		// A first read takes every event: Google hands out a sync token only
		// for an unfiltered read. Events not ours are skipped by the caller.
		if sync_token != "" { query.Set( "syncToken" , sync_token ) }
		if page_token != "" { query.Set( "pageToken" , page_token ) }
		page := &eventPage{}
		if err = service.call( ctx , refresh , http.MethodGet , calendarPath( calendar_id ) , query , nil , page ); err != nil { return }
		events = append( events , page.Items... )
		if page.NextPageToken == "" {
			next = page.NextSyncToken
			return
		}
		page_token = page.NextPageToken
	}
}

// Calendar is one calendar the person can add events to.
type Calendar struct {
	ID      string `json:"id"`
	Summary string `json:"summary"`
	Primary bool   `json:"primary,omitempty"`
}

func ( service *Service ) calendars( ctx context.Context , refresh string ) ( list []Calendar , err error ) {
	reply := &struct{ Items []Calendar `json:"items"` }{}
	err = service.call( ctx , refresh , http.MethodGet , "/users/me/calendarList" , url.Values{ "minAccessRole": { "writer" } } , nil , reply )
	list = reply.Items
	return
}

// primaryEmail is the account's address: its main calendar's id.
func ( service *Service ) primaryEmail( ctx context.Context , refresh string ) ( email string , err error ) {
	reply := &Calendar{}
	err = service.call( ctx , refresh , http.MethodGet , "/users/me/calendarList/primary" , nil , nil , reply )
	email = reply.ID
	return
}

// revoke tells Google to forget the grant. Best effort: disconnecting
// forgets the token here whatever Google says.
func ( service *Service ) revoke( ctx context.Context , refresh string ) {
	request , err := http.NewRequestWithContext( ctx , http.MethodPost , service.RevokeURL , strings.NewReader( url.Values{ "token": { refresh } }.Encode() ) )
	if err != nil { return }
	request.Header.Set( "Content-Type" , "application/x-www-form-urlencoded" )
	response , err := service.HTTP.Do( request )
	if err == nil { response.Body.Close() }
}
