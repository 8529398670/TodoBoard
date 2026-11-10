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

type commentRequest struct {
	Text string `json:"text"`
}

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

// commentCard loads :card_id for reading its comments, refusing a public
// card link as if the comments were not there.
func ( handlers *Handlers ) commentCard( c fiber.Ctx ) ( principal *access.Principal , board *models.Board , card *models.Card , err error ) {
	principal , board , card , err = handlers.requireCard( c , access.View )
	if card == nil { return }
	if access.QuickLink( principal ) {
		board , card = nil , nil
		err = c.Status( fiber.StatusForbidden ).JSON( fiber.Map{ "error": "forbidden" } )
	}
	return
}

func ( handlers *Handlers ) ListComments( c fiber.Ctx ) ( err error ) {
	principal , board , card , err := handlers.commentCard( c )
	if card == nil { return }
	comments , list_err := models.ListComments( handlers.Store , board.ID , card.ID )
	if list_err != nil {
		err = serverError( c )
		return
	}
	rows := []fiber.Map{}
	for _ , comment := range comments { rows = append( rows , commentPayload( principal , board , comment ) ) }
	err = c.JSON( fiber.Map{ "comments": rows , "can_comment": access.CanComment( principal , board , card ) } )
	return
}

func ( handlers *Handlers ) AddComment( c fiber.Ctx ) ( err error ) {
	principal , board , card , err := handlers.commentCard( c )
	if card == nil { return }
	var body commentRequest
	if ok , bind_err := bindBody( c , &body ); ok == false { return bind_err }
	comment , refused := handlers.addComment( principal , board , card , body.Text )
	if refused != nil {
		err = refused.send( c )
		return
	}
	err = c.JSON( commentPayload( principal , board , comment ) )
	return
}

// addComment is posting a comment for both doors, the page and MCP.
func ( handlers *Handlers ) addComment( principal *access.Principal , board *models.Board , card *models.Card , text string ) ( comment *models.Comment , refused *refusal ) {
	if access.CanComment( principal , board , card ) == false {
		refused = refuse( fiber.StatusForbidden , "forbidden" )
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

// loadComment is :comment_id on a card the principal can read.
func ( handlers *Handlers ) loadComment( c fiber.Ctx ) ( principal *access.Principal , board *models.Board , comment *models.Comment , err error ) {
	principal , board , card , err := handlers.commentCard( c )
	if card == nil { return }
	comment_id , _ := paramID( c , "comment_id" )
	comment , get_err := models.GetComment( handlers.Store , board.ID , card.ID , comment_id )
	if get_err != nil {
		board = nil
		err = notFound( c , "comment_not_found" )
	}
	return
}

func ( handlers *Handlers ) EditComment( c fiber.Ctx ) ( err error ) {
	principal , board , comment , err := handlers.loadComment( c )
	if comment == nil { return }
	if canEditComment( principal , comment ) == false {
		err = c.Status( fiber.StatusForbidden ).JSON( fiber.Map{ "error": "forbidden" } )
		return
	}
	var body commentRequest
	if ok , bind_err := bindBody( c , &body ); ok == false { return bind_err }
	updated , edit_err := models.EditComment( handlers.Store , board.ID , comment.CardID , comment.ID , body.Text )
	if edit_err != nil {
		err = modelError( c , edit_err )
		return
	}
	err = c.JSON( commentPayload( principal , board , updated ) )
	return
}

func ( handlers *Handlers ) RemoveComment( c fiber.Ctx ) ( err error ) {
	principal , board , comment , err := handlers.loadComment( c )
	if comment == nil { return }
	if canRemoveComment( principal , board , comment ) == false {
		err = c.Status( fiber.StatusForbidden ).JSON( fiber.Map{ "error": "forbidden" } )
		return
	}
	if remove_err := models.RemoveComment( handlers.Store , board.ID , comment.CardID , comment.ID ); remove_err != nil {
		err = modelError( c , remove_err )
		return
	}
	err = c.JSON( fiber.Map{ "ok": true } )
	return
}
