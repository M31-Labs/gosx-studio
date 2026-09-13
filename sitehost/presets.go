package sitehost

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"m31labs.dev/gosx"
	"m31labs.dev/gosx-admin/blockstudio"
)

// presets.go lets an owner keep a section they got right and drop it onto
// other pages: "Save as a preset" on any block, then "Your presets" in the
// sidebar. A preset is a stored block; inserting one makes a copy.

// Preset is one saved block.
type Preset struct {
	ID      string                    `json:"id"`
	Name    string                    `json:"name"`
	Kind    string                    `json:"kind"`
	Block   blockstudio.BlockInstance `json:"block"`
	Created time.Time                 `json:"created"`
}

type presetStore struct {
	mu      sync.Mutex
	path    string
	loaded  bool
	presets []Preset
}

func newPresetStore(path string) *presetStore { return &presetStore{path: path} }

func (o Options) presetsPath() string {
	if o.DataPath == "" {
		return ""
	}
	return filepath.Join(filepath.Dir(o.DataPath), "presets.json")
}

func (s *presetStore) loadLocked() {
	if s.loaded {
		return
	}
	s.loaded = true
	if s.path == "" {
		return
	}
	raw, err := os.ReadFile(s.path)
	if err != nil {
		return
	}
	var file struct {
		Presets []Preset `json:"presets"`
	}
	if json.Unmarshal(raw, &file) == nil {
		s.presets = file.Presets
	}
}

func (s *presetStore) saveLocked() error {
	if s.path == "" {
		return nil
	}
	raw, err := json.MarshalIndent(struct {
		Presets []Preset `json:"presets"`
	}{s.presets}, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	temp := s.path + ".tmp"
	if err := os.WriteFile(temp, raw, 0o600); err != nil {
		return err
	}
	return os.Rename(temp, s.path)
}

func (s *presetStore) list() []Preset {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.loadLocked()
	out := make([]Preset, len(s.presets))
	copy(out, s.presets)
	return out
}

func (s *presetStore) get(id string) (Preset, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.loadLocked()
	for _, preset := range s.presets {
		if preset.ID == id {
			return preset, true
		}
	}
	return Preset{}, false
}

func (s *presetStore) add(name, kind string, block blockstudio.BlockInstance) (Preset, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.loadLocked()
	preset := Preset{ID: "p" + randomHex(5), Name: name, Kind: kind, Block: block, Created: timeNow().UTC()}
	s.presets = append(s.presets, preset)
	if len(s.presets) > 100 {
		s.presets = s.presets[len(s.presets)-100:]
	}
	return preset, s.saveLocked()
}

func (s *presetStore) remove(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.loadLocked()
	kept := s.presets[:0]
	for _, preset := range s.presets {
		if preset.ID != id {
			kept = append(kept, preset)
		}
	}
	s.presets = kept
	return s.saveLocked()
}

// ---------- routes ----------

type presetRequest struct {
	Name  string             `json:"name"`
	Block editorBlockPayload `json:"block"`
}

type presetResult struct {
	OK      bool   `json:"ok"`
	ID      string `json:"id,omitempty"`
	Name    string `json:"name,omitempty"`
	Kind    string `json:"kind,omitempty"`
	Label   string `json:"label,omitempty"`
	Message string `json:"message,omitempty"`
}

func (h *Host) mountPresets(mux *http.ServeMux) {
	mux.HandleFunc("POST /admin/api/presets", h.handlePresetSave)
	mux.HandleFunc("GET /admin/api/presets/{id}", h.handlePresetFresh)
	mux.HandleFunc("POST /admin/api/presets/{id}/delete", h.handlePresetDelete)
}

func (h *Host) handlePresetSave(w http.ResponseWriter, r *http.Request) {
	var payload presetRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&payload); err != nil {
		writeJSON(w, http.StatusBadRequest, presetResult{Message: "We couldn't read that. Try again."})
		return
	}
	name := strings.TrimSpace(payload.Name)
	if len(name) > 60 {
		name = name[:60]
	}
	if name == "" {
		writeJSON(w, http.StatusOK, presetResult{Message: "Give the preset a name."})
		return
	}
	doc := h.payloadDocument([]editorBlockPayload{payload.Block})
	if len(doc.Blocks) == 0 {
		writeJSON(w, http.StatusOK, presetResult{Message: "There's nothing in that section to keep yet."})
		return
	}
	block := doc.Blocks[0]
	preset, err := h.presets.add(name, editorKind(block.Key), block)
	if err != nil {
		writeJSON(w, http.StatusOK, presetResult{Message: "We couldn't save the preset. Try again."})
		return
	}
	h.auditContent(r, "preset.saved", "Saved the preset “"+name+"”")
	writeJSON(w, http.StatusOK, presetResult{OK: true, ID: preset.ID, Name: preset.Name, Kind: preset.Kind, Label: presetKindLabel(preset.Kind)})
}

// handlePresetFresh renders a copy of the preset for the canvas.
func (h *Host) handlePresetFresh(w http.ResponseWriter, r *http.Request) {
	preset, ok := h.presets.get(r.PathValue("id"))
	if !ok {
		http.NotFound(w, r)
		return
	}
	block := preset.Block
	block.ID = block.Key + "-preset"
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Block-Kind", preset.Kind)
	w.Header().Set("X-Block-Spacing", block.Values[spacingKey].String)
	_, _ = w.Write([]byte(gosx.RenderHTML(h.renderBlockInner(preset.Kind, block))))
}

func (h *Host) handlePresetDelete(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.presets.get(r.PathValue("id")); !ok {
		writeJSON(w, http.StatusNotFound, presetResult{Message: "That preset is already gone."})
		return
	}
	if err := h.presets.remove(r.PathValue("id")); err != nil {
		writeJSON(w, http.StatusOK, presetResult{Message: "We couldn't remove it. Try again."})
		return
	}
	writeJSON(w, http.StatusOK, presetResult{OK: true})
}

// renderPresetAdds is the "Your presets" group in the sidebar.
func (h *Host) renderPresetAdds() gosx.Node {
	presets := h.presets.list()
	items := make([]gosx.Node, 0, len(presets))
	for _, preset := range presets {
		items = append(items, gosx.El("div", gosx.Attrs(gosx.Attr("class", "ed-preset"), gosx.Attr("data-preset", preset.ID)),
			gosx.El("button", gosx.Attrs(gosx.Attr("type", "button"), gosx.Attr("class", "ed-add ed-preset__add"), gosx.Attr("data-add-preset", preset.ID)),
				gosx.El("strong", nil, gosx.Text(preset.Name)), gosx.El("span", nil, gosx.Text(presetKindLabel(preset.Kind)))),
			gosx.El("button", gosx.Attrs(gosx.Attr("type", "button"), gosx.Attr("class", "ed-preset__delete"), gosx.Attr("data-preset-delete", preset.ID), gosx.Attr("title", "Forget this preset"), gosx.Attr("aria-label", "Forget the preset "+preset.Name)), gosx.Text("✕"))))
	}
	hint := "Save any section with the ★ tool and it appears here, ready to drop onto another page."
	if len(items) > 0 {
		hint = "A copy of a section you saved. Change the copy without touching the preset."
	}
	return gosx.El("div", gosx.Attrs(gosx.Attr("class", "ed-side__block"), gosx.Attr("data-presets", "true")),
		gosx.El("h2", nil, gosx.Text("Your presets")),
		gosx.El("p", gosx.Attrs(gosx.Attr("class", "ed-hint"), gosx.Attr("data-presets-hint", "true")), gosx.Text(hint)),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "ed-add-grid"), gosx.Attr("data-presets-list", "true")), gosx.Fragment(items...)))
}

func presetKindLabel(kind string) string {
	if spec, ok := compositeByKey(kind); ok {
		return spec.Label
	}
	labels := map[string]string{"heading": "Heading", "paragraph": "Text", "quote": "Quote", "button": "Button", "image": "Picture", "gallery": "Gallery", "video": "Video", "columns": "Columns", "list": "List", "divider": "Divider", "section": "Section break", "form": "Form", "product": "Product"}
	return firstNonEmpty(labels[kind], kind)
}
