package sitehost

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"m31labs.dev/gosx-studio/cms/content"
	cmsstore "m31labs.dev/gosx-studio/cms/store"
)

// agent.go makes the site agent-native: every editor operation is also a
// bearer-authenticated JSON call under /agent/v1, described by a schema and
// an OpenAPI document, so any agent can read, build, and publish a site
// the same way the editor does. The MCP endpoint (mcp.go) and the owner's
// own browser assistant (WebMCP, webmcp.js) are two clients of this API.

const (
	agentPathPrefix = "/agent"
	agentAPIPrefix  = "/agent/v1"
	agentKeyPrefix  = "gsk_"
	agentAPIVersion = "1"
)

// Scopes an agent key can carry.
const (
	scopeRead     = "read"
	scopeWrite    = "write"
	scopePublish  = "publish"
	scopeSettings = "settings"
)

var agentScopes = []string{scopeRead, scopeWrite, scopePublish, scopeSettings}

var agentScopeBlurbs = map[string]string{
	scopeRead:     "Read pages, posts, products, pictures, messages, and visitor counts.",
	scopeWrite:    "Create and change pages, posts, products, pictures, and presets as drafts.",
	scopePublish:  "Publish and take pages and posts offline.",
	scopeSettings: "Change the site's name, description, header, footer, and Look.",
}

// AgentKey is one credential an owner gave to an agent.
type AgentKey struct {
	ID       string    `json:"id"`
	Name     string    `json:"name"`
	Hash     string    `json:"hash"`
	Hint     string    `json:"hint"` // the first characters, to tell keys apart
	Scopes   []string  `json:"scopes"`
	Created  time.Time `json:"created"`
	LastUsed time.Time `json:"lastUsed,omitempty"`
	Revoked  bool      `json:"revoked,omitempty"`
}

func (k AgentKey) has(scope string) bool {
	for _, s := range k.Scopes {
		if s == scope {
			return true
		}
	}
	return false
}

type agentStore struct {
	mu     sync.Mutex
	path   string
	loaded bool
	keys   []AgentKey
}

func newAgentStore(path string) *agentStore { return &agentStore{path: path} }

func (o Options) agentsPath() string {
	if o.DataPath == "" {
		return ""
	}
	return filepath.Join(filepath.Dir(o.DataPath), "agents.json")
}

func (s *agentStore) loadLocked() {
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
	var file struct {
		Keys []AgentKey `json:"keys"`
	}
	if json.Unmarshal(raw, &file) == nil {
		s.keys = file.Keys
	}
}

func (s *agentStore) saveLocked() error {
	if s.path == "" {
		return nil
	}
	raw, err := json.MarshalIndent(struct {
		Keys []AgentKey `json:"keys"`
	}{s.keys}, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	temp := s.path + ".tmp"
	if err := os.WriteFile(temp, raw, 0o600); err != nil {
		return err
	}
	return os.Rename(temp, s.path)
}

func hashAgentToken(token string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(token)))
	return hex.EncodeToString(sum[:])
}

// create mints a key and returns the one-time token with it.
func (s *agentStore) create(name string, scopes []string) (AgentKey, string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.loadLocked()
	token := agentKeyPrefix + randomHex(20)
	key := AgentKey{ID: "k" + randomHex(4), Name: name, Hash: hashAgentToken(token), Hint: token[:len(agentKeyPrefix)+6], Scopes: normalizeScopes(scopes), Created: timeNow().UTC()}
	s.keys = append(s.keys, key)
	return key, token, s.saveLocked()
}

func normalizeScopes(scopes []string) []string {
	out := []string{}
	for _, want := range agentScopes {
		for _, scope := range scopes {
			if strings.EqualFold(strings.TrimSpace(scope), want) {
				out = append(out, want)
				break
			}
		}
	}
	if len(out) == 0 {
		out = []string{scopeRead}
	}
	return out
}

func (s *agentStore) list() []AgentKey {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.loadLocked()
	out := make([]AgentKey, len(s.keys))
	copy(out, s.keys)
	return out
}

func (s *agentStore) byToken(token string) (AgentKey, bool) {
	hash := hashAgentToken(token)
	s.mu.Lock()
	defer s.mu.Unlock()
	s.loadLocked()
	for i, key := range s.keys {
		if key.Hash == hash && !key.Revoked {
			s.keys[i].LastUsed = timeNow().UTC()
			// A used key is worth remembering; a failed write is not worth
			// failing the request over.
			_ = s.saveLocked()
			return s.keys[i], true
		}
	}
	return AgentKey{}, false
}

func (s *agentStore) revoke(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.loadLocked()
	for i := range s.keys {
		if s.keys[i].ID == id {
			s.keys[i].Revoked = true
		}
	}
	return s.saveLocked()
}

// ---------- identity ----------

// agentIdentity is who is calling the agent API: a key, a pre-provisioned
// platform key, or the signed-in person's own browser (WebMCP).
type agentIdentity struct {
	Name   string
	KeyID  string
	Scopes map[string]bool
	User   User // set when the call is made on behalf of a signed-in person
}

func (id agentIdentity) can(scope string) bool { return id.Scopes[scope] }

type agentIdentityKey struct{}

func (h *Host) agentCaller(r *http.Request) (agentIdentity, bool) {
	id, ok := r.Context().Value(agentIdentityKey{}).(agentIdentity)
	return id, ok
}

func identityFromKey(key AgentKey) agentIdentity {
	scopes := map[string]bool{}
	for _, scope := range key.Scopes {
		scopes[scope] = true
	}
	return agentIdentity{Name: key.Name, KeyID: key.ID, Scopes: scopes}
}

func identityFromUser(user User) agentIdentity {
	scopes := map[string]bool{scopeRead: true, scopeWrite: true}
	if roleRank(user.Role) >= roleRank(roleAdmin) {
		scopes[scopePublish] = true
		scopeSettingsForAdmin(scopes)
	}
	return agentIdentity{Name: firstNonEmpty(user.Name, "the owner"), Scopes: scopes, User: user}
}

func scopeSettingsForAdmin(scopes map[string]bool) { scopes[scopeSettings] = true }

// browserIdentity is the signed-in person calling from a page this site
// served: a valid CSRF token, plus a session (or a laptop site with no
// accounts, where the owner is whoever sits at it).
func (h *Host) browserIdentity(r *http.Request) (agentIdentity, bool) {
	token := strings.TrimSpace(r.Header.Get(csrfHeader))
	if token == "" || !tokensEqual(token, h.csrfToken()) {
		return agentIdentity{}, false
	}
	if site := strings.ToLower(strings.TrimSpace(r.Header.Get("Sec-Fetch-Site"))); site != "" && site != "same-origin" && site != "none" {
		return agentIdentity{}, false
	}
	if user, ok := h.sessionUser(r); ok {
		return identityFromUser(user), true
	}
	if h.users.count() == 0 && strings.TrimSpace(h.opts.AdminPassword) == "" {
		return agentIdentity{Name: "the owner", Scopes: map[string]bool{scopeRead: true, scopeWrite: true, scopePublish: true, scopeSettings: true}}, true
	}
	return agentIdentity{}, false
}

// lookupAgentToken resolves a bearer token to an identity: a stored key, or
// one of the platform's pre-provisioned keys.
func (h *Host) lookupAgentToken(token string) (agentIdentity, bool) {
	token = strings.TrimSpace(token)
	if token == "" {
		return agentIdentity{}, false
	}
	for _, provisioned := range h.opts.AgentKeys {
		if provisioned != "" && secureEqual(provisioned, token) {
			return agentIdentity{Name: firstNonEmpty(h.opts.ManagedBy, "the platform"), KeyID: "platform", Scopes: map[string]bool{scopeRead: true, scopeWrite: true, scopePublish: true, scopeSettings: true}}, true
		}
	}
	key, ok := h.agents.byToken(token)
	if !ok {
		return agentIdentity{}, false
	}
	return identityFromKey(key), true
}

// agentAuth turns a bearer token into an identity or refuses the call.
func (h *Host) agentAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, ok := h.agentCaller(r); ok {
			next.ServeHTTP(w, r)
			return
		}
		ip := remoteHost(r)
		if h.authFailures.blocked(ip, timeNow()) {
			agentError(w, http.StatusTooManyRequests, "rate_limited", "Too many failed attempts. Wait a few minutes and try again.")
			return
		}
		raw := strings.TrimSpace(r.Header.Get("Authorization"))
		if raw == "" {
			// The person's own browser (WebMCP, or the editor itself): the
			// session cookie says who, and the CSRF token proves the call
			// came from a page this site served, not from another site.
			if identity, ok := h.browserIdentity(r); ok {
				next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), agentIdentityKey{}, identity)))
				return
			}
		}
		token := ""
		if len(raw) > 7 && strings.EqualFold(raw[:7], "Bearer ") {
			token = strings.TrimSpace(raw[7:])
		}
		identity, ok := h.lookupAgentToken(token)
		if !ok {
			h.authFailures.record(ip, timeNow())
			w.Header().Set("WWW-Authenticate", `Bearer realm="gosx-site agent api"`)
			agentError(w, http.StatusUnauthorized, "unauthorized", "Send an agent key as a bearer token. An admin creates keys under Agents in the site's admin.")
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), agentIdentityKey{}, identity)))
	})
}

// requireScope answers false, having written the refusal, when the caller
// lacks the scope.
func (h *Host) requireScope(w http.ResponseWriter, r *http.Request, scope string) bool {
	identity, _ := h.agentCaller(r)
	if identity.can(scope) {
		return true
	}
	agentError(w, http.StatusForbidden, "forbidden", "This key cannot "+scopeVerb(scope)+". Ask an admin for a key with the “"+scope+"” scope.")
	return false
}

func scopeVerb(scope string) string {
	switch scope {
	case scopeWrite:
		return "change content"
	case scopePublish:
		return "publish"
	case scopeSettings:
		return "change settings"
	}
	return "read"
}

// requireReady refuses content calls until the wizard (or /agent/v1/setup)
// has built the site.
func (h *Host) requireReady(w http.ResponseWriter) bool {
	if h.SetupComplete() {
		return true
	}
	agentError(w, http.StatusConflict, "setup_required", "This site is not set up yet. Call POST /agent/v1/setup with a title and a kind first.")
	return false
}

// agentUser is who the audit log names for a call.
func (h *Host) agentUser(r *http.Request) User {
	identity, _ := h.agentCaller(r)
	if identity.User.Name != "" || identity.User.Email != "" {
		return identity.User
	}
	return User{Name: "Agent “" + firstNonEmpty(identity.Name, "unnamed") + "”"}
}

func (h *Host) agentAudit(r *http.Request, event, summary string) {
	if strings.TrimSpace(summary) == "" {
		return
	}
	h.audit(r, h.agentUser(r), event, summary)
}

// ---------- JSON helpers ----------

type agentErrorBody struct {
	Error agentErrorDetail `json:"error"`
}

type agentErrorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func agentError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, status, agentErrorBody{Error: agentErrorDetail{Code: code, Message: message}})
}

func agentJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, status, value)
}

func decodeAgentBody(w http.ResponseWriter, r *http.Request, into any, limit int64) bool {
	if r.Body == nil {
		agentError(w, http.StatusBadRequest, "bad_request", "Send a JSON body.")
		return false
	}
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, limit))
	if err := decoder.Decode(into); err != nil {
		if errors.Is(err, io.EOF) {
			agentError(w, http.StatusBadRequest, "bad_request", "Send a JSON body.")
			return false
		}
		agentError(w, http.StatusBadRequest, "bad_request", "The JSON body could not be read: "+err.Error())
		return false
	}
	return true
}

// ---------- routes ----------

func (h *Host) mountAgent(mux *http.ServeMux) {
	open := http.NewServeMux()
	open.HandleFunc("GET "+agentAPIPrefix, h.handleAgentIndex)
	open.HandleFunc("GET "+agentAPIPrefix+"/{$}", h.handleAgentIndex)
	open.HandleFunc("GET "+agentAPIPrefix+"/openapi.json", h.handleAgentOpenAPI)
	open.HandleFunc("GET "+agentAPIPrefix+"/schema", h.handleAgentSchema)

	api := http.NewServeMux()
	api.HandleFunc("POST "+agentAPIPrefix+"/setup", h.handleAgentSetup)
	api.HandleFunc("GET "+agentAPIPrefix+"/site", h.handleAgentSite)
	api.HandleFunc("PATCH "+agentAPIPrefix+"/site", h.handleAgentSitePatch)
	api.HandleFunc("GET "+agentAPIPrefix+"/pages", h.handleAgentPages)
	api.HandleFunc("POST "+agentAPIPrefix+"/pages", h.handleAgentPageCreate)
	api.HandleFunc("GET "+agentAPIPrefix+"/pages/{id}", h.handleAgentPage)
	api.HandleFunc("PUT "+agentAPIPrefix+"/pages/{id}", h.handleAgentPagePut)
	api.HandleFunc("PATCH "+agentAPIPrefix+"/pages/{id}", h.handleAgentPagePatch)
	api.HandleFunc("POST "+agentAPIPrefix+"/pages/{id}/publish", h.handleAgentPagePublish)
	api.HandleFunc("POST "+agentAPIPrefix+"/pages/{id}/actions", h.handleAgentPageAction)
	api.HandleFunc("GET "+agentAPIPrefix+"/pages/{id}/markdown", h.handleAgentPageMarkdown)
	api.HandleFunc("GET "+agentAPIPrefix+"/posts", h.handleAgentPosts)
	api.HandleFunc("POST "+agentAPIPrefix+"/posts", h.handleAgentPostCreate)
	api.HandleFunc("GET "+agentAPIPrefix+"/posts/{id}", h.handleAgentPost)
	api.HandleFunc("PUT "+agentAPIPrefix+"/posts/{id}", h.handleAgentPostPut)
	api.HandleFunc("POST "+agentAPIPrefix+"/posts/{id}/publish", h.handleAgentPostPublish)
	api.HandleFunc("GET "+agentAPIPrefix+"/products", h.handleAgentProducts)
	api.HandleFunc("POST "+agentAPIPrefix+"/products", h.handleAgentProductSave)
	api.HandleFunc("GET "+agentAPIPrefix+"/products/{id}", h.handleAgentProduct)
	api.HandleFunc("PUT "+agentAPIPrefix+"/products/{id}", h.handleAgentProductSave)
	api.HandleFunc("DELETE "+agentAPIPrefix+"/products/{id}", h.handleAgentProductDelete)
	api.HandleFunc("GET "+agentAPIPrefix+"/media", h.handleAgentMedia)
	api.HandleFunc("POST "+agentAPIPrefix+"/media", h.handleAgentMediaUpload)
	api.HandleFunc("GET "+agentAPIPrefix+"/look", h.handleAgentLook)
	api.HandleFunc("PUT "+agentAPIPrefix+"/look", h.handleAgentLookPut)
	api.HandleFunc("GET "+agentAPIPrefix+"/presets", h.handleAgentPresets)
	api.HandleFunc("POST "+agentAPIPrefix+"/presets", h.handleAgentPresetSave)
	api.HandleFunc("DELETE "+agentAPIPrefix+"/presets/{id}", h.handleAgentPresetDelete)
	api.HandleFunc("GET "+agentAPIPrefix+"/forms", h.handleAgentForms)
	api.HandleFunc("GET "+agentAPIPrefix+"/messages", h.handleAgentMessages)
	api.HandleFunc("GET "+agentAPIPrefix+"/stats", h.handleAgentStats)
	api.HandleFunc("GET "+agentAPIPrefix+"/activity", h.handleAgentActivity)
	api.HandleFunc("POST "+agentPathPrefix+"/mcp", h.handleMCP)
	api.HandleFunc("GET "+agentPathPrefix+"/mcp", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Allow", "POST, DELETE")
		agentError(w, http.StatusMethodNotAllowed, "method_not_allowed", "This MCP endpoint speaks JSON over POST; it does not open a stream.")
	})
	api.HandleFunc("DELETE "+agentPathPrefix+"/mcp", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusNoContent) })
	api.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		agentError(w, http.StatusNotFound, "not_found", "No such call. See "+agentAPIPrefix+"/openapi.json for what the agent API offers.")
	})

	guarded := h.agentAuth(api)
	mux.Handle(agentPathPrefix+"/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if r.Method == http.MethodGet && (path == agentAPIPrefix || path == agentAPIPrefix+"/" || path == agentAPIPrefix+"/openapi.json" || path == agentAPIPrefix+"/schema") {
			open.ServeHTTP(w, r)
			return
		}
		guarded.ServeHTTP(w, r)
	}))
	mux.HandleFunc("GET "+agentPathPrefix, func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, agentAPIPrefix, http.StatusSeeOther)
	})
}

// agentDispatch runs one agent API call in-process as the given identity.
// The MCP tools go through it, so there is exactly one implementation of
// every operation.
func (h *Host) agentDispatch(ctx context.Context, identity agentIdentity, method, path string, body []byte, from *http.Request) (int, http.Header, []byte) {
	var reader io.Reader
	if len(body) > 0 {
		reader = strings.NewReader(string(body))
	}
	req, err := http.NewRequestWithContext(context.WithValue(ctx, agentIdentityKey{}, identity), method, path, reader)
	if err != nil {
		return http.StatusBadRequest, http.Header{}, []byte(`{"error":{"code":"bad_request","message":"` + err.Error() + `"}}`)
	}
	if len(body) > 0 {
		req.Header.Set("Content-Type", "application/json")
	}
	if from != nil {
		req.Host = from.Host
		req.Header.Set("X-Forwarded-Proto", from.Header.Get("X-Forwarded-Proto"))
		req.RemoteAddr = from.RemoteAddr
		if client := from.Header.Get(editorClientHeader); client != "" {
			req.Header.Set(editorClientHeader, client)
		}
	}
	rec := &memoryResponse{header: http.Header{}, status: http.StatusOK}
	h.agentRoutes().ServeHTTP(rec, req)
	return rec.status, rec.header, []byte(rec.body.String())
}

// agentRoutes is the agent mux alone, for in-process dispatch.
func (h *Host) agentRoutes() http.Handler {
	h.agentOnce.Do(func() {
		mux := http.NewServeMux()
		h.mountAgent(mux)
		h.mountAgentPublic(mux)
		h.agentMux = mux
	})
	return h.agentMux
}

type memoryResponse struct {
	header http.Header
	status int
	body   strings.Builder
	wrote  bool
}

func (m *memoryResponse) Header() http.Header { return m.header }
func (m *memoryResponse) WriteHeader(status int) {
	if !m.wrote {
		m.status = status
		m.wrote = true
	}
}
func (m *memoryResponse) Write(b []byte) (int, error) {
	m.wrote = true
	return m.body.WriteString(string(b))
}

// ---------- index, schema, setup ----------

type agentIndex struct {
	Name        string            `json:"name"`
	Version     string            `json:"version"`
	Site        string            `json:"site"`
	BaseURL     string            `json:"baseUrl"`
	SetupDone   bool              `json:"setupComplete"`
	Features    []string          `json:"features"`
	Auth        string            `json:"auth"`
	Links       map[string]string `json:"links"`
	Description string            `json:"description"`
}

func (h *Host) handleAgentIndex(w http.ResponseWriter, r *http.Request) {
	settings := h.settings()
	base := h.absoluteBase(r)
	features := h.opts.Features
	if len(features) == 0 {
		features = AllFeatures
	}
	agentJSON(w, http.StatusOK, agentIndex{
		Name: "gosx-site agent api", Version: agentAPIVersion, Site: firstNonEmpty(settings.Title, h.opts.SiteTitle), BaseURL: base,
		SetupDone: h.SetupComplete(), Features: features,
		Auth: "Authorization: Bearer <agent key>; an admin creates keys under Agents in the site's admin.",
		Links: map[string]string{
			"openapi": base + agentAPIPrefix + "/openapi.json", "schema": base + agentAPIPrefix + "/schema", "mcp": base + agentPathPrefix + "/mcp",
			"llms": base + "/llms.txt", "llmsFull": base + "/llms-full.txt", "site": base + agentAPIPrefix + "/site", "pages": base + agentAPIPrefix + "/pages",
		},
		Description: "Read and change this website the way its editor does: pages are lists of blocks, blocks are ready-made sections with named fields and repeated items, and every change is a draft until published.",
	})
}

type agentSetupRequest struct {
	Title    string `json:"title"`
	Tagline  string `json:"tagline"`
	Kind     string `json:"kind"`
	Template string `json:"template"`
	Email    string `json:"email"`
	Phone    string `json:"phone"`
	Location string `json:"location"`
}

func (h *Host) handleAgentSetup(w http.ResponseWriter, r *http.Request) {
	if !h.requireScope(w, r, scopeSettings) {
		return
	}
	if h.SetupComplete() {
		agentError(w, http.StatusConflict, "already_set_up", "This site is already set up. Change it through /agent/v1/site and /agent/v1/pages.")
		return
	}
	var payload agentSetupRequest
	if !decodeAgentBody(w, r, &payload, 64<<10) {
		return
	}
	if strings.TrimSpace(payload.Title) == "" {
		agentError(w, http.StatusUnprocessableEntity, "invalid", "Give the site a title.")
		return
	}
	kind := SiteKindByKey(payload.Kind).Key
	answers := SetupAnswers{SiteTitle: payload.Title, Tagline: payload.Tagline, Kind: kind, Template: payload.Template, Email: payload.Email, Phone: payload.Phone, Location: payload.Location}
	answers.Template = templateFor(answers).Key
	if err := h.CompleteSetup(answers); err != nil {
		agentError(w, http.StatusInternalServerError, "failed", "The site could not be built: "+err.Error())
		return
	}
	h.agentAudit(r, "setup.completed", "Set the site up as “"+strings.TrimSpace(payload.Title)+"”")
	h.handleAgentSite(w, r)
}

// ---------- site ----------

type agentSite struct {
	Title       string            `json:"title"`
	Tagline     string            `json:"tagline,omitempty"`
	Description string            `json:"description,omitempty"`
	Kind        string            `json:"kind,omitempty"`
	BaseURL     string            `json:"baseUrl"`
	Domain      string            `json:"domain,omitempty"`
	Locale      string            `json:"locale,omitempty"`
	Contact     agentContact      `json:"contact"`
	Header      agentHeader       `json:"header"`
	Footer      agentFooter       `json:"footer"`
	Social      map[string]string `json:"social,omitempty"`
	Features    []string          `json:"features"`
	Look        agentLook         `json:"look"`
	Pages       []agentPageRow    `json:"pages"`
	Menu        []agentMenuItem   `json:"menu"`
	Counts      map[string]int    `json:"counts"`
	ReviewFirst bool              `json:"reviewRequired"`
	ManagedBy   string            `json:"managedBy,omitempty"`
}

type agentContact struct {
	Email    string `json:"email,omitempty"`
	Phone    string `json:"phone,omitempty"`
	Location string `json:"location,omitempty"`
}

type agentHeader struct {
	AnnounceText string `json:"announceText,omitempty"`
	AnnounceLink string `json:"announceLink,omitempty"`
	AnnounceOn   bool   `json:"announceOn"`
	Sticky       bool   `json:"sticky"`
	MenuButton   string `json:"menuButton,omitempty"`
	MenuButtonTo string `json:"menuButtonTo,omitempty"`
}

type agentFooter struct {
	Menu  bool       `json:"menu"`
	Links [][]string `json:"links,omitempty"`
}

type agentMenuItem struct {
	Title    string          `json:"title"`
	URL      string          `json:"url"`
	Children []agentMenuItem `json:"children,omitempty"`
}

func (h *Host) handleAgentSite(w http.ResponseWriter, r *http.Request) {
	if !h.requireReady(w) {
		return
	}
	if !h.requireScope(w, r, scopeRead) {
		return
	}
	agentJSON(w, http.StatusOK, h.agentSiteView(r))
}

func (h *Host) agentSiteView(r *http.Request) agentSite {
	settings := h.settings()
	m := settings.Metadata
	chrome := chromeFromSettings(settings)
	features := h.opts.Features
	if len(features) == 0 {
		features = AllFeatures
	}
	social := map[string]string{}
	for key, value := range brandFromSettings(settings).Social {
		if strings.TrimSpace(value) != "" {
			social[key] = value
		}
	}
	links := [][]string{}
	for _, link := range chrome.FooterLinks {
		links = append(links, []string{link[0], link[1]})
	}
	pages, _ := h.store.ListPages(cmsstore.PageFilter{})
	rows := make([]agentPageRow, 0, len(pages))
	published := 0
	for _, page := range pages {
		row := h.agentPageRow(r, page)
		if row.Status == "published" {
			published++
		}
		rows = append(rows, row)
	}
	posts, _ := h.store.ListPosts(cmsstore.PostFilter{})
	return agentSite{
		Title: firstNonEmpty(settings.Title, h.opts.SiteTitle), Tagline: strings.TrimSpace(m["tagline"]), Description: settings.Description,
		Kind: firstNonEmpty(m["siteKind"], "other"), BaseURL: h.absoluteBase(r), Domain: h.Domain(), Locale: firstNonEmpty(settings.Locale, "en"),
		Contact: agentContact{Email: strings.TrimSpace(m["contactEmail"]), Phone: strings.TrimSpace(m["contactPhone"]), Location: strings.TrimSpace(m["contactLocation"])},
		Header:  agentHeader{AnnounceText: chrome.AnnounceText, AnnounceLink: chrome.AnnounceLink, AnnounceOn: chrome.AnnounceOn, Sticky: chrome.Sticky, MenuButton: chrome.MenuButton, MenuButtonTo: chrome.MenuButtonTo},
		Footer:  agentFooter{Menu: chrome.FooterMenu, Links: links}, Social: social, Features: features, Look: h.agentLookView(),
		Pages: rows, Menu: h.agentMenu(r), Counts: map[string]int{"pages": len(pages), "publishedPages": published, "posts": len(posts), "products": len(h.products.list()), "unreadMessages": h.unreadMessages()},
		ReviewFirst: h.reviewRequired(), ManagedBy: h.opts.ManagedBy,
	}
}

func (h *Host) agentMenu(r *http.Request) []agentMenuItem {
	base := h.absoluteBase(r)
	out := []agentMenuItem{}
	for _, entry := range h.navTree() {
		item := agentMenuItem{Title: entry.Page.Title, URL: base + publicPath(entry.Page.Slug)}
		for _, child := range entry.Children {
			item.Children = append(item.Children, agentMenuItem{Title: child.Title, URL: base + publicPath(child.Slug)})
		}
		out = append(out, item)
	}
	return out
}

type agentSitePatch struct {
	Title       *string           `json:"title,omitempty"`
	Tagline     *string           `json:"tagline,omitempty"`
	Description *string           `json:"description,omitempty"`
	Kind        *string           `json:"kind,omitempty"`
	Contact     *agentContact     `json:"contact,omitempty"`
	Header      *agentHeaderPatch `json:"header,omitempty"`
	Footer      *agentFooterPatch `json:"footer,omitempty"`
	Social      map[string]string `json:"social,omitempty"`
	CustomCSS   *string           `json:"customCss,omitempty"`
}

type agentHeaderPatch struct {
	AnnounceText *string `json:"announceText,omitempty"`
	AnnounceLink *string `json:"announceLink,omitempty"`
	AnnounceOn   *bool   `json:"announceOn,omitempty"`
	Sticky       *bool   `json:"sticky,omitempty"`
	MenuButton   *string `json:"menuButton,omitempty"`
	MenuButtonTo *string `json:"menuButtonTo,omitempty"`
}

type agentFooterPatch struct {
	Menu  *bool      `json:"menu,omitempty"`
	Links [][]string `json:"links,omitempty"`
}

func (h *Host) handleAgentSitePatch(w http.ResponseWriter, r *http.Request) {
	if !h.requireReady(w) {
		return
	}
	if !h.requireScope(w, r, scopeSettings) {
		return
	}
	var patch agentSitePatch
	if !decodeAgentBody(w, r, &patch, 256<<10) {
		return
	}
	current := h.settings()
	metadata := cloneMetadata(current.Metadata)
	set := func(key string, value *string) {
		if value == nil {
			return
		}
		if trimmed := strings.TrimSpace(*value); trimmed != "" {
			metadata[key] = trimmed
		} else {
			delete(metadata, key)
		}
	}
	setBool := func(key string, value *bool) {
		if value == nil {
			return
		}
		if *value {
			metadata[key] = "true"
		} else {
			delete(metadata, key)
		}
	}
	set("tagline", patch.Tagline)
	if patch.Kind != nil {
		metadata["siteKind"] = SiteKindByKey(*patch.Kind).Key
	}
	if patch.Contact != nil {
		for key, value := range map[string]string{"contactEmail": patch.Contact.Email, "contactPhone": patch.Contact.Phone, "contactLocation": patch.Contact.Location} {
			value := value
			set(key, &value)
		}
	}
	if patch.Header != nil {
		set(announceTextKey, patch.Header.AnnounceText)
		if patch.Header.AnnounceLink != nil {
			safe := safeLinkHref(*patch.Header.AnnounceLink)
			set(announceLinkKey, &safe)
		}
		setBool(announceOnKey, patch.Header.AnnounceOn)
		setBool(headerStickyKey, patch.Header.Sticky)
		set(menuButtonKey, patch.Header.MenuButton)
		if patch.Header.MenuButtonTo != nil {
			safe := safeLinkHref(*patch.Header.MenuButtonTo)
			set(menuButtonURLKey, &safe)
		}
	}
	if patch.Footer != nil {
		setBool(footerMenuKey, patch.Footer.Menu)
		if patch.Footer.Links != nil {
			lines := []string{}
			for _, link := range patch.Footer.Links {
				if len(link) == 2 && strings.TrimSpace(link[0]) != "" && safeLinkHref(link[1]) != "" {
					lines = append(lines, strings.TrimSpace(link[0])+" | "+safeLinkHref(link[1]))
				}
			}
			joined := strings.Join(lines, "\n")
			set(footerLinksKey, &joined)
		}
	}
	for network, value := range patch.Social {
		for _, known := range socialNetworkKeys() {
			if strings.EqualFold(known, network) {
				value := value
				set(socialKey(known), &value)
			}
		}
	}
	if patch.CustomCSS != nil {
		css := sanitizeCustomCSS(*patch.CustomCSS)
		set(customCSSKey, &css)
	}
	input := cmsstore.SiteSettingsInput{
		Title: current.Title, Description: current.Description, BaseURL: current.BaseURL, Locale: firstNonEmpty(current.Locale, "en"), Metadata: metadata,
	}
	if patch.Title != nil && strings.TrimSpace(*patch.Title) != "" {
		input.Title = strings.TrimSpace(*patch.Title)
	}
	if patch.Description != nil {
		input.Description = strings.TrimSpace(*patch.Description)
	}
	if _, _, err := h.store.PreviewSiteSettings(input); err != nil {
		agentError(w, http.StatusInternalServerError, "failed", "The settings could not be saved.")
		return
	}
	if _, _, err := h.store.PublishSiteSettings(); err != nil {
		agentError(w, http.StatusInternalServerError, "failed", "The settings could not be published.")
		return
	}
	h.agentAudit(r, "settings.saved", "Changed site settings")
	agentJSON(w, http.StatusOK, h.agentSiteView(r))
}

// ---------- pages ----------

type agentPageRow struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Slug      string    `json:"slug"`
	URL       string    `json:"url"`
	EditURL   string    `json:"editUrl"`
	Status    string    `json:"status"` // published or draft
	Live      bool      `json:"live"`   // reachable by visitors right now
	Hidden    bool      `json:"hiddenFromMenu"`
	Offline   bool      `json:"offline"`
	Archived  bool      `json:"archived"`
	NavParent string    `json:"navParent,omitempty"`
	Updated   time.Time `json:"updated"`
	Blocks    int       `json:"blockCount"`
}

type agentPage struct {
	agentPageRow
	Description string               `json:"description"`
	Blocks      []editorBlockPayload `json:"blocks"`
	Checks      []string             `json:"checks"`
}

func (h *Host) agentPageRow(r *http.Request, page cmsstore.Page) agentPageRow {
	status := "draft"
	if page.State.Publish == cmsstore.PublishStatePublished {
		status = "published"
	}
	base := h.absoluteBase(r)
	return agentPageRow{
		ID: page.ID, Title: page.Title, Slug: page.Slug, URL: base + publicPath(page.Slug), EditURL: base + "/admin/edit/" + page.ID,
		Status: status, Live: h.isLive(page), Hidden: page.Metadata[pageNavHiddenKey] == "true", Offline: page.Metadata[pageOfflineKey] == "true",
		Archived: page.Metadata[pageArchivedKey] == "true", NavParent: PageNavParent(page), Updated: page.Updated, Blocks: len(page.Body.Blocks),
	}
}

func (h *Host) agentPageView(r *http.Request, page cmsstore.Page) agentPage {
	description := pageMetaValue(page, "metaDescription", page.Description)
	return agentPage{agentPageRow: h.agentPageRow(r, page), Description: description, Blocks: h.documentPayload(page.Body), Checks: h.readinessChecks("page", description, page.Body)}
}

func (h *Host) handleAgentPages(w http.ResponseWriter, r *http.Request) {
	if !h.requireReady(w) || !h.requireScope(w, r, scopeRead) {
		return
	}
	pages, err := h.store.ListPages(cmsstore.PageFilter{})
	if err != nil {
		agentError(w, http.StatusInternalServerError, "failed", "Pages could not be listed.")
		return
	}
	rows := make([]agentPageRow, 0, len(pages))
	for _, page := range pages {
		rows = append(rows, h.agentPageRow(r, page))
	}
	agentJSON(w, http.StatusOK, map[string]any{"pages": rows})
}

func (h *Host) agentPageByRef(w http.ResponseWriter, ref string) (cmsstore.Page, bool) {
	ref = strings.TrimSpace(ref)
	if page, ok, _ := h.store.PageByID(ref); ok {
		return page, true
	}
	if page, ok, _ := h.store.PageBySlug(normalizeSlug(ref)); ok {
		return page, true
	}
	agentError(w, http.StatusNotFound, "not_found", "No page has the id or address “"+ref+"”. List pages with GET /agent/v1/pages.")
	return cmsstore.Page{}, false
}

func (h *Host) handleAgentPage(w http.ResponseWriter, r *http.Request) {
	if !h.requireReady(w) || !h.requireScope(w, r, scopeRead) {
		return
	}
	page, ok := h.agentPageByRef(w, r.PathValue("id"))
	if !ok {
		return
	}
	agentJSON(w, http.StatusOK, h.agentPageView(r, page))
}

type agentPageWrite struct {
	Title       *string              `json:"title,omitempty"`
	Slug        *string              `json:"slug,omitempty"`
	Description *string              `json:"description,omitempty"`
	NavParent   *string              `json:"navParent,omitempty"`
	Template    string               `json:"template,omitempty"`
	Blocks      []editorBlockPayload `json:"blocks"`
	Publish     bool                 `json:"publish,omitempty"`
	// PATCH-only block edits, applied in this order: replace, remove,
	// insert, append. Indexes refer to the page as it was before the call.
	Replace []agentBlockAt       `json:"replace,omitempty"`
	Remove  []int                `json:"remove,omitempty"`
	Insert  []agentBlockAt       `json:"insert,omitempty"`
	Append  []editorBlockPayload `json:"append,omitempty"`
	Move    *agentBlockMove      `json:"move,omitempty"`
}

type agentBlockAt struct {
	Index int                `json:"index"`
	Block editorBlockPayload `json:"block"`
}

type agentBlockMove struct {
	From int `json:"from"`
	To   int `json:"to"`
}

func (h *Host) handleAgentPageCreate(w http.ResponseWriter, r *http.Request) {
	if !h.requireReady(w) || !h.requireScope(w, r, scopeWrite) {
		return
	}
	var payload agentPageWrite
	if !decodeAgentBody(w, r, &payload, 4<<20) {
		return
	}
	title := ""
	if payload.Title != nil {
		title = strings.TrimSpace(*payload.Title)
	}
	if title == "" {
		agentError(w, http.StatusUnprocessableEntity, "invalid", "Give the page a title.")
		return
	}
	slug := normalizeSlug(title)
	if payload.Slug != nil && strings.TrimSpace(*payload.Slug) != "" {
		slug = normalizeSlug(*payload.Slug)
	}
	if message := reservedSlugMessage(slug); message != "" {
		agentError(w, http.StatusUnprocessableEntity, "invalid", message)
		return
	}
	if _, exists, _ := h.store.PageBySlug(slug); exists {
		agentError(w, http.StatusConflict, "slug_taken", "Another page already uses /"+slug+". Pick a different address.")
		return
	}
	body := PageTemplateByKey(payload.Template).build(title)
	if payload.Blocks != nil {
		body = h.payloadDocument(payload.Blocks)
	}
	page, err := h.store.CreatePage(cmsstore.PageInput{Slug: slug, Title: title, Body: body})
	if err != nil {
		agentError(w, http.StatusInternalServerError, "failed", "The page could not be created.")
		return
	}
	if payload.Description != nil || payload.NavParent != nil {
		save := editorSavePayload{Title: title, Slug: slug, Blocks: h.documentPayload(page.Body)}
		if payload.Description != nil {
			save.Description = *payload.Description
		}
		if payload.NavParent != nil {
			save.NavParent = *payload.NavParent
		}
		if result := h.applyPageSave(r, page, save); !result.OK {
			agentError(w, http.StatusUnprocessableEntity, "invalid", result.Message)
			return
		}
		page, _, _ = h.store.PageByID(page.ID)
	}
	h.agentAudit(r, "page.created", "Created the page “"+title+"”")
	if payload.Publish && h.publishForAgent(w, r, &page) != nil {
		return
	}
	agentJSON(w, http.StatusCreated, h.agentPageView(r, page))
}

// publishForAgent publishes when the caller may, writing the refusal when
// not. It returns a non-nil error only after writing a response.
func (h *Host) publishForAgent(w http.ResponseWriter, r *http.Request, page *cmsstore.Page) error {
	if !h.requireScope(w, r, scopePublish) {
		return errors.New("forbidden")
	}
	if h.reviewRequired() {
		if identity, _ := h.agentCaller(r); identity.User.ID == "" || roleRank(identity.User.Role) < roleRank(roleAdmin) {
			agentError(w, http.StatusConflict, "review_required", "This site needs an admin to approve changes before they go live. The draft is saved; an admin can publish it from Review.")
			return errors.New("review")
		}
	}
	*page = h.clearReviewOnPage(*page)
	result := h.publishPage(*page)
	if !result.OK {
		agentError(w, http.StatusUnprocessableEntity, "not_published", result.Message)
		return errors.New("not published")
	}
	h.agentAudit(r, "page.published", firstNonEmpty(result.Message, "Published")+": “"+page.Title+"”")
	h.notifyChanged(r, "page", page.ID)
	*page, _, _ = h.store.PageByID(page.ID)
	return nil
}

func (h *Host) handleAgentPagePut(w http.ResponseWriter, r *http.Request) {
	if !h.requireReady(w) || !h.requireScope(w, r, scopeWrite) {
		return
	}
	page, ok := h.agentPageByRef(w, r.PathValue("id"))
	if !ok {
		return
	}
	var payload agentPageWrite
	if !decodeAgentBody(w, r, &payload, 4<<20) {
		return
	}
	if payload.Blocks == nil {
		agentError(w, http.StatusUnprocessableEntity, "invalid", "PUT replaces the whole page: send \"blocks\". Use PATCH to change part of it.")
		return
	}
	h.finishAgentPageWrite(w, r, page, payload, payload.Blocks)
}

func (h *Host) handleAgentPagePatch(w http.ResponseWriter, r *http.Request) {
	if !h.requireReady(w) || !h.requireScope(w, r, scopeWrite) {
		return
	}
	page, ok := h.agentPageByRef(w, r.PathValue("id"))
	if !ok {
		return
	}
	var payload agentPageWrite
	if !decodeAgentBody(w, r, &payload, 4<<20) {
		return
	}
	blocks := h.documentPayload(page.Body)
	if payload.Blocks != nil {
		blocks = payload.Blocks
	}
	blocks, err := applyBlockEdits(blocks, payload)
	if err != nil {
		agentError(w, http.StatusUnprocessableEntity, "invalid", err.Error())
		return
	}
	h.finishAgentPageWrite(w, r, page, payload, blocks)
}

// applyBlockEdits applies replace, remove, insert, append, and move to a
// block list. Indexes refer to the list as it was before the call.
func applyBlockEdits(blocks []editorBlockPayload, payload agentPageWrite) ([]editorBlockPayload, error) {
	out := make([]editorBlockPayload, len(blocks))
	copy(out, blocks)
	for _, edit := range payload.Replace {
		if edit.Index < 0 || edit.Index >= len(out) {
			return nil, errors.New("replace: no block at index " + strconv.Itoa(edit.Index) + " (the page has " + strconv.Itoa(len(out)) + ")")
		}
		out[edit.Index] = edit.Block
	}
	removed := map[int]bool{}
	for _, index := range payload.Remove {
		if index < 0 || index >= len(out) {
			return nil, errors.New("remove: no block at index " + strconv.Itoa(index))
		}
		removed[index] = true
	}
	inserts := map[int][]editorBlockPayload{}
	for _, edit := range payload.Insert {
		if edit.Index < 0 || edit.Index > len(out) {
			return nil, errors.New("insert: index " + strconv.Itoa(edit.Index) + " is past the end")
		}
		inserts[edit.Index] = append(inserts[edit.Index], edit.Block)
	}
	next := make([]editorBlockPayload, 0, len(out)+len(payload.Insert)+len(payload.Append))
	for i := 0; i <= len(out); i++ {
		next = append(next, inserts[i]...)
		if i < len(out) && !removed[i] {
			next = append(next, out[i])
		}
	}
	next = append(next, payload.Append...)
	if payload.Move != nil {
		from, to := payload.Move.From, payload.Move.To
		if from < 0 || from >= len(next) || to < 0 || to >= len(next) {
			return nil, errors.New("move: indexes must be within the page")
		}
		block := next[from]
		next = append(next[:from], next[from+1:]...)
		rest := append([]editorBlockPayload{}, next[to:]...)
		next = append(append(next[:to], block), rest...)
	}
	return next, nil
}

func (h *Host) finishAgentPageWrite(w http.ResponseWriter, r *http.Request, page cmsstore.Page, payload agentPageWrite, blocks []editorBlockPayload) {
	save := editorSavePayload{Title: page.Title, Slug: page.Slug, Description: pageMetaValue(page, "metaDescription", page.Description), NavParent: PageNavParent(page), PublishAt: page.Metadata[publishAtKey], Blocks: blocks}
	if payload.Title != nil {
		save.Title = *payload.Title
	}
	if payload.Slug != nil {
		save.Slug = *payload.Slug
	}
	if payload.Description != nil {
		save.Description = *payload.Description
	}
	if payload.NavParent != nil {
		save.NavParent = *payload.NavParent
	}
	result := h.applyPageSave(r, page, save)
	if !result.OK {
		agentError(w, http.StatusUnprocessableEntity, "invalid", result.Message)
		return
	}
	h.agentAudit(r, "page.saved", "Changed the page “"+save.Title+"”")
	updated, _, _ := h.store.PageByID(page.ID)
	if payload.Publish && h.publishForAgent(w, r, &updated) != nil {
		return
	}
	agentJSON(w, http.StatusOK, h.agentPageView(r, updated))
}

func (h *Host) handleAgentPagePublish(w http.ResponseWriter, r *http.Request) {
	if !h.requireReady(w) {
		return
	}
	page, ok := h.agentPageByRef(w, r.PathValue("id"))
	if !ok {
		return
	}
	if h.publishForAgent(w, r, &page) != nil {
		return
	}
	agentJSON(w, http.StatusOK, h.agentPageView(r, page))
}

type agentActionRequest struct {
	Action string `json:"action"`
}

func (h *Host) handleAgentPageAction(w http.ResponseWriter, r *http.Request) {
	if !h.requireReady(w) {
		return
	}
	page, ok := h.agentPageByRef(w, r.PathValue("id"))
	if !ok {
		return
	}
	var payload agentActionRequest
	if !decodeAgentBody(w, r, &payload, 4<<10) {
		return
	}
	action := strings.ToLower(strings.TrimSpace(payload.Action))
	scope := scopeWrite
	if action == "offline" || action == "online" || action == "archive" || action == "restore" {
		scope = scopePublish
	}
	if !h.requireScope(w, r, scope) {
		return
	}
	message, err := h.pageAction(page.ID, action)
	if err != nil {
		agentError(w, http.StatusUnprocessableEntity, "invalid", "Unknown action “"+action+"”. Use offline, online, archive, restore, hide, show, up, or down.")
		return
	}
	h.agentAudit(r, "page."+action, message)
	updated, _, _ := h.store.PageByID(page.ID)
	agentJSON(w, http.StatusOK, map[string]any{"message": message, "page": h.agentPageRow(r, updated)})
}

func (h *Host) handleAgentPageMarkdown(w http.ResponseWriter, r *http.Request) {
	if !h.requireReady(w) || !h.requireScope(w, r, scopeRead) {
		return
	}
	page, ok := h.agentPageByRef(w, r.PathValue("id"))
	if !ok {
		return
	}
	w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	_, _ = w.Write([]byte(h.pageMarkdown(page, h.absoluteBase(r))))
}

// ---------- posts ----------

type agentPostRow struct {
	ID      string    `json:"id"`
	Title   string    `json:"title"`
	Slug    string    `json:"slug"`
	URL     string    `json:"url"`
	EditURL string    `json:"editUrl"`
	Status  string    `json:"status"`
	Live    bool      `json:"live"`
	Excerpt string    `json:"excerpt,omitempty"`
	Author  string    `json:"author,omitempty"`
	Tags    []string  `json:"tags,omitempty"`
	Updated time.Time `json:"updated"`
}

type agentPost struct {
	agentPostRow
	Blocks []editorBlockPayload `json:"blocks"`
	Checks []string             `json:"checks"`
}

type agentPostWrite struct {
	Title   *string              `json:"title,omitempty"`
	Slug    *string              `json:"slug,omitempty"`
	Excerpt *string              `json:"excerpt,omitempty"`
	Author  *string              `json:"author,omitempty"`
	Tags    []string             `json:"tags,omitempty"`
	Blocks  []editorBlockPayload `json:"blocks"`
	Publish bool                 `json:"publish,omitempty"`
}

func (h *Host) agentPostRow(r *http.Request, post cmsstore.Post) agentPostRow {
	status := "draft"
	if post.State.Publish == cmsstore.PublishStatePublished {
		status = "published"
	}
	_, live := h.livePost(post)
	base := h.absoluteBase(r)
	return agentPostRow{ID: post.ID, Title: post.Title, Slug: post.Slug, URL: base + postPath(post.Slug), EditURL: base + "/admin/edit/post/" + post.ID, Status: status, Live: live, Excerpt: post.Excerpt, Author: post.Author, Tags: post.Tags, Updated: post.Updated}
}

func (h *Host) agentPostView(r *http.Request, post cmsstore.Post) agentPost {
	return agentPost{agentPostRow: h.agentPostRow(r, post), Blocks: h.documentPayload(post.Body), Checks: h.readinessChecks("post", "", post.Body)}
}

func (h *Host) requireFeatureForAgent(w http.ResponseWriter, feature string) bool {
	if h.featureOn(feature) {
		return true
	}
	agentError(w, http.StatusForbidden, "not_in_plan", "This site's plan does not include "+feature+".")
	return false
}

func (h *Host) handleAgentPosts(w http.ResponseWriter, r *http.Request) {
	if !h.requireReady(w) || !h.requireScope(w, r, scopeRead) || !h.requireFeatureForAgent(w, FeatureBlog) {
		return
	}
	posts, _ := h.store.ListPosts(cmsstore.PostFilter{})
	rows := make([]agentPostRow, 0, len(posts))
	for _, post := range posts {
		rows = append(rows, h.agentPostRow(r, post))
	}
	agentJSON(w, http.StatusOK, map[string]any{"posts": rows})
}

func (h *Host) agentPostByRef(w http.ResponseWriter, ref string) (cmsstore.Post, bool) {
	ref = strings.TrimSpace(ref)
	if post, ok, _ := h.store.PostByID(ref); ok {
		return post, true
	}
	if post, ok, _ := h.store.PostBySlug(normalizeSlug(ref)); ok {
		return post, true
	}
	agentError(w, http.StatusNotFound, "not_found", "No post has the id or address “"+ref+"”.")
	return cmsstore.Post{}, false
}

func (h *Host) handleAgentPost(w http.ResponseWriter, r *http.Request) {
	if !h.requireReady(w) || !h.requireScope(w, r, scopeRead) || !h.requireFeatureForAgent(w, FeatureBlog) {
		return
	}
	post, ok := h.agentPostByRef(w, r.PathValue("id"))
	if !ok {
		return
	}
	agentJSON(w, http.StatusOK, h.agentPostView(r, post))
}

func (h *Host) handleAgentPostCreate(w http.ResponseWriter, r *http.Request) {
	if !h.requireReady(w) || !h.requireScope(w, r, scopeWrite) || !h.requireFeatureForAgent(w, FeatureBlog) {
		return
	}
	var payload agentPostWrite
	if !decodeAgentBody(w, r, &payload, 4<<20) {
		return
	}
	title := ""
	if payload.Title != nil {
		title = strings.TrimSpace(*payload.Title)
	}
	if title == "" {
		agentError(w, http.StatusUnprocessableEntity, "invalid", "Give the post a title.")
		return
	}
	slug := h.freePostSlug(normalizeSlug(title), "")
	if payload.Slug != nil && strings.TrimSpace(*payload.Slug) != "" {
		slug = h.freePostSlug(normalizeSlug(*payload.Slug), "")
	}
	if slug == "" {
		slug = h.freePostSlug("post", "")
	}
	body := h.payloadDocument(payload.Blocks)
	if len(body.Blocks) == 0 {
		body = document(block(0, content.BlockParagraph, values("text", "Start writing your post here.")))
	}
	post, err := h.store.CreatePost(cmsstore.PostInput{Slug: slug, Title: title, Body: body})
	if err != nil {
		agentError(w, http.StatusInternalServerError, "failed", "The post could not be created.")
		return
	}
	if payload.Excerpt != nil || payload.Author != nil || payload.Tags != nil {
		save := postSavePayload{editorSavePayload: editorSavePayload{Title: title, Slug: slug, Blocks: h.documentPayload(post.Body)}}
		if payload.Excerpt != nil {
			save.Excerpt = *payload.Excerpt
		}
		if payload.Author != nil {
			save.Author = *payload.Author
		}
		save.Tags = strings.Join(payload.Tags, ", ")
		if result := h.applyPostSave(r, post, save); !result.OK {
			agentError(w, http.StatusUnprocessableEntity, "invalid", result.Message)
			return
		}
		post, _, _ = h.store.PostByID(post.ID)
	}
	h.agentAudit(r, "post.created", "Created the post “"+title+"”")
	if payload.Publish && h.publishPostForAgent(w, r, &post) != nil {
		return
	}
	agentJSON(w, http.StatusCreated, h.agentPostView(r, post))
}

func (h *Host) publishPostForAgent(w http.ResponseWriter, r *http.Request, post *cmsstore.Post) error {
	if !h.requireScope(w, r, scopePublish) {
		return errors.New("forbidden")
	}
	if h.reviewRequired() {
		if identity, _ := h.agentCaller(r); identity.User.ID == "" || roleRank(identity.User.Role) < roleRank(roleAdmin) {
			agentError(w, http.StatusConflict, "review_required", "This site needs an admin to approve changes before they go live. The draft is saved.")
			return errors.New("review")
		}
	}
	result := h.publishPost(*post)
	if !result.OK {
		agentError(w, http.StatusUnprocessableEntity, "not_published", result.Message)
		return errors.New("not published")
	}
	h.agentAudit(r, "post.published", firstNonEmpty(result.Message, "Published")+": “"+post.Title+"”")
	h.notifyChanged(r, "post", post.ID)
	*post, _, _ = h.store.PostByID(post.ID)
	return nil
}

func (h *Host) handleAgentPostPut(w http.ResponseWriter, r *http.Request) {
	if !h.requireReady(w) || !h.requireScope(w, r, scopeWrite) || !h.requireFeatureForAgent(w, FeatureBlog) {
		return
	}
	post, ok := h.agentPostByRef(w, r.PathValue("id"))
	if !ok {
		return
	}
	var payload agentPostWrite
	if !decodeAgentBody(w, r, &payload, 4<<20) {
		return
	}
	save := postSavePayload{editorSavePayload: editorSavePayload{Title: post.Title, Slug: post.Slug, PublishAt: post.Metadata[publishAtKey], Blocks: h.documentPayload(post.Body)}, Excerpt: post.Excerpt, Author: post.Author, Tags: strings.Join(post.Tags, ", ")}
	if payload.Title != nil {
		save.Title = *payload.Title
	}
	if payload.Slug != nil {
		save.Slug = *payload.Slug
	}
	if payload.Excerpt != nil {
		save.Excerpt = *payload.Excerpt
	}
	if payload.Author != nil {
		save.Author = *payload.Author
	}
	if payload.Tags != nil {
		save.Tags = strings.Join(payload.Tags, ", ")
	}
	if payload.Blocks != nil {
		save.Blocks = payload.Blocks
	}
	result := h.applyPostSave(r, post, save)
	if !result.OK {
		agentError(w, http.StatusUnprocessableEntity, "invalid", result.Message)
		return
	}
	h.agentAudit(r, "post.saved", "Changed the post “"+save.Title+"”")
	updated, _, _ := h.store.PostByID(post.ID)
	if payload.Publish && h.publishPostForAgent(w, r, &updated) != nil {
		return
	}
	agentJSON(w, http.StatusOK, h.agentPostView(r, updated))
}

func (h *Host) handleAgentPostPublish(w http.ResponseWriter, r *http.Request) {
	if !h.requireReady(w) || !h.requireFeatureForAgent(w, FeatureBlog) {
		return
	}
	post, ok := h.agentPostByRef(w, r.PathValue("id"))
	if !ok {
		return
	}
	if h.publishPostForAgent(w, r, &post) != nil {
		return
	}
	agentJSON(w, http.StatusOK, h.agentPostView(r, post))
}

// ---------- products ----------

type agentProduct struct {
	ID          string         `json:"id"`
	Slug        string         `json:"slug"`
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	Price       string         `json:"price"`
	PriceCents  int64          `json:"priceCents"`
	Compare     string         `json:"compare,omitempty"`
	Currency    string         `json:"currency"`
	Images      []ProductImage `json:"images,omitempty"`
	Stock       int            `json:"stock"`
	TrackStock  bool           `json:"trackStock"`
	Ships       bool           `json:"ships"`
	Active      bool           `json:"active"`
	Kind        string         `json:"kind,omitempty"`
	URL         string         `json:"url"`
	Updated     time.Time      `json:"updated"`
}

type agentProductWrite struct {
	Name        *string        `json:"name,omitempty"`
	Slug        *string        `json:"slug,omitempty"`
	Description *string        `json:"description,omitempty"`
	Price       *string        `json:"price,omitempty"`
	Compare     *string        `json:"compare,omitempty"`
	Images      []ProductImage `json:"images,omitempty"`
	Stock       *int           `json:"stock,omitempty"`
	TrackStock  *bool          `json:"trackStock,omitempty"`
	Ships       *bool          `json:"ships,omitempty"`
	Active      *bool          `json:"active,omitempty"`
	Kind        *string        `json:"kind,omitempty"`
}

func (h *Host) agentProductView(r *http.Request, product Product) agentProduct {
	currency := h.currency()
	view := agentProduct{
		ID: product.ID, Slug: product.Slug, Name: product.Name, Description: product.Description, Price: moneyDecimal(product.Price), PriceCents: product.Price,
		Currency: currency, Images: product.Images, Stock: product.Stock, TrackStock: product.TrackStock, Ships: product.Ships, Active: product.Active, Kind: product.Kind,
		URL: h.absoluteBase(r) + shopPath + "/" + product.Slug, Updated: product.Updated,
	}
	if product.Compare > 0 {
		view.Compare = moneyDecimal(product.Compare)
	}
	return view
}

func moneyDecimal(cents int64) string {
	sign := ""
	if cents < 0 {
		sign = "-"
		cents = -cents
	}
	return sign + strconv.FormatInt(cents/100, 10) + "." + twoDigits(int(cents%100))
}

func twoDigits(n int) string {
	if n < 10 {
		return "0" + strconv.Itoa(n)
	}
	return strconv.Itoa(n)
}

func (h *Host) handleAgentProducts(w http.ResponseWriter, r *http.Request) {
	if !h.requireReady(w) || !h.requireScope(w, r, scopeRead) || !h.requireFeatureForAgent(w, FeatureShop) {
		return
	}
	products := h.products.list()
	out := make([]agentProduct, 0, len(products))
	for _, product := range products {
		out = append(out, h.agentProductView(r, product))
	}
	agentJSON(w, http.StatusOK, map[string]any{"products": out, "currency": h.currency()})
}

func (h *Host) handleAgentProduct(w http.ResponseWriter, r *http.Request) {
	if !h.requireReady(w) || !h.requireScope(w, r, scopeRead) || !h.requireFeatureForAgent(w, FeatureShop) {
		return
	}
	product, ok := h.products.get(r.PathValue("id"))
	if !ok {
		agentError(w, http.StatusNotFound, "not_found", "No product has that id.")
		return
	}
	agentJSON(w, http.StatusOK, h.agentProductView(r, product))
}

func (h *Host) handleAgentProductSave(w http.ResponseWriter, r *http.Request) {
	if !h.requireReady(w) || !h.requireScope(w, r, scopeWrite) || !h.requireFeatureForAgent(w, FeatureShop) {
		return
	}
	var product Product
	creating := r.Method == http.MethodPost
	if !creating {
		existing, ok := h.products.get(r.PathValue("id"))
		if !ok {
			agentError(w, http.StatusNotFound, "not_found", "No product has that id.")
			return
		}
		product = existing
	} else {
		product = Product{Active: true, Ships: true, Kind: "physical"}
	}
	var payload agentProductWrite
	if !decodeAgentBody(w, r, &payload, 1<<20) {
		return
	}
	if payload.Name != nil {
		product.Name = strings.TrimSpace(*payload.Name)
	}
	if product.Name == "" {
		agentError(w, http.StatusUnprocessableEntity, "invalid", "Give the product a name.")
		return
	}
	if payload.Slug != nil {
		product.Slug = normalizeSlug(*payload.Slug)
	}
	if payload.Description != nil {
		product.Description = strings.TrimSpace(*payload.Description)
	}
	currency := h.currency()
	if payload.Price != nil {
		price, err := parseMoney(*payload.Price, currency)
		if err != nil {
			agentError(w, http.StatusUnprocessableEntity, "invalid", "The price “"+*payload.Price+"” could not be read. Send a number such as 12.50.")
			return
		}
		product.Price = price
	}
	if payload.Compare != nil {
		compare, err := parseMoney(*payload.Compare, currency)
		if err != nil {
			agentError(w, http.StatusUnprocessableEntity, "invalid", "The compare price could not be read.")
			return
		}
		product.Compare = compare
	}
	if payload.Images != nil {
		images := []ProductImage{}
		for _, image := range payload.Images {
			if url := safeLinkHref(strings.TrimSpace(image.URL)); url != "" {
				images = append(images, ProductImage{URL: url, Alt: strings.TrimSpace(image.Alt)})
			}
		}
		product.Images = images
	}
	if payload.Stock != nil {
		product.Stock = *payload.Stock
	}
	if payload.TrackStock != nil {
		product.TrackStock = *payload.TrackStock
	}
	if payload.Ships != nil {
		product.Ships = *payload.Ships
	}
	if payload.Active != nil {
		product.Active = *payload.Active
	}
	if payload.Kind != nil {
		product.Kind = strings.ToLower(strings.TrimSpace(*payload.Kind))
	}
	saved, err := h.products.put(product)
	if err != nil {
		agentError(w, http.StatusInternalServerError, "failed", "The product could not be saved.")
		return
	}
	event, status := "product.saved", http.StatusOK
	if creating {
		event, status = "product.created", http.StatusCreated
	}
	h.agentAudit(r, event, "Saved the product “"+saved.Name+"”")
	agentJSON(w, status, h.agentProductView(r, saved))
}

func (h *Host) handleAgentProductDelete(w http.ResponseWriter, r *http.Request) {
	if !h.requireReady(w) || !h.requireScope(w, r, scopeWrite) || !h.requireFeatureForAgent(w, FeatureShop) {
		return
	}
	product, ok := h.products.get(r.PathValue("id"))
	if !ok {
		agentError(w, http.StatusNotFound, "not_found", "No product has that id.")
		return
	}
	if err := h.products.remove(product.ID); err != nil {
		agentError(w, http.StatusInternalServerError, "failed", "The product could not be removed.")
		return
	}
	h.agentAudit(r, "product.deleted", "Removed the product “"+product.Name+"”")
	agentJSON(w, http.StatusOK, map[string]any{"ok": true, "id": product.ID})
}

// ---------- media ----------

type agentMediaUpload struct {
	Data string `json:"data"` // base64 image bytes, with or without a data: prefix
	URL  string `json:"url"`  // or: fetch this picture and keep a copy
	Name string `json:"name"`
}

func (h *Host) handleAgentMedia(w http.ResponseWriter, r *http.Request) {
	if !h.requireReady(w) || !h.requireScope(w, r, scopeRead) {
		return
	}
	entries := h.media.list()
	base := h.absoluteBase(r)
	out := make([]map[string]any, 0, len(entries))
	for _, entry := range entries {
		out = append(out, map[string]any{"url": entry.url(), "absoluteUrl": base + entry.url(), "thumb": entry.thumb(), "width": entry.Width, "height": entry.Height, "size": entry.Size, "uploaded": entry.Uploaded})
	}
	agentJSON(w, http.StatusOK, map[string]any{"pictures": out})
}

func (h *Host) handleAgentMediaUpload(w http.ResponseWriter, r *http.Request) {
	if !h.requireReady(w) || !h.requireScope(w, r, scopeWrite) {
		return
	}
	var data []byte
	contentType := r.Header.Get("Content-Type")
	switch {
	case strings.HasPrefix(contentType, "multipart/form-data"):
		if err := r.ParseMultipartForm(maxUploadBytes + 1024); err != nil {
			agentError(w, http.StatusBadRequest, "bad_request", "The upload could not be read.")
			return
		}
		file, _, err := r.FormFile("file")
		if err != nil {
			agentError(w, http.StatusBadRequest, "bad_request", "Send the picture as the “file” part.")
			return
		}
		defer file.Close()
		data, _ = io.ReadAll(io.LimitReader(file, maxUploadBytes+1))
	case strings.HasPrefix(contentType, "image/"):
		data, _ = io.ReadAll(io.LimitReader(r.Body, maxUploadBytes+1))
	default:
		var payload agentMediaUpload
		if !decodeAgentBody(w, r, &payload, 16<<20) {
			return
		}
		raw := strings.TrimSpace(payload.Data)
		if raw == "" && payload.URL != "" {
			fetched, err := h.fetchRemoteImage(r.Context(), payload.URL)
			if err != nil {
				agentError(w, http.StatusUnprocessableEntity, "invalid", "The picture at that address could not be fetched: "+err.Error())
				return
			}
			data = fetched
		} else {
			if comma := strings.Index(raw, ","); strings.HasPrefix(raw, "data:") && comma > 0 {
				raw = raw[comma+1:]
			}
			decoded, err := base64.StdEncoding.DecodeString(raw)
			if err != nil {
				decoded, err = base64.RawStdEncoding.DecodeString(raw)
			}
			if err != nil {
				agentError(w, http.StatusBadRequest, "bad_request", "Send the picture as base64 in “data”, or a fetchable address in “url”.")
				return
			}
			data = decoded
		}
	}
	url, err := h.storeUpload(strings.NewReader(string(data)))
	if err != nil {
		switch {
		case errors.Is(err, errUploadTooLarge):
			agentError(w, http.StatusRequestEntityTooLarge, "too_large", "Pictures can be up to 10 MB.")
		case errors.Is(err, errUploadNotImage):
			agentError(w, http.StatusUnsupportedMediaType, "not_an_image", "Send a PNG, JPEG, GIF, or WebP picture.")
		default:
			agentError(w, http.StatusInternalServerError, "failed", "The picture could not be stored.")
		}
		return
	}
	h.agentAudit(r, "media.uploaded", "Added a picture")
	agentJSON(w, http.StatusCreated, map[string]any{"url": url, "absoluteUrl": h.absoluteBase(r) + url})
}

func (h *Host) fetchRemoteImage(ctx context.Context, address string) ([]byte, error) {
	if !strings.HasPrefix(address, "https://") && !strings.HasPrefix(address, "http://") {
		return nil, errors.New("only http(s) addresses can be fetched")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, address, nil)
	if err != nil {
		return nil, err
	}
	client := &http.Client{Timeout: 20 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("the server answered " + resp.Status)
	}
	return io.ReadAll(io.LimitReader(resp.Body, maxUploadBytes+1))
}

// ---------- look ----------

type agentLook struct {
	Palette  string `json:"palette"`
	Fonts    string `json:"fonts"`
	Accent   string `json:"accent,omitempty"`
	Buttons  string `json:"buttons"`
	Spacing  string `json:"spacing"`
	Headings string `json:"headings"`
	Width    string `json:"width"`
	Ground   string `json:"ground,omitempty"`
	Ink      string `json:"ink,omitempty"`
	FontHead string `json:"fontHead,omitempty"`
	FontBody string `json:"fontBody,omitempty"`
}

func (h *Host) agentLookView() agentLook {
	m := h.settings().Metadata
	theme := h.theme()
	return agentLook{
		Palette: firstNonEmpty(m[themePaletteKey], theme.Palette.Key), Fonts: firstNonEmpty(m[themeFontsKey], theme.Fonts.Key), Accent: theme.Accent,
		Buttons: firstNonEmpty(theme.Buttons, "soft"), Spacing: firstNonEmpty(theme.Spacing, "normal"), Headings: firstNonEmpty(theme.Headings, "normal"), Width: firstNonEmpty(theme.Width, "normal"),
		Ground: theme.Ground, Ink: theme.Ink, FontHead: theme.FontHead, FontBody: theme.FontBody,
	}
}

func (h *Host) handleAgentLook(w http.ResponseWriter, r *http.Request) {
	if !h.requireReady(w) || !h.requireScope(w, r, scopeRead) {
		return
	}
	agentJSON(w, http.StatusOK, h.agentLookView())
}

func (h *Host) handleAgentLookPut(w http.ResponseWriter, r *http.Request) {
	if !h.requireReady(w) || !h.requireScope(w, r, scopeSettings) {
		return
	}
	current := h.agentLookView()
	var patch map[string]*string
	if !decodeAgentBody(w, r, &patch, 64<<10) {
		return
	}
	pick := func(key, fallback string) string {
		if value, ok := patch[key]; ok && value != nil {
			return strings.TrimSpace(*value)
		}
		return fallback
	}
	choice := ThemeChoice{
		Palette: pick("palette", current.Palette), Fonts: pick("fonts", current.Fonts), Accent: pick("accent", current.Accent), Buttons: pick("buttons", current.Buttons),
		Spacing: pick("spacing", current.Spacing), Headings: pick("headings", current.Headings), Width: pick("width", current.Width), Ground: pick("ground", current.Ground), Ink: pick("ink", current.Ink),
		FontHead: pick("fontHead", current.FontHead), FontBody: pick("fontBody", current.FontBody),
	}
	if _, err := h.SaveTheme(choice); err != nil {
		agentError(w, http.StatusInternalServerError, "failed", "The Look could not be saved.")
		return
	}
	h.agentAudit(r, "look.saved", "Changed the Look")
	agentJSON(w, http.StatusOK, h.agentLookView())
}

// ---------- presets, forms, messages, stats, activity ----------

func (h *Host) handleAgentPresets(w http.ResponseWriter, r *http.Request) {
	if !h.requireReady(w) || !h.requireScope(w, r, scopeRead) {
		return
	}
	presets := h.presets.list()
	out := make([]map[string]any, 0, len(presets))
	for _, preset := range presets {
		out = append(out, map[string]any{"id": preset.ID, "name": preset.Name, "kind": preset.Kind, "created": preset.Created, "block": h.blockPayload(preset.Block)})
	}
	agentJSON(w, http.StatusOK, map[string]any{"presets": out})
}

func (h *Host) handleAgentPresetSave(w http.ResponseWriter, r *http.Request) {
	if !h.requireReady(w) || !h.requireScope(w, r, scopeWrite) {
		return
	}
	var payload presetRequest
	if !decodeAgentBody(w, r, &payload, 1<<20) {
		return
	}
	name := strings.TrimSpace(payload.Name)
	if name == "" {
		agentError(w, http.StatusUnprocessableEntity, "invalid", "Give the preset a name.")
		return
	}
	doc := h.payloadDocument([]editorBlockPayload{payload.Block})
	if len(doc.Blocks) == 0 {
		agentError(w, http.StatusUnprocessableEntity, "invalid", "The block is empty or its kind is unknown. See /agent/v1/schema.")
		return
	}
	preset, err := h.presets.add(name, editorKind(doc.Blocks[0].Key), doc.Blocks[0])
	if err != nil {
		agentError(w, http.StatusInternalServerError, "failed", "The preset could not be saved.")
		return
	}
	h.agentAudit(r, "preset.saved", "Saved the preset “"+name+"”")
	agentJSON(w, http.StatusCreated, map[string]any{"id": preset.ID, "name": preset.Name, "kind": preset.Kind})
}

func (h *Host) handleAgentPresetDelete(w http.ResponseWriter, r *http.Request) {
	if !h.requireReady(w) || !h.requireScope(w, r, scopeWrite) {
		return
	}
	if _, ok := h.presets.get(r.PathValue("id")); !ok {
		agentError(w, http.StatusNotFound, "not_found", "No preset has that id.")
		return
	}
	if err := h.presets.remove(r.PathValue("id")); err != nil {
		agentError(w, http.StatusInternalServerError, "failed", "The preset could not be removed.")
		return
	}
	agentJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (h *Host) handleAgentForms(w http.ResponseWriter, r *http.Request) {
	if !h.requireReady(w) || !h.requireScope(w, r, scopeRead) {
		return
	}
	forms := h.forms.list()
	out := []map[string]any{{"id": "contact", "name": "Contact", "fields": []string{"name", "email", "message"}, "builtIn": true}}
	for _, form := range forms {
		fields := []string{}
		for _, field := range form.Fields {
			fields = append(fields, field.Label)
		}
		out = append(out, map[string]any{"id": form.ID, "name": form.Name, "fields": fields, "button": form.Button})
	}
	agentJSON(w, http.StatusOK, map[string]any{"forms": out, "hint": "Place a form on a page with a block of kind \"form\" whose \"form\" is one of these ids."})
}

func (h *Host) handleAgentMessages(w http.ResponseWriter, r *http.Request) {
	if !h.requireReady(w) || !h.requireScope(w, r, scopeRead) {
		return
	}
	messages, _ := h.messages.list()
	limit := 50
	if raw := r.URL.Query().Get("limit"); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n > 0 && n <= 500 {
			limit = n
		}
	}
	sort.Slice(messages, func(i, j int) bool { return messages[i].Received.After(messages[j].Received) })
	if len(messages) > limit {
		messages = messages[:limit]
	}
	agentJSON(w, http.StatusOK, map[string]any{"messages": messages, "unread": h.unreadMessages()})
}

func (h *Host) handleAgentStats(w http.ResponseWriter, r *http.Request) {
	if !h.requireReady(w) || !h.requireScope(w, r, scopeRead) || !h.requireFeatureForAgent(w, FeatureStats) {
		return
	}
	days := 30
	if raw := r.URL.Query().Get("days"); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n > 0 && n <= 365 {
			days = n
		}
	}
	summary := h.stats.summary(timeNow(), days)
	rows := func(in []statsRow) []map[string]any {
		out := make([]map[string]any, 0, len(in))
		for _, row := range in {
			out = append(out, map[string]any{"label": row.Label, "count": row.Count})
		}
		return out
	}
	points := make([]map[string]any, 0, len(summary.Days))
	for _, point := range summary.Days {
		points = append(points, map[string]any{"day": point.Day, "views": point.Views, "visitors": point.Visitors})
	}
	agentJSON(w, http.StatusOK, map[string]any{"days": days, "views": summary.Views, "visitors": summary.Visitors, "weekVisitors": summary.WeekVisitors, "byDay": points, "pages": rows(summary.Pages), "sources": rows(summary.Sources), "devices": rows(summary.Devices), "empty": summary.Empty})
}

func (h *Host) handleAgentActivity(w http.ResponseWriter, r *http.Request) {
	if !h.requireReady(w) || !h.requireScope(w, r, scopeRead) {
		return
	}
	entries := h.auditLog.newest(50)
	out := make([]map[string]any, 0, len(entries))
	for _, entry := range entries {
		out = append(out, map[string]any{"at": entry.At, "who": entry.User, "event": entry.Event, "summary": entry.Summary})
	}
	agentJSON(w, http.StatusOK, map[string]any{"activity": out})
}
