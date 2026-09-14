package sitehost

import (
	"bytes"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"net/http"
	"time"

	"m31labs.dev/gosx-studio/hostruntime"
)

// assets.go serves the site's static files. Every file carries a short
// content hash on its URL, so a deploy never serves a stale copy from a
// browser or a CDN: the URL changes with the file, and a matching copy may
// be cached for a year.

//go:embed assets/editor.js
var editorScript []byte

//go:embed assets/consent.js
var consentScript []byte

//go:embed assets/effects.js
var effectsScript []byte

//go:embed assets/stats.js
var statsScript []byte

const (
	editorScriptPath  = "/_gosx/site/editor.js"
	effectsScriptPath = "/_gosx/site/effects.js"
)

var (
	editorScriptURL     = assetHref(editorScriptPath, editorScript)
	effectsScriptURL    = assetHref(effectsScriptPath, effectsScript)
	consentScriptURL    = assetHref(consentScriptPath, consentScript)
	statsScriptURL      = assetHref(statsScriptPath, statsScript)
	webMCPScriptURL     = assetHref(webMCPScriptPath, webMCPScriptBody)
	publicStylesheetURL = assetHref(publicStylesheetPath, []byte(siteCSS))
)

var editorScriptModTime = time.Now()

// assetVersion is the content hash that rides on an asset's URL.
func assetVersion(body []byte) string {
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:6])
}

func assetHref(path string, body []byte) string {
	return hostruntime.AssetHref(path, assetVersion(body))
}

// assetHandler serves one file. A request that names the current version
// gets an immutable, year-long cache; any other request gets five minutes.
func assetHandler(name string, body []byte, contentType string) http.Handler {
	version := assetVersion(body)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", contentType)
		w.Header().Set("X-Content-Type-Options", "nosniff")
		if r.URL.Query().Get("v") == version {
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		} else {
			w.Header().Set("Cache-Control", "public, max-age=300")
		}
		http.ServeContent(w, r, name, editorScriptModTime, bytes.NewReader(body))
	})
}

const scriptContentType = "text/javascript; charset=utf-8"

func editorScriptHandler() http.Handler {
	return assetHandler("editor.js", editorScript, scriptContentType)
}
func effectsScriptHandler() http.Handler {
	return assetHandler("effects.js", effectsScript, scriptContentType)
}
func consentScriptHandler() http.Handler {
	return assetHandler("consent.js", consentScript, scriptContentType)
}
func statsScriptHandler() http.Handler {
	return assetHandler("stats.js", statsScript, scriptContentType)
}
