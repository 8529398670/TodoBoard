package routes

import (
	testing "testing"

	models "todoboard/server/models"
)

func TestCardTransferAccess( t *testing.T ) {
	fixture := newMCPFixture( t )
	key := fixture.memberKey( t )
	store := fixture.store
	home , _ := models.CreateBoard( store , fixture.owner.ID , "Home" , []string{ "To do" } )
	away , _ := models.CreateBoard( store , fixture.owner.ID , "Away" , []string{ "Inbox" } )
	hidden , _ := models.CreateBoard( store , fixture.owner.ID , "Hidden" , []string{ "Inbox" } )
	models.SetMember( store , home.ID , fixture.member.ID , models.BoardRoleEditor )
	models.SetMember( store , away.ID , fixture.member.ID , models.BoardRoleViewer )
	card , _ := models.CreateCard( store , home.ID , home.Lists[ 0 ].ID , "Errand" , nil , fixture.owner.ID , "Owner" )
	base := "/api/boards/" + itoa( home.ID ) + "/cards/" + itoa( card.ID )
	to := func( board *models.Board , extra string ) string {
		return `{"board":"` + itoa( board.ID ) + `","list_id":` + itoa( board.Lists[ 0 ].ID ) + extra + `}`
	}

	if status , _ := commentCall( t , fixture , "POST" , base+"/send" , to( away , "" ) , "" , key , "" ); status != 403 { t.Errorf( "send to a board you only view: %d" , status ) }
	if status , _ := commentCall( t , fixture , "POST" , base+"/send" , to( hidden , "" ) , "" , key , "" ); status != 404 { t.Errorf( "send to a board you are not on: %d" , status ) }
	if status , _ := commentCall( t , fixture , "POST" , base+"/send" , to( home , "" ) , "" , key , "" ); status != 403 { t.Errorf( "send to its own board: %d" , status ) }

	// Copying back from a board you only view: a plain copy yes, a mirror no.
	models.SetMember( store , away.ID , fixture.member.ID , models.BoardRoleEditor )
	status , reply := commentCall( t , fixture , "POST" , base+"/send" , to( away , `,"top":true` ) , "" , key , "" )
	if status != 200 || reply[ "board_id" ] != float64( away.ID ) || reply[ "card_path" ] == "" { t.Fatalf( "send: %d %v" , status , reply ) }
	models.SetMember( store , away.ID , fixture.member.ID , models.BoardRoleViewer )
	sent := "/api/boards/" + itoa( away.ID ) + "/cards/" + itoa( card.ID )
	if status , _ := commentCall( t , fixture , "POST" , sent+"/copy" , to( home , `,"mirror":"one_way"` ) , "" , key , "" ); status != 403 { t.Errorf( "mirror a card you only view: %d" , status ) }
	if status , _ := commentCall( t , fixture , "POST" , sent+"/copy" , to( home , `,"mirror":"sideways"` ) , "" , key , "" ); status != 400 { t.Errorf( "unknown mirror: %d" , status ) }
	status , reply = commentCall( t , fixture , "POST" , sent+"/copy" , to( home , "" ) , "" , key , "" )
	if status != 200 || reply[ "mirror" ] != "" { t.Fatalf( "plain copy: %d %v" , status , reply ) }

	// A one-way copy refuses its synced fields over HTTP too.
	models.SetMember( store , away.ID , fixture.member.ID , models.BoardRoleEditor )
	status , reply = commentCall( t , fixture , "POST" , sent+"/copy" , to( home , `,"mirror":"one_way"` ) , "" , key , "" )
	if status != 200 || reply[ "mirror" ] != models.MirrorCopy { t.Fatalf( "one-way copy: %d %v" , status , reply ) }
	copy_path := "/api/boards/" + itoa( home.ID ) + "/cards/" + itoa( uint64( reply[ "id" ].( float64 ) ) )
	if status , reply := commentCall( t , fixture , "POST" , copy_path , `{"title":"Mine now"}` , "" , key , "" ); status != 409 || reply[ "error" ] != "card_mirrored" {
		t.Errorf( "edit a one-way copy: %d %v" , status , reply )
	}
	status , reply = commentCall( t , fixture , "GET" , copy_path+"/links" , "" , "" , key , "" )
	links , _ := reply[ "links" ].( []any )
	if status != 200 || len( links ) != 1 || links[ 0 ].( map[ string ]any )[ "direction" ] != "in" { t.Fatalf( "links: %d %v" , status , reply ) }
	link_id := itoa( uint64( links[ 0 ].( map[ string ]any )[ "id" ].( float64 ) ) )
	if status , _ := commentCall( t , fixture , "POST" , copy_path+"/links/"+link_id+"/delete" , "" , "" , key , "" ); status != 200 { t.Errorf( "unlink: %d" , status ) }
	if status , _ := commentCall( t , fixture , "POST" , copy_path , `{"title":"Mine now"}` , "" , key , "" ); status != 200 { t.Errorf( "edit after unlink: %d" , status ) }
}
