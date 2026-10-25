package routes

import (
	fmt "fmt"
	time "time"

	fiber "github.com/gofiber/fiber/v3"

	access "todoboard/server/access"
	models "todoboard/server/models"
)

// Backups as JSON files: every board someone can open, in one file, and one
// column at a time. Importing always adds -- new boards owned by whoever
// imports, or cards appended to a column -- and never overwrites. A whole
// backup comes back in steps (see static/js/backup.js): ImportBoard makes
// each board's lists and labels, then ImportColumn adds the cards a batch at
// a time, so no request has to carry more than the body limit.

type importBoardRequest struct {
	Board *models.BoardSnapshot `json:"board"`
}

type importColumnRequest struct {
	Labels []*models.LabelSnapshot `json:"labels"`
	Cards  []*models.CardSnapshot  `json:"cards"`
}

// ExportBackup answers GET /api/backup with every board the caller is on,
// each as much of it as they can see -- what they would be shown on the
// board itself. Boards reached through a single shared card are left out.
func ( handlers *Handlers ) ExportBackup( c fiber.Ctx ) ( err error ) {
	principal := principalFrom( c )
	summaries , _ , list_err := models.ListBoardsForUser( handlers.Store , principal.UserID )
	if list_err != nil {
		err = serverError( c )
		return
	}
	backup := &models.Backup{ Format: models.BackupFormat , Version: models.SnapshotVersion , ExportedAt: time.Now().UTC() , Boards: []*models.BoardSnapshot{} }
	person := handlers.personNamer()
	for _ , summary := range summaries {
		board , get_err := models.GetBoard( handlers.Store , summary.ID )
		if get_err != nil { continue }
		cards , cards_err := models.ListCards( handlers.Store , board.ID )
		if cards_err != nil {
			err = serverError( c )
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

// ImportBoard answers POST /api/backup/board: one board from a backup, as a
// new board the caller owns, with its lists and labels but no cards yet.
func ( handlers *Handlers ) ImportBoard( c fiber.Ctx ) ( err error ) {
	var body importBoardRequest
	if ok , bind_err := bindBody( c , &body ); ok == false { return bind_err }
	if body.Board == nil {
		err = badRequest( c , "invalid_input" )
		return
	}
	principal := principalFrom( c )
	board , lists , import_err := models.ImportBoardSkeleton( handlers.Store , principal.UserID , body.Board )
	if import_err != nil {
		err = modelError( c , import_err )
		return
	}
	err = c.JSON( fiber.Map{ "id": board.ID , "slug": board.Slug , "lists": lists } )
	return
}

// ExportColumn answers GET .../lists/:list_id/export with that column's file:
// its cards the caller can see, and the labels they use.
func ( handlers *Handlers ) ExportColumn( c fiber.Ctx ) ( err error ) {
	principal , board , list , err := handlers.columnFor( c , access.View )
	if list == nil { return }
	cards , list_err := models.ListCards( handlers.Store , board.ID )
	if list_err != nil {
		err = serverError( c )
		return
	}
	visible := func( card *models.Card ) bool { return access.Card( principal , board , card ) != access.None }
	c.Set( fiber.HeaderCacheControl , "no-store" )
	err = c.JSON( models.SnapshotColumn( board , list , cards , visible , handlers.personNamer() ) )
	return
}

// ImportColumn answers POST .../lists/:list_id/import: appends cards to the
// column. It needs the same rights as adding a card there.
func ( handlers *Handlers ) ImportColumn( c fiber.Ctx ) ( err error ) {
	principal , board , list , err := handlers.columnFor( c , access.Edit )
	if list == nil { return }
	var body importColumnRequest
	if ok , bind_err := bindBody( c , &body ); ok == false { return bind_err }
	count , import_err := models.ImportCards( handlers.Store , board.ID , list.ID , body.Labels , body.Cards , principal.UserID , principal.Name )
	if import_err != nil {
		err = modelError( c , import_err )
		return
	}
	err = c.JSON( fiber.Map{ "count": count } )
	return
}

// columnFor loads the board and the :list_id column, answering for the
// caller when it is missing or they lack level on it.
func ( handlers *Handlers ) columnFor( c fiber.Ctx , level access.Level ) ( principal *access.Principal , board *models.Board , list *models.List , err error ) {
	principal , board , err = handlers.boardFor( c )
	if board == nil { return }
	list_id , ok := paramID( c , "list_id" )
	if ok == false || board.List( list_id ) == nil {
		err = notFound( c , "list_not_found" )
		return
	}
	if access.List( principal , board , list_id ) < level {
		err = forbidden( c , "forbidden" )
		return
	}
	list = board.List( list_id )
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
