package sitehost

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func tinyPNG(t *testing.T) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 2, 2))
	img.Set(0, 0, color.RGBA{R: 200, G: 40, B: 40, A: 255})
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func postUpload(t *testing.T, handler http.Handler, filename string, data []byte) *httptest.ResponseRecorder {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write(data); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/admin/api/upload", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec
}

func TestUploadStoresServesAndDedupes(t *testing.T) {
	host, handler := newTestHost(t)
	data := tinyPNG(t)

	rec := postUpload(t, handler, "shop.png", data)
	body := rec.Body.String()
	if rec.Code != http.StatusOK || !strings.Contains(body, `"ok":true`) {
		t.Fatalf("upload = %d: %s", rec.Code, body)
	}
	mustContain(t, body, `"url":"/uploads/`, "upload returns a public URL")
	url := body[strings.Index(body, "/uploads/"):]
	url = url[:strings.Index(url, `"`)]
	if !strings.HasSuffix(url, ".png") {
		t.Fatalf("sniffed type should give a .png name, got %q", url)
	}

	// The file is on disk beside the site data, and served with the right type.
	if _, err := os.Stat(filepath.Join(host.Options().uploadDir(), strings.TrimPrefix(url, "/uploads/"))); err != nil {
		t.Fatalf("uploaded file not on disk: %v", err)
	}
	served := get(t, handler, url)
	if served.Code != http.StatusOK {
		t.Fatalf("GET %s = %d, want 200", url, served.Code)
	}
	if got := served.Header().Get("Content-Type"); got != "image/png" {
		t.Fatalf("served Content-Type = %q, want image/png", got)
	}
	if got := served.Header().Get("X-Content-Type-Options"); got != "nosniff" {
		t.Fatalf("served without nosniff: %q", got)
	}
	if !bytes.Equal(served.Body.Bytes(), data) {
		t.Fatal("served bytes differ from the upload")
	}

	// Same bytes again: same URL, one file.
	again := postUpload(t, handler, "different-name.png", data).Body.String()
	mustContain(t, again, `"url":"`+url+`"`, "identical uploads share one URL")
	entries, _ := os.ReadDir(host.Options().uploadDir())
	if len(entries) != 1 {
		t.Fatalf("expected one stored file after a duplicate upload, found %d", len(entries))
	}
}

func TestUploadRejectsNonImagesByContentNotName(t *testing.T) {
	_, handler := newTestHost(t)
	body := postUpload(t, handler, "totally-a-picture.png", []byte("<script>alert(1)</script>")).Body.String()
	if strings.Contains(body, `"ok":true`) {
		t.Fatal("a text file with a .png name was accepted")
	}
	mustContain(t, body, "doesn't look like a picture", "the refusal is in plain words")

	svg := []byte(`<?xml version="1.0"?><svg xmlns="http://www.w3.org/2000/svg"><script>alert(1)</script></svg>`)
	if strings.Contains(postUpload(t, handler, "logo.svg", svg).Body.String(), `"ok":true`) {
		t.Fatal("SVG must be refused: it can carry script")
	}
}

func TestUploadRejectsOversize(t *testing.T) {
	_, handler := newTestHost(t)
	// A valid PNG header followed by enough padding to cross the cap.
	data := append(tinyPNG(t), make([]byte, maxUploadBytes)...)
	body := postUpload(t, handler, "huge.png", data).Body.String()
	if strings.Contains(body, `"ok":true`) {
		t.Fatal("an oversize upload was accepted")
	}
	mustContain(t, body, "too big", "the refusal says why")
}

func TestUploadServingIsConfinedToStoredNames(t *testing.T) {
	host, handler := newTestHost(t)
	// Plant a file that was not produced by an upload; it must not be served.
	dir := host.Options().uploadDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "secret.txt"), []byte("nope"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"/uploads/secret.txt", "/uploads/..%2fsite.json", "/uploads/abc.png"} {
		if code := get(t, handler, path).Code; code != http.StatusNotFound {
			t.Fatalf("GET %s = %d, want 404", path, code)
		}
	}
	// A literal ".." never reaches the handler: net/http cleans the path and
	// redirects, and the redirect target is a public page slug, not a file.
	rec := get(t, handler, "/uploads/../site.json")
	if rec.Code != http.StatusTemporaryRedirect && rec.Code != http.StatusMovedPermanently && rec.Code != http.StatusNotFound {
		t.Fatalf("GET /uploads/../site.json = %d, want a redirect away from the upload directory or 404", rec.Code)
	}
	if location := rec.Header().Get("Location"); strings.HasPrefix(location, "/uploads/") {
		t.Fatalf("traversal redirected back under /uploads/: %q", location)
	}
	if body := get(t, handler, "/site.json").Body.String(); strings.Contains(body, `"pages"`) {
		t.Fatal("the site data file is reachable through the public site")
	}
}

func TestEditorImageBlockOffersUpload(t *testing.T) {
	host, handler := newTestHost(t)
	page, ok, err := host.Store().PageBySlug("menu")
	if err != nil || !ok {
		t.Fatal("menu page missing")
	}
	// Give the page an image block so the control renders.
	payload := `{"title":"Menu","slug":"menu","blocks":[{"kind":"image","url":"/uploads/x.png","alt":"Loaves"}]}`
	req := httptest.NewRequest(http.MethodPost, "/admin/api/pages/"+page.ID, strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	handler.ServeHTTP(httptest.NewRecorder(), req)

	body := get(t, handler, "/admin/edit/"+page.ID).Body.String()
	mustContain(t, body, `data-upload="true"`, "image blocks carry an upload control")
	mustContain(t, body, `accept="image/png,image/jpeg,image/gif,image/webp"`, "the picker is limited to image types")
}
