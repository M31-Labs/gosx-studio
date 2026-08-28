# Cross-host ownership audit

Scope: verify what Studio owns vs. what each reference host implements, for
every product surface the Noni-handoff roadmap touches, and check hosts for
copied Studio engine/runtime code. Hosts audited: `muddy-noni-commerce`
(branch `codex/noni-pagecanvas-release`) and `pajaritos-forest-school`
(branch `codex/pajaritos-pagecanvas-release`), both read-only for this audit.
Both pin `m31labs.dev/gosx-studio v0.6.2-0.20260712025550-fcf3957aebcc` and
`m31labs.dev/gosx v0.29.6-0.20260711234809-cac23fe9ead5`, no `replace`
directive, no `go.work`.

Method: grep both host trees for Studio's own internal function/type/table
names (`OperationChangeSet`, `isExactInverse`, `classifyOperationScope`,
`mintRestorePoint`, `replayPublish`/`replayRestore`, `resolveHistory`,
`studio_field_heads`/`studio_projection_outbox`/`studio_operation_attempts`,
`package engine`/`package sqlstore`/`package canvas`/`package panels`/
`package workbenchruntime`), and check that public-facing pages reference
`studio.StylesheetPath`/`interactionruntime.Script()` (imported) rather than
copied JS/CSS. Cross-referenced against the slice reports in
`.tiller/scratch/codex/handoff-*.md` for what each host actually wired.

## Verdict: no copied Studio engine/runtime code in either host

Zero hits for any Studio-internal engine/render/ledger symbol or SQL table
name as an actual implementation in either host tree. The only 2 hits for a
ledger table name (`studio_field_heads`) are doc-comment prose describing the
*consequence* of quarantining an operation, not a re-implementation of the
table:

- `muddy-noni-commerce/internal/cms/studio_operations.go:129`
- `pajaritos-forest-school/internal/site/durable_authoring.go:700`

Both hosts correctly consume Studio's canvas/panel/runtime rendering by
import + call (`studio.StylesheetPath`, `interactionruntime.Script()`), never
by copying JS/CSS:

- `muddy-noni-commerce/cmd/muddy-noni/main.go:162,187`
- `pajaritos-forest-school/cmd/pajaritos/main.go:113,132`

**One real cross-host duplication exists, but it is not copied-from-Studio
code** (Studio does not own or export this type today): both hosts
independently wrote a near-identical ~400-line `ProjectingService` /
`DraftProjector` orchestration layer in their own `internal/studiohost/
collaboration.go` (Noni: 391 lines / 20 funcs; Pajaritos: 431 lines / 21
funcs). Pajaritos' own wiring report says this plainly: "ported verbatim in
architecture from Noni's" (`handoff-0506-pajaritos-wiring-report.md`). A
line-diff shows identical structure/logic (`reconcileLocked`,
`nonRetryableProjectionError`, `requiresPreAcceptValidation`,
`quarantineReasonSince`, `protocolFailure`/`invalidOperationFailure`), differing
only in host-specific naming and which host package's error sentinels
`nonRetryableProjectionError` checks (`cms.ErrStudio*` vs `site.Err*`). This
is the one architectural gap this slice's conformance package (`conformance/`)
partially closes (see below) — it formalizes the `DraftProjector` *interface*
both hosts already implement so a third host has one canonical Go type
instead of re-deriving the convention from two other repos, and proves it
against in-repo fakes. It does **not** yet promote `ProjectingService` itself
(the outbox-drain loop, not just the storage-adapter interface) into Studio,
because that loop's one host-specific decision — is this projection error
retryable or quarantine-worthy? — is currently answered by each host's own
error sentinels, not by anything Studio exports. Closing that gap (e.g. a
shared `ErrDraftProjectionDiverged`/`ErrDraftIdempotencyConflict` Studio
exports, or an `IsRetryable(error) bool` method added to `DraftProjector`) is
the natural next step, out of this slice's scope.

## Per-surface ownership

| Surface | Studio owns | Host implements |
|---|---|---|
| **Canvas** | Rendering/editing engine (`canvas/`, `workbenchruntime/`, `panels/`, `hostruntime/`), selection/inline-edit/contextual-panel runtimes | `.gsx` page markup carrying `data-studio-*` canvas identity attributes; which routes/sections are canvas-editable |
| **Identity** | `core.CanvasIdentity` schema + `Stable()`/`Normalize()` | Emitting the matching `data-studio-route`/`-page-id`/`-field`/`-block-key`/`-node-id` attributes in each host's own templates |
| **Operations** | `authoring.OperationKind`/`OperationRequest`/`OperationRecord` protocol (14 kinds: 5 core + 9 HANDOFF-12 instance/interaction/flow kinds), `SetKindForTarget`/`ClearsTarget`, validation | Dispatch from the host's own mutation/action handlers into either (a) the collaboration ledger (`collab.Service.Submit`, durable) or (b) the host's own ephemeral `AuthoringMutationResult` path (non-durable) — see the Noni/Pajaritos asymmetry below |
| **History** | `cms/lifecycle/engine` publish/restore/promote state machine, restore-point minting, retention policy, idempotency-by-`OperationID` replay | `HostLifecycle` adapter (`SnapshotLive`/`DraftDiff`/`Readiness`/`ApplyPublish`/`RestoreLive`/`OperationLog`/`PublishedOperationRevision`) over the host's own content store |
| **Collab** | `cms/studio/collab` (+`sqlstore`): ledger tables, `Service`/`Transport`/`Room`, capability model, `RepairFieldHead` recovery primitive | `DraftProjector` adapter (`StudioTargetValue`/`ProjectStudioOperation`/`CanApplyStudioOperation`/`QuarantineStudioOperation`) + the duplicated `ProjectingService` outbox-drain wrapper (see verdict above) + principal/nonce/session wiring |
| **Instances** | `core/instances.go` (`ComponentDefinition`/effective-controls resolution), the 4 durable instance operation kinds | The definition/instance store itself (Noni: `SharedComponentDefinition`/`ComponentInstance`; Pajaritos: `SharedDefinitions`/`ComponentInstances`), palette/admin panel |
| **Assets** | `cms/media` (`Store`/`AssetDeleter` interfaces, `DefaultUploadPolicy`, `GuardDeleteMediaAsset`, `AssetReadiness`, `NormalizeFocalPoint`) | The actual file-backed storage + admin routes (Noni: pre-existing; Pajaritos: net-new in HANDOFF-05/06, "zero media substrate before this slice") |
| **Interactions** | `core/interactions.go` (`Interaction` schema, allow-listed kind/effect/duration/delay, `Validate`), the 2 durable interaction operation kinds, `interactionruntime` public runtime | Attachment storage + draft/live split (both hosts have `LiveInteractions`/`InteractionsLiveMigrated`, HANDOFF-15) + which page sections are wired |
| **Flows** | `cms/flows` (`Document`, field/action schema), the 3 durable flow operation kinds | Which real host form the flow addresses (Noni: Newsletter, no flow literally named "signup"; Pajaritos: `schedule-tour`/`submit`, the real contact form) |
| **Lifecycle** | `cms/lifecycle/engine.Engine` (the only sequencer: idempotency, readiness gate, restore-point mint/retention, promote) | One `HostLifecycle` target per host (see limitations below); Promote never reaches host code by design (`Engine.Promote` never reads `e.Host`) |

## Host-specific limitations (honest, from the slice reports)

### Noni (`muddy-noni-commerce`)

- **9 new durable operation kinds are not ledgered.** Noni's `DraftProjector`
  (`internal/cms/studio_operations.go`'s `ProjectStudioOperation`) still only
  handles the 5 core kinds (`set-field`/`set-style`/`reset-style`/`undo`/
  `redo`) — confirmed by reading the file directly (no `shared-field`/
  `instances.*`/`interactions.*`/`flows.*` case exists). Interactions/shared-
  components/flow-field edits (`handoff-0506-noni-wiring-report.md`) go
  through the **older, ephemeral** `authoring.ApplyAuthoringMutation` path
  into Noni's own non-ledgered stores (`internal/cms/interactions.go`,
  `shared_components.go`), not through `collab.Service.Submit`. This means:
  no multi-actor conflict resolution, no actor-scoped undo/redo via the
  ledger, and no `cms/lifecycle/engine.OperationLog` visibility for these 9
  kinds on Noni today. This is exactly the host-adoption TODO #1
  `handoff-12-durable-ops-consolidation-report.md` left open; Pajaritos closed
  it (see below), Noni has not yet.
- **Shared components have no draft/live split and one public route.**
  `handoff-15-interactions-draftlive-report.md` documents this as a
  deliberate, deferred gap (not silently inconsistent): editing a shared
  component/instance takes effect immediately on the live site (matches
  interactions' pre-fix behavior); only `app/about/page.server.go`'s
  `studio-note` instance renders on a real public route, and no instance is
  seeded by default.
- **4 collaboration roles** (`studio-viewer`/`studio-author`/`studio-designer`/
  `studio-publisher`, plus an `admin` catch-all with every capability) —
  `internal/studiohost/collaboration.go`'s `CollaborationPrincipal`.
- Has a discard-draft action (`internal/cms/store.go`,
  `app/admin/editor/page.server.go`); Pajaritos does not (below).

### Pajaritos (`pajaritos-forest-school`)

- **Settings-only `HostLifecycle` target.** `internal/site/lifecycle_adapter.go`
  wires exactly one `TargetRef` (`cmsstore.ResourceKindSiteSettings`/`"site"`)
  through `cms/lifecycle/engine.Engine`
  (`handoff-07-s5-pajaritos-report.md`); pages still publish through the
  legacy `PublishPage` path directly, bypassing the engine entirely (no
  restore points, no readiness gate, no idempotent-publish replay for pages).
- **No bulk-discard action.** Confirmed by grep: no
  `Discard`/`Reset`/`Revert` function exists anywhere in the repo (Noni has
  one). An operator can only override/detach/restore individual instance
  fields, or restore a whole prior revision through the lifecycle engine —
  there is no "throw away every uncommitted draft edit" action.
- **All 14 durable operation kinds ARE ledgered**, unlike Noni:
  `internal/site/durable_authoring.go`'s `ProjectStudioOperation` uses
  `authoring.SetKindForTarget`/`ClearsTarget` generically for all 9 new
  kinds, closing HANDOFF-12's host-adoption TODO #1 in full
  (`handoff-0506-pajaritos-wiring-report.md`). A real kind-derivation bug
  (style-only check rejecting every "clearing" kind) was found and fixed in
  the process.
- **Single "admin" role only** (`internal/studiohost/collaboration.go`'s
  `UserIsAdmin`/`CollaborationPrincipal`) — every authenticated admin gets
  every capability (View+Author+Design+Publish); there is no
  viewer/author/designer/publisher split like Noni's.
- Interactions/shared-components also have no draft/live split (same
  deferred-gap decision as Noni, `handoff-15-interactions-draftlive-report.md`).
- Media library has exactly one real referencing use case (the hero image);
  no other content model references an asset yet (`handoff-0506-pajaritos-
  wiring-report.md` caveat 4) — an honest scope limit, not a placeholder.

## Grep evidence appendix

```
# No copied Studio engine/render/ledger internals in either host:
$ grep -rln 'mintRestorePoint\|replayPublish\|replayRestore\|classifyOperationScope\|isExactInverse\|OperationChangeSet\|DraftChangeScope\|ScopeInstances\b' --include='*.go' <host>
(no output, both hosts)

$ grep -rln 'func.*resolveHistory\|studio_field_heads\|studio_projection_outbox\|studio_operation_attempts' --include='*.go' <host>
muddy-noni-commerce/internal/cms/studio_operations.go   # doc comment only, line 129
pajaritos-forest-school/internal/site/durable_authoring.go  # doc comment only, line 700

$ grep -rln '^package canvas$\|^package workbenchruntime$\|^package panels$' --include='*.go' <host>
(no output, both hosts)

# ProjectingService duplication (host-to-host, not Studio-owned):
$ wc -l muddy-noni-commerce/internal/studiohost/collaboration.go pajaritos-forest-school/internal/studiohost/collaboration.go
391 / 431 lines, 20 / 21 funcs

# Noni's 9-new-kind ledger gap:
$ grep -n 'OperationSetSharedField\|OperationSetInteraction\|OperationSetFlowField' muddy-noni-commerce/internal/cms/studio_operations.go
(no output -- confirms only the 5 core kinds are handled)

# Pajaritos' full 14-kind adoption:
$ grep -n 'SetKindForTarget\|ClearsTarget' pajaritos-forest-school/internal/site/durable_authoring.go
durable_authoring.go:787: kind := authoring.SetKindForTarget(target, record.After.Present)
durable_authoring.go:788: if !record.After.Present && !authoring.ClearsTarget(kind) {

# Discard-draft asymmetry:
$ grep -rln 'func.*[Dd]iscard' --include='*.go' muddy-noni-commerce | grep -v _test
internal/cms/store.go
app/admin/editor/page.server.go
$ grep -rln 'func.*[Dd]iscard' --include='*.go' pajaritos-forest-school | grep -v _test
(no output)
```
