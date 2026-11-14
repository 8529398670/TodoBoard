package routes

import (
	mcp "todoboard/server/mcp"
)

// The tools /mcp offers (see server/mcp for the protocol and the rules a tool
// follows). A tool is declared with its route, by mcp="tool_name" in the
// handler's @route(...), and the route table (routes_generated.go) registers it with
// mcp.ForRoute: the same handler, the same checks on its arguments, so the
// two doors answer the same way and refuse for the same reasons. A tool that
// adds to its route's answer (where to send a file's bytes) names a call=
// that does; those live in mcp_tools_accounts.go.
//
// The rule: every route an API key may call has a tool, or an mcp_http
// reason in its declaration for staying HTTP only. mcp_parity_test.go fails
// until that holds. What a key may not call -- managing API keys, setting
// your own password, PIN or two-factor, and the owner's screens -- needs a browser session (access=SessionOnly), so it has
// no tool either.
//
// After adding or changing a tool, run scripts/api_docs.sh to rewrite the
// route table, config/mcp.json, config/api.md and config/mcp.md; the tests fail until they
// are current.
//
// An app's tools that stand for no route go in a new file in this package
// (mcp_tools_<feature>.go), registered from registerMCPTools below.

func ( handlers *Handlers ) registerMCPTools( server *mcp.Server ) {
	handlers.registerDeclaredTools( server )
}
