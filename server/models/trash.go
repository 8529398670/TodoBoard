package models

import (
	errors "errors"
	sort "sort"
	time "time"

	db "todoboard/server/db"
)

// A board's trash. Deleting a card moves it out of BucketCards into
// BucketTrash under the same key, so it keeps its id, and every reader of
// cards -- the board, exports, print, reminders, MCP -- stops seeing it
// without having to know trash exists. Restoring moves it back. Cards stay
// for trashRetention, then go for good the next time the trash is read.
//
// Card-level shares, share links and subscriptions are removed on the way in
// (as a delete always did), so a restored card comes back unshared.

const trashRetention = 30 * 24 * time.Hour

var ErrNotInTrash = errors.New( "card not in trash" )

// TrashCard moves one card to its board's trash.
func TrashCard( store *db.Store , board_id uint64 , card_id uint64 , actor_name string ) ( err error ) {
	err = store.Write( func( tx *db.Tx ) ( tx_err error ) {
		card , tx_err := getCardTx( tx , board_id , card_id )
		if tx_err != nil { return }
		if tx_err = trashCardTx( tx , card , actor_name ); tx_err != nil { return }
		tx_err = touchBoardTx( tx , board_id )
		return
	} )
	return
}

func trashCardTx( tx *db.Tx , card *Card , actor_name string ) ( err error ) {
	if err = detachCardTx( tx , card ); err != nil { return }
	now := time.Now().UTC()
	card.SharedWith = map[uint64]string{}
	card.DeletedAt , card.DeletedBy = &now , actor_name
	card.Rev += 1
	if err = tx.Put( db.BucketTrash , cardKey( card.BoardID , card.ID ) , card ); err != nil { return }
	err = tx.Delete( db.BucketCards , cardKey( card.BoardID , card.ID ) )
	return
}

// ListTrash returns a board's trashed cards, most recently deleted first,
// dropping any older than trashRetention.
func ListTrash( store *db.Store , board_id uint64 ) ( cards []*Card , err error ) {
	cards = []*Card{}
	err = store.Write( func( tx *db.Tx ) ( tx_err error ) {
		cutoff := time.Now().UTC().Add( -trashRetention )
		expired := []*Card{}
		tx_err = tx.ForEachPrefix( db.BucketTrash , db.Key( board_id ) , func() any { return &Card{} } ,
			func( key []byte , item any ) bool {
				card := item.( *Card )
				normalizeCard( card )
				if card.DeletedAt != nil && card.DeletedAt.Before( cutoff ) {
					expired = append( expired , card )
				} else {
					cards = append( cards , card )
				}
				return true
			} )
		if tx_err != nil { return }
		for _ , card := range expired {
			if tx_err = tx.Delete( db.BucketTrash , cardKey( board_id , card.ID ) ); tx_err != nil { return }
		}
		return
	} )
	if err != nil {
		cards = nil
		return
	}
	sort.SliceStable( cards , func( i int , j int ) bool { return deletedAt( cards[ i ] ).After( deletedAt( cards[ j ] ) ) } )
	return
}

func deletedAt( card *Card ) ( result time.Time ) {
	if card.DeletedAt != nil { result = *card.DeletedAt }
	return
}

func getTrashedTx( tx *db.Tx , board_id uint64 , card_id uint64 ) ( card *Card , err error ) {
	card = &Card{}
	if err = tx.Get( db.BucketTrash , cardKey( board_id , card_id ) , card ); err != nil {
		card = nil
		if errors.Is( err , db.ErrNotFound ) { err = ErrNotInTrash }
		return
	}
	normalizeCard( card )
	return
}

// RestoreCard puts a trashed card back on its board, at the end of the list
// it was on -- or, when that list has since been deleted or archived, of the
// first open list.
func RestoreCard( store *db.Store , board_id uint64 , card_id uint64 , actor_name string ) ( card *Card , err error ) {
	err = store.Write( func( tx *db.Tx ) ( tx_err error ) {
		board , tx_err := getBoardTx( tx , board_id )
		if tx_err != nil { return }
		card , tx_err = getTrashedTx( tx , board_id , card_id )
		if tx_err != nil { return }
		if list := board.List( card.ListID ); list == nil || list.Archived {
			open := []*List{}
			for _ , candidate := range board.SortedLists() {
				if candidate.Archived == false { open = append( open , candidate ) }
			}
			if len( open ) == 0 { return ErrListNotFound }
			card.ListID = open[ 0 ].ID
		}
		card.Position , tx_err = endOfListTx( tx , board_id , card.ListID )
		if tx_err != nil { return }
		card.DeletedAt , card.DeletedBy = nil , ""
		if tx_err = tx.Delete( db.BucketTrash , cardKey( board_id , card_id ) ); tx_err != nil { return }
		tx_err = saveCardTx( tx , board , card , actor_name )
		return
	} )
	if err != nil { card = nil }
	return
}

// PurgeCard deletes one trashed card for good.
func PurgeCard( store *db.Store , board_id uint64 , card_id uint64 ) ( err error ) {
	err = store.Write( func( tx *db.Tx ) ( tx_err error ) {
		if _ , tx_err = getTrashedTx( tx , board_id , card_id ); tx_err != nil { return }
		tx_err = tx.Delete( db.BucketTrash , cardKey( board_id , card_id ) )
		return
	} )
	return
}

// EmptyTrash deletes every trashed card of a board for good.
func EmptyTrash( store *db.Store , board_id uint64 ) ( removed int , err error ) {
	err = store.Write( func( tx *db.Tx ) ( tx_err error ) {
		removed , tx_err = tx.DeletePrefix( db.BucketTrash , db.Key( board_id ) )
		return
	} )
	return
}
