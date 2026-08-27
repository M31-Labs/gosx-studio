# Enterprise Editor Polish Spec

Date: 2026-08-27

> Historical scope: this document preserves the prior STAGE16 release and its
> evidence. The current round-2 identity and delta are in the [Editor Polish
> Round 2 proof record](./2026-08-27-editor-polish-round2.md).

Status: the exact-pin Noni release is deployed to staging and the local Studio
and Noni verification gates are complete. The current image, data receipt, and
browser evidence are recorded below. This remains a bounded polish record, not
enterprise certification, all-browser support, owner acceptance, or a
zero-warning claim.

This slice carries the visual and interaction quality contract for the shared
Studio editor surfaces. It preserves the existing Paper & Ink territory and
proves small, reusable CSS corrections against the actual Page CMS detail
renderer markup and the existing content/media browser runtimes.

## Baseline and ownership

- Historical Studio baseline: clean `35f0ef2`; historical Noni baseline:
  `0ff2f9b`, exact-pinned to that Studio baseline.
- Current deployed Noni source: clean commit
  `6e605dc60c75cb4b380940f9df29a9ed2c652ac1`.
- Current deployed Studio source: `1f6fee87439c347a5d76e4d420ce4b2bd8002a03`;
  the deployed module is
  `v0.6.2-0.20260827212051-1f6fee87439c` with `Replace: null` and no
  development overlay.
- Historical CP-NONI20 source checkpoint:
  `813b474337d6e55be89b2522b59d34e3a1061157`; it is an ancestor of the
  deployed source, not the deployed image identity.
- Current staging image: `sha256:2d3f7d5c8a449a4026910d5b38d78e75647d78b9263295fbb9731e2f9624da72`,
  tag `20260827-145730-6e605dc-staging`, generation/observedGeneration
  `174/174`, one pod Ready. The earlier `3eb` and `3d60` images remain
  historical deployment evidence, not the current release.
- Resolved user emails remain unchanged; actual owner walkthrough and
  acceptance are unverified.
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
  the final QA07/CONFIG24 standalone matrix is 270/270 (90 per engine) with an
  explicit WebKit property limitation.
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
- `POLISH-INTEGRATION-08`: final no-overlay exact-pin gate complete: 12/12
  real-host Chromium tests, full Noni Go and full-race verification, module
  verification, TypeScript, production GoSX build, and normal server build
  passed. Root accepted desktop/mobile conflict viewport geometry. TEST18's
  bounded correction records 303 transport proof separately from DOM readback
  and confirms media URL/alt persistence after remount; its corrected rerun and
  production regeneration passed.
- `POLISH-BOOK-09`: this status/spec reconciliation is complete; it does not
  certify the overall polish or deployment.
- `POLISH-GATE-10`: historical pre-UX06 local Studio run recorded 47
  test-bearing Go packages, focused race, TypeScript, and standalone Chromium
  125/126 with one intentional skip. It is superseded for current standalone
  counts by QA07's frozen 270/270 three-engine matrix.
- `POLISH-TEST-18`: complete integration-test-only closeout correction owned by
  Integration08; the exact Studio pin/runtime stayed frozen, and the corrected
  10-case no-overlay Chromium run, TypeScript rerun, and production regeneration
  passed as a separate historical test-only result.
- `POLISH-CROSSBROWSER-19`: historical independent Firefox/WebKit evidence is
  recorded; its quality-fixture subset passed 18/18 across Chromium, Firefox,
  and WebKit. Its original CONFIG24 result was 267/267 before UX06; the final
  QA07/CONFIG24 rerun is 270/270 across the same nine files. No all-browser
  support claim is made.
- `POLISH-CROSS-CONFIG24`: complete reproducible opt-in gate; exactly nine
  standalone files ran serially in explicit Chromium, Firefox, and WebKit
  projects. The final QA07 rerun is authoritative at 90/90/90 (**270/270**
  total) after the Search Enter regression case was added, with per-run/
  per-project artifacts and no reference-app or harness boot.
- `POLISH-WEBKIT-22`: complete quality-test portability subgate, 6/6 per
  engine. Chromium and Firefox support `forced-color-adjust` and activate
  forced-colors; WebKit activates the media query but reports the CSS property
  unsupported, which is recorded rather than treated as a blanket pass.
- `POLISH-WEBKIT-21`: complete bounded content-test investigation; the original
  diagonal WebKit input showed the marker but did not dispatch `drop`, while a
  documented second real endpoint move makes the pointer test pass 2/2 in
  Chromium, Firefox, and WebKit. No runtime workaround or general WebKit DnD
  claim is made.
- `POLISH-CP-NONI-20`: historical Noni source checkpoint
  `813b474337d6e55be89b2522b59d34e3a1061157` is pushed with the exact Studio
  pin and no replacement; it remains separate from the deployed source
  `6e605dc60c75cb4b380940f9df29a9ed2c652ac1`.

## Historical Gate10 local verification packet (pre-UX06)

`POLISH-GATE-10` was the Studio-only QA pass after SAVE07 source freeze and an
explicit root GO. It did not build or boot Noni. Its unique artifact root was
`.tiller/scratch/codex/enterprise-polish-20260827/gate10/`; the generated
`.canopy/index.json` remains an excluded tooling artifact. The pre-UX06
Chromium count below is historical; QA07's 270/270 matrix is authoritative for
the current standalone evidence.

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

The historical CONFIG24 inventory was 89 tests per project, 9 unique files,
and zero forbidden reference/harness entries. Its result was **267/267 passed**
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

### QA07 current three-engine rerun

QA07 reran the same nine standalone files after UX06 added the Search Enter
zero-POST case. It used the tracked config with a unique artifact root:

```text
cd /home/draco/work/gosx-studio/parity_e2e
env -u QUALITY_ARTIFACT_ROOT GOSX_STUDIO_ENTERPRISE_ARTIFACT_ROOT=/home/draco/work/gosx-studio/.tiller/scratch/codex/staging-polish-20260827/quality/qa07/parity PLAYWRIGHT_BROWSERS_PATH=/home/draco/.cache/ms-playwright ./node_modules/.bin/playwright test --config=playwright.enterprise.config.ts
```

The inventory was **270 tests in 9 files** and the matrix passed **270/270**
(90 Chromium, 90 Firefox, 90 WebKit), with zero skips/failures. The QA07
report is `.tiller/scratch/codex/staging-polish-20260827/quality/qa07/report.md`;
its per-project runner and quality directories are isolated under that parity
artifact root.

## Current deployed release and final-gate evidence

The exact-pin release is deployed to staging. Studio source
`1f6fee87439c347a5d76e4d420ce4b2bd8002a03` resolves through
`m31labs.dev/gosx-studio` at
`v0.6.2-0.20260827212051-1f6fee87439c` with `GOWORK=off`, empty `GOFLAGS`,
and `Replace: null`; no development replacement was used for the released
runtime. The deployed Noni source is
`6e605dc60c75cb4b380940f9df29a9ed2c652ac1`. The immutable staging image is
`sha256:2d3f7d5c8a449a4026910d5b38d78e75647d78b9263295fbb9731e2f9624da72`
with tag `20260827-145730-6e605dc-staging` and generation/observedGeneration
`174/174`; one pod is Ready. A later documentation HEAD is distinct from both
source identities.

STAGE16 versioned module JavaScript, deployment CSS/outer-media assets, two
versioned aliases, and all six manifest runtime checks passed. BROWSER17 passed
10/10 public routes, 2/2 keyboard cases, and 2/2 unauthenticated
admin-boundary cases with no reported asset/request/console/page errors. STAGE12
completed its read-only post-deploy comparison at `2026-08-27T22:08:26Z`; the
receipt SHA-256 is
`c4ed8559ca195bb9fb791c11fced5f267804d1f86f001edb8a66932e027ae1fb`, against
the STAGE06 pre-deploy baseline. The manifest
`82529b5ecd4ef7b73e6b949a4349c3e7ce48fb555d80cabb6fad88a4bce51256` matched
(4 files, 289,303 B, media 0), file comparisons and both SQLite checks passed,
and final live identity passed.

The current 12-case host result belongs to INTEGRATION08B/TEST11B. TEST11B's
test-only checkpoint is `457cdd7acbd4829b7f5519b46709a233ea6adc22`, with source
hash `69791a92e2007c163aeb5b8ddcd5d6101f52b66987388d7dfafa107087ef179c`; its
report is `.tiller/scratch/codex/enterprise-polish-20260827/integration08b/integration-report.md`.
The fixture includes the current UX06 Search Enter case and verifies that
Enter does not issue a POST while Search is focused. It models the separate
outer deployment tag and shared Page CMS module release keys; no application
fix was made for the earlier fixture-only mismatch.

CACHE10's current release-key evidence is recorded in
`.tiller/scratch/codex/enterprise-polish-20260827/cache10/report.md`: the
shared Page CMS module URL and outer CSS/media deployment URLs are versioned
from their respective release keys. This is the accepted rollout path; it does
not make the bare CDN cache residual a pass.

Final Integration08B evidence from the no-overlay exact-pin run:

- Full no-overlay Noni Go and full-race verification passed, as did module
  verification, TypeScript, the production GoSX build, and the normal server
  build.
- Real-host Chromium: **12/12** passed with zero failures/skips; root accepted
  desktop/mobile conflict viewport geometry. This is actual host evidence, not
  an all-browser or enterprise certification claim.
- Studio QA07's standalone matrix passed **270/270** cases (**90 each** in
  Chromium, Firefox, and WebKit), distinct from the 12/12 real-host Chromium
  cases. The production build observed **89** static warnings; no comparative
  baseline is asserted.
- Chromium and Firefox report the forced-color property supported; WebKit
  reports `CSS.supports('forced-color-adjust', 'auto') === false` while
  `matchMedia('(forced-colors: active)').matches === true` after emulation. The
  evidence is explicit capability coverage, not generalized all-browser
  support.

`POLISH-TEST-18` was a historical bounded correction against the earlier Studio
runtime `v0.6.2-0.20260827190001-8a4c9b8de356` to
`parity_e2e/reference_apps_enterprise_polish_test.ts`: actual HTTP 303 proof is
recorded separately from DOM readback (no synthesized JSON payload), and media
URL/alt persistence is asserted after remount. Its corrected TypeScript/
10-case no-overlay rerun and production regeneration passed; the Studio/Noni
runtime under test was the earlier `8a4c9b8de3560111bfc3fdf915e0b39988a8c0b8`
pin, not the current deployed `1f6fee87439c347a5d76e4d420ce4b2bd8002a03`
source recorded above.

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
- CP13 Studio CSS checkpoint `8a4c9b8de3560111bfc3fdf915e0b39988a8c0b8` and
  the CP-NONI20 source checkpoint `813b474337d6e55be89b2522b59d34e3a1061157`
  remain historical evidence. The released Studio module resolves at the
  exact pin `v0.6.2-0.20260827212051-1f6fee87439c` with no replacement; the
  deployed Noni source and image are recorded above. TEST18 and CONFIG24 are
  complete test evidence.
- No production re-seed, DNS operation, auth/email work, live-data mutation, or
  production performance certification is included in this polish record.
  STAGE16 staging deployment and STAGE12 read-only verification are recorded
  above. The production build recorded 89 warnings; without comparative
  evidence this spec does not label them pre-existing and does not claim zero
  warnings.
- Standalone screenshots do not certify the full Noni-mounted workbench,
  server actions, persistence, publish boundary, or navigation lifecycle.
- STATUS11's long-message geometry and recovery-link evidence is standalone
  component proof only; its UA-like typography/base-form rendering does not
  stand in for the actual Noni host screenshot.
- The media list CSS is proven against runtime-generated markup; host-specific
  wrappers have known Integration08 host screenshots, but those captures do not
  establish broad visual signoff beyond the recorded scenarios.
- Chromium, Firefox, and WebKit quality fixtures are verified, and QA07/CONFIG24
  passes 270/270 across the three engines (90 per engine); 267/267 is retained
  as historical pre-UX06 evidence. WebKit does not support the
  `forced-color-adjust` property according to `CSS.supports`, while forced-
  colors emulation is active; this is explicit capability evidence, not a
  product failure or an all-browser certification claim. WEBKIT21 completed as
  a bounded test-input investigation with no runtime workaround retained.
- The old four-hour fresh-cache policy cannot be retroactively evicted for
  already-fresh browser/CDN responses. The current bare CSS stale-HIT residual
  remains a technical follow-up; the accepted rollout path uses versioned URLs
  after a full refresh. The latest future IMS probe returned current HTTP 200;
  the false IMS 304 result is historical STAGE04 evidence. Save or reconcile
  unsaved drafts before refreshing.
- STAGE16 and STAGE12 staging gates are complete. Actual owner walkthrough,
  sign-in, and acceptance remain unverified; this record makes no enterprise or
  all-browser certification claim. Any runtime worker change during testing
  must be re-run before a checkpoint is accepted.

Checkpoint records: CP13 Studio, CP-NONI20 Noni, the frozen QA07/CONFIG24
270/270 three-engine run, the no-overlay exact-pin Integration08B evidence,
WEBKIT22's portability slice, STAGE16, and STAGE12 are complete evidence.
TEST18 and WEBKIT21 are complete bounded follow-ups. The current staging
release is available for the owner walkthrough; owner sign-in and acceptance
remain external next gates, and no deployment or enterprise certification is
claimed here.

Arbiter next action: root reviews the five DOC13 documentation edits and
preserves all ignored reports/evidence, then records the final documentation
checkpoint without conflating the later docs HEAD with the deployed source.
Integration08B/CP-NONI20, STAGE16, and STAGE12 evidence remain separately
identified; root schedules the authenticated owner walkthrough. Nothing here
is enterprise/deployment certified.
