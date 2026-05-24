# Architecture

GoSX Studio is the reusable website authoring layer that sits beside `gosx-cms` and `gosx-admin`.

## Library Roles

`gosx-cms` owns content management:

- content schemas
- media records
- lifecycle state
- content storage
- generated assets and their provenance

`gosx-admin` owns operator back-office surfaces:

- CRUD/resource management
- dashboards
- internal workflows
- resource actions
- operational reporting

`gosx-studio` owns website customization and authoring:

- page canvas
- visual inspectors
- no-code configuration controls
- section and block layout tools
- flow authoring
- publish readiness
- showcase plugins

## Composition Model

Host applications configure Studio. Editors use Studio.

The host application supplies:

- content adapters
- permission adapters
- server actions
- route bindings
- product copy
- feature flags
- design tokens

Studio supplies:

- `.gsx` surfaces for visible editor UI
- GoSX engines for heavy client-side interactions
- GoSX islands for focused reactive controls
- extension points for plugins
- common authoring language for non-technical operators

## Site Map Contract

Studio's core authoring model is a no-code site map.

Each `Page` maps a route to a top-level GoSX page component. Pages can declare an editor-facing group such as Site, Store, Content, Flows, or Utility so the visual site-map board can stay compact for non-technical operators. Each page owns ordered `Component` entries that point at concrete GoSX components supplied by the host app, CMS catalogs, Studio, or plugins.

`Component.Source` names the broad owner of a component, while `Component.Binding` is an opaque adapter key such as a CMS collection, flow, section, or plugin artifact. Studio can show friendly labels and readiness status without exposing handler refs, schemas, or Go DSL details to editors.

`Component.Controls` describes the no-code fields a component exposes. A control can be text, media, choice, toggle, number, link, color, source, flow, or Scene3D. The control binding remains opaque to Studio; the host adapter decides how each value is loaded and persisted.

The site map is intentionally not a generic JSON page builder. It is a compact visual graph over typed GoSX components. Editors manipulate pages, sections, and component settings; host apps decide how those changes persist.

`CompositionLibrary` describes what editors can add without seeing Go or handler details. `PageBlueprint` entries are page starters such as a landing page, collection page, content page, or flow page. `ComponentTemplate` entries are palette cards such as hero, product grid, gallery, contact flow, or showcase viewer. The site-map engine can render those as a compact authoring tray while retaining GoSX component names and opaque bindings for the host adapter.

`CompositionIntent` describes the editor's draft operation in no-code language while keeping enough typed data for the host to apply it. Intent kinds cover creating a page from a blueprint and adding a component template to a selected page. Steps let Studio render the operation as a compact plan before persistence, with GoSX component names and opaque bindings still available to host actions and engines.

## Canvas Preview Contract

`CanvasWorkspace` describes the live editing surface for a selected page. Blocks point back to GoSX components and opaque host bindings; actions describe the no-code commands an editor can use on the current selection.

`CanvasPreviewShell` names the `.gsx`-authored overlay affordances the canvas engine can bind to: preview overlays, drop lines, dock controls, outline templates, field hotspot templates, inline edit classes, style-impact markers, viewport controls, and selection labels. Engines should reuse this shell markup rather than constructing UI with ad hoc browser scripts.

## Host Adapter Contract

Studio consumes host resources through `ResourceAdapter` declarations instead of importing host packages. The default adapter set covers media, pages, products, orders, contacts, settings, revisions, lifecycle, and flows. Each adapter declares an editor-facing label, the Studio surface that should use it, resource capabilities, and opaque `ResourceBinding` keys such as `media.assets`, `pages.routes`, `products.collection`, or `lifecycle.schedule`.

Adapters describe what Studio can ask for; host applications decide how those requests load, validate, mutate, preview, publish, restore, or execute. This keeps `gosx-cms`, `gosx-admin`, and app-specific repositories behind a configuration boundary.

## Engine Contract

Heavy interactions belong behind `Engine` declarations:

- canvas editing
- site map graph editing
- block layout drag/drop
- flow authoring
- Scene3D and showcase viewers

The engine declaration names the mount id, surface, kind, and capabilities. Host apps configure and mount engines; Studio owns the authoring controls and interaction affordances.

## Runtime API Contract

`RuntimeContract` describes the browser APIs an engine exposes after it is mounted. These contracts make the interactive surface explicit: host apps can configure Studio and call named runtime methods, while editors keep using `.gsx` components, islands, and engines instead of authored JavaScript or Go DSL internals.

The default contracts currently include `GoSXStudioPreviewRuntime` for preview-shell mounting, block visibility, text and attribute mirroring, theme application, style-impact marking, custom CSS, font CSS, header-logo preview updates, inline text requests, and field cycling; `GoSXStudioWorkbenchRuntime` for workbench chrome binding, mode switching, viewport and zoom controls, style-state targeting, rail/focus/activity toggles, rail resizing, width reads, and saved layout events; `GoSXStudioBrandRuntime` for Brand inspector logo placement, snapping, keyboard nudging, and preview handoff; `GoSXStudioStyleRuntime` for Look inspector theme controls, custom CSS, font CSS, style recipes, reset-to-kit behavior, hover impact previews, and impact readouts; and `GoSXStudioBlockLayoutRuntime` for row lookup, selection, movement, drag/drop commits, visibility state, and component-palette state. These globals are implementation points for GoSX engines, not a separate no-code programming model for editors.

## Noni Proving Ground

Noni's Mud Relics remains the reference implementation while Studio is extracted. The proving-ground app should keep using operator-facing language such as "Website editor". The reusable package can keep internal names such as GoSX Studio for package boundaries, contracts, and docs.

The extraction should move from concrete Noni components toward stable Studio contracts without flattening CMS, Admin, and Studio into one package.
