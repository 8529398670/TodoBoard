package models

import (
	errors "errors"
	reflect "reflect"
	strings "strings"
	time "time"

	db "todoboard/server/db"
	encryption "todoboard/server/encryption"
)

// Sending, copying and mirroring cards between boards.
//
// Send moves a card to another board whole: it keeps its id and ULID, so its
// link still finds it, and takes its comments and pictures along. Labels are
// matched on the new board by name (made there when missing), assignees are
// kept only if they are on that board, and per-person shares, subscriptions
// and share links are dropped, being about the board it left.
//
// Copy makes a new card on any board, the same one included, from the
// card's content, comments and pictures. It can be linked to the original:
// one way, where the original's changes flow to the copy and the copy's
// synced fields cannot be edited, or two way, where either side's changes
// flow to the other. A card can be in any number of links, and changes walk
// along a chain of them (never back round a loop). What syncs: title,
// description, checklist, dates, price, product, picture, To do / Doing /
// Done, and comments. What each side keeps: column, place, day, labels,
// assignees and rank.
//
// Trashing, archiving or deleting either side ends its links; the other card
// stays as it is. Restoring does not bring a link back.

const (
	MirrorOneWay = "one_way"
	MirrorTwoWay = "two_way"
	MirrorNone   = ""

	// Card.Mirror: what a card shows of its links.
	MirrorLinked = "linked" // in a two-way link, or the source of a one-way one
	MirrorCopy   = "copy"   // the copy end of a one-way link: synced fields are read-only
)

var (
	ErrMirrored      = errors.New( "card is a one-way mirror of another" )
	ErrLinkNotFound  = errors.New( "card link not found" )
)

func ValidMirrorMode( mode string ) ( ok bool ) {
	ok = mode == MirrorNone || mode == MirrorOneWay || mode == MirrorTwoWay
	return
}

// CardEnd names one card on one board.
type CardEnd struct {
	BoardID uint64 `json:"board_id"`
	CardID  uint64 `json:"card_id"`
}

func endOf( card *Card ) ( end CardEnd ) {
	end = CardEnd{ BoardID: card.BoardID , CardID: card.ID }
	return
}

// CardLink ties two cards. For a one-way link A is the source and B the copy.
type CardLink struct {
	ID        uint64    `json:"id"`
	A         CardEnd   `json:"a"`
	B         CardEnd   `json:"b"`
	Mode      string    `json:"mode"`
	CreatedBy string    `json:"created_by"`
	CreatedAt time.Time `json:"created_at"`
}

type cardLinkRef struct {
	LinkID uint64 `json:"link_id"`
}

func ( link *CardLink ) other( end CardEnd ) ( result CardEnd ) {
	result = link.A
	if link.A == end { result = link.B }
	return
}

// flowsFrom reports whether changes to end travel along this link.
func ( link *CardLink ) flowsFrom( end CardEnd ) ( result bool ) {
	result = link.Mode == MirrorTwoWay || link.A == end
	return
}

// locks reports whether end is the read-only copy of a one-way link.
func ( link *CardLink ) locks( end CardEnd ) ( result bool ) {
	result = link.Mode == MirrorOneWay && link.B == end
	return
}

func linkKey( link_id uint64 ) ( key []byte ) {
	key = encryption.Uint64ToBytes( link_id )
	return
}

func linkRefKey( end CardEnd , link_id uint64 ) ( key []byte ) {
	key = db.Key( end.BoardID , end.CardID , link_id )
	return
}

// linksOfTx is every link a card is in. A ref left pointing at a link that
// is gone is cleared on the way.
func linksOfTx( tx *db.Tx , end CardEnd ) ( links []*CardLink , err error ) {
	ids := []uint64{}
	err = tx.ForEachPrefix( db.BucketCardLinkRefs , db.Key( end.BoardID , end.CardID ) , func() any { return &cardLinkRef{} } ,
		func( key []byte , item any ) bool {
			ids = append( ids , item.( *cardLinkRef ).LinkID )
			return true
		} )
	if err != nil { return }
	for _ , id := range ids {
		link := &CardLink{}
		if get_err := tx.Get( db.BucketCardLinks , linkKey( id ) , link ); get_err != nil {
			if err = tx.Delete( db.BucketCardLinkRefs , linkRefKey( end , id ) ); err != nil { return }
			continue
		}
		links = append( links , link )
	}
	return
}

// mirrorState is what Card.Mirror should say, from a card's links.
func mirrorState( end CardEnd , links []*CardLink ) ( state string ) {
	for _ , link := range links {
		if link.locks( end ) { return MirrorCopy }
		state = MirrorLinked
	}
	return
}

// refreshMirrorTx rewrites the stored card's Mirror from its links. A card
// that is not on its board (trashed, gone) is left alone.
func refreshMirrorTx( tx *db.Tx , end CardEnd ) ( err error ) {
	card , get_err := getCardTx( tx , end.BoardID , end.CardID )
	if get_err != nil { return }
	links , err := linksOfTx( tx , end )
	if err != nil { return }
	if state := mirrorState( end , links ); state != card.Mirror {
		card.Mirror = state
		if err = tx.Put( db.BucketCards , cardKey( end.BoardID , end.CardID ) , card ); err != nil { return }
		err = touchBoardTx( tx , end.BoardID )
	}
	return
}

func addLinkTx( tx *db.Tx , a CardEnd , b CardEnd , mode string , actor_name string ) ( link *CardLink , err error ) {
	if mode != MirrorOneWay && mode != MirrorTwoWay { return nil , ErrBadInput }
	id , err := tx.NextSequence( db.BucketCardLinks )
	if err != nil { return }
	link = &CardLink{ ID: id , A: a , B: b , Mode: mode , CreatedBy: actor_name , CreatedAt: time.Now().UTC() }
	if err = tx.Put( db.BucketCardLinks , linkKey( id ) , link ); err != nil { return }
	for _ , end := range []CardEnd{ a , b } {
		if err = tx.Put( db.BucketCardLinkRefs , linkRefKey( end , id ) , &cardLinkRef{ LinkID: id } ); err != nil { return }
	}
	for _ , end := range []CardEnd{ a , b } {
		if err = refreshMirrorTx( tx , end ); err != nil { return }
	}
	return
}

func removeLinkTx( tx *db.Tx , link *CardLink ) ( err error ) {
	if err = tx.Delete( db.BucketCardLinks , linkKey( link.ID ) ); err != nil { return }
	for _ , end := range []CardEnd{ link.A , link.B } {
		if err = tx.Delete( db.BucketCardLinkRefs , linkRefKey( end , link.ID ) ); err != nil { return }
	}
	for _ , end := range []CardEnd{ link.A , link.B } {
		if err = refreshMirrorTx( tx , end ); err != nil { return }
	}
	return
}

// unlinkCardTx ends every link card is in, for a card being trashed,
// archived or deleted. card is the caller's copy, which it goes on to save:
// its Mirror is cleared here, since the stored one may be rewritten under it.
func unlinkCardTx( tx *db.Tx , card *Card ) ( err error ) {
	links , err := linksOfTx( tx , endOf( card ) )
	if err != nil { return }
	for _ , link := range links {
		if err = removeLinkTx( tx , link ); err != nil { return }
	}
	card.Mirror = MirrorNone
	return
}

// UnlinkCard ends one of a card's links.
func UnlinkCard( store *db.Store , board_id uint64 , card_id uint64 , link_id uint64 ) ( err error ) {
	err = store.Write( func( tx *db.Tx ) ( tx_err error ) {
		link := &CardLink{}
		if tx.Get( db.BucketCardLinks , linkKey( link_id ) , link ) != nil { return ErrLinkNotFound }
		end := CardEnd{ BoardID: board_id , CardID: card_id }
		if link.A != end && link.B != end { return ErrLinkNotFound }
		tx_err = removeLinkTx( tx , link )
		return
	} )
	return
}

// CardLinkView is one of a card's links as its card editor lists it.
type CardLinkView struct {
	ID         uint64 `json:"id"`
	Mode       string `json:"mode"`
	// "out": this card's changes flow to the other; "in": the other's flow
	// here (this card is a one-way copy); "both": two-way.
	Direction  string `json:"direction"`
	BoardID    uint64 `json:"board_id"`
	BoardTitle string `json:"board_title"`
	CardID     uint64 `json:"card_id"`
	CardTitle  string `json:"card_title"`
	ListTitle  string `json:"list_title"`
	CardPath   string `json:"card_path"`
}

// ListCardLinks is a card's links, with what the other end is. A link whose
// other card is gone (its board deleted) is ended on the way.
func ListCardLinks( store *db.Store , board_id uint64 , card_id uint64 ) ( views []*CardLinkView , err error ) {
	views = []*CardLinkView{}
	err = store.Write( func( tx *db.Tx ) ( tx_err error ) {
		end := CardEnd{ BoardID: board_id , CardID: card_id }
		links , tx_err := linksOfTx( tx , end )
		if tx_err != nil { return }
		for _ , link := range links {
			other := link.other( end )
			board , board_err := getBoardTx( tx , other.BoardID )
			card , card_err := getCardTx( tx , other.BoardID , other.CardID )
			if board_err != nil || card_err != nil {
				if tx_err = removeLinkTx( tx , link ); tx_err != nil { return }
				continue
			}
			direction := "both"
			if link.Mode == MirrorOneWay {
				direction = "out"
				if link.B == end { direction = "in" }
			}
			view := &CardLinkView{ ID: link.ID , Mode: link.Mode , Direction: direction , BoardID: board.ID , BoardTitle: board.Title ,
				CardID: card.ID , CardTitle: card.Title , CardPath: CardPath( board , card ) }
			if list := board.List( card.ListID ); list != nil { view.ListTitle = list.Title }
			views = append( views , view )
		}
		return
	} )
	if err != nil { views = nil }
	return
}

// syncedEqual reports whether two cards agree on everything a mirror syncs.
func syncedEqual( a *Card , b *Card ) ( result bool ) {
	if a.Title != b.Title || a.Description != b.Description || a.Price != b.Price { return }
	if a.Done != b.Done || a.Doing != b.Doing { return }
	if timeChanged( a.StartAt , b.StartAt ) || a.StartTimed != b.StartTimed { return }
	if timeChanged( a.DueAt , b.DueAt ) || a.DueAllDay != b.DueAllDay { return }
	if timeChanged( a.ImageAt , b.ImageAt ) { return }
	if reflect.DeepEqual( a.Product , b.Product ) == false { return }
	if len( a.Checklist ) != len( b.Checklist ) { return }
	for index , item := range a.Checklist {
		if *item != *b.Checklist[ index ] { return }
	}
	result = true
	return
}

func copyTime( value *time.Time ) ( result *time.Time ) {
	if value == nil { return }
	copied := *value
	result = &copied
	return
}

// copySynced puts from's synced fields on to.
func copySynced( from *Card , to *Card ) {
	to.Title , to.Description , to.Price = from.Title , from.Description , from.Price
	to.Done , to.Doing , to.DoneAt = from.Done , from.Doing , copyTime( from.DoneAt )
	to.StartAt , to.StartTimed = copyTime( from.StartAt ) , from.StartTimed
	to.DueAt , to.DueAllDay = copyTime( from.DueAt ) , from.DueAllDay
	to.ImageAt = copyTime( from.ImageAt )
	to.Product = nil
	if from.Product != nil {
		product := *from.Product
		to.Product = &product
	}
	to.Checklist = make( []*ChecklistItem , 0 , len( from.Checklist ) )
	for _ , item := range from.Checklist {
		copied := *item
		to.Checklist = append( to.Checklist , &copied )
	}
	if from.NextItemID > to.NextItemID { to.NextItemID = from.NextItemID }
}

// copyBlobTx copies a card's picture (or product photo) row to another card,
// or clears the other's when from has none.
func copyBlobTx( tx *db.Tx , bucket string , from CardEnd , to CardEnd ) ( err error ) {
	image := &ProductImage{}
	if tx.Get( bucket , cardKey( from.BoardID , from.CardID ) , image ) == nil {
		err = tx.Put( bucket , cardKey( to.BoardID , to.CardID ) , image )
		return
	}
	err = tx.Delete( bucket , cardKey( to.BoardID , to.CardID ) )
	return
}

// mirrorsOfTx walks card's links the way its changes flow and returns every
// card they reach, nearest first, each once and never card itself. Ends that
// are not on a board (trashed, gone) are skipped and not walked through.
func mirrorsOfTx( tx *db.Tx , card *Card ) ( reached []*Card , err error ) {
	start := endOf( card )
	seen := map[CardEnd]bool{ start: true }
	queue := []CardEnd{ start }
	for len( queue ) > 0 {
		current := queue[ 0 ]
		queue = queue[ 1: ]
		links , links_err := linksOfTx( tx , current )
		if links_err != nil { return nil , links_err }
		for _ , link := range links {
			if link.flowsFrom( current ) == false { continue }
			next := link.other( current )
			if seen[ next ] { continue }
			seen[ next ] = true
			target , get_err := getCardTx( tx , next.BoardID , next.CardID )
			if get_err != nil { continue }
			reached = append( reached , target )
			queue = append( queue , next )
		}
	}
	return
}

// syncMirrorsTx carries card's synced fields to every card its changes reach,
// after card has been saved. On a one-way copy a change to a synced field is
// refused (the whole write is), and moving or relabelling one is not.
func syncMirrorsTx( tx *db.Tx , card *Card , actor_name string ) ( err error ) {
	end := endOf( card )
	links , err := linksOfTx( tx , end )
	if err != nil || len( links ) == 0 { return }
	for _ , link := range links {
		if link.locks( end ) == false { continue }
		source , get_err := getCardTx( tx , link.A.BoardID , link.A.CardID )
		if get_err == nil && syncedEqual( source , card ) == false { return ErrMirrored }
	}
	targets , err := mirrorsOfTx( tx , card )
	if err != nil { return }
	now := time.Now().UTC()
	for _ , target := range targets {
		if syncedEqual( card , target ) { continue }
		to := endOf( target )
		if timeChanged( card.ImageAt , target.ImageAt ) {
			if err = copyBlobTx( tx , db.BucketCardImages , end , to ); err != nil { return }
		}
		if reflect.DeepEqual( card.Product , target.Product ) == false {
			if err = copyBlobTx( tx , db.BucketProductImages , end , to ); err != nil { return }
		}
		// Rev guards typing in the title and description; only a change to
		// those should tell someone mid-edit it changed under them.
		if card.Title != target.Title || card.Description != target.Description { target.Rev += 1 }
		copySynced( card , target )
		target.UpdatedAt , target.UpdatedBy = now , actor_name
		if err = tx.Put( db.BucketCards , cardKey( to.BoardID , to.CardID ) , target ); err != nil { return }
		if err = touchBoardTx( tx , to.BoardID ); err != nil { return }
	}
	return
}

// mapLabels is card's labels as they are on dst: the same ids on the same
// board, else matched by name, made on dst when it has none by that name
// (and room for it).
func mapLabels( src *Board , dst *Board , ids []uint64 ) ( result []uint64 ) {
	result = []uint64{}
	for _ , id := range ids {
		if src.ID == dst.ID {
			if dst.Label( id ) != nil && containsID( result , id ) == false { result = append( result , id ) }
			continue
		}
		label := src.Label( id )
		if label == nil { continue }
		var found *Label
		for _ , candidate := range dst.Labels {
			if strings.EqualFold( strings.TrimSpace( candidate.Name ) , strings.TrimSpace( label.Name ) ) {
				found = candidate
				break
			}
		}
		if found == nil {
			made , add_err := AddLabel( dst , label.Name , label.Color )
			if add_err != nil { continue }
			found = made
		}
		if containsID( result , found.ID ) == false { result = append( result , found.ID ) }
	}
	return
}

func keepMembers( board *Board , ids []uint64 ) ( result []uint64 ) {
	result = []uint64{}
	for _ , id := range ids {
		if board.RoleOf( id ) != "" && containsID( result , id ) == false { result = append( result , id ) }
	}
	return
}

// placeInListTx is where a card lands in list_id: position when given, else
// the top or the end.
func placeInListTx( tx *db.Tx , board *Board , list_id uint64 , top bool , position *float64 ) ( place float64 , err error ) {
	list := board.List( list_id )
	if list == nil || list.Archived { return 0 , ErrListNotFound }
	if position != nil {
		place = *position
		return
	}
	if top == false {
		place , err = endOfListTx( tx , board.ID , list_id )
		return
	}
	place = positionStep
	first := true
	err = tx.ForEachPrefix( db.BucketCards , db.Key( board.ID ) , func() any { return &Card{} } ,
		func( key []byte , item any ) bool {
			if card := item.( *Card ); card.ListID == list_id && ( first || card.Position - positionStep < place ) {
				place , first = card.Position - positionStep , false
			}
			return true
		} )
	return
}

// TransferPlace says where a sent or copied card goes on its new board.
type TransferPlace struct {
	BoardID  uint64
	ListID   uint64
	Top      bool
	Position *float64
}

// SendCard moves a card to another board. See the top of this file.
func SendCard( store *db.Store , board_id uint64 , card_id uint64 , place TransferPlace , actor_name string ) ( card *Card , err error ) {
	if place.BoardID == board_id { return nil , ErrBadInput }
	err = store.Write( func( tx *db.Tx ) ( tx_err error ) {
		src , tx_err := getBoardTx( tx , board_id )
		if tx_err != nil { return }
		dst , tx_err := getBoardTx( tx , place.BoardID )
		if tx_err != nil { return }
		card , tx_err = getCardTx( tx , board_id , card_id )
		if tx_err != nil { return }
		position , tx_err := placeInListTx( tx , dst , place.ListID , place.Top , place.Position )
		if tx_err != nil { return }
		from := endOf( card )
		to := CardEnd{ BoardID: dst.ID , CardID: card.ID }

		// What belonged to the old board goes.
		if tx_err = detachCardTx( tx , card ); tx_err != nil { return }
		card.SharedWith = map[uint64]string{}
		card.LabelIDs = mapLabels( src , dst , card.LabelIDs )
		card.Assignees = keepMembers( dst , card.Assignees )
		card.Rank = 0

		// Its rows move with it.
		if tx_err = tx.Delete( db.BucketCards , cardKey( from.BoardID , from.CardID ) ); tx_err != nil { return }
		for _ , bucket := range []string{ db.BucketCardImages , db.BucketProductImages } {
			if tx_err = copyBlobTx( tx , bucket , from , to ); tx_err != nil { return }
			if tx_err = tx.Delete( bucket , cardKey( from.BoardID , from.CardID ) ); tx_err != nil { return }
		}
		comments := []*Comment{}
		tx_err = tx.ForEachPrefix( db.BucketComments , db.Key( from.BoardID , from.CardID ) , func() any { return &Comment{} } ,
			func( key []byte , item any ) bool {
				comments = append( comments , item.( *Comment ) )
				return true
			} )
		if tx_err != nil { return }
		if tx_err = deleteCommentsTx( tx , from.BoardID , from.CardID ); tx_err != nil { return }
		for _ , comment := range comments {
			comment.BoardID = to.BoardID
			if tx_err = tx.Put( db.BucketComments , commentKey( to.BoardID , to.CardID , comment.ID ) , comment ); tx_err != nil { return }
		}
		links , tx_err := linksOfTx( tx , from )
		if tx_err != nil { return }
		for _ , link := range links {
			if link.A == from { link.A = to }
			if link.B == from { link.B = to }
			if tx_err = tx.Put( db.BucketCardLinks , linkKey( link.ID ) , link ); tx_err != nil { return }
			if tx_err = tx.Delete( db.BucketCardLinkRefs , linkRefKey( from , link.ID ) ); tx_err != nil { return }
			if tx_err = tx.Put( db.BucketCardLinkRefs , linkRefKey( to , link.ID ) , &cardLinkRef{ LinkID: link.ID } ); tx_err != nil { return }
		}

		card.BoardID , card.ListID , card.Position = dst.ID , place.ListID , position
		dst.NextCardSerial += 1
		card.Serial = dst.NextCardSerial
		if tx_err = saveCardTx( tx , dst , card , actor_name ); tx_err != nil { return }
		if tx_err = renormalizeCardsTx( tx , dst.ID , card.ListID ); tx_err != nil { return }
		if tx_err = putBoardTx( tx , src ); tx_err != nil { return }
		card , tx_err = getCardTx( tx , dst.ID , card.ID )
		return
	} )
	if err != nil { card = nil }
	return
}

// CopyCard copies a card onto a board (its own included), linked to it by
// mirror unless that is MirrorNone. See the top of this file.
func CopyCard( store *db.Store , board_id uint64 , card_id uint64 , place TransferPlace , mirror string , created_by uint64 , actor_name string ) ( copied *Card , err error ) {
	if ValidMirrorMode( mirror ) == false { return nil , ErrBadInput }
	err = store.Write( func( tx *db.Tx ) ( tx_err error ) {
		src , tx_err := getBoardTx( tx , board_id )
		if tx_err != nil { return }
		dst := src
		if place.BoardID != board_id {
			dst , tx_err = getBoardTx( tx , place.BoardID )
			if tx_err != nil { return }
		}
		card , tx_err := getCardTx( tx , board_id , card_id )
		if tx_err != nil { return }
		position , tx_err := placeInListTx( tx , dst , place.ListID , place.Top , place.Position )
		if tx_err != nil { return }
		id , tx_err := tx.NextSequence( db.BucketCards )
		if tx_err != nil { return }
		now := time.Now().UTC()
		copied = &Card{
			ID: id , BoardID: dst.ID , ListID: place.ListID , Position: position , Day: card.Day ,
			LabelIDs: mapLabels( src , dst , card.LabelIDs ) , Assignees: keepMembers( dst , card.Assignees ) ,
			CreatedBy: created_by , CreatedAt: now , UpdatedAt: now , UpdatedBy: actor_name , Rev: 1 ,
		}
		copySynced( card , copied )
		dst.stampCard( copied , now )
		normalizeCard( copied )
		from , to := endOf( card ) , endOf( copied )
		if tx_err = tx.Put( db.BucketCards , cardKey( to.BoardID , to.CardID ) , copied ); tx_err != nil { return }
		for _ , bucket := range []string{ db.BucketCardImages , db.BucketProductImages } {
			if tx_err = copyBlobTx( tx , bucket , from , to ); tx_err != nil { return }
		}
		if tx_err = copyCommentsTx( tx , from , to ); tx_err != nil { return }
		if tx_err = renormalizeCardsTx( tx , dst.ID , copied.ListID ); tx_err != nil { return }
		if tx_err = putBoardTx( tx , dst ); tx_err != nil { return }
		if dst != src {
			if tx_err = touchBoardTx( tx , src.ID ); tx_err != nil { return }
		}
		if mirror != MirrorNone {
			if _ , tx_err = addLinkTx( tx , from , to , mirror , actor_name ); tx_err != nil { return }
		}
		copied , tx_err = getCardTx( tx , to.BoardID , to.CardID )
		return
	} )
	if err != nil { copied = nil }
	return
}

// copyCommentsTx copies a card's comments to another, oldest first, giving
// any without a thread one so later edits can find their copies.
func copyCommentsTx( tx *db.Tx , from CardEnd , to CardEnd ) ( err error ) {
	comments := []*Comment{}
	err = tx.ForEachPrefix( db.BucketComments , db.Key( from.BoardID , from.CardID ) , func() any { return &Comment{} } ,
		func( key []byte , item any ) bool {
			comments = append( comments , item.( *Comment ) )
			return true
		} )
	if err != nil { return }
	for _ , comment := range comments {
		if comment.Thread == "" {
			comment.Thread = newULID( comment.CreatedAt )
			if err = tx.Put( db.BucketComments , commentKey( from.BoardID , from.CardID , comment.ID ) , comment ); err != nil { return }
		}
		if err = putCommentCopyTx( tx , comment , to ); err != nil { return }
	}
	return
}

func putCommentCopyTx( tx *db.Tx , comment *Comment , to CardEnd ) ( err error ) {
	id , err := tx.NextSequence( db.BucketComments )
	if err != nil { return }
	copied := *comment
	copied.ID , copied.BoardID , copied.CardID = id , to.BoardID , to.CardID
	err = tx.Put( db.BucketComments , commentKey( to.BoardID , to.CardID , id ) , &copied )
	return
}

// mirrorCommentTx carries a new, edited (text set) or removed (remove) comment
// to the cards card's changes reach, matched there by thread.
func mirrorCommentTx( tx *db.Tx , board_id uint64 , card_id uint64 , comment *Comment , remove bool ) ( err error ) {
	if comment.Thread == "" { return }
	card , err := getCardTx( tx , board_id , card_id )
	if err != nil { return }
	targets , err := mirrorsOfTx( tx , card )
	if err != nil { return }
	for _ , target := range targets {
		to := endOf( target )
		var match *Comment
		tx_err := tx.ForEachPrefix( db.BucketComments , db.Key( to.BoardID , to.CardID ) , func() any { return &Comment{} } ,
			func( key []byte , item any ) bool {
				if found := item.( *Comment ); found.Thread == comment.Thread {
					match = found
					return false
				}
				return true
			} )
		if tx_err != nil { return tx_err }
		switch {
		case remove && match != nil:
			err = tx.Delete( db.BucketComments , commentKey( to.BoardID , to.CardID , match.ID ) )
		case remove:
			continue
		case match != nil:
			match.Text , match.EditedAt = comment.Text , copyTime( comment.EditedAt )
			err = tx.Put( db.BucketComments , commentKey( to.BoardID , to.CardID , match.ID ) , match )
		default:
			err = putCommentCopyTx( tx , comment , to )
		}
		if err != nil { return }
		if err = touchBoardTx( tx , to.BoardID ); err != nil { return }
	}
	return
}
