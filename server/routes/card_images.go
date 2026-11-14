package routes

import (
	fiber "github.com/gofiber/fiber/v3"

	access "todoboard/server/access"
	models "todoboard/server/models"
	product "todoboard/server/product"
)

// A picture pasted onto a card. The browser shrinks it first and sends it
// in the JSON body, base64, like every other write.
type cardImageRequest struct {
	cardPath
	Data []byte `json:"data" jsonschema_description:"The picture's bytes, base64: a PNG, JPEG, GIF or WebP."`
}

// mediaCardReply is the card as the caller sees it, as a route's reply.
func mediaCardReply( principal *access.Principal , board *models.Board , card *models.Card ) ( reply *fiber.Map ) {
	payload := cardPayload( card , access.Card( principal , board , card ) , access.Board( principal , board ) )
	reply = &payload
	return
}

// setCardImage stores the picture and answers with the card.
//
// @route("POST /api/boards/{board_id}/cards/{card_id}/image", summary="Put a picture on a card",
//     tag="Cards", errors=(400, 403, 404),
//     description="Store a picture on a card (base64 in data), replacing any it had, and answer with the card.",
//     mcp_http="A file (a picture, a spreadsheet or a backup), which travels over HTTP rather than as a tool answer.",
// )
func ( handlers *Handlers ) setCardImage( c fiber.Ctx , in *cardImageRequest ) ( reply *fiber.Map , err error ) {
	principal , board , card , err := handlers.requireCardAt( c , &in.cardPath , access.Edit )
	if err != nil { return }
	kind := product.ImageType( in.Data )
	if kind == "" || len( in.Data ) > models.MaxCardImageBytes {
		err = refuse( fiber.StatusBadRequest , "card_image_invalid" )
		return
	}
	updated , set_err := models.SetCardImage( handlers.Store , board.ID , card.ID , &models.ProductImage{ ContentType: kind , Data: in.Data } )
	if set_err != nil {
		err = modelErr( set_err )
		return
	}
	handlers.activity( principal , board.ID , "card_image" , map[string]string{ "card": card.Title } )
	reply = mediaCardReply( principal , board , updated )
	return
}

// sharedSetCardImage is setCardImage for a share link's guest.
//
// @route("POST /api/shared/{board_id}/cards/{card_id}/image", summary="Put a picture on a card (share link)",
//     tag="Shared", access=PUBLIC, use=[handlers.SharePrincipal], errors=(400, 403, 404),
//     description="setCardImage for a guest holding a share link in X-Share-Token.",
// )
func ( handlers *Handlers ) sharedSetCardImage( c fiber.Ctx , in *cardImageRequest ) ( reply *fiber.Map , err error ) {
	reply , err = handlers.setCardImage( c , in )
	return
}

// removeCardImage takes the picture off a card.
//
// @route("POST /api/boards/{board_id}/cards/{card_id}/image/delete", summary="Take a card's picture off",
//     tag="Cards", errors=(403, 404),
//     description="Remove the picture from a card and answer with the card. The card itself is not touched.",
//     mcp="remove_card_image", destructive=True,
// )
func ( handlers *Handlers ) removeCardImage( c fiber.Ctx , in *cardPath ) ( reply *fiber.Map , err error ) {
	principal , board , card , err := handlers.requireCardAt( c , in , access.Edit )
	if err != nil { return }
	updated , set_err := models.SetCardImage( handlers.Store , board.ID , card.ID , nil )
	if set_err != nil {
		err = modelErr( set_err )
		return
	}
	handlers.activity( principal , board.ID , "card_image" , map[string]string{ "card": card.Title } )
	reply = mediaCardReply( principal , board , updated )
	return
}

// sharedRemoveCardImage is removeCardImage for a share link's guest.
//
// @route("POST /api/shared/{board_id}/cards/{card_id}/image/delete", summary="Take a card's picture off (share link)",
//     tag="Shared", access=PUBLIC, use=[handlers.SharePrincipal], errors=(403, 404),
//     description="removeCardImage for a guest holding a share link in X-Share-Token.",
// )
func ( handlers *Handlers ) sharedRemoveCardImage( c fiber.Ctx , in *cardPath ) ( reply *fiber.Map , err error ) {
	reply , err = handlers.removeCardImage( c , in )
	return
}

// cardImage serves a card's picture to whoever may read the card.
//
// @route("GET /api/boards/{board_id}/cards/{card_id}/image", summary="A card's picture",
//     tag="Cards", errors=(403, 404), reply_type="application/octet-stream",
//     description="The picture on a card, as the image file itself.",
//     mcp_http="A file (a picture, a spreadsheet or a backup), which travels over HTTP rather than as a tool answer.",
// )
func ( handlers *Handlers ) cardImage( c fiber.Ctx , in *cardPath ) ( err error ) {
	_ , board , card , err := handlers.requireCardAt( c , in , access.View )
	if err != nil { return }
	image , get_err := models.GetCardImage( handlers.Store , board.ID , card.ID )
	if get_err != nil || image == nil || product.ImageType( image.Data ) == "" {
		err = refuse( fiber.StatusNotFound , "card_image_not_found" )
		return
	}
	err = sendImage( c , image )
	return
}

// sharedCardImage is cardImage for a share link's guest.
//
// @route("GET /api/shared/{board_id}/cards/{card_id}/image", summary="A card's picture (share link)",
//     tag="Shared", access=PUBLIC, use=[handlers.SharePrincipal], errors=(403, 404),
//     reply_type="application/octet-stream",
//     description="cardImage for a guest holding a share link in X-Share-Token.",
// )
func ( handlers *Handlers ) sharedCardImage( c fiber.Ctx , in *cardPath ) ( err error ) {
	err = handlers.cardImage( c , in )
	return
}

// sendImage answers with a stored picture, cached privately for an hour.
func sendImage( c fiber.Ctx , image *models.ProductImage ) ( err error ) {
	c.Set( fiber.HeaderContentType , image.ContentType )
	c.Set( fiber.HeaderCacheControl , "private, max-age=3600" )
	c.Set( "X-Content-Type-Options" , "nosniff" )
	err = c.Send( image.Data )
	return
}
