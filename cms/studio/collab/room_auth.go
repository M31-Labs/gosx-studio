package collab

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	admincollab "m31labs.dev/gosx-admin/blockstudio/collab"
	"m31labs.dev/gosx/hub"
)

const roomActorMetadataKey = "gosx.studio.room.actor"

// roomActorMetadata serializes only server-authenticated identity into the
// Hub's private connection metadata. The metadata is never sent to the peer;
// handlers read it through Client.Metadata instead of accepting JSON claims.
func roomActorMetadata(actor admincollab.Actor) hub.ConnectionMetadata {
	data, err := json.Marshal(actor)
	if err != nil {
		return nil
	}
	return hub.ConnectionMetadata{roomActorMetadataKey: string(data)}
}

func normalizeRoomActor(actor admincollab.Actor) (admincollab.Actor, error) {
	actor = admincollab.NormalizeActor(actor)
	if strings.TrimSpace(actor.ID) == "" {
		return admincollab.Actor{}, errors.New("authenticated actor id is required")
	}
	switch actor.Kind {
	case admincollab.ActorHuman, admincollab.ActorAgent, admincollab.ActorAutomation, admincollab.ActorSystem:
		return actor, nil
	default:
		return admincollab.Actor{}, fmt.Errorf("unsupported authenticated actor kind %q", actor.Kind)
	}
}

func roomActorFromClient(client *hub.Client) (admincollab.Actor, error) {
	if client == nil {
		return admincollab.Actor{}, errors.New("collaboration connection is missing")
	}
	raw, ok := client.Metadata(roomActorMetadataKey)
	if !ok || strings.TrimSpace(raw) == "" {
		return admincollab.Actor{}, errors.New("authenticated actor metadata is missing")
	}
	var actor admincollab.Actor
	if err := json.Unmarshal([]byte(raw), &actor); err != nil {
		return admincollab.Actor{}, errors.New("authenticated actor metadata is malformed")
	}
	return normalizeRoomActor(actor)
}

func (r *Room) roomClientAuthorized(client *hub.Client) bool {
	if r == nil || r.hub == nil || client == nil || client.Hub != r.hub {
		return false
	}
	_, err := roomActorFromClient(client)
	return err == nil
}

func (r *Room) trustedActor(ctx *hub.Context) (admincollab.Actor, bool) {
	if ctx == nil || ctx.Hub != r.hub || ctx.Client == nil || ctx.Client.Hub != r.hub {
		return admincollab.Actor{}, false
	}
	actor, err := roomActorFromClient(ctx.Client)
	if err == nil {
		return actor, true
	}
	// A caller that mounted Room.Hub() directly bypassed Room.ServeHTTP's
	// request resolver. Disconnect it at the first event so it cannot remain a
	// raw Hub client while Room state is being broadcast to other connections.
	if ctx.Hub != nil {
		ctx.Hub.Disconnect(ctx.Client.ID, "authenticated collaboration actor required")
	}
	return admincollab.Actor{}, false
}

// canDecisionPolicy composes the legacy operation policy with the explicit
// decision policy. Authorize is retained as a veto for compatibility, never
// as the grant for a state transition. A legacy Authorize without the new
// decision hook fails closed so an operation policy that permits suggestions
// or comments cannot silently permit accepting, rejecting, resolving, or
// reopening them.
func (r *Room) canDecisionPolicy(actor admincollab.Actor, decision Decision, operationKind admincollab.OperationKind) bool {
	decision.TargetID = strings.TrimSpace(decision.TargetID)
	if r.authorize != nil {
		if !r.authorize(actor, decisionVetoOperation(actor, decision, operationKind)) {
			return false
		}
		if r.authorizeDecision == nil {
			return false
		}
	}
	if r.authorizeDecision != nil {
		return r.authorizeDecision(actor, decision)
	}
	return true
}

// decisionVetoOperation is the best backwards-compatible projection of a
// decision into Authorize's operation-shaped callback. It carries the
// trusted actor and target for legacy vetoes, while the explicit callback is
// the only API that receives the exact decision action.
func decisionVetoOperation(actor admincollab.Actor, decision Decision, kind admincollab.OperationKind) admincollab.Operation {
	return admincollab.Operation{
		ActorID:   actor.ID,
		ActorKind: actor.Kind,
		Kind:      kind,
		Target:    admincollab.Target{BlockID: decision.TargetID},
	}
}

// stampOperationActor applies the connection actor to the operation and to
// nested suggestion operations. This keeps all operation audit fields tied to
// the trusted actor even when a client supplied forged IDs, kinds, or
// capabilities in its payload.
func stampOperationActor(op admincollab.Operation, actor admincollab.Actor) admincollab.Operation {
	op.ActorID = actor.ID
	op.ActorKind = actor.Kind
	if op.Kind != admincollab.OpSuggest || len(op.Payload) == 0 {
		return op
	}
	var payload admincollab.SuggestPayload
	if err := json.Unmarshal(op.Payload, &payload); err != nil {
		return op
	}
	for index := range payload.Operations {
		payload.Operations[index] = stampOperationActor(payload.Operations[index], actor)
	}
	op.Payload = admincollab.Payload(payload)
	return op
}
