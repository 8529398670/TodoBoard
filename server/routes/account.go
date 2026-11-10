package routes

import (
	errors "errors"

	fiber "github.com/gofiber/fiber/v3"

	logs "todoboard/server/logs"
	models "todoboard/server/models"
	security "todoboard/server/security"
)

type pinRequest struct {
	Pin       string `json:"pin"`
	CSRFToken string `json:"csrf_token"`
}

type renameRequest struct {
	DisplayName string `json:"display_name"`
	// Expect is optional: the name the caller expects to replace. The
	// frontend always sends it, so its undo cannot overwrite a rename made
	// elsewhere.
	Expect    *string `json:"expect"`
	CSRFToken string  `json:"csrf_token"`
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
		err = serverError( c , row_err )
		return
	}
	err = c.JSON( payload )
	return
}

// meRow is GetMe's answer, shared with the MCP whoami tool so the two cannot
// describe the same caller differently.
func ( handlers *Handlers ) meRow( actor *security.Actor ) ( payload fiber.Map , err error ) {
	user := actor.User

	// is_owner tells the UI which account-management buttons to offer; the
	// server checks ownership again on every request that depends on it.
	owner_id , err := models.OwnerID( handlers.Store )
	if err != nil { return }
	login_link , err := models.LiveLoginLink( handlers.Store , user.ID )
	if err != nil { return }

	display := fiber.Map{ "timezone": "" , "hour12": nil , "number_cards": false , "plain_icons": false , "done_on_top": false , "add_on_top": false , "theme": nil , "own_theme_on_boards": false , "swipe_columns": true , "folder_sort": "" , "label_colors": []models.SavedColor{} }
	if prefs , prefs_err := models.GetUserPrefs( handlers.Store , user.ID ); prefs_err == nil {
		display = displayPayload( prefs )
	}
	payload = fiber.Map{
		"authenticated":    true,
		"display":          display,
		"id":               user.ID,
		"display_name":     user.DisplayName,
		"role":             actor.Role,
		"account_role":     user.Role,
		"is_owner":         user.ID == owner_id,
		"login_link":       login_link,
		"maintenance":      handlers.cfg().MaintenanceMode(),
		"version":          handlers.Version,
		"announcement":     handlers.cfg().Announcement,
		"repo":             handlers.cfg().Repo,
		"description":      handlers.cfg().Description,
		"show_about":       handlers.cfg().ShowAbout,
		"openapi_url":      handlers.openAPIURL(),
		"mcp_url":          handlers.mcpURL(),
		"has_pin":          user.HasPin(),
		"username":           user.Username,
		"has_password":       user.HasPassword(),
		"password_blocked":   user.PasswordBlocked,
		"two_factor":         user.TwoFactorOn(),
		"passwords_enabled":  handlers.cfg().PasswordsEnabled,
		"two_factor_enabled": handlers.cfg().TwoFactorEnabled,
		"auth":             actor.Method(),
		"api_keys_enabled": handlers.cfg().APIKeysEnabled,
		"mcp_enabled": handlers.cfg().MCPEnabled,
		"invite_links_enabled": handlers.inviteLinksOn(),
		// Whether track.js sends anything, and whether it sends clicks.
		"logs_enabled":   handlers.Logs.Enabled(),
		"logs_ui_clicks": handlers.Logs.Enabled() && handlers.cfg().Logs.UIClicks,
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
	payload , refused := handlers.renameAccount( c , body.DisplayName , body.Expect )
	if refused != nil {
		err = refused.send( c )
		return
	}
	err = c.JSON( payload )
	return
}

// renameAccount is RenameAccount after the request is read, for it and the
// MCP rename_account tool.
func ( handlers *Handlers ) renameAccount( c fiber.Ctx , display_name string , expect *string ) ( payload fiber.Map , refused *refusal ) {
	if models.ValidDisplayName( display_name ) == false {
		refused = refuse( fiber.StatusBadRequest , "display name must be 1-80 characters" )
		return
	}
	user := security.UserFrom( c )
	rename_err := models.RenameUser( handlers.Store , user.ID , display_name , expect )
	var conflict_err *models.RenameConflictError
	if errors.As( rename_err , &conflict_err ) {
		refused = refuse( fiber.StatusConflict , conflict_err.Error() , fiber.Map{ "display_name": conflict_err.Current } )
		return
	}
	if rename_err != nil {
		refused = internalFailure( rename_err )
		return
	}
	handlers.audit( c , logs.KindAuth , "renamed own account" , fiber.Map{ "name": display_name } )
	payload = fiber.Map{ "ok": true , "display_name": display_name }
	return
}

type displayRequest struct {
	Timezone         string        `json:"timezone"`
	Hour12           *bool         `json:"hour12"`
	NumberCards      *bool         `json:"number_cards"`
	PlainIcons       *bool         `json:"plain_icons"`
	DoneOnTop        *bool         `json:"done_on_top"`
	AddOnTop         *bool         `json:"add_on_top"`
	Theme            *models.DisplayTheme `json:"theme"`
	ClearTheme       bool          `json:"clear_theme"`
	OwnThemeOnBoards *bool         `json:"own_theme_on_boards"`
	SwipeColumns     *bool         `json:"swipe_columns"`
	FolderSort       *string       `json:"folder_sort"`
	CSRFToken        string        `json:"csrf_token"`
}

func displayPayload( prefs *models.UserPrefs ) ( payload fiber.Map ) {
	payload = fiber.Map{ "timezone": prefs.DisplayTimezone , "hour12": prefs.Hour12 , "number_cards": prefs.NumberCards , "plain_icons": prefs.PlainIcons , "done_on_top": prefs.DoneOnTop , "add_on_top": prefs.AddOnTop , "theme": prefs.Theme , "own_theme_on_boards": prefs.OwnThemeOnBoards , "swipe_columns": prefs.SwipeColumns == nil || *prefs.SwipeColumns , "folder_sort": prefs.FolderSort , "label_colors": prefs.LabelColors }
	if prefs.LabelColors == nil { payload[ "label_colors" ] = []models.SavedColor{} }
	return
}

type labelColorsRequest struct {
	Colors    []models.SavedColor `json:"colors"`
	CSRFToken string              `json:"csrf_token"`
}

// SaveLabelColors replaces the person's own label colours with the list
// sent, in its order: the board's label editor adds to it, the account page
// renames and removes. Labels already using a removed colour keep it.
func ( handlers *Handlers ) SaveLabelColors( c fiber.Ctx ) ( err error ) {
	var body labelColorsRequest
	if ok , bind_err := bindBody( c , &body ); ok == false { return bind_err }
	if handlers.Guard.CheckCSRF( c , body.CSRFToken ) == false {
		err = forbidden( c , "invalid csrf token" )
		return
	}
	user := security.UserFrom( c )
	prefs , get_err := models.GetUserPrefs( handlers.Store , user.ID )
	if get_err != nil {
		err = serverError( c , get_err )
		return
	}
	prefs.LabelColors = body.Colors
	if save_err := models.SaveUserPrefs( handlers.Store , user.ID , prefs ); save_err != nil {
		err = badRequest( c , "invalid_label_colors" )
		return
	}
	err = c.JSON( displayPayload( prefs ) )
	return
}

// SaveDisplay stores how this person wants times shown: a timezone ("" for
// the browser's own) and 12- or 24-hour clocks (null for the locale's) --
// and, when sent, whether cards are numbered and status icons grey by
// default, the person's theme (clear_theme back to plain light), and
// whether boards with a theme of their own keep showing the person's, phone
// swipe columns, and how home page folders are ordered.
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
		err = serverError( c , get_err )
		return
	}
	prefs.DisplayTimezone , prefs.Hour12 = body.Timezone , body.Hour12
	if body.Timezone != "" { prefs.Timezone = body.Timezone }
	if body.NumberCards != nil { prefs.NumberCards = *body.NumberCards }
	if body.PlainIcons != nil { prefs.PlainIcons = *body.PlainIcons }
	if body.DoneOnTop != nil { prefs.DoneOnTop = *body.DoneOnTop }
	if body.AddOnTop != nil { prefs.AddOnTop = *body.AddOnTop }
	if body.OwnThemeOnBoards != nil { prefs.OwnThemeOnBoards = *body.OwnThemeOnBoards }
	if body.SwipeColumns != nil { prefs.SwipeColumns = body.SwipeColumns }
	if body.FolderSort != nil { prefs.FolderSort = *body.FolderSort }
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
		err = serverError( c , set_err )
		return
	}
	handlers.audit( c , logs.KindAuth , "pin changed" , fiber.Map{ "has_pin": body.Pin != "" } )
	err = c.JSON( fiber.Map{ "ok": true , "has_pin": body.Pin != "" } )
	return
}

// CreateMyLoginLink mints a login link for the caller's own account: the way
// to sign in on another device without asking an admin. Once by default, or
// reusable, like ReissueLogin -- it is ReissueLogin pointed at yourself, so it
// has the same effect: any link the account could still use stops working,
// and existing sessions stay signed in.
//
// Session only (see routes.go): a login link becomes a session, so an API
// key that could call this could turn itself into one.
func ( handlers *Handlers ) CreateMyLoginLink( c fiber.Ctx ) ( err error ) {
	var body loginLinkBody
	if c.Bind().JSON( &body ) != nil {
		err = badRequest( c , "malformed request body" )
		return
	}
	if handlers.Guard.CheckCSRF( c , body.CSRFToken ) == false {
		err = forbidden( c , "invalid csrf token" )
		return
	}
	options , ok := handlers.loginLinkOptions( c , body.LoginLinkRequest )
	if ok == false { return }
	credential , info , token_err := models.IssueLoginLink( handlers.Store , security.UserFrom( c ).ID , options )
	if token_err != nil {
		err = serverError( c , token_err )
		return
	}
	handlers.audit( c , logs.KindAuth , "own login link created" , fiber.Map{ "max_uses": info.MaxUses , "expires_at": info.ExpiresAt } )
	err = c.JSON( loginLinkResponse( credential , info ) )
	return
}

// CancelMyLoginLink stops the caller's own live login link working.
func ( handlers *Handlers ) CancelMyLoginLink( c fiber.Ctx ) ( err error ) {
	var body csrfOnlyRequest
	if c.Bind().JSON( &body ) != nil {
		err = badRequest( c , "malformed request body" )
		return
	}
	if handlers.Guard.CheckCSRF( c , body.CSRFToken ) == false {
		err = forbidden( c , "invalid csrf token" )
		return
	}
	removed , cancel_err := models.CancelLoginLink( handlers.Store , security.UserFrom( c ).ID )
	if cancel_err != nil {
		err = serverError( c , cancel_err )
		return
	}
	handlers.audit( c , logs.KindAuth , "own login link cancelled" , fiber.Map{ "removed": removed } )
	err = c.JSON( fiber.Map{ "ok": true , "removed": removed } )
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
	handlers.audit( c , logs.KindAuth , "signed out" , nil )
	handlers.destroyCookieSession( c )
	handlers.Guard.ClearSessionCookie( c )
	err = c.JSON( fiber.Map{ "ok": true } )
	return
}

// openAPIURL is where the API reference is read: config/openapi.json in
// app.repo when that is set, so it opens as a page there, else this server's
// own copy, which needs the same sign-in.
func ( handlers *Handlers ) openAPIURL() ( result string ) {
	result = handlers.cfg().RepoFile( "config/openapi.json" )
	if result == "" {
		result = "/api/openapi.json"
	}
	return
}

// mcpURL is where the MCP reference is read: config/mcp.json in app.repo when
// that is set, else this server's own copy, which needs the same sign-in.
func ( handlers *Handlers ) mcpURL() ( result string ) {
	result = handlers.cfg().RepoFile( "config/mcp.json" )
	if result == "" {
		result = "/api/mcp.json"
	}
	return
}
