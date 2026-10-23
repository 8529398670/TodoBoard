package routes

import (
	errors "errors"

	fiber "github.com/gofiber/fiber/v3"

	access "todoboard/server/access"
	models "todoboard/server/models"
	security "todoboard/server/security"
)

type createBoardRequest struct {
	Title string   `json:"title"`
	Lists []string `json:"lists"`
}

type listRequest struct {
	Title    string   `json:"title"`
	Position *float64 `json:"position"`
}

type labelRequest struct {
	Name  *string `json:"name"`
	Color *string `json:"color"`
}

// ListMyBoards is the home page: boards I own or am a member of, the folders
// I sort them into, and single cards shared with me from boards I am not on.
func ( handlers *Handlers ) ListMyBoards( c fiber.Ctx ) ( err error ) {
	user := security.UserFrom( c )
	boards , cards , list_err := models.ListBoardsForUser( handlers.Store , user.ID )
	if list_err != nil {
		err = serverError( c )
		return
	}
	folders , folder_err := models.ListFolders( handlers.Store , user.ID )
	if folder_err != nil {
		err = serverError( c )
		return
	}
	err = c.JSON( fiber.Map{ "boards": boards , "folders": folders , "shared_cards": cards } )
	return
}

func ( handlers *Handlers ) CreateBoard( c fiber.Ctx ) ( err error ) {
	var body createBoardRequest
	if ok , bind_err := bindBody( c , &body ); ok == false { return bind_err }
	principal := principalFrom( c )
	board , create_err := models.CreateBoard( handlers.Store , principal.UserID , body.Title , body.Lists )
	if create_err != nil {
		err = modelError( c , create_err )
		return
	}
	err = c.JSON( fiber.Map{ "id": board.ID , "slug": board.Slug } )
	return
}

// GetBoard returns everything a board view needs in one response: the board,
// the cards this caller may see, the names behind every user id mentioned,
// and what this caller may do.
func ( handlers *Handlers ) GetBoard( c fiber.Ctx ) ( err error ) {
	principal , board , err := handlers.boardFor( c )
	if board == nil { return }
	cards , list_err := models.ListCards( handlers.Store , board.ID )
	if list_err != nil {
		err = serverError( c )
		return
	}
	err = c.JSON( handlers.boardPayload( principal , board , cards ) )
	return
}

// GetBoardStamp is the cheap call a board view polls: the revision only.
func ( handlers *Handlers ) GetBoardStamp( c fiber.Ctx ) ( err error ) {
	_ , board , err := handlers.boardFor( c )
	if board == nil { return }
	err = c.JSON( fiber.Map{ "rev": board.Rev } )
	return
}

func ( handlers *Handlers ) boardPayload( principal *access.Principal , board *models.Board , all_cards []*models.Card ) ( payload fiber.Map ) {
	board_level := access.Board( principal , board )
	// A column link sees that one column as a board of its own; a card-only
	// caller sees just their cards. Neither is shown the member list.
	list_only := access.ListOnly( principal , board )
	card_only := board_level == access.None && list_only == 0
	whole_board := board_level != access.None
	reported := board_level
	if list_only != 0 { reported = access.List( principal , board , list_only ) }

	visible := []fiber.Map{}
	visible_lists := map[uint64]bool{}
	people_ids := map[uint64]bool{ board.OwnerID: true }
	for _ , card := range all_cards {
		level := access.Card( principal , board , card )
		if level == access.None { continue }
		visible_lists[ card.ListID ] = true
		for _ , id := range card.Assignees { people_ids[ id ] = true }
		visible = append( visible , cardPayload( card , level , board_level ) )
	}

	lists := []*models.List{}
	for _ , list := range board.SortedLists() {
		if list_only != 0 && list.ID != list_only { continue }
		if card_only && visible_lists[ list.ID ] == false { continue }
		lists = append( lists , list )
	}

	members := []fiber.Map{}
	if whole_board {
		for id := range board.Members { people_ids[ id ] = true }
	}
	people := map[uint64]string{}
	for id := range people_ids {
		if user , get_err := models.GetUser( handlers.Store , id ); get_err == nil {
			people[ id ] = user.DisplayName
		}
	}
	if whole_board {
		members = append( members , fiber.Map{ "user_id": board.OwnerID , "role": models.BoardRoleOwner , "display_name": people[ board.OwnerID ] } )
		for id , role := range board.Members {
			members = append( members , fiber.Map{ "user_id": id , "role": role , "display_name": people[ id ] } )
		}
	}

	me := fiber.Map{ "guest": principal.IsGuest() }
	if principal.IsGuest() == false {
		me[ "user_id" ] = principal.UserID
		me[ "role" ] = board.RoleOf( principal.UserID )
		if ref := models.GetBoardRef( handlers.Store , principal.UserID , board.ID ); ref != nil {
			me[ "plain_icons" ] = ref.PlainIcons
		}
	}

	payload = fiber.Map{
		"board": fiber.Map{
			"id":           board.ID,
			"title":        board.Title,
			"slug":         board.Slug,
			"layout":       board.Layout,
			"numbering":    board.Numbering,
			"owner_id":     board.OwnerID,
			"default_view": board.DefaultView,
			"color":        board.Color,
			"theme":        board.Theme,
			"rev":          board.Rev,
			"lists":        lists,
			"labels":       board.Labels,
			"members":      members,
		},
		"cards":     visible,
		"people":    people,
		"access":    reported.String(),
		"card_only": card_only,
		"list_only": list_only,
		"me":        me,
	}
	return
}

// cardPayload is a card as a caller sees it. Who else a card is shared with
// is the owner's business, so it is only included for someone who manages
// the board.
func cardPayload( card *models.Card , level access.Level , board_level access.Level ) ( payload fiber.Map ) {
	payload = fiber.Map{
		"id":          card.ID,
		"board_id":    card.BoardID,
		"list_id":     card.ListID,
		"position":    card.Position,
		"title":       card.Title,
		"description": card.Description,
		"label_ids":   card.LabelIDs,
		"assignees":   card.Assignees,
		"start_at":    card.StartAt,
		"start_all_day": card.StartAt != nil && card.StartTimed == false,
		"due_at":      card.DueAt,
		"due_all_day": card.DueAllDay,
		"done":        card.Done,
		"done_at":     card.DoneAt,
		"checklist":   card.Checklist,
		"archived":    card.Archived,
		"created_at":  card.CreatedAt,
		"updated_at":  card.UpdatedAt,
		"updated_by":  card.UpdatedBy,
		"rev":         card.Rev,
		"access":      level.String(),
	}
	if board_level == access.Manage {
		payload[ "shared_with" ] = card.SharedWith
	}
	return
}

func ( handlers *Handlers ) UpdateBoard( c fiber.Ctx ) ( err error ) {
	_ , board , err := handlers.requireBoard( c , access.Edit )
	if board == nil { return }
	var body models.BoardSettings
	if ok , bind_err := bindBody( c , &body ); ok == false { return bind_err }
	updated , update_err := models.UpdateBoardSettings( handlers.Store , board.ID , &body )
	if errors.Is( update_err , models.ErrSlugTaken ) {
		err = c.Status( fiber.StatusConflict ).JSON( fiber.Map{ "error": "slug_taken" } )
		return
	}
	if update_err != nil {
		err = modelError( c , update_err )
		return
	}
	err = c.JSON( fiber.Map{ "rev": updated.Rev , "slug": updated.Slug } )
	return
}

// DeleteBoard needs the owner and a browser session: it cannot be undone,
// so a leaked API key must not be able to do it.
func ( handlers *Handlers ) DeleteBoard( c fiber.Ctx ) ( err error ) {
	_ , board , err := handlers.requireBoard( c , access.Manage )
	if board == nil { return }
	if delete_err := models.DeleteBoard( handlers.Store , board.ID ); delete_err != nil {
		err = modelError( c , delete_err )
		return
	}
	err = c.JSON( fiber.Map{ "ok": true } )
	return
}

// plain_icons null means "use my account default".
type myViewRequest struct {
	PlainIcons *bool `json:"plain_icons"`
}

// SetMyBoardView saves how this board looks to the caller alone -- greyscale
// status icons, for now. Only people on the board have somewhere to keep it.
func ( handlers *Handlers ) SetMyBoardView( c fiber.Ctx ) ( err error ) {
	principal , board , err := handlers.requireBoard( c , access.View )
	if board == nil { return }
	var body myViewRequest
	if ok , bind_err := bindBody( c , &body ); ok == false { return bind_err }
	if set_err := models.SetBoardPlainIcons( handlers.Store , principal.UserID , board.ID , body.PlainIcons ); set_err != nil {
		err = modelError( c , set_err )
		return
	}
	err = c.JSON( fiber.Map{ "plain_icons": body.PlainIcons } )
	return
}

// LeaveBoard lets a member remove themselves. The owner cannot leave; they
// delete the board or hand it over instead.
func ( handlers *Handlers ) LeaveBoard( c fiber.Ctx ) ( err error ) {
	principal , board , err := handlers.requireBoard( c , access.View )
	if board == nil { return }
	if board.OwnerID == principal.UserID {
		err = badRequest( c , "owner_cannot_leave" )
		return
	}
	if _ , set_err := models.SetMember( handlers.Store , board.ID , principal.UserID , "" ); set_err != nil {
		err = modelError( c , set_err )
		return
	}
	err = c.JSON( fiber.Map{ "ok": true } )
	return
}

func ( handlers *Handlers ) CreateList( c fiber.Ctx ) ( err error ) {
	_ , board , err := handlers.requireBoard( c , access.Edit )
	if board == nil { return }
	var body listRequest
	if ok , bind_err := bindBody( c , &body ); ok == false { return bind_err }
	var created *models.List
	_ , update_err := models.MutateBoard( handlers.Store , board.ID , func( target *models.Board ) ( mutate_err error ) {
		created , mutate_err = models.AddList( target , body.Title , body.Position )
		return
	} )
	if update_err != nil {
		err = modelError( c , update_err )
		return
	}
	err = c.JSON( created )
	return
}

func ( handlers *Handlers ) UpdateList( c fiber.Ctx ) ( err error ) {
	_ , board , err := handlers.requireBoard( c , access.Edit )
	if board == nil { return }
	list_id , ok := paramID( c , "list_id" )
	if ok == false {
		err = notFound( c , "list_not_found" )
		return
	}
	var body models.ListUpdate
	if ok , bind_err := bindBody( c , &body ); ok == false { return bind_err }
	_ , update_err := models.MutateBoard( handlers.Store , board.ID , func( target *models.Board ) error {
		return body.Apply( target , list_id )
	} )
	if update_err != nil {
		err = modelError( c , update_err )
		return
	}
	err = c.JSON( fiber.Map{ "ok": true } )
	return
}

func ( handlers *Handlers ) DeleteList( c fiber.Ctx ) ( err error ) {
	_ , board , err := handlers.requireBoard( c , access.Edit )
	if board == nil { return }
	list_id , ok := paramID( c , "list_id" )
	if ok == false {
		err = notFound( c , "list_not_found" )
		return
	}
	if delete_err := models.DeleteList( handlers.Store , board.ID , list_id ); delete_err != nil {
		err = modelError( c , delete_err )
		return
	}
	err = c.JSON( fiber.Map{ "ok": true } )
	return
}

func ( handlers *Handlers ) CreateLabel( c fiber.Ctx ) ( err error ) {
	_ , board , err := handlers.requireBoard( c , access.Edit )
	if board == nil { return }
	var body labelRequest
	if ok , bind_err := bindBody( c , &body ); ok == false { return bind_err }
	if body.Color == nil {
		err = badRequest( c , "invalid_input" )
		return
	}
	name := ""
	if body.Name != nil { name = *body.Name }
	var created *models.Label
	_ , update_err := models.MutateBoard( handlers.Store , board.ID , func( target *models.Board ) ( mutate_err error ) {
		created , mutate_err = models.AddLabel( target , name , *body.Color )
		return
	} )
	if update_err != nil {
		err = modelError( c , update_err )
		return
	}
	err = c.JSON( created )
	return
}

func ( handlers *Handlers ) UpdateLabel( c fiber.Ctx ) ( err error ) {
	_ , board , err := handlers.requireBoard( c , access.Edit )
	if board == nil { return }
	label_id , ok := paramID( c , "label_id" )
	if ok == false {
		err = notFound( c , "label_not_found" )
		return
	}
	var body labelRequest
	if ok , bind_err := bindBody( c , &body ); ok == false { return bind_err }
	_ , update_err := models.MutateBoard( handlers.Store , board.ID , func( target *models.Board ) error {
		return models.UpdateLabel( target , label_id , body.Name , body.Color )
	} )
	if update_err != nil {
		err = modelError( c , update_err )
		return
	}
	err = c.JSON( fiber.Map{ "ok": true } )
	return
}

func ( handlers *Handlers ) DeleteLabel( c fiber.Ctx ) ( err error ) {
	_ , board , err := handlers.requireBoard( c , access.Edit )
	if board == nil { return }
	label_id , ok := paramID( c , "label_id" )
	if ok == false {
		err = notFound( c , "label_not_found" )
		return
	}
	if delete_err := models.DeleteLabel( handlers.Store , board.ID , label_id ); delete_err != nil {
		err = modelError( c , delete_err )
		return
	}
	err = c.JSON( fiber.Map{ "ok": true } )
	return
}
