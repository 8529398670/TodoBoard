package routes

import (
	json "encoding/json"
	http "net/http"
	strings "strings"
	testing "testing"

	models "todoboard/server/models"
)

// What the Account page sends: the notification fields and nothing else.
func notifyForm( key string ) ( body string ) {
	body = `{"pushover_user_key":"` + key + `","pushover_device":"phone","timezone":"Europe/Paris","all_day_reminder_hour":7,"reminder_lead_minutes":[60],` +
		`"assignee_defaults":{"due":true,"assigned":true,"changes":false},"watch_defaults":{"due":false,"assigned":false,"changes":true},"shared_with_me":false,"paused":true}`
	return
}

// Saving the notification form must leave every display setting alone: they
// have their own endpoint, and this form never sends them.
func TestSavingNotificationsKeepsDisplaySettings( t *testing.T ) {
	fixture := newKeyBoards( t )
	key , _ := fixture.key( t , "" )
	display := `{"timezone":"Asia/Tokyo","hour12":true,"number_cards":true,"plain_icons":true,"done_on_top":true,"add_on_top":true,` +
		`"theme":{"mode":"dark","accent":"#112233"},"own_theme_on_boards":true,"swipe_columns":false,"folder_sort":"manual"}`
	if status , text := fixture.call( t , http.MethodPost , "/api/account/display" , key , "" , display ); status != http.StatusOK {
		t.Fatalf( "display: %d %s" , status , text )
	}
	_ , before := fixture.call( t , http.MethodGet , "/api/me" , key , "" , "" )

	if status , text := fixture.call( t , http.MethodPost , "/api/notify/prefs" , key , "" , notifyForm( "" ) ); status != http.StatusOK {
		t.Fatalf( "prefs: %d %s" , status , text )
	}
	_ , after := fixture.call( t , http.MethodGet , "/api/me" , key , "" , "" )

	was , now := displayOf( t , before ) , displayOf( t , after )
	if was != now {
		t.Fatalf( "display settings changed:\nwas %s\nnow %s" , was , now )
	}
}

func displayOf( t *testing.T , me string ) ( display string ) {
	t.Helper()
	var parsed struct {
		Display json.RawMessage `json:"display"`
	}
	if err := json.Unmarshal( []byte( me ) , &parsed ); err != nil { t.Fatal( err ) }
	display = string( parsed.Display )
	return
}

func TestDeliveryChoiceKeepsThePushoverKey( t *testing.T ) {
	fixture := newKeyBoards( t )
	key , _ := fixture.key( t , "" )
	pushover_key := strings.Repeat( "a" , 30 )
	post := func( body string ) ( status int , text string ) {
		status , text = fixture.call( t , http.MethodPost , "/api/notify/prefs" , key , "" , body )
		return
	}
	withVia := func( via string ) ( body string ) {
		body = strings.TrimSuffix( notifyForm( "" ) , "}" ) + `,"notify_via":"` + via + `"}`
		return
	}
	if status , text := post( notifyForm( pushover_key ) ); status != http.StatusOK { t.Fatalf( "%d %s" , status , text ) }

	status , text := post( withVia( "devices" ) )
	if status != http.StatusOK { t.Fatalf( "%d %s" , status , text ) }
	if strings.Contains( text , `"notify_via":"devices"` ) == false { t.Errorf( "reply: %s" , text ) }
	prefs , _ := models.GetUserPrefs( fixture.store , fixture.member.ID )
	if prefs.NotifyVia != models.NotifyViaDevices || prefs.PushoverUserKey != pushover_key { t.Errorf( "saved: %q, key kept: %v" , prefs.NotifyVia , prefs.PushoverUserKey == pushover_key ) }

	if status , text = post( notifyForm( "" ) ); status != http.StatusOK { t.Fatalf( "%d %s" , status , text ) }
	prefs , _ = models.GetUserPrefs( fixture.store , fixture.member.ID )
	if prefs.NotifyVia != models.NotifyViaDevices { t.Errorf( "a form without the choice changed it to %q" , prefs.NotifyVia ) }

	if status , _ = post( withVia( "carrier-pigeon" ) ); status != http.StatusBadRequest { t.Errorf( "an unknown choice got %d" , status ) }
}

func TestSavingNotificationsSavesTheForm( t *testing.T ) {
	fixture := newKeyBoards( t )
	key , _ := fixture.key( t , "" )
	if status , text := fixture.call( t , http.MethodPost , "/api/notify/prefs" , key , "" , notifyForm( "" ) ); status != http.StatusOK {
		t.Fatalf( "%d %s" , status , text )
	}
	prefs , err := models.GetUserPrefs( fixture.store , fixture.member.ID )
	if err != nil { t.Fatal( err ) }
	if prefs.PushoverDevice != "phone" || prefs.Timezone != "Europe/Paris" || prefs.AllDayReminderHour != 7 || len( prefs.ReminderLeadMinutes ) != 1 ||
		prefs.AssigneeDefaults.Changes || prefs.WatchDefaults.Changes == false || prefs.SharedWithMe || prefs.Paused == false {
		t.Fatalf( "%+v" , prefs )
	}
}

// An empty key keeps the saved one, because the browser never had it; "-"
// removes it.
func TestSavingNotificationsPushoverKey( t *testing.T ) {
	fixture := newKeyBoards( t )
	key , _ := fixture.key( t , "" )
	saved := "abcdefghijklmnopqrstuvwxyz1234"
	for _ , step := range []struct{ sent , want string }{ { saved , saved } , { "" , saved } , { "-" , "" } } {
		if status , text := fixture.call( t , http.MethodPost , "/api/notify/prefs" , key , "" , notifyForm( step.sent ) ); status != http.StatusOK {
			t.Fatalf( "sent %q: %d %s" , step.sent , status , text )
		}
		prefs , err := models.GetUserPrefs( fixture.store , fixture.member.ID )
		if err != nil { t.Fatal( err ) }
		if prefs.PushoverUserKey != step.want {
			t.Fatalf( "sent %q: key is %q" , step.sent , prefs.PushoverUserKey )
		}
	}
}
