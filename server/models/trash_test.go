package models

import (
	errors "errors"
	testing "testing"
	time "time"

	db "todoboard/server/db"
)

func TestTrashRestoreAndPurge( t *testing.T ) {
	store := testStore( t )
	owner , _ := CreateUser( store , "Owner" , RoleAdmin )
	board , _ := CreateBoard( store , owner.ID , "B" , []string{ "To do" , "Doing" , "Done" } )
	todo := board.SortedLists()[ 0 ].ID
	doing := board.SortedLists()[ 1 ].ID

	card , _ := CreateCard( store , board.ID , todo , "Keep me" , nil , owner.ID , "Owner" )
	if err := TrashCard( store , board.ID , card.ID , "Owner" ); err != nil { t.Fatal( err ) }
	if _ , err := GetCard( store , board.ID , card.ID ); errors.Is( err , ErrCardNotFound ) == false {
		t.Errorf( "a trashed card is off the board, got %v" , err )
	}
	trashed , _ := ListTrash( store , board.ID )
	if len( trashed ) != 1 || trashed[ 0 ].ID != card.ID || trashed[ 0 ].DeletedBy != "Owner" {
		t.Fatalf( "the card should be in the trash: %+v" , trashed )
	}

	restored , err := RestoreCard( store , board.ID , card.ID , "Owner" )
	if err != nil { t.Fatal( err ) }
	if restored.ID != card.ID || restored.ListID != todo || restored.DeletedAt != nil {
		t.Errorf( "restore keeps the id and list: %+v" , restored )
	}
	if trashed , _ = ListTrash( store , board.ID ); len( trashed ) != 0 {
		t.Errorf( "restored card should leave the trash" )
	}

	// Deleting a column trashes its cards; restoring one lands in the first
	// open column.
	moved , _ := CreateCard( store , board.ID , doing , "On doing" , nil , owner.ID , "Owner" )
	if err = DeleteList( store , board.ID , doing , "Owner" ); err != nil { t.Fatal( err ) }
	if restored , err = RestoreCard( store , board.ID , moved.ID , "Owner" ); err != nil { t.Fatal( err ) }
	if restored.ListID != todo {
		t.Errorf( "a card whose column is gone goes to the first column, got list %d" , restored.ListID )
	}

	// Purge and empty.
	TrashCard( store , board.ID , card.ID , "Owner" )
	TrashCard( store , board.ID , moved.ID , "Owner" )
	if err = PurgeCard( store , board.ID , card.ID ); err != nil { t.Fatal( err ) }
	if _ , err = RestoreCard( store , board.ID , card.ID , "Owner" ); errors.Is( err , ErrNotInTrash ) == false {
		t.Errorf( "a purged card cannot come back, got %v" , err )
	}
	if removed , _ := EmptyTrash( store , board.ID ); removed != 1 {
		t.Errorf( "empty should remove the one left, removed %d" , removed )
	}
}

func TestTrashExpires( t *testing.T ) {
	store := testStore( t )
	owner , _ := CreateUser( store , "Owner" , RoleAdmin )
	board , _ := CreateBoard( store , owner.ID , "B" , []string{ "L" } )
	card , _ := CreateCard( store , board.ID , board.Lists[ 0 ].ID , "old" , nil , owner.ID , "Owner" )
	TrashCard( store , board.ID , card.ID , "Owner" )
	store.Write( func( tx *db.Tx ) error {
		stored , _ := getTrashedTx( tx , board.ID , card.ID )
		long_ago := time.Now().UTC().Add( -trashRetention - time.Hour )
		stored.DeletedAt = &long_ago
		return tx.Put( db.BucketTrash , cardKey( board.ID , card.ID ) , stored )
	} )
	if trashed , _ := ListTrash( store , board.ID ); len( trashed ) != 0 {
		t.Errorf( "cards past retention should be purged on read" )
	}
}
