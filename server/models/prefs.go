package models

import (
	errors "errors"
	regexp "regexp"
	sort "sort"
	strings "strings"
	time "time"

	db "todoboard/server/db"
	encryption "todoboard/server/encryption"
)

// EventSet is which notifications someone wants for a board or a card.
//
//   Due      reminders before a card is due, and when it falls overdue
//   Assigned being added as a card's assignee
//   Changes  the card being moved, completed, reopened or edited
type EventSet struct {
	Due      bool `json:"due"`
	Assigned bool `json:"assigned"`
	Changes  bool `json:"changes"`
}

// UserPrefs is kept apart from User so the account record the template owns
// stays exactly as the template defines it.
//
// AssigneeDefaults applies to cards the person is assigned to and
// WatchDefaults to every other card they can see, unless a board or card
// subscription says otherwise. SharedWithMe is account-wide: it is about the
// person, not about any one board.
type UserPrefs struct {
	PushoverUserKey     string   `json:"pushover_user_key"`
	PushoverDevice      string   `json:"pushover_device"`
	Timezone            string   `json:"timezone"`
	ReminderLeadMinutes []int    `json:"reminder_lead_minutes"`
	AllDayReminderHour  int      `json:"all_day_reminder_hour"`
	AssigneeDefaults    EventSet `json:"assignee_defaults"`
	WatchDefaults       EventSet `json:"watch_defaults"`
	SharedWithMe        bool     `json:"shared_with_me"`
	Paused              bool     `json:"paused"`

	// How times are shown. DisplayTimezone "" follows the browser; Hour12
	// nil follows the browser's locale. Choosing a display timezone also
	// sets Timezone, so reminders fire on the same clock the person reads.
	DisplayTimezone string `json:"display_timezone"`
	Hour12          *bool  `json:"hour12"`

	// NumberCards is the person's default for boards whose own numbering
	// setting is left to each viewer.
	NumberCards bool `json:"number_cards"`

	// PlainIcons shows Doing / Done icons in greyscale on every board this
	// person has not set otherwise (BoardRef.PlainIcons).
	PlainIcons bool `json:"plain_icons"`

	// DoneOnTop and AddOnTop are the person's defaults for boards whose own
	// placement settings are left to each viewer (Board.DonePlacement,
	// Board.AddPlacement): finished cards and ticked checklist items go to
	// the top of their list, and the add-card box sits above the cards.
	DoneOnTop bool `json:"done_on_top"`
	AddOnTop  bool `json:"add_on_top"`

	// SwipeColumns shows a board one full-width column at a time on a phone
	// held upright, swiped between. nil -- never chosen -- means on. Stored
	// under a new key: the old "swipe_columns" was saved as false by anyone
	// who changed any display setting, so it could not tell a choice from
	// the old default.
	SwipeColumns *bool `json:"phone_one_column"`

	// FolderSort is how the home page orders folders: "" by name, "manual"
	// by the order they were dragged into (Folder.Position).
	FolderSort string `json:"folder_sort"`

	// Theme is the person's own look everywhere; nil is plain light.
	// OwnThemeOnBoards keeps it on boards that have a theme of their own.
	Theme            *Theme `json:"theme"`
	OwnThemeOnBoards bool   `json:"own_theme_on_boards"`
}

const FolderSortManual = "manual"

func DefaultUserPrefs() ( prefs *UserPrefs ) {
	prefs = &UserPrefs{
		Timezone:            "UTC",
		ReminderLeadMinutes: []int{ 1440 , 60 },
		AllDayReminderHour:  9,
		AssigneeDefaults:    EventSet{ Due: true , Assigned: true , Changes: true },
		WatchDefaults:       EventSet{ Due: false , Assigned: true , Changes: false },
		SharedWithMe:        true,
	}
	return
}

// Pushover user and group keys are 30 characters of [A-Za-z0-9].
var pushoverKeyPattern = regexp.MustCompile( `^[A-Za-z0-9]{30}$` )
var pushoverDevicePattern = regexp.MustCompile( `^[A-Za-z0-9_-]{0,25}$` )

func ( prefs *UserPrefs ) Location() ( location *time.Location ) {
	location , err := time.LoadLocation( prefs.Timezone )
	if err != nil || prefs.Timezone == "" { location = time.UTC }
	return
}

func ( prefs *UserPrefs ) CanPush() ( result bool ) {
	result = prefs.PushoverUserKey != "" && prefs.Paused == false
	return
}

// Validate cleans what can be cleaned and refuses the rest.
func ( prefs *UserPrefs ) Validate() ( err error ) {
	prefs.PushoverUserKey = strings.TrimSpace( prefs.PushoverUserKey )
	prefs.PushoverDevice = strings.TrimSpace( prefs.PushoverDevice )
	if prefs.PushoverUserKey != "" && pushoverKeyPattern.MatchString( prefs.PushoverUserKey ) == false { return ErrBadInput }
	if pushoverDevicePattern.MatchString( prefs.PushoverDevice ) == false { return ErrBadInput }
	if _ , load_err := time.LoadLocation( prefs.Timezone ); load_err != nil || prefs.Timezone == "" { return ErrBadInput }
	if prefs.DisplayTimezone != "" {
		if _ , load_err := time.LoadLocation( prefs.DisplayTimezone ); load_err != nil { return ErrBadInput }
	}
	if prefs.Theme != nil {
		if err = prefs.Theme.Validate(); err != nil { return }
	}
	if prefs.FolderSort != "" && prefs.FolderSort != FolderSortManual { return ErrBadInput }
	if prefs.AllDayReminderHour < 0 || prefs.AllDayReminderHour > 23 { return ErrBadInput }
	leads := []int{}
	for _ , lead := range prefs.ReminderLeadMinutes {
		// Up to 30 days ahead; 0 is "at the due time", which overdue
		// already covers, so it is dropped rather than sent twice.
		if lead <= 0 || lead > 43200 || containsInt( leads , lead ) { continue }
		leads = append( leads , lead )
	}
	if len( leads ) > 5 { return ErrBadInput }
	sort.Sort( sort.Reverse( sort.IntSlice( leads ) ) )
	prefs.ReminderLeadMinutes = leads
	return
}

func containsInt( values []int , target int ) ( result bool ) {
	for _ , value := range values {
		if value == target { return true }
	}
	return
}

func GetUserPrefs( store *db.Store , user_id uint64 ) ( prefs *UserPrefs , err error ) {
	prefs = DefaultUserPrefs()
	err = store.Get( db.BucketUserPrefs , encryption.Uint64ToBytes( user_id ) , prefs )
	if errors.Is( err , db.ErrNotFound ) { err = nil }
	if err != nil { prefs = nil }
	return
}

func SaveUserPrefs( store *db.Store , user_id uint64 , prefs *UserPrefs ) ( err error ) {
	if err = prefs.Validate(); err != nil { return }
	err = store.Put( db.BucketUserPrefs , encryption.Uint64ToBytes( user_id ) , prefs )
	return
}

// ---------------------------------------------------------------------------
// subscriptions
// ---------------------------------------------------------------------------

const (
	subscriptionBoard = uint64( 'b' )
	subscriptionCard  = uint64( 'c' )
)

// Subscription overrides the defaults for one board or one card. It carries
// its own coordinates so a scan of the bucket can be filtered without
// decoding the key.
type Subscription struct {
	UserID  uint64   `json:"user_id"`
	Kind    uint64   `json:"kind"`
	BoardID uint64   `json:"board_id"`
	CardID  uint64   `json:"card_id"`
	Events  EventSet `json:"events"`
}

func subscriptionKey( user_id uint64 , kind uint64 , board_id uint64 , card_id uint64 ) ( key []byte ) {
	if kind == subscriptionBoard {
		key = db.Key( user_id , kind , board_id )
		return
	}
	key = db.Key( user_id , kind , board_id , card_id )
	return
}

func getSubscription( store *db.Store , user_id uint64 , kind uint64 , board_id uint64 , card_id uint64 ) ( sub *Subscription ) {
	found := &Subscription{}
	if store.Get( db.BucketSubscriptions , subscriptionKey( user_id , kind , board_id , card_id ) , found ) != nil { return }
	sub = found
	return
}

func GetBoardSubscription( store *db.Store , user_id uint64 , board_id uint64 ) ( events *EventSet ) {
	if sub := getSubscription( store , user_id , subscriptionBoard , board_id , 0 ); sub != nil { events = &sub.Events }
	return
}

func GetCardSubscription( store *db.Store , user_id uint64 , board_id uint64 , card_id uint64 ) ( events *EventSet ) {
	if sub := getSubscription( store , user_id , subscriptionCard , board_id , card_id ); sub != nil { events = &sub.Events }
	return
}

// SetBoardSubscription stores an override, or with events nil removes it so
// the defaults apply again.
func SetBoardSubscription( store *db.Store , user_id uint64 , board_id uint64 , events *EventSet ) ( err error ) {
	err = setSubscription( store , user_id , subscriptionBoard , board_id , 0 , events )
	return
}

func SetCardSubscription( store *db.Store , user_id uint64 , board_id uint64 , card_id uint64 , events *EventSet ) ( err error ) {
	err = setSubscription( store , user_id , subscriptionCard , board_id , card_id , events )
	return
}

func setSubscription( store *db.Store , user_id uint64 , kind uint64 , board_id uint64 , card_id uint64 , events *EventSet ) ( err error ) {
	key := subscriptionKey( user_id , kind , board_id , card_id )
	if events == nil {
		err = store.Delete( db.BucketSubscriptions , key )
		return
	}
	err = store.Put( db.BucketSubscriptions , key , &Subscription{ UserID: user_id , Kind: kind , BoardID: board_id , CardID: card_id , Events: *events } )
	return
}

func deleteSubscriptionTx( tx *db.Tx , user_id uint64 , kind uint64 , board_id uint64 , card_id uint64 ) ( err error ) {
	err = tx.Delete( db.BucketSubscriptions , subscriptionKey( user_id , kind , board_id , card_id ) )
	return
}

// SubscriptionSource says where an effective EventSet came from, so the UI
// can show "inherited from board" next to a checkbox.
const (
	SourceCard     = "card"
	SourceBoard    = "board"
	SourceAssignee = "assignee_defaults"
	SourceWatch    = "watch_defaults"
)

// EffectiveEvents resolves card override, then board override, then the
// person's defaults for assigned or other cards. card may be nil for a
// board-level question.
func EffectiveEvents( store *db.Store , prefs *UserPrefs , user_id uint64 , board_id uint64 , card *Card ) ( events EventSet , source string ) {
	if card != nil {
		if override := GetCardSubscription( store , user_id , board_id , card.ID ); override != nil {
			events , source = *override , SourceCard
			return
		}
	}
	if override := GetBoardSubscription( store , user_id , board_id ); override != nil {
		events , source = *override , SourceBoard
		return
	}
	if card != nil && containsID( card.Assignees , user_id ) {
		events , source = prefs.AssigneeDefaults , SourceAssignee
		return
	}
	events , source = prefs.WatchDefaults , SourceWatch
	return
}
