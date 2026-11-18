package models

import (
	errors "errors"
	time "time"

	db "todoboard/server/db"
	encryption "todoboard/server/encryption"
)

// GoogleAccount is one person's connected Google Calendar (server/gcal). The
// refresh token is a credential for their calendar: it is never sent back to
// a browser, and the database is encrypted at rest.
type GoogleAccount struct {
	RefreshToken string     `json:"refresh_token"`
	Email        string     `json:"email"`
	// CalendarID is the calendar cards go to; "primary" is their main one.
	CalendarID   string     `json:"calendar_id"`
	// SyncToken is Google's bookmark for changes since the last read.
	SyncToken    string     `json:"sync_token,omitempty"`
	// Problem is the last sync's failure, for the settings screen; "" when
	// the last sync went through.
	Problem      string     `json:"problem,omitempty"`
	ConnectedAt  time.Time  `json:"connected_at"`
	LastSync     *time.Time `json:"last_sync,omitempty"`
}

// GoogleLink is the calendar event one card is synced to, for one person.
// Fingerprint is what the event was last made to say (gcal.Fingerprint), so
// a change on either side shows as a difference from it and our own write
// coming back from Google does not.
type GoogleLink struct {
	EventID     string `json:"event_id"`
	CalendarID  string `json:"calendar_id"`
	Fingerprint string `json:"fingerprint"`
}

func GetGoogleAccount( store *db.Store , user_id uint64 ) ( account *GoogleAccount , err error ) {
	found := &GoogleAccount{}
	err = store.Get( db.BucketGoogleAccounts , encryption.Uint64ToBytes( user_id ) , found )
	if errors.Is( err , db.ErrNotFound ) {
		err = nil
		return
	}
	if err == nil { account = found }
	return
}

func SaveGoogleAccount( store *db.Store , user_id uint64 , account *GoogleAccount ) ( err error ) {
	err = store.Put( db.BucketGoogleAccounts , encryption.Uint64ToBytes( user_id ) , account )
	return
}

// UpdateGoogleAccount changes a saved account in place; ErrNotFound-free: an
// account disconnected meanwhile is left gone.
func UpdateGoogleAccount( store *db.Store , user_id uint64 , change func( account *GoogleAccount ) ) ( err error ) {
	err = store.Write( func( tx *db.Tx ) ( tx_err error ) {
		account := &GoogleAccount{}
		tx_err = tx.Get( db.BucketGoogleAccounts , encryption.Uint64ToBytes( user_id ) , account )
		if errors.Is( tx_err , db.ErrNotFound ) { return nil }
		if tx_err != nil { return }
		change( account )
		tx_err = tx.Put( db.BucketGoogleAccounts , encryption.Uint64ToBytes( user_id ) , account )
		return
	} )
	return
}

// DeleteGoogleAccount forgets the connection, every card's link to an event,
// and every board's sync choice for this person.
func DeleteGoogleAccount( store *db.Store , user_id uint64 ) ( err error ) {
	err = store.Write( func( tx *db.Tx ) ( tx_err error ) {
		if tx_err = tx.Delete( db.BucketGoogleAccounts , encryption.Uint64ToBytes( user_id ) ); tx_err != nil && errors.Is( tx_err , db.ErrNotFound ) == false { return }
		tx_err = nil
		if _ , tx_err = tx.DeletePrefix( db.BucketGoogleLinks , db.Key( user_id ) ); tx_err != nil { return }
		refs := map[string]*BoardRef{}
		tx_err = tx.ForEachPrefix( db.BucketUserBoards , db.Key( user_id ) , func() any { return &BoardRef{} } , func( key []byte , item any ) bool {
			if ref := item.( *BoardRef ); ref.GoogleSync { refs[ string( key ) ] = ref }
			return true
		} )
		if tx_err != nil { return }
		for key , ref := range refs {
			ref.GoogleSync = false
			if tx_err = tx.Put( db.BucketUserBoards , []byte( key ) , ref ); tx_err != nil { return }
		}
		return
	} )
	return
}

// GoogleAccountUsers is everyone with a connected calendar.
func GoogleAccountUsers( store *db.Store ) ( user_ids []uint64 , err error ) {
	err = store.ForEach( db.BucketGoogleAccounts , func() any { return &GoogleAccount{} } , func( key []byte , item any ) bool {
		user_ids = append( user_ids , encryption.BytesToUint64( key ) )
		return true
	} )
	return
}

// SetBoardGoogleSync stores whether this person syncs the board to their
// calendar.
func SetBoardGoogleSync( store *db.Store , user_id uint64 , board_id uint64 , on bool ) ( err error ) {
	err = updateBoardRef( store , user_id , board_id , func( ref *BoardRef ) { ref.GoogleSync = on } )
	return
}

// GoogleSyncBoards is the boards this person syncs to their calendar.
func GoogleSyncBoards( store *db.Store , user_id uint64 ) ( board_ids []uint64 , err error ) {
	err = store.ForEachPrefix( db.BucketUserBoards , db.Key( user_id ) , func() any { return &BoardRef{} } , func( key []byte , item any ) bool {
		if item.( *BoardRef ).GoogleSync && len( key ) == 16 { board_ids = append( board_ids , encryption.BytesToUint64( key[ 8: ] ) ) }
		return true
	} )
	return
}

// GoogleLinks is every card of a board linked to an event for this person,
// by card id.
func GoogleLinks( store *db.Store , user_id uint64 , board_id uint64 ) ( links map[uint64]*GoogleLink , err error ) {
	links = map[uint64]*GoogleLink{}
	err = store.ForEachPrefix( db.BucketGoogleLinks , db.Key( user_id , board_id ) , func() any { return &GoogleLink{} } , func( key []byte , item any ) bool {
		if len( key ) == 24 { links[ encryption.BytesToUint64( key[ 16: ] ) ] = item.( *GoogleLink ) }
		return true
	} )
	return
}

func GetGoogleLink( store *db.Store , user_id uint64 , board_id uint64 , card_id uint64 ) ( link *GoogleLink , err error ) {
	found := &GoogleLink{}
	err = store.Get( db.BucketGoogleLinks , db.Key( user_id , board_id , card_id ) , found )
	if errors.Is( err , db.ErrNotFound ) {
		err = nil
		return
	}
	if err == nil { link = found }
	return
}

func PutGoogleLink( store *db.Store , user_id uint64 , board_id uint64 , card_id uint64 , link *GoogleLink ) ( err error ) {
	err = store.Put( db.BucketGoogleLinks , db.Key( user_id , board_id , card_id ) , link )
	return
}

func DeleteGoogleLink( store *db.Store , user_id uint64 , board_id uint64 , card_id uint64 ) ( err error ) {
	err = store.Delete( db.BucketGoogleLinks , db.Key( user_id , board_id , card_id ) )
	if errors.Is( err , db.ErrNotFound ) { err = nil }
	return
}
