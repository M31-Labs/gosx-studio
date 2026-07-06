package flows

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

var (
	ErrFlowNotFound    = errors.New("flow not found")
	ErrActionNotFound  = errors.New("flow action not found")
	ErrHandlerNotFound = errors.New("flow handler not found")
)

type ExecutableSource string

const (
	ExecutableSourcePublication ExecutableSource = "publication"
	ExecutableSourceDraft       ExecutableSource = "draft"
	ExecutableSourceDocument    ExecutableSource = "document"
)

type ExecutableFlow struct {
	Document   Document
	Definition Definition
	Source     ExecutableSource
	RevisionID string
}

type Submission struct {
	FlowKey             string
	DocumentID          string
	ActionKey           string
	Values              map[string]string
	UseDraftFallback    bool
	UseDocumentFallback bool
	Context             map[string]any
}

type Result struct {
	Flow        ExecutableFlow
	Action      DocumentAction
	HandlerRef  string
	Values      map[string]string
	FieldErrors ValidationErrors
	Output      any
}

type Handler func(context.Context, Submission) (any, error)

type Registry struct {
	handlers map[string]Handler
}

func NewRegistry() *Registry {
	return &Registry{handlers: map[string]Handler{}}
}

func (r *Registry) Register(ref string, handler Handler) {
	if r == nil || handler == nil {
		return
	}
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return
	}
	if r.handlers == nil {
		r.handlers = map[string]Handler{}
	}
	r.handlers[ref] = handler
}

func (r *Registry) Handler(ref string) (Handler, bool) {
	if r == nil {
		return nil, false
	}
	handler, ok := r.handlers[strings.TrimSpace(ref)]
	return handler, ok
}

type Executor struct {
	Documents    DocumentStore
	Drafts       DraftStore
	Publications PublicationStore
	Registry     *Registry
}

func ExecuteFlow(ctx context.Context, executor Executor, submission Submission) (Result, error) {
	return executor.Execute(ctx, submission)
}

func (e Executor) Execute(ctx context.Context, submission Submission) (Result, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	flow, err := e.Resolve(submission)
	if err != nil {
		return Result{}, err
	}
	action, ok := FindDocumentAction(flow.Document, submission.ActionKey)
	if !ok {
		return Result{Flow: flow}, fmt.Errorf("%w: %s", ErrActionNotFound, normalizeKey(submission.ActionKey))
	}
	values := cloneSubmissionValues(submission.Values)
	fieldErrors := ValidateActionPayload(action, values)
	result := Result{
		Flow:        flow,
		Action:      action,
		HandlerRef:  action.HandlerRef,
		Values:      values,
		FieldErrors: fieldErrors,
	}
	if len(fieldErrors) != 0 {
		return result, nil
	}
	handler, ok := e.Registry.Handler(action.HandlerRef)
	if !ok {
		return result, fmt.Errorf("%w: %s", ErrHandlerNotFound, action.HandlerRef)
	}
	submission.FlowKey = flow.Document.Key
	submission.DocumentID = documentResourceID(flow.Document)
	submission.ActionKey = action.Key
	submission.Values = values
	output, err := handler(ctx, submission)
	if err != nil {
		return result, err
	}
	result.Output = output
	return result, nil
}

func (e Executor) Resolve(submission Submission) (ExecutableFlow, error) {
	id := firstNonEmpty(submission.DocumentID, submission.FlowKey)
	flow, ok := ResolveExecutableFlowVersion(e.Publications, e.Drafts, e.Documents, id, submission.UseDraftFallback, submission.UseDocumentFallback)
	if !ok {
		return ExecutableFlow{}, fmt.Errorf("%w: %s", ErrFlowNotFound, normalizeKey(id))
	}
	return flow, nil
}

func ResolveExecutableFlow(publications PublicationStore, documents DocumentStore, id string, useDocumentFallback bool) (ExecutableFlow, bool) {
	return ResolveExecutableFlowVersion(publications, nil, documents, id, false, useDocumentFallback)
}

func ResolveExecutableFlowVersion(publications PublicationStore, drafts DraftStore, documents DocumentStore, id string, useDraftFallback, useDocumentFallback bool) (ExecutableFlow, bool) {
	id = normalizeKey(firstNonEmpty(id, strings.TrimSpace(id)))
	if id == "" {
		return ExecutableFlow{}, false
	}
	if publications != nil {
		if publication, ok := publications.GetFlowPublication(id); ok {
			document := NormalizeDocument(publication.Document, WithUnknownStepBlocks())
			return ExecutableFlow{
				Document:   document,
				Definition: DefinitionFromDocument(document),
				Source:     ExecutableSourcePublication,
				RevisionID: publication.RevisionID,
			}, true
		}
	}
	if useDraftFallback && drafts != nil {
		if draft, ok := drafts.GetFlowDraft(id); ok {
			document := NormalizeDocument(draft.Document, WithUnknownStepBlocks())
			return ExecutableFlow{
				Document:   document,
				Definition: DefinitionFromDocument(document),
				Source:     ExecutableSourceDraft,
				RevisionID: draft.BaseRevisionID,
			}, true
		}
	}
	if useDocumentFallback && documents != nil {
		if document, ok := documents.GetFlowDocument(id); ok {
			document = NormalizeDocument(document, WithUnknownStepBlocks())
			return ExecutableFlow{
				Document:   document,
				Definition: DefinitionFromDocument(document),
				Source:     ExecutableSourceDocument,
			}, true
		}
	}
	return ExecutableFlow{}, false
}

func FindDocumentAction(document Document, actionKey string) (DocumentAction, bool) {
	actionKey = normalizeKey(actionKey)
	document = NormalizeDocument(document, WithUnknownStepBlocks())
	for _, action := range document.Actions {
		if action.Key == actionKey {
			return action, true
		}
	}
	return DocumentAction{}, false
}

func DocumentCanExecute(document Document) bool {
	document = NormalizeDocument(document, WithUnknownStepBlocks())
	if len(document.Actions) == 0 {
		return false
	}
	if len(ValidateDocument(document)) != 0 {
		return false
	}
	for _, action := range document.Actions {
		if strings.TrimSpace(action.HandlerRef) == "" {
			return false
		}
	}
	return true
}

func ExecutableDocumentCount(documents []Document) int {
	count := 0
	for _, document := range documents {
		if DocumentCanExecute(document) {
			count++
		}
	}
	return count
}

func ValidateActionPayload(action DocumentAction, values map[string]string) ValidationErrors {
	actions := normalizeDocumentActions([]DocumentAction{action})
	if len(actions) == 0 {
		return ValidationErrors{}
	}
	action = actions[0]
	errs := ValidationErrors{}
	for _, field := range action.Fields {
		name := strings.TrimSpace(field.Name)
		if name == "" {
			continue
		}
		value := strings.TrimSpace(values[name])
		if field.Required && value == "" {
			errs[name] = "This field is required."
			continue
		}
		if value != "" && emailFieldName(name) && !strings.Contains(value, "@") {
			errs[name] = "Enter a valid email address."
		}
	}
	return errs
}

func emailFieldName(name string) bool {
	name = strings.TrimSpace(strings.ToLower(name))
	return name == "email" || strings.HasSuffix(name, "email")
}

func cloneSubmissionValues(values map[string]string) map[string]string {
	if values == nil {
		return map[string]string{}
	}
	out := make(map[string]string, len(values))
	for key, value := range values {
		out[key] = value
	}
	return out
}
