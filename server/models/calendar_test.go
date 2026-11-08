package models

import (
	testing "testing"
	time "time"
)

func TestDayFromTitle( t *testing.T ) {
	cases := map[string]string{
		"06NOV2026":        "2026-11-06",
		"6 Nov 2026":       "2026-11-06",
		"06-nov-2026":      "2026-11-06",
		"Nov 6 2026":       "2026-11-06",
		"November 6, 2026": "2026-11-06",
		"Nov 6th 2026":     "2026-11-06",
		"2026-11-06":       "2026-11-06",
		"2026/1/9":         "2026-01-09",
		"31FEB2026":        "",
		"To do":            "",
		"2026":             "",
	}
	for title , want := range cases {
		if got := DayFromTitle( title ); got != want { t.Errorf( "DayFromTitle(%q) = %q, want %q" , title , got , want ) }
	}
	if ValidDay( "2026-02-30" ) || ValidDay( "07NOV2026" ) || ValidDay( "" ) == false || ValidDay( "2026-11-07" ) == false {
		t.Error( "ValidDay accepts or refuses the wrong things" )
	}
}

func TestCalendarBoard( t *testing.T ) {
	store := testStore( t )
	owner , _ := CreateUser( store , "Owner" , RoleAdmin )
	board , _ := CreateBoard( store , owner.ID , "Daily" , []string{ "05NOV2026" , "06NOV2026" , "Someday" } )
	old , _ := CreateCard( store , board.ID , board.Lists[ 0 ].ID , "Old open" , nil , owner.ID , "Owner" )
	done , _ := CreateCard( store , board.ID , board.Lists[ 0 ].ID , "Old done" , nil , owner.ID , "Owner" )
	yes := true
	PatchCard( store , board.ID , done.ID , &CardPatch{ Done: &yes } , "Owner" )
	own_day := "2026-12-25"
	kept , _ := CreateCard( store , board.ID , board.Lists[ 1 ].ID , "Has its own day" , nil , owner.ID , "Owner" )
	PatchCard( store , board.ID , kept.ID , &CardPatch{ Day: &own_day } , "Owner" )
	loose , _ := CreateCard( store , board.ID , board.Lists[ 2 ].ID , "No date column" , nil , owner.ID , "Owner" )

	bad := "07NOV2026"
	if _ , _ , err := PatchCard( store , board.ID , loose.ID , &CardPatch{ Day: &bad } , "Owner" ); err == nil { t.Error( "a day must be 2026-11-07 shaped" ) }

	// Becoming a calendar dates the cards in dated columns, and no others.
	layout := LayoutCalendar
	if _ , err := UpdateBoardSettings( store , board.ID , &BoardSettings{ Layout: &layout } ); err != nil { t.Fatal( err ) }
	if dated , err := DaysFromListTitles( store , board.ID ); err != nil || dated != 2 { t.Fatalf( "dated %d: %v" , dated , err ) }
	day := func( id uint64 ) string { card , _ := GetCard( store , board.ID , id ) ; return card.Day }
	if day( old.ID ) != "2026-11-05" || day( done.ID ) != "2026-11-05" || day( kept.ID ) != own_day || day( loose.ID ) != "" {
		t.Errorf( "days: %q %q %q %q" , day( old.ID ) , day( done.ID ) , day( kept.ID ) , day( loose.ID ) )
	}

	// Carry over moves what is unfinished from the past to today.
	today := time.Now().UTC().Format( dayLayout )
	if moved , err := CarryOverCards( store , board.ID , today , "Owner" ); err != nil || moved != 1 { t.Fatalf( "moved %d: %v" , moved , err ) }
	if day( old.ID ) != today || day( done.ID ) != "2026-11-05" || day( kept.ID ) != own_day { t.Errorf( "after carry over: %q %q %q" , day( old.ID ) , day( done.ID ) , day( kept.ID ) ) }
	if moved , _ := CarryOverCards( store , board.ID , today , "Owner" ); moved != 0 { t.Errorf( "a second carry over moved %d" , moved ) }
	if _ , err := CarryOverCards( store , board.ID , "2001-01-01" , "Owner" ); err == nil { t.Error( "a today far from the server's must be refused" ) }

	// Off, nothing moves.
	off := false
	UpdateBoardSettings( store , board.ID , &BoardSettings{ CarryOver: &off } )
	past := "2026-11-01"
	PatchCard( store , board.ID , old.ID , &CardPatch{ Day: &past } , "Owner" )
	if moved , _ := CarryOverCards( store , board.ID , today , "Owner" ); moved != 0 { t.Errorf( "carry over off moved %d" , moved ) }
}
