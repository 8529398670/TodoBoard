package gcal

import (
	sha256 "crypto/sha256"
	hex "encoding/hex"
	strconv "strconv"
	strings "strings"
	time "time"

	models "todoboard/server/models"
)

// How a card becomes a calendar event and back. A card with a due date is
// an event ending then: all day when the due date has no time, from its
// start when it has one, else the half hour before. On a calendar board a
// card filed under a day but due on none is an all-day event on that day.
// Anything else has no event.

// DoneMark goes before a finished card's title in the calendar.
const DoneMark = "✓ "

// shortEvent is a timed card's event length when it has no start.
const shortEvent = 30 * time.Minute

const dayFormat = "2006-01-02"

// Private extended properties on every event we make: which board and card
// it stands for. Events without them are not ours and are left alone.
const (
	propBoard = "todoboardBoard"
	propCard  = "todoboardCard"
)

// EventTime is Google's start or end: a date for an all-day event, a
// dateTime otherwise. Both are always sent, the unused one null, so a
// patch that switches between them clears the other.
type EventTime struct {
	Date     *string `json:"date"`
	DateTime *string `json:"dateTime"`
}

type ExtendedProperties struct {
	Private map[string]string `json:"private,omitempty"`
}

type Event struct {
	ID                 string              `json:"id,omitempty"`
	Status             string              `json:"status,omitempty"`
	Summary            string              `json:"summary"`
	Description        string              `json:"description"`
	Start              *EventTime          `json:"start,omitempty"`
	End                *EventTime          `json:"end,omitempty"`
	ExtendedProperties *ExtendedProperties `json:"extendedProperties,omitempty"`
}

func dateTime( value string ) ( result *EventTime ) {
	result = &EventTime{ DateTime: &value }
	return
}

func date( value string ) ( result *EventTime ) {
	result = &EventTime{ Date: &value }
	return
}

// EventFor is the event a card should have, or nil for none. link is the
// card's address on this server, put at the end of the description.
func EventFor( board *models.Board , card *models.Card , link string ) ( event *Event ) {
	if card == nil || card.Archived { return }
	event = &Event{
		Summary: card.Title,
		ExtendedProperties: &ExtendedProperties{ Private: map[string]string{
			propBoard: strconv.FormatUint( board.ID , 10 ),
			propCard:  strconv.FormatUint( card.ID , 10 ),
		} },
	}
	if card.Done { event.Summary = DoneMark + card.Title }
	description := strings.TrimSpace( card.Description )
	if link != "" {
		if description != "" { description += "\n\n" }
		description += link
	}
	event.Description = description
	switch {
	case card.DueAt != nil && card.DueAllDay:
		last := card.DueAt.UTC()
		first := last
		if card.StartAt != nil && card.StartTimed == false && card.StartAt.UTC().Before( last ) { first = card.StartAt.UTC() }
		event.Start , event.End = date( first.Format( dayFormat ) ) , date( last.AddDate( 0 , 0 , 1 ).Format( dayFormat ) )
	case card.DueAt != nil:
		end := card.DueAt.UTC()
		start := end.Add( -shortEvent )
		if card.StartAt != nil && card.StartTimed && card.StartAt.Before( end ) { start = card.StartAt.UTC() }
		event.Start , event.End = dateTime( start.Format( time.RFC3339 ) ) , dateTime( end.Format( time.RFC3339 ) )
	case board.Layout == models.LayoutCalendar && card.Day != "":
		day , err := time.Parse( dayFormat , card.Day )
		if err != nil {
			event = nil
			return
		}
		event.Start , event.End = date( card.Day ) , date( day.AddDate( 0 , 0 , 1 ).Format( dayFormat ) )
	default:
		event = nil
	}
	return
}

// span is an event's start and end in one form whichever way Google wrote
// them: all-day dates as they are, times in UTC.
func span( event *Event ) ( start string , end string , all_day bool , ok bool ) {
	if event.Start == nil || event.End == nil { return }
	if event.Start.Date != nil && event.End.Date != nil {
		start , end , all_day , ok = *event.Start.Date , *event.End.Date , true , true
		return
	}
	if event.Start.DateTime == nil || event.End.DateTime == nil { return }
	from , err := time.Parse( time.RFC3339 , *event.Start.DateTime )
	if err != nil { return }
	to , err := time.Parse( time.RFC3339 , *event.End.DateTime )
	if err != nil { return }
	start , end , ok = from.UTC().Format( time.RFC3339 ) , to.UTC().Format( time.RFC3339 ) , true
	return
}

// Fingerprint is what an event says that sync cares about: title,
// description and when. Ours and Google's copy of one event match exactly
// when neither side has changed it.
func Fingerprint( event *Event ) ( result string ) {
	if event == nil { return }
	start , end , all_day , _ := span( event )
	hash := sha256.New()
	for _ , part := range []string{ strings.TrimSpace( event.Summary ) , strings.TrimSpace( event.Description ) , start , end , strconv.FormatBool( all_day ) } {
		hash.Write( []byte( part ) )
		hash.Write( []byte{ 0 } )
	}
	result = hex.EncodeToString( hash.Sum( nil ) )[ :32 ]
	return
}

// cardDate is a card date the way CardPatch takes it.
func cardDate( value *time.Time , all_day bool ) ( result string ) {
	if value == nil { return }
	if all_day {
		result = value.UTC().Format( dayFormat )
		return
	}
	result = value.UTC().Format( time.RFC3339 )
	return
}

// PatchFrom is the change an event edited in Google makes to its card: its
// title and when, never its description (the card's is the one kept). Only
// what differs is set; nil when nothing does.
func PatchFrom( board *models.Board , card *models.Card , event *Event ) ( patch *models.CardPatch ) {
	changed := &models.CardPatch{}
	any := false
	title := strings.TrimSpace( strings.TrimPrefix( strings.TrimSpace( event.Summary ) , strings.TrimSpace( DoneMark ) ) )
	if title != "" && title != card.Title {
		changed.Title = &title
		any = true
	}
	start , end , all_day , ok := span( event )
	if ok {
		current_due := cardDate( card.DueAt , card.DueAllDay )
		current_start := cardDate( card.StartAt , card.StartAt != nil && card.StartTimed == false )
		want_due , want_start := current_due , current_start
		if all_day {
			first , first_err := time.Parse( dayFormat , start )
			after , after_err := time.Parse( dayFormat , end )
			if first_err == nil && after_err == nil {
				last := after.AddDate( 0 , 0 , -1 )
				if last.Before( first ) { last = first }
				single := last.Equal( first )
				if single && card.DueAt == nil && board.Layout == models.LayoutCalendar {
					// Filed under a day, not due: it moves to the new day.
					if card.Day != start {
						day := start
						changed.Day = &day
						any = true
					}
				} else {
					want_due = last.Format( dayFormat )
					if single {
						if card.StartAt != nil { want_start = "" }
					} else {
						want_start = first.Format( dayFormat )
					}
				}
			}
		} else {
			from , _ := time.Parse( time.RFC3339 , start )
			to , _ := time.Parse( time.RFC3339 , end )
			want_due = end
			if card.StartAt != nil || to.Sub( from ) != shortEvent { want_start = start }
		}
		if want_due != current_due {
			changed.DueAt = &want_due
			any = true
		}
		if want_start != current_start {
			changed.StartAt = &want_start
			any = true
		}
	}
	if any { patch = changed }
	return
}

// eventCard is the board and card an event of ours stands for.
func eventCard( event *Event ) ( board_id uint64 , card_id uint64 , ok bool ) {
	if event.ExtendedProperties == nil { return }
	props := event.ExtendedProperties.Private
	board , board_err := strconv.ParseUint( props[ propBoard ] , 10 , 64 )
	card , card_err := strconv.ParseUint( props[ propCard ] , 10 , 64 )
	if board_err != nil || card_err != nil { return }
	board_id , card_id , ok = board , card , true
	return
}
