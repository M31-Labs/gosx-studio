package sitehost

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// mcp.go speaks the Model Context Protocol so an assistant such as Claude
// Code can drive a site with tools. Each tool is a thin wrapper over one
// agent API call; the site serves MCP itself over Streamable HTTP at
// /agent/mcp, and `gosx-site mcp` bridges stdio to that same API for
// clients that only speak stdio.

const mcpProtocolVersion = "2025-06-18"

// mcpDo performs one agent API call: method, path (with query), JSON body.
type mcpDo func(ctx context.Context, method, path string, body []byte) (int, []byte, error)

type mcpTool struct {
	Name        string
	Description string
	Scope       string
	Schema      map[string]any
	// Call turns the tool's arguments into an API call.
	Call func(args map[string]any) (method, path string, body any)
}

type mcpRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type mcpResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Result  any             `json:"result,omitempty"`
	Error   *mcpError       `json:"error,omitempty"`
}

type mcpError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func objectSchema(required []string, props map[string]any) map[string]any {
	schema := map[string]any{"type": "object", "properties": props}
	if len(required) > 0 {
		schema["required"] = required
	}
	return schema
}

func prop(kind, description string) map[string]any {
	return map[string]any{"type": kind, "description": description}
}

func argString(args map[string]any, key string) string {
	switch value := args[key].(type) {
	case string:
		return strings.TrimSpace(value)
	case float64:
		return strconv.FormatFloat(value, 'f', -1, 64)
	case bool:
		if value {
			return "true"
		}
		return "false"
	}
	return ""
}

func argBool(args map[string]any, key string) bool {
	switch value := args[key].(type) {
	case bool:
		return value
	case string:
		return value == "true" || value == "yes"
	}
	return false
}

func withoutKeys(args map[string]any, keys ...string) map[string]any {
	out := map[string]any{}
	for key, value := range args {
		out[key] = value
	}
	for _, key := range keys {
		delete(out, key)
	}
	return out
}

var blockSchema = map[string]any{"type": "object", "description": "One block as get_schema describes: {kind, ...}. Simple blocks use text/url/alt/style…; ready-made sections use fields, items, variant.", "required": []string{"kind"}, "properties": map[string]any{"kind": prop("string", "Block kind from get_schema")}, "additionalProperties": true}
var blocksSchema = map[string]any{"type": "array", "items": blockSchema}

// mcpTools is the tool list, in the order an agent reads it.
func mcpTools() []mcpTool {
	pageRef := prop("string", "The page's id or address (slug), e.g. \"home\" or \"page_3\"")
	return []mcpTool{
		{Name: "get_site", Description: "The site at a glance: name, tagline, kind, contact, header, footer, Look, every page with its status, and the menu. Call this first.", Scope: scopeRead, Schema: objectSchema(nil, map[string]any{}),
			Call: func(args map[string]any) (string, string, any) { return http.MethodGet, agentAPIPrefix + "/site", nil }},
		{Name: "get_schema", Description: "Every block kind with its fields, repeated items, variants, and fixed choices; page templates; site kinds; Look options. Read it before composing blocks.", Scope: "", Schema: objectSchema(nil, map[string]any{}),
			Call: func(args map[string]any) (string, string, any) {
				return http.MethodGet, agentAPIPrefix + "/schema", nil
			}},
		{Name: "update_site", Description: "Change the site's title, tagline, description, kind, contact details, header (announcement, sticky, menu button), footer (menu, links), social links, or custom CSS. Send only what changes.", Scope: scopeSettings,
			Schema: objectSchema(nil, map[string]any{"title": prop("string", ""), "tagline": prop("string", ""), "description": prop("string", "One line for search results"), "kind": prop("string", "A site kind key"), "contact": map[string]any{"type": "object", "properties": map[string]any{"email": prop("string", ""), "phone": prop("string", ""), "location": prop("string", "")}}, "header": map[string]any{"type": "object", "properties": map[string]any{"announceText": prop("string", ""), "announceLink": prop("string", ""), "announceOn": prop("boolean", ""), "sticky": prop("boolean", ""), "menuButton": prop("string", "Label of the header button"), "menuButtonTo": prop("string", "Where it goes")}}, "footer": map[string]any{"type": "object", "properties": map[string]any{"menu": prop("boolean", ""), "links": map[string]any{"type": "array", "items": map[string]any{"type": "array", "items": prop("string", ""), "description": "[label, href]"}}}}, "social": map[string]any{"type": "object", "additionalProperties": prop("string", "URL"), "description": "instagram, facebook, tiktok, youtube, x, linkedin"}, "customCss": prop("string", "")}),
			Call: func(args map[string]any) (string, string, any) {
				return http.MethodPatch, agentAPIPrefix + "/site", args
			}},
		{Name: "list_pages", Description: "Every page with id, address, status (published/draft), whether it is live, hidden from the menu, and its block count.", Scope: scopeRead, Schema: objectSchema(nil, map[string]any{}),
			Call: func(args map[string]any) (string, string, any) { return http.MethodGet, agentAPIPrefix + "/pages", nil }},
		{Name: "get_page", Description: "A page with all its blocks in editable form, plus readiness checks.", Scope: scopeRead, Schema: objectSchema([]string{"page"}, map[string]any{"page": pageRef}),
			Call: func(args map[string]any) (string, string, any) {
				return http.MethodGet, agentAPIPrefix + "/pages/" + url.PathEscape(argString(args, "page")), nil
			}},
		{Name: "page_markdown", Description: "A page rendered as Markdown, to read what a visitor would.", Scope: scopeRead, Schema: objectSchema([]string{"page"}, map[string]any{"page": pageRef}),
			Call: func(args map[string]any) (string, string, any) {
				return http.MethodGet, agentAPIPrefix + "/pages/" + url.PathEscape(argString(args, "page")) + "/markdown", nil
			}},
		{Name: "create_page", Description: "Create a page. Give blocks to compose it, or a template key from get_schema for a ready layout. It stays a draft unless publish is true.", Scope: scopeWrite,
			Schema: objectSchema([]string{"title"}, map[string]any{"title": prop("string", ""), "slug": prop("string", "Address like \"about-us\" (default: from the title)"), "description": prop("string", "One line for search results"), "template": prop("string", "Page template key"), "navParent": prop("string", "Id of the menu page this sits under"), "blocks": blocksSchema, "publish": prop("boolean", "Publish at once")}),
			Call: func(args map[string]any) (string, string, any) {
				return http.MethodPost, agentAPIPrefix + "/pages", args
			}},
		{Name: "replace_page", Description: "Replace a page's blocks entirely (and optionally its title, address, description). Prefer edit_page for small changes.", Scope: scopeWrite,
			Schema: objectSchema([]string{"page", "blocks"}, map[string]any{"page": pageRef, "blocks": blocksSchema, "title": prop("string", ""), "slug": prop("string", ""), "description": prop("string", ""), "publish": prop("boolean", "")}),
			Call: func(args map[string]any) (string, string, any) {
				return http.MethodPut, agentAPIPrefix + "/pages/" + url.PathEscape(argString(args, "page")), withoutKeys(args, "page")
			}},
		{Name: "edit_page", Description: "Change part of a page: title, address, description, menu parent, or blocks by index (replace, remove, insert, append, move). Indexes refer to the page before the call; read it with get_page first.", Scope: scopeWrite,
			Schema: objectSchema([]string{"page"}, map[string]any{"page": pageRef, "title": prop("string", ""), "slug": prop("string", ""), "description": prop("string", ""), "navParent": prop("string", ""),
				"replace": map[string]any{"type": "array", "items": objectSchema([]string{"index", "block"}, map[string]any{"index": prop("integer", ""), "block": blockSchema})},
				"remove":  map[string]any{"type": "array", "items": prop("integer", "")},
				"insert":  map[string]any{"type": "array", "items": objectSchema([]string{"index", "block"}, map[string]any{"index": prop("integer", "Insert before this index"), "block": blockSchema})},
				"append":  blocksSchema, "move": objectSchema([]string{"from", "to"}, map[string]any{"from": prop("integer", ""), "to": prop("integer", "")}), "publish": prop("boolean", "")}),
			Call: func(args map[string]any) (string, string, any) {
				return http.MethodPatch, agentAPIPrefix + "/pages/" + url.PathEscape(argString(args, "page")), withoutKeys(args, "page")
			}},
		{Name: "publish_page", Description: "Make the page's saved draft live.", Scope: scopePublish, Schema: objectSchema([]string{"page"}, map[string]any{"page": pageRef}),
			Call: func(args map[string]any) (string, string, any) {
				return http.MethodPost, agentAPIPrefix + "/pages/" + url.PathEscape(argString(args, "page")) + "/publish", nil
			}},
		{Name: "page_action", Description: "offline (hide from visitors), online, archive, restore, hide (from the menu), show, up, down (menu order).", Scope: scopeWrite, Schema: objectSchema([]string{"page", "action"}, map[string]any{"page": pageRef, "action": map[string]any{"type": "string", "enum": []string{"offline", "online", "archive", "restore", "hide", "show", "up", "down"}}}),
			Call: func(args map[string]any) (string, string, any) {
				return http.MethodPost, agentAPIPrefix + "/pages/" + url.PathEscape(argString(args, "page")) + "/actions", map[string]any{"action": argString(args, "action")}
			}},
		{Name: "list_posts", Description: "Blog posts with status.", Scope: scopeRead, Schema: objectSchema(nil, map[string]any{}),
			Call: func(args map[string]any) (string, string, any) { return http.MethodGet, agentAPIPrefix + "/posts", nil }},
		{Name: "get_post", Description: "A post with its blocks.", Scope: scopeRead, Schema: objectSchema([]string{"post"}, map[string]any{"post": prop("string", "Post id or address")}),
			Call: func(args map[string]any) (string, string, any) {
				return http.MethodGet, agentAPIPrefix + "/posts/" + url.PathEscape(argString(args, "post")), nil
			}},
		{Name: "create_post", Description: "Write a blog post (a draft unless publish is true).", Scope: scopeWrite, Schema: objectSchema([]string{"title"}, map[string]any{"title": prop("string", ""), "slug": prop("string", ""), "excerpt": prop("string", ""), "author": prop("string", ""), "tags": map[string]any{"type": "array", "items": prop("string", "")}, "blocks": blocksSchema, "publish": prop("boolean", "")}),
			Call: func(args map[string]any) (string, string, any) {
				return http.MethodPost, agentAPIPrefix + "/posts", args
			}},
		{Name: "update_post", Description: "Change a post's title, address, excerpt, author, tags, or blocks.", Scope: scopeWrite, Schema: objectSchema([]string{"post"}, map[string]any{"post": prop("string", "Post id or address"), "title": prop("string", ""), "slug": prop("string", ""), "excerpt": prop("string", ""), "author": prop("string", ""), "tags": map[string]any{"type": "array", "items": prop("string", "")}, "blocks": blocksSchema, "publish": prop("boolean", "")}),
			Call: func(args map[string]any) (string, string, any) {
				return http.MethodPut, agentAPIPrefix + "/posts/" + url.PathEscape(argString(args, "post")), withoutKeys(args, "post")
			}},
		{Name: "publish_post", Description: "Make a post live.", Scope: scopePublish, Schema: objectSchema([]string{"post"}, map[string]any{"post": prop("string", "")}),
			Call: func(args map[string]any) (string, string, any) {
				return http.MethodPost, agentAPIPrefix + "/posts/" + url.PathEscape(argString(args, "post")) + "/publish", nil
			}},
		{Name: "list_products", Description: "Products in the shop with prices.", Scope: scopeRead, Schema: objectSchema(nil, map[string]any{}),
			Call: func(args map[string]any) (string, string, any) {
				return http.MethodGet, agentAPIPrefix + "/products", nil
			}},
		{Name: "save_product", Description: "Create a product (no id) or change one (with id): name, description, price as a decimal string, images, stock, active.", Scope: scopeWrite,
			Schema: objectSchema([]string{"name"}, map[string]any{"id": prop("string", "Leave empty to create"), "name": prop("string", ""), "slug": prop("string", ""), "description": prop("string", ""), "price": prop("string", "e.g. \"12.50\""), "compare": prop("string", "A crossed-out was-price"), "images": map[string]any{"type": "array", "items": objectSchema([]string{"url"}, map[string]any{"url": prop("string", ""), "alt": prop("string", "")})}, "stock": prop("integer", ""), "trackStock": prop("boolean", ""), "ships": prop("boolean", ""), "active": prop("boolean", ""), "kind": map[string]any{"type": "string", "enum": []string{"physical", "digital", "subscription", "booking"}}}),
			Call: func(args map[string]any) (string, string, any) {
				if id := argString(args, "id"); id != "" {
					return http.MethodPut, agentAPIPrefix + "/products/" + url.PathEscape(id), withoutKeys(args, "id")
				}
				return http.MethodPost, agentAPIPrefix + "/products", withoutKeys(args, "id")
			}},
		{Name: "delete_product", Description: "Remove a product.", Scope: scopeWrite, Schema: objectSchema([]string{"id"}, map[string]any{"id": prop("string", "")}),
			Call: func(args map[string]any) (string, string, any) {
				return http.MethodDelete, agentAPIPrefix + "/products/" + url.PathEscape(argString(args, "id")), nil
			}},
		{Name: "list_media", Description: "Uploaded pictures with their URLs and sizes, to reuse in blocks.", Scope: scopeRead, Schema: objectSchema(nil, map[string]any{}),
			Call: func(args map[string]any) (string, string, any) { return http.MethodGet, agentAPIPrefix + "/media", nil }},
		{Name: "upload_image", Description: "Add a picture from base64 data or by fetching a URL; returns the URL to use in blocks.", Scope: scopeWrite, Schema: objectSchema(nil, map[string]any{"data": prop("string", "Base64 image bytes"), "url": prop("string", "Or an http(s) address to fetch")}),
			Call: func(args map[string]any) (string, string, any) {
				return http.MethodPost, agentAPIPrefix + "/media", args
			}},
		{Name: "get_look", Description: "The site's Look: palette, fonts, accent, buttons, spacing, headings, width.", Scope: scopeRead, Schema: objectSchema(nil, map[string]any{}),
			Call: func(args map[string]any) (string, string, any) { return http.MethodGet, agentAPIPrefix + "/look", nil }},
		{Name: "set_look", Description: "Change the Look. Keys from get_schema.look; send only what changes. Custom palette: palette \"custom\" with ground and ink hex colours. Custom fonts: fonts \"custom\" with fontHead and fontBody Google Fonts names.", Scope: scopeSettings,
			Schema: objectSchema(nil, map[string]any{"palette": prop("string", ""), "fonts": prop("string", ""), "accent": prop("string", "Hex colour"), "buttons": prop("string", ""), "spacing": prop("string", ""), "headings": prop("string", ""), "width": prop("string", ""), "ground": prop("string", ""), "ink": prop("string", ""), "fontHead": prop("string", ""), "fontBody": prop("string", "")}),
			Call:   func(args map[string]any) (string, string, any) { return http.MethodPut, agentAPIPrefix + "/look", args }},
		{Name: "list_presets", Description: "Sections the owner saved to reuse; each comes with its block.", Scope: scopeRead, Schema: objectSchema(nil, map[string]any{}),
			Call: func(args map[string]any) (string, string, any) {
				return http.MethodGet, agentAPIPrefix + "/presets", nil
			}},
		{Name: "save_preset", Description: "Keep a block as a named preset for the sidebar.", Scope: scopeWrite, Schema: objectSchema([]string{"name", "block"}, map[string]any{"name": prop("string", ""), "block": blockSchema}),
			Call: func(args map[string]any) (string, string, any) {
				return http.MethodPost, agentAPIPrefix + "/presets", args
			}},
		{Name: "list_forms", Description: "Forms a page can hold (ids for the form block).", Scope: scopeRead, Schema: objectSchema(nil, map[string]any{}),
			Call: func(args map[string]any) (string, string, any) { return http.MethodGet, agentAPIPrefix + "/forms", nil }},
		{Name: "list_messages", Description: "Messages visitors sent through forms, newest first.", Scope: scopeRead, Schema: objectSchema(nil, map[string]any{"limit": prop("integer", "")}),
			Call: func(args map[string]any) (string, string, any) {
				path := agentAPIPrefix + "/messages"
				if limit := argString(args, "limit"); limit != "" {
					path += "?limit=" + url.QueryEscape(limit)
				}
				return http.MethodGet, path, nil
			}},
		{Name: "get_stats", Description: "Visitor counts by day, page, source, and device.", Scope: scopeRead, Schema: objectSchema(nil, map[string]any{"days": prop("integer", "1–365, default 30")}),
			Call: func(args map[string]any) (string, string, any) {
				path := agentAPIPrefix + "/stats"
				if days := argString(args, "days"); days != "" {
					path += "?days=" + url.QueryEscape(days)
				}
				return http.MethodGet, path, nil
			}},
		{Name: "get_activity", Description: "Recent changes made by people and agents.", Scope: scopeRead, Schema: objectSchema(nil, map[string]any{}),
			Call: func(args map[string]any) (string, string, any) {
				return http.MethodGet, agentAPIPrefix + "/activity", nil
			}},
		{Name: "complete_setup", Description: "Build a brand-new site that has not been set up: title, kind, and contact details. Only works before the first setup.", Scope: scopeSettings, Schema: objectSchema([]string{"title", "kind"}, map[string]any{"title": prop("string", ""), "tagline": prop("string", ""), "kind": prop("string", "Site kind key from get_schema.siteKinds"), "template": prop("string", "A starter key from get_schema.starters"), "email": prop("string", ""), "phone": prop("string", ""), "location": prop("string", "")}),
			Call: func(args map[string]any) (string, string, any) {
				return http.MethodPost, agentAPIPrefix + "/setup", args
			}},
	}
}

func mcpToolByName(name string) (mcpTool, bool) {
	for _, tool := range mcpTools() {
		if tool.Name == name {
			return tool, true
		}
	}
	return mcpTool{}, false
}

// mcpServer answers JSON-RPC messages using do for every tool call.
type mcpServer struct {
	do       mcpDo
	siteName string
}

func (s mcpServer) handle(ctx context.Context, raw []byte) (*mcpResponse, bool) {
	var req mcpRequest
	if err := json.Unmarshal(raw, &req); err != nil {
		return &mcpResponse{JSONRPC: "2.0", Error: &mcpError{Code: -32700, Message: "parse error: " + err.Error()}}, true
	}
	notification := len(req.ID) == 0 || string(req.ID) == "null"
	reply := func(result any) (*mcpResponse, bool) {
		if notification {
			return nil, false
		}
		return &mcpResponse{JSONRPC: "2.0", ID: req.ID, Result: result}, true
	}
	fail := func(code int, message string) (*mcpResponse, bool) {
		if notification {
			return nil, false
		}
		return &mcpResponse{JSONRPC: "2.0", ID: req.ID, Error: &mcpError{Code: code, Message: message}}, true
	}
	switch req.Method {
	case "initialize":
		return reply(map[string]any{
			"protocolVersion": mcpProtocolVersion,
			"capabilities":    map[string]any{"tools": map[string]any{"listChanged": false}, "resources": map[string]any{"listChanged": false}},
			"serverInfo":      map[string]any{"name": "gosx-site", "title": firstNonEmpty(s.siteName, "GoSX site"), "version": agentAPIVersion},
			"instructions":    "This server edits one website. Call get_site first, then get_schema before composing blocks. Writes are drafts until publish_page is called. Keep copy short and in the owner's voice.",
		})
	case "notifications/initialized", "notifications/cancelled", "notifications/progress":
		return nil, false
	case "ping":
		return reply(map[string]any{})
	case "tools/list":
		tools := []map[string]any{}
		for _, tool := range mcpTools() {
			tools = append(tools, map[string]any{"name": tool.Name, "description": tool.Description, "inputSchema": tool.Schema})
		}
		return reply(map[string]any{"tools": tools})
	case "tools/call":
		var params struct {
			Name      string         `json:"name"`
			Arguments map[string]any `json:"arguments"`
		}
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return fail(-32602, "invalid params")
		}
		tool, ok := mcpToolByName(params.Name)
		if !ok {
			return fail(-32602, "unknown tool "+params.Name)
		}
		if params.Arguments == nil {
			params.Arguments = map[string]any{}
		}
		method, path, body := tool.Call(params.Arguments)
		var raw []byte
		if body != nil {
			raw, _ = json.Marshal(body)
		}
		status, out, err := s.do(ctx, method, path, raw)
		if err != nil {
			return reply(map[string]any{"content": []map[string]any{{"type": "text", "text": "The site could not be reached: " + err.Error()}}, "isError": true})
		}
		text := strings.TrimSpace(string(out))
		result := map[string]any{"content": []map[string]any{{"type": "text", "text": text}}}
		if status >= 400 {
			result["isError"] = true
		} else if json.Valid(out) && strings.HasPrefix(text, "{") {
			var structured map[string]any
			if json.Unmarshal(out, &structured) == nil {
				result["structuredContent"] = structured
			}
		}
		return reply(result)
	case "resources/list":
		return reply(map[string]any{"resources": []map[string]any{
			{"uri": "site://llms.txt", "name": "llms.txt", "title": "The site for agents", "mimeType": "text/markdown", "description": "What the site is, its pages, and how to work with it."},
			{"uri": "site://schema", "name": "schema", "title": "Block schema", "mimeType": "application/json", "description": "Every block kind, its fields, and its choices."},
			{"uri": "site://site", "name": "site", "title": "The site", "mimeType": "application/json", "description": "Name, contact, header, footer, Look, pages, and menu."},
		}})
	case "resources/read":
		var params struct {
			URI string `json:"uri"`
		}
		_ = json.Unmarshal(req.Params, &params)
		path, mime := "", "application/json"
		switch params.URI {
		case "site://llms.txt":
			path, mime = llmsPath, "text/markdown"
		case "site://schema":
			path = agentAPIPrefix + "/schema"
		case "site://site":
			path = agentAPIPrefix + "/site"
		default:
			return fail(-32002, "unknown resource "+params.URI)
		}
		status, out, err := s.do(ctx, http.MethodGet, path, nil)
		if err != nil || status >= 400 {
			return fail(-32002, "the resource could not be read")
		}
		return reply(map[string]any{"contents": []map[string]any{{"uri": params.URI, "mimeType": mime, "text": string(out)}}})
	case "prompts/list":
		return reply(map[string]any{"prompts": []map[string]any{
			{"name": "build_site", "title": "Build the site from a description", "description": "Turn a short description of a business into pages, sections, and copy.", "arguments": []map[string]any{{"name": "description", "description": "What the business is, who it serves, what it wants visitors to do.", "required": true}}},
			{"name": "improve_page", "title": "Improve a page", "description": "Tighten copy, add missing sections, and fix readiness checks on one page.", "arguments": []map[string]any{{"name": "page", "description": "Page id or address", "required": true}}},
		}})
	case "prompts/get":
		var params struct {
			Name      string            `json:"name"`
			Arguments map[string]string `json:"arguments"`
		}
		_ = json.Unmarshal(req.Params, &params)
		switch params.Name {
		case "build_site":
			return reply(map[string]any{"messages": []map[string]any{{"role": "user", "content": map[string]any{"type": "text", "text": "Build this website. Business: " + params.Arguments["description"] + "\n\nSteps: call get_site and get_schema; set the site's tagline, description, and contact with update_site; give the home page a hero, three feature cards, a call to action, and the sections a visitor to this kind of business expects; create the other pages (about, services or menu, contact) from templates and fill their copy; keep every sentence short and concrete; do not publish unless asked. Finish with a two-line summary of what you made."}}}})
		case "improve_page":
			return reply(map[string]any{"messages": []map[string]any{{"role": "user", "content": map[string]any{"type": "text", "text": "Improve the page \"" + params.Arguments["page"] + "\". Read it with get_page and page_markdown, look at its checks, then use edit_page to tighten copy, add what is missing, and fix every check. Do not publish. Say what you changed in three lines."}}}})
		}
		return fail(-32602, "unknown prompt "+params.Name)
	default:
		return fail(-32601, "method not found: "+req.Method)
	}
}

// handleMCP is the Streamable HTTP transport: one JSON-RPC message (or a
// batch) per POST, answered as JSON.
func (h *Host) handleMCP(w http.ResponseWriter, r *http.Request) {
	identity, _ := h.agentCaller(r)
	raw, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 8<<20))
	if err != nil {
		agentError(w, http.StatusBadRequest, "bad_request", "The message could not be read.")
		return
	}
	server := mcpServer{siteName: h.settings().Title, do: func(ctx context.Context, method, path string, body []byte) (int, []byte, error) {
		status, _, out := h.agentDispatch(ctx, identity, method, path, body, r)
		return status, out, nil
	}}
	trimmed := strings.TrimSpace(string(raw))
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("MCP-Protocol-Version", mcpProtocolVersion)
	if strings.HasPrefix(trimmed, "[") {
		var batch []json.RawMessage
		if err := json.Unmarshal(raw, &batch); err != nil {
			agentError(w, http.StatusBadRequest, "bad_request", "The batch could not be read.")
			return
		}
		responses := []*mcpResponse{}
		for _, message := range batch {
			if response, ok := server.handle(r.Context(), message); ok {
				responses = append(responses, response)
			}
		}
		if len(responses) == 0 {
			w.WriteHeader(http.StatusAccepted)
			return
		}
		writeJSON(w, http.StatusOK, responses)
		return
	}
	response, ok := server.handle(r.Context(), raw)
	if !ok {
		w.WriteHeader(http.StatusAccepted)
		return
	}
	writeJSON(w, http.StatusOK, response)
}

// RunMCPStdio bridges a stdio MCP client to a site's agent API: each line
// in is one JSON-RPC message, each line out one response. The command
// `gosx-site mcp --site URL --key KEY` uses it.
func RunMCPStdio(ctx context.Context, in io.Reader, out io.Writer, siteURL, key string, client *http.Client) error {
	siteURL = strings.TrimRight(strings.TrimSpace(siteURL), "/")
	if siteURL == "" {
		return errors.New("a site address is required")
	}
	if client == nil {
		client = http.DefaultClient
	}
	do := func(ctx context.Context, method, path string, body []byte) (int, []byte, error) {
		var reader io.Reader
		if len(body) > 0 {
			reader = strings.NewReader(string(body))
		}
		req, err := http.NewRequestWithContext(ctx, method, siteURL+path, reader)
		if err != nil {
			return 0, nil, err
		}
		if len(body) > 0 {
			req.Header.Set("Content-Type", "application/json")
		}
		if key != "" {
			req.Header.Set("Authorization", "Bearer "+key)
		}
		req.Header.Set("Accept", "application/json, text/markdown")
		resp, err := client.Do(req)
		if err != nil {
			return 0, nil, err
		}
		defer resp.Body.Close()
		data, err := io.ReadAll(io.LimitReader(resp.Body, 16<<20))
		return resp.StatusCode, data, err
	}
	server := mcpServer{do: do}
	scanner := bufio.NewScanner(in)
	scanner.Buffer(make([]byte, 0, 1<<20), 32<<20)
	writer := bufio.NewWriter(out)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		response, ok := server.handle(ctx, []byte(line))
		if !ok {
			continue
		}
		encoded, _ := json.Marshal(response)
		if _, err := writer.Write(append(encoded, '\n')); err != nil {
			return err
		}
		if err := writer.Flush(); err != nil {
			return err
		}
	}
	return scanner.Err()
}
