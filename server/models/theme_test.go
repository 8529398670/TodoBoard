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
