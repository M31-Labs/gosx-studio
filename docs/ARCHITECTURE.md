# Architecture

## Portal model

GoSX Studio is the whole web-editing admin portal for no-code,
user-authored websites: canvas designer and content/commerce/media/settings
back-office in one shell, Webflow-shaped.

This retires the earlier "CMS owns content / Admin owns back-office / Studio
owns authoring" triad. That three-library split is no longer what the code
does:

- `gosx-admin` is a **generic back-office toolkit dependency**. Studio
  consumes exactly one of its packages, `gosx-admin/blockstudio` (see
  `shell/cmsbridge.go`). Long term, if `blockstudio` proves studio-specific,
  it may move into `gosx-studio/shell` and admin shrinks to ops-CRUD
  primitives for non-website tools.
- The content-storage subsystem (`content, media, lifecycle{,/sqlstore},
  flows, style, store{,/file,/memory}, render, blocks`) that used to live in
  the standalone `gosx-cms` module has been folded into this module as
  `cms/{content,media,lifecycle,lifecycle/sqlstore,flows,style,store,store/file,
  store/memory,render,blocks}` — see "Release model" below and
  [`cms/PROVENANCE.md`](../cms/PROVENANCE.md). `cms/studio` (~64 files:
  `panels.go`, `workbench.go`, `chrome.go`, `site_canvas.go`,
  `flow_authoring.go`, `collab/`, …) is portal UI, not content storage, but it
  stayed a distinct top-tier package rather than merging into
  `shell`/`panels`/`backoffice` (the merge that the original restructure sketch
  floated was rejected before execution — see "Release model").
- `gosx-studio` is the portal itself: everything a site owner touches, from
  the canvas to the CRUD back-office, now including the folded `cms/` content
  layer.

## Package DAG

The module is a strict import DAG. Arrows mean "may import":

```text
core                  → (stdlib, m31labs.dev/gosx only)
authoring             → core
hostruntime           → core, the *runtime packages
canvas                → core
sitemap               → core, authoring
panels                → core, authoring
backoffice            → core
cms/<storage×8>       → stdlib, m31labs.dev/gosx, gosx-admin only
                        (blocks, content, flows, lifecycle{,/sqlstore},
                        media, render, store{,/file,/memory}; bottom tier,
                        beside core; zero imports of cms/studio or the root
                        facade)
shell                 → core, authoring, canvas, sitemap, panels, hostruntime,
                        cms/{lifecycle,media}, gosx-admin/blockstudio
studio (root)         → everything (deprecated facade only; no logic);
                        never imports cms/studio
cms/studio            → cms/{lifecycle,flows,style}, cms/studio/collab
                        (top tier, above root; see "vestigial facade edge"
                        below for the one deviation from the pre-fold-in plan)
```

Rule: no package may import a peer or anything above it in the DAG. `core`
imports nothing internal to the module. `shell` imports everything else
(it is the composition root — the portal shell that hosts mount). `backoffice`
and `shell` never import each other: `shell` composes editor chrome and
panels, `backoffice` renders the CRUD portal pages, and the one file that
composes both editor-and-CRUD concerns (`backend_editor_workbench.go`) lives
in `shell` because it is chrome composition, not a backend CRUD page.

**Hard rule, gated every release:** none of
`core`/`authoring`/`hostruntime`/`canvas`/`sitemap`/`panels`/`backoffice`/`shell`
may ever import `cms/studio` — that would close a cycle through the root
facade (`cms/studio → root → shell → cms/{lifecycle,media}` is a DAG path,
not a loop, only as long as nothing upstream of `cms/studio` imports it back).
Gate: `go list -deps ./core/... ./authoring/... ./hostruntime/... ./canvas/...
./sitemap/... ./panels/... ./backoffice/... ./shell/...` must contain zero
references to `m31labs.dev/gosx-studio/cms/studio`.

Two seams are designed on purpose, not accidents to clean up later:

- **`shell/cmsbridge.go`** is the *only* file in the module that imports
  `cms/{lifecycle,media}` (formerly the standalone `gosx-cms/{lifecycle,media}`)
  and `gosx-admin/blockstudio`. It holds the `Store` interface a host
  implements to feed Studio content, and the three converters
  (`shellBlockSummaries`, `shellMediaSummaries`, `shellRevisionSummaries`)
  from cms/admin types to Studio's pure
  `BlockSummary`/`MediaSummary`/`RevisionSummary` structs (which stay in
  `shell/options.go`, cms/admin-free). Before the fold-in, this was the one
  file that imported a separate `gosx-cms` module; the cutover slice rewrote
  its two imports in place (`gosx-cms/{lifecycle,media}` →
  `gosx-studio/cms/{lifecycle,media}`) and dropped the module-level `require`
  + `replace` — no other change to the file.
- **Consumer-side engine-runtime interfaces.** `canvas.StudioEngineRuntime`
  and `sitemap.SiteMapEngineRuntime` are implemented by
  `hostruntime.RuntimeConfig` through Go structural typing. `hostruntime`
  does not import either package — the interfaces stay owned by their
  consumers (`canvas`, `sitemap`), not centralized upward.
- **The `cms/studio` → root facade edge (vestigial).** The pre-fold-in plan
  assumed `cms/studio` imported the root `studio` facade
  (`gosxstudio "m31labs.dev/gosx-studio"`) and that this import had to be
  preserved byte-for-byte during the move, since rewriting its ~200
  `gosxstudio.`-prefixed references to subpackages was scoped as v0.7 work.
  Verification during the fold-in (`cms/PROVENANCE.md`, S3) found the
  premise was wrong: no `.go` file in the frozen `gosx-cms` source (or the
  copied `cms/studio` package) actually imports the bare
  `m31labs.dev/gosx-studio` package — `go list -deps ./cms/studio/...` shows
  its closure is only `cms/{lifecycle,flows,media,style}` +
  `cms/studio/collab` + external deps. The `m31labs.dev/gosx-studio` entry in
  the standalone `gosx-cms` module's `go.mod` was inert metadata (never
  referenced by any import), confirmed when `go mod tidy` dropped it cleanly
  from the tombstoned `v0.2.1` release with no code change required. There is
  therefore no v0.7 facade-rewrite debt for `cms/studio` to pay down; the
  `gosxstudio:`-prefixed strings visible in `cms/studio/assets_test.go` are
  DOM/event-name string literals asserted by the runtime bundle tests, not Go
  imports.

## Contracts

### Site Map Contract

Studio's core authoring model is a no-code site map.

Each `core.Page` maps a route to a top-level GoSX page component. Pages can
declare an editor-facing group such as Site, Store, Content, Flows, or
Utility so the visual site-map board can stay compact for non-technical
operators. Each page owns ordered `core.Component` entries that point at
concrete GoSX components supplied by the host app, CMS catalogs, Studio, or
plugins.

`Component.Source` names the broad owner of a component, while
`Component.Binding` is an opaque adapter key such as a CMS collection, flow,
section, or plugin artifact. Studio can show friendly labels and readiness
status without exposing handler refs, schemas, or Go DSL details to editors.

`Component.Controls` describes the no-code fields a component exposes. A
control can be text, media, choice, toggle, number, link, color, source,
flow, or Scene3D. The control binding remains opaque to Studio; the host
adapter decides how each value is loaded and persisted.

The site map is intentionally not a generic JSON page builder. It is a
compact visual graph over typed GoSX components. Editors manipulate pages,
sections, and component settings; host apps decide how those changes
persist.

`core.CompositionLibrary` describes what editors can add without seeing Go or
handler details. `PageBlueprint` entries are page starters such as a landing
page, collection page, content page, or flow page. `ComponentTemplate`
entries are palette cards such as hero, product grid, gallery, contact flow,
or showcase viewer. The `sitemap` package's engine renders those as a compact
authoring tray while retaining GoSX component names and opaque bindings for
the host adapter.

`core.CompositionIntent` describes the editor's draft operation in no-code
language while keeping enough typed data for the host to apply it through
`authoring`. Intent kinds cover creating a page from a blueprint and adding a
component template to a selected page. Steps let Studio render the operation
as a compact plan before persistence, with GoSX component names and opaque
bindings still available to host actions and engines.

### Canvas Preview Contract

`core.CanvasWorkspace` describes the live editing surface for a selected
page. Blocks point back to GoSX components and opaque host bindings; actions
describe the no-code commands an editor can use on the current selection.

`shell.CanvasSurface` — despite the name — is shell-chrome canvas-bar state
(the workbench's canvas toolbar/status), not the canvas engine. It stays in
`shell`.

`core.CanvasPreviewShell` names the `.gsx`-authored overlay affordances the
`canvas` package's engine can bind to: preview overlays, drop lines, dock
controls, outline templates, field hotspot templates, inline edit classes,
style-impact markers, viewport controls, and selection labels. Engines
should reuse this shell markup rather than constructing UI with ad hoc
browser scripts.

### Host Adapter Contract

Studio consumes host resources through `core.ResourceAdapter` declarations
instead of importing host packages. The default adapter set covers media,
pages, products, orders, contacts, settings, revisions, lifecycle, and
flows. Each adapter declares an editor-facing label, the Studio surface that
should use it, resource capabilities, and opaque `core.ResourceBinding` keys
such as `media.assets`, `pages.routes`, `products.collection`, or
`lifecycle.schedule`.

Adapters describe what Studio can ask for; host applications decide how
those requests load, validate, mutate, preview, publish, restore, or
execute. This keeps `cms/*`, `gosx-admin`, and app-specific repositories
behind a configuration boundary.

`shell.ShellConfig` is the reusable chrome contract for host-mounted Studio
surfaces. It normalizes editor-safe labels, modes, panels, host resource
links, action targets, permission flags, engine globals, feature flags, and
the canvas preview shell. Host apps can override this surface without owning
platform UI or shipping ad hoc browser scripts.

### Engine Contract

Heavy interactions belong behind `core.Engine` declarations:

- canvas editing
- site map graph editing
- block layout drag/drop
- flow authoring
- Scene3D and showcase viewers

The engine declaration names the mount id, surface, kind, and capabilities.
Host apps configure and mount engines; Studio owns the authoring controls
and interaction affordances.

### Site Map Canvas Contract

`SiteMap.CompositionWorkspace()` turns pages, components, and resources into
a graph. `CompositionWorkspace.CanvasLayout()` adds deterministic node
points and SVG path data for visual site-map engines, so host apps can
render Miro/Figma-like composition boards without inventing layout
contracts or exposing Go/handler details to editors.

### Runtime API Contract

`core.RuntimeContract` describes the browser APIs an engine exposes after it
is mounted. These contracts make the interactive surface explicit: host apps
can configure Studio and call named runtime methods, while editors keep
using `.gsx` components, islands, and engines instead of authored JavaScript
or Go DSL internals.

The default contracts include `GoSXStudioPreviewRuntime` for preview-shell
mounting, block visibility, text and attribute mirroring, theme application,
style-impact marking, custom CSS, font CSS, header-logo preview updates,
inline text requests, and field cycling — these methods are `.gsx`-authored
editor-side writers (`previewruntime/island_runtime.js`) that publish to a
`$preview.*` shared-signal namespace. A separate preview-side subscriber
bundle (`previewruntime/subscriber_runtime.js`, served at
`/_gosx/studio/preview-subscriber.js`) mounts on the storefront page when
loaded in the editor preview iframe (per the `gosx-preview=1` query param +
`island.EnablePreviewBootstrap()` opt-in). The cross-frame postMessage relay
(gosx `Bridge.EnableCrossFrameRelay`) routes signal writes between the
editor frame's WASM and the iframe's WASM.

`GoSXStudioWorkbenchRuntime` covers workbench chrome binding, mode
switching, viewport and zoom controls, style-state targeting, rail/focus/
activity toggles, rail resizing, width reads, and saved layout events;
`GoSXStudioSelectionRuntime` covers block selection, workspace target
selection, field focus, selection commandbar actions, and style-scope
readouts; `GoSXStudioFieldRuntime` covers field-to-preview mirroring and
copy controls; `GoSXStudioBrandRuntime` covers Brand inspector logo
placement, snapping, keyboard nudging, and preview handoff;
`GoSXStudioStyleRuntime` covers Look inspector theme controls, custom CSS,
font CSS, style recipes, reset-to-kit behavior, hover impact previews, and
impact readouts; `GoSXStudioBlockLayoutRuntime` covers row lookup,
selection, movement, drag/drop commits, visibility state, and
component-palette state. These globals are implementation points for GoSX
engines, not a separate no-code programming model for editors.

`hostruntime` embeds the reusable Studio runtime assets: the shared
stylesheet, the engine factory runtime, and the preview-shell runtime the
canvas engine mounts. Host apps mount the public `/_gosx/studio/*` URLs
through `hostruntime.MountRuntimes`; app-specific CMS/Admin scripts stay
outside this package unless their ownership is explicitly moved into Studio.

## Back-office surfaces

`backoffice` is a first-class product surface, not an afterthought: the
dashboard, per-domain CRUD pages (blog, products, pages, gallery,
categories, orders, contacts), the media library, settings, search, and
storefront preview. It was extracted from Muddy Noni's `app/admin/*`
back-office pages (2026-06-28/29) into reusable, host-agnostic `Render*`
functions in Slice 7 of the package restructure.

Hosts own the data adapters (loading and persisting resources); Studio owns
the rendered pages, forms, and table/detail scaffolding. `backoffice` imports
only `core` — it has no dependency on `shell`, `canvas`, `sitemap`, or
`panels`, and `shell` does not import `backoffice` either (see the DAG rule
above): the two compose independently, and a host wires them together at the
route level.

## Runtime & islands

The sixteen `*runtime` packages (`fieldruntime`, `selectionruntime`,
`workbenchruntime`, `styleruntime`, `previewruntime`, `blocklayoutruntime`,
`brandruntime`, `inspectorruntime`, `sitemapruntime`, `authoringruntime`,
`inlineeditruntime`, and the four `canvas*runtime` packages) are each a
go:embed browser-runtime bundle: one package per engine's client-side
JS/WASM surface, mounted by `hostruntime` at the public `/_gosx/studio/*`
paths. These packages are intentionally untouched by the package
restructure — they were already correctly isolated before Slices 1-9 moved
the flat root into `core`/`authoring`/`canvas`/`sitemap`/`panels`/
`backoffice`/`shell`/`hostruntime`.

`parity_e2e/` is the Playwright harness that proves rendered output is
byte-identical across the restructure and exercises the reference apps
(Muddy Noni, Pajaritos) end to end; see `parity_e2e/README.md` for how to run
it. The `data-studio-*`/`data-gosx-studio-*` attributes it asserts on are
themselves a compatibility surface: they are the rendered-page contract
between Studio's Go/`.gsx` output and both the browser runtimes above and the
Playwright harness, and no package move may rename them.

**`cms/studio/assets` is a deliberate fork of `hostruntime/assets`, not a
dedup target.** `cms/studio/assets/{workbench_runtime,command_palette,
state_runtime}.js` share filenames with `hostruntime/assets/*.js` and open
with near-identical bytes, because both trace back to the same original
runtime code before the standalone `gosx-cms` module and this module's
`hostruntime` diverged. The fold-in moved `cms/studio/assets/` as-is
(`assets.go`'s `go:embed` is package-directory-relative, so the files had to
land beside their own package) and made **no attempt to reconcile or dedup**
the two runtimes against each other — that risked silently changing rendered
bytes for either `studio`'s or `cms/studio`'s consumers mid-fold-in.
Canonicalizing the two into one shared runtime (or proving they've fully
diverged and should stay separate) is explicitly deferred to a v0.7 decision;
until then, treat `cms/studio/assets/*.js` and `hostruntime/assets/*.js` as
two independently-versioned bundles that happen to share names.

## Release model

The root `studio` package is a deprecated compatibility facade (type aliases
+ forwarding wrappers, no logic) for one release cycle, v0.6.x. Every symbol
muddy-noni-commerce, pajaritos-forest-school, and `cms/studio` (formerly
`gosx-cms/studio`) used before the restructure still resolves through the
facade; each symbol is marked `// Deprecated: use <pkg>.<Name>.`.
`facade_completeness_test.go` gates the facade in gosx-studio's own CI: it
references every exported `compat_*.go` symbol, so an accidental rename or
deletion is a local compile failure instead of a silent break only caught
when a consumer repo happens to be built.

**The `gosx-studio`↔`gosx-cms` module cycle described in earlier drafts of
this document is dissolved, not just planned around.** It used to be real:
`gosx-cms/go.mod` required `m31labs.dev/gosx-studio`, and
`gosx-studio/go.mod` required `m31labs.dev/gosx-cms` (for the `lifecycle` and
`media` packages `shell/cmsbridge.go` touches), forcing a two-step lockstep
tagging dance on every studio release. The fold-in (full record in
[`cms/PROVENANCE.md`](../cms/PROVENANCE.md)) closed it: all 13 `gosx-cms`
packages now live under `cms/` in this module; `shell/cmsbridge.go` and
`shell/options.go` were rewritten in place to import
`gosx-studio/cms/{lifecycle,media}`; and this module's `go.mod` dropped the
`gosx-cms` `require` + `replace` entirely. There is no cycle left to lockstep
around.

Two decisions changed between the original Phase-2 sketch and execution:

- **No `gosx-cms` v0.3.0 alias-facade release.** The sketch's runner-up —
  publish `gosx-cms` as a type-alias facade over `gosx-studio/cms/*` — was
  rejected: maintaining a second ~700-symbol facade across 13 packages for
  exactly two in-house consumers (muddy-noni-commerce,
  pajaritos-forest-school) wasn't worth it. Instead the standalone
  `gosx-cms` repository is **frozen and tombstoned**: README replaced with a
  fold-in notice + import-path mapping table, a `// Deprecated:` comment
  added above its `module` line, final tag `v0.2.1` cut, and all package code
  left intact so existing `v0.2.x` pins keep resolving unchanged. Consumers
  migrate by a mechanical prefix sed (`m31labs.dev/gosx-cms/` →
  `m31labs.dev/gosx-studio/cms/`) directly against the folded paths, not
  through any facade.
- **`gosx-cms/studio` did not merge into `shell`/`panels`/`backoffice`.** It
  moved to `cms/studio` (+ `cms/studio/collab`) as its own package instead —
  see the Package DAG section above. The merge floated in the original sketch
  was rejected before execution: both sides have colliding names
  (`style_panel.go`, overlapping workbench/panel symbols), the merge wasn't
  mechanical, and the two consumers already imported `gosx-cms/studio`
  directly — keeping `cms/studio` as a preserved path let their migration
  stay a pure prefix sed too. Dissolving `cms/studio` into the root packages
  remains an option for a future release, not a commitment made here.

The fold-in targets `gosx-studio` v0.6.0 (this module's `go.mod` also picked
up `modernc.org/sqlite` and bumped `gosx-admin` to v0.2.0 for the
`cms/lifecycle/sqlstore` and `cms/studio` dependencies the folded code needs);
tagging and the proxy-resolution verification are a separate release slice.
`gosx-admin`'s remit is unaffected either way: it stays a standalone toolkit
whose only Studio-relevant export is `blockstudio`.

## Noni proving ground

Noni's Mud Relics (muddy-noni-commerce) remains a reference implementation,
alongside Pajaritos. Both proving-ground apps use operator-facing language
such as "Website editor"; the reusable package keeps internal names such as
GoSX Studio for package boundaries, contracts, and docs. Client-facing copy
in either app must never mention GoSX Studio by name (`reference-apps.yml`'s
"Guard client copy against internal platform wording" step enforces this).
