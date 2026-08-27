# Enterprise Editor Polish Spec

Date: 2026-08-27

Status: local component-quality and STATUS11 visual gates complete; SAVE07 is
frozen, while the Gate10 rerun and real Noni integration remain pending. This
is not certification, staging validation, or a deployment claim.

This slice carries the visual and interaction quality contract for the shared
Studio editor surfaces. It preserves the existing Paper & Ink territory and
proves small, reusable CSS corrections against the actual Page CMS detail
renderer markup and the existing content/media browser runtimes.

## Baseline and ownership

- Studio baseline: clean `35f0ef2`.
- Noni baseline: `0ff2f9b`, exact-pinned to Studio `35f0ef2`.
- Existing staging: `3eb` digest unchanged; resolved user emails unchanged.
- Quality owner: `POLISH-QUALITY-04`, Luna max, local-only execution.
- Content interaction ownership remains with `POLISH-CONTENT-01` (34/34 local;
  click/focus, filtering, history, and editor behavior).
- Media interaction ownership remains with `POLISH-MEDIA-02` (11/11 local;
  gallery focus, raw-toggle, search, and announcements).
- Save-state runtime ownership is `POLISH-SAVE-07`, implemented by
  `luna_polish_content`; its marked-revision, pending-lock, native-form,
  validation-focus, and conflict behavior is covered by the content runtime
  and revision contract tests. `POLISH-SAVE-03A`, owned by
  `luna_polish_save_audit`, remains the separate readonly three-case
  adversarial regression audit.
- Runtime asset cache revalidation is tracked separately as `POLISH-ASSET-05`;
  its changed-ETag/200 and unchanged-ETag/304 checks are release-reliability
  evidence only. Focused/race/vet plus IMS-only and weak-If-Range follow-up
  checks are locally complete; there is no deployed claim.
- Page CMS stale-tab compare-and-swap is tracked separately as `POLISH-CAS-06`.
  It is distinct from existing Home collaboration CAS and remains fail-closed
  for stale revisions. All Noni normal/race checks passed through the isolated
  development overlay; exact-pin verification is still pending.

Adjacent packet status:

- `POLISH-ASSET-05`: host/runtime asset cache reliability; changed ETag returns
  200, unchanged ETag returns 304, and HEAD/no-cache, IMS-only, and weak
  If-Range behavior are locally verified. Status: local complete release-
  reliability gate; no deployed claim.
- `POLISH-CAS-06`: Page CMS optional expected-revision propagation and atomic
  stale-tab guard; stale writes fail closed with 409 and successful saves
  rotate the expected revision. Status: local normal/race complete via
  `.tiller/scratch/cas-20260827.mod`; exact-pin gate pending; separate from
  Home collaboration CAS.

Current execution board:

- `POLISH-QUALITY-04`: local complete, 6/6 focused browser tests; component
  geometry only, actual Noni visuals pending.
- `POLISH-STATUS-11`: local complete; explicit error/recovery placement and
  narrow stacked save-status rows are covered by the CSS contract and 409/
  422/create-link fixture evidence. Actual Noni visual confirmation remains
  Integration08 work.
- `POLISH-CONTENT-01`: local complete, 34/34 focused browser cases.
- `POLISH-MEDIA-02`: local complete, 11/11 focused browser cases.
- `POLISH-SAVE-07`: implementation and focused tests are frozen per
  `save07-report.md` (42/42 focused Go/Node/TS checks); exact-pin integration
  and the clean Gate10 rerun remain pending.
- `POLISH-SAVE-03A`: local readonly adversarial audit complete (3/3); its
  fixture is not a runtime implementation owner and Gate10 will rerun it.
- `POLISH-ASSET-05`: local complete focused/race/vet plus IMS-only and weak
  If-Range follow-up; no deployed claim.
- `POLISH-CAS-06`: local Noni normal/race complete under isolated overlay;
  released exact pin still pending.
- `POLISH-INTEGRATION-08`: DEVGO preliminary run used the isolated CAS overlay;
  the prior real-host 409 visual gate failed on collapsed status geometry and
  awaits STATUS11's frozen CSS plus a fresh capture. It remains the sole Noni
  build owner for the final no-overlay exact-pin two-tab,
  managed-navigation/native-undo/media-keyboard, and screenshot gate.
- `POLISH-BOOK-09`: this status/spec reconciliation is complete; it does not
  certify the overall polish or deployment.
- `POLISH-GATE-10`: run 1 completed after GO; full Go/race and TypeScript passed,
  but standalone browser execution was 122/124 passed with one intentional
  skip and one pending-navigation failure. Run 2 on the frozen source is now
  locally complete: 125/126 passed, one intentional skip, zero failures, and
  unchanged pre/post source fingerprint; actual Noni integration remains
  separate.

## Gate10 local verification packet

`POLISH-GATE-10` is a final Studio-only QA pass after SAVE07 source freeze and
an explicit root GO. It does not build or boot Noni. Its unique artifact root
is `.tiller/scratch/codex/enterprise-polish-20260827/gate10/`; the generated
`.canopy/index.json` remains an excluded tooling artifact.

The standalone Chromium fixture set is exactly:

```text
parity_e2e/authoring_surface_workflows_test.ts
parity_e2e/authoringruntime_test.ts
parity_e2e/contenteditorruntime_test.ts
parity_e2e/contenteditor_focus_test.ts
parity_e2e/contenteditor_pointer_test.ts
parity_e2e/contenteditor_revision_test.ts
parity_e2e/contenteditor_save_adversarial_test.ts
parity_e2e/enterprise_editor_quality_test.ts
parity_e2e/flow_designer_durable_ops_test.ts
parity_e2e/interactions_smoke_test.ts
parity_e2e/mediaruntime_test.ts
parity_e2e/sectionorderruntime_test.ts
parity_e2e/sitemapruntime_test.ts
parity_e2e/state_history_modern_test.ts
parity_e2e/inlineeditruntime_test.ts
```

This list includes the content editor focus/pointer/revision/adversarial
contracts, media, Page CMS quality, state/workbench-adjacent history, and the
other true local fixture contracts. `inlineeditruntime_test.ts` is retained
for inventory but currently has an intentional skipped placeholder. The
Playwright `blocklayoutruntime_test.ts` and `workbenchruntime_test.ts` files
are harness-backed external-host parity suites and are excluded from Gate10;
their Go packages remain covered by the full Go suite. All `reference_apps*`
files and any harness/server boot that could build Noni are excluded. Actual
Noni screenshots and managed-host behavior belong only to INTEGRATION08.

After GO, run these exact commands with Node 24.7.0 and the existing Chromium
cache, writing Playwright results under the unique Gate10 root:

```text
GOWORK=off GOFLAGS= go test ./...
GOWORK=off GOFLAGS= go test -race ./contenteditorruntime ./mediaruntime ./sectionorderruntime ./hostruntime ./backoffice ./internal/runtimeasset
(cd parity_e2e && /home/draco/.nvm/versions/node/v24.7.0/bin/node ./node_modules/typescript/bin/tsc --noEmit -p tsconfig.json)
(cd parity_e2e && PLAYWRIGHT_BROWSERS_PATH=/home/draco/.cache/ms-playwright /home/draco/.nvm/versions/node/v24.7.0/bin/node ./node_modules/@playwright/test/cli.js test --project=chromium --output=/home/draco/work/gosx-studio/.tiller/scratch/codex/enterprise-polish-20260827/gate10/playwright authoring_surface_workflows_test.ts authoringruntime_test.ts contenteditorruntime_test.ts contenteditor_focus_test.ts contenteditor_pointer_test.ts contenteditor_revision_test.ts contenteditor_save_adversarial_test.ts enterprise_editor_quality_test.ts flow_designer_durable_ops_test.ts interactions_smoke_test.ts mediaruntime_test.ts sectionorderruntime_test.ts sitemapruntime_test.ts state_history_modern_test.ts inlineeditruntime_test.ts)
```

The Gate10 report records the exact pre/post source fingerprint and
`git diff --stat`, pass/skip/fail counts, actual failures, source changes seen
during execution, artifact paths, and caveats. It makes no enterprise
performance, deployment, or visual signoff claim; the standalone quality
screenshots remain component geometry evidence.

## Visual contract

The authoritative visual system remains the `## Visual System` section of
[`2026-08-27-modern-editor-turnover.md`](2026-08-27-modern-editor-turnover.md):
Paper & Ink canvas, Fraunces display type, Space Grotesk body type, JetBrains
Mono metadata, terracotta/ink hierarchy, tokenized spacing/radii, and subtle
motion. This pass does not introduce a new palette, typography system, shell,
or freeform layout model.

The owned stylesheet additions are deliberately component-scoped:

- `.media-list-editor` and its generated item/field/action children stay
  within the host width, wrap long labels/URLs, and stack at the existing
  820px breakpoint.
- Generated media search/status text receives the same bounded text treatment
  as the existing picker controls.
- Forced-colors mode restores semantic system colors (`Canvas`, `CanvasText`,
  `ButtonText`, and `Highlight`) and a visible 2px focus ring while retaining
  native forced-color adjustment.
- Reduced-motion mode removes transition and transform feedback from editor,
  picker, gallery-list, and button surfaces. Committed state remains visible.
- Save failure/conflict diagnostics use an explicit full editor track; field
  errors and recovery links wrap within that track, while the existing compact
  idle/dirty/saving/saved arrangement remains. At the existing 820px breakpoint,
  failure rows stack so a visible Retry/current-version link cannot starve the
  message column. Recovery links retain tokenized focus and coarse-pointer
  target treatment.

## Fixture contract

`parity_e2e/enterprise_editor_quality_test.ts` is intentionally standalone. It
loads `hostruntime/assets/studio.css`,
`contenteditorruntime/content_editor.js`, and
`mediaruntime/media_runtime.js` in a local `page.setContent` fixture. Its DOM
is the faithful `backoffice/backend_page_detail.go` Page CMS detail contract:
the renderer root/data hooks, heading, datalist, enhanced form, primary and
secondary action rows, content editor, SEO fields, and revision-history node.
The fixture adds only body/viewport normalization; it contains no synthetic
workbench rails, shell grid, command palette, or test-only layout overrides.

The fixture proves component-level evidence for:

- primary Page CMS controls, long headings, long route/asset URLs, and no page
  overflow at 1600x900, 1280x800, and 390x844;
- pointer/keyboard text editing and move-down focus fallback on the real
  content-editor runtime markup;
- forced colors, reduced motion, visible keyboard focus, and a narrow 1.25x
  CSS zoom state;
- the real datalist-backed media picker in a dense 24-asset state, its empty
  content-editor state, and the generated media-lines gallery editor,
  including selected-asset styling and removal back to an explicit empty
  gallery state;
- structured 409 conflict diagnostics with a marked revision and safe
  current-version link at 1280x800 and 390x844, plus a structured 422
  validation response and create response that preserves a newer local edit
  while exposing the saved-item recovery link at 390x844;
- a 100-block select/move/filter/restore sequence with timing written only as
  fixture evidence, never as an enterprise performance budget or certification.

Screenshots capture initial and meaningful interaction states for all three
required viewports. Full workbench composition and host-mounted screenshots
remain an integration-pass responsibility; these fixtures must not be read as
proof of that shell's layout.

## Acceptance gates

The root acceptance packet remains authoritative:

- Paper & Ink is preserved; no broad redesign, arbitrary freeform rewrite, or
  new dependency is introduced.
- Real pointer/keyboard editing must retain focus, caret, and native undo.
- Reversible operations need a focus fallback and sensible announcements.
- A pending save must not permit competing writes or false success; failed
  saves must retain local edits, with no silent draft loss.
- Cold/managed navigation must tolerate repeated mount.
- Primary controls must be visible at 1600x900, 1280x800, and 390x844, with a
  narrow/zoom variant, forced-colors and reduced-motion support, and full
  keyboard reachability.
- A representative 100-block sequence is recorded without an unsupported
  enterprise-performance claim.
- No production, DNS, live-data, email, or auth state is changed.
- SAVE07 must prove pending-action lock, marked revision rotation, native form
  semantics, validation focus, and non-destructive conflict comparison before
  the overall checkpoint.
- INTEGRATION08 must capture actual Noni screenshots and real two-tab behavior,
  then verify the exact Studio pin without the development overlay.

## Evidence and verification

Ignored quality evidence is written under
`.tiller/scratch/codex/enterprise-polish-20260827/qa/quality04/`:

- `page-detail-initial-{1600,1280,390}.png` and
  `page-detail-interaction-{1600,1280,390}.png`;
- `forced-colors-reduced-motion-page-detail-1280.png`;
- `media-picker-dense-empty-{1280,390,390-zoom}.png` and
  `media-list-gallery-{390,selected-390,empty-390}.png`;
- `100-block-interaction.png`;
- `responsive-evidence.json`, `accessibility-evidence.json`,
  `media-empty-evidence.json`, `100-block-timing.json`,
  `save-conflict-evidence.json`, and `save-validation-created-evidence.json`;
- `save-conflict-{1280,390}.png`, `save-validation-390.png`, and
  `save-created-link-390.png`.

Peer status sources consumed for this board are
`.tiller/scratch/codex/enterprise-polish-20260827/{content-report.md,media-report.md,asset-cache-report.md,cas-report.md,qa/quality04/quality-report.md}`.

Verification commands:

```text
GOWORK=off GOFLAGS= go test ./hostruntime -run 'TestEnterpriseEditorPolish' -count=1
(cd parity_e2e && /home/draco/.nvm/versions/node/v24.7.0/bin/node ./node_modules/typescript/bin/tsc --noEmit -p tsconfig.json)
(cd parity_e2e && PLAYWRIGHT_BROWSERS_PATH=/home/draco/.cache/ms-playwright /home/draco/.nvm/versions/node/v24.7.0/bin/node ./node_modules/@playwright/test/cli.js test enterprise_editor_quality_test.ts --project=chromium --output=/home/draco/work/gosx-studio/.tiller/scratch/codex/enterprise-polish-20260827/status11/playwright)
```

The quality fixture's source-defined screenshots remain under
`qa/quality04/`; the browser runner's report/output is unique under
`status11/playwright/`, so concurrent content, media, save, asset, and CAS
suites cannot overwrite its runner artifacts.

## Explicit exclusions and residual risk

- No content/media/save runtime implementation changes are part of this slice;
  peer owners' reports and tests remain required for those gates. SAVE07 is
  frozen per its report; the clean Gate10 rerun and exact-pin integration remain
  separate.
- STATUS11 has not built Noni or verified the released exact pin. INTEGRATION08
  is the sole Noni build owner and may use its preliminary overlay only for
  discovery; its final run must be no-overlay/exact-pin. No deployment, staging
  re-seed, DNS operation, auth/email work, live data access, or production
  performance certification is included.
- Standalone screenshots do not certify the full Noni-mounted workbench,
  server actions, persistence, publish boundary, or navigation lifecycle.
- STATUS11's long-message geometry and recovery-link evidence is standalone
  component proof only; its UA-like typography/base-form rendering does not
  stand in for the actual Noni host screenshot.
- The media list CSS is proven against runtime-generated markup; host-specific
  wrappers still require the later integration screenshot pass.
- Any runtime worker change during testing must be re-run by root before a
  checkpoint is accepted.

Checkpoint candidate: the local QUALITY04, STATUS11, CONTENT01, MEDIA02, and
ASSET05 slices are candidates; CAS06 is a candidate only with its exact-pin
caveat. The overall polish checkpoint is not ready until Gate10 reruns cleanly
and INTEGRATION08 completes. No commit or push was performed by this owner.

Arbiter next action: root reviews/freezes STATUS11, releases INTEGRATION08 for
the fresh actual-host 1280/390 capture and later no-overlay exact-pin gate, then
reruns Gate10 with a fresh source fingerprint. Nothing here is
enterprise/deployment certified.
