package static

import (
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
