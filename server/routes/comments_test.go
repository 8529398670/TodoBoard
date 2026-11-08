package routes

import (
	json "encoding/json"
	io "io"
	httptest "net/http/httptest"
	strings "strings"
	testing "testing"

	models "todoboard/server/models"
)

// commentRequest sends one request as a guest (credential set) or with an
// API key, and decodes the JSON reply.
func commentCall( t *testing.T , fixture *mcpFixture , method string , path string , body string , credential string , key string , name string ) ( status int , reply map[ string ]any ) {
	t.Helper()
	request := httptest.NewRequest( method , path , strings.NewReader( body ) )
	request.Header.Set( "Content-Type" , "application/json" )
	if credential != "" { request.Header.Set( "X-Share-Token" , credential ) }
	if key != "" { request.Header.Set( "Authorization" , "Bearer "+key ) }
	if name != "" { request.Header.Set( "X-Guest-Name" , name ) }
	response , err := fixture.app.Test( request )
	if err != nil { t.Fatal( err ) }
	raw , _ := io.ReadAll( response.Body )
	status = response.StatusCode
	json.Unmarshal( raw , &reply )
	return
}

// commentBoard is the owner's board with one card, the member as a viewer.
func commentBoard( t *testing.T , fixture *mcpFixture ) ( board *models.Board , card *models.Card ) {
	t.Helper()
	board , err := models.CreateBoard( fixture.store , fixture.owner.ID , "Talk" , []string{ "To do" } )
	if err != nil { t.Fatal( err ) }
	if _ , err = models.SetMember( fixture.store , board.ID , fixture.member.ID , models.BoardRoleViewer ); err != nil { t.Fatal( err ) }
	card , err = models.CreateCard( fixture.store , board.ID , board.Lists[ 0 ].ID , "Discuss" , nil , fixture.owner.ID , "Owner" )
	if err != nil { t.Fatal( err ) }
	return
}

func TestCommentLinks( t *testing.T ) {
	fixture := newMCPFixture( t )
	board , card := commentBoard( t , fixture )
	base := "/api/shared/" + itoa( board.ID ) + "/cards/" + itoa( card.ID ) + "/comments"
	comment_link , _ , err := models.IssueShareLink( fixture.store , board.ID , 0 , 0 , nil , models.ModeComment , "" , 0 , fixture.owner.ID )
	if err != nil { t.Fatal( err ) }
	view_link , _ , _ := models.IssueShareLink( fixture.store , board.ID , 0 , 0 , nil , models.ModeView , "" , 0 , fixture.owner.ID )

	// A comment link reads the board like a view link, and can comment.
	_ , payload := sharedBoard( t , fixture , comment_link , board.ID , "" )
	if payload[ "access" ] != "view" { t.Errorf( "a comment link reads like a view link: %v" , payload[ "access" ] ) }
	row := payload[ "cards" ].( []any )[ 0 ].( map[ string ]any )
	if row[ "can_comment" ] != true { t.Errorf( "card should say it takes comments: %v" , row ) }
	if status , _ := commentCall( t , fixture , "POST" , "/api/shared/"+itoa( board.ID )+"/cards/"+itoa( card.ID ) , `{"title":"x"}` , comment_link , "" , "Sam" ); status != 403 {
		t.Errorf( "a comment link must not edit the card: %d" , status )
	}

	// Named, it posts; with no name it is refused until the board allows it.
	status , reply := commentCall( t , fixture , "POST" , base , `{"text":"Looks good"}` , comment_link , "" , "Sam" )
	if status != 200 || reply[ "by" ] != "guest:Sam" || reply[ "can_edit" ] != false { t.Fatalf( "named guest comment: %d %v" , status , reply ) }
	if status , reply = commentCall( t , fixture , "POST" , base , `{"text":"psst"}` , comment_link , "" , "" ); status != 403 || reply[ "error" ] != "comment_name_required" {
		t.Errorf( "anonymous while off: %d %v" , status , reply )
	}
	on := true
	if _ , err = models.UpdateBoardSettings( fixture.store , board.ID , &models.BoardSettings{ AllowAnonComments: &on } ); err != nil { t.Fatal( err ) }
	if status , reply = commentCall( t , fixture , "POST" , base , `{"text":"psst"}` , comment_link , "" , "" ); status != 200 || reply[ "by" ] != "guest:" {
		t.Errorf( "anonymous while on: %d %v" , status , reply )
	}

	// A view link reads the thread and posts nothing.
	status , reply = commentCall( t , fixture , "GET" , base , "" , view_link , "" , "" )
	if status != 200 || len( reply[ "comments" ].( []any ) ) != 2 || reply[ "can_comment" ] != false { t.Errorf( "view link reading: %d %v" , status , reply ) }
	if status , _ = commentCall( t , fixture , "POST" , base , `{"text":"hi"}` , view_link , "" , "Viv" ); status != 403 { t.Errorf( "a view link posted: %d" , status ) }

	// Empty and over-long comments are refused.
	if status , _ = commentCall( t , fixture , "POST" , base , `{"text":"   "}` , comment_link , "" , "Sam" ); status != 400 { t.Errorf( "empty comment: %d" , status ) }
	if status , _ = commentCall( t , fixture , "POST" , base , `{"text":"`+strings.Repeat( "a" , models.MaxCommentLength+1 )+`"}` , comment_link , "" , "Sam" ); status != 400 { t.Errorf( "long comment: %d" , status ) }
}

func TestCommentEditAndRemove( t *testing.T ) {
	fixture := newMCPFixture( t )
	board , card := commentBoard( t , fixture )
	base := "/api/boards/" + itoa( board.ID ) + "/cards/" + itoa( card.ID ) + "/comments"
	member_key := fixture.memberKey( t )

	// A board viewer reads but does not comment, until made an editor.
	if status , _ := commentCall( t , fixture , "POST" , base , `{"text":"hi"}` , "" , member_key , "" ); status != 403 { t.Errorf( "a viewer commented: %d" , status ) }
	models.SetMember( fixture.store , board.ID , fixture.member.ID , models.BoardRoleEditor )
	status , mine := commentCall( t , fixture , "POST" , base , `{"text":"first"}` , "" , member_key , "" )
	if status != 200 || mine[ "by" ] != "Member" || mine[ "can_edit" ] != true { t.Fatalf( "member comment: %d %v" , status , mine ) }
	_ , owners := commentCall( t , fixture , "POST" , base , `{"text":"owner here"}` , "" , fixture.admin_key , "" )
	mine_path := base + "/" + itoa( uint64( mine[ "id" ].( float64 ) ) )
	owners_path := base + "/" + itoa( uint64( owners[ "id" ].( float64 ) ) )

	// Only the author edits; the author or the owner removes.
	if status , reply := commentCall( t , fixture , "POST" , mine_path , `{"text":"first, edited"}` , "" , member_key , "" ); status != 200 || reply[ "text" ] != "first, edited" || reply[ "edited_at" ] == nil {
		t.Errorf( "author edit: %d %v" , status , reply )
	}
	if status , _ := commentCall( t , fixture , "POST" , owners_path , `{"text":"hijack"}` , "" , member_key , "" ); status != 403 { t.Errorf( "edited someone else's: %d" , status ) }
	if status , _ := commentCall( t , fixture , "POST" , owners_path+"/remove" , "" , "" , member_key , "" ); status != 403 { t.Errorf( "removed the owner's: %d" , status ) }
	if status , _ := commentCall( t , fixture , "POST" , mine_path+"/remove" , "" , "" , fixture.admin_key , "" ); status != 200 { t.Errorf( "owner removing a member's: %d" , status ) }

	// The board shows the count; a public card link shows no comments.
	_ , payload := commentCall( t , fixture , "GET" , "/api/boards/"+itoa( board.ID ) , "" , "" , member_key , "" )
	if row := payload[ "cards" ].( []any )[ 0 ].( map[ string ]any ); row[ "comment_count" ] != float64( 1 ) { t.Errorf( "comment count: %v" , row[ "comment_count" ] ) }
	on := true
	models.UpdateBoardSettings( fixture.store , board.ID , &models.BoardSettings{ AllowQuickShare: &on } )
	quick , _ , _ := models.QuickShareLink( fixture.store , board.ID , card.ID , fixture.owner.ID )
	if status , _ := commentCall( t , fixture , "GET" , "/api/shared/"+itoa( board.ID )+"/cards/"+itoa( card.ID )+"/comments" , "" , quick , "" , "" ); status != 403 {
		t.Errorf( "a public link read comments: %d" , status )
	}
	if status , _ := commentCall( t , fixture , "POST" , "/api/boards/"+itoa( board.ID ) , `{"allow_anon_comments":true}` , "" , member_key , "" ); status != 403 {
		t.Errorf( "an editor switched anonymous comments: %d" , status )
	}

	// Trash keeps the thread; purging the card removes it.
	if err := models.TrashCard( fixture.store , board.ID , card.ID , "Owner" ); err != nil { t.Fatal( err ) }
	if comments , _ := models.ListComments( fixture.store , board.ID , card.ID ); len( comments ) != 1 { t.Errorf( "trash lost the comments: %d" , len( comments ) ) }
	if err := models.PurgeCard( fixture.store , board.ID , card.ID ); err != nil { t.Fatal( err ) }
	if comments , _ := models.ListComments( fixture.store , board.ID , card.ID ); len( comments ) != 0 { t.Errorf( "purge left %d comments" , len( comments ) ) }
}

func TestCommentTools( t *testing.T ) {
	fixture := newMCPFixture( t )
	board , card := commentBoard( t , fixture )
	args := `{"board":` + itoa( board.ID ) + `,"card_id":` + itoa( card.ID )
	_ , reply := fixture.post( t , fixture.admin_key , callTool( "add_comment" , args+`,"text":"From an agent"}` ) )
	if posted := toolResult( t , reply ); posted[ "by" ] != "Owner" { t.Errorf( "posted: %v" , posted ) }
	_ , reply = fixture.post( t , fixture.memberKey( t ) , callTool( "list_comments" , args+`}` ) )
	if listed := toolResult( t , reply ); len( listed[ "comments" ].( []any ) ) != 1 { t.Errorf( "listed: %v" , listed ) }
	_ , reply = fixture.post( t , fixture.memberKey( t ) , callTool( "add_comment" , args+`,"text":"viewer"}` ) )
	if failed , _ := toolFailed( t , reply ); failed == false { t.Error( "a viewer's key commented" ) }
}
