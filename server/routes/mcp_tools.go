package routes

import (

	fiber "github.com/gofiber/fiber/v3"

	mcp "todoboard/server/mcp"
	models "todoboard/server/models"
)

// The tools /mcp offers (see server/mcp for the protocol and the rules a tool
// follows). Each one is a thin wrapper over the helper its HTTP route uses --
// meRow, teamRows, setUserDisabled, createFile, ... -- so the two doors answer
// the same way and refuse for the same reasons.
//
// The rule: every route an API key may call has a tool here, named in its
// Route, or an "x-mcp-http" reason in config/openapi.json for staying HTTP
// only. mcp_parity_test.go fails until that holds. What a key may not call
// -- managing users, keys and sign-in, and erasing anything for good -- needs
// a browser session (see RequireSession), so it has no tool either.
//
// After adding or changing a tool, run scripts/mcp_json.sh to rewrite
// config/mcp.json; mcp_manifest_test.go fails until it is current.
//
// An app's own tools go in a new file in this package (mcp_tools_<feature>.go)
// with a registerXTools function called from registerMCPTools below -- the
// board and card tools are in mcp_tools_boards.go.

var userIDSchema = fiber.Map{ "type": "integer" , "minimum": 1 , "description": "The account's id, from list_team or admin_list_users." }

func ( handlers *Handlers ) registerMCPTools( server *mcp.Server ) {
	server.Register( &mcp.Tool{
		Name:        "whoami",
		Route:       "GET /api/me",
		Title:       "Who am I",
		Description: "The account this API key belongs to, the role it acts with (role -- never higher than the key's own scope), and the account's own role.",
		ReadOnly:    true,
		Call: func( call *mcp.Call ) ( result any , err error ) {
			if err = call.Bind( &struct{}{} ); err != nil { return }
			result , err = handlers.meRow( call.Actor )
			return
		},
	} )

	server.Register( &mcp.Tool{
		Name:        "list_team",
		Route:       "GET /api/team",
		Title:       "List team",
		Description: "People with an active account, a page at a time: id and display name. Use the id wherever another tool asks for a user. next is the offset of the following page, or null on the last one.",
		ReadOnly:    true,
		InputSchema: pageSchema,
		Call: func( call *mcp.Call ) ( result any , err error ) {
			result , err = toolPage( call , "members" , handlers.teamRows )
			return
		},
	} )

	server.Register( &mcp.Tool{
		Name:        "admin_list_users",
		Route:       "GET /api/admin/users",
		Title:       "List users (admin)",
		Description: "Accounts, a page at a time, including disabled ones, with each one's role and which is the owner. next is the offset of the following page, or null on the last one.",
		ReadOnly:    true,
		AdminOnly:   true,
		InputSchema: pageSchema,
		Call: func( call *mcp.Call ) ( result any , err error ) {
			result , err = toolPage( call , "users" , handlers.userRows )
			return
		},
	} )

	server.Register( &mcp.Tool{
		Name:        "admin_set_user_disabled",
		Route:       "POST /api/admin/users/{user_id}/disabled",
		Title:       "Disable or enable a user (admin)",
		Description: "Switch an account off (disabled: true) or back on (false). Disabling signs it out everywhere and voids its live login link; its API keys stop working while it is off. Reversible. The owner and your own account cannot be disabled.",
		AdminOnly:   true,
		InputSchema: fiber.Map{
			"type": "object",
			"properties": fiber.Map{
				"user_id":  userIDSchema,
				"disabled": fiber.Map{ "type": "boolean" },
			},
			"required":             []string{ "user_id" , "disabled" },
			"additionalProperties": false,
		},
		Call: func( call *mcp.Call ) ( result any , err error ) {
			var args struct {
				UserID   uint64 `json:"user_id"`
				Disabled *bool  `json:"disabled"`
			}
			if err = call.Bind( &args ); err != nil { return }
			if args.UserID == 0 || args.Disabled == nil {
				err = mcp.Refuse( "user_id and disabled are required" )
				return
			}
			if refused := handlers.setUserDisabled( call.Actor.User.ID , args.UserID , *args.Disabled ); refused != nil {
				err = refusalError( refused )
				return
			}
			result = fiber.Map{ "ok": true }
			return
		},
	} )

	server.Register( &mcp.Tool{
		Name:        "admin_revoke_access",
		Route:       "POST /api/admin/users/{user_id}/revoke-access",
		Title:       "Revoke a user's access (admin)",
		Description: "For an account that may be compromised: ends every session and voids its live login link and every API key it holds, at once. The account stays enabled; someone with a browser session must issue a new login link to let its owner back in. The owner and your own account cannot be revoked.",
		Destructive: true,
		AdminOnly:   true,
		InputSchema: fiber.Map{
			"type":                 "object",
			"properties":           fiber.Map{ "user_id": userIDSchema },
			"required":             []string{ "user_id" },
			"additionalProperties": false,
		},
		Call: func( call *mcp.Call ) ( result any , err error ) {
			var args struct {
				UserID uint64 `json:"user_id"`
			}
			if err = call.Bind( &args ); err != nil { return }
			if args.UserID == 0 {
				err = mcp.Refuse( "user_id is required" )
				return
			}
			revoked , refused := handlers.revokeAccess( call.Actor.User.ID , args.UserID )
			if refused != nil {
				err = refusalError( refused )
				return
			}
			result = fiber.Map{ "ok": true , "revoked": revoked }
			return
		},
	} )

	server.Register( &mcp.Tool{
		Name:        "rename_account",
		Route:       "POST /api/account/rename",
		Title:       "Rename your account",
		Description: "Change your own display name (1-80 characters). expect, when given, is the name you think it has now; if someone changed it meanwhile the rename is refused with the current name, so nothing is overwritten unseen.",
		InputSchema: fiber.Map{
			"type": "object",
			"properties": fiber.Map{
				"display_name": fiber.Map{ "type": "string" },
				"expect":       fiber.Map{ "type": "string" , "description": "The current name, from whoami. Optional." },
			},
			"required":             []string{ "display_name" },
			"additionalProperties": false,
		},
		Call: func( call *mcp.Call ) ( result any , err error ) {
			var args struct {
				DisplayName string  `json:"display_name"`
				Expect      *string `json:"expect"`
			}
			if err = call.Bind( &args ); err != nil { return }
			result , err = answer( handlers.renameAccount( call.Ctx , args.DisplayName , args.Expect ) )
			return
		},
	} )


	server.Register( &mcp.Tool{
		Name:        "admin_list_invite_links",
		Route:       "GET /api/admin/invite-links",
		Title:       "List invite links (admin)",
		Description: "Group invite links, a page at a time: label, places used and left, expiry, and whether each still works. The links themselves are never shown again after creation. next is the offset of the following page, or null on the last one.",
		ReadOnly:    true,
		AdminOnly:   true,
		Enabled:     handlers.inviteLinksOn,
		InputSchema: pageSchema,
		Call: func( call *mcp.Call ) ( result any , err error ) {
			result , err = toolPage( call , "invite_links" , handlers.inviteLinkRows )
			return
		},
	} )

	server.Register( &mcp.Tool{
		Name:        "admin_revoke_invite_link",
		Route:       "POST /api/admin/invite-links/{link_id}/revoke",
		Title:       "Switch off an invite link (admin)",
		Description: "Stop an invite link from letting anyone else join or sign in through it. Accounts it already made keep working. Cannot be switched back on.",
		Destructive: true,
		AdminOnly:   true,
		Enabled:     handlers.inviteLinksOn,
		InputSchema: fiber.Map{
			"type": "object",
			"properties": fiber.Map{
				"link_id": fiber.Map{ "type": "string" , "description": "The id from admin_list_invite_links." },
			},
			"required":             []string{ "link_id" },
			"additionalProperties": false,
		},
		Call: func( call *mcp.Call ) ( result any , err error ) {
			var args struct {
				LinkID string `json:"link_id"`
			}
			if err = call.Bind( &args ); err != nil { return }
			if models.RevokeInviteLink( handlers.Store , args.LinkID ) != nil {
				err = mcp.Refuse( "no such invite link" )
				return
			}
			result = fiber.Map{ "ok": true }
			return
		},
	} )

	handlers.registerBoardTools( server )
	handlers.registerTransferTools( server )
}

// answer hands a shared helper's result to MCP: the payload, or its refusal
// in the words the HTTP route would use.
func answer( payload fiber.Map , refused *refusal ) ( result any , err error ) {
	if refused != nil {
		err = refusalError( refused )
		return
	}
	result = payload
	return
}


// refusalError hands a shared rule's refusal to MCP in the same words the
// HTTP route would use. A server-side failure stays a plain error, so it is
// logged and reported only as "internal error".
func refusalError( refused *refusal ) ( err error ) {
	if refused.status >= 500 {
		err = refused.cause
		if err == nil {
			err = fiber.NewError( refused.status , refused.message )
		}
		return
	}
	err = mcp.Refuse( refused.message )
	return
}
