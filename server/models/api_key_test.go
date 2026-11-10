package models

import (
	errors "errors"
	filepath "path/filepath"
	strings "strings"
	testing "testing"
	time "time"

	config "todoboard/server/config"
	db "todoboard/server/db"
)

func keyTestStore( t *testing.T ) ( store *db.Store ) {
	t.Helper()
	cfg := &config.Config{ DatabasePath: filepath.Join( t.TempDir() , "test.db" ) , SecretKey: strings.Repeat( "ab" , 32 ) }
	store , err := db.Open( cfg )
	if err != nil { t.Fatal( err ) }
	t.Cleanup( func() { store.Close() } )
	return
}

// A row written before board limits existed has neither field. It must keep
// working as it always did, on every board.
func TestOldKeyRowHasNoLimit( t *testing.T ) {
	store := keyTestStore( t )
	old := map[ string ]any{ "id": "oldkey" , "user_id": 1 , "name": "cron" , "role": RoleUser , "secret_hash": "x" , "created_at": time.Now().UTC() }
	if err := store.Put( db.BucketAPIKeys , []byte( "oldkey" ) , old ); err != nil { t.Fatal( err ) }
	key , err := GetAPIKey( store , "oldkey" )
	if err != nil { t.Fatal( err ) }
	if key.Limited() || key.OnlyBoards() || key.AllowsBoard( 42 ) == false {
		t.Errorf( "old row is limited: %+v" , key )
	}
}

func TestAPIKeyBoardRules( t *testing.T ) {
	store := keyTestStore( t )
	owner , err := CreateUser( store , "Owner" , RoleAdmin )
	if err != nil { t.Fatal( err ) }
	other , err := CreateUser( store , "Other" , RoleUser )
	if err != nil { t.Fatal( err ) }
	mine , err := CreateBoard( store , owner.ID , "Mine" , nil )
	if err != nil { t.Fatal( err ) }
	theirs , err := CreateBoard( store , other.ID , "Theirs" , nil )
	if err != nil { t.Fatal( err ) }
	shared , err := CreateBoard( store , other.ID , "Shared" , nil )
	if err != nil { t.Fatal( err ) }
	card , err := CreateCard( store , shared.ID , shared.Lists[ 0 ].ID , "One" , nil , other.ID , "Other" )
	if err != nil { t.Fatal( err ) }
	if _ , err = ShareCardWithUser( store , shared.ID , card.ID , owner.ID , ModeView , "Other" ); err != nil { t.Fatal( err ) }

	refused := []struct {
		name string
		role string
		mode string
		ids  []uint64
	}{
		{ "unknown mode" , RoleUser , "sometimes" , []uint64{ mine.ID } },
		{ "only, nothing chosen" , RoleUser , BoardAccessOnly , nil },
		{ "except, nothing chosen" , RoleUser , BoardAccessExcept , []uint64{} },
		{ "a board that is not yours" , RoleUser , BoardAccessOnly , []uint64{ mine.ID , theirs.ID } },
		{ "a board that does not exist" , RoleUser , BoardAccessExcept , []uint64{ 9999 } },
		{ "an admin key" , RoleAdmin , BoardAccessOnly , []uint64{ mine.ID } },
		{ "too many" , RoleUser , BoardAccessOnly , make( []uint64 , APIKeyMaxBoards+1 ) },
	}
	for _ , item := range refused {
		if _ , _ , err := IssueAPIKeyLimited( store , owner.ID , "k" , item.role , time.Hour , item.mode , item.ids ); errors.Is( err , ErrAPIKeyBoards ) == false {
			t.Errorf( "%s: got %v" , item.name , err )
		}
	}

	// Duplicates fold, and a board reached through one shared card counts.
	_ , key , err := IssueAPIKeyLimited( store , owner.ID , "k" , RoleUser , time.Hour , BoardAccessOnly , []uint64{ shared.ID , mine.ID , mine.ID } )
	if err != nil { t.Fatal( err ) }
	if len( key.BoardIDs ) != 2 || key.AllowsBoard( mine.ID ) == false || key.AllowsBoard( shared.ID ) == false || key.AllowsBoard( theirs.ID ) {
		t.Errorf( "issued: %+v" , key )
	}

	// An edit is held to the same rules, and survives the trip through storage.
	if err = SetAPIKeyBoards( store , key.ID , BoardAccessExcept , []uint64{ theirs.ID } ); errors.Is( err , ErrAPIKeyBoards ) == false {
		t.Errorf( "edit to another's board: %v" , err )
	}
	if err = SetAPIKeyBoards( store , key.ID , BoardAccessExcept , []uint64{ mine.ID } ); err != nil { t.Fatal( err ) }
	loaded , err := GetAPIKey( store , key.ID )
	if err != nil { t.Fatal( err ) }
	if loaded.AllowsBoard( mine.ID ) || loaded.AllowsBoard( theirs.ID ) == false {
		t.Errorf( "after edit: %+v" , loaded )
	}
	if err = SetAPIKeyBoards( store , key.ID , BoardAccessAll , nil ); err != nil { t.Fatal( err ) }
	if loaded , _ = GetAPIKey( store , key.ID ); loaded.Limited() || len( loaded.BoardIDs ) != 0 {
		t.Errorf( "after clearing: %+v" , loaded )
	}

	// An admin key cannot be edited into a limit, or a dead key at all.
	_ , admin_key , err := IssueAPIKey( store , owner.ID , "a" , RoleAdmin , time.Hour )
	if err != nil { t.Fatal( err ) }
	if err = SetAPIKeyBoards( store , admin_key.ID , BoardAccessOnly , []uint64{ mine.ID } ); errors.Is( err , ErrAPIKeyBoards ) == false {
		t.Errorf( "limited an admin key: %v" , err )
	}
	if err = RevokeAPIKey( store , key.ID ); err != nil { t.Fatal( err ) }
	if err = SetAPIKeyBoards( store , key.ID , BoardAccessOnly , []uint64{ mine.ID } ); errors.Is( err , ErrAPIKeyDead ) == false {
		t.Errorf( "edited a dead key: %v" , err )
	}
}
