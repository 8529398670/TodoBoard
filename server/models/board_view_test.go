package models

import (
	errors "errors"
	strings "strings"
	testing "testing"
)

func TestCenterColumnsIsSavedAndBackedUp( t *testing.T ) {
	store := testStore( t )
	owner , _ := CreateUser( store , "Owner" , RoleAdmin )
	board , err := CreateBoard( store , owner.ID , "Wide" , []string{ "One" , "Two" } )
	if err != nil { t.Fatal( err ) }
	if board.CenterColumns { t.Error( "a new board keeps its columns to the left" ) }

	on := true
	board , err = UpdateBoardSettings( store , board.ID , &BoardSettings{ CenterColumns: &on } )
	if err != nil { t.Fatal( err ) }
	if board.CenterColumns == false { t.Error( "center_columns was not saved" ) }

	numbering := NumberingOn
	board , err = UpdateBoardSettings( store , board.ID , &BoardSettings{ Numbering: &numbering } )
	if err != nil { t.Fatal( err ) }
	if board.CenterColumns == false { t.Error( "another setting's save cleared center_columns" ) }

	snap := SnapshotBoard( board , nil , everyCard , 0 , nil )
	copy , _ , err := ImportBoardSkeleton( store , owner.ID , snap )
	if err != nil { t.Fatal( err ) }
	if copy.CenterColumns == false { t.Error( "a restored board lost center_columns" ) }
}

func TestStageNames( t *testing.T ) {
	store := testStore( t )
	owner , _ := CreateUser( store , "Owner" , RoleAdmin )
	board , err := CreateBoard( store , owner.ID , "Staged" , []string{ "Work" } )
	if err != nil { t.Fatal( err ) }
	layout := LayoutStaged
	if _ , err = UpdateBoardSettings( store , board.ID , &BoardSettings{ Layout: &layout } ); err != nil { t.Fatal( err ) }

	backlog , review := "  Backlog  " , "Review"
	board , err = UpdateBoardSettings( store , board.ID , &BoardSettings{ StageTodo: &backlog } )
	if err != nil { t.Fatal( err ) }
	board , err = UpdateBoardSettings( store , board.ID , &BoardSettings{ StageDoing: &review } )
	if err != nil { t.Fatal( err ) }
	if board.StageTodo != "Backlog" { t.Errorf( "the name should be trimmed and kept: %q" , board.StageTodo ) }
	if board.StageDoing != "Review" || board.StageDone != "" { t.Errorf( "one name's save touched another: %+v" , board ) }

	too_long := strings.Repeat( "é" , maxStageName + 1 )
	if _ , err = UpdateBoardSettings( store , board.ID , &BoardSettings{ StageDone: &too_long } ); errors.Is( err , ErrBadInput ) == false {
		t.Errorf( "a name over the limit should be refused, got %v" , err )
	}
	longest := strings.Repeat( "é" , maxStageName )
	if _ , err = UpdateBoardSettings( store , board.ID , &BoardSettings{ StageDone: &longest } ); err != nil {
		t.Errorf( "the limit counts characters, not bytes: %v" , err )
	}

	snap := SnapshotBoard( board , nil , everyCard , 0 , nil )
	copy , _ , err := ImportBoardSkeleton( store , owner.ID , snap )
	if err != nil { t.Fatal( err ) }
	if copy.StageTodo != "Backlog" || copy.StageDoing != "Review" { t.Errorf( "a restored board lost its stage names: %+v" , copy ) }

	// A hand-edited file can claim anything: that name is dropped, not the board.
	snap.StageDone = too_long
	copy , _ , err = ImportBoardSkeleton( store , owner.ID , snap )
	if err != nil { t.Fatal( err ) }
	if copy.StageDone != "" { t.Errorf( "a name over the limit should not be restored: %q" , copy.StageDone ) }

	empty := ""
	board , err = UpdateBoardSettings( store , board.ID , &BoardSettings{ StageTodo: &empty } )
	if err != nil { t.Fatal( err ) }
	if board.StageTodo != "" { t.Errorf( "an empty box goes back to the app's word: %q" , board.StageTodo ) }
}

func TestListStageView( t *testing.T ) {
	store := testStore( t )
	owner , _ := CreateUser( store , "Owner" , RoleAdmin )
	board , err := CreateBoard( store , owner.ID , "Staged" , []string{ "Work" , "Home" } )
	if err != nil { t.Fatal( err ) }
	list := board.SortedLists()[ 0 ]
	if list.StageView != "" { t.Errorf( "a new column follows the navbar: %q" , list.StageView ) }

	set := func( view string ) ( err error ) {
		update := &ListUpdate{ StageView: &view }
		_ , err = MutateBoard( store , board.ID , func( board *Board ) error { return update.Apply( board , list.ID ) } )
		return
	}
	for _ , view := range []string{ "todo" , "doing" , "done" , "" } {
		if err = set( view ); err != nil { t.Errorf( "%q should be allowed: %v" , view , err ) }
	}
	if err = set( "all" ); errors.Is( err , ErrBadInput ) == false {
		t.Errorf( "All is no stage a column can hold, got %v" , err )
	}

	if err = set( "done" ); err != nil { t.Fatal( err ) }
	board , _ = GetBoard( store , board.ID )
	snap := SnapshotBoard( board , nil , everyCard , 0 , nil )
	if snap.Lists[ 0 ].StageView != "done" { t.Fatalf( "the backup left out stage_view: %+v" , snap.Lists[ 0 ] ) }
	copy , _ , err := ImportBoardSkeleton( store , owner.ID , snap )
	if err != nil { t.Fatal( err ) }
	if copy.SortedLists()[ 0 ].StageView != "done" { t.Error( "a restored column lost stage_view" ) }

	// A hand-edited file can claim anything: that column just follows the navbar.
	snap.Lists[ 0 ].StageView = "sometimes"
	copy , _ , err = ImportBoardSkeleton( store , owner.ID , snap )
	if err != nil { t.Fatal( err ) }
	if copy.SortedLists()[ 0 ].StageView != "" { t.Errorf( "a bad view should not be restored: %q" , copy.SortedLists()[ 0 ].StageView ) }
}

func TestCollapsedListsArePersonal( t *testing.T ) {
	store := testStore( t )
	owner , _ := CreateUser( store , "Owner" , RoleAdmin )
	other , _ := CreateUser( store , "Other" , RoleUser )
	board , err := CreateBoard( store , owner.ID , "Wide" , []string{ "One" , "Two" } )
	if err != nil { t.Fatal( err ) }
	if board , err = SetMember( store , board.ID , other.ID , BoardRoleEditor ); err != nil { t.Fatal( err ) }
	lists := board.SortedLists()
	first , second := lists[ 0 ].ID , lists[ 1 ].ID

	plain := true
	if err = SetBoardPlainIcons( store , owner.ID , board.ID , &plain ); err != nil { t.Fatal( err ) }
	kept , err := SetBoardCollapsedLists( store , owner.ID , board , []uint64{ first , 999999 , first , second } )
	if err != nil { t.Fatal( err ) }
	if len( kept ) != 2 || kept[ 0 ] != first || kept[ 1 ] != second {
		t.Errorf( "unknown and repeated ids should go, the rest stay in order: %v" , kept )
	}
	if again , _ := GetBoard( store , board.ID ); again.Rev != board.Rev {
		t.Error( "how the board looks to one person is no change to the board" )
	}
	ref := GetBoardRef( store , owner.ID , board.ID )
	if len( ref.CollapsedLists ) != 2 { t.Errorf( "not saved: %+v" , ref ) }
	if ref.PlainIcons == nil || *ref.PlainIcons == false { t.Error( "saving collapsed columns touched plain_icons" ) }
	if len( GetBoardRef( store , other.ID , board.ID ).CollapsedLists ) != 0 { t.Error( "one person's collapsed columns reached another" ) }

	if err = SetBoardPlainIcons( store , owner.ID , board.ID , nil ); err != nil { t.Fatal( err ) }
	if len( GetBoardRef( store , owner.ID , board.ID ).CollapsedLists ) != 2 { t.Error( "saving plain_icons touched collapsed columns" ) }

	if board , err = SetMember( store , board.ID , other.ID , BoardRoleViewer ); err != nil { t.Fatal( err ) }
	if _ , err = SetBoardCollapsedLists( store , other.ID , board , []uint64{ second } ); err != nil { t.Fatal( err ) }
	if board , err = SetMember( store , board.ID , other.ID , BoardRoleEditor ); err != nil { t.Fatal( err ) }
	if len( GetBoardRef( store , other.ID , board.ID ).CollapsedLists ) != 1 { t.Error( "a change of role lost collapsed columns" ) }

	stranger , _ := CreateUser( store , "Stranger" , RoleUser )
	if _ , err = SetBoardCollapsedLists( store , stranger.ID , board , []uint64{ first } ); errors.Is( err , ErrBoardNotFound ) == false {
		t.Errorf( "someone with no row on the board has nowhere to keep this, got %v" , err )
	}
}
