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

// lookUpProduct reads link and stores what it found on the card; the link
// in the card's text is swapped for its clean form (see product.CleanURL),
// and a card titled with nothing but the link is renamed after the product.
func ( handlers *Handlers ) lookUpProduct( board_id uint64 , card_id uint64 , link string ) ( card *models.Card , err error ) {
	ctx , cancel := context.WithTimeout( context.Background() , 25 * time.Second )
	defer cancel()
	info , data , image_type , fetch_err := product.Fetch( ctx , link )
	found := &models.Product{ URL: product.CleanURL( link ) , Status: models.ProductOK , FetchedAt: time.Now().UTC() }
	var image *models.ProductImage
	if fetch_err != nil {
		found.Status = models.ProductFailed
	} else {
		found.URL , found.Title , found.Description , found.Price = info.URL , info.Title , info.Description , info.Price
		if data != nil { image = &models.ProductImage{ ContentType: image_type , Data: data } }
	}
	card , err = models.SetCardProduct( handlers.Store , board_id , card_id , link , found , image )
	return
}

// refreshProduct reads the card's link again now and answers with the card.
// Asked for by hand, so it reads any shop's link, shopping column or not.
//
// @route("POST /api/boards/{board_id}/cards/{card_id}/product/refresh", summary="Look a card's product up again",
//     tag="Cards", errors=(400, 403, 404),
//     description="Read the shop link in a card's title or description (or the one it was last looked up "
//         "from) again now, and store the product's name, price and photo on the card. Works for any shop's "
//         "link, shopping column or not. Answers with the card; product.status is failed when the page "
//         "could not be read.",
//     mcp="refresh_product",
// )
func ( handlers *Handlers ) refreshProduct( c fiber.Ctx , in *cardPath ) ( reply *fiber.Map , err error ) {
	principal , board , card , err := handlers.requireCardAt( c , in , access.Edit )
	if err != nil { return }
	link := product.FindURL( card.Title + "\n" + card.Description )
	if link == "" && card.Product != nil { link = card.Product.URL }
	if link == "" {
		err = refuse( fiber.StatusBadRequest , "product_no_link" )
		return
	}
	productSlots <- struct{}{}
	updated , lookup_err := handlers.lookUpProduct( board.ID , card.ID , link )
	<-productSlots
	if lookup_err != nil {
		err = modelErr( lookup_err )
		return
	}
	reply = mediaCardReply( principal , board , updated )
	return
}

// sharedRefreshProduct is refreshProduct for a share link's guest.
//
// @route("POST /api/shared/{board_id}/cards/{card_id}/product/refresh", summary="Look a card's product up again (share link)",
//     tag="Shared", access=PUBLIC, use=[handlers.SharePrincipal], errors=(400, 403, 404),
//     description="refreshProduct for a guest holding a share link in X-Share-Token.",
// )
func ( handlers *Handlers ) sharedRefreshProduct( c fiber.Ctx , in *cardPath ) ( reply *fiber.Map , err error ) {
	reply , err = handlers.refreshProduct( c , in )
	return
}

// removeProduct takes the product off a card; the link in its text stays.
//
// @route("POST /api/boards/{board_id}/cards/{card_id}/product/delete", summary="Take a card's product off",
//     tag="Cards", errors=(403, 404),
//     description="Remove the product (name, price, photo) looked up for a card. The link in its text stays. "
//         "Answers with the card.",
//     mcp="remove_product", destructive=True,
// )
func ( handlers *Handlers ) removeProduct( c fiber.Ctx , in *cardPath ) ( reply *fiber.Map , err error ) {
	principal , board , card , err := handlers.requireCardAt( c , in , access.Edit )
	if err != nil { return }
	updated , set_err := models.SetCardProduct( handlers.Store , board.ID , card.ID , "" , nil , nil )
	if set_err != nil {
		err = modelErr( set_err )
		return
	}
	reply = mediaCardReply( principal , board , updated )
	return
}

// sharedRemoveProduct is removeProduct for a share link's guest.
//
// @route("POST /api/shared/{board_id}/cards/{card_id}/product/delete", summary="Take a card's product off (share link)",
//     tag="Shared", access=PUBLIC, use=[handlers.SharePrincipal], errors=(403, 404),
//     description="removeProduct for a guest holding a share link in X-Share-Token.",
// )
func ( handlers *Handlers ) sharedRemoveProduct( c fiber.Ctx , in *cardPath ) ( reply *fiber.Map , err error ) {
	reply , err = handlers.removeProduct( c , in )
	return
}

// productImage serves a card's product photo to whoever may read the card.
//
// @route("GET /api/boards/{board_id}/cards/{card_id}/product-image", summary="A card's product photo",
//     tag="Cards", errors=(403, 404), reply_type="application/octet-stream",
//     description="The photo of the product looked up for a card, as the image file itself.",
//     mcp_http="A file (a picture, a spreadsheet or a backup), which travels over HTTP rather than as a tool answer.",
// )
func ( handlers *Handlers ) productImage( c fiber.Ctx , in *cardPath ) ( err error ) {
	_ , board , card , err := handlers.requireCardAt( c , in , access.View )
	if err != nil { return }
	image , get_err := models.GetProductImage( handlers.Store , board.ID , card.ID )
	if get_err != nil || image == nil || product.ImageType( image.Data ) == "" {
		err = refuse( fiber.StatusNotFound , "product_image_not_found" )
		return
	}
	err = sendImage( c , image )
	return
}

// sharedProductImage is productImage for a share link's guest.
//
// @route("GET /api/shared/{board_id}/cards/{card_id}/product-image", summary="A card's product photo (share link)",
//     tag="Shared", access=PUBLIC, use=[handlers.SharePrincipal], errors=(403, 404),
//     reply_type="application/octet-stream",
//     description="productImage for a guest holding a share link in X-Share-Token.",
// )
func ( handlers *Handlers ) sharedProductImage( c fiber.Ctx , in *cardPath ) ( err error ) {
	err = handlers.productImage( c , in )
	return
}
