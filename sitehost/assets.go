package sitehost

import (
	"bytes"
	_ "embed"
	"net/http"
	"time"
)

//go:embed assets/editor.js
var editorScript []byte

//go:embed assets/consent.js
var consentScript []byte

//go:embed assets/stats.js
var statsScript []byte

const editorScriptPath = "/_gosx/site/editor.js"

var editorScriptModTime = time.Now()

func editorScriptHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
		w.Header().Set("Cache-Control", "public, max-age=300")
		http.ServeContent(w, r, "editor.js", editorScriptModTime, bytes.NewReader(editorScript))
	})
}

func consentScriptHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
		w.Header().Set("Cache-Control", "public, max-age=3600")
		http.ServeContent(w, r, "consent.js", editorScriptModTime, bytes.NewReader(consentScript))
	})
}

func statsScriptHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
		w.Header().Set("Cache-Control", "public, max-age=3600")
		http.ServeContent(w, r, "stats.js", editorScriptModTime, bytes.NewReader(statsScript))
	})
}
