# GoSX Studio

GoSX Studio is the web-editing admin portal for no-code, user-authored
websites. It is the entire surface a site owner lives in day to day: canvas
design, content, commerce, media, and settings back-office, all in one
GoSX-rendered shell. Think Webflow-shaped — host-configured, GoSX-rendered,
one editor, one back-office, one portal.

Reference deployments: **Muddy Noni** (commerce) and **Pajaritos** (school
site).

Earlier docs described Studio as "the authoring layer, intentionally separate
from `gosx-cms` and `gosx-admin`." That framing is retired — the code moved
past it. Studio is the portal; `gosx-admin` is a generic back-office toolkit
dependency (Studio consumes only `gosx-admin/blockstudio`); `gosx-cms` is
content storage that a Phase 2 fold-in is expected to bring inside this module
(see [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) §"Release model").

## What hosts provide vs. what Studio provides

Hosts supply adapters, persistence, permissions, routes, and copy. Studio
supplies contracts, chrome, panels, canvas engines, back-office pages, and
runtime islands.

Concretely, the host application supplies:

- content adapters
- shell labels, modes, panels, and resource links
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

## Package map

The module is a strict import DAG: `core` sits at the bottom, `shell`
composes everything above it, and peers never import each other. See
[docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) for the full diagram.

| Package | Responsibility |
|---|---|
| `core` | Pure data contracts and defaults: the site-map/canvas/composition/flow/engine/resource type system, zero UI or host imports. |
| `authoring` | The typed server-action mutation boundary: operations, form codec, adapter interface, action handler, authoring surface assembly, and the style/appearance/section-field mutation family. |
| `hostruntime` | Embedded runtime assets, bundle concatenation, and the public `/_gosx/studio/*` paths and HTTP mounting. |
| `canvas` | Page-canvas engine hosting and server-rendered surface markup (artboards, thumbnails, block-layout DOM contract). |
| `sitemap` | The visual site-map board: graph engine render, board/view projections, authoring panels/forms, site navigator. |
| `panels` | Editor inspector/designer panels and their `.gsx` islands (right/left rail content units). |
| `backoffice` | The CRUD portal surfaces — dashboard, per-domain index/detail pages, media library, settings, search, storefront preview. First-class per the portal definition. |
| `shell` | The portal shell: host-facing `ShellConfig`/`Shell`/`Store` contracts, readiness rail, workbench chrome renderers, and the editor workbench composition root. |
| `*runtime` islands (`fieldruntime`, `selectionruntime`, `workbenchruntime`, `styleruntime`, `previewruntime`, `blocklayoutruntime`, `brandruntime`, `inspectorruntime`, `sitemapruntime`, `authoringruntime`, `inlineeditruntime`, `canvas*runtime`) | The go:embed browser-runtime bundles the shell/canvas/sitemap/panels packages mount — one package per engine's client-side JS/WASM surface. |
| `plugins/showcase3d` | The CMS/Studio contract for source photos, generated model artifacts, provenance, moderation, lifecycle readiness, no-code placement controls, and Scene3D viewer descriptors. |
| `studio` (root) | Deprecated compatibility facade — type aliases and forwarding wrappers only, no logic. See "Versioning & compatibility" below. |

## Quick start

1. Mount the embedded runtime assets and stylesheet: `hostruntime.MountRuntimes` /
   `hostruntime.DefaultRuntimeConfig`.
2. Configure the portal shell: `shell.ShellConfig` / `shell.DefaultShellConfig`
   (labels, modes, panels, resource links, engine globals, feature flags).
3. Wire the mutation boundary: `authoring.AuthoringActionHandler` against your
   `authoring.AuthoringAdapter` implementation.
4. Render the editor route with `shell.RenderBackendEditorPage`, and the CRUD
   routes with the `backoffice` package's per-domain `Render*` pages.

The canonical host adapter is muddy-noni-commerce's
`internal/studiohost/adapter.go`, which builds a `hostruntime.RuntimeConfig`
and `shell.ShellConfig` (host labels, resource links, panels, actions) end to
end.

## Versioning & compatibility

The root `studio` package is a **deprecated compatibility facade** for one
release cycle (v0.6.x): every symbol muddy, pajaritos, and `gosx-cms/studio`
used before the package restructure still resolves through a type alias or
thin forwarding wrapper, each marked `// Deprecated:`. New code should import
the subpackages above directly (`core`, `authoring`, `shell`, `canvas`,
`sitemap`, `panels`, `backoffice`, `hostruntime`) rather than the root.

A `gosx-cms` module fold-in is planned for Phase 2, after this facade window
closes — see [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) §"Release model"
for the plan and current status.

See `docs/WEBFLOW_CLASS_EDITOR_GAP_INVENTORY.md` for the detailed gap
inventory and work plan toward a Webflow-class no-code editor.
