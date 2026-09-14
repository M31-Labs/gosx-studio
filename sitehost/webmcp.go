package sitehost

import (
	_ "embed"
	"net/http"

	"m31labs.dev/gosx"
)

// webmcp.go serves the script that announces the editor's and admin's
// tools to a browser with a built-in assistant (WebMCP). The tools call
// the agent API as the signed-in person; see agentAuth.

const webMCPScriptPath = "/_gosx/site/webmcp.js"

//go:embed assets/webmcp.js
var webMCPScriptBody []byte

func webMCPScriptHandler() http.Handler {
	return assetHandler("webmcp.js", webMCPScriptBody, scriptContentType)
}

// webMCPScript is the tag every admin page and the editor carry.
func webMCPScript() gosx.Node {
	return gosx.El("script", gosx.Attrs(gosx.Attr("src", webMCPScriptURL), gosx.Attr("defer", "defer")))
}
