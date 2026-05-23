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
- Use GoSX engines for heavy interactions: canvas, graph editing, drag/drop, inline preview overlays, 3D viewers, and timeline-style tools.
- Use GoSX islands for smaller reactive panels: inspectors, publish readiness, filters, pickers, and local field validation.
- Keep visible product language plain: "Website editor", "Home", "Look", "Brand", "Publish", and "Advanced".

## Package Boundary

This module starts with contracts and plugin placement. The first extraction target is the Studio surface now being proven in `muddy-noni-commerce`.

The eventual combined product should consume:

```text
gosx-cms + gosx-admin + gosx-studio
```

That combined product should not live inside Noni's site. Noni's site should remain a high-signal reference implementation and proving ground.
