package sitehost

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func bigPNG(t *testing.T, w, h int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y += 50 {
		for x := 0; x < w; x += 50 {
			img.Set(x, y, color.RGBA{R: uint8(x % 255), G: uint8(y % 255), B: 90, A: 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func uploadedURL(t *testing.T, body string) string {
	t.Helper()
	start := strings.Index(body, "/uploads/")
	if start < 0 {
		t.Fatalf("no upload url in %s", body)
	}
	rest := body[start:]
	return rest[:strings.Index(rest, `"`)]
}

func TestUploadMakesVariantsAndRecordsDimensions(t *testing.T) {
	host, handler := newTestHost(t)
	body := postUpload(t, handler, "hero.png", bigPNG(t, 2000, 1200)).Body.String()
	imgURL := uploadedURL(t, body)
	name := strings.TrimPrefix(imgURL, "/uploads/")

	entry, ok := host.media.get(name)
	if !ok {
		t.Fatal("upload not recorded in the media index")
	}
	if entry.Width != 2000 || entry.Height != 1200 {
		t.Fatalf("dimensions = %dx%d", entry.Width, entry.Height)
	}
	if len(entry.Variants) != 3 {
		t.Fatalf("variants = %+v, want 480/960/1600", entry.Variants)
	}
	for _, variant := range entry.Variants {
		if _, err := os.Stat(filepath.Join(host.Options().uploadDir(), variant.Name)); err != nil {
			t.Fatalf("variant %s not on disk", variant.Name)
		}
		rec := get(t, handler, "/uploads/"+variant.Name)
		if rec.Code != http.StatusOK || rec.Header().Get("Content-Type") != "image/png" {
			t.Fatalf("variant %s = %d %s", variant.Name, rec.Code, rec.Header().Get("Content-Type"))
		}
		cfg, _, err := image.DecodeConfig(bytes.NewReader(rec.Body.Bytes()))
		if err != nil || cfg.Width != variant.Width {
			t.Fatalf("variant %s decoded %dx%d (%v), want width %d", variant.Name, cfg.Width, cfg.Height, err, variant.Width)
		}
	}
	// Aspect ratio survives the resize.
	if v := entry.Variants[0]; v.Width != 480 || v.Height != 288 {
		t.Fatalf("480 variant = %dx%d, want 480x288", v.Width, v.Height)
	}
}

func TestSmallUploadGetsNoVariantsButStillDimensions(t *testing.T) {
	host, handler := newTestHost(t)
	imgURL := uploadedURL(t, postUpload(t, handler, "s.png", bigPNG(t, 300, 200)).Body.String())
	entry, _ := host.media.get(strings.TrimPrefix(imgURL, "/uploads/"))
	if entry.Width != 300 || len(entry.Variants) != 0 {
		t.Fatalf("small image: %+v", entry)
	}
}

func TestPublicPageRendersResponsiveImages(t *testing.T) {
	host, handler := newTestHost(t)
	imgURL := uploadedURL(t, postUpload(t, handler, "hero.png", bigPNG(t, 2000, 1200)).Body.String())
	id := firstPageID(t, host, "menu")
	postJSON(t, handler, "/admin/api/pages/"+id, `{"title":"Menu","slug":"menu","blocks":[{"kind":"image","url":"`+imgURL+`","alt":"Our loaves"},{"kind":"image","url":"https://example.com/remote.jpg","alt":"Remote"}]}`)
	post(t, handler, "/admin/api/pages/"+id+"/publish", url.Values{})

	body := get(t, handler, "/menu").Body.String()
	mustContain(t, body, `width="2000" height="1200"`, "uploaded images carry intrinsic dimensions")
	mustContain(t, body, `loading="lazy"`, "images load lazily")
	mustContain(t, body, `decoding="async"`, "images decode off the main thread")
	mustContain(t, body, `srcset="`, "uploaded images carry a srcset")
	mustContain(t, body, `-w480.png 480w`, "the 480 rendition is offered")
	mustContain(t, body, imgURL+` 2000w`, "the original is offered at its width")
	mustContain(t, body, `sizes="(max-width: 720px) 100vw, 720px"`, "sizes matches the layout")
	mustContain(t, body, `alt="Our loaves"`, "alt text survives")
	// The pasted remote picture: lazy, but nothing is known about it.
	at := strings.Index(body, "remote.jpg")
	tagStart := strings.LastIndex(body[:at], "<img")
	tagEnd := at + strings.Index(body[at:], ">")
	tag := body[tagStart : tagEnd+1]
	if strings.Contains(tag, "srcset") || strings.Contains(tag, "width=") {
		t.Fatalf("a remote image must not get a fabricated srcset or size: %s", tag)
	}
	mustContain(t, tag, `loading="lazy"`, "a remote image still loads lazily")
}

func TestMediaLibraryListsUsageAndDeletes(t *testing.T) {
	host, handler := newTestHost(t)
	imgURL := uploadedURL(t, postUpload(t, handler, "hero.png", bigPNG(t, 1000, 500)).Body.String())
	name := strings.TrimPrefix(imgURL, "/uploads/")
	id := firstPageID(t, host, "menu")
	postJSON(t, handler, "/admin/api/pages/"+id, `{"title":"Menu","slug":"menu","blocks":[{"kind":"image","url":"`+imgURL+`","alt":"x"}]}`)

	page := get(t, handler, "/admin/media").Body.String()
	mustContain(t, page, `class="admin-media__thumb" src="/uploads/`+strings.TrimSuffix(name, ".png")+`-w480.png"`, "the library shows the small rendition")
	mustContain(t, page, "1000 × 500", "dimensions are shown")
	mustContain(t, page, "Used on 1 page", "usage is counted")
	mustContain(t, page, "Delete anyway", "deleting a used picture is a deliberate act")

	api := get(t, handler, "/admin/api/media").Body.String()
	mustContain(t, api, `"url":"`+imgURL+`"`, "the picker API lists the picture")
	mustContain(t, api, `"thumb":"/uploads/`, "the picker API offers a thumb")

	rec := post(t, handler, "/admin/media/"+name+"/delete", url.Values{})
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("delete = %d", rec.Code)
	}
	if _, ok := host.media.get(name); ok {
		t.Fatal("deleted picture still indexed")
	}
	if _, err := os.Stat(filepath.Join(host.Options().uploadDir(), name)); !os.IsNotExist(err) {
		t.Fatal("deleted picture still on disk")
	}
	if code := get(t, handler, imgURL).Code; code != http.StatusNotFound {
		t.Fatal("deleted picture still served")
	}
	mustContain(t, get(t, handler, "/admin/media").Body.String(), "No pictures yet", "the library empties")

	// Variants cannot be deleted by name, only the original.
	if rec := post(t, handler, "/admin/media/"+strings.TrimSuffix(name, ".png")+"-w480.png/delete", url.Values{}); !strings.Contains(rec.Header().Get("Location"), "find+that") {
		t.Fatalf("variant delete = %q", rec.Header().Get("Location"))
	}
}

func TestEditorOffersThePicturePicker(t *testing.T) {
	host, handler := newTestHost(t)
	id := firstPageID(t, host, "menu")
	postJSON(t, handler, "/admin/api/pages/"+id, `{"title":"Menu","slug":"menu","blocks":[{"kind":"image","url":"/uploads/x.png","alt":""}]}`)
	mustContain(t, get(t, handler, "/admin/edit/"+id).Body.String(), `data-library="true"`, "image blocks offer the library")
	mustContain(t, get(t, handler, "/admin").Body.String(), `href="/admin/media"`, "the admin nav links to Pictures")
}
