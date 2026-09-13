package sitehost

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// agentCall makes one agent API request with a bearer key.
func agentCall(t *testing.T, handler http.Handler, method, path, token, body string) *httptest.ResponseRecorder {
	t.Helper()
	var reader *strings.Reader
	if body != "" {
		reader = strings.NewReader(body)
	} else {
		reader = strings.NewReader("")
	}
	req := httptest.NewRequest(method, path, reader)
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec
}

func agentKeyFor(t *testing.T, host *Host, scopes ...string) string {
	t.Helper()
	_, token, err := host.agents.create("test", scopes)
	if err != nil {
		t.Fatal(err)
	}
	return token
}

func decodeJSON(t *testing.T, rec *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var out map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("not JSON (%d): %s", rec.Code, rec.Body.String())
	}
	return out
}

func TestAdminsMakeAndRevokeAgentKeys(t *testing.T) {
	host, handler := newTestHost(t)
	page := get(t, handler, "/admin/agents").Body.String()
	mustContain(t, page, "Make a key", "the Agents page exists")
	mustContain(t, page, "No keys yet", "and starts empty")
	mustContain(t, page, `href="/llms.txt"`, "and explains how agents read the site")

	rec := post(t, handler, "/admin/agents", url.Values{"name": {"Claude Code on my laptop"}, "scope": {"read", "write"}})
	body := rec.Body.String()
	mustContain(t, body, "Copy it now and keep it somewhere safe", "the key is shown once")
	token := regexp.MustCompile(`value="(gsk_[a-f0-9]+)"`).FindStringSubmatch(body)
	if token == nil {
		t.Fatalf("no key in the page: %s", body[:400])
	}
	mustContain(t, body, "claude mcp add --transport http wildflower-bakery https://wildflower.example/agent/mcp --header &#34;Authorization: Bearer "+token[1]+"&#34;", "with the Claude Code command")
	mustContain(t, body, `&#34;mcpServers&#34;:{&#34;wildflower-bakery&#34;:{&#34;type&#34;:&#34;http&#34;,&#34;url&#34;:&#34;https://wildflower.example/agent/mcp&#34;`, "and an .mcp.json snippet")
	keys := host.agents.list()
	if len(keys) != 1 || keys[0].Name != "Claude Code on my laptop" || strings.Join(keys[0].Scopes, ",") != "read,write" || keys[0].Hash == token[1] {
		t.Fatalf("stored keys: %+v", keys)
	}
	if !strings.Contains(get(t, handler, "/admin/agents").Body.String(), "gsk_"+token[1][4:10]+"…") {
		t.Fatal("the list shows a hint of the key, not the key")
	}

	site := agentCall(t, handler, http.MethodGet, "/agent/v1/site", token[1], "")
	if site.Code != http.StatusOK || !strings.Contains(site.Body.String(), `"title":"Wildflower Bakery"`) {
		t.Fatalf("site with key: %d %s", site.Code, site.Body.String())
	}
	if used := host.agents.list()[0].LastUsed; used.IsZero() {
		t.Fatal("use is remembered")
	}

	post(t, handler, "/admin/agents/"+keys[0].ID+"/revoke", url.Values{})
	if rec := agentCall(t, handler, http.MethodGet, "/agent/v1/site", token[1], ""); rec.Code != http.StatusUnauthorized {
		t.Fatalf("revoked key still works: %d", rec.Code)
	}
	mustContain(t, get(t, handler, "/admin/agents").Body.String(), "Revoked", "the list says so")
	mustContain(t, get(t, handler, "/admin/activity").Body.String(), "Created the agent key", "the audit log has it")
}

func TestAgentAPIRefusesWithoutAKeyAndOutsideScope(t *testing.T) {
	host, handler := newTestHost(t)
	for _, path := range []string{"/agent/v1", "/agent/v1/schema", "/agent/v1/openapi.json"} {
		if rec := agentCall(t, handler, http.MethodGet, path, "", ""); rec.Code != http.StatusOK {
			t.Fatalf("%s should be open: %d", path, rec.Code)
		}
	}
	index := decodeJSON(t, agentCall(t, handler, http.MethodGet, "/agent/v1", "", ""))
	if index["setupComplete"] != true || index["links"].(map[string]any)["mcp"] != "https://wildflower.example/agent/mcp" {
		t.Fatalf("index: %v", index)
	}
	rec := agentCall(t, handler, http.MethodGet, "/agent/v1/site", "", "")
	if rec.Code != http.StatusUnauthorized || rec.Header().Get("WWW-Authenticate") == "" || !strings.Contains(rec.Body.String(), `"code":"unauthorized"`) {
		t.Fatalf("no key: %d %s", rec.Code, rec.Body.String())
	}
	if rec := agentCall(t, handler, http.MethodGet, "/agent/v1/site", "gsk_nope", ""); rec.Code != http.StatusUnauthorized {
		t.Fatalf("bad key: %d", rec.Code)
	}
	reader := agentKeyFor(t, host, "read")
	if rec := agentCall(t, handler, http.MethodGet, "/agent/v1/pages", reader, ""); rec.Code != http.StatusOK {
		t.Fatalf("read scope reads: %d", rec.Code)
	}
	rec = agentCall(t, handler, http.MethodPatch, "/agent/v1/pages/home", reader, `{"title":"Nope"}`)
	if rec.Code != http.StatusForbidden || !strings.Contains(rec.Body.String(), "“write” scope") {
		t.Fatalf("read scope must not write: %d %s", rec.Code, rec.Body.String())
	}
	if rec := agentCall(t, handler, http.MethodPost, "/agent/v1/pages/home/publish", reader, ""); rec.Code != http.StatusForbidden {
		t.Fatalf("read scope must not publish: %d", rec.Code)
	}
	if rec := agentCall(t, handler, http.MethodGet, "/agent/v1/nothing", reader, ""); rec.Code != http.StatusNotFound || !strings.Contains(rec.Body.String(), "openapi.json") {
		t.Fatalf("unknown call: %d %s", rec.Code, rec.Body.String())
	}
	openapi := decodeJSON(t, agentCall(t, handler, http.MethodGet, "/agent/v1/openapi.json", "", ""))
	if openapi["openapi"] != "3.1.0" || openapi["paths"].(map[string]any)["/agent/v1/pages/{id}"] == nil {
		t.Fatal("the OpenAPI document describes the pages")
	}
	schema := decodeJSON(t, agentCall(t, handler, http.MethodGet, "/agent/v1/schema", "", ""))
	kinds := schema["blockKinds"].([]any)
	found := false
	for _, kind := range kinds {
		k := kind.(map[string]any)
		if k["kind"] == "pricing" && k["items"].(map[string]any)["name"] == "plan" {
			found = true
		}
	}
	if !found {
		t.Fatal("the schema lists pricing with its plan items")
	}
}

func TestAgentsReadAndChangePagesLikeTheEditor(t *testing.T) {
	host, handler := newTestHost(t)
	token := agentKeyFor(t, host, "read", "write", "publish", "settings")
	list := decodeJSON(t, agentCall(t, handler, http.MethodGet, "/agent/v1/pages", token, ""))
	pages := list["pages"].([]any)
	if len(pages) < 3 || pages[0].(map[string]any)["slug"] != "home" {
		t.Fatalf("pages: %v", pages)
	}
	home := decodeJSON(t, agentCall(t, handler, http.MethodGet, "/agent/v1/pages/home", token, ""))
	blocks := home["blocks"].([]any)
	if home["status"] != "published" || len(blocks) == 0 || blocks[0].(map[string]any)["kind"] != "hero" {
		t.Fatalf("home: %v", home)
	}
	hero := blocks[0].(map[string]any)
	if hero["fields"].(map[string]any)["headline"] == "" {
		t.Fatal("the hero comes back with its fields")
	}

	// A small edit: change the hero headline, add a paragraph at the end.
	hero["fields"].(map[string]any)["headline"] = "Bread worth **the walk**"
	edit, _ := json.Marshal(map[string]any{"replace": []map[string]any{{"index": 0, "block": hero}}, "append": []map[string]any{{"kind": "paragraph", "text": "Added by an agent.", "spacing": "roomy"}}, "description": "Fresh bread daily"})
	rec := agentCall(t, handler, http.MethodPatch, "/agent/v1/pages/home", token, string(edit))
	if rec.Code != http.StatusOK {
		t.Fatalf("patch: %d %s", rec.Code, rec.Body.String())
	}
	after := decodeJSON(t, rec)
	blocks = after["blocks"].([]any)
	last := blocks[len(blocks)-1].(map[string]any)
	if last["kind"] != "paragraph" || last["text"] != "Added by an agent." || last["spacing"] != "roomy" || after["description"] != "Fresh bread daily" {
		t.Fatalf("after patch: %v", last)
	}
	if blocks[0].(map[string]any)["fields"].(map[string]any)["headline"] != "Bread worth **the walk**" {
		t.Fatal("the headline changed")
	}
	if !strings.Contains(get(t, handler, "/").Body.String(), "Added by an agent.") {
		if after["live"] != true {
			t.Fatal("home stays live")
		}
	}
	// A draft until published: the public page still shows the old text.
	if strings.Contains(get(t, handler, "/").Body.String(), "Added by an agent.") {
		t.Fatal("changes are drafts until published")
	}
	rec = agentCall(t, handler, http.MethodPost, "/agent/v1/pages/home/publish", token, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("publish: %d %s", rec.Code, rec.Body.String())
	}
	mustContain(t, get(t, handler, "/").Body.String(), `<div class="site-space site-space--roomy"><p>Added by an agent.</p></div>`, "published with its spacing")
	mustContain(t, get(t, handler, "/").Body.String(), "Bread worth <strong>the walk</strong>", "and the new headline")

	// Move and remove by index.
	count := len(blocks)
	rec = agentCall(t, handler, http.MethodPatch, "/agent/v1/pages/home", token, `{"remove":[`+itoa(count-1)+`]}`)
	if got := len(decodeJSON(t, rec)["blocks"].([]any)); got != count-1 {
		t.Fatalf("remove: %d blocks, want %d", got, count-1)
	}
	if rec := agentCall(t, handler, http.MethodPatch, "/agent/v1/pages/home", token, `{"remove":[99]}`); rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("bad index: %d", rec.Code)
	}

	// Create from a template, then replace wholesale.
	rec = agentCall(t, handler, http.MethodPost, "/agent/v1/pages", token, `{"title":"Questions","template":"faq","description":"What people ask"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create: %d %s", rec.Code, rec.Body.String())
	}
	created := decodeJSON(t, rec)
	if created["slug"] != "questions" || created["status"] != "draft" || len(created["blocks"].([]any)) == 0 {
		t.Fatalf("created: %v", created)
	}
	if rec := agentCall(t, handler, http.MethodPost, "/agent/v1/pages", token, `{"title":"Questions"}`); rec.Code != http.StatusConflict {
		t.Fatalf("duplicate slug: %d", rec.Code)
	}
	rec = agentCall(t, handler, http.MethodPut, "/agent/v1/pages/questions", token, `{"blocks":[{"kind":"faq","fields":{"heading":"Questions"},"items":[{"question":"Do you deliver?","answer":"Yes, in town."}]}],"publish":true}`)
	if rec.Code != http.StatusOK || decodeJSON(t, rec)["status"] != "published" {
		t.Fatalf("put+publish: %d %s", rec.Code, rec.Body.String())
	}
	mustContain(t, get(t, handler, "/questions").Body.String(), "Do you deliver?", "the page is live")
	md := agentCall(t, handler, http.MethodGet, "/agent/v1/pages/questions/markdown", token, "")
	mustContain(t, md.Header().Get("Content-Type"), "text/markdown", "markdown is markdown")
	mustContain(t, md.Body.String(), "- **Do you deliver?** — Yes, in town.", "with the questions as a list")

	// Actions.
	rec = agentCall(t, handler, http.MethodPost, "/agent/v1/pages/questions/actions", token, `{"action":"hide"}`)
	if rec.Code != http.StatusOK || decodeJSON(t, rec)["page"].(map[string]any)["hiddenFromMenu"] != true {
		t.Fatalf("hide: %d %s", rec.Code, rec.Body.String())
	}
	if rec := agentCall(t, handler, http.MethodPost, "/agent/v1/pages/questions/actions", token, `{"action":"explode"}`); rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("unknown action: %d", rec.Code)
	}
	mustContain(t, get(t, handler, "/admin/activity").Body.String(), "Agent “test”", "agent changes are audited under the key's name")
}

func TestAgentsManageTheSiteLookPostsProductsMediaAndPresets(t *testing.T) {
	host, handler := newTestHost(t)
	token := agentKeyFor(t, host, "read", "write", "publish", "settings")

	rec := agentCall(t, handler, http.MethodPatch, "/agent/v1/site", token, `{"tagline":"Bread, daily","contact":{"email":"hi@wildflower.example","phone":"555 0100"},"header":{"announceText":"Closed Monday","announceOn":true,"sticky":true,"menuButton":"Order","menuButtonTo":"/shop"},"footer":{"menu":true,"links":[["Press","/press"],["Bad","javascript:alert(1)"]]},"social":{"instagram":"https://instagram.com/wildflower","myspace":"x"}}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("patch site: %d %s", rec.Code, rec.Body.String())
	}
	site := decodeJSON(t, rec)
	if site["tagline"] != "Bread, daily" || site["contact"].(map[string]any)["email"] != "hi@wildflower.example" || site["header"].(map[string]any)["sticky"] != true {
		t.Fatalf("site: %v", site)
	}
	if links := site["footer"].(map[string]any)["links"].([]any); len(links) != 1 {
		t.Fatalf("unsafe footer links are dropped: %v", links)
	}
	if social := site["social"].(map[string]any); social["instagram"] != "https://instagram.com/wildflower" || social["myspace"] != nil {
		t.Fatalf("social: %v", social)
	}
	public := get(t, handler, "/").Body.String()
	mustContain(t, public, "Closed Monday", "the announcement is live at once")
	mustContain(t, public, "Order", "and the menu button")

	rec = agentCall(t, handler, http.MethodPut, "/agent/v1/look", token, `{"palette":"custom","ground":"#fff8f0","ink":"#222222","headings":"big"}`)
	look := decodeJSON(t, rec)
	if rec.Code != http.StatusOK || look["palette"] != "custom" || look["ground"] != "#fff8f0" || look["headings"] != "big" {
		t.Fatalf("look: %d %v", rec.Code, look)
	}
	if again := decodeJSON(t, agentCall(t, handler, http.MethodGet, "/agent/v1/look", token, "")); again["ink"] != "#222222" {
		t.Fatalf("look again: %v", again)
	}

	rec = agentCall(t, handler, http.MethodPost, "/agent/v1/posts", token, `{"title":"Our new oven","excerpt":"It's big.","tags":["news","oven"],"blocks":[{"kind":"paragraph","text":"We built it in May."}],"publish":true}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("post: %d %s", rec.Code, rec.Body.String())
	}
	created := decodeJSON(t, rec)
	if created["status"] != "published" || created["slug"] != "our-new-oven" {
		t.Fatalf("post created: %v", created)
	}
	mustContain(t, get(t, handler, "/blog/our-new-oven").Body.String(), "We built it in May.", "the post is live")
	rec = agentCall(t, handler, http.MethodPut, "/agent/v1/posts/our-new-oven", token, `{"title":"Our big new oven","tags":["news"]}`)
	if rec.Code != http.StatusOK || decodeJSON(t, rec)["title"] != "Our big new oven" {
		t.Fatalf("post put: %d %s", rec.Code, rec.Body.String())
	}
	if posts := decodeJSON(t, agentCall(t, handler, http.MethodGet, "/agent/v1/posts", token, ""))["posts"].([]any); len(posts) != 1 {
		t.Fatalf("posts: %v", posts)
	}

	rec = agentCall(t, handler, http.MethodPost, "/agent/v1/products", token, `{"name":"Weekly loaf","price":"8.50","description":"One loaf a week.","images":[{"url":"/uploads/loaf.png","alt":"A loaf"}]}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("product: %d %s", rec.Code, rec.Body.String())
	}
	product := decodeJSON(t, rec)
	if product["price"] != "8.50" || product["priceCents"].(float64) != 850 || product["slug"] != "weekly-loaf" || product["active"] != true {
		t.Fatalf("product: %v", product)
	}
	id := product["id"].(string)
	if rec := agentCall(t, handler, http.MethodPut, "/agent/v1/products/"+id, token, `{"price":"nine"}`); rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("bad price: %d", rec.Code)
	}
	rec = agentCall(t, handler, http.MethodPut, "/agent/v1/products/"+id, token, `{"price":"9","stock":3,"trackStock":true}`)
	if rec.Code != http.StatusOK || decodeJSON(t, rec)["price"] != "9.00" {
		t.Fatalf("product put: %d %s", rec.Code, rec.Body.String())
	}
	mustContain(t, get(t, handler, "/shop").Body.String(), "Weekly loaf", "the shop lists it")
	if rec := agentCall(t, handler, http.MethodDelete, "/agent/v1/products/"+id, token, ""); rec.Code != http.StatusOK {
		t.Fatalf("delete: %d", rec.Code)
	}
	if products := decodeJSON(t, agentCall(t, handler, http.MethodGet, "/agent/v1/products", token, ""))["products"].([]any); len(products) != 0 {
		t.Fatal("the product is gone")
	}

	upload, _ := json.Marshal(map[string]string{"data": "data:image/png;base64," + base64.StdEncoding.EncodeToString(tinyPNG(t))})
	rec = agentCall(t, handler, http.MethodPost, "/agent/v1/media", token, string(upload))
	if rec.Code != http.StatusCreated || !strings.HasPrefix(decodeJSON(t, rec)["url"].(string), "/uploads/") {
		t.Fatalf("upload: %d %s", rec.Code, rec.Body.String())
	}
	if rec := agentCall(t, handler, http.MethodPost, "/agent/v1/media", token, `{"data":"aGVsbG8="}`); rec.Code != http.StatusUnsupportedMediaType {
		t.Fatalf("not an image: %d %s", rec.Code, rec.Body.String())
	}
	if pictures := decodeJSON(t, agentCall(t, handler, http.MethodGet, "/agent/v1/media", token, ""))["pictures"].([]any); len(pictures) != 1 {
		t.Fatalf("pictures: %v", pictures)
	}

	rec = agentCall(t, handler, http.MethodPost, "/agent/v1/presets", token, `{"name":"Our promise","block":{"kind":"cta","fields":{"headline":"Come by"}}}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("preset: %d %s", rec.Code, rec.Body.String())
	}
	presetID := decodeJSON(t, rec)["id"].(string)
	presets := decodeJSON(t, agentCall(t, handler, http.MethodGet, "/agent/v1/presets", token, ""))["presets"].([]any)
	if len(presets) != 1 || presets[0].(map[string]any)["block"].(map[string]any)["fields"].(map[string]any)["headline"] != "Come by" {
		t.Fatalf("presets: %v", presets)
	}
	if rec := agentCall(t, handler, http.MethodDelete, "/agent/v1/presets/"+presetID, token, ""); rec.Code != http.StatusOK {
		t.Fatalf("preset delete: %d", rec.Code)
	}

	post(t, handler, "/contact/send", url.Values{"page": {"/contact"}, "name": {"Sam"}, "email": {"sam@example.com"}, "message": {"Do you cater?"}})
	messages := decodeJSON(t, agentCall(t, handler, http.MethodGet, "/agent/v1/messages?limit=5", token, ""))
	if list := messages["messages"].([]any); len(list) != 1 || list[0].(map[string]any)["body"] != "Do you cater?" {
		t.Fatalf("messages: %v", messages)
	}
	for _, path := range []string{"/agent/v1/stats?days=7", "/agent/v1/activity", "/agent/v1/forms"} {
		if rec := agentCall(t, handler, http.MethodGet, path, token, ""); rec.Code != http.StatusOK {
			t.Fatalf("%s: %d %s", path, rec.Code, rec.Body.String())
		}
	}
	forms := decodeJSON(t, agentCall(t, handler, http.MethodGet, "/agent/v1/forms", token, ""))["forms"].([]any)
	if forms[0].(map[string]any)["id"] != "contact" {
		t.Fatalf("forms: %v", forms)
	}
}

func TestAPlatformKeyCanBuildASiteFromNothing(t *testing.T) {
	host, err := Open(Options{DataPath: filepath.Join(t.TempDir(), "site.json"), SiteTitle: "New site", BaseURL: "https://new.example", AgentKeys: []string{"gsk_platform"}, NoBackups: true})
	if err != nil {
		t.Fatal(err)
	}
	handler := host.Handler()
	rec := agentCall(t, handler, http.MethodGet, "/agent/v1/site", "gsk_platform", "")
	if rec.Code != http.StatusConflict || !strings.Contains(rec.Body.String(), "setup_required") {
		t.Fatalf("before setup: %d %s", rec.Code, rec.Body.String())
	}
	if index := decodeJSON(t, agentCall(t, handler, http.MethodGet, "/agent/v1", "", "")); index["setupComplete"] != false {
		t.Fatal("the index says setup is pending")
	}
	if rec := agentCall(t, handler, http.MethodPost, "/agent/v1/setup", "gsk_other", `{"title":"x"}`); rec.Code != http.StatusUnauthorized {
		t.Fatalf("wrong key: %d", rec.Code)
	}
	rec = agentCall(t, handler, http.MethodPost, "/agent/v1/setup", "gsk_platform", `{"title":"Mill Lane Bakery","tagline":"Bread worth the walk","kind":"food","email":"ana@example.com","location":"1 Mill Lane"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("setup: %d %s", rec.Code, rec.Body.String())
	}
	site := decodeJSON(t, rec)
	if site["title"] != "Mill Lane Bakery" || site["kind"] != "food" || len(site["pages"].([]any)) < 3 {
		t.Fatalf("site after setup: %v", site)
	}
	if rec := agentCall(t, handler, http.MethodPost, "/agent/v1/setup", "gsk_platform", `{"title":"Again","kind":"food"}`); rec.Code != http.StatusConflict {
		t.Fatalf("setup twice: %d", rec.Code)
	}
	mustContain(t, get(t, handler, "/").Body.String(), "Mill Lane Bakery", "the site is live")
	if !host.SetupComplete() {
		t.Fatal("setup is complete")
	}
}

func TestPublicPagesReadAsMarkdownAndTheSiteHasAnLLMsFile(t *testing.T) {
	_, handler := newTestHost(t)
	llms := get(t, handler, "/llms.txt")
	mustContain(t, llms.Header().Get("Content-Type"), "text/markdown", "llms.txt is markdown")
	body := llms.Body.String()
	for _, want := range []string{"# Wildflower Bakery", "## Pages", "[Wildflower Bakery](https://wildflower.example/)", "https://wildflower.example/llms-full.txt", "https://wildflower.example/agent/v1/openapi.json", "https://wildflower.example/agent/mcp", "?format=md"} {
		mustContain(t, body, want, "llms.txt describes the site for agents")
	}
	full := get(t, handler, "/llms-full.txt").Body.String()
	mustContain(t, full, "<!-- https://wildflower.example/ -->", "llms-full.txt carries every page")
	mustContain(t, full, "## ", "with headings")

	md := get(t, handler, "/?format=md")
	mustContain(t, md.Header().Get("Content-Type"), "text/markdown", "a page asked for as markdown is markdown")
	mustContain(t, md.Body.String(), "# Wildflower Bakery", "with its title")
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Accept", "text/markdown")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	mustContain(t, rec.Header().Get("Content-Type"), "text/markdown", "Accept: text/markdown works too")
	req = httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,text/markdown;q=0.5")
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	mustContain(t, rec.Header().Get("Content-Type"), "text/html", "a browser that also accepts markdown gets HTML")
}
