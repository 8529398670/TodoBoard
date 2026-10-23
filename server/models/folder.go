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
// is kept on that user's BoardRef. One level only; folders hold boards, not
// other folders.
type Folder struct {
	ID        uint64    `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

var ErrFolderNotFound = errors.New( "folder not found" )

const maxFoldersPerUser = 200

func folderKey( user_id uint64 , folder_id uint64 ) ( key []byte ) {
	key = db.Key( user_id , folder_id )
	return
}

// ListFolders returns a person's folders by name.
func ListFolders( store *db.Store , user_id uint64 ) ( folders []*Folder , err error ) {
	folders = []*Folder{}
	err = store.ForEachPrefix( db.BucketFolders , db.Key( user_id ) , func() any { return &Folder{} } ,
		func( key []byte , item any ) bool {
			folders = append( folders , item.( *Folder ) )
			return true
		} )
	sort.SliceStable( folders , func( i int , j int ) bool {
		return strings.ToLower( folders[ i ].Name ) < strings.ToLower( folders[ j ].Name )
	} )
	return
}

func CreateFolder( store *db.Store , user_id uint64 , name string ) ( folder *Folder , err error ) {
	clean , ok := cleanTitle( name , 80 )
	if ok == false {
		err = ErrBadInput
		return
	}
	err = store.Write( func( tx *db.Tx ) ( tx_err error ) {
		count := 0
		tx_err = tx.ForEachPrefix( db.BucketFolders , db.Key( user_id ) , func() any { return &Folder{} } ,
			func( key []byte , item any ) bool {
				count += 1
				return true
			} )
		if tx_err != nil { return }
		if count >= maxFoldersPerUser { return ErrBadInput }
		id , tx_err := tx.NextSequence( db.BucketFolders )
		if tx_err != nil { return }
		folder = &Folder{ ID: id , Name: clean , CreatedAt: time.Now().UTC() }
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

// DeleteFolder removes the folder; its boards go back to unfiled, in the same
// transaction, so no board is left filed under a folder that is gone.
func DeleteFolder( store *db.Store , user_id uint64 , folder_id uint64 ) ( err error ) {
	err = store.Write( func( tx *db.Tx ) ( tx_err error ) {
		if tx_err = tx.Get( db.BucketFolders , folderKey( user_id , folder_id ) , &Folder{} ); tx_err != nil {
			if errors.Is( tx_err , db.ErrNotFound ) { tx_err = ErrFolderNotFound }
			return
		}
		filed := map[uint64]*BoardRef{}
		tx_err = tx.ForEachPrefix( db.BucketUserBoards , db.Key( user_id ) , func() any { return &BoardRef{} } ,
			func( key []byte , item any ) bool {
				if ref := item.( *BoardRef ); ref.Folder == folder_id { filed[ encryption.BytesToUint64( key[ 8:16 ] ) ] = ref }
				return true
			} )
		if tx_err != nil { return }
		for board_id , ref := range filed {
			ref.Folder = 0
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
