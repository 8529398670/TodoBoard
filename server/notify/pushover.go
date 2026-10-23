package notify

import (
	context "context"
	json "encoding/json"
	errors "errors"
	fmt "fmt"
	io "io"
	http "net/http"
	url "net/url"
	strconv "strconv"
	strings "strings"
	time "time"
)

// Message is one Pushover notification. Field limits are Pushover's own:
// title 250 characters, message 1024, url 512, url_title 100.
type Message struct {
	User     string
	Device   string
	Title    string
	Body     string
	URL      string
	URLTitle string
}

// Pushover is a minimal client for the one call this app makes. There is no
// SDK on purpose: the API is a single form POST, and a dependency would be
// more code to audit than the call itself.
type Pushover struct {
	Token  string
	APIURL string
	HTTP   *http.Client
}

// ErrRejected is a 4xx from Pushover other than 429: a bad user key, a
// disabled app token. Retrying cannot fix it.
var ErrRejected = errors.New( "pushover rejected the message" )

func NewPushover( token string , api_url string ) ( client *Pushover ) {
	client = &Pushover{
		Token:  token,
		APIURL: api_url,
		HTTP:   &http.Client{ Timeout: 15 * time.Second },
	}
	return
}

func truncate( value string , max_chars int ) ( result string ) {
	runes := []rune( value )
	if len( runes ) <= max_chars { return value }
	result = string( runes[ :max_chars - 1 ] ) + "…"
	return
}

// Send posts one message. A rejection comes back wrapping ErrRejected with
// Pushover's own explanation, which the "send a test" button shows as-is --
// "user identifier is not a valid user, group, or subscribed user key" is far
// more useful to a person than any wording of ours.
func ( client *Pushover ) Send( ctx context.Context , message Message ) ( err error ) {
	form := url.Values{}
	form.Set( "token" , client.Token )
	form.Set( "user" , message.User )
	if message.Device != "" { form.Set( "device" , message.Device ) }
	form.Set( "title" , truncate( message.Title , 250 ) )
	form.Set( "message" , truncate( message.Body , 1024 ) )
	if message.URL != "" && len( message.URL ) <= 512 {
		form.Set( "url" , message.URL )
		form.Set( "url_title" , truncate( message.URLTitle , 100 ) )
	}

	request , err := http.NewRequestWithContext( ctx , http.MethodPost , client.APIURL , strings.NewReader( form.Encode() ) )
	if err != nil { return }
	request.Header.Set( "Content-Type" , "application/x-www-form-urlencoded" )
	response , err := client.HTTP.Do( request )
	if err != nil { return }
	defer response.Body.Close()
	raw , _ := io.ReadAll( io.LimitReader( response.Body , 64 * 1024 ) )

	if response.StatusCode == http.StatusOK { return }
	var payload struct {
		Errors []string `json:"errors"`
	}
	json.Unmarshal( raw , &payload )
	detail := strings.Join( payload.Errors , "; " )
	if detail == "" { detail = response.Status }
	if response.StatusCode >= 400 && response.StatusCode < 500 && response.StatusCode != http.StatusTooManyRequests {
		err = fmt.Errorf( "%w: %s" , ErrRejected , detail )
		return
	}
	err = &retryableError{ status: response.StatusCode , detail: detail , after: retryAfter( response ) }
	return
}

type retryableError struct {
	status int
	detail string
	after  time.Duration
}

func ( err *retryableError ) Error() ( result string ) {
	result = fmt.Sprintf( "pushover answered %d: %s" , err.status , err.detail )
	return
}

func retryAfter( response *http.Response ) ( wait time.Duration ) {
	if seconds , err := strconv.Atoi( response.Header.Get( "Retry-After" ) ); err == nil && seconds > 0 && seconds < 3600 {
		wait = time.Duration( seconds ) * time.Second
	}
	return
}
