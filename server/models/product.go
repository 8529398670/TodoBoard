package models

import (
	time "time"

	db "todoboard/server/db"
)

// Product is what a shop's page said about the thing a card links to:
// filled in by the server after the link appears (see routes/products.go).
// The photo is kept in BucketProductImages, served by the app itself, so a
// board never loads anything from a shop's servers.
type Product struct {
	URL         string    `json:"url"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Price       string    `json:"price"`
	HasImage    bool      `json:"has_image"`
	// "ok", or "failed" when the page could not be read (often a robot
	// check); the link is still kept.
	Status      string    `json:"status"`
	FetchedAt   time.Time `json:"fetched_at"`
}

const (
	ProductOK     = "ok"
	ProductFailed = "failed"
)

// ProductImage is a card's product photo as stored.
type ProductImage struct {
	ContentType string `json:"content_type"`
	Data        []byte `json:"data"`
}

// SetCardProduct stores what was found for a card's link (nil clears it)
// and its photo (nil removes it). It moves the board's revision so open
// pages pick it up, but not the card's: the card's text did not change, so
// someone typing in it must not be told it was edited under them.
func SetCardProduct( store *db.Store , board_id uint64 , card_id uint64 , product *Product , image *ProductImage ) ( card *Card , err error ) {
	err = store.Write( func( tx *db.Tx ) ( tx_err error ) {
		card , tx_err = getCardTx( tx , board_id , card_id )
		if tx_err != nil { return }
		key := cardKey( board_id , card_id )
		if image != nil && product != nil {
			product.HasImage = true
			tx_err = tx.Put( db.BucketProductImages , key , image )
		} else {
			if product != nil { product.HasImage = false }
			tx_err = tx.Delete( db.BucketProductImages , key )
		}
		if tx_err != nil { return }
		card.Product = product
		if tx_err = tx.Put( db.BucketCards , key , card ); tx_err != nil { return }
		tx_err = touchBoardTx( tx , board_id )
		return
	} )
	if err != nil { card = nil }
	return
}

func GetProductImage( store *db.Store , board_id uint64 , card_id uint64 ) ( image *ProductImage , err error ) {
	image = &ProductImage{}
	if err = store.Get( db.BucketProductImages , cardKey( board_id , card_id ) , image ); err != nil { image = nil }
	return
}
