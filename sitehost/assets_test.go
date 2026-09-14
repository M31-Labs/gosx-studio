package sitehost

import (
	"strings"
	"testing"
)

func TestStaticFilesCarryAContentHashSoADeployIsNeverStale(t *testing.T) {
	_, handler := newTestHost(t)
	home := get(t, handler, "/").Body.String()
	for _, want := range []string{`href="` + publicStylesheetURL + `"`, `src="` + effectsScriptURL + `"`} {
		mustContain(t, home, want, "the page names each file by its content hash")
	}
	if !strings.Contains(publicStylesheetURL, publicStylesheetPath+"?v=") || len(publicStylesheetURL) < len(publicStylesheetPath)+10 {
		t.Fatalf("stylesheet url = %q", publicStylesheetURL)
	}
	current := get(t, handler, publicStylesheetURL)
	if got := current.Header().Get("Cache-Control"); got != "public, max-age=31536000, immutable" {
		t.Fatalf("the current version caches for a year, got %q", got)
	}
	mustContain(t, current.Body.String(), ".site-fx-host", "and is the stylesheet")
	for _, path := range []string{publicStylesheetPath, publicStylesheetPath + "?v=old", effectsScriptPath, editorScriptPath, webMCPScriptPath, statsScriptPath, consentScriptPath} {
		rec := get(t, handler, path)
		if rec.Code != 200 || rec.Header().Get("Cache-Control") != "public, max-age=300" {
			t.Fatalf("%s: %d %q, want a short cache for any other name", path, rec.Code, rec.Header().Get("Cache-Control"))
		}
	}
	if assetVersion([]byte("a")) == assetVersion([]byte("b")) {
		t.Fatal("different files, different versions")
	}
}
