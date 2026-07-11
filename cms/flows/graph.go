package flows

import (
	"errors"
	"net/url"
	"sort"
	"strings"
)

type StepKind string

const (
	StepContent StepKind = "content"
	StepForm    StepKind = "form"
	StepSuccess StepKind = "success"
	StepError   StepKind = "error"
)

type TransitionKind string

const (
	TransitionNext    TransitionKind = "next"
	TransitionSuccess TransitionKind = "success"
	TransitionError   TransitionKind = "error"
)

type NavigationKind string

const (
	NavigationNone    NavigationKind = "none"
	NavigationPage    NavigationKind = "page"
	NavigationSection NavigationKind = "section"
)

type FieldRule struct {
	Name     string `json:"name"`
	Label    string `json:"label"`
	Type     string `json:"type"`
	Required bool   `json:"required"`
	Pattern  string `json:"pattern,omitempty"`
}
type FlowStep struct {
	ID           string      `json:"id"`
	Label        string      `json:"label"`
	Kind         StepKind    `json:"kind"`
	ComponentID  string      `json:"componentId,omitempty"`
	DefinitionID string      `json:"definitionId,omitempty"`
	InstanceID   string      `json:"instanceId,omitempty"`
	AssetID      string      `json:"assetId,omitempty"`
	Fields       []FieldRule `json:"fields,omitempty"`
}
type FlowTransition struct {
	ID          string         `json:"id"`
	FromStepID  string         `json:"fromStepId"`
	ToStepID    string         `json:"toStepId"`
	Kind        TransitionKind `json:"kind"`
	Navigation  NavigationKind `json:"navigation,omitempty"`
	Destination string         `json:"destination,omitempty"`
}
type Graph struct {
	ID            string           `json:"id"`
	Name          string           `json:"name"`
	Steps         []FlowStep       `json:"steps"`
	Transitions   []FlowTransition `json:"transitions"`
	InitialStepID string           `json:"initialStepId"`
	Revision      uint64           `json:"revision"`
}

var ErrInvalidGraph = errors.New("invalid flow graph")

func (g Graph) Normalize() Graph {
	g.ID = strings.TrimSpace(g.ID)
	g.Name = strings.TrimSpace(g.Name)
	g.InitialStepID = strings.TrimSpace(g.InitialStepID)
	for i := range g.Steps {
		s := &g.Steps[i]
		s.ID = strings.TrimSpace(s.ID)
		s.Label = strings.TrimSpace(s.Label)
		s.ComponentID = strings.TrimSpace(s.ComponentID)
		s.DefinitionID = strings.TrimSpace(s.DefinitionID)
		s.InstanceID = strings.TrimSpace(s.InstanceID)
		s.AssetID = strings.TrimSpace(s.AssetID)
		for j := range s.Fields {
			f := &s.Fields[j]
			f.Name = strings.TrimSpace(f.Name)
			f.Label = strings.TrimSpace(f.Label)
			f.Type = strings.TrimSpace(f.Type)
			f.Pattern = strings.TrimSpace(f.Pattern)
		}
	}
	for i := range g.Transitions {
		t := &g.Transitions[i]
		t.ID = strings.TrimSpace(t.ID)
		t.FromStepID = strings.TrimSpace(t.FromStepID)
		t.ToStepID = strings.TrimSpace(t.ToStepID)
		t.Destination = strings.TrimSpace(t.Destination)
	}
	return g
}
func (g Graph) Validate() error {
	g = g.Normalize()
	if !token(g.ID) || g.Name == "" || len(g.Steps) == 0 || !token(g.InitialStepID) {
		return ErrInvalidGraph
	}
	steps := map[string]bool{}
	for _, s := range g.Steps {
		if !token(s.ID) || steps[s.ID] {
			return ErrInvalidGraph
		}
		steps[s.ID] = true
		refs := 0
		for _, v := range []string{s.ComponentID, s.DefinitionID, s.InstanceID} {
			if v != "" {
				refs++
			}
		}
		if refs > 1 {
			return ErrInvalidGraph
		}
		switch s.Kind {
		case StepContent, StepForm, StepSuccess, StepError:
		default:
			return ErrInvalidGraph
		}
		for _, f := range s.Fields {
			if !token(f.Name) || (f.Type != "text" && f.Type != "email" && f.Type != "textarea" && f.Type != "checkbox") || strings.ContainsAny(f.Pattern, "<>`/") {
				return ErrInvalidGraph
			}
		}
	}
	if !steps[g.InitialStepID] {
		return ErrInvalidGraph
	}
	seen := map[string]bool{}
	for _, t := range g.Transitions {
		if !token(t.ID) || seen[t.ID] || !steps[t.FromStepID] || !steps[t.ToStepID] {
			return ErrInvalidGraph
		}
		seen[t.ID] = true
		switch t.Kind {
		case TransitionNext, TransitionSuccess, TransitionError:
		default:
			return ErrInvalidGraph
		}
		switch t.Navigation {
		case "", NavigationNone:
			if t.Destination != "" {
				return ErrInvalidGraph
			}
		case NavigationSection:
			if !token(t.Destination) {
				return ErrInvalidGraph
			}
		case NavigationPage:
			if !safePage(t.Destination) {
				return ErrInvalidGraph
			}
		default:
			return ErrInvalidGraph
		}
	}
	return nil
}
func (g Graph) Step(id string) (FlowStep, bool) {
	for _, s := range g.Steps {
		if s.ID == id {
			return s, true
		}
	}
	return FlowStep{}, false
}
func (g Graph) TransitionsFrom(id string) []FlowTransition {
	out := []FlowTransition{}
	for _, t := range g.Transitions {
		if t.FromStepID == id {
			out = append(out, t)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}
func token(v string) bool {
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
func safePage(v string) bool {
	if strings.HasPrefix(v, "/") {
		return !strings.HasPrefix(v, "//") && !strings.ContainsAny(v, "\\\r\n")
	}
	u, e := url.Parse(v)
	return e == nil && u.Scheme == "https" && u.Host != "" && !strings.ContainsAny(v, "\r\n")
}
