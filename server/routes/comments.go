package routes

import (
	strings "strings"
	utf8 "unicode/utf8"

	fiber "github.com/gofiber/fiber/v3"

	access "todoboard/server/access"
	models "todoboard/server/models"
)

// A card's comments. Anyone who can read the card reads them, except on a
// card's public link, which shows the card and nothing else. Posting is for
// those access.CanComment allows; a guest who gave no name posts only where
// the board allows anonymous comments. A member edits and removes their own;
// the owner removes anyone's. A guest's comment is final once posted, since
// nothing ties the next request to the guest who wrote it.

// commentPayload adds what the page needs to offer edit and remove.
func commentPayload( principal *access.Principal , board *models.Board , comment *models.Comment ) ( payload fiber.Map ) {
	payload = fiber.Map{
		"id":         comment.ID,
		"card_id":    comment.CardID,
		"author_id":  comment.AuthorID,
		"by":         comment.By,
		"text":       comment.Text,
		"created_at": comment.CreatedAt,
		"edited_at":  comment.EditedAt,
		"can_edit":   canEditComment( principal , comment ),
		"can_remove": canRemoveComment( principal , board , comment ),
	}
	return
}

func canEditComment( principal *access.Principal , comment *models.Comment ) ( result bool ) {
	result = principal.IsGuest() == false && comment.AuthorID != 0 && comment.AuthorID == principal.UserID
	return
}

func canRemoveComment( principal *access.Principal , board *models.Board , comment *models.Comment ) ( result bool ) {
	result = canEditComment( principal , comment ) || access.Board( principal , board ) == access.Manage
	return
}

type commentRequest struct {
	cardPath
	Text string `json:"text" jsonschema:"maxLength=4000" jsonschema_description:"Markdown, up to 4000 characters."`
}

type editCommentRequest struct {
	commentPath
	Text string `json:"text" jsonschema:"maxLength=4000" jsonschema_description:"Markdown, up to 4000 characters; replaces the comment's text."`
}

type commentsReply struct {
	Comments   []fiber.Map `json:"comments" jsonschema_description:"Oldest first."`
	CanComment bool        `json:"can_comment" jsonschema_description:"Whether the caller may post one."`
}

// commentCard loads a card for reading its comments, refusing a public card
// link as if the comments were not there.
func ( handlers *Handlers ) commentCard( c fiber.Ctx , path *cardPath ) ( principal *access.Principal , board *models.Board , card *models.Card , err error ) {
	principal , board , card , err = handlers.requireCardAt( c , path , access.View )
	if err != nil { return }
	if access.QuickLink( principal ) {
		board , card = nil , nil
		err = refuse( fiber.StatusForbidden , "forbidden" )
	}
	return
}

// listComments is a card's comments, oldest first.
//
// @route("GET /api/boards/{board_id}/cards/{card_id}/comments", summary="Read a card's comments",
//     tag="Comments", errors=(403, 404),
//     description="A card's comments, oldest first: who wrote each (by: a member's name, guest:Name for a "
//         "guest, guest: alone for an anonymous one) and when, and whether you may post one (can_comment).",
//     mcp="list_comments", read_only=True,
// )
func ( handlers *Handlers ) listComments( c fiber.Ctx , in *cardPath ) ( reply *commentsReply , err error ) {
	principal , board , card , err := handlers.commentCard( c , in )
	if err != nil { return }
	comments , list_err := models.ListComments( handlers.Store , board.ID , card.ID )
	if list_err != nil {
		err = list_err
		return
	}
	rows := []fiber.Map{}
	for _ , comment := range comments { rows = append( rows , commentPayload( principal , board , comment ) ) }
	reply = &commentsReply{ Comments: rows , CanComment: access.CanComment( principal , board , card ) }
	return
}

// sharedListComments is listComments for a share link's guest.
//
// @route("GET /api/shared/{board_id}/cards/{card_id}/comments", summary="Read a card's comments (share link)",
//     tag="Shared", access=PUBLIC, use=[handlers.SharePrincipal], errors=(403, 404),
//     description="listComments for a guest holding a share link in X-Share-Token.",
// )
func ( handlers *Handlers ) sharedListComments( c fiber.Ctx , in *cardPath ) ( reply *commentsReply , err error ) {
	reply , err = handlers.listComments( c , in )
	return
}

// addComment posts a comment on a card.
//
// @route("POST /api/boards/{board_id}/cards/{card_id}/comments", summary="Comment on a card",
//     tag="Comments", errors=(400, 403, 404, 409),
//     description="Post a comment on a card, signed with your name. Markdown, up to 4000 characters. Needs "
//         "edit rights on the card, or a card shared with you to comment on.",
//     mcp="add_comment",
// )
func ( handlers *Handlers ) addComment( c fiber.Ctx , in *commentRequest ) ( reply *fiber.Map , err error ) {
	principal , board , card , err := handlers.commentCard( c , &in.cardPath )
	if err != nil { return }
	comment , refused := handlers.addCommentRule( principal , board , card , in.Text )
	if refused != nil {
		err = refused
		return
	}
	payload := commentPayload( principal , board , comment )
	reply = &payload
	return
}

// sharedAddComment is addComment for a share link's guest.
//
// @route("POST /api/shared/{board_id}/cards/{card_id}/comments", summary="Comment on a card (share link)",
//     tag="Shared", access=PUBLIC, use=[handlers.SharePrincipal], errors=(400, 403, 404, 409),
//     description="addComment for a guest holding a share link in X-Share-Token.",
// )
func ( handlers *Handlers ) sharedAddComment( c fiber.Ctx , in *commentRequest ) ( reply *fiber.Map , err error ) {
	reply , err = handlers.addComment( c , in )
	return
}

// addCommentRule is posting a comment, once the card is loaded.
func ( handlers *Handlers ) addCommentRule( principal *access.Principal , board *models.Board , card *models.Card , text string ) ( comment *models.Comment , refused *refusal ) {
	if access.CanComment( principal , board , card ) == false {
		refused = writeRefusal( board , access.Edit )
		return
	}
	name , named := access.CommentName( principal , board )
	if named == false {
		refused = refuse( fiber.StatusForbidden , "comment_name_required" )
		return
	}
	author_id := uint64( 0 )
	if principal.IsGuest() == false { author_id = principal.UserID }
	comment , add_err := models.AddComment( handlers.Store , board.ID , card.ID , author_id , name , text )
	if add_err != nil {
		refused = modelRefusal( add_err )
		return
	}
	handlers.Notify.CardChanged( board.ID , card.ID , models.CardChange{ Commented: true , Comment: commentSnippet( comment.Text ) , Title: card.Title } , author_id , name )
	return
}

// commentSnippet is the first line of a comment, cut to fit a notice.
func commentSnippet( text string ) ( result string ) {
	result = strings.TrimSpace( strings.SplitN( text , "\n" , 2 )[ 0 ] )
	if utf8.RuneCountInString( result ) > 80 { result = string( []rune( result )[ :79 ] ) + "…" }
	return
}

// loadComment is a comment on a card the principal can read.
func ( handlers *Handlers ) loadComment( c fiber.Ctx , path *commentPath ) ( principal *access.Principal , board *models.Board , comment *models.Comment , err error ) {
	principal , board , card , err := handlers.commentCard( c , &path.cardPath )
	if err != nil { return }
	comment , get_err := models.GetComment( handlers.Store , board.ID , card.ID , path.CommentID )
	if get_err != nil {
		board , comment = nil , nil
		err = refuse( fiber.StatusNotFound , "comment_not_found" )
	}
	return
}

// editComment changes the text of one of the caller's own comments.
//
// @route("POST /api/boards/{board_id}/cards/{card_id}/comments/{comment_id}", summary="Edit a comment",
//     tag="Comments", errors=(400, 403, 404),
//     description="Change the text of a comment you wrote. Comment ids are in list_comments.",
//     mcp="edit_comment",
// )
func ( handlers *Handlers ) editComment( c fiber.Ctx , in *editCommentRequest ) ( reply *fiber.Map , err error ) {
	principal , board , comment , err := handlers.loadComment( c , &in.commentPath )
	if err != nil { return }
	if canEditComment( principal , comment ) == false {
		err = refuse( fiber.StatusForbidden , "forbidden" )
		return
	}
	updated , edit_err := models.EditComment( handlers.Store , board.ID , comment.CardID , comment.ID , in.Text )
	if edit_err != nil {
		err = modelErr( edit_err )
		return
	}
	payload := commentPayload( principal , board , updated )
	reply = &payload
	return
}

// sharedEditComment is editComment for a share link's guest.
//
// @route("POST /api/shared/{board_id}/cards/{card_id}/comments/{comment_id}", summary="Edit a comment (share link)",
//     tag="Shared", access=PUBLIC, use=[handlers.SharePrincipal], errors=(400, 403, 404),
//     description="editComment for a guest holding a share link in X-Share-Token.",
// )
func ( handlers *Handlers ) sharedEditComment( c fiber.Ctx , in *editCommentRequest ) ( reply *fiber.Map , err error ) {
	reply , err = handlers.editComment( c , in )
	return
}

// removeComment deletes a comment: the caller's own, or anyone's for the
// board's owner.
//
// @route("POST /api/boards/{board_id}/cards/{card_id}/comments/{comment_id}/remove", summary="Remove a comment",
//     tag="Comments", errors=(400, 403, 404),
//     description="Delete a comment for good: one you wrote, or anyone's on a board you manage. This cannot be undone.",
//     mcp="remove_comment", destructive=True,
// )
func ( handlers *Handlers ) removeComment( c fiber.Ctx , in *commentPath ) ( reply *okReply , err error ) {
	principal , board , comment , err := handlers.loadComment( c , in )
	if err != nil { return }
	if canRemoveComment( principal , board , comment ) == false {
		err = refuse( fiber.StatusForbidden , "forbidden" )
		return
	}
	if remove_err := models.RemoveComment( handlers.Store , board.ID , comment.CardID , comment.ID ); remove_err != nil {
		err = modelErr( remove_err )
		return
	}
	reply = &okReply{ Ok: true }
	return
}

// sharedRemoveComment is removeComment for a share link's guest.
//
// @route("POST /api/shared/{board_id}/cards/{card_id}/comments/{comment_id}/remove", summary="Remove a comment (share link)",
//     tag="Shared", access=PUBLIC, use=[handlers.SharePrincipal], errors=(400, 403, 404),
//     description="removeComment for a guest holding a share link in X-Share-Token.",
// )
func ( handlers *Handlers ) sharedRemoveComment( c fiber.Ctx , in *commentPath ) ( reply *okReply , err error ) {
	reply , err = handlers.removeComment( c , in )
	return
}
