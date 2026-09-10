package sitehost

import (
	"bytes"
	_ "embed"
	"net/http"
	"time"
)

//go:embed assets/editor.js
var editorScript []byte

const editorScriptPath = "/_gosx/site/editor.js"

var editorScriptModTime = time.Now()

func editorScriptHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
		w.Header().Set("Cache-Control", "public, max-age=300")
		http.ServeContent(w, r, "editor.js", editorScriptModTime, bytes.NewReader(editorScript))
	})
}
