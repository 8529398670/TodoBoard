package routes

import (
	io "io"
	httptest "net/http/httptest"
	sort "sort"
	strings "strings"
	testing "testing"

	access "todoboard/server/access"
	models "todoboard/server/models"
)

// A card's public link: anyone holding it reads that one card, sees nothing
// of the board around it, and loses it the moment the owner switches it off.

const quickSecretTitle = "Launch plans nobody should see"

// quickBoard is a board with quick sharing on, and a card with one of the
// board's two labels. Everything private is worded so a test can search for it.
func quickBoard( t *testing.T , fixture *mcpFixture ) ( board *models.Board , card *models.Card , credential string ) {
	t.Helper()
	board , err := models.CreateBoard( fixture.store , fixture.owner.ID , quickSecretTitle , []string{ "Private column" } )
	if err != nil { t.Fatal( err ) }
	on := true
	if board , err = models.UpdateBoardSettings( fixture.store , board.ID , &models.BoardSettings{ AllowQuickShare: &on } ); err != nil { t.Fatal( err ) }
	var shown *models.Label
	board , err = models.MutateBoard( fixture.store , board.ID , func( target *models.Board ) ( mutate_err error ) {
		if shown , mutate_err = models.AddLabel( target , "Shown label" , "red" ); mutate_err != nil { return }
		_ , mutate_err = models.AddLabel( target , "Unused label" , "blue" )
		return
	} )
	if err != nil { t.Fatal( err ) }
	list_id := board.Lists[ 0 ].ID
	card , err = models.CreateCard( fixture.store , board.ID , list_id , "Public card" , nil , fixture.owner.ID , "guest" )
	if err != nil { t.Fatal( err ) }
	if _ , err = models.CreateCard( fixture.store , board.ID , list_id , "Other private card" , nil , fixture.owner.ID , "guest" ); err != nil { t.Fatal( err ) }
	labels := []uint64{ shown.ID }
	assignees := []uint64{ fixture.member.ID }
	if card , _ , err = models.PatchCard( fixture.store , board.ID , card.ID , &models.CardPatch{ LabelIDs: &labels , Assignees: &assignees } , "Editor Name" ); err != nil { t.Fatal( err ) }
	credential , _ , err = models.QuickShareLink( fixture.store , board.ID , card.ID , fixture.owner.ID )
	if err != nil { t.Fatal( err ) }
	return
}

func guestGet( t *testing.T , fixture *mcpFixture , credential string , path string ) ( status int , body string ) {
	t.Helper()
	request := httptest.NewRequest( "GET" , path , nil )
	request.Header.Set( "X-Share-Token" , credential )
	response , err := fixture.app.Test( request )
	if err != nil { t.Fatal( err ) }
	raw , _ := io.ReadAll( response.Body )
	status , body = response.StatusCode , string( raw )
	return
}

func keysOf( values map[ string ]any ) ( keys []string ) {
	for key := range values { keys = append( keys , key ) }
	sort.Strings( keys )
	return
}

func TestQuickSharePayloadIsOnlyTheCard( t *testing.T ) {
	fixture := newMCPFixture( t )
	board , card , credential := quickBoard( t , fixture )

	status , raw := guestGet( t , fixture , credential , "/api/shared/"+itoa( board.ID ) )
	if status != 200 { t.Fatalf( "status %d: %s" , status , raw ) }
	for _ , private := range []string{ quickSecretTitle , "Private column" , "Other private card" , "Unused label" , "Owner" , "Member" , "Editor Name" , board.Slug , card.UID } {
		if private != "" && strings.Contains( raw , private ) { t.Errorf( "%q is in the public payload: %s" , private , raw ) }
	}
	if strings.Contains( raw , "Shown label" ) == false { t.Errorf( "the card's own label is missing: %s" , raw ) }

	_ , payload := sharedBoard( t , fixture , credential , board.ID , "" )
	if payload[ "quick" ] != true || payload[ "card_only" ] != true { t.Errorf( "payload: %v" , payload ) }
	whole := payload[ "board" ].( map[ string ]any )
	if got := strings.Join( keysOf( whole ) , " " ); got != "id labels layout lists members rev title" { t.Errorf( "board keys: %s" , got ) }
	cards := payload[ "cards" ].( []any )
	if len( cards ) != 1 { t.Fatalf( "%d cards" , len( cards ) ) }
	row := cards[ 0 ].( map[ string ]any )
	want := "access archived assignees checklist description doing done due_all_day due_at id image_at label_ids list_id position price product rev start_all_day start_at title"
	if got := strings.Join( keysOf( row ) , " " ); got != want { t.Errorf( "card keys: %s" , got ) }
	if row[ "access" ] != "view" { t.Errorf( "access: %v" , row[ "access" ] ) }
}

func TestQuickShareInfoAndStampHideTheBoard( t *testing.T ) {
	fixture := newMCPFixture( t )
	board , card , credential := quickBoard( t , fixture )

	_ , info := guestGet( t , fixture , credential , "/api/shared" )
	if strings.Contains( info , quickSecretTitle ) || strings.Contains( info , `"quick":true` ) == false { t.Errorf( "info: %s" , info ) }

	// Other cards changing must not move what the link polls.
	_ , before := guestGet( t , fixture , credential , "/api/shared/"+itoa( board.ID )+"/stamp" )
	if _ , err := models.CreateCard( fixture.store , board.ID , board.Lists[ 0 ].ID , "Another" , nil , fixture.owner.ID , "guest" ); err != nil { t.Fatal( err ) }
	_ , after := guestGet( t , fixture , credential , "/api/shared/"+itoa( board.ID )+"/stamp" )
	if before != after { t.Errorf( "the stamp moved with another card: %s then %s" , before , after ) }

	title := "Renamed"
	if _ , _ , err := models.PatchCard( fixture.store , board.ID , card.ID , &models.CardPatch{ Title: &title } , "guest" ); err != nil { t.Fatal( err ) }
	_ , changed := guestGet( t , fixture , credential , "/api/shared/"+itoa( board.ID )+"/stamp" )
	if changed == after { t.Errorf( "the stamp did not move with the card: %s" , changed ) }
}

func TestQuickShareSettingBlocksEveryDoor( t *testing.T ) {
	fixture := newMCPFixture( t )
	board , card , credential := quickBoard( t , fixture )
	id , card_path := itoa( board.ID ) , "/cards/"+itoa( card.ID )
	doors := []string{ "/api/shared" , "/api/shared/"+id , "/api/shared/"+id+"/stamp" , "/api/shared/"+id+card_path+"/image" , "/api/shared/"+id+card_path+"/product-image" , "/api/shared/"+id+"/export?format=csv" }

	off , on := false , true
	if _ , err := models.UpdateBoardSettings( fixture.store , board.ID , &models.BoardSettings{ AllowQuickShare: &off } ); err != nil { t.Fatal( err ) }
	for _ , door := range doors {
		if status , body := guestGet( t , fixture , credential , door ); status != 404 || strings.Contains( body , "share_link_invalid" ) == false {
			t.Errorf( "%s with sharing off: %d %s" , door , status , body )
		}
	}
	if _ , err := models.UpdateBoardSettings( fixture.store , board.ID , &models.BoardSettings{ AllowQuickShare: &on } ); err != nil { t.Fatal( err ) }
	if status , body := guestGet( t , fixture , credential , "/api/shared/"+id ); status != 200 { t.Errorf( "back on: %d %s" , status , body ) }
}

func TestQuickShareExportRefused( t *testing.T ) {
	fixture := newMCPFixture( t )
	board , _ , credential := quickBoard( t , fixture )
	if status , body := guestGet( t , fixture , credential , "/api/shared/"+itoa( board.ID )+"/export?format=csv" ); status != 404 || strings.Contains( body , quickSecretTitle ) {
		t.Errorf( "export: %d %s" , status , body )
	}
}

func TestQuickShareRefusesWrites( t *testing.T ) {
	fixture := newMCPFixture( t )
	board , _ , credential := quickBoard( t , fixture )
	if status , _ := sharedBoard( t , fixture , credential , board.ID , `{"title":"Hijacked"}` ); status == 200 { t.Errorf( "a public link renamed the board" ) }
}

func TestQuickShareSameLinkTwice( t *testing.T ) {
	fixture := newMCPFixture( t )
	board , card , first := quickBoard( t , fixture )
	second , _ , err := models.QuickShareLink( fixture.store , board.ID , card.ID , fixture.member.ID )
	if err != nil { t.Fatal( err ) }
	if first != second { t.Errorf( "a second share gave a new link" ) }
	links , _ := models.ListShareLinks( fixture.store , board.ID )
	if len( links ) != 1 { t.Errorf( "%d links" , len( links ) ) }
	if strings.Contains( links[ 0 ].SecretHash , strings.SplitN( first , "." , 2 )[ 1 ] ) { t.Errorf( "the secret is stored" ) }
}

func TestQuickShareAfterRevokeIsNew( t *testing.T ) {
	fixture := newMCPFixture( t )
	board , card , first := quickBoard( t , fixture )
	link := models.LoadShareLink( fixture.store , first )
	if err := models.RevokeShareLink( fixture.store , link.ID ); err != nil { t.Fatal( err ) }
	if status , _ := guestGet( t , fixture , first , "/api/shared" ); status != 404 { t.Errorf( "a revoked link answered %d" , status ) }
	second , _ , err := models.QuickShareLink( fixture.store , board.ID , card.ID , fixture.owner.ID )
	if err != nil { t.Fatal( err ) }
	if first == second { t.Errorf( "the revoked link came back" ) }
	if status , _ := guestGet( t , fixture , second , "/api/shared" ); status != 200 { t.Errorf( "the new link answered %d" , status ) }
}

func TestQuickShareDiesWithTrash( t *testing.T ) {
	fixture := newMCPFixture( t )
	board , card , credential := quickBoard( t , fixture )
	if err := models.DeleteCard( fixture.store , board.ID , card.ID ); err != nil { t.Fatal( err ) }
	if status , _ := guestGet( t , fixture , credential , "/api/shared" ); status != 404 { t.Errorf( "a trashed card's link answered %d" , status ) }
}

func TestQuickShareRefusesOffAndArchived( t *testing.T ) {
	fixture := newMCPFixture( t )
	board , card , _ := quickBoard( t , fixture )
	archived := true
	if _ , _ , err := models.PatchCard( fixture.store , board.ID , card.ID , &models.CardPatch{ Archived: &archived } , "guest" ); err != nil { t.Fatal( err ) }
	second , err := models.CreateCard( fixture.store , board.ID , board.Lists[ 0 ].ID , "Second" , nil , fixture.owner.ID , "guest" )
	if err != nil { t.Fatal( err ) }
	if _ , _ , err = models.QuickShareLink( fixture.store , board.ID , card.ID , fixture.owner.ID ); err != models.ErrCardArchived { t.Errorf( "archived: %v" , err ) }
	off := false
	if _ , err = models.UpdateBoardSettings( fixture.store , board.ID , &models.BoardSettings{ AllowQuickShare: &off } ); err != nil { t.Fatal( err ) }
	if _ , _ , err = models.QuickShareLink( fixture.store , board.ID , second.ID , fixture.owner.ID ); err != models.ErrQuickShareOff { t.Errorf( "off: %v" , err ) }
}

func TestOwnerLinksIgnoreQuickSetting( t *testing.T ) {
	fixture := newMCPFixture( t )
	board , card , _ := quickBoard( t , fixture )
	credential , _ , err := models.IssueShareLink( fixture.store , board.ID , card.ID , 0 , nil , models.ModeView , "" , 0 , fixture.owner.ID )
	if err != nil { t.Fatal( err ) }
	off := false
	if _ , err = models.UpdateBoardSettings( fixture.store , board.ID , &models.BoardSettings{ AllowQuickShare: &off } ); err != nil { t.Fatal( err ) }
	if status , _ := guestGet( t , fixture , credential , "/api/shared/"+itoa( board.ID ) ); status != 200 { t.Errorf( "an owner's link answered %d" , status ) }
}

func TestOnlyManageChangesQuickShare( t *testing.T ) {
	fixture := newMCPFixture( t )
	board , card , _ := quickBoard( t , fixture )
	edit_link , _ , err := models.IssueShareLink( fixture.store , board.ID , 0 , 0 , nil , models.ModeEdit , "" , 0 , fixture.owner.ID )
	if err != nil { t.Fatal( err ) }
	if status , _ := sharedBoard( t , fixture , edit_link , board.ID , `{"allow_quick_share":false}` ); status != 403 { t.Errorf( "an edit link: %d" , status ) }
	// A key, even the owner's, cannot switch it either.
	request := httptest.NewRequest( "POST" , "/api/boards/"+itoa( board.ID ) , strings.NewReader( `{"allow_quick_share":false}` ) )
	request.Header.Set( "Content-Type" , "application/json" )
	request.Header.Set( "Authorization" , "Bearer "+fixture.user_key )
	response , err := fixture.app.Test( request )
	if err != nil { t.Fatal( err ) }
	if response.StatusCode != 403 { t.Errorf( "an API key: %d" , response.StatusCode ) }
	// And a key cannot mint a link, whatever the account can do.
	request = httptest.NewRequest( "POST" , "/api/boards/"+itoa( board.ID )+"/cards/"+itoa( card.ID )+"/quick-share" , strings.NewReader( `{}` ) )
	request.Header.Set( "Content-Type" , "application/json" )
	request.Header.Set( "Authorization" , "Bearer "+fixture.user_key )
	if response , err = fixture.app.Test( request ); err != nil { t.Fatal( err ) }
	if response.StatusCode == 200 { t.Errorf( "an API key minted a public link" ) }
}

func TestQuickShareNotOnGuestRoutes( t *testing.T ) {
	fixture := newMCPFixture( t )
	board , card , credential := quickBoard( t , fixture )
	request := httptest.NewRequest( "POST" , "/api/shared/"+itoa( board.ID )+"/cards/"+itoa( card.ID )+"/quick-share" , strings.NewReader( `{}` ) )
	request.Header.Set( "Content-Type" , "application/json" )
	request.Header.Set( "X-Share-Token" , credential )
	response , err := fixture.app.Test( request )
	if err != nil { t.Fatal( err ) }
	if response.StatusCode == 200 { t.Errorf( "a guest minted a public link" ) }
}

func TestShareLinkListShowsQuick( t *testing.T ) {
	fixture := newMCPFixture( t )
	board , _ , _ := quickBoard( t , fixture )
	links , err := models.ListShareLinks( fixture.store , board.ID )
	if err != nil { t.Fatal( err ) }
	if len( links ) != 1 || shareLinkPayload( links[ 0 ] )[ "quick" ] != true { t.Errorf( "links: %v" , links ) }
}

// Who may publish a card is asked of the board: a person with this card alone
// is below the bar, and so is a viewer.
func TestQuickShareWhoCanMint( t *testing.T ) {
	fixture := newMCPFixture( t )
	board , card , _ := quickBoard( t , fixture )
	principal := &access.Principal{ UserID: fixture.member.ID }
	if access.Board( principal , board ) >= access.Edit { t.Fatalf( "the member starts with no rights" ) }
	if _ , err := models.ShareCardWithUser( fixture.store , board.ID , card.ID , fixture.member.ID , models.ModeEdit , "Owner" ); err != nil { t.Fatal( err ) }
	if board , _ = models.GetBoard( fixture.store , board.ID ); access.Board( principal , board ) >= access.Edit { t.Errorf( "one shared card reaches board edit" ) }
	if _ , err := models.SetMember( fixture.store , board.ID , fixture.member.ID , models.BoardRoleViewer ); err != nil { t.Fatal( err ) }
	if board , _ = models.GetBoard( fixture.store , board.ID ); access.Board( principal , board ) >= access.Edit { t.Errorf( "a viewer reaches board edit" ) }
	if _ , err := models.SetMember( fixture.store , board.ID , fixture.member.ID , models.BoardRoleEditor ); err != nil { t.Fatal( err ) }
	if board , _ = models.GetBoard( fixture.store , board.ID ); access.Board( principal , board ) != access.Edit { t.Errorf( "an editor is below board edit" ) }
}
