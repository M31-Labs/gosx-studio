package collab

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"m31labs.dev/gosx-admin/blockstudio"
	admincollab "m31labs.dev/gosx-admin/blockstudio/collab"
	"m31labs.dev/gosx/hub"
)

type Resource struct {
	Kind  string `json:"kind"`
	ID    string `json:"id"`
	Title string `json:"title,omitempty"`
}

type Store interface {
	LoadDraft(Resource) (blockstudio.Document, bool, error)
	SaveDraft(Resource, blockstudio.Document) error
}

type AuthorizeFunc func(admincollab.Actor, admincollab.Operation) bool

type Options struct {
	Resource     Resource
	Document     blockstudio.Document
	Store        Store
	Hub          *hub.Hub
	Authorize    AuthorizeFunc
	PersistEvery time.Duration
}

type PresenceState string
type ProposalStatus string
type CommentStatus string

const (
	PresenceViewing    PresenceState = "viewing"
	PresenceEditing    PresenceState = "editing"
	PresenceSuggesting PresenceState = "suggesting"
	PresenceRunning    PresenceState = "running_task"

	ProposalPending  ProposalStatus = "pending"
	ProposalAccepted ProposalStatus = "accepted"
	ProposalRejected ProposalStatus = "rejected"

	CommentOpen     CommentStatus = "open"
	CommentResolved CommentStatus = "resolved"
)

type Presence struct {
	Actor      admincollab.Actor  `json:"actor"`
	State      PresenceState      `json:"state"`
	Selection  admincollab.Target `json:"selection,omitempty"`
	LastActive time.Time          `json:"lastActive"`
}

type Snapshot struct {
	Resource          Resource                 `json:"resource"`
	Document          blockstudio.Document     `json:"document"`
	Presence          []Presence               `json:"presence,omitempty"`
	Suggestions       []admincollab.Suggestion `json:"suggestions,omitempty"`
	ProposalDecisions []ProposalDecision       `json:"proposalDecisions,omitempty"`
	Comments          []admincollab.Comment    `json:"comments,omitempty"`
	CommentDecisions  []CommentDecision        `json:"commentDecisions,omitempty"`
	Reviews           []admincollab.Review     `json:"reviews,omitempty"`
	Updated           time.Time                `json:"updated"`
}

type ProposalDecision struct {
	SuggestionID string             `json:"suggestionId"`
	Status       ProposalStatus     `json:"status"`
	Actor        admincollab.Actor  `json:"actor"`
	Reason       string             `json:"reason,omitempty"`
	Review       admincollab.Review `json:"review,omitempty"`
	Created      time.Time          `json:"created"`
}

type CommentDecision struct {
	CommentID string            `json:"commentId"`
	Status    CommentStatus     `json:"status"`
	Actor     admincollab.Actor `json:"actor"`
	Reason    string            `json:"reason,omitempty"`
	Created   time.Time         `json:"created"`
}

type Room struct {
	mu               sync.RWMutex
	resource         Resource
	doc              blockstudio.Document
	store            Store
	hub              *hub.Hub
	authorize        AuthorizeFunc
	persistEvery     time.Duration
	lastPersist      time.Time
	presence         map[string]Presence
	suggestions      []admincollab.Suggestion
	decisions        []ProposalDecision
	comments         []admincollab.Comment
	commentDecisions []CommentDecision
	reviews          []admincollab.Review
	updated          time.Time
}

var _ http.Handler = (*Room)(nil)

func NewRoom(options Options) (*Room, error) {
	resource := normalizeResource(options.Resource)
	if resource.Kind == "" || resource.ID == "" {
		return nil, fmt.Errorf("studio collab room requires resource kind and id")
	}
	doc := cloneDocument(options.Document)
	if options.Store != nil {
		if saved, ok, err := options.Store.LoadDraft(resource); err != nil {
			return nil, err
		} else if ok {
			doc = cloneDocument(saved)
		}
	}
	if doc.Version <= 0 {
		doc.Version = 1
	}
	h := options.Hub
	if h == nil {
		h = hub.New("studio:" + resource.Kind + ":" + resource.ID)
	}
	room := &Room{
		resource:     resource,
		doc:          doc,
		store:        options.Store,
		hub:          h,
		authorize:    options.Authorize,
		persistEvery: options.PersistEvery,
		presence:     map[string]Presence{},
		updated:      time.Now().UTC(),
	}
	room.registerHub()
	return room, nil
}

func (r *Room) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	r.hub.ServeHTTP(w, req)
}

func (r *Room) Hub() *hub.Hub {
	return r.hub
}

func (r *Room) Snapshot() Snapshot {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.snapshotLocked()
}

func (r *Room) Join(actor admincollab.Actor, state PresenceState) Presence {
	actor = admincollab.NormalizeActor(actor)
	if state == "" {
		state = PresenceViewing
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	presence := Presence{Actor: actor, State: state, LastActive: time.Now().UTC()}
	r.presence[actor.ID] = presence
	r.broadcastLocked("studio.presence", r.presenceListLocked())
	return presence
}

func (r *Room) Leave(actorID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.presence, cleanID(actorID))
	r.broadcastLocked("studio.presence", r.presenceListLocked())
}

func (r *Room) SetPresence(actor admincollab.Actor, state PresenceState, selection admincollab.Target) Presence {
	actor = admincollab.NormalizeActor(actor)
	if state == "" {
		state = PresenceViewing
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	presence := Presence{Actor: actor, State: state, Selection: selection, LastActive: time.Now().UTC()}
	r.presence[actor.ID] = presence
	r.broadcastLocked("studio.presence", r.presenceListLocked())
	return presence
}

func (r *Room) ApplyOperation(actor admincollab.Actor, op admincollab.Operation) (Snapshot, error) {
	return r.ApplyTransaction(admincollab.Transaction{
		ID:         op.ID,
		Actor:      actor,
		Operations: []admincollab.Operation{op},
	})
}

func (r *Room) ApplyTransaction(tx admincollab.Transaction) (Snapshot, error) {
	actor := admincollab.NormalizeActor(tx.Actor)
	r.mu.Lock()
	defer r.mu.Unlock()
	ops := make([]admincollab.Operation, 0, len(tx.Operations))
	for _, op := range tx.Operations {
		if op.ActorID == "" {
			op.ActorID = actor.ID
		}
		if op.ActorKind == "" {
			op.ActorKind = actor.Kind
		}
		if !r.can(actor, op) {
			return Snapshot{}, fmt.Errorf("actor %q is not allowed to %s", actor.ID, op.Kind)
		}
		ops = append(ops, op)
	}
	tx.Actor = actor
	tx.Operations = ops
	review := admincollab.ReviewTransaction(r.doc, tx)
	result, err := admincollab.Apply(r.doc, ops)
	if err != nil {
		return Snapshot{}, err
	}
	r.doc = result.Document
	r.suggestions = append(r.suggestions, result.Suggestions...)
	r.comments = append(r.comments, result.Comments...)
	if review.OperationCount > 0 {
		r.reviews = append(r.reviews, review)
	}
	r.updated = time.Now().UTC()
	r.touchPresenceLocked(actor, presenceStateForOps(ops))
	if err := r.persistLocked(); err != nil {
		return Snapshot{}, err
	}
	snapshot := r.snapshotLocked()
	r.broadcastLocked("studio.snapshot", snapshot)
	return snapshot, nil
}

func (r *Room) AcceptSuggestion(actor admincollab.Actor, suggestionID string) (Snapshot, error) {
	actor = admincollab.NormalizeActor(actor)
	suggestionID = strings.TrimSpace(suggestionID)
	r.mu.Lock()
	defer r.mu.Unlock()
	if !r.canDecide(actor) {
		return Snapshot{}, fmt.Errorf("actor %q is not allowed to accept suggestions", actor.ID)
	}
	suggestion, ok := r.pendingSuggestionLocked(suggestionID)
	if !ok {
		return Snapshot{}, fmt.Errorf("suggestion %q is not pending", suggestionID)
	}
	ops := acceptedSuggestionOperations(suggestion, actor)
	if len(ops) == 0 {
		return Snapshot{}, fmt.Errorf("suggestion %q has no operations to accept", suggestionID)
	}
	for _, op := range ops {
		if !r.can(actor, op) {
			return Snapshot{}, fmt.Errorf("actor %q is not allowed to accept %s", actor.ID, op.Kind)
		}
	}
	tx := admincollab.Transaction{
		ID:         "accept-" + suggestion.ID,
		Actor:      actor,
		Operations: ops,
	}
	review := admincollab.ReviewTransaction(r.doc, tx)
	result, err := admincollab.Apply(r.doc, ops)
	if err != nil {
		return Snapshot{}, err
	}
	r.doc = result.Document
	r.comments = append(r.comments, result.Comments...)
	r.reviews = append(r.reviews, review)
	r.decisions = append(r.decisions, ProposalDecision{
		SuggestionID: suggestion.ID,
		Status:       ProposalAccepted,
		Actor:        actor,
		Review:       review,
		Created:      time.Now().UTC(),
	})
	r.updated = time.Now().UTC()
	r.touchPresenceLocked(actor, PresenceEditing)
	if err := r.persistLocked(); err != nil {
		return Snapshot{}, err
	}
	snapshot := r.snapshotLocked()
	r.broadcastLocked("studio.snapshot", snapshot)
	return snapshot, nil
}

func (r *Room) RejectSuggestion(actor admincollab.Actor, suggestionID, reason string) (Snapshot, error) {
	actor = admincollab.NormalizeActor(actor)
	suggestionID = strings.TrimSpace(suggestionID)
	reason = strings.TrimSpace(reason)
	r.mu.Lock()
	defer r.mu.Unlock()
	if !r.canDecide(actor) {
		return Snapshot{}, fmt.Errorf("actor %q is not allowed to reject suggestions", actor.ID)
	}
	suggestion, ok := r.pendingSuggestionLocked(suggestionID)
	if !ok {
		return Snapshot{}, fmt.Errorf("suggestion %q is not pending", suggestionID)
	}
	r.decisions = append(r.decisions, ProposalDecision{
		SuggestionID: suggestion.ID,
		Status:       ProposalRejected,
		Actor:        actor,
		Reason:       reason,
		Created:      time.Now().UTC(),
	})
	r.updated = time.Now().UTC()
	r.touchPresenceLocked(actor, PresenceViewing)
	snapshot := r.snapshotLocked()
	r.broadcastLocked("studio.snapshot", snapshot)
	return snapshot, nil
}

func (r *Room) AddComment(actor admincollab.Actor, target admincollab.Target, body string) (Snapshot, error) {
	actor = admincollab.NormalizeActor(actor)
	body = strings.TrimSpace(body)
	if body == "" {
		return Snapshot{}, fmt.Errorf("comment body is required")
	}
	now := time.Now().UTC()
	return r.ApplyOperation(actor, admincollab.Operation{
		ID:        commentOperationID(actor, now),
		ActorID:   actor.ID,
		ActorKind: actor.Kind,
		Clock:     now.Format("20060102T150405.000000000Z"),
		Target:    target,
		Kind:      admincollab.OpComment,
		Payload:   admincollab.Payload(admincollab.CommentPayload{Body: body}),
		Created:   now,
	})
}

func (r *Room) ResolveComment(actor admincollab.Actor, commentID, reason string) (Snapshot, error) {
	return r.setCommentStatus(actor, commentID, CommentResolved, reason)
}

func (r *Room) ReopenComment(actor admincollab.Actor, commentID, reason string) (Snapshot, error) {
	return r.setCommentStatus(actor, commentID, CommentOpen, reason)
}

func (r *Room) Persist() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.persistLocked()
}

func (r *Room) registerHub() {
	r.hub.Latch("studio.snapshot")
	r.hub.Latch("studio.presence")
	r.hub.On("studio.join", func(ctx *hub.Context) {
		var msg struct {
			Actor admincollab.Actor `json:"actor"`
			State PresenceState     `json:"state,omitempty"`
		}
		if err := json.Unmarshal(ctx.Data, &msg); err != nil {
			return
		}
		r.Join(msg.Actor, msg.State)
		r.hub.Send(ctx.Client.ID, "studio.snapshot", r.Snapshot())
	})
	r.hub.On("studio.presence", func(ctx *hub.Context) {
		var msg struct {
			Actor     admincollab.Actor  `json:"actor"`
			State     PresenceState      `json:"state,omitempty"`
			Selection admincollab.Target `json:"selection,omitempty"`
		}
		if err := json.Unmarshal(ctx.Data, &msg); err != nil {
			return
		}
		r.SetPresence(msg.Actor, msg.State, msg.Selection)
	})
	r.hub.On("studio.operation", func(ctx *hub.Context) {
		var msg struct {
			Actor     admincollab.Actor     `json:"actor"`
			Operation admincollab.Operation `json:"operation"`
		}
		if err := json.Unmarshal(ctx.Data, &msg); err != nil {
			return
		}
		_, _ = r.ApplyOperation(msg.Actor, msg.Operation)
	})
	r.hub.On("studio.transaction", func(ctx *hub.Context) {
		var tx admincollab.Transaction
		if err := json.Unmarshal(ctx.Data, &tx); err != nil {
			return
		}
		_, _ = r.ApplyTransaction(tx)
	})
	r.hub.On("studio.acceptSuggestion", func(ctx *hub.Context) {
		var msg struct {
			Actor        admincollab.Actor `json:"actor"`
			SuggestionID string            `json:"suggestionId"`
		}
		if err := json.Unmarshal(ctx.Data, &msg); err != nil {
			return
		}
		_, _ = r.AcceptSuggestion(msg.Actor, msg.SuggestionID)
	})
	r.hub.On("studio.rejectSuggestion", func(ctx *hub.Context) {
		var msg struct {
			Actor        admincollab.Actor `json:"actor"`
			SuggestionID string            `json:"suggestionId"`
			Reason       string            `json:"reason,omitempty"`
		}
		if err := json.Unmarshal(ctx.Data, &msg); err != nil {
			return
		}
		_, _ = r.RejectSuggestion(msg.Actor, msg.SuggestionID, msg.Reason)
	})
	r.hub.On("studio.comment", func(ctx *hub.Context) {
		var msg struct {
			Actor  admincollab.Actor  `json:"actor"`
			Target admincollab.Target `json:"target,omitempty"`
			Body   string             `json:"body"`
		}
		if err := json.Unmarshal(ctx.Data, &msg); err != nil {
			return
		}
		_, _ = r.AddComment(msg.Actor, msg.Target, msg.Body)
	})
	r.hub.On("studio.resolveComment", func(ctx *hub.Context) {
		var msg struct {
			Actor     admincollab.Actor `json:"actor"`
			CommentID string            `json:"commentId"`
			Reason    string            `json:"reason,omitempty"`
		}
		if err := json.Unmarshal(ctx.Data, &msg); err != nil {
			return
		}
		_, _ = r.ResolveComment(msg.Actor, msg.CommentID, msg.Reason)
	})
	r.hub.On("studio.reopenComment", func(ctx *hub.Context) {
		var msg struct {
			Actor     admincollab.Actor `json:"actor"`
			CommentID string            `json:"commentId"`
			Reason    string            `json:"reason,omitempty"`
		}
		if err := json.Unmarshal(ctx.Data, &msg); err != nil {
			return
		}
		_, _ = r.ReopenComment(msg.Actor, msg.CommentID, msg.Reason)
	})
}

func (r *Room) can(actor admincollab.Actor, op admincollab.Operation) bool {
	if r.authorize != nil {
		return r.authorize(actor, op)
	}
	if actor.Kind == admincollab.ActorSystem {
		return true
	}
	switch op.Kind {
	case admincollab.OpComment:
		return hasCapability(actor, admincollab.CapabilityComment) || hasCapability(actor, admincollab.CapabilityEdit)
	case admincollab.OpSuggest:
		return hasCapability(actor, admincollab.CapabilitySuggest) || hasCapability(actor, admincollab.CapabilityEdit)
	default:
		return hasCapability(actor, admincollab.CapabilityEdit)
	}
}

func (r *Room) canDecide(actor admincollab.Actor) bool {
	if actor.Kind == admincollab.ActorSystem {
		return true
	}
	return hasCapability(actor, admincollab.CapabilityEdit) || hasCapability(actor, admincollab.CapabilityPublish)
}

func (r *Room) canDecideComment(actor admincollab.Actor) bool {
	if actor.Kind == admincollab.ActorSystem {
		return true
	}
	return hasCapability(actor, admincollab.CapabilityComment) || hasCapability(actor, admincollab.CapabilityEdit) || hasCapability(actor, admincollab.CapabilityPublish)
}

func (r *Room) setCommentStatus(actor admincollab.Actor, commentID string, status CommentStatus, reason string) (Snapshot, error) {
	actor = admincollab.NormalizeActor(actor)
	commentID = strings.TrimSpace(commentID)
	reason = strings.TrimSpace(reason)
	status = normalizeCommentStatus(status)
	r.mu.Lock()
	defer r.mu.Unlock()
	if !r.canDecideComment(actor) {
		return Snapshot{}, fmt.Errorf("actor %q is not allowed to update comments", actor.ID)
	}
	if _, ok := r.commentLocked(commentID); !ok {
		return Snapshot{}, fmt.Errorf("comment %q does not exist", commentID)
	}
	if r.commentStatusLocked(commentID) == status {
		return Snapshot{}, fmt.Errorf("comment %q is already %s", commentID, status)
	}
	r.commentDecisions = append(r.commentDecisions, CommentDecision{
		CommentID: commentID,
		Status:    status,
		Actor:     actor,
		Reason:    reason,
		Created:   time.Now().UTC(),
	})
	r.updated = time.Now().UTC()
	r.touchPresenceLocked(actor, PresenceViewing)
	snapshot := r.snapshotLocked()
	r.broadcastLocked("studio.snapshot", snapshot)
	return snapshot, nil
}

func (r *Room) commentLocked(commentID string) (admincollab.Comment, bool) {
	commentID = strings.TrimSpace(commentID)
	if commentID == "" {
		return admincollab.Comment{}, false
	}
	for _, comment := range r.comments {
		if comment.ID == commentID {
			return comment, true
		}
	}
	return admincollab.Comment{}, false
}

func (r *Room) commentStatusLocked(commentID string) CommentStatus {
	status := CommentOpen
	for _, decision := range r.commentDecisions {
		if decision.CommentID == commentID {
			status = normalizeCommentStatus(decision.Status)
		}
	}
	return status
}

func (r *Room) pendingSuggestionLocked(suggestionID string) (admincollab.Suggestion, bool) {
	suggestionID = strings.TrimSpace(suggestionID)
	if suggestionID == "" || r.proposalStatusLocked(suggestionID) != ProposalPending {
		return admincollab.Suggestion{}, false
	}
	for _, suggestion := range r.suggestions {
		if suggestion.ID == suggestionID {
			return suggestion, true
		}
	}
	return admincollab.Suggestion{}, false
}

func (r *Room) proposalStatusLocked(suggestionID string) ProposalStatus {
	status := ProposalPending
	for _, decision := range r.decisions {
		if decision.SuggestionID == suggestionID {
			status = decision.Status
		}
	}
	return status
}

func (r *Room) persistLocked() error {
	if r.store == nil {
		return nil
	}
	if r.persistEvery > 0 && !r.lastPersist.IsZero() && time.Since(r.lastPersist) < r.persistEvery {
		return nil
	}
	if err := r.store.SaveDraft(r.resource, cloneDocument(r.doc)); err != nil {
		return err
	}
	r.lastPersist = time.Now().UTC()
	return nil
}

func (r *Room) snapshotLocked() Snapshot {
	return Snapshot{
		Resource:          r.resource,
		Document:          cloneDocument(r.doc),
		Presence:          r.presenceListLocked(),
		Suggestions:       cloneSuggestions(r.suggestions),
		ProposalDecisions: cloneProposalDecisions(r.decisions),
		Comments:          cloneComments(r.comments),
		CommentDecisions:  cloneCommentDecisions(r.commentDecisions),
		Reviews:           cloneReviews(r.reviews),
		Updated:           r.updated,
	}
}

func (r *Room) presenceListLocked() []Presence {
	out := make([]Presence, 0, len(r.presence))
	for _, presence := range r.presence {
		out = append(out, presence)
	}
	return sortPresence(out)
}

func (r *Room) touchPresenceLocked(actor admincollab.Actor, state PresenceState) {
	if actor.ID == "" {
		return
	}
	if state == "" {
		state = PresenceEditing
	}
	presence := r.presence[actor.ID]
	presence.Actor = actor
	presence.State = state
	presence.LastActive = time.Now().UTC()
	r.presence[actor.ID] = presence
}

func (r *Room) broadcastLocked(event string, value any) {
	if r.hub != nil {
		r.hub.Broadcast(event, value)
	}
}

func presenceStateForOps(ops []admincollab.Operation) PresenceState {
	for _, op := range ops {
		switch op.Kind {
		case admincollab.OpSuggest:
			return PresenceSuggesting
		case admincollab.OpComment:
			return PresenceViewing
		}
	}
	return PresenceEditing
}

func hasCapability(actor admincollab.Actor, capability admincollab.Capability) bool {
	for _, value := range actor.Capabilities {
		if value == capability {
			return true
		}
	}
	return false
}

func acceptedSuggestionOperations(suggestion admincollab.Suggestion, actor admincollab.Actor) []admincollab.Operation {
	ops := admincollab.NormalizeOperations(suggestion.Operations)
	for index := range ops {
		if ops[index].ActorID == "" {
			ops[index].ActorID = firstNonEmpty(suggestion.ActorID, actor.ID)
		}
		if ops[index].ActorKind == "" {
			ops[index].ActorKind = firstNonEmptyActorKind(suggestion.ActorKind, actor.Kind)
		}
	}
	return ops
}

func normalizeCommentStatus(status CommentStatus) CommentStatus {
	if status == CommentResolved {
		return CommentResolved
	}
	return CommentOpen
}

func commentOperationID(actor admincollab.Actor, created time.Time) string {
	id := cleanID(actor.ID)
	if id == "" {
		id = "actor"
	}
	return fmt.Sprintf("comment-%s-%d", id, created.UnixNano())
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func firstNonEmptyActorKind(values ...admincollab.ActorKind) admincollab.ActorKind {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return admincollab.ActorHuman
}
