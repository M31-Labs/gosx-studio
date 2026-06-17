package studio

import (
	"m31labs.dev/gosx"
)

// RenderStylePanel renders a brand color-palette editor as a managed authoring
// form that persists the whole palette in one Save through the existing
// draft→publish pipeline. It is designed to be embedded in a host's style or
// brand inspector — it emits no visible platform copy and is host-brandable.
//
// The form element (id=formID) carries the host authoring route as its action and
// data-gosx-studio-authoring-managed="true" so authoringruntime's island_runtime.js
// intercepts the submit via fetch+JSON and POSTs to that action, then calls
// handleResult (preview refresh, save chrome). This matches the existing managed-form
// pattern: the runtime checks form.hasAttribute(MANAGED_FORM_ATTR) and reads form.action.
//
// PLACEMENT: because this emits a <form>, the host MUST render the panel OUTSIDE the
// editor's workbench <form> — HTML forbids nested forms (a nested <form> is dropped by
// the parser and the managed marker lost). Put it in the host's external
// managed-forms region, where the site-map authoring forms already live.
//
// view["palette"] must be []map[string]any or []map[string]string with entries:
//
//	{
//	  "key":      "accent",          // logical token key → data-editor-color-key
//	  "name":     "colorAccent",     // form field name submitted to the host
//	  "label":    "Accent",          // display label
//	  "cssVar":   "--color-accent",  // → data-editor-color-token (MUST be CSS var)
//	  "value":    "#b5651d",         // current hex
//	  "fallback": "#b5651d",         // preset default for reset-to-kit
//	}
//
// The live-preview contract: styleruntime's bindTheme reads [data-editor-color-token]
// and publishes $preview.theme.colors keyed by the attribute VALUE = the CSS variable.
// The preview subscriber sets that exact CSS property on the iframe. Each swatch input
// MUST carry data-editor-color-token=<cssVar>. If the field name lands there instead,
// live preview silently breaks.
//
// The host adapter receives color values in mutation.RawForm (e.g.
// mutation.RawForm["colorAccent"]) because save-appearance uses host-defined field
// names rather than standard authoring fields.
//
// Fonts are the fast follow: styleruntime's bindFontsIsland + $preview.theme.fontsCustom
// already exist for the next panel slice.
func RenderStylePanel(view map[string]any, formID, action, csrfToken string) gosx.Node {
	palette := stylePaletteEntries(view)

	// Empty or nil palette → render an inert section, no crash.
	if len(palette) == 0 {
		return gosx.El("section", gosx.Attrs(
			gosx.Attr("data-gosx-studio-style-panel", "true"),
			gosx.Attr("data-gosx-studio-authoring-panel-renderer", "gosx-studio"),
		))
	}

	// The form carries the managed-form marker so the authoring runtime intercepts
	// submit. Hidden inputs carry the operation kind so the handler validates correctly.
	formChildren := []gosx.Node{
		gosx.El("input", gosx.Attrs(
			gosx.Attr("type", "hidden"),
			gosx.Attr("name", AuthoringFieldOperation),
			gosx.Attr("value", string(AuthoringOperationSaveAppearance)),
		)),
	}
	// CSRF: authoringruntime reads the form's csrf_token field and forwards it as the
	// X-CSRF-Token header, so a managed POST must carry it (blank token → field omitted,
	// letting the host's own CSRF handling decide).
	if csrfToken != "" {
		formChildren = append(formChildren, gosx.El("input", gosx.Attrs(
			gosx.Attr("type", "hidden"),
			gosx.Attr("name", "csrf_token"),
			gosx.Attr("value", csrfToken),
		)))
	}

	// Color-token grid: one row per palette entry.
	for _, entry := range palette {
		name := entry["name"]
		label := entry["label"]
		cssVar := entry["cssVar"] // data-editor-color-token value — must be the CSS var
		key := entry["key"]
		value := entry["value"]
		fallback := entry["fallback"]

		row := gosx.El("div", gosx.Attrs(
			gosx.Attr("data-gosx-studio-style-color-row", key),
		),
			gosx.El("label", nil,
				gosx.El("span", nil, gosx.Text(label)),
				// Primary color swatch — carries data-editor-color-token=cssVar for
				// styleruntime live preview and the form field name for submission.
				gosx.El("input", gosx.Attrs(
					gosx.Attr("form", formID),
					gosx.Attr("type", "color"),
					gosx.Attr("name", name),
					gosx.Attr("value", value),
					gosx.Attr("data-editor-color-token", cssVar),
					gosx.Attr("data-editor-color-key", key),
				)),
				// Read-only hex mirror — display companion, not submitted (no name attr).
				// Mirrors the inspector color-widget shape (inspector.go ControlColor case).
				gosx.El("input", gosx.Attrs(
					gosx.Attr("type", "text"),
					gosx.Attr("value", value),
					gosx.Attr("data-gosx-studio-inspector-color-mirror", "true"),
				)),
			),
			// Per-token reset button. Carries data-gosx-studio-style-reset + data-fallback
			// so existing styleruntime reset semantics (or a tiny future handler) can revert.
			// type=button prevents accidental form submission.
			gosx.El("button", gosx.Attrs(
				gosx.Attr("type", "button"),
				gosx.Attr("data-gosx-studio-style-reset", key),
				gosx.Attr("data-fallback", fallback),
			), gosx.Text("Reset")),
		)
		formChildren = append(formChildren, row)
	}

	// The form itself is NOT emitted here; it is the outer workbench form or a
	// host-managed <form id=formID> element. We output the save/cancel buttons
	// bound to formID so the host can place the form anywhere in the document.
	//
	// NOTE: the managed-form marker must land on the actual <form id=formID>
	// element, not this section, because island_runtime.js checks
	// form.hasAttribute("data-gosx-studio-authoring-managed") on the submit event
	// target. The section carries the panel-level renderer attribute only.
	//
	// The approach taken here: emit a <form> element with id=formID inside the
	// section. This keeps the pattern self-contained (no external form required)
	// and places the managed-form marker correctly on the form element.
	formEl := gosx.El("form", gosx.Attrs(
		gosx.Attr("id", formID),
		gosx.Attr("method", "post"),
		gosx.Attr("action", action),
		gosx.Attr("data-gosx-studio-authoring-managed", "true"),
	), gosx.Fragment(formChildren...))

	saveBtn := gosx.El("button", gosx.Attrs(
		gosx.Attr("form", formID),
		gosx.Attr("type", "submit"),
	), gosx.Text("Save"))

	cancelBtn := gosx.El("button", gosx.Attrs(
		gosx.Attr("form", formID),
		gosx.Attr("type", "reset"),
		gosx.Attr("data-gosx-studio-inspector-cancel", "true"),
	), gosx.Text("Cancel"))

	return gosx.El("section", gosx.Attrs(
		gosx.Attr("data-gosx-studio-style-panel", "true"),
		gosx.Attr("data-gosx-studio-authoring-panel-renderer", "gosx-studio"),
	),
		formEl,
		saveBtn,
		cancelBtn,
	)
}

// stylePaletteEntries extracts and normalizes palette entries from the view map.
// Accepts []map[string]any or []map[string]string (mirrors inspectorControlOptions tolerance).
func stylePaletteEntries(view map[string]any) []map[string]string {
	raw, ok := view["palette"]
	if !ok || raw == nil {
		return nil
	}
	switch typed := raw.(type) {
	case []map[string]string:
		out := make([]map[string]string, 0, len(typed))
		for _, entry := range typed {
			out = append(out, entry)
		}
		return out
	case []map[string]any:
		out := make([]map[string]string, 0, len(typed))
		for _, entry := range typed {
			out = append(out, map[string]string{
				"key":      workbenchMapString(entry, "key"),
				"name":     workbenchMapString(entry, "name"),
				"label":    workbenchMapString(entry, "label"),
				"cssVar":   workbenchMapString(entry, "cssVar"),
				"value":    workbenchMapString(entry, "value"),
				"fallback": workbenchMapString(entry, "fallback"),
			})
		}
		return out
	default:
		return nil
	}
}
