package shell

import (
	"strings"
	"testing"

	"m31labs.dev/gosx"
)

func TestRenderBackendEditorMediaDatalistEscapesContentTypeMetadata(t *testing.T) {
	html := gosx.RenderHTML(RenderBackendEditorMediaDatalist([]BackendEditorMediaAsset{{
		URL:         "/media/extensionless",
		Filename:    "extensionless",
		Alt:         "Artwork",
		ContentType: "image/png\" data-media-escaped=\"yes",
	}}))

	if !strings.Contains(html, `data-media-content-type="image/png&#34; data-media-escaped=&#34;yes"`) {
		t.Fatalf("editor datalist missing escaped content type: %s", html)
	}
	if strings.Contains(html, `data-media-escaped="yes"`) {
		t.Fatalf("editor datalist emitted an attribute from content type metadata: %s", html)
	}
}
