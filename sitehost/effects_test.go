package sitehost

import (
	"net/http"
	"net/url"
	"strings"
	"testing"
)

func TestBandsCarryABackdropAndDepthToThePage(t *testing.T) {
	host, handler := newTestHost(t)
	id := firstPageID(t, host, "menu")
	payload := `{"title":"Menu","slug":"menu","description":"x","blocks":[
		{"kind":"section","style":"dark","fx":"orbs","motion":"slow","intensity":"bold","seed":"Loaf-42!","depth":"tilt"},
		{"kind":"features","fields":{"heading":"Why us"},"items":[{"icon":"","title":"Fresh","text":"Every morning."}]},
		{"kind":"section","style":"plain","fx":"bogus","motion":"fast"},
		{"kind":"paragraph","text":"After a plain break."}
	]}`
	mustContain(t, postJSON(t, handler, "/admin/api/pages/"+id, payload).Body.String(), `"ok":true`, "the save works")
	page, _, _ := host.Store().PageByID(id)
	options := sectionOptionsOf(page.Body.Blocks[0])
	if options.Effect != "orbs" || options.Motion != "slow" || options.Intensity != "bold" || options.Seed != "Loaf-42" || options.Depth != "tilt" {
		t.Fatalf("stored options: %+v", options)
	}
	if plain := sectionOptionsOf(page.Body.Blocks[2]); plain.Effect != "none" || plain.Motion != "normal" {
		t.Fatalf("unknown choices fall back: %+v", plain)
	}
	post(t, handler, "/admin/api/pages/"+id+"/publish", url.Values{})
	public := get(t, handler, "/menu").Body.String()
	mustContain(t, public, `<section class="site-section site-section--dark site-fx-host site-tilt" data-fx="orbs" data-fx-motion="slow" data-fx-intensity="bold" data-fx-seed="Loaf-42" data-tilt="true"><canvas class="site-fx" aria-hidden="true"></canvas>`, "the band is a host with its canvas first")
	mustContain(t, public, `<section class="site-section site-section--plain"><div class="site-section__inner">`, "a band without a backdrop is unchanged")
	mustContain(t, public, `data-site-motion="full"`, "the body carries the site's motion setting")
	mustContain(t, public, `src="`+effectsScriptURL+`"`, "and loads the engine")
	agent := agentCall(t, handler, http.MethodGet, "/agent/v1/pages/menu", agentKeyFor(t, host, "read"), "").Body.String()
	mustContain(t, agent, `"fx":"orbs","motion":"slow","intensity":"bold","seed":"Loaf-42","depth":"tilt"`, "the agent API reads the backdrop back")
}

func TestHeroesTakeABackdropWithControlsInTheEditor(t *testing.T) {
	host, handler := newTestHost(t)
	id := firstPageID(t, host, "menu")
	payload := `{"title":"Menu","slug":"menu","description":"x","blocks":[
		{"kind":"hero","variant":"poster","fields":{"headline":"Big type","effect":"aurora","motion":"still","intensity":"subtle","seed":"abc"}},
		{"kind":"hero","variant":"stage","fields":{"headline":"On a floor"}},
		{"kind":"hero","variant":"center","fields":{"headline":"Plain","effect":"nonsense"}}
	]}`
	mustContain(t, postJSON(t, handler, "/admin/api/pages/"+id, payload).Body.String(), `"ok":true`, "the save works")
	page, _, _ := host.Store().PageByID(id)
	if got := page.Body.Blocks[0].Values["effect"].String; got != "aurora" {
		t.Fatalf("effect stored = %q", got)
	}
	if got := page.Body.Blocks[2].Values["effect"].String; got != "" {
		t.Fatalf("an unknown effect is dropped, got %q", got)
	}
	post(t, handler, "/admin/api/pages/"+id+"/publish", url.Values{})
	public := get(t, handler, "/menu").Body.String()
	mustContain(t, public, `class="site-hero site-hero--poster site-hero--fx site-fx-host" data-fx="aurora" data-fx-motion="still" data-fx-intensity="subtle" data-fx-seed="abc"><canvas class="site-fx" aria-hidden="true"></canvas>`, "a poster hero hosts its backdrop")
	mustContain(t, public, `class="site-hero site-hero--stage site-hero--fx site-fx-host" data-fx="grid"`, "a stage hero stands on a grid by default")
	mustContain(t, public, `<div class="site-hero site-hero--center"><div class="site-hero__copy">`, "a hero without a backdrop is unchanged")
	if strings.Contains(public, `data-fx-field`) {
		t.Fatal("the controls never reach the page")
	}
	editor := get(t, handler, "/admin/edit/"+id).Body.String()
	mustContain(t, editor, `<select data-field="effect" data-fx-field="effect" aria-label="Backdrop">`, "the hero has a backdrop select")
	mustContain(t, editor, `<option value="aurora" selected="selected">Aurora</option>`, "showing the current one")
	mustContain(t, editor, `<input type="hidden" data-field="seed" data-fx-seed="true" value="abc" />`, "the seed rides along")
	mustContain(t, editor, `data-shuffle="true"`, "with a shuffle")
	mustContain(t, editor, `data-site-motion="full"`, "the canvas knows the site's motion")
}

func TestTheSectionBarOffersBackdropsWithALivePreview(t *testing.T) {
	host, handler := newTestHost(t)
	id := firstPageID(t, host, "menu")
	postJSON(t, handler, "/admin/api/pages/"+id, `{"title":"Menu","slug":"menu","description":"x","blocks":[{"kind":"section","style":"tinted","fx":"waves","seed":"tide"},{"kind":"paragraph","text":"x"}]}`)
	editor := get(t, handler, "/admin/edit/"+id).Body.String()
	for _, want := range []string{
		`<select data-section-fx="true" aria-label="Backdrop">`, `<option value="waves" selected="selected">Waves</option>`, `data-section-motion="true"`, `data-section-intensity="true"`, `data-section-depth="true"`,
		`<input type="hidden" data-section-seed="true" value="tide" />`, `data-section-shuffle="true"`,
		`class="ed-fx-preview" data-fx-preview="true" data-fx="waves" data-fx-motion="normal" data-fx-intensity="normal" data-fx-seed="tide"`,
	} {
		mustContain(t, editor, want, "the section bar carries the backdrop controls")
	}
	js := get(t, handler, editorScriptPath).Body.String()
	for _, want := range []string{"function applySectionFx", "function applyHeroFx", "data-section-shuffle", "window.gosxFX.rescan(article)", `if (node.tagName === "SELECT") return node.value;`} {
		mustContain(t, js, want, "the editor script drives the backdrops live")
	}
	engine := get(t, handler, effectsScriptPath)
	mustContain(t, engine.Header().Get("Content-Type"), "javascript", "the engine is served")
	for _, want := range []string{"effects.aurora", "effects.particles", "effects.waves", "effects.orbs", "effects.grid", "effects.stars", "effects.topo", "effects.ribbons", "prefers-reduced-motion", "data-site-motion", "IntersectionObserver", "window.gosxFX"} {
		mustContain(t, engine.Body.String(), want, "the engine has every effect and its guards")
	}
	css := get(t, handler, publicStylesheetPath).Body.String()
	for _, want := range []string{".site-fx-host { position: relative; isolation: isolate; }", ".site-fx { position: absolute; inset: 0;", ".site-hero--poster", ".site-hero--stage", ".site-tilt__card { transition"} {
		mustContain(t, css, want, "the stylesheet has the backdrop rules")
	}
}

func TestTheLookHasAMotionPreference(t *testing.T) {
	host, handler := newTestHost(t)
	id := firstPageID(t, host, "menu")
	editor := get(t, handler, "/admin/edit/"+id).Body.String()
	mustContain(t, editor, `data-look-motion="calm"`, "the Look offers a calm setting")
	mustContain(t, postJSON(t, handler, "/admin/api/theme", `{"palette":"warm","fonts":"classic","motion":"calm"}`).Body.String(), `"ok":true`, "the Look saves")
	if got := host.theme().Motion; got != "calm" {
		t.Fatalf("motion = %q", got)
	}
	mustContain(t, get(t, handler, "/").Body.String(), `data-site-motion="calm"`, "every page carries it")
	token := agentKeyFor(t, host, "read", "settings")
	mustContain(t, agentCall(t, handler, http.MethodGet, "/agent/v1/look", token, "").Body.String(), `"motion":"calm"`, "agents read it")
	mustContain(t, agentCall(t, handler, http.MethodPut, "/agent/v1/look", token, `{"motion":"off"}`).Body.String(), `"motion":"off"`, "and set it")
	if rec := agentCall(t, handler, http.MethodPut, "/agent/v1/look", token, `{"motion":"wild"}`); !strings.Contains(rec.Body.String(), `"motion":"full"`) {
		t.Fatalf("an unknown motion falls back: %s", rec.Body.String())
	}
}

func TestNewSitesGetTheirOwnBackdrop(t *testing.T) {
	for kind, want := range map[string]string{"food": "aurora", "services": "topo", "shop": "orbs", "portfolio": "stars", "community": "waves", "other": "particles"} {
		pages := StarterSiteFor(SetupAnswers{SiteTitle: "Orbit " + kind, Kind: kind})
		hero := pages[0].Body.Blocks[0]
		if hero.Key != "hero" || hero.Values["effect"].String != want || hero.Values["motion"].String != "slow" || hero.Values["intensity"].String != "subtle" {
			t.Fatalf("%s hero: %+v", kind, hero.Values)
		}
		if seed := hero.Values["seed"].String; seed == "" || seed != fxSeedFor("Orbit "+kind) {
			t.Fatalf("%s seed = %q", kind, seed)
		}
	}
	if fxSeedFor("Mill Lane Bakery") == fxSeedFor("Mill Lane Cafe") {
		t.Fatal("different names, different seeds")
	}
	schema := decodeJSON(t, agentCall(t, newTestHostHandler(t), http.MethodGet, "/agent/v1/schema", "", ""))
	backdrops := schema["backdrops"].(map[string]any)
	if effects := backdrops["effects"].([]any); len(effects) != 9 || effects[1] != "aurora" {
		t.Fatalf("schema backdrops: %v", effects)
	}
}

func newTestHostHandler(t *testing.T) http.Handler {
	_, handler := newTestHost(t)
	return handler
}
