// Package runtimeasset serves immutable snapshots of browser runtime assets
// with validators suitable for mutable, unfingerprinted URLs.
package runtimeasset

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"time"
)

// Handler returns an HTTP handler for a browser asset.
//
// The asset is copied when the handler is constructed so callers can safely
// reuse or modify their input slice after construction. Its validator is
// derived from that same snapshot, making the ETag stable for the served
// representation. A weak validator remains valid when an outer middleware
// applies content encoding to the response.
//
// Runtime assets use mutable, unfingerprinted URLs, so responses explicitly
// ask caches to revalidate. A zero modtime is passed to http.ServeContent to
// avoid advertising a synthetic Last-Modified date; ServeContent still owns
// the standard conditional, range, and HEAD behavior.
func Handler(name string, data []byte, contentType string) http.Handler {
	snapshot := bytes.Clone(data)
	etag := contentETag(snapshot)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		header := w.Header()
		header.Set("Content-Type", contentType)
		header.Set("X-Content-Type-Options", "nosniff")
		header.Set("Cache-Control", "no-cache")
		header.Set("ETag", etag)
		http.ServeContent(w, r, name, time.Time{}, bytes.NewReader(snapshot))
	})
}

func contentETag(data []byte) string {
	digest := sha256.Sum256(data)
	return `W/"` + hex.EncodeToString(digest[:]) + `"`
}
