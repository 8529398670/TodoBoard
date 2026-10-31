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
	if _ , err = SetCardProduct( store , board.ID , card.ID , &Product{ URL: "https://example.com" , Price: "$30" , Status: ProductOK } , nil ); err != nil { t.Fatal( err ) }
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
