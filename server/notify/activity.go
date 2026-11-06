package notify

import (
	fmt "fmt"
	strings "strings"
	time "time"

	models "todoboard/server/models"
)

// A person subscribed to a board's activity hears about nearly everything
// that happens on it, gathered into one notification per board rather than
// one per change. Each change is worded when it is published, because a card
// that was just trashed can no longer be looked up to say what it was called.
//
// Rearranging is left out on purpose: reorders, ranks and column widths would
// bury everything else.

// maxActivityLines is how many changes one notification spells out.
const maxActivityLines = 20

type pendingActivity struct {
	boardID uint64
	userID  uint64
	since   time.Time
	lines   []string
	extra   int
}

// Activity reports something that happened on a board. what names its wording
// in language.yaml (push.activity_<what>), and values fill it in.
func ( service *Service ) Activity( board_id uint64 , actor_id uint64 , actor_name string , what string , values map[string]string ) {
	service.publish( event{ kind: eventActivity , boardID: board_id , actorID: actor_id , actorName: actor_name , what: what , values: values } )
}

func ( service *Service ) handleActivity( item event ) {
	board , err := models.GetBoard( service.store , item.boardID )
	if err != nil { return }
	what := ""
	if item.what != "" {
		what = service.text( "push.activity_" + item.what , item.values )
	} else {
		what = service.cardActivity( board , item.change )
	}
	if what == "" { return }
	line := service.text( "push.activity_line" , map[string]string{ "actor": service.actorLabel( item.actorName ) , "what": what } )
	for _ , user_id := range recipients( board , nil ) {
		if user_id == item.actorID { continue }
		prefs := service.prefsFor( user_id )
		if prefs == nil { continue }
		events , _ := models.EffectiveEvents( service.store , prefs , user_id , board.ID , nil )
		if events.Activity == false { continue }
		// Being assigned has a notification of its own.
		if events.Assigned && containsUser( item.change.NewAssignees , user_id ) { continue }
		service.queueActivity( board.ID , user_id , line )
	}
}

// cardActivity words what a patch did to a card, or "" if nothing worth
// telling. An archive also counts as an edit, which says less.
func ( service *Service ) cardActivity( board *models.Board , change models.CardChange ) ( result string ) {
	parts := []string{}
	if change.Moved {
		list_title := ""
		if list := board.List( change.ToListID ); list != nil { list_title = list.Title }
		parts = append( parts , service.text( "push.change_moved" , map[string]string{ "list": list_title } ) )
	}
	if change.Completed { parts = append( parts , service.text( "push.change_completed" , nil ) ) }
	if change.Reopened { parts = append( parts , service.text( "push.change_reopened" , nil ) ) }
	if change.Archived { parts = append( parts , service.text( "push.change_archived" , nil ) ) }
	if change.Unarchived { parts = append( parts , service.text( "push.change_unarchived" , nil ) ) }
	if change.DueChanged { parts = append( parts , service.text( "push.change_due" , nil ) ) }
	if change.StartChanged { parts = append( parts , service.text( "push.change_start" , nil ) ) }
	if change.LabelsChanged { parts = append( parts , service.text( "push.change_labels" , nil ) ) }
	if len( change.NewAssignees ) > 0 { parts = append( parts , service.text( "push.change_assigned" , nil ) ) }
	if change.AssigneesRemoved { parts = append( parts , service.text( "push.change_unassigned" , nil ) ) }
	if change.Edited && change.Archived == false && change.Unarchived == false { parts = append( parts , service.text( "push.change_edited" , nil ) ) }
	if len( parts ) == 0 { return }
	result = service.text( "push.activity_card" , map[string]string{ "card": change.Title , "changes": strings.Join( parts , ", " ) } )
	return
}

func containsUser( user_ids []uint64 , target uint64 ) ( result bool ) {
	for _ , id := range user_ids {
		if id == target { return true }
	}
	return
}

func ( service *Service ) queueActivity( board_id uint64 , user_id uint64 , line string ) {
	service.mutex.Lock()
	defer service.mutex.Unlock()
	key := fmt.Sprintf( "%d/%d" , board_id , user_id )
	entry := service.activity[ key ]
	if entry == nil {
		entry = &pendingActivity{ boardID: board_id , userID: user_id , since: service.now() }
		service.activity[ key ] = entry
	}
	if containsString( entry.lines , line ) == false {
		if len( entry.lines ) < maxActivityLines {
			entry.lines = append( entry.lines , line )
		} else {
			entry.extra += 1
		}
	}
	if service.cfg.ChangeDebounce == 0 {
		go safely( "flush" , func() { service.flush( false ) } )
	}
}

// sendActivity asks again who may hear this: someone can have left the board
// or changed their mind while the window was open.
func ( service *Service ) sendActivity( entry *pendingActivity ) {
	board , err := models.GetBoard( service.store , entry.boardID )
	if err != nil || board.RoleOf( entry.userID ) == "" { return }
	prefs := service.prefsFor( entry.userID )
	if prefs == nil { return }
	if events , _ := models.EffectiveEvents( service.store , prefs , entry.userID , board.ID , nil ); events.Activity == false { return }

	total := len( entry.lines ) + entry.extra
	title := "push.activity_title_many"
	if total == 1 { title = "push.activity_title_one" }
	lines := entry.lines
	if entry.extra > 0 { lines = append( lines , service.text( "push.activity_more" , map[string]string{ "n": fmt.Sprint( entry.extra ) } ) ) }
	service.enqueue( entry.userID , prefs , service.linked( Notice{
		Title:    service.text( title , map[string]string{ "board": board.Title , "n": fmt.Sprint( total ) } ),
		Body:     strings.Join( lines , "\n" ),
		URLTitle: service.text( "push.open_board" , nil ),
		Topic:    fmt.Sprintf( "board-%d" , board.ID ),
		TTL:      routineTTL,
	} , board.ID , 0 ) )
}
