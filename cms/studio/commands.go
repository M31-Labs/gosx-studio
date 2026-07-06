package studio

import (
	"strings"

	"m31labs.dev/gosx"
)

type CommandKind string

const (
	CommandMode            CommandKind = "mode"
	CommandViewport        CommandKind = "viewport"
	CommandZoom            CommandKind = "zoom"
	CommandCanvas          CommandKind = "canvas"
	CommandInsert          CommandKind = "insert"
	CommandSelectionAction CommandKind = "selection-action"
	CommandHistory         CommandKind = "history"
	CommandToggle          CommandKind = "toggle"
	CommandLink            CommandKind = "link"
	CommandSubmit          CommandKind = "submit"
)

type Command struct {
	Key      string
	Label    string
	Summary  string
	Group    string
	Kind     CommandKind
	Target   string
	Href     string
	Shortcut string
	Keywords []string
	Primary  bool
}

type CommandPaletteOptions struct {
	Class       string
	Launcher    string
	Title       string
	SearchHint  string
	EmptyTitle  string
	EmptyDetail string
	Commands    []Command
}

func RenderCommandPalette(options CommandPaletteOptions) gosx.Node {
	className := firstNonEmpty(options.Class, "gosx-studio__command-palette")
	launcher := firstNonEmpty(options.Launcher, "Command")
	title := firstNonEmpty(options.Title, "Command palette")
	searchHint := firstNonEmpty(options.SearchHint, "Search actions, blocks, routes, and modes")
	emptyTitle := firstNonEmpty(options.EmptyTitle, "No commands")
	emptyDetail := firstNonEmpty(options.EmptyDetail, "Try a different search.")
	listID := "studio-command-list"
	commands := normalizeCommands(options.Commands)
	commandNodes := make([]gosx.Node, 0, len(commands))
	for _, command := range commands {
		commandNodes = append(commandNodes, renderCommand(command))
	}
	return gosx.El("div", gosx.Attrs(
		gosx.Attr("class", className),
		gosx.Attr("data-studio-command-palette", "true"),
		gosx.Attr("data-studio-command-state", "closed"),
	),
		gosx.El("button", gosx.Attrs(
			gosx.Attr("class", "studio-command-launcher"),
			gosx.Attr("type", "button"),
			gosx.Attr("data-studio-command-open", "true"),
			gosx.Attr("aria-haspopup", "dialog"),
			gosx.Attr("aria-expanded", "false"),
		),
			gosx.El("span", nil, gosx.Text(launcher)),
			gosx.El("kbd", nil, gosx.Text("Ctrl K")),
		),
		gosx.El("div", gosx.Attrs(
			gosx.Attr("class", "studio-command-overlay"),
			gosx.Attr("data-studio-command-overlay", "true"),
			gosx.Attr("hidden", "hidden"),
		),
			gosx.El("section", gosx.Attrs(
				gosx.Attr("class", "studio-command-dialog"),
				gosx.Attr("role", "dialog"),
				gosx.Attr("aria-modal", "true"),
				gosx.Attr("aria-label", title),
			),
				gosx.El("div", gosx.Attrs(gosx.Attr("class", "studio-command-head")),
					gosx.El("div", nil,
						gosx.El("p", gosx.Attrs(gosx.Attr("class", "kicker")), gosx.Text("Quick actions")),
						gosx.El("h2", nil, gosx.Text(title)),
					),
					gosx.El("button", gosx.Attrs(
						gosx.Attr("type", "button"),
						gosx.Attr("data-studio-command-close", "true"),
						gosx.Attr("aria-label", "Close command palette"),
					), gosx.Text("Close")),
				),
				gosx.El("label", gosx.Attrs(gosx.Attr("class", "studio-command-search")),
					gosx.El("span", nil, gosx.Text("Search")),
					gosx.El("input", gosx.Attrs(
						gosx.Attr("type", "search"),
						gosx.Attr("role", "combobox"),
						gosx.Attr("aria-autocomplete", "list"),
						gosx.Attr("aria-expanded", "false"),
						gosx.Attr("aria-controls", listID),
						gosx.Attr("data-studio-command-search", "true"),
						gosx.Attr("placeholder", searchHint),
						gosx.Attr("autocomplete", "off"),
					)),
				),
				gosx.El("div", gosx.Attrs(
					gosx.Attr("id", listID),
					gosx.Attr("class", "studio-command-list"),
					gosx.Attr("role", "listbox"),
					gosx.Attr("data-studio-command-list", "true"),
				), gosx.Fragment(commandNodes...)),
				gosx.El("p", gosx.Attrs(
					gosx.Attr("class", "studio-command-empty"),
					gosx.Attr("data-studio-command-empty", "true"),
					gosx.Attr("hidden", "hidden"),
				),
					gosx.El("strong", nil, gosx.Text(emptyTitle)),
					gosx.El("span", nil, gosx.Text(emptyDetail)),
				),
			),
		),
	)
}

func NewCommand(kind CommandKind, key, label, target string) Command {
	return Command{Kind: kind, Key: key, Label: label, Target: target}
}

func renderCommand(command Command) gosx.Node {
	attrs := []any{
		gosx.Attr("class", commandClass(command)),
		gosx.Attr("type", "button"),
		gosx.Attr("role", "option"),
		gosx.Attr("data-studio-command", command.Key),
		gosx.Attr("data-studio-command-kind", string(command.Kind)),
		gosx.Attr("data-studio-command-target", command.Target),
		gosx.Attr("data-studio-command-search-text", commandSearchText(command)),
	}
	if command.Href != "" {
		attrs = append(attrs, gosx.Attr("data-studio-command-href", command.Href))
	}
	if command.Shortcut != "" {
		attrs = append(attrs, gosx.Attr("data-studio-command-shortcut", command.Shortcut))
	}
	children := []gosx.Node{
		gosx.El("span", gosx.Attrs(gosx.Attr("class", "studio-command-item__group")), gosx.Text(command.Group)),
		gosx.El("span", gosx.Attrs(gosx.Attr("class", "studio-command-item__label")), gosx.Text(command.Label)),
	}
	if command.Summary != "" {
		children = append(children, gosx.El("span", gosx.Attrs(gosx.Attr("class", "studio-command-item__summary")), gosx.Text(command.Summary)))
	}
	if command.Shortcut != "" {
		children = append(children, gosx.El("kbd", nil, gosx.Text(command.Shortcut)))
	}
	return gosx.El("button", gosx.Attrs(attrs...), gosx.Fragment(children...))
}

func normalizeCommands(commands []Command) []Command {
	out := make([]Command, 0, len(commands))
	seen := map[string]bool{}
	for _, command := range commands {
		command.Key = normalizeKey(command.Key)
		command.Label = strings.TrimSpace(command.Label)
		command.Summary = strings.TrimSpace(command.Summary)
		command.Group = strings.TrimSpace(command.Group)
		command.Target = strings.TrimSpace(command.Target)
		command.Href = strings.TrimSpace(command.Href)
		command.Shortcut = strings.TrimSpace(command.Shortcut)
		command.Kind = normalizeCommandKind(command.Kind)
		if command.Key == "" {
			command.Key = normalizeKey(string(command.Kind) + "-" + firstNonEmpty(command.Target, command.Href, command.Label))
		}
		if command.Label == "" || seen[command.Key] {
			continue
		}
		if command.Group == "" {
			command.Group = commandKindLabel(command.Kind)
		}
		seen[command.Key] = true
		out = append(out, command)
	}
	return out
}

func normalizeCommandKind(kind CommandKind) CommandKind {
	switch kind {
	case CommandMode, CommandViewport, CommandZoom, CommandCanvas, CommandInsert, CommandSelectionAction, CommandHistory, CommandToggle, CommandLink, CommandSubmit:
		return kind
	default:
		return CommandLink
	}
}

func commandClass(command Command) string {
	className := "studio-command-item studio-command-item--" + string(command.Kind)
	if command.Primary {
		className += " studio-command-item--primary"
	}
	return className
}

func commandSearchText(command Command) string {
	parts := []string{command.Label, command.Summary, command.Group, string(command.Kind), command.Target, command.Href}
	parts = append(parts, command.Keywords...)
	return strings.ToLower(strings.Join(parts, " "))
}

func commandKindLabel(kind CommandKind) string {
	switch kind {
	case CommandMode:
		return "Mode"
	case CommandViewport:
		return "Viewport"
	case CommandZoom:
		return "Zoom"
	case CommandCanvas:
		return "Canvas"
	case CommandInsert:
		return "Insert"
	case CommandSelectionAction:
		return "Selection"
	case CommandHistory:
		return "History"
	case CommandToggle:
		return "View"
	case CommandSubmit:
		return "Save"
	default:
		return "Open"
	}
}
