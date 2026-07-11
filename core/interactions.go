package core

import (
	"encoding/json"
	"errors"
	"net/url"
	"sort"
	"strings"
)

func SerializeInteractionGraphs(graphs []InteractionGraph) (string, error) {
	for i := range graphs {
		graphs[i] = graphs[i].Normalize()
		if err := graphs[i].Validate(); err != nil {
			return "", err
		}
	}
	raw, err := json.Marshal(graphs)
	return string(raw), err
}

type InteractionEvent string

const (
	InteractionClick         InteractionEvent = "click"
	InteractionSubmit        InteractionEvent = "submit"
	InteractionHover         InteractionEvent = "hover"
	InteractionFocus         InteractionEvent = "focus"
	InteractionViewportEnter InteractionEvent = "viewport-enter"
)

type InteractionActionKind string

const (
	InteractionShow        InteractionActionKind = "show"
	InteractionHide        InteractionActionKind = "hide"
	InteractionToggleClass InteractionActionKind = "toggle-class"
	InteractionNavigate    InteractionActionKind = "navigate"
	InteractionScroll      InteractionActionKind = "scroll"
	InteractionFocusAction InteractionActionKind = "focus"
	InteractionSetVariant  InteractionActionKind = "set-variant"
	InteractionPlayMedia   InteractionActionKind = "play-media"
	InteractionPauseMedia  InteractionActionKind = "pause-media"
)

type InteractionConditionKind string

const (
	ConditionAlways          InteractionConditionKind = "always"
	ConditionReducedMotion   InteractionConditionKind = "reduced-motion"
	ConditionNoReducedMotion InteractionConditionKind = "no-reduced-motion"
	ConditionViewportMin     InteractionConditionKind = "viewport-min"
)

type InteractionTarget struct {
	CanvasID     string `json:"canvasId,omitempty"`
	DefinitionID string `json:"definitionId,omitempty"`
	InstanceID   string `json:"instanceId,omitempty"`
}
type InteractionCondition struct {
	Kind  InteractionConditionKind `json:"kind"`
	Value string                   `json:"value,omitempty"`
}
type InteractionAction struct {
	ID       string                `json:"id"`
	Kind     InteractionActionKind `json:"kind"`
	TargetID string                `json:"targetId,omitempty"`
	Value    string                `json:"value,omitempty"`
	AssetID  string                `json:"assetId,omitempty"`
	DelayMS  int                   `json:"delayMs,omitempty"`
}
type InteractionGraph struct {
	ID         string                 `json:"id"`
	Target     InteractionTarget      `json:"target"`
	Event      InteractionEvent       `json:"event"`
	Conditions []InteractionCondition `json:"conditions,omitempty"`
	Actions    []InteractionAction    `json:"actions"`
	Enabled    bool                   `json:"enabled"`
	Revision   uint64                 `json:"revision"`
}

var ErrInvalidInteraction = errors.New("invalid interaction graph")

func (g InteractionGraph) Normalize() InteractionGraph {
	g.ID = strings.TrimSpace(g.ID)
	g.Target.CanvasID = strings.TrimSpace(g.Target.CanvasID)
	g.Target.DefinitionID = strings.TrimSpace(g.Target.DefinitionID)
	g.Target.InstanceID = strings.TrimSpace(g.Target.InstanceID)
	for i := range g.Actions {
		a := &g.Actions[i]
		a.ID = strings.TrimSpace(a.ID)
		a.TargetID = strings.TrimSpace(a.TargetID)
		a.Value = strings.TrimSpace(a.Value)
		a.AssetID = strings.TrimSpace(a.AssetID)
	}
	if len(g.Conditions) == 0 {
		g.Conditions = []InteractionCondition{{Kind: ConditionAlways}}
	}
	return g
}
func (g InteractionGraph) Validate() error {
	g = g.Normalize()
	if !safeToken(g.ID) || targetCount(g.Target) != 1 || len(g.Actions) == 0 {
		return ErrInvalidInteraction
	}
	switch g.Event {
	case InteractionClick, InteractionSubmit, InteractionHover, InteractionFocus, InteractionViewportEnter:
	default:
		return ErrInvalidInteraction
	}
	seen := map[string]bool{}
	for _, c := range g.Conditions {
		switch c.Kind {
		case ConditionAlways, ConditionReducedMotion, ConditionNoReducedMotion:
		case ConditionViewportMin:
			if c.Value != "mobile" && c.Value != "tablet" && c.Value != "desktop" {
				return ErrInvalidInteraction
			}
		default:
			return ErrInvalidInteraction
		}
	}
	for _, a := range g.Actions {
		if !safeToken(a.ID) || seen[a.ID] || a.DelayMS < 0 || a.DelayMS > 60000 {
			return ErrInvalidInteraction
		}
		seen[a.ID] = true
		switch a.Kind {
		case InteractionShow, InteractionHide, InteractionScroll, InteractionFocusAction:
			if !safeDOMID(a.TargetID) {
				return ErrInvalidInteraction
			}
		case InteractionToggleClass:
			if !safeDOMID(a.TargetID) || !safeClass(a.Value) {
				return ErrInvalidInteraction
			}
		case InteractionNavigate:
			if !safeNavigation(a.Value) {
				return ErrInvalidInteraction
			}
		case InteractionSetVariant:
			if !safeToken(a.TargetID) || !safeToken(a.Value) {
				return ErrInvalidInteraction
			}
		case InteractionPlayMedia, InteractionPauseMedia:
			if !safeDOMID(a.TargetID) || !safeToken(a.AssetID) {
				return ErrInvalidInteraction
			}
		default:
			return ErrInvalidInteraction
		}
	}
	return nil
}
func InteractionGraphForTarget(graphs map[string]InteractionGraph, canvasID, definitionID, instanceID string) []InteractionGraph {
	out := []InteractionGraph{}
	for _, g := range graphs {
		if !g.Enabled {
			continue
		}
		if instanceID != "" && g.Target.InstanceID == instanceID || g.Target.InstanceID == "" && definitionID != "" && g.Target.DefinitionID == definitionID || g.Target.InstanceID == "" && g.Target.DefinitionID == "" && canvasID != "" && g.Target.CanvasID == canvasID {
			out = append(out, g)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}
func targetCount(t InteractionTarget) int {
	n := 0
	if t.CanvasID != "" {
		n++
	}
	if t.DefinitionID != "" {
		n++
	}
	if t.InstanceID != "" {
		n++
	}
	return n
}
func safeToken(v string) bool {
	if v == "" || len(v) > 128 {
		return false
	}
	for _, r := range v {
		if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-' || r == '_' || r == '.') {
			return false
		}
	}
	return true
}
func safeDOMID(v string) bool {
	return safeToken(v) && !strings.EqualFold(v, "window") && !strings.EqualFold(v, "document")
}
func safeClass(v string) bool { return safeToken(v) && !strings.HasPrefix(v, "js-") }
func safeNavigation(v string) bool {
	if strings.HasPrefix(v, "/") {
		return !strings.HasPrefix(v, "//") && !strings.ContainsAny(v, "\r\n\\")
	}
	u, e := url.Parse(v)
	if e != nil || strings.ContainsAny(v, "\r\n") {
		return false
	}
	if u.Scheme == "https" {
		return u.Host != ""
	}
	return u.Scheme == "mailto" && u.Opaque != "" && strings.Contains(u.Opaque, "@")
}
