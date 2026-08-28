# Host integration guide

How a new host (a third GoSX site, beyond the Noni/Pajaritos reference pair)
integrates GoSX Studio's editor. Terse and factual — see
[`CROSS_HOST_OWNERSHIP_AUDIT.md`](CROSS_HOST_OWNERSHIP_AUDIT.md) for what
Studio owns vs. what a host implements, and
[`ARCHITECTURE.md`](ARCHITECTURE.md) for the in-module package DAG.

## Contents

1. [Module dependency](#1-module-dependency)
2. [Adapter surfaces to implement](#2-adapter-surfaces-to-implement)
   - [2.1 HostLifecycle](#21-hostlifecycle)
   - [2.2 DraftProjector](#22-draftprojector)
   - [2.3 Collaboration transport mount + auth](#23-collaboration-transport-mount--auth)
   - [2.4 Asset/media adapters](#24-assetmedia-adapters)
   - [2.5 Binding resolvers](#25-binding-resolvers)
   - [2.6 Interaction persistence](#26-interaction-persistence)
3. [Panel/runtime mount points](#3-panelruntime-mount-points)
4. [Conformance suites](#4-conformance-suites)
5. [E2E patterns to adopt](#5-e2e-patterns-to-adopt)
6. [Version/migration notes for this release line](#6-versionmigration-notes-for-this-release-line)

## 1. Module dependency

```
go get m31labs.dev/gosx-studio@<pseudo-version-or-tag>
go get m31labs.dev/gosx@<pseudo-version-or-tag>   # gosx-studio's own runtime dependency
```

- A normal consumer resolves the published modules without a `replace`
  directive or `go.work`. For a reproducible module-graph check, disable
  workspace discovery and inherited module flags explicitly:

  ```text
  env GOWORK=off GOFLAGS= go list -m -json m31labs.dev/gosx-studio
  ```

  Candidate replacements stay in the isolated host source copy described in
  §5; an inherited `GOFLAGS=-modfile=...` is insufficient for the GoSX CLI,
  which reconstructs `GOFLAGS` internally.
- **Verified Noni pair (2026-08-28):** Noni uses
  `m31labs.dev/gosx-studio v0.6.2-0.20260828060955-cb313609fb1d`, from Studio
  source commit `cb313609fb1d28937a110dab97698491a05e5030`, with GoSX runtime
  `v0.53.8`, at Noni checkpoint `532f8e4`. This is dated Noni evidence; do
  not infer a current Pajaritos pin or a latest release from it.
- A Studio upgrade may legitimately require exact rendered-markup golden
  updates and adapter changes. Review each diff against the intended contract;
  never blindly auto-accept changed goldens.
- `GOPRIVATE`/vanity-import redirect: `go get` fetches
  `m31labs.dev/gosx-studio` from `github.com/M31-Labs/gosx-studio` directly;
  no special `--build-context` or Docker vendoring is needed.

## 2. Adapter surfaces to implement

### 2.1 HostLifecycle

`m31labs.dev/gosx-studio/cms/lifecycle/engine.HostLifecycle` is the single
seam `cms/lifecycle/engine.Engine` calls for publish/restore/promote. Engine
owns sequencing (idempotency-by-`OperationID` replay, the readiness gate,
restore-point minting before every mutation, retention). Your host only
marshals its own state and runs its own transaction:

```go
type HostLifecycle interface {
    SnapshotLive(ctx context.Context, ref TargetRef) (snapshot any, ok bool, err error)
    DraftDiff(ctx context.Context, ref TargetRef) (diff lifecycle.RevisionDiff, hasDraft bool, err error)
    Readiness(ctx context.Context, ref TargetRef) ([]Blocker, error)
    ApplyPublish(ctx context.Context, cmd PublishCommand) (PublishOutcome, error)
    RestoreLive(ctx context.Context, cmd RestoreCommand) (RestoreOutcome, error)
    OperationLog(ctx context.Context, ref TargetRef, sinceRevision uint64) ([]authoring.OperationRecord, error)
    PublishedOperationRevision(ctx context.Context, ref TargetRef) (uint64, error)
}
```

Wire one `Engine` per host, sharing `Revisions`/`Ledger` (both satisfied by
`cms/lifecycle/sqlstore.Store` — open one instance, hand it to both `Engine`
and your `HostLifecycle` implementation so restore-point ids resolve against
the same store):

```go
store, closeStore, err := lifecyclesql.Open(dbPath)
host := myhost.NewLifecycleAdapter(store, myContentStore)
e := &engine.Engine{Revisions: store, Ledger: store, Host: host}
```

`ApplyPublish`/`RestoreLive` **MUST** be idempotent on `cmd.OperationID`
(Engine already de-dupes via ledger replay before calling either, but a host
MAY additionally de-dupe as defense-in-depth). If you maintain a companion
resource keyed by a target's content revision (an interactions
`ContentRevision`, a shared-component instance snapshot, ...), mint it keyed
by `cmd.RestorePointID` inside the same `RestoreLive` transaction — see
`RestoreCommand.RestorePointID`'s doc comment for the full "restore of a
restore" contract this closes.

Decide your target scope honestly. Pajaritos wires exactly one
(`ResourceKindSiteSettings`/`"site"`) and leaves pages on a legacy publish
path; Noni wires per-page. Both are valid — do not fake broader lifecycle
coverage than your host actually has.

Prove your adapter with `conformance.RunHostLifecycleConformance` (§4)
before wiring it into production routes.

### 2.2 DraftProjector

The collaboration ledger (`cms/studio/collab`) accepts an operation, then
calls your host back to project it into your own draft store. This
interface is not (yet) declared inside `cms/studio/collab` itself — it is
formally declared, once, in `conformance.DraftProjector`
(`m31labs.dev/gosx-studio/conformance`), matching what both reference hosts
already independently implement:

```go
type DraftProjector interface {
    StudioTargetValue(authoring.OperationTarget) (authoring.OperationValue, error)
    ProjectStudioOperation(authoring.OperationRecord) error
    CanApplyStudioOperation(authoring.OperationRequest) error
    QuarantineStudioOperation(record authoring.OperationRecord, reason string) error
}
```

Contract your implementation must satisfy (see `conformance/draftprojector.go`
doc comments for the exact scenarios, and
[§B in the ownership audit](CROSS_HOST_OWNERSHIP_AUDIT.md) for why this
matters — it is the "P1" outbox-wedge fix both hosts already carry):

- `ProjectStudioOperation` is idempotent by `record.ID` (exact replay is a
  no-op); the same id with different content is rejected, not re-applied.
- `ProjectStudioOperation` rejects (no mutation) a record whose `Before`
  diverges from the target's actual current value — this is the case your
  outbox-drain loop (below) should quarantine, not endlessly retry.
- `CanApplyStudioOperation` is a pure dry run: never mutates state, shares
  its validation with `ProjectStudioOperation` so pre-accept and post-accept
  can never diverge.
- `QuarantineStudioOperation` durably records a poisoned operation for audit
  without applying its value.

You will also need the **outbox-drain wrapper** around
`collab.Service`/`collab.OperationStore` that calls your `DraftProjector` as
the ledger's accepted operations arrive (`Submit`/`Resume`/background
reconcile). This wrapper (`ProjectingService` in both reference hosts'
`internal/studiohost/collaboration.go`) is **not yet Studio-owned** — see the
ownership audit's verdict for exactly why (one host-specific decision,
classifying which projection errors are retryable vs. quarantine-worthy,
currently lives in each host's own error sentinels). Until Studio exports a
generic version, write your own following the reference hosts' shape:
`Submit` reconciles any backlog, then dry-runs via `CanApplyStudioOperation`
for concrete-value kinds (`set-field`/`set-style`/`reset-style`), then seeds
+ submits to the ledger, then reconciles again and surfaces a same-call
quarantine as a failure to the submitting actor (not a false-positive
success).

Prove your `DraftProjector` with `conformance.RunDraftProjectorConformance`
(§4) — start with `conformance.CoreTargetCases()` (the 5 core kinds always
apply); add `conformance.ExtendedTargetCases()` once you have wired the 9
HANDOFF-12 instance/interaction/flow kinds through your projector (Pajaritos
has; Noni has not yet — see the ownership audit).

### 2.3 Collaboration transport mount + auth

Mount `cms/studio/collab.Transport` (a `http.Handler`) on your own
authenticated route, e.g. `/admin/editor/collab`:

```go
service, _ := collab.NewService(resourceKey, ledgerStore)         // or your DraftProjector-wrapping ProjectingService
transport, _ := collab.NewTransport(collab.TransportOptions{
    Service:      projectingService, // must satisfy collab.TransportService
    Authenticate: myPrincipalResolver, // func(*http.Request) (collab.Principal, error)
})
mux.Handle("/admin/editor/collab", sessionMiddleware(transport))
```

`Authenticate` is where host-specific authorization happens — nonce
validation + signed-session lookup + role→capability mapping. Both reference
hosts nonce-gate the WebSocket dial (`EnsureCollaborationNonce`/
`ValidateCollaborationNonce` in their own `internal/studiohost`) so a stolen
`ws://` URL alone can't join; only your own signed session can mint a valid
nonce. Map your roles to `collab.Capability{View,Author,Design,Publish}`
deliberately and honestly — Noni has 4 granular roles, Pajaritos has 1
(`admin` gets every capability). Never grant `collab.CapabilityRepair`
through this resolver; mint it only inside your own privileged
recovery/repair code path (see `OperationStore.RepairFieldHead`).

Render `panels.RenderCollaborationPanel(panels.CollaborationPanelOptions{
Endpoint: <nonce-bearing URL>, Resource: resourceKey.StableKey()})` in your
editor page; it renders nothing meaningful without a mounted transport and a
signed session to mint a nonce from (render conditionally).

### 2.4 Asset/media adapters

Implement `cms/media.Store` (+ `cms/media.AssetDeleter` if you support
deletion) over your own file/object storage. Reuse Studio's own primitives
rather than re-deriving them: `cms/media.DefaultUploadPolicy` (validation),
`cms/media.GuardDeleteMediaAsset` (referenced-deletion guard),
`cms/media.AssetReadiness` (alt-text-required-only-when-referenced),
`cms/media.NormalizeFocalPoint`. Pajaritos had zero media substrate before
adopting this surface (HANDOFF-05/06) — a host with no asset-referencing
content model yet can legitimately skip this until it has one.

### 2.5 Binding resolvers

Implement `authoring.ResourceBindingAdapter` per `core.ResourceKind` your
components can bind to (CMS collection, media, flow, ...):

```go
type ResourceBindingAdapter interface {
    Kind() core.ResourceKind
    List(ctx context.Context, query string) ([]ResourceBindingOption, error)
    Validate(ctx context.Context, binding string) (resolved bool, reason string, err error)
}
```

Index them in an `authoring.ResourceBindingAdapters` map and build a
`core.BindingResolver` via `authoring.SiteMapBindingResolver(ctx, adapters,
componentResourceKind)` for `core.SiteMap.BindingDiagnostics` — this is what
feeds `core.BrokenBindingDiagnostics` into your `HostLifecycle.Readiness`
blockers.

### 2.6 Interaction persistence

Studio owns `core.Interaction`'s schema/allow-lists/validation
(`core/interactions.go`) and the 2 durable interaction operation kinds
(`set-interaction`/`remove-interaction`, `authoring.FieldInteractionsEntry`).
You own storage, keyed by whatever component-identity convention you already
use, plus a **draft/live split**: both reference hosts keep a
`LiveInteractions` set distinct from the draft-editable set, plus an
`InteractionsLiveMigrated`-style one-time migration guard for existing data
(HANDOFF-15) — copy this pattern; without it, an unpublished interaction edit
leaks straight to the public site. Publish your interactions store through
`HostLifecycle.ApplyPublish` (copy `Live* = clone(Draft*)`), and provide a
matching `RestoreLiveInteractions`-style hook so `RestoreLive` can roll them
back too.

## 3. Panel/runtime mount points

Studio ships pre-built server-rendered panels (`panels.RenderXPanel(...)
gosx.Node`) and client runtimes (`<pkg>runtime.Script() []byte`,
`hostruntime.StylesheetPath`) — mount by import + call, never by copying the
JS/CSS into your own tree (the ownership audit's grep evidence confirms
neither reference host does this). Link the stylesheet once per admin page
and inline each runtime script your mounted panels need, e.g.:

```go
ctx.AddHead(gosx.El("link", gosx.Attrs(gosx.Attr("rel", "stylesheet"), gosx.Attr("href", studio.StylesheetPath))))
ctx.AddHead(gosx.El("script", nil, gosx.RawHTML(string(interactionruntime.Script()))))
```

Relevant panels for a full editor mount: `RenderDirectEditPanel`,
`RenderResponsiveLayoutInspector`, `RenderStylePanel`,
`RenderCollaborationPanel`, `RenderInteractionsPanel`,
`RenderFlowFieldEditorPanel`, `RenderPublishPanel`,
`RenderRevisionHistoryPanel`, `RenderAdvancedPanel` family. Grep
`panels/*.go` for `^func Render` for the full current list; each panel's
`*Options` struct documents what it needs from your host.

## 4. Conformance suites

`m31labs.dev/gosx-studio/conformance` packages the exact scenarios Studio's
own in-repo fakes are tested against into `RunXConformance(t, factory)` entry
points you call from your own `_test.go`:

- `conformance.RunHostLifecycleConformance(t, factory)` — §2.1.
- `conformance.RunDraftProjectorConformance(t, factory, cases)` — §2.2.
- `conformance.RunOperationProtocolConformance(t)` — no factory; proves the
  wire protocol itself (all 14 `authoring.OperationKind` values) is
  JSON-stable, independent of your adapter.

See the package's own `doc.go` for the fixture wiring pattern and
`*_selftest_test.go` files for a complete runnable example.

## 5. E2E patterns to adopt

The reference-app browser matrix lives in `gosx-studio/parity_e2e/
reference_apps_*_test.ts`, runs against real booted hosts selected by
`.github/reference-app-refs.json`, and is invoked by the explicit file list in
`parity_e2e/package.json`'s `test:reference-apps` script. That current list
includes the collaboration, interactions, and media-asset fixtures, as well
as `reference_apps_enterprise_polish_test.ts`,
`reference_apps_modern_editor_test.ts`, and
`reference_apps_page_cms_lifecycle_test.ts`, among the other reference-app
contracts. Treat that script and the checked-in workflow as the coverage
source of truth; do not use obsolete omission counts.

To add your host to this matrix:

1. Add a `GOSX_STUDIO_<HOST>_REPO` env var + host-boot wiring to
   `parity_e2e/reference_apps_harness.ts` (follow the existing
   `GOSX_STUDIO_MUDDY_REPO`/`GOSX_STUDIO_PAJARITOS_REPO` pattern).
2. Add your host's ref to `.github/reference-app-refs.json`.
3. Review every current fixture in the `test:reference-apps` list for host
   setup; no per-host test files are needed when the shared harness is enough.

Validation distinguishes two module identities. In the candidate lane,
`.github/workflows/reference-apps.yml` checks out Studio and exports
`GOSX_STUDIO_CANDIDATE_REPO` plus `GOSX_STUDIO_CANDIDATE_SHA`. The harness makes
an isolated sibling source copy of each host, adds a candidate-only
`replace` to that copy's `go.mod`, and requires Go's `Module.Replace.Dir` to
resolve to the checked-out candidate while the checkout SHA matches the
expected SHA. The untouched host checkout is not the candidate module graph.
That proof must remain distinct from a separately verified normal consumer
lane using the published Studio pseudo-version with no `replace` and no
`go.work`.

The `npm run test:shared-runtime` lane uses
`playwright.enterprise.config.ts` and its three explicit browser projects
(Chromium, Firefox, WebKit) for standalone shared-runtime fixtures; it does
not boot a reference host. The `npm run test:reference-apps` host lane uses
the normal Playwright configuration, boots the registered real hosts, and
currently runs the host Chromium project. A three-engine shared-runtime pass
is therefore not three-engine real-host coverage.

Remote CI has not been run for this documentation refresh, and an automatic
published-module/no-overlay host lane is still missing. Keep candidate
source-copy results, normal published-pin proof, shared-runtime results, and
host-browser results reported separately.

## 6. Version/migration notes for this release line

Additive-only migrations a host adopting this release line should know
about (all backward compatible; no existing host had to migrate data to stay
on an older shape):

- **`authoring.OperationKind` gained 9 new durable kinds** (HANDOFF-12):
  `set-shared-field`, `override-instance`, `detach-instance`,
  `restore-instance`, `set-interaction`, `remove-interaction`,
  `set-flow-field`, `remove-flow-field`, `set-flow-action` — alongside the 5
  core kinds (`set-field`/`set-style`/`reset-style`/`undo`/`redo`), for 14
  total. `authoring.OperationTarget` gained a `ControlKey` field for these.
  A host may adopt some or none of the 9 — see the ownership audit's
  Noni/Pajaritos asymmetry for what "adopt" concretely means (ledgering
  through `DraftProjector`, not just editing the value some other way).
- **`LiveInteractions`/`InteractionsLiveMigrated`-shaped draft/live split**
  (HANDOFF-15, "P1" review fix): if you already had a non-drafted
  interactions store, add a live-side field and a one-time migration flag
  that backfills it from existing data exactly once. Both reference hosts'
  migration is idempotent (safe to run again on an already-migrated store).
- **Collaboration ledger sqlite tables** (`cms/studio/collab/sqlstore`):
  `studio_resources`, `studio_field_heads`, `studio_operation_attempts`,
  `studio_projection_outbox`, plus HANDOFF-12's additive
  `studio_field_head_repairs` table (`OperationStore.RepairFieldHead`/
  `RepairHistory`) — entirely Studio-owned; a host only opens the store at a
  path of its choosing, never creates or migrates these tables itself.
- **Quarantine fields**: a `DraftProjector` implementation should carry its
  own durable quarantine record type (Noni: `StudioQuarantinedOperation`;
  Pajaritos: its own equivalent) — this is host-owned storage, not a Studio
  migration, but every host adopting `QuarantineStudioOperation` needs one.
- **Shared-component stores**: `core/instances.go`'s
  `ComponentDefinition`/effective-controls model is additive; a host with no
  existing component-instance concept starts from zero (Pajaritos did, in
  HANDOFF-05/06) rather than migrating anything.
- **Revision `ActionRestorePoint` kinds**: `cms/store.ActionRestorePoint`
  (`= ActionRestorePrePub`) and `cms/store.ActionRestoreApplied` are the two
  `Revision.Action` values `cms/lifecycle/engine.Engine` mints/expects; a
  host's `RevisionStore` (typically `cms/lifecycle/sqlstore.Store`, shared
  verbatim) needs no schema change to support them — they are just action
  string values on the existing `Revision` row.
