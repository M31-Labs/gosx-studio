package sitehost

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"m31labs.dev/gosx"
)

// audit.go is the activity log: who did what, when, from where.
//
// It answers the question a team asks the morning after — "who changed
// this?" — from a plain JSON-lines file beside the site data. Content
// changes come through the editor's own endpoints and are recorded there;
// sign-ins, invites, role changes, and settings saves are recorded where
// they happen. The admin screen shows the newest 200 and the whole file can
// be downloaded.

const auditKeep = 5000

type auditEntry struct {
	At      time.Time `json:"at"`
	User    string    `json:"user"`
	Email   string    `json:"email"`
	Event   string    `json:"event"`
	Summary string    `json:"summary"`
	From    string    `json:"from,omitempty"`
}

type auditStore struct {
	mu      sync.Mutex
	path    string
	loaded  bool
	entries []auditEntry
}

func newAuditStore(path string) *auditStore { return &auditStore{path: path} }

func (o Options) auditPath() string {
	if o.DataPath == "" {
		return ""
	}
	return filepath.Join(filepath.Dir(o.DataPath), "activity.json")
}

func (s *auditStore) loadLocked() {
	if s.loaded {
		return
	}
	s.loaded = true
	if s.path == "" {
		return
	}
	raw, err := os.ReadFile(s.path)
	if err != nil {
		return
	}
	_ = json.Unmarshal(raw, &s.entries)
}

func (s *auditStore) record(user User, event, summary, from string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.loadLocked()
	s.entries = append(s.entries, auditEntry{At: timeNow().UTC(), User: firstNonEmpty(user.Name, "someone"), Email: user.Email, Event: event, Summary: summary, From: from})
	if len(s.entries) > auditKeep {
		s.entries = s.entries[len(s.entries)-auditKeep:]
	}
	if s.path == "" {
		return
	}
	raw, err := json.Marshal(s.entries)
	if err != nil {
		return
	}
	_ = os.MkdirAll(filepath.Dir(s.path), 0o755)
	temp := s.path + ".tmp"
	if os.WriteFile(temp, raw, 0o600) == nil {
		_ = os.Rename(temp, s.path)
	}
}

func (s *auditStore) newest(n int) []auditEntry {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.loadLocked()
	out := make([]auditEntry, 0, n)
	for index := len(s.entries) - 1; index >= 0 && len(out) < n; index-- {
		out = append(out, s.entries[index])
	}
	return out
}

func (h *Host) mountAudit(mux *http.ServeMux) {
	mux.HandleFunc("GET /admin/activity", h.handleAdminActivity)
	mux.HandleFunc("GET /admin/activity.json", h.handleAdminActivityExport)
}

func (h *Host) handleAdminActivity(w http.ResponseWriter, r *http.Request) {
	entries := h.auditLog.newest(200)
	var listing gosx.Node
	if len(entries) == 0 {
		listing = gosx.El("section", gosx.Attrs(gosx.Attr("class", "admin-empty")),
			gosx.El("h2", nil, gosx.Text("Nothing yet")),
			gosx.El("p", nil, gosx.Text("Sign-ins, publishes, settings changes, and invites are recorded here as they happen.")))
	} else {
		rows := make([]gosx.Node, 0, len(entries))
		for _, entry := range entries {
			rows = append(rows, gosx.El("tr", nil,
				gosx.El("td", nil, gosx.Text(formatWhen(entry.At))),
				gosx.El("td", nil, gosx.Text(entry.User)),
				gosx.El("td", nil, gosx.Text(entry.Summary)),
				gosx.El("td", nil, gosx.El("code", nil, gosx.Text(entry.Event))),
			))
		}
		listing = gosx.El("section", gosx.Attrs(gosx.Attr("class", "admin-panel")),
			gosx.El("table", gosx.Attrs(gosx.Attr("class", "admin-table")),
				gosx.El("thead", nil, gosx.El("tr", nil, gosx.El("th", nil, gosx.Text("When")), gosx.El("th", nil, gosx.Text("Who")), gosx.El("th", nil, gosx.Text("What")), gosx.El("th", nil, gosx.Text("")))),
				gosx.El("tbody", nil, gosx.Fragment(rows...))),
			gosx.El("p", gosx.Attrs(gosx.Attr("class", "admin-actions")),
				gosx.El("a", gosx.Attrs(gosx.Attr("class", "admin-secondary"), gosx.Attr("href", "/admin/activity.json")), gosx.Text("Download the whole log"))),
		)
	}
	body := h.renderAdminShell("activity", "Activity", "Who did what on this site, newest first.", adminStatus{}, listing)
	h.writeDocument(w, http.StatusOK, h.adminMeta("Activity"), body)
}

func (h *Host) handleAdminActivityExport(w http.ResponseWriter, r *http.Request) {
	entries := h.auditLog.newest(auditKeep)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="activity.json"`)
	w.Header().Set("Cache-Control", "no-store")
	_ = json.NewEncoder(w).Encode(entries)
}

// auditContent is the hook editor and admin handlers call for content
// events; it reads the signed-in person from the request.
func (h *Host) auditContent(r *http.Request, event, summary string) {
	user, _ := h.currentUser(r)
	if user.Name == "" && h.users.count() == 0 {
		user.Name = "the owner"
	}
	if strings.TrimSpace(summary) == "" {
		return
	}
	h.audit(r, user, event, summary)
}
