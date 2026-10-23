package routes

import (
	fiber "github.com/gofiber/fiber/v3"

	models "todoboard/server/models"
	security "todoboard/server/security"
)

// Folders are each person's own way of sorting their home page, so every
// call here acts on the caller's folders and the caller's view of a board --
// never on the board itself, and never on anyone else's home page.

type folderRequest struct {
	Name string `json:"name"`
}

type boardFolderRequest struct {
	FolderID uint64 `json:"folder_id"` // 0: not in a folder
}

func ( handlers *Handlers ) CreateFolder( c fiber.Ctx ) ( err error ) {
	var body folderRequest
	if ok , bind_err := bindBody( c , &body ); ok == false { return bind_err }
	folder , create_err := models.CreateFolder( handlers.Store , security.UserFrom( c ).ID , body.Name )
	if create_err != nil {
		err = modelError( c , create_err )
		return
	}
	err = c.JSON( folder )
	return
}

func ( handlers *Handlers ) RenameFolder( c fiber.Ctx ) ( err error ) {
	folder_id , ok := paramID( c , "folder_id" )
	if ok == false {
		err = notFound( c , "folder_not_found" )
		return
	}
	var body folderRequest
	if ok , bind_err := bindBody( c , &body ); ok == false { return bind_err }
	folder , rename_err := models.RenameFolder( handlers.Store , security.UserFrom( c ).ID , folder_id , body.Name )
	if rename_err != nil {
		err = modelError( c , rename_err )
		return
	}
	err = c.JSON( folder )
	return
}

func ( handlers *Handlers ) DeleteFolder( c fiber.Ctx ) ( err error ) {
	folder_id , ok := paramID( c , "folder_id" )
	if ok == false {
		err = notFound( c , "folder_not_found" )
		return
	}
	if delete_err := models.DeleteFolder( handlers.Store , security.UserFrom( c ).ID , folder_id ); delete_err != nil {
		err = modelError( c , delete_err )
		return
	}
	err = c.JSON( fiber.Map{ "ok": true } )
	return
}

// SetBoardFolder files one of the caller's boards. Only boards on their home
// page can be filed; the model answers board_not_found for any other.
func ( handlers *Handlers ) SetBoardFolder( c fiber.Ctx ) ( err error ) {
	board_id , ok := paramID( c , "board_id" )
	if ok == false {
		err = notFound( c , "board_not_found" )
		return
	}
	var body boardFolderRequest
	if ok , bind_err := bindBody( c , &body ); ok == false { return bind_err }
	if set_err := models.SetBoardFolder( handlers.Store , security.UserFrom( c ).ID , board_id , body.FolderID ); set_err != nil {
		err = modelError( c , set_err )
		return
	}
	err = c.JSON( fiber.Map{ "ok": true } )
	return
}
