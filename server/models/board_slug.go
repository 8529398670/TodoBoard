package models

import (
	errors "errors"
	fmt "fmt"
	regexp "regexp"
	strings "strings"
	unicode "unicode"

	norm "golang.org/x/text/unicode/norm"

	db "todoboard/server/db"
)

// A board's URL is /b/<slug>. Slugs are unique across the server, kept in
// BucketBoardSlugs (keyed by a hash, see db.LookupKey) in the same
// transaction as the board, and never all digits -- /b/<number> is still
// the board ID, which is what older links and notifications carry.

var ErrSlugTaken = errors.New( "slug taken" )

const maxSlugLength = 60

var slugPattern = regexp.MustCompile( `^[a-z0-9]+(?:-[a-z0-9]+)*$` )
var digitsPattern = regexp.MustCompile( `^[0-9]+$` )

type slugRef struct {
	BoardID uint64 `json:"board_id"`
}

func ValidSlug( slug string ) ( result bool ) {
	result = len( slug ) <= maxSlugLength && slugPattern.MatchString( slug ) && digitsPattern.MatchString( slug ) == false
	return
}

// Slugify makes a slug from a title: accents dropped, everything that is not
// a letter or digit collapsed to single dashes. A title with nothing usable
// in it ("✓✓✓") becomes "board".
func Slugify( title string ) ( slug string ) {
	var builder strings.Builder
	dash := false
	for _ , r := range norm.NFKD.String( strings.ToLower( title ) ) {
		switch {
		case unicode.Is( unicode.Mn , r ):
			continue
		case r < 128 && ( unicode.IsLetter( r ) || unicode.IsDigit( r ) ):
			if dash && builder.Len() > 0 { builder.WriteByte( '-' ) }
			builder.WriteRune( r )
			dash = false
		default:
			dash = true
		}
	}
	slug = builder.String()
	if len( slug ) > maxSlugLength { slug = strings.Trim( slug[ :maxSlugLength ] , "-" ) }
	if slug == "" || digitsPattern.MatchString( slug ) { slug = "board-" + slug }
	slug = strings.Trim( slug , "-" )
	return
}

func slugOwnerTx( tx *db.Tx , slug string ) ( board_id uint64 ) {
	ref := &slugRef{}
	if tx.Get( db.BucketBoardSlugs , tx.LookupKey( slug ) , ref ) == nil { board_id = ref.BoardID }
	return
}

// claimSlugTx points slug at board, releasing the board's previous slug.
// Refuses a slug held by another board.
func claimSlugTx( tx *db.Tx , board *Board , slug string ) ( err error ) {
	if ValidSlug( slug ) == false { return ErrBadInput }
	if slug == board.Slug { return }
	if owner := slugOwnerTx( tx , slug ); owner != 0 && owner != board.ID { return ErrSlugTaken }
	if board.Slug != "" {
		if err = tx.Delete( db.BucketBoardSlugs , tx.LookupKey( board.Slug ) ); err != nil { return }
	}
	if err = tx.Put( db.BucketBoardSlugs , tx.LookupKey( slug ) , &slugRef{ BoardID: board.ID } ); err != nil { return }
	board.Slug = slug
	return
}

// claimFreeSlugTx gives board the first free one of base, base-2, base-3...
func claimFreeSlugTx( tx *db.Tx , board *Board , base string ) ( err error ) {
	for attempt := 1; attempt < 1000; attempt += 1 {
		candidate := base
		if attempt > 1 {
			suffix := fmt.Sprintf( "-%d" , attempt )
			if len( candidate ) + len( suffix ) > maxSlugLength { candidate = strings.Trim( candidate[ :maxSlugLength - len( suffix ) ] , "-" ) }
			candidate += suffix
		}
		err = claimSlugTx( tx , board , candidate )
		if errors.Is( err , ErrSlugTaken ) == false { return }
	}
	return
}

// BoardIDForSlug resolves a slug, or 0 when nothing holds it.
func BoardIDForSlug( store *db.Store , slug string ) ( board_id uint64 ) {
	if ValidSlug( slug ) == false { return }
	ref := &slugRef{}
	if store.Get( db.BucketBoardSlugs , store.LookupKey( slug ) , ref ) == nil { board_id = ref.BoardID }
	return
}

// EnsureBoardSlugs gives every board written before slugs existed one made
// from its title. Run at startup; a no-op once every board has one.
func EnsureBoardSlugs( store *db.Store ) ( err error ) {
	missing := []uint64{}
	err = store.ForEach( db.BucketBoards , func() any { return &Board{} } , func( key []byte , item any ) bool {
		if board := item.( *Board ); board.Slug == "" { missing = append( missing , board.ID ) }
		return true
	} )
	if err != nil || len( missing ) == 0 { return }
	err = store.Write( func( tx *db.Tx ) ( tx_err error ) {
		for _ , board_id := range missing {
			board , get_err := getBoardTx( tx , board_id )
			if get_err != nil { continue }
			if tx_err = claimFreeSlugTx( tx , board , Slugify( board.Title ) ); tx_err != nil { return }
			// Written directly, not through putBoardTx: assigning a slug is
			// not an edit anyone made, so it should not bump Rev.
			if tx_err = tx.Put( db.BucketBoards , boardKey( board.ID ) , board ); tx_err != nil { return }
		}
		return
	} )
	return
}
