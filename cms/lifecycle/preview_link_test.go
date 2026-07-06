package lifecycle

import (
	"strings"
	"testing"
	"time"
)

func TestPreviewLinkSignVerifyRoundTrip(t *testing.T) {
	now := time.Date(2026, 5, 17, 20, 0, 0, 0, time.UTC)
	link, err := NewPreviewLink(PreviewLinkInput{
		ResourceKind: " page ",
		ResourceID:   " home ",
		Route:        "/pages/home",
		Audience:     "client",
		Issuer:       "draco",
		Nonce:        "nonce-1",
	}, now)
	if err != nil {
		t.Fatal(err)
	}
	if !link.Created.Equal(now) || !link.Expires.Equal(now.Add(DefaultPreviewTTL)) {
		t.Fatalf("unexpected timestamps: %#v", link)
	}
	token, err := SignPreviewLink(link, []byte("secret"))
	if err != nil {
		t.Fatal(err)
	}
	verified, err := VerifyPreviewLink(token, []byte("secret"), now.Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if verified.ResourceKind != "page" || verified.ResourceID != "home" || verified.Route != "/pages/home" {
		t.Fatalf("unexpected verified link: %#v", verified)
	}
}

func TestPreviewLinkRejectsTamperingAndExpiry(t *testing.T) {
	now := time.Date(2026, 5, 17, 20, 0, 0, 0, time.UTC)
	link, err := NewPreviewLink(PreviewLinkInput{
		ResourceKind: "settings",
		ResourceID:   "site",
		TTL:          time.Hour,
	}, now)
	if err != nil {
		t.Fatal(err)
	}
	token, err := SignPreviewLink(link, []byte("secret"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := VerifyPreviewLink(token+"x", []byte("secret"), now); err == nil {
		t.Fatal("expected tampered token to fail")
	}
	if _, err := VerifyPreviewLink(token, []byte("wrong-secret"), now); err == nil {
		t.Fatal("expected wrong secret to fail")
	}
	if _, err := VerifyPreviewLink(token, []byte("secret"), now.Add(2*time.Hour)); err == nil {
		t.Fatal("expected expired token to fail")
	}
}

func TestPreviewLinkValidation(t *testing.T) {
	now := time.Date(2026, 5, 17, 20, 0, 0, 0, time.UTC)
	if _, err := NewPreviewLink(PreviewLinkInput{ResourceKind: "page"}, now); err == nil {
		t.Fatal("expected missing resource id error")
	}
	if _, err := NewPreviewLink(PreviewLinkInput{
		ResourceKind: "page",
		ResourceID:   "p1",
		TTL:          MaxPreviewTTL + time.Hour,
	}, now); err == nil {
		t.Fatal("expected ttl limit error")
	}
	if _, err := SignPreviewLink(PreviewLink{}, []byte("secret")); err == nil {
		t.Fatal("expected invalid link error")
	}
	if _, err := SignPreviewLink(PreviewLink{ResourceKind: "page", ResourceID: "p1", Created: now, Expires: now.Add(time.Hour)}, nil); err == nil {
		t.Fatal("expected missing secret error")
	}
}

func TestPreviewURLBuildsRelativeAndAbsoluteLinks(t *testing.T) {
	now := time.Date(2026, 5, 17, 20, 0, 0, 0, time.UTC)
	link, err := NewPreviewLink(PreviewLinkInput{ResourceKind: "page", ResourceID: "p1", Route: "/pages/care?preview=1"}, now)
	if err != nil {
		t.Fatal(err)
	}
	relative, err := PreviewURL(link, "token", PreviewLinkOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if relative != "/pages/care?preview=1&preview_token=token" && relative != "/pages/care?preview_token=token&preview=1" {
		t.Fatalf("unexpected relative url: %q", relative)
	}
	absolute, err := PreviewURL(link, "token", PreviewLinkOptions{
		BaseURL:    "https://example.test",
		Path:       "/blog/post",
		QueryParam: "draft",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(absolute, "https://example.test/blog/post?draft=token") {
		t.Fatalf("unexpected absolute url: %q", absolute)
	}
	if _, err := PreviewURL(link, "", PreviewLinkOptions{}); err == nil {
		t.Fatal("expected missing token error")
	}
}
