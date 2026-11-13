package models

import (
	errors "errors"
	testing "testing"
	time "time"

	db "todoboard/server/db"
)

func TestBoardTrashRestoreAndPurge( t *testing.T ) {
	store := testStore( t )
	owner , _ := CreateUser( store , "Owner" , RoleAdmin )
	member , _ := CreateUser( store , "Member" , RoleUser )
	board , _ := CreateBoard( store , owner.ID , "Lab" , []string{ "To do" , "Doing" , "Done" } )
	SetMember( store , board.ID , member.ID , BoardRoleEditor )
	card , _ := CreateCard( store , board.ID , board.SortedLists()[ 0 ].ID , "Keep me" , nil , owner.ID , "Owner" )

	if err := TrashBoard( store , board.ID , owner.ID ); err != nil { t.Fatal( err ) }
	if _ , err := GetBoard( store , board.ID ); errors.Is( err , ErrBoardNotFound ) == false {
		t.Errorf( "a trashed board reads as not found, got %v" , err )
	}
	for _ , user_id := range []uint64{ owner.ID , member.ID } {
		if boards , _ , _ := ListBoardsForUser( store , user_id ); len( boards ) != 0 {
			t.Errorf( "a trashed board is off every home page: %+v" , boards )
		}
	}
	// The slug stays claimed, so a restore gets its URL back.
	if BoardIDForSlug( store , "lab" ) != board.ID { t.Error( "the slug is kept while trashed" ) }

	trashed , _ := ListTrashedBoards( store , owner.ID )
	if len( trashed ) != 1 || trashed[ 0 ].ID != board.ID { t.Fatalf( "owner's trash: %+v" , trashed ) }
	if theirs , _ := ListTrashedBoards( store , member.ID ); len( theirs ) != 0 {
		t.Errorf( "a member has no board trash of the owner's boards: %+v" , theirs )
	}
	if _ , err := RestoreBoard( store , member.ID , board.ID ); errors.Is( err , ErrBoardNotInTrash ) == false {
		t.Errorf( "only the owner restores, got %v" , err )
	}

	restored , err := RestoreBoard( store , owner.ID , board.ID )
	if err != nil { t.Fatal( err ) }
	if restored.Trashed() || restored.Members[ member.ID ] != BoardRoleEditor {
		t.Errorf( "restore brings the board back as it was: %+v" , restored )
	}
	if _ , err = GetCard( store , board.ID , card.ID ); err != nil { t.Errorf( "cards come back too: %v" , err ) }
	if boards , _ , _ := ListBoardsForUser( store , member.ID ); len( boards ) != 1 { t.Error( "members get it back" ) }
	if _ , err = RestoreBoard( store , owner.ID , board.ID ); errors.Is( err , ErrBoardNotInTrash ) == false {
		t.Errorf( "restoring a board not in the trash, got %v" , err )
	}

	// Delete forever needs it in the trash first.
	if err = PurgeTrashedBoard( store , owner.ID , board.ID ); errors.Is( err , ErrBoardNotInTrash ) == false {
		t.Errorf( "purging a live board, got %v" , err )
	}
	TrashBoard( store , board.ID , owner.ID )
	if err = PurgeTrashedBoard( store , member.ID , board.ID ); errors.Is( err , ErrBoardNotInTrash ) == false {
		t.Errorf( "a member cannot purge, got %v" , err )
	}
	if err = PurgeTrashedBoard( store , owner.ID , board.ID ); err != nil { t.Fatal( err ) }
	if _ , err = getBoardAny( store , board.ID ); errors.Is( err , ErrBoardNotFound ) == false {
		t.Errorf( "purged board is gone, got %v" , err )
	}
	if BoardIDForSlug( store , "lab" ) != 0 { t.Error( "a purged board frees its slug" ) }
	if GetBoardRef( store , member.ID , board.ID ) != nil { t.Error( "a purged board leaves no member rows" ) }
}

func TestBoardTrashNeverExpiresAndEmpties( t *testing.T ) {
	store := testStore( t )
	owner , _ := CreateUser( store , "Owner" , RoleAdmin )
	other , _ := CreateUser( store , "Other" , RoleUser )
	first , _ := CreateBoard( store , owner.ID , "First" , nil )
	second , _ := CreateBoard( store , owner.ID , "Second" , nil )
	theirs , _ := CreateBoard( store , other.ID , "Theirs" , nil )
	TrashBoard( store , first.ID , owner.ID )
	TrashBoard( store , second.ID , owner.ID )
	TrashBoard( store , theirs.ID , other.ID )

	// Deleted years ago and still there.
	store.Write( func( tx *db.Tx ) error {
		board , _ := getBoardAnyTx( tx , first.ID )
		long_ago := time.Now().UTC().Add( -5 * 365 * 24 * time.Hour )
		board.DeletedAt = &long_ago
		return tx.Put( db.BucketBoards , boardKey( first.ID ) , board )
	} )
	trashed , _ := ListTrashedBoards( store , owner.ID )
	if len( trashed ) != 2 || trashed[ 0 ].ID != second.ID { t.Fatalf( "newest first, nothing expires: %+v" , trashed ) }

	removed , err := EmptyBoardTrash( store , owner.ID )
	if err != nil || removed != 2 { t.Fatalf( "removed %d, %v" , removed , err ) }
	if left , _ := ListTrashedBoards( store , other.ID ); len( left ) != 1 {
		t.Error( "emptying my trash leaves other people's alone" )
	}
}

func TestBoardArchive( t *testing.T ) {
	store := testStore( t )
	owner , _ := CreateUser( store , "Owner" , RoleAdmin )
	member , _ := CreateUser( store , "Member" , RoleUser )
	board , _ := CreateBoard( store , owner.ID , "Lab" , nil )
	SetMember( store , board.ID , member.ID , BoardRoleViewer )

	summary := func( user_id uint64 ) ( result *BoardSummary ) {
		boards , _ , _ := ListBoardsForUser( store , user_id )
		if len( boards ) == 1 { result = boards[ 0 ] }
		return
	}

	// For me: only my own row changes.
	if err := SetBoardArchivedFor( store , member.ID , board.ID , true ); err != nil { t.Fatal( err ) }
	if mine := summary( member.ID ); mine == nil || mine.ArchivedForMe == false || mine.Archived {
		t.Errorf( "archived for me: %+v" , mine )
	}
	if owners := summary( owner.ID ); owners.ArchivedForMe || owners.Archived {
		t.Errorf( "nobody else sees it: %+v" , owners )
	}
	SetBoardArchivedFor( store , member.ID , board.ID , false )
	if summary( member.ID ).ArchivedForMe { t.Error( "unarchived for me" ) }

	// For everyone.
	archived , err := SetBoardArchived( store , board.ID , true , "Owner" )
	if err != nil || archived.Archived() == false || archived.ArchivedBy != "Owner" { t.Fatalf( "%+v %v" , archived , err ) }
	if summary( member.ID ).Archived == false || summary( owner.ID ).Archived == false {
		t.Error( "archived for everyone shows on every home page" )
	}
	if unarchived , _ := SetBoardArchived( store , board.ID , false , "Owner" ); unarchived.Archived() {
		t.Error( "unarchived for everyone" )
	}

	if err = SetBoardArchivedFor( store , member.ID , 9999 , true ); errors.Is( err , ErrBoardNotFound ) == false {
		t.Errorf( "not on that board, got %v" , err )
	}
}
