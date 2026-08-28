# Modern Editor Turnover Spec

Date: 2026-08-27

This spec is the acceptance and visual contract for the modern editor turnover
slice. The goal is not a broad Studio redesign. The goal is real drag/drop and
best-in-class authoring behavior inside the existing GoSX Studio Paper & Ink
operational editor, with Muddy Noni consuming host adapters, data, and actions
without forking shared editor primitives.

## Non-Goals

- Do not claim or implement arbitrary freeform page layout.
- Do not claim AI authoring, generative layout, or autonomous design changes.
- Do not claim live checkout or storefront commerce edits beyond the editor
  surfaces actually backed by host actions.
- Do not replace the existing Studio aesthetic territory, shell, or runtime
  architecture as part of this turnover.

## Visual System

The Studio editor UI preserves the current Paper & Ink operational-tool
territory from `hostruntime/assets/studio.css`: warm paper canvas, restrained
ink text, terracotta affordances, compact controls, visible structure, and a
dense authoring cadence. It should feel like a precise shared editing tool, not
a marketing page or visual refresh.

- Territory: Paper & Ink.
- Display font: Fraunces, weight 700, for compact panel headings and editor
  section titles.
- Body font: Space Grotesk, weights 400, 600, and 700, for tool labels, field
  labels, body copy, and buttons.
- Mono font: JetBrains Mono, weight 500, for status chips, counters, runtime
  labels, and technical metadata.
- Type scale: compact minor-third labels from `--text-xs` through `--text-4xl`;
  use smaller steps inside panels, rails, cards, toolbars, and form controls.
- Dominant color: `--color-canvas` `#f6f0e8`.
- Secondary colors: `--color-canvas-soft` `#fdfaf6` and `--color-panel`
  `#fffaf2`.
- Text hierarchy against `--color-canvas`: `--color-ink` `#2a241e` is 13.54:1
  and AAA; `--color-ink-soft` `#4c4036` is 8.86:1 and AAA; `--color-muted`
  `#76685c` is 4.75:1 and AA.
- Accent: `--color-accent` `#b86a4a` is 3.57:1 against `--color-canvas`, and
  white text on this accent is 4.04:1. Use it for borders, focus rings, drop
  indicators, selection outlines, icon-only marks, and large text only. Small
  action text on a warm canvas must use `--color-accent-strong` `#965036`
  because it reaches 5.30:1 against `--color-canvas`.
- Support colors: `--color-moss`, `--color-sun`, `--color-stone`,
  `--color-success`, and `--color-danger` remain for readiness, lifecycle,
  validation, and destructive states.
- Motion: Subtle. Use 180ms for hover/focus feedback, 260ms for panel and
  selection state, 520ms for drawer/workspace changes, `--ease-out` for
  entering/revealing/settling, and `--ease-spring` only for bounded tactile
  feedback such as button press or drop confirmation.
- Reduced motion: all implementation must provide a `prefers-reduced-motion:
  reduce` override that disables nonessential transitions, animated scrolling,
  and transform-based feedback while preserving visible state changes.
- Spacing: 8px-rooted responsive scale from `--space-xs` through
  `--space-3xl`. Tool surfaces should prefer `--space-xs` through
  `--space-md`; page-level bands can use `--space-lg` and above.
- Radius: cards and tool panels stay at 8px through `--radius-card`; pill
  controls use `--radius-pill`.

Paste-ready token block, copied from `hostruntime/assets/studio.css`:

```css
:root,
.gosx-studio {
  --font-display: "Fraunces", Georgia, serif;
  --font-body: "Space Grotesk", "Avenir Next", sans-serif;
  --font-mono: "JetBrains Mono", monospace;
  --text-xs: clamp(0.75rem, 0.72rem + 0.12vw, 0.82rem);
  --text-sm: clamp(0.88rem, 0.84rem + 0.18vw, 0.98rem);
  --text-md: clamp(1rem, 0.96rem + 0.2vw, 1.12rem);
  --text-lg: clamp(1.25rem, 1.16rem + 0.42vw, 1.5rem);
  --text-xl: clamp(1.56rem, 1.39rem + 0.8vw, 2rem);
  --text-2xl: clamp(1.95rem, 1.68rem + 1.2vw, 2.7rem);
  --text-3xl: clamp(2.44rem, 1.98rem + 2vw, 3.6rem);
  --text-4xl: clamp(3.05rem, 2.35rem + 3vw, 4.8rem);
  --color-canvas: #f6f0e8;
  --color-canvas-soft: #fdfaf6;
  --color-panel: #fffaf2;
  --color-ink: #2a241e;
  --color-ink-soft: #4c4036;
  --color-muted: #76685c;
  --color-line: rgba(42, 36, 30, 0.16);
  --color-line-strong: rgba(42, 36, 30, 0.28);
  --color-accent: #b86a4a;
  --color-accent-strong: #965036;
  --color-moss: #2f5143;
  --color-sun: #e5b36a;
  --color-stone: #cbb39a;
  --color-success: #2f6d4f;
  --color-danger: #9f3d2f;
  --space-xs: clamp(0.75rem, 0.7rem + 0.25vw, 0.9rem);
  --space-sm: clamp(1rem, 0.92rem + 0.35vw, 1.2rem);
  --space-md: clamp(1.5rem, 1.32rem + 0.8vw, 2rem);
  --space-lg: clamp(2rem, 1.7rem + 1.3vw, 2.8rem);
  --space-xl: clamp(3rem, 2.45rem + 2.4vw, 4.4rem);
  --space-2xl: clamp(4rem, 3.15rem + 3.8vw, 6.2rem);
  --space-3xl: clamp(6rem, 4.7rem + 5.8vw, 9.5rem);
  --radius-card: 8px;
  --radius-pill: 999px;
  --duration-fast: 180ms;
  --duration-base: 260ms;
  --duration-slow: 520ms;
  --ease-out: cubic-bezier(0.16, 1, 0.3, 1);
  --ease-spring: cubic-bezier(0.34, 1.56, 0.64, 1);
}
```

## Architecture Contract

- Reusable editor logic, runtime assets, visual tokens, keyboard contracts,
  block identity rules, drag/drop mechanics, and shared UI primitives live in
  GoSX Studio.
- Muddy Noni owns only host adapters, data loading, authorization, validation,
  and server actions. Noni-specific copy, product nouns, routes, and persistence
  details must not leak into shared Studio primitives.
- The current Page CMS runtime gap is explicit: behavior currently served from
  Noni as `public/content-editor.js` must move into shared Studio when it is
  reusable. Noni should consume the canonical Studio endpoint afterward. Preserve
  the old route as a compatibility alias only if it is needed for rollout.
- Shared Studio packages must respect the import DAG in `docs/ARCHITECTURE.md`:
  host adapters configure Studio; Studio does not import host application code.
- Browser runtimes remain implementation points for GoSX engines and `.gsx`
  chrome. They are not a separate no-code programming model for editors.

## Interaction Acceptance

The editor is acceptable only when each item below is true in both shared Studio
and the Noni-mounted surface that consumes it.

- Drag/drop is real: every movable block has a visible drag handle, before/after
  drop indicators are visible while hovering valid targets, and only a valid
  drop commits a reorder.
- Cancel paths are non-mutating: Escape, pointer cancel, drag leaving the valid
  surface, invalid target release, and explicit cancel leave the block order and
  persisted draft unchanged.
- Inputs are practical across modalities: mouse and pointer/touch drag are
  supported where browser/platform behavior permits; keyboard move controls are
  always available.
- Dragging is never the only pointer path. The same reorder operation must also
  be available through click/tap controls because keyboard-only alternatives do
  not satisfy the pointer accessibility need described by WCAG 2.2 Dragging
  Movements: https://www.w3.org/WAI/WCAG22/Understanding/dragging-movements.html.
- Pointer targets for drag handles, move controls, insert controls, destructive
  actions, and inspector controls must be at least 24 by 24 CSS pixels, with
  44 by 44 CSS pixels recommended for touch-heavy controls. This follows WCAG
  2.2 Target Size (Minimum): https://www.w3.org/WAI/WCAG22/Understanding/target-size-minimum.html.
  This is a scoped interaction contract, not a claim of full WCAG compliance.
- Block identity and order are stable: reordering preserves block IDs,
  per-block data bindings, current selection where sensible, and persisted order
  after reload.
- Selection is meaningful: selected content is visibly distinct, has focusable
  controls, and drives a context inspector that reflects the selected block.
- Authoring commands exist and work: insert, duplicate, delete, and search
  blocks are reachable without knowing code names.
- Undo/redo is per editor instance. Shortcuts are scoped so typing, text
  selection, native clipboard, and browser editing behaviors are not hijacked.
- Drafts remain nonpublic until publish. Save, publish, and preview states must
  distinguish draft, public, dirty, saving, saved, failed, offline, and conflict
  states truthfully.
- Unsaved changes have a leave guard for navigation, reload, and editor route
  changes.
- Errors are visible and recoverable. Failed saves do not erase local edits, and
  retry paths are obvious.
- Responsive preview is usable at desktop, laptop, and mobile widths. Essential
  controls remain reachable on mobile without horizontal overflow.
- Keyboard focus order is coherent. Focus rings are visible and use the visual
  token system.
- Reduced motion users see the same committed state changes without animated
  movement as the only signal.

## Layout Fit Gates

- Desktop: 1600x900 must show the editor shell, selected content, context
  inspector, save status, and primary authoring controls without overlap.
- Laptop: 1280x800 must preserve the same essential workflow with panels and
  controls fitting without clipped text or unusable horizontal scrolling.
- Mobile: 390x844 must have no page-level horizontal overflow; essential
  controls for selecting, moving, saving, and previewing blocks must remain
  reachable.

## QA Matrix

Every user-facing claim needs proof across these dimensions:

| Claim | Required user input | Persistence/reload | Draft/public proof | Screenshot state |
|---|---|---|---|---|
| Valid drag reorder commits | Mouse drag, pointer/touch where practical, keyboard move | Reload shows new order and same block IDs | Public storefront unchanged before publish | Before drag, drop indicator, after drop |
| Canceled drag does not mutate | Escape, outside release, invalid target release | Reload shows original order | No draft or public mutation from cancel | Drag active and canceled/end state |
| Editable text remains safe | Type text, select text, copy/paste, undo/redo inside field | Reload after save preserves text | Public unchanged until publish | Focused editable field and saved status |
| Insert/duplicate/delete/search work | Keyboard and pointer activation | Reload reflects saved draft changes | Publish boundary verified separately | Search results, inserted block, duplicated block, delete confirmation |
| Offline or failed save is truthful | Simulated offline/network failure, retry | Failed save does not overwrite previous persisted state | Public remains last published version | Offline/failed status and recovered saved state |
| Inspector tracks selection | Select multiple block types | Reload preserves selected order/data, not stale inspector data | Inspector edits remain draft-only until publish | Selected block plus matching inspector |
| Responsive preview fits | Resize to 1600x900, 1280x800, 390x844 | Reload each viewport with same draft | Public preview checked separately | One screenshot per viewport |

All browser tests must use isolated temporary data. Do not mutate real customer
content, staging production-like records, DNS, secrets, payment systems, or live
commerce state while proving this slice. The interactive `js_repl` browser skill
is unavailable for this run; QA uses the current isolated Playwright harness and
screenshot review without changing Codex configuration.

## Release Gate

Release is blocked until all of the following are complete:

- Normal Go tests pass for the touched packages.
- Focused Playwright coverage proves the drag/drop, selection, persistence,
  draft/public boundary, save failure, keyboard, reduced-motion, and responsive
  gates listed above.
- Production build succeeds.
- Final adversarial review signs off that shared Studio owns reusable editor
  behavior and Noni owns adapters/data/actions only.
- Noni module pin is updated to the verified Studio revision.
- Existing guarded staging backup/deploy procedure runs only after the local
  gates pass. Production, DNS, secrets, and real customer data remain off-limits
  for this turnover unless explicitly authorized later.
