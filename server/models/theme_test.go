package models

import (
	errors "errors"
	testing "testing"
)

func TestThemeValidate( t *testing.T ) {
	theme := &Theme{ Accent: " #1F5FD6 " }
	if err := theme.Validate(); err != nil { t.Fatal( err ) }
	if theme.Mode != ThemeLight || theme.Accent != "#1f5fd6" {
		t.Errorf( "mode should default to light and colours be cleaned: %+v" , theme )
	}

	bad := []*Theme{
		{ Mode: "auto" },
		{ Accent: "red" },
		{ Background: "#fff" },
		{ Card: "#12345g" },
		{ Column: "url(x)" },
	}
	for _ , candidate := range bad {
		if err := candidate.Validate(); errors.Is( err , ErrBadInput ) == false {
			t.Errorf( "%+v should be refused, got %v" , candidate , err )
		}
	}
}

func TestBoardTheme( t *testing.T ) {
	store := testStore( t )
	owner , _ := CreateUser( store , "Owner" , RoleAdmin )
	board , err := CreateBoard( store , owner.ID , "Themed" , nil )
	if err != nil { t.Fatal( err ) }
	if board.Theme != nil { t.Fatalf( "a new board has no theme of its own: %+v" , board.Theme ) }

	updated , err := UpdateBoardSettings( store , board.ID , &BoardSettings{ Theme: &Theme{ Mode: ThemeDark , Accent: "#ABCDEF" } } )
	if err != nil { t.Fatal( err ) }
	if updated.Theme == nil || updated.Theme.Mode != ThemeDark || updated.Theme.Accent != "#abcdef" {
		t.Errorf( "theme not stored: %+v" , updated.Theme )
	}

	if _ , err = UpdateBoardSettings( store , board.ID , &BoardSettings{ Theme: &Theme{ Accent: "blue" } } ); errors.Is( err , ErrBadInput ) == false {
		t.Errorf( "a bad colour must be refused, got %v" , err )
	}

	updated , err = UpdateBoardSettings( store , board.ID , &BoardSettings{ ClearTheme: true } )
	if err != nil { t.Fatal( err ) }
	if updated.Theme != nil { t.Errorf( "clear_theme should go back to each person's own: %+v" , updated.Theme ) }
}
