package routes

import (
	errors "errors"

	fiber "github.com/gofiber/fiber/v3"

	models "todoboard/server/models"
	security "todoboard/server/security"
)

type pinRequest struct {
	Pin       string `json:"pin"`
	CSRFToken string `json:"csrf_token"`
}

type renameRequest struct {
	DisplayName string `json:"display_name"`
	CSRFToken   string `json:"csrf_token"`
}

// GetMe is the frontend's first call on every page load: it answers "who am
// I, what may I do, and what CSRF token do I echo back". Handing the token
// out here means no page needs a hidden form field or a template variable.
//
// "role" is the *effective* role, not the account's: a request made with a
// user-scoped API key belonging to an admin reports "user", because that is
// what it may actually do. account_role reports the account itself, for the
// rare caller that needs to tell the two apart. Reporting the account's role
// as "role" would have a client believe it has permissions the server will
// refuse to act on.
func ( handlers *Handlers ) GetMe( c fiber.Ctx ) ( err error ) {
	payload , row_err := handlers.meRow( security.ActorFrom( c ) )
	if row_err != nil {
		err = serverError( c )
		return
	}
	err = c.JSON( payload )
	return
}

// meRow is shared with the MCP whoami tool.
func ( handlers *Handlers ) meRow( actor *security.Actor ) ( payload fiber.Map , err error ) {
	user := actor.User

	// is_owner tells the UI which account-management buttons to offer; the
	// server checks ownership again on every request that depends on it.
	owner_id , err := models.OwnerID( handlers.Store )
	if err != nil { return }

	display := fiber.Map{ "timezone": "" , "hour12": nil , "number_cards": false , "plain_icons": false , "theme": nil , "own_theme_on_boards": false }
	if prefs , prefs_err := models.GetUserPrefs( handlers.Store , user.ID ); prefs_err == nil {
		display = displayPayload( prefs )
	}

	payload = fiber.Map{
		"display":          display,
		"authenticated":    true,
		"id":               user.ID,
		"display_name":     user.DisplayName,
		"role":             actor.Role,
		"account_role":     user.Role,
		"is_owner":         user.ID == owner_id,
		"has_pin":          user.HasPin(),
		"auth":             actor.Method(),
		"api_keys_enabled": handlers.Config.APIKeysEnabled,
	}
	// Only a session has a CSRF token. A key-authenticated request has no
	// ambient credential to protect, so there is nothing to hand back -- see
	// Guard.CheckCSRF.
	if actor.Session != nil {
		payload[ "csrf_token" ] = actor.Session.CSRFToken
	}
	if actor.APIKey != nil {
		payload[ "api_key_id" ] = actor.APIKey.ID
		payload[ "api_key_name" ] = actor.APIKey.Name
	}
	return
}

func ( handlers *Handlers ) RenameAccount( c fiber.Ctx ) ( err error ) {
	var body renameRequest
	if c.Bind().JSON( &body ) != nil {
		err = badRequest( c , "malformed request body" )
		return
	}
	if handlers.Guard.CheckCSRF( c , body.CSRFToken ) == false {
		err = forbidden( c , "invalid csrf token" )
		return
	}
	if models.ValidDisplayName( body.DisplayName ) == false {
		err = badRequest( c , "display name must be 1-80 characters" )
		return
	}

	user := security.UserFrom( c )
	if models.RenameUser( handlers.Store , user.ID , body.DisplayName ) != nil {
		err = serverError( c )
		return
	}
	err = c.JSON( fiber.Map{ "ok": true , "display_name": body.DisplayName } )
	return
}

type displayRequest struct {
	Timezone         string        `json:"timezone"`
	Hour12           *bool         `json:"hour12"`
	NumberCards      *bool         `json:"number_cards"`
	PlainIcons       *bool         `json:"plain_icons"`
	Theme            *models.Theme `json:"theme"`
	ClearTheme       bool          `json:"clear_theme"`
	OwnThemeOnBoards *bool         `json:"own_theme_on_boards"`
	CSRFToken        string        `json:"csrf_token"`
}

func displayPayload( prefs *models.UserPrefs ) ( payload fiber.Map ) {
	payload = fiber.Map{ "timezone": prefs.DisplayTimezone , "hour12": prefs.Hour12 , "number_cards": prefs.NumberCards , "plain_icons": prefs.PlainIcons , "theme": prefs.Theme , "own_theme_on_boards": prefs.OwnThemeOnBoards }
	return
}

// SaveDisplay stores how this person wants times shown: a timezone ("" for
// the browser's own) and 12- or 24-hour clocks (null for the locale's) --
// and, when sent, whether cards are numbered and status icons grey by
// default, the person's theme (clear_theme back to plain light), and
// whether boards with a theme of their own keep showing the person's.
func ( handlers *Handlers ) SaveDisplay( c fiber.Ctx ) ( err error ) {
	var body displayRequest
	if ok , bind_err := bindBody( c , &body ); ok == false { return bind_err }
	if handlers.Guard.CheckCSRF( c , body.CSRFToken ) == false {
		err = forbidden( c , "invalid csrf token" )
		return
	}
	user := security.UserFrom( c )
	prefs , get_err := models.GetUserPrefs( handlers.Store , user.ID )
	if get_err != nil {
		err = serverError( c )
		return
	}
	prefs.DisplayTimezone , prefs.Hour12 = body.Timezone , body.Hour12
	if body.Timezone != "" { prefs.Timezone = body.Timezone }
	if body.NumberCards != nil { prefs.NumberCards = *body.NumberCards }
	if body.PlainIcons != nil { prefs.PlainIcons = *body.PlainIcons }
	if body.OwnThemeOnBoards != nil { prefs.OwnThemeOnBoards = *body.OwnThemeOnBoards }
	if body.Theme != nil {
		if body.Theme.Validate() != nil {
			err = badRequest( c , "invalid_theme" )
			return
		}
		prefs.Theme = body.Theme
	}
	if body.ClearTheme { prefs.Theme = nil }
	if save_err := models.SaveUserPrefs( handlers.Store , user.ID , prefs ); save_err != nil {
		err = badRequest( c , "invalid_timezone" )
		return
	}
	err = c.JSON( displayPayload( prefs ) )
	return
}

// SetPin sets, or with an empty pin clears, the PIN asked for when this
// account signs back in through an invite link (see models/pin.go).
func ( handlers *Handlers ) SetPin( c fiber.Ctx ) ( err error ) {
	var body pinRequest
	if c.Bind().JSON( &body ) != nil {
		err = badRequest( c , "malformed request body" )
		return
	}
	if handlers.Guard.CheckCSRF( c , body.CSRFToken ) == false {
		err = forbidden( c , "invalid csrf token" )
		return
	}
	set_err := models.SetUserPin( handlers.Store , security.UserFrom( c ).ID , body.Pin )
	if errors.Is( set_err , models.ErrBadPin ) {
		err = badRequest( c , "bad_pin" )
		return
	}
	if set_err != nil {
		err = serverError( c )
		return
	}
	err = c.JSON( fiber.Map{ "ok": true , "has_pin": body.Pin != "" } )
	return
}

// IssueOwnLoginLink is ReissueLogin for yourself: a one-time link to sign in
// on another device, or to keep for when this browser loses its cookie. Like
// any new link it replaces one the account had not used yet; this session
// stays signed in.
func ( handlers *Handlers ) IssueOwnLoginLink( c fiber.Ctx ) ( err error ) {
	var body csrfOnlyRequest
	if c.Bind().JSON( &body ) != nil {
		err = badRequest( c , "malformed request body" )
		return
	}
	if handlers.Guard.CheckCSRF( c , body.CSRFToken ) == false {
		err = forbidden( c , "invalid csrf token" )
		return
	}
	credential , token_err := models.IssueLoginToken( handlers.Store , security.UserFrom( c ).ID , handlers.Config.LoginTokenTTL )
	if token_err != nil {
		err = serverError( c )
		return
	}
	err = c.JSON( fiber.Map{
		"login_path":         "/login/" + credential,
		"expires_in_seconds": int( handlers.Config.LoginTokenTTL.Seconds() ),
	} )
	return
}

// Logout deletes the session server-side as well as clearing the cookie, so
// a copy of the cookie captured earlier is dead too.
//
// It is held to the CSRF token like every other state change. SameSite=Strict
// already stops another site from sending the cookie here, but not a page on
// a sibling subdomain, and being signed out by someone else's page is the
// first step of tricking a person into signing in to the wrong account.
//
// Called with an API key it does nothing and says so by succeeding: there is
// no session to destroy. A key is ended by revoking it, not by logging out.
func ( handlers *Handlers ) Logout( c fiber.Ctx ) ( err error ) {
	var body csrfOnlyRequest
	if c.Bind().JSON( &body ) != nil {
		err = badRequest( c , "malformed request body" )
		return
	}
	if handlers.Guard.CheckCSRF( c , body.CSRFToken ) == false {
		err = forbidden( c , "invalid csrf token" )
		return
	}
	handlers.destroyCookieSession( c )
	handlers.Guard.ClearSessionCookie( c )
	err = c.JSON( fiber.Map{ "ok": true } )
	return
}
