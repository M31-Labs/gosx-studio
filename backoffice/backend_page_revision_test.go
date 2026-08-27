package backoffice

import (
	"strings"
	"testing"

	"m31labs.dev/gosx"
)

func TestRenderBackendPageDetailRevisionInputOptInAndFailClosed(t *testing.T) {
	expected := "sha256:page-state"
	html := gosx.RenderHTML(RenderBackendPageDetailForm(BackendPageDetailPageProps{
		ExpectedRevision: &expected,
		Page: BackendPageDetailValues{
			ID: "page_1",
		},
	}))

	if want := `<input type="hidden" name="expectedRevision" value="sha256:page-state" data-content-editor-revision="true" />`; !strings.Contains(html, want) {
		t.Fatalf("revision-enabled form missing marked hidden token %q:\n%s", want, html)
	}
	if got := strings.Count(html, `name="expectedRevision"`); got != 1 {
		t.Fatalf("revision-enabled form should render one token input, got %d:\n%s", got, html)
	}

	empty := ""
	emptyHTML := gosx.RenderHTML(RenderBackendPageDetailForm(BackendPageDetailPageProps{
		ExpectedRevision: &empty,
		Page:             BackendPageDetailValues{ID: "page_1"},
	}))
	if want := `<input type="hidden" name="expectedRevision" value="" data-content-editor-revision="true" />`; !strings.Contains(emptyHTML, want) {
		t.Fatalf("guarded empty token must still render a marked hidden field %q:\n%s", want, emptyHTML)
	}

	unprotectedHTML := gosx.RenderHTML(RenderBackendPageDetailForm(BackendPageDetailPageProps{
		Page: BackendPageDetailValues{ID: "page_1"},
	}))
	if strings.Contains(unprotectedHTML, `name="expectedRevision"`) || strings.Contains(unprotectedHTML, `data-content-editor-revision="true"`) {
		t.Fatalf("unprotected host must not receive a revision input:\n%s", unprotectedHTML)
	}
}
