package routes

import (
	fiber "github.com/gofiber/fiber/v3"

	models "todoboard/server/models"
	security "todoboard/server/security"
)

// Folders are each person's own way of sorting their home page, so every
// call here acts on the caller's folders and the caller's view of a board --
// never on the board itself, and never on anyone else's home page. A key held
// to some boards cannot use them (refuseLimitedKey): a folder's name could
// say what is on boards it cannot open.

type folderPath struct {
	FolderID uint64 `json:"folder_id" path:"folder_id" jsonschema_description:"A folder's id, from list_boards (folders)."`
}

type folderRequest struct {
	Name     string `json:"name"`
	ParentID uint64 `json:"parent_id,omitempty" jsonschema_description:"The folder to make it in; 0 or absent: at the top."`
}

type renameFolderRequest struct {
	folderPath
	Name string `json:"name"`
}

type folderParentRequest struct {
	folderPath
	ParentID uint64 `json:"parent_id" jsonschema_description:"The folder to put it in; 0: at the top."`
}

type folderOrderRequest struct {
	IDs []uint64 `json:"ids" jsonschema_description:"Your folder ids, first to last."`
}

type boardFolderRequest struct {
	boardPath
	FolderID uint64 `json:"folder_id" jsonschema_description:"The folder to file it in; 0: not in a folder."`
}

// createFolder makes one of the caller's home page folders.
//
// @route("POST /api/folders", summary="Make a folder",
//     tag="Folders",
//     description="Make a folder on your own home page, at the top or inside another of yours (parent_id).",
//     mcp="create_folder",
// )
func ( handlers *Handlers ) createFolder( c fiber.Ctx , in *folderRequest ) ( reply *models.Folder , err error ) {
	if err = refuseLimitedKey( c ); err != nil { return }
	reply , create_err := models.CreateFolderIn( handlers.Store , security.UserFrom( c ).ID , in.Name , in.ParentID )
	if create_err != nil { err = modelErr( create_err ) }
	return
}

// setFolderOrder stores the order the caller dragged their folders into.
//
// @route("POST /api/folders/order", summary="Order your folders",
//     tag="Folders",
//     description="Put your folders in this order, first to last.",
//     mcp="order_folders",
// )
func ( handlers *Handlers ) setFolderOrder( c fiber.Ctx , in *folderOrderRequest ) ( reply *okReply , err error ) {
	if err = refuseLimitedKey( c ); err != nil { return }
	if order_err := models.SetFolderOrder( handlers.Store , security.UserFrom( c ).ID , in.IDs ); order_err != nil {
		err = modelErr( order_err )
		return
	}
	reply = &okReply{ Ok: true }
	return
}

// renameFolder renames one of the caller's folders.
//
// @route("POST /api/folders/{folder_id}", summary="Rename a folder",
//     tag="Folders",
//     description="Rename one of your home page folders.",
//     mcp="rename_folder",
// )
func ( handlers *Handlers ) renameFolder( c fiber.Ctx , in *renameFolderRequest ) ( reply *models.Folder , err error ) {
	if err = refuseLimitedKey( c ); err != nil { return }
	reply , rename_err := models.RenameFolder( handlers.Store , security.UserFrom( c ).ID , in.FolderID , in.Name )
	if rename_err != nil { err = modelErr( rename_err ) }
	return
}

// moveFolder puts one of the caller's folders inside another of theirs, or
// back at the top.
//
// @route("POST /api/folders/{folder_id}/parent", summary="Move a folder",
//     tag="Folders",
//     description="Put one of your folders inside another of yours, or back at the top (parent_id 0).",
//     mcp="move_folder",
// )
func ( handlers *Handlers ) moveFolder( c fiber.Ctx , in *folderParentRequest ) ( reply *models.Folder , err error ) {
	if err = refuseLimitedKey( c ); err != nil { return }
	reply , move_err := models.MoveFolder( handlers.Store , security.UserFrom( c ).ID , in.FolderID , in.ParentID )
	if move_err != nil { err = modelErr( move_err ) }
	return
}

// deleteFolder removes one of the caller's folders; the boards in it go back
// to the top of their home page.
//
// @route("POST /api/folders/{folder_id}/delete", summary="Delete a folder",
//     tag="Folders",
//     description="Remove one of your folders. Boards in it are not touched: they go back to the top of your home page.",
//     mcp="delete_folder", destructive=True,
// )
func ( handlers *Handlers ) deleteFolder( c fiber.Ctx , in *folderPath ) ( reply *okReply , err error ) {
	if err = refuseLimitedKey( c ); err != nil { return }
	if delete_err := models.DeleteFolder( handlers.Store , security.UserFrom( c ).ID , in.FolderID ); delete_err != nil {
		err = modelErr( delete_err )
		return
	}
	reply = &okReply{ Ok: true }
	return
}

// setBoardFolder files one of the caller's boards. Only boards on their home
// page can be filed; the model answers board_not_found for any other.
//
// @route("POST /api/boards/{board_id}/folder", summary="File a board in a folder",
//     tag="Folders",
//     description="File one of your boards in one of your folders on your home page, or take it out (folder_id 0).",
//     mcp="set_board_folder",
// )
func ( handlers *Handlers ) setBoardFolder( c fiber.Ctx , in *boardFolderRequest ) ( reply *okReply , err error ) {
	if err = refuseLimitedKey( c ); err != nil { return }
	_ , board , err := handlers.boardAt( c , in.BoardID )
	if err != nil { return }
	if set_err := models.SetBoardFolder( handlers.Store , security.UserFrom( c ).ID , board.ID , in.FolderID ); set_err != nil {
		err = modelErr( set_err )
		return
	}
	reply = &okReply{ Ok: true }
	return
}
