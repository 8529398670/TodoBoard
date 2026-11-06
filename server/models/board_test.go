package models

import (
	errors "errors"
	strings "strings"
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
	if err = DeleteList( store , board.ID , todo , "Owner" ); err != nil { t.Fatal( err ) }
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

	credential , link , err := IssueShareLink( store , board.ID , card.ID , 0 , nil , ModeEdit , "for Sam" , 0 , owner.ID )
	if err != nil { t.Fatal( err ) }
	if loaded := LoadShareLink( store , credential ); loaded == nil || loaded.CardID != card.ID {
		t.Fatalf( "a fresh link should load" )
	}
	if LoadShareLink( store , link.ID + ".wrong" ) != nil {
		t.Errorf( "a wrong secret must not load" )
	}
	if _ , _ , err = IssueShareLink( store , board.ID , 0 , 0 , nil , "admin" , "" , 0 , owner.ID ); err == nil {
		t.Errorf( "a link mode must be view or edit" )
	}
	if err = RevokeShareLink( store , link.ID ); err != nil { t.Fatal( err ) }
	if LoadShareLink( store , credential ) != nil {
		t.Errorf( "a revoked link must not load" )
	}

	// A column link names a column that exists, and never a card as well.
	if _ , _ , err = IssueShareLink( store , board.ID , card.ID , board.Lists[ 0 ].ID , nil , ModeView , "" , 0 , owner.ID ); err == nil {
		t.Errorf( "a link cannot be for a card and a column at once" )
	}
	if _ , _ , err = IssueShareLink( store , board.ID , 0 , board.Lists[ 0 ].ID , []uint64{ board.Lists[ 0 ].ID } , ModeView , "" , 0 , owner.ID ); err == nil {
		t.Errorf( "only a board link picks columns" )
	}
	if _ , _ , err = IssueShareLink( store , board.ID , 0 , 0 , []uint64{ 9999 } , ModeView , "" , 0 , owner.ID ); errors.Is( err , ErrListNotFound ) == false {
		t.Errorf( "an unknown visible column: %v" , err )
	}
	if _ , _ , err = IssueShareLink( store , board.ID , 0 , 9999 , nil , ModeView , "" , 0 , owner.ID ); errors.Is( err , ErrListNotFound ) == false {
		t.Errorf( "a column link needs a real column, got %v" , err )
	}

	// Deleting the card takes its links with it.
	credential , _ , _ = IssueShareLink( store , board.ID , card.ID , 0 , nil , ModeView , "" , 7 , owner.ID )
	if err = DeleteCard( store , board.ID , card.ID ); err != nil { t.Fatal( err ) }
	if LoadShareLink( store , credential ) != nil {
		t.Errorf( "a deleted card's link must not load" )
	}
}

func TestNewBoardDefaults( t *testing.T ) {
	store := testStore( t )
	owner , _ := CreateUser( store , "Owner" , RoleAdmin )

	// New boards are three columns, with three lists even if fewer were named.
	board , err := CreateBoard( store , owner.ID , "B" , []string{ "Backlog" } )
	if err != nil { t.Fatal( err ) }
	if board.Layout != LayoutThree || len( board.Lists ) != 3 || board.SortedLists()[ 0 ].Title != "Backlog" {
		t.Errorf( "a new board should be three columns: %q %d lists" , board.Layout , len( board.Lists ) )
	}

	on := NumberingOn
	if _ , err = UpdateBoardSettings( store , board.ID , &BoardSettings{ Numbering: &on } ); err != nil { t.Fatal( err ) }
	if board.ShowRank { t.Error( "a new board should not show ranks outside All" ) }
	show_rank := true
	updated , err := UpdateBoardSettings( store , board.ID , &BoardSettings{ ShowRank: &show_rank } )
	if err != nil { t.Fatal( err ) }
	if updated.ShowRank == false { t.Error( "show_rank was not saved" ) }
	updated , err = UpdateBoardSettings( store , board.ID , &BoardSettings{ Numbering: &on } )
	if err != nil { t.Fatal( err ) }
	if updated.ShowRank == false { t.Error( "another setting's save cleared show_rank" ) }
	bad := "sometimes"
	if _ , err = UpdateBoardSettings( store , board.ID , &BoardSettings{ Numbering: &bad } ); errors.Is( err , ErrBadInput ) == false {
		t.Errorf( "numbering must be on, off or empty, got %v" , err )
	}

	// Deleting a column takes its column links.
	list := board.SortedLists()[ 1 ].ID
	credential , _ , err := IssueShareLink( store , board.ID , 0 , list , nil , ModeEdit , "" , 0 , owner.ID )
	if err != nil { t.Fatal( err ) }
	if loaded := LoadShareLink( store , credential ); loaded == nil || loaded.ListID != list {
		t.Fatalf( "a fresh column link should load" )
	}
	if err = DeleteList( store , board.ID , list , "Owner" ); err != nil { t.Fatal( err ) }
	if LoadShareLink( store , credential ) != nil {
		t.Errorf( "a deleted column's link must not load" )
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

func TestFolders( t *testing.T ) {
	store := testStore( t )
	owner , _ := CreateUser( store , "Owner" , RoleAdmin )
	bob , _ := CreateUser( store , "Bob" , RoleUser )
	board , _ := CreateBoard( store , owner.ID , "B" , nil )
	if _ , err := SetMember( store , board.ID , bob.ID , BoardRoleViewer ); err != nil { t.Fatal( err ) }

	work , err := CreateFolder( store , owner.ID , " Work " )
	if err != nil || work.Name != "Work" { t.Fatalf( "create folder: %v %+v" , err , work ) }
	if _ , err = CreateFolder( store , owner.ID , "   " ); errors.Is( err , ErrBadInput ) == false {
		t.Errorf( "a blank folder name must be refused, got %v" , err )
	}
	if err = SetBoardFolder( store , owner.ID , board.ID , work.ID ); err != nil { t.Fatal( err ) }

	// Folders are private: Bob cannot file into Owner's folder, and Owner's
	// filing does not move the board on Bob's home page.
	if err = SetBoardFolder( store , bob.ID , board.ID , work.ID ); errors.Is( err , ErrFolderNotFound ) == false {
		t.Errorf( "someone else's folder must not be usable, got %v" , err )
	}
	boards , _ , _ := ListBoardsForUser( store , bob.ID )
	if len( boards ) != 1 || boards[ 0 ].Folder != 0 {
		t.Errorf( "bob's board should be unfiled: %+v" , boards )
	}

	// A change of role keeps the filing.
	bob_folder , _ := CreateFolder( store , bob.ID , "Shared" )
	SetBoardFolder( store , bob.ID , board.ID , bob_folder.ID )
	SetMember( store , board.ID , bob.ID , BoardRoleEditor )
	boards , _ , _ = ListBoardsForUser( store , bob.ID )
	if boards[ 0 ].Folder != bob_folder.ID || boards[ 0 ].Role != BoardRoleEditor {
		t.Errorf( "a role change should keep the folder: %+v" , boards[ 0 ] )
	}

	// Deleting a folder unfiles its boards.
	if err = DeleteFolder( store , owner.ID , work.ID ); err != nil { t.Fatal( err ) }
	boards , _ , _ = ListBoardsForUser( store , owner.ID )
	if boards[ 0 ].Folder != 0 {
		t.Errorf( "a deleted folder's boards should be unfiled: %+v" , boards[ 0 ] )
	}
	if folders , _ := ListFolders( store , owner.ID ); len( folders ) != 0 {
		t.Errorf( "the folder should be gone: %+v" , folders )
	}
}

func TestArchiveListCards( t *testing.T ) {
	store := testStore( t )
	owner , _ := CreateUser( store , "Owner" , RoleAdmin )
	board , _ := CreateBoard( store , owner.ID , "B" , nil )
	lists := board.SortedLists()
	a , _ := CreateCard( store , board.ID , lists[ 2 ].ID , "a" , nil , owner.ID , "" )
	b , _ := CreateCard( store , board.ID , lists[ 2 ].ID , "b" , nil , owner.ID , "" )
	other , _ := CreateCard( store , board.ID , lists[ 0 ].ID , "other" , nil , owner.ID , "" )
	archived := true
	PatchCard( store , board.ID , b.ID , &CardPatch{ Archived: &archived } , "Owner" )

	// Picking cards archives just those, and never a card on another list.
	c , _ := CreateCard( store , board.ID , lists[ 2 ].ID , "c" , nil , owner.ID , "" )
	ids , err := ArchiveListCards( store , board.ID , lists[ 2 ].ID , []uint64{ c.ID , other.ID } , "Owner" )
	if err != nil { t.Fatal( err ) }
	if len( ids ) != 1 || ids[ 0 ] != c.ID {
		t.Errorf( "only the picked card on this column is archived, got %v" , ids )
	}

	ids , err = ArchiveListCards( store , board.ID , lists[ 2 ].ID , nil , "Owner" )
	if err != nil { t.Fatal( err ) }
	if len( ids ) != 1 || ids[ 0 ] != a.ID {
		t.Errorf( "only the column's open cards are archived, got %v" , ids )
	}
	if reloaded , _ := GetCard( store , board.ID , a.ID ); reloaded.Archived == false {
		t.Errorf( "the card should be archived" )
	}
	if reloaded , _ := GetCard( store , board.ID , other.ID ); reloaded.Archived {
		t.Errorf( "a card in another column must be left alone" )
	}
	if _ , err = ArchiveListCards( store , board.ID , 9999 , nil , "Owner" ); errors.Is( err , ErrListNotFound ) == false {
		t.Errorf( "an unknown column must error, got %v" , err )
	}
}

func TestBoardPlainIcons( t *testing.T ) {
	store := testStore( t )
	owner , _ := CreateUser( store , "Owner" , RoleAdmin )
	bob , _ := CreateUser( store , "Bob" , RoleUser )
	board , _ := CreateBoard( store , owner.ID , "B" , nil )
	SetMember( store , board.ID , bob.ID , BoardRoleEditor )

	grey := true
	if err := SetBoardPlainIcons( store , bob.ID , board.ID , &grey ); err != nil { t.Fatal( err ) }
	if ref := GetBoardRef( store , bob.ID , board.ID ); ref == nil || ref.PlainIcons == nil || *ref.PlainIcons == false || ref.Role != BoardRoleEditor {
		t.Errorf( "bob's choice should be saved with his role intact: %+v" , ref )
	}
	if ref := GetBoardRef( store , owner.ID , board.ID ); ref == nil || ref.PlainIcons != nil {
		t.Errorf( "bob's choice is his alone: %+v" , ref )
	}
	// nil goes back to following the account default.
	SetBoardPlainIcons( store , bob.ID , board.ID , nil )
	if ref := GetBoardRef( store , bob.ID , board.ID ); ref.PlainIcons != nil {
		t.Errorf( "clearing the board choice should leave it unset: %+v" , ref )
	}
	stranger , _ := CreateUser( store , "Stranger" , RoleUser )
	if err := SetBoardPlainIcons( store , stranger.ID , board.ID , &grey ); errors.Is( err , ErrBoardNotFound ) == false {
		t.Errorf( "someone not on the board has nothing to set, got %v" , err )
	}
}

func TestListWidth( t *testing.T ) {
	board := &Board{ Lists: []*List{ { ID: 1 , Title: "L" } } }
	set := func( width int ) error { return ( &ListUpdate{ Width: &width } ).Apply( board , 1 ) }

	if err := set( 340 ); err != nil || board.Lists[ 0 ].Width != 340 {
		t.Errorf( "a width is kept as given: %v %d" , err , board.Lists[ 0 ].Width )
	}
	set( 50 )
	if board.Lists[ 0 ].Width != ListWidthMin { t.Errorf( "too narrow is raised to the minimum: %d" , board.Lists[ 0 ].Width ) }
	set( 5000 )
	if board.Lists[ 0 ].Width != ListWidthMax { t.Errorf( "too wide is lowered to the maximum: %d" , board.Lists[ 0 ].Width ) }
	if err := set( -1 ); !errors.Is( err , ErrBadInput ) { t.Errorf( "a negative width is refused: %v" , err ) }
	set( 0 )
	if board.Lists[ 0 ].Width != 0 { t.Errorf( "0 goes back to the standard width: %d" , board.Lists[ 0 ].Width ) }
}

func TestBoardText( t *testing.T ) {
	store := testStore( t )
	owner , _ := CreateUser( store , "Owner" , RoleUser )
	board , err := CreateBoard( store , owner.ID , "Text" , []string{ "A" , "B" , "C" } )
	if err != nil { t.Fatal( err ) }
	write := func( settings *BoardSettings ) ( updated *Board , write_err error ) {
		updated , write_err = UpdateBoardSettings( store , board.ID , settings )
		return
	}
	text := func( value string ) *string { return &value }
	rev := func( value uint64 ) *uint64 { return &value }

	// Off by default, and writing to it is refused until it is on.
	if board.PurposeEnabled { t.Errorf( "a new board has the Purpose section on" ) }
	if _ , err = write( &BoardSettings{ Purpose: text( "one" ) } ); errors.Is( err , ErrPurposeOff ) == false {
		t.Errorf( "purpose while off: %v" , err )
	}
	on , off := true , false

	updated , err := write( &BoardSettings{ PurposeEnabled: &on , Purpose: text( "one" ) , PurposeRev: rev( 0 ) } )
	if err != nil || updated.Purpose != "one" || updated.PurposeRev != 1 || updated.NotesRev != 0 { t.Fatalf( "first write: %+v , %v" , updated , err ) }

	// The same text again is not an edit: nobody else's draft goes stale.
	updated , err = write( &BoardSettings{ Purpose: text( "one" ) , PurposeRev: rev( 1 ) } )
	if err != nil || updated.PurposeRev != 1 { t.Errorf( "same text: %+v , %v" , updated , err ) }

	if _ , err = write( &BoardSettings{ Purpose: text( "two" ) , PurposeRev: rev( 0 ) } ); errors.Is( err , ErrBoardTextStale ) == false {
		t.Errorf( "stale rev: %v" , err )
	}
	// One stale half refuses the whole change.
	if _ , err = write( &BoardSettings{ Notes: text( "n" ) , NotesRev: rev( 0 ) , Purpose: text( "two" ) , PurposeRev: rev( 9 ) } ); errors.Is( err , ErrBoardTextStale ) == false {
		t.Errorf( "half stale: %v" , err )
	}
	stored , _ := GetBoard( store , board.ID )
	if stored.Purpose != "one" || stored.Notes != "" { t.Errorf( "after refusals: %+v" , stored ) }

	if _ , err = write( &BoardSettings{ Notes: text( strings.Repeat( "x" , maxBoardTextBytes + 1 ) ) } ); errors.Is( err , ErrBadInput ) == false {
		t.Errorf( "too long: %v" , err )
	}
	if _ , err = write( &BoardSettings{ Notes: text( "\xff" ) } ); errors.Is( err , ErrBadInput ) == false {
		t.Errorf( "not utf-8: %v" , err )
	}

	// A card change moves the board's rev but not the text's.
	list := stored.SortedLists()[ 0 ].ID
	if _ , err = CreateCard( store , board.ID , list , "Card" , nil , owner.ID , "Owner" ); err != nil { t.Fatal( err ) }
	after , _ := GetBoard( store , board.ID )
	if after.Rev == stored.Rev || after.PurposeRev != stored.PurposeRev { t.Errorf( "revs after a card: %d %d" , after.Rev , after.PurposeRev ) }

	// A backup carries both across; one too long to keep is dropped alone.
	snap := SnapshotBoard( after , nil , func( *Card ) bool { return true } , 0 , func( uint64 ) string { return "" } )
	snap.Notes = strings.Repeat( "x" , maxBoardTextBytes + 1 )
	imported , _ , err := ImportBoardSkeleton( store , owner.ID , snap )
	if err != nil { t.Fatal( err ) }
	if imported.Purpose != "one" || imported.Notes != "" || imported.PurposeEnabled == false { t.Errorf( "imported: %+v" , imported ) }

	// Switching it off keeps the text for switching back on.
	if _ , err = write( &BoardSettings{ PurposeEnabled: &off } ); err != nil { t.Fatal( err ) }
	updated , err = write( &BoardSettings{ PurposeEnabled: &on } )
	if err != nil || updated.Purpose != "one" { t.Errorf( "back on: %+v , %v" , updated , err ) }
}
