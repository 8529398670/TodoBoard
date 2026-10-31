// Package access is the only place that decides what someone may do with a
// board or a card. Every board and card handler asks here; none of them looks
// at roles, members, or share links itself. One function to audit is the
// point -- the same reason all crypto lives in server/encryption.
//
// There are two kinds of caller and they meet here:
//
//   - a signed-in user (a session or an API key, which server/security has
//     already turned into one account), whose rights come from the board's
//     owner and member list and from cards shared with them directly, and
//   - a guest holding a share link, whose rights are exactly what the link
//     says and never more: a link can be "view" or "edit", for the whole
//     board, one column, or one card, and it can never manage the board.
//
// App admins get nothing extra. Admin rights in this app decide who may sign
// in, not whose boards they may read.
package access

import (
	models "todoboard/server/models"
)

type Level int

const (
	None   Level = iota
	View         // read
	Edit         // change content
	Manage       // members, share links, deleting the board: the owner only
)

func ( level Level ) String() ( result string ) {
	switch level {
	case View:
		result = "view"
	case Edit:
		result = "edit"
	case Manage:
		result = "manage"
	default:
		result = "none"
	}
	return
}

// Principal is who is asking. Exactly one of UserID and Share is set.
type Principal struct {
	UserID uint64
	Share  *models.ShareLink
	// Name is what an edit is attributed to: the account's display name, or
	// what a guest typed, marked as a guest by the caller.
	Name string
}

func ( principal *Principal ) IsGuest() ( result bool ) {
	result = principal.Share != nil
	return
}

func modeLevel( mode string ) ( level Level ) {
	switch mode {
	case models.ModeEdit:
		level = Edit
	case models.ModeView:
		level = View
	}
	return
}

// Board is the principal's level on the board as a whole: lists, every card,
// board settings. Someone with access to a single column or card only gets
// None here -- see List and Card.
func Board( principal *Principal , board *models.Board ) ( level Level ) {
	if principal == nil || board == nil { return }
	if principal.Share != nil {
		link := principal.Share
		if link.BoardID != board.ID || link.CardID != 0 || link.ListID != 0 || link.Live() == false { return }
		level = modeLevel( link.Mode )
		// A link that shows only some columns reads the board's shape but
		// changes nothing board-wide: its edit rights are per column (List).
		if link.Restricted() && level > View { level = View }
		return
	}
	switch board.RoleOf( principal.UserID ) {
	case models.BoardRoleOwner:
		level = Manage
	case models.BoardRoleEditor:
		level = Edit
	case models.BoardRoleViewer:
		level = View
	}
	return
}

// List is the principal's level on one column: their board-wide level (capped
// at Edit), or what a column link grants on its own column. Edit here means
// adding, editing, reordering, archiving and deleting cards in that column --
// not renaming or removing the column itself, which stays a board edit.
func List( principal *Principal , board *models.Board , list_id uint64 ) ( level Level ) {
	if principal == nil || board == nil || board.List( list_id ) == nil { return }
	if HiddenList( principal , board , list_id ) { return }
	level = Board( principal , board )
	if level > Edit { level = Edit }
	if link := principal.Share; link != nil && link.BoardID == board.ID && link.Live() {
		if link.ListID == list_id || ( link.Restricted() && link.Shows( list_id ) ) {
			if direct := modeLevel( link.Mode ); direct > level { level = direct }
		}
	}
	return
}

// HiddenList reports whether the principal holds a board link that shows
// this column's title but not its cards.
func HiddenList( principal *Principal , board *models.Board , list_id uint64 ) ( result bool ) {
	if principal == nil || principal.Share == nil || board == nil { return }
	link := principal.Share
	if link.BoardID != board.ID || link.CardID != 0 || link.ListID != 0 || link.Live() == false { return }
	result = link.Shows( list_id ) == false
	return
}

// Card is the principal's level on one card: the better of their board-wide
// level (capped at Edit -- managing is a board thing), anything a column link
// grants on the card's column, and anything granted on this card alone.
func Card( principal *Principal , board *models.Board , card *models.Card ) ( level Level ) {
	if principal == nil || board == nil || card == nil || card.BoardID != board.ID { return }
	if HiddenList( principal , board , card.ListID ) { return }
	level = Board( principal , board )
	if level > Edit { level = Edit }
	var direct Level
	if principal.Share != nil {
		link := principal.Share
		if link.BoardID == board.ID && link.Live() && ( link.CardID == card.ID || ( link.ListID != 0 && link.ListID == card.ListID ) || ( link.Restricted() && link.Shows( card.ListID ) ) ) {
			direct = modeLevel( link.Mode )
		}
	} else {
		direct = modeLevel( card.SharedWith[ principal.UserID ] )
	}
	if direct > level { level = direct }
	return
}

// CardOnly reports whether the principal reaches this board only through one
// or more individual cards, which decides how much of the board they are
// shown.
func CardOnly( principal *Principal , board *models.Board ) ( result bool ) {
	result = Board( principal , board ) == None && ListOnly( principal , board ) == 0
	return
}

// ListOnly is the column a column link confines its holder to, or 0.
func ListOnly( principal *Principal , board *models.Board ) ( list_id uint64 ) {
	if principal == nil || principal.Share == nil || board == nil { return }
	link := principal.Share
	if link.BoardID != board.ID || link.ListID == 0 || link.Live() == false || board.List( link.ListID ) == nil { return }
	list_id = link.ListID
	return
}
