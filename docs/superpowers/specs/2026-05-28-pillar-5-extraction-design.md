# Pillar 5 — Extraction Completion Design

**Date:** 2026-05-28
**Status:** Draft, pending review
**Owner:** odvcencio
**Pillar:** 5 of 5 (foundation; unblocks Pillars 1-4)

## Context

The "Webflow-in-a-box" vision for `gosx-studio` decomposes into five pillars:

1. Universal content model (CMS-side: encode any content as a generic primitive)
2. Non-technical editor UX (progressive disclosure, templates, trust, AI-assist)
3. Plugin / vertical architecture (ecommerce, 3D scenes, etc. as bundles)
4. Hosting + multi-tenancy (the gap between "library" and "product")
5. **Extraction completion (this document)** — the boring-but-essential refactor that unblocks 1-4

Today's `gosx-studio` is a contract library + browser-runtime asset pipeline. The actual editor implementation lives in two wrong places:

- **`gosx-cms/studio/`** — 56 files, 12,561 lines of editor renderers (94 `Render*` functions), panel `Options` structs, Shell/Workbench/View types. Mis-located inside the CMS module. Hyphae records this as "the primary active area of gosx-cms development" — new editor work is still being added to the wrong module.
- **`muddy-noni-commerce/app/admin/editor/page.gsx`** — 56 `.gsx` editor components (~3,050 lines) including `StudioWorkbenchShell`, `StudioToolbar`, `StudioModebar`, `StudioCanvasBar`, `StudioPreviewFrame`, `HomeInspectorPanel`, `LookPanel` + 10 Look sub-components, `BrandPanel`, `PublishPanel`, `SiteNavigatorPanel`, `BlockLibraryPanel`, `CanvasEnginePanel`, `SiteMapEnginePanel`, `BlockLayoutEnginePanel`, `FlowDesignerPanel`, `AdvancedPanel`. Structurally reusable — they consume `data.*` / `actions.*` only, with zero Noni-domain imports in the component bodies. Mis-located inside the host.

The current `gosx-studio` module is correctly extracted for: Go contracts (`SiteMap`/`Page`/`Component`/`Control`/`CanvasWorkspace`/`ShellConfig`/`RuntimeContract`), 7 typed browser-side runtime packages (post Phase 3 burn-down), `studio.css`, and one plugin (`showcase3d`). Everything else needed to call `gosx-studio` "the website editor" still lives elsewhere.

A second host exists: **`~/work/pajaritos-forest-school`** (`github.com/odvcencio/pajaritos-forest-school`, on `gosx-cms v0.1.56` — older module path). Its editor route is 42 lines of `.gsx` + 297 lines of `page.server.go`. It does not import `gosx-studio` at all. It consumes most of the editor through a small `internal/site/store` layer, but it also imports `cmsstudio` directly from `app/admin/editor/page.server.go`, `app/admin/editor/page_server_test.go`, and `internal/site/store_test.go` for lifecycle/flow helpers (`ParseLifecyclePublishAt`, `PublishConfiguredFlow`, `SaveConfiguredFlowDrafts`, `LoadLifecycleReviewState`, `LifecyclePublishApproval`, `SiteCanvasPositionInputName`, etc.). Pajaritos proves the editor is already content-agnostic (its content shape is programs/sessions/flows, not commerce) and demonstrates the target shape Noni should reach after Pillar 5.

For reference, Noni imports `cmsstudio` 403 times across its codebase; pajaritos has a few dozen direct call sites. `gosx-admin` does NOT depend on `cmsstudio` (verified — zero references), so it is not affected by this extraction. The only modules touched are: `gosx-studio` (destination), `gosx-cms` (source + relocation of content types), `muddy-noni-commerce` (heavy consumer), `pajaritos-forest-school` (light consumer).

## Principles

1. **Dependency direction is fixed: `host → gosx-studio → gosx-cms`.** CMS must never import Studio. Studio imports CMS freely — that's the legitimate direction.
2. **CMS owns content; Studio owns editor.** Anything that produces a `gosx.Node` belongs in Studio. Anything that describes content, schemas, lifecycle state, or storage belongs in CMS. This is the boundary the README has always stated but the code has never honored.
3. **CMS will need a generic `Document`/`Schema` primitive (Pillar 1)** to make "any content" real. Pillar 5 doesn't deliver that primitive — but every choice here leaves room for it (no hardcoded knowledge of specific content shapes in Studio's APIs).
4. **`gosx-studio` package layout is flat**, with one carve-out: the 7 existing `*runtime/` sub-packages stay separate (their boundaries are JS-bundle composition concerns, not Go organization concerns).
5. **End state**: Noni's editor route drops from ~7,000 lines to ~500. Pajaritos's editor route stays at its current ~340 lines (proving Pillar 5 didn't leak Noni-shape into Studio). `gosx-cms/studio/` package ceases to exist. `gosx-studio` is the complete reusable editor.

## Scope

### Moves into `gosx-studio`

| Source | Approximate lines | Destination |
|---|---|---|
| `gosx-cms/studio/*.go` (56 files, 94 `Render*` functions, panel `Options` structs, `Shell`/`Workbench`/`View`/`Metric`/`Viewport`/`Panel`/`Action` types) | ~12,561 | `gosx-studio/` (flat package, alongside `studio.go`/`shell.go`/`runtime.go`) |
| `muddy-noni-commerce/app/admin/editor/page.gsx` (56 components) | ~3,050 | `gosx-studio/*.gsx` |
| `muddy-noni-commerce/internal/gosxstudio/runtime.go` (`WorkbenchRuntimeHandler` / `CommandRuntimeHandler` / `StateRuntimeHandler`) | ~140 | `gosx-studio/runtime.go` — resolves the three dangling path constants (`WorkbenchRuntimePath`, `CommandRuntimePath`, `StateRuntimePath`) so an integrator no longer has to mount them by hand |
| `muddy-noni-commerce/app/admin/editor/page.server.go::editorStudioFeatureFlagAttrs()` | ~15 | `gosx-studio/shell.go::FeatureFlagAttrs(ShellConfig)` |

### Stays in `gosx-cms`, but relocates within CMS

These types are currently in `gosx-cms/studio/` but are content/state types, not editor types. They keep their CMS home but exit the `studio` sub-package and drop any "Studio" prefix.

**When relocated**: each type moves in the batch whose Render functions consume it. `PublishReview` and related publish types relocate during batch 9 (Publish); `HealthReport` and `PerformanceSignal` during batch 10 (Activity); `LifecycleActivityItem` during batch 10 (Activity); `RevisionItem` during batch 9 (Publish, since revision history is rendered there); `Readiness` during whichever batch first needs it (likely batch 9 — Publish, then re-used by 10 — Activity). Batch 12 (Cleanup) verifies nothing content-shaped remains in `gosx-cms/studio/` before deleting the package.

| Type | Was | Now |
|---|---|---|
| `cmsstudio.PublishReview` | `gosx-cms/studio/publish_review.go` | `gosx-cms/lifecycle/publish_review.go` (or `cms/publish/`) |
| `cmsstudio.HealthReport` | `gosx-cms/studio/health.go` | `gosx-cms/health/health.go` |
| `cmsstudio.Readiness` | `gosx-cms/studio/readiness.go` | `gosx-cms/lifecycle/readiness.go` |
| `cmsstudio.LifecycleActivityItem` | `gosx-cms/studio/workbench_panels.go` | `gosx-cms/lifecycle/activity.go` |
| `cmsstudio.PerformanceSignal` | `gosx-cms/studio/performance.go` | `gosx-cms/health/performance.go` |
| `cmsstudio.RevisionItem` | `gosx-cms/studio/panels.go` | `gosx-cms/lifecycle/revisions.go` (likely already lives there; needs dedup) |

### Cross-package CMS types with `Studio*` prefix — renamed in place

Code smell: the `Studio*` prefix in `gosx-cms` was a sign that CMS knew about its own editor. With the editor now in `gosx-studio`, the prefix is meaningless. Renames:

| Was | Now |
|---|---|
| `cmsflows.StudioFlow` | `cmsflows.Flow` |
| `cmsflows.StudioStep` | `cmsflows.Step` |
| `cmsflows.StudioAction` | `cmsflows.Action` |
| `cmsstyle.RecipeView` | `cmsstyle.Recipe` (if no name conflict) or `cmsstyle.AuthoringView` |
| `cmsblocks.HomeBlockCatalog` (function) | stays as is — `Catalog` is fine; "Home" is content-shape, leave it (will generalize in Pillar 1) |

### Host glue that stays in hosts

- **Noni's content store** (`internal/cms` and surrounding) — Noni-specific data, stays in Noni
- **Pajaritos's `internal/site/store`** — Pajaritos-specific data, stays in Pajaritos
- Both hosts conform to the new `studio.Store` interface introduced by Pillar 5 (§4)

### Not in scope for Pillar 5

- **Universal `Document`/`Schema` primitive** (Pillar 1). Studio's `Store.ContentViews()` returns the existing concrete views during Pillar 5; Pillar 1 generalizes the return type.
- **Editor UX redesign for non-technical users** (Pillar 2). Lifted components ship at parity with their current behavior.
- **Plugin contract maturation** for ecommerce + 3D (Pillar 3). The `studio.Feature` interface stays as-is.
- **Hosting / multi-tenancy** (Pillar 4). Out of library scope entirely.

## New public API surface introduced

Formalized from the pajaritos `internal/site/store` pattern — this becomes the documented integration path for any host. The starting shape mirrors what `cmsstudio.Options` (`gosx-cms/studio/studio.go:13`) requires today, lifted into an interface so hosts can swap implementations:

```go
// gosx-studio/store.go — the host integration interface.
package studio

import (
    "m31labs.dev/gosx-cms/lifecycle"
    "m31labs.dev/gosx-cms/media"
)

// Store is the small contract a host implements to feed Studio with content.
// Pajaritos and Noni each implement this once; Studio composes it into the
// editor surface via HostShell.
//
// Starting shape — Pillar 5 lifts this directly from what cmsstudio.Options
// already requires from its callers today. Each method returns the same
// concrete shape Studio currently renders against, just behind an interface
// so Pillar 1 (universal content model) has a single seam to widen later
// rather than touching every host.
type Store interface {
    // LeftPanels and RightPanels return the rail panels Studio renders in
    // the workbench shell. Each Panel carries Children []gosx.Node that the
    // host populates (this is the existing cmsstudio.Options.Left/Right
    // shape — Pillar 5 does NOT change how panels are constructed, only
    // where the type lives).
    LeftPanels()  []Panel
    RightPanels() []Panel

    // MediaAssets backs the media datalist and brand media picker.
    // Today's type: m31labs.dev/gosx-cms/media.Asset.
    MediaAssets() []media.Asset

    // Revisions backs the revision history panel.
    // Today's type: m31labs.dev/gosx-cms/lifecycle.Revision (sometimes
    // aliased store.Revision in pajaritos).
    Revisions() []lifecycle.Revision

    // Readiness backs the publish review panel.
    // Today's type: cmsstudio.Readiness. Batch 9 (Publish) relocates this
    // to gosx-cms/lifecycle/readiness.go per §Scope — after that batch the
    // signature is lifecycle.Readiness.
    Readiness() Readiness

    // Adapters returns the host-bound resource adapters (media, pages,
    // products, orders, contacts, settings, revisions, lifecycle, flows).
    // Same shape as DefaultResourceAdapters() returns today.
    Adapters() []ResourceAdapter
}

// HostShell composes a Store and a ShellConfig into the existing
// Shell/View/Workbench machinery. Replaces the per-host cmsstudio.New
// boilerplate today (Noni's editorStudioShell at
// app/admin/editor/page.server.go:267; pajaritos's StudioShell in
// internal/site/store.go).
func HostShell(store Store, shell ShellConfig) Shell

// FeatureFlagAttrs returns the data-* attribute map that the workbench root
// must render so the BridgeShim flag gates evaluate to true.
// Replaces Noni's editorStudioFeatureFlagAttrs() helper.
func FeatureFlagAttrs(shell ShellConfig) map[string]string
```

**Notes:**

- `Panel`, `Readiness`, and `ResourceAdapter` are existing types. `Panel` lives in `gosx-cms/studio/` today and moves to `gosx-studio` during the migration. `Readiness` lives in `gosx-cms/studio/` today and moves to `gosx-cms/lifecycle/` during batch 9 (it's content-state, not editor-shaped per §Scope). `ResourceAdapter` already lives in `gosx-studio`. Each method signature above resolves cleanly against the post-migration package layout.
- The `Readiness` type starts as a re-export from `cmsstudio` during batches 2-8, then becomes a true Studio type in batch 9 when `cmsstudio.Readiness` moves to `gosx-cms/lifecycle/readiness.go` and Studio's `Store.Readiness()` switches to returning `lifecycle.Readiness`. Hosts implementing `Store` after batch 9 see the cleaner signature; hosts that implemented it earlier need a minor signature update at batch 9.
- Pillar 5 does NOT introduce a generic "ContentView" type. Today's reality is that hosts hand-build the `[]gosx.Node` children of each `Panel`. Pillar 1 will add a generic `Document`/`Schema` primitive that lets Studio render unknown content without hand-built nodes — but that's a separate effort, and the `Store` interface above intentionally exposes today's panel-building surface so Pillar 1 can widen it incrementally.

## Migration approach: branch-by-batch

12 batches, lowest-coupled first. Each batch is internally consistent (no shim left behind, no temporary type aliases) and ends with:

- `gosx-studio` builds + tests green
- `gosx-cms` builds + tests green (the package shrinks after each batch)
- Noni's `go build ./... && go test ./...` green
- Pajaritos's `go build ./... && go test ./...` green
- One `buckley commit --yes --minimal-output` per batch

`gosx-admin` is intentionally absent — it does not depend on `cmsstudio` (verified). It still rebuilds transitively if its own `gosx-cms` dependency changes, but no `gosx-admin` source needs editing in any batch.

Per-batch work shape:

1. Move the Go files for the surface from `gosx-cms/studio/` → `gosx-studio/`
2. Move the related `.gsx` components from Noni's `page.gsx` → `gosx-studio/*.gsx`
3. Update Noni's imports (`cmsstudio.X` → `studio.X`) for that surface — Noni has 403 total `cmsstudio` references, batched per surface
4. Update Pajaritos's imports for that surface. Touch points include `internal/site/store.go`, `internal/site/store_test.go`, `app/admin/editor/page.server.go`, and `app/admin/editor/page_server_test.go`. Batches 9 (Publish) and 11 (Flow) carry the heaviest pajaritos diff because the direct pajaritos→cmsstudio imports are lifecycle and flow helpers; batches 2-8 mostly touch only `internal/site/store.go`
5. Run all four `go test` suites; commit

### Batch order

1. **Prep** — bump pajaritos to current `m31labs.dev/*` module paths (currently on `github.com/odvcencio/gosx-cms v0.1.56`). Announce freeze on `gosx-cms/studio/` commits.
2. **Chrome** — toolbar, modebar, command palette, save status, history controls
3. **Canvas** — site canvas, preview frame, viewport switcher, zoom controls, canvas status, canvas insert shelf, canvas selection commandbar
4. **Site navigator + site map** — site nav panel, layer list, site map engine panel + island
5. **Block library + block layout** — palette, layout panel, drag/handle/list bindings
6. **Home inspector** — home inspector panel + island + content fields
7. **Look** — style scope, workbench, impact, choice group, field, select, radio, swatches, color grid (10 sub-components)
8. **Brand** — brand panel, brand fields, brand media picker island
9. **Publish** — publish review (data → CMS per §Scope), save status, preview share, revision history
10. **Activity** — activity panel, health, performance (data types → CMS per §Scope)
11. **Flow** — flow editor, flow cards, flow library, flow designer panel + island
12. **Advanced + finalization** — advanced tools, advanced fields, the 3 runtime handlers, `FeatureFlagAttrs` helper, delete `gosx-cms/studio/` entirely, doc cleanup (§6)

## Test strategy

- **Per batch**: existing `cmsstudio` tests move alongside their renderers. They use only `Options`-shaped inputs today, so they pass at the new location unchanged.
- **New integration test in `gosx-studio`**: a stub host that implements `Store` with synthetic content. Renders the full editor against `DefaultShellConfig().Normalize()`. Confirms Studio is self-contained — no Noni / Pajaritos imports, only legitimate CMS data imports. This test becomes the gate that the package boundary stays clean.
- **Per-host smoke test**: Noni and Pajaritos each get a single end-to-end test that mounts the Studio editor route and asserts the page returns 200 + contains the expected workbench mount points (`[data-gosx-studio-workbench="true"]`, etc.). These are the cross-repo integration canaries.
- **`parity_e2e`**: retire after batch 12. The dual-path A/B comparison the harness was designed for no longer exists, and the harness's README + comments are stale ("the legacy bundle" find-replace artifacts riddle the descriptions). Preferred option: delete entirely — Pillar 2 will want different e2e infrastructure anyway. Alternative: convert to a single-mode smoke test that loads the editor in a real browser.

## Documentation cleanup (rolls into batch 12)

- Scrub the ~110 "the legacy bundle" find-replace artifacts across `.go`/`.ts`/`.md` (sample command: `grep -rn "the legacy bundle" --include='*.go' /home/draco/work/gosx-studio` finds them all). Restore original file:line references where recoverable from git history.
- Replace `gosx-studio/README.md`'s boundary statement with the corrected version: "Studio imports CMS freely; CMS never imports Studio. Studio is the authoring surface for any content CMS encodes."
- Update the `initiative.gosx-studio-current-state` hyphae object: Pillars 1–4 now sequenced, Pillar 5 complete.
- Write a hyphae lesson on the `StudioXxx`-naming smell: "when a sibling package needs a prefix matching its sibling, the boundary is wrong."

## Risk

| Risk | Mitigation |
|---|---|
| A renderer touches a CMS type that turns out to be content-shaped (belongs in CMS) but is structurally entangled with a Studio render | Per-batch judgment call; document each in commit message; the `Store` interface gives a clean escape hatch (move the type behind the interface) |
| Pajaritos's older module path means batches that rename CMS types break pajaritos's older API | Batch 1 ("Prep") bumps pajaritos to current module paths before any renames |
| Active development happens in `gosx-cms/studio/` during the migration window (hyphae notes it's "the primary active area") | Announce the freeze at start of batch 1; any in-flight cmsstudio work lands in gosx-studio post-batch-3 instead |
| End state changes the host integration shape; downstream hosts that aren't Noni/Pajaritos (if any) break silently | After batch 12, `hypha graph backlinks` on the `gosx-cms/studio` import to find any unaccounted consumers; coordinate before deleting the package |

## Sequencing

- **Estimated wall-clock**: 3-4 weeks at 1 batch every 2-3 days. Worktree fan-out (per the [graft_coord_for_parallel_agents](../../../../.claude/projects/-home-draco-work-muddy-noni-commerce/memory/graft_coord_for_parallel_agents.md) preference) can compress batches that touch disjoint files.
- **Commit cadence**: one `buckley commit --yes --minimal-output` per batch boundary, plus intra-batch commits per TDD step per the [commit_per_tdd_step_cadence](../../../../.claude/projects/-home-draco-work-muddy-noni-commerce/memory/commit_per_tdd_step_cadence.md) memory.
- **Branch discipline**: per `gosx_initiative_fast_and_loose`, merge directly to main on each batch; no PR ceremony.

## Success criteria

- `gosx-cms/studio/` package no longer exists
- `gosx-studio` exposes a `Store` interface as the documented host integration path
- Both Noni and Pajaritos compile + test green with the new `gosx-studio` API and zero references to `gosx-cms/studio`
- Pajaritos's editor route stays at ~340 lines (proving no Noni-shape leaked into Studio)
- Noni's editor route shrinks from ~7,000 lines to <600 lines (host glue + Noni-specific content store only)
- A `gosx-studio` integration test renders the full editor against a synthetic stub host with no Noni / Pajaritos / gosx-cms-internal dependencies (only the public CMS data types it legitimately uses)
- The three dangling path constants (`WorkbenchRuntimePath` / `CommandRuntimePath` / `StateRuntimePath`) resolve to handlers Studio ships
- Zero "the legacy bundle" find-replace artifacts remain in source
- `parity_e2e` is either deleted or converted to a single-mode smoke test
