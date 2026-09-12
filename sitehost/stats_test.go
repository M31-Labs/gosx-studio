package sitehost

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	cmsstore "m31labs.dev/gosx-studio/cms/store"
)

const browserUA = "Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X) AppleWebKit/605.1.15 Safari/604.1"

// hitFrom sends one beacon the way stats.js does: a bare POST, no token.
func hitFrom(t *testing.T, handler http.Handler, ip, userAgent, payload string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, statsHitPath, strings.NewReader(payload))
	req.RemoteAddr = ip + ":51234"
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Content-Type", "text/plain")
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("hit = %d %s", rec.Code, rec.Body.String())
	}
	return rec
}

func TestBeaconGoesOnServedPublicPagesOnly(t *testing.T) {
	host, handler := newTestHost(t)
	beacon := `<script src="` + statsScriptPath + `" defer="defer">`
	mustContain(t, get(t, handler, "/").Body.String(), beacon, "the home page carries the beacon")
	mustContain(t, get(t, handler, "/menu").Body.String(), beacon, "so does a live page")
	if strings.Contains(get(t, handler, "/admin").Body.String(), beacon) {
		t.Fatal("the admin must not count itself")
	}
	if strings.Contains(get(t, handler, "/no-such-page").Body.String(), beacon) {
		t.Fatal("a 404 is not a visit")
	}
	script := get(t, handler, statsScriptPath)
	if script.Code != http.StatusOK || !strings.HasPrefix(script.Header().Get("Content-Type"), "text/javascript") {
		t.Fatalf("beacon script = %d %s", script.Code, script.Header().Get("Content-Type"))
	}
	mustContain(t, script.Body.String(), "sendBeacon", "the script beacons")
	if strings.Contains(script.Body.String(), "document.cookie") || strings.Contains(script.Body.String(), "localStorage") {
		t.Fatal("the beacon must not touch cookies or storage")
	}

	if err := host.updateSettingsMetadata(func(m cmsstore.Metadata) { m[statsOffKey] = "true" }); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(get(t, handler, "/").Body.String(), beacon) {
		t.Fatal("turning counting off must remove the beacon")
	}
	mustContain(t, get(t, handler, "/admin/stats").Body.String(), "Visitor counting is off", "the Visitors page says so")
	hitFrom(t, handler, "203.0.113.9", browserUA, `{"p":"/","r":"","w":390}`)
	if s := host.stats.summary(timeNow(), 30); s.Views != 0 {
		t.Fatalf("hits must be dropped while counting is off, got %d", s.Views)
	}
}

func TestHitsCountViewsAndDailyVisitors(t *testing.T) {
	base := time.Date(2026, 9, 12, 15, 0, 0, 0, time.UTC)
	timeNow = func() time.Time { return base }
	t.Cleanup(func() { timeNow = time.Now })
	host, handler := newTestHost(t)

	mustContain(t, get(t, handler, "/admin/stats").Body.String(), "No visits yet", "the Visitors page starts empty")

	// One phone visits twice from Google, a desktop once directly.
	hitFrom(t, handler, "203.0.113.5", browserUA, `{"p":"/menu","r":"https://www.google.com/search?q=bakery","w":390}`)
	hitFrom(t, handler, "203.0.113.5", browserUA, `{"p":"/","r":"https://wildflower.example/menu","w":390}`)
	hitFrom(t, handler, "203.0.113.7", "Mozilla/5.0 (X11; Linux x86_64) Firefox/130.0", `{"p":"/menu","r":"","w":1440}`)
	// Not visitors: a crawler, an admin path, junk.
	hitFrom(t, handler, "203.0.113.8", "Googlebot/2.1 (+http://www.google.com/bot.html)", `{"p":"/","r":"","w":1024}`)
	hitFrom(t, handler, "203.0.113.5", browserUA, `{"p":"/admin/pages","r":"","w":390}`)
	hitFrom(t, handler, "203.0.113.5", browserUA, `not json`)

	summary := host.stats.summary(base, 30)
	if summary.Views != 3 || summary.Visitors != 2 || summary.WeekVisitors != 2 {
		t.Fatalf("views=%d visitors=%d week=%d, want 3/2/2", summary.Views, summary.Visitors, summary.WeekVisitors)
	}
	if len(summary.Pages) == 0 || summary.Pages[0].Label != "/menu" || summary.Pages[0].Count != 2 {
		t.Fatalf("pages = %+v", summary.Pages)
	}
	sources := map[string]int{}
	for _, row := range summary.Sources {
		sources[row.Label] = row.Count
	}
	if sources["google.com"] != 1 || sources["Direct"] != 2 {
		t.Fatalf("sources = %+v (own-site referrers count as direct)", summary.Sources)
	}
	devices := map[string]int{}
	for _, row := range summary.Devices {
		devices[row.Label] = row.Count
	}
	if devices["Phone"] != 2 || devices["Desktop"] != 1 {
		t.Fatalf("devices = %+v", summary.Devices)
	}

	page := get(t, handler, "/admin/stats").Body.String()
	mustContain(t, page, "Visitors, last 30 days", "the page has the headline numbers")
	mustContain(t, page, `<rect class="stats-bar"`, "and a bar chart")
	mustContain(t, page, "12 Sep: 2 visitors, 3 page views", "with a readable tooltip per day")
	mustContain(t, page, "google.com", "sources are listed")
	mustContain(t, page, "Phone", "devices are listed")
	mustContain(t, get(t, handler, "/admin").Body.String(), "Visitors this week", "the dashboard shows the week")

	// Tomorrow the same phone is a new visitor: the salt rotated.
	timeNow = func() time.Time { return base.Add(24 * time.Hour) }
	hitFrom(t, handler, "203.0.113.5", browserUA, `{"p":"/","r":"","w":390}`)
	if s := host.stats.summary(timeNow(), 30); s.Visitors != 3 || s.Views != 4 {
		t.Fatalf("after a day: visitors=%d views=%d, want 3/4", s.Visitors, s.Views)
	}
	if got := host.stats.summary(timeNow(), 1).Visitors; got != 1 {
		t.Fatalf("today alone = %d visitors, want 1", got)
	}

	// Numbers survive a restart; nothing personal is in the file.
	host.stats.flush()
	path := filepath.Join(filepath.Dir(host.Options().DataPath), "stats.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("stats file: %v", err)
	}
	for _, secret := range []string{"203.0.113", "iPhone", "Firefox"} {
		if strings.Contains(string(raw), secret) {
			t.Fatalf("the stats file must not contain %q", secret)
		}
	}
	reopened, err := Open(host.Options())
	if err != nil {
		t.Fatal(err)
	}
	if s := reopened.stats.summary(timeNow(), 30); s.Visitors != 3 || s.Views != 4 {
		t.Fatalf("after reopen: visitors=%d views=%d", s.Visitors, s.Views)
	}
	// And the same phone is still not a new visitor today after the restart.
	hitFrom(t, reopened.Handler(), "203.0.113.5", browserUA, `{"p":"/","r":"","w":390}`)
	if s := reopened.stats.summary(timeNow(), 1); s.Visitors != 1 || s.Views != 2 {
		t.Fatalf("after reopen and a repeat visit: visitors=%d views=%d, want 1/2", s.Visitors, s.Views)
	}
}

func TestStatsSettingsSwitch(t *testing.T) {
	host, handler := newTestHost(t)
	mustContain(t, get(t, handler, "/admin/settings").Body.String(), `name="countVisitors" value="on" checked="checked"`, "counting is on by default")
	postSettings(t, handler, map[string]string{"title": "Wildflower Bakery"}, nil)
	if host.statsEnabled() {
		t.Fatal("saving settings with the box unticked turns counting off")
	}
	postSettings(t, handler, map[string]string{"title": "Wildflower Bakery", "countVisitors": "on"}, nil)
	if !host.statsEnabled() {
		t.Fatal("ticking the box turns counting back on")
	}
}

func TestStatsHelpers(t *testing.T) {
	own := []string{"example.com:8080", "wildflower.example"}
	if got := sourceFromReferrer("https://www.Google.com/search", own); got != "google.com" {
		t.Fatalf("google = %q", got)
	}
	if got := sourceFromReferrer("https://www.wildflower.example/menu", own); got != "Direct" {
		t.Fatalf("own site = %q", got)
	}
	if got := sourceFromReferrer("http://example.com:8080/", own); got != "Direct" {
		t.Fatalf("own host with port = %q", got)
	}
	if got := sourceFromReferrer("garbage", own); got != "Direct" {
		t.Fatalf("garbage = %q", got)
	}
	for width, want := range map[int]string{0: "Unknown", 390: "Phone", 820: "Tablet", 1440: "Desktop"} {
		if got := deviceFromWidth(width); got != want {
			t.Fatalf("width %d = %q, want %q", width, got, want)
		}
	}
	table := map[string]int{}
	for i := 0; i < statsMaxKeys+50; i++ {
		countKey(table, "/junk-"+itoa(i))
	}
	if len(table) != statsMaxKeys+1 || table[statsOtherKey] != 50 {
		t.Fatalf("long tail: %d keys, other=%d", len(table), table[statsOtherKey])
	}
}
