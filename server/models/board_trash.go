package models

import (
	errors "errors"
	sort "sort"
	time "time"

	db "todoboard/server/db"
	encryption "todoboard/server/encryption"
)

// Archiving and the board trash.
//
// Archiving comes in two kinds. The owner archives a board for everyone
// (Board.ArchivedAt): it leaves every member's home grid and opens read-only
// until it is unarchived. Anyone on a board archives it for themselves
// (BoardRef.Archived): it leaves their own home grid and nothing else
// changes.
//
// Deleting a board marks it trashed (Board.DeletedAt) rather than removing
// it. GetBoard then reads it as not found, so every reader -- the board,
// share links, the home page, backups, MCP -- stops seeing it, while its
// cards, members, slug and links stay put for a restore. Only the owner sees
// it, in their board trash, and it stays there until they restore it or
// delete it for good: the board trash never empties itself.

var ErrBoardNotInTrash = errors.New( "board not in trash" )

// SetBoardArchived archives (or unarchives) a board for everyone.
func SetBoardArchived( store *db.Store , board_id uint64 , archived bool , actor_name string ) ( board *Board , err error ) {
	board , err = MutateBoard( store , board_id , func( board *Board ) error {
		if archived {
			if board.ArchivedAt == nil {
				now := time.Now().UTC()
				board.ArchivedAt , board.ArchivedBy = &now , actor_name
			}
		} else {
			board.ArchivedAt , board.ArchivedBy = nil , ""
		}
		return nil
	} )
	return
}

// SetBoardArchivedFor archives (or unarchives) a board on one person's own
// home page.
func SetBoardArchivedFor( store *db.Store , user_id uint64 , board_id uint64 , archived bool ) ( err error ) {
	err = updateBoardRef( store , user_id , board_id , func( ref *BoardRef ) { ref.Archived = archived } )
	return
}

// TrashBoard moves a board to its owner's trash.
func TrashBoard( store *db.Store , board_id uint64 , actor_id uint64 ) ( err error ) {
	_ , err = MutateBoard( store , board_id , func( board *Board ) error {
		now := time.Now().UTC()
		board.DeletedAt , board.DeletedBy = &now , actor_id
		return nil
	} )
	return
}

// trashedBoardTx is a board in user_id's trash: one they own that is
// trashed. Anything else is ErrBoardNotInTrash.
func trashedBoardTx( tx *db.Tx , user_id uint64 , board_id uint64 ) ( board *Board , err error ) {
	board , err = getBoardAnyTx( tx , board_id )
	if err == nil && ( board.Trashed() == false || board.OwnerID != user_id ) { err = ErrBoardNotInTrash }
	if err != nil {
		board = nil
		if errors.Is( err , ErrBoardNotFound ) { err = ErrBoardNotInTrash }
	}
	return
}

// RestoreBoard takes a board out of user_id's trash, back as it was.
func RestoreBoard( store *db.Store , user_id uint64 , board_id uint64 ) ( board *Board , err error ) {
	err = store.Write( func( tx *db.Tx ) ( tx_err error ) {
		board , tx_err = trashedBoardTx( tx , user_id , board_id )
		if tx_err != nil { return }
		board.DeletedAt , board.DeletedBy = nil , 0
		tx_err = putBoardTx( tx , board )
		return
	} )
	if err != nil { board = nil }
	return
}

// TrashedBoard is one row of the board trash.
type TrashedBoard struct {
	ID        uint64    `json:"id"`
	Title     string    `json:"title"`
	Color     string    `json:"color"`
	DeletedAt time.Time `json:"deleted_at"`
}

// ListTrashedBoards is user_id's board trash, most recently deleted first.
func ListTrashedBoards( store *db.Store , user_id uint64 ) ( boards []*TrashedBoard , err error ) {
	boards = []*TrashedBoard{}
	ids := []uint64{}
	err = store.ForEachPrefix( db.BucketUserBoards , db.Key( user_id ) ,
		func() any { return &BoardRef{} } ,
		func( key []byte , item any ) bool {
			if item.( *BoardRef ).Role == BoardRoleOwner { ids = append( ids , encryption.BytesToUint64( key[ 8:16 ] ) ) }
			return true
		} )
	if err != nil { return }
	for _ , board_id := range ids {
		board , get_err := getBoardAny( store , board_id )
		if get_err != nil || board.Trashed() == false || board.OwnerID != user_id { continue }
		boards = append( boards , &TrashedBoard{ ID: board.ID , Title: board.Title , Color: board.Color , DeletedAt: *board.DeletedAt } )
	}
	sort.SliceStable( boards , func( i int , j int ) bool { return boards[ i ].DeletedAt.After( boards[ j ].DeletedAt ) } )
	return
}

// PurgeTrashedBoard deletes a board in user_id's trash for good.
func PurgeTrashedBoard( store *db.Store , user_id uint64 , board_id uint64 ) ( err error ) {
	err = store.Write( func( tx *db.Tx ) ( tx_err error ) {
		if _ , tx_err = trashedBoardTx( tx , user_id , board_id ); tx_err != nil { return }
		tx_err = deleteBoardTx( tx , board_id )
		return
	} )
	return
}

// EmptyBoardTrash deletes every board in user_id's trash for good.
func EmptyBoardTrash( store *db.Store , user_id uint64 ) ( removed int , err error ) {
	trashed , err := ListTrashedBoards( store , user_id )
	if err != nil { return }
	err = store.Write( func( tx *db.Tx ) ( tx_err error ) {
		for _ , row := range trashed {
			// Restored since it was listed: not ours to delete any more.
			if _ , gone := trashedBoardTx( tx , user_id , row.ID ); gone != nil { continue }
			if tx_err = deleteBoardTx( tx , row.ID ); tx_err != nil { return }
			removed += 1
		}
		return
	} )
	if err != nil { removed = 0 }
	return
}

// PurgeBoard removes a board and everything hanging off it at once, trashed
// or not: for undoing a half-made copy, never for a person's delete.
func PurgeBoard( store *db.Store , board_id uint64 ) ( err error ) {
	err = store.Write( func( tx *db.Tx ) ( tx_err error ) {
		tx_err = deleteBoardTx( tx , board_id )
		return
	} )
	return
}
