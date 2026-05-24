# Roadmap

## 1. Extraction Boundary

- Define the Studio host configuration API.
- Stabilize the site map contract for pages composed from GoSX components, source ownership, opaque adapter bindings, and publish-readiness status.
- Stabilize the engine declaration contract for canvas, site map, block layout, flow, and Scene3D surfaces.
- Define content, media, page, section, theme, and publish adapter interfaces.
- Move reusable editor contracts out of `muddy-noni-commerce`.
- Keep Noni-specific copy, resource names, and routes in the Noni app.

## 2. Studio Surfaces

Extract the current editor into reusable `.gsx` surfaces:

- `StudioShell.gsx`
- `SiteNavigator.gsx`
- `SiteMapEngine.gsx`
- `CanvasEngine.gsx`
- `BlockLayoutEngine.gsx`
- `InspectorIsland.gsx`
- `FlowDesigner.gsx`
- `PublishPanel.gsx`

Heavy client-side behavior belongs in GoSX engines. Small stateful panels belong in islands. Host apps should configure Studio rather than shipping authored JavaScript.

## 3. Plugin System

- Add a plugin registry for Studio features.
- Let CMS plugins expose generated content assets and metadata.
- Let Studio plugins expose authoring controls and placement UI.
- Let GoSX engines own high-interaction public or editor experiences.

The first proposed plugin is Showcase 3D: CMS stores source photos and generated model metadata, Studio places and previews the model, and Scene3D renders inline and pop-out viewers.

## 4. Productization

Create a product package that composes:

```text
gosx-cms + gosx-admin + gosx-studio
```

That product can ship the full no-code web platform. Individual libraries should remain independently useful and testable.
