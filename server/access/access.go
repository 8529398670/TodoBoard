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
//     board or for one card, and it can never manage the board.
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
// board settings. Someone with access to a single card only gets None here --
// see Card.
func Board( principal *Principal , board *models.Board ) ( level Level ) {
	if principal == nil || board == nil { return }
	if principal.Share != nil {
		link := principal.Share
		if link.BoardID != board.ID || link.CardID != 0 || link.Live() == false { return }
		level = modeLevel( link.Mode )
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

// Card is the principal's level on one card: the better of their board-wide
// level (capped at Edit -- managing is a board thing) and anything granted on
// this card alone.
func Card( principal *Principal , board *models.Board , card *models.Card ) ( level Level ) {
	if principal == nil || board == nil || card == nil || card.BoardID != board.ID { return }
	level = Board( principal , board )
	if level > Edit { level = Edit }
	var direct Level
	if principal.Share != nil {
		link := principal.Share
		if link.BoardID == board.ID && link.CardID == card.ID && link.Live() { direct = modeLevel( link.Mode ) }
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
	result = Board( principal , board ) == None
	return
}
