package models

import (
	sort "sort"
	strings "strings"
	time "time"
	utf8 "unicode/utf8"

	db "todoboard/server/db"
)

// Backups: a board, or one column, as a JSON file someone can keep and bring
// back. A snapshot carries no server IDs -- labels are matched by a ref that
// only means something inside the file -- and no people: members, share
// links and assignees belong to a server, so an import never restores them
// (assignee names are written for whoever reads the file).
//
// Importing always makes something new: a backup becomes new boards owned by
// the importer, a column file appends cards to a column. Nothing existing is
// overwritten.

const (
	BackupFormat    = "todoboard-backup"
	ColumnFormat    = "todoboard-column"
	SnapshotVersion = 1

	// maxImportCards bounds one import request; the client sends a big column
	// in several.
	maxImportCards = 2000
)

type LabelSnapshot struct {
	Ref   uint64 `json:"ref"`
	Name  string `json:"name"`
	Color string `json:"color"`
}

type ChecklistSnapshot struct {
	Text string `json:"text"`
	Done bool   `json:"done"`
}

type CardSnapshot struct {
	Title       string               `json:"title"`
	Description string               `json:"description"`
	Labels      []uint64             `json:"labels"`
	StartAt     *time.Time           `json:"start_at"`
	StartTimed  bool                 `json:"start_timed"`
	DueAt       *time.Time           `json:"due_at"`
	DueAllDay   bool                 `json:"due_all_day"`
	Done        bool                 `json:"done"`
	DoneAt      *time.Time           `json:"done_at"`
	Archived    bool                 `json:"archived"`
	Checklist   []*ChecklistSnapshot `json:"checklist"`
	Assignees   []string             `json:"assignees,omitempty"`
}

type ListSnapshot struct {
	Title    string          `json:"title"`
	Archived bool            `json:"archived"`
	Cards    []*CardSnapshot `json:"cards"`
}

type BoardSnapshot struct {
	Title       string           `json:"title"`
	Slug        string           `json:"slug"`
	Layout      string           `json:"layout"`
	Numbering   string           `json:"numbering"`
	DefaultView string           `json:"default_view"`
	Color       string           `json:"color"`
	Theme       *Theme           `json:"theme"`
	Labels      []*LabelSnapshot `json:"labels"`
	Lists       []*ListSnapshot  `json:"lists"`
}

// Backup is the whole-account file.
type Backup struct {
	Format     string           `json:"format"`
	Version    int              `json:"version"`
	ExportedAt time.Time        `json:"exported_at"`
	Boards     []*BoardSnapshot `json:"boards"`
}

// ColumnBackup is one column's file. Labels are the ones its cards use.
type ColumnBackup struct {
	Format     string           `json:"format"`
	Version    int              `json:"version"`
	ExportedAt time.Time        `json:"exported_at"`
	Labels     []*LabelSnapshot `json:"labels"`
	List       *ListSnapshot    `json:"list"`
}

// SnapshotBoard writes board as a snapshot: lists in board order, each with
// the cards visible() lets through in column order. only_list, when not 0,
// keeps just that list. person names an assignee.
func SnapshotBoard( board *Board , cards []*Card , visible func( card *Card ) bool , only_list uint64 , person func( id uint64 ) string ) ( snap *BoardSnapshot ) {
	snap = &BoardSnapshot{
		Title: board.Title , Slug: board.Slug , Layout: board.Layout , Numbering: board.Numbering ,
		DefaultView: board.DefaultView , Color: board.Color , Theme: board.Theme ,
		Labels: snapshotLabels( board , nil ) , Lists: []*ListSnapshot{} ,
	}
	for _ , list := range board.SortedLists() {
		if only_list != 0 && list.ID != only_list { continue }
		snap.Lists = append( snap.Lists , SnapshotList( list , cards , visible , person ) )
	}
	return
}

// SnapshotList is one list with its visible cards, in column order.
func SnapshotList( list *List , cards []*Card , visible func( card *Card ) bool , person func( id uint64 ) string ) ( snap *ListSnapshot ) {
	snap = &ListSnapshot{ Title: list.Title , Archived: list.Archived , Cards: []*CardSnapshot{} }
	mine := []*Card{}
	for _ , card := range cards {
		if card.ListID == list.ID && visible( card ) { mine = append( mine , card ) }
	}
	sort.SliceStable( mine , func( i int , j int ) bool { return mine[ i ].Position < mine[ j ].Position } )
	for _ , card := range mine { snap.Cards = append( snap.Cards , snapshotCard( card , person ) ) }
	return
}

// SnapshotColumn is a column file for list.
func SnapshotColumn( board *Board , list *List , cards []*Card , visible func( card *Card ) bool , person func( id uint64 ) string ) ( backup *ColumnBackup ) {
	snap := SnapshotList( list , cards , visible , person )
	used := map[uint64]bool{}
	for _ , card := range cards {
		if card.ListID != list.ID || visible( card ) == false { continue }
		for _ , id := range card.LabelIDs { used[ id ] = true }
	}
	backup = &ColumnBackup{
		Format: ColumnFormat , Version: SnapshotVersion , ExportedAt: time.Now().UTC() ,
		Labels: snapshotLabels( board , used ) , List: snap ,
	}
	return
}

func snapshotLabels( board *Board , only map[uint64]bool ) ( labels []*LabelSnapshot ) {
	labels = []*LabelSnapshot{}
	for _ , label := range board.Labels {
		if only != nil && only[ label.ID ] == false { continue }
		labels = append( labels , &LabelSnapshot{ Ref: label.ID , Name: label.Name , Color: label.Color } )
	}
	return
}

func snapshotCard( card *Card , person func( id uint64 ) string ) ( snap *CardSnapshot ) {
	snap = &CardSnapshot{
		Title: card.Title , Description: card.Description , Labels: append( []uint64{} , card.LabelIDs... ) ,
		StartAt: card.StartAt , StartTimed: card.StartTimed , DueAt: card.DueAt , DueAllDay: card.DueAllDay ,
		Done: card.Done , DoneAt: card.DoneAt , Archived: card.Archived , Checklist: []*ChecklistSnapshot{} ,
	}
	items := append( []*ChecklistItem{} , card.Checklist... )
	sort.SliceStable( items , func( i int , j int ) bool { return items[ i ].Position < items[ j ].Position } )
	for _ , item := range items { snap.Checklist = append( snap.Checklist , &ChecklistSnapshot{ Text: item.Text , Done: item.Done } ) }
	if person != nil {
		for _ , id := range card.Assignees { snap.Assignees = append( snap.Assignees , person( id ) ) }
	}
	return
}

// ImportBoardSkeleton makes a new board owned by owner_id from snap's title,
// settings, labels and lists -- not its cards, which arrive afterwards
// through ImportCards, list by list, so no single request has to carry a
// whole board. Settings the server does not know fall back to defaults; the
// slug is the file's if it is free, else the next free one. lists are the
// new list IDs, in the order of snap.Lists.
func ImportBoardSkeleton( store *db.Store , owner_id uint64 , snap *BoardSnapshot ) ( board *Board , lists []uint64 , err error ) {
	title , ok := cleanTitle( snap.Title , 120 )
	if ok == false || len( snap.Lists ) > 100 {
		err = ErrBadInput
		return
	}
	now := time.Now().UTC()
	err = store.Write( func( tx *db.Tx ) ( tx_err error ) {
		id , tx_err := tx.NextSequence( db.BucketBoards )
		if tx_err != nil { return }
		board = &Board{
			ID: id , Title: title , OwnerID: owner_id , Members: map[uint64]string{} ,
			Lists: []*List{} , Labels: []*Label{} , Layout: LayoutFree , DefaultView: ViewKanban ,
			Rev: 1 , CreatedAt: now , UpdatedAt: now ,
		}
		if snap.Layout == LayoutThree { board.Layout = LayoutThree }
		if snap.Numbering == NumberingOn || snap.Numbering == NumberingOff { board.Numbering = snap.Numbering }
		if ValidBoardView( snap.DefaultView ) { board.DefaultView = snap.DefaultView }
		if ValidColor( snap.Color ) { board.Color = snap.Color }
		if snap.Theme != nil && snap.Theme.Validate() == nil { board.Theme = snap.Theme }
		for _ , label := range snap.Labels {
			// A label this server would refuse is dropped; its cards lose it.
			if label != nil && labelMatch( board , label.Name , label.Color ) == nil { AddLabel( board , label.Name , label.Color ) }
		}
		lists = []uint64{}
		for index , list := range snap.Lists {
			if list == nil { return ErrBadInput }
			list_title , list_ok := cleanTitle( list.Title , 120 )
			if list_ok == false { return ErrBadInput }
			added := &List{ ID: board.nextID() , Title: list_title , Position: float64( index + 1 ) * positionStep , Archived: list.Archived }
			board.Lists = append( board.Lists , added )
			lists = append( lists , added.ID )
		}
		if board.Layout == LayoutThree {
			if tx_err = ensureThreeLists( board , nil ); tx_err != nil { return }
		}
		base := snap.Slug
		if ValidSlug( base ) == false { base = Slugify( title ) }
		if tx_err = claimFreeSlugTx( tx , board , base ); tx_err != nil { return }
		if tx_err = tx.Put( db.BucketBoards , boardKey( id ) , board ); tx_err != nil { return }
		tx_err = tx.Put( db.BucketUserBoards , db.Key( owner_id , id ) , &BoardRef{ Role: BoardRoleOwner } )
		return
	} )
	if err != nil {
		board = nil
		lists = nil
	}
	return
}

// ImportCards appends cards to the end of list_id in one transaction: all of
// them or none. labels are the file's; each is matched to the board's label
// of the same name and colour, or added. A card naming a label the file does
// not describe, or one the board cannot take, just goes without it.
func ImportCards( store *db.Store , board_id uint64 , list_id uint64 , labels []*LabelSnapshot , cards []*CardSnapshot , created_by uint64 , actor_name string ) ( count int , err error ) {
	if len( cards ) > maxImportCards { return 0 , ErrBadInput }
	now := time.Now().UTC()
	err = store.Write( func( tx *db.Tx ) ( tx_err error ) {
		board , tx_err := getBoardTx( tx , board_id )
		if tx_err != nil { return }
		if board.List( list_id ) == nil { return ErrListNotFound }
		by_ref := map[uint64]uint64{}
		for _ , label := range labels {
			if label == nil { continue }
			found := labelMatch( board , label.Name , label.Color )
			if found == nil {
				if added , add_err := AddLabel( board , label.Name , label.Color ); add_err == nil { found = added }
			}
			if found != nil { by_ref[ label.Ref ] = found.ID }
		}
		position , tx_err := endOfListTx( tx , board_id , list_id )
		if tx_err != nil { return }
		for _ , snap := range cards {
			if snap == nil { return ErrBadInput }
			card , build_err := importedCard( snap , by_ref )
			if build_err != nil { return build_err }
			id , seq_err := tx.NextSequence( db.BucketCards )
			if seq_err != nil { return seq_err }
			card.ID , card.BoardID , card.ListID , card.Position = id , board_id , list_id , position
			card.CreatedBy , card.CreatedAt , card.UpdatedAt , card.UpdatedBy , card.Rev = created_by , now , now , actor_name , 1
			position += positionStep
			if tx_err = tx.Put( db.BucketCards , cardKey( board_id , id ) , card ); tx_err != nil { return }
			count += 1
		}
		if tx_err = renormalizeCardsTx( tx , board_id , list_id ); tx_err != nil { return }
		tx_err = putBoardTx( tx , board )
		return
	} )
	if err != nil { count = 0 }
	return
}

// importedCard checks snap the way editing a card would and builds the card.
func importedCard( snap *CardSnapshot , by_ref map[uint64]uint64 ) ( card *Card , err error ) {
	title , ok := cleanTitle( snap.Title , maxCardTitleChars )
	if ok == false || len( snap.Description ) > maxDescriptionBytes || utf8.ValidString( snap.Description ) == false || len( snap.Checklist ) > maxChecklistItems {
		err = ErrBadInput
		return
	}
	card = &Card{
		Title: title , Description: snap.Description ,
		StartAt: utcTime( snap.StartAt ) , StartTimed: snap.StartTimed && snap.StartAt != nil ,
		DueAt: utcTime( snap.DueAt ) , DueAllDay: snap.DueAllDay && snap.DueAt != nil ,
		Done: snap.Done , Archived: snap.Archived ,
	}
	if snap.Done {
		card.DoneAt = utcTime( snap.DoneAt )
		if card.DoneAt == nil {
			now := time.Now().UTC()
			card.DoneAt = &now
		}
	}
	normalizeCard( card )
	seen := map[uint64]bool{}
	for _ , ref := range snap.Labels {
		if id , known := by_ref[ ref ]; known && seen[ id ] == false {
			seen[ id ] = true
			card.LabelIDs = append( card.LabelIDs , id )
		}
	}
	for _ , item := range snap.Checklist {
		if item == nil { continue }
		added , add_err := AddChecklistItem( card , item.Text )
		if add_err != nil { return nil , add_err }
		added.Done = item.Done
	}
	return
}

func labelMatch( board *Board , name string , color string ) ( label *Label ) {
	name = strings.TrimSpace( name )
	for _ , existing := range board.Labels {
		if existing.Name == name && existing.Color == color { return existing }
	}
	return
}

func utcTime( value *time.Time ) ( result *time.Time ) {
	if value == nil { return }
	utc := value.UTC()
	result = &utc
	return
}
