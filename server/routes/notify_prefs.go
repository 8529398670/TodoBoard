package routes

import (
	errors "errors"

	fiber "github.com/gofiber/fiber/v3"

	access "todoboard/server/access"
	models "todoboard/server/models"
	notify "todoboard/server/notify"
	security "todoboard/server/security"
)

type prefsRequest struct {
	models.UserPrefs
	CSRFToken string `json:"csrf_token"`
}

type subscriptionRequest struct {
	// Events nil clears the override, so the defaults apply again.
	Events *models.EventSet `json:"events"`
}

// GetNotifyPrefs returns the caller's notification settings. The Pushover key
// is returned masked: it is a credential for the person's phone, and there is
// no reason to send it back to a browser once saved.
func ( handlers *Handlers ) GetNotifyPrefs( c fiber.Ctx ) ( err error ) {
	user := security.UserFrom( c )
	prefs , get_err := models.GetUserPrefs( handlers.Store , user.ID )
	if get_err != nil {
		err = serverError( c )
		return
	}
	err = c.JSON( prefsPayload( prefs , handlers.Notify.Enabled() ) )
	return
}

func prefsPayload( prefs *models.UserPrefs , enabled bool ) ( payload fiber.Map ) {
	masked := ""
	if prefs.PushoverUserKey != "" { masked = prefs.PushoverUserKey[ :4 ] + "…" + prefs.PushoverUserKey[ len( prefs.PushoverUserKey ) - 3: ] }
	payload = fiber.Map{
		"push_enabled":          enabled,
		"has_pushover_key":      prefs.PushoverUserKey != "",
		"pushover_key_hint":     masked,
		"pushover_device":       prefs.PushoverDevice,
		"timezone":              prefs.Timezone,
		"reminder_lead_minutes": prefs.ReminderLeadMinutes,
		"all_day_reminder_hour": prefs.AllDayReminderHour,
		"assignee_defaults":     prefs.AssigneeDefaults,
		"watch_defaults":        prefs.WatchDefaults,
		"shared_with_me":        prefs.SharedWithMe,
		"paused":                prefs.Paused,
	}
	return
}

// SaveNotifyPrefs replaces the settings. An empty pushover_user_key keeps the
// saved one (the browser never had it); "-" removes it.
func ( handlers *Handlers ) SaveNotifyPrefs( c fiber.Ctx ) ( err error ) {
	var body prefsRequest
	if ok , bind_err := bindBody( c , &body ); ok == false { return bind_err }
	if handlers.Guard.CheckCSRF( c , body.CSRFToken ) == false {
		err = forbidden( c , "invalid csrf token" )
		return
	}
	user := security.UserFrom( c )
	current , get_err := models.GetUserPrefs( handlers.Store , user.ID )
	if get_err != nil {
		err = serverError( c )
		return
	}
	prefs := body.UserPrefs
	// Display settings have their own endpoint; this form does not send them.
	prefs.DisplayTimezone , prefs.Hour12 = current.DisplayTimezone , current.Hour12
	switch prefs.PushoverUserKey {
	case "":
		prefs.PushoverUserKey = current.PushoverUserKey
	case "-":
		prefs.PushoverUserKey = ""
	}
	if save_err := models.SaveUserPrefs( handlers.Store , user.ID , &prefs ); save_err != nil {
		err = badRequest( c , "invalid_prefs" )
		return
	}
	err = c.JSON( prefsPayload( &prefs , handlers.Notify.Enabled() ) )
	return
}

func ( handlers *Handlers ) TestNotification( c fiber.Ctx ) ( err error ) {
	var body struct {
		CSRFToken string `json:"csrf_token"`
	}
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
	send_err := handlers.Notify.Test( prefs )
	switch {
	case send_err == nil:
		err = c.JSON( fiber.Map{ "ok": true } )
	case errors.Is( send_err , notify.ErrDisabled ):
		err = badRequest( c , "push_disabled" )
	case errors.Is( send_err , notify.ErrNoUserKey ):
		err = badRequest( c , "push_no_key" )
	case errors.Is( send_err , notify.ErrRejected ):
		// Pushover's own words, which name the actual problem.
		err = c.Status( fiber.StatusBadRequest ).JSON( fiber.Map{ "error": "push_rejected" , "detail": send_err.Error() } )
	default:
		err = c.Status( fiber.StatusBadGateway ).JSON( fiber.Map{ "error": "push_unreachable" } )
	}
	return
}

// Subscriptions: per board and per card overrides of the defaults. Anyone
// with an account who can see the board (or the card) may set their own.

func ( handlers *Handlers ) GetBoardSubscription( c fiber.Ctx ) ( err error ) {
	principal , board , err := handlers.requireBoard( c , access.View )
	if board == nil { return }
	prefs , _ := models.GetUserPrefs( handlers.Store , principal.UserID )
	events , source := models.EffectiveEvents( handlers.Store , prefs , principal.UserID , board.ID , nil )
	err = c.JSON( fiber.Map{ "events": events , "source": source , "push_enabled": handlers.Notify.Enabled() , "has_pushover_key": prefs.PushoverUserKey != "" } )
	return
}

func ( handlers *Handlers ) SetBoardSubscription( c fiber.Ctx ) ( err error ) {
	principal , board , err := handlers.requireBoard( c , access.View )
	if board == nil { return }
	var body subscriptionRequest
	if ok , bind_err := bindBody( c , &body ); ok == false { return bind_err }
	if set_err := models.SetBoardSubscription( handlers.Store , principal.UserID , board.ID , body.Events ); set_err != nil {
		err = serverError( c )
		return
	}
	err = handlers.GetBoardSubscription( c )
	return
}

func ( handlers *Handlers ) GetCardSubscription( c fiber.Ctx ) ( err error ) {
	principal , board , card , err := handlers.requireCard( c , access.View )
	if card == nil { return }
	prefs , _ := models.GetUserPrefs( handlers.Store , principal.UserID )
	events , source := models.EffectiveEvents( handlers.Store , prefs , principal.UserID , board.ID , card )
	err = c.JSON( fiber.Map{ "events": events , "source": source , "push_enabled": handlers.Notify.Enabled() , "has_pushover_key": prefs.PushoverUserKey != "" } )
	return
}

func ( handlers *Handlers ) SetCardSubscription( c fiber.Ctx ) ( err error ) {
	principal , board , card , err := handlers.requireCard( c , access.View )
	if card == nil { return }
	var body subscriptionRequest
	if ok , bind_err := bindBody( c , &body ); ok == false { return bind_err }
	if set_err := models.SetCardSubscription( handlers.Store , principal.UserID , board.ID , card.ID , body.Events ); set_err != nil {
		err = serverError( c )
		return
	}
	err = handlers.GetCardSubscription( c )
	return
}
