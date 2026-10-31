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

// Roles a person can hold on one board. The owner is not stored in Members --
// OwnerID says who it is -- so there is exactly one place that answers "who
// owns this", and it cannot disagree with a members map.
const (
	BoardRoleOwner  = "owner"
	BoardRoleEditor = "editor"
	BoardRoleViewer = "viewer"
)

// Views a board can open in. Every board has all three; DefaultView is only
// what it opens in for someone who has not picked one.
const (
	ViewKanban   = "kanban"
	ViewTodo     = "todo"
	ViewTimeline = "timeline"
)

// Layouts. Free is any number of lists; Three pins the board to its first
// three lists (to do / doing / done), shown as equal columns. New boards
// start as Three.
const (
	LayoutFree  = ""
	LayoutThree = "three"
)

// Card numbering. Default leaves it to each viewer's account setting; On and
// Off decide it for everyone on the board.
const (
	NumberingDefault = ""
	NumberingOn      = "on"
	NumberingOff     = "off"
)

// Where finished cards go and where the add-card box sits. Default leaves
// it to each viewer's account setting; Top and Bottom decide it for
// everyone on the board.
const (
	PlacementDefault = ""
	PlacementTop     = "top"
	PlacementBottom  = "bottom"
)

func validPlacement( value string ) ( result bool ) {
	result = value == PlacementDefault || value == PlacementTop || value == PlacementBottom
	return
}

var (
	ErrBoardNotFound = errors.New( "board not found" )
	ErrListNotFound  = errors.New( "list not found" )
	ErrLabelNotFound = errors.New( "label not found" )
	ErrBadInput      = errors.New( "invalid input" )
)

// Board holds its lists and labels inline: they are small, always read
// together with the board, and edited rarely compared to cards. Cards live in
// their own bucket (see card.go) so that a move or an edit rewrites one card
// rather than the whole board.
//
// Rev increases on every change to the board *or any of its cards*, in the
// same transaction. It is what a client polls (GET .../stamp) to learn that
// something changed without downloading the board.
type Board struct {
	ID          uint64            `json:"id"`
	Title       string            `json:"title"`
	Slug        string            `json:"slug"`
	Layout      string            `json:"layout"`
	Numbering   string            `json:"numbering"`
	// DonePlacement: where a card marked done, or a ticked checklist item,
	// lands in its list. AddPlacement: where the add-card box sits.
	DonePlacement string          `json:"done_placement"`
	AddPlacement  string          `json:"add_placement"`
	OwnerID     uint64            `json:"owner_id"`
	Members     map[uint64]string `json:"members"`
	Lists       []*List           `json:"lists"`
	Labels      []*Label          `json:"labels"`
	DefaultView string            `json:"default_view"`
	Color       string            `json:"color"`
	Theme       *Theme            `json:"theme"`
	// Shopping: product links from any shop are looked up on every column
	// (List.Shopping does one column).
	Shopping    bool              `json:"shopping"`
	Rev         uint64            `json:"rev"`
	NextLocalID uint64            `json:"next_local_id"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
}

type List struct {
	ID       uint64  `json:"id"`
	Title    string  `json:"title"`
	Position float64 `json:"position"`
	Archived bool    `json:"archived"`
	// Shopping: any shop's product link in a card here is looked up, not
	// only Amazon's (see product.go).
	Shopping bool    `json:"shopping"`
}

type Label struct {
	ID    uint64 `json:"id"`
	Name  string `json:"name"`
	Color string `json:"color"`
}

// RoleOf answers which role user_id holds here, or "" for none.
func ( board *Board ) RoleOf( user_id uint64 ) ( role string ) {
	if user_id == 0 { return }
	if user_id == board.OwnerID {
		role = BoardRoleOwner
		return
	}
	role = board.Members[ user_id ]
	return
}

func ( board *Board ) List( list_id uint64 ) ( list *List ) {
	for _ , candidate := range board.Lists {
		if candidate.ID == list_id {
			list = candidate
			return
		}
	}
	return
}

func ( board *Board ) Label( label_id uint64 ) ( label *Label ) {
	for _ , candidate := range board.Labels {
		if candidate.ID == label_id {
			label = candidate
			return
		}
	}
	return
}

func ( board *Board ) nextID() ( id uint64 ) {
	board.NextLocalID += 1
	id = board.NextLocalID
	return
}

// SortedLists returns lists in display order without mutating the board.
func ( board *Board ) SortedLists() ( lists []*List ) {
	lists = append( []*List{} , board.Lists... )
	sort.SliceStable( lists , func( i int , j int ) bool { return lists[ i ].Position < lists[ j ].Position } )
	return
}

func ValidBoardView( view string ) ( result bool ) {
	result = view == ViewKanban || view == ViewTodo || view == ViewTimeline
	return
}

func ValidMemberRole( role string ) ( result bool ) {
	result = role == BoardRoleEditor || role == BoardRoleViewer
	return
}

// cleanTitle trims and bounds the short free-text names (board, list, label,
// card titles). The bound is in characters, not bytes, so it does not cut a
// multi-byte character in half.
func cleanTitle( value string , max_chars int ) ( result string , ok bool ) {
	result = strings.TrimSpace( value )
	ok = result != "" && utf8.RuneCountInString( result ) <= max_chars
	return
}

// Colors are stored as a small closed set of names rather than arbitrary CSS,
// so nothing a user types ever ends up inside a style.
var boardColors = map[string]bool{
	"": true, "blue": true, "green": true, "orange": true, "red": true,
	"purple": true, "pink": true, "teal": true, "gray": true, "yellow": true,
}

func ValidColor( color string ) ( result bool ) {
	result = boardColors[ color ]
	return
}

func boardKey( board_id uint64 ) ( key []byte ) {
	key = encryption.Uint64ToBytes( board_id )
	return
}

// CreateBoard makes a board owned by owner_id with the given lists, in order.
// List names come from the client (language.yaml), not from here.
func CreateBoard( store *db.Store , owner_id uint64 , title string , list_titles []string ) ( board *Board , err error ) {
	clean , ok := cleanTitle( title , 120 )
	if ok == false {
		err = ErrBadInput
		return
	}
	now := time.Now().UTC()
	err = store.Write( func( tx *db.Tx ) ( tx_err error ) {
		id , tx_err := tx.NextSequence( db.BucketBoards )
		if tx_err != nil { return }
		board = &Board{
			ID:          id,
			Title:       clean,
			OwnerID:     owner_id,
			Members:     map[uint64]string{},
			Lists:       []*List{},
			Labels:      []*Label{},
			Layout:      LayoutThree,
			DefaultView: ViewKanban,
			Rev:         1,
			CreatedAt:   now,
			UpdatedAt:   now,
		}
		for index , name := range list_titles {
			list_title , list_ok := cleanTitle( name , 120 )
			if list_ok == false || index >= 20 { continue }
			board.Lists = append( board.Lists , &List{ ID: board.nextID() , Title: list_title , Position: float64( index + 1 ) * positionStep } )
		}
		// A caller that named fewer than three lists still gets three.
		if tx_err = ensureThreeLists( board , list_titles ); tx_err != nil { return }
		if tx_err = claimFreeSlugTx( tx , board , Slugify( clean ) ); tx_err != nil { return }
		if tx_err = tx.Put( db.BucketBoards , boardKey( id ) , board ); tx_err != nil { return }
		tx_err = tx.Put( db.BucketUserBoards , db.Key( owner_id , id ) , &BoardRef{ Role: BoardRoleOwner } )
		return
	} )
	if err != nil { board = nil }
	return
}

func GetBoard( store *db.Store , board_id uint64 ) ( board *Board , err error ) {
	board = &Board{}
	if err = store.Get( db.BucketBoards , boardKey( board_id ) , board ); err != nil {
		board = nil
		if errors.Is( err , db.ErrNotFound ) { err = ErrBoardNotFound }
		return
	}
	normalizeBoard( board )
	return
}

func normalizeBoard( board *Board ) {
	if board.Members == nil { board.Members = map[uint64]string{} }
	if board.Lists == nil { board.Lists = []*List{} }
	if board.Labels == nil { board.Labels = []*Label{} }
}

func getBoardTx( tx *db.Tx , board_id uint64 ) ( board *Board , err error ) {
	board = &Board{}
	if err = tx.Get( db.BucketBoards , boardKey( board_id ) , board ); err != nil {
		board = nil
		if errors.Is( err , db.ErrNotFound ) { err = ErrBoardNotFound }
		return
	}
	normalizeBoard( board )
	return
}

// putBoardTx stamps the change and writes the board. Every write of a board
// goes through here, which is what keeps Rev honest.
func putBoardTx( tx *db.Tx , board *Board ) ( err error ) {
	board.Rev += 1
	board.UpdatedAt = time.Now().UTC()
	err = tx.Put( db.BucketBoards , boardKey( board.ID ) , board )
	return
}

// touchBoardTx bumps a board's Rev for a change that happened to one of its
// cards.
func touchBoardTx( tx *db.Tx , board_id uint64 ) ( err error ) {
	board , err := getBoardTx( tx , board_id )
	if err != nil { return }
	err = putBoardTx( tx , board )
	return
}

// MutateBoard is the one path for edits to a board's own fields, lists and
// labels: load, change, stamp, write, all in one transaction.
func MutateBoard( store *db.Store , board_id uint64 , mutate func( board *Board ) error ) ( board *Board , err error ) {
	err = store.Write( func( tx *db.Tx ) ( tx_err error ) {
		board , tx_err = getBoardTx( tx , board_id )
		if tx_err != nil { return }
		if tx_err = mutate( board ); tx_err != nil { return }
		tx_err = putBoardTx( tx , board )
		return
	} )
	if err != nil { board = nil }
	return
}

// BoardSettings is a partial update; nil means "leave alone".
//
// Slug "" means "make one from the title again". ListTitles names the lists
// to add when switching to LayoutThree leaves the board with fewer than three
// -- the names come from the client (language.yaml), like at creation.
// ClearTheme drops the board's theme so each person sees their own again.
type BoardSettings struct {
	Title       *string  `json:"title"`
	DefaultView *string  `json:"default_view"`
	Color       *string  `json:"color"`
	Slug        *string  `json:"slug"`
	Layout      *string  `json:"layout"`
	Numbering   *string  `json:"numbering"`
	ListTitles  []string `json:"list_titles"`
	Theme       *Theme   `json:"theme"`
	ClearTheme  bool     `json:"clear_theme"`
	Shopping    *bool    `json:"shopping"`
	DonePlacement *string `json:"done_placement"`
	AddPlacement  *string `json:"add_placement"`
}

// UpdateBoardSettings applies settings in one transaction; the slug index
// has to change together with the board.
func UpdateBoardSettings( store *db.Store , board_id uint64 , settings *BoardSettings ) ( board *Board , err error ) {
	err = store.Write( func( tx *db.Tx ) ( tx_err error ) {
		board , tx_err = getBoardTx( tx , board_id )
		if tx_err != nil { return }
		if tx_err = settings.Apply( board ); tx_err != nil { return }
		if settings.Slug != nil {
			slug := strings.ToLower( strings.TrimSpace( *settings.Slug ) )
			if slug == "" {
				tx_err = claimFreeSlugTx( tx , board , Slugify( board.Title ) )
			} else {
				tx_err = claimSlugTx( tx , board , slug )
			}
			if tx_err != nil { return }
		}
		tx_err = putBoardTx( tx , board )
		return
	} )
	if err != nil { board = nil }
	return
}

func ( settings *BoardSettings ) Apply( board *Board ) ( err error ) {
	if settings.Title != nil {
		clean , ok := cleanTitle( *settings.Title , 120 )
		if ok == false { return ErrBadInput }
		board.Title = clean
	}
	if settings.DefaultView != nil {
		if ValidBoardView( *settings.DefaultView ) == false { return ErrBadInput }
		board.DefaultView = *settings.DefaultView
	}
	if settings.Color != nil {
		if ValidColor( *settings.Color ) == false { return ErrBadInput }
		board.Color = *settings.Color
	}
	if settings.Theme != nil {
		if err = settings.Theme.Validate(); err != nil { return }
		board.Theme = settings.Theme
	}
	if settings.ClearTheme { board.Theme = nil }
	if settings.Shopping != nil { board.Shopping = *settings.Shopping }
	if settings.Layout != nil {
		if *settings.Layout != LayoutFree && *settings.Layout != LayoutThree { return ErrBadInput }
		board.Layout = *settings.Layout
		if board.Layout == LayoutThree { err = ensureThreeLists( board , settings.ListTitles ) }
	}
	if settings.Numbering != nil {
		value := *settings.Numbering
		if value != NumberingDefault && value != NumberingOn && value != NumberingOff { return ErrBadInput }
		board.Numbering = value
	}
	if settings.DonePlacement != nil {
		if validPlacement( *settings.DonePlacement ) == false { return ErrBadInput }
		board.DonePlacement = *settings.DonePlacement
	}
	if settings.AddPlacement != nil {
		if validPlacement( *settings.AddPlacement ) == false { return ErrBadInput }
		board.AddPlacement = *settings.AddPlacement
	}
	return
}

// ensureThreeLists makes sure a three-column board has three open lists to
// show, adding them from titles. Extra lists are left alone: the layout only
// hides them, so switching back brings them back.
func ensureThreeLists( board *Board , titles []string ) ( err error ) {
	open := 0
	for _ , list := range board.Lists {
		if list.Archived == false { open += 1 }
	}
	for index := open; index < 3; index += 1 {
		title := []string{ "To do" , "Doing" , "Done" }[ index ]
		if index < len( titles ) && strings.TrimSpace( titles[ index ] ) != "" { title = titles[ index ] }
		if _ , err = AddList( board , title , nil ); err != nil { return }
	}
	return
}

// AddList appends a list (or places it at position when given).
func AddList( board *Board , title string , position *float64 ) ( list *List , err error ) {
	clean , ok := cleanTitle( title , 120 )
	if ok == false || len( board.Lists ) >= 100 {
		err = ErrBadInput
		return
	}
	list = &List{ ID: board.nextID() , Title: clean }
	if position != nil {
		list.Position = *position
	} else {
		list.Position = positionStep
		for _ , existing := range board.Lists {
			if existing.Position >= list.Position { list.Position = existing.Position + positionStep }
		}
	}
	board.Lists = append( board.Lists , list )
	renormalizeLists( board )
	return
}

type ListUpdate struct {
	Title    *string  `json:"title"`
	Position *float64 `json:"position"`
	Archived *bool    `json:"archived"`
	Shopping *bool    `json:"shopping"`
}

func ( update *ListUpdate ) Apply( board *Board , list_id uint64 ) ( err error ) {
	list := board.List( list_id )
	if list == nil { return ErrListNotFound }
	if update.Title != nil {
		clean , ok := cleanTitle( *update.Title , 120 )
		if ok == false { return ErrBadInput }
		list.Title = clean
	}
	if update.Position != nil {
		list.Position = *update.Position
		renormalizeLists( board )
	}
	if update.Archived != nil {
		list.Archived = *update.Archived
	}
	if update.Shopping != nil {
		list.Shopping = *update.Shopping
	}
	return
}

func renormalizeLists( board *Board ) {
	lists := board.SortedLists()
	positions := make( []float64 , len( lists ) )
	for index , list := range lists { positions[ index ] = list.Position }
	if needsRenormalize( positions ) == false { return }
	for index , list := range lists { list.Position = float64( index + 1 ) * positionStep }
}

func AddLabel( board *Board , name string , color string ) ( label *Label , err error ) {
	clean := strings.TrimSpace( name )
	if utf8.RuneCountInString( clean ) > 40 || ValidColor( color ) == false || color == "" || len( board.Labels ) >= 50 {
		err = ErrBadInput
		return
	}
	label = &Label{ ID: board.nextID() , Name: clean , Color: color }
	board.Labels = append( board.Labels , label )
	return
}

func UpdateLabel( board *Board , label_id uint64 , name *string , color *string ) ( err error ) {
	label := board.Label( label_id )
	if label == nil { return ErrLabelNotFound }
	if name != nil {
		clean := strings.TrimSpace( *name )
		if utf8.RuneCountInString( clean ) > 40 { return ErrBadInput }
		label.Name = clean
	}
	if color != nil {
		if ValidColor( *color ) == false || *color == "" { return ErrBadInput }
		label.Color = *color
	}
	return
}

// DeleteLabel removes the label and strips it from every card, in one
// transaction, so no card is left pointing at a label that no longer exists.
func DeleteLabel( store *db.Store , board_id uint64 , label_id uint64 ) ( err error ) {
	err = store.Write( func( tx *db.Tx ) ( tx_err error ) {
		board , tx_err := getBoardTx( tx , board_id )
		if tx_err != nil { return }
		kept := []*Label{}
		for _ , label := range board.Labels {
			if label.ID != label_id { kept = append( kept , label ) }
		}
		if len( kept ) == len( board.Labels ) { return ErrLabelNotFound }
		board.Labels = kept
		tx_err = eachCardTx( tx , board_id , func( card *Card ) ( changed bool ) {
			before := len( card.LabelIDs )
			card.LabelIDs = removeID( card.LabelIDs , label_id )
			changed = len( card.LabelIDs ) != before
			return
		} )
		if tx_err != nil { return }
		tx_err = putBoardTx( tx , board )
		return
	} )
	return
}

// DeleteList removes a list and its share links. Its cards go to the
// board's trash, so they can still be restored (into another column).
func DeleteList( store *db.Store , board_id uint64 , list_id uint64 , actor_name string ) ( err error ) {
	err = store.Write( func( tx *db.Tx ) ( tx_err error ) {
		board , tx_err := getBoardTx( tx , board_id )
		if tx_err != nil { return }
		kept := []*List{}
		for _ , list := range board.Lists {
			if list.ID != list_id { kept = append( kept , list ) }
		}
		if len( kept ) == len( board.Lists ) { return ErrListNotFound }
		board.Lists = kept

		doomed := []*Card{}
		tx_err = tx.ForEachPrefix( db.BucketCards , db.Key( board_id ) ,
			func() any { return &Card{} } ,
			func( key []byte , item any ) bool {
				if card := item.( *Card ); card.ListID == list_id { doomed = append( doomed , card ) }
				return true
			} )
		if tx_err != nil { return }
		for _ , card := range doomed {
			if tx_err = trashCardTx( tx , card , actor_name ); tx_err != nil { return }
		}
		_ , tx_err = tx.DeleteWhere( db.BucketShareLinks , func() any { return &ShareLink{} } ,
			func( key []byte , item any ) bool {
				link := item.( *ShareLink )
				return link.BoardID == board_id && link.ListID == list_id
			} )
		if tx_err != nil { return }
		tx_err = putBoardTx( tx , board )
		return
	} )
	return
}

// BoardRef is the per-user index row: which boards can this person open, and
// as what. The board's own OwnerID/Members are the source of truth; this is
// kept in step with them in the same transaction so listing "my boards" is a
// prefix scan instead of a read of every board.
//
// Folder is where this person files the board on their home page (0: not
// filed). It is the one thing here that is theirs rather than the board's,
// which is why it lives on this per-user row -- see folder.go.
//
// PlainIcons is this person's choice for this board's Doing / Done icons --
// greyscale or colour -- also theirs alone, so also here. nil follows their
// account default (UserPrefs.PlainIcons).
type BoardRef struct {
	Role       string `json:"role"`
	Folder     uint64 `json:"folder"`
	PlainIcons *bool  `json:"plain_icons"`
}

// GetBoardRef is one person's row for one board, or nil when they are not on
// it.
func GetBoardRef( store *db.Store , user_id uint64 , board_id uint64 ) ( ref *BoardRef ) {
	found := &BoardRef{}
	if store.Get( db.BucketUserBoards , db.Key( user_id , board_id ) , found ) != nil { return }
	ref = found
	return
}

// SetBoardPlainIcons stores whether this person sees the board's status
// icons in greyscale; nil goes back to their account default.
func SetBoardPlainIcons( store *db.Store , user_id uint64 , board_id uint64 , plain *bool ) ( err error ) {
	err = store.Write( func( tx *db.Tx ) ( tx_err error ) {
		ref := &BoardRef{}
		if tx_err = tx.Get( db.BucketUserBoards , db.Key( user_id , board_id ) , ref ); tx_err != nil {
			if errors.Is( tx_err , db.ErrNotFound ) { tx_err = ErrBoardNotFound }
			return
		}
		ref.PlainIcons = plain
		tx_err = tx.Put( db.BucketUserBoards , db.Key( user_id , board_id ) , ref )
		return
	} )
	return
}

// SetMember adds, changes, or (role "") removes a member.
func SetMember( store *db.Store , board_id uint64 , user_id uint64 , role string ) ( board *Board , err error ) {
	if role != "" && ValidMemberRole( role ) == false {
		err = ErrBadInput
		return
	}
	err = store.Write( func( tx *db.Tx ) ( tx_err error ) {
		board , tx_err = getBoardTx( tx , board_id )
		if tx_err != nil { return }
		if user_id == board.OwnerID { return ErrBadInput }
		if role == "" {
			delete( board.Members , user_id )
			if tx_err = tx.Delete( db.BucketUserBoards , db.Key( user_id , board_id ) ); tx_err != nil { return }
			if tx_err = deleteSubscriptionTx( tx , user_id , subscriptionBoard , board_id , 0 ); tx_err != nil { return }
			tx_err = unassignTx( tx , board_id , user_id )
			if tx_err != nil { return }
		} else {
			user := &User{}
			if tx_err = tx.Get( db.BucketUsers , encryption.Uint64ToBytes( user_id ) , user ); tx_err != nil { return }
			board.Members[ user_id ] = role
			// A change of role keeps where they filed the board.
			ref := &BoardRef{}
			tx.Get( db.BucketUserBoards , db.Key( user_id , board_id ) , ref )
			ref.Role = role
			if tx_err = tx.Put( db.BucketUserBoards , db.Key( user_id , board_id ) , ref ); tx_err != nil { return }
		}
		tx_err = putBoardTx( tx , board )
		return
	} )
	if err != nil { board = nil }
	return
}

// BoardSummary is one row of the home page.
type BoardSummary struct {
	ID      uint64    `json:"id"`
	Title   string    `json:"title"`
	Slug    string    `json:"slug"`
	Role    string    `json:"role"`
	Color   string    `json:"color"`
	OwnerID uint64    `json:"owner_id"`
	Folder  uint64    `json:"folder_id"`
	Updated time.Time `json:"updated_at"`
}

// SharedCardSummary is a card someone shared with this user on a board they
// are not otherwise a member of.
type SharedCardSummary struct {
	BoardID    uint64 `json:"board_id"`
	BoardTitle string `json:"board_title"`
	BoardSlug  string `json:"board_slug"`
	CardID     uint64 `json:"card_id"`
	CardTitle  string `json:"card_title"`
	Mode       string `json:"mode"`
}

func ListBoardsForUser( store *db.Store , user_id uint64 ) ( boards []*BoardSummary , cards []*SharedCardSummary , err error ) {
	boards = []*BoardSummary{}
	cards = []*SharedCardSummary{}
	refs := map[uint64]*BoardRef{}
	err = store.ForEachPrefix( db.BucketUserBoards , db.Key( user_id ) ,
		func() any { return &BoardRef{} } ,
		func( key []byte , item any ) bool {
			refs[ encryption.BytesToUint64( key[ 8:16 ] ) ] = item.( *BoardRef )
			return true
		} )
	if err != nil { return }
	for board_id , ref := range refs {
		board , get_err := GetBoard( store , board_id )
		if get_err != nil { continue }
		role := board.RoleOf( user_id )
		if role == "" { continue }
		boards = append( boards , &BoardSummary{ ID: board.ID , Title: board.Title , Slug: board.Slug , Role: role , Color: board.Color , OwnerID: board.OwnerID , Folder: ref.Folder , Updated: board.UpdatedAt } )
	}
	sort.Slice( boards , func( i int , j int ) bool { return boards[ i ].Updated.After( boards[ j ].Updated ) } )

	type cardRef struct{ board_id , card_id uint64 ; mode string }
	card_refs := []cardRef{}
	err = store.ForEachPrefix( db.BucketUserCards , db.Key( user_id ) ,
		func() any { return &CardRef{} } ,
		func( key []byte , item any ) bool {
			card_refs = append( card_refs , cardRef{ encryption.BytesToUint64( key[ 8:16 ] ) , encryption.BytesToUint64( key[ 16:24 ] ) , item.( *CardRef ).Mode } )
			return true
		} )
	if err != nil { return }
	for _ , ref := range card_refs {
		if _ , member := refs[ ref.board_id ]; member { continue }
		board , board_err := GetBoard( store , ref.board_id )
		if board_err != nil { continue }
		card , card_err := GetCard( store , ref.board_id , ref.card_id )
		if card_err != nil { continue }
		cards = append( cards , &SharedCardSummary{ BoardID: board.ID , BoardTitle: board.Title , BoardSlug: board.Slug , CardID: card.ID , CardTitle: card.Title , Mode: ref.mode } )
	}
	return
}

// DeleteBoard removes the board and everything hanging off it: cards, index
// rows, share links, and subscriptions.
func DeleteBoard( store *db.Store , board_id uint64 ) ( err error ) {
	err = store.Write( func( tx *db.Tx ) ( tx_err error ) {
		tx_err = deleteBoardTx( tx , board_id )
		return
	} )
	return
}

func deleteBoardTx( tx *db.Tx , board_id uint64 ) ( err error ) {
	board , err := getBoardTx( tx , board_id )
	if err != nil { return }

	cards := []*Card{}
	err = tx.ForEachPrefix( db.BucketCards , db.Key( board_id ) , func() any { return &Card{} } ,
		func( key []byte , item any ) bool {
			cards = append( cards , item.( *Card ) )
			return true
		} )
	if err != nil { return }
	for _ , card := range cards {
		if err = deleteCardTx( tx , card ); err != nil { return }
	}
	if _ , err = tx.DeletePrefix( db.BucketTrash , db.Key( board_id ) ); err != nil { return }
	if _ , err = tx.DeletePrefix( db.BucketProductImages , db.Key( board_id ) ); err != nil { return }
	if _ , err = tx.DeletePrefix( db.BucketCardImages , db.Key( board_id ) ); err != nil { return }

	if err = tx.Delete( db.BucketUserBoards , db.Key( board.OwnerID , board_id ) ); err != nil { return }
	for user_id := range board.Members {
		if err = tx.Delete( db.BucketUserBoards , db.Key( user_id , board_id ) ); err != nil { return }
	}
	_ , err = tx.DeleteWhere( db.BucketSubscriptions , func() any { return &Subscription{} } ,
		func( key []byte , item any ) bool { return item.( *Subscription ).BoardID == board_id } )
	if err != nil { return }

	_ , err = tx.DeleteWhere( db.BucketShareLinks , func() any { return &ShareLink{} } ,
		func( key []byte , item any ) bool { return item.( *ShareLink ).BoardID == board_id } )
	if err != nil { return }
	if board.Slug != "" {
		if err = tx.Delete( db.BucketBoardSlugs , tx.LookupKey( board.Slug ) ); err != nil { return }
	}
	err = tx.Delete( db.BucketBoards , boardKey( board_id ) )
	return
}

// removeBoardAccessTx is the board half of deleting an account: memberships
// and card grants go, and boards the person owned pass to heir_id, so that
// deleting someone never silently deletes their team's work.
func removeBoardAccessTx( tx *db.Tx , user_id uint64 , heir_id uint64 ) ( err error ) {
	refs := map[uint64]string{}
	err = tx.ForEachPrefix( db.BucketUserBoards , db.Key( user_id ) , func() any { return &BoardRef{} } ,
		func( key []byte , item any ) bool {
			refs[ encryption.BytesToUint64( key[ 8:16 ] ) ] = item.( *BoardRef ).Role
			return true
		} )
	if err != nil { return }
	for board_id := range refs {
		board , get_err := getBoardTx( tx , board_id )
		if get_err != nil { continue }
		delete( board.Members , user_id )
		if board.OwnerID == user_id {
			if heir_id == 0 || heir_id == user_id {
				if err = deleteBoardTx( tx , board_id ); err != nil { return }
				continue
			}
			delete( board.Members , heir_id )
			board.OwnerID = heir_id
			if err = tx.Put( db.BucketUserBoards , db.Key( heir_id , board_id ) , &BoardRef{ Role: BoardRoleOwner } ); err != nil { return }
		}
		if err = unassignTx( tx , board_id , user_id ); err != nil { return }
		if err = putBoardTx( tx , board ); err != nil { return }
	}
	if _ , err = tx.DeletePrefix( db.BucketUserBoards , db.Key( user_id ) ); err != nil { return }

	// Card grants: drop them from the cards themselves as well as the index.
	card_keys := [][]byte{}
	err = tx.ForEachPrefix( db.BucketUserCards , db.Key( user_id ) , func() any { return &CardRef{} } ,
		func( key []byte , item any ) bool {
			card_keys = append( card_keys , append( []byte( nil ) , key[ 8:24 ]... ) )
			return true
		} )
	if err != nil { return }
	for _ , card_key := range card_keys {
		card := &Card{}
		if tx.Get( db.BucketCards , card_key , card ) != nil { continue }
		delete( card.SharedWith , user_id )
		card.Assignees = removeID( card.Assignees , user_id )
		if err = tx.Put( db.BucketCards , card_key , card ); err != nil { return }
	}
	if _ , err = tx.DeletePrefix( db.BucketUserCards , db.Key( user_id ) ); err != nil { return }
	if _ , err = tx.DeletePrefix( db.BucketFolders , db.Key( user_id ) ); err != nil { return }
	if _ , err = tx.DeletePrefix( db.BucketSubscriptions , db.Key( user_id ) ); err != nil { return }
	err = tx.Delete( db.BucketUserPrefs , encryption.Uint64ToBytes( user_id ) )
	return
}

// unassignTx takes someone off every card of a board they can no longer see
// -- except cards still shared with them directly.
func unassignTx( tx *db.Tx , board_id uint64 , user_id uint64 ) ( err error ) {
	err = eachCardTx( tx , board_id , func( card *Card ) ( changed bool ) {
		if card.SharedWith[ user_id ] != "" || containsID( card.Assignees , user_id ) == false { return }
		card.Assignees = removeID( card.Assignees , user_id )
		changed = true
		return
	} )
	return
}

func removeID( ids []uint64 , target uint64 ) ( result []uint64 ) {
	result = []uint64{}
	for _ , id := range ids {
		if id != target { result = append( result , id ) }
	}
	return
}

func containsID( ids []uint64 , target uint64 ) ( result bool ) {
	for _ , id := range ids {
		if id == target { return true }
	}
	return
}
