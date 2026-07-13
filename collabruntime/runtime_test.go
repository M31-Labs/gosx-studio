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
		`[data-studio-preview-route][aria-pressed='true']`,
		`return canonicalRoute(value || window.location.pathname || "/")`,
		`pageId: detail.pageId || detail.pageID || currentPageID()`,
		`now - lastCursor < 50`,
		`send("studio.cursor.submit"`,
		`payload.source.connectionId === connectionID`,
		`gosxstudio:remote-selection`,
		`data-studio-remote-selection-source`,
		`gosxstudio:preview-route-change`,
		`storageSet(sequenceKey`,
		`data-studio-requires-author`,
		`data-studio-requires-design`,
		`[data-gosx-studio-operation-kind]`,
		`node.closest("[data-studio-layout-control]")`,
		`pending[operationID].resolve(ack)`,
		`available: function ()`,
		`permissions({}); connect();`,
		`gosxstudio:collab-heads:`,
		`expectedHead: function (request)`,
		`heads[targetKey(ack.record.target)]`,
		`setStatus("syncing", "Syncing…")`,
		`else { synced = true; permissions(serverPermissions); setStatus("connected", "Live"); }`,
		`return synced && !!connectionID`,
		`Object.prototype.hasOwnProperty.call(heads, key)`,
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

// TestRuntimeBroadensPresenceToEveryFacepileAndToggleSummaryDetail guards
// handoff-4 (punch #7 -- presence to top chrome): the collaborators
// facepile+count now lives in BOTH the toolbar's compact summary
// (shell.RenderWorkbenchCollaborationSummary) and the full detail panel
// (panels.RenderCollaborationPanel), fed by the SAME renderPresence
// payload -- this is a presentation-only broadening of which DOM nodes
// get written (document-wide query, not the one panel passed to
// create(panel)), not a change to the socket/presence protocol itself.
// Clicking the toolbar summary must reveal the (collapsed-by-default)
// detail panel.
func TestRuntimeBroadensPresenceToEveryFacepileAndToggleSummaryDetail(t *testing.T) {
	script := string(Script())
	for _, want := range []string{
		`function facepiles() { return qsa(document, "[data-studio-collab-facepile]"); }`,
		`function collabCounts() { return qsa(document, "[data-studio-collab-count]"); }`,
		`function collabSummaries() { return qsa(document, "[data-studio-collab-summary]"); }`,
		`facepiles().forEach(function (facepile) {`,
		`collabCounts().forEach(function (node) { node.textContent = String(members.length); });`,
		`collabSummaries().forEach(function (node) { node.setAttribute("data-studio-collab-state", value); });`,
		`function toggleCollaborationDetail(button)`,
		`var reveal = target.hasAttribute("hidden");`,
		`function bindCollaborationSummaries(root)`,
		`bindCollaborationSummaries(root);`,
	} {
		if !strings.Contains(script, want) {
			t.Fatalf("runtime missing %q", want)
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
