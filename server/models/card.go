package models

import (
	errors "errors"
	math "math"
	sort "sort"
	strings "strings"
	time "time"
	utf8 "unicode/utf8"

	db "todoboard/server/db"
	encryption "todoboard/server/encryption"
)

// Card modes for sharing one card with someone who is not on the board.
const (
	ModeView = "view"
	ModeEdit = "edit"
)

const (
	// positionStep is the gap between neighbours after a renormalize. Moving
	// something between two neighbours takes the midpoint, so a list can
	// absorb dozens of moves into the same gap before it needs respacing.
	positionStep = 1024.0

	// minPositionGap is how close two neighbours may get before the whole
	// list is respaced. Far above float64's precision limit on purpose.
	minPositionGap = 1e-6

	maxDescriptionBytes = 100 * 1024
	maxChecklistItems   = 200
)

var (
	ErrCardNotFound = errors.New( "card not found" )
	ErrItemNotFound = errors.New( "checklist item not found" )

	// ErrStale means the client edited text based on an older copy of the
	// card than the one stored. Answered with 409 so the client can keep its
	// draft and show what changed instead of silently overwriting someone.
	ErrStale = errors.New( "card changed since it was loaded" )
)

// Card is one card. Position orders cards within ListID; it is a float so a
// move only rewrites the card that moved (see positionStep).
type Card struct {
	ID          uint64            `json:"id"`
	BoardID     uint64            `json:"board_id"`
	ListID      uint64            `json:"list_id"`
	Position    float64           `json:"position"`
	Title       string            `json:"title"`
	Description string            `json:"description"`
	LabelIDs    []uint64          `json:"label_ids"`
	Assignees   []uint64          `json:"assignees"`
	StartAt     *time.Time        `json:"start_at"`
	// StartTimed is false for a date-only start -- every start written
	// before start times existed, so they read as all-day without a
	// migration. DueAllDay predates it and keeps its own sense.
	StartTimed  bool              `json:"start_timed"`
	DueAt       *time.Time        `json:"due_at"`
	DueAllDay   bool              `json:"due_all_day"`
	Done        bool              `json:"done"`
	DoneAt      *time.Time        `json:"done_at"`
	Checklist   []*ChecklistItem  `json:"checklist"`
	SharedWith  map[uint64]string `json:"shared_with"`
	Archived    bool              `json:"archived"`
	CreatedBy   uint64            `json:"created_by"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
	UpdatedBy   string            `json:"updated_by"`
	Rev         uint64            `json:"rev"`
	NextItemID  uint64            `json:"next_item_id"`
}

type ChecklistItem struct {
	ID       uint64  `json:"id"`
	Text     string  `json:"text"`
	Done     bool    `json:"done"`
	Position float64 `json:"position"`
}

// CardRef is the per-user index row for a card shared with that user.
type CardRef struct {
	Mode string `json:"mode"`
}

func cardKey( board_id uint64 , card_id uint64 ) ( key []byte ) {
	key = db.Key( board_id , card_id )
	return
}

func normalizeCard( card *Card ) {
	if card.LabelIDs == nil { card.LabelIDs = []uint64{} }
	if card.Assignees == nil { card.Assignees = []uint64{} }
	if card.Checklist == nil { card.Checklist = []*ChecklistItem{} }
	if card.SharedWith == nil { card.SharedWith = map[uint64]string{} }
	sort.SliceStable( card.Checklist , func( i int , j int ) bool { return card.Checklist[ i ].Position < card.Checklist[ j ].Position } )
}

func GetCard( store *db.Store , board_id uint64 , card_id uint64 ) ( card *Card , err error ) {
	card = &Card{}
	if err = store.Get( db.BucketCards , cardKey( board_id , card_id ) , card ); err != nil {
		card = nil
		if errors.Is( err , db.ErrNotFound ) { err = ErrCardNotFound }
		return
	}
	normalizeCard( card )
	return
}

func getCardTx( tx *db.Tx , board_id uint64 , card_id uint64 ) ( card *Card , err error ) {
	card = &Card{}
	if err = tx.Get( db.BucketCards , cardKey( board_id , card_id ) , card ); err != nil {
		card = nil
		if errors.Is( err , db.ErrNotFound ) { err = ErrCardNotFound }
		return
	}
	normalizeCard( card )
	return
}

func ListCards( store *db.Store , board_id uint64 ) ( cards []*Card , err error ) {
	cards = []*Card{}
	err = store.ForEachPrefix( db.BucketCards , db.Key( board_id ) , func() any { return &Card{} } ,
		func( key []byte , item any ) bool {
			card := item.( *Card )
			normalizeCard( card )
			cards = append( cards , card )
			return true
		} )
	return
}

// eachCardTx rewrites every card of a board that visit reports as changed.
func eachCardTx( tx *db.Tx , board_id uint64 , visit func( card *Card ) bool ) ( err error ) {
	changed := []*Card{}
	err = tx.ForEachPrefix( db.BucketCards , db.Key( board_id ) , func() any { return &Card{} } ,
		func( key []byte , item any ) bool {
			card := item.( *Card )
			normalizeCard( card )
			if visit( card ) { changed = append( changed , card ) }
			return true
		} )
	if err != nil { return }
	for _ , card := range changed {
		card.Rev += 1
		if err = tx.Put( db.BucketCards , cardKey( card.BoardID , card.ID ) , card ); err != nil { return }
	}
	return
}

// CreateCard puts a new card at the end of list_id, or at position if given.
func CreateCard( store *db.Store , board_id uint64 , list_id uint64 , title string , position *float64 , created_by uint64 , actor_name string ) ( card *Card , err error ) {
	clean , ok := cleanTitle( title , 300 )
	if ok == false {
		err = ErrBadInput
		return
	}
	now := time.Now().UTC()
	err = store.Write( func( tx *db.Tx ) ( tx_err error ) {
		board , tx_err := getBoardTx( tx , board_id )
		if tx_err != nil { return }
		if board.List( list_id ) == nil { return ErrListNotFound }
		id , tx_err := tx.NextSequence( db.BucketCards )
		if tx_err != nil { return }
		card = &Card{
			ID: id , BoardID: board_id , ListID: list_id , Title: clean ,
			CreatedBy: created_by , CreatedAt: now , UpdatedAt: now , UpdatedBy: actor_name , Rev: 1 ,
		}
		normalizeCard( card )
		if position != nil {
			card.Position = *position
		} else {
			card.Position , tx_err = endOfListTx( tx , board_id , list_id )
			if tx_err != nil { return }
		}
		if tx_err = tx.Put( db.BucketCards , cardKey( board_id , id ) , card ); tx_err != nil { return }
		if tx_err = renormalizeCardsTx( tx , board_id , list_id ); tx_err != nil { return }
		tx_err = putBoardTx( tx , board )
		return
	} )
	if err != nil { card = nil }
	return
}

func endOfListTx( tx *db.Tx , board_id uint64 , list_id uint64 ) ( position float64 , err error ) {
	position = positionStep
	err = tx.ForEachPrefix( db.BucketCards , db.Key( board_id ) , func() any { return &Card{} } ,
		func( key []byte , item any ) bool {
			if card := item.( *Card ); card.ListID == list_id && card.Position >= position {
				position = card.Position + positionStep
			}
			return true
		} )
	return
}

func needsRenormalize( positions []float64 ) ( result bool ) {
	sort.Float64s( positions )
	for index := 1; index < len( positions ); index += 1 {
		if positions[ index ] - positions[ index - 1 ] < minPositionGap { return true }
	}
	for _ , position := range positions {
		if math.IsNaN( position ) || math.IsInf( position , 0 ) { return true }
	}
	return
}

// renormalizeCardsTx respaces a list's cards when two have crowded together.
// It is a no-op almost always, so the common move stays a one-card write.
func renormalizeCardsTx( tx *db.Tx , board_id uint64 , list_id uint64 ) ( err error ) {
	cards := []*Card{}
	err = tx.ForEachPrefix( db.BucketCards , db.Key( board_id ) , func() any { return &Card{} } ,
		func( key []byte , item any ) bool {
			if card := item.( *Card ); card.ListID == list_id { cards = append( cards , card ) }
			return true
		} )
	if err != nil { return }
	positions := make( []float64 , len( cards ) )
	for index , card := range cards { positions[ index ] = card.Position }
	if needsRenormalize( positions ) == false { return }
	sort.SliceStable( cards , func( i int , j int ) bool { return cards[ i ].Position < cards[ j ].Position } )
	for index , card := range cards {
		card.Position = float64( index + 1 ) * positionStep
		card.Rev += 1
		if err = tx.Put( db.BucketCards , cardKey( board_id , card.ID ) , card ); err != nil { return }
	}
	return
}

// CardPatch is a partial update; nil fields are left alone. Rev, when set,
// must match the stored card for a Title or Description change -- those are
// the edits where two people typing over each other loses work. Toggling,
// moving and dating are last-writer-wins, which is what people expect of a
// drag or a checkbox.
type CardPatch struct {
	Rev         *uint64   `json:"rev"`
	Title       *string   `json:"title"`
	Description *string   `json:"description"`
	ListID      *uint64   `json:"list_id"`
	Position    *float64  `json:"position"`
	LabelIDs    *[]uint64 `json:"label_ids"`
	Assignees   *[]uint64 `json:"assignees"`
	// Dates travel as strings: "" clears, "2026-10-30" is an all-day date,
	// anything else must be RFC 3339.
	StartAt  *string `json:"start_at"`
	DueAt    *string `json:"due_at"`
	Done     *bool   `json:"done"`
	Archived *bool   `json:"archived"`
}

// Moves reports whether the patch relocates the card, which needs board-level
// edit rights rather than just rights on this one card.
func ( patch *CardPatch ) Moves() ( result bool ) {
	result = patch.ListID != nil || patch.Position != nil || patch.Archived != nil
	return
}

// CardChange describes what a patch did, for notifications.
type CardChange struct {
	Moved         bool
	FromListID    uint64
	Completed     bool
	Reopened      bool
	Edited        bool
	NewAssignees  []uint64
	DueChanged    bool
}

func ( change *CardChange ) Any() ( result bool ) {
	result = change.Moved || change.Completed || change.Reopened || change.Edited || len( change.NewAssignees ) > 0 || change.DueChanged
	return
}

// ParseCardDate reads the date format CardPatch documents.
func ParseCardDate( value string ) ( result *time.Time , all_day bool , err error ) {
	value = strings.TrimSpace( value )
	if value == "" { return }
	if parsed , day_err := time.Parse( "2006-01-02" , value ); day_err == nil {
		result , all_day = &parsed , true
		return
	}
	parsed , err := time.Parse( time.RFC3339 , value )
	if err != nil {
		err = ErrBadInput
		return
	}
	utc := parsed.UTC()
	result = &utc
	return
}

func ( patch *CardPatch ) apply( board *Board , card *Card ) ( change CardChange , err error ) {
	if patch.Rev != nil && ( patch.Title != nil || patch.Description != nil ) && *patch.Rev != card.Rev {
		err = ErrStale
		return
	}
	if patch.Title != nil {
		clean , ok := cleanTitle( *patch.Title , 300 )
		if ok == false { return change , ErrBadInput }
		change.Edited = change.Edited || clean != card.Title
		card.Title = clean
	}
	if patch.Description != nil {
		if len( *patch.Description ) > maxDescriptionBytes || utf8.ValidString( *patch.Description ) == false { return change , ErrBadInput }
		change.Edited = change.Edited || *patch.Description != card.Description
		card.Description = *patch.Description
	}
	if patch.ListID != nil {
		if board.List( *patch.ListID ) == nil { return change , ErrListNotFound }
		if *patch.ListID != card.ListID {
			change.Moved , change.FromListID = true , card.ListID
		}
		card.ListID = *patch.ListID
	}
	if patch.Position != nil {
		if math.IsNaN( *patch.Position ) || math.IsInf( *patch.Position , 0 ) { return change , ErrBadInput }
		card.Position = *patch.Position
	}
	if patch.LabelIDs != nil {
		labels := []uint64{}
		for _ , id := range *patch.LabelIDs {
			if board.Label( id ) != nil && containsID( labels , id ) == false { labels = append( labels , id ) }
		}
		card.LabelIDs = labels
	}
	if patch.Assignees != nil {
		// Only people who can see the whole board, or this card, can be
		// assigned -- assigning a stranger would leak the card to nobody and
		// confuse everybody.
		assignees := []uint64{}
		for _ , id := range *patch.Assignees {
			if containsID( assignees , id ) { continue }
			if board.RoleOf( id ) == "" && card.SharedWith[ id ] == "" { continue }
			assignees = append( assignees , id )
			if containsID( card.Assignees , id ) == false { change.NewAssignees = append( change.NewAssignees , id ) }
		}
		card.Assignees = assignees
	}
	if patch.StartAt != nil {
		start , all_day , date_err := ParseCardDate( *patch.StartAt )
		if date_err != nil { return change , date_err }
		card.StartAt , card.StartTimed = start , start != nil && all_day == false
	}
	if patch.DueAt != nil {
		due , all_day , date_err := ParseCardDate( *patch.DueAt )
		if date_err != nil { return change , date_err }
		change.DueChanged = timeChanged( card.DueAt , due ) || all_day != card.DueAllDay
		card.DueAt , card.DueAllDay = due , all_day
	}
	if patch.Done != nil && *patch.Done != card.Done {
		card.Done = *patch.Done
		if card.Done {
			now := time.Now().UTC()
			card.DoneAt = &now
			change.Completed = true
		} else {
			card.DoneAt = nil
			change.Reopened = true
		}
	}
	if patch.Archived != nil {
		change.Edited = change.Edited || *patch.Archived != card.Archived
		card.Archived = *patch.Archived
	}
	return
}

func timeChanged( a *time.Time , b *time.Time ) ( result bool ) {
	if a == nil || b == nil {
		result = a != b
		return
	}
	result = a.Equal( *b ) == false
	return
}

// PatchCard applies patch atomically and stamps both the card and its board.
func PatchCard( store *db.Store , board_id uint64 , card_id uint64 , patch *CardPatch , actor_name string ) ( card *Card , change CardChange , err error ) {
	err = store.Write( func( tx *db.Tx ) ( tx_err error ) {
		board , tx_err := getBoardTx( tx , board_id )
		if tx_err != nil { return }
		card , tx_err = getCardTx( tx , board_id , card_id )
		if tx_err != nil { return }
		change , tx_err = patch.apply( board , card )
		if tx_err != nil { return }
		tx_err = saveCardTx( tx , board , card , actor_name )
		if tx_err != nil { return }
		if patch.ListID != nil || patch.Position != nil {
			tx_err = renormalizeCardsTx( tx , board_id , card.ListID )
			if tx_err != nil { return }
			// The renormalize may have rewritten this card's position.
			card , tx_err = getCardTx( tx , board_id , card_id )
		}
		return
	} )
	if err != nil { card = nil }
	return
}

func saveCardTx( tx *db.Tx , board *Board , card *Card , actor_name string ) ( err error ) {
	card.Rev += 1
	card.UpdatedAt = time.Now().UTC()
	card.UpdatedBy = actor_name
	if err = tx.Put( db.BucketCards , cardKey( card.BoardID , card.ID ) , card ); err != nil { return }
	err = putBoardTx( tx , board )
	return
}

// MutateCard runs mutate on a loaded card and saves it -- the path for the
// checklist operations, which each touch one item.
func MutateCard( store *db.Store , board_id uint64 , card_id uint64 , actor_name string , mutate func( board *Board , card *Card ) error ) ( card *Card , err error ) {
	err = store.Write( func( tx *db.Tx ) ( tx_err error ) {
		board , tx_err := getBoardTx( tx , board_id )
		if tx_err != nil { return }
		card , tx_err = getCardTx( tx , board_id , card_id )
		if tx_err != nil { return }
		if tx_err = mutate( board , card ); tx_err != nil { return }
		tx_err = saveCardTx( tx , board , card , actor_name )
		return
	} )
	if err != nil { card = nil }
	return
}

func AddChecklistItem( card *Card , text string ) ( item *ChecklistItem , err error ) {
	clean , ok := cleanTitle( text , 500 )
	if ok == false || len( card.Checklist ) >= maxChecklistItems {
		err = ErrBadInput
		return
	}
	card.NextItemID += 1
	position := positionStep
	for _ , existing := range card.Checklist {
		if existing.Position >= position { position = existing.Position + positionStep }
	}
	item = &ChecklistItem{ ID: card.NextItemID , Text: clean , Position: position }
	card.Checklist = append( card.Checklist , item )
	return
}

type ChecklistUpdate struct {
	Text     *string  `json:"text"`
	Done     *bool    `json:"done"`
	Position *float64 `json:"position"`
}

func ( update *ChecklistUpdate ) Apply( card *Card , item_id uint64 ) ( err error ) {
	var item *ChecklistItem
	for _ , candidate := range card.Checklist {
		if candidate.ID == item_id { item = candidate }
	}
	if item == nil { return ErrItemNotFound }
	if update.Text != nil {
		clean , ok := cleanTitle( *update.Text , 500 )
		if ok == false { return ErrBadInput }
		item.Text = clean
	}
	if update.Done != nil { item.Done = *update.Done }
	if update.Position != nil {
		if math.IsNaN( *update.Position ) || math.IsInf( *update.Position , 0 ) { return ErrBadInput }
		item.Position = *update.Position
		positions := []float64{}
		for _ , existing := range card.Checklist { positions = append( positions , existing.Position ) }
		if needsRenormalize( positions ) {
			normalizeCard( card )
			for index , existing := range card.Checklist { existing.Position = float64( index + 1 ) * positionStep }
		}
	}
	return
}

func DeleteChecklistItem( card *Card , item_id uint64 ) ( err error ) {
	kept := []*ChecklistItem{}
	for _ , item := range card.Checklist {
		if item.ID != item_id { kept = append( kept , item ) }
	}
	if len( kept ) == len( card.Checklist ) { return ErrItemNotFound }
	card.Checklist = kept
	return
}

// ArchiveListCards archives open cards in one list -- every one, or with
// card_ids only those (ids on other lists are ignored) -- in one transaction,
// and returns which cards it archived: what an undo needs to bring exactly
// those back.
func ArchiveListCards( store *db.Store , board_id uint64 , list_id uint64 , card_ids []uint64 , actor_name string ) ( ids []uint64 , err error ) {
	ids = []uint64{}
	err = store.Write( func( tx *db.Tx ) ( tx_err error ) {
		board , tx_err := getBoardTx( tx , board_id )
		if tx_err != nil { return }
		if board.List( list_id ) == nil { return ErrListNotFound }
		now := time.Now().UTC()
		tx_err = eachCardTx( tx , board_id , func( card *Card ) ( changed bool ) {
			if card.ListID != list_id || card.Archived { return }
			if card_ids != nil && containsID( card_ids , card.ID ) == false { return }
			card.Archived , card.UpdatedAt , card.UpdatedBy = true , now , actor_name
			ids = append( ids , card.ID )
			changed = true
			return
		} )
		if tx_err != nil { return }
		tx_err = putBoardTx( tx , board )
		return
	} )
	if err != nil { ids = nil }
	return
}

func DeleteCard( store *db.Store , board_id uint64 , card_id uint64 ) ( err error ) {
	err = store.Write( func( tx *db.Tx ) ( tx_err error ) {
		card , tx_err := getCardTx( tx , board_id , card_id )
		if tx_err != nil { return }
		if tx_err = deleteCardTx( tx , card ); tx_err != nil { return }
		tx_err = touchBoardTx( tx , board_id )
		return
	} )
	return
}

// deleteCardTx removes a card and every row that points at it.
func deleteCardTx( tx *db.Tx , card *Card ) ( err error ) {
	for user_id := range card.SharedWith {
		if err = tx.Delete( db.BucketUserCards , db.Key( user_id , card.BoardID , card.ID ) ); err != nil { return }
	}
	_ , err = tx.DeleteWhere( db.BucketSubscriptions , func() any { return &Subscription{} } ,
		func( key []byte , item any ) bool {
			sub := item.( *Subscription )
			return sub.Kind == subscriptionCard && sub.BoardID == card.BoardID && sub.CardID == card.ID
		} )
	if err != nil { return }
	_ , err = tx.DeleteWhere( db.BucketShareLinks , func() any { return &ShareLink{} } ,
		func( key []byte , item any ) bool {
			link := item.( *ShareLink )
			return link.BoardID == card.BoardID && link.CardID == card.ID
		} )
	if err != nil { return }
	err = tx.Delete( db.BucketCards , cardKey( card.BoardID , card.ID ) )
	return
}

// ShareCardWithUser grants (or with mode "" removes) one person's access to
// one card, keeping the card and the per-user index in step.
func ShareCardWithUser( store *db.Store , board_id uint64 , card_id uint64 , user_id uint64 , mode string , actor_name string ) ( card *Card , err error ) {
	if mode != "" && mode != ModeView && mode != ModeEdit {
		err = ErrBadInput
		return
	}
	err = store.Write( func( tx *db.Tx ) ( tx_err error ) {
		board , tx_err := getBoardTx( tx , board_id )
		if tx_err != nil { return }
		card , tx_err = getCardTx( tx , board_id , card_id )
		if tx_err != nil { return }
		user := &User{}
		if tx_err = tx.Get( db.BucketUsers , encryption.Uint64ToBytes( user_id ) , user ); tx_err != nil { return }
		if mode == "" {
			delete( card.SharedWith , user_id )
			if board.RoleOf( user_id ) == "" { card.Assignees = removeID( card.Assignees , user_id ) }
			tx_err = tx.Delete( db.BucketUserCards , db.Key( user_id , board_id , card_id ) )
		} else {
			card.SharedWith[ user_id ] = mode
			tx_err = tx.Put( db.BucketUserCards , db.Key( user_id , board_id , card_id ) , &CardRef{ Mode: mode } )
		}
		if tx_err != nil { return }
		tx_err = saveCardTx( tx , board , card , actor_name )
		return
	} )
	if err != nil { card = nil }
	return
}
