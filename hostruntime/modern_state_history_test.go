package hostruntime

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestModernStateRuntimeScopesDocumentShortcuts(t *testing.T) {
	script := string(StateRuntimeScript())
	for _, want := range []string{
		`function editableShortcutTarget(target)`,
		`input, textarea, select`,
		`contenteditable`,
		`function activeFormForEvent(event)`,
		`event.isComposing`,
		`event.__gosxStudioStateShortcutHandled`,
		`activeFormForEvent(event) !== form`,
		`saveShortcutSubmitter(form)`,
		`submitter.hasAttribute && submitter.hasAttribute("formaction")`,
		`submitter.hasAttribute && submitter.hasAttribute("formmethod")`,
		`function formCSRFToken(formData)`,
		`function sameOriginURL(value)`,
		`"Accept": "application/json"`,
		`"X-Requested-With": "XMLHttpRequest"`,
		`requestHeaders["X-CSRF-Token"] = csrfToken`,
	} {
		if !strings.Contains(script, want) {
			t.Fatalf("state runtime missing scoped shortcut fragment %q", want)
		}
	}
	if strings.Contains(script, `if (event.defaultPrevented || !document.contains(form)) return;
      if ((event.metaKey || event.ctrlKey) && event.key.toLowerCase() === "z")`) {
		t.Fatal("state runtime still has the old unconditional document-level history shortcut handler")
	}
}

func TestModernStateRuntimeGuardsPendingNavigationAndDetachedResponses(t *testing.T) {
	script := string(StateRuntimeScript())
	for _, want := range []string{
		`var nativeNavigationPending = false`,
		`var disposed = false`,
		`var intentionalNavigation = false`,
		`function formActive()`,
		`if (!formActive()) {`,
		`window.addEventListener("popstate"`,
		`event.stopImmediatePropagation()`,
		`lastCommittedURL`,
		`nativeNavigationPending = true`,
		`!isDirty() || nativeNavigationPending`,
	} {
		if !strings.Contains(script, want) {
			t.Fatalf("state runtime missing pending-navigation safety fragment %q", want)
		}
	}
}

func TestModernStateRuntimeTreatsClientActionFailuresAsDirty(t *testing.T) {
	script := string(StateRuntimeScript())
	for _, want := range []string{
		`function parseActionResponse(response)`,
		`function responseLooksUnauthenticated(response)`,
		`function redirectLooksUnauthenticated(redirect)`,
		`function structuredActionSuccess(response, result)`,
		`redirected to sign-in`,
		`actionFailureMessage(result, "Studio action failed; no structured success response")`,
		`structuredActionSuccess(response, result)`,
		`no structured success response`,
		`setState(form, "dirty", "action-stale"`,
		`setState(form, "error", "action"`,
	} {
		if !strings.Contains(script, want) {
			t.Fatalf("state runtime missing truthful client-action handling fragment %q", want)
		}
	}
}

func TestModernStateRuntimeMirrorsCMSStudioAsset(t *testing.T) {
	host := string(StateRuntimeScript())
	cmsPath := filepath.Join("..", "cms", "studio", "assets", "state_runtime.js")
	cms, err := os.ReadFile(cmsPath)
	if err != nil {
		t.Fatalf("read CMS state runtime mirror: %v", err)
	}
	if host != string(cms) {
		t.Fatalf("CMS state runtime mirror differs from hostruntime/assets/state_runtime.js")
	}
}
