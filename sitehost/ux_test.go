package sitehost

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestTheEditorHasThePaletteShortcutsCoachAndFinder(t *testing.T) {
	host, handler := newTestHost(t)
	id := firstPageID(t, host, "home")
	editor := get(t, handler, "/admin/edit/"+id).Body.String()
	for _, want := range []string{
		`data-palette-open="true"`, `<kbd>Ctrl K</kbd>`, `class="ed-palette" data-palette="true" hidden="hidden" role="dialog"`, `role="combobox"`, `id="ed-palette-list" role="listbox"`,
		`data-shortcuts="true"`, `<kbd>Alt+↑</kbd>`, `data-shortcuts-open="true"`,
		`class="ed-coach" data-coach="true" hidden="hidden"`, `Hover a section for its tools`,
		`data-add-filter="true"`, `Find a section… pricing, hours, map`,
		`class="ed-empty" data-empty="true" hidden="hidden"`, `Add your first section`,
		`data-preview-draft="true"`, `href="/admin/preview/` + id + `"`, `>View live</a>`,
		`<datalist id="site-links">`, `<option value="/menu" label="Menu">`, `list="site-links"`,
		`data-pages="true"`, `"href":"/admin/edit/` + id + `"`,
		`data-label="Hero"`, `class="ed-fab"`, `data-side-toggle="true"`, `id="ed-side"`, `data-side-close="true"`,
		`src="` + webMCPScriptPath + `"`, `data-kind="page"`,
	} {
		mustContain(t, editor, want, "the editor carries the UX pass")
	}
	if strings.Contains(editor, "Ask your site") || strings.Contains(editor, "data-assistant") {
		t.Fatal("the built-in assistant is gone")
	}
	js := get(t, handler, editorScriptPath).Body.String()
	for _, want := range []string{"function openPalette", "function buildCommands", "function removeBlock", "function moveBlock", `key.toLowerCase() === "k"`, "gosx.editor.coach", "window.gosxEditor =", `"Undo", function () { undo(); }`} {
		mustContain(t, js, want, "the editor script has the palette, keys, and toast")
	}
	if strings.Contains(js, "data-assistant") {
		t.Fatal("no assistant code remains in the script")
	}
	css := get(t, handler, publicStylesheetPath).Body.String()
	for _, want := range []string{".ed-palette__item.is-active", ".ed-block::before { content: attr(data-label)", ".ed-side.is-open { transform: none; }", ".ed-fab { display: inline-flex", ".ed-toast__action"} {
		mustContain(t, css, want, "the stylesheet has the new pieces")
	}
}

func TestOwnersPreviewTheirDraftBeforePublishing(t *testing.T) {
	host, handler := newTestHost(t)
	id := firstPageID(t, host, "menu")
	mustContain(t, postJSON(t, handler, "/admin/api/pages/"+id, `{"title":"Menu","slug":"menu","description":"x","blocks":[{"kind":"paragraph","text":"Only in the draft for now."}]}`).Body.String(), `"ok":true`, "the draft saves")
	if strings.Contains(get(t, handler, "/menu").Body.String(), "Only in the draft for now.") {
		t.Fatal("visitors do not see the draft")
	}
	preview := get(t, handler, "/admin/preview/"+id)
	if preview.Code != http.StatusOK {
		t.Fatalf("preview: %d", preview.Code)
	}
	body := preview.Body.String()
	mustContain(t, body, "Only in the draft for now.", "the preview shows the draft")
	mustContain(t, body, `data-preview-banner="true"`, "with a preview banner")
	mustContain(t, body, `href="/admin/edit/`+id+`">Back to editing</a>`, "and a way back")
	if strings.Contains(body, `contenteditable`) || strings.Contains(body, "ed-block") {
		t.Fatal("the preview is the public rendering, not the editor")
	}
	if get(t, handler, "/admin/preview/nope").Code != http.StatusNotFound {
		t.Fatal("an unknown page is not found")
	}
	postID := createPost(t, handler, "Our oven")
	savePost(t, handler, postID, `{"title":"Our oven","slug":"our-oven","blocks":[{"kind":"paragraph","text":"Draft post words."}]}`)
	mustContain(t, get(t, handler, "/admin/preview/post/"+postID).Body.String(), "Draft post words.", "posts preview too")
	mustContain(t, get(t, handler, "/admin/edit/post/"+postID).Body.String(), `href="/admin/preview/post/`+postID+`"`, "the post editor links its preview")
}

func TestPreviewNeedsASignedInPerson(t *testing.T) {
	host, handler := newGuardedHost(t)
	id := firstPageID(t, host, "menu")
	rec := get(t, handler, "/admin/preview/"+id)
	if rec.Code != http.StatusSeeOther || !strings.Contains(rec.Header().Get("Location"), "/admin/login") {
		t.Fatalf("preview without a session: %d %q", rec.Code, rec.Header().Get("Location"))
	}
}

func TestTheBrowserItselfCanUseTheAgentAPIWithTheCSRFToken(t *testing.T) {
	host, handler := newTestHost(t)
	id := firstPageID(t, host, "menu")
	// No key, no token: refused.
	if rec := agentCall(t, handler, http.MethodGet, "/agent/v1/site", "", ""); rec.Code != http.StatusUnauthorized {
		t.Fatalf("no token: %d", rec.Code)
	}
	// The page's own script sends the CSRF token; on a laptop site that is the owner.
	req := httptest.NewRequest(http.MethodPatch, "/agent/v1/pages/"+id, strings.NewReader(`{"append":[{"kind":"paragraph","text":"From the browser's assistant."}]}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(csrfHeader, csrfToken(t, handler))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("browser call: %d %s", rec.Code, rec.Body.String())
	}
	page, _, _ := host.Store().PageByID(id)
	if last := page.Body.Blocks[len(page.Body.Blocks)-1]; last.Values["text"].String != "From the browser's assistant." {
		t.Fatalf("the change landed: %+v", last.Values)
	}
	mustContain(t, get(t, handler, "/admin/activity").Body.String(), "Changed the page", "and is audited")
	// A wrong token is refused, as is a cross-site origin.
	req = httptest.NewRequest(http.MethodGet, "/agent/v1/site", nil)
	req.Header.Set(csrfHeader, "nope")
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("bad token: %d", rec.Code)
	}
	req = httptest.NewRequest(http.MethodGet, "/agent/v1/site", nil)
	req.Header.Set(csrfHeader, csrfToken(t, handler))
	req.Header.Set("Sec-Fetch-Site", "cross-site")
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("cross-site: %d", rec.Code)
	}
}

func TestASignedInEditorGetsTheirOwnScopesThroughTheBrowser(t *testing.T) {
	host, handler := newGuardedHost(t)
	id := firstPageID(t, host, "menu")
	// Nobody signed in: the CSRF token alone is not enough on a guarded site.
	req := httptest.NewRequest(http.MethodGet, "/agent/v1/site", nil)
	req.Header.Set(csrfHeader, csrfToken(t, handler))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("signed out: %d", rec.Code)
	}
	editorCookie := signInAs(t, host, handler, "eve@example.com", "Eve", roleEditor)
	token := csrfTokenFrom(getWithCookie(t, handler, "/admin", editorCookie).Body.String())
	if token == "" {
		t.Fatal("a signed-in editor sees a CSRF token")
	}
	req = httptest.NewRequest(http.MethodPost, "/agent/v1/pages/"+id+"/publish", nil)
	req.Header.Set(csrfHeader, token)
	req.AddCookie(editorCookie)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("an editor cannot publish through the browser tools: %d %s", rec.Code, rec.Body.String())
	}
	req = httptest.NewRequest(http.MethodGet, "/agent/v1/pages", nil)
	req.Header.Set(csrfHeader, token)
	req.AddCookie(editorCookie)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("an editor can read: %d", rec.Code)
	}
}

func TestAdminPagesAnnounceWebMCPTools(t *testing.T) {
	_, handler := newTestHost(t)
	dashboard := get(t, handler, "/admin").Body.String()
	mustContain(t, dashboard, `src="`+webMCPScriptPath+`"`, "every admin page loads the WebMCP script")
	pages := get(t, handler, "/admin/pages").Body.String()
	mustContain(t, pages, `toolname="create_page_form" tooldescription="Add a new page`, "forms carry declarative tool names")
	mustContain(t, get(t, handler, "/admin/settings").Body.String(), `toolname="site_settings_form"`, "settings too")
	agents := get(t, handler, "/admin/agents").Body.String()
	mustContain(t, agents, "assistant can do here", "the Agents page explains WebMCP")
	mustContain(t, agents, `data-webmcp-tools="true"`, "and lists the tools")
	if strings.Contains(agents, "anthropic://") || strings.Contains(agents, "Ask your site") {
		t.Fatal("no built-in assistant is offered")
	}
	script := get(t, handler, webMCPScriptPath)
	mustContain(t, script.Header().Get("Content-Type"), "javascript", "the script is served")
	js := script.Body.String()
	for _, want := range []string{"navigator.modelContext", "registerTool", "provideContext", `name: "add_section"`, `name: "publish_this_page"`, `name: "get_site"`, `name: "open_page_in_editor"`, "X-CSRF-Token", "window.gosxWebMCP"} {
		mustContain(t, js, want, "the script announces the tools")
	}
}

func TestSectionFinderAndLinkListRenderForPosts(t *testing.T) {
	_, handler := newTestHost(t)
	postID := createPost(t, handler, "Our oven")
	editor := get(t, handler, "/admin/edit/post/"+postID).Body.String()
	mustContain(t, editor, `data-kind="post"`, "the post editor says what it edits")
	mustContain(t, editor, `data-add-filter="true"`, "with the finder")
	mustContain(t, editor, `<datalist id="site-links">`, "and the link list")
	mustContain(t, editor, `This post is empty.`, "and its own empty state")
	_ = url.Values{}
}

// signInAs makes a person with the given role and hands back their session
// cookie, the way a browser would carry it.
func signInAs(t *testing.T, host *Host, handler http.Handler, email, name, role string) *http.Cookie {
	t.Helper()
	// The guarded host needs an owner first; the first account is always one.
	if host.users.count() == 0 {
		if _, err := host.users.put(User{Email: "owner@example.com", Name: "Owner", Role: roleAdmin}); err != nil {
			t.Fatal(err)
		}
	}
	user, err := host.users.put(User{Email: email, Name: name, Role: role})
	if err != nil {
		t.Fatal(err)
	}
	token, err := host.users.newSession(user.ID)
	if err != nil {
		t.Fatal(err)
	}
	return &http.Cookie{Name: sessionCookie, Value: token}
}

func csrfTokenFrom(body string) string {
	marker := `<meta name="csrf-token" content="`
	start := strings.Index(body, marker)
	if start < 0 {
		return ""
	}
	rest := body[start+len(marker):]
	return rest[:strings.Index(rest, `"`)]
}
