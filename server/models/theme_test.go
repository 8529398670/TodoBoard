package models

import (
	errors "errors"
	testing "testing"
)

func TestThemeValidate( t *testing.T ) {
	theme := &DisplayTheme{ Accent: " #1F5FD6 " }
	if err := theme.Validate(); err != nil { t.Fatal( err ) }
	if theme.Mode != ThemeLight || theme.Accent != "#1f5fd6" {
		t.Errorf( "mode should default to light and colours be cleaned: %+v" , theme )
	}

	bad := []*DisplayTheme{
		{ Mode: "auto" },
		{ Accent: "red" },
		{ Background: "#fff" },
		{ Card: "#12345g" },
		{ Column: "url(x)" },
		{ Second: "green" },
		{ Overrides: map[string]string{ "nonsense": "#000000" } },
		{ Overrides: map[string]string{ "face": "white" } },
		{ Button: &ThemeButton{ Shape: "blob" } },
	}
	for _ , candidate := range bad {
		if err := candidate.Validate(); errors.Is( err , ErrBadInput ) == false {
			t.Errorf( "%+v should be refused, got %v" , candidate , err )
		}
	}
}

// A board theme from today's editor: a second color, overrides (the board's
// own column and face among them) and a button style, all kept.
func TestThemeValidateKeepsLayers( t *testing.T ) {
	theme := &DisplayTheme{ Mode: ThemeDark , Accent: "#8C7CFF" , Second: "#ffb86b" ,
		Overrides: map[string]string{ "column": "#13162A" , "face": "#1c2038" } , Button: &ThemeButton{ Shape: "Pill" } }
	if err := theme.Validate(); err != nil { t.Fatal( err ) }
	if theme.Accent != "#8c7cff" || theme.Overrides[ "column" ] != "#13162a" || theme.Button == nil || theme.Button.Shape != "pill" {
		t.Errorf( "not kept, cleaned: %+v" , theme )
	}
	empty := &DisplayTheme{ Overrides: map[string]string{} , Button: &ThemeButton{} }
	if err := empty.Validate(); err != nil { t.Fatal( err ) }
	if empty.Overrides != nil || empty.Button != nil { t.Errorf( "empty overrides and button are not stored: %+v" , empty ) }
}

// A board theme keeps its colors for the mode not in use, cleaned, and drops
// any for the mode it is in.
func TestThemeValidateKeepsOtherMode( t *testing.T ) {
	theme := &DisplayTheme{ Mode: ThemeLight , Accent: "#1f5fd6" , Second: "#0b7a75" ,
		Light: &ThemeColors{ Accent: "#000000" , Second: "#000000" },
		Dark:  &ThemeColors{ Accent: "#FFB81C" , Second: "#ffd100" , Background: "#000000" , Overrides: map[string]string{ "face": "#0B3D2E" } } }
	if err := theme.Validate(); err != nil { t.Fatal( err ) }
	if theme.Light != nil { t.Errorf( "a light theme's light is kept: %+v" , theme.Light ) }
	if theme.Dark == nil || theme.Dark.Accent != "#ffb81c" || theme.Dark.Overrides[ "face" ] != "#0b3d2e" {
		t.Errorf( "dark not kept, cleaned: %+v" , theme.Dark )
	}
	bad := &DisplayTheme{ Mode: ThemeDark , Accent: "#1f5fd6" , Light: &ThemeColors{ Accent: "blue" , Second: "#000000" } }
	if errors.Is( bad.Validate() , ErrBadInput ) == false { t.Error( "a bad kept color is refused" ) }
}

func TestBoardTheme( t *testing.T ) {
	store := testStore( t )
	owner , _ := CreateUser( store , "Owner" , RoleAdmin )
	board , err := CreateBoard( store , owner.ID , "Themed" , nil )
	if err != nil { t.Fatal( err ) }
	if board.Theme != nil { t.Fatalf( "a new board has no theme of its own: %+v" , board.Theme ) }

	updated , err := UpdateBoardSettings( store , board.ID , &BoardSettings{ Theme: &DisplayTheme{ Mode: ThemeDark , Accent: "#ABCDEF" } } )
	if err != nil { t.Fatal( err ) }
	if updated.Theme == nil || updated.Theme.Mode != ThemeDark || updated.Theme.Accent != "#abcdef" {
		t.Errorf( "theme not stored: %+v" , updated.Theme )
	}

	if _ , err = UpdateBoardSettings( store , board.ID , &BoardSettings{ Theme: &DisplayTheme{ Accent: "blue" } } ); errors.Is( err , ErrBadInput ) == false {
		t.Errorf( "a bad colour must be refused, got %v" , err )
	}

	updated , err = UpdateBoardSettings( store , board.ID , &BoardSettings{ ClearTheme: true } )
	if err != nil { t.Fatal( err ) }
	if updated.Theme != nil { t.Errorf( "clear_theme should go back to each person's own: %+v" , updated.Theme ) }
}

// A theme's labels are cleaned like a board's: names trimmed, colors a
// palette name or #rrggbb in lower case; anything else is refused.
func TestCleanThemesLabels( t *testing.T ) {
	base := Theme{ ID: "t1" , Name: "One" , Mode: "light" , Accent: "#112233" , Second: "#445566" }
	good := base
	good.Labels = []ThemeLabel{ { Name: " Urgent " , Color: "red" } , { Name: "" , Color: "#AABBCC" } }
	cleaned , err := CleanThemes( []Theme{ good } )
	if err != nil { t.Fatal( err ) }
	labels := cleaned[ 0 ].Labels
	if len( labels ) != 2 || labels[ 0 ].Name != "Urgent" || labels[ 1 ].Color != "#aabbcc" {
		t.Errorf( "labels not cleaned: %+v" , labels )
	}
	for _ , label := range []ThemeLabel{ { Name: "x" , Color: "url(x)" } , { Name: "x" , Color: "" } , { Name: string( make( []rune , 41 ) ) , Color: "red" } } {
		bad := base
		bad.Labels = []ThemeLabel{ label }
		if _ , err := CleanThemes( []Theme{ bad } ); err == nil { t.Errorf( "%+v should be refused" , label ) }
	}
	many := base
	for len( many.Labels ) <= MaxThemeLabels { many.Labels = append( many.Labels , ThemeLabel{ Color: "red" } ) }
	if _ , err := CleanThemes( []Theme{ many } ); err == nil { t.Error( "too many labels should be refused" ) }
}
