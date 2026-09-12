package sitehost

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// uploads.go stores the owner's pictures on local disk and serves them.
//
// Studio's cms/media package defines a media library contract, but ships no
// implementation of it — every host wrote its own. A site owner adding a photo
// of their shop does not need a media library; they need the photo to land on
// the page. So the default host keeps uploads as files in a directory beside
// the site data, named by content hash so a repeated upload of the same photo
// is one file and a crafted filename can never escape the directory.

const (
	uploadsURLPrefix = "/uploads/"
	maxUploadBytes   = 10 << 20
)

// uploadTypes maps the sniffed MIME type to the extension a stored file gets.
// SVG is deliberately absent: an SVG can carry script, and serving one an
// owner uploaded from a page on the same origin is a cross-site scripting
// door nobody asked for.
var uploadTypes = map[string]string{
	"image/png":  ".png",
	"image/jpeg": ".jpg",
	"image/gif":  ".gif",
	"image/webp": ".webp",
}

var uploadName = regexp.MustCompile(`^[a-f0-9]{24}\.(png|jpg|gif|webp)$`)

// UploadDir is where pictures are stored. It defaults to an "uploads" folder
// beside the site's data file.
func (o Options) uploadDir() string {
	if dir := strings.TrimSpace(o.UploadDir); dir != "" {
		return dir
	}
	if o.DataPath == "" {
		return ""
	}
	return filepath.Join(filepath.Dir(o.DataPath), "uploads")
}

var (
	errUploadNotImage = errors.New("not an image")
	errUploadTooLarge = errors.New("too large")
)

// storeUpload writes an image to the upload directory and returns its public
// URL. The stored name is the SHA-256 of the bytes, so the same picture
// uploaded twice is stored once.
func (h *Host) storeUpload(r io.Reader) (string, error) {
	dir := h.opts.uploadDir()
	if dir == "" {
		return "", errors.New("uploads are not configured")
	}
	data, err := io.ReadAll(io.LimitReader(r, maxUploadBytes+1))
	if err != nil {
		return "", err
	}
	if len(data) > maxUploadBytes {
		return "", errUploadTooLarge
	}
	if len(data) == 0 {
		return "", errUploadNotImage
	}
	sniffed := http.DetectContentType(data)
	extension, ok := uploadTypes[sniffed]
	if !ok {
		return "", errUploadNotImage
	}

	sum := sha256.Sum256(data)
	name := hex.EncodeToString(sum[:12]) + extension
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	target := filepath.Join(dir, name)
	if _, err := os.Stat(target); err == nil {
		return uploadsURLPrefix + name, nil
	}
	// Write to a temporary name and rename, so a half-written file is never
	// served.
	temp, err := os.CreateTemp(dir, "upload-*")
	if err != nil {
		return "", err
	}
	if _, err := temp.Write(data); err != nil {
		temp.Close()
		os.Remove(temp.Name())
		return "", err
	}
	if err := temp.Close(); err != nil {
		os.Remove(temp.Name())
		return "", err
	}
	if err := os.Rename(temp.Name(), target); err != nil {
		os.Remove(temp.Name())
		return "", err
	}
	return uploadsURLPrefix + name, nil
}

func (h *Host) mountUploads(mux *http.ServeMux) {
	mux.HandleFunc("GET "+uploadsURLPrefix+"{name}", h.handleServeUpload)
	mux.HandleFunc("POST /admin/api/upload", h.handleUpload)
}

// handleServeUpload serves one stored picture. The name is validated against
// the exact shape storeUpload produces, so nothing outside the directory —
// and nothing that was not put there by an upload — can be read through it.
func (h *Host) handleServeUpload(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if !uploadName.MatchString(name) {
		http.NotFound(w, r)
		return
	}
	dir := h.opts.uploadDir()
	if dir == "" {
		http.NotFound(w, r)
		return
	}
	path := filepath.Join(dir, name)
	file, err := os.Open(path)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil || info.IsDir() {
		http.NotFound(w, r)
		return
	}
	contentType := "application/octet-stream"
	for mime, extension := range uploadTypes {
		if strings.HasSuffix(name, extension) {
			contentType = mime
		}
	}
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("X-Content-Type-Options", "nosniff")
	// Content-addressed: the bytes behind this URL can never change.
	w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	http.ServeContent(w, r, name, info.ModTime(), file)
}

type uploadResult struct {
	OK      bool   `json:"ok"`
	Message string `json:"message,omitempty"`
	URL     string `json:"url,omitempty"`
}

func (h *Host) handleUpload(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxUploadBytes+(1<<20))
	if err := r.ParseMultipartForm(1 << 20); err != nil {
		writeJSON(w, http.StatusOK, uploadResult{Message: "That file is too big. Pictures up to 10 MB work best."})
		return
	}
	file, _, err := r.FormFile("file")
	if err != nil {
		writeJSON(w, http.StatusOK, uploadResult{Message: "Choose a picture to upload."})
		return
	}
	defer file.Close()

	url, err := h.storeUpload(file)
	switch {
	case errors.Is(err, errUploadNotImage):
		writeJSON(w, http.StatusOK, uploadResult{Message: "That doesn't look like a picture. PNG, JPEG, GIF, and WebP work."})
	case errors.Is(err, errUploadTooLarge):
		writeJSON(w, http.StatusOK, uploadResult{Message: "That picture is too big. Pictures up to 10 MB work best."})
	case err != nil:
		writeJSON(w, http.StatusOK, uploadResult{Message: "We couldn't save that picture. Try again."})
	default:
		writeJSON(w, http.StatusOK, uploadResult{OK: true, URL: url})
	}
}
