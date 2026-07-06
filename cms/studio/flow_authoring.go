package studio

import (
	"fmt"
	"strings"
	"time"

	cmsflows "m31labs.dev/gosx-studio/cms/flows"
)

type FlowDocumentDraftStore interface {
	cmsflows.DocumentStore
	cmsflows.DraftStore
}

type FlowAuthoringStore interface {
	FlowDocumentDraftStore
	cmsflows.PublicationStore
}

type StudioFlowLibraryBuildOptions struct {
	Store        FlowAuthoringStore
	Routes       map[string]string
	EmbedTargets map[string]string
}

type FlowDraftSaveOptions struct {
	AuthorID       string
	BaseRevisionID string
	Now            time.Time
}

type FlowPublishOptions struct {
	FlowKey  string
	AuthorID string
	Now      time.Time
}

func BuildStudioFlowLibrary(definitions []cmsflows.Definition, options StudioFlowLibraryBuildOptions) []cmsflows.StudioFlow {
	return cmsflows.StudioLibrary(ConfiguredFlowDefinitions(options.Store, definitions), cmsflows.StudioLibraryOptions{
		Routes:       options.Routes,
		EmbedTargets: options.EmbedTargets,
	})
}

func ConfiguredFlowDefinitions(store FlowAuthoringStore, definitions []cmsflows.Definition) []cmsflows.Definition {
	definitions = cmsflows.Catalog(definitions...)
	if store == nil {
		return definitions
	}
	out := make([]cmsflows.Definition, 0, len(definitions))
	for _, definition := range definitions {
		definition = cmsflows.Normalize(definition)
		if draft, ok := store.GetFlowDraft(definition.Key); ok {
			out = append(out, cmsflows.DefinitionFromDocument(draft.Document))
			continue
		}
		if publication, ok := store.GetFlowPublication(definition.Key); ok {
			out = append(out, cmsflows.DefinitionFromDocument(publication.Document))
			continue
		}
		if document, ok := store.GetFlowDocument(definition.Key); ok {
			out = append(out, cmsflows.DefinitionFromDocument(document))
			continue
		}
		out = append(out, definition)
	}
	return cmsflows.Catalog(out...)
}

func SaveConfiguredFlowDrafts(store FlowDocumentDraftStore, definitions []cmsflows.Definition, form map[string]string, options FlowDraftSaveOptions) error {
	if store == nil {
		return nil
	}
	for _, definition := range cmsflows.Catalog(definitions...) {
		if _, err := SaveConfiguredFlowDraft(store, definition, form, options); err != nil {
			return err
		}
	}
	return nil
}

func SaveConfiguredFlowDraft(store FlowDocumentDraftStore, definition cmsflows.Definition, form map[string]string, options FlowDraftSaveOptions) (cmsflows.Draft, error) {
	if store == nil {
		return cmsflows.Draft{}, fmt.Errorf("flow store is not available")
	}
	definition = cmsflows.Normalize(definition)
	if definition.Key == "" {
		return cmsflows.Draft{}, fmt.Errorf("flow definition is missing a key")
	}
	now := flowAuthoringTime(options.Now)
	document := cmsflows.DocumentFromDefinition(definition, cmsflows.DocumentOptions{ID: definition.Key})
	if err := store.SaveFlowDocument(document); err != nil {
		return cmsflows.Draft{}, err
	}
	return cmsflows.SaveConfiguredDraft(store, document, cmsflows.DraftConfig{
		AuthorID:       firstNonEmpty(options.AuthorID, "studio"),
		BaseRevisionID: strings.TrimSpace(options.BaseRevisionID),
		HandlerRefs:    FlowHandlerRefs(definition, form),
		StepLabels:     FlowStepLabels(definition, form),
		Now:            now,
	})
}

func PublishConfiguredFlow(store FlowAuthoringStore, definitions []cmsflows.Definition, form map[string]string, options FlowPublishOptions) (cmsflows.PublishResult, error) {
	if store == nil {
		return cmsflows.PublishResult{}, fmt.Errorf("flow store is not available")
	}
	flowKey := normalizeKey(options.FlowKey)
	definition, ok := cmsflows.Find(definitions, flowKey)
	if !ok {
		return cmsflows.PublishResult{}, fmt.Errorf("flow was not found: %s", flowKey)
	}
	now := flowAuthoringTime(options.Now)
	if _, err := SaveConfiguredFlowDraft(store, definition, form, FlowDraftSaveOptions{
		AuthorID: firstNonEmpty(options.AuthorID, "studio"),
		Now:      now,
	}); err != nil {
		return cmsflows.PublishResult{}, err
	}
	return cmsflows.PublishStoredDraft(store, definition.Key, firstNonEmpty(options.AuthorID, "studio"), now)
}

func FlowHandlerRefs(definition cmsflows.Definition, form map[string]string) map[string]string {
	definition = cmsflows.Normalize(definition)
	refs := map[string]string{}
	if len(definition.Actions) == 0 {
		return refs
	}
	name := FlowHandlerRefInputName(definition.Key)
	if value, ok := form[name]; ok {
		refs[definition.Actions[0].Key] = value
	}
	return refs
}

func FlowStepLabels(definition cmsflows.Definition, form map[string]string) map[string]string {
	definition = cmsflows.Normalize(definition)
	labels := map[string]string{}
	for _, step := range definition.Steps {
		name := FlowStepLabelInputName(definition.Key, step.Key)
		if value, ok := form[name]; ok {
			labels[step.Key] = value
		}
	}
	return labels
}

func flowAuthoringTime(value time.Time) time.Time {
	if value.IsZero() {
		return time.Now().UTC()
	}
	return value.UTC()
}
