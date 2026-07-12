// Package conformance is a set of host-runnable Go conformance suites that
// prove a host's own adapter implementations satisfy Studio's cross-host
// contracts, without Studio ever importing the host and without the host
// needing to import any other host's code.
//
// Studio owns the contract shape (the interfaces, the wire protocol, the
// sequencing rules); each host owns exactly one adapter implementation of
// each contract, plus its own storage. This package packages up the exact
// scenarios Studio's own in-repo fake implementations are already tested
// against (cms/lifecycle/engine's fakeHost, and the DraftProjector pattern
// both current host adapters independently implement) into exported
// `RunXConformance(t, factory)` entry points a host calls from its own
// `_test.go` file, passing a factory that wires the host's real adapter.
//
// # Contracts covered
//
//   - HostLifecycle (cms/lifecycle/engine.HostLifecycle): the draft->publish/
//     restore seam every host implements for cms/lifecycle/engine.Engine.
//     See lifecycle.go / RunHostLifecycleConformance.
//   - DraftProjector: the collaboration ledger's draft-projection seam
//     (StudioTargetValue/ProjectStudioOperation/CanApplyStudioOperation/
//     QuarantineStudioOperation) every host implements to bridge
//     cms/studio/collab's accepted-operation outbox into its own draft store.
//     This interface is not yet declared inside cms/studio/collab itself --
//     it is a convention both current host adapters (Noni's
//     internal/studiohost.DraftProjector, Pajaritos'
//     *site.DurableAuthoringStore) independently, identically implement.
//     This package is where it is first formally, exportedly declared, so a
//     THIRD host has one canonical Go type to implement against instead of
//     re-deriving the convention by reading two other repos. See
//     draftprojector.go / RunDraftProjectorConformance.
//   - The durable authoring operation protocol (authoring.OperationRequest /
//     authoring.OperationRecord) for all 14 authoring.OperationKind values
//     (the 5 core kinds plus the 9 instance/interaction/flow kinds added in
//     HANDOFF-12): JSON stability, Validate, and the SetKindForTarget/
//     ClearsTarget/InverseRequest/CanonicalRequest invariants every ledger
//     implementation and DraftProjector adapter must agree on. This suite
//     takes no factory -- it exercises Studio's own authoring package
//     directly. See operationprotocol.go / RunOperationProtocolConformance.
//
// # How a host wires its factory
//
// A host adds one `_test.go` file (in its own repo, importing this package)
// per contract it wants proven, e.g.:
//
//	func TestGosxStudioHostLifecycleConformance(t *testing.T) {
//	    conformance.RunHostLifecycleConformance(t, func(t *testing.T) conformance.LifecycleFixture {
//	        store, closeStore, _ := lifecyclesql.Open(":memory:") // or the host's real store
//	        t.Cleanup(func() { _ = closeStore() })
//	        host := myhost.NewLifecycleAdapter(store, ...) // the host's real HostLifecycle
//	        return conformance.LifecycleFixture{
//	            Engine: &engine.Engine{Revisions: store, Ledger: store, Host: host},
//	            Target: engine.TargetRef{ResourceKind: "page", ResourceID: "conformance-fixture"},
//	            SeedLive:  host.SeedLiveForTest,
//	            SeedDraft: host.SeedDraftForTest,
//	            SeedBlockers: host.SeedBlockersForTest,
//	            CurrentLive: host.CurrentLiveForTest,
//	        }
//	    })
//	}
//
// See each Run*Conformance function's doc comment for its exact fixture
// contract, and the package's own *_selftest_test.go files for a complete,
// runnable example (a small in-memory fake standing in for a host adapter),
// proving every suite actually runs green inside this repo before any host
// adopts it.
package conformance
