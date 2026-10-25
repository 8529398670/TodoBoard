package models

import (
	errors "errors"
	strings "strings"
	testing "testing"
	time "time"
)

func everyCard( card *Card ) bool { return true }

func TestSnapshotRoundTrip( t *testing.T ) {
	store := testStore( t )
	owner , _ := CreateUser( store , "Owner" , RoleAdmin )
	board , err := CreateBoard( store , owner.ID , "Plans" , []string{ "To do" , "Doing" , "Done" } )
	if err != nil { t.Fatal( err ) }
	board , _ = MutateBoard( store , board.ID , func( board *Board ) error {
		_ , label_err := AddLabel( board , "Urgent" , "red" )
		return label_err
	} )
	label := board.Labels[ 0 ].ID
	todo := board.SortedLists()[ 0 ].ID

	first , _ := CreateCard( store , board.ID , todo , "First $x^2$" , nil , owner.ID , "Owner" )
	second , _ := CreateCard( store , board.ID , todo , "Second" , nil , owner.ID , "Owner" )
	due := "2026-11-02"
	description := "Body **bold**"
	done := true
	PatchCard( store , board.ID , first.ID , &CardPatch{ Description: &description , LabelIDs: &[]uint64{ label } , DueAt: &due , Done: &done } , "Owner" )
	MutateCard( store , board.ID , second.ID , "Owner" , func( board *Board , card *Card ) error {
		AddChecklistItem( card , "one" )
		item , _ := AddChecklistItem( card , "two" )
		item.Done = true
		return nil
	} )

	board , _ = GetBoard( store , board.ID )
	cards , _ := ListCards( store , board.ID )
	snap := SnapshotBoard( board , cards , everyCard , 0 , nil )
	if len( snap.Lists ) != 3 || len( snap.Lists[ 0 ].Cards ) != 2 || snap.Lists[ 0 ].Cards[ 0 ].Title != "First $x^2$" {
		t.Fatalf( "snapshot lost lists or order: %+v" , snap.Lists )
	}

	copy , lists , err := ImportBoardSkeleton( store , owner.ID , snap )
	if err != nil { t.Fatal( err ) }
	if copy.ID == board.ID || copy.Slug == board.Slug || len( lists ) != 3 || len( copy.Labels ) != 1 {
		t.Fatalf( "import should be a new board with a fresh slug: %+v" , copy )
	}
	for index , list := range snap.Lists {
		if _ , err = ImportCards( store , copy.ID , lists[ index ] , snap.Labels , list.Cards , owner.ID , "Owner" ); err != nil { t.Fatal( err ) }
	}
	copy , _ = GetBoard( store , copy.ID )
	again , _ := ListCards( store , copy.ID )
	back := SnapshotBoard( copy , again , everyCard , 0 , nil )
	got := back.Lists[ 0 ].Cards
	if len( got ) != 2 || got[ 0 ].Title != "First $x^2$" || got[ 0 ].Description != description || got[ 0 ].Done == false || got[ 0 ].DueAt == nil {
		t.Fatalf( "first card did not survive: %+v" , got )
	}
	if len( got[ 0 ].Labels ) != 1 || got[ 0 ].Labels[ 0 ] != copy.Labels[ 0 ].ID {
		t.Errorf( "label not mapped onto the new board: %+v" , got[ 0 ].Labels )
	}
	if len( got[ 1 ].Checklist ) != 2 || got[ 1 ].Checklist[ 1 ].Done == false || got[ 1 ].Checklist[ 0 ].Text != "one" {
		t.Errorf( "checklist did not survive: %+v" , got[ 1 ].Checklist )
	}
	refs , _ , _ := ListBoardsForUser( store , owner.ID )
	if len( refs ) != 2 { t.Errorf( "importer should own both boards, has %d" , len( refs ) ) }
}

func TestImportCardsAppendsAndMatchesLabels( t *testing.T ) {
	store := testStore( t )
	owner , _ := CreateUser( store , "Owner" , RoleAdmin )
	board , _ := CreateBoard( store , owner.ID , "Plans" , []string{ "To do" , "Doing" , "Done" } )
	todo := board.SortedLists()[ 0 ].ID
	existing , _ := CreateCard( store , board.ID , todo , "Already here" , nil , owner.ID , "Owner" )

	labels := []*LabelSnapshot{ { Ref: 7 , Name: "Bug" , Color: "red" } , { Ref: 8 , Name: "Bad" , Color: "not-a-colour" } }
	cards := []*CardSnapshot{
		{ Title: "Imported" , Labels: []uint64{ 7 , 8 , 99 } },
		{ Title: "Also" , Labels: []uint64{ 7 } },
	}
	count , err := ImportCards( store , board.ID , todo , labels , cards , owner.ID , "Owner" )
	if err != nil || count != 2 { t.Fatalf( "import: %d %v" , count , err ) }
	// Same file again: the label is matched, not added twice.
	if _ , err = ImportCards( store , board.ID , todo , labels , cards[ :1 ] , owner.ID , "Owner" ); err != nil { t.Fatal( err ) }

	board , _ = GetBoard( store , board.ID )
	if len( board.Labels ) != 1 { t.Fatalf( "want one Bug label, have %+v" , board.Labels ) }
	all , _ := ListCards( store , board.ID )
	snap := SnapshotList( board.List( todo ) , all , everyCard , nil )
	if len( snap.Cards ) != 4 || snap.Cards[ 0 ].Title != existing.Title || snap.Cards[ 1 ].Title != "Imported" {
		t.Fatalf( "cards should be appended after the existing one: %+v" , snap.Cards )
	}
	if len( snap.Cards[ 1 ].Labels ) != 1 { t.Errorf( "unknown and refused labels should be dropped: %+v" , snap.Cards[ 1 ].Labels ) }
}

func TestImportRejectsBadInput( t *testing.T ) {
	store := testStore( t )
	owner , _ := CreateUser( store , "Owner" , RoleAdmin )
	board , _ := CreateBoard( store , owner.ID , "Plans" , nil )
	todo := board.SortedLists()[ 0 ].ID

	long := strings.Repeat( "x" , maxCardTitleChars + 1 )
	cards := []*CardSnapshot{ { Title: "fine" } , { Title: long } }
	if _ , err := ImportCards( store , board.ID , todo , nil , cards , owner.ID , "Owner" ); errors.Is( err , ErrBadInput ) == false {
		t.Errorf( "an over-long title should refuse the batch, got %v" , err )
	}
	all , _ := ListCards( store , board.ID )
	if len( all ) != 0 { t.Errorf( "a refused batch should add nothing, added %d" , len( all ) ) }

	// Long titles below the cap are fine now.
	if _ , err := CreateCard( store , board.ID , todo , strings.Repeat( "y" , 2000 ) , nil , owner.ID , "Owner" ); err != nil {
		t.Errorf( "a 2000-character title should be allowed: %v" , err )
	}
	if _ , _ , err := ImportBoardSkeleton( store , owner.ID , &BoardSnapshot{ Title: "  " } ); errors.Is( err , ErrBadInput ) == false {
		t.Errorf( "a board needs a title, got %v" , err )
	}
	if _ , err := ImportCards( store , board.ID , 999 , nil , cards[ :1 ] , owner.ID , "Owner" ); errors.Is( err , ErrListNotFound ) == false {
		t.Errorf( "unknown list: %v" , err )
	}
}

func TestSnapshotKeepsSlugWhenFree( t *testing.T ) {
	store := testStore( t )
	owner , _ := CreateUser( store , "Owner" , RoleAdmin )
	start := time.Date( 2026 , 11 , 1 , 9 , 30 , 0 , 0 , time.UTC )
	snap := &BoardSnapshot{ Title: "Trip" , Slug: "summer-trip" , Layout: LayoutThree , DefaultView: "bogus" , Color: "teal" ,
		Lists: []*ListSnapshot{ { Title: "Ideas" , Cards: []*CardSnapshot{ { Title: "Go" , StartAt: &start , StartTimed: true } } } } }
	board , lists , err := ImportBoardSkeleton( store , owner.ID , snap )
	if err != nil { t.Fatal( err ) }
	if board.Slug != "summer-trip" || board.DefaultView != ViewKanban || board.Color != "teal" {
		t.Errorf( "settings not taken as given or defaulted: %+v" , board )
	}
	// A three-column board always gets its three columns.
	if len( board.Lists ) != 3 || len( lists ) != 1 { t.Errorf( "three-column layout: %d lists, %d mapped" , len( board.Lists ) , len( lists ) ) }
	second , _ , _ := ImportBoardSkeleton( store , owner.ID , snap )
	if second.Slug == "summer-trip" { t.Error( "a taken slug must not be reused" ) }
}
