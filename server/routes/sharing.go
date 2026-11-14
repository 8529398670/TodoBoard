package routes

import (
	fiber "github.com/gofiber/fiber/v3"

	access "todoboard/server/access"
	apidoc "todoboard/server/apidoc"
	models "todoboard/server/models"
)

// Sharing is the board owner's business (access.Manage): who is on the
// board, which single cards are shared with whom, and the share links.

type memberRequest struct {
	boardPath
	UserID uint64 `json:"user_id" jsonschema_description:"The account to add, change or remove, from list_team."`
	Role   string `json:"role" jsonschema:"enum=editor,enum=viewer,enum=" jsonschema_description:"editor or viewer; empty removes them from the board."`
}

type cardShareRequest struct {
	cardPath
	UserID uint64 `json:"user_id" jsonschema_description:"The account to share the card with, from list_team."`
	Mode   string `json:"mode" jsonschema:"enum=view,enum=edit,enum=" jsonschema_description:"view or edit; empty stops sharing the card with them."`
}

type shareLinkRequest struct {
	boardPath
	CardID uint64 `json:"card_id,omitempty"`
	ListID uint64 `json:"list_id,omitempty"`
	// A board link only: the columns whose cards it shows (empty: all).
	VisibleListIDs []uint64 `json:"visible_list_ids,omitempty"`
	Mode   string `json:"mode"`
	Label  string `json:"label,omitempty"`
	Days   int    `json:"days,omitempty"`
}

type shareLinkPath struct {
	boardPath
	LinkID string `json:"link_id" path:"link_id" jsonschema:"minLength=1" jsonschema_description:"A share link's id, from list_shares."`
}

// setBoardMember adds someone to the board, changes their role, or takes
// them off it.
//
// @route("POST /api/boards/{board_id}/members", summary="Add, change or remove a board member",
//     tag="Sharing", errors=(400, 404),
//     description="Put an account on the board as an editor or a viewer, change their role, or take them "
//         "off it (role empty). Only the board's owner may. The owner's own place cannot be changed.",
//     mcp="add_board_member",
// )
func ( handlers *Handlers ) setBoardMember( c fiber.Ctx , in *memberRequest ) ( reply *okReply , err error ) {
	principal , board , err := handlers.requireBoardAt( c , in.BoardID , access.Manage )
	if err != nil { return }
	if in.UserID == board.OwnerID {
		err = refuse( fiber.StatusBadRequest , "invalid_input" )
		return
	}
	was_member := board.RoleOf( in.UserID ) != ""
	if target , get_err := models.GetUser( handlers.Store , in.UserID ); get_err != nil || target.Disabled() {
		if in.Role != "" {
			err = refuse( fiber.StatusNotFound , "user_not_found" )
			return
		}
	}
	if _ , set_err := models.SetMember( handlers.Store , board.ID , in.UserID , in.Role ); set_err != nil {
		err = modelErr( set_err )
		return
	}
	if was_member == false && in.Role != "" {
		handlers.Notify.BoardShared( board.ID , in.UserID , principal.UserID , principal.Name )
	}
	if ( was_member == false ) != ( in.Role == "" ) {
		name := ""
		if target , get_err := models.GetUser( handlers.Store , in.UserID ); get_err == nil { name = target.DisplayName }
		what := "member_removed"
		if in.Role != "" { what = "member_added" }
		handlers.activity( principal , board.ID , what , map[string]string{ "name": name } )
	}
	reply = &okReply{ Ok: true }
	return
}

// shareCardWithUser shares one card with an account that is not on the
// board, or stops sharing it.
//
// @route("POST /api/boards/{board_id}/cards/{card_id}/share-user", summary="Share a card with someone",
//     tag="Sharing", errors=(404,),
//     description="Share one card with an account, to view or to edit, without putting them on the "
//         "board; mode empty stops sharing it with them. Only the board's owner may. Answers the card.",
//     mcp="share_card_with_user",
// )
func ( handlers *Handlers ) shareCardWithUser( c fiber.Ctx , in *cardShareRequest ) ( reply *fiber.Map , err error ) {
	principal , board , err := handlers.requireBoardAt( c , in.BoardID , access.Manage )
	if err != nil { return }
	if in.CardID == 0 {
		err = refuse( fiber.StatusNotFound , "card_not_found" )
		return
	}
	before , get_err := models.GetCard( handlers.Store , board.ID , in.CardID )
	if get_err != nil {
		err = modelErr( get_err )
		return
	}
	if target , user_err := models.GetUser( handlers.Store , in.UserID ); user_err != nil || ( target.Disabled() && in.Mode != "" ) {
		err = refuse( fiber.StatusNotFound , "user_not_found" )
		return
	}
	card , share_err := models.ShareCardWithUser( handlers.Store , board.ID , in.CardID , in.UserID , in.Mode , principal.Name )
	if share_err != nil {
		err = modelErr( share_err )
		return
	}
	if in.Mode != "" && before.SharedWith[ in.UserID ] == "" && board.RoleOf( in.UserID ) == "" {
		handlers.Notify.CardShared( board.ID , card.ID , in.UserID , principal.UserID , principal.Name )
	}
	payload := cardPayload( card , access.Card( principal , board , card ) , access.Manage )
	reply = &payload
	return
}

// shareLinkPayload never includes the secret: it is shown once, when the
// link is created, and only its hash is kept.
func shareLinkPayload( link *models.ShareLink ) ( payload fiber.Map ) {
	payload = fiber.Map{
		"id":           link.ID,
		"card_id":      link.CardID,
		"list_id":      link.ListID,
		"visible_list_ids": link.VisibleListIDs,
		"quick":        link.Quick,
		"mode":         link.Mode,
		"label":        link.Label,
		"created_at":   link.CreatedAt,
		"expires_at":   link.ExpiresAt,
		"revoked_at":   link.RevokedAt,
		"last_used_at": link.LastUsedAt,
		"live":         link.Live(),
	}
	return
}

// listShareLinks is every share link of the board, live or not.
//
// @route("GET /api/boards/{board_id}/shares", summary="List a board's share links",
//     tag="Sharing",
//     description="The board's share links, live or not: id, what each opens (the board, a column or one "
//         "card), mode, label, when it was made, expires, was revoked and last used, whether it is a quick "
//         "link, and who made it. Never the link itself, which is shown once when it is made. Only the "
//         "board's owner may.",
//     mcp="list_shares", read_only=True,
// )
func ( handlers *Handlers ) listShareLinks( c fiber.Ctx , in *boardPath ) ( reply *[]fiber.Map , err error ) {
	_ , board , err := handlers.requireBoardAt( c , in.BoardID , access.Manage )
	if err != nil { return }
	links , list_err := models.ListShareLinks( handlers.Store , board.ID )
	if list_err != nil {
		err = internalFailure( list_err )
		return
	}
	result := []fiber.Map{}
	for _ , link := range links {
		row := shareLinkPayload( link )
		// Editors make quick links, and only the owner sees this list.
		if user , get_err := models.GetUser( handlers.Store , link.CreatedBy ); get_err == nil { row[ "created_by_name" ] = user.DisplayName }
		result = append( result , row )
	}
	reply = &result
	return
}

// createShareLink mints a credential, so -- like creating a user or an API
// key -- it needs a browser session. A leaked API key cannot hand the board
// out to the world.
//
// @route("POST /api/boards/{board_id}/shares", summary="Make a share link",
//     tag="Sharing", access=SESSION_ONLY,
//     description="Make a link that opens the board, one column (list_id) or one card (card_id) to anyone "
//         "holding it, to view or to edit, for days days (0: no end). share_path is the link, shown only "
//         "this once. Only the board's owner may, from a browser session.",
// )
func ( handlers *Handlers ) createShareLink( c fiber.Ctx , in *shareLinkRequest ) ( reply *fiber.Map , err error ) {
	principal , board , err := handlers.requireBoardAt( c , in.BoardID , access.Manage )
	if err != nil { return }
	credential , link , issue_err := models.IssueShareLink( handlers.Store , board.ID , in.CardID , in.ListID , in.VisibleListIDs , in.Mode , in.Label , in.Days , principal.UserID )
	if issue_err != nil {
		err = modelErr( issue_err )
		return
	}
	payload := shareLinkPayload( link )
	payload[ "share_path" ] = "/s/" + credential
	reply = &payload
	return
}

type sharePathReply struct {
	SharePath string `json:"share_path"`
}

// quickShareCard answers with the card's public link, for the board's owner
// and editors. Asked of the board, not the card: someone with just this card
// shared with them is not the one to publish it. A session only, because it
// hands out a credential, as createShareLink does.
//
// @route("POST /api/boards/{board_id}/cards/{card_id}/quick-share", summary="Get a card's public link",
//     tag="Sharing", access=SESSION_ONLY, errors=(404,),
//     description="The card's quick public link, made the first time it is asked for. For the board's "
//         "owner and editors, from a browser session.",
// )
func ( handlers *Handlers ) quickShareCard( c fiber.Ctx , in *cardPath ) ( reply *sharePathReply , err error ) {
	principal , board , err := handlers.requireBoardAt( c , in.BoardID , access.Edit )
	if err != nil { return }
	if in.CardID == 0 {
		err = refuse( fiber.StatusNotFound , "card_not_found" )
		return
	}
	credential , _ , share_err := models.QuickShareLink( handlers.Store , board.ID , in.CardID , principal.UserID )
	if share_err != nil {
		err = modelErr( share_err )
		return
	}
	reply = &sharePathReply{ SharePath: "/s/" + credential }
	return
}

func ( handlers *Handlers ) shareLinkOnBoard( link_id string , board *models.Board ) ( link *models.ShareLink , err error ) {
	found , get_err := models.GetShareLink( handlers.Store , link_id )
	if get_err != nil || found.BoardID != board.ID {
		err = refuse( fiber.StatusNotFound , "share_link_not_found" )
		return
	}
	link = found
	return
}

// revokeShareLink stops a share link working, keeping it in the list.
//
// @route("POST /api/boards/{board_id}/shares/{link_id}/revoke", summary="Revoke a share link",
//     tag="Sharing", errors=(404,),
//     description="Stop one of the board's share links working (link_id from list_shares). It stays in "
//         "the list, marked revoked. Only the board's owner may.",
//     mcp="revoke_share", destructive=True,
// )
func ( handlers *Handlers ) revokeShareLink( c fiber.Ctx , in *shareLinkPath ) ( reply *okReply , err error ) {
	_ , board , err := handlers.requireBoardAt( c , in.BoardID , access.Manage )
	if err != nil { return }
	link , err := handlers.shareLinkOnBoard( in.LinkID , board )
	if err != nil { return }
	if revoke_err := models.RevokeShareLink( handlers.Store , link.ID ); revoke_err != nil {
		err = modelErr( revoke_err )
		return
	}
	reply = &okReply{ Ok: true }
	return
}

// deleteShareLink removes a share link altogether.
//
// @route("POST /api/boards/{board_id}/shares/{link_id}/delete", summary="Delete a share link",
//     tag="Sharing", errors=(404,),
//     description="Remove one of the board's share links from the list (link_id from list_shares); a "
//         "live one stops working. Only the board's owner may.",
//     mcp="delete_share", destructive=True,
// )
func ( handlers *Handlers ) deleteShareLink( c fiber.Ctx , in *shareLinkPath ) ( reply *okReply , err error ) {
	_ , board , err := handlers.requireBoardAt( c , in.BoardID , access.Manage )
	if err != nil { return }
	link , err := handlers.shareLinkOnBoard( in.LinkID , board )
	if err != nil { return }
	if delete_err := models.DeleteShareLink( handlers.Store , link.ID ); delete_err != nil {
		err = modelErr( delete_err )
		return
	}
	reply = &okReply{ Ok: true }
	return
}

// sharedInfo tells a guest page what its link opens. It is the first call a
// share page makes.
//
// @route("GET /api/shared", summary="What a share link opens",
//     tag="Shared", access=PUBLIC, use=[handlers.SharePrincipal], errors=(404,),
//     description="For a guest holding a share link in X-Share-Token: the board (and its title), column "
//         "or card it opens, its mode and label, and whether guests may comment. A quick card link "
//         "answers board_id, card_id, mode and quick only.",
// )
func ( handlers *Handlers ) sharedInfo( c fiber.Ctx , in *apidoc.None ) ( reply *fiber.Map , err error ) {
	principal := principalFrom( c )
	if principal == nil || principal.Share == nil {
		err = refuse( fiber.StatusNotFound , "share_link_invalid" )
		return
	}
	link := principal.Share
	if link.Quick {
		reply = &fiber.Map{ "board_id": link.BoardID , "card_id": link.CardID , "mode": link.Mode , "quick": true }
		return
	}
	board , get_err := models.GetBoard( handlers.Store , link.BoardID )
	if get_err != nil {
		err = refuse( fiber.StatusNotFound , "share_link_invalid" )
		return
	}
	list_title := ""
	if list := board.List( link.ListID ); list != nil { list_title = list.Title }
	reply = &fiber.Map{
		"board_id":    link.BoardID,
		"board_title": board.Title,
		"card_id":     link.CardID,
		"list_id":     link.ListID,
		"list_title":  list_title,
		"mode":        link.Mode,
		"label":       link.Label,
		"anon_comments": board.AllowAnonComments,
	}
	return
}

// ShowSharePage answers /s/<credential> with the guest page. Like a login
// link, the credential is in the URL, so no request the page makes may carry
// it onward in a Referer header.
func ( handlers *Handlers ) ShowSharePage( c fiber.Ctx ) ( err error ) {
	c.Set( fiber.HeaderReferrerPolicy , "no-referrer" )
	c.Set( "X-Robots-Tag" , "noindex" )
	err = handlers.Static.ServeFile( c , "board.html" )
	return
}

func ( handlers *Handlers ) ShowBoardPage( c fiber.Ctx ) ( err error ) {
	err = handlers.Static.ServeFile( c , "board.html" )
	return
}

func ( handlers *Handlers ) ShowPrintPage( c fiber.Ctx ) ( err error ) {
	c.Set( fiber.HeaderReferrerPolicy , "no-referrer" )
	c.Set( "X-Robots-Tag" , "noindex" )
	err = handlers.Static.ServeFile( c , "print.html" )
	return
}
