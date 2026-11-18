package routes

import (
	context "context"
	errors "errors"
	time "time"

	fiber "github.com/gofiber/fiber/v3"

	access "todoboard/server/access"
	apidoc "todoboard/server/apidoc"
	gcal "todoboard/server/gcal"
	models "todoboard/server/models"
	security "todoboard/server/security"
)

// Google Calendar sync (server/gcal): each person connects their own Google
// account and picks the boards whose dated cards go on their calendar.

// googleStateCookie holds the state of a connection under way, in the browser
// that began it: the callback takes Google's answer only in that browser, so
// someone cannot get their own half-made connection finished by another
// person's Google account. Lax, since Google's redirect back is a top-level
// navigation from another site, which a Strict cookie would not ride along
// on -- and why the callback cannot rely on the session cookie either.
const googleStateCookie = "todoboard_google_state"

// googleSettings is the gcal service's view of config.yaml, read each time
// since it can change while the server runs.
func ( handlers *Handlers ) GoogleSettings() ( settings gcal.Settings ) {
	cfg := handlers.cfg()
	settings = gcal.Settings{ ClientID: cfg.GoogleClientID , ClientSecret: cfg.GoogleClientSecret , PublicURL: cfg.PublicURL }
	return
}

func ( handlers *Handlers ) googleOn() ( result bool ) {
	result = handlers.Google != nil && handlers.Google.Enabled()
	return
}

func ( handlers *Handlers ) requireGoogle() ( err error ) {
	if handlers.googleOn() == false { err = refuse( fiber.StatusNotFound , "google_off" ) }
	return
}

// googleTouch says a board's cards changed, for anyone syncing it.
func ( handlers *Handlers ) googleTouch( board_id uint64 ) {
	if handlers.Google != nil { handlers.Google.Touch( board_id ) }
}

type googleStatusReply struct {
	Available  bool       `json:"available" jsonschema_description:"Whether this server is set up for Google Calendar at all."`
	Connected  bool       `json:"connected"`
	Email      string     `json:"email,omitempty" jsonschema_description:"The connected Google account."`
	CalendarID string     `json:"calendar_id,omitempty" jsonschema_description:"The calendar cards go to; primary is the account's main one."`
	Boards     []uint64   `json:"boards" jsonschema_description:"Ids of the boards you sync."`
	Problem    string     `json:"problem,omitempty" jsonschema_description:"Why the last sync failed: revoked (connect again) or failed (it will try again)."`
	LastSync   *time.Time `json:"last_sync,omitempty"`
}

// getGoogleStatus says whether the caller's Google Calendar is connected,
// and to what. The token itself never leaves the server.
//
// @route("GET /api/google", summary="Your Google Calendar connection",
//     tag="Google Calendar",
//     description="Whether Google Calendar sync is available on this server, whether your account is "
//         "connected (and which Google account and calendar), the boards you sync, and how the last sync went. "
//         "Connecting happens in a browser, in Settings: Google asks you there.",
//     mcp="get_google_calendar", read_only=True,
// )
func ( handlers *Handlers ) getGoogleStatus( c fiber.Ctx , in *apidoc.None ) ( reply *googleStatusReply , err error ) {
	user := security.UserFrom( c )
	reply = &googleStatusReply{ Available: handlers.googleOn() , Boards: []uint64{} }
	account , get_err := models.GetGoogleAccount( handlers.Store , user.ID )
	if get_err != nil {
		err = internalFailure( get_err )
		return
	}
	if account == nil { return }
	reply.Connected , reply.Email , reply.CalendarID = true , account.Email , account.CalendarID
	reply.Problem , reply.LastSync = account.Problem , account.LastSync
	if boards , _ := models.GoogleSyncBoards( handlers.Store , user.ID ); boards != nil { reply.Boards = boards }
	return
}

// connectGoogle sends the browser to Google to ask for access to the
// caller's calendar. Google sends it back to googleCallback.
//
// @route("GET /api/google/connect", summary="Connect your Google Calendar",
//     tag="Google Calendar", access=SESSION_ONLY, errors=(404,),
//     description="Redirects to Google, which asks you to let this server manage events on your calendars. "
//         "404 google_off: the server has no Google OAuth client set up.",
// )
func ( handlers *Handlers ) connectGoogle( c fiber.Ctx , in *apidoc.None ) ( err error ) {
	if err = handlers.requireGoogle(); err != nil { return }
	user := security.UserFrom( c )
	state , address := handlers.Google.Begin( user.ID )
	c.Cookie( &fiber.Cookie{
		Name: googleStateCookie , Value: state , Path: "/api/google" , MaxAge: 600 ,
		Secure: handlers.cfg().SecureCookies , HTTPOnly: true , SameSite: "Lax" ,
	} )
	err = c.Redirect().Status( fiber.StatusSeeOther ).To( address )
	return
}

type googleCallbackRequest struct {
	Code  string `query:"code" json:"code,omitempty"`
	State string `query:"state" json:"state,omitempty"`
	Error string `query:"error" json:"error,omitempty"`
}

// googleCallback is where Google sends the browser back. It finishes the
// connection begun in this browser, then lands on the Google Calendar
// screen with how it went (?google=connected, denied or failed).
//
// @route("GET /api/google/callback", summary="Google's answer to a connection",
//     tag="Google Calendar", access=PUBLIC,
//     description="Where Google sends you back after you answer. Takes the answer only in the browser that "
//         "began connecting, within ten minutes, once; then redirects to the Google Calendar screen.",
// )
func ( handlers *Handlers ) googleCallback( c fiber.Ctx , in *googleCallbackRequest ) ( err error ) {
	outcome := "connected"
	browser := c.Cookies( googleStateCookie )
	c.Cookie( &fiber.Cookie{ Name: googleStateCookie , Value: "" , Path: "/api/google" , MaxAge: -1 , Secure: handlers.cfg().SecureCookies , HTTPOnly: true , SameSite: "Lax" } )
	switch {
	case handlers.googleOn() == false:
		outcome = "failed"
	case in.Error != "":
		outcome = "denied"
		handlers.Google.Forget( in.State )
	case in.State == "" || browser != in.State:
		outcome = "failed"
		handlers.Google.Forget( in.State )
	default:
		ctx , cancel := context.WithTimeout( context.Background() , 30 * time.Second )
		user_id , finish_err := handlers.Google.Finish( ctx , in.State , in.Code )
		cancel()
		if finish_err != nil {
			outcome = "failed"
			if errors.Is( finish_err , gcal.ErrBadState ) == false { handlers.Logs.System( "warn" , "google calendar connection failed" , map[string]any{ "error": finish_err.Error() } ) }
		} else {
			handlers.Google.TouchUser( user_id )
		}
	}
	err = c.Redirect().Status( fiber.StatusSeeOther ).To( "/?google=" + outcome + "#google" )
	return
}

type googleSettingsRequest struct {
	CalendarID string `json:"calendar_id" jsonschema:"minLength=1,maxLength=300" jsonschema_description:"A calendar id from list_google_calendars; primary is the account's main calendar."`
}

// setGoogleCalendar picks which of the caller's calendars cards go on. The
// next sync moves the events already made.
//
// @route("POST /api/google", summary="Choose your Google calendar",
//     tag="Google Calendar", errors=(400, 404),
//     description="Which calendar your synced cards go on. Events already made move to it on the next sync. "
//         "404 google_off or not_connected.",
//     mcp="set_google_calendar",
// )
func ( handlers *Handlers ) setGoogleCalendar( c fiber.Ctx , in *googleSettingsRequest ) ( reply *googleStatusReply , err error ) {
	if err = handlers.requireGoogle(); err != nil { return }
	if err = refuseLimitedKey( c ); err != nil { return }
	user := security.UserFrom( c )
	account , _ := models.GetGoogleAccount( handlers.Store , user.ID )
	if account == nil {
		err = refuse( fiber.StatusNotFound , "not_connected" )
		return
	}
	if set_err := handlers.Google.SetCalendar( user.ID , in.CalendarID ); set_err != nil {
		err = internalFailure( set_err )
		return
	}
	reply , err = handlers.getGoogleStatus( c , nil )
	return
}

type googleCalendarsReply struct {
	Calendars []gcal.Calendar `json:"calendars"`
}

// listGoogleCalendars is the calendars the caller can put events on.
//
// @route("GET /api/google/calendars", summary="Your Google calendars",
//     tag="Google Calendar", errors=(404, 409),
//     description="The calendars in your connected Google account that you can add events to. "
//         "404 google_off or not_connected; 409 google_unreachable: Google did not answer, or refused (connect again).",
//     mcp="list_google_calendars", read_only=True,
// )
func ( handlers *Handlers ) listGoogleCalendars( c fiber.Ctx , in *apidoc.None ) ( reply *googleCalendarsReply , err error ) {
	if err = handlers.requireGoogle(); err != nil { return }
	user := security.UserFrom( c )
	account , _ := models.GetGoogleAccount( handlers.Store , user.ID )
	if account == nil {
		err = refuse( fiber.StatusNotFound , "not_connected" )
		return
	}
	ctx , cancel := context.WithTimeout( context.Background() , 20 * time.Second )
	defer cancel()
	list , list_err := handlers.Google.Calendars( ctx , user.ID )
	if list_err != nil {
		err = refuse( fiber.StatusConflict , "google_unreachable" )
		return
	}
	if list == nil { list = []gcal.Calendar{} }
	reply = &googleCalendarsReply{ Calendars: list }
	return
}

type googleDisconnectRequest struct {
	RemoveEvents bool `json:"remove_events,omitempty" jsonschema_description:"true also deletes the events this server made on your calendar; otherwise they stay as they are."`
}

// disconnectGoogle forgets the caller's Google account, and tells Google to.
//
// @route("POST /api/google/disconnect", summary="Disconnect your Google Calendar",
//     tag="Google Calendar", errors=(404,),
//     description="Stops syncing and forgets your Google account here (and asks Google to revoke the access). "
//         "remove_events also deletes the events it made; otherwise they stay on your calendar, no longer synced.",
//     mcp="disconnect_google_calendar", destructive=True,
// )
func ( handlers *Handlers ) disconnectGoogle( c fiber.Ctx , in *googleDisconnectRequest ) ( reply *okReply , err error ) {
	if err = refuseLimitedKey( c ); err != nil { return }
	user := security.UserFrom( c )
	if handlers.Google == nil {
		err = refuse( fiber.StatusNotFound , "google_off" )
		return
	}
	ctx , cancel := context.WithTimeout( context.Background() , 2 * time.Minute )
	defer cancel()
	if disconnect_err := handlers.Google.Disconnect( ctx , user.ID , in.RemoveEvents ); disconnect_err != nil {
		err = internalFailure( disconnect_err )
		return
	}
	reply = &okReply{ Ok: true }
	return
}

type boardGoogleRequest struct {
	boardPath
	Sync bool `json:"sync" jsonschema_description:"true copies this board's dated cards to your Google Calendar and back; false stops, and takes its events off your calendar."`
}

type boardGoogleReply struct {
	Sync bool `json:"sync"`
}

// setBoardGoogleSync turns syncing one board to the caller's calendar on or
// off. Theirs alone, like the rest of my-view.
//
// @route("POST /api/boards/{board_id}/my-view/google", summary="Sync a board to your Google Calendar",
//     tag="Google Calendar", errors=(403, 404),
//     description="Turns syncing this board's dated cards with your Google Calendar on or off, for you "
//         "alone. Off deletes the events it made for this board. 404 not_connected: connect Google first.",
//     mcp="set_my_board_google_sync",
// )
func ( handlers *Handlers ) setBoardGoogleSync( c fiber.Ctx , in *boardGoogleRequest ) ( reply *boardGoogleReply , err error ) {
	if err = handlers.requireGoogle(); err != nil { return }
	principal , board , err := handlers.requireBoardAt( c , in.BoardID , access.View )
	if err != nil { return }
	if in.Sync {
		if account , _ := models.GetGoogleAccount( handlers.Store , principal.UserID ); account == nil {
			err = refuse( fiber.StatusNotFound , "not_connected" )
			return
		}
	}
	ctx , cancel := context.WithTimeout( context.Background() , 2 * time.Minute )
	defer cancel()
	if set_err := handlers.Google.SetBoardSync( ctx , principal.UserID , board.ID , in.Sync ); set_err != nil {
		err = modelErr( set_err )
		return
	}
	reply = &boardGoogleReply{ Sync: in.Sync }
	return
}
