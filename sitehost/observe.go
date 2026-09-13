package sitehost

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// observe.go is what an operator needs when the site is theirs to run: a
// request id on every response, one JSON line per request in the log, and
// counters an admin (or a scraper with the admin's session) can read.

const requestIDHeader = "X-Request-ID"

type metrics struct {
	mu        sync.Mutex
	started   time.Time
	requests  map[string]int64 // by status class and area: "2xx public"
	durations map[string]int64 // summed milliseconds, same keys
	slow      int64            // requests over a second
}

func newMetrics() *metrics {
	return &metrics{started: timeNow(), requests: map[string]int64{}, durations: map[string]int64{}}
}

func (m *metrics) record(area string, status int, elapsed time.Duration) {
	key := strconv.Itoa(status/100) + "xx " + area
	m.mu.Lock()
	defer m.mu.Unlock()
	m.requests[key]++
	m.durations[key] += elapsed.Milliseconds()
	if elapsed > time.Second {
		m.slow++
	}
}

type statusWriter struct {
	http.ResponseWriter
	status int
	bytes  int64
}

func (w *statusWriter) WriteHeader(code int) {
	if w.status == 0 {
		w.status = code
	}
	w.ResponseWriter.WriteHeader(code)
}

func (w *statusWriter) Write(b []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	n, err := w.ResponseWriter.Write(b)
	w.bytes += int64(n)
	return n, err
}

// Flush keeps streaming responses streaming.
func (w *statusWriter) Flush() {
	if flusher, ok := w.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

func requestArea(path string) string {
	switch {
	case strings.HasPrefix(path, "/admin/api/"):
		return "api"
	case isAdminPath(path):
		return "admin"
	case strings.HasPrefix(path, "/uploads/") || strings.HasPrefix(path, "/_gosx/"):
		return "assets"
	default:
		return "public"
	}
}

// observe wraps every request: id, timing, counters, and the log line.
func (h *Host) observe(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := strings.TrimSpace(r.Header.Get(requestIDHeader))
		if id == "" || len(id) > 64 {
			id = randomHex(8)
		}
		w.Header().Set(requestIDHeader, id)
		sw := &statusWriter{ResponseWriter: w}
		start := time.Now()
		next.ServeHTTP(sw, r)
		elapsed := time.Since(start)
		if sw.status == 0 {
			sw.status = http.StatusOK
		}
		area := requestArea(r.URL.Path)
		if !strings.HasSuffix(r.URL.Path, "/events") {
			h.metrics.record(area, sw.status, elapsed)
		}
		if h.opts.LogRequests && h.logWriter() != nil && area != "assets" {
			line := map[string]any{
				"ts": timeNow().UTC().Format(time.RFC3339Nano), "id": id, "method": r.Method, "path": r.URL.Path,
				"status": sw.status, "ms": elapsed.Milliseconds(), "bytes": sw.bytes, "ip": clientIP(r), "area": area,
			}
			if ua := r.UserAgent(); ua != "" {
				if len(ua) > 120 {
					ua = ua[:120]
				}
				line["ua"] = ua
			}
			if user, ok := h.sessionUser(r); ok {
				line["user"] = user.Email
			}
			raw, _ := json.Marshal(line)
			h.logMu.Lock()
			_, _ = h.logWriter().Write(append(raw, '\n'))
			h.logMu.Unlock()
		}
	})
}

func (h *Host) logWriter() io.Writer {
	if h.opts.LogWriter != nil {
		return h.opts.LogWriter
	}
	return os.Stderr
}

// handleMetrics is Prometheus text: request counts and latency by status
// class and area, plus how much the site holds.
func (h *Host) handleMetrics(w http.ResponseWriter, r *http.Request) {
	h.metrics.mu.Lock()
	keys := make([]string, 0, len(h.metrics.requests))
	for key := range h.metrics.requests {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	var b strings.Builder
	b.WriteString("# HELP gosx_site_requests_total Requests served, by status class and area.\n# TYPE gosx_site_requests_total counter\n")
	for _, key := range keys {
		class, area, _ := strings.Cut(key, " ")
		b.WriteString("gosx_site_requests_total{status=\"" + class + "\",area=\"" + area + "\"} " + strconv.FormatInt(h.metrics.requests[key], 10) + "\n")
	}
	b.WriteString("# HELP gosx_site_request_milliseconds_total Time spent serving, by status class and area.\n# TYPE gosx_site_request_milliseconds_total counter\n")
	for _, key := range keys {
		class, area, _ := strings.Cut(key, " ")
		b.WriteString("gosx_site_request_milliseconds_total{status=\"" + class + "\",area=\"" + area + "\"} " + strconv.FormatInt(h.metrics.durations[key], 10) + "\n")
	}
	b.WriteString("# HELP gosx_site_slow_requests_total Requests that took over a second.\n# TYPE gosx_site_slow_requests_total counter\ngosx_site_slow_requests_total " + strconv.FormatInt(h.metrics.slow, 10) + "\n")
	b.WriteString("# HELP gosx_site_uptime_seconds Seconds since the process started.\n# TYPE gosx_site_uptime_seconds gauge\ngosx_site_uptime_seconds " + strconv.FormatInt(int64(timeNow().Sub(h.metrics.started).Seconds()), 10) + "\n")
	h.metrics.mu.Unlock()

	counts := map[string]int{}
	counts["pages"] = len(h.livePages())
	counts["posts"] = len(h.livePosts())
	counts["products"] = len(h.activeProducts())
	counts["orders_to_send"] = h.openOrders()
	counts["messages_unread"] = h.unreadMessages()
	counts["review_waiting"] = len(h.reviewQueue())
	counts["users"] = h.users.count()
	names := make([]string, 0, len(counts))
	for name := range counts {
		names = append(names, name)
	}
	sort.Strings(names)
	b.WriteString("# HELP gosx_site_content Live content and work waiting, by kind.\n# TYPE gosx_site_content gauge\n")
	for _, name := range names {
		b.WriteString("gosx_site_content{kind=\"" + name + "\"} " + strconv.Itoa(counts[name]) + "\n")
	}
	w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	_, _ = w.Write([]byte(b.String()))
}
