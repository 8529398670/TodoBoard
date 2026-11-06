package routes

import (
	json "encoding/json"
	http "net/http"
	os "os"
	regexp "regexp"
	sort "sort"
	strings "strings"
	testing "testing"

	fiber "github.com/gofiber/fiber/v3"
)

// notInTheSpec is what the route table has that is not an API: the pages a
// link opens, and the catch-all.
var notInTheSpec = map[string]bool{
	"GET /login/*": true,
	"GET /join/*":  true,
	"GET /b/*":     true,
	"GET /s/*":     true,
	"GET /print/*": true,
	"GET /*":       true,
}

func specFile( t *testing.T ) ( raw []byte ) {
	t.Helper()
	raw , err := os.ReadFile( "../../config/openapi.json" )
	if err != nil { t.Fatal( err ) }
	return
}

// specRoutes is every "METHOD /path" the spec lists.
func specRoutes( t *testing.T ) ( routes map[string]bool ) {
	t.Helper()
	var document struct {
		Paths map[string]map[string]any `json:"paths"`
	}
	if err := json.Unmarshal( specFile( t ) , &document ); err != nil { t.Fatal( err ) }
	routes = map[string]bool{}
	for path , methods := range document.Paths {
		for method := range methods {
			routes[ strings.ToUpper( method ) + " " + path ] = true
		}
	}
	return
}

// A new route that is not in config/openapi.json fails here, and so does an
// entry for a route that has gone. The fixture has MCP on, so /mcp is in the
// route table.
func TestTheSpecAndTheRouteTableAgree( t *testing.T ) {
	fixture := newMCPFixture( t )
	parameter := regexp.MustCompile( `:(\w+)` )
	registered := map[string]bool{}
	for _ , route := range fixture.app.GetRoutes( true ) {
		if route.Method != http.MethodGet && route.Method != http.MethodPost && route.Method != http.MethodDelete { continue }
		name := route.Method + " " + parameter.ReplaceAllString( route.Path , "{$1}" )
		if notInTheSpec[ name ] == false {
			registered[ name ] = true
		}
	}
	spec := specRoutes( t )
	problems := []string{}
	for name := range registered {
		if spec[ name ] == false { problems = append( problems , "not in config/openapi.json: " + name ) }
	}
	for name := range spec {
		if registered[ name ] == false { problems = append( problems , "in config/openapi.json but not a route: " + name ) }
	}
	sort.Strings( problems )
	if len( problems ) > 0 {
		t.Fatal( strings.Join( problems , "\n" ) )
	}
}

// Signed in by either door is enough; signed out is not.
func TestTheSpecIsForSignedInCallers( t *testing.T ) {
	fixture := newKeyBoards( t )
	fixture.handlers.OpenAPI = specFile( t )
	if status , _ := fixture.call( t , http.MethodGet , "/api/openapi.json" , "" , "" , "" ); status != fiber.StatusUnauthorized {
		t.Fatalf( "signed out: status %d" , status )
	}
	cookie , _ := fixture.session( t )
	key , _ := fixture.key( t , "" )
	for name , caller := range map[string][2]string{ "session": { "" , cookie } , "key": { key , "" } } {
		status , text := fixture.call( t , http.MethodGet , "/api/openapi.json" , caller[ 0 ] , caller[ 1 ] , "" )
		var document map[string]any
		if err := json.Unmarshal( []byte( text ) , &document ); err != nil { t.Fatalf( "%s: %v" , name , err ) }
		if status != fiber.StatusOK || document[ "openapi" ] != "3.1.0" {
			t.Fatalf( "%s: status %d, openapi %v" , name , status , document[ "openapi" ] )
		}
	}
}

// With no spec set, the route says so rather than sending an empty 200.
func TestNoSpecIsNotFound( t *testing.T ) {
	fixture := newKeyBoards( t )
	key , _ := fixture.key( t , "" )
	if status , _ := fixture.call( t , http.MethodGet , "/api/openapi.json" , key , "" , "" ); status != fiber.StatusNotFound {
		t.Fatalf( "status %d" , status )
	}
}
