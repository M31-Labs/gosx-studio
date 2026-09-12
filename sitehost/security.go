package sitehost

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"net/http"
	"strings"
	"sync"
	"time"

	"m31labs.dev/gosx"
)

// security.go is the baseline every later phase builds on.
//
// Three things live here. A CSRF token on every authenticated POST: the admin
// area can run on basic auth, and a browser attaches cached basic-auth
// credentials to a cross-site form post, so without a token any page on the
// internet could archive the owner's pages. A Content Security Policy and the
// usual hardening headers on every response. And a lockout after repeated
// failed sign-ins, so the admin password cannot be guessed at line speed.

const (
	csrfFormField  = "_csrf"
	csrfHeader     = "X-CSRF-Token"
	csrfMetaName   = "csrf-token"
	authFailLimit  = 10
	authFailWindow = 10 * time.Minute
)

// csrfToken is the per-install synchronizer token. It is derived from the
// admin password so it is stable across restarts on a guarded server — an
// owner with an admin tab open should not be signed out of it by a deploy —
// and random per process on an unguarded localhost server, where there is no
// secret to derive it from.
func (h *Host) csrfToken() string {
	h.securityOnce.Do(func() {
		secret := []byte(strings.TrimSpace(h.opts.AdminPassword))
		if len(secret) == 0 {
			secret = make([]byte, 32)
			_, _ = rand.Read(secret)
		}
		mac := hmac.New(sha256.New, secret)
		mac.Write([]byte("gosx-site csrf v1"))
		h.csrf = hex.EncodeToString(mac.Sum(nil))
	})
	return h.csrf
}

// csrfField is the hidden input every admin form carries.
func (h *Host) csrfField() gosx.Node {
	return gosx.El("input", gosx.Attrs(
		gosx.Attr("type", "hidden"),
		gosx.Attr("name", csrfFormField),
		gosx.Attr("value", h.csrfToken()),
	))
}

func tokensEqual(a, b string) bool {
	return a != "" && subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

// csrfExempt paths are unauthenticated by design: the wizard runs before
// there is an owner to protect, and the contact form is for the public.
func csrfExempt(path string) bool {
	return strings.HasPrefix(path, "/setup") || path == contactSendPath
}

// requireCSRF rejects any state-changing request that did not originate from
// a page this server rendered.
func (h *Host) requireCSRF(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet || r.Method == http.MethodHead || r.Method == http.MethodOptions || csrfExempt(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}
		// Modern browsers say where a request came from; a cross-site origin
		// is refused before the token is even looked at.
		if site := strings.ToLower(strings.TrimSpace(r.Header.Get("Sec-Fetch-Site"))); site != "" && site != "same-origin" && site != "none" {
			h.writeCSRFFailure(w, r)
			return
		}
		token := strings.TrimSpace(r.Header.Get(csrfHeader))
		contentType := r.Header.Get("Content-Type")
		if token == "" && strings.HasPrefix(contentType, "application/x-www-form-urlencoded") {
			if err := r.ParseForm(); err == nil {
				token = strings.TrimSpace(r.PostFormValue(csrfFormField))
			}
		}
		if token == "" && strings.HasPrefix(contentType, "multipart/form-data") {
			// A plain HTML upload form (the Settings page's logo field)
			// cannot send a header; its token is a field like any other.
			if err := r.ParseMultipartForm(32 << 20); err == nil {
				token = strings.TrimSpace(r.PostFormValue(csrfFormField))
			}
		}
		if !tokensEqual(token, h.csrfToken()) {
			h.writeCSRFFailure(w, r)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (h *Host) writeCSRFFailure(w http.ResponseWriter, r *http.Request) {
	if strings.HasPrefix(r.URL.Path, "/admin/api/") || strings.Contains(r.Header.Get("Accept"), "application/json") {
		writeJSON(w, http.StatusForbidden, editorSaveResult{Message: "This page has been open a while. Reload it and try again — your text is still here."})
		return
	}
	body := h.renderAdminShell("", "That didn't go through",
		"This form was out of date, which can happen after the site restarts. Go back, reload the page, and try again.",
		adminStatus{},
		gosx.El("p", gosx.Attrs(gosx.Attr("class", "admin-actions")),
			gosx.El("a", gosx.Attrs(gosx.Attr("class", "admin-button"), gosx.Attr("href", "/admin")), gosx.Text("Back to your site"))),
	)
	h.writeDocument(w, http.StatusForbidden, h.adminMeta("Try again"), body)
}

// ---------- headers ----------

// contentSecurityPolicy is one policy for the public site and the admin
// area. Scripts only from this origin; styles from this origin, inline (the
// theme tag and swatch colours), and Google Fonts; images from anywhere an
// owner might paste a link to; nothing may frame the site.
const contentSecurityPolicy = "default-src 'self'; " +
	"script-src 'self'; " +
	"style-src 'self' 'unsafe-inline' https://fonts.googleapis.com; " +
	"font-src 'self' https://fonts.gstatic.com data:; " +
	"img-src 'self' data: https: http:; " +
	"media-src 'self' https:; " +
	"connect-src 'self'; " +
	"frame-ancestors 'none'; " +
	"form-action 'self'; " +
	"base-uri 'self'; " +
	"object-src 'none'"

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		header := w.Header()
		header.Set("Content-Security-Policy", contentSecurityPolicy)
		header.Set("X-Content-Type-Options", "nosniff")
		header.Set("X-Frame-Options", "DENY")
		header.Set("Referrer-Policy", "strict-origin-when-cross-origin")
		header.Set("Permissions-Policy", "camera=(), microphone=(), geolocation=(), payment=()")
		next.ServeHTTP(w, r)
	})
}

// ---------- sign-in lockout ----------

// rateLimiter is a small sliding-window counter keyed by remote host.
type rateLimiter struct {
	mu     sync.Mutex
	limit  int
	window time.Duration
	hits   map[string][]time.Time
}

func newRateLimiter(limit int, window time.Duration) *rateLimiter {
	return &rateLimiter{limit: limit, window: window, hits: map[string][]time.Time{}}
}

// blocked reports whether the host has already hit the limit inside the window.
func (l *rateLimiter) blocked(host string, now time.Time) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	return len(l.recent(host, now)) >= l.limit
}

// record adds one hit for the host.
func (l *rateLimiter) record(host string, now time.Time) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.hits[host] = append(l.recent(host, now), now)
}

// clear forgets a host, for example after a successful sign-in.
func (l *rateLimiter) clear(host string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.hits, host)
}

func (l *rateLimiter) recent(host string, now time.Time) []time.Time {
	cutoff := now.Add(-l.window)
	kept := l.hits[host][:0]
	for _, hit := range l.hits[host] {
		if hit.After(cutoff) {
			kept = append(kept, hit)
		}
	}
	l.hits[host] = kept
	return kept
}
