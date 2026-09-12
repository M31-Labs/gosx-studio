package sitehost

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"m31labs.dev/gosx"
	cmsstore "m31labs.dev/gosx-studio/cms/store"
)

// stats.go counts visitors without cookies.
//
// Every hosted-site product answers "is anyone visiting?" with a third-party
// script and a cookie notice. This host answers it itself: a tiny beacon on
// each public page reports the path, the referring site, and the screen
// width; the server keeps daily totals and nothing else. A visitor is counted
// once per day by a hash of their address and browser under a salt that is
// generated fresh each day and never written anywhere but today's file, so
// yesterday's hashes cannot be matched to today's. No cookies, no consent
// notice, no personal data at rest.

const (
	statsHitPath       = "/stats/hit"
	statsScriptPath    = "/_gosx/site/stats.js"
	statsOffKey        = "statsOff"
	statsRetentionDays = 400
	statsMaxKeys       = 200
	statsFlushDelay    = 5 * time.Second
	statsOtherKey      = "(other)"
)

type dayStats struct {
	Views    int            `json:"views"`
	Visitors int            `json:"visitors"`
	Paths    map[string]int `json:"paths,omitempty"`
	Sources  map[string]int `json:"sources,omitempty"`
	Devices  map[string]int `json:"devices,omitempty"`
}

type statsFile struct {
	Days  map[string]*dayStats `json:"days"`
	Today string               `json:"today,omitempty"`
	Salt  string               `json:"salt,omitempty"`
	Seen  []string             `json:"seen,omitempty"`
}

type statsStore struct {
	mu        sync.Mutex
	path      string
	loaded    bool
	data      statsFile
	seen      map[string]bool
	dirty     bool
	lastWrite time.Time
	pending   *time.Timer
}

func newStatsStore(path string) *statsStore {
	return &statsStore{path: path, seen: map[string]bool{}}
}

func (o Options) statsPath() string {
	if o.DataPath == "" {
		return ""
	}
	return filepath.Join(filepath.Dir(o.DataPath), "stats.json")
}

func (s *statsStore) loadLocked() {
	if s.loaded {
		return
	}
	s.loaded = true
	s.data.Days = map[string]*dayStats{}
	if s.path == "" {
		return
	}
	raw, err := os.ReadFile(s.path)
	if err != nil {
		return
	}
	var file statsFile
	if err := json.Unmarshal(raw, &file); err != nil {
		return
	}
	if file.Days == nil {
		file.Days = map[string]*dayStats{}
	}
	s.data = file
	for _, hash := range file.Seen {
		s.seen[hash] = true
	}
}

// writeLocked stores the file atomically and drops days past retention.
func (s *statsStore) writeLocked() {
	s.dirty = false
	s.lastWrite = timeNow()
	if s.path == "" {
		return
	}
	cutoff := timeNow().UTC().AddDate(0, 0, -statsRetentionDays).Format("2006-01-02")
	for day := range s.data.Days {
		if day < cutoff {
			delete(s.data.Days, day)
		}
	}
	s.data.Seen = make([]string, 0, len(s.seen))
	for hash := range s.seen {
		s.data.Seen = append(s.data.Seen, hash)
	}
	sort.Strings(s.data.Seen)
	raw, err := json.Marshal(s.data)
	if err != nil {
		return
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return
	}
	temp := s.path + ".tmp"
	if err := os.WriteFile(temp, raw, 0o600); err != nil {
		return
	}
	_ = os.Rename(temp, s.path)
}

// flush writes now. Hits are coalesced: a busy site writes at most once
// every few seconds, a quiet one right away.
func (s *statsStore) flush() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.loadLocked()
	if s.pending != nil {
		s.pending.Stop()
		s.pending = nil
	}
	s.writeLocked()
}

func (s *statsStore) scheduleWriteLocked() {
	s.dirty = true
	if timeNow().Sub(s.lastWrite) >= statsFlushDelay {
		s.writeLocked()
		return
	}
	if s.pending == nil {
		s.pending = time.AfterFunc(statsFlushDelay, func() {
			s.mu.Lock()
			defer s.mu.Unlock()
			s.pending = nil
			if s.dirty {
				s.writeLocked()
			}
		})
	}
}

// hit is one page view.
type hit struct {
	Path    string
	Source  string
	Device  string
	Visitor string // salted hash, valid for today only
}

func (s *statsStore) record(now time.Time, ip, userAgent string, path, referrer string, width int, ownHosts []string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.loadLocked()

	day := now.UTC().Format("2006-01-02")
	if s.data.Today != day || s.data.Salt == "" {
		s.data.Today = day
		s.data.Salt = randomHex(16)
		s.seen = map[string]bool{}
	}
	entry := s.data.Days[day]
	if entry == nil {
		entry = &dayStats{Paths: map[string]int{}, Sources: map[string]int{}, Devices: map[string]int{}}
		s.data.Days[day] = entry
	}
	for _, m := range []*map[string]int{&entry.Paths, &entry.Sources, &entry.Devices} {
		if *m == nil {
			*m = map[string]int{}
		}
	}

	sum := sha256.Sum256([]byte(s.data.Salt + "|" + ip + "|" + userAgent))
	visitor := hex.EncodeToString(sum[:16])

	entry.Views++
	if !s.seen[visitor] {
		s.seen[visitor] = true
		entry.Visitors++
	}
	countKey(entry.Paths, path)
	countKey(entry.Sources, sourceFromReferrer(referrer, ownHosts))
	countKey(entry.Devices, deviceFromWidth(width))
	s.scheduleWriteLocked()
}

// countKey increments one key, folding the long tail into "(other)" so a
// flood of junk paths cannot grow the file without bound.
func countKey(table map[string]int, key string) {
	if _, known := table[key]; !known && len(table) >= statsMaxKeys {
		key = statsOtherKey
	}
	table[key]++
}

func randomHex(n int) string {
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return strconv.FormatInt(timeNow().UnixNano(), 16)
	}
	return hex.EncodeToString(buf)
}

// sourceFromReferrer is the referring site's host name, "Direct" when there
// is none, and "Direct" too when the referrer is the site's own pages.
func sourceFromReferrer(referrer string, ownHosts []string) string {
	referrer = strings.TrimSpace(referrer)
	if referrer == "" {
		return "Direct"
	}
	parsed, err := url.Parse(referrer)
	if err != nil || parsed.Host == "" {
		return "Direct"
	}
	host := bareHost(parsed.Hostname())
	if host == "" {
		return "Direct"
	}
	for _, own := range ownHosts {
		if host == bareHost(own) {
			return "Direct"
		}
	}
	return host
}

func bareHost(host string) string {
	host = strings.ToLower(strings.TrimSpace(host))
	if h, _, err := net.SplitHostPort(host); err == nil {
		host = h
	}
	return strings.TrimPrefix(host, "www.")
}

// ownHosts is every name this site answers to: the request's host and the
// address in Settings.
func (h *Host) ownHosts(r *http.Request) []string {
	hosts := []string{r.Host}
	if parsed, err := url.Parse(strings.TrimSpace(h.settings().BaseURL)); err == nil && parsed.Host != "" {
		hosts = append(hosts, parsed.Host)
	}
	return hosts
}

func deviceFromWidth(width int) string {
	switch {
	case width <= 0:
		return "Unknown"
	case width < 768:
		return "Phone"
	case width < 1100:
		return "Tablet"
	default:
		return "Desktop"
	}
}

// isBot is a light filter on the user agent; anything that announces itself
// as a crawler is not a visitor.
func isBot(userAgent string) bool {
	ua := strings.ToLower(userAgent)
	if ua == "" {
		return true
	}
	for _, mark := range []string{"bot", "crawl", "spider", "slurp", "headless", "lighthouse", "pingdom", "monitor", "preview", "fetch", "curl/", "wget/", "python-requests", "go-http-client"} {
		if strings.Contains(ua, mark) {
			return true
		}
	}
	return false
}

func clientIP(r *http.Request) string {
	if forwarded := strings.TrimSpace(r.Header.Get("X-Forwarded-For")); forwarded != "" {
		if first := strings.TrimSpace(strings.Split(forwarded, ",")[0]); first != "" {
			return first
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// ---------- HTTP ----------

func (h *Host) mountStats(mux *http.ServeMux) {
	mux.Handle("GET "+statsScriptPath, statsScriptHandler())
	mux.HandleFunc("POST "+statsHitPath, h.handleStatsHit)
	mux.HandleFunc("GET /admin/stats", h.handleAdminStats)
	mux.HandleFunc("GET /admin/stats/{$}", h.handleAdminStats)
}

// statsEnabled is the owner's switch, on unless turned off in Settings.
func (h *Host) statsEnabled() bool {
	return h.settings().Metadata[statsOffKey] != "true"
}

type statsPayload struct {
	Path     string `json:"p"`
	Referrer string `json:"r"`
	Width    int    `json:"w"`
}

func (h *Host) handleStatsHit(w http.ResponseWriter, r *http.Request) {
	// Whatever happens, the browser gets nothing back: this endpoint is
	// fire-and-forget and must never be a way to learn anything.
	w.Header().Set("Cache-Control", "no-store")
	defer w.WriteHeader(http.StatusNoContent)
	if !h.statsEnabled() || !h.SetupComplete() {
		return
	}
	userAgent := r.UserAgent()
	if isBot(userAgent) {
		return
	}
	var payload statsPayload
	raw, err := io.ReadAll(io.LimitReader(r.Body, 4<<10))
	if err != nil || json.Unmarshal(raw, &payload) != nil {
		return
	}
	path := strings.TrimSpace(payload.Path)
	if !strings.HasPrefix(path, "/") || len(path) > 200 || strings.HasPrefix(path, "/admin") || strings.HasPrefix(path, "/setup") {
		return
	}
	h.stats.record(timeNow(), clientIP(r), userAgent, path, payload.Referrer, payload.Width, h.ownHosts(r))
}

// ---------- the summary the owner reads ----------

type statsPoint struct {
	Day      string // 2006-01-02
	Label    string // "12 Sep"
	Views    int
	Visitors int
}

type statsRow struct {
	Label string
	Count int
	Share int // percent of the largest row, for the bar
}

type statsSummary struct {
	Days         []statsPoint
	Views        int
	Visitors     int
	WeekVisitors int
	MaxVisitors  int
	Pages        []statsRow
	Sources      []statsRow
	Devices      []statsRow
	Empty        bool
}

// summary aggregates the last n days, oldest first, ending today.
func (s *statsStore) summary(now time.Time, n int) statsSummary {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.loadLocked()

	out := statsSummary{Days: make([]statsPoint, 0, n)}
	pages, sources, devices := map[string]int{}, map[string]int{}, map[string]int{}
	today := now.UTC()
	for i := n - 1; i >= 0; i-- {
		date := today.AddDate(0, 0, -i)
		key := date.Format("2006-01-02")
		point := statsPoint{Day: key, Label: date.Format("2 Jan")}
		if entry, ok := s.data.Days[key]; ok {
			point.Views, point.Visitors = entry.Views, entry.Visitors
			out.Views += entry.Views
			out.Visitors += entry.Visitors
			if i < 7 {
				out.WeekVisitors += entry.Visitors
			}
			if entry.Visitors > out.MaxVisitors {
				out.MaxVisitors = entry.Visitors
			}
			for k, v := range entry.Paths {
				pages[k] += v
			}
			for k, v := range entry.Sources {
				sources[k] += v
			}
			for k, v := range entry.Devices {
				devices[k] += v
			}
		}
		out.Days = append(out.Days, point)
	}
	out.Pages = topRows(pages, 10)
	out.Sources = topRows(sources, 10)
	out.Devices = topRows(devices, 4)
	out.Empty = out.Views == 0
	return out
}

func topRows(table map[string]int, limit int) []statsRow {
	rows := make([]statsRow, 0, len(table))
	for label, count := range table {
		rows = append(rows, statsRow{Label: label, Count: count})
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].Count != rows[j].Count {
			return rows[i].Count > rows[j].Count
		}
		return rows[i].Label < rows[j].Label
	})
	if len(rows) > limit {
		rows = rows[:limit]
	}
	if len(rows) > 0 && rows[0].Count > 0 {
		max := rows[0].Count
		for i := range rows {
			rows[i].Share = rows[i].Count * 100 / max
		}
	}
	return rows
}

// ---------- the Visitors page ----------

func (h *Host) handleAdminStats(w http.ResponseWriter, r *http.Request) {
	summary := h.stats.summary(timeNow(), 30)
	settings := h.settings()

	var sections []gosx.Node
	if !h.statsEnabled() {
		sections = append(sections, gosx.El("section", gosx.Attrs(gosx.Attr("class", "admin-empty")),
			gosx.El("h2", nil, gosx.Text("Visitor counting is off")),
			gosx.El("p", nil, gosx.Text("Turn it on in Settings to see how many people visit and where they come from. It's anonymous: no cookies, nothing personal stored.")),
			gosx.El("p", nil, gosx.El("a", gosx.Attrs(gosx.Attr("class", "admin-button"), gosx.Attr("href", "/admin/settings")), gosx.Text("Open Settings"))),
		))
	} else if summary.Empty {
		address := strings.TrimRight(firstNonEmpty(settings.BaseURL, "your site's address"), "/")
		sections = append(sections, gosx.El("section", gosx.Attrs(gosx.Attr("class", "admin-empty")),
			gosx.El("h2", nil, gosx.Text("No visits yet")),
			gosx.El("p", nil, gosx.Text("Once people open your site you'll see how many came each day, which pages they read, and where they found you. Share "+address+" to get started.")),
		))
	} else {
		sections = append(sections,
			gosx.El("div", gosx.Attrs(gosx.Attr("class", "admin-stats")),
				adminStat(summary.Visitors, "Visitors, last 30 days"),
				adminStat(summary.Views, "Page views, last 30 days"),
				adminStat(summary.WeekVisitors, "Visitors, last 7 days"),
			),
			gosx.El("section", gosx.Attrs(gosx.Attr("class", "admin-panel")),
				gosx.El("h2", nil, gosx.Text("Visitors per day")),
				renderStatsChart(summary),
			),
			gosx.El("div", gosx.Attrs(gosx.Attr("class", "admin-columns")),
				renderStatsTable("Pages people read", "Page", "Views", summary.Pages),
				renderStatsTable("Where they came from", "Source", "Views", summary.Sources),
				renderStatsTable("What they used", "Device", "Views", summary.Devices),
			),
		)
	}

	body := h.renderAdminShell("stats", "Visitors",
		"Counted without cookies. Nothing personal is stored, so no consent notice is needed for this.",
		adminStatus{}, sections...)
	h.writeDocument(w, http.StatusOK, h.adminMeta("Visitors"), body)
}

// renderStatsChart is an inline SVG bar chart: one bar per day, sized by
// attribute rather than style so it needs no script and no inline CSS.
func renderStatsChart(summary statsSummary) gosx.Node {
	const width, height, top, bottom = 900, 210, 26, 28
	n := len(summary.Days)
	if n == 0 {
		return gosx.Fragment()
	}
	slot := float64(width) / float64(n)
	barWidth := slot * 0.68
	max := summary.MaxVisitors
	if max == 0 {
		max = 1
	}
	plotHeight := float64(height - top - bottom)

	nodes := make([]gosx.Node, 0, n*2+4)
	for i, point := range summary.Days {
		x := float64(i)*slot + (slot-barWidth)/2
		barHeight := plotHeight * float64(point.Visitors) / float64(max)
		if point.Visitors > 0 && barHeight < 2 {
			barHeight = 2
		}
		y := float64(top) + plotHeight - barHeight
		nodes = append(nodes, gosx.El("rect", gosx.Attrs(
			gosx.Attr("class", "stats-bar"),
			gosx.Attr("x", fmtFloat(x)), gosx.Attr("y", fmtFloat(y)),
			gosx.Attr("width", fmtFloat(barWidth)), gosx.Attr("height", fmtFloat(barHeight)),
			gosx.Attr("rx", "2"),
		),
			gosx.El("title", nil, gosx.Text(point.Label+": "+plural(point.Visitors, "visitor")+", "+plural(point.Views, "page view"))),
		))
		// Every seventh day gets a label, plus the ends; a weekly label that
		// would crowd the last one is dropped.
		if i == 0 || i == n-1 || (i%7 == 0 && i < n-3) {
			anchor := "middle"
			if i == 0 {
				anchor = "start"
			} else if i == n-1 {
				anchor = "end"
			}
			nodes = append(nodes, gosx.El("text", gosx.Attrs(
				gosx.Attr("class", "stats-label"),
				gosx.Attr("x", fmtFloat(x+barWidth/2)), gosx.Attr("y", strconv.Itoa(height-8)),
				gosx.Attr("text-anchor", anchor),
			), gosx.Text(point.Label)))
		}
	}
	nodes = append(nodes,
		gosx.El("line", gosx.Attrs(gosx.Attr("class", "stats-axis"), gosx.Attr("x1", "0"), gosx.Attr("x2", strconv.Itoa(width)),
			gosx.Attr("y1", fmtFloat(float64(top)+plotHeight)), gosx.Attr("y2", fmtFloat(float64(top)+plotHeight)))),
		gosx.El("text", gosx.Attrs(gosx.Attr("class", "stats-label"), gosx.Attr("x", strconv.Itoa(width)), gosx.Attr("y", "12"), gosx.Attr("text-anchor", "end")),
			gosx.Text("peak "+plural(summary.MaxVisitors, "visitor")+" in a day")),
	)
	return gosx.El("svg", gosx.Attrs(
		gosx.Attr("class", "stats-chart"),
		gosx.Attr("viewBox", "0 0 "+strconv.Itoa(width)+" "+strconv.Itoa(height)),
		gosx.Attr("role", "img"),
		gosx.Attr("aria-label", "Visitors per day for the last 30 days, peaking at "+plural(summary.MaxVisitors, "visitor")),
	), gosx.Fragment(nodes...))
}

func renderStatsTable(heading, labelHead, countHead string, rows []statsRow) gosx.Node {
	if len(rows) == 0 {
		return gosx.Fragment()
	}
	body := make([]gosx.Node, 0, len(rows))
	for _, row := range rows {
		body = append(body, gosx.El("tr", nil,
			gosx.El("td", nil,
				gosx.El("span", gosx.Attrs(gosx.Attr("class", "stats-row__label")), gosx.Text(row.Label)),
				gosx.El("svg", gosx.Attrs(gosx.Attr("class", "stats-row__bar"), gosx.Attr("viewBox", "0 0 100 4"), gosx.Attr("preserveAspectRatio", "none"), gosx.Attr("aria-hidden", "true")),
					gosx.El("rect", gosx.Attrs(gosx.Attr("x", "0"), gosx.Attr("y", "0"), gosx.Attr("width", strconv.Itoa(row.Share)), gosx.Attr("height", "4")))),
			),
			gosx.El("td", gosx.Attrs(gosx.Attr("class", "stats-count")), gosx.Text(strconv.Itoa(row.Count))),
		))
	}
	return gosx.El("section", gosx.Attrs(gosx.Attr("class", "admin-panel")),
		gosx.El("h2", nil, gosx.Text(heading)),
		gosx.El("table", gosx.Attrs(gosx.Attr("class", "admin-table stats-table")),
			gosx.El("thead", nil, gosx.El("tr", nil, gosx.El("th", nil, gosx.Text(labelHead)), gosx.El("th", gosx.Attrs(gosx.Attr("class", "stats-count")), gosx.Text(countHead)))),
			gosx.El("tbody", nil, gosx.Fragment(body...)),
		),
	)
}

func fmtFloat(v float64) string { return strconv.FormatFloat(v, 'f', 1, 64) }

func plural(n int, noun string) string {
	if n == 1 {
		return "1 " + noun
	}
	return strconv.Itoa(n) + " " + noun + "s"
}

// renderStatsField is the Settings switch.
func renderStatsField(settings cmsstore.SiteSettings) gosx.Node {
	attrs := []any{gosx.Attr("type", "checkbox"), gosx.Attr("name", "countVisitors"), gosx.Attr("value", "on")}
	if settings.Metadata[statsOffKey] != "true" {
		attrs = append(attrs, gosx.Attr("checked", "checked"))
	}
	return gosx.Fragment(
		gosx.El("h2", gosx.Attrs(gosx.Attr("class", "admin-subhead")), gosx.Text("Visitors")),
		gosx.El("label", gosx.Attrs(gosx.Attr("class", "admin-check")),
			gosx.El("input", gosx.Attrs(attrs...)),
			gosx.Text(" Count visitors (anonymous: no cookies, nothing personal stored, no consent notice needed)"),
		),
		gosx.El("p", gosx.Attrs(gosx.Attr("class", "admin-hint")),
			gosx.Text("See the numbers under Visitors.")),
	)
}

func applyStatsField(r *http.Request, metadata cmsstore.Metadata) {
	if r.PostFormValue("countVisitors") == "on" {
		delete(metadata, statsOffKey)
	} else {
		metadata[statsOffKey] = "true"
	}
}
