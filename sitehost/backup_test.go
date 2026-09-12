package sitehost

import (
	"archive/zip"
	"bytes"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Backups run inside the request under test, never in a goroutine that
// could outlive the test's temp dir.
func init() { backupAsync = false }

func readZip(t *testing.T, data []byte) map[string]string {
	t.Helper()
	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatalf("not a zip: %v", err)
	}
	out := map[string]string{}
	for _, file := range reader.File {
		rc, err := file.Open()
		if err != nil {
			t.Fatal(err)
		}
		body, _ := io.ReadAll(rc)
		rc.Close()
		out[file.Name] = string(body)
	}
	return out
}

func TestExportHoldsTheWholeSiteAndNothingSecret(t *testing.T) {
	host, handler := newTestHost(t)
	imgURL := uploadedURL(t, postUpload(t, handler, "hero.png", bigPNG(t, 300, 200)).Body.String())
	post(t, handler, "/contact/send", url.Values{"page": {"/contact"}, "name": {"Sam"}, "email": {"sam@example.com"}, "message": {"Keep me"}})
	createForm(t, handler, "Quotes", "blank")
	// A certificate cache that must never leave the server.
	certs := host.Options().certDir()
	os.MkdirAll(certs, 0o755)
	os.WriteFile(filepath.Join(certs, "wildflower.example"), []byte("PRIVATE KEY"), 0o600)

	rec := get(t, handler, "/admin/export.zip")
	if rec.Code != http.StatusOK || rec.Header().Get("Content-Type") != "application/zip" {
		t.Fatalf("export = %d %s", rec.Code, rec.Header().Get("Content-Type"))
	}
	mustContain(t, rec.Header().Get("Content-Disposition"), `filename="wildflower-bakery-export-`, "the download is named after the site")
	files := readZip(t, rec.Body.Bytes())
	for _, want := range []string{"site/README.txt", "site/site.json", "site/messages.json", "site/forms.json", "site/uploads/" + strings.TrimPrefix(imgURL, "/uploads/")} {
		if _, ok := files[want]; !ok {
			t.Fatalf("export lacks %s; has %v", want, keys(files))
		}
	}
	mustContain(t, files["site/messages.json"], "Keep me", "messages travel with the site")
	mustContain(t, files["site/README.txt"], "gosx-site -data site/site.json", "the README says how to restore")
	for name := range files {
		if strings.Contains(name, "certs") || strings.Contains(files[name], "PRIVATE KEY") {
			t.Fatalf("export must not include certificates: %s", name)
		}
	}
}

func keys(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

func TestDailyBackupsAreMadeListedDownloadedAndPruned(t *testing.T) {
	base := time.Date(2026, 9, 12, 9, 0, 0, 0, time.UTC)
	timeNow = func() time.Time { return base }
	t.Cleanup(func() { timeNow = time.Now })
	host, err := Open(Options{DataPath: filepath.Join(t.TempDir(), "site.json"), SiteTitle: "Wildflower", SiteKind: "food", Seed: true})
	if err != nil {
		t.Fatal(err)
	}
	handler := host.Handler()

	// The first request of the day makes the day's backup.
	settings := get(t, handler, "/admin/settings").Body.String()
	backups := host.listBackups()
	if len(backups) != 1 || backups[0].Name != "site-2026-09-12-0900.zip" {
		t.Fatalf("first backup = %+v", backups)
	}
	mustContain(t, settings, "Backups and export", "Settings has the panel")

	// Not due again for a day, however many requests come in.
	timeNow = func() time.Time { return base.Add(5 * time.Hour) }
	get(t, handler, "/")
	host.backupIfDue()
	if len(host.listBackups()) != 1 {
		t.Fatal("a second backup within a day")
	}
	timeNow = func() time.Time { return base.Add(25 * time.Hour) }
	get(t, handler, "/")
	if len(host.listBackups()) != 2 {
		t.Fatal("no backup after a day")
	}

	settings = get(t, handler, "/admin/settings").Body.String()
	mustContain(t, settings, `href="/admin/backups/site-2026-09-13-1000.zip"`, "backups are downloadable from Settings")
	rec := get(t, handler, "/admin/backups/site-2026-09-13-1000.zip")
	if rec.Code != http.StatusOK || rec.Header().Get("Content-Type") != "application/zip" {
		t.Fatalf("download = %d", rec.Code)
	}
	if _, ok := readZip(t, rec.Body.Bytes())["site/site.json"]; !ok {
		t.Fatal("a backup is a full export")
	}
	if code := get(t, handler, "/admin/backups/..%2Fsite.json").Code; code != http.StatusNotFound {
		t.Fatalf("path escape = %d", code)
	}

	// "Back up now" works, and old ones are pruned to fourteen.
	dir := host.Options().backupDir()
	for day := 1; day <= 16; day++ {
		os.WriteFile(filepath.Join(dir, "site-2026-08-"+pad2(day)+"-0900.zip"), []byte("old"), 0o600)
	}
	rec = post(t, handler, "/admin/backups", url.Values{})
	mustContain(t, rec.Header().Get("Location"), "Backed+up+as", "the owner is told")
	if got := len(host.listBackups()); got != backupKeep {
		t.Fatalf("after pruning: %d backups, want %d", got, backupKeep)
	}
	if _, err := os.Stat(filepath.Join(dir, "site-2026-08-01-0900.zip")); !os.IsNotExist(err) {
		t.Fatal("the oldest backup should have been pruned")
	}

	// A site with backups off still exports, and says so.
	quiet, _ := Open(Options{DataPath: filepath.Join(t.TempDir(), "site.json"), SiteTitle: "Q", SiteKind: "simple", Seed: true, NoBackups: true})
	mustContain(t, get(t, quiet.Handler(), "/admin/settings").Body.String(), "Automatic backups are turned off", "the panel says backups are off")
	if len(quiet.listBackups()) != 0 {
		t.Fatal("no backups when they are off")
	}
}

func pad2(n int) string {
	if n < 10 {
		return "0" + string(rune('0'+n))
	}
	return string(rune('0'+n/10)) + string(rune('0'+n%10))
}
