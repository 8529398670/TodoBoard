package models

import (
	errors "errors"
	testing "testing"

	db "todoboard/server/db"
)

func TestSlugify( t *testing.T ) {
	cases := map[string]string{
		"Lab Plans":          "lab-plans",
		"  Café — Q4 2026! ": "cafe-q4-2026",
		"2026":               "board-2026",
		"✓✓✓":                "board",
	}
	for title , want := range cases {
		if got := Slugify( title ); got != want { t.Errorf( "Slugify(%q) = %q, want %q" , title , got , want ) }
	}
	if ValidSlug( "123" ) || ValidSlug( "a--b" ) || ValidSlug( "-a" ) || ValidSlug( "A" ) || ValidSlug( "a" ) == false {
		t.Error( "ValidSlug accepts or refuses the wrong things" )
	}
}

func TestBoardSlugs( t *testing.T ) {
	store := testStore( t )
	owner , _ := CreateUser( store , "Owner" , RoleAdmin )

	first , _ := CreateBoard( store , owner.ID , "Lab" , nil )
	second , _ := CreateBoard( store , owner.ID , "Lab" , nil )
	if first.Slug != "lab" || second.Slug != "lab-2" {
		t.Fatalf( "clashing titles should get lab, lab-2: %q %q" , first.Slug , second.Slug )
	}
	if BoardIDForSlug( store , "lab-2" ) != second.ID {
		t.Error( "slug does not resolve to its board" )
	}

	// A slug held by another board is refused; a free one moves the index.
	taken := "lab"
	if _ , err := UpdateBoardSettings( store , second.ID , &BoardSettings{ Slug: &taken } ); errors.Is( err , ErrSlugTaken ) == false {
		t.Errorf( "taking another board's slug: got %v" , err )
	}
	fresh := "my-lab"
	if _ , err := UpdateBoardSettings( store , second.ID , &BoardSettings{ Slug: &fresh } ); err != nil { t.Fatal( err ) }
	if BoardIDForSlug( store , "lab-2" ) != 0 || BoardIDForSlug( store , "my-lab" ) != second.ID {
		t.Error( "the old slug must be released and the new one claimed" )
	}

	// Renaming keeps the slug; an empty slug regenerates it from the title.
	title := "Chemistry"
	updated , _ := UpdateBoardSettings( store , second.ID , &BoardSettings{ Title: &title } )
	if updated.Slug != "my-lab" { t.Errorf( "a rename must not move the URL, got %q" , updated.Slug ) }
	empty := ""
	updated , _ = UpdateBoardSettings( store , second.ID , &BoardSettings{ Slug: &empty } )
	if updated.Slug != "chemistry" { t.Errorf( "empty slug should regenerate, got %q" , updated.Slug ) }

	// Deleting a board frees its slug.
	if err := DeleteBoard( store , first.ID ); err != nil { t.Fatal( err ) }
	if BoardIDForSlug( store , "lab" ) != 0 { t.Error( "a deleted board's slug must be freed" ) }
}

func TestEnsureBoardSlugsBackfills( t *testing.T ) {
	store := testStore( t )
	owner , _ := CreateUser( store , "Owner" , RoleAdmin )
	board , _ := CreateBoard( store , owner.ID , "Old Board" , nil )

	// Simulate a board written before slugs existed.
	err := store.Write( func( tx *db.Tx ) error {
		if err := tx.Delete( db.BucketBoardSlugs , tx.LookupKey( board.Slug ) ); err != nil { return err }
		board.Slug = ""
		return tx.Put( db.BucketBoards , boardKey( board.ID ) , board )
	} )
	if err != nil { t.Fatal( err ) }

	if err = EnsureBoardSlugs( store ); err != nil { t.Fatal( err ) }
	loaded , _ := GetBoard( store , board.ID )
	if loaded.Slug != "old-board" || BoardIDForSlug( store , "old-board" ) != board.ID {
		t.Errorf( "backfill missed: %q" , loaded.Slug )
	}
	if loaded.Rev != board.Rev { t.Error( "a backfill is not an edit and must not bump Rev" ) }
}

func TestThreeColumnLayout( t *testing.T ) {
	store := testStore( t )
	owner , _ := CreateUser( store , "Owner" , RoleAdmin )
	board , _ := CreateBoard( store , owner.ID , "Solo" , []string{ "Inbox" } )

	three := LayoutThree
	updated , err := UpdateBoardSettings( store , board.ID , &BoardSettings{ Layout: &three , ListTitles: []string{ "To do" , "Doing" , "Done" } } )
	if err != nil { t.Fatal( err ) }
	if updated.Layout != LayoutThree || len( updated.Lists ) != 3 {
		t.Fatalf( "three-column layout needs three lists: %d" , len( updated.Lists ) )
	}
	if updated.SortedLists()[ 0 ].Title != "Inbox" || updated.SortedLists()[ 2 ].Title != "Done" {
		t.Error( "existing lists stay first; missing ones are added after them" )
	}
	bad := "five"
	if _ , err = UpdateBoardSettings( store , board.ID , &BoardSettings{ Layout: &bad } ); errors.Is( err , ErrBadInput ) == false {
		t.Errorf( "unknown layout: got %v" , err )
	}
}

func TestTimedStart( t *testing.T ) {
	store := testStore( t )
	owner , _ := CreateUser( store , "Owner" , RoleAdmin )
	board , _ := CreateBoard( store , owner.ID , "Dates" , []string{ "To do" } )
	card , _ := CreateCard( store , board.ID , board.Lists[ 0 ].ID , "Meet" , nil , owner.ID , "Owner" )

	timed := "2026-10-23T10:00:00Z"
	patched , _ , err := PatchCard( store , board.ID , card.ID , &CardPatch{ StartAt: &timed } , "Owner" )
	if err != nil { t.Fatal( err ) }
	if patched.StartTimed == false { t.Error( "an RFC 3339 start is timed" ) }

	day := "2026-10-23"
	patched , _ , _ = PatchCard( store , board.ID , card.ID , &CardPatch{ StartAt: &day } , "Owner" )
	if patched.StartTimed { t.Error( "a date-only start is all-day" ) }

	clear := ""
	patched , _ , _ = PatchCard( store , board.ID , card.ID , &CardPatch{ StartAt: &clear } , "Owner" )
	if patched.StartAt != nil || patched.StartTimed { t.Error( "clearing the start clears both" ) }
}
