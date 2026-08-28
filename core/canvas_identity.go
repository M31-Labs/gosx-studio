package core

import (
	"net/url"
	"strings"
)

// Canvas identity is issued by the rendered page, then resolved by Studio.
// Field is the most specific identity and therefore wins over its containing
// block or node. Route scopes every locator so identities cannot leak between
// pages that happen to use the same field or component keys.
const (
	CanvasRouteAttr    = "data-studio-route"
	CanvasPageAttr     = "data-studio-page-id"
	CanvasFieldAttr    = "data-studio-field"
	CanvasBlockAttr    = "data-studio-block-key"
	CanvasNodeAttr     = "data-studio-node-id"
	CanvasEditableAttr = "data-studio-editable"
)

type CanvasIdentity struct {
	Route    string
	PageID   string
	Field    string
	BlockKey string
	NodeID   string
	Kind     string
}

func (identity CanvasIdentity) Normalize() CanvasIdentity {
	identity.Route = normalizeCanvasRoute(identity.Route)
	identity.PageID = strings.TrimSpace(identity.PageID)
	identity.Field = strings.TrimSpace(identity.Field)
	identity.BlockKey = strings.TrimSpace(identity.BlockKey)
	identity.NodeID = strings.TrimSpace(identity.NodeID)
	identity.Kind = strings.TrimSpace(identity.Kind)
	if identity.Field != "" {
		identity.Kind = "field"
	} else if identity.BlockKey != "" {
		identity.Kind = "block"
	} else if identity.NodeID != "" {
		identity.Kind = "node"
	} else if identity.PageID != "" {
		identity.Kind = "page"
	}
	return identity
}

func (identity CanvasIdentity) Stable() bool {
	identity = identity.Normalize()
	return identity.Route != "" && (identity.Field != "" || identity.BlockKey != "" || identity.NodeID != "" || identity.PageID != "")
}

func normalizeCanvasRoute(route string) string {
	route = strings.TrimSpace(route)
	if route == "" {
		return "/"
	}
	if parsed, err := url.Parse(route); err == nil && (parsed.IsAbs() || strings.HasPrefix(route, "//")) {
		route = parsed.Path
	}
	if index := strings.IndexByte(route, '?'); index >= 0 {
		route = route[:index]
	}
	if index := strings.IndexByte(route, '#'); index >= 0 {
		route = route[:index]
	}
	if route == "" {
		return "/"
	}
	return route
}
