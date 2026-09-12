package sitehost

import (
	"bytes"
	"encoding/json"
	"errors"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/image/draw"
	_ "golang.org/x/image/webp" // decode config for WebP uploads

	"m31labs.dev/gosx"
	"m31labs.dev/gosx-studio/cms/render"
	cmsstore "m31labs.dev/gosx-studio/cms/store"
	"m31labs.dev/gosx-studio/internal/mediaurl"
)

// media.go is the picture library and the responsive-image pipeline.
//
// Studio's cms/media declares a VariantGenerator and ships no implementation,
// and cms/render emitted <img src alt> with no dimensions, no srcset, and no
// lazy loading — every image was a layout-shift and a slow-load waiting to
// happen. Here every upload is measured, resized into a few widths, and
// recorded in a small index; the render hook turns that into an <img> a
// browser can lay out before the bytes arrive and fetch at the right size.

var variantWidths = []int{480, 960, 1600}

const (
	mediaIndexFile = "index.json"
	mediaSizes     = "(max-width: 720px) 100vw, 720px"
)

type mediaVariant struct {
	Name   string `json:"name"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
}

// mediaEntry is one uploaded picture and what was made from it.
type mediaEntry struct {
	Name        string         `json:"name"`
	Width       int            `json:"width"`
	Height      int            `json:"height"`
	Size        int64          `json:"size"`
	ContentType string         `json:"contentType"`
	Uploaded    time.Time      `json:"uploaded"`
	Variants    []mediaVariant `json:"variants,omitempty"`
}

func (e mediaEntry) url() string { return uploadsURLPrefix + e.Name }

// thumb is the smallest rendition, for the library and the picker.
func (e mediaEntry) thumb() string {
	if len(e.Variants) > 0 {
		return uploadsURLPrefix + e.Variants[0].Name
	}
	return e.url()
}

type mediaIndex struct {
	mu      sync.Mutex
	dir     string
	entries map[string]mediaEntry
	loaded  bool
}

func newMediaIndex(dir string) *mediaIndex {
	return &mediaIndex{dir: dir, entries: map[string]mediaEntry{}}
}

func (m *mediaIndex) path() string { return filepath.Join(m.dir, mediaIndexFile) }

func (m *mediaIndex) loadLocked() error {
	if m.loaded || m.dir == "" {
		m.loaded = true
		return nil
	}
	m.loaded = true
	data, err := os.ReadFile(m.path())
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	var list []mediaEntry
	if err := json.Unmarshal(data, &list); err != nil {
		return err
	}
	for _, entry := range list {
		m.entries[entry.Name] = entry
	}
	return nil
}

func (m *mediaIndex) saveLocked() error {
	if m.dir == "" {
		return nil
	}
	list := make([]mediaEntry, 0, len(m.entries))
	for _, entry := range m.entries {
		list = append(list, entry)
	}
	sort.Slice(list, func(i, j int) bool { return list[i].Uploaded.After(list[j].Uploaded) })
	data, err := json.MarshalIndent(list, "", "  ")
	if err != nil {
		return err
	}
	temp := m.path() + ".tmp"
	if err := os.WriteFile(temp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(temp, m.path())
}

func (m *mediaIndex) get(name string) (mediaEntry, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := m.loadLocked(); err != nil {
		return mediaEntry{}, false
	}
	entry, ok := m.entries[name]
	return entry, ok
}

func (m *mediaIndex) list() []mediaEntry {
	m.mu.Lock()
	defer m.mu.Unlock()
	_ = m.loadLocked()
	out := make([]mediaEntry, 0, len(m.entries))
	for _, entry := range m.entries {
		out = append(out, entry)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Uploaded.After(out[j].Uploaded) })
	return out
}

// ensure records an upload and produces its variants, once.
func (m *mediaIndex) ensure(name string, data []byte, contentType string) (mediaEntry, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := m.loadLocked(); err != nil {
		return mediaEntry{}, err
	}
	if entry, ok := m.entries[name]; ok {
		return entry, nil
	}
	entry := mediaEntry{Name: name, Size: int64(len(data)), ContentType: contentType, Uploaded: time.Now().UTC()}
	config, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err == nil {
		entry.Width, entry.Height = config.Width, config.Height
		entry.Variants = m.makeVariants(name, data, format, config)
	}
	m.entries[name] = entry
	return entry, m.saveLocked()
}

// makeVariants writes smaller renditions for PNG and JPEG sources. GIF may
// animate and WebP has no encoder in the standard library, so both are
// served as uploaded.
func (m *mediaIndex) makeVariants(name string, data []byte, format string, config image.Config) []mediaVariant {
	if format != "png" && format != "jpeg" {
		return nil
	}
	src, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil
	}
	extension := filepath.Ext(name)
	base := strings.TrimSuffix(name, extension)
	variants := make([]mediaVariant, 0, len(variantWidths))
	for _, width := range variantWidths {
		if width >= config.Width {
			continue
		}
		height := int(float64(config.Height) * float64(width) / float64(config.Width))
		if height < 1 {
			height = 1
		}
		dst := image.NewRGBA(image.Rect(0, 0, width, height))
		draw.CatmullRom.Scale(dst, dst.Bounds(), src, src.Bounds(), draw.Over, nil)
		var buf bytes.Buffer
		switch format {
		case "jpeg":
			err = jpeg.Encode(&buf, dst, &jpeg.Options{Quality: 82})
		default:
			err = png.Encode(&buf, dst)
		}
		if err != nil {
			continue
		}
		variantName := base + "-w" + strconv.Itoa(width) + extension
		if err := os.WriteFile(filepath.Join(m.dir, variantName), buf.Bytes(), 0o644); err != nil {
			continue
		}
		variants = append(variants, mediaVariant{Name: variantName, Width: width, Height: height})
	}
	return variants
}

// remove deletes a picture and its variants from disk and the index.
func (m *mediaIndex) remove(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := m.loadLocked(); err != nil {
		return err
	}
	entry, ok := m.entries[name]
	if !ok {
		return errors.New("not found")
	}
	for _, variant := range entry.Variants {
		_ = os.Remove(filepath.Join(m.dir, variant.Name))
	}
	_ = os.Remove(filepath.Join(m.dir, entry.Name))
	delete(m.entries, name)
	return m.saveLocked()
}

var _ = gif.Decode // gif stays registered for DecodeConfig

// ---------- rendering ----------

// imageHook renders image blocks with everything a browser wants: intrinsic
// size, a srcset of the variants, lazy loading. Pictures that were pasted as
// links (not uploaded) get lazy loading and nothing else, because nothing is
// known about them.
func (h *Host) imageHook() render.Hook {
	return func(ctx render.Context) (gosx.Node, bool) {
		rawURL := strings.TrimSpace(ctx.Ref)
		alt, _ := ctx.Block["alt"].(string)
		safeURL, ok := mediaurl.ForImage(rawURL)
		if !ok {
			return gosx.Fragment(), false
		}
		attrs := []any{
			gosx.Attr("src", safeURL),
			gosx.Attr("alt", alt),
			gosx.Attr("loading", "lazy"),
			gosx.Attr("decoding", "async"),
		}
		if strings.HasPrefix(safeURL, uploadsURLPrefix) {
			if entry, found := h.media.get(strings.TrimPrefix(safeURL, uploadsURLPrefix)); found && entry.Width > 0 {
				attrs = append(attrs, gosx.Attr("width", strconv.Itoa(entry.Width)), gosx.Attr("height", strconv.Itoa(entry.Height)))
				if len(entry.Variants) > 0 {
					parts := make([]string, 0, len(entry.Variants)+1)
					for _, variant := range entry.Variants {
						parts = append(parts, uploadsURLPrefix+variant.Name+" "+strconv.Itoa(variant.Width)+"w")
					}
					parts = append(parts, entry.url()+" "+strconv.Itoa(entry.Width)+"w")
					attrs = append(attrs, gosx.Attr("srcset", strings.Join(parts, ", ")), gosx.Attr("sizes", mediaSizes))
				}
			}
		}
		return gosx.El("figure", nil, gosx.El("img", gosx.Attrs(attrs...))), true
	}
}

// ---------- the library ----------

func (h *Host) mountMedia(mux *http.ServeMux) {
	mux.HandleFunc("GET /admin/media", h.handleAdminMedia)
	mux.HandleFunc("GET /admin/media/{$}", h.handleAdminMedia)
	mux.HandleFunc("POST /admin/media/{name}/delete", h.handleAdminMediaDelete)
	mux.HandleFunc("GET /admin/api/media", h.handleMediaAPI)
}

// mediaUsage counts the pages whose content references each picture.
func (h *Host) mediaUsage() map[string]int {
	usage := map[string]int{}
	pages, err := h.store.ListPages(cmsstore.PageFilter{})
	if err != nil {
		return usage
	}
	for _, page := range pages {
		seen := map[string]bool{}
		for _, block := range page.Body.Blocks {
			for _, value := range block.Values {
				if url := strings.TrimSpace(value.String); strings.HasPrefix(url, uploadsURLPrefix) && !seen[url] {
					seen[url] = true
					usage[strings.TrimPrefix(url, uploadsURLPrefix)]++
				}
			}
		}
		for _, key := range []string{"metaImageUrl"} {
			if url := strings.TrimSpace(page.Metadata[key]); strings.HasPrefix(url, uploadsURLPrefix) && !seen[url] {
				usage[strings.TrimPrefix(url, uploadsURLPrefix)]++
			}
		}
	}
	brand := h.brand()
	for _, url := range []string{brand.LogoURL, brand.FaviconURL} {
		if strings.HasPrefix(url, uploadsURLPrefix) {
			usage[strings.TrimPrefix(url, uploadsURLPrefix)]++
		}
	}
	return usage
}

func humanSize(size int64) string {
	switch {
	case size >= 1<<20:
		return strconv.FormatFloat(float64(size)/(1<<20), 'f', 1, 64) + " MB"
	case size >= 1<<10:
		return strconv.FormatInt(size/(1<<10), 10) + " KB"
	default:
		return strconv.FormatInt(size, 10) + " B"
	}
}

func (h *Host) handleAdminMedia(w http.ResponseWriter, r *http.Request) {
	entries := h.media.list()
	usage := h.mediaUsage()
	status := adminStatus{Message: r.URL.Query().Get("status")}

	var listing gosx.Node
	if len(entries) == 0 {
		listing = gosx.El("section", gosx.Attrs(gosx.Attr("class", "admin-empty")),
			gosx.El("h2", nil, gosx.Text("No pictures yet")),
			gosx.El("p", nil, gosx.Text("Pictures you upload while editing a page collect here, so you can use them again or tidy up. Add one from any page's Image block.")),
		)
	} else {
		cards := make([]gosx.Node, 0, len(entries))
		for _, entry := range entries {
			used := usage[entry.Name]
			usedText := "Not used on any page"
			if used == 1 {
				usedText = "Used on 1 page"
			} else if used > 1 {
				usedText = "Used on " + strconv.Itoa(used) + " pages"
			}
			deleteLabel := "Delete"
			if used > 0 {
				deleteLabel = "Delete anyway"
			}
			cards = append(cards, gosx.El("article", gosx.Attrs(gosx.Attr("class", "admin-media")),
				gosx.El("img", gosx.Attrs(gosx.Attr("class", "admin-media__thumb"), gosx.Attr("src", entry.thumb()), gosx.Attr("alt", ""), gosx.Attr("loading", "lazy"))),
				gosx.El("div", gosx.Attrs(gosx.Attr("class", "admin-media__meta")),
					gosx.El("span", nil, gosx.Text(strconv.Itoa(entry.Width)+" × "+strconv.Itoa(entry.Height)+" · "+humanSize(entry.Size))),
					gosx.El("span", gosx.Attrs(gosx.Attr("class", "admin-media__usage")), gosx.Text(usedText)),
					gosx.El("input", gosx.Attrs(gosx.Attr("class", "admin-media__link"), gosx.Attr("type", "text"), gosx.Attr("readonly", "readonly"), gosx.Attr("value", entry.url()), gosx.Attr("aria-label", "Picture link"))),
				),
				gosx.El("form", gosx.Attrs(gosx.Attr("method", "post"), gosx.Attr("action", "/admin/media/"+entry.Name+"/delete"), gosx.Attr("class", "admin-inline-form")),
					h.csrfField(),
					gosx.El("button", gosx.Attrs(gosx.Attr("class", "admin-row-btn"), gosx.Attr("type", "submit"), gosx.Attr("data-action", "archive")), gosx.Text(deleteLabel)),
				),
			))
		}
		listing = gosx.El("div", gosx.Attrs(gosx.Attr("class", "admin-media-grid")), gosx.Fragment(cards...))
	}

	body := h.renderAdminShell("media", "Pictures",
		"Everything you've uploaded, newest first. Deleting a picture that a page still uses leaves a gap on that page.",
		status, listing)
	h.writeDocument(w, http.StatusOK, h.adminMeta("Pictures"), body)
}

func (h *Host) handleAdminMediaDelete(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if !uploadName.MatchString(name) || strings.Contains(name, "-w") {
		http.Redirect(w, r, "/admin/media?status="+queryEscape("We couldn't find that picture."), http.StatusSeeOther)
		return
	}
	if err := h.media.remove(name); err != nil {
		http.Redirect(w, r, "/admin/media?status="+queryEscape("We couldn't find that picture."), http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/admin/media?status="+queryEscape("Picture deleted."), http.StatusSeeOther)
}

type mediaAPIEntry struct {
	URL    string `json:"url"`
	Thumb  string `json:"thumb"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
}

func (h *Host) handleMediaAPI(w http.ResponseWriter, r *http.Request) {
	entries := h.media.list()
	out := make([]mediaAPIEntry, 0, len(entries))
	for _, entry := range entries {
		out = append(out, mediaAPIEntry{URL: entry.url(), Thumb: entry.thumb(), Width: entry.Width, Height: entry.Height})
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "pictures": out})
}
