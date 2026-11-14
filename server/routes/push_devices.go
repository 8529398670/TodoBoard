package routes

import (
	errors "errors"

	fiber "github.com/gofiber/fiber/v3"

	apidoc "todoboard/server/apidoc"
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
	Label     string `json:"label,omitempty"`
}

type pushDevicePath struct {
	DeviceID string `json:"device_id" path:"device_id" jsonschema:"minLength=1"`
}

type pushDevicesReply struct {
	PublicKey string      `json:"public_key"`
	Devices   []fiber.Map `json:"devices"`
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

// listPushDevices is the caller's browsers that take notifications.
//
// @route("GET /api/notify/devices", summary="Your browsers that take notifications",
//     tag="Notifications", access=SESSION_ONLY,
//     description="The browsers your notifications go to (id, label, when added and last reached), and "
//         "the server's web push public key (empty when web push is off).",
// )
func ( handlers *Handlers ) listPushDevices( c fiber.Ctx , in *apidoc.None ) ( reply *pushDevicesReply , err error ) {
	user := security.UserFrom( c )
	subscriptions , list_err := models.ListPushSubscriptions( handlers.Store , user.ID )
	if list_err != nil {
		err = internalFailure( list_err )
		return
	}
	devices := []fiber.Map{}
	for _ , subscription := range subscriptions { devices = append( devices , devicePayload( subscription ) ) }
	reply = &pushDevicesReply{ PublicKey: handlers.Notify.WebPushKey() , Devices: devices }
	return
}

// savePushDevice is also how a browser checks in: the settings page sends its
// subscription each time it opens, so one the browser replaced is caught.
//
// @route("POST /api/notify/devices", summary="Add or check in a browser for notifications",
//     tag="Notifications", access=SESSION_ONLY, errors=(400,),
//     description="Save this browser's web push subscription (endpoint and keys) so notifications reach "
//         "it, or check it in again. push_disabled: web push is off; push_endpoint_refused: not a push "
//         "service the server sends to; push_too_many_devices: remove one first.",
// )
func ( handlers *Handlers ) savePushDevice( c fiber.Ctx , in *pushDeviceRequest ) ( reply *fiber.Map , err error ) {
	if handlers.Notify.WebPushKey() == "" {
		err = refuse( fiber.StatusBadRequest , "push_disabled" )
		return
	}
	if notify.EndpointAllowed( in.Endpoint ) == false {
		err = refuse( fiber.StatusBadRequest , "push_endpoint_refused" )
		return
	}
	user := security.UserFrom( c )
	subscription , save_err := models.SavePushSubscription( handlers.Store , user.ID , in.Endpoint , in.P256dh , in.Auth , in.Label )
	if errors.Is( save_err , models.ErrTooManyDevices ) {
		err = refuse( fiber.StatusBadRequest , "push_too_many_devices" )
		return
	}
	if save_err != nil {
		err = modelErr( save_err )
		return
	}
	payload := devicePayload( subscription )
	reply = &payload
	return
}

// deletePushDevice stops notifications going to one of the caller's browsers.
//
// @route("POST /api/notify/devices/{device_id}/delete", summary="Remove a browser from notifications",
//     tag="Notifications", access=SESSION_ONLY, errors=(404,),
//     description="Stop notifications going to one of your browsers.",
// )
func ( handlers *Handlers ) deletePushDevice( c fiber.Ctx , in *pushDevicePath ) ( reply *okReply , err error ) {
	user := security.UserFrom( c )
	if delete_err := models.DeletePushSubscription( handlers.Store , user.ID , in.DeviceID ); delete_err != nil {
		err = modelErr( delete_err )
		return
	}
	reply = &okReply{ Ok: true }
	return
}

// canReceive is whether anything would arrive for this person.
func ( handlers *Handlers ) canReceive( user_id uint64 , prefs *models.UserPrefs ) ( result bool ) {
	result = handlers.Notify.Reachable( user_id , prefs )
	return
}
