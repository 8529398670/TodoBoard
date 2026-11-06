package models

import (
	errors "errors"
	testing "testing"

	db "todoboard/server/db"
)

func TestFolderOrder( t *testing.T ) {
	store := testStore( t )
	user , _ := CreateUser( store , "Owner" , RoleAdmin )
	work , _ := CreateFolder( store , user.ID , "Work" )
	CreateFolder( store , user.ID , "Home" )
	misc , _ := CreateFolder( store , user.ID , "Misc" )

	names := func() ( result []string ) {
		folders , _ := ListFolders( store , user.ID )
		for _ , folder := range folders { result = append( result , folder.Name ) }
		return
	}
	if got := names(); got[ 0 ] != "Work" || got[ 2 ] != "Misc" {
		t.Errorf( "new folders go last: %v" , got )
	}
	if err := SetFolderOrder( store , user.ID , []uint64{ misc.ID , work.ID } ); err != nil { t.Fatal( err ) }
	if got := names(); got[ 0 ] != "Misc" || got[ 1 ] != "Work" || got[ 2 ] != "Home" {
		t.Errorf( "named folders first, the rest after: %v" , got )
	}
}

func folderIn( t *testing.T , store *db.Store , user_id uint64 , name string , parent_id uint64 ) ( folder *Folder ) {
	t.Helper()
	folder , err := CreateFolderIn( store , user_id , name , parent_id )
	if err != nil { t.Fatal( err ) }
	return
}

func TestFoldersNest( t *testing.T ) {
	store := testStore( t )
	user , _ := CreateUser( store , "Owner" , RoleUser )
	other , _ := CreateUser( store , "Other" , RoleUser )
	top := folderIn( t , store , user.ID , "Top" , 0 )
	middle := folderIn( t , store , user.ID , "Middle" , top.ID )
	bottom := folderIn( t , store , user.ID , "Bottom" , middle.ID )
	if bottom.Parent != middle.ID { t.Errorf( "not filed under its parent: %+v" , bottom ) }

	if _ , err := CreateFolderIn( store , user.ID , "Too deep" , bottom.ID ); errors.Is( err , ErrFolderDepth ) == false {
		t.Errorf( "a fourth level should be refused, got %v" , err )
	}
	if _ , err := CreateFolderIn( store , user.ID , "Nowhere" , 99999 ); errors.Is( err , ErrFolderNotFound ) == false {
		t.Errorf( "no such parent, got %v" , err )
	}
	theirs := folderIn( t , store , other.ID , "Theirs" , 0 )
	if _ , err := CreateFolderIn( store , user.ID , "Into theirs" , theirs.ID ); errors.Is( err , ErrFolderNotFound ) == false {
		t.Errorf( "another person's folder is no parent, got %v" , err )
	}
}

func TestMoveFolder( t *testing.T ) {
	store := testStore( t )
	user , _ := CreateUser( store , "Owner" , RoleUser )
	top := folderIn( t , store , user.ID , "Top" , 0 )
	middle := folderIn( t , store , user.ID , "Middle" , top.ID )
	bottom := folderIn( t , store , user.ID , "Bottom" , middle.ID )
	spare := folderIn( t , store , user.ID , "Spare" , 0 )
	spare_child := folderIn( t , store , user.ID , "Spare child" , spare.ID )

	// Into itself or anything in it would cut it off from the top.
	for _ , parent := range []uint64{ top.ID , middle.ID , bottom.ID } {
		if _ , err := MoveFolder( store , user.ID , top.ID , parent ); errors.Is( err , ErrBadInput ) == false {
			t.Errorf( "moved a folder under %d: %v" , parent , err )
		}
	}

	// A folder brings what is in it, and all of that has to fit.
	if _ , err := MoveFolder( store , user.ID , spare.ID , middle.ID ); errors.Is( err , ErrFolderDepth ) == false {
		t.Errorf( "two levels under the second should not fit, got %v" , err )
	}
	moved , err := MoveFolder( store , user.ID , spare.ID , top.ID )
	if err != nil { t.Fatal( err ) }
	if moved.Parent != top.ID { t.Errorf( "not moved: %+v" , moved ) }
	folders , _ := ListFolders( store , user.ID )
	last := 0.0
	for _ , folder := range folders {
		if folder.Parent == top.ID && folder.ID != spare.ID && folder.Position > last { last = folder.Position }
	}
	if moved.Position <= last { t.Errorf( "a moved folder goes last among its new siblings: %v after %v" , moved.Position , last ) }

	// Out to the top, and a move to where it already is changes nothing.
	if moved , err = MoveFolder( store , user.ID , spare_child.ID , 0 ); err != nil || moved.Parent != 0 { t.Fatalf( "%+v %v" , moved , err ) }
	position := moved.Position
	if moved , err = MoveFolder( store , user.ID , spare_child.ID , 0 ); err != nil || moved.Position != position { t.Errorf( "%+v %v" , moved , err ) }

	if _ , err = MoveFolder( store , user.ID , 99999 , 0 ); errors.Is( err , ErrFolderNotFound ) == false { t.Errorf( "no such folder, got %v" , err ) }
	if _ , err = MoveFolder( store , user.ID , top.ID , 99999 ); errors.Is( err , ErrFolderNotFound ) == false { t.Errorf( "no such parent, got %v" , err ) }
}

func TestFolderHeight( t *testing.T ) {
	shape := []*Folder{ { ID: 1 } , { ID: 2 , Parent: 1 } , { ID: 3 , Parent: 2 } , { ID: 4 , Parent: 1 } , { ID: 5 } }
	for id , want := range map[uint64]int{ 1: 3 , 2: 2 , 3: 1 , 4: 1 , 5: 1 } {
		if got := folderHeight( shape , id ); got != want { t.Errorf( "folder %d: height %d, want %d" , id , got , want ) }
	}
}

func TestDeleteMiddleFolderMovesItsContentsUp( t *testing.T ) {
	store := testStore( t )
	user , _ := CreateUser( store , "Owner" , RoleUser )
	top := folderIn( t , store , user.ID , "Top" , 0 )
	middle := folderIn( t , store , user.ID , "Middle" , top.ID )
	bottom := folderIn( t , store , user.ID , "Bottom" , middle.ID )
	board , _ := CreateBoard( store , user.ID , "Plans" , []string{ "To do" } )
	kept , _ := CreateBoard( store , user.ID , "Deep" , []string{ "To do" } )
	if err := SetBoardFolder( store , user.ID , board.ID , middle.ID ); err != nil { t.Fatal( err ) }
	if err := SetBoardFolder( store , user.ID , kept.ID , bottom.ID ); err != nil { t.Fatal( err ) }

	if err := DeleteFolder( store , user.ID , middle.ID ); err != nil { t.Fatal( err ) }
	if ref := GetBoardRef( store , user.ID , board.ID ); ref.Folder != top.ID { t.Errorf( "the board should move up to Top: %+v" , ref ) }
	if ref := GetBoardRef( store , user.ID , kept.ID ); ref.Folder != bottom.ID { t.Errorf( "a board further down stays put: %+v" , ref ) }
	folders , _ := ListFolders( store , user.ID )
	for _ , folder := range folders {
		if folder.ID == bottom.ID && folder.Parent != top.ID { t.Errorf( "the folder should move up to Top: %+v" , folder ) }
		if folder.ID == middle.ID { t.Error( "the folder is still there" ) }
	}

	// At the top there is nowhere further up: boards are unfiled, folders top level.
	if err := DeleteFolder( store , user.ID , top.ID ); err != nil { t.Fatal( err ) }
	if ref := GetBoardRef( store , user.ID , board.ID ); ref.Folder != 0 { t.Errorf( "the board should be unfiled: %+v" , ref ) }
	folders , _ = ListFolders( store , user.ID )
	if len( folders ) != 1 || folders[ 0 ].Parent != 0 { t.Errorf( "the folder should be at the top: %+v" , folders ) }
}

// SetFolderOrder is one order across every level, and only how siblings
// compare is read, so a nested tree needs nothing new from it.
func TestFolderOrderAcrossLevels( t *testing.T ) {
	store := testStore( t )
	user , _ := CreateUser( store , "Owner" , RoleUser )
	top := folderIn( t , store , user.ID , "Top" , 0 )
	first := folderIn( t , store , user.ID , "First" , top.ID )
	second := folderIn( t , store , user.ID , "Second" , top.ID )
	if err := SetFolderOrder( store , user.ID , []uint64{ top.ID , second.ID , first.ID } ); err != nil { t.Fatal( err ) }
	folders , _ := ListFolders( store , user.ID )
	if folders[ 1 ].ID != second.ID || folders[ 2 ].ID != first.ID { t.Errorf( "wrong order: %+v" , folders ) }
	if folders[ 1 ].Parent != top.ID || folders[ 2 ].Parent != top.ID { t.Errorf( "reordering moved a folder: %+v" , folders ) }
}
