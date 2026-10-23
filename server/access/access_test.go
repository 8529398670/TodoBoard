package access

import (
	testing "testing"
	time "time"

	models "todoboard/server/models"
)

// The whole permission table from the plan, row by row. If a row here
// changes, it should be because the rules changed on purpose.
func TestLevels( t *testing.T ) {
	board := &models.Board{ ID: 1 , OwnerID: 10 , Members: map[uint64]string{ 11: models.BoardRoleEditor , 12: models.BoardRoleViewer } }
	card := &models.Card{ ID: 5 , BoardID: 1 , SharedWith: map[uint64]string{ 13: models.ModeEdit , 14: models.ModeView , 12: models.ModeEdit } }
	other_card := &models.Card{ ID: 6 , BoardID: 1 , SharedWith: map[uint64]string{} }
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
