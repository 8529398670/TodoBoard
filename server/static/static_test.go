package static

import (
	io "io"
	http "net/http"
	httptest "net/http/httptest"
	strings "strings"
	testing "testing"
	fstest "testing/fstest"

	fiber "github.com/gofiber/fiber/v3"
)

// A service worker must come from the site root with a script type, or a
// browser refuses it; a stale one is what no-store prevents.
func TestServesServiceWorkerAndManifest( t *testing.T ) {
	server , err := New( fstest.MapFS{
		"index.html":           { Data: []byte( "index" ) },
		"sw.js":                { Data: []byte( "self;" ) },
		"manifest.webmanifest": { Data: []byte( "{}" ) },
	} )
	if err != nil { t.Fatal( err ) }
	app := fiber.New()
	app.Get( "/*" , server.Handler )
	cases := map[string]string{ "/sw.js": "javascript" , "/manifest.webmanifest": "manifest+json" }
	for path , kind := range cases {
		response , err := app.Test( httptest.NewRequest( "GET" , path , nil ) )
		if err != nil { t.Fatal( err ) }
		if response.StatusCode != 200 { t.Errorf( "%s: %d" , path , response.StatusCode ) }
		if strings.Contains( response.Header.Get( "Content-Type" ) , kind ) == false { t.Errorf( "%s type: %q" , path , response.Header.Get( "Content-Type" ) ) }
	}
}

func httptestRequest( path string ) ( request *http.Request ) {
	request , _ = http.NewRequest( http.MethodGet , path , nil )
	return
}

// A text asset is revalidated: the tag comes back as a 304 with no body
// while the file is unchanged, and a changed file gets a new tag and its bytes.
func TestTextAssetsRevalidate( t *testing.T ) {
	files := fstest.MapFS{ "js/app.js": { Data: []byte( "one" ) } }
	server , _ := New( files )
	app := fiber.New()
	app.Get( "/*" , server.Handler )
	get := func( tag string ) ( response *http.Response , body string ) {
		request := httptestRequest( "/js/app.js" )
		request.Header.Set( "Accept-Encoding" , "gzip" )
		if tag != "" { request.Header.Set( "If-None-Match" , tag ) }
		response , err := app.Test( request )
		if err != nil { t.Fatal( err ) }
		raw , _ := io.ReadAll( response.Body )
		body = string( raw )
		return
	}

	first , _ := get( "" )
	tag := first.Header.Get( "ETag" )
	if first.StatusCode != http.StatusOK || tag == "" { t.Fatalf( "first: %d %q" , first.StatusCode , tag ) }
	if cache := first.Header.Get( "Cache-Control" ); cache != "private, no-cache" { t.Errorf( "Cache-Control %q" , cache ) }

	again , body := get( tag )
	if again.StatusCode != http.StatusNotModified || body != "" { t.Errorf( "unchanged: %d %q" , again.StatusCode , body ) }
	if other , _ := get( `W/"0", ` + tag ); other.StatusCode != http.StatusNotModified { t.Errorf( "listed: %d" , other.StatusCode ) }

	files[ "js/app.js" ] = &fstest.MapFile{ Data: []byte( "two" ) }
	changed , _ := get( tag )
	if changed.StatusCode != http.StatusOK || changed.Header.Get( "ETag" ) == tag { t.Errorf( "changed: %d %q" , changed.StatusCode , changed.Header.Get( "ETag" ) ) }
}

