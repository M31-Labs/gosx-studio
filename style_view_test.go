package studio

import (
	"testing"
)

func TestBuildStyleControlViewFullShape(t *testing.T) {
	paperSwatches := []StyleControlOption{
		{Value: "#f4ece1", Label: "Sand"},
		{Value: "#2f2823", Label: "Ink"},
	}
	specs := []StyleControlSpec{
		{Property: "text-align", Label: "Alignment", Widget: "segment", Options: []StyleControlOption{
			{Value: "left", Label: "Left"},
			{Value: "center", Label: "Center"},
		}},
		{Property: "padding-block", Label: "Vertical padding", Widget: "number", Unit: "px"},
		{Property: "background-color", Label: "Section background", Widget: "swatch", Options: paperSwatches},
	}
	sections := []StyleSection{
		{Key: "hero", Label: "hero"},
	}
	lookup := func(key, property string) string {
		if key == "hero" && property == "text-align" {
			return "center"
		}
		if key == "hero" && property == "padding-block" {
			return "96px"
		}
		return ""
	}

	view := BuildStyleControlView(sections, specs, lookup, "/action", "csrf-tok")

	if view["hasControls"] != true {
		t.Fatal("expected hasControls true")
	}
	shells, _ := view["shells"].([]map[string]any)
	groups, _ := view["groups"].([]map[string]any)

	if len(shells) != 3 {
		t.Fatalf("expected 3 shells (one per property), got %d", len(shells))
	}
	if len(groups) != 1 {
		t.Fatalf("expected 1 group, got %d", len(groups))
	}

	group := groups[0]
	if group["sectionKey"] != "hero" {
		t.Fatalf("group sectionKey wrong: %v", group["sectionKey"])
	}
	if group["inspectorFor"] != "hero" {
		t.Fatalf("group inspectorFor wrong: %v", group["inspectorFor"])
	}

	controls, _ := group["controls"].([]map[string]any)
	if len(controls) != 3 {
		t.Fatalf("expected 3 controls, got %d", len(controls))
	}

	// text-align: segment, value "center" must be checked
	var align map[string]any
	for _, c := range controls {
		if c["property"] == "text-align" {
			align = c
		}
	}
	if align == nil || align["isSegment"] != true {
		t.Fatalf("text-align must be segment: %v", align)
	}
	opts, _ := align["options"].([]map[string]any)
	checkedCenter := false
	for _, o := range opts {
		if o["value"] == "center" && o["checked"] == true {
			checkedCenter = true
		}
	}
	if !checkedCenter {
		t.Fatalf("center option must be checked: %v", opts)
	}

	// padding-block: number, value carries draft value
	for _, c := range controls {
		if c["property"] == "padding-block" {
			if c["isNumber"] != true || c["value"] != "96px" {
				t.Fatalf("padding-block wrong: %v", c)
			}
		}
	}

	// background-color: swatch
	for _, c := range controls {
		if c["property"] == "background-color" {
			if c["isSwatch"] != true {
				t.Fatalf("background-color must be swatch: %v", c)
			}
			copts, _ := c["options"].([]map[string]any)
			if len(copts) != 2 {
				t.Fatalf("expected 2 swatch options, got %d", len(copts))
			}
			// swatchStyle must be set
			if copts[0]["swatchStyle"] != "background:#f4ece1" {
				t.Fatalf("swatchStyle wrong: %v", copts[0]["swatchStyle"])
			}
		}
	}

	// shells: formID, componentKey, property, breakpoint, state, csrf, hasCsrf
	if shells[0]["componentKey"] != "hero" {
		t.Fatalf("shell componentKey wrong: %v", shells[0])
	}
	if shells[0]["hasCsrf"] != true {
		t.Fatalf("shell hasCsrf must be true when token non-empty: %v", shells[0])
	}
	if shells[0]["breakpoint"] != StyleBreakpointBase {
		t.Fatalf("shell breakpoint must be base: %v", shells[0]["breakpoint"])
	}
	if shells[0]["state"] != StyleStateDefault {
		t.Fatalf("shell state must be default: %v", shells[0]["state"])
	}

	// formID matches convention: editorSetStyle-<key>-<property-sanitized>
	expectedFormID := "editorSetStyle-" + StyleFormIDPart("hero") + "-" + StyleFormIDPart("text-align")
	if shells[0]["formID"] != expectedFormID {
		t.Fatalf("shell formID wrong: got %v want %v", shells[0]["formID"], expectedFormID)
	}
}

func TestBuildStyleControlViewEmptyWithoutSections(t *testing.T) {
	view := BuildStyleControlView(nil, []StyleControlSpec{
		{Property: "text-align", Label: "Alignment", Widget: "segment"},
	}, nil, "/a", "")
	if view["hasControls"] != false {
		t.Fatal("no sections must yield hasControls false")
	}
	shells, _ := view["shells"].([]map[string]any)
	if len(shells) != 0 {
		t.Fatalf("no sections must yield no shells, got %d", len(shells))
	}
}

func TestBuildStyleControlViewSkipsBlankKey(t *testing.T) {
	sections := []StyleSection{
		{Key: "", Label: "empty key"},
		{Key: "hero", Label: "hero"},
	}
	specs := []StyleControlSpec{
		{Property: "color", Label: "Color", Widget: "number"},
	}
	view := BuildStyleControlView(sections, specs, nil, "/a", "")
	groups, _ := view["groups"].([]map[string]any)
	if len(groups) != 1 || groups[0]["sectionKey"] != "hero" {
		t.Fatalf("blank-key section must be skipped: %v", groups)
	}
}

func TestBuildStyleControlViewNoCsrfWhenEmpty(t *testing.T) {
	sections := []StyleSection{{Key: "s1", Label: "S1"}}
	specs := []StyleControlSpec{{Property: "color", Label: "Color", Widget: "number"}}
	view := BuildStyleControlView(sections, specs, nil, "/a", "")
	shells, _ := view["shells"].([]map[string]any)
	if len(shells) == 0 {
		t.Fatal("expected at least one shell")
	}
	if shells[0]["hasCsrf"] != false {
		t.Fatalf("hasCsrf must be false when token empty: %v", shells[0])
	}
}

func TestBuildSectionConfigViewFullShape(t *testing.T) {
	categoryOptions := []ConfigControlOption{
		{Value: "", Label: "All products"},
		{Value: "ceramics", Label: "Ceramics"},
	}
	sections := []ConfigSection{
		{
			Key:   "collection",
			Label: "collection",
			Specs: []ConfigControlSpec{
				{Field: "category", Label: "Category", Widget: "select", Options: categoryOptions},
				{Field: "limit", Label: "Max items", Widget: "number"},
			},
			ValueLookup: func(field string) string {
				if field == "limit" {
					return "4"
				}
				return ""
			},
		},
		{
			Key:   "postlist",
			Label: "postlist",
			Specs: []ConfigControlSpec{
				{Field: "tag", Label: "Tag", Widget: "text"},
				{Field: "limit", Label: "Max items", Widget: "number"},
			},
			ValueLookup: func(field string) string { return "" },
		},
	}

	view := BuildSectionConfigView(sections, "/action", "csrf-x")

	if view["hasControls"] != true {
		t.Fatal("expected hasControls true")
	}
	shells, _ := view["shells"].([]map[string]any)
	groups, _ := view["groups"].([]map[string]any)

	if len(shells) != 4 { // collection(2) + postlist(2)
		t.Fatalf("expected 4 shells, got %d", len(shells))
	}
	if len(groups) != 2 {
		t.Fatalf("expected 2 groups, got %d", len(groups))
	}

	// binding format: home.section.<key>.<field>
	foundBinding := false
	for _, shell := range shells {
		if shell["binding"] == "home.section.collection.category" {
			foundBinding = true
		}
	}
	if !foundBinding {
		t.Fatalf("expected home.section.collection.category binding: %v", shells)
	}

	// collection group controls
	var collectionGroup map[string]any
	for _, g := range groups {
		if g["sectionKey"] == "collection" {
			collectionGroup = g
		}
	}
	if collectionGroup == nil {
		t.Fatal("expected collection group")
	}
	controls, _ := collectionGroup["controls"].([]map[string]any)
	foundLimit := false
	for _, c := range controls {
		if c["label"] == "Max items" && c["value"] == "4" && c["isNumber"] == true {
			foundLimit = true
		}
	}
	if !foundLimit {
		t.Fatalf("collection limit must carry prefilled value: %v", controls)
	}

	// select options for category
	var catControl map[string]any
	for _, c := range controls {
		if c["label"] == "Category" {
			catControl = c
		}
	}
	if catControl == nil {
		t.Fatal("expected category control")
	}
	if catControl["isSelect"] != true {
		t.Fatalf("category must be isSelect: %v", catControl)
	}

	// hasCsrf
	if shells[0]["hasCsrf"] != true {
		t.Fatalf("hasCsrf must be true when token non-empty: %v", shells[0])
	}
}

func TestBuildSectionConfigViewEmptyWhenNoSections(t *testing.T) {
	view := BuildSectionConfigView(nil, "/a", "")
	if view["hasControls"] != false {
		t.Fatal("empty sections must yield hasControls false")
	}
	shells, _ := view["shells"].([]map[string]any)
	if len(shells) != 0 {
		t.Fatalf("expected empty shells slice, got %d", len(shells))
	}
}

func TestBuildSectionConfigViewSkipsBlankKey(t *testing.T) {
	sections := []ConfigSection{
		{Key: "", Label: "bad", Specs: []ConfigControlSpec{{Field: "x", Label: "X", Widget: "text"}}},
	}
	view := BuildSectionConfigView(sections, "/a", "")
	if view["hasControls"] != false {
		t.Fatal("blank key section must be skipped")
	}
}

func TestBuildCustomCSSView(t *testing.T) {
	view := BuildCustomCSSView(".body{}", "/authoring", "tok123")
	if view["formID"] != "editorCustomCssForm" {
		t.Fatalf("formID wrong: %v", view["formID"])
	}
	if view["action"] != "/authoring" {
		t.Fatalf("action wrong: %v", view["action"])
	}
	if view["value"] != ".body{}" {
		t.Fatalf("value wrong: %v", view["value"])
	}
	if view["csrf"] != "tok123" {
		t.Fatalf("csrf wrong: %v", view["csrf"])
	}
	if view["hasCsrf"] != true {
		t.Fatalf("hasCsrf must be true when token non-empty: %v", view["hasCsrf"])
	}
}

func TestBuildCustomCSSViewNoCSRF(t *testing.T) {
	view := BuildCustomCSSView("", "/authoring", "")
	if view["hasCsrf"] != false {
		t.Fatalf("hasCsrf must be false when token empty: %v", view["hasCsrf"])
	}
}
