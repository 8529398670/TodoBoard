package models

import (
	testing "testing"
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
