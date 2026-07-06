package studio

import (
	"strings"
	"testing"

	"m31labs.dev/gosx"
)

func TestRenderCommandPalette(t *testing.T) {
	html := gosx.RenderHTML(RenderCommandPalette(CommandPaletteOptions{
		Class: "studio-command-palette",
		Commands: []Command{
			{Kind: CommandMode, Key: "mode-structure", Label: "Structure", Group: "Modes", Target: "structure", Shortcut: "1"},
			{Kind: CommandInsert, Label: "Hero", Target: "hero", Summary: "Add hero block", Primary: true, Keywords: []string{"home"}},
			{Kind: CommandLink, Key: "media", Label: "Media", Href: "/admin/media"},
			{Label: ""},
		},
	}))
	if !strings.Contains(html, `data-studio-command-palette="true"`) || !strings.Contains(html, `data-studio-command-open="true"`) || !strings.Contains(html, `data-studio-command-search="true"`) {
		t.Fatalf("expected command palette shell hooks, got: %s", html)
	}
	if !strings.Contains(html, `role="combobox"`) || !strings.Contains(html, `aria-controls="studio-command-list"`) || !strings.Contains(html, `role="listbox"`) {
		t.Fatalf("expected accessible command palette hooks, got: %s", html)
	}
	if !strings.Contains(html, `data-studio-command-kind="mode"`) || !strings.Contains(html, `data-studio-command-target="structure"`) {
		t.Fatalf("expected mode command hooks, got: %s", html)
	}
	if !strings.Contains(html, `data-studio-command-shortcut="1"`) {
		t.Fatalf("expected shortcut hook, got: %s", html)
	}
	if !strings.Contains(html, `studio-command-item--primary`) || !strings.Contains(html, `data-studio-command-kind="insert"`) {
		t.Fatalf("expected primary insert command, got: %s", html)
	}
	if !strings.Contains(html, `data-studio-command-href="/admin/media"`) {
		t.Fatalf("expected link command href hook, got: %s", html)
	}
	if strings.Contains(html, `data-studio-command-kind=""`) {
		t.Fatalf("expected invalid commands to be skipped: %s", html)
	}
}

func TestNormalizeCommandsDedupeAndDefaults(t *testing.T) {
	commands := normalizeCommands([]Command{
		{Kind: CommandMode, Label: "Style", Target: "style"},
		{Kind: CommandMode, Label: "Style duplicate", Target: "style"},
		{Kind: "unknown", Key: "media", Label: "Media", Href: "/admin/media"},
	})
	if len(commands) != 2 {
		t.Fatalf("expected generated duplicate key to dedupe, got %#v", commands)
	}
	if commands[0].Key != "mode-style" || commands[0].Group != "Mode" {
		t.Fatalf("unexpected generated command: %#v", commands[0])
	}
	if commands[1].Kind != CommandLink || commands[1].Group != "Open" {
		t.Fatalf("unexpected unknown command normalization: %#v", commands[1])
	}
}

func TestStudioCommandsBuildsDefaultBlocksAndFlows(t *testing.T) {
	commands := StudioCommands(StudioCommandOptions{
		Shell: Shell{
			Modes:     []Mode{NewMode("structure", "Structure", true)},
			Viewports: []Viewport{NewViewport("desktop", "Desktop", "100%", true)},
			Canvas:    CanvasSurface{Zoom: "fit"},
			Actions: []Action{
				{Key: "public", Label: "Public site", Href: "/"},
				{Key: "save", Label: "Save", Href: "/save"},
			},
		},
		Blocks: []CommandBlock{{Key: "hero", Label: "Hero", Summary: "Homepage hero"}},
		Flows: []CommandFlow{{
			Key:            "schedule-tour",
			Label:          "Schedule tour",
			Description:    "Request a visit",
			Route:          "/contact?flow=schedule-tour",
			HasRoute:       true,
			EmbedTarget:    "tour-form",
			HasEmbedTarget: true,
		}},
		Extra:     []Command{{Kind: CommandLink, Key: "media", Label: "Media", Href: "/admin/media"}},
		SaveLabel: "Save checkpoint",
	})
	byKey := map[string]Command{}
	for _, command := range commands {
		byKey[command.Key] = command
	}
	for _, key := range []string{"save", "undo", "redo", "toggle-layers", "mode-structure", "viewport-desktop", "zoom-fit", "canvas-fit", "canvas-open-selected", "canvas-nudge-left", "open-public", "selection-reveal", "insert-hero", "media", "open-flow-schedule-tour", "insert-flow-schedule-tour"} {
		if byKey[key].Key == "" {
			t.Fatalf("expected command %q in %#v", key, commands)
		}
	}
	if byKey["save"].Label != "Save checkpoint" || byKey["open-public"].Label != "Open public site" || byKey["undo"].Kind != CommandHistory || byKey["redo"].Shortcut != "Ctrl Shift Z" || byKey["canvas-fit"].Kind != CommandCanvas || byKey["canvas-open-selected"].Target != "open-selected" || byKey["insert-hero"].Kind != CommandInsert || byKey["open-flow-schedule-tour"].Href == "" || byKey["insert-flow-schedule-tour"].Target != "tour-form" {
		t.Fatalf("unexpected generated commands: %#v", byKey)
	}
}

func TestShellActionCommandsBuildLinks(t *testing.T) {
	commands := ShellActionCommands([]Action{
		{Key: "public", Label: "Public site", Href: "/"},
		{Key: "preview", Label: "Preview storefront", Href: "/admin/storefront", Primary: true},
		{Key: "save", Label: "Save", Href: "/save"},
		{Key: "empty", Label: "Empty"},
	})
	if len(commands) != 2 {
		t.Fatalf("expected only linkable non-save actions, got %#v", commands)
	}
	if commands[0].Key != "open-public" || commands[0].Label != "Open public site" || commands[0].Href != "/" {
		t.Fatalf("unexpected public action command: %#v", commands[0])
	}
	if commands[1].Key != "open-preview" || commands[1].Label != "Preview storefront" || !commands[1].Primary {
		t.Fatalf("unexpected preview action command: %#v", commands[1])
	}
}
