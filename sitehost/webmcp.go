package sitehost

import (
	"bytes"
	_ "embed"
	"net/http"
	"time"

	"m31labs.dev/gosx"
)

// webmcp.go serves the script that announces the editor's and admin's
// tools to a browser with a built-in assistant (WebMCP). The tools call
// the agent API as the signed-in person; see agentAuth.

const webMCPScriptPath = "/_gosx/site/webmcp.js"

//go:embed assets/webmcp.js
var webMCPScriptBody []byte

var webMCPScriptModTime = time.Now()

func webMCPScriptHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
		w.Header().Set("Cache-Control", "public, max-age=300")
		http.ServeContent(w, r, "webmcp.js", webMCPScriptModTime, bytes.NewReader(webMCPScriptBody))
	})
}

// webMCPScript is the tag every admin page and the editor carry.
func webMCPScript() gosx.Node {
	return gosx.El("script", gosx.Attrs(gosx.Attr("src", webMCPScriptPath), gosx.Attr("defer", "defer")))
}
