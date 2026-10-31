package models

import (
	testing "testing"
)

func TestProductRenamesLinkOnlyCard( t *testing.T ) {
	store := testStore( t )
	owner , _ := CreateUser( store , "Owner" , RoleAdmin )
	board , err := CreateBoard( store , owner.ID , "Shopping" , []string{ "Want" } )
	if err != nil { t.Fatal( err ) }
	list := board.SortedLists()[ 0 ].ID
	link := "https://shop.example/item/42"
	found := func() *Product { return &Product{ URL: link , Title: "  Blue Kettle  " , Status: ProductOK } }

	// Only the link: renamed, link kept in the description, rev moved.
	bare , _ := CreateCard( store , board.ID , list , "  " + link + " " , nil , owner.ID , "Owner" )
	named , err := SetCardProduct( store , board.ID , bare.ID , found() , nil )
	if err != nil { t.Fatal( err ) }
	if named.Title != "Blue Kettle" || named.Description != link || named.Rev <= bare.Rev {
		t.Errorf( "link-only card not renamed: %+v" , named )
	}

	// Anything else in the title: left alone.
	worded , _ := CreateCard( store , board.ID , list , "Buy this " + link , nil , owner.ID , "Owner" )
	kept , _ := SetCardProduct( store , board.ID , worded.ID , found() , nil )
	if kept.Title != "Buy this " + link || kept.Rev != worded.Rev {
		t.Errorf( "worded card must keep its title: %+v" , kept )
	}

	// A failed lookup, or one with no name, renames nothing.
	failed , _ := CreateCard( store , board.ID , list , link , nil , owner.ID , "Owner" )
	after , _ := SetCardProduct( store , board.ID , failed.ID , &Product{ URL: link , Title: "X" , Status: ProductFailed } , nil )
	if after.Title != link { t.Errorf( "failed lookup renamed the card: %q" , after.Title ) }
	after , _ = SetCardProduct( store , board.ID , failed.ID , &Product{ URL: link , Status: ProductOK } , nil )
	if after.Title != link { t.Errorf( "untitled product renamed the card: %q" , after.Title ) }

	// A description already holding the link is not given it twice.
	described , _ := CreateCard( store , board.ID , list , link , nil , owner.ID , "Owner" )
	text := "see " + link
	PatchCard( store , board.ID , described.ID , &CardPatch{ Description: &text } , "Owner" )
	after , _ = SetCardProduct( store , board.ID , described.ID , found() , nil )
	if after.Title != "Blue Kettle" || after.Description != text {
		t.Errorf( "description changed when it already had the link: %+v" , after )
	}
}
