package models

import (
	hex "encoding/hex"
	errors "errors"
	strings "strings"
	time "time"
	utf8 "unicode/utf8"

	db "todoboard/server/db"
	encryption "todoboard/server/encryption"
)

// PushSubscription is one browser that takes this person's notifications:
// where to send them and the keys to seal them with. The endpoint is a
// credential of sorts, since anyone holding it and the keys can reach that
// browser, so it is never sent back out.
type PushSubscription struct {
	ID          string     `json:"id"`
	UserID      uint64     `json:"user_id"`
	Endpoint    string     `json:"endpoint"`
	P256dh      string     `json:"p256dh"`
	Auth        string     `json:"auth"`
	Label       string     `json:"label"`
	CreatedAt   time.Time  `json:"created_at"`
	LastSuccess *time.Time `json:"last_success,omitempty"`
}

// VAPIDKeys is this server's identity to the push services. Browsers
// subscribe against the public half, so losing these strands every
// subscription: they live in the database, with what depends on them, and not
// in config, which Docker mounts read-only.
type VAPIDKeys struct {
	Private string `json:"private"`
	Public  string `json:"public"`
}

const metaVAPIDKey = "vapid_keys"

// maxPushSubscriptions is per person. Nobody has more browsers than this, and
// each is another request for every notification.
const maxPushSubscriptions = 10

const maxPushEndpointBytes = 1024

var ErrTooManyDevices = errors.New( "too many devices" )

// EnsureVAPIDKeys returns the server's keys, making them the first time.
func EnsureVAPIDKeys( store *db.Store ) ( keys *VAPIDKeys , err error ) {
	err = store.Write( func( tx *db.Tx ) ( tx_err error ) {
		found := &VAPIDKeys{}
		tx_err = tx.Get( db.BucketMeta , []byte( metaVAPIDKey ) , found )
		if tx_err == nil {
			keys = found
			return
		}
		if errors.Is( tx_err , db.ErrNotFound ) == false { return }
		private_key , public_key , tx_err := encryption.GenerateVAPIDKeys()
		if tx_err != nil { return }
		keys = &VAPIDKeys{ Private: private_key , Public: public_key }
		tx_err = tx.Put( db.BucketMeta , []byte( metaVAPIDKey ) , keys )
		return
	} )
	if err != nil { keys = nil }
	return
}

func pushKey( user_id uint64 , id []byte ) ( key []byte ) {
	key = append( db.Key( user_id ) , id... )
	return
}

// SavePushSubscription adds a browser, or refreshes one already here. One
// browser belongs to one person: when someone else signs in on it, the old
// subscription goes, or they would keep getting the last person's news.
func SavePushSubscription( store *db.Store , user_id uint64 , endpoint string , p256dh string , auth string , label string ) ( subscription *PushSubscription , err error ) {
	label = strings.TrimSpace( label )
	if utf8.RuneCountInString( label ) > 80 || len( endpoint ) > maxPushEndpointBytes || encryption.ValidWebPushKeys( p256dh , auth ) == false {
		return nil , ErrBadInput
	}
	lookup := store.LookupKey( endpoint )
	id := hex.EncodeToString( lookup )
	err = store.Write( func( tx *db.Tx ) ( tx_err error ) {
		stale := [][]byte{}
		mine := 0
		var existing *PushSubscription
		tx_err = tx.ForEach( db.BucketPushSubscriptions , func() any { return &PushSubscription{} } ,
			func( key []byte , item any ) bool {
				found := item.( *PushSubscription )
				if found.ID == id && found.UserID != user_id { stale = append( stale , append( []byte( nil ) , key... ) ) }
				if found.UserID == user_id {
					mine += 1
					if found.ID == id { existing = found }
				}
				return true
			} )
		if tx_err != nil { return }
		for _ , key := range stale {
			if tx_err = tx.Delete( db.BucketPushSubscriptions , key ); tx_err != nil { return }
		}
		if existing == nil && mine >= maxPushSubscriptions { return ErrTooManyDevices }
		subscription = &PushSubscription{ ID: id , UserID: user_id , Endpoint: endpoint , P256dh: p256dh , Auth: auth , Label: label , CreatedAt: time.Now().UTC() }
		if existing != nil {
			subscription.CreatedAt , subscription.LastSuccess = existing.CreatedAt , existing.LastSuccess
		}
		tx_err = tx.Put( db.BucketPushSubscriptions , pushKey( user_id , lookup ) , subscription )
		return
	} )
	if err != nil { subscription = nil }
	return
}

func ListPushSubscriptions( store *db.Store , user_id uint64 ) ( subscriptions []*PushSubscription , err error ) {
	subscriptions = []*PushSubscription{}
	err = store.ForEachPrefix( db.BucketPushSubscriptions , db.Key( user_id ) , func() any { return &PushSubscription{} } ,
		func( key []byte , item any ) bool {
			subscriptions = append( subscriptions , item.( *PushSubscription ) )
			return true
		} )
	return
}

func DeletePushSubscription( store *db.Store , user_id uint64 , id string ) ( err error ) {
	lookup , decode_err := hex.DecodeString( id )
	if decode_err != nil { return ErrBadInput }
	err = store.Delete( db.BucketPushSubscriptions , pushKey( user_id , lookup ) )
	return
}

// MarkPushDelivered notes that a browser took a notification. Only once an
// hour: it is for showing "last reached" and every write costs a disk sync.
func MarkPushDelivered( store *db.Store , subscription *PushSubscription , now time.Time ) ( err error ) {
	if subscription.LastSuccess != nil && now.Sub( *subscription.LastSuccess ) < time.Hour { return }
	lookup , decode_err := hex.DecodeString( subscription.ID )
	if decode_err != nil { return decode_err }
	key := pushKey( subscription.UserID , lookup )
	err = store.Write( func( tx *db.Tx ) ( tx_err error ) {
		current := &PushSubscription{}
		if tx_err = tx.Get( db.BucketPushSubscriptions , key , current ); tx_err != nil {
			if errors.Is( tx_err , db.ErrNotFound ) { tx_err = nil }
			return
		}
		stamp := now.UTC()
		current.LastSuccess = &stamp
		tx_err = tx.Put( db.BucketPushSubscriptions , key , current )
		return
	} )
	return
}
