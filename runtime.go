package studio

import (
	"bytes"
	"embed"
	"net/http"
	"net/url"
	"strings"
	"time"

	"m31labs.dev/gosx-studio/authoringruntime"
	"m31labs.dev/gosx-studio/blocklayoutruntime"
	"m31labs.dev/gosx-studio/brandruntime"
	"m31labs.dev/gosx-studio/fieldruntime"
	"m31labs.dev/gosx-studio/inspectorruntime"
	"m31labs.dev/gosx-studio/previewruntime"
	"m31labs.dev/gosx-studio/selectionruntime"
	"m31labs.dev/gosx-studio/sitemapruntime"
	"m31labs.dev/gosx-studio/styleruntime"
	"m31labs.dev/gosx-studio/workbenchruntime"
)

//go:embed assets/studio.css
var runtimeFS embed.FS

// EngineRuntimeScript returns the concatenated island runtime bundles for
// every Phase 3 slice. The legacy monolithic JS bundles (studio-engines.js
// and preview-runtime.js) were deleted on 2026-05-27 — the island paths
// now own every runtime contract by default. See
// ~/.hyphae/spaces/m31labs-gosx/plans/gosx-vm-unification-and-editor-bridge-burn-down.md
// Phase 3 Section E for the burn-down history.
func EngineRuntimeScript() []byte {
	slices := [][]byte{
		fieldruntime.Bundle(),
		selectionruntime.Bundle(),
		brandruntime.Bundle(),
		blocklayoutruntime.Bundle(),
		styleruntime.Bundle(),
		workbenchruntime.Bundle(),
		previewruntime.Bundle(),
		authoringruntime.Bundle(),
		sitemapruntime.Bundle(),
		inspectorruntime.Bundle(),
	}
	total := 0
	for _, slice := range slices {
		if len(slice) > 0 {
			total += len(slice) + 1
		}
	}
	out := make([]byte, 0, total)
	for _, slice := range slices {
		if len(slice) == 0 {
			continue
		}
		out = append(out, slice...)
		out = append(out, '\n')
	}
	return out
}

func Stylesheet() []byte {
	return readRuntime("assets/studio.css")
}

func EngineRuntimeHandler() http.Handler {
	return ScriptHandler("studio-engines.js", EngineRuntimeScript())
}

// PreviewSubscriberScript returns the JS bundle the storefront mounts
// when loaded inside the editor preview iframe. Unlike the editor-side
// engine bundle, this script is intended for the storefront page —
// public storefront visitors never load it (the iframe's preview-mode
// bootstrap per ADR 0009 is gated on ?gosx-preview=1 query param).
//
// The bundle wires window.__gosx_observe_shared_signal subscribers
// against the $preview.* signal namespace; the cross-frame postMessage
// relay (gosx/client/bridge/cross_frame.go) delivers writes from the
// editor frame, and the subscribers apply the corresponding DOM
// mutations locally inside the iframe.
//
// See ~/.hyphae/spaces/m31labs-gosx/plans/2026-05-26-phase-3-slice-6-previewruntime.md
// Section F and the package docstring at
// gosx-studio/previewruntime/runtime.go for the contract.
func PreviewSubscriberScript() []byte {
	return previewruntime.PreviewSubscriberScript()
}

// AuthoringRuntimeScript returns the browser runtime that reacts to successful
// Studio authoring action results. It is included in EngineRuntimeScript for
// hosts that serve the Studio engine bundle, and is exposed separately for
// hosts that still compose their editor chrome from gosx-cms/studio helpers.
func AuthoringRuntimeScript() []byte {
	return authoringruntime.Bundle()
}

// SiteMapRuntimeScript returns the browser runtime that owns site-map board
// filters, detail tabs, palette state, and selection synchronization for hosts
// that have not yet switched to the full Studio engine bundle.
func SiteMapRuntimeScript() []byte {
	return sitemapruntime.Bundle()
}

// PreviewSubscriberHandler serves PreviewSubscriberScript() at
// shell.PreviewSubscriberPath. Storefront pages reference this URL in
// a <script defer> tag emitted by the preview-mode bootstrap.
func PreviewSubscriberHandler() http.Handler {
	return ScriptHandler("preview-subscriber.js", PreviewSubscriberScript())
}

func StylesheetHandler() http.Handler {
	return AssetHandler("studio.css", Stylesheet(), "text/css; charset=utf-8")
}

func ScriptHandler(name string, data []byte) http.Handler {
	return AssetHandler(name, data, "text/javascript; charset=utf-8")
}

func AssetHandler(name string, data []byte, contentType string) http.Handler {
	modtime := RuntimeAssetModTime()
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", contentType)
		w.Header().Set("X-Content-Type-Options", "nosniff")
		http.ServeContent(w, r, name, modtime, bytes.NewReader(data))
	})
}

func RuntimeAssetModTime() time.Time {
	return time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
}

func AssetHref(path string, version string) string {
	version = strings.TrimSpace(version)
	if version == "" {
		return path
	}
	separator := "?"
	if strings.Contains(path, "?") {
		separator = "&"
	}
	return path + separator + "v=" + url.QueryEscape(version)
}

func readRuntime(path string) []byte {
	data, err := runtimeFS.ReadFile(path)
	if err != nil {
		return nil
	}
	return data
}
