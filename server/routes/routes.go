// Package routes contains one file per group of endpoints, plus this file,
// which is the only place that knows the full URL map.
//
// The API here stays deliberately small: it covers logging in, managing who
// can log in, issuing the API keys that let non-browser callers in, and
// reading the UI's text. Everything else an app does belongs in a new file in
// this package (server side) and in static/js/ (browser side) -- see
// references/architecture.md for where the line sits and why.
package routes

import (
	fiber "github.com/gofiber/fiber/v3"

	config "todoboard/server/config"
	db "todoboard/server/db"
	language "todoboard/server/language"
	mcp "todoboard/server/mcp"
	models "todoboard/server/models"
	notify "todoboard/server/notify"
	security "todoboard/server/security"
	static "todoboard/server/static"
)

// Handlers carries the dependencies every route group needs. Passing this in
// rather than reaching for package-level globals is what makes the handlers
// testable and keeps the wiring visible in main.go.
type Handlers struct {
	Store    *db.Store
	Config   *config.Config
	Guard    *security.Guard
	Language *language.Language
	Static   *static.Server
	Pins     *models.PinGuard
	Notify   *notify.Service
	// Version is reported to MCP clients as serverInfo.version.
	Version string
}

func New( store *db.Store , cfg *config.Config , guard *security.Guard , lang *language.Language , static_server *static.Server , notifier *notify.Service , version string ) ( handlers *Handlers ) {
	handlers = &Handlers{
		Store:    store,
		Config:   cfg,
		Guard:    guard,
		Language: lang,
		Static:   static_server,
		Pins:     models.NewPinGuard(),
		Notify:   notifier,
		Version:  version,
	}
	return
}

// Register wires every route. Read it top to bottom to see the whole surface
// of this server.
func ( handlers *Handlers ) Register( app *fiber.App ) {
	// Resolve the session once, before anything else looks at the request.
	app.Use( handlers.Guard.LoadUser )

	// Public: the login link's page and the call it makes, and the UI's text
	// (needed to render the signed-out page at all). Opening a login link
	// only shows a page; the link is spent by the POST that page sends when
	// the person confirms. See auth.go.
	app.Get( "/login/*" , handlers.ShowLoginPage )
	app.Post( "/api/login" , handlers.RedeemLoginLink )
	app.Get( "/api/language" , handlers.GetLanguage )
	app.Get( "/api/health" , handlers.GetHealth )

	// Public: invite links (see join.go). Holding the link is the permission,
	// so these sit before the signed-in "/api" group below -- registered
	// after it, they would pick up RequireLogin from the shared prefix.
	app.Get( "/join/*" , handlers.ShowJoinPage )
	app.Post( "/api/join/info" , handlers.JoinInfo )
	app.Post( "/api/join/match" , handlers.JoinMatch )
	app.Post( "/api/join/claim" , handlers.JoinClaim )
	app.Post( "/api/join/create" , handlers.JoinCreate )

	// Board pages. Each reads its own URL; nothing is decided on the GET.
	app.Get( "/b/*" , handlers.ShowBoardPage )
	app.Get( "/s/*" , handlers.ShowSharePage )
	app.Get( "/print/*" , handlers.ShowPrintPage )

	// Public: guests holding a share link (X-Share-Token). Before the "/api"
	// group for the same reason as the invite routes above. What a link may
	// do is decided per request in server/access, never by role.
	shared := app.Group( "/api/shared" , handlers.SharePrincipal )
	shared.Get( "" , handlers.SharedInfo )
	handlers.registerBoardContent( shared )

	// Any signed-in user.
	account := app.Group( "/api" , handlers.Guard.RequireLogin )
	account.Get( "/me" , handlers.GetMe )
	account.Post( "/account/rename" , handlers.RenameAccount )
	account.Post( "/account/display" , handlers.SaveDisplay )
	// A session only: a PIN lets its holder sign in through an invite link,
	// so a key that could set one could turn itself into a session.
	account.Post( "/account/pin" , handlers.Guard.RequireSession , handlers.SetPin )
	// A session only, for the same reason as every route that hands out a
	// login link: the link becomes a session. See the admin group below.
	account.Post( "/account/login-link" , handlers.Guard.RequireSession , handlers.CreateMyLoginLink )
	account.Post( "/logout" , handlers.Logout )
	account.Get( "/team" , handlers.ListTeam )

	// Notification settings are the caller's own; the test push needs a
	// session only because it is a button in the account page.
	account.Get( "/notify/prefs" , handlers.GetNotifyPrefs )
	account.Post( "/notify/prefs" , handlers.SaveNotifyPrefs )
	account.Post( "/notify/test" , handlers.TestNotification )

	// Boards, for signed-in people. MemberPrincipal checks the CSRF token on
	// every write, so the handlers below need not. Who may do what on a
	// given board is server/access's decision, not the route's.
	boards := app.Group( "/api/boards" , handlers.Guard.RequireLogin , handlers.MemberPrincipal )
	boards.Get( "" , handlers.ListMyBoards )
	boards.Post( "" , handlers.CreateBoard )
	handlers.registerBoardContent( boards )
	boards.Post( "/:board_id/leave" , handlers.LeaveBoard )
	boards.Post( "/:board_id/folder" , handlers.SetBoardFolder )
	boards.Post( "/:board_id/my-view" , handlers.SetMyBoardView )
	boards.Post( "/:board_id/delete" , handlers.Guard.RequireSession , handlers.DeleteBoard )
	boards.Post( "/:board_id/members" , handlers.SetBoardMember )
	boards.Post( "/:board_id/cards/:card_id/share-user" , handlers.ShareCardWithUser )
	boards.Get( "/:board_id/shares" , handlers.ListShareLinks )
	boards.Post( "/:board_id/shares" , handlers.Guard.RequireSession , handlers.CreateShareLink )
	boards.Post( "/:board_id/shares/:link_id/revoke" , handlers.RevokeShareLink )
	boards.Post( "/:board_id/shares/:link_id/delete" , handlers.DeleteShareLink )
	boards.Get( "/:board_id/subscription" , handlers.GetBoardSubscription )
	boards.Post( "/:board_id/subscription" , handlers.SetBoardSubscription )
	boards.Get( "/:board_id/cards/:card_id/subscription" , handlers.GetCardSubscription )
	boards.Post( "/:board_id/cards/:card_id/subscription" , handlers.SetCardSubscription )

	// Backups: every board you are on as one file, and boards back from it.
	backup := app.Group( "/api/backup" , handlers.Guard.RequireLogin , handlers.MemberPrincipal )
	backup.Get( "" , handlers.ExportBackup )
	backup.Post( "/board" , handlers.ImportBoard )

	// Home page folders: each person's own. MemberPrincipal is here for its
	// CSRF check on writes.
	folders := app.Group( "/api/folders" , handlers.Guard.RequireLogin , handlers.MemberPrincipal )
	folders.Post( "" , handlers.CreateFolder )
	folders.Post( "/order" , handlers.SetFolderOrder )
	folders.Post( "/:folder_id" , handlers.RenameFolder )
	folders.Post( "/:folder_id/delete" , handlers.DeleteFolder )

	// API keys: any signed-in user manages their own.
	//
	// Note the group prefix. A group's middleware is mounted on a path
	// *prefix*, not on the routes written under it, so it applies to every
	// route registered after it whose path starts with that prefix. Mounting
	// these two on "/api" would put RequireSession in front of /api/admin/users
	// as well, and an admin-scoped API key would then be refused there. That is
	// also why the prefix is exactly "/api/keys": a future "/api/keysets" would
	// pick up this middleware by sharing the string, so keep new routes off
	// these prefixes unless they want these rules.
	keys := app.Group( "/api/keys" , handlers.RequireAPIKeys , handlers.Guard.RequireLogin , handlers.Guard.RequireSession )
	keys.Get( "" , handlers.ListMyAPIKeys )
	keys.Post( "" , handlers.CreateMyAPIKey )
	keys.Post( "/:key_id/revoke" , handlers.RevokeMyAPIKey )
	keys.Post( "/:key_id/delete" , handlers.DeleteMyAPIKey )

	// Admin only: deciding who gets in. Note this is the *only* thing admin
	// rights gate in the base template -- see architecture.md on defaulting
	// new features to "any signed-in user".
	//
	// The two routes that hand out a login link also require a browser
	// session, attached per route rather than on a group prefix. A login link
	// becomes a session, and a session can mint API keys -- so if an
	// admin-scoped key could create accounts or reissue links, a leaked key
	// could turn itself into a fresh session and a fresh key, and revoking
	// the leaked one would not end the compromise. The rule is: an API key
	// may use what an account can do, but it never creates credentials.
	//
	// Deleting an account requires a session too, for a different reason: it
	// cannot be undone. A leaked admin key can disable accounts, which an
	// admin can reverse; it must not be able to erase them. Who may delete
	// whom (the owner rules) is enforced in models.DeleteUser.
	admin := app.Group( "/api/admin" , handlers.Guard.RequireAdmin )
	admin.Get( "/users" , handlers.ListUsers )
	admin.Post( "/users" , handlers.Guard.RequireSession , handlers.CreateUser )
	admin.Post( "/users/:user_id/reissue-login" , handlers.Guard.RequireSession , handlers.ReissueLogin )
	admin.Post( "/users/:user_id/disabled" , handlers.SetUserDisabled )
	admin.Post( "/users/:user_id/revoke-access" , handlers.RevokeAccess )
	admin.Post( "/users/:user_id/delete" , handlers.Guard.RequireSession , handlers.DeleteUser )

	// Invite links: one link for a group, each person who opens it joins as a
	// new user (see invite_links.go). Creating one needs a session for the
	// same reason as creating a user -- the link mints accounts and sessions.
	// Deleting one needs a session like deleting a user. Listing and
	// switching one off are safe for an admin key.
	admin.Get( "/invite-links" , handlers.ListInviteLinks )
	admin.Post( "/invite-links" , handlers.Guard.RequireSession , handlers.CreateInviteLink )
	admin.Post( "/invite-links/:link_id/revoke" , handlers.RevokeInviteLink )
	admin.Post( "/invite-links/:link_id/delete" , handlers.Guard.RequireSession , handlers.DeleteInviteLink )

	// Admin view of every key in the app -- "who has standing access" -- and
	// the ability to revoke or delete someone else's. Its own prefix for the
	// same reason as above: mounted on "/api/admin" it would gate the account
	// routes too.
	admin_keys := app.Group( "/api/admin/keys" , handlers.RequireAPIKeys , handlers.Guard.RequireAdmin , handlers.Guard.RequireSession )
	admin_keys.Get( "" , handlers.ListAllAPIKeys )
	admin_keys.Post( "/:key_id/revoke" , handlers.RevokeAnyAPIKey )
	admin_keys.Post( "/:key_id/delete" , handlers.DeleteAnyAPIKey )

	// MCP: agents calling the app with an API key (see server/mcp and
	// mcp_tools.go). Keys only -- RequireAPIKey turns a browser session away
	// -- and every tool runs at the key's effective role and through
	// server/access, so this is the API above in another shape, not a wider
	// one. One exact path, not a group: a group on "/mcp" would also catch
	// any later "/mcp..." route.
	if handlers.Config.MCPEnabled && handlers.Config.APIKeysEnabled {
		mcp_server := mcp.New( handlers.Config.ProjectName , handlers.Version )
		handlers.registerMCPTools( mcp_server )
		app.Post( "/mcp" , handlers.Guard.RequireAPIKey , mcp_server.Handle )
		app.Get( "/mcp" , mcp_server.MethodNotAllowed )
		app.Delete( "/mcp" , mcp_server.MethodNotAllowed )
	}

	// Catch-all static serving. Must be registered last -- it matches every
	// remaining path, so anything added after this line never runs.
	app.Get( "/*" , handlers.Static.Handler )
}

// registerBoardContent mounts the routes members and guests share: reading a
// board and editing its content. The same handlers answer both; the
// principal middleware on each group is the only difference.
func ( handlers *Handlers ) registerBoardContent( group fiber.Router ) {
	group.Get( "/:board_id" , handlers.GetBoard )
	group.Get( "/:board_id/stamp" , handlers.GetBoardStamp )
	group.Get( "/:board_id/export" , handlers.ExportBoard )
	group.Post( "/:board_id" , handlers.UpdateBoard )
	group.Post( "/:board_id/lists" , handlers.CreateList )
	group.Post( "/:board_id/lists/:list_id" , handlers.UpdateList )
	group.Post( "/:board_id/lists/:list_id/delete" , handlers.DeleteList )
	group.Post( "/:board_id/lists/:list_id/archive-cards" , handlers.ArchiveListCards )
	group.Get( "/:board_id/lists/:list_id/export" , handlers.ExportColumn )
	group.Post( "/:board_id/lists/:list_id/import" , handlers.ImportColumn )
	group.Post( "/:board_id/labels" , handlers.CreateLabel )
	group.Post( "/:board_id/labels/:label_id" , handlers.UpdateLabel )
	group.Post( "/:board_id/labels/:label_id/delete" , handlers.DeleteLabel )
	group.Post( "/:board_id/cards" , handlers.CreateCard )
	group.Post( "/:board_id/cards/:card_id" , handlers.PatchCard )
	group.Post( "/:board_id/cards/:card_id/delete" , handlers.DeleteCard )
	group.Get( "/:board_id/cards/:card_id/product-image" , handlers.ProductImage )
	group.Post( "/:board_id/cards/:card_id/product/refresh" , handlers.RefreshProduct )
	group.Post( "/:board_id/cards/:card_id/product/delete" , handlers.RemoveProduct )
	group.Get( "/:board_id/cards/:card_id/image" , handlers.CardImage )
	group.Post( "/:board_id/cards/:card_id/image" , handlers.SetCardImage )
	group.Post( "/:board_id/cards/:card_id/image/delete" , handlers.RemoveCardImage )
	group.Get( "/:board_id/trash" , handlers.ListTrash )
	group.Post( "/:board_id/trash/empty" , handlers.EmptyTrash )
	group.Post( "/:board_id/trash/:card_id/restore" , handlers.RestoreCard )
	group.Post( "/:board_id/trash/:card_id/delete" , handlers.PurgeCard )
	group.Post( "/:board_id/cards/:card_id/checklist" , handlers.AddChecklistItem )
	group.Post( "/:board_id/cards/:card_id/checklist/:item_id" , handlers.UpdateChecklistItem )
	group.Post( "/:board_id/cards/:card_id/checklist/:item_id/delete" , handlers.DeleteChecklistItem )
}

// badRequest / notFound keep error bodies uniform, so the frontend has one
// shape to handle and no handler accidentally leaks internal detail.
func badRequest( c fiber.Ctx , message string ) ( err error ) {
	err = c.Status( fiber.StatusBadRequest ).JSON( fiber.Map{ "error": message } )
	return
}

func notFound( c fiber.Ctx , message string ) ( err error ) {
	err = c.Status( fiber.StatusNotFound ).JSON( fiber.Map{ "error": message } )
	return
}

func forbidden( c fiber.Ctx , message string ) ( err error ) {
	err = c.Status( fiber.StatusForbidden ).JSON( fiber.Map{ "error": message } )
	return
}

func serverError( c fiber.Ctx ) ( err error ) {
	err = c.Status( fiber.StatusInternalServerError ).JSON( fiber.Map{ "error": "internal error" } )
	return
}
