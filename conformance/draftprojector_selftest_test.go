package conformance_test

// This file proves conformance.RunDraftProjectorConformance actually runs,
// end to end, against a tiny in-memory conformance.DraftProjector
// implementation standing in for a real host's collaboration draft store
// (e.g. Noni's internal/cms.Store, Pajaritos' *site.DurableAuthoringStore).
// See conformance/doc.go for the wiring pattern a real host uses instead of
// this self-test's in-memory fake.

import (
	"errors"
	"sync"
	"testing"

	"m31labs.dev/gosx-studio/authoring"
	"m31labs.dev/gosx-studio/conformance"
)

var errSelfTestIdempotency = errors.New("conformance selftest: operation id reused with different content")
var errSelfTestDiverged = errors.New("conformance selftest: before value diverged from current draft")

// selfTestProjector is a minimal, generic (kind-agnostic) in-memory
// conformance.DraftProjector: it keys every target by its TargetKey and
// stores whatever After value was last projected, with no per-kind business
// rules of its own. A real host's projector additionally interprets the
// value against its own domain model (a page's field, a shared component's
// control, ...) -- this self-test only needs to prove the generic
// idempotency/divergence/quarantine mechanics the conformance suite checks.
type selfTestProjector struct {
	mu         sync.Mutex
	values     map[string]authoring.OperationValue
	byID       map[string]authoring.OperationRecord
	quarantine []authoring.OperationRecord
}

func newSelfTestProjector() *selfTestProjector {
	return &selfTestProjector{values: map[string]authoring.OperationValue{}, byID: map[string]authoring.OperationRecord{}}
}

func (p *selfTestProjector) StudioTargetValue(target authoring.OperationTarget) (authoring.OperationValue, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.values[target.Key(authoring.OperationSetField)], nil
}

func (p *selfTestProjector) ProjectStudioOperation(record authoring.OperationRecord) error {
	record = record.Normalize()
	p.mu.Lock()
	defer p.mu.Unlock()
	key := record.Target.Key(authoring.OperationSetField)
	if prior, ok := p.byID[record.ID]; ok {
		if recordsMatch(prior, record) {
			return nil // idempotent replay
		}
		return errSelfTestIdempotency
	}
	current := p.values[key]
	if current != record.Before {
		return errSelfTestDiverged
	}
	p.values[key] = record.After
	p.byID[record.ID] = record
	return nil
}

func (p *selfTestProjector) CanApplyStudioOperation(req authoring.OperationRequest) error {
	req = req.Normalize()
	p.mu.Lock()
	defer p.mu.Unlock()
	// Dry run only: read current state, never write it.
	_ = p.values[req.Target.Key(authoring.OperationSetField)]
	return nil
}

func (p *selfTestProjector) QuarantineStudioOperation(record authoring.OperationRecord, reason string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.quarantine = append(p.quarantine, record.Normalize())
	return nil
}

func recordsMatch(a, b authoring.OperationRecord) bool {
	return a.Kind == b.Kind && a.Target == b.Target && a.Before == b.Before && a.After == b.After
}

func TestSelfDraftProjectorConformanceCore(t *testing.T) {
	conformance.RunDraftProjectorConformance(t, func(t *testing.T) conformance.DraftProjectorFixture {
		return conformance.DraftProjectorFixture{Projector: newSelfTestProjector()}
	}, conformance.CoreTargetCases())
}

func TestSelfDraftProjectorConformanceExtended(t *testing.T) {
	conformance.RunDraftProjectorConformance(t, func(t *testing.T) conformance.DraftProjectorFixture {
		return conformance.DraftProjectorFixture{Projector: newSelfTestProjector()}
	}, conformance.ExtendedTargetCases())
}
