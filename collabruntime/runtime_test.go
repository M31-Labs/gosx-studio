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
// TestRuntimeClassifiesInteractionOperationsAsDesignCapability guards
// handoff-31: set-interaction/remove-interaction require CapabilityDesign
// server-side (sqlstore.Apply groups them with style, HANDOFF-12), so the
// client-side optimistic-enable gate in permissions() must classify them as
// design actions too -- a studio-author (Author-only, no Design) principal
// must see these buttons DISABLED, matching what the server would reject,
// instead of an enabled button that always fails server-side.
func TestRuntimeClassifiesInteractionOperationsAsDesignCapability(t *testing.T) {
	script := string(Script())
	for _, want := range []string{
		`var kind = node.getAttribute("data-gosx-studio-operation-kind");`,
		`kind === "set-interaction" || kind === "remove-interaction"`,
	} {
		if !strings.Contains(script, want) {
			t.Fatalf("runtime missing interaction-as-design classification %q", want)
		}
	}
}

// TestRuntimeTargetKeyIncludesControlKey guards handoff-31's discovered
// cache-collision bug: two interactions (or shared-field/flow-field
// targets) attached to the SAME component share the same
// route/pageId/field/componentKey and differ ONLY by ControlKey. Before
// this fix, targetKey() omitted ControlKey entirely, so accepting the
// FIRST one's operation cached a head the SECOND (a distinct,
// never-before-written target) then wrongly reused as its own
// expectedTargetHead, getting rejected as a false stale-field conflict —
// reproduced live attaching reveal-on-scroll then hover-focus-state to the
// same canvas component in the same page load.
func TestRuntimeTargetKeyIncludesControlKey(t *testing.T) {
	script := string(Script())
	if !strings.Contains(script, `target.controlKey || ""`) {
		t.Fatalf("collabruntime's targetKey() must include target.controlKey so distinct interaction/instance/flow targets on the same component never collide:\n%s", script)
	}
}

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
