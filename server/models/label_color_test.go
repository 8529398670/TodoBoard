package models

import (
	testing "testing"
)

func TestLabelColors( t *testing.T ) {
	board := &Board{}
	for _ , good := range []string{ "blue" , "#1a2b3c" , "#ABCDEF" } {
		if _ , err := AddLabel( board , "x" , good ); err != nil { t.Errorf( "AddLabel(%q): %v" , good , err ) }
	}
	if board.Labels[ 2 ].Color != "#abcdef" { t.Errorf( "hex colours are stored lower-case: %q" , board.Labels[ 2 ].Color ) }
	for _ , bad := range []string{ "" , "navy" , "#123" , "#12345g" , "red; background:url(x)" , "#1234567" } {
		if _ , err := AddLabel( board , "x" , bad ); err == nil { t.Errorf( "AddLabel(%q) should be refused" , bad ) }
	}
	hex := "#00ff00"
	if err := UpdateLabel( board , board.Labels[ 0 ].ID , nil , &hex ); err != nil || board.Labels[ 0 ].Color != hex {
		t.Errorf( "UpdateLabel to a hex colour: %v %q" , err , board.Labels[ 0 ].Color )
	}
	// A board's own colour stays on the palette.
	if ValidColor( "#00ff00" ) { t.Error( "board colours must stay palette names" ) }
}

func TestSavedLabelColors( t *testing.T ) {
	store := testStore( t )
	user , _ := CreateUser( store , "Ada" , RoleUser )
	prefs , _ := GetUserPrefs( store , user.ID )
	prefs.LabelColors = []SavedColor{ { Name: " Brand " , Color: "#AA0000" } , { Color: "#aa0000" } , { Color: "#00aa00" } }
	if err := SaveUserPrefs( store , user.ID , prefs ); err != nil { t.Fatal( err ) }
	again , _ := GetUserPrefs( store , user.ID )
	if len( again.LabelColors ) != 2 || again.LabelColors[ 0 ] != ( SavedColor{ Name: "Brand" , Color: "#aa0000" } ) {
		t.Errorf( "saved colours should be cleaned and de-duplicated: %+v" , again.LabelColors )
	}
	prefs.LabelColors = []SavedColor{ { Color: "red" } }
	if SaveUserPrefs( store , user.ID , prefs ) == nil { t.Error( "a palette name is not a saved colour" ) }
	prefs.LabelColors = make( []SavedColor , MaxSavedColors + 1 )
	for index := range prefs.LabelColors { prefs.LabelColors[ index ].Color = "#0000" + string( "0123456789abcdef"[ index / 16 ] ) + string( "0123456789abcdef"[ index % 16 ] ) }
	if SaveUserPrefs( store , user.ID , prefs ) == nil { t.Errorf( "more than %d colours should be refused" , MaxSavedColors ) }
}
