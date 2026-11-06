package models

import (
	errors "errors"
	strings "strings"
	testing "testing"

	db "todoboard/server/db"
)

func TestBoardPlacementSettings( t *testing.T ) {
	store := testStore( t )
	owner , _ := CreateUser( store , "Owner" , RoleAdmin )
	board , err := CreateBoard( store , owner.ID , "Plans" , []string{ "To do" } )
	if err != nil { t.Fatal( err ) }
	if board.DonePlacement != PlacementDefault || board.AddPlacement != PlacementDefault {
		t.Fatalf( "a new board leaves placement to each person: %+v" , board )
	}
	top , bottom := PlacementTop , PlacementBottom
	updated , err := UpdateBoardSettings( store , board.ID , &BoardSettings{ DonePlacement: &top , AddPlacement: &bottom } )
	if err != nil { t.Fatal( err ) }
	if updated.DonePlacement != PlacementTop || updated.AddPlacement != PlacementBottom {
		t.Errorf( "placements not stored: %+v" , updated )
	}
	bad := "middle"
	if _ , err = UpdateBoardSettings( store , board.ID , &BoardSettings{ DonePlacement: &bad } ); errors.Is( err , ErrBadInput ) == false {
		t.Errorf( "unknown placement: got %v, want ErrBadInput" , err )
	}
}

func TestCardManualPrice( t *testing.T ) {
	store := testStore( t )
	owner , _ := CreateUser( store , "Owner" , RoleAdmin )
	board , _ := CreateBoard( store , owner.ID , "Shop" , []string{ "To buy" } )
	card , _ := CreateCard( store , board.ID , board.SortedLists()[ 0 ].ID , "Kettle" , nil , owner.ID , "Owner" )

	price := "  $24.99 "
	patched , change , err := PatchCard( store , board.ID , card.ID , &CardPatch{ Price: &price } , "Owner" )
	if err != nil { t.Fatal( err ) }
	if patched.Price != "$24.99" || change.Edited == false {
		t.Errorf( "price not trimmed and stored: %q %+v" , patched.Price , change )
	}

	// Looking the product up again leaves the typed price alone.
	if _ , err = SetCardProduct( store , board.ID , card.ID , "" , &Product{ URL: "https://example.com" , Price: "$30" , Status: ProductOK } , nil ); err != nil { t.Fatal( err ) }
	again , _ := GetCard( store , board.ID , card.ID )
	if again.Price != "$24.99" || again.Product.Price != "$30" {
		t.Errorf( "a product lookup must not touch the typed price: %q / %q" , again.Price , again.Product.Price )
	}

	long := strings.Repeat( "9" , maxPriceChars + 1 )
	if _ , _ , err = PatchCard( store , board.ID , card.ID , &CardPatch{ Price: &long } , "Owner" ); errors.Is( err , ErrBadInput ) == false {
		t.Errorf( "overlong price: got %v, want ErrBadInput" , err )
	}
	empty := ""
	cleared , _ , _ := PatchCard( store , board.ID , card.ID , &CardPatch{ Price: &empty } , "Owner" )
	if cleared.Price != "" { t.Errorf( "an empty price clears it, got %q" , cleared.Price ) }
}

func TestCardImage( t *testing.T ) {
	store := testStore( t )
	owner , _ := CreateUser( store , "Owner" , RoleAdmin )
	board , _ := CreateBoard( store , owner.ID , "Pics" , []string{ "To do" } )
	card , _ := CreateCard( store , board.ID , board.SortedLists()[ 0 ].ID , "Photo" , nil , owner.ID , "Owner" )

	image := &ProductImage{ ContentType: "image/png" , Data: []byte( "\x89PNG\r\n\x1a\nrest" ) }
	stored , err := SetCardImage( store , board.ID , card.ID , image )
	if err != nil { t.Fatal( err ) }
	if stored.ImageAt == nil || stored.Rev != card.Rev {
		t.Errorf( "a picture sets image_at and leaves the card's rev: %+v" , stored )
	}
	got , err := GetCardImage( store , board.ID , card.ID )
	if err != nil || string( got.Data ) != string( image.Data ) { t.Fatalf( "picture not read back: %v" , err ) }

	big := &ProductImage{ ContentType: "image/png" , Data: make( []byte , MaxCardImageBytes + 1 ) }
	if _ , err = SetCardImage( store , board.ID , card.ID , big ); errors.Is( err , ErrBadInput ) == false {
		t.Errorf( "oversized picture: got %v, want ErrBadInput" , err )
	}

	removed , _ := SetCardImage( store , board.ID , card.ID , nil )
	if removed.ImageAt != nil { t.Errorf( "removing clears image_at" ) }
	if _ , err = GetCardImage( store , board.ID , card.ID ); errors.Is( err , db.ErrNotFound ) == false {
		t.Errorf( "removed picture still stored: %v" , err )
	}

	// Purging the card from the trash takes its picture with it.
	if _ , err = SetCardImage( store , board.ID , card.ID , image ); err != nil { t.Fatal( err ) }
	if err = TrashCard( store , board.ID , card.ID , "Owner" ); err != nil { t.Fatal( err ) }
	if err = PurgeCard( store , board.ID , card.ID ); err != nil { t.Fatal( err ) }
	if _ , err = GetCardImage( store , board.ID , card.ID ); errors.Is( err , db.ErrNotFound ) == false {
		t.Errorf( "a purged card's picture must go too: %v" , err )
	}
}

func stagedBoard( t *testing.T ) ( store *db.Store , board *Board , list uint64 , owner *User ) {
	t.Helper()
	store = testStore( t )
	owner , _ = CreateUser( store , "Owner" , RoleAdmin )
	board , err := CreateBoard( store , owner.ID , "Staged" , []string{ "Work" , "Home" } )
	if err != nil { t.Fatal( err ) }
	layout := LayoutStaged
	if board , err = UpdateBoardSettings( store , board.ID , &BoardSettings{ Layout: &layout } ); err != nil { t.Fatal( err ) }
	list = board.SortedLists()[ 0 ].ID
	return
}

func rankCard( t *testing.T , store *db.Store , board *Board , id uint64 , place int ) ( card *Card ) {
	t.Helper()
	card , _ , err := PatchCard( store , board.ID , id , &CardPatch{ RankPlace: &place } , "Owner" )
	if err != nil { t.Fatal( err ) }
	return
}

// placesOf is the ranking as card titles, most important first.
func placesOf( t *testing.T , store *db.Store , board_id uint64 ) ( titles string ) {
	t.Helper()
	board , _ := GetBoard( store , board_id )
	cards , err := ListCards( store , board_id )
	if err != nil { t.Fatal( err ) }
	places := RankPlaces( board , cards )
	ordered := make( []string , len( places ) )
	for _ , card := range cards {
		if place := places[ card.ID ]; place > 0 { ordered[ place - 1 ] = card.Title }
	}
	titles = strings.Join( ordered , " " )
	return
}

func TestStagedLayoutSetting( t *testing.T ) {
	store , board , _ , _ := stagedBoard( t )
	if board.Layout != LayoutStaged { t.Fatalf( "layout not stored: %q" , board.Layout ) }
	if len( board.SortedLists() ) != 3 { t.Errorf( "a staged board adds no lists of its own: %+v" , board.Lists ) }
	bad := "four"
	if _ , err := UpdateBoardSettings( store , board.ID , &BoardSettings{ Layout: &bad } ); errors.Is( err , ErrBadInput ) == false {
		t.Errorf( "unknown layout: got %v, want ErrBadInput" , err )
	}
}

func TestCardStages( t *testing.T ) {
	store , board , list , owner := stagedBoard( t )
	card , _ := CreateCard( store , board.ID , list , "Task" , nil , owner.ID , "Owner" )
	yes , no := true , false

	doing , change , err := PatchCard( store , board.ID , card.ID , &CardPatch{ Doing: &yes } , "Owner" )
	if err != nil { t.Fatal( err ) }
	if doing.Doing == false || doing.Done || change.Edited == false { t.Errorf( "not started: %+v %+v" , doing , change ) }

	done , change , err := PatchCard( store , board.ID , card.ID , &CardPatch{ Done: &yes } , "Owner" )
	if err != nil { t.Fatal( err ) }
	if done.Doing || done.Done == false || done.DoneAt == nil || change.Completed == false {
		t.Errorf( "finishing a card leaves Doing: %+v %+v" , done , change )
	}

	again , change , err := PatchCard( store , board.ID , card.ID , &CardPatch{ Doing: &yes } , "Owner" )
	if err != nil { t.Fatal( err ) }
	if again.Doing == false || again.Done || again.DoneAt != nil || change.Reopened == false {
		t.Errorf( "starting a finished card reopens it: %+v %+v" , again , change )
	}

	if _ , _ , err = PatchCard( store , board.ID , card.ID , &CardPatch{ Doing: &yes , Done: &yes } , "Owner" ); errors.Is( err , ErrBadInput ) == false {
		t.Errorf( "both stages at once: got %v, want ErrBadInput" , err )
	}
	back , _ , err := PatchCard( store , board.ID , card.ID , &CardPatch{ Doing: &no , Done: &no } , "Owner" )
	if err != nil { t.Fatal( err ) }
	if back.Doing || back.Done { t.Errorf( "both flags off is To do: %+v" , back ) }
}

func TestRankPlaces( t *testing.T ) {
	store , board , list , owner := stagedBoard( t )
	other := board.SortedLists()[ 1 ].ID
	ids := map[string]uint64{}
	for index , title := range []string{ "a" , "b" , "c" , "d" } {
		target := list
		if index % 2 == 1 { target = other }
		card , _ := CreateCard( store , board.ID , target , title , nil , owner.ID , "Owner" )
		ids[ title ] = card.ID
		rankCard( t , store , board , card.ID , 99 )
	}
	if got := placesOf( t , store , board.ID ); got != "a b c d" { t.Fatalf( "a place past the end means last, across lists: %q" , got ) }

	rankCard( t , store , board , ids[ "d" ] , 2 )
	if got := placesOf( t , store , board.ID ); got != "a d b c" { t.Errorf( "moved up: %q" , got ) }
	rankCard( t , store , board , ids[ "a" ] , 3 )
	if got := placesOf( t , store , board.ID ); got != "d b a c" { t.Errorf( "moved down: %q" , got ) }

	// Finishing #1 moves everyone up; reopening puts it back.
	yes , no := true , false
	PatchCard( store , board.ID , ids[ "d" ] , &CardPatch{ Done: &yes } , "Owner" )
	if got := placesOf( t , store , board.ID ); got != "b a c" { t.Errorf( "after finishing the first: %q" , got ) }
	three := 1
	if _ , _ , err := PatchCard( store , board.ID , ids[ "d" ] , &CardPatch{ RankPlace: &three } , "Owner" ); errors.Is( err , ErrBadInput ) == false {
		t.Errorf( "a finished card takes no place: got %v" , err )
	}
	PatchCard( store , board.ID , ids[ "d" ] , &CardPatch{ Done: &no } , "Owner" )
	if got := placesOf( t , store , board.ID ); got != "d b a c" { t.Errorf( "after reopening: %q" , got ) }

	// Shelving a list takes its cards out of the numbering.
	shelved := true
	if err := store.Write( func( tx *db.Tx ) ( tx_err error ) {
		current , tx_err := getBoardTx( tx , board.ID )
		if tx_err != nil { return }
		if tx_err = ( &ListUpdate{ Archived: &shelved } ).Apply( current , other ); tx_err != nil { return }
		return putBoardTx( tx , current )
	} ); err != nil { t.Fatal( err ) }
	if got := placesOf( t , store , board.ID ); got != "a c" { t.Errorf( "after archiving a list: %q" , got ) }

	board , _ = GetBoard( store , board.ID )
	cleared := rankCard( t , store , board , ids[ "a" ] , 0 )
	if cleared.Rank != 0 { t.Errorf( "0 takes the card out: %v" , cleared.Rank ) }
	if got := placesOf( t , store , board.ID ); got != "c" { t.Errorf( "after clearing: %q" , got ) }

	negative := -1
	if _ , _ , err := PatchCard( store , board.ID , ids[ "a" ] , &CardPatch{ RankPlace: &negative } , "Owner" ); errors.Is( err , ErrBadInput ) == false {
		t.Errorf( "negative place: got %v" , err )
	}
}

// Ranks crowding together respace without touching anyone else's Rev: Rev is
// what refuses a stale title edit.
func TestRankRespaceLeavesRev( t *testing.T ) {
	store , board , list , owner := stagedBoard( t )
	first , _ := CreateCard( store , board.ID , list , "first" , nil , owner.ID , "Owner" )
	second , _ := CreateCard( store , board.ID , list , "second" , nil , owner.ID , "Owner" )
	third , _ := CreateCard( store , board.ID , list , "third" , nil , owner.ID , "Owner" )
	unranked , _ := CreateCard( store , board.ID , list , "unranked" , nil , owner.ID , "Owner" )
	for _ , card := range []*Card{ first , second , third } { rankCard( t , store , board , card.ID , 99 ) }
	bystander , _ := GetCard( store , board.ID , first.ID )
	// Halving the gap under the first card runs out of room quickly.
	for round := 0; round < 80; round += 1 {
		rankCard( t , store , board , second.ID , 1 )
		rankCard( t , store , board , third.ID , 1 )
	}
	if got := placesOf( t , store , board.ID ); got != "third second first" { t.Fatalf( "order lost in respacing: %q" , got ) }
	after , _ := GetCard( store , board.ID , first.ID )
	if after.Rev != bystander.Rev { t.Errorf( "a bystander's rev moved: %d to %d" , bystander.Rev , after.Rev ) }
	idle , _ := GetCard( store , board.ID , unranked.ID )
	if idle.Rank != 0 { t.Errorf( "an unranked card stays unranked: %v" , idle.Rank ) }
}
