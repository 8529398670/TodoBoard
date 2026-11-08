package models

import (
	sort "sort"
	time "time"

	db "todoboard/server/db"
	encryption "todoboard/server/encryption"
)

// Every card has three identities, each for a different reader. Card.ID is
// the key: global, in share links and API paths, never shown. Card.UID is a
// ULID, unique across servers: the card's public ID, the last part of its
// link /b/<board>/<column>/<ULID> (see CardPath), and its "Card ID" in an
// export. Card.Serial is 1, 2, 3 per board, kept for life and never reused,
// but no longer shown anywhere. The number a card shows on the board is
// none of these: that is its place in the column (see Numbering).

// crockford is the ULID alphabet: no I, L, O or U.
const crockford = "0123456789ABCDEFGHJKMNPQRSTVWXYZ"

// newULID is 48 bits of milliseconds then 80 random bits, as 26 characters
// that sort by time.
func newULID( at time.Time ) ( result string ) {
	raw := make( []byte , 16 )
	milliseconds := uint64( at.UnixMilli() )
	for index := 5; index >= 0; index -= 1 {
		raw[ index ] = byte( milliseconds )
		milliseconds >>= 8
	}
	copy( raw[ 6: ] , encryption.GenerateRandomBytes( 10 ) )
	// 26 characters hold 130 bits, so two zero bits come before the 128.
	text := make( []byte , 26 )
	for index := range text {
		value := 0
		for bit := index * 5 - 2; bit < index * 5 + 3; bit += 1 {
			value <<= 1
			if bit >= 0 && raw[ bit / 8 ] & ( 0x80 >> ( bit % 8 ) ) != 0 { value |= 1 }
		}
		text[ index ] = crockford[ value ]
	}
	result = string( text )
	return
}

// stampCard gives a new card the board's next number and a ULID from at. The
// caller saves the board with it.
func ( board *Board ) stampCard( card *Card , at time.Time ) {
	board.NextCardSerial += 1
	card.Serial = board.NextCardSerial
	card.UID = newULID( at )
}

// EnsureCardIdentity numbers the cards of every board written before cards
// had a Serial and a UID, oldest first, trash included so a restored card has
// its number too. Run at startup; a board that has numbered a card is done.
func EnsureCardIdentity( store *db.Store ) ( err error ) {
	pending := []uint64{}
	err = store.ForEach( db.BucketBoards , func() any { return &Board{} } , func( key []byte , item any ) bool {
		if board := item.( *Board ); board.NextCardSerial == 0 { pending = append( pending , board.ID ) }
		return true
	} )
	if err != nil { return }
	for _ , board_id := range pending {
		if err = stampBoardCards( store , board_id ); err != nil { return }
	}
	return
}

func stampBoardCards( store *db.Store , board_id uint64 ) ( err error ) {
	type stored struct {
		bucket string
		card   *Card
	}
	err = store.Write( func( tx *db.Tx ) ( tx_err error ) {
		board , tx_err := getBoardTx( tx , board_id )
		if tx_err != nil { return }
		found := []stored{}
		for _ , bucket := range []string{ db.BucketCards , db.BucketTrash } {
			tx_err = tx.ForEachPrefix( bucket , db.Key( board_id ) , func() any { return &Card{} } ,
				func( key []byte , item any ) bool {
					found = append( found , stored{ bucket , item.( *Card ) } )
					return true
				} )
			if tx_err != nil { return }
		}
		if len( found ) == 0 { return }
		sort.Slice( found , func( i int , j int ) bool {
			a , b := found[ i ].card , found[ j ].card
			if a.CreatedAt.Equal( b.CreatedAt ) { return a.ID < b.ID }
			return a.CreatedAt.Before( b.CreatedAt )
		} )
		// Written directly, not through putBoardTx or eachCardTx: numbering
		// is not an edit anyone made, so it should not bump Rev.
		for _ , entry := range found {
			board.stampCard( entry.card , entry.card.CreatedAt )
			if tx_err = tx.Put( entry.bucket , cardKey( board_id , entry.card.ID ) , entry.card ); tx_err != nil { return }
		}
		tx_err = tx.Put( db.BucketBoards , boardKey( board_id ) , board )
		return
	} )
	return
}
