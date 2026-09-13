package sitehost

import (
	"encoding/json"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"m31labs.dev/gosx"
)

// collab.go lets two people edit one page at the same time without
// surprising each other.
//
// Every open editor holds a small event stream. It says who else is on the
// page, and it says when someone else saved. The browser then fetches the
// freshly rendered blocks and swaps them in, unless the person is in the
// middle of typing, in which case it waits for a pause and says so. There
// is no merging: the last save wins, and History keeps every version. That
// is honest, and it is what a small team actually needs.

const (
	editorClientHeader = "X-Editor-Client"
	collabPing         = 25 * time.Second
)

type collabEvent struct {
	Name string
	Data any
}

type collabSub struct {
	client string
	name   string
	ch     chan collabEvent
}

type presenceEntry struct {
	Name   string `json:"name"`
	Client string `json:"client"`
}

type changeNotice struct {
	By     string `json:"by"`
	Client string `json:"client"`
	Live   bool   `json:"live"`
	Chip   string `json:"chip"`
}

// collabHub is who is on which document right now.
type collabHub struct {
	mu   sync.Mutex
	docs map[string]map[*collabSub]struct{}
}

func newCollabHub() *collabHub { return &collabHub{docs: map[string]map[*collabSub]struct{}{}} }

func docKey(kind, id string) string { return kind + ":" + id }

func (c *collabHub) join(doc string, sub *collabSub) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.docs[doc] == nil {
		c.docs[doc] = map[*collabSub]struct{}{}
	}
	c.docs[doc][sub] = struct{}{}
}

func (c *collabHub) leave(doc string, sub *collabSub) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.docs[doc], sub)
	if len(c.docs[doc]) == 0 {
		delete(c.docs, doc)
	}
}

// presence lists everyone on a document, steady in order.
func (c *collabHub) presence(doc string) []presenceEntry {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([]presenceEntry, 0, len(c.docs[doc]))
	for sub := range c.docs[doc] {
		out = append(out, presenceEntry{Name: sub.name, Client: sub.client})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Client < out[j].Client })
	return out
}

// broadcast hands an event to everyone on the document. A tab that cannot
// keep up misses an event rather than holding everyone else.
func (c *collabHub) broadcast(doc string, event collabEvent) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for sub := range c.docs[doc] {
		select {
		case sub.ch <- event:
		default:
		}
	}
}

func (c *collabHub) count(doc string) int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.docs[doc])
}

// ---------- host side ----------

func (h *Host) mountCollab(mux *http.ServeMux) {
	mux.HandleFunc("GET /admin/api/pages/{id}/events", func(w http.ResponseWriter, r *http.Request) { h.handleEvents(w, r, "page") })
	mux.HandleFunc("GET /admin/api/posts/{id}/events", func(w http.ResponseWriter, r *http.Request) { h.handleEvents(w, r, "post") })
	mux.HandleFunc("GET /admin/api/pages/{id}/canvas", func(w http.ResponseWriter, r *http.Request) { h.handleCanvas(w, r, "page") })
	mux.HandleFunc("GET /admin/api/posts/{id}/canvas", func(w http.ResponseWriter, r *http.Request) { h.handleCanvas(w, r, "post") })
}

// editorName is what other people see this person as.
func (h *Host) editorName(r *http.Request) string {
	if user, ok := h.currentUser(r); ok && strings.TrimSpace(user.Name) != "" {
		return user.Name
	}
	return "The owner"
}

func (h *Host) subjectFor(kind, id string) (editorSubject, bool) {
	switch kind {
	case "page":
		page, ok, _ := h.store.PageByID(id)
		if !ok {
			return editorSubject{}, false
		}
		return h.pageSubject(page), true
	case "post":
		post, ok, _ := h.store.PostByID(id)
		if !ok {
			return editorSubject{}, false
		}
		return h.postSubject(post), true
	}
	return editorSubject{}, false
}

// notifyChanged tells every other open editor that this one saved or
// published.
func (h *Host) notifyChanged(r *http.Request, kind, id string) {
	subject, ok := h.subjectFor(kind, id)
	if !ok {
		return
	}
	h.collab.broadcast(docKey(kind, id), collabEvent{Name: "changed", Data: changeNotice{
		By: h.editorName(r), Client: strings.TrimSpace(r.Header.Get(editorClientHeader)), Live: subject.Live, Chip: subjectChip(subject),
	}})
}

func (h *Host) handleEvents(w http.ResponseWriter, r *http.Request, kind string) {
	id := r.PathValue("id")
	if _, ok := h.subjectFor(kind, id); !ok {
		http.NotFound(w, r)
		return
	}
	controller := http.NewResponseController(w)
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)

	doc := docKey(kind, id)
	client := strings.TrimSpace(r.URL.Query().Get("client"))
	if client == "" || len(client) > 32 {
		client = randomHex(6)
	}
	sub := &collabSub{client: client, name: h.editorName(r), ch: make(chan collabEvent, 16)}
	h.collab.join(doc, sub)
	presence := func() { h.collab.broadcast(doc, collabEvent{Name: "presence", Data: h.collab.presence(doc)}) }
	presence()
	defer func() {
		h.collab.leave(doc, sub)
		presence()
	}()

	write := func(event collabEvent) bool {
		raw, err := json.Marshal(event.Data)
		if err != nil {
			return true
		}
		if _, err := w.Write([]byte("event: " + event.Name + "\ndata: " + string(raw) + "\n\n")); err != nil {
			return false
		}
		return controller.Flush() == nil
	}
	ticker := time.NewTicker(collabPing)
	defer ticker.Stop()
	for {
		select {
		case <-r.Context().Done():
			return
		case event := <-sub.ch:
			if !write(event) {
				return
			}
		case <-ticker.C:
			if _, err := w.Write([]byte(": ping\n\n")); err != nil || controller.Flush() != nil {
				return
			}
		}
	}
}

// canvasResult is the freshly rendered blocks, for a tab that was told
// someone else saved.
type canvasResult struct {
	OK    bool   `json:"ok"`
	Title string `json:"title"`
	HTML  string `json:"html"`
	Live  bool   `json:"live"`
	Chip  string `json:"chip"`
}

func (h *Host) handleCanvas(w http.ResponseWriter, r *http.Request, kind string) {
	subject, ok := h.subjectFor(kind, r.PathValue("id"))
	if !ok {
		writeJSON(w, http.StatusNotFound, canvasResult{})
		return
	}
	blocks := make([]gosx.Node, 0, len(subject.Body.Blocks))
	for index, instance := range subject.Body.Blocks {
		blocks = append(blocks, h.renderEditableBlock(index, instance))
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, canvasResult{OK: true, Title: subject.Title, HTML: gosx.RenderHTML(gosx.Fragment(blocks...)), Live: subject.Live, Chip: subjectChip(subject)})
}

// subjectChip is the status word in the editor's top bar.
func subjectChip(subject editorSubject) string {
	switch {
	case subject.Review.Requested:
		return "Waiting for review"
	case !subject.Scheduled.IsZero():
		return "Scheduled for " + formatPostDate(subject.Scheduled)
	case subject.Live:
		return "Live"
	}
	return "Not published yet"
}

// renderPeople is the row of avatars in the top bar; the script fills it.
func renderPeople() gosx.Node {
	return gosx.El("div", gosx.Attrs(gosx.Attr("class", "ed-people"), gosx.Attr("data-people", "true"), gosx.Attr("hidden", "hidden"), gosx.Attr("aria-label", "Also editing right now"), gosx.Attr("role", "status")))
}
