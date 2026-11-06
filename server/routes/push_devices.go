package routes

import (
	errors "errors"

	fiber "github.com/gofiber/fiber/v3"

	models "todoboard/server/models"
	notify "todoboard/server/notify"
	security "todoboard/server/security"
)

// Browsers that take notifications. Each is the caller's own, so these sit
// with the account routes, which leave the CSRF check to each handler.

type pushDeviceRequest struct {
	Endpoint  string `json:"endpoint"`
	P256dh    string `json:"p256dh"`
	Auth      string `json:"auth"`
	Label     string `json:"label"`
	CSRFToken string `json:"csrf_token"`
}

// devicePayload leaves the endpoint out: a browser recognises itself by id.
func devicePayload( subscription *models.PushSubscription ) ( payload fiber.Map ) {
	payload = fiber.Map{
		"id":           subscription.ID,
		"label":        subscription.Label,
		"created_at":   subscription.CreatedAt,
		"last_success": subscription.LastSuccess,
	}
	return
}

func ( handlers *Handlers ) ListPushDevices( c fiber.Ctx ) ( err error ) {
	user := security.UserFrom( c )
	subscriptions , list_err := models.ListPushSubscriptions( handlers.Store , user.ID )
	if list_err != nil {
		err = serverError( c )
		return
	}
	devices := []fiber.Map{}
	for _ , subscription := range subscriptions { devices = append( devices , devicePayload( subscription ) ) }
	err = c.JSON( fiber.Map{ "public_key": handlers.Notify.WebPushKey() , "devices": devices } )
	return
}

// SavePushDevice is also how a browser checks in: the settings page sends its
// subscription each time it opens, so one the browser replaced is caught.
func ( handlers *Handlers ) SavePushDevice( c fiber.Ctx ) ( err error ) {
	var body pushDeviceRequest
	if ok , bind_err := bindBody( c , &body ); ok == false { return bind_err }
	if handlers.Guard.CheckCSRF( c , body.CSRFToken ) == false {
		err = forbidden( c , "invalid csrf token" )
		return
	}
	if handlers.Notify.WebPushKey() == "" {
		err = badRequest( c , "push_disabled" )
		return
	}
	if notify.EndpointAllowed( body.Endpoint ) == false {
		err = badRequest( c , "push_endpoint_refused" )
		return
	}
	user := security.UserFrom( c )
	subscription , save_err := models.SavePushSubscription( handlers.Store , user.ID , body.Endpoint , body.P256dh , body.Auth , body.Label )
	if errors.Is( save_err , models.ErrTooManyDevices ) {
		err = badRequest( c , "push_too_many_devices" )
		return
	}
	if save_err != nil {
		err = modelError( c , save_err )
		return
	}
	err = c.JSON( devicePayload( subscription ) )
	return
}

func ( handlers *Handlers ) DeletePushDevice( c fiber.Ctx ) ( err error ) {
	var body struct {
		CSRFToken string `json:"csrf_token"`
	}
	if ok , bind_err := bindBody( c , &body ); ok == false { return bind_err }
	if handlers.Guard.CheckCSRF( c , body.CSRFToken ) == false {
		err = forbidden( c , "invalid csrf token" )
		return
	}
	user := security.UserFrom( c )
	if delete_err := models.DeletePushSubscription( handlers.Store , user.ID , c.Params( "device_id" ) ); delete_err != nil {
		err = modelError( c , delete_err )
		return
	}
	err = c.JSON( fiber.Map{ "ok": true } )
	return
}

// canReceive is whether anything would arrive for this person.
func ( handlers *Handlers ) canReceive( user_id uint64 , prefs *models.UserPrefs ) ( result bool ) {
	result = handlers.Notify.Reachable( user_id , prefs )
	return
}
