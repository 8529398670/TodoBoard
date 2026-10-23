package notify

import (
	fmt "fmt"
	log "log"
	time "time"

	db "todoboard/server/db"
	models "todoboard/server/models"
)

// leadOverdue is the ledger's name for the "now overdue" notification, which
// is a reminder with no lead time.
const leadOverdue = 0

const (
	// A reminder whose moment passed while the server was down is still sent
	// if it is no older than this; after that it is stale news and skipped.
	reminderGrace = 6 * time.Hour
	overdueGrace  = 24 * time.Hour

	ledgerKeep = 30 * 24 * time.Hour
)

// SentRecord is one row of the reminder ledger. The key says what was sent
// -- card, person, lead time, and the due moment it was computed from -- so
// moving a due date produces new keys and the reminders fire again, while the
// same due date can never remind the same person twice.
type SentRecord struct {
	At time.Time `json:"at"`
}

func ledgerKey( card_id uint64 , user_id uint64 , lead_minutes int , due time.Time ) ( key []byte ) {
	key = db.Key( card_id , user_id , uint64( lead_minutes ) , uint64( due.Unix() ) )
	return
}

// DueMoment is when a card is due for one person. A timed due date is an
// instant; an all-day date means "that day", so it is placed at the person's
// chosen reminder hour in their own timezone.
func DueMoment( card *models.Card , prefs *models.UserPrefs ) ( due time.Time , ok bool ) {
	if card.DueAt == nil { return }
	ok = true
	if card.DueAllDay == false {
		due = *card.DueAt
		return
	}
	year , month , day := card.DueAt.UTC().Date()
	due = time.Date( year , month , day , prefs.AllDayReminderHour , 0 , 0 , 0 , prefs.Location() )
	return
}

// reminder is one notification the scheduler decided is due now.
type reminder struct {
	lead int
	due  time.Time
}

// dueReminders lists the reminders that should fire at now for one card and
// one person, ignoring the ledger.
func dueReminders( card *models.Card , prefs *models.UserPrefs , now time.Time ) ( result []reminder ) {
	if card.Done || card.Archived { return }
	due , ok := DueMoment( card , prefs )
	if ok == false { return }
	for _ , lead := range prefs.ReminderLeadMinutes {
		fire := due.Add( -time.Duration( lead ) * time.Minute )
		if now.Before( fire ) || now.Sub( fire ) > reminderGrace || now.After( due ) { continue }
		result = append( result , reminder{ lead: lead , due: due } )
	}
	if now.Before( due ) == false && now.Sub( due ) <= overdueGrace {
		result = append( result , reminder{ lead: leadOverdue , due: due } )
	}
	return
}

func ( service *Service ) scheduleLoop() {
	ticker := time.NewTicker( service.cfg.DueCheckInterval )
	defer ticker.Stop()
	purge := time.NewTicker( 6 * time.Hour )
	defer purge.Stop()
	service.checkDue()
	for {
		select {
		case <-service.stop:
			return
		case <-ticker.C:
			service.checkDue()
		case <-purge.C:
			service.purgeLedger()
		}
	}
}

// checkDue walks every card with a due date. A full scan is fine at the size
// a bolt-backed team board reaches; it reads, it does not hold the writer.
func ( service *Service ) checkDue() {
	now := service.now()
	cards := []*models.Card{}
	err := service.store.ForEach( db.BucketCards , func() any { return &models.Card{} } ,
		func( key []byte , item any ) bool {
			if card := item.( *models.Card ); card.DueAt != nil && card.Done == false && card.Archived == false {
				cards = append( cards , card )
			}
			return true
		} )
	if err != nil {
		log.Printf( "[notify] due scan failed: %v" , err )
		return
	}

	boards := map[uint64]*models.Board{}
	prefs_cache := map[uint64]*models.UserPrefs{}
	for _ , card := range cards {
		board , seen := boards[ card.BoardID ]
		if seen == false {
			board , _ = models.GetBoard( service.store , card.BoardID )
			boards[ card.BoardID ] = board
		}
		if board == nil { continue }
		for _ , user_id := range recipients( board , card ) {
			prefs , cached := prefs_cache[ user_id ]
			if cached == false {
				prefs = service.prefsFor( user_id )
				prefs_cache[ user_id ] = prefs
			}
			if prefs == nil { continue }
			fires := dueReminders( card , prefs , now )
			if len( fires ) == 0 { continue }
			if events , _ := models.EffectiveEvents( service.store , prefs , user_id , board.ID , card ); events.Due == false { continue }
			for _ , fire := range fires {
				service.sendReminder( board , card , user_id , prefs , fire , now )
			}
		}
	}
}

func ( service *Service ) sendReminder( board *models.Board , card *models.Card , user_id uint64 , prefs *models.UserPrefs , fire reminder , now time.Time ) {
	key := ledgerKey( card.ID , user_id , fire.lead , fire.due )
	if service.store.Get( db.BucketNotifySent , key , &SentRecord{} ) == nil { return }
	// Recorded before sending: a crash between the two loses one reminder,
	// where the other order would repeat it on every restart.
	if err := service.store.Put( db.BucketNotifySent , key , &SentRecord{ At: now.UTC() } ); err != nil { return }

	when := fire.due.In( prefs.Location() )
	due_text := when.Format( service.text( "push.due_time_format" , nil ) )
	if card.DueAllDay { due_text = when.Format( service.text( "push.due_date_format" , nil ) ) }
	list_title := ""
	if list := board.List( card.ListID ); list != nil { list_title = list.Title }
	values := map[string]string{
		"card":  card.Title,
		"board": board.Title,
		"list":  list_title,
		"due":   due_text,
		"in":    service.leadText( fire.lead ),
	}
	title_key := "push.due_soon_title"
	if fire.lead == leadOverdue { title_key = "push.overdue_title" }
	service.enqueue( prefs , Message{
		Title: service.text( title_key , values ),
		Body:  service.text( "push.due_body" , values ),
		URL:   service.cardURL( board.ID , card.ID ),
	} )
}

// leadText words a lead time ("1 day", "2 hours", "30 minutes").
func ( service *Service ) leadText( minutes int ) ( result string ) {
	switch {
	case minutes >= 1440 && minutes % 1440 == 0:
		result = service.plural( minutes / 1440 , "push.unit_day" , "push.unit_days" )
	case minutes >= 60 && minutes % 60 == 0:
		result = service.plural( minutes / 60 , "push.unit_hour" , "push.unit_hours" )
	default:
		result = service.plural( minutes , "push.unit_minute" , "push.unit_minutes" )
	}
	return
}

func ( service *Service ) plural( count int , one string , many string ) ( result string ) {
	key := many
	if count == 1 { key = one }
	result = service.text( key , map[string]string{ "n": fmt.Sprint( count ) } )
	return
}

func ( service *Service ) purgeLedger() {
	cutoff := service.now().Add( -ledgerKeep )
	removed , err := service.store.DeleteWhere( db.BucketNotifySent , func() any { return &SentRecord{} } ,
		func( key []byte , item any ) bool { return item.( *SentRecord ).At.Before( cutoff ) } )
	if err == nil && removed > 0 {
		log.Printf( "[notify] purged %d old reminder records" , removed )
	}
}
