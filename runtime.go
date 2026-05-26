package studio

import (
	"bytes"
	"embed"
	"net/http"
	"net/url"
	"strings"
	"time"

	"m31labs.dev/gosx-studio/blocklayoutruntime"
	"m31labs.dev/gosx-studio/brandruntime"
	"m31labs.dev/gosx-studio/fieldruntime"
	"m31labs.dev/gosx-studio/previewruntime"
	"m31labs.dev/gosx-studio/selectionruntime"
	"m31labs.dev/gosx-studio/styleruntime"
)

//go:embed assets/preview-runtime.js assets/studio-engines.js assets/studio.css
var runtimeFS embed.FS

func PreviewRuntimeScript() []byte {
	return readRuntime("assets/preview-runtime.js")
}

func StudioEngineRuntimeScript() []byte {
	return readRuntime("assets/studio-engines.js")
}

func EngineRuntimeScript() []byte {
	preview := PreviewRuntimeScript()
	engines := StudioEngineRuntimeScript()
	// Phase 3 burn-down append: each slice's island runtime + bridge shim
	// gets concatenated here so the legacy and island paths ship together
	// in one bundle, with a feature-flag selecting which runs (see
	// ~/.hyphae/spaces/m31labs-gosx/plans/2026-05-25-phase-3-slice-1-fieldruntime.md
	// Section C). Order: legacy bundle (preview + engines) first so the
	// island runtime can reference its globals (window.GoSXStudioPreviewRuntime
	// for the mirroring fallback). Slices 2..7 will append their bundles
	// the same way.
	slices := [][]byte{
		fieldruntime.Bundle(),
		selectionruntime.Bundle(),
		brandruntime.Bundle(),
		blocklayoutruntime.Bundle(),
		styleruntime.Bundle(),
		previewruntime.Bundle(),
	}
	totalSliceLen := 0
	for _, slice := range slices {
		totalSliceLen += len(slice) + 1
	}
	if len(preview) == 0 && len(engines) == 0 {
		// No legacy bundle at all; just return the slice bundles joined.
		out := make([]byte, 0, totalSliceLen)
		for _, slice := range slices {
			if len(slice) == 0 {
				continue
			}
			out = append(out, slice...)
			out = append(out, '\n')
		}
		return out
	}
	out := make([]byte, 0, len(preview)+1+len(engines)+totalSliceLen)
	if len(preview) > 0 {
		out = append(out, preview...)
		out = append(out, '\n')
	}
	if len(engines) > 0 {
		out = append(out, engines...)
		out = append(out, '\n')
	}
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
