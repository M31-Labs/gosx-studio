package sitehost

import (
	"net/http"
	"net/url"
	"strings"
	"testing"
)

func TestOwnersGiveAnyBlockMoreOrLessRoom(t *testing.T) {
	host, handler := newTestHost(t)
	id := firstPageID(t, host, "menu")
	payload := `{"title":"Menu","slug":"menu","description":"x","blocks":[
		{"kind":"paragraph","text":"Tight one","spacing":"tight"},
		{"kind":"heading","text":"Roomy one","level":"2","spacing":"roomy"},
		{"kind":"paragraph","text":"Odd one","spacing":"huge"},
		{"kind":"cta","fields":{"headline":"Come by"},"spacing":"extra"}
	]}`
	mustContain(t, postJSON(t, handler, "/admin/api/pages/"+id, payload).Body.String(), `"ok":true`, "the save works")
	page, _, _ := host.Store().PageByID(id)
	if got := page.Body.Blocks[0].Values[spacingKey].String; got != "tight" {
		t.Fatalf("spacing stored = %q", got)
	}
	if got := page.Body.Blocks[2].Values[spacingKey].String; got != "" {
		t.Fatalf("an unknown spacing is dropped, got %q", got)
	}
	post(t, handler, "/admin/api/pages/"+id+"/publish", url.Values{})
	public := get(t, handler, "/menu").Body.String()
	mustContain(t, public, `<div class="site-space site-space--tight"><p>Tight one</p></div>`, "the public page wraps a tight block")
	mustContain(t, public, `<div class="site-space site-space--extra"><div class="site-cta`, "a section gets extra room too")
	if strings.Contains(public, `site-space--huge`) {
		t.Fatal("an unknown spacing must not reach the page")
	}
	editor := get(t, handler, "/admin/edit/"+id).Body.String()
	mustContain(t, editor, `class="ed-block site-space--roomy" data-block="heading" data-index="1" tabindex="0" data-spacing="roomy"`, "the canvas carries the spacing")
	mustContain(t, editor, `<select class="ed-tool ed-tool--select" data-tool-spacing="true"`, "every block has the spacing tool")
	mustContain(t, editor, `<option value="roomy" selected="selected">Roomy</option>`, "the tool shows the current choice")
	css := get(t, handler, publicStylesheetPath).Body.String()
	mustContain(t, css, `.site-space.site-space--extra > * { margin-block: 96px !important; }`, "the page styles the spacing")
}

func TestRepeatedItemsAndGalleryPicturesCanBeReordered(t *testing.T) {
	host, handler := newTestHost(t)
	id := firstPageID(t, host, "menu")
	payload := `{"title":"Menu","slug":"menu","description":"x","blocks":[
		{"kind":"features","fields":{"heading":"Why"},"items":[{"icon":"","title":"One","text":"a"},{"icon":"","title":"Two","text":"b"}]},
		{"kind":"gallery","images":[{"url":"/uploads/a.png","alt":"A"},{"url":"/uploads/b.png","alt":"B"}]}
	]}`
	mustContain(t, postJSON(t, handler, "/admin/api/pages/"+id, payload).Body.String(), `"ok":true`, "the save works")
	editor := get(t, handler, "/admin/edit/"+id).Body.String()
	mustContain(t, editor, `<span class="ed-item__tools" contenteditable="false"><button type="button" class="ed-item__tool" data-item-grab="true" title="Drag to reorder" aria-label="Drag to reorder">⠿</button><button type="button" class="ed-item__tool" data-item-up="true" title="Move up" aria-label="Move up">↑</button><button type="button" class="ed-item__tool" data-item-down="true" title="Move down" aria-label="Move down">↓</button><button type="button" class="ed-item__tool" data-item-remove="true" title="Remove" aria-label="Remove">✕</button></span>`, "each card has grab, arrows, and remove")
	mustContain(t, editor, `<figure class="site-gallery__item ed-gallery__item ed-item" data-item="true"><span class="ed-item__tools"`, "gallery pictures are items too")
	if strings.Contains(editor, "data-gremove") {
		t.Fatal("the old gallery remove button is gone")
	}
	post(t, handler, "/admin/api/pages/"+id+"/publish", url.Values{})
	public := get(t, handler, "/menu").Body.String()
	if strings.Contains(public, "ed-item") {
		t.Fatal("item tools never reach the public page")
	}
	js := get(t, handler, editorScriptPath).Body.String()
	for _, want := range []string{`[data-item-up],[data-item-down]`, `data-item-grab`, `elementFromPoint`, `ed-item--dragging`, `event.key.toLowerCase() === "s"`} {
		mustContain(t, js, want, "the editor script reorders items and saves with Ctrl+S")
	}
}

func TestOwnersKeepASectionAsAPresetAndDropItOnAnotherPage(t *testing.T) {
	host, handler := newTestHost(t)
	id := firstPageID(t, host, "menu")
	editor := get(t, handler, "/admin/edit/"+id).Body.String()
	mustContain(t, editor, `<h2>Your presets</h2>`, "the sidebar has a presets group")
	mustContain(t, editor, `Save any section with the ★ tool`, "the empty group explains itself")
	mustContain(t, editor, `data-tool="preset"`, "every block has the save tool")

	rec := postJSON(t, handler, "/admin/api/presets", `{"name":"  Our promise  ","block":{"kind":"cta","variant":"band","fields":{"headline":"Come by","text":"We open at seven.","button":"Find us","url":"/contact"},"spacing":"roomy"}}`)
	body := rec.Body.String()
	mustContain(t, body, `"ok":true`, "the preset saves")
	mustContain(t, body, `"name":"Our promise"`, "the name is trimmed")
	mustContain(t, body, `"kind":"cta"`, "the kind comes back")
	presets := host.presets.list()
	if len(presets) != 1 || presets[0].Block.Key != "cta" || presets[0].Block.Values[spacingKey].String != "roomy" {
		t.Fatalf("stored presets: %+v", presets)
	}
	presetID := presets[0].ID

	mustContain(t, postJSON(t, handler, "/admin/api/presets", `{"name":"","block":{"kind":"paragraph","text":"x"}}`).Body.String(), `Give the preset a name.`, "a preset needs a name")
	mustContain(t, postJSON(t, handler, "/admin/api/presets", `{"name":"Empty","block":{"kind":"bogus"}}`).Body.String(), `nothing in that section`, "an unknown block is refused")

	editor = get(t, handler, "/admin/edit/"+id).Body.String()
	mustContain(t, editor, `data-add-preset="`+presetID+`"><strong>Our promise</strong><span>Call to action</span>`, "the sidebar lists the preset")
	mustContain(t, editor, `data-preset-delete="`+presetID+`"`, "the preset can be forgotten")

	fresh := get(t, handler, "/admin/api/presets/"+presetID)
	if fresh.Code != http.StatusOK || fresh.Header().Get("X-Block-Kind") != "cta" || fresh.Header().Get("X-Block-Spacing") != "roomy" {
		t.Fatalf("fresh preset: %d %v", fresh.Code, fresh.Header())
	}
	mustContain(t, fresh.Body.String(), `class="site-cta site-cta--band`, "the copy renders the saved section")
	mustContain(t, fresh.Body.String(), `data-field="headline"`, "the copy is editable")
	if get(t, handler, "/admin/api/presets/nope").Code != http.StatusNotFound {
		t.Fatal("an unknown preset is not found")
	}

	mustContain(t, postJSON(t, handler, "/admin/api/presets/"+presetID+"/delete", `{}`).Body.String(), `"ok":true`, "the delete works")
	if len(host.presets.list()) != 0 {
		t.Fatal("the preset is gone")
	}
	mustContain(t, postJSON(t, handler, "/admin/api/presets/"+presetID+"/delete", `{}`).Body.String(), `already gone`, "a second delete says so")

	reopened := newPresetStore(host.presets.path)
	if len(reopened.list()) != 0 {
		t.Fatal("the store persists")
	}
}

func TestPresetsNeedAnEditorSession(t *testing.T) {
	_, handler := newGuardedHost(t)
	rec := postJSON(t, handler, "/admin/api/presets", `{"name":"x","block":{"kind":"paragraph","text":"x"}}`)
	if rec.Code == http.StatusOK && strings.Contains(rec.Body.String(), `"ok":true`) {
		t.Fatalf("an anonymous request saved a preset: %d %s", rec.Code, rec.Body.String())
	}
}

func TestPublicPagesCarryEnterpriseGradePolish(t *testing.T) {
	host, handler := newTestHost(t)
	id := firstPageID(t, host, "menu")
	payload := `{"title":"Menu","slug":"menu","description":"x","blocks":[
		{"kind":"faq","fields":{"heading":"Questions"},"items":[{"question":"Do you **deliver**?","answer":"Within the city, yes."},{"question":"Gluten free?","answer":""}]}
	]}`
	mustContain(t, postJSON(t, handler, "/admin/api/pages/"+id, payload).Body.String(), `"ok":true`, "the save works")
	post(t, handler, "/admin/api/pages/"+id+"/publish", url.Values{})
	public := get(t, handler, "/menu").Body.String()
	mustContain(t, public, `<a class="site-skip" href="#main">Skip to content</a>`, "keyboard users can skip the header")
	mustContain(t, public, `<meta name="theme-color" content="`, "the browser chrome takes the site's ground colour")
	mustContain(t, public, `"@type":"FAQPage"`, "questions become rich results")
	mustContain(t, public, `{"@type":"Question","acceptedAnswer":{"@type":"Answer","text":"Within the city, yes."},"name":"Do you deliver?"}`, "the question text is plain")
	if strings.Contains(public, `"name":"Gluten free?"`) {
		t.Fatal("a question without an answer is left out")
	}
	admin := get(t, handler, "/admin/edit/"+id).Body.String()
	if strings.Contains(admin, `name="theme-color"`) {
		t.Fatal("the editor keeps its own chrome")
	}
	css := get(t, handler, publicStylesheetPath).Body.String()
	for _, want := range []string{
		`.site-skip:focus, .site-skip:focus-visible { top: 12px;`, `.gosx-site--public :focus-visible { outline: 2px solid var(--site-accent);`,
		`.gosx-site--public [id] { scroll-margin-top: 96px; }`, `html { scroll-behavior: smooth; }`, `@media (prefers-reduced-motion: reduce)`,
		`@media print {`, `.site-button:hover, .site-article .button:hover { filter: brightness(1.06);`, `@keyframes site-fade`,
	} {
		mustContain(t, css, want, "the stylesheet carries the polish")
	}
}
