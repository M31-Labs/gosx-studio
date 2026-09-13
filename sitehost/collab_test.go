package sitehost

import (
	"bufio"
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

// eventStream opens the editor's event stream the way a browser tab does
// and hands back a channel of raw lines.
func eventStream(t *testing.T, ctx context.Context, serverURL, path string, session *http.Cookie) (<-chan string, *http.Response) {
	t.Helper()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, serverURL+path, nil)
	if err != nil {
		t.Fatal(err)
	}
	req.AddCookie(session)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	lines := make(chan string, 64)
	go func() {
		reader := bufio.NewReader(res.Body)
		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				close(lines)
				return
			}
			lines <- strings.TrimRight(line, "\n")
		}
	}()
	return lines, res
}

func awaitLine(t *testing.T, lines <-chan string, match func(string) bool, why string) string {
	t.Helper()
	deadline := time.After(3 * time.Second)
	for {
		select {
		case line, ok := <-lines:
			if !ok {
				t.Fatalf("%s: stream closed", why)
			}
			if match(line) {
				return line
			}
		case <-deadline:
			t.Fatalf("%s: no such event within 3s", why)
		}
	}
}

func TestCoEditingStreamsPresenceAndChanges(t *testing.T) {
	host, handler := newGuardedHost(t)
	rec := post(t, handler, loginPath, url.Values{"name": {"Ana Ruiz"}, "email": {"ana@example.com"}, "password": {"correct horse battery"}, "serverPassword": {"server-secret"}})
	session := cookieNamed(rec, sessionCookie)
	id := firstPageID(t, host, "menu")
	mustContain(t, getWithCookie(t, handler, "/admin/edit/"+id, session).Body.String(), `data-people="true"`, "the editor has room for the people on the page")

	server := httptest.NewServer(handler)
	defer server.Close()
	ctxA, cancelA := context.WithCancel(context.Background())
	defer cancelA()
	linesA, resA := eventStream(t, ctxA, server.URL, "/admin/api/pages/"+id+"/events?client=tabA", session)
	if resA.StatusCode != http.StatusOK || !strings.HasPrefix(resA.Header.Get("Content-Type"), "text/event-stream") {
		t.Fatalf("stream = %d %q", resA.StatusCode, resA.Header.Get("Content-Type"))
	}
	awaitLine(t, linesA, func(l string) bool { return l == "event: presence" }, "joining announces presence")
	awaitLine(t, linesA, func(l string) bool { return strings.Contains(l, `"name":"Ana Ruiz","client":"tabA"`) }, "with this tab in it")

	// A second tab arrives; the first hears about it.
	ctxB, cancelB := context.WithCancel(context.Background())
	linesB, _ := eventStream(t, ctxB, server.URL, "/admin/api/pages/"+id+"/events?client=tabB", session)
	awaitLine(t, linesA, func(l string) bool {
		return strings.Contains(l, `"client":"tabA"`) && strings.Contains(l, `"client":"tabB"`)
	}, "both tabs are listed")
	awaitLine(t, linesB, func(l string) bool { return strings.Contains(l, `"client":"tabB"`) }, "the new tab sees itself")
	if host.collab.count(docKey("page", id)) != 2 {
		t.Fatalf("hub count = %d", host.collab.count(docKey("page", id)))
	}

	// Tab B saves; tab A is told who and which tab, then fetches the blocks.
	req := httptest.NewRequest(http.MethodPost, "/admin/api/pages/"+id, strings.NewReader(`{"title":"Menu","slug":"menu","description":"x","blocks":[{"kind":"paragraph","text":"Autumn tasting menu"}]}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-CSRF-Token", csrfWith(t, handler, session))
	req.Header.Set(editorClientHeader, "tabB")
	req.AddCookie(session)
	saveRec := httptest.NewRecorder()
	handler.ServeHTTP(saveRec, req)
	mustContain(t, saveRec.Body.String(), `"ok":true`, "the save works")
	awaitLine(t, linesA, func(l string) bool { return l == "event: changed" }, "tab A hears the change")
	change := awaitLine(t, linesA, func(l string) bool { return strings.Contains(l, `"client":"tabB"`) }, "and which tab made it")
	mustContain(t, change, `"by":"Ana Ruiz"`, "and who")
	mustContain(t, change, `"live":true`, "and whether the page is live")
	canvas := getWithCookie(t, handler, "/admin/api/pages/"+id+"/canvas", session)
	mustContain(t, canvas.Body.String(), "Autumn tasting menu", "the canvas endpoint has the new blocks")
	mustContain(t, canvas.Body.String(), `"title":"Menu"`, "and the title")
	mustContain(t, canvas.Body.String(), `class=\"ed-block\"`, "rendered as editable blocks")
	if canvas.Header().Get("Cache-Control") != "no-store" {
		t.Fatal("the canvas must not be cached")
	}

	// Publishing tells the room the chip changed.
	pub := httptest.NewRequest(http.MethodPost, "/admin/api/pages/"+id+"/publish", nil)
	pub.Header.Set("X-CSRF-Token", csrfWith(t, handler, session))
	pub.Header.Set(editorClientHeader, "tabB")
	pub.AddCookie(session)
	handler.ServeHTTP(httptest.NewRecorder(), pub)
	awaitLine(t, linesA, func(l string) bool { return strings.Contains(l, `"chip":"Live"`) }, "publishing is announced")

	// Tab B leaves; tab A sees the room shrink.
	cancelB()
	awaitLine(t, linesA, func(l string) bool {
		return strings.HasPrefix(l, "data: ") && strings.Contains(l, `"client":"tabA"`) && !strings.Contains(l, "tabB")
	}, "tab B is gone")

	// Strangers get nothing.
	anon, err := http.Get(server.URL + "/admin/api/pages/" + id + "/events")
	if err != nil {
		t.Fatal(err)
	}
	anon.Body.Close()
	if anon.StatusCode == http.StatusOK && strings.HasPrefix(anon.Header.Get("Content-Type"), "text/event-stream") {
		t.Fatal("a stream must need a sign-in")
	}
}
