package models

import (
	strings "strings"
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
//
// The exceptions change the text, so the card's revision moves too: the
// link it was looked up from (from) is swapped in its title and description
// for the product's cleaned URL, and a card whose title is nothing but the
// link takes the product's name as its title, the link moving to its
// description so it is not lost.
func SetCardProduct( store *db.Store , board_id uint64 , card_id uint64 , from string , product *Product , image *ProductImage ) ( card *Card , err error ) {
	err = store.Write( func( tx *db.Tx ) ( tx_err error ) {
		card , tx_err = getCardTx( tx , board_id , card_id )
		if tx_err != nil { return }
		cleanProductLink( card , from , product )
		nameAfterProduct( card , product )
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

// cleanProductLink puts the product's cleaned URL in place of the link it
// was looked up from, so the card's text matches what is stored and is not
// looked up again.
func cleanProductLink( card *Card , from string , product *Product ) {
	if product == nil || from == "" || from == product.URL { return }
	title := strings.ReplaceAll( card.Title , from , product.URL )
	description := strings.ReplaceAll( card.Description , from , product.URL )
	if title == card.Title && description == card.Description { return }
	card.Title , card.Description = title , description
	card.Rev += 1
	card.UpdatedAt = time.Now().UTC()
}

// nameAfterProduct retitles a card that is only its product's link.
func nameAfterProduct( card *Card , product *Product ) {
	if product == nil || product.Status != ProductOK { return }
	if strings.TrimSpace( card.Title ) != product.URL { return }
	title , ok := cleanTitle( product.Title , maxCardTitleChars )
	if ok == false { return }
	card.Title = title
	if strings.Contains( card.Description , product.URL ) == false {
		card.Description = strings.TrimSpace( product.URL + "\n\n" + card.Description )
	}
	card.Rev += 1
	card.UpdatedAt = time.Now().UTC()
}

func GetProductImage( store *db.Store , board_id uint64 , card_id uint64 ) ( image *ProductImage , err error ) {
	image = &ProductImage{}
	if err = store.Get( db.BucketProductImages , cardKey( board_id , card_id ) , image ); err != nil { image = nil }
	return
}
