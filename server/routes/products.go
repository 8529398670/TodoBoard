package routes

import (
	context "context"
	time "time"

	fiber "github.com/gofiber/fiber/v3"

	access "todoboard/server/access"
	models "todoboard/server/models"
	product "todoboard/server/product"
)

// At most this many shop pages are being read at once; more wait their turn.
var productSlots = make( chan struct{} , 3 )

// productLink is the link a card's product should come from, or "": the
// first link in its title or body, if it is Amazon's or the card sits on a
// shopping board or column -- and it is not the link already looked up.
func productLink( board *models.Board , card *models.Card ) ( link string ) {
	found := product.FindURL( card.Title + "\n" + card.Description )
	if found == "" { return }
	if card.Product != nil && card.Product.URL == found { return }
	shopping := board.Shopping
	if list := board.List( card.ListID ); list != nil && list.Shopping { shopping = true }
	if shopping || product.IsAmazon( found ) { link = found }
	return
}

// lookUpProductLater reads a card's product link in the background, after
// a create or edit has been answered; open pages see the result on their
// next poll.
func ( handlers *Handlers ) lookUpProductLater( board *models.Board , card *models.Card ) {
	if board == nil || card == nil { return }
	link := productLink( board , card )
	if link == "" { return }
	board_id , card_id := board.ID , card.ID
	go func() {
		productSlots <- struct{}{}
		defer func() { <-productSlots }()
		handlers.lookUpProduct( board_id , card_id , link )
	}()
}

func ( handlers *Handlers ) lookUpProduct( board_id uint64 , card_id uint64 , link string ) ( card *models.Card , err error ) {
	ctx , cancel := context.WithTimeout( context.Background() , 25 * time.Second )
	defer cancel()
	info , data , image_type , fetch_err := product.Fetch( ctx , link )
	found := &models.Product{ URL: link , Status: models.ProductOK , FetchedAt: time.Now().UTC() }
	var image *models.ProductImage
	if fetch_err != nil {
		found.Status = models.ProductFailed
	} else {
		found.Title , found.Description , found.Price = info.Title , info.Description , info.Price
		if data != nil { image = &models.ProductImage{ ContentType: image_type , Data: data } }
	}
	card , err = models.SetCardProduct( handlers.Store , board_id , card_id , found , image )
	return
}

// RefreshProduct reads the card's link again now and answers with the card.
// Asked for by hand, so it reads any shop's link, shopping column or not.
func ( handlers *Handlers ) RefreshProduct( c fiber.Ctx ) ( err error ) {
	principal , board , card , err := handlers.requireCard( c , access.Edit )
	if card == nil { return }
	link := product.FindURL( card.Title + "\n" + card.Description )
	if link == "" && card.Product != nil { link = card.Product.URL }
	if link == "" {
		err = badRequest( c , "product_no_link" )
		return
	}
	productSlots <- struct{}{}
	updated , lookup_err := handlers.lookUpProduct( board.ID , card.ID , link )
	<-productSlots
	if lookup_err != nil {
		err = modelError( c , lookup_err )
		return
	}
	err = handlers.cardReply( c , principal , board , updated , nil )
	return
}

// RemoveProduct takes the product off a card; the link in its text stays.
func ( handlers *Handlers ) RemoveProduct( c fiber.Ctx ) ( err error ) {
	principal , board , card , err := handlers.requireCard( c , access.Edit )
	if card == nil { return }
	updated , set_err := models.SetCardProduct( handlers.Store , board.ID , card.ID , nil , nil )
	if set_err != nil {
		err = modelError( c , set_err )
		return
	}
	err = handlers.cardReply( c , principal , board , updated , nil )
	return
}

// ProductImage serves a card's product photo to whoever may read the card.
func ( handlers *Handlers ) ProductImage( c fiber.Ctx ) ( err error ) {
	_ , board , card , err := handlers.requireCard( c , access.View )
	if card == nil { return }
	image , get_err := models.GetProductImage( handlers.Store , board.ID , card.ID )
	if get_err != nil || image == nil || product.ImageType( image.Data ) == "" {
		err = notFound( c , "product_image_not_found" )
		return
	}
	c.Set( fiber.HeaderContentType , image.ContentType )
	c.Set( fiber.HeaderCacheControl , "private, max-age=3600" )
	c.Set( "X-Content-Type-Options" , "nosniff" )
	err = c.Send( image.Data )
	return
}

