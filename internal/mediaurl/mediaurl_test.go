package mediaurl

import "testing"

func TestForImageURLContract(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want string
		ok   bool
	}{
		{name: "root relative", raw: "/media/no-extension", want: "/media/no-extension", ok: true},
		{name: "dot relative", raw: "../media/cup.jpg", want: "../media/cup.jpg", ok: true},
		{name: "ordinary relative", raw: "media/cup.jpg", want: "media/cup.jpg", ok: true},
		{name: "http", raw: "http://cdn.example.test/cup", want: "http://cdn.example.test/cup", ok: true},
		{name: "https", raw: "HTTPS://cdn.example.test/cup", want: "HTTPS://cdn.example.test/cup", ok: true},
		{name: "mailto rejected for image", raw: "mailto:artist@example.test"},
		{name: "tel rejected for image", raw: "tel:+15550100"},
		{name: "javascript rejected", raw: "javascript:alert(1)"},
		{name: "data rejected", raw: "data:image/png;base64,AAAA"},
		{name: "blob rejected", raw: "blob:https://example.test/id"},
		{name: "file rejected", raw: "file:///tmp/cup.jpg"},
		{name: "protocol relative rejected", raw: "//cdn.example.test/cup.jpg"},
		{name: "triple slash rejected", raw: "///cdn.example.test/cup.jpg"},
		{name: "opaque http rejected", raw: "http:cup.jpg"},
		{name: "malformed host rejected", raw: "https:///cup.jpg"},
		{name: "empty authority with extra slash rejected", raw: "https:////host/cup.jpg"},
		{name: "invalid percent escape rejected", raw: "/media/cup%ZZ.jpg"},
		{name: "space rejected", raw: "/media/cup image.jpg"},
		{name: "control rejected", raw: "/media/cup.jpg\n"},
		{name: "C1 control rejected", raw: "/media/cup.jpg\u0085"},
		{name: "backslash rejected", raw: `\\evil\\cup.jpg`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := ForImage(tt.raw)
			if got != tt.want || ok != tt.ok {
				t.Fatalf("ForImage(%q) = (%q, %t), want (%q, %t)", tt.raw, got, ok, tt.want, tt.ok)
			}
		})
	}
}

func TestForLinkURLContract(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want string
		ok   bool
	}{
		{name: "relative", raw: "/shop?from=home#top", want: "/shop?from=home#top", ok: true},
		{name: "mailto", raw: "mailto:artist@example.test", want: "mailto:artist@example.test", ok: true},
		{name: "tel", raw: "tel:+15550100", want: "tel:+15550100", ok: true},
		{name: "active scheme", raw: "javascript:alert(1)"},
		{name: "unknown scheme", raw: "custom:thing"},
		{name: "data", raw: "data:text/html,hi"},
		{name: "empty", raw: ""},
		{name: "protocol relative", raw: "//cdn.example.test/contact"},
		{name: "triple slash", raw: "///cdn.example.test/contact"},
		{name: "invalid percent escape", raw: "/contact%ZZ"},
		{name: "C1 control", raw: "/contact\u0085"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := ForLink(tt.raw)
			if got != tt.want || ok != tt.ok {
				t.Fatalf("ForLink(%q) = (%q, %t), want (%q, %t)", tt.raw, got, ok, tt.want, tt.ok)
			}
		})
	}
}
