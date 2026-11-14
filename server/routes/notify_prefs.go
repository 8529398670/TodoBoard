package routes

import (
	errors "errors"

	fiber "github.com/gofiber/fiber/v3"

	access "todoboard/server/access"
	apidoc "todoboard/server/apidoc"
	models "todoboard/server/models"
	notify "todoboard/server/notify"
	security "todoboard/server/security"
)

// prefsRequest is the notification form: the part of UserPrefs it owns.
type prefsRequest struct {
	PushoverUserKey     string          `json:"pushover_user_key,omitempty" jsonschema_description:"Empty or absent keeps the saved key; \"-\" removes it."`
	PushoverDevice      string          `json:"pushover_device,omitempty"`
	Timezone            string          `json:"timezone,omitempty" jsonschema_description:"An IANA zone such as Europe/Paris, for when reminders fire."`
	ReminderLeadMinutes []int           `json:"reminder_lead_minutes,omitempty" jsonschema_description:"How long before a due time to be reminded, in minutes."`
	AllDayReminderHour  int             `json:"all_day_reminder_hour,omitempty" jsonschema:"minimum=0,maximum=23" jsonschema_description:"The hour a card due on a day (no time) is reminded about."`
	AssigneeDefaults    models.EventSet `json:"assignee_defaults" jsonschema_description:"What to be told about cards you are assigned to."`
	WatchDefaults       models.EventSet `json:"watch_defaults" jsonschema_description:"What to be told about every other card you can see."`
	SharedWithMe        bool            `json:"shared_with_me,omitempty"`
	Paused              bool            `json:"paused,omitempty"`
	NotifyVia           string          `json:"notify_via,omitempty" jsonschema_description:"Which ways of reaching you are used. Left out, it stays as it is."`
}

type subscriptionEvents struct {
	// Events nil clears the override, so the defaults apply again.
	Events *models.EventSet `json:"events" jsonschema_description:"Which events to be told about; null or absent clears the override, so your defaults apply again."`
}

type boardSubscriptionRequest struct {
	boardPath
	subscriptionEvents
}

type cardSubscriptionRequest struct {
	cardPath
	subscriptionEvents
}

// pushUnreachable is TestNotification's 502: the push service could not be
// reached. A refusal at 500 or over is sent as a plain server error, so this
// one answers for itself.
type pushUnreachable struct{}

func ( pushUnreachable ) Error() ( text string ) {
	text = "push_unreachable"
	return
}

func ( pushUnreachable ) Send( c fiber.Ctx ) ( err error ) {
	err = c.Status( fiber.StatusBadGateway ).JSON( fiber.Map{ "error": "push_unreachable" } )
	return
}

func ( pushUnreachable ) Refused() ( status int , message string ) {
	status , message = fiber.StatusBadGateway , "push_unreachable"
	return
}

// getNotifyPrefs returns the caller's notification settings. The Pushover key
// is returned masked: it is a credential for the person's phone, and there is
// no reason to send it back to a browser once saved.
//
// @route("GET /api/notify/prefs", summary="Your notification settings",
//     tag="Notifications",
//     description="How and when you are told about things: Pushover (the key only masked) or this "
//         "browser's push, timezone, reminder lead times, the hour for all-day reminders, the events you "
//         "hear about as an assignee or watcher, whether shares with you are announced, and whether it is "
//         "all paused. Also whether the server can send push at all and whether anything would reach you.",
//     mcp="get_notify_prefs", read_only=True,
// )
func ( handlers *Handlers ) getNotifyPrefs( c fiber.Ctx , in *apidoc.None ) ( reply *fiber.Map , err error ) {
	user := security.UserFrom( c )
	prefs , get_err := models.GetUserPrefs( handlers.Store , user.ID )
	if get_err != nil {
		err = internalFailure( get_err )
		return
	}
	payload := handlers.prefsReply( user.ID , prefs )
	reply = &payload
	return
}

// prefsReply is prefsPayload plus what only the server can tell about how
// this person can be reached.
func ( handlers *Handlers ) prefsReply( user_id uint64 , prefs *models.UserPrefs ) ( payload fiber.Map ) {
	payload = prefsPayload( prefs , handlers.Notify.Enabled() )
	payload[ "pushover_enabled" ] = handlers.Notify.PushoverOn()
	payload[ "web_push_enabled" ] = handlers.Notify.WebPushKey() != ""
	payload[ "can_receive" ] = handlers.canReceive( user_id , prefs )
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
		"notify_via":            prefs.NotifyVia,
	}
	return
}

// saveNotifyPrefs replaces the settings. An empty pushover_user_key keeps the
// saved one (the browser never had it); "-" removes it. Not for a key held to
// some boards: these settings are account-wide, so it could send itself news
// of boards it cannot open.
//
// @route("POST /api/notify/prefs", summary="Save your notification settings",
//     tag="Notifications", errors=(400, 403),
//     description="Replace your notification settings (display settings are kept). An empty "
//         "pushover_user_key keeps the saved one; \"-\" removes it. notify_via left out stays as it is. "
//         "Answers the settings as get_notify_prefs does.",
//     mcp_note="Read them with get_notify_prefs first and send them back changed: what you leave out is cleared.",
//     mcp="set_notify_prefs",
// )
func ( handlers *Handlers ) saveNotifyPrefs( c fiber.Ctx , in *prefsRequest ) ( reply *fiber.Map , err error ) {
	if err = refuseLimitedKey( c ); err != nil { return }
	user := security.UserFrom( c )
	current , get_err := models.GetUserPrefs( handlers.Store , user.ID )
	if get_err != nil {
		err = internalFailure( get_err )
		return
	}
	// Start from what is saved and take only what this form owns. Display
	// settings have their own endpoint, and this way one added later is kept
	// without anyone having to remember to list it here.
	prefs := *current
	prefs.PushoverDevice , prefs.Timezone = in.PushoverDevice , in.Timezone
	prefs.ReminderLeadMinutes , prefs.AllDayReminderHour = in.ReminderLeadMinutes , in.AllDayReminderHour
	prefs.AssigneeDefaults , prefs.WatchDefaults = in.AssigneeDefaults , in.WatchDefaults
	prefs.SharedWithMe , prefs.Paused = in.SharedWithMe , in.Paused
	// Left out, it stays as it is: an older client does not know to send it.
	if in.NotifyVia != "" { prefs.NotifyVia = in.NotifyVia }
	switch in.PushoverUserKey {
	case "":
	case "-":
		prefs.PushoverUserKey = ""
	default:
		prefs.PushoverUserKey = in.PushoverUserKey
	}
	if save_err := models.SaveUserPrefs( handlers.Store , user.ID , &prefs ); save_err != nil {
		err = refuse( fiber.StatusBadRequest , "invalid_prefs" )
		return
	}
	payload := handlers.prefsReply( user.ID , &prefs )
	reply = &payload
	return
}

// sendTestNotification sends the caller a test notification the way their
// settings say.
//
// @route("POST /api/notify/test", summary="Send yourself a test notification",
//     tag="Notifications", errors=(400,),
//     description="Send yourself a test notification the way your settings say. push_disabled: the server "
//         "sends none; push_no_key: nothing set up to reach you; push_rejected: Pushover said no (detail "
//         "has its words); push_unreachable: the push service could not be reached.",
//     mcp="send_test_notification",
// )
func ( handlers *Handlers ) sendTestNotification( c fiber.Ctx , in *apidoc.None ) ( reply *okReply , err error ) {
	user := security.UserFrom( c )
	prefs , get_err := models.GetUserPrefs( handlers.Store , user.ID )
	if get_err != nil {
		err = internalFailure( get_err )
		return
	}
	send_err := handlers.Notify.Test( user.ID , prefs )
	switch {
	case send_err == nil:
		reply = &okReply{ Ok: true }
	case errors.Is( send_err , notify.ErrDisabled ):
		err = refuse( fiber.StatusBadRequest , "push_disabled" )
	case errors.Is( send_err , notify.ErrNoTarget ):
		err = refuse( fiber.StatusBadRequest , "push_no_key" )
	case errors.Is( send_err , notify.ErrRejected ):
		// Pushover's own words, which name the actual problem.
		err = refuse( fiber.StatusBadRequest , "push_rejected" , fiber.Map{ "detail": send_err.Error() } )
	default:
		err = pushUnreachable{}
	}
	return
}

// Subscriptions: per board and per card overrides of the defaults. Anyone
// with an account who can see the board (or the card) may set their own.

func ( handlers *Handlers ) subscriptionReply( principal *access.Principal , board *models.Board , card *models.Card ) ( reply *fiber.Map ) {
	prefs , _ := models.GetUserPrefs( handlers.Store , principal.UserID )
	events , source := models.EffectiveEvents( handlers.Store , prefs , principal.UserID , board.ID , card )
	reply = &fiber.Map{ "events": events , "source": source , "push_enabled": handlers.Notify.Enabled() , "has_pushover_key": prefs.PushoverUserKey != "" , "can_receive": handlers.canReceive( principal.UserID , prefs ) }
	return
}

// getBoardSubscription is which of the board's events reach the caller.
//
// @route("GET /api/boards/{board_id}/subscription", summary="Your notifications for a board",
//     tag="Notifications",
//     description="Which of the board's events reach you (due dates, being assigned, changes, activity) "
//         "and where that comes from (source: your defaults or this board's override), with whether push "
//         "is on and whether anything would reach you.",
//     mcp="get_board_subscription", read_only=True,
// )
func ( handlers *Handlers ) getBoardSubscription( c fiber.Ctx , in *boardPath ) ( reply *fiber.Map , err error ) {
	principal , board , err := handlers.requireBoardAt( c , in.BoardID , access.View )
	if err != nil { return }
	reply = handlers.subscriptionReply( principal , board , nil )
	return
}

// setBoardSubscription sets or clears the caller's override for a board.
//
// @route("POST /api/boards/{board_id}/subscription", summary="Set your notifications for a board",
//     tag="Notifications",
//     description="Choose which of the board's events reach you, or clear your choice (events null) so "
//         "your defaults apply again. Answers as get_board_subscription does.",
//     mcp="set_board_subscription",
// )
func ( handlers *Handlers ) setBoardSubscription( c fiber.Ctx , in *boardSubscriptionRequest ) ( reply *fiber.Map , err error ) {
	principal , board , err := handlers.requireBoardAt( c , in.BoardID , access.View )
	if err != nil { return }
	if set_err := models.SetBoardSubscription( handlers.Store , principal.UserID , board.ID , in.Events ); set_err != nil {
		err = internalFailure( set_err )
		return
	}
	reply = handlers.subscriptionReply( principal , board , nil )
	return
}

// getCardSubscription is which of the card's events reach the caller.
//
// @route("GET /api/boards/{board_id}/cards/{card_id}/subscription", summary="Your notifications for a card",
//     tag="Notifications",
//     description="Which of the card's events reach you and where that comes from (source: your defaults, "
//         "the board's override or this card's), with whether push is on and whether anything would reach you.",
//     mcp="get_card_subscription", read_only=True,
// )
func ( handlers *Handlers ) getCardSubscription( c fiber.Ctx , in *cardPath ) ( reply *fiber.Map , err error ) {
	principal , board , card , err := handlers.requireCardAt( c , in , access.View )
	if err != nil { return }
	reply = handlers.subscriptionReply( principal , board , card )
	return
}

// setCardSubscription sets or clears the caller's override for a card.
//
// @route("POST /api/boards/{board_id}/cards/{card_id}/subscription", summary="Set your notifications for a card",
//     tag="Notifications",
//     description="Choose which of the card's events reach you, or clear your choice (events null) so the "
//         "board's or your defaults apply again. Answers as get_card_subscription does.",
//     mcp="set_card_subscription",
// )
func ( handlers *Handlers ) setCardSubscription( c fiber.Ctx , in *cardSubscriptionRequest ) ( reply *fiber.Map , err error ) {
	principal , board , card , err := handlers.requireCardAt( c , &in.cardPath , access.View )
	if err != nil { return }
	if set_err := models.SetCardSubscription( handlers.Store , principal.UserID , board.ID , card.ID , in.Events ); set_err != nil {
		err = internalFailure( set_err )
		return
	}
	reply = handlers.subscriptionReply( principal , board , card )
	return
}
