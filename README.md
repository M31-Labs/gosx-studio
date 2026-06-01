# GoSX Studio

GoSX Studio is the website customization and authoring layer for GoSX sites.

It is intentionally separate from:

- `gosx-cms`: content management, content schemas, lifecycle, media, and content storage.
- `gosx-admin`: back-office operator tools, resource CRUD, dashboards, and Retool-style internal surfaces.
- `gosx-studio`: site canvas, inspectors, no-code authoring, theme controls, flow editing, publish tools, and showcase plugins.

Noni's Mud Relics is the proving ground for the combined product: CMS plus Admin plus Studio. The site is already proving the CMS and Admin sides. The Studio work should move here so the authoring layer becomes reusable instead of remaining embedded in one ecommerce app.

## Initial Scope

- Extract the current Noni editor surface into reusable `.gsx` Studio components.
- Keep host apps responsible for adapters, resources, permissions, actions, data, and labels.
- Represent every authored site as an editable site map: pages compose GoSX components, and those components expose no-code controls.
- Use GoSX engines for heavy interactions: canvas, graph editing, drag/drop, inline preview overlays, 3D viewers, and timeline-style tools.
- Use GoSX islands for smaller reactive panels: inspectors, publish readiness, filters, pickers, and local field validation.
- Keep visible product language plain: "Website editor", "Home", "Look", "Brand", "Publish", and "Advanced".

## Package Boundary

This module starts with contracts and plugin placement. The first extraction target is the Studio surface now being proven in `muddy-noni-commerce`.

The first public contracts are intentionally small:

- `SiteMap`, `Page`, and `Component` describe no-code page composition in terms of GoSX components, editor-facing page groups, broad source kinds, opaque host/CMS bindings, and readiness status.
- `CanvasWorkspace`, `CanvasBlock`, `CanvasViewport`, `CanvasZoomLevel`, `CanvasAction`, and `CanvasPreviewShell` describe the live page-canvas surface: preview breakpoints, zoom choices, selectable GoSX-backed blocks, no-code controls, dock actions, GoSX-authored overlay templates, inline editing affordances, and style-impact markers.
- `Control` and `ControlOption` describe the editor-facing fields each component exposes: text, media, choices, links, color, source bindings, flows, and Scene3D/model controls.
- `CompositionLibrary`, `PageBlueprint`, and `ComponentTemplate` describe the editor-facing building blocks that a site-map engine can expose as page starters and a component palette.
- `CompositionIntent` and `CompositionStep` describe draft no-code operations such as creating a page from a blueprint or adding a component template to the selected route.
- `AuthoringSurface` and `NoCodeAuthoringSurface` assemble the site map, selected page, palette, draft intents, workspace graph, and canvas layout into the host-facing no-code platform model exposed by `HostShell`.
- `AuthoringMutation`, `AuthoringAdapter`, and `AuthoringActionHandler` describe the typed server-action boundary for applying no-code edits while host apps retain persistence, validation, preview refresh, and publish ownership.
- `ResourceAdapter` and `ResourceBinding` describe host-owned media, pages, products, orders, contacts, settings, revisions, lifecycle, and flow resources without importing host internals.
- `Engine` declares heavy interaction surfaces and the capabilities a host app can mount.
- `RuntimeContract`, `RuntimeMethod`, and `RuntimePayloadField` declare the browser runtime APIs that engines expose, including the preview, workbench, brand, style, and block-layout globals used by canvas editing.
- `HostConfig` ties product labels, features, engines, and the editable site map together.
- `ShellConfig` describes the reusable Studio chrome a host app configures: operator labels, modes, panels, resource links, engine globals, server actions, permissions, feature flags, and canvas preview shell.
- Runtime assets and handlers serve the shared Studio stylesheet plus the combined preview and engine runtime at the public `/_gosx/studio/*` paths. Host apps can mount these assets while keeping their own CMS/Admin scripts behind separate adapters.
- `plugins/showcase3d` defines the CMS/Studio contract for source photos, generated model artifacts, provenance, moderation, lifecycle readiness, no-code placement controls, and Scene3D viewer descriptors.

The eventual combined product should consume:

```text
gosx-cms + gosx-admin + gosx-studio
```

That combined product should not live inside Noni's site. Noni's site should remain a high-signal reference implementation and proving ground.

See `docs/WEBFLOW_CLASS_EDITOR_GAP_INVENTORY.md` for the detailed gap inventory and work plan toward a Webflow-class no-code editor.
