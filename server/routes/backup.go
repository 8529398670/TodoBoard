package routes

import (
	json "encoding/json"
	fmt "fmt"
	strconv "strconv"
	time "time"

	fiber "github.com/gofiber/fiber/v3"

	access "todoboard/server/access"
	apidoc "todoboard/server/apidoc"
	models "todoboard/server/models"
)

// Backups as JSON files: every board someone can open, in one file, and one
// column at a time. Importing always adds -- new boards owned by whoever
// imports, or cards appended to a column -- and never overwrites. A whole
// backup comes back in steps (see static/js/backup.js): importBoard makes
// each board's lists and labels, then importColumn adds the cards a batch at
// a time, so no request has to carry more than the body limit.

// importBoardRequest and importColumnRequest are read by the handlers
// themselves (raw_body), as bindBody did: a backup file from an older
// version may lack fields or carry ones this version no longer has.
type importBoardRequest struct {
	Board     *models.BoardSnapshot `json:"board"`
	CSRFToken string                `json:"csrf_token,omitempty"`
}

func ( importBoardRequest ) APIDoc() ( text string ) {
	text = "One board of a backup file (boards[n] of GET /api/backup), without its lists' cards."
	return
}

type importColumnRequest struct {
	Labels    []*models.LabelSnapshot `json:"labels"`
	Cards     []*models.CardSnapshot  `json:"cards"`
	CSRFToken string                  `json:"csrf_token,omitempty"`
}

func ( importColumnRequest ) APIDoc() ( text string ) {
	text = "Cards to add to a column, and the labels they use, as a column's file or a backup board's list holds them."
	return
}

// readImportBody decodes a raw_body route's JSON body, and checks a browser
// session's csrf_token in it, which the framework does not look for there.
// A share link's guest needs none (its header cannot be sent cross-site).
func ( handlers *Handlers ) readImportBody( c fiber.Ctx , body any , token func() string , guest bool ) ( err error ) {
	if json.Unmarshal( c.Body() , body ) != nil {
		err = refuse( fiber.StatusBadRequest , "malformed request body" )
		return
	}
	if guest == false && handlers.Guard.CheckCSRF( c , token() ) == false {
		err = refuse( fiber.StatusForbidden , "invalid csrf token" )
	}
	return
}

// exportBackup answers with every board the caller is on, each as much of
// it as they can see -- what they would be shown on the board itself.
// Boards reached through a single shared card are left out.
//
// @route("GET /api/backup", summary="Download a backup of your boards",
//     tag="Backup", reply_type="application/json",
//     description="Every board you are on, as one JSON file to keep: each as much of it as you can see, "
//         "with its columns, labels and cards. Boards reached through a single shared card are left out.",
//     mcp_http="A file (a picture, a spreadsheet or a backup), which travels over HTTP rather than as a tool answer.",
// )
func ( handlers *Handlers ) exportBackup( c fiber.Ctx , in *apidoc.None ) ( err error ) {
	principal := principalFrom( c )
	summaries , _ , list_err := handlers.boardsFor( principal )
	if list_err != nil {
		err = list_err
		return
	}
	backup := &models.Backup{ Format: models.BackupFormat , Version: models.SnapshotVersion , ExportedAt: time.Now().UTC() , Boards: []*models.BoardSnapshot{} }
	person := handlers.personNamer()
	for _ , summary := range summaries {
		board , get_err := models.GetBoard( handlers.Store , summary.ID )
		if get_err != nil { continue }
		cards , cards_err := models.ListCards( handlers.Store , board.ID )
		if cards_err != nil {
			err = cards_err
			return
		}
		visible := func( card *models.Card ) bool { return access.Card( principal , board , card ) != access.None }
		backup.Boards = append( backup.Boards , models.SnapshotBoard( board , cards , visible , access.ListOnly( principal , board ) , person ) )
	}
	c.Set( fiber.HeaderContentDisposition , fmt.Sprintf( `attachment; filename="todoboard-backup-%s.json"` , time.Now().UTC().Format( "2006-01-02" ) ) )
	c.Set( fiber.HeaderCacheControl , "no-store" )
	err = c.JSON( backup )
	return
}

// importBoard makes one board from a backup, as a new board the caller
// owns, with its lists and labels but no cards yet.
//
// @route("POST /api/backup/board", summary="Restore a board from a backup",
//     tag="Backup", errors=(400, 403), raw_body=importBoardRequest,
//     description="Make a new board you own from one board of a backup file, with its columns and labels "
//         "but no cards yet: add those a batch at a time with POST /api/boards/{board_id}/lists/{list_id}/import, "
//         "using the list ids answered (in the backup's order). Never overwrites anything.",
//     mcp_http="A backup file's contents, which the page sends in steps; deliberately not a tool.",
// )
func ( handlers *Handlers ) importBoard( c fiber.Ctx , in *apidoc.None ) ( reply *fiber.Map , err error ) {
	if err = refuseOnlyBoardsKey( c ); err != nil { return }
	var body importBoardRequest
	if err = handlers.readImportBody( c , &body , func() string { return body.CSRFToken } , false ); err != nil { return }
	if body.Board == nil {
		err = refuse( fiber.StatusBadRequest , "invalid_input" )
		return
	}
	principal := principalFrom( c )
	board , lists , import_err := models.ImportBoardSkeleton( handlers.Store , principal.UserID , body.Board )
	if import_err != nil {
		err = modelErr( import_err )
		return
	}
	reply = &fiber.Map{ "id": board.ID , "slug": board.Slug , "lists": lists }
	return
}

// exportColumn answers with that column's file: its cards the caller can
// see, and the labels they use.
//
// @route("GET /api/boards/{board_id}/lists/{list_id}/export", summary="Download a column",
//     tag="Backup", errors=(403, 404), reply_type="application/json",
//     description="One column as a JSON file: its cards you can see, and the labels they use.",
//     mcp_http="A file (a picture, a spreadsheet or a backup), which travels over HTTP rather than as a tool answer.",
// )
func ( handlers *Handlers ) exportColumn( c fiber.Ctx , in *listPath ) ( err error ) {
	principal , board , list , err := handlers.columnAt( c , in , access.View )
	if err != nil { return }
	cards , list_err := models.ListCards( handlers.Store , board.ID )
	if list_err != nil {
		err = list_err
		return
	}
	visible := func( card *models.Card ) bool { return access.Card( principal , board , card ) != access.None }
	c.Set( fiber.HeaderCacheControl , "no-store" )
	err = c.JSON( models.SnapshotColumn( board , list , cards , visible , handlers.personNamer() ) )
	return
}

// sharedExportColumn is exportColumn for a share link's guest.
//
// @route("GET /api/shared/{board_id}/lists/{list_id}/export", summary="Download a column (share link)",
//     tag="Shared", access=PUBLIC, use=[handlers.SharePrincipal], errors=(403, 404), reply_type="application/json",
//     description="exportColumn for a guest holding a share link in X-Share-Token.",
// )
func ( handlers *Handlers ) sharedExportColumn( c fiber.Ctx , in *listPath ) ( err error ) {
	err = handlers.exportColumn( c , in )
	return
}

// importColumn appends cards to the column. It needs the same rights as
// adding a card there.
//
// @route("POST /api/boards/{board_id}/lists/{list_id}/import", summary="Add cards to a column from a file",
//     tag="Backup", errors=(400, 403, 404), raw_body=importColumnRequest,
//     description="Append the cards of a column's file (or a batch of a backup board's cards) to a column, "
//         "with the labels they use. Needs the rights to add a card there. Never overwrites anything. "
//         "Answers how many cards were added.",
//     mcp_http="A backup file's contents, which the page sends in steps; deliberately not a tool.",
// )
func ( handlers *Handlers ) importColumn( c fiber.Ctx , in *listPath ) ( reply *fiber.Map , err error ) {
	reply , err = handlers.importColumnFor( c , in , false )
	return
}

// sharedImportColumn is importColumn for a share link's guest.
//
// @route("POST /api/shared/{board_id}/lists/{list_id}/import", summary="Add cards to a column from a file (share link)",
//     tag="Shared", access=PUBLIC, use=[handlers.SharePrincipal], errors=(400, 403, 404), raw_body=importColumnRequest,
//     description="importColumn for a guest holding a share link in X-Share-Token.",
// )
func ( handlers *Handlers ) sharedImportColumn( c fiber.Ctx , in *listPath ) ( reply *fiber.Map , err error ) {
	reply , err = handlers.importColumnFor( c , in , true )
	return
}

func ( handlers *Handlers ) importColumnFor( c fiber.Ctx , in *listPath , guest bool ) ( reply *fiber.Map , err error ) {
	var body importColumnRequest
	if err = handlers.readImportBody( c , &body , func() string { return body.CSRFToken } , guest ); err != nil { return }
	principal , board , list , err := handlers.columnAt( c , in , access.Edit )
	if err != nil { return }
	count , import_err := models.ImportCards( handlers.Store , board.ID , list.ID , body.Labels , body.Cards , principal.UserID , principal.Name )
	if import_err != nil {
		err = modelErr( import_err )
		return
	}
	if count > 0 { handlers.activity( principal , board.ID , "cards_imported" , map[string]string{ "list": list.Title , "n": strconv.Itoa( count ) } ) }
	reply = &fiber.Map{ "count": count }
	return
}

// columnAt loads the board and the column a listPath names, refused (as
// the error) when it is missing or the caller lacks level on it.
func ( handlers *Handlers ) columnAt( c fiber.Ctx , in *listPath , level access.Level ) ( principal *access.Principal , board *models.Board , list *models.List , err error ) {
	principal , board , err = handlers.boardAt( c , in.BoardID )
	if err != nil { return }
	if board.List( in.ListID ) == nil {
		err = refuse( fiber.StatusNotFound , "list_not_found" )
		return
	}
	if access.List( principal , board , in.ListID ) < level {
		err = refuse( fiber.StatusForbidden , "forbidden" )
		return
	}
	list = board.List( in.ListID )
	return
}

// personNamer names user IDs for a file, looking each one up once.
func ( handlers *Handlers ) personNamer() ( person func( id uint64 ) string ) {
	people := map[uint64]string{}
	person = func( id uint64 ) string {
		if name , known := people[ id ]; known { return name }
		name := "?"
		if user , get_err := models.GetUser( handlers.Store , id ); get_err == nil { name = user.DisplayName }
		people[ id ] = name
		return name
	}
	return
}
