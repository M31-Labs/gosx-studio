package sitehost

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"m31labs.dev/gosx"
)

// assistant.go is the owner's assistant: a box in the editor and on the
// dashboard where a non-technical owner says what they want in plain words
// and a model does it with the same tools any agent gets. The model never
// touches the store directly; every change goes through the agent API as
// the signed-in person, so scopes, review, locks, and the audit log apply.

// AssistantProvider completes one turn of a tool-using conversation.
type AssistantProvider interface {
	Name() string
	Complete(ctx context.Context, req AssistantRequest) (AssistantResponse, error)
}

type AssistantRequest struct {
	System    string
	Messages  []AssistantMessage
	Tools     []AssistantTool
	MaxTokens int
}

type AssistantMessage struct {
	Role    string // user or assistant
	Content []AssistantPart
}

// AssistantPart is one content block: text, a tool call, or a tool result.
type AssistantPart struct {
	Type      string         `json:"type"`
	Text      string         `json:"text,omitempty"`
	ID        string         `json:"id,omitempty"`
	Name      string         `json:"name,omitempty"`
	Input     map[string]any `json:"input,omitempty"`
	ToolUseID string         `json:"tool_use_id,omitempty"`
	Result    string         `json:"-"`
	IsError   bool           `json:"is_error,omitempty"`
}

type AssistantTool struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"input_schema"`
}

type AssistantResponse struct {
	Content    []AssistantPart
	StopReason string
}

// ParseAssistantURL reads anthropic://KEY?model=claude-sonnet-5 (the
// default model when none is given). Empty means no assistant.
func ParseAssistantURL(raw string) (AssistantProvider, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return nil, err
	}
	switch parsed.Scheme {
	case "anthropic":
		key := parsed.Host
		if parsed.User != nil {
			key = parsed.User.Username()
		}
		if key == "" {
			return nil, errors.New("anthropic://KEY needs the API key")
		}
		model := firstNonEmpty(parsed.Query().Get("model"), "claude-sonnet-5")
		return &anthropicProvider{key: key, model: model, endpoint: firstNonEmpty(parsed.Query().Get("endpoint"), "https://api.anthropic.com/v1/messages")}, nil
	case "echo":
		return echoProvider{}, nil
	}
	return nil, errors.New("unknown assistant scheme " + parsed.Scheme + " (use anthropic://KEY?model=…)")
}

func (o Options) assistantProvider() AssistantProvider {
	if o.Assistant != nil {
		return o.Assistant
	}
	provider, err := ParseAssistantURL(o.AssistantURL)
	if err != nil {
		return nil
	}
	return provider
}

// ---------- Anthropic ----------

type anthropicProvider struct {
	key      string
	model    string
	endpoint string
	client   *http.Client
}

func (p *anthropicProvider) Name() string { return "Claude (" + p.model + ")" }

func (p *anthropicProvider) Complete(ctx context.Context, req AssistantRequest) (AssistantResponse, error) {
	type message struct {
		Role    string `json:"role"`
		Content []any  `json:"content"`
	}
	messages := make([]message, 0, len(req.Messages))
	for _, m := range req.Messages {
		content := make([]any, 0, len(m.Content))
		for _, part := range m.Content {
			switch part.Type {
			case "text":
				content = append(content, map[string]any{"type": "text", "text": part.Text})
			case "tool_use":
				content = append(content, map[string]any{"type": "tool_use", "id": part.ID, "name": part.Name, "input": part.Input})
			case "tool_result":
				content = append(content, map[string]any{"type": "tool_result", "tool_use_id": part.ToolUseID, "content": part.Result, "is_error": part.IsError})
			}
		}
		messages = append(messages, message{Role: m.Role, Content: content})
	}
	body := map[string]any{"model": p.model, "max_tokens": firstPositive(req.MaxTokens, 4096), "system": req.System, "messages": messages}
	if len(req.Tools) > 0 {
		body["tools"] = req.Tools
	}
	raw, _ := json.Marshal(body)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.endpoint, bytes.NewReader(raw))
	if err != nil {
		return AssistantResponse{}, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", p.key)
	httpReq.Header.Set("anthropic-version", "2023-06-01")
	client := p.client
	if client == nil {
		client = &http.Client{Timeout: 120 * time.Second}
	}
	resp, err := client.Do(httpReq)
	if err != nil {
		return AssistantResponse{}, err
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if resp.StatusCode >= 400 {
		var failure struct {
			Error struct {
				Message string `json:"message"`
			} `json:"error"`
		}
		_ = json.Unmarshal(data, &failure)
		return AssistantResponse{}, errors.New("the model answered " + resp.Status + ": " + firstNonEmpty(failure.Error.Message, string(data)))
	}
	var parsed struct {
		Content []struct {
			Type  string         `json:"type"`
			Text  string         `json:"text"`
			ID    string         `json:"id"`
			Name  string         `json:"name"`
			Input map[string]any `json:"input"`
		} `json:"content"`
		StopReason string `json:"stop_reason"`
	}
	if err := json.Unmarshal(data, &parsed); err != nil {
		return AssistantResponse{}, err
	}
	out := AssistantResponse{StopReason: parsed.StopReason}
	for _, part := range parsed.Content {
		out.Content = append(out.Content, AssistantPart{Type: part.Type, Text: part.Text, ID: part.ID, Name: part.Name, Input: part.Input})
	}
	return out, nil
}

// echoProvider is the assistant with no model behind it: it appends the
// owner's words to the page as a paragraph. It exists so the box can be
// tried, and tested end to end, without an API key.
type echoProvider struct{}

func (echoProvider) Name() string { return "Echo (no model; for trying the box)" }

func (echoProvider) Complete(ctx context.Context, req AssistantRequest) (AssistantResponse, error) {
	if len(req.Messages) == 0 {
		return AssistantResponse{StopReason: "end_turn"}, nil
	}
	last := req.Messages[len(req.Messages)-1]
	if last.Role == "user" && len(last.Content) > 0 && last.Content[0].Type == "text" {
		prompt := last.Content[0].Text
		page := ""
		if at := strings.Index(req.System, assistantDocMarker); at >= 0 {
			rest := req.System[at:]
			if i := strings.Index(rest, `"id":"`); i >= 0 {
				rest = rest[i+6:]
				if j := strings.Index(rest, `"`); j > 0 {
					page = rest[:j]
				}
			}
		}
		if page == "" {
			return AssistantResponse{StopReason: "end_turn", Content: []AssistantPart{{Type: "text", Text: "Echo can only change the page you are editing. Open a page and ask again."}}}, nil
		}
		return AssistantResponse{StopReason: "tool_use", Content: []AssistantPart{{Type: "tool_use", ID: "echo-1", Name: "edit_page", Input: map[string]any{"page": page, "append": []any{map[string]any{"kind": "paragraph", "text": prompt}}}}}}, nil
	}
	return AssistantResponse{StopReason: "end_turn", Content: []AssistantPart{{Type: "text", Text: "Added your words to the end of the page as a paragraph. Connect a real assistant under Agents to get more than an echo."}}}, nil
}

func firstPositive(values ...int) int {
	for _, value := range values {
		if value > 0 {
			return value
		}
	}
	return 0
}

// ---------- the loop ----------

type assistantTask struct {
	Kind   string // page, post, or site
	ID     string
	Prompt string
}

type assistantOutcome struct {
	OK      bool     `json:"ok"`
	Reply   string   `json:"reply,omitempty"`
	Steps   []string `json:"steps,omitempty"`
	Changed bool     `json:"changed"`
	Message string   `json:"message,omitempty"`
}

const assistantMaxRounds = 16

// assistantDocMarker introduces the page the owner is editing in the
// system prompt; the echo provider reads the page id after it.
const assistantDocMarker = "The page the owner is editing (JSON; use its id in tool calls):\n"

// assistantTools are the MCP tools as the model sees them, minus what the
// caller may not do.
func assistantTools(identity agentIdentity) []AssistantTool {
	out := []AssistantTool{}
	for _, tool := range mcpTools() {
		if tool.Name == "complete_setup" {
			continue
		}
		if tool.Scope != "" && !identity.can(tool.Scope) {
			continue
		}
		out = append(out, AssistantTool{Name: tool.Name, Description: tool.Description, InputSchema: tool.Schema})
	}
	return out
}

// assistantSystem is what the model knows before it starts: the site, the
// document at hand, the block schema in brief, and the house rules.
func (h *Host) assistantSystem(ctx context.Context, r *http.Request, identity agentIdentity, task assistantTask) string {
	var b strings.Builder
	b.WriteString("You are the website assistant for a small business owner who is not technical. You change their website with the tools you are given. The owner sees the result on their screen as soon as you finish.\n\n")
	b.WriteString("Rules:\n- Read before you write: the site and the page are below; call get_schema if you need field names.\n- Prefer ready-made sections (hero, features, testimonials, pricing, faq, cta, stats, team, hours, imagetext, map) over loose paragraphs. Keep the owner's existing content unless asked to replace it.\n- Write short, concrete, warm copy in the business's own voice. No filler, no exclamation marks, no invented facts: leave a clear placeholder such as \"(your price)\" where you lack a fact.\n- Use edit_page with replace/insert/append/remove for small changes; use replace_page only to rebuild a page.\n- Never publish unless the owner asked to publish. Never change the Look or settings unless asked.\n- When done, answer in one to three short sentences saying what you changed, in plain words, with no tool names or JSON.\n\n")
	if _, _, site := h.agentDispatch(ctx, identity, http.MethodGet, agentAPIPrefix+"/site", nil, r); len(site) > 0 {
		b.WriteString("The site (JSON):\n" + clip(string(site), 12000) + "\n\n")
	}
	switch task.Kind {
	case "page":
		if _, _, page := h.agentDispatch(ctx, identity, http.MethodGet, agentAPIPrefix+"/pages/"+url.PathEscape(task.ID), nil, r); len(page) > 0 {
			b.WriteString(assistantDocMarker + clip(string(page), 24000) + "\n\n")
		}
	case "post":
		if _, _, post := h.agentDispatch(ctx, identity, http.MethodGet, agentAPIPrefix+"/posts/"+url.PathEscape(task.ID), nil, r); len(post) > 0 {
			b.WriteString("The post the owner is editing (JSON; use its id in tool calls):\n" + clip(string(post), 24000) + "\n\n")
		}
	default:
		b.WriteString("The owner is on the dashboard, so the request may concern any page or the whole site. Build or change pages as needed; keep the home page first.\n\n")
	}
	b.WriteString("Block kinds in brief (call get_schema for details):\n")
	for _, spec := range composites {
		line := "- " + spec.Key + ": " + spec.Blurb
		if len(spec.Fields) > 0 {
			keys := []string{}
			for _, field := range spec.Fields {
				keys = append(keys, field.Key)
			}
			line += " Fields: " + strings.Join(keys, ", ") + "."
		}
		if spec.Item != nil {
			keys := []string{}
			for _, field := range spec.Item {
				keys = append(keys, field.Key)
			}
			line += " Items (" + spec.ItemName + "): " + strings.Join(keys, ", ") + "."
		}
		if len(spec.Variants) > 0 {
			keys := []string{}
			for _, variant := range spec.Variants {
				keys = append(keys, variant.Key)
			}
			line += " Variants: " + strings.Join(keys, ", ") + "."
		}
		b.WriteString(line + "\n")
	}
	b.WriteString("- heading (text, level 2|3), paragraph (text), quote (text), list (text: one line each), button (text, url, look), image (url, alt, caption), gallery (images[]), video (url), columns (text, text2, text3), divider, section (style plain|tinted|accent|dark|image, align, width, space, anchor: starts a band for the blocks after it), form (form id), product (product id).\n")
	b.WriteString("Text accepts **bold**, _italic_, [label](href), and line breaks.\n")
	return b.String()
}

func clip(text string, max int) string {
	if len(text) <= max {
		return text
	}
	return text[:max] + "…(cut)"
}

// runAssistant carries out one request and reports what happened.
func (h *Host) runAssistant(ctx context.Context, r *http.Request, identity agentIdentity, task assistantTask) assistantOutcome {
	if h.assistant == nil {
		return assistantOutcome{Message: "No assistant is connected to this site yet. An admin connects one under Agents."}
	}
	prompt := strings.TrimSpace(task.Prompt)
	if prompt == "" {
		return assistantOutcome{Message: "Say what you'd like changed."}
	}
	if len(prompt) > 4000 {
		prompt = prompt[:4000]
	}
	ctx, cancel := context.WithTimeout(ctx, 3*time.Minute)
	defer cancel()
	req := AssistantRequest{System: h.assistantSystem(ctx, r, identity, task), Tools: assistantTools(identity), MaxTokens: 4096}
	req.Messages = []AssistantMessage{{Role: "user", Content: []AssistantPart{{Type: "text", Text: prompt}}}}
	outcome := assistantOutcome{}
	for round := 0; round < assistantMaxRounds; round++ {
		resp, err := h.assistant.Complete(ctx, req)
		if err != nil {
			outcome.Message = "The assistant stopped: " + err.Error()
			if outcome.Changed {
				outcome.Message += " Some changes were already made; check the page."
			}
			return outcome
		}
		req.Messages = append(req.Messages, AssistantMessage{Role: "assistant", Content: resp.Content})
		results := []AssistantPart{}
		texts := []string{}
		for _, part := range resp.Content {
			switch part.Type {
			case "text":
				if text := strings.TrimSpace(part.Text); text != "" {
					texts = append(texts, text)
				}
			case "tool_use":
				tool, ok := mcpToolByName(part.Name)
				if !ok {
					results = append(results, AssistantPart{Type: "tool_result", ToolUseID: part.ID, Result: "unknown tool", IsError: true})
					continue
				}
				args := part.Input
				if args == nil {
					args = map[string]any{}
				}
				method, path, body := tool.Call(args)
				var raw []byte
				if body != nil {
					raw, _ = json.Marshal(body)
				}
				status, _, out := h.agentDispatch(ctx, identity, method, path, raw, r)
				isError := status >= 400
				outcome.Steps = append(outcome.Steps, describeStep(tool, args, isError))
				if !isError && method != http.MethodGet {
					outcome.Changed = true
				}
				results = append(results, AssistantPart{Type: "tool_result", ToolUseID: part.ID, Result: clip(string(out), 16000), IsError: isError})
			}
		}
		if len(results) == 0 || resp.StopReason == "end_turn" && len(results) == 0 {
			outcome.OK = true
			outcome.Reply = strings.Join(texts, "\n\n")
			if outcome.Reply == "" {
				outcome.Reply = "Done."
			}
			return outcome
		}
		req.Messages = append(req.Messages, AssistantMessage{Role: "user", Content: results})
	}
	outcome.OK = true
	outcome.Reply = "I made the changes I could in the time allowed. Check the page and ask again for anything still missing."
	return outcome
}

// describeStep is the plain-words line an owner sees for one tool call.
func describeStep(tool mcpTool, args map[string]any, failed bool) string {
	subject := firstNonEmpty(argString(args, "page"), argString(args, "post"), argString(args, "title"), argString(args, "name"))
	verbs := map[string]string{
		"get_site": "Looked at the site", "get_schema": "Checked what sections are available", "update_site": "Changed site settings", "list_pages": "Listed the pages", "get_page": "Read the page",
		"page_markdown": "Read the page", "create_page": "Created a page", "replace_page": "Rebuilt the page", "edit_page": "Changed the page", "publish_page": "Published the page", "page_action": "Changed the page's status",
		"list_posts": "Listed the posts", "get_post": "Read the post", "create_post": "Wrote a post", "update_post": "Changed the post", "publish_post": "Published the post", "list_products": "Looked at the shop",
		"save_product": "Saved a product", "delete_product": "Removed a product", "list_media": "Looked at the pictures", "upload_image": "Added a picture", "get_look": "Looked at the Look", "set_look": "Changed the Look",
		"list_presets": "Looked at the presets", "save_preset": "Saved a preset", "list_forms": "Looked at the forms", "list_messages": "Read the messages", "get_stats": "Looked at visitor counts", "get_activity": "Looked at recent activity",
	}
	line := firstNonEmpty(verbs[tool.Name], tool.Name)
	if subject != "" {
		line += " (" + subject + ")"
	}
	if failed {
		line += " — that didn't work"
	}
	return line
}

// ---------- routes ----------

const assistantAPIPath = "/admin/api/assistant"

func (h *Host) mountAssistant(mux *http.ServeMux) {
	mux.HandleFunc("GET "+assistantAPIPath, h.handleAssistantStatus)
	mux.HandleFunc("POST "+assistantAPIPath, h.handleAssistantAsk)
	mux.HandleFunc("POST /admin/assistant", h.handleAssistantForm)
}

func (h *Host) assistantIdentity(r *http.Request) agentIdentity {
	if user, ok := h.currentUser(r); ok {
		return identityFromUser(user)
	}
	return agentIdentity{Name: "the owner", Scopes: map[string]bool{scopeRead: true, scopeWrite: true, scopePublish: true, scopeSettings: true}}
}

func (h *Host) handleAssistantStatus(w http.ResponseWriter, r *http.Request) {
	name := ""
	if h.assistant != nil {
		name = h.assistant.Name()
	}
	writeJSON(w, http.StatusOK, map[string]any{"available": h.assistant != nil, "name": name})
}

type assistantAsk struct {
	Kind   string `json:"kind"`
	ID     string `json:"id"`
	Prompt string `json:"prompt"`
}

func (h *Host) handleAssistantAsk(w http.ResponseWriter, r *http.Request) {
	var ask assistantAsk
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10)).Decode(&ask); err != nil {
		writeJSON(w, http.StatusBadRequest, assistantOutcome{Message: "We couldn't read that. Try again."})
		return
	}
	kind := strings.ToLower(strings.TrimSpace(ask.Kind))
	if kind != "page" && kind != "post" {
		kind = "site"
	}
	outcome := h.runAssistant(r.Context(), r, h.assistantIdentity(r), assistantTask{Kind: kind, ID: strings.TrimSpace(ask.ID), Prompt: ask.Prompt})
	if outcome.Changed {
		h.auditContent(r, "assistant.changed", "Asked the assistant: “"+clip(strings.TrimSpace(ask.Prompt), 120)+"”")
	}
	writeJSON(w, http.StatusOK, outcome)
}

// handleAssistantForm is the dashboard's box: a plain form, so it works
// without any script, and the page waits while the assistant works.
func (h *Host) handleAssistantForm(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Redirect(w, r, "/admin", http.StatusSeeOther)
		return
	}
	outcome := h.runAssistant(r.Context(), r, h.assistantIdentity(r), assistantTask{Kind: "site", Prompt: r.PostFormValue("prompt")})
	if outcome.Changed {
		h.auditContent(r, "assistant.changed", "Asked the assistant: “"+clip(strings.TrimSpace(r.PostFormValue("prompt")), 120)+"”")
	}
	h.renderDashboard(w, r, &outcome)
}

// ---------- UI ----------

// renderAssistantBox is the "Ask" box, in the editor sidebar and on the
// dashboard. kind and id tell the assistant what the owner is looking at.
func (h *Host) renderAssistantBox(kind, id string, outcome *assistantOutcome, asForm bool) gosx.Node {
	if h.assistant == nil {
		return gosx.El("div", gosx.Attrs(gosx.Attr("class", "ed-side__block ed-assist ed-assist--off"), gosx.Attr("data-assistant", "off")),
			gosx.El("h2", nil, gosx.Text("Ask your site")),
			gosx.El("p", gosx.Attrs(gosx.Attr("class", "ed-hint")), gosx.Text("Say what you want in plain words and an assistant does it. "),
				gosx.El("a", gosx.Attrs(gosx.Attr("href", "/admin/agents")), gosx.Text("Connect an assistant"))))
	}
	examples := []string{"Add a pricing section with three plans", "Rewrite the intro so it sounds warmer", "Add our opening hours and a map"}
	if kind == "site" {
		examples = []string{"We're a family bakery in Oakland; write the home page", "Add an About page that tells our story", "Put our opening hours on every page's footer"}
	}
	chips := make([]gosx.Node, 0, len(examples))
	for _, example := range examples {
		chips = append(chips, gosx.El("button", gosx.Attrs(gosx.Attr("type", "button"), gosx.Attr("class", "ed-assist__chip"), gosx.Attr("data-assistant-example", example)), gosx.Text(example)))
	}
	var result gosx.Node = gosx.El("div", gosx.Attrs(gosx.Attr("class", "ed-assist__reply"), gosx.Attr("data-assistant-reply", "true"), gosx.Attr("aria-live", "polite")))
	if outcome != nil {
		steps := make([]gosx.Node, 0, len(outcome.Steps))
		for _, step := range outcome.Steps {
			steps = append(steps, gosx.El("li", nil, gosx.Text(step)))
		}
		class := "ed-assist__reply"
		text := outcome.Reply
		if !outcome.OK {
			class += " ed-assist__reply--error"
			text = outcome.Message
		}
		result = gosx.El("div", gosx.Attrs(gosx.Attr("class", class), gosx.Attr("data-assistant-reply", "true"), gosx.Attr("aria-live", "polite")),
			gosx.El("p", nil, gosx.Text(text)),
			gosx.El("ul", gosx.Attrs(gosx.Attr("class", "ed-assist__steps")), gosx.Fragment(steps...)))
	}
	fields := gosx.Fragment(
		gosx.El("textarea", gosx.Attrs(gosx.Attr("class", "ed-assist__prompt"), gosx.Attr("name", "prompt"), gosx.Attr("rows", "3"), gosx.Attr("placeholder", "What would you like changed?"), gosx.Attr("data-assistant-prompt", "true"), gosx.Attr("aria-label", "What would you like changed?"))),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "ed-assist__chips")), gosx.Fragment(chips...)),
		gosx.El("button", gosx.Attrs(gosx.Attr("type", ternary(asForm, "submit", "button")), gosx.Attr("class", "ed-assist__send"), gosx.Attr("data-assistant-send", "true")), gosx.Text("Do it")),
		result,
	)
	if asForm {
		return gosx.El("section", gosx.Attrs(gosx.Attr("class", "admin-panel ed-assist"), gosx.Attr("data-assistant", kind)),
			gosx.El("h2", nil, gosx.Text("Tell your site what you want")),
			gosx.El("p", gosx.Attrs(gosx.Attr("class", "admin-hint")), gosx.Text("Describe your business or ask for a change in plain words. The assistant ("+h.assistant.Name()+") edits your pages as drafts; you publish when you're happy.")),
			gosx.El("form", gosx.Attrs(gosx.Attr("method", "post"), gosx.Attr("action", "/admin/assistant"), gosx.Attr("class", "ed-assist__form")), h.csrfField(), fields))
	}
	return gosx.El("div", gosx.Attrs(gosx.Attr("class", "ed-side__block ed-assist"), gosx.Attr("data-assistant", kind), gosx.Attr("data-assistant-id", id)),
		gosx.El("h2", nil, gosx.Text("Ask your site")),
		gosx.El("p", gosx.Attrs(gosx.Attr("class", "ed-hint")), gosx.Text("Say what you want in plain words. The assistant changes this "+firstNonEmpty(kind, "page")+" as a draft; you can undo it from History.")),
		fields)
}

func ternary(condition bool, yes, no string) string {
	if condition {
		return yes
	}
	return no
}
