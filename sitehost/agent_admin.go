package sitehost

import (
	"net/http"
	"strings"

	"m31labs.dev/gosx"
)

// agent_admin.go is the Agents page: where an admin gives an agent a
// key, sees what each key may do, and copies the settings that connect
// Claude Code, Claude Desktop, or any MCP client to the site.

const agentsAdminPath = "/admin/agents"

func (h *Host) mountAgentAdmin(mux *http.ServeMux) {
	mux.HandleFunc("GET "+agentsAdminPath, h.handleAdminAgents)
	mux.HandleFunc("GET "+agentsAdminPath+"/{$}", h.handleAdminAgents)
	mux.HandleFunc("POST "+agentsAdminPath, h.handleAdminAgentCreate)
	mux.HandleFunc("POST "+agentsAdminPath+"/{$}", h.handleAdminAgentCreate)
	mux.HandleFunc("POST "+agentsAdminPath+"/{id}/revoke", h.handleAdminAgentRevoke)
}

func (h *Host) handleAdminAgents(w http.ResponseWriter, r *http.Request) {
	h.renderAdminAgents(w, r, adminStatus{Message: r.URL.Query().Get("status")}, "", "")
}

func (h *Host) handleAdminAgentCreate(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		h.renderAdminAgents(w, r, adminStatus{Message: "We couldn't read that form. Try again.", Error: true}, "", "")
		return
	}
	name := strings.TrimSpace(r.PostFormValue("name"))
	if name == "" {
		h.renderAdminAgents(w, r, adminStatus{Message: "Give the key a name, such as the program that will use it.", Error: true}, "", "")
		return
	}
	if len(name) > 60 {
		name = name[:60]
	}
	key, token, err := h.agents.create(name, r.PostForm["scope"])
	if err != nil {
		h.renderAdminAgents(w, r, adminStatus{Message: "We couldn't save the key. Try again.", Error: true}, "", "")
		return
	}
	h.auditContent(r, "agent.key.created", "Created the agent key “"+key.Name+"” ("+strings.Join(key.Scopes, ", ")+")")
	h.renderAdminAgents(w, r, adminStatus{Message: "The key “" + key.Name + "” is ready. Copy it now: it is shown once."}, token, key.Name)
}

func (h *Host) handleAdminAgentRevoke(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	name := id
	for _, key := range h.agents.list() {
		if key.ID == id {
			name = key.Name
		}
	}
	if err := h.agents.revoke(id); err != nil {
		http.Redirect(w, r, agentsAdminPath+"?status="+queryEscape("We couldn't revoke that key. Try again."), http.StatusSeeOther)
		return
	}
	h.auditContent(r, "agent.key.revoked", "Revoked the agent key “"+name+"”")
	http.Redirect(w, r, agentsAdminPath+"?status="+queryEscape("The key “"+name+"” no longer works."), http.StatusSeeOther)
}

func (h *Host) renderAdminAgents(w http.ResponseWriter, r *http.Request, status adminStatus, freshToken, freshName string) {
	base := h.absoluteBase(r)
	keys := h.agents.list()

	var fresh gosx.Node = gosx.Fragment()
	if freshToken != "" {
		fresh = gosx.El("section", gosx.Attrs(gosx.Attr("class", "admin-panel admin-panel--key"), gosx.Attr("data-fresh-key", "true")),
			gosx.El("h2", nil, gosx.Text("Your new key for “"+freshName+"”")),
			gosx.El("p", nil, gosx.Text("Copy it now and keep it somewhere safe. We only show it this once; if you lose it, make a new one.")),
			gosx.El("input", gosx.Attrs(gosx.Attr("class", "admin-secret admin-secret--wide"), gosx.Attr("type", "text"), gosx.Attr("readonly", "readonly"), gosx.Attr("value", freshToken), gosx.Attr("aria-label", "Agent key"), gosx.Attr("onclick", "this.select()"))),
			gosx.El("h3", nil, gosx.Text("Connect Claude Code")),
			gosx.El("pre", gosx.Attrs(gosx.Attr("class", "admin-code")), gosx.Text("claude mcp add --transport http "+normalizeSlug(firstNonEmpty(h.settings().Title, "my-site"))+" "+base+agentPathPrefix+"/mcp --header \"Authorization: Bearer "+freshToken+"\"")),
			gosx.El("h3", nil, gosx.Text("Or any MCP client (.mcp.json)")),
			gosx.El("pre", gosx.Attrs(gosx.Attr("class", "admin-code")), gosx.Text(`{"mcpServers":{"`+normalizeSlug(firstNonEmpty(h.settings().Title, "my-site"))+`":{"type":"http","url":"`+base+agentPathPrefix+`/mcp","headers":{"Authorization":"Bearer `+freshToken+`"}}}}`)),
			gosx.El("h3", nil, gosx.Text("Or plain HTTP")),
			gosx.El("pre", gosx.Attrs(gosx.Attr("class", "admin-code")), gosx.Text("curl -H \"Authorization: Bearer "+freshToken+"\" "+base+agentAPIPrefix+"/site")),
		)
	}

	rows := make([]gosx.Node, 0, len(keys))
	for _, key := range keys {
		state := "Active"
		if key.Revoked {
			state = "Revoked"
		}
		lastUsed := "Never used"
		if !key.LastUsed.IsZero() {
			lastUsed = "Used " + key.LastUsed.Format("2 Jan 2006 15:04")
		}
		var action gosx.Node = gosx.Text("")
		if !key.Revoked {
			action = gosx.El("form", gosx.Attrs(gosx.Attr("method", "post"), gosx.Attr("action", agentsAdminPath+"/"+key.ID+"/revoke"), gosx.Attr("class", "admin-inline-form")),
				h.csrfField(),
				gosx.El("button", gosx.Attrs(gosx.Attr("class", "admin-row-btn"), gosx.Attr("type", "submit"), gosx.Attr("data-action", "archive")), gosx.Text("Revoke")))
		}
		rows = append(rows, gosx.El("tr", gosx.Attrs(gosx.Attr("data-key", key.ID)),
			gosx.El("td", nil, gosx.El("strong", nil, gosx.Text(key.Name)), gosx.El("br", nil), gosx.El("code", nil, gosx.Text(key.Hint+"…"))),
			gosx.El("td", nil, gosx.Text(strings.Join(key.Scopes, ", "))),
			gosx.El("td", nil, gosx.Text(state), gosx.El("br", nil), gosx.El("span", gosx.Attrs(gosx.Attr("class", "admin-muted")), gosx.Text(lastUsed))),
			gosx.El("td", nil, action),
		))
	}
	var listing gosx.Node
	if len(rows) == 0 {
		listing = gosx.El("p", gosx.Attrs(gosx.Attr("class", "admin-hint")), gosx.Text("No keys yet. Make one below to let an agent work on this site."))
	} else {
		listing = gosx.El("table", gosx.Attrs(gosx.Attr("class", "admin-table")),
			gosx.El("thead", nil, gosx.El("tr", nil, gosx.El("th", nil, gosx.Text("Key")), gosx.El("th", nil, gosx.Text("May")), gosx.El("th", nil, gosx.Text("State")), gosx.El("th", nil, gosx.Text("")))),
			gosx.El("tbody", nil, gosx.Fragment(rows...)))
	}

	scopeBoxes := make([]gosx.Node, 0, len(agentScopes))
	for _, scope := range agentScopes {
		attrs := []any{gosx.Attr("type", "checkbox"), gosx.Attr("name", "scope"), gosx.Attr("value", scope), gosx.Attr("id", "scope-"+scope)}
		if scope == scopeRead || scope == scopeWrite {
			attrs = append(attrs, gosx.Attr("checked", "checked"))
		}
		scopeBoxes = append(scopeBoxes, gosx.El("label", gosx.Attrs(gosx.Attr("class", "admin-check"), gosx.Attr("for", "scope-"+scope)),
			gosx.El("input", gosx.Attrs(attrs...)),
			gosx.El("span", nil, gosx.El("strong", nil, gosx.Text(scope)), gosx.Text(" — "+agentScopeBlurbs[scope]))))
	}
	create := gosx.El("section", gosx.Attrs(gosx.Attr("class", "admin-panel")),
		gosx.El("h2", nil, gosx.Text("Make a key")),
		gosx.El("form", gosx.Attrs(gosx.Attr("method", "post"), gosx.Attr("action", agentsAdminPath), gosx.Attr("class", "admin-form"), gosx.Attr("toolname", "create_agent_key_form"), gosx.Attr("tooldescription", "Make a new agent key with a name and the scopes it may use.")),
			h.csrfField(),
			adminTextField("name", "What will use it", "", "Claude Code on my laptop"),
			gosx.El("div", gosx.Attrs(gosx.Attr("class", "admin-field")), gosx.El("span", gosx.Attrs(gosx.Attr("class", "admin-label")), gosx.Text("What it may do")), gosx.Fragment(scopeBoxes...)),
			gosx.El("button", gosx.Attrs(gosx.Attr("class", "admin-button"), gosx.Attr("type", "submit")), gosx.Text("Create key")),
		))

	how := gosx.El("section", gosx.Attrs(gosx.Attr("class", "admin-panel")),
		gosx.El("h2", nil, gosx.Text("How agents see this site")),
		gosx.El("ul", gosx.Attrs(gosx.Attr("class", "admin-list")),
			gosx.El("li", nil, gosx.El("strong", nil, gosx.Text("Reading. ")), gosx.Text("Every public page is also Markdown ("), gosx.El("code", nil, gosx.Text("?format=md")), gosx.Text("), and "), gosx.El("a", gosx.Attrs(gosx.Attr("href", llmsPath)), gosx.Text("/llms.txt")), gosx.Text(" describes the whole site. No key needed.")),
			gosx.El("li", nil, gosx.El("strong", nil, gosx.Text("Editing. ")), gosx.Text("With a key, an agent uses the same operations as the editor: "), gosx.El("a", gosx.Attrs(gosx.Attr("href", agentAPIPrefix+"/openapi.json")), gosx.Text("the API")), gosx.Text(", "), gosx.El("a", gosx.Attrs(gosx.Attr("href", agentAPIPrefix+"/schema")), gosx.Text("the block schema")), gosx.Text(", and MCP at "), gosx.El("code", nil, gosx.Text(base+agentPathPrefix+"/mcp")), gosx.Text(".")),
			gosx.El("li", nil, gosx.El("strong", nil, gosx.Text("Safety. ")), gosx.Text("Every change is a draft until published, every action is in the audit log under the key's name, and revoking a key stops it at once.")),
			gosx.El("li", nil, gosx.El("strong", nil, gosx.Text("Your browser. ")), gosx.Text("If your browser has a built-in assistant that speaks WebMCP, it can work this admin and the editor for you: every admin page announces its tools to the browser, and they act as you, with your permissions. No key needed.")),
		))
	tools := gosx.El("section", gosx.Attrs(gosx.Attr("class", "admin-panel"), gosx.Attr("data-webmcp", "true")),
		gosx.El("h2", nil, gosx.Text("What your browser's assistant can do here")),
		gosx.El("p", gosx.Attrs(gosx.Attr("class", "admin-hint")), gosx.Text("These tools are announced to the browser on the editor and admin pages (WebMCP). A browser without a built-in assistant ignores them.")),
		gosx.El("ul", gosx.Attrs(gosx.Attr("class", "admin-list"), gosx.Attr("data-webmcp-tools", "true")),
			gosx.El("li", gosx.Attrs(gosx.Attr("class", "admin-muted")), gosx.Text("The list appears once the page's script runs."))))

	body := h.renderAdminShell("agents", "Agents",
		"Let agents read and build this site. Keys are like passwords for programs: each has a name, a set of things it may do, and can be revoked.",
		status, fresh, listing, create, how, tools)
	h.writeDocument(w, http.StatusOK, h.adminMeta("Agents"), body)
}
