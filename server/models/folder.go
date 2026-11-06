package models

import (
	errors "errors"
	sort "sort"
	strings "strings"
	time "time"

	db "todoboard/server/db"
	encryption "todoboard/server/encryption"
)

// Folder groups boards on one person's home page. Folders are private: the
// same shared board can sit in a different folder -- or none -- for each of
// its members, so a folder belongs to a user, and which folder a board is in
// is kept on that user's BoardRef. Folders hold boards and other folders, up
// to maxFolderDepth levels deep.
//
// Parent is the folder this one sits in, 0 at the top. Folders made before it
// existed are all 0, so they stay at the top.
//
// Position is the person's own order, used when they sort folders by hand
// (UserPrefs.FolderSort). Folders made before it existed are all 0, so they
// keep reading A to Z until first dragged. It is one order across every level:
// only how a folder's siblings compare means anything.
type Folder struct {
	ID        uint64    `json:"id"`
	Name      string    `json:"name"`
	Parent    uint64    `json:"parent_id"`
	Position  float64   `json:"position"`
	CreatedAt time.Time `json:"created_at"`
}

var (
	ErrFolderNotFound = errors.New( "folder not found" )
	ErrFolderDepth    = errors.New( "folders are nested too deep" )
)

const (
	maxFoldersPerUser = 200
	maxFolderDepth    = 3
)

func folderKey( user_id uint64 , folder_id uint64 ) ( key []byte ) {
	key = db.Key( user_id , folder_id )
	return
}

// ListFolders returns a person's folders in their own order, ties by name.
// Sorting A to Z instead is the page's choice.
func ListFolders( store *db.Store , user_id uint64 ) ( folders []*Folder , err error ) {
	folders = []*Folder{}
	err = store.ForEachPrefix( db.BucketFolders , db.Key( user_id ) , func() any { return &Folder{} } ,
		func( key []byte , item any ) bool {
			folders = append( folders , item.( *Folder ) )
			return true
		} )
	sortFolders( folders )
	return
}

func sortFolders( folders []*Folder ) {
	sort.SliceStable( folders , func( i int , j int ) bool {
		if folders[ i ].Position != folders[ j ].Position { return folders[ i ].Position < folders[ j ].Position }
		return strings.ToLower( folders[ i ].Name ) < strings.ToLower( folders[ j ].Name )
	} )
}

func listFoldersTx( tx *db.Tx , user_id uint64 ) ( folders []*Folder , err error ) {
	folders = []*Folder{}
	err = tx.ForEachPrefix( db.BucketFolders , db.Key( user_id ) , func() any { return &Folder{} } ,
		func( key []byte , item any ) bool {
			folders = append( folders , item.( *Folder ) )
			return true
		} )
	sortFolders( folders )
	return
}

// foldersByID is the lookup the nesting checks walk.
func foldersByID( folders []*Folder ) ( by_id map[uint64]*Folder ) {
	by_id = map[uint64]*Folder{}
	for _ , folder := range folders { by_id[ folder.ID ] = folder }
	return
}

// folderDepth is how many levels down a folder sits, 1 at the top and 0 for
// the top itself. It stops at len( by_id ) steps, so a loop in stored data
// reads as too deep rather than hanging.
func folderDepth( by_id map[uint64]*Folder , folder_id uint64 ) ( depth int ) {
	for folder := by_id[ folder_id ]; folder != nil && depth <= len( by_id ); folder = by_id[ folder.Parent ] {
		depth += 1
	}
	return
}

// folderHeight is how many levels a folder and what is inside it take up: 1
// with no folders in it. Each folder walks up to see if it is inside this one,
// so like folderDepth it stops rather than hangs on a loop in stored data.
func folderHeight( folders []*Folder , folder_id uint64 ) ( height int ) {
	by_id := foldersByID( folders )
	height = 1
	for _ , folder := range folders {
		levels := 1
		for at := folder; at != nil && levels <= len( folders ); at = by_id[ at.Parent ] {
			if at.ID == folder_id {
				if levels > height { height = levels }
				break
			}
			levels += 1
		}
	}
	return
}

// lastFolderPosition is where a folder goes to come after the others.
func lastFolderPosition( folders []*Folder ) ( position float64 ) {
	position = positionStep
	for _ , other := range folders {
		if other.Position >= position { position = other.Position + positionStep }
	}
	return
}

func CreateFolder( store *db.Store , user_id uint64 , name string ) ( folder *Folder , err error ) {
	folder , err = CreateFolderIn( store , user_id , name , 0 )
	return
}

// CreateFolderIn makes a folder inside parent_id, or with 0 at the top.
func CreateFolderIn( store *db.Store , user_id uint64 , name string , parent_id uint64 ) ( folder *Folder , err error ) {
	clean , ok := cleanTitle( name , 80 )
	if ok == false {
		err = ErrBadInput
		return
	}
	err = store.Write( func( tx *db.Tx ) ( tx_err error ) {
		existing , tx_err := listFoldersTx( tx , user_id )
		if tx_err != nil { return }
		if len( existing ) >= maxFoldersPerUser { return ErrBadInput }
		if parent_id != 0 {
			by_id := foldersByID( existing )
			if by_id[ parent_id ] == nil { return ErrFolderNotFound }
			if folderDepth( by_id , parent_id ) >= maxFolderDepth { return ErrFolderDepth }
		}
		id , tx_err := tx.NextSequence( db.BucketFolders )
		if tx_err != nil { return }
		// A new folder goes last in the person's own order.
		folder = &Folder{ ID: id , Name: clean , Parent: parent_id , Position: lastFolderPosition( existing ) , CreatedAt: time.Now().UTC() }
		tx_err = tx.Put( db.BucketFolders , folderKey( user_id , id ) , folder )
		return
	} )
	if err != nil { folder = nil }
	return
}

func RenameFolder( store *db.Store , user_id uint64 , folder_id uint64 , name string ) ( folder *Folder , err error ) {
	clean , ok := cleanTitle( name , 80 )
	if ok == false {
		err = ErrBadInput
		return
	}
	err = store.Write( func( tx *db.Tx ) ( tx_err error ) {
		folder = &Folder{}
		if tx_err = tx.Get( db.BucketFolders , folderKey( user_id , folder_id ) , folder ); tx_err != nil {
			if errors.Is( tx_err , db.ErrNotFound ) { tx_err = ErrFolderNotFound }
			return
		}
		folder.Name = clean
		tx_err = tx.Put( db.BucketFolders , folderKey( user_id , folder_id ) , folder )
		return
	} )
	if err != nil { folder = nil }
	return
}

// MoveFolder puts a folder, and everything in it, inside parent_id, or with 0
// at the top, after whatever is there already. A folder cannot go inside
// itself or anything in it, and nothing can end up too deep.
func MoveFolder( store *db.Store , user_id uint64 , folder_id uint64 , parent_id uint64 ) ( folder *Folder , err error ) {
	err = store.Write( func( tx *db.Tx ) ( tx_err error ) {
		folders , tx_err := listFoldersTx( tx , user_id )
		if tx_err != nil { return }
		by_id := foldersByID( folders )
		folder = by_id[ folder_id ]
		if folder == nil { return ErrFolderNotFound }
		if folder.Parent == parent_id { return }
		if parent_id != 0 {
			if by_id[ parent_id ] == nil { return ErrFolderNotFound }
			// Up from the new parent, the folder must not be on the way.
			steps := 0
			for at := by_id[ parent_id ]; at != nil && steps <= len( folders ); at = by_id[ at.Parent ] {
				if at.ID == folder_id { return ErrBadInput }
				steps += 1
			}
		}
		if folderDepth( by_id , parent_id ) + folderHeight( folders , folder_id ) > maxFolderDepth { return ErrFolderDepth }
		folder.Parent = parent_id
		folder.Position = lastFolderPosition( folders )
		tx_err = tx.Put( db.BucketFolders , folderKey( user_id , folder_id ) , folder )
		return
	} )
	if err != nil { folder = nil }
	return
}

// SetFolderOrder writes a person's whole folder order at once: ids first to
// last. Folders not named keep their place after the named ones. Nesting does
// not change it: the page sends every level, and only how siblings compare is
// ever read.
func SetFolderOrder( store *db.Store , user_id uint64 , ids []uint64 ) ( err error ) {
	err = store.Write( func( tx *db.Tx ) ( tx_err error ) {
		folders , tx_err := listFoldersTx( tx , user_id )
		if tx_err != nil { return }
		rank := map[uint64]int{}
		for index , id := range ids {
			if _ , seen := rank[ id ]; seen == false { rank[ id ] = index }
		}
		sort.SliceStable( folders , func( i int , j int ) bool {
			ri , oki := rank[ folders[ i ].ID ]
			rj , okj := rank[ folders[ j ].ID ]
			if oki && okj { return ri < rj }
			return oki && okj == false
		} )
		for index , folder := range folders {
			folder.Position = float64( index + 1 ) * positionStep
			if tx_err = tx.Put( db.BucketFolders , folderKey( user_id , folder.ID ) , folder ); tx_err != nil { return }
		}
		return
	} )
	return
}

// DeleteFolder removes the folder. Its boards and the folders in it move up
// to where it was -- for one at the top, boards to unfiled and folders to the
// top -- in the same transaction, so nothing is left under a folder that is
// gone.
func DeleteFolder( store *db.Store , user_id uint64 , folder_id uint64 ) ( err error ) {
	err = store.Write( func( tx *db.Tx ) ( tx_err error ) {
		folder := &Folder{}
		if tx_err = tx.Get( db.BucketFolders , folderKey( user_id , folder_id ) , folder ); tx_err != nil {
			if errors.Is( tx_err , db.ErrNotFound ) { tx_err = ErrFolderNotFound }
			return
		}
		folders , tx_err := listFoldersTx( tx , user_id )
		if tx_err != nil { return }
		for _ , child := range folders {
			if child.Parent != folder_id { continue }
			child.Parent = folder.Parent
			if tx_err = tx.Put( db.BucketFolders , folderKey( user_id , child.ID ) , child ); tx_err != nil { return }
		}
		filed := map[uint64]*BoardRef{}
		tx_err = tx.ForEachPrefix( db.BucketUserBoards , db.Key( user_id ) , func() any { return &BoardRef{} } ,
			func( key []byte , item any ) bool {
				if ref := item.( *BoardRef ); ref.Folder == folder_id { filed[ encryption.BytesToUint64( key[ 8:16 ] ) ] = ref }
				return true
			} )
		if tx_err != nil { return }
		for board_id , ref := range filed {
			ref.Folder = folder.Parent
			if tx_err = tx.Put( db.BucketUserBoards , db.Key( user_id , board_id ) , ref ); tx_err != nil { return }
		}
		tx_err = tx.Delete( db.BucketFolders , folderKey( user_id , folder_id ) )
		return
	} )
	return
}

// SetBoardFolder files a board this person can open under one of their
// folders, or with folder_id 0 takes it out again.
func SetBoardFolder( store *db.Store , user_id uint64 , board_id uint64 , folder_id uint64 ) ( err error ) {
	err = store.Write( func( tx *db.Tx ) ( tx_err error ) {
		ref := &BoardRef{}
		if tx_err = tx.Get( db.BucketUserBoards , db.Key( user_id , board_id ) , ref ); tx_err != nil {
			if errors.Is( tx_err , db.ErrNotFound ) { tx_err = ErrBoardNotFound }
			return
		}
		if folder_id != 0 {
			if tx_err = tx.Get( db.BucketFolders , folderKey( user_id , folder_id ) , &Folder{} ); tx_err != nil {
				if errors.Is( tx_err , db.ErrNotFound ) { tx_err = ErrFolderNotFound }
				return
			}
		}
		ref.Folder = folder_id
		tx_err = tx.Put( db.BucketUserBoards , db.Key( user_id , board_id ) , ref )
		return
	} )
	return
}
