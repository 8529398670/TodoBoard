package models

import (
	regexp "regexp"
	strings "strings"
	time "time"

	db "todoboard/server/db"
)

// A calendar board (LayoutCalendar) files each card under a day, Card.Day,
// and shows the days a month, a week or a day at a time, each with its own
// To do, Doing and Done from the card's flags. The board's lists still hold
// the cards, so switching the layout back shows them where they were; the
// calendar just does not look at them. A card with no day waits in a tray.

const dayLayout = "2006-01-02"

// ValidDay is "" or a real date written 2026-11-07.
func ValidDay( day string ) ( result bool ) {
	if day == "" { return true }
	_ , err := time.Parse( dayLayout , day )
	result = err == nil
	return
}

// CarriesOver is the board's carry-over setting, on unless turned off.
func ( board *Board ) CarriesOver() ( result bool ) {
	result = board.CarryOver == nil || *board.CarryOver
	return
}

var (
	// 06NOV2026, 6 Nov 2026, 06-Nov-2026
	dayMonthYear = regexp.MustCompile( `^(\d{1,2})[\s\-./]*([A-Za-z]{3,9})[\s\-./,]*(\d{4})$` )
	// Nov 6 2026, November 6, 2026
	monthDayYear = regexp.MustCompile( `^([A-Za-z]{3,9})[\s\-./]*(\d{1,2})(?:st|nd|rd|th)?[\s\-./,]*(\d{4})$` )
	// 2026-11-06, 2026/11/06
	yearMonthDay = regexp.MustCompile( `^(\d{4})[\-./](\d{1,2})[\-./](\d{1,2})$` )
)

// DayFromTitle reads a column title that is a date -- 06NOV2026,
// 2026-11-06, Nov 6 2026 and the like -- as a day, or "" if it is not one.
// It is how an existing board of dated columns becomes a calendar.
func DayFromTitle( title string ) ( day string ) {
	clean := strings.TrimSpace( title )
	parse := func( year string , month string , date string ) string {
		var parsed time.Time
		var err error
		if len( month ) <= 2 {
			parsed , err = time.Parse( "2006-1-2" , year+"-"+month+"-"+date )
		} else {
			name := strings.ToUpper( month[ :1 ] ) + strings.ToLower( month[ 1:3 ] )
			parsed , err = time.Parse( "2006-Jan-2" , year+"-"+name+"-"+date )
		}
		if err != nil { return "" }
		return parsed.Format( dayLayout )
	}
	if match := dayMonthYear.FindStringSubmatch( clean ); match != nil {
		day = parse( match[ 3 ] , match[ 2 ] , match[ 1 ] )
	} else if match := monthDayYear.FindStringSubmatch( clean ); match != nil {
		day = parse( match[ 3 ] , match[ 1 ] , match[ 2 ] )
	} else if match := yearMonthDay.FindStringSubmatch( clean ); match != nil {
		day = parse( match[ 1 ] , match[ 2 ] , match[ 3 ] )
	}
	return
}

// DaysFromListTitles gives every card with no day, in a column whose title
// is a date, that day. Run when a board becomes a calendar; a card with a
// day of its own keeps it. Answers how many cards it dated.
func DaysFromListTitles( store *db.Store , board_id uint64 ) ( dated int , err error ) {
	err = store.Write( func( tx *db.Tx ) ( tx_err error ) {
		board , tx_err := getBoardTx( tx , board_id )
		if tx_err != nil { return }
		days := map[uint64]string{}
		for _ , list := range board.Lists {
			if day := DayFromTitle( list.Title ); day != "" { days[ list.ID ] = day }
		}
		if len( days ) == 0 { return }
		tx_err = eachCardTx( tx , board_id , func( card *Card ) bool {
			if card.Day != "" || days[ card.ListID ] == "" { return false }
			card.Day = days[ card.ListID ]
			dated += 1
			return true
		} )
		if tx_err != nil || dated == 0 { return }
		tx_err = putBoardTx( tx , board )
		return
	} )
	return
}

// CarryOverCards moves every unfinished, unarchived card filed under a day
// before today to today. today is the caller's, since a day starts at
// different times for different people; it must be within a day of the
// server's date either way, which every timezone is. Answers how many moved.
func CarryOverCards( store *db.Store , board_id uint64 , today string , actor_name string ) ( moved int , err error ) {
	parsed , parse_err := time.Parse( dayLayout , today )
	server_today := time.Now().UTC().Truncate( 24 * time.Hour )
	if parse_err != nil || parsed.Before( server_today.AddDate( 0 , 0 , -1 ) ) || parsed.After( server_today.AddDate( 0 , 0 , 1 ) ) {
		err = ErrBadInput
		return
	}
	err = store.Write( func( tx *db.Tx ) ( tx_err error ) {
		board , tx_err := getBoardTx( tx , board_id )
		if tx_err != nil { return }
		if board.Layout != LayoutCalendar || board.CarriesOver() == false { return }
		now := time.Now().UTC()
		tx_err = eachCardTx( tx , board_id , func( card *Card ) bool {
			// Days written 2026-11-07 sort as text.
			if card.Day == "" || card.Day >= today || card.Done || card.Archived { return false }
			card.Day = today
			card.UpdatedAt , card.UpdatedBy = now , actor_name
			moved += 1
			return true
		} )
		if tx_err != nil || moved == 0 { return }
		tx_err = putBoardTx( tx , board )
		return
	} )
	return
}
