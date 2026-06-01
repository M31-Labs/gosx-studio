# GoSX Studio Webflow-Class Editor Gap Inventory

This inventory defines what is missing to make GoSX Studio a full no-code web
development editor in the Webflow class, while staying native to GoSX instead
of becoming a generic JSON page builder.

## Target Promise

The finished platform should let a non-technical site author:

- Create and organize pages from visual blueprints.
- Add, remove, reorder, and configure typed GoSX components.
- Edit text, links, media, layout, style, CMS bindings, flows, and showcase
  plugins without seeing Go handlers or component identifiers.
- Preview and tune responsive breakpoints.
- Save drafts, publish to staging, promote to production, and roll back.
- Collaborate safely with presence, history, permissions, and audit trails.
- Extend the editor through Studio plugins without copying app-specific editor
  code into every host.
- Keep reference client sites host-branded in public and admin UI. Footer
  attribution such as "Made with GoSX by M31 Labs" is acceptable, but visible
  Muddy/Noni and Pajaritos copy should not name "GoSX Studio."

The architectural constraint is important: Studio should expose a no-code graph
over typed GoSX components, server actions, engines, islands, CMS resources, and
host adapters. It should not erase GoSX into an untyped website blob.

## Current Inventory

The current codebase has the right contract direction, but the product surface is
still split across `gosx-studio`, `gosx-cms/studio`, Muddy Noni, and Pajaritos.

| Area | Evidence | Current state |
| --- | --- | --- |
| Studio package boundary | `README.md`, `docs/ARCHITECTURE.md`, `docs/ROADMAP.md` | The docs already define Studio as the authoring layer beside CMS/Admin, not inside a host app. |
| Host shell contract | `store.go:12`, `store.go:27` | Hosts can feed Studio panels, media, revisions, readiness, adapters, and a normalized shell. |
| Reusable workbench shell projection | `workbench_shell_view.go` | Studio now owns the shared editor shell view payload for workbench chrome, preview-shell attrs, zoom levels, rail resizers, CSRF/action metadata, and host label overrides. |
| Reusable workbench toolbar renderer | `workbench_toolbar.go` | Studio now renders shared shell chrome from `WorkbenchShellView`: toolbar title, summary, mode controls, metrics, command palette, save status, history controls, preview link, and save button. |
| Reusable site-map authoring panels | `sitemap_authoring_panels.go` | Studio now renders shared page metadata, editable control, reorder, duplicate, visibility, delete, composition-intent, and optional page-list panels from `AuthoringSiteMapView`. Muddy/Noni and Pajaritos feed host views/options instead of duplicating that panel markup. |
| Reusable site-map authoring forms | `sitemap_authoring_forms.go` | Studio now renders the hidden managed forms used by site-map authoring panels, including scoped composition-intent forms, operation forms, and CSRF inputs. Muddy/Noni and Pajaritos no longer duplicate that form scaffold. |
| Reusable site-map engine frame | `sitemap_engine.go` | Studio now renders shared site-map engine frame segments: root semantics, header chrome, stats, source legend, and renderer markers. Muddy/Noni consumes those segments around the shared site-map board renderer. |
| Reusable site-map board renderer | `sitemap_board.go` | Studio now renders the visible site-map board from `AuthoringSiteMapView`: filters, focus controls, map lanes, selected-node details, composition intents, blueprints, palette, component/source/control views, and minimap. Muddy/Noni consumes the shared node instead of its old local `SiteMapCanvasIsland`. |
| Reusable site-map board runtime | `sitemapruntime/` | Studio now owns the browser interaction contract for site-map board filters, detail tabs, palette selection, node selection, and fragment remount binding. Muddy/Noni runs it against the shared board renderer, and Pajaritos can mount the standalone runtime beside its server-composed shell. |
| Authoring projection | `authoring.go:9`, `authoring.go:21`, `authoring.go:47` | `AuthoringSurface` exposes pages, selected page, palette, intents, workspace graph, and canvas layout. |
| Authoring mutation contract | `mutations.go` | Studio now has typed operation kinds, form field names, `AuthoringMutation`, `AuthoringAdapter`, validation/result envelopes, and a GoSX action handler for host-owned persistence. |
| Typed page/component model | `studio.go:151`, `studio.go:156`, `studio.go:292` | Pages, components, controls, canvas blocks, blueprints, templates, and composition intents exist as contracts. |
| Runtime contracts | `studio.go:452`, runtime subpackages | Preview, workbench, selection, field, brand, style, and block layout runtimes are being moved into Studio-owned packages. |
| Adapter declarations | `studio.go:510` | Resource adapters describe CMS/admin/page/media/lifecycle/flow surfaces, but mostly describe capability instead of executing mutations. |
| Product roadmap | `docs/ROADMAP.md:18` through `docs/ROADMAP.md:27` | The missing reusable `.gsx` surfaces are already named: shell, navigator, site-map, canvas, block-layout, inspector, flow designer, publish panel. |
| Noni proving ground | `muddy-noni-commerce/app/admin/editor/page.server.go:267`, `muddy-noni-commerce/app/admin/editor/site_map.server.go:456` | Noni still owns a large custom editor implementation, but now registers the Studio authoring action and persists the home hero headline through the shared mutation contract. |
| Noni runtime bridge | `muddy-noni-commerce/internal/gosxstudio/runtime.go:109` | Some editor runtimes still bridge through `gosx-cms/studio`, so the ownership boundary is not complete. |
| Pajaritos proving ground | `pajaritos-forest-school/internal/site/store.go:927`, `pajaritos-forest-school/internal/site/store.go:970` | Pajaritos exposes `hostShell` plus an `authoringAction`, but still renders most editor panels through CMS Studio helper surfaces. |
| Pajaritos Studio adapter | `pajaritos-forest-school/internal/site/store.go:556`, `pajaritos-forest-school/internal/site/store.go:1230` | A lightweight host store, site-map bridge, and first save-control mutation prove the adapter shape can serve a second app. |

## Missing Pieces

### 1. Reusable Studio `.gsx` Surfaces

Current:

- The roadmap names `StudioShell.gsx`, `SiteNavigator.gsx`, `SiteMapEngine.gsx`,
  `CanvasEngine.gsx`, `BlockLayoutEngine.gsx`, `InspectorIsland.gsx`,
  `FlowDesigner.gsx`, and `PublishPanel.gsx`.
- Host apps still render most visible UI through Noni-specific functions or
  `gosx-cms/studio` helpers.
- `WorkbenchShellView` now extracts the first shared shell projection into
  `gosx-studio`, and both Noni and Pajaritos consume it for workbench metadata,
  preview-shell attributes, zoom controls, rail resizers, and save/preview
  labels.
- `RenderWorkbenchToolbar`, `RenderWorkbenchModebar`,
  `RenderWorkbenchMetricStrip`, and `RenderWorkbenchCommandPalette` now give
  Go-rendered hosts a Studio-owned toolbar path that consumes
  `WorkbenchShellView`. Muddy/Noni and Pajaritos both use it for visible editor
  toolbar chrome while preserving host-branded copy, existing CSS hooks, and the
  approved footer attribution shape.
- `RenderSiteMapAuthoringPanels` now gives Go-rendered hosts a Studio-owned
  renderer for page metadata, editable-control, component reorder/duplicate/
  visibility/delete, composition-intent, and page-list panels. Muddy/Noni and
  Pajaritos consume it with host styling options, so visible copy remains
  host-branded and the approved small GoSX/M31 footer attribution stays outside
  the editor panel renderer.
- `RenderSiteMapAuthoringForms` now gives hosts the matching Studio-owned
  hidden managed form scaffold for those panels, including CSRF input support
  and the existing managed GoSX form attributes.
- `RenderSiteMapEngineSegments` now gives hosts Studio-owned site-map engine
  frame chrome for root semantics, summary stats, source legend, and stable
  renderer markers while preserving host-owned interaction slots.
- `RenderSiteMapBoard` now gives hosts a Studio-owned visible site-map board
  renderer for map/build/components/controls/sources views. Muddy/Noni consumes
  that node from route data, so the app-local `SiteMapCanvasIsland` board is
  gone and the board behavior delegates to `sitemapruntime`.
- `sitemapruntime` now gives hosts a Studio-owned site-map board interaction
  runtime for filters, detail tabs, palette state, node selection, and
  fragment remount binding.

Missing:

- Studio-owned `.gsx` components for the complete editor shell.
- A single host-facing render path that consumes `ShellConfig`,
  `AuthoringSurfaceView`, adapters, and panels.
- Clear extension slots for host app panels without letting host apps replace
  the whole editor surface.

Acceptance:

- Noni and Pajaritos can mount the same Studio shell with different labels,
  resources, palettes, and actions.
- App-specific `.gsx` editor pages shrink to route wrappers and host data
  loading.

### 2. Authoring Mutation API

Current:

- `CompositionIntent` describes create-page and add-component operations.
- `Control` describes editor-facing fields.
- `Store` is read-oriented. It feeds panels, media, revisions, readiness, and
  adapters, but does not define how Studio applies authoring operations.
- `AuthoringMutation`, `AuthoringAdapter`, and `AuthoringActionHandler` now
  define the first executable mutation boundary.
- `AuthoringSurfaceView` now exposes stable mutation field names plus form
  payloads for composition intents and component controls.
- `AuthoringMutationView`, `AuthoringSurfaceView`, and `AuthoringSiteMapView`
  now expose ordered `formInputs`, so host renderers can build visible apply,
  page metadata, component operation, and control forms without duplicating
  every hidden action field.
- Noni applies create-page, home add-component, and hero-headline mutations.
- Pajaritos applies create-page, family-form add-component, hero-headline, and
  flow step/submit label mutations.
- `authoringruntime` now listens for successful GoSX action results, marks the
  changed editor object, updates workbench save feedback, and refreshes preview
  frames when the host mutation result requests it.
- Managed authoring forms now keep editable-control, page metadata,
  visibility, and reorder edits in the active editor instead of forcing a
  full-page redirect for every persisted change.
- Structural authoring results can now request targeted editor fragment
  refreshes. The browser runtime fetches the current editor document, replaces
  host-declared fragments, remounts editor runtimes, and then selects the
  changed page/component before emitting authoring result feedback.
- `AuthoringSiteMapView` now exposes the first editable page metadata mutation
  payload, and Noni/Pajaritos persist page title/route updates with field-level
  validation through the same authoring action boundary.
- `AuthoringSiteMapView` now exposes the first editable component visibility
  mutation payload. Noni persists homepage section visibility, and Pajaritos
  persists the home family-request form visibility through the same action
  contract.
- `AuthoringSiteMapView` now exposes component reorder payloads. Noni persists
  homepage section order through its settings store, and Pajaritos persists home
  section order through preview settings metadata.
- `AuthoringSiteMapView` now exposes component delete payloads. Noni removes a
  storefront section by disabling it in site settings, and Pajaritos removes a
  home section from the configured order while keeping it restorable from the
  component palette.
- `AuthoringSiteMapView` now exposes component duplicate payloads with a
  template key and target insert position. Noni persists repeatable homepage
  section instances, and Pajaritos persists repeatable home sections through
  preview settings metadata.

Missing:

- Cross-operation validation beyond the current create-page, add-component,
  save-control, page metadata, visibility, reorder, duplicate, and delete paths.
- Route-level examples showing how host apps register `AuthoringActionHandler`
  beside page modules.

Acceptance:

- The site-map and canvas engines can call one Studio-defined action contract
  and have Noni/Pajaritos persist the result through their own stores.

### 3. Site Map Engine

Current:

- `SiteMap.CompositionWorkspace()` and `WorkspaceCanvas` can describe a graph.
- Noni has app-local functions for page views and composition intents.
- Pajaritos builds a simple site map but does not get a reusable graph editor.
- The shared site-map projection can identify an editable page and provide a
  ready-to-submit page metadata mutation payload.
- The shared site-map projection now exposes scoped apply-intent payloads for
  create-page and add-component drafts. Noni and Pajaritos render visible
  "Create page" / "Add section" controls from those payloads, and Pajaritos now
  derives its editor panel site-map from `NoCodeAuthoringSurface`.
- `RenderSiteMapAuthoringPanels` now renders the first Studio-owned site-map
  authoring control stack for both reference apps while preserving legacy data
  attributes for existing runtimes and tests.
- `RenderSiteMapAuthoringForms` now renders the matching managed form stack for
  composition intents and site-map operations in both reference apps.
- `RenderSiteMapEngineSegments` now owns the first visible site-map engine
  frame segments. Muddy/Noni wraps the shared board renderer with those
  Studio-rendered root, header, and source legend nodes; Pajaritos stays on the
  same shared site-map projection contracts as the simpler second-host
  baseline.
- `RenderSiteMapBoard` now owns Muddy/Noni's visible site-map board markup:
  board toolbar, map lanes, selected node cards, connection rail, build panel,
  blueprints, palette, component/source/control views, and minimap. The
  renderer emits the existing runtime data hooks without visible platform copy.
- `sitemapruntime` now owns the client-side board interaction contract. It
  drives site-map group filters, focus/detail/palette toggles, zoom/density/
  grid state, selected node details, composition-intent active state, and
  rebinds after managed fragment refreshes. Muddy/Noni now runs the runtime
  against the Studio-rendered board, while Pajaritos mounts the standalone
  runtime next to its server-composed workbench.

Missing:

- A complete `SiteMapEngine.gsx` surface that combines frame, board, managed
  forms, and host slots into one reusable Studio component instead of requiring
  host route assembly.
- Pajaritos still uses its simpler CMS Studio site canvas as the visible map;
  it should either adopt `RenderSiteMapBoard` or provide a host-styled board
  slot on top of the same Studio contracts.
- Full route editing across all editable pages, page grouping, selection state,
  and conflict validation inside the visual graph.
- Keyboard and pointer interactions suitable for a visual graph. **Partial
  (2026-06-01):** continuous infinite-canvas pan/zoom (wheel-to-cursor,
  drag-to-pan, keyboard `+`/`-`/`0`, live zoom readout) and marquee
  multi-select (shift-drag rubber-band, shift-click toggle, `Escape`-clear)
  shipped on the shared board via `sitemapruntime` + `RenderSiteMapBoard`. Still
  missing: keyboard node navigation and node drag-to-reposition.

Acceptance:

- An editor can create a new page from a blueprint, add a component to an
  existing page, select it, and see the change reflected in the canvas preview.

### 4. Canvas Engine and Inline Editing

Current:

- `CanvasWorkspace` models viewports, zoom, preview shell, blocks, and actions.
- Preview, selection, field, workbench, brand, style, and block-layout runtimes
  exist as separate Studio runtime packages.

Missing:

- A unified canvas engine that mounts the preview iframe, overlays selection
  affordances, maps DOM nodes to `CanvasBlock`, and coordinates runtime calls.
- Inline text editing with draft isolation and explicit save/cancel semantics.
- Drag/drop placement and reorder operations that are persisted through the host
  mutation API.
- Empty states, loading states, missing selector diagnostics, and runtime error
  reporting.

Acceptance:

- The editor can select a rendered section, edit a text field inline, reorder
  sections visually, and persist those edits without host-specific browser
  scripts.

### 5. Inspector and Control System

Current:

- `ControlKind` covers text, media, choice, toggle, number, link, color, source,
  flow, Scene3D, and similar field surfaces.
- Noni still has custom inspector field view code in the app.

Missing:

- A reusable inspector island for every `ControlKind`.
- Validation, dirty state, required fields, field grouping, advanced fields, and
  dependent options.
- Media picker, source picker, flow picker, link picker, and Scene3D/model
  picker widgets that speak Studio adapter contracts.
- Control value loading and saving through the authoring mutation API.

Acceptance:

- A host exposes controls on a component template, and Studio renders complete
  editing UI without host-specific inspector code.

### 6. Style System and Design Tokens

Current:

- `styleruntime` exists and supports theme controls, CSS, style impact previews,
  reset-to-kit behavior, and workbench state.
- Style behavior is not yet a product-level contract comparable to a Webflow
  style panel.

Missing:

- A design-token model for colors, type, spacing, radii, shadows, borders,
  assets, and named recipes.
- Selector and component-scope strategy that does not expose raw CSS as the
  primary authoring path.
- Breakpoint-specific overrides.
- Pseudo state editing for hover/focus/active where appropriate.
- Guardrails that prevent invalid or inaccessible style combinations.

Acceptance:

- An editor can change a section's visual treatment, inspect which nodes are
  affected, preview responsive changes, and reset to the design system without
  writing CSS.

### 7. Layout and Box Model Editing

Current:

- `blocklayoutruntime` has rows, library, selection, drag binding, reorder, and
  visibility runtime pieces.
- There is no complete Webflow-like layout model in Studio.

Missing:

- A Studio layout schema for stack/grid/flex/section/container patterns.
- Visual controls for spacing, alignment, order, wrapping, columns, and
  responsive behavior.
- Component palette insertion into named regions.
- Collision and constraint feedback for invalid drops.

Acceptance:

- An editor can build a page structure from components, arrange it responsively,
  and keep the result mapped to typed GoSX components.

### 8. CMS, Content, and Resource Binding

Current:

- `ResourceAdapter` and `ResourceBinding` describe host resources.
- `gosx-cms` owns content schemas, media records, lifecycle, storage, and
  generated assets.

Missing:

- Executable adapter interfaces for listing, selecting, previewing, validating,
  and saving bound resources.
- A common source-picker UI for CMS collections, products, media, contacts,
  flows, settings, and generated artifacts.
- A binding resolution protocol for preview render, publish validation, and
  missing resource warnings.

Acceptance:

- A component can bind to CMS content or host resources through a Studio picker,
  and the preview/publish panels understand whether that binding is valid.

### 9. Media and Asset Manager

Current:

- Hosts can pass media assets to Studio.
- Showcase 3D defines a plugin contract for generated model artifacts.

Missing:

- Upload, search, select, replace, crop, focal point, alt text, captions,
  metadata, licensing, and version history.
- Asset readiness checks that feed publish review.
- Generated-asset lifecycle UI for images, 3D models, and future media types.

Acceptance:

- An editor can replace a hero image, set focal point and alt text, preview the
  result across breakpoints, and satisfy publish readiness without leaving
  Studio.

### 10. Flow and Form Designer

Current:

- Flow models exist in Studio contracts.
- Noni and Pajaritos still rely heavily on CMS Studio and app-specific flow
  rendering.

Missing:

- `FlowDesigner.gsx` and associated engine/island pieces owned by Studio.
- Form field editing, validation rules, submit actions, handler binding,
  lifecycle status, preview execution, and publish readiness.
- A host adapter for executing, saving, and publishing flow drafts.

Acceptance:

- A non-technical editor can create or modify a contact/signup/purchase flow,
  bind it to a page component, test it, and publish it with readiness checks.

### 11. Publish, Revision, and Environment Promotion

Current:

- Readiness and lifecycle concepts exist.
- Deploy scripts now distinguish staging/prod links for Noni and Pajaritos, but
  the editor does not own an end-to-end environment promotion workflow.

Missing:

- Draft, review, scheduled publish, staging preview, production promotion,
  rollback, and audit-log contracts.
- Generic publish panel that reads host readiness and writes host lifecycle
  actions.
- Environment URL slots for staging and production, including `TBD` support
  where production is not ready.
- Restore-point creation before publish.

Acceptance:

- From Studio, an editor can preview staging, resolve readiness warnings,
  publish, see what changed, and roll back if needed.

### 12. Responsive Preview and Breakpoint Overrides

Current:

- `CanvasViewport` and zoom levels exist.
- Preview/runtime pieces can mirror some state into the iframe.

Missing:

- Breakpoint-specific style/layout/control persistence.
- Device presets, custom widths, orientation testing, and overflow diagnostics.
- Visual indicators for which settings inherit versus override.

Acceptance:

- Desktop, tablet, and mobile edits can be made intentionally, previewed
  accurately, and saved without cross-breakpoint surprises.

### 13. Interactions and Animation

Current:

- There is no product-level interactions panel comparable to Webflow
  interactions.

Missing:

- Motion primitives for reveal, scroll, hover, state transitions, timeline-like
  sequences, and route/page transitions.
- A safe runtime contract for interaction previews.
- Accessibility and reduced-motion handling by default.

Acceptance:

- Editors can add common interactions without code, and published pages respect
  performance and accessibility constraints.

### 14. Collaboration, Undo, and History

Current:

- GoSX has hub, CRDT, and workspace primitives available at the framework level.
- Studio does not yet expose collaborative editing behavior.

Missing:

- Presence, shared selection, live cursors, conflict-free drafts, autosave,
  undo/redo, branchable revisions, comments, and handoff states.
- Permission checks before hub connection and before privileged mutations.
- Durable history that can explain who changed what and when.

Acceptance:

- Two editors can work in the same Studio project, see each other's selection,
  avoid destructive overwrites, undo local work, and recover prior versions.

### 15. Permissions, Workspaces, and Product Packaging

Current:

- `ShellConfig` includes labels, actions, features, resources, and permissions.
- The roadmap calls for a product package composing `gosx-cms + gosx-admin +
  gosx-studio`.

Missing:

- Workspace/project/team model.
- Role-based permissions for edit, design, publish, media, flow, admin, and
  plugin operations.
- Hosted product package with setup, templates, onboarding, billing hooks if
  needed, and deployment environment configuration.
- First-run project wizard and blank-site/site-template scaffolds.

Acceptance:

- A new project can be created from a template, assigned to users, edited in
  Studio, connected to deployment targets, and published without bespoke app
  wiring.

### 16. Plugin Registry and Marketplace Surface

Current:

- Showcase 3D is the first plugin-shaped contract.

Missing:

- Plugin registry, capability negotiation, installation/configuration UI,
  permissions, lifecycle, asset ownership, and publish readiness integration.
- A common placement UI for plugin components.

Acceptance:

- A plugin can add component templates, controls, generated assets, canvas
  preview behavior, and publish checks without patching Studio core.

### 17. Quality, Docs, and Public Readiness

Current:

- Go tests and parity e2e tests exist for several runtime slices.
- `parity_e2e/authoringruntime_test.ts` now covers the browser-side authoring
  result feedback loop without requiring a host server: changed-object
  selection, canvas selection sync, preview refresh, save-state feedback, and
  `gosxstudio:authoring-result` dispatch.
- Noni and Pajaritos now have persisted editor-surface smoke coverage that
  submits the actual visible authoring payloads for create-page, add-component,
  duplicate-component, and edit-control paths into their host persistence
  layers.
- The Studio Playwright parity harness now clicks visible authoring controls for
  create-page, add-component, duplicate-component, edit-control, page metadata,
  visibility, and reorder workflows, then verifies managed action feedback
  selects the changed surface, refreshes the preview, and updates save-state UI.
- A local Pajaritos browser check now clicks the visible create-page control
  through a real admin editor page, posts through the GoSX authoring action with
  CSRF, follows the redirect, and verifies the new page appears. That check also
  caught and fixed a transformed canvas overlay that was intercepting authoring
  controls.
- `parity_e2e/reference_apps_authoring_test.ts` now provides an opt-in
  browser harness that boots sibling Muddy/Noni and Pajaritos apps with temp
  data, verifies visible authoring buttons receive pointer events, and submits
  create-page, add-section, duplicate-section, and save-field actions through
  the real admin editor action flow. Muddy verifies duplicate/save-field state
  after reload; Pajaritos verifies native create-page, add-section, duplicate,
  and save-field payloads, redirects, browser-visible authoring feedback, and
  persisted page/section reflection, staging/production readiness, preview
  refresh after client-side editor actions, and a visible restore-point
  rollback after reload.
- `.github/workflows/reference-apps.yml` now promotes the Muddy/Noni and
  Pajaritos browser workflow into default `gosx-studio` CI by checking out the
  sibling app layout and running the reference-app e2e suite.
- `parity_e2e/reference_apps_visual_a11y_test.ts` now adds the first
  host-backed browser quality gate for both reference apps. It validates the
  editor shell, visible site-map/canvas surface, publishing section,
  responsive preview iframe, desktop/mobile viewport fit, accessible names,
  keyboard focus, reduced-motion behavior, sampled contrast, clipped command
  text, and pointer-safe authoring controls. This gate caught and fixed a
  hidden Muddy preview mode, a preview-to-publish workbench runtime alias, a
  Muddy toolbar pointer overlap, and a Pajaritos mobile toolbar overflow.
- `parity_e2e/reference_apps_performance_test.ts` now enforces the first
  reference-app browser performance budgets for shell ready time, preview
  iframe latency, publish readiness latency, runtime payload bytes, largest
  runtime resource size, preview-frame count, canvas node/overlay counts,
  layout sampling, and publish-readiness item count. The initial baseline
  guards Muddy/Noni's current editor runtime payload, including the large dev
  WASM runtime, and keeps Pajaritos on a much lighter runtime path.
- `docs/PUBLIC_READINESS.md` names the release posture concerns.

Missing:

- Broader visual, accessibility, and performance gates for media, flow
  designer, inspector, source binding, and plugin workflows beyond the first
  reference-app browser gate.
- Golden or diff-based visual regression baselines for the shell, canvas,
  inspector, site map, media, flow designer, publish panel, and responsive
  preview.
- Deeper accessibility checks for complete keyboard navigation paths, focus
  order, screen reader structure, and full WCAG contrast coverage.
- Production-build/TinyGo size budgets, third-host baselines, and deeper
  performance coverage for media, inspector, flow designer, source binding,
  plugins, and publish promotion beyond the first reference-app editor gate.
- API versioning and migration notes before public release.

Acceptance:

- The same no-code workflows pass in both reference apps, docs show how a third
  host integrates Studio, and the public API is intentionally small.

## Extraction Hotspots

These are the current places to mine for reusable Studio behavior.

| Repository | Hotspot | Why it matters |
| --- | --- | --- |
| `gosx-studio` | `authoring.go` | New neutral authoring projection. This should become the data source for shell, site-map, canvas, and inspector surfaces. |
| `gosx-studio` | runtime packages | Existing browser behaviors should be composed into engines rather than copied as host scripts. |
| `muddy-noni-commerce` | `app/admin/editor/site_map.server.go:456` through `app/admin/editor/site_map.server.go:1005` | Noni has site-map view, composition intent, and control-view logic that overlaps with Studio `AuthoringSurface`. |
| `muddy-noni-commerce` | `app/admin/editor/page.server.go:267` and after | Noni owns a large shell/inspector/publish/workbench implementation that should move behind reusable Studio surfaces. |
| `muddy-noni-commerce` | `internal/gosxstudio/runtime.go:109` | Runtime ownership still passes through `gosx-cms/studio` for workbench, command palette, and state runtime assets. |
| `pajaritos-forest-school` | `internal/site/store.go:824` | Pajaritos still builds most CMS Studio panels locally. |
| `pajaritos-forest-school` | `internal/site/store.go:1051` through `internal/site/store.go:1162` | Pajaritos proves that a second host can feed a Studio store and site map. |
| `gosx-cms` | `studio/*` | This package still owns behavior that belongs either in CMS lifecycle/content or Studio authoring surfaces. |

## Phased Work Plan

### Phase 0: Stabilize the Boundary

- Freeze the names and responsibilities of CMS, Admin, Studio, and the future
  product package.
- Decide what remains in `gosx-cms/studio` versus what moves to `gosx-studio`.
- Add migration notes for host apps.
- Keep Noni and Pajaritos green while extraction proceeds.

### Phase 1: Add Authoring Actions

- Introduce an `AuthoringAdapter` or equivalent mutation interface. Done in the
  first execution slice.
- Implement apply-intent, save-control, route metadata, first visibility,
  reorder, duplicate, and delete operations. Broader validation operations
  remain.
- Add server-action helpers and focused tests.
- Wire both Noni and Pajaritos to the same action contract. Pajaritos and Noni
  now have create-page, add-component, save-control, page metadata, visibility,
  reorder, duplicate, and delete slices.

### Phase 2: Extract the Visible Shell

- Build `StudioShell.gsx`, `SiteNavigator.gsx`, `InspectorIsland.gsx`, and
  `PublishPanel.gsx` in `gosx-studio`. The first reusable workbench shell view
  payload is done; complete `.gsx` ownership remains.
- Replace app-local editor chrome with the shared shell. Noni and Pajaritos now
  share the Go-side workbench shell projection, but still render host-local
  wrappers/panels.
- Preserve app-specific panels as slots or adapter-fed views.

### Phase 3: Ship Core Engines

- Build `SiteMapEngine.gsx` for page graph authoring.
- Build `CanvasEngine.gsx` for preview, selection, inline editing, and runtime
  coordination.
- Build `BlockLayoutEngine.gsx` around the existing block layout runtime.
- Add e2e coverage for create page, add component, edit field, reorder, and
  responsive preview.

### Phase 4: Complete Product Workflows

- Extract flow/form design into Studio.
- Add media manager and source picker surfaces.
- Add design-token/style/layout editing with breakpoint overrides.
- Add publish/revision/environment promotion as a generic workflow.

### Phase 5: Productize

- Create the product package that composes `gosx-cms + gosx-admin +
  gosx-studio`.
- Add project/workspace/team setup.
- Add hosted deployment configuration.
- Add plugin registry and first-run project templates.
- Prepare public docs and API stability guarantees.

## Next Execution Queue

1. Extract Noni site-map view and composition-intent logic into
   `gosx-studio`, replacing duplicate app-local view builders with
   `AuthoringSurfaceView`.
2. Done on 2026-06-01: add a Studio authoring mutation interface for applying
   composition intents and saving control values.
3. Done on 2026-06-01: add server-action helpers for Studio authoring
   mutations and wire Pajaritos/Noni composition/control persistence for the
   first real content flows.
4. Done on 2026-06-01: add first page title/route metadata mutation payloads
   and host persistence in both reference apps.
5. Done on 2026-06-01: add first component visibility mutation payloads and
   host persistence in both reference apps.
6. Done on 2026-06-01: add first component reorder mutation payloads and host
   persistence in both reference apps.
7. Done on 2026-06-01: add component delete mutation payloads and host
   persistence in both reference apps.
8. In progress on 2026-06-01: build a reusable shell path that consumes
   `ShellConfig` and `AuthoringSurfaceView`. The shared Go workbench shell
   projection, Studio-owned toolbar renderer, and first frame/stage/rail
   renderer primitives are done; the complete `StudioShell.gsx` surface
   remains.
9. Done on 2026-06-01: add duplicate component operations with a dynamic
   component-instance model, host-backed persistence, and visible site-map
   controls.
10. Done on 2026-06-01: add a Studio-owned site-map authoring panel renderer
    for metadata, editable fields, reorder, duplicate, visibility, delete,
    composition intents, and page lists; adopt it in Muddy/Noni and Pajaritos.
11. Done on 2026-06-01: add a Studio-owned managed form scaffold for site-map
    authoring panels, including composition-intent forms, operation forms, and
    CSRF support; adopt it in Muddy/Noni and Pajaritos.
12. Done on 2026-06-01: add Studio-owned site-map engine frame segments for
    root semantics, header chrome, stats, and source legend; adopt them in
    Muddy/Noni around the current interaction island.
13. Done on 2026-06-01: add a Studio-owned site-map board interaction runtime
    and expose Muddy/Noni data hooks plus the Pajaritos standalone runtime
    mount.
14. Move workbench, command palette, and state runtime ownership out of
   `gosx-cms/studio` when the dependent hosts are ready.
15. Implement `SiteMapEngine.gsx` against the existing `CompositionWorkspace`
   and `WorkspaceCanvas` contracts.
16. Implement `InspectorIsland.gsx` for the current `ControlKind` set with
   dirty/valid/saving states.
17. Implement canvas selection and inline text editing through the Studio
    preview/selection/field runtimes.
18. Add Noni and Pajaritos e2e smoke tests for create page, add component,
    edit control, responsive preview, staging publish readiness, and rollback.
19. Write the visual-system spec for the Studio editor UI before the major
    shell styling pass, including density, typography, color roles,
    accessibility states, and responsive behavior.
20. Done on 2026-06-01: render visible create-page/add-section apply controls
    from shared site-map composition intent payloads in Noni and Pajaritos.
21. Done on 2026-06-01: add reference-app visual/accessibility gates for the
    Studio shell, site-map/canvas surface, publishing section, responsive
    preview, keyboard focus, accessible names, reduced motion, contrast, and
    desktop/mobile viewport fit.
22. Done on 2026-06-01: add reference-app performance budgets for runtime
    payload, preview latency, canvas overlay cost, and publish readiness.
23. Done on 2026-06-01: add the structural authoring fragment-refresh contract
    and graduate create-page, add-section, duplicate-section, and
    delete-section capable forms to managed in-place authoring in both
    reference apps.
24. Done on 2026-06-01: add the Studio-owned visible site-map board renderer
    and adopt it in Muddy/Noni, replacing the local `SiteMapCanvasIsland`
    markup while preserving the runtime data contract.

## Execution Log

- 2026-06-01: Added `mutations.go` and `mutations_test.go` with typed
  authoring operations, stable form field names, `AuthoringMutation`,
  `AuthoringAdapter`, validation/result envelopes, and
  `AuthoringActionHandler`. `AuthoringSurfaceView` now carries mutation field
  names plus form payloads for intents and controls. Verified with
  `go test ./...` in `gosx-studio`.
- 2026-06-01: Wired Pajaritos to `AuthoringActionHandler` with an
  `authoring` action path and a host `ApplyAuthoringMutation` implementation
  for `pages.home.hero.headline`. The public home page now renders the headline
  from site settings metadata, and the Studio control carries the current
  value. Verified with `go test ./...` and `gosx check app/page.gsx
  app/admin/editor/page.gsx` in `pajaritos-forest-school`.
- 2026-06-01: Wired Noni to `AuthoringActionHandler` with an `authoring` action
  path and a host adapter for `home.hero.headline`. The action saves through
  Noni's CMS settings pipeline, creates a revision checkpoint, and returns a
  preview refresh result for the Studio runtime. Noni site-map controls now
  carry the same mutation payload and DOM authoring metadata for placed
  components. Verified with `go test ./...` plus `gosx check app/page.gsx
  app/admin/editor/page.gsx` in `muddy-noni-commerce`.
- 2026-06-01: Replaced Noni's duplicated site-map view assembly with the shared
  `gosx-studio` site-map projection and verified the host shell exposes the
  shared `siteMap` payload in Pajaritos.
- 2026-06-01: Added host persistence for composition intents: Noni can create a
  draft page and enable a home section, while Pajaritos can create draft pages
  and enable a family request form on the home page.
- 2026-06-01: Extended Pajaritos control persistence beyond site settings so
  flow step labels and submit labels save through the same
  `AuthoringActionHandler` mutation boundary.
- 2026-06-01: Added `authoringruntime`, bundled it into the Studio engine
  runtime, and exposed `AuthoringRuntimeScript()` for hosts still using
  `gosx-cms/studio` composition helpers. Muddy now proves the delegated engine
  bundle contains the runtime, and Pajaritos injects the standalone runtime
  into its generated workbench. Successful authoring actions can now refresh
  previews, mark the changed node/control, and publish a
  `gosxstudio:authoring-result` event.
- 2026-06-01: Added shared page metadata mutation payloads to the site-map
  projection and wired both reference apps to persist page title/route edits
  through `AuthoringActionHandler`. Noni validates reserved storefront routes;
  Pajaritos validates duplicate content-page routes. Both apps now expose
  visible page detail controls without adding client copy that names GoSX
  Studio.
- 2026-06-01: Added shared component visibility mutation payloads and wired the
  first persisted visibility operation in both reference apps. Noni can show or
  hide homepage sections, while Pajaritos can show or hide the home family
  request form, with both returning preview-refresh authoring changes.
- 2026-06-01: Added a Playwright smoke test for `authoringruntime` that runs in
  a real Chromium page and verifies successful authoring results select the
  changed component, sync canvas selection, refresh the preview iframe, update
  save feedback, and emit `gosxstudio:authoring-result`.
- 2026-06-01: Added shared component duplicate mutation payloads and wired
  repeatable home-section persistence in both reference apps. Noni now stores
  instance keys such as `hero__copy_2` while preserving the template key for
  rendering and settings saves; Pajaritos stores repeatable ordered home
  section instances in metadata.
- 2026-06-01: Added visible create-page/add-section apply controls for
  composition intents. `AuthoringSiteMapView` now exposes action labels,
  scoped form IDs, and explicit authoring payload fields; Muddy and Pajaritos
  render those controls without leaking platform copy, and Pajaritos now builds
  its panel site-map from `NoCodeAuthoringSurface`.
- 2026-06-01: Added `WorkbenchShellView` and `CanvasPreviewShellView` in
  `gosx-studio` as the first reusable shell extraction slice. Muddy now adapts
  its editor workbench shell map through that shared projection instead of
  constructing preview-shell attrs, zoom levels, and rail resizers locally.
  Pajaritos now exposes `hostWorkbenchShell` from the same projection and uses
  it for editor host metadata. Both apps keep visible client copy host-branded
  while footer attribution remains limited to "Made with GoSX by M31 Labs."
- 2026-06-01: Added persisted editor-surface smoke tests in both reference apps.
  The tests pull the create-page, add-component, duplicate-component, and
  edit-control payloads exposed to the editor surface, submit them through the
  host mutation paths, and assert that pages, home sections, duplicated
  sections, and text controls actually persist.
- 2026-06-01: Added a browser-click authoring surface smoke in
  `parity_e2e/authoring_surface_workflows_test.ts`. The test clicks visible
  create-page, add-section, duplicate-section, and save-headline controls,
  drives scoped/external forms, simulates GoSX managed action results, and
  verifies authoring runtime selection, preview refresh, save feedback, and
  persisted workflow state.
- 2026-06-01: Ran the first real Pajaritos admin editor browser click against a
  local app server. The create-page form now submits through the visible
  control and persists after redirect; Pajaritos also has a CSS regression test
  for the canvas stacking and pointer-events rules that keep transformed site
  map layers from blocking authoring controls while preserving canvas-node
  clicks.
- 2026-06-01: Added ordered `formInputs` to Studio authoring mutation views,
  surface controls/intents, and site-map page/component/control operations.
  Muddy and Pajaritos now render visible create-page/add-section, page
  metadata, component reorder, duplicate, visibility, and delete hidden-input
  payloads from the shared Studio projection instead of app-local field lists.
- 2026-06-01: Added opt-in reference-app browser e2e coverage in
  `parity_e2e/reference_apps_authoring_test.ts`. It boots sibling Muddy/Noni
  and Pajaritos apps with temp data directories, checks that visible
  create-page/add-section buttons are not blocked by overlays, and verifies the
  real authoring POST/redirect path in both admin editors.
- 2026-06-01: Added a top-level `editableControl` projection to
  `AuthoringSiteMapView` and wired Muddy/Pajaritos visible save-field panels to
  the same ordered `formInputs` contract. Both apps now isolate metadata,
  editable-field, reorder, duplicate, visibility, and delete panels behind
  scoped external forms so the main save form no longer leaks competing hidden
  operations into authoring submissions.
- 2026-06-01: Broadened the reference-app browser e2e to click duplicate and
  save-field controls. Muddy now asserts duplicate and headline state after a
  browser reload; Pajaritos asserts pointer reachability, native
  duplicate/save-control payloads, 303 redirects, and visible success feedback
  through its host PRG notice path.
- 2026-06-01: Wired Pajaritos persisted CMS pages into the Studio site-map
  projection and added a visible page/section summary to the editor shell. The
  reference-app browser e2e now verifies create-page and add-section payloads,
  success notices, and reloaded page/section state for Pajaritos.
- 2026-06-01: Added a Pajaritos restore-point panel in the publishing section
  plus a `restoreRevision` editor action for site settings rollback. The
  reference-app browser e2e now clicks the visible rollback control after
  authoring changes and verifies the restored section state after reload.
- 2026-06-01: Promoted the Muddy/Noni plus Pajaritos reference-app browser e2e
  into `gosx-studio` default CI. Pajaritos now exposes staging/production
  environment readiness in the publishing section, and the browser e2e asserts
  publish readiness plus preview refresh after a client-side editor action.
- 2026-06-01: Added a shared reference-app Playwright harness plus
  `reference_apps_visual_a11y_test.ts`, and wired it into the default
  reference-app CI workflow. The gate checks both reference app editors across
  desktop and mobile for shell/canvas/publish/preview visibility, pointer-safe
  controls, accessible names, keyboard focus, reduced motion, sampled contrast,
  clipped command text, and viewport fit. It also drove fixes in Muddy's
  visible Preview mode/runtime mapping/toolbar layout and Pajaritos' mobile
  editor overflow.
- 2026-06-01: Added `reference_apps_performance_test.ts` and wired it into the
  default reference-app browser workflow. The gate measures shell ready time,
  preview iframe readiness, publish readiness, editor runtime payload bytes,
  largest runtime resources, preview-frame count, canvas/site-map node count,
  overlay count, layout sample time, and publish-readiness item count. The
  first focused run established Muddy/Noni around a 5.6 MB dev editor runtime
  payload with a 4.7 MB WASM resource, while Pajaritos stayed around a 38 KB
  runtime payload.
- 2026-06-01: Added `RenderWorkbenchToolbar`,
  `RenderWorkbenchSaveStatus`, and `RenderWorkbenchHistoryControls` in
  `gosx-studio` as the next visible shell extraction. Pajaritos now renders its
  top editor toolbar from the Studio-owned `WorkbenchShellView` path, including
  host-branded title, summary, save state, history controls, preview link, and
  save action without reintroducing "GoSX Studio" client copy.
- 2026-06-01: Extended the shared toolbar renderer with Studio-owned modebar,
  metric-strip, and command-palette chrome. Muddy/Noni now renders its top
  editor toolbar from `WorkbenchShellView` and deleted the app-local toolbar,
  modebar, metric strip, command palette, save status, and history controls.
  The reference-app browser suite verifies both apps across authoring,
  performance, desktop/mobile visual integrity, accessibility, and host-branded
  copy expectations.
- 2026-06-01: Added `RenderWorkbenchFrame` and
  `RenderWorkbenchRailResizer` to `gosx-studio` as the next shell extraction
  path. The renderer owns root/form/stage/rail/main/canvas-shell semantics,
  stable renderer markers, CSRF/autosave/state attributes, feature-flag
  passthrough, and host slots without injecting product copy. Muddy/Noni now
  consumes the Studio-owned rail resizer and removed its local `.gsx` resizer
  component while keeping the existing route-owned panel markup intact.
- 2026-06-01: Flipped Muddy/Noni's editor shell onto
  `RenderWorkbenchFrameSegments` for the outer root, form, stage, rails, rail
  resizers, main canvas section, canvas shell, and board. The route now keeps
  only client-specific panel slots and action forms in `.gsx`; shared shell
  semantics and native form/autosave/CSRF/state attributes come from
  `gosx-studio`.
- 2026-06-01: Flipped Pajaritos' server-composed editor workbench onto
  `RenderWorkbenchFrame` while preserving its CMS sections, family-form editor,
  lifecycle controls, environment readiness, restore points, and inline runtime
  nodes. Both reference apps now consume Studio-owned frame semantics for their
  primary editor shells.
- 2026-06-01: Tightened the host-backed reference-app authoring smoke test so
  real Muddy/Noni and Pajaritos field edits run through GoSX managed form
  feedback, emit `gosxstudio:authoring-result`, record changed-object selection
  on the workbench, and refresh preview state without relying on visible
  "GoSX Studio" copy. Muddy admin now loads the GoSX navigation/form runtime,
  and both reference apps mark editable-control authoring forms as managed
  POST forms while leaving create/add/duplicate flows on their native
  redirect-safe path.
- 2026-06-01: Burned down the in-place field-save UX gap. Studio now owns
  marked authoring form submits before the generic navigation layer, posts them
  as JSON actions, keeps the active editor panel mounted, updates visible
  save-detail chrome with the returned authoring message, records changed
  object selection, and refreshes preview without a manual page reload.
  The runtime also avoids treating the workbench form's save-state attribute as
  display text, preventing field saves from deleting the editor shell.
- 2026-06-01: Extended the in-place authoring path to page metadata edits in
  both reference apps. Muddy/Noni and Pajaritos now mark page metadata forms as
  Studio-managed authoring forms, and the reference-app browser workflow edits
  title and route fields without leaving the active editor, while asserting
  page-level authoring results, selected-page feedback, save-detail chrome, and
  preview refresh.
- 2026-06-01: Extended in-place authoring to component visibility controls.
  Managed visibility forms now keep the editor mounted, refresh preview,
  select the affected component, update save-detail chrome, and flip the local
  panel from Hide to Show (or back) by rewriting the next
  `gosx_studio_visible` value after a successful result.
- 2026-06-01: Extended in-place authoring to component reorder controls.
  Managed reorder forms now keep the editor mounted, refresh preview, select
  the affected component, update save-detail chrome, and rewrite the local
  position output, next target field, and Move up/down button after a
  successful result in both reference apps.
- 2026-06-01: Added the structural authoring fragment-refresh contract and
  graduated create-page, add-section, duplicate-section, and delete-section
  capable forms to managed in-place authoring. Studio action results now carry
  refresh fragment selectors; the authoring runtime fetches the editor
  document, replaces those fragments, remounts runtimes, refreshes preview, and
  preserves visible save feedback. Muddy/Noni and Pajaritos both use the
  contract for structural authoring controls without adding client-visible
  platform copy.
- 2026-06-01: Added `RenderSiteMapAuthoringPanels` in `gosx-studio` and moved
  Muddy/Noni and Pajaritos page metadata, editable field, reorder, duplicate,
  visibility, delete, composition-intent, and page-list panel markup behind
  that shared renderer. The renderer keeps legacy data attributes for the
  existing browser workflows while avoiding visible platform-branded copy.
- 2026-06-01: Added `RenderSiteMapAuthoringForms` in `gosx-studio` and moved
  Muddy/Noni and Pajaritos hidden composition-intent and site-map operation
  form scaffolds behind it. The shared renderer owns managed GoSX form attrs
  and CSRF input support for the forms submitted by the shared panels.
- 2026-06-01: Added `RenderSiteMapEngineSegments` in `gosx-studio` as the
  first visible site-map engine extraction. The renderer owns root semantics,
  header chrome, summary stats, source legend, and stable renderer markers
  without adding client-visible platform copy. Muddy/Noni now wraps its
  existing `SiteMapCanvasIsland` with those Studio-rendered segments while the
  full interaction board remains the next extraction target.
- 2026-06-01: Added `sitemapruntime` in `gosx-studio` and included it in the
  engine runtime bundle plus a standalone `SiteMapRuntimeScript()`. The
  runtime owns board filter/detail/palette/zoom/density/grid controls,
  selected-node detail synchronization, composition-intent active state, and
  fragment remount binding. Muddy/Noni exposes the matching data hooks on its
  current `SiteMapCanvasIsland`, and Pajaritos mounts the standalone runtime
  next to its server-composed shell.
- 2026-06-01: Added `RenderSiteMapBoard` in `gosx-studio` and moved Muddy/Noni's
  visible site-map board behind that shared renderer. The old app-local
  `SiteMapCanvasIsland` markup is gone, and the board keeps the existing
  interaction/runtime data hooks without adding visible platform copy.
- 2026-06-01: Added the host-facing `RenderSiteMapEngine` render path in
  `gosx-studio`. It now composes site-map frame chrome, authoring panels,
  engine host mount, shared board, and external managed forms into one
  integration contract. Muddy/Noni now passes only `data.siteMapEngineSurface`
  inside the workbench form and `data.siteMapAuthoringForms` outside it, so the
  host route no longer manually assembles the site-map surface while preserving
  valid form nesting and engine runtime registration.
- 2026-06-01 (sequoia): Added a continuous infinite-canvas viewport ("Figma
  feel") to the shared site-map board. `sitemapruntime` now drives
  wheel-to-cursor zoom, drag-to-pan, keyboard `+`/`-`/`0` zoom, and
  `setState({scale,panX,panY})` against a new
  `[data-studio-site-map-pan-surface]` transform layer, with a live
  `[data-studio-site-map-zoom-readout]`. `RenderSiteMapBoard` emits the pan
  surface (wrapping SVG paths + node layers, leaving the selection card as a
  fixed overlay), a focusable/`aria-label`led canvas region, and zoom
  in/out/reset action controls. Parity coverage added in
  `parity_e2e/sitemapruntime_test.ts` (6 cases: initial transform, zoom
  buttons, deterministic `setState` pan, wheel zoom, keyboard zoom, drag pan);
  board markup contract asserted in `sitemap_board_test.go`. Backward
  compatible — the legacy discrete `zoom` enum (fit/wide) is untouched.
- 2026-06-01 (sequoia): Added marquee multi-select to the shared site-map
  board. `sitemapruntime` now supports shift+left-drag rubber-band selection
  (a dynamically created `[data-studio-site-map-marquee]` overlay; AABB
  intersection against node rects), shift-click to toggle a node in/out of the
  multi-selection, and `Escape` to clear it. A new `selectedNodes` state field
  (`data-studio-site-map-selected-nodes`) is part of the state machine, so
  `syncSelection` preserves multi-selection across unrelated state changes; a
  node renders selected when it is the primary `selectedNode` OR in the
  multi-set. Pan stays on plain left-drag (Step A) — marquee is additive on
  shift. 4 new parity cases in `sitemapruntime_test.ts` (rubber-band,
  shift-toggle, Escape, survives-sync); runtime-only, no renderer change.
- 2026-06-01 (fir): Shipped the opt-in **Canvas2D site-map engine foundation**
  (Hybrid parallel track). New `RenderSiteMapCanvasEngine(view,
  SiteMapCanvasOptions{Enabled:...})` in `sitemap_canvas.go` renders the same
  `AuthoringSiteMapView` page graph onto the merged Phase-2 `gosx.CanvasBoard`
  (surface kind `canvas2d`): workspace nodes → `rect` + `label` board nodes laid
  out deterministically by layer-column / node-row (the view exposes no numeric
  coords), `workspaceLinks` → `line` connectors between node centers, group-keyed
  fill colors, pick events wired to `handleSiteMapPick`. **Default OFF** — emits
  an inert hidden hook unless `Enabled:true`; the `RenderSiteMapBoard` DOM path
  is untouched. Go coverage in `sitemap_canvas_test.go`; full gosx-studio suite
  green. Merged to main as `0b23c8e`.
- 2026-06-01 (birch): Shipped **keyboard node navigation** on the shared board.
  Arrow keys move the primary `selectedNode` to the spatially-nearest visible
  node (cost = primary-axis distance + 2× perpendicular, half-plane filtered;
  first arrow with no selection picks topmost-leftmost). `Enter`/`Space`
  dispatch `gosxstudio:site-map-activate {key}` for hosts to open the inspector.
  6 parity cases; merged to main as `b597deb`.
- 2026-06-01 (sequoia): Hotfix `f3847c8` — keyboard handler must not hijack
  `Enter`/`Space`/arrows when focus is on an interactive control
  (button/link/contenteditable). Added an `isInteractiveControl` guard + a parity
  regression test. (Caught by the reference-app e2e: a focused "Add" button +
  `Enter` was being `preventDefault`ed, blocking native activation.)

## Next Execution Slice

Shipped on the shared board: infinite-canvas pan/zoom, marquee multi-select, and
keyboard node navigation (arrows + `Enter`/`Space` activate, regression-fixed to
respect focused controls). A `:focus-visible` ring is a minor host-CSS
follow-up. The next practical slices, in order, are:

1. **Node drag-to-reposition with persisted canvas positions**: free placement
   on the pan surface emitting `gosxstudio:sitemap-node-moved {key,x,y}`, host
   persistence into the "Saved canvas positions" (`layout-graph`) store, and
   snap-to-grid.
2. **Parallel track — Canvas2D engine (foundation shipped 2026-06-01, `0b23c8e`).**
   `RenderSiteMapCanvasEngine` (opt-in, default-off) mounts `gosx.CanvasBoard`
   for the page graph. Remaining: browser/WASM hydration e2e for the canvas
   surface; grid/marquee/drag wiring on it (the primitive ships
   render/pan/zoom/pick only); deterministic positions persisted instead of
   structural grid; and a host opt-in + flag flip once at parity with the DOM
   board.
3. Decide the Pajaritos map path: adopt the shared board directly, or provide a
   host-styled board slot that still uses `AuthoringSiteMapView` plus
   `sitemapruntime`.
4. Keep broadening visual/accessibility/performance checks as inspector, media,
   flow, and publish surfaces move behind reusable Studio-owned render paths.

## Definition of Done

GoSX Studio is fulfilling the no-code web development promise when:

- A new host can integrate Studio by implementing adapters and catalogs, not by
  copying Noni editor code.
- A blank project can create pages, add sections, edit content/media/style,
  tune responsive views, bind CMS resources, configure flows, and publish.
- Noni and Pajaritos use the same Studio shell and action contracts.
- Staging and production promotion are visible from the editor, even when a
  production URL is intentionally marked `TBD`.
- Plugin components can be installed, configured, placed, previewed, and
  validated through the same authoring model.
- Collaboration, history, undo/redo, permissions, audit, and rollback protect
  real teams from destructive edits.
- Public docs explain the package boundary, host integration steps, adapter
  contracts, action contracts, runtime contracts, and supported editor
  workflows.
