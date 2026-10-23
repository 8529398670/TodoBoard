package access

import (
	testing "testing"
	time "time"

	models "todoboard/server/models"
)

// The whole permission table from the plan, row by row. If a row here
// changes, it should be because the rules changed on purpose.
func TestLevels( t *testing.T ) {
	board := &models.Board{ ID: 1 , OwnerID: 10 , Members: map[uint64]string{ 11: models.BoardRoleEditor , 12: models.BoardRoleViewer } ,
		Lists: []*models.List{ { ID: 7 } , { ID: 8 } } }
	card := &models.Card{ ID: 5 , BoardID: 1 , ListID: 7 , SharedWith: map[uint64]string{ 13: models.ModeEdit , 14: models.ModeView , 12: models.ModeEdit } }
	other_card := &models.Card{ ID: 6 , BoardID: 1 , ListID: 8 , SharedWith: map[uint64]string{} }
	past := time.Now().Add( -time.Hour )

	type row struct {
		name      string
		principal *Principal
		board     Level
		card      Level
		other     Level
	}
	rows := []row{
		{ "owner" , &Principal{ UserID: 10 } , Manage , Edit , Edit },
		{ "editor" , &Principal{ UserID: 11 } , Edit , Edit , Edit },
		{ "viewer, card shared for edit" , &Principal{ UserID: 12 } , View , Edit , View },
		{ "card-only editor" , &Principal{ UserID: 13 } , None , Edit , None },
		{ "card-only viewer" , &Principal{ UserID: 14 } , None , View , None },
		{ "stranger" , &Principal{ UserID: 99 } , None , None , None },
		{ "guest board view" , &Principal{ Share: &models.ShareLink{ BoardID: 1 , Mode: models.ModeView } } , View , View , View },
		{ "guest board edit" , &Principal{ Share: &models.ShareLink{ BoardID: 1 , Mode: models.ModeEdit } } , Edit , Edit , Edit },
		{ "guest card edit" , &Principal{ Share: &models.ShareLink{ BoardID: 1 , CardID: 5 , Mode: models.ModeEdit } } , None , Edit , None },
		{ "guest column view" , &Principal{ Share: &models.ShareLink{ BoardID: 1 , ListID: 7 , Mode: models.ModeView } } , None , View , None },
		{ "guest column edit" , &Principal{ Share: &models.ShareLink{ BoardID: 1 , ListID: 7 , Mode: models.ModeEdit } } , None , Edit , None },
		{ "guest other board" , &Principal{ Share: &models.ShareLink{ BoardID: 2 , Mode: models.ModeEdit } } , None , None , None },
		{ "guest revoked" , &Principal{ Share: &models.ShareLink{ BoardID: 1 , Mode: models.ModeEdit , RevokedAt: &past } } , None , None , None },
		{ "guest expired" , &Principal{ Share: &models.ShareLink{ BoardID: 1 , Mode: models.ModeEdit , ExpiresAt: &past } } , None , None , None },
		{ "nil" , nil , None , None , None },
	}
	for _ , item := range rows {
		if got := Board( item.principal , board ); got != item.board {
			t.Errorf( "%s: board level %v, want %v" , item.name , got , item.board )
		}
		if got := Card( item.principal , board , card ); got != item.card {
			t.Errorf( "%s: card level %v, want %v" , item.name , got , item.card )
		}
		if got := Card( item.principal , board , other_card ); got != item.other {
			t.Errorf( "%s: other card level %v, want %v" , item.name , got , item.other )
		}
	}

	// A card that claims to belong to another board never matches.
	foreign := &models.Card{ ID: 5 , BoardID: 2 }
	if Card( &Principal{ UserID: 10 } , board , foreign ) != None {
		t.Errorf( "a card from another board must not inherit this board's rights" )
	}
}

// A column link reaches its own column and nothing else; everyone else's
// column level is their board level, capped at Edit.
func TestListLevels( t *testing.T ) {
	board := &models.Board{ ID: 1 , OwnerID: 10 , Members: map[uint64]string{ 12: models.BoardRoleViewer } ,
		Lists: []*models.List{ { ID: 7 } , { ID: 8 } } }
	column_edit := &Principal{ Share: &models.ShareLink{ BoardID: 1 , ListID: 7 , Mode: models.ModeEdit } }
	column_view := &Principal{ Share: &models.ShareLink{ BoardID: 1 , ListID: 7 , Mode: models.ModeView } }
	card_link := &Principal{ Share: &models.ShareLink{ BoardID: 1 , CardID: 5 , Mode: models.ModeEdit } }

	checks := []struct {
		name      string
		principal *Principal
		list_id   uint64
		want      Level
	}{
		{ "owner, capped at edit" , &Principal{ UserID: 10 } , 7 , Edit },
		{ "viewer" , &Principal{ UserID: 12 } , 8 , View },
		{ "column edit, own column" , column_edit , 7 , Edit },
		{ "column edit, other column" , column_edit , 8 , None },
		{ "column view, own column" , column_view , 7 , View },
		{ "card link" , card_link , 7 , None },
		{ "missing column" , &Principal{ UserID: 10 } , 9 , None },
	}
	for _ , check := range checks {
		if got := List( check.principal , board , check.list_id ); got != check.want {
			t.Errorf( "%s: list level %v, want %v" , check.name , got , check.want )
		}
	}
	if ListOnly( column_edit , board ) != 7 || CardOnly( column_edit , board ) {
		t.Errorf( "a column link is list-only, not card-only" )
	}
	if ListOnly( card_link , board ) != 0 || CardOnly( card_link , board ) == false {
		t.Errorf( "a card link is card-only" )
	}
}
