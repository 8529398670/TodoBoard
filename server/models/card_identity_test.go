package models

import (
	errors "errors"
	strings "strings"
	testing "testing"
	time "time"

	db "todoboard/server/db"
)

func TestNewULID( t *testing.T ) {
	earlier := newULID( time.UnixMilli( 1700000000000 ) )
	later := newULID( time.UnixMilli( 1700000000001 ) )
	if len( earlier ) != 26 { t.Fatalf( "a ULID is 26 characters, got %q" , earlier ) }
	for _ , char := range earlier {
		if strings.ContainsRune( crockford , char ) == false { t.Errorf( "%q is not in the ULID alphabet" , char ) }
	}
	if earlier >= later { t.Errorf( "ULIDs should sort by time: %s then %s" , earlier , later ) }
	if earlier[ 0 ] > '7' { t.Errorf( "the first character only holds three bits: %s" , earlier ) }
	// The ULID spec's own example: this time encodes to these ten characters.
	if got := newULID( time.UnixMilli( 1469918176385 ) )[ :10 ]; got != "01ARYZ6S41" {
		t.Errorf( "time part: got %s, want 01ARYZ6S41" , got )
	}
	if newULID( time.UnixMilli( 1 ) ) == newULID( time.UnixMilli( 1 ) ) { t.Error( "the same millisecond should still give two ULIDs" ) }
}

func TestCardSerialsCountPerBoard( t *testing.T ) {
	store := testStore( t )
	owner , _ := CreateUser( store , "Owner" , RoleAdmin )
	one , _ := CreateBoard( store , owner.ID , "One" , []string{ "To do" , "Done" } )
	two , _ := CreateBoard( store , owner.ID , "Two" , []string{ "To do" } )
	todo := one.SortedLists()[ 0 ].ID

	first , _ := CreateCard( store , one.ID , todo , "First" , nil , owner.ID , "Owner" )
	second , _ := CreateCard( store , one.ID , todo , "Second" , nil , owner.ID , "Owner" )
	other , _ := CreateCard( store , two.ID , two.SortedLists()[ 0 ].ID , "Other" , nil , owner.ID , "Owner" )
	if first.Serial != 1 || second.Serial != 2 || other.Serial != 1 {
		t.Fatalf( "serials: got %d, %d and %d, want 1, 2 and 1" , first.Serial , second.Serial , other.Serial )
	}
	if len( first.UID ) != 26 || first.UID == second.UID { t.Errorf( "each card needs its own ULID: %q %q" , first.UID , second.UID ) }

	// Moving a card keeps its number and its ULID.
	done := one.SortedLists()[ 1 ].ID
	moved , _ , err := PatchCard( store , one.ID , first.ID , &CardPatch{ ListID: &done } , "Owner" )
	if err != nil { t.Fatal( err ) }
	if moved.Serial != 1 || moved.UID != first.UID { t.Errorf( "a move changed the card's identity: %+v" , moved ) }

	// A number is never reused, even once its card has gone.
	if err = TrashCard( store , one.ID , second.ID , "Owner" ); err != nil { t.Fatal( err ) }
	if err = PurgeCard( store , one.ID , second.ID ); err != nil { t.Fatal( err ) }
	third , _ := CreateCard( store , one.ID , todo , "Third" , nil , owner.ID , "Owner" )
	if third.Serial != 3 { t.Errorf( "got serial %d after a purge, want 3" , third.Serial ) }
}

// forgetIdentity puts a board back as it was before cards had an identity.
func forgetIdentity( t *testing.T , store *db.Store , board_id uint64 ) {
	t.Helper()
	err := store.Write( func( tx *db.Tx ) ( tx_err error ) {
		for _ , bucket := range []string{ db.BucketCards , db.BucketTrash } {
			found := []*Card{}
			tx_err = tx.ForEachPrefix( bucket , db.Key( board_id ) , func() any { return &Card{} } , func( key []byte , item any ) bool {
				found = append( found , item.( *Card ) )
				return true
			} )
			if tx_err != nil { return }
			for _ , card := range found {
				card.Serial , card.UID = 0 , ""
				if tx_err = tx.Put( bucket , cardKey( board_id , card.ID ) , card ); tx_err != nil { return }
			}
		}
		board , tx_err := getBoardTx( tx , board_id )
		if tx_err != nil { return }
		board.NextCardSerial = 0
		tx_err = tx.Put( db.BucketBoards , boardKey( board_id ) , board )
		return
	} )
	if err != nil { t.Fatal( err ) }
}

func TestEnsureCardIdentity( t *testing.T ) {
	store := testStore( t )
	owner , _ := CreateUser( store , "Owner" , RoleAdmin )
	board , _ := CreateBoard( store , owner.ID , "Old" , []string{ "To do" } )
	todo := board.SortedLists()[ 0 ].ID
	empty , _ := CreateBoard( store , owner.ID , "Empty" , []string{ "To do" } )
	first , _ := CreateCard( store , board.ID , todo , "First" , nil , owner.ID , "Owner" )
	gone , _ := CreateCard( store , board.ID , todo , "Trashed" , nil , owner.ID , "Owner" )
	last , _ := CreateCard( store , board.ID , todo , "Last" , nil , owner.ID , "Owner" )
	if err := TrashCard( store , board.ID , gone.ID , "Owner" ); err != nil { t.Fatal( err ) }
	forgetIdentity( t , store , board.ID )
	before_card , _ := GetCard( store , board.ID , first.ID )
	before_board , _ := GetBoard( store , board.ID )

	if err := EnsureCardIdentity( store ); err != nil { t.Fatal( err ) }

	numbered , _ := GetCard( store , board.ID , first.ID )
	final , _ := GetCard( store , board.ID , last.ID )
	if numbered.Serial != 1 || final.Serial != 3 { t.Errorf( "oldest first: got %d and %d, want 1 and 3" , numbered.Serial , final.Serial ) }
	if len( numbered.UID ) != 26 { t.Errorf( "no ULID: %q" , numbered.UID ) }
	if want := newULID( numbered.CreatedAt )[ :10 ]; numbered.UID[ :10 ] != want { t.Errorf( "ULID time should be the card's: %s, want %s" , numbered.UID , want ) }
	if numbered.Rev != before_card.Rev { t.Errorf( "numbering is not an edit: card rev %d, was %d" , numbered.Rev , before_card.Rev ) }
	after_board , _ := GetBoard( store , board.ID )
	if after_board.Rev != before_board.Rev { t.Errorf( "numbering is not an edit: board rev %d, was %d" , after_board.Rev , before_board.Rev ) }
	if after_board.NextCardSerial != 3 { t.Errorf( "counter: got %d, want 3" , after_board.NextCardSerial ) }

	// The trashed card was numbered too, and keeps it when it comes back.
	restored , err := RestoreCard( store , board.ID , gone.ID , "Owner" )
	if err != nil { t.Fatal( err ) }
	if restored.Serial != 2 || restored.UID == "" { t.Errorf( "trashed card not numbered: %+v" , restored ) }

	// Running again changes nothing, and new cards carry on from the last.
	uid := numbered.UID
	if err = EnsureCardIdentity( store ); err != nil { t.Fatal( err ) }
	again , _ := GetCard( store , board.ID , first.ID )
	if again.Serial != 1 || again.UID != uid { t.Errorf( "a second run renumbered: %+v" , again ) }
	next , _ := CreateCard( store , board.ID , todo , "Next" , nil , owner.ID , "Owner" )
	if next.Serial != 4 { t.Errorf( "got serial %d, want 4" , next.Serial ) }

	// An empty board stays ready to start at 1.
	still , _ := GetBoard( store , empty.ID )
	if still.NextCardSerial != 0 { t.Errorf( "empty board counter: %d" , still.NextCardSerial ) }
}

func TestPriceText( t *testing.T ) {
	fetched := &Product{ Price: "$30" , Status: ProductOK }
	failed := &Product{ Price: "$30" , Status: ProductFailed }
	cases := []struct {
		card *Card
		want string
	}{
		{ &Card{} , "" },
		{ &Card{ Price: "$25" } , "$25" },
		{ &Card{ Product: fetched } , "$30" },
		{ &Card{ Price: "$25" , Product: fetched } , "$25" },
		{ &Card{ Product: failed } , "" },
	}
	for _ , item := range cases {
		if got := item.card.PriceText(); got != item.want { t.Errorf( "%+v: got %q, want %q" , item.card , got , item.want ) }
	}
}

func TestSnapshotKeepsPriceAndShopping( t *testing.T ) {
	store := testStore( t )
	owner , _ := CreateUser( store , "Owner" , RoleAdmin )
	board , _ := CreateBoard( store , owner.ID , "Shop" , []string{ "To buy" , "Bought" } )
	board , _ = MutateBoard( store , board.ID , func( board *Board ) error {
		board.Shopping = true
		board.SortedLists()[ 1 ].Shopping = true
		return nil
	} )
	todo := board.SortedLists()[ 0 ].ID
	typed , _ := CreateCard( store , board.ID , todo , "Typed" , nil , owner.ID , "Owner" )
	found , _ := CreateCard( store , board.ID , todo , "Fetched" , nil , owner.ID , "Owner" )
	CreateCard( store , board.ID , todo , "Plain" , nil , owner.ID , "Owner" )
	price := "$24.99"
	PatchCard( store , board.ID , typed.ID , &CardPatch{ Price: &price } , "Owner" )
	if _ , err := SetCardProduct( store , board.ID , found.ID , "" , &Product{ URL: "https://example.com" , Price: "$30" , Status: ProductOK } , nil ); err != nil { t.Fatal( err ) }

	cards , _ := ListCards( store , board.ID )
	snap := SnapshotBoard( board , cards , everyCard , 0 , nil )
	copy , lists , err := ImportBoardSkeleton( store , owner.ID , snap )
	if err != nil { t.Fatal( err ) }
	if copy.Shopping == false || copy.List( lists[ 1 ] ).Shopping == false || copy.List( lists[ 0 ] ).Shopping {
		t.Errorf( "shopping flags did not survive: board %v, lists %+v" , copy.Shopping , copy.Lists )
	}
	if _ , err = ImportCards( store , copy.ID , lists[ 0 ] , snap.Labels , snap.Lists[ 0 ].Cards , owner.ID , "Owner" ); err != nil { t.Fatal( err ) }

	copied , _ := ListCards( store , copy.ID )
	prices := map[string]string{}
	uids := map[string]bool{ typed.UID: true }
	for _ , card := range copied {
		prices[ card.Title ] = card.Price
		if uids[ card.UID ] || card.UID == "" { t.Errorf( "imported card needs a fresh ULID: %q" , card.UID ) }
		uids[ card.UID ] = true
	}
	if prices[ "Typed" ] != "$24.99" || prices[ "Fetched" ] != "$30" || prices[ "Plain" ] != "" {
		t.Errorf( "prices did not survive: %+v" , prices )
	}
	serials := map[uint64]bool{}
	for _ , card := range copied { serials[ card.Serial ] = true }
	if len( serials ) != 3 || serials[ 1 ] == false || serials[ 3 ] == false { t.Errorf( "imported cards should number from 1: %+v" , serials ) }
	next , _ := CreateCard( store , copy.ID , lists[ 0 ] , "After" , nil , owner.ID , "Owner" )
	if next.Serial != 4 { t.Errorf( "got serial %d after an import, want 4" , next.Serial ) }

	for _ , bad := range []string{ strings.Repeat( "9" , maxPriceChars + 1 ) , "line\nbreak" } {
		_ , err = ImportCards( store , copy.ID , lists[ 0 ] , nil , []*CardSnapshot{ { Title: "Bad" , Price: bad } } , owner.ID , "Owner" )
		if errors.Is( err , ErrBadInput ) == false { t.Errorf( "price %q: got %v, want ErrBadInput" , bad , err ) }
	}
}
