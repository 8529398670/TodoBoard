package models

import (
	errors "errors"
	sort "sort"
	strings "strings"
	time "time"
	utf8 "unicode/utf8"

	db "todoboard/server/db"
	encryption "todoboard/server/encryption"
)

// Comments are a card's conversation, kept apart from the card in
// BucketComments under boardID|cardID|commentID, so a long thread never
// makes the card itself heavy and posting one never trips someone's
// description edit (the card's rev does not move; the board's does, so open
// boards see it). Who may read, post, edit and remove them is decided in
// server/access and the routes; this file only keeps them.

var ErrCommentNotFound = errors.New( "comment not found" )

const MaxCommentLength = 4000

// Comment.By is who wrote it, the same way Card.UpdatedBy reads: a member's
// display name, "guest:Sam" for a named guest, or "guest:" for one who
// stayed anonymous. AuthorID is the member's user id, 0 for a guest; it is
// what decides who may edit, since names change.
type Comment struct {
	ID        uint64     `json:"id"`
	BoardID   uint64     `json:"board_id"`
	CardID    uint64     `json:"card_id"`
	AuthorID  uint64     `json:"author_id"`
	By        string     `json:"by"`
	Text      string     `json:"text"`
	CreatedAt time.Time  `json:"created_at"`
	EditedAt  *time.Time `json:"edited_at"`
	// Thread is shared by a comment's copies on mirrored cards, so an edit
	// or removal finds them (see card_link.go). "" on older comments.
	Thread    string     `json:"thread,omitempty"`
}

func commentKey( board_id uint64 , card_id uint64 , comment_id uint64 ) ( key []byte ) {
	key = db.Key( board_id , card_id , comment_id )
	return
}

// cleanComment trims the text and refuses an empty or over-long one.
func cleanComment( text string ) ( result string , err error ) {
	result = strings.TrimSpace( text )
	if result == "" || utf8.RuneCountInString( result ) > MaxCommentLength { err = ErrBadInput }
	return
}

// ListComments is a card's comments, oldest first.
func ListComments( store *db.Store , board_id uint64 , card_id uint64 ) ( comments []*Comment , err error ) {
	comments = []*Comment{}
	err = store.ForEachPrefix( db.BucketComments , db.Key( board_id , card_id ) , func() any { return &Comment{} } , func( key []byte , item any ) bool {
		comments = append( comments , item.( *Comment ) )
		return true
	} )
	sort.SliceStable( comments , func( a , b int ) bool { return comments[ a ].ID < comments[ b ].ID } )
	return
}

// CommentCounts is how many comments each card on a board has, read in one
// pass for the board payload.
func CommentCounts( store *db.Store , board_id uint64 ) ( counts map[uint64]int , err error ) {
	counts = map[uint64]int{}
	err = store.ForEachPrefix( db.BucketComments , db.Key( board_id ) , func() any { return &Comment{} } , func( key []byte , item any ) bool {
		counts[ encryption.BytesToUint64( key[ 8:16 ] ) ] += 1
		return true
	} )
	return
}

func GetComment( store *db.Store , board_id uint64 , card_id uint64 , comment_id uint64 ) ( comment *Comment , err error ) {
	comment = &Comment{}
	if err = store.Get( db.BucketComments , commentKey( board_id , card_id , comment_id ) , comment ); err != nil {
		comment = nil
		err = ErrCommentNotFound
	}
	return
}

// AddComment posts a comment on a card that exists and is not in the trash.
func AddComment( store *db.Store , board_id uint64 , card_id uint64 , author_id uint64 , by string , text string ) ( comment *Comment , err error ) {
	clean , err := cleanComment( text )
	if err != nil { return }
	err = store.Write( func( tx *db.Tx ) ( tx_err error ) {
		card := &Card{}
		if tx.Get( db.BucketCards , cardKey( board_id , card_id ) , card ) != nil { return ErrCardNotFound }
		id , tx_err := tx.NextSequence( db.BucketComments )
		if tx_err != nil { return }
		now := time.Now().UTC()
		comment = &Comment{ ID: id , BoardID: board_id , CardID: card_id , AuthorID: author_id , By: by , Text: clean , CreatedAt: now , Thread: newULID( now ) }
		if tx_err = tx.Put( db.BucketComments , commentKey( board_id , card_id , id ) , comment ); tx_err != nil { return }
		if tx_err = touchBoardTx( tx , board_id ); tx_err != nil { return }
		tx_err = mirrorCommentTx( tx , board_id , card_id , comment , false )
		return
	} )
	if err != nil { comment = nil }
	return
}

// EditComment replaces a comment's text.
func EditComment( store *db.Store , board_id uint64 , card_id uint64 , comment_id uint64 , text string ) ( comment *Comment , err error ) {
	clean , err := cleanComment( text )
	if err != nil { return }
	err = store.Write( func( tx *db.Tx ) ( tx_err error ) {
		comment = &Comment{}
		key := commentKey( board_id , card_id , comment_id )
		if tx.Get( db.BucketComments , key , comment ) != nil { return ErrCommentNotFound }
		now := time.Now().UTC()
		comment.Text , comment.EditedAt = clean , &now
		if tx_err = tx.Put( db.BucketComments , key , comment ); tx_err != nil { return }
		if tx_err = touchBoardTx( tx , board_id ); tx_err != nil { return }
		tx_err = mirrorCommentTx( tx , board_id , card_id , comment , false )
		return
	} )
	if err != nil { comment = nil }
	return
}

func RemoveComment( store *db.Store , board_id uint64 , card_id uint64 , comment_id uint64 ) ( err error ) {
	err = store.Write( func( tx *db.Tx ) ( tx_err error ) {
		key := commentKey( board_id , card_id , comment_id )
		comment := &Comment{}
		if tx.Get( db.BucketComments , key , comment ) != nil { return ErrCommentNotFound }
		if tx_err = tx.Delete( db.BucketComments , key ); tx_err != nil { return }
		if tx_err = touchBoardTx( tx , board_id ); tx_err != nil { return }
		tx_err = mirrorCommentTx( tx , board_id , card_id , comment , true )
		return
	} )
	return
}

// deleteCommentsTx removes a card's whole thread, when the card itself goes
// for good. A card in the trash keeps its comments, so restoring it brings
// them back.
func deleteCommentsTx( tx *db.Tx , board_id uint64 , card_id uint64 ) ( err error ) {
	_ , err = tx.DeletePrefix( db.BucketComments , db.Key( board_id , card_id ) )
	return
}
