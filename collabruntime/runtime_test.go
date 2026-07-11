package collabruntime

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRuntimeOwnsTrustedRealtimeContracts(t *testing.T) {
	script := string(Script())
	for _, want := range []string{
		`new WebSocket(socketURL(endpoint))`,
		`send("studio.sync.resume", { afterSequence: acceptedSequence })`,
		`if (data.hasMore) send("studio.sync.resume"`,
		`send("studio.operation.submit", { request: request })`,
		`send("studio.selection.submit", selection)`,
		`now - lastCursor < 50`,
		`send("studio.cursor.submit"`,
		`payload.source.connectionId === connectionID`,
		`gosxstudio:remote-selection`,
		`data-studio-remote-selection-source`,
		`gosxstudio:preview-route-change`,
		`storageSet(sequenceKey`,
		`data-studio-requires-author`,
		`data-studio-requires-design`,
		`pending[operationID].resolve(ack)`,
		`available: function ()`,
		`permissions({}); connect();`,
	} {
		if !strings.Contains(script, want) {
			t.Fatalf("runtime missing %q", want)
		}
	}
	for _, forbidden := range []string{`X-Studio-Actor`, `capabilities:`, `actorId:`, `localStorage.setItem`} {
		if strings.Contains(script, forbidden) {
			t.Fatalf("runtime contains client authority/storage fragment %q", forbidden)
		}
	}
}

func TestHandlerServesNosniffJavaScript(t *testing.T) {
	rec := httptest.NewRecorder()
	Handler().ServeHTTP(rec, httptest.NewRequest("GET", "/collab.js", nil))
	if rec.Code != 200 || !strings.Contains(rec.Header().Get("Content-Type"), "javascript") || rec.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Fatalf("headers=%v status=%d", rec.Header(), rec.Code)
	}
}
