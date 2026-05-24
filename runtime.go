package studio

import (
	"bytes"
	"embed"
	"net/http"
	"net/url"
	"strings"
	"time"
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
	if len(preview) == 0 {
		return engines
	}
	if len(engines) == 0 {
		return preview
	}
	out := make([]byte, 0, len(preview)+1+len(engines))
	out = append(out, preview...)
	out = append(out, '\n')
	out = append(out, engines...)
	return out
}

func Stylesheet() []byte {
	return readRuntime("assets/studio.css")
}

func EngineRuntimeHandler() http.Handler {
	return ScriptHandler("studio-engines.js", EngineRuntimeScript())
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
