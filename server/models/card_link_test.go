package models

import (
	errors "errors"
	testing "testing"

	db "todoboard/server/db"
)

type linkFixture struct {
	store  *db.Store
	owner  *User
	other  *User
	home   *Board
	away   *Board
}

func newLinkFixture( t *testing.T ) ( fixture *linkFixture ) {
	t.Helper()
	store := testStore( t )
	owner , _ := CreateUser( store , "Owner" , RoleAdmin )
	other , _ := CreateUser( store , "Other" , RoleUser )
	home , _ := CreateBoard( store , owner.ID , "Home" , []string{ "To do" , "Done" } )
	away , _ := CreateBoard( store , owner.ID , "Away" , []string{ "Inbox" , "Later" } )
	fixture = &linkFixture{ store: store , owner: owner , other: other , home: home , away: away }
	return
}

func ( fixture *linkFixture ) card( t *testing.T , board *Board , title string ) ( card *Card ) {
	t.Helper()
	card , err := CreateCard( fixture.store , board.ID , board.Lists[ 0 ].ID , title , nil , fixture.owner.ID , "Owner" )
	if err != nil { t.Fatal( err ) }
	return
}

func ( fixture *linkFixture ) get( t *testing.T , board_id uint64 , card_id uint64 ) ( card *Card ) {
	t.Helper()
	card , err := GetCard( fixture.store , board_id , card_id )
	if err != nil { t.Fatalf( "card %d on %d: %v" , card_id , board_id , err ) }
	return
}

func ( fixture *linkFixture ) board( t *testing.T , board_id uint64 ) ( board *Board ) {
	t.Helper()
	board , err := GetBoard( fixture.store , board_id )
	if err != nil { t.Fatal( err ) }
	return
}

func TestSendCardMovesEverything( t *testing.T ) {
	fixture := newLinkFixture( t )
	store := fixture.store
	SetMember( store , fixture.home.ID , fixture.other.ID , BoardRoleEditor )
	home , _ := MutateBoard( store , fixture.home.ID , func( board *Board ) ( err error ) {
		_ , err = AddLabel( board , "Urgent" , "red" )
		return
	} )
	away , _ := MutateBoard( store , fixture.away.ID , func( board *Board ) ( err error ) {
		_ , err = AddLabel( board , "urgent" , "blue" )
		return
	} )
	card := fixture.card( t , home , "Travel" )
	labels := []uint64{ home.Labels[ 0 ].ID }
	people := []uint64{ fixture.owner.ID , fixture.other.ID }
	PatchCard( store , home.ID , card.ID , &CardPatch{ LabelIDs: &labels , Assignees: &people } , "Owner" )
	ShareCardWithUser( store , home.ID , card.ID , fixture.other.ID , ModeView , "Owner" )
	AddComment( store , home.ID , card.ID , fixture.owner.ID , "Owner" , "Bring passport" )
	SetCardImage( store , home.ID , card.ID , &ProductImage{ ContentType: "image/png" , Data: []byte( "png" ) } )
	before := fixture.get( t , home.ID , card.ID )

	sent , err := SendCard( store , home.ID , card.ID , TransferPlace{ BoardID: away.ID , ListID: away.Lists[ 1 ].ID } , "Owner" )
	if err != nil { t.Fatal( err ) }
	if _ , err := GetCard( store , home.ID , card.ID ); errors.Is( err , ErrCardNotFound ) == false { t.Error( "the card is still on the board it left" ) }
	if sent.ID != card.ID || sent.UID != before.UID || sent.BoardID != away.ID || sent.ListID != away.Lists[ 1 ].ID {
		t.Errorf( "sent card: %+v" , sent )
	}
	if len( sent.LabelIDs ) != 1 || sent.LabelIDs[ 0 ] != away.Labels[ 0 ].ID { t.Errorf( "labels match by name: %v" , sent.LabelIDs ) }
	if len( sent.Assignees ) != 1 || sent.Assignees[ 0 ] != fixture.owner.ID { t.Errorf( "only members stay assigned: %v" , sent.Assignees ) }
	if len( sent.SharedWith ) != 0 { t.Error( "shares stay with the board they were made on" ) }
	if comments , _ := ListComments( store , away.ID , card.ID ); len( comments ) != 1 || comments[ 0 ].BoardID != away.ID { t.Errorf( "comments: %+v" , comments ) }
	if comments , _ := ListComments( store , home.ID , card.ID ); len( comments ) != 0 { t.Error( "comments left behind" ) }
	if image , _ := GetCardImage( store , away.ID , card.ID ); image == nil { t.Error( "the picture did not come along" ) }
	if image , _ := GetCardImage( store , home.ID , card.ID ); image != nil { t.Error( "the picture stayed behind too" ) }

	// A label the new board lacks is made there.
	fresh := fixture.card( t , home , "Fresh" )
	MutateBoard( store , home.ID , func( board *Board ) ( err error ) {
		_ , err = AddLabel( board , "Work" , "green" )
		return
	} )
	home = fixture.board( t , home.ID )
	work := []uint64{ home.Labels[ 1 ].ID }
	PatchCard( store , home.ID , fresh.ID , &CardPatch{ LabelIDs: &work } , "Owner" )
	moved , err := SendCard( store , home.ID , fresh.ID , TransferPlace{ BoardID: away.ID , ListID: away.Lists[ 0 ].ID , Top: true } , "Owner" )
	if err != nil { t.Fatal( err ) }
	away = fixture.board( t , away.ID )
	if label := away.Label( moved.LabelIDs[ 0 ] ); label == nil || label.Name != "Work" || label.Color != "green" { t.Errorf( "made label: %+v" , label ) }

	// And back again, to where it was: what undo does.
	position := before.Position
	back , err := SendCard( store , away.ID , card.ID , TransferPlace{ BoardID: home.ID , ListID: before.ListID , Position: &position } , "Owner" )
	if err != nil || back.BoardID != home.ID || back.Position != position { t.Fatalf( "sent back: %+v %v" , back , err ) }
	if _ , err := SendCard( store , home.ID , card.ID , TransferPlace{ BoardID: home.ID , ListID: before.ListID } , "Owner" ); err == nil { t.Error( "sending to its own board is a move, not a send" ) }
}

func TestCopyCardMirrors( t *testing.T ) {
	fixture := newLinkFixture( t )
	store := fixture.store
	home , away := fixture.home , fixture.away
	card := fixture.card( t , home , "Plan trip" )
	AddChecklistItemTo := func( board_id uint64 , card_id uint64 , text string ) error {
		_ , err := MutateCard( store , board_id , card_id , "Owner" , func( _ *Board , target *Card ) ( inner error ) {
			_ , inner = AddChecklistItem( target , text )
			return
		} )
		return err
	}
	AddChecklistItemTo( home.ID , card.ID , "Book" )
	AddComment( store , home.ID , card.ID , fixture.owner.ID , "Owner" , "First" )

	// A plain copy is its own card.
	plain , err := CopyCard( store , home.ID , card.ID , TransferPlace{ BoardID: home.ID , ListID: home.Lists[ 1 ].ID } , MirrorNone , fixture.owner.ID , "Owner" )
	if err != nil { t.Fatal( err ) }
	if plain.ID == card.ID || plain.UID == card.UID || plain.Title != card.Title || len( plain.Checklist ) != 1 || plain.Mirror != MirrorNone { t.Errorf( "plain copy: %+v" , plain ) }
	if comments , _ := ListComments( store , home.ID , plain.ID ); len( comments ) != 1 { t.Error( "a copy brings the comments" ) }

	// Two-way: either side's changes reach the other.
	twin , err := CopyCard( store , home.ID , card.ID , TransferPlace{ BoardID: away.ID , ListID: away.Lists[ 0 ].ID } , MirrorTwoWay , fixture.owner.ID , "Owner" )
	if err != nil { t.Fatal( err ) }
	if fixture.get( t , home.ID , card.ID ).Mirror != MirrorLinked || twin.Mirror != MirrorLinked { t.Error( "both ends of a two-way link show it" ) }
	title := "Plan the trip"
	PatchCard( store , home.ID , card.ID , &CardPatch{ Title: &title } , "Owner" )
	if fixture.get( t , away.ID , twin.ID ).Title != title { t.Error( "title did not reach the twin" ) }
	yes := true
	PatchCard( store , away.ID , twin.ID , &CardPatch{ Done: &yes } , "Owner" )
	if fixture.get( t , home.ID , card.ID ).Done == false { t.Error( "done did not come back" ) }
	AddChecklistItemTo( away.ID , twin.ID , "Pack" )
	if len( fixture.get( t , home.ID , card.ID ).Checklist ) != 2 { t.Error( "checklist did not come back" ) }
	if plain_now := fixture.get( t , home.ID , plain.ID ); plain_now.Title == title { t.Error( "an unlinked copy changed" ) }

	// Each side keeps its own place.
	later := away.Lists[ 1 ].ID
	PatchCard( store , away.ID , twin.ID , &CardPatch{ ListID: &later } , "Owner" )
	if fixture.get( t , home.ID , card.ID ).ListID != home.Lists[ 0 ].ID { t.Error( "a column move crossed the link" ) }

	// Comments, by thread, both ways.
	comment , _ := AddComment( store , away.ID , twin.ID , fixture.owner.ID , "Owner" , "From away" )
	home_comments , _ := ListComments( store , home.ID , card.ID )
	if len( home_comments ) != 2 || home_comments[ 1 ].Text != "From away" { t.Fatalf( "home comments: %+v" , home_comments ) }
	EditComment( store , home.ID , card.ID , home_comments[ 1 ].ID , "Edited at home" )
	if found , _ := GetComment( store , away.ID , twin.ID , comment.ID ); found.Text != "Edited at home" { t.Errorf( "edit: %q" , found.Text ) }
	RemoveComment( store , away.ID , twin.ID , comment.ID )
	if home_comments , _ = ListComments( store , home.ID , card.ID ); len( home_comments ) != 1 { t.Error( "a removed comment stayed on the other side" ) }

	// One-way: the copy follows and cannot be edited where it syncs.
	follower , err := CopyCard( store , home.ID , card.ID , TransferPlace{ BoardID: away.ID , ListID: away.Lists[ 0 ].ID } , MirrorOneWay , fixture.owner.ID , "Owner" )
	if err != nil { t.Fatal( err ) }
	if follower.Mirror != MirrorCopy { t.Errorf( "one-way copy shows %q" , follower.Mirror ) }
	changed := "Changed on the copy"
	if _ , _ , err := PatchCard( store , away.ID , follower.ID , &CardPatch{ Title: &changed } , "Owner" ); errors.Is( err , ErrMirrored ) == false { t.Errorf( "edit on a one-way copy: %v" , err ) }
	PatchCard( store , away.ID , follower.ID , &CardPatch{ ListID: &later } , "Owner" )
	if fixture.get( t , away.ID , follower.ID ).ListID != later { t.Error( "a one-way copy can still be moved" ) }
	source_title := "From the source"
	PatchCard( store , home.ID , card.ID , &CardPatch{ Title: &source_title } , "Owner" )
	if fixture.get( t , away.ID , follower.ID ).Title != source_title { t.Error( "the one-way copy did not follow" ) }
	// The chain: the twin follows the source, and the follower the source,
	// but nothing goes back up from the follower.
	if fixture.get( t , away.ID , twin.ID ).Title != source_title { t.Error( "the twin did not follow" ) }
	local , _ := AddComment( store , away.ID , follower.ID , fixture.owner.ID , "Owner" , "Only here" )
	if home_comments , _ = ListComments( store , home.ID , card.ID ); len( home_comments ) != 1 { t.Errorf( "a one-way copy's comment went up: %v" , local ) }

	links , _ := ListCardLinks( store , home.ID , card.ID )
	if len( links ) != 2 { t.Fatalf( "links: %+v" , links ) }

	// Trashing one end unlinks; the other stays. Restoring does not relink.
	TrashCard( store , away.ID , twin.ID , "Owner" )
	if links , _ = ListCardLinks( store , home.ID , card.ID ); len( links ) != 1 { t.Errorf( "after trash: %+v" , links ) }
	RestoreCard( store , away.ID , twin.ID , "Owner" )
	again := "After the restore"
	PatchCard( store , home.ID , card.ID , &CardPatch{ Title: &again } , "Owner" )
	if fixture.get( t , away.ID , twin.ID ).Title == again { t.Error( "a restored card is linked again" ) }

	// Archiving unlinks too, and then the copy is free.
	PatchCard( store , away.ID , follower.ID , &CardPatch{ Archived: &yes } , "Owner" )
	if fixture.get( t , home.ID , card.ID ).Mirror != MirrorNone { t.Error( "the source still shows a link" ) }
	if _ , _ , err := PatchCard( store , away.ID , follower.ID , &CardPatch{ Title: &changed } , "Owner" ); err != nil { t.Errorf( "an unlinked copy is editable: %v" , err ) }

	// Unlinking by hand.
	other , _ := CopyCard( store , home.ID , card.ID , TransferPlace{ BoardID: away.ID , ListID: away.Lists[ 0 ].ID } , MirrorTwoWay , fixture.owner.ID , "Owner" )
	links , _ = ListCardLinks( store , away.ID , other.ID )
	if err := UnlinkCard( store , away.ID , other.ID , links[ 0 ].ID ); err != nil { t.Fatal( err ) }
	if fixture.get( t , away.ID , other.ID ).Mirror != MirrorNone { t.Error( "unlinked card still shows a link" ) }
	if err := UnlinkCard( store , away.ID , other.ID , links[ 0 ].ID ); errors.Is( err , ErrLinkNotFound ) == false { t.Errorf( "second unlink: %v" , err ) }
}

func TestMirrorLoopStops( t *testing.T ) {
	fixture := newLinkFixture( t )
	store := fixture.store
	a := fixture.card( t , fixture.home , "Loop" )
	b , _ := CopyCard( store , fixture.home.ID , a.ID , TransferPlace{ BoardID: fixture.away.ID , ListID: fixture.away.Lists[ 0 ].ID } , MirrorTwoWay , fixture.owner.ID , "Owner" )
	c , _ := CopyCard( store , fixture.away.ID , b.ID , TransferPlace{ BoardID: fixture.home.ID , ListID: fixture.home.Lists[ 1 ].ID } , MirrorTwoWay , fixture.owner.ID , "Owner" )
	// Close the loop: c back to a.
	store.Write( func( tx *db.Tx ) ( err error ) {
		_ , err = addLinkTx( tx , endOf( c ) , endOf( a ) , MirrorTwoWay , "Owner" )
		return
	} )
	title := "Round and round"
	if _ , _ , err := PatchCard( store , fixture.home.ID , a.ID , &CardPatch{ Title: &title } , "Owner" ); err != nil { t.Fatal( err ) }
	if fixture.get( t , fixture.away.ID , b.ID ).Title != title || fixture.get( t , fixture.home.ID , c.ID ).Title != title { t.Error( "the loop did not sync" ) }
	AddComment( store , fixture.home.ID , c.ID , fixture.owner.ID , "Owner" , "Once" )
	for _ , end := range []*Card{ a , b , c } {
		if comments , _ := ListComments( store , end.BoardID , end.ID ); len( comments ) != 1 { t.Errorf( "card %d has %d comments" , end.ID , len( comments ) ) }
	}
}
