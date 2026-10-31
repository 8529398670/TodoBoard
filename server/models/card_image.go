package models

import (
	time "time"

	db "todoboard/server/db"
)

// MaxCardImageBytes caps a pasted picture. The browser shrinks it before
// sending, so a real one is far smaller; the cap stops a raw upload.
const MaxCardImageBytes = 512 * 1024

// SetCardImage stores the picture pasted onto a card (nil removes it). Like
// SetCardProduct it moves the board's revision but not the card's, so
// someone typing in the card is not told it was edited under them.
func SetCardImage( store *db.Store , board_id uint64 , card_id uint64 , image *ProductImage ) ( card *Card , err error ) {
	if image != nil && len( image.Data ) > MaxCardImageBytes { return nil , ErrBadInput }
	err = store.Write( func( tx *db.Tx ) ( tx_err error ) {
		card , tx_err = getCardTx( tx , board_id , card_id )
		if tx_err != nil { return }
		key := cardKey( board_id , card_id )
		if image != nil {
			now := time.Now().UTC()
			card.ImageAt = &now
			tx_err = tx.Put( db.BucketCardImages , key , image )
		} else {
			card.ImageAt = nil
			tx_err = tx.Delete( db.BucketCardImages , key )
		}
		if tx_err != nil { return }
		if tx_err = tx.Put( db.BucketCards , key , card ); tx_err != nil { return }
		tx_err = touchBoardTx( tx , board_id )
		return
	} )
	if err != nil { card = nil }
	return
}

func GetCardImage( store *db.Store , board_id uint64 , card_id uint64 ) ( image *ProductImage , err error ) {
	image = &ProductImage{}
	if err = store.Get( db.BucketCardImages , cardKey( board_id , card_id ) , image ); err != nil { image = nil }
	return
}
