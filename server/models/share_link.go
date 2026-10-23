package models

import (
	errors "errors"
	sort "sort"
	strings "strings"
	time "time"
	utf8 "unicode/utf8"

	db "todoboard/server/db"
	encryption "todoboard/server/encryption"
)

// ShareLink lets anyone holding it open a board, one column, or one card,
// without an account -- read-only or editable. It is a bearer credential like an API
// key, shaped and checked the same way: "<id>.<secret>", the id is the bolt
// key, and only a sha256 of the 256-bit secret is stored, so a copy of the
// database does not hand out working links.
//
// What a link can never do is manage the board: add members, see or mint
// other links, or delete anything but cards and lists. That ceiling lives in
// server/access, not here.
type ShareLink struct {
	ID         string     `json:"id"`
	SecretHash string     `json:"secret_hash"`
	BoardID    uint64     `json:"board_id"`
	CardID     uint64     `json:"card_id"` // 0: the whole board (or a column)
	ListID     uint64     `json:"list_id"` // 0: not a column link
	Mode       string     `json:"mode"`
	Label      string     `json:"label"`
	CreatedBy  uint64     `json:"created_by"`
	CreatedAt  time.Time  `json:"created_at"`
	ExpiresAt  *time.Time `json:"expires_at,omitempty"`
	RevokedAt  *time.Time `json:"revoked_at,omitempty"`
	LastUsedAt *time.Time `json:"last_used_at,omitempty"`
}

var ErrShareLinkNotFound = errors.New( "share link not found" )

const maxShareLinksPerBoard = 100

func ( link *ShareLink ) Live() ( result bool ) {
	if link.RevokedAt != nil { return }
	if link.ExpiresAt != nil && time.Now().After( *link.ExpiresAt ) { return }
	result = true
	return
}

// IssueShareLink mints a link for the board, one card (card_id), or one
// column (list_id) -- never both of the last two. days 0 means it does not
// expire.
func IssueShareLink( store *db.Store , board_id uint64 , card_id uint64 , list_id uint64 , mode string , label string , days int , created_by uint64 ) ( credential string , link *ShareLink , err error ) {
	label = strings.TrimSpace( label )
	if ( mode != ModeView && mode != ModeEdit ) || utf8.RuneCountInString( label ) > 80 || days < 0 || days > 3650 || ( card_id != 0 && list_id != 0 ) {
		err = ErrBadInput
		return
	}
	link_id := encryption.GenerateURLToken( 12 )
	secret := encryption.GenerateURLToken( 32 )
	now := time.Now().UTC()
	link = &ShareLink{
		ID: link_id , SecretHash: encryption.Sha256Hex( secret ) , BoardID: board_id , CardID: card_id , ListID: list_id ,
		Mode: mode , Label: label , CreatedBy: created_by , CreatedAt: now ,
	}
	if days > 0 {
		expires := now.Add( time.Duration( days ) * 24 * time.Hour )
		link.ExpiresAt = &expires
	}
	err = store.Write( func( tx *db.Tx ) ( tx_err error ) {
		board , tx_err := getBoardTx( tx , board_id )
		if tx_err != nil { return }
		if list_id != 0 && board.List( list_id ) == nil { return ErrListNotFound }
		if card_id != 0 {
			if _ , tx_err = getCardTx( tx , board_id , card_id ); tx_err != nil { return }
		}
		count := 0
		tx_err = tx.ForEach( db.BucketShareLinks , func() any { return &ShareLink{} } ,
			func( key []byte , item any ) bool {
				if item.( *ShareLink ).BoardID == board_id { count += 1 }
				return true
			} )
		if tx_err != nil { return }
		if count >= maxShareLinksPerBoard { return ErrBadInput }
		tx_err = tx.Put( db.BucketShareLinks , []byte( link_id ) , link )
		return
	} )
	if err != nil {
		link = nil
		return
	}
	credential = link_id + "." + secret
	return
}

// LoadShareLink returns nil for anything that is not a live link -- wrong
// shape, unknown, wrong secret, expired, or revoked -- without saying which.
func LoadShareLink( store *db.Store , credential string ) ( link *ShareLink ) {
	link_id , secret , ok := splitCredential( strings.TrimSpace( credential ) )
	if ok == false { return }
	found := &ShareLink{}
	if store.Get( db.BucketShareLinks , []byte( link_id ) , found ) != nil { return }
	if encryption.ConstantTimeEqual( found.SecretHash , encryption.Sha256Hex( secret ) ) == false { return }
	if found.Live() == false { return }
	touchShareLink( store , found )
	link = found
	return
}

func touchShareLink( store *db.Store , link *ShareLink ) {
	now := time.Now().UTC()
	if link.LastUsedAt != nil && now.Sub( *link.LastUsedAt ) < 10 * time.Minute { return }
	store.UpdateValue( db.BucketShareLinks , []byte( link.ID ) , func() any { return &ShareLink{} } ,
		func( item any ) error {
			item.( *ShareLink ).LastUsedAt = &now
			return nil
		} )
}

func ListShareLinks( store *db.Store , board_id uint64 ) ( links []*ShareLink , err error ) {
	links = []*ShareLink{}
	err = store.ForEach( db.BucketShareLinks , func() any { return &ShareLink{} } ,
		func( key []byte , item any ) bool {
			if link := item.( *ShareLink ); link.BoardID == board_id { links = append( links , link ) }
			return true
		} )
	sort.Slice( links , func( i int , j int ) bool { return links[ i ].CreatedAt.After( links[ j ].CreatedAt ) } )
	return
}

func GetShareLink( store *db.Store , link_id string ) ( link *ShareLink , err error ) {
	link = &ShareLink{}
	if err = store.Get( db.BucketShareLinks , []byte( link_id ) , link ); err != nil {
		link = nil
		if errors.Is( err , db.ErrNotFound ) { err = ErrShareLinkNotFound }
	}
	return
}

func RevokeShareLink( store *db.Store , link_id string ) ( err error ) {
	now := time.Now().UTC()
	err = store.UpdateValue( db.BucketShareLinks , []byte( link_id ) , func() any { return &ShareLink{} } ,
		func( item any ) error {
			link := item.( *ShareLink )
			if link.RevokedAt == nil { link.RevokedAt = &now }
			return nil
		} )
	if errors.Is( err , db.ErrNotFound ) { err = ErrShareLinkNotFound }
	return
}

func DeleteShareLink( store *db.Store , link_id string ) ( err error ) {
	err = store.Delete( db.BucketShareLinks , []byte( link_id ) )
	return
}
