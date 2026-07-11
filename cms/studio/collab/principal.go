package collab

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
)

type Capability string

const (
	CapabilityView    Capability = "view"
	CapabilityAuthor  Capability = "author"
	CapabilityDesign  Capability = "design"
	CapabilityPublish Capability = "publish"
)

// Principal is supplied by the host after authentication. It is deliberately
// absent from every client operation payload.
type Principal struct {
	ActorID      string              `json:"-"`
	DisplayName  string              `json:"-"`
	Capabilities map[Capability]bool `json:"-"`
}

func (p Principal) Clone() Principal {
	clone := Principal{ActorID: strings.TrimSpace(p.ActorID), DisplayName: strings.TrimSpace(p.DisplayName), Capabilities: map[Capability]bool{}}
	for capability, allowed := range p.Capabilities {
		clone.Capabilities[capability] = allowed
	}
	return clone
}

func (p Principal) Validate() error {
	if strings.TrimSpace(p.ActorID) == "" {
		return fmt.Errorf("authenticated actor id is required")
	}
	return nil
}

func (p Principal) Can(capability Capability) bool { return p.Capabilities[capability] }

type ResourceKey struct {
	TenantID  string `json:"tenantId"`
	ProjectID string `json:"projectId"`
	Kind      string `json:"kind"`
	ID        string `json:"id"`
}

func (r ResourceKey) Normalize() ResourceKey {
	r.TenantID = strings.TrimSpace(r.TenantID)
	r.ProjectID = strings.TrimSpace(r.ProjectID)
	r.Kind = strings.ToLower(strings.TrimSpace(r.Kind))
	r.ID = strings.TrimSpace(r.ID)
	return r
}

func (r ResourceKey) Validate() error {
	r = r.Normalize()
	if r.TenantID == "" || r.ProjectID == "" || r.Kind == "" || r.ID == "" {
		return fmt.Errorf("tenant, project, resource kind, and resource id are required")
	}
	return nil
}

func (r ResourceKey) StableKey() string {
	r = r.Normalize()
	raw, _ := json.Marshal(r)
	sum := sha256.Sum256(raw)
	return "resource:" + hex.EncodeToString(sum[:])
}
