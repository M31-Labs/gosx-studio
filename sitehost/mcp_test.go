package sitehost

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func mcpPost(t *testing.T, handler http.Handler, token, message string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/agent/mcp", strings.NewReader(message))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec
}

func TestTheSiteSpeaksMCPOverHTTP(t *testing.T) {
	host, handler := newTestHost(t)
	token := agentKeyFor(t, host, "read", "write", "publish")

	if rec := mcpPost(t, handler, "", `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`); rec.Code != http.StatusUnauthorized {
		t.Fatalf("MCP needs a key: %d", rec.Code)
	}
	rec := mcpPost(t, handler, token, `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"test","version":"1"}}}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("initialize: %d %s", rec.Code, rec.Body.String())
	}
	init := decodeJSON(t, rec)["result"].(map[string]any)
	if init["protocolVersion"] != mcpProtocolVersion || init["serverInfo"].(map[string]any)["title"] != "Wildflower Bakery" {
		t.Fatalf("initialize result: %v", init)
	}
	if rec := mcpPost(t, handler, token, `{"jsonrpc":"2.0","method":"notifications/initialized"}`); rec.Code != http.StatusAccepted {
		t.Fatalf("a notification is accepted silently: %d %s", rec.Code, rec.Body.String())
	}

	tools := decodeJSON(t, mcpPost(t, handler, token, `{"jsonrpc":"2.0","id":2,"method":"tools/list"}`))["result"].(map[string]any)["tools"].([]any)
	names := map[string]bool{}
	for _, tool := range tools {
		entry := tool.(map[string]any)
		names[entry["name"].(string)] = true
		if entry["inputSchema"].(map[string]any)["type"] != "object" {
			t.Fatalf("tool %s has no object schema", entry["name"])
		}
	}
	for _, want := range []string{"get_site", "get_schema", "edit_page", "create_page", "publish_page", "set_look", "upload_image", "list_messages"} {
		if !names[want] {
			t.Fatalf("missing tool %s in %v", want, names)
		}
	}

	rec = mcpPost(t, handler, token, `{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"get_site","arguments":{}}}`)
	result := decodeJSON(t, rec)["result"].(map[string]any)
	if result["isError"] == true || result["structuredContent"].(map[string]any)["title"] != "Wildflower Bakery" {
		t.Fatalf("get_site: %v", result)
	}
	content := result["content"].([]any)[0].(map[string]any)
	if content["type"] != "text" || !strings.Contains(content["text"].(string), `"title":"Wildflower Bakery"`) {
		t.Fatalf("content: %v", content)
	}

	rec = mcpPost(t, handler, token, `{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"edit_page","arguments":{"page":"home","append":[{"kind":"quote","text":"Best bread in town."}]}}}`)
	result = decodeJSON(t, rec)["result"].(map[string]any)
	if result["isError"] == true {
		t.Fatalf("edit_page: %v", result)
	}
	page := decodeJSON(t, mcpPost(t, handler, token, `{"jsonrpc":"2.0","id":5,"method":"tools/call","params":{"name":"get_page","arguments":{"page":"home"}}}`))["result"].(map[string]any)["structuredContent"].(map[string]any)
	blocks := page["blocks"].([]any)
	if last := blocks[len(blocks)-1].(map[string]any); last["kind"] != "quote" || last["text"] != "Best bread in town." {
		t.Fatalf("the quote is on the page: %v", last)
	}

	rec = mcpPost(t, handler, token, `{"jsonrpc":"2.0","id":6,"method":"tools/call","params":{"name":"set_look","arguments":{"palette":"ink"}}}`)
	result = decodeJSON(t, rec)["result"].(map[string]any)
	if result["isError"] != true || !strings.Contains(result["content"].([]any)[0].(map[string]any)["text"].(string), "settings") {
		t.Fatalf("a tool outside the key's scope is an error result, not a crash: %v", result)
	}
	rec = mcpPost(t, handler, token, `{"jsonrpc":"2.0","id":7,"method":"tools/call","params":{"name":"fly","arguments":{}}}`)
	if failure := decodeJSON(t, rec)["error"]; failure == nil || !strings.Contains(failure.(map[string]any)["message"].(string), "unknown tool") {
		t.Fatalf("unknown tool: %s", rec.Body.String())
	}
	if failure := decodeJSON(t, mcpPost(t, handler, token, `{"jsonrpc":"2.0","id":8,"method":"nope"}`))["error"]; failure == nil {
		t.Fatal("unknown method is a JSON-RPC error")
	}

	resources := decodeJSON(t, mcpPost(t, handler, token, `{"jsonrpc":"2.0","id":9,"method":"resources/list"}`))["result"].(map[string]any)["resources"].([]any)
	if len(resources) != 3 {
		t.Fatalf("resources: %v", resources)
	}
	read := decodeJSON(t, mcpPost(t, handler, token, `{"jsonrpc":"2.0","id":10,"method":"resources/read","params":{"uri":"site://llms.txt"}}`))["result"].(map[string]any)["contents"].([]any)[0].(map[string]any)
	if !strings.HasPrefix(read["text"].(string), "# Wildflower Bakery") {
		t.Fatalf("llms resource: %v", read)
	}
	prompt := decodeJSON(t, mcpPost(t, handler, token, `{"jsonrpc":"2.0","id":11,"method":"prompts/get","params":{"name":"build_site","arguments":{"description":"a bakery"}}}`))["result"].(map[string]any)
	if !strings.Contains(prompt["messages"].([]any)[0].(map[string]any)["content"].(map[string]any)["text"].(string), "Business: a bakery") {
		t.Fatalf("prompt: %v", prompt)
	}

	batch := mcpPost(t, handler, token, `[{"jsonrpc":"2.0","id":12,"method":"ping"},{"jsonrpc":"2.0","method":"notifications/initialized"}]`)
	var responses []map[string]any
	if err := json.Unmarshal(batch.Body.Bytes(), &responses); err != nil || len(responses) != 1 {
		t.Fatalf("batch: %d %s", batch.Code, batch.Body.String())
	}
	if rec := agentCall(t, handler, http.MethodGet, "/agent/mcp", token, ""); rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("GET is refused: %d", rec.Code)
	}
}

func TestTheStdioBridgeTalksToARemoteSite(t *testing.T) {
	host, handler := newTestHost(t)
	token := agentKeyFor(t, host, "read", "write")
	server := httptest.NewServer(handler)
	defer server.Close()

	in := strings.NewReader(strings.Join([]string{
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`,
		`{"jsonrpc":"2.0","method":"notifications/initialized"}`,
		`{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"list_pages","arguments":{}}}`,
		`{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"edit_page","arguments":{"page":"home","append":[{"kind":"paragraph","text":"From stdio."}]}}}`,
	}, "\n") + "\n")
	var out bytes.Buffer
	if err := RunMCPStdio(context.Background(), in, &out, server.URL, token, server.Client()); err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	if len(lines) != 3 {
		t.Fatalf("expected three responses, got %d:\n%s", len(lines), out.String())
	}
	var first map[string]any
	_ = json.Unmarshal([]byte(lines[0]), &first)
	if first["result"].(map[string]any)["serverInfo"].(map[string]any)["name"] != "gosx-site" {
		t.Fatalf("initialize over stdio: %s", lines[0])
	}
	if !strings.Contains(lines[1], `\"slug\":\"home\"`) {
		t.Fatalf("list_pages over stdio: %s", lines[1])
	}
	page, _, _ := host.Store().PageBySlug("home")
	last := page.Body.Blocks[len(page.Body.Blocks)-1]
	if last.Values["text"].String != "From stdio." {
		t.Fatalf("the edit reached the store: %+v", last.Values)
	}
	if err := RunMCPStdio(context.Background(), strings.NewReader(""), &out, "", "", nil); err == nil {
		t.Fatal("a site address is required")
	}
}
