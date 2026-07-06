package lifecycle

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

type RevisionChangeKind string

const (
	RevisionChangeAdded   RevisionChangeKind = "added"
	RevisionChangeRemoved RevisionChangeKind = "removed"
	RevisionChangeChanged RevisionChangeKind = "changed"
)

type RevisionDiff struct {
	FromRevisionID string           `json:"fromRevisionId,omitempty"`
	ToRevisionID   string           `json:"toRevisionId,omitempty"`
	ResourceKind   string           `json:"resourceKind,omitempty"`
	ResourceID     string           `json:"resourceId,omitempty"`
	Summary        string           `json:"summary"`
	Changes        []RevisionChange `json:"changes,omitempty"`
}

type RevisionChange struct {
	Path   string             `json:"path"`
	Kind   RevisionChangeKind `json:"kind"`
	Before string             `json:"before,omitempty"`
	After  string             `json:"after,omitempty"`
}

func DiffRevisions(from, to Revision) (RevisionDiff, error) {
	fromValues, err := flattenSnapshot(from.Snapshot)
	if err != nil {
		return RevisionDiff{}, fmt.Errorf("decode from revision %q: %w", strings.TrimSpace(from.ID), err)
	}
	toValues, err := flattenSnapshot(to.Snapshot)
	if err != nil {
		return RevisionDiff{}, fmt.Errorf("decode to revision %q: %w", strings.TrimSpace(to.ID), err)
	}
	diff := RevisionDiff{
		FromRevisionID: strings.TrimSpace(from.ID),
		ToRevisionID:   strings.TrimSpace(to.ID),
		ResourceKind:   firstNonEmpty(to.ResourceKind, from.ResourceKind),
		ResourceID:     firstNonEmpty(to.ResourceID, from.ResourceID),
	}
	keys := map[string]bool{}
	for key := range fromValues {
		keys[key] = true
	}
	for key := range toValues {
		keys[key] = true
	}
	ordered := make([]string, 0, len(keys))
	for key := range keys {
		ordered = append(ordered, key)
	}
	sort.Strings(ordered)
	for _, key := range ordered {
		before, hadBefore := fromValues[key]
		after, hasAfter := toValues[key]
		switch {
		case !hadBefore && hasAfter:
			diff.Changes = append(diff.Changes, RevisionChange{Path: key, Kind: RevisionChangeAdded, After: after})
		case hadBefore && !hasAfter:
			diff.Changes = append(diff.Changes, RevisionChange{Path: key, Kind: RevisionChangeRemoved, Before: before})
		case before != after:
			diff.Changes = append(diff.Changes, RevisionChange{Path: key, Kind: RevisionChangeChanged, Before: before, After: after})
		}
	}
	diff.Summary = RevisionDiffSummary(diff.Changes)
	return diff, nil
}

func RevisionDiffSummary(changes []RevisionChange) string {
	if len(changes) == 0 {
		return "No changes."
	}
	added, removed, changed := 0, 0, 0
	for _, change := range changes {
		switch change.Kind {
		case RevisionChangeAdded:
			added++
		case RevisionChangeRemoved:
			removed++
		default:
			changed++
		}
	}
	parts := []string{}
	if changed > 0 {
		parts = append(parts, pluralize(changed, "changed field", "changed fields"))
	}
	if added > 0 {
		parts = append(parts, pluralize(added, "added field", "added fields"))
	}
	if removed > 0 {
		parts = append(parts, pluralize(removed, "removed field", "removed fields"))
	}
	return strings.Join(parts, ", ") + "."
}

func flattenSnapshot(raw json.RawMessage) (map[string]string, error) {
	if len(raw) == 0 {
		return nil, fmt.Errorf("snapshot is empty")
	}
	var value any
	if err := json.Unmarshal(raw, &value); err != nil {
		return nil, err
	}
	out := map[string]string{}
	flattenValue("", value, out)
	return out, nil
}

func flattenValue(path string, value any, out map[string]string) {
	switch typed := value.(type) {
	case map[string]any:
		if len(typed) == 0 {
			out[pathOrRoot(path)] = "{}"
			return
		}
		keys := make([]string, 0, len(typed))
		for key := range typed {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			flattenValue(joinPath(path, key), typed[key], out)
		}
	case []any:
		if len(typed) == 0 {
			out[pathOrRoot(path)] = "[]"
			return
		}
		for index, item := range typed {
			flattenValue(fmt.Sprintf("%s[%d]", pathOrRoot(path), index), item, out)
		}
	default:
		out[pathOrRoot(path)] = revisionValueString(typed)
	}
}

func revisionValueString(value any) string {
	switch typed := value.(type) {
	case nil:
		return "null"
	case string:
		return typed
	case bool:
		if typed {
			return "true"
		}
		return "false"
	default:
		data, err := json.Marshal(typed)
		if err != nil {
			return fmt.Sprint(typed)
		}
		return string(data)
	}
}

func joinPath(prefix, key string) string {
	key = strings.TrimSpace(key)
	if prefix == "" {
		return key
	}
	if key == "" {
		return prefix
	}
	return prefix + "." + key
}

func pathOrRoot(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return "$"
	}
	return path
}

func pluralize(count int, singular, plural string) string {
	label := plural
	if count == 1 {
		label = singular
	}
	return fmt.Sprintf("%d %s", count, label)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
