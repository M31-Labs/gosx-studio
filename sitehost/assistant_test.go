package sitehost

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"
)

// scriptedAssistant plays back canned turns, recording what it was asked.
type scriptedAssistant struct {
	turns []AssistantResponse
	calls []AssistantRequest
}

func (s *scriptedAssistant) Name() string { return "Scripted" }

func (s *scriptedAssistant) Complete(ctx context.Context, req AssistantRequest) (AssistantResponse, error) {
	s.calls = append(s.calls, req)
	if len(s.turns) == 0 {
		return AssistantResponse{StopReason: "end_turn", Content: []AssistantPart{{Type: "text", Text: "Done."}}}, nil
	}
	turn := s.turns[0]
	s.turns = s.turns[1:]
	return turn, nil
}

func newAssistantHost(t *testing.T, provider AssistantProvider) (*Host, http.Handler) {
	t.Helper()
	host, err := Open(Options{
		DataPath: filepath.Join(t.TempDir(), "site.json"), SiteTitle: "Wildflower Bakery", BaseURL: "https://wildflower.example",
		SiteKind: "food", Seed: true, NoBackups: true, Assistant: provider,
	})
	if err != nil {
		t.Fatal(err)
	}
	return host, host.Handler()
}

func TestOwnersAskTheirSiteForChangesInPlainWords(t *testing.T) {
	scripted := &scriptedAssistant{turns: []AssistantResponse{
		{StopReason: "tool_use", Content: []AssistantPart{
			{Type: "text", Text: "Adding a pricing section."},
			{Type: "tool_use", ID: "t1", Name: "edit_page", Input: map[string]any{"page": "home", "append": []any{map[string]any{"kind": "pricing", "fields": map[string]any{"heading": "Plans"}, "items": []any{map[string]any{"name": "Weekly loaf", "price": "$8", "period": "a week"}}}}}},
		}},
		{StopReason: "tool_use", Content: []AssistantPart{
			{Type: "tool_use", ID: "t2", Name: "publish_page", Input: map[string]any{"page": "home"}},
		}},
		{StopReason: "end_turn", Content: []AssistantPart{{Type: "text", Text: "I added a Plans section with one plan and published the page."}}},
	}}
	host, handler := newAssistantHost(t, scripted)
	id := firstPageID(t, host, "home")

	editor := get(t, handler, "/admin/edit/"+id).Body.String()
	mustContain(t, editor, `data-assistant="page" data-assistant-id="`+id+`"`, "the editor has the Ask box for this page")
	mustContain(t, editor, "Add a pricing section with three plans", "with example asks")
	mustContain(t, get(t, handler, "/admin/api/assistant").Body.String(), `"available":true`, "the script can see the assistant is on")

	rec := postJSON(t, handler, "/admin/api/assistant", `{"kind":"page","id":"`+id+`","prompt":"Add a pricing section with our weekly loaf"}`)
	body := rec.Body.String()
	mustContain(t, body, `"ok":true`, "the ask succeeds")
	mustContain(t, body, `"reply":"I added a Plans section with one plan and published the page."`, "with the assistant's own words")
	mustContain(t, body, `"Changed the page (home)"`, "and plain-words steps")
	mustContain(t, body, `"Published the page (home)"`, "for every tool call")
	mustContain(t, body, `"changed":true`, "and says the page changed")

	page, _, _ := host.Store().PageByID(id)
	last := page.Body.Blocks[len(page.Body.Blocks)-1]
	if last.Key != "pricing" || last.Values["items"].List[0].Object["name"].String != "Weekly loaf" {
		t.Fatalf("the pricing section is on the page: %+v", last)
	}
	mustContain(t, get(t, handler, "/").Body.String(), "Weekly loaf", "and published")

	if len(scripted.calls) != 3 {
		t.Fatalf("turns: %d", len(scripted.calls))
	}
	system := scripted.calls[0].System
	mustContain(t, system, "Wildflower Bakery", "the model knows the site")
	mustContain(t, system, `"id":"`+id+`"`, "and the page at hand")
	mustContain(t, system, "Never publish unless the owner asked", "and the rules")
	toolNames := []string{}
	for _, tool := range scripted.calls[0].Tools {
		toolNames = append(toolNames, tool.Name)
	}
	if !strings.Contains(strings.Join(toolNames, ","), "edit_page") || strings.Contains(strings.Join(toolNames, ","), "complete_setup") {
		t.Fatalf("tools: %v", toolNames)
	}
	// The tool result went back to the model.
	if result := scripted.calls[1].Messages[len(scripted.calls[1].Messages)-1].Content[0]; result.Type != "tool_result" || result.ToolUseID != "t1" || !strings.Contains(result.Result, `"kind":"pricing"`) {
		t.Fatalf("tool result: %+v", result)
	}
	mustContain(t, get(t, handler, "/admin/activity").Body.String(), "Asked the assistant", "the ask is audited")
}

func TestTheDashboardBoxBuildsTheSiteWithoutAnyScript(t *testing.T) {
	scripted := &scriptedAssistant{turns: []AssistantResponse{
		{StopReason: "tool_use", Content: []AssistantPart{{Type: "tool_use", ID: "t1", Name: "create_page", Input: map[string]any{"title": "Our story", "blocks": []any{map[string]any{"kind": "paragraph", "text": "We started in 2014."}}}}}},
		{StopReason: "end_turn", Content: []AssistantPart{{Type: "text", Text: "I wrote an Our story page as a draft."}}},
	}}
	host, handler := newAssistantHost(t, scripted)
	dashboard := get(t, handler, "/admin").Body.String()
	mustContain(t, dashboard, "Tell your site what you want", "the dashboard has the box")
	mustContain(t, dashboard, `action="/admin/assistant"`, "as a plain form")

	rec := post(t, handler, "/admin/assistant", url.Values{"prompt": {"We're a family bakery; write our story"}})
	body := rec.Body.String()
	mustContain(t, body, "I wrote an Our story page as a draft.", "the reply is on the dashboard")
	mustContain(t, body, "Created a page (Our story)", "with the steps")
	if _, ok, _ := host.Store().PageBySlug("our-story"); !ok {
		t.Fatal("the page exists")
	}
}

func TestWithoutAnAssistantTheBoxPointsToAgents(t *testing.T) {
	host, handler := newTestHost(t)
	id := firstPageID(t, host, "home")
	editor := get(t, handler, "/admin/edit/"+id).Body.String()
	mustContain(t, editor, `data-assistant="off"`, "the box is off")
	mustContain(t, editor, `href="/admin/agents">Connect an assistant</a>`, "and says how to turn it on")
	if strings.Contains(get(t, handler, "/admin").Body.String(), "Tell your site what you want") {
		t.Fatal("the dashboard box needs an assistant")
	}
	mustContain(t, postJSON(t, handler, "/admin/api/assistant", `{"kind":"page","id":"`+id+`","prompt":"hi"}`).Body.String(), "No assistant is connected", "the API says so")
	mustContain(t, get(t, handler, "/admin/agents").Body.String(), "-assistant anthropic://YOUR_KEY", "and the Agents page tells the operator how")
}

func TestEditorsCannotPublishThroughTheAssistant(t *testing.T) {
	scripted := &scriptedAssistant{turns: []AssistantResponse{
		{StopReason: "tool_use", Content: []AssistantPart{{Type: "tool_use", ID: "t1", Name: "publish_page", Input: map[string]any{"page": "home"}}}},
		{StopReason: "end_turn", Content: []AssistantPart{{Type: "text", Text: "Tried."}}},
	}}
	host, _ := newAssistantHost(t, scripted)
	// An editor's identity carries read and write only.
	identity := identityFromUser(User{ID: "u2", Name: "Eve", Role: roleEditor})
	outcome := host.runAssistant(context.Background(), httptest.NewRequest(http.MethodPost, "/admin/api/assistant", nil), identity, assistantTask{Kind: "page", ID: firstPageID(t, host, "home"), Prompt: "publish it"})
	if !outcome.OK || outcome.Changed {
		t.Fatalf("outcome: %+v", outcome)
	}
	for _, tool := range scripted.calls[0].Tools {
		if tool.Name == "publish_page" {
			t.Fatal("publish is not offered to an editor")
		}
	}
	if len(outcome.Steps) != 1 || !strings.Contains(outcome.Steps[0], "didn't work") {
		t.Fatalf("steps: %v", outcome.Steps)
	}
}

func TestTheEchoAssistantAppendsTheOwnersWords(t *testing.T) {
	provider, err := ParseAssistantURL("echo://")
	if err != nil || provider == nil {
		t.Fatal(err)
	}
	host, handler := newAssistantHost(t, provider)
	id := firstPageID(t, host, "home")
	body := postJSON(t, handler, "/admin/api/assistant", `{"kind":"page","id":"`+id+`","prompt":"We bake at dawn."}`).Body.String()
	mustContain(t, body, `"ok":true`, "echo works")
	mustContain(t, body, `"changed":true`, "and changes the page")
	page, _, _ := host.Store().PageByID(id)
	if last := page.Body.Blocks[len(page.Body.Blocks)-1]; last.Values["text"].String != "We bake at dawn." {
		t.Fatalf("echo appended: %+v", last.Values)
	}
	if _, err := ParseAssistantURL("anthropic://"); err == nil {
		t.Fatal("anthropic needs a key")
	}
	if provider, err := ParseAssistantURL("anthropic://sk-test?model=claude-opus-5"); err != nil || provider.Name() != "Claude (claude-opus-5)" {
		t.Fatalf("anthropic url: %v %v", provider, err)
	}
	if _, err := ParseAssistantURL("magic://x"); err == nil {
		t.Fatal("unknown schemes are refused")
	}
}
