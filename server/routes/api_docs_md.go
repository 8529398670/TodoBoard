package routes

import (
	fiber "github.com/gofiber/fiber/v3"

	apidoc "todoboard/server/apidoc"
)

// getAPIDoc hands out config/api.md, the API reference written as tables for
// people, which scripts/api_docs.sh writes beside openapi.json. The /docs/api
// page renders it when app.repo is not set, so the API keys card always has a
// readable reference to link to. Signed-in callers only, like openapi.json.
//
// @route("GET /api/api.md", summary="The API reference as Markdown tables for people",
//     tag="Monitoring", feature="openapi", errors=(404,), reply_type="text/markdown",
//     mcp_http="The API reference itself, for people; GET /api/openapi.json is the same as JSON.",
// )
func ( handlers *Handlers ) getAPIDoc( c fiber.Ctx , in *apidoc.None ) ( err error ) {
	err = handlers.sendMarkdown( c , handlers.APIDoc )
	return
}

// getMCPDoc hands out config/mcp.md, the MCP tools as tables for people. The
// /docs/mcp page renders it when app.repo is not set.
//
// @route("GET /api/mcp.md", summary="The MCP tools as Markdown tables for people",
//     tag="Monitoring", feature="mcp", errors=(404,), reply_type="text/markdown",
//     mcp_http="The MCP reference itself, for people; tools/list gives the same list.",
// )
func ( handlers *Handlers ) getMCPDoc( c fiber.Ctx , in *apidoc.None ) ( err error ) {
	err = handlers.sendMarkdown( c , handlers.MCPDoc )
	return
}

func ( handlers *Handlers ) sendMarkdown( c fiber.Ctx , body []byte ) ( err error ) {
	if len( body ) == 0 {
		err = notFound( c , "not found" )
		return
	}
	c.Set( fiber.HeaderContentType , "text/markdown; charset=utf-8" )
	err = c.Send( body )
	return
}

// ShowDocsPage is /docs/api and /docs/mcp (docs.js shows an error for any other
// /docs/ path): the readable reference, rendered
// in the browser from GET /api/api.md or /api/mcp.md.
func ( handlers *Handlers ) ShowDocsPage( c fiber.Ctx ) ( err error ) {
	c.Set( fiber.HeaderReferrerPolicy , "no-referrer" )
	c.Set( "X-Robots-Tag" , "noindex" )
	err = handlers.Static.ServeFile( c , "docs.html" )
	return
}

// apiDocURL is where the readable API reference is read: config/api.md in
// app.repo when that is set, so the repository's site renders it, else this
// server's own page.
func ( handlers *Handlers ) apiDocURL() ( result string ) {
	result = handlers.cfg().RepoFile( "config/api.md" )
	if result == "" {
		result = "/docs/api"
	}
	return
}

// mcpDocURL is the same for config/mcp.md.
func ( handlers *Handlers ) mcpDocURL() ( result string ) {
	result = handlers.cfg().RepoFile( "config/mcp.md" )
	if result == "" {
		result = "/docs/mcp"
	}
	return
}
