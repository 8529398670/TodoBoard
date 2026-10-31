package routes

import (
	fiber "github.com/gofiber/fiber/v3"

	access "todoboard/server/access"
	models "todoboard/server/models"
	product "todoboard/server/product"
)

// A picture pasted onto a card. The browser shrinks it first and sends it
// in the JSON body, base64, like every other write (boardFor has already
// checked its CSRF token).
type cardImageRequest struct {
	Data []byte `json:"data"`
}

// SetCardImage stores the picture and answers with the card.
func ( handlers *Handlers ) SetCardImage( c fiber.Ctx ) ( err error ) {
	principal , board , card , err := handlers.requireCard( c , access.Edit )
	if card == nil { return }
	var body cardImageRequest
	if ok , bind_err := bindBody( c , &body ); ok == false { return bind_err }
	kind := product.ImageType( body.Data )
	if kind == "" || len( body.Data ) > models.MaxCardImageBytes {
		err = badRequest( c , "card_image_invalid" )
		return
	}
	updated , set_err := models.SetCardImage( handlers.Store , board.ID , card.ID , &models.ProductImage{ ContentType: kind , Data: body.Data } )
	if set_err != nil {
		err = modelError( c , set_err )
		return
	}
	err = handlers.cardReply( c , principal , board , updated , nil )
	return
}

// RemoveCardImage takes the picture off a card.
func ( handlers *Handlers ) RemoveCardImage( c fiber.Ctx ) ( err error ) {
	principal , board , card , err := handlers.requireCard( c , access.Edit )
	if card == nil { return }
	updated , set_err := models.SetCardImage( handlers.Store , board.ID , card.ID , nil )
	if set_err != nil {
		err = modelError( c , set_err )
		return
	}
	err = handlers.cardReply( c , principal , board , updated , nil )
	return
}

// CardImage serves a card's picture to whoever may read the card.
func ( handlers *Handlers ) CardImage( c fiber.Ctx ) ( err error ) {
	_ , board , card , err := handlers.requireCard( c , access.View )
	if card == nil { return }
	image , get_err := models.GetCardImage( handlers.Store , board.ID , card.ID )
	if get_err != nil || image == nil || product.ImageType( image.Data ) == "" {
		err = notFound( c , "card_image_not_found" )
		return
	}
	c.Set( fiber.HeaderContentType , image.ContentType )
	c.Set( fiber.HeaderCacheControl , "private, max-age=3600" )
	c.Set( "X-Content-Type-Options" , "nosniff" )
	err = c.Send( image.Data )
	return
}
