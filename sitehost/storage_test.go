package sitehost

import (
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSQLiteIsTheDefaultStoreAndSurvivesRestart(t *testing.T) {
	dir := t.TempDir()
	host, err := Open(Options{DataPath: filepath.Join(dir, "site.db"), SiteTitle: "Blue Door", SiteKind: "food", Seed: true, NoBackups: true})
	if err != nil {
		t.Fatal(err)
	}
	handler := host.Handler()
	id := firstPageID(t, host, "menu")
	postJSON(t, handler, "/admin/api/pages/"+id, `{"title":"Menu","slug":"menu","description":"x","blocks":[{"kind":"paragraph","text":"Kept in SQLite"}]}`)
	post(t, handler, "/admin/api/pages/"+id+"/publish", url.Values{})
	mustContain(t, get(t, handler, "/menu").Body.String(), "Kept in SQLite", "the page publishes")
	if closer, ok := host.Store().(interface{ Close() error }); ok {
		closer.Close()
	}

	again, err := Open(Options{DataPath: filepath.Join(dir, "site.db"), SiteTitle: "Blue Door", SiteKind: "food", Seed: true, NoBackups: true})
	if err != nil {
		t.Fatal(err)
	}
	mustContain(t, get(t, again.Handler(), "/menu").Body.String(), "Kept in SQLite", "and is there after a restart")
	mustContain(t, get(t, again.Handler(), "/admin/history/page/"+id).Body.String(), ">Published<", "with its history")

	// The export carries a consistent database.
	files := readZip(t, get(t, again.Handler(), "/admin/export.zip").Body.Bytes())
	if _, ok := files["site/site.db"]; !ok {
		t.Fatalf("export lacks site.db: %v", keys(files))
	}
	mustContain(t, files["site/README.txt"], "gosx-site -data site/site.db", "the README names the database")
	copyPath := filepath.Join(t.TempDir(), "site.db")
	os.WriteFile(copyPath, []byte(files["site/site.db"]), 0o600)
	restored, err := Open(Options{DataPath: copyPath, NoBackups: true})
	if err != nil {
		t.Fatal(err)
	}
	mustContain(t, get(t, restored.Handler(), "/menu").Body.String(), "Kept in SQLite", "the exported database opens as the same site")
}

func TestAnOlderJSONSiteMovesIntoSQLite(t *testing.T) {
	dir := t.TempDir()
	old, err := Open(Options{DataPath: filepath.Join(dir, "site.json"), SiteTitle: "Old Timer", SiteKind: "services", Seed: true, NoBackups: true})
	if err != nil {
		t.Fatal(err)
	}
	id := firstPageID(t, old, "services")
	postJSON(t, old.Handler(), "/admin/api/pages/"+id, `{"title":"Services","slug":"services","description":"x","blocks":[{"kind":"paragraph","text":"From the JSON days"}]}`)
	post(t, old.Handler(), "/admin/api/pages/"+id+"/publish", url.Values{})

	moved, err := Open(Options{DataPath: filepath.Join(dir, "site.db"), SiteTitle: "Old Timer", NoBackups: true})
	if err != nil {
		t.Fatal(err)
	}
	if moved.Migrated != filepath.Join(dir, "site.json") {
		t.Fatalf("Migrated = %q", moved.Migrated)
	}
	if _, err := os.Stat(filepath.Join(dir, "site.json.migrated")); err != nil {
		t.Fatal("the JSON file is kept, renamed")
	}
	if _, err := os.Stat(filepath.Join(dir, "site.json")); !os.IsNotExist(err) {
		t.Fatal("the JSON file must not be read twice")
	}
	mustContain(t, get(t, moved.Handler(), "/services").Body.String(), "From the JSON days", "the content moved")
	if !moved.SetupComplete() {
		t.Fatal("settings moved too")
	}
	if strings.TrimSpace(moved.settings().Title) != "Old Timer" {
		t.Fatalf("title after move = %q", moved.settings().Title)
	}
}
