# Enterprise Editor Polish Spec

Date: 2026-08-27

Status: final local candidate evidence is complete and the Studio checkpoint is
pushed; the known exact-pin Noni integration passed locally with no replacement.
TEST18's bounded integration-test-only correction/rerun is complete, and the
CONFIG24 opt-in three-engine standalone gate passed 267/267 after the WEBKIT21
test-input investigation completed. The candidate is local/not deployed,
staging approval is pending, and this is not enterprise certification,
all-browser support, or a zero-warning claim.

This slice carries the visual and interaction quality contract for the shared
Studio editor surfaces. It preserves the existing Paper & Ink territory and
proves small, reusable CSS corrections against the actual Page CMS detail
renderer markup and the existing content/media browser runtimes.

## Baseline and ownership

- Studio baseline: clean `35f0ef2`.
- Noni baseline: `0ff2f9b`, exact-pinned to Studio `35f0ef2`.
- Current Studio candidate: pushed commit
  `8a4c9b8de3560111bfc3fdf915e0b39988a8c0b8`.
- Current Noni candidate: `m31labs.dev/gosx-studio`
  `v0.6.2-0.20260827190001-8a4c9b8de356`; final `go list -m` reported
  `Replace: null` with no development overlay.
- Noni source checkpoint: CP-NONI20 reports pushed commit
  `813b474337d6e55be89b2522b59d34e3a1061157`; its exact eight-file CAS/pin
  checkpoint retains the Studio pin above and no replacement.
- Existing staging: `3eb` digest unchanged; resolved user emails unchanged.
- Email configuration remains settled and unchanged; actual owner walkthrough
  and acceptance are unverified.
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
  development overlay; the known final exact-pin run passed without a
  replacement, and TEST18's corrected rerun is complete.

Adjacent packet status:

- `POLISH-ASSET-05`: host/runtime asset cache reliability; changed ETag returns
  200, unchanged ETag returns 304, and HEAD/no-cache, IMS-only, and weak
  If-Range behavior are locally verified. Status: local complete release-
  reliability gate; no deployed claim.
- `POLISH-CAS-06`: Page CMS optional expected-revision propagation and atomic
  stale-tab guard; stale writes fail closed with 409 and successful saves
  rotate the expected revision. Status: final no-overlay exact-pin integration
  passed; separate from Home collaboration CAS.

Current execution board:

- `POLISH-QUALITY-04`: local complete, 6/6 focused browser tests; component
  geometry only. Actual Noni Chromium screenshots are recorded by Integration08;
  the six-case portability subgate is now 18/18 across Chromium, Firefox, and
  WebKit with an explicit WebKit property limitation.
- `POLISH-STATUS-11`: local complete; explicit error/recovery placement and
  narrow stacked save-status rows are covered by the CSS contract and 409/
  422/create-link fixture evidence. Root accepted the final Noni desktop/mobile
  conflict viewport geometry through Integration08.
- `POLISH-CONTENT-01`: local complete, 34/34 focused browser cases.
- `POLISH-MEDIA-02`: local complete, 11/11 focused browser cases.
- `POLISH-SAVE-07`: implementation and focused tests are frozen per
  `save07-report.md` (42/42 focused browser cases plus related Go/Node/TS
  checks); exact-pin integration passed in the final Noni gate.
- `POLISH-SAVE-03A`: local readonly adversarial audit complete (3/3); its
  fixture is not a runtime implementation owner; Gate10 reran it successfully.
- `POLISH-ASSET-05`: local complete focused/race/vet plus IMS-only and weak
  If-Range follow-up; no deployed claim.
- `POLISH-CAS-06`: local Noni normal/race passed under the development overlay;
  final released exact pin is verified by Integration08 with no replacement.
- `POLISH-INTEGRATION-08`: final no-overlay exact-pin gate complete: 10/10
  real-host Chromium tests, 67 Noni Go packages, two-package race gate, module
  verification, production GoSX build, and normal server build passed. Root
  accepted desktop/mobile conflict viewport geometry. TEST18's bounded
  correction now records 303 transport proof separately from DOM readback and
  confirms media URL/alt persistence after remount; its corrected rerun and
  production regeneration passed.
- `POLISH-BOOK-09`: this status/spec reconciliation is complete; it does not
  certify the overall polish or deployment.
- `POLISH-GATE-10`: final local Studio run is complete: 47 test-bearing Go
  packages passed (49 discovered including two no-test packages), focused race
  passed, TypeScript passed, and standalone Chromium was 125/126 with one
  intentional skip and zero failures; source fingerprint was unchanged.
- `POLISH-TEST-18`: complete integration-test-only closeout correction owned by
  Integration08; the exact Studio pin/runtime stayed frozen, and the corrected
  10-case no-overlay Chromium run, TypeScript rerun, and production regeneration
  passed.
- `POLISH-CROSSBROWSER-19`: historical independent Firefox/WebKit evidence is
  recorded; its quality-fixture subset passed 18/18 across Chromium, Firefox,
  and WebKit. CONFIG24 now supplies the reproducible 267/267 nine-file matrix;
  no all-browser support claim is made.
- `POLISH-CROSS-CONFIG24`: complete reproducible opt-in gate; exactly nine
  standalone files ran serially in explicit Chromium, Firefox, and WebKit
  projects (89/89/89, 267/267 total), with per-run/per-project artifacts and
  no reference-app or harness boot.
- `POLISH-WEBKIT-22`: complete quality-test portability subgate, 6/6 per
  engine. Chromium and Firefox support `forced-color-adjust` and activate
  forced-colors; WebKit activates the media query but reports the CSS property
  unsupported, which is recorded rather than treated as a blanket pass.
- `POLISH-WEBKIT-21`: complete bounded content-test investigation; the original
  diagonal WebKit input showed the marker but did not dispatch `drop`, while a
  documented second real endpoint move makes the pointer test pass 2/2 in
  Chromium, Firefox, and WebKit. No runtime workaround or general WebKit DnD
  claim is made.
- `POLISH-CP-NONI-20`: Noni source checkpoint
  `813b474337d6e55be89b2522b59d34e3a1061157` is pushed with the exact Studio
  pin and no replacement; local docs record the completed TEST18/WEBKIT21
  follow-up and remain local/not deployed.

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

## CONFIG24 reproducible three-engine standalone gate

`parity_e2e/playwright.enterprise.config.ts` is an opt-in configuration that
inherits the safe defaults while allowlisting exactly these nine standalone
fixtures: `contenteditorruntime_test.ts`, `contenteditor_focus_test.ts`,
`contenteditor_pointer_test.ts`, `contenteditor_revision_test.ts`,
`contenteditor_save_adversarial_test.ts`, `enterprise_editor_quality_test.ts`,
`mediaruntime_test.ts`, `sectionorderruntime_test.ts`, and
`state_history_modern_test.ts`. It uses explicit Chromium, Firefox, and WebKit
projects, serial execution, one worker, and no `reference_apps*`, harness, or
server/Noni boot. `GOSX_STUDIO_ENTERPRISE_ARTIFACT_ROOT` selects an optional
unique run root; unset and empty values use the ignored task-local default,
and each project receives separate runner and quality-artifact directories.

The final local command was:

```text
(cd parity_e2e && \
  GOSX_STUDIO_ENTERPRISE_ARTIFACT_ROOT=../.tiller/scratch/codex/enterprise-polish-20260827/crossbrowser24/final-20260827-fixed \
  PLAYWRIGHT_BROWSERS_PATH=/home/draco/.cache/ms-playwright \
  ./node_modules/.bin/playwright test --config=playwright.enterprise.config.ts)
```

Inventory was 89 tests per project, 9 unique files, and zero forbidden
reference/harness entries. The authoritative result was **267/267 passed**
(89 Chromium, 89 Firefox, 89 WebKit), one worker, zero skips/failures. The
quality fixture captured initial/interaction 1600/1280/390 screenshots,
forced-colors/reduced-motion evidence, dense/empty media states, structured
409/422 status states, and 100-block timing without an enterprise performance
claim. Chromium and Firefox support `forced-color-adjust`; WebKit records
`CSS.supports('forced-color-adjust', 'auto') === false` while forced-colors
media emulation is active. Focus and reduced-motion assertions remain
unconditional, and this engine evidence is not a Safari, iOS, physical-device,
or all-browser certification claim.

The exact counts, capability JSON, artifact roots, and unchanged source
fingerprint are recorded in
`.tiller/scratch/codex/enterprise-polish-20260827/crossbrowser24/config24-report.md`.

## Current local candidate and final-gate evidence

The current candidate is local/not deployed. Studio commit
`8a4c9b8de3560111bfc3fdf915e0b39988a8c0b8` is pushed. Noni resolves
`m31labs.dev/gosx-studio` at
`v0.6.2-0.20260827190001-8a4c9b8de356` with `GOWORK=off`, empty `GOFLAGS`,
and `Replace: null`; no development replacement was used for the known final
gate. CP-NONI20 reports the Noni source checkpoint pushed at
`813b474337d6e55be89b2522b59d34e3a1061157`, retaining that exact dependency
pin with no replacement. Staging approval remains pending, and the settled
email configuration is unchanged. Actual owner walkthrough/acceptance is
unverified.

Final Integration08 evidence from the no-overlay exact-pin run:

- Noni Go: 67 packages discovered, including no-test and generated packages;
  all passed.
- Noni race: `internal/cms` and `app/admin/pages` passed.
- `go mod verify` and exact module resolution passed with no replacement.
- Real-host Chromium: 10/10 passed, one worker, zero skips/failures in the
  known run; root accepted desktop/mobile conflict viewport geometry.
- Production GoSX build and normal Go server build passed. The production
  build emitted 89 warnings; this record does not assert whether they are
  pre-existing without comparative warning evidence.
- Studio Gate10: 47 test-bearing Go packages passed (49 discovered including
  two no-test packages), focused race and TypeScript passed, and standalone
  Chromium was 125 passed, 1 intentional skip, 0 failures.
- Additional engines: the quality fixture is verified 6/6 in Chromium, Firefox,
  and WebKit, and CONFIG24's exact nine-file matrix is 267/267 (89 per engine).
  Firefox and Chromium report the forced-color property supported; WebKit
  reports `CSS.supports('forced-color-adjust', 'auto') === false` while
  `matchMedia('(forced-colors: active)').matches === true` after emulation. The
  evidence is not generalized to all-browser support.

`POLISH-TEST-18` was a bounded correction to
`parity_e2e/reference_apps_enterprise_polish_test.ts`: actual HTTP 303 proof is
recorded separately from DOM readback (no synthesized JSON payload), and media
URL/alt persistence is asserted after remount. Its corrected TypeScript/
10-case no-overlay rerun and production regeneration passed; the Studio/Noni
runtime pin remains frozen.

`POLISH-CROSSBROWSER-19` is intentionally separate from the real-host Noni
gate. Its historical report covers nine standalone Studio fixtures in Firefox
and WebKit and identifies browser/tool availability and platform limitations.
The reproducible CONFIG24 gate now reruns those nine fixtures across all three
engines with no missing-engine substitution or silent pass.

`POLISH-WEBKIT-22` records the bounded quality-fixture result under
`.tiller/scratch/codex/enterprise-polish-20260827/crossbrowser19/`: Chromium,
Firefox, and WebKit each passed all six quality cases in unique output roots.
Focus visibility and reduced-motion assertions remain unconditional. Only the
`forced-color-adjust` computed-style checks are conditional on
`CSS.supports`; WebKit's unsupported property is explicit in its capability and
accessibility evidence. This does not change shared CSS or runtime behavior.

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
Final checkpoint sources are `integration08/integration-report.md`,
`cp13-checkpoint-report.md`, and the CP-NONI20 report; TEST18 and
CROSSBROWSER19/WEBKIT22/WEBKIT21/CONFIG24 reports are recorded; local docs
are ready for root's checkpoint review.

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

- No content/media/save runtime implementation changes are part of this docs
  closeout; peer owners' reports and tests remain the source evidence for those
  gates. SAVE07, CAS06, TEST18, WEBKIT21, and the known Integration08 exact-pin
  run are frozen or complete as reported.
- CP13 Studio commit `8a4c9b8de3560111bfc3fdf915e0b39988a8c0b8` and CP-NONI20
  Noni commit `813b474337d6e55be89b2522b59d34e3a1061157` are checkpoint
  candidates. The released Studio module resolves at the exact pin
  `v0.6.2-0.20260827190001-8a4c9b8de356` with no replacement in the known
  Integration08 run; TEST18 and CONFIG24 are complete local evidence.
- No deployment, staging re-seed, DNS operation, auth/email work, live data
  access, or production performance certification is included. The production
  build recorded 89 warnings; without comparative evidence this spec does not
  label them pre-existing and does not claim zero warnings.
- Standalone screenshots do not certify the full Noni-mounted workbench,
  server actions, persistence, publish boundary, or navigation lifecycle.
- STATUS11's long-message geometry and recovery-link evidence is standalone
  component proof only; its UA-like typography/base-form rendering does not
  stand in for the actual Noni host screenshot.
- The media list CSS is proven against runtime-generated markup; host-specific
  wrappers have known Integration08 host screenshots, but those captures do not
  establish broad visual signoff beyond the recorded scenarios.
- Chromium, Firefox, and WebKit quality fixtures are verified, and CONFIG24
  passes 267/267 across the three engines. WebKit does not support the
  `forced-color-adjust` property according to `CSS.supports`, while forced-
  colors emulation is active; this is explicit capability evidence, not a
  product failure or an all-browser certification claim. WEBKIT21 completed as
  a bounded test-input investigation with no runtime workaround retained.
- The old four-hour fresh-cache policy cannot be retroactively evicted for
  already-fresh browser/CDN responses; rollout may require a hard refresh or
  the existing versioned-URL mechanism.
- Staging approval and actual owner walkthrough/acceptance remain unverified;
  the current candidate is local/not deployed. Any runtime worker change during
  testing must be re-run by root before a checkpoint is accepted.

Checkpoint candidate: CP13 Studio, CP-NONI20 Noni, the clean Gate10 run,
CONFIG24's 267/267 three-engine run, the no-overlay exact-pin Integration08
evidence, and WEBKIT22's quality-test portability slice are local checkpoint
candidates. TEST18 and WEBKIT21 are complete bounded follow-ups, and this spec
is ready for root's local documentation/checkpoint review. Staging approval and
owner walkthrough are external next gates, not blockers to freezing accurate
local documentation; no deployment or enterprise certification is claimed.

Arbiter next action: root reviews the six-file Studio checkpoint (`studio.css`,
the CSS Go contract, the quality fixture, the pointer fixture, CONFIG24's
tracked config, and README), preserves all ignored reports/evidence, and
records the local docs checkpoint. Integration08/CP-NONI20 exact-pin evidence
remains separately owned; root decides staging approval and owner walkthrough.
Nothing here is enterprise/deployment certified.
