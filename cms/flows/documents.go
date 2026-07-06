package flows

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"m31labs.dev/gosx-admin/blockstudio"
	"m31labs.dev/gosx-admin/workbench"
	"m31labs.dev/gosx-studio/cms/lifecycle"
)

const (
	DocumentKind = "flow.document"
	ResourceKind = "flow"

	FlowKeyAppointment     = "appointment"
	FlowKeyCheckoutHandoff = "checkout-handoff"
	FlowKeyContact         = "contact"
	FlowKeyEnrollment      = "enrollment"
	FlowKeyNewsletter      = "newsletter"
	FlowKeyPurchaseRequest = "purchase-request"
	FlowKeyScheduleTour    = "schedule-tour"

	ActionDraftSaved = "flow.draft_saved"
	ActionPublished  = "flow.published"
	ActionRestored   = "flow.revision_restored"
)

type Document struct {
	Version     int                    `json:"version"`
	ID          string                 `json:"id,omitempty"`
	Key         string                 `json:"key"`
	Label       string                 `json:"label"`
	Description string                 `json:"description,omitempty"`
	Steps       []DocumentStep         `json:"steps"`
	Actions     []DocumentAction       `json:"actions,omitempty"`
	State       lifecycle.PublishState `json:"state,omitempty"`
	Created     time.Time              `json:"created,omitempty"`
	Updated     time.Time              `json:"updated,omitempty"`
	Published   time.Time              `json:"published,omitempty"`
}

type DocumentStep struct {
	Key    string               `json:"key"`
	Label  string               `json:"label"`
	Blocks blockstudio.Document `json:"blocks"`
}

type DocumentAction struct {
	Key        string            `json:"key"`
	Label      string            `json:"label"`
	HandlerRef string            `json:"handlerRef"`
	Fields     []workbench.Field `json:"fields,omitempty"`
}

type DocumentOptions struct {
	ID        string
	State     lifecycle.PublishState
	Created   time.Time
	Updated   time.Time
	Published time.Time
}

type Draft struct {
	Document       Document             `json:"document"`
	BaseRevisionID string               `json:"baseRevisionId,omitempty"`
	AuthorID       string               `json:"authorId,omitempty"`
	State          lifecycle.DraftState `json:"state"`
	Updated        time.Time            `json:"updated"`
}

type Publication struct {
	Document   Document  `json:"document"`
	RevisionID string    `json:"revisionId,omitempty"`
	AuthorID   string    `json:"authorId,omitempty"`
	Published  time.Time `json:"published"`
}

type Instance struct {
	ID         string                 `json:"id,omitempty"`
	DocumentID string                 `json:"documentId,omitempty"`
	FlowKey    string                 `json:"flowKey"`
	Definition Definition             `json:"definition"`
	Source     Document               `json:"source"`
	RevisionID string                 `json:"revisionId,omitempty"`
	State      lifecycle.PublishState `json:"state,omitempty"`
	Created    time.Time              `json:"created,omitempty"`
	Updated    time.Time              `json:"updated,omitempty"`
}

type InstanceOptions struct {
	ID         string
	RevisionID string
	State      lifecycle.PublishState
	Created    time.Time
	Updated    time.Time
}

type PublishResult struct {
	Publication Publication        `json:"publication"`
	Revision    lifecycle.Revision `json:"revision"`
}

type DocumentConfig struct {
	HandlerRefs map[string]string
	StepLabels  map[string]string
}

type DraftConfig struct {
	AuthorID       string
	BaseRevisionID string
	HandlerRefs    map[string]string
	StepLabels     map[string]string
	Now            time.Time
}

type HandlerRefs map[string]string

type DocumentStore interface {
	GetFlowDocument(id string) (Document, bool)
	SaveFlowDocument(Document) error
}

type DraftStore interface {
	GetFlowDraft(documentID string) (Draft, bool)
	SaveFlowDraft(Draft) error
}

type PublicationStore interface {
	GetFlowPublication(documentID string) (Publication, bool)
	SaveFlowPublication(Publication) error
}

type DraftPublicationStore interface {
	DraftStore
	PublicationStore
}

type NormalizeDocumentOption func(*normalizeDocumentOptions)

type normalizeDocumentOptions struct {
	DefaultBlockCatalog []blockstudio.Definition
	StepBlockCatalogs   map[string][]blockstudio.Definition
	PreserveUnknownStep bool
}

func WithStepBlockCatalog(stepKey string, catalog []blockstudio.Definition) NormalizeDocumentOption {
	return func(options *normalizeDocumentOptions) {
		if options.StepBlockCatalogs == nil {
			options.StepBlockCatalogs = map[string][]blockstudio.Definition{}
		}
		options.StepBlockCatalogs[normalizeKey(stepKey)] = cloneBlockCatalog(catalog)
	}
}

func WithDefaultStepBlockCatalog(catalog []blockstudio.Definition) NormalizeDocumentOption {
	return func(options *normalizeDocumentOptions) {
		options.DefaultBlockCatalog = cloneBlockCatalog(catalog)
	}
}

func WithUnknownStepBlocks() NormalizeDocumentOption {
	return func(options *normalizeDocumentOptions) {
		options.PreserveUnknownStep = true
	}
}

func StandardCatalog(handlerRefs HandlerRefs) []Definition {
	return Catalog(
		Contact(handlerRefs[FlowKeyContact]),
		PurchaseRequest(handlerRefs[FlowKeyPurchaseRequest]),
		CheckoutHandoff(handlerRefs[FlowKeyCheckoutHandoff]),
		Newsletter(handlerRefs[FlowKeyNewsletter]),
		Appointment(handlerRefs[FlowKeyAppointment]),
		ScheduleTour(handlerRefs[FlowKeyScheduleTour]),
		Enrollment(handlerRefs[FlowKeyEnrollment]),
	)
}

func StandardDocuments(handlerRefs HandlerRefs, options DocumentOptions) []Document {
	return DocumentsFromDefinitions(options, StandardCatalog(handlerRefs)...)
}

func DocumentsFromDefinitions(options DocumentOptions, definitions ...Definition) []Document {
	out := make([]Document, 0, len(definitions))
	for _, definition := range Catalog(definitions...) {
		out = append(out, DocumentFromDefinition(definition, options))
	}
	return out
}

func DocumentFromDefinition(definition Definition, options DocumentOptions) Document {
	definition = Normalize(definition)
	steps := make([]DocumentStep, 0, len(definition.Steps))
	for _, step := range definition.Steps {
		steps = append(steps, DocumentStep{
			Key:    step.Key,
			Label:  step.Label,
			Blocks: cloneBlockDocument(step.Blocks),
		})
	}
	actions := make([]DocumentAction, 0, len(definition.Actions))
	for _, action := range definition.Actions {
		actions = append(actions, documentActionFromAction(action))
	}
	return NormalizeDocument(Document{
		Version:     1,
		ID:          strings.TrimSpace(options.ID),
		Key:         definition.Key,
		Label:       definition.Label,
		Description: definition.Description,
		Steps:       steps,
		Actions:     actions,
		State:       options.State,
		Created:     options.Created,
		Updated:     options.Updated,
		Published:   options.Published,
	})
}

func NormalizeDocument(document Document, opts ...NormalizeDocumentOption) Document {
	options := normalizeDocumentOptions{}
	for _, opt := range opts {
		if opt != nil {
			opt(&options)
		}
	}
	document.Version = normalizeVersion(document.Version)
	document.ID = strings.TrimSpace(document.ID)
	document.Key = normalizeKey(firstNonEmpty(document.Key, document.Label))
	document.Label = firstNonEmpty(document.Label, document.Key)
	document.Description = strings.TrimSpace(document.Description)
	if document.State == "" {
		document.State = lifecycle.PublishStateDraft
	}
	document.Steps = normalizeDocumentSteps(document.Key, document.Steps, options)
	document.Actions = normalizeDocumentActions(document.Actions)
	return document
}

func ValidateDocument(document Document) ValidationErrors {
	return Validate(DefinitionFromDocument(document))
}

func DefinitionFromDocument(document Document) Definition {
	document = NormalizeDocument(document)
	steps := make([]Step, 0, len(document.Steps))
	for _, step := range document.Steps {
		steps = append(steps, Step{
			Key:    step.Key,
			Label:  step.Label,
			Blocks: cloneBlockDocument(step.Blocks),
		})
	}
	actions := make([]Action, 0, len(document.Actions))
	for _, action := range document.Actions {
		actions = append(actions, actionFromDocumentAction(action))
	}
	return Normalize(Definition{
		Key:         document.Key,
		Label:       document.Label,
		Description: document.Description,
		Steps:       steps,
		Actions:     actions,
	})
}

func InstanceFromDocument(document Document, options InstanceOptions) Instance {
	document = NormalizeDocument(document)
	state := options.State
	if state == "" {
		state = document.State
	}
	return Instance{
		ID:         strings.TrimSpace(options.ID),
		DocumentID: documentResourceID(document),
		FlowKey:    document.Key,
		Definition: DefinitionFromDocument(document),
		Source:     CloneDocument(document),
		RevisionID: strings.TrimSpace(options.RevisionID),
		State:      state,
		Created:    options.Created,
		Updated:    options.Updated,
	}
}

func NewDraft(document Document, authorID, baseRevisionID string, now time.Time) Draft {
	document = NormalizeDocument(document)
	document.State = lifecycle.PublishStateDraft
	document.Updated = timeOrExisting(now, document.Updated)
	return Draft{
		Document:       document,
		BaseRevisionID: strings.TrimSpace(baseRevisionID),
		AuthorID:       strings.TrimSpace(authorID),
		State:          lifecycle.DraftStateDraft,
		Updated:        document.Updated,
	}
}

func NewPublication(document Document, authorID, revisionID string, now time.Time) (Publication, error) {
	document = NormalizeDocument(document)
	if errs := ValidateDocument(document); len(errs) != 0 {
		return Publication{}, fmt.Errorf("flow document is invalid: %v", errs)
	}
	document.State = lifecycle.PublishStatePublished
	document.Updated = timeOrExisting(now, document.Updated)
	document.Published = timeOrExisting(now, document.Published)
	return Publication{
		Document:   document,
		RevisionID: strings.TrimSpace(revisionID),
		AuthorID:   strings.TrimSpace(authorID),
		Published:  document.Published,
	}, nil
}

func PublishDraft(draft Draft, authorID string, now time.Time) (PublishResult, error) {
	publication, err := NewPublication(draft.Document, authorID, "", now)
	if err != nil {
		return PublishResult{}, err
	}
	revision, err := NewDocumentRevision(publication.Document, ActionPublished, "Published flow document.", now)
	if err != nil {
		return PublishResult{}, err
	}
	publication.RevisionID = revision.ID
	return PublishResult{Publication: publication, Revision: revision}, nil
}

func ConfigureDocument(document Document, config DocumentConfig) Document {
	document = NormalizeDocument(document, WithUnknownStepBlocks())
	if len(config.HandlerRefs) == 0 && len(config.StepLabels) == 0 {
		return document
	}
	for index := range document.Steps {
		key := normalizeKey(document.Steps[index].Key)
		if label, ok := config.StepLabels[key]; ok {
			document.Steps[index].Label = strings.TrimSpace(label)
		}
	}
	for index := range document.Actions {
		key := normalizeKey(document.Actions[index].Key)
		if handlerRef, ok := config.HandlerRefs[key]; ok {
			document.Actions[index].HandlerRef = strings.TrimSpace(handlerRef)
		}
	}
	return NormalizeDocument(document, WithUnknownStepBlocks())
}

func SaveConfiguredDraft(store DraftStore, document Document, config DraftConfig) (Draft, error) {
	if store == nil {
		return Draft{}, fmt.Errorf("%w: %s", ErrFlowNotFound, normalizeKey(document.Key))
	}
	document = ConfigureDocument(document, DocumentConfig{
		HandlerRefs: config.HandlerRefs,
		StepLabels:  config.StepLabels,
	})
	draft := NewDraft(document, config.AuthorID, config.BaseRevisionID, config.Now)
	if err := store.SaveFlowDraft(draft); err != nil {
		return Draft{}, err
	}
	return draft, nil
}

func PublishStoredDraft(store DraftPublicationStore, documentID, authorID string, now time.Time) (PublishResult, error) {
	if store == nil {
		return PublishResult{}, fmt.Errorf("%w: %s", ErrFlowNotFound, normalizeKey(documentID))
	}
	draft, ok := store.GetFlowDraft(documentID)
	if !ok {
		return PublishResult{}, fmt.Errorf("%w: %s", ErrFlowNotFound, normalizeKey(documentID))
	}
	result, err := PublishDraft(draft, authorID, now)
	if err != nil {
		return PublishResult{}, err
	}
	if err := store.SaveFlowPublication(result.Publication); err != nil {
		return PublishResult{}, err
	}
	return result, nil
}

func NewDocumentRevision(document Document, action, summary string, now time.Time) (lifecycle.Revision, error) {
	document = NormalizeDocument(document)
	revision, err := lifecycle.NewRevision(lifecycle.RevisionInput{
		ID:            revisionID(document, now),
		ResourceKind:  ResourceKind,
		ResourceID:    documentResourceID(document),
		ResourceTitle: document.Label,
		Action:        action,
		Summary:       summary,
		Snapshot:      document,
	}, now)
	if err != nil {
		return lifecycle.Revision{}, err
	}
	return revision, nil
}

func DecodeDocumentRevision(revision lifecycle.Revision) (Document, error) {
	document, err := lifecycle.DecodeSnapshot[Document](revision)
	if err != nil {
		return Document{}, err
	}
	return NormalizeDocument(document), nil
}

func DocumentRevisionFilter(document Document, limit int) lifecycle.Filter {
	document = NormalizeDocument(document)
	return lifecycle.Filter{
		ResourceKind: ResourceKind,
		ResourceID:   documentResourceID(document),
		Limit:        limit,
	}
}

func CloneDocument(document Document) Document {
	document.Steps = cloneDocumentSteps(document.Steps)
	document.Actions = cloneDocumentActions(document.Actions)
	return document
}

func CloneDocuments(documents []Document) []Document {
	out := make([]Document, len(documents))
	for i, document := range documents {
		out[i] = CloneDocument(document)
	}
	return out
}

func StepDocumentKind(flowKey, stepKey string) string {
	flowKey = normalizeKey(flowKey)
	stepKey = normalizeKey(stepKey)
	if flowKey == "" {
		flowKey = "flow"
	}
	if stepKey == "" {
		stepKey = "step"
	}
	return DocumentKind + "." + flowKey + "." + stepKey
}

func normalizeDocumentSteps(flowKey string, steps []DocumentStep, options normalizeDocumentOptions) []DocumentStep {
	out := make([]DocumentStep, 0, len(steps))
	seen := map[string]bool{}
	for index, step := range steps {
		step.Key = normalizeKey(firstNonEmpty(step.Key, step.Label))
		if step.Key == "" {
			step.Key = "step"
		}
		if seen[step.Key] {
			step.Key = fmt.Sprintf("%s-%d", step.Key, index+1)
		}
		seen[step.Key] = true
		step.Label = firstNonEmpty(step.Label, step.Key)
		step.Blocks = normalizeStepBlockDocument(flowKey, step, options)
		out = append(out, step)
	}
	return out
}

func normalizeDocumentActions(actions []DocumentAction) []DocumentAction {
	out := make([]DocumentAction, 0, len(actions))
	seen := map[string]bool{}
	for _, action := range actions {
		action.Key = normalizeKey(firstNonEmpty(action.Key, action.Label))
		action.Label = firstNonEmpty(action.Label, action.Key)
		action.HandlerRef = strings.TrimSpace(action.HandlerRef)
		action.Fields = cloneFields(action.Fields)
		if action.Key == "" || seen[action.Key] {
			continue
		}
		seen[action.Key] = true
		out = append(out, action)
	}
	return out
}

func normalizeStepBlockDocument(flowKey string, step DocumentStep, options normalizeDocumentOptions) blockstudio.Document {
	doc := cloneBlockDocument(step.Blocks)
	doc.Version = normalizeVersion(doc.Version)
	doc.Kind = strings.TrimSpace(doc.Kind)
	if doc.Kind == "" {
		doc.Kind = StepDocumentKind(flowKey, step.Key)
	}
	catalog := options.DefaultBlockCatalog
	if options.StepBlockCatalogs != nil {
		if stepCatalog, ok := options.StepBlockCatalogs[normalizeKey(step.Key)]; ok {
			catalog = stepCatalog
		}
	}
	if len(catalog) > 0 {
		normalizeOpts := []blockstudio.NormalizeOption{}
		if options.PreserveUnknownStep {
			normalizeOpts = append(normalizeOpts, blockstudio.WithUnknownBlocks())
		}
		return blockstudio.NormalizeDocument(doc, catalog, normalizeOpts...)
	}
	return normalizeLooseBlockDocument(doc)
}

func normalizeLooseBlockDocument(doc blockstudio.Document) blockstudio.Document {
	doc.Version = normalizeVersion(doc.Version)
	doc.Kind = strings.TrimSpace(doc.Kind)
	blocks := make([]blockstudio.BlockInstance, 0, len(doc.Blocks))
	for index, block := range doc.Blocks {
		block.Key = normalizeKey(block.Key)
		if block.Key == "" {
			continue
		}
		block.ID = strings.TrimSpace(block.ID)
		if block.ID == "" {
			block.ID = fmt.Sprintf("%s-%d", block.Key, index+1)
		}
		if block.Order <= 0 {
			block.Order = index + 1
		}
		block.Values = cloneValues(block.Values)
		blocks = append(blocks, block)
	}
	sort.SliceStable(blocks, func(i, j int) bool {
		if blocks[i].Order == blocks[j].Order {
			return i < j
		}
		return blocks[i].Order < blocks[j].Order
	})
	for index := range blocks {
		blocks[index].Order = index + 1
	}
	doc.Blocks = blocks
	return doc
}

func documentActionFromAction(action Action) DocumentAction {
	return DocumentAction{
		Key:        action.Key,
		Label:      action.Label,
		HandlerRef: action.HandlerRef,
		Fields:     cloneFields(action.Fields),
	}
}

func actionFromDocumentAction(action DocumentAction) Action {
	return Action{
		Key:        action.Key,
		Label:      action.Label,
		HandlerRef: action.HandlerRef,
		Fields:     cloneFields(action.Fields),
	}
}

func cloneDocumentSteps(steps []DocumentStep) []DocumentStep {
	out := make([]DocumentStep, len(steps))
	for i, step := range steps {
		out[i] = DocumentStep{
			Key:    step.Key,
			Label:  step.Label,
			Blocks: cloneBlockDocument(step.Blocks),
		}
	}
	return out
}

func cloneDocumentActions(actions []DocumentAction) []DocumentAction {
	out := make([]DocumentAction, len(actions))
	for i, action := range actions {
		out[i] = DocumentAction{
			Key:        action.Key,
			Label:      action.Label,
			HandlerRef: action.HandlerRef,
			Fields:     cloneFields(action.Fields),
		}
	}
	return out
}

func cloneBlockDocument(doc blockstudio.Document) blockstudio.Document {
	doc.Blocks = cloneBlockInstances(doc.Blocks)
	return doc
}

func cloneBlockInstances(blocks []blockstudio.BlockInstance) []blockstudio.BlockInstance {
	out := make([]blockstudio.BlockInstance, len(blocks))
	for i, block := range blocks {
		block.Values = cloneValues(block.Values)
		out[i] = block
	}
	return out
}

func cloneValues(values blockstudio.Values) blockstudio.Values {
	if values == nil {
		return nil
	}
	out := make(blockstudio.Values, len(values))
	for key, value := range values {
		out[key] = cloneValue(value)
	}
	return out
}

func cloneValue(value blockstudio.Value) blockstudio.Value {
	value.List = cloneValueList(value.List)
	if value.Object != nil {
		object := make(map[string]blockstudio.Value, len(value.Object))
		for key, item := range value.Object {
			object[key] = cloneValue(item)
		}
		value.Object = object
	}
	if value.Media != nil {
		media := *value.Media
		value.Media = &media
	}
	if value.Relation != nil {
		relation := *value.Relation
		value.Relation = &relation
	}
	return value
}

func cloneValueList(values []blockstudio.Value) []blockstudio.Value {
	if values == nil {
		return nil
	}
	out := make([]blockstudio.Value, len(values))
	for i, value := range values {
		out[i] = cloneValue(value)
	}
	return out
}

func cloneFields(fields []workbench.Field) []workbench.Field {
	out := make([]workbench.Field, len(fields))
	for i, field := range fields {
		field.Options = append([]string(nil), field.Options...)
		out[i] = field
	}
	return out
}

func cloneBlockCatalog(catalog []blockstudio.Definition) []blockstudio.Definition {
	out := make([]blockstudio.Definition, len(catalog))
	copy(out, catalog)
	return out
}

func documentResourceID(document Document) string {
	if strings.TrimSpace(document.ID) != "" {
		return strings.TrimSpace(document.ID)
	}
	return normalizeKey(document.Key)
}

func revisionID(document Document, now time.Time) string {
	id := documentResourceID(document)
	if id == "" || now.IsZero() {
		return ""
	}
	return fmt.Sprintf("%s_%d", id, now.UnixNano())
}

func normalizeVersion(version int) int {
	if version <= 0 {
		return 1
	}
	return version
}

func timeOrExisting(value, existing time.Time) time.Time {
	if value.IsZero() {
		return existing
	}
	return value
}
