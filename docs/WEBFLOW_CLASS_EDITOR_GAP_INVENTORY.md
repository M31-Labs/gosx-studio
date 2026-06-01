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

Missing:

- A Studio-owned site-map engine that renders pages, routes, components,
  source bindings, readiness, and allowed operations.
- Full route editing across all editable pages, page grouping, selection state,
  and conflict validation inside the visual graph.
- Keyboard and pointer interactions suitable for a visual graph.

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
  create-page, add-component, duplicate-component, and edit-control workflows,
  then verifies managed action feedback selects the changed surface, refreshes
  the preview, and updates save-state UI.
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
  after reload; Pajaritos verifies native duplicate/save-field payloads,
  redirects, and browser-visible authoring feedback after reload.
- `docs/PUBLIC_READINESS.md` names the release posture concerns.

Missing:

- Default-CI browser-driven editor workflows across both Noni and Pajaritos,
  plus Pajaritos draft-state reload reflection, preview refresh, staging
  publish readiness, and rollback.
- Visual regression checks for the shell, canvas, inspector, site map, media,
  flow designer, publish panel, and responsive preview.
- Accessibility checks for keyboard navigation, focus, labels, reduced motion,
  contrast, and screen reader structure.
- Performance budgets for runtime payloads, preview latency, canvas overlays,
  and publish readiness checks.
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
   projection is done; the complete `StudioShell.gsx` surface remains.
9. Done on 2026-06-01: add duplicate component operations with a dynamic
   component-instance model, host-backed persistence, and visible site-map
   controls.
10. Move workbench, command palette, and state runtime ownership out of
   `gosx-cms/studio` when the dependent hosts are ready.
11. Implement `SiteMapEngine.gsx` against the existing `CompositionWorkspace`
   and `WorkspaceCanvas` contracts.
12. Implement `InspectorIsland.gsx` for the current `ControlKind` set with
   dirty/valid/saving states.
13. Implement canvas selection and inline text editing through the Studio
    preview/selection/field runtimes.
14. Add Noni and Pajaritos e2e smoke tests for create page, add component,
    edit control, responsive preview, staging publish readiness, and rollback.
15. Write the visual-system spec for the Studio editor UI before the major
    shell styling pass, including density, typography, color roles,
    accessibility states, and responsive behavior.
16. Done on 2026-06-01: render visible create-page/add-section apply controls
    from shared site-map composition intent payloads in Noni and Pajaritos.

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

## Next Execution Slice

The next practical slice is full browser automation plus the next visible shell
extraction:

- Promote the opt-in Muddy/Pajaritos reference-app browser e2e into default CI
  and add broader Pajaritos draft-state assertions for create-page/add-section
  reloads, preview-refresh, staging publish readiness, and rollback workflows
  with persisted browser-visible assertions.
- Continue the shell extraction by moving the next visible wrapper/toolbar
  piece behind a Studio-owned render path that consumes `WorkbenchShellView`.
- Extend `authoringruntime` host-backed smoke coverage to assert
  changed-object selection and preview refresh after those persisted actions.

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
