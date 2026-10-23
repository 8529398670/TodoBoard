package models

import (
	errors "errors"
	testing "testing"
)


func TestBoardCardLifecycle( t *testing.T ) {
	store := testStore( t )
	owner , _ := CreateUser( store , "Owner" , RoleAdmin )
	bob , _ := CreateUser( store , "Bob" , RoleUser )

	board , err := CreateBoard( store , owner.ID , "  Plans  " , []string{ "To do" , "Doing" , "Done" , "  " } )
	if err != nil { t.Fatal( err ) }
	if board.Title != "Plans" || len( board.Lists ) != 3 {
		t.Fatalf( "board not created as asked: %+v" , board )
	}
	todo := board.SortedLists()[ 0 ].ID
	done := board.SortedLists()[ 2 ].ID

	first , err := CreateCard( store , board.ID , todo , "First" , nil , owner.ID , "Owner" )
	if err != nil { t.Fatal( err ) }
	second , _ := CreateCard( store , board.ID , todo , "Second" , nil , owner.ID , "Owner" )
	if second.Position <= first.Position {
		t.Errorf( "a new card goes to the end: %v then %v" , first.Position , second.Position )
	}

	// Moving between lists reports the move and bumps the board revision.
	before , _ := GetBoard( store , board.ID )
	done_flag := true
	moved , change , err := PatchCard( store , board.ID , first.ID , &CardPatch{ ListID: &done , Done: &done_flag } , "Owner" )
	if err != nil { t.Fatal( err ) }
	if moved.ListID != done || change.Moved == false || change.Completed == false {
		t.Errorf( "move/complete not reported: %+v" , change )
	}
	after , _ := GetBoard( store , board.ID )
	if after.Rev <= before.Rev {
		t.Errorf( "a card change must bump the board rev (%d -> %d)" , before.Rev , after.Rev )
	}

	// A text edit based on an old revision is refused; a toggle is not.
	stale := uint64( 1 )
	title := "Renamed"
	if _ , _ , err = PatchCard( store , board.ID , first.ID , &CardPatch{ Rev: &stale , Title: &title } , "Owner" ); errors.Is( err , ErrStale ) == false {
		t.Errorf( "stale title edit: got %v, want ErrStale" , err )
	}
	not_done := false
	if _ , _ , err = PatchCard( store , board.ID , first.ID , &CardPatch{ Rev: &stale , Done: &not_done } , "Owner" ); err != nil {
		t.Errorf( "a toggle with an old rev should still apply: %v" , err )
	}

	// Assigning someone who cannot see the card is silently dropped.
	assignees := []uint64{ bob.ID , owner.ID }
	patched , change , _ := PatchCard( store , board.ID , first.ID , &CardPatch{ Assignees: &assignees } , "Owner" )
	if len( patched.Assignees ) != 1 || patched.Assignees[ 0 ] != owner.ID {
		t.Errorf( "only people on the board can be assigned, got %v" , patched.Assignees )
	}
	if _ , err = SetMember( store , board.ID , bob.ID , BoardRoleEditor ); err != nil { t.Fatal( err ) }
	patched , change , _ = PatchCard( store , board.ID , first.ID , &CardPatch{ Assignees: &assignees } , "Owner" )
	if len( patched.Assignees ) != 2 || len( change.NewAssignees ) != 1 || change.NewAssignees[ 0 ] != bob.ID {
		t.Errorf( "assigning a member: %v new=%v" , patched.Assignees , change.NewAssignees )
	}

	boards , _ , _ := ListBoardsForUser( store , bob.ID )
	if len( boards ) != 1 || boards[ 0 ].Role != BoardRoleEditor {
		t.Errorf( "bob should see the board as editor: %+v" , boards )
	}

	// Removing a member unassigns them.
	if _ , err = SetMember( store , board.ID , bob.ID , "" ); err != nil { t.Fatal( err ) }
	reloaded , _ := GetCard( store , board.ID , first.ID )
	if containsID( reloaded.Assignees , bob.ID ) {
		t.Errorf( "a removed member must be unassigned" )
	}

	// Deleting a list takes its cards.
	if err = DeleteList( store , board.ID , todo ); err != nil { t.Fatal( err ) }
	if _ , err = GetCard( store , board.ID , second.ID ); errors.Is( err , ErrCardNotFound ) == false {
		t.Errorf( "card on a deleted list should be gone, got %v" , err )
	}
}

func TestPositionsRenormalize( t *testing.T ) {
	store := testStore( t )
	owner , _ := CreateUser( store , "Owner" , RoleAdmin )
	board , _ := CreateBoard( store , owner.ID , "B" , []string{ "L" } )
	list := board.Lists[ 0 ].ID
	a , _ := CreateCard( store , board.ID , list , "a" , nil , owner.ID , "" )
	b , _ := CreateCard( store , board.ID , list , "b" , nil , owner.ID , "" )

	// Keep inserting between a and b until they would collide.
	low , high := a.Position , b.Position
	for index := 0; index < 80; index += 1 {
		middle := ( low + high ) / 2
		if _ , err := CreateCard( store , board.ID , list , "x" , &middle , owner.ID , "" ); err != nil { t.Fatal( err ) }
		high = middle
		reloaded , _ := GetCard( store , board.ID , a.ID )
		low = reloaded.Position
		cards , _ := ListCards( store , board.ID )
		for _ , card := range cards {
			if card.ID == b.ID { high = card.Position }
		}
		// high is no longer b's neighbour; re-read the smallest above low.
		for _ , card := range cards {
			if card.Position > low && card.Position < high { high = card.Position }
		}
	}
	cards , _ := ListCards( store , board.ID )
	positions := []float64{}
	for _ , card := range cards { positions = append( positions , card.Position ) }
	if needsRenormalize( positions ) {
		t.Errorf( "positions crowded together and were never respaced: %v" , positions )
	}
}

func TestShareLinks( t *testing.T ) {
	store := testStore( t )
	owner , _ := CreateUser( store , "Owner" , RoleAdmin )
	board , _ := CreateBoard( store , owner.ID , "B" , []string{ "L" } )
	card , _ := CreateCard( store , board.ID , board.Lists[ 0 ].ID , "c" , nil , owner.ID , "" )

	credential , link , err := IssueShareLink( store , board.ID , card.ID , ModeEdit , "for Sam" , 0 , owner.ID )
	if err != nil { t.Fatal( err ) }
	if loaded := LoadShareLink( store , credential ); loaded == nil || loaded.CardID != card.ID {
		t.Fatalf( "a fresh link should load" )
	}
	if LoadShareLink( store , link.ID + ".wrong" ) != nil {
		t.Errorf( "a wrong secret must not load" )
	}
	if _ , _ , err = IssueShareLink( store , board.ID , 0 , "admin" , "" , 0 , owner.ID ); err == nil {
		t.Errorf( "a link mode must be view or edit" )
	}
	if err = RevokeShareLink( store , link.ID ); err != nil { t.Fatal( err ) }
	if LoadShareLink( store , credential ) != nil {
		t.Errorf( "a revoked link must not load" )
	}

	// Deleting the card takes its links with it.
	credential , _ , _ = IssueShareLink( store , board.ID , card.ID , ModeView , "" , 7 , owner.ID )
	if err = DeleteCard( store , board.ID , card.ID ); err != nil { t.Fatal( err ) }
	if LoadShareLink( store , credential ) != nil {
		t.Errorf( "a deleted card's link must not load" )
	}
}

func TestCardShareAndDeleteUser( t *testing.T ) {
	store := testStore( t )
	owner , _ := CreateUser( store , "Owner" , RoleAdmin )
	alice , _ := CreateUser( store , "Alice" , RoleUser )
	carol , _ := CreateUser( store , "Carol" , RoleUser )

	board , _ := CreateBoard( store , alice.ID , "Alice's" , []string{ "L" } )
	card , _ := CreateCard( store , board.ID , board.Lists[ 0 ].ID , "c" , nil , alice.ID , "" )
	if _ , err := ShareCardWithUser( store , board.ID , card.ID , carol.ID , ModeView , "Alice" ); err != nil { t.Fatal( err ) }

	_ , shared , _ := ListBoardsForUser( store , carol.ID )
	if len( shared ) != 1 || shared[ 0 ].CardID != card.ID {
		t.Errorf( "carol should see the shared card: %+v" , shared )
	}

	// Deleting Alice hands her board to the admin who deleted her.
	if _ , err := DeleteUser( store , alice.ID , owner.ID ); err != nil { t.Fatal( err ) }
	reloaded , err := GetBoard( store , board.ID )
	if err != nil || reloaded.OwnerID != owner.ID {
		t.Fatalf( "board should pass to the deleting admin: %v %+v" , err , reloaded )
	}
	boards , _ , _ := ListBoardsForUser( store , owner.ID )
	if len( boards ) != 1 || boards[ 0 ].Role != BoardRoleOwner {
		t.Errorf( "the heir should list the board as owner: %+v" , boards )
	}

	// Deleting Carol removes her card grant from the card itself.
	if _ , err = DeleteUser( store , carol.ID , owner.ID ); err != nil { t.Fatal( err ) }
	reloaded_card , _ := GetCard( store , board.ID , card.ID )
	if reloaded_card.SharedWith[ carol.ID ] != "" {
		t.Errorf( "a deleted user's card grant should be gone" )
	}
}

func TestEffectiveEvents( t *testing.T ) {
	store := testStore( t )
	owner , _ := CreateUser( store , "Owner" , RoleAdmin )
	board , _ := CreateBoard( store , owner.ID , "B" , []string{ "L" } )
	card , _ := CreateCard( store , board.ID , board.Lists[ 0 ].ID , "c" , nil , owner.ID , "" )
	prefs := DefaultUserPrefs()

	events , source := EffectiveEvents( store , prefs , owner.ID , board.ID , card )
	if source != SourceWatch || events.Due {
		t.Errorf( "unassigned card should use watch defaults: %v %+v" , source , events )
	}
	card.Assignees = []uint64{ owner.ID }
	if events , source = EffectiveEvents( store , prefs , owner.ID , board.ID , card ); source != SourceAssignee || events.Due == false {
		t.Errorf( "assigned card should use assignee defaults: %v %+v" , source , events )
	}
	SetBoardSubscription( store , owner.ID , board.ID , &EventSet{ Changes: true } )
	if events , source = EffectiveEvents( store , prefs , owner.ID , board.ID , card ); source != SourceBoard || events.Due {
		t.Errorf( "a board override beats defaults: %v %+v" , source , events )
	}
	SetCardSubscription( store , owner.ID , board.ID , card.ID , &EventSet{ Due: true } )
	if events , source = EffectiveEvents( store , prefs , owner.ID , board.ID , card ); source != SourceCard || events.Due == false || events.Changes {
		t.Errorf( "a card override beats the board: %v %+v" , source , events )
	}
	SetCardSubscription( store , owner.ID , board.ID , card.ID , nil )
	if _ , source = EffectiveEvents( store , prefs , owner.ID , board.ID , card ); source != SourceBoard {
		t.Errorf( "clearing the card override falls back to the board, got %v" , source )
	}

	bad := DefaultUserPrefs()
	bad.PushoverUserKey = "short"
	if bad.Validate() == nil {
		t.Errorf( "a malformed pushover key must be refused" )
	}
	bad = DefaultUserPrefs()
	bad.Timezone = "Mars/Olympus"
	if bad.Validate() == nil {
		t.Errorf( "an unknown timezone must be refused" )
	}
}
