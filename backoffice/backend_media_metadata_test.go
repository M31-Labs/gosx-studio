package backoffice

import (
	"strings"
	"testing"

	"m31labs.dev/gosx"
)

func TestBackendMediaDatalistsExposeEscapedContentTypeMetadata(t *testing.T) {
	const contentType = `image/svg+xml" data-media-escaped="yes`

	tests := []struct {
		name   string
		render func(BackendMediaAssetForTest) gosx.Node
	}{
		{name: "page index", render: func(asset BackendMediaAssetForTest) gosx.Node {
			return RenderBackendPageIndexMediaDatalist([]BackendPageIndexMediaAsset{{URL: asset.URL, Filename: asset.Filename, Alt: asset.Alt, ContentType: asset.ContentType}})
		}},
		{name: "page detail", render: func(asset BackendMediaAssetForTest) gosx.Node {
			return RenderBackendPageDetailMediaDatalist([]BackendPageDetailMediaAsset{{URL: asset.URL, Filename: asset.Filename, Alt: asset.Alt, ContentType: asset.ContentType}})
		}},
		{name: "gallery index", render: func(asset BackendMediaAssetForTest) gosx.Node {
			return RenderBackendGalleryIndexMediaDatalist([]BackendGalleryIndexMediaAsset{{URL: asset.URL, Filename: asset.Filename, Alt: asset.Alt, ContentType: asset.ContentType}})
		}},
		{name: "gallery detail", render: func(asset BackendMediaAssetForTest) gosx.Node {
			return RenderBackendGalleryDetailMediaDatalist([]BackendGalleryDetailMediaAsset{{URL: asset.URL, Filename: asset.Filename, Alt: asset.Alt, ContentType: asset.ContentType}})
		}},
		{name: "blog index", render: func(asset BackendMediaAssetForTest) gosx.Node {
			return RenderBackendBlogIndexMediaDatalist([]BackendBlogIndexMediaAsset{{URL: asset.URL, Filename: asset.Filename, Alt: asset.Alt, ContentType: asset.ContentType}})
		}},
		{name: "blog detail", render: func(asset BackendMediaAssetForTest) gosx.Node {
			return RenderBackendBlogDetailMediaDatalist([]BackendBlogDetailMediaAsset{{URL: asset.URL, Filename: asset.Filename, Alt: asset.Alt, ContentType: asset.ContentType}})
		}},
		{name: "product index", render: func(asset BackendMediaAssetForTest) gosx.Node {
			return RenderBackendProductIndexMediaDatalist([]BackendProductIndexMediaAsset{{URL: asset.URL, Filename: asset.Filename, Alt: asset.Alt, ContentType: asset.ContentType}})
		}},
		{name: "product detail", render: func(asset BackendMediaAssetForTest) gosx.Node {
			return RenderBackendProductDetailMediaDatalist([]BackendProductDetailMediaAsset{{URL: asset.URL, Filename: asset.Filename, Alt: asset.Alt, ContentType: asset.ContentType}})
		}},
		{name: "settings", render: func(asset BackendMediaAssetForTest) gosx.Node {
			return RenderBackendSettingsMediaDatalist([]BackendSettingsMediaAsset{{URL: asset.URL, Filename: asset.Filename, Alt: asset.Alt, ContentType: asset.ContentType}})
		}},
	}

	asset := BackendMediaAssetForTest{
		URL:         "/media/extensionless",
		Filename:    `cup"><script>ignored</script>`,
		Alt:         `<Cup & vase>`,
		ContentType: contentType,
	}
	wantType := `data-media-content-type="image/svg+xml&#34; data-media-escaped=&#34;yes"`
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			html := gosx.RenderHTML(tt.render(asset))
			if !strings.Contains(html, `value="/media/extensionless"`) || !strings.Contains(html, wantType) {
				t.Fatalf("%s datalist missing escaped metadata: %s", tt.name, html)
			}
			if strings.Contains(html, `<script>`) || strings.Contains(html, `data-media-escaped="yes"`) {
				t.Fatalf("%s datalist emitted unescaped metadata: %s", tt.name, html)
			}
		})
	}

	withoutMetadata := gosx.RenderHTML(RenderBackendPageIndexMediaDatalist([]BackendPageIndexMediaAsset{{URL: "/media/cup.jpg", Filename: "cup.jpg"}}))
	if strings.Contains(withoutMetadata, "data-media-content-type") {
		t.Fatalf("absent content type should preserve extension-only fallback metadata shape: %s", withoutMetadata)
	}
}

type BackendMediaAssetForTest struct {
	URL         string
	Filename    string
	Alt         string
	ContentType string
}
