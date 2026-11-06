package notify

import (
	strings "strings"
	testing "testing"

	db "todoboard/server/db"
	models "todoboard/server/models"
)

// activityBoard is a board with an owner and a member, the member subscribed
// to all of its activity.
func activityBoard( t *testing.T , store *db.Store ) ( board *models.Board , owner *models.User , member *models.User ) {
	t.Helper()
	owner , _ = models.CreateUser( store , "Owner" , models.RoleAdmin )
	member , _ = models.CreateUser( store , "Member" , models.RoleUser )
	board , err := models.CreateBoard( store , owner.ID , "Plans" , []string{ "To do" , "Done" } )
	if err != nil { t.Fatal( err ) }
	if board , err = models.SetMember( store , board.ID , member.ID , models.BoardRoleEditor ); err != nil { t.Fatal( err ) }
	withKey( t , store , member.ID )
	subscribeActivity( t , store , member.ID , board.ID )
	return
}

func subscribeActivity( t *testing.T , store *db.Store , user_id uint64 , board_id uint64 ) {
	t.Helper()
	if err := models.SetBoardSubscription( store , user_id , board_id , &models.EventSet{ Assigned: true , Changes: true , Activity: true } ); err != nil { t.Fatal( err ) }
}

func TestActivityBatchesPerBoard( t *testing.T ) {
	service , mock , store := setup( t )
	board , owner , _ := activityBoard( t , store )
	for _ , title := range []string{ "One" , "Two" , "Three" } {
		service.Activity( board.ID , owner.ID , "Owner" , "card_added" , map[string]string{ "card": title , "list": "To do" } )
	}
	drain( service )
	if mock.count() != 0 { t.Fatalf( "sent before the window closed: %v" , mock.last() ) }
	service.flush( true )
	drain( service )
	if mock.count() != 1 { t.Fatalf( "%d notifications for one board" , mock.count() ) }
	sent := mock.last()
	if sent[ "title" ] != "Plans: 3 changes" { t.Errorf( "title: %q" , sent[ "title" ] ) }
	for _ , title := range []string{ "One" , "Two" , "Three" } {
		if strings.Contains( sent[ "message" ] , title ) == false { t.Errorf( "%s is missing: %q" , title , sent[ "message" ] ) }
	}
	if strings.Contains( sent[ "url" ] , "/b/" ) == false { t.Errorf( "url: %q" , sent[ "url" ] ) }
}

func TestActivityCapsLines( t *testing.T ) {
	service , mock , store := setup( t )
	board , owner , _ := activityBoard( t , store )
	for count := 0 ; count < maxActivityLines + 5 ; count += 1 {
		service.Activity( board.ID , owner.ID , "Owner" , "card_added" , map[string]string{ "card": "Card " + string( rune( 'A' + count ) ) , "list": "To do" } )
	}
	drain( service )
	service.flush( true )
	drain( service )
	sent := mock.last()
	if sent[ "title" ] != "Plans: 25 changes" { t.Errorf( "title: %q" , sent[ "title" ] ) }
	if strings.Contains( sent[ "message" ] , "and 5 more" ) == false { t.Errorf( "message: %q" , sent[ "message" ] ) }
}

func TestActivityExcludesActor( t *testing.T ) {
	service , mock , store := setup( t )
	board , _ , member := activityBoard( t , store )
	service.Activity( board.ID , member.ID , "Member" , "board_notes" , nil )
	drain( service )
	service.flush( true )
	drain( service )
	if mock.count() != 0 { t.Errorf( "told about their own change: %v" , mock.last() ) }
}

func TestActivityRechecksMembership( t *testing.T ) {
	service , mock , store := setup( t )
	board , owner , member := activityBoard( t , store )
	service.Activity( board.ID , owner.ID , "Owner" , "board_notes" , nil )
	drain( service )
	if _ , err := models.SetMember( store , board.ID , member.ID , "" ); err != nil { t.Fatal( err ) }
	service.flush( true )
	drain( service )
	if mock.count() != 0 { t.Errorf( "someone who left was told: %v" , mock.last() ) }
}

func TestActivityKeepsDeletedCardTitle( t *testing.T ) {
	service , mock , store := setup( t )
	board , owner , _ := activityBoard( t , store )
	card , err := models.CreateCard( store , board.ID , board.Lists[ 0 ].ID , "Doomed card" , nil , owner.ID , "Owner" )
	if err != nil { t.Fatal( err ) }
	if err = models.TrashCard( store , board.ID , card.ID , "Owner" ); err != nil { t.Fatal( err ) }
	service.Activity( board.ID , owner.ID , "Owner" , "card_trashed" , map[string]string{ "card": card.Title } )
	drain( service )
	service.flush( true )
	drain( service )
	if strings.Contains( mock.last()[ "message" ] , "Doomed card" ) == false { t.Errorf( "message: %v" , mock.last() ) }
}

func TestActivityReplacesChanges( t *testing.T ) {
	service , mock , store := setup( t )
	board , owner , _ := activityBoard( t , store )
	card , err := models.CreateCard( store , board.ID , board.Lists[ 0 ].ID , "Card" , nil , owner.ID , "Owner" )
	if err != nil { t.Fatal( err ) }
	title := "Renamed"
	_ , change , err := models.PatchCard( store , board.ID , card.ID , &models.CardPatch{ Title: &title } , "Owner" )
	if err != nil { t.Fatal( err ) }
	service.CardChanged( board.ID , card.ID , change , owner.ID , "Owner" )
	drain( service )
	service.flush( true )
	drain( service )
	if mock.count() != 1 { t.Fatalf( "%d notifications for one change" , mock.count() ) }
	if strings.Contains( mock.last()[ "title" ] , "Plans" ) == false { t.Errorf( "not the board's activity: %v" , mock.last() ) }
}

func TestActivitySkipsAReorder( t *testing.T ) {
	service , mock , store := setup( t )
	board , owner , _ := activityBoard( t , store )
	card , err := models.CreateCard( store , board.ID , board.Lists[ 0 ].ID , "Card" , nil , owner.ID , "Owner" )
	if err != nil { t.Fatal( err ) }
	position := 5.0
	_ , change , err := models.PatchCard( store , board.ID , card.ID , &models.CardPatch{ Position: &position } , "Owner" )
	if err != nil { t.Fatal( err ) }
	service.CardChanged( board.ID , card.ID , change , owner.ID , "Owner" )
	drain( service )
	service.flush( true )
	drain( service )
	if mock.count() != 0 { t.Errorf( "a reorder was news: %v" , mock.last() ) }
}

func TestActivityWordsLabelsAndArchive( t *testing.T ) {
	service , mock , store := setup( t )
	board , owner , _ := activityBoard( t , store )
	card , err := models.CreateCard( store , board.ID , board.Lists[ 0 ].ID , "Card" , nil , owner.ID , "Owner" )
	if err != nil { t.Fatal( err ) }
	archived := true
	_ , change , err := models.PatchCard( store , board.ID , card.ID , &models.CardPatch{ Archived: &archived } , "Owner" )
	if err != nil { t.Fatal( err ) }
	service.CardChanged( board.ID , card.ID , change , owner.ID , "Owner" )
	drain( service )
	service.flush( true )
	drain( service )
	message := mock.last()[ "message" ]
	if strings.Contains( message , "archived" ) == false || strings.Contains( message , "edited" ) { t.Errorf( "message: %q" , message ) }
}

func TestActivityOffByDefault( t *testing.T ) {
	service , mock , store := setup( t )
	board , owner , member := activityBoard( t , store )
	if err := models.SetBoardSubscription( store , member.ID , board.ID , nil ); err != nil { t.Fatal( err ) }
	service.Activity( board.ID , owner.ID , "Owner" , "board_notes" , nil )
	drain( service )
	service.flush( true )
	drain( service )
	if mock.count() != 0 { t.Errorf( "activity reached someone who never asked: %v" , mock.last() ) }
}

func TestPanicDoesNotStopWorker( t *testing.T ) {
	ran := false
	safely( "test" , func() { panic( "boom" ) } )
	safely( "test" , func() { ran = true } )
	if ran == false { t.Errorf( "work after a panic did not run" ) }
}
