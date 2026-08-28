# Repository Investigation — REPORT3 — 2026-08-27

## CURRENT release-candidate addendum — 2026-08-28 UTC

**Status: deployed and verified within the tested staging scope; turnover and
hardening remain open.** This addendum supersedes the obsolete POLISH4 interim
preface while preserving the original REPORT3 body below. It is not release,
enterprise, or owner-acceptance certification.

### Current capabilities and evidence

- Root's source checkpoint is `cb313609fb1d28937a110dab97698491a05e5030`,
  descended from gesture parent `6c6d0f4e71455147554badec1c2f8d25aefdad5f`.
  The normal same-branch push is confirmed for that runtime source checkpoint.
  This documentation addendum and the separately reviewed host-guide changes
  are not runtime changes; their later documentation commits are separate.
- Final WSL development verification passes all 47 test-bearing Go packages
  plus 2 no-test packages in normal and race mode, all 66 parity TypeScript
  files, the candidate helper suite 5/5, and the 24/24 source-hash comparison
  after the intended three-file Gallery change.
- Gallery focused native Windows checks pass 5/5 per engine (15/15 combined)
  with clean exits. The actual candidate-host Gallery test passes 1/1 in
  15.4s at 320/390/1280 with readable layouts, native Tab, and media-filter
  coverage. Root visually reviewed all three screenshots and accepts this
  tested Gallery scope. The final Gallery/matrix reports are frozen and owned
  cleanup is complete; owner acceptance remains a separate turnover gate.
- The final Gallery/matrix reports are frozen at 315 passes with clean exits,
  native identities, 14 blob hashes stable, and owned cleanup complete. This is
  the 105-test-per-engine, 11-fixture native Chrome/Firefox/WebKit matrix run
  with native Windows Chrome/Firefox controllers and the WSL-controller/native
  Windows `run-server` route for WebKit. Two earlier Windows-controller WebKit
  runs (104 and 105 assertions) ended in teardown failure and remain
  historical, not green results.
- The consumer exact-published-pin run resolves
  `v0.6.2-0.20260828060955-cb313609fb1d` for the `cb313609...` source through
  normal `go get`, with no local replacement. The authorized Contacts
  expectation preserves attribute order and weakening nothing. The clean
  consumer checkpoint is `532f8e45968b4c9fbf771569e65f5e04e9ed91da` across
  exactly nine files (module pair plus seven goldens): normal/race covers 45
  source packages (39 test-bearing and 6 no-test), and CMS no-emit passes. An
  exploratory raw Astro `tsc` invocation lacks ambient setup, while the
  installed no-emit command with `--types astro/client` passes; the production
  GoSX build passes with 89 reachability/loader warnings, without a
  warning-by-warning comparison. Post-checkpoint `go version -m` confirms the
  exact `cb313609...` origin hash and
  `vcs.modified=false`. Native Windows Chrome exact-pin QA is 14/14 in 2.6m
  with exit 0 through the WSL controller; the six-file run's archive/cleanup
  is complete, inherited `MUDDY_*` values were cleared, fresh fixture stores
  were used, and the fallback-artifact caveat is documented. No other host
  engine is implied by this Chrome result.

- The guarded backup/isolated-restore run passed in
  `release/fresh-backup-report.md` as run
  `20260828T064217732401768Z`, with receipt SHA
  `cb2e8fb25180187e2c79bb82f954b0b17112ecca96b82d94af1eb97264016ffa`.
  Exact manifests and SQLite checks passed, the restore PVC was retained. At
  backup completion the prior `d773...` staging identity was restored at
  generation 179/179 after the 1→0→1 scale transition, with pod `67np5` and
  one Ready replica; generation 177 is pre-backup historical. The receipt plus
  unchanged-native-FW preflight passed. Root independently hashed the receipt,
  manifests, archive, and `deploy.fw`, accepted baseline script `daef980f...`,
  and issued native staging DEPLOY GO to `release_preflight`. The native deploy
  passed in 2m24s with exit 0 and unchanged native FW; runtime used Noni
  `532f8e4`, the exact Studio pin `cb313...`, and image
  `sha256:bf29389f045568fffb8b0b0d64dd98fa2a76c6e8a3837e47c598a9a439d008b7`
  (`20260828-000734-532f8e4-staging`). Staging is generation 180/180 with
  one Ready replica on the same Deployment PVC/PV, `worker=false`, and
  checkout 0; this current identity supersedes the restored `d773...` state.

- Post-deploy data verification retry passed with exit 0. Receipt
  `/home/draco/backups/muddy-noni/postdeploy-20260828T071420269088584Z/postdeploy-data-verify-report.json`
  has SHA
  `a5620dff16b4098810e533039dbc85c8999ed040d70f38f996a2af1788e4c221`;
  the exact four files total 289303 bytes, media count is 0, the manifest
  begins `82529b5e...`, CMS flows and both DB matches passed, both
  `quick_check` and `integrity_check` passed for both DBs, the helper count is
  0, and `finalIdentity=true`. The initial
  SQLite-PATH issue was pre-capture tooling and is resolved with the existing
  miniconda CLI; no install or assertion change was made.

- Native Windows public QA passed 3/3 in 11.6s on Chrome 152/Win32 with a
  Win32 controller and artifact root
  `.tiller/scratch/codex/polish-round4-20260827/staging-public-qa/runs/windows-chrome-20260828-001831640`;
  the frozen owner report SHA is
  `744480d047615a9e45762b6a008854ba6f8952a53afcd98e4ab7b9845804a8c1`.
  All six Home/Gallery/Shop desktop 1440px and 390px checks returned HTTP 200
  with overflow 0; 72 tracked assets returned 200, health reported framework
  `0.53.8`, and the exact versioned Studio CSS body hash is `9227934d...`.
  Both admin boundary checks passed same-origin/Google-link behavior;
  authenticated dashboard/editor renderer markers were absent from anonymous
  responses. This is a passed anonymous auth boundary, not verified
  owner-authenticated login. Root visually checked all six screenshots without
  clipping. One `favicon.ico` 404 was observed; no JS page errors,
  failed requests, or tracked-asset HTTP errors occurred, so this is not a
  zero-console-error claim. Existing photo/title mismatches and imagery/content
  review remain a separate owner follow-up, not a new Studio defect or
  customer-data change; site-icon branding remains an owner choice.

- Current source/walkthrough capability evidence covers Home drag and keyboard
  ordering, and PageCMS blocks with undo/redo plus draft/preview/publish/
  history/media flows. The product remains a structured editor rather than an
  arbitrary free-form canvas; email behavior is unchanged.

### Current finding mapping

- **F2:** Candidate repository/SHA identity, isolated source-copy execution,
  `Module.Replace.Dir` graph assertion, helper tests, and actual-host Gallery
  proof are implemented. No remote GitHub CI result was verified or claimed in
  this pass.
- **F3:** The three missing suites are wired and the 105×3 shared matrix has
  frozen 315-pass native evidence. Host CI remains Chromium-only and no remote
  GitHub CI result was verified or claimed in this pass.
- **F6/F6b:** Cancellation/no-op behavior and native regressions are fixed in
  the tested scope. The older full-consumer BlockLayout fixture precondition
  gap is not claimed as a pass.
- **F12:** Gallery readability, keyboard, and visual gates are closed for the
  tested 320/390/1280 viewports and native engines; owner acceptance remains a
  separate turnover gate.
- **F4:** The manual published no-overlay exact-pin host proof passed for the
  native Windows Chrome 14/14 run; an automatic published no-overlay CI lane
  is still missing. **F1** remains conditional:
  standalone Room/auth topology is open, while Noni uses authenticated
  `NewTransport`; no breach claim is made.
- **F5 (verified scope):** `HOST_INTEGRATION_GUIDE.md` sections 1 and 5 now
  carry dated verified Noni pins, no-blind-golden and isolated-source-copy/
  `GOFLAGS` caveats, and current CI-selection wording. This closes the stale
  pin/CI-wording scope only; it is not a full adapter-guide audit.
- **F6c/F6d/F6e–F6f, F7, F8, F9, F10, and F11** remain qualified open per the
  original investigation, including physical-touch coverage and the
  published-consumer/adapter-contract limits.

### Remaining gates and boundaries

The native Windows Chrome exact-pin host proof passed 14/14 and the guarded
backup receipt/preflight passed. Native staging deploy passed in 2m24s with
exit 0 and unchanged native FW, using image
`sha256:bf29389f045568fffb8b0b0d64dd98fa2a76c6e8a3837e47c598a9a439d008b7`
and tag `20260828-000734-532f8e4-staging`; staging is generation 180/180
with one Ready replica on the same PVC/PV, `worker=false`, checkout 0. The
post-deploy data retry passed with exit 0 and final identity true. Native
Windows public QA passed 3/3 in 11.6s on Chrome 152/Win32 with six
Home/Gallery/Shop checks at 1440px and 390px, HTTP 200, overflow 0, exact CSS
hash `9227934d...`, and same-origin/anonymous-auth-boundary checks passing;
owner-authenticated login remains unverified. Root saw no visual clipping.
One favicon 404 was observed, so no zero-console-error claim is made. Imagery
and content remain a separate owner review; site-icon branding remains an owner
choice, not a Studio defect or customer-data change. Generation 177 is
pre-backup historical; at backup completion the old `d773...` identity was
restored at generation 179, while current staging is the deployed image at
generation 180/180. The clean nine-file Noni checkpoint, published-pin
normal/race reruns, and CMS no-emit now pass. Functional/layout staging
verification is complete within this bounded evidence; final hardening and
remaining owner login/content-review/turnover records remain open. Owner login,
walkthrough, and acceptance are turnover gates only, not prerequisites for the already
authorized staging deploy. The guide and investigation documents are
checkpointed separately from the runtime source at Studio docs checkpoint
`4d26a4f2e97191358ee6674061880b4d4dd10c7d`. The product remains a structured editor rather than
an arbitrary Squarespace-style canvas, and email behavior is unchanged.

This report makes no enterprise certification, penetration test, dependency-CVE
scan, live-auth validation, or line-by-line audit claim. It contains no raw
logs, customer contents, or sensitive live metadata. Supporting evidence:

- [Local source checkpoint](../.tiller/scratch/codex/polish-round4-20260827/checkpoint-polish/report.md)
- [Final developer verification](../.tiller/scratch/codex/polish-round4-20260827/final-dev-verify-after-gallery/report.md)
- [Gallery CSS/focused report](../.tiller/scratch/codex/polish-round4-20260827/gallery-css-fix/report.md)
- [Final candidate Gallery report](../.tiller/scratch/codex/polish-round4-20260827/validation/run-server-candidate-gallery-final-20260828T0540Z/candidate-gallery-final-report.md)
- [CI candidate report](../.tiller/scratch/codex/polish-round4-20260827/ci/report.md)
- [Consumer compatibility report](../.tiller/scratch/codex/polish-round4-20260827/consumer/report.md)
- [Noni exact-pin checkpoint](../.tiller/scratch/codex/polish-round4-20260827/consumer/exact-pin/report.md)
- [Native Windows Chrome exact-pin QA log](../.tiller/scratch/codex/polish-round4-20260827/validation/exact-pin-qa-20260828T060842Z/artifacts/run-20260828T063152Z/logs/exact-pin-14.log)
- [Fresh guarded backup report](../.tiller/scratch/codex/polish-round4-20260827/release/fresh-backup-report.md)
- [Native deploy report](../.tiller/scratch/codex/polish-round4-20260827/release/native-deploy-report.md)
- [Post-deploy data report](../.tiller/scratch/codex/polish-round4-20260827/release/postdeploy-data-report.md)
- [Final staging public-QA report](../.tiller/scratch/codex/polish-round4-20260827/staging-public-qa/report.md)
- [Host-guide correction report](../.tiller/scratch/codex/polish-round4-20260827/consumer/host-guide-report.md)
- [Final shared-runtime matrix (with historical teardown notes)](../.tiller/scratch/codex/polish-round4-20260827/final-windows-matrix/report.md)
- [Release preflight](../.tiller/scratch/codex/polish-round4-20260827/release/preflight-report.md)

## Historical REPORT3 body — retained unchanged

**Status: COMPLETE INVESTIGATION — READINESS OPEN.** UI and data static review plus the Gallery final-run facts are incorporated. Required turnover acceptance remains pending, but it is not a prerequisite for completing this investigation artifact. This is an investigation artifact, not release certification.

## Executive result

The static investigation is complete at the supplied scope; readiness remains open. The principal open or conditional gates are CI candidate identity, regression selection and engine coverage, published-consumer reproducibility, stale host instructions, collaboration authorization topology, pointer cancellation, lifecycle teardown, media metadata/policy propagation, file-adapter failure semantics, URL-policy ownership, and Gallery visual acceptance.

The repository shows strong implementation and release-discipline signals, but this report does not promote it to enterprise-grade.

## Scope and method

- Real checkout: `//wsl.localhost/Ubuntu/home/draco/work/gosx-studio`. `C:\home\draco\work\gosx-studio` is not the valid checkout.
- Root-validated review snapshot: `b866f455a61be5f606b7af5ce0edd5ef063fe98d`, reported clean at review start.
- Inventory: 594 tracked files; 434 Go files, including 207 Go tests; 33 JS files; 31 GSX files; 57 browser-test files.
- Method: inventory, ownership coverage, and targeted source/test/CI tracing; not every line audited.
- Excluded: ignored/generated data, `node_modules`, full sibling-repository audit, penetration testing, dependency-CVE scan, fresh full-Go/full-browser runs, and live-auth validation.
- The investigation/report-authoring work made no source edits and did not invoke test/build commands. Separately, the combined polish workstream owns three Gallery files (CSS, Go test, TS) and supplied cached Windows Firefox observations. No fresh Linux Go/host tests, deploy, Git mutation, auth change, or data change was made by this report task.

## Subsystem coverage and ownership

| Area | Coverage | State |
|---|---|---|
| Root | README, docs, `.github`, `go.mod`, `go.sum`, root compatibility/facade tests, conformance, `.canopy`, release evidence | Inventory-wide sampled review complete |
| Data | `core`, `authoring`, `cms`, `internal`, plugins, modules | Inventory-wide sampled review complete; file EOF/rollback/persistence/render points root-corroborated |
| UI | All `*runtime` except media, plus canvas, panels, backoffice, shell, sitemap | Inventory-wide sampled review complete; conditional findings F6–F6f |
| Polish | Media runtime and Gallery CSS; three worker-owned Gallery files | Partial; final run incorporated, visual readiness open |

## Ranked findings

### F1 — P1 conditional: standalone collaboration Room can trust forged client authority

**Confidence:** high for the traced path. **Precondition:** a host exposes this standalone Room handler to an untrusted peer. This is conditional exposure, not an observed Noni breach.

- `room.go:146` delegates `ServeHTTP` to the Hub; `registerHub` around `377–391` JSON-decodes client `Actor`/`Transaction` and passes them to `Apply`. [Room](//wsl.localhost/Ubuntu/home/draco/work/gosx-studio/cms/studio/collab/room.go:146)
- The decision helpers are in `room.go:450–479`: `Actor.Kind == ActorSystem` or client capabilities can be trusted, and `canDecide` / `canDecideComment` do not consult the `Authorize` callback. [Decision path](//wsl.localhost/Ubuntu/home/draco/work/gosx-studio/cms/studio/collab/room.go:450)
- If reachable by an untrusted peer, forged system/capability claims can bypass room authorization.
- The inspected Noni mount is a different path: `muddy-noni-commerce/cmd/muddy-noni/main.go:119–130` uses `NewService` plus `NewTransport(Authenticate: studiohost.CollaborationResolver(authn))`, not this Room handler. [Noni mount](//wsl.localhost/Ubuntu/home/draco/work/muddy-noni-commerce/cmd/muddy-noni/main.go:119)

No live exploit was performed and no Noni breach was observed. **Required action:** resolve the actor from the authenticated connection, ignore client-supplied authority, fail closed, and add forged actor/capability regressions; verify deployment topology.

### F2 — P1: CI candidate identity gap can test the published module instead of the PR

**Confidence:** high. **Precondition:** the host/reference-app workflow relies on the checked-out module graph without an explicit Studio candidate assertion or override.

- The workflow checks out Studio and host branches (`.github/workflows/reference-apps.yml:24–62`) and invokes the reference-app script. [Workflow](//wsl.localhost/Ubuntu/home/draco/work/gosx-studio/.github/workflows/reference-apps.yml:24)
- `reference_apps_harness.ts:597–608` builds with the host as cwd; `:793–795` runs `go run` from that cwd with `GOWORK=off`. [Harness](//wsl.localhost/Ubuntu/home/draco/work/gosx-studio/parity_e2e/reference_apps_harness.ts:597)
- Noni `go.mod:13` pins `v0.6.2-0.20260827230336-06854e5ecfe9`, while Studio review is `b866f45…` plus patches; host tests may exercise the published module rather than the PR candidate. [Noni go.mod](//wsl.localhost/Ubuntu/home/draco/work/muddy-noni-commerce/go.mod:13)

**Required action:** assert exact candidate module identity and use an isolated candidate override or exact published candidate in CI.

### F3 — P1/P2: explicit CI selection leaves suites and engine coverage non-guaranteed

**Confidence:** high. **Precondition:** the explicit script list remains the reference-app coverage source of truth.

- `parity_e2e/package.json:10` omits `reference_apps_enterprise_polish_test.ts`, `reference_apps_modern_editor_test.ts`, `reference_apps_page_cms_lifecycle_test.ts`, and standalone content/media suites. [Script](//wsl.localhost/Ubuntu/home/draco/work/gosx-studio/parity_e2e/package.json:10)
- `parity_e2e/playwright.config.ts:43–47` selects Chromium only; the three manual engine checks have no automatic coverage guarantee. [Browser config](//wsl.localhost/Ubuntu/home/draco/work/gosx-studio/parity_e2e/playwright.config.ts:43)

**Required action:** include named suites explicitly and add a maintained Firefox/WebKit/Chromium matrix or machine-checkable contract.

### F4 — P2: local replaces weaken published-consumer reproducibility

**Confidence:** high. **Precondition:** reviewed workflows check out siblings without refs while Studio’s module file uses local replacements.

- `go.mod:5–7` locally replaces GoSX/admin dependencies. [Module graph](//wsl.localhost/Ubuntu/home/draco/work/gosx-studio/go.mod:5)
- This is development coupling, not proof that a clean published consumer resolves the intended release.

**Required action:** add a released-module/no-overlay CI lane and report it separately from local development.

### F5 — P2: host integration guide contains stale version/pin instructions

**Confidence:** high. **Precondition:** the guide remains an onboarding source.

- `docs/HOST_INTEGRATION_GUIDE.md:32–35` presents July 12 pins (`fcf395…`, GoSX `0.29`) while current Noni evidence is `0.53.8` / `06854e5…`. [Guide](//wsl.localhost/Ubuntu/home/draco/work/gosx-studio/docs/HOST_INTEGRATION_GUIDE.md:32)

**Required action:** refresh instructions and separate historical examples from current pins.

### F6 — P1 conditional: older BlockLayout runtime commits pointer-cancel/no-op reorder

**Confidence:** high for this runtime. **Precondition:** host enables the older BlockLayout engine bundle and binds `[data-block-studio-handle]`; do not generalize to newer Home/PageCMS runtimes.

- `blocklayoutruntime/island_runtime.js:594–610` uses one finish path for `pointerup` and `pointercancel`; there is no saved pre-drag order or cancel branch. `renumberIsland` at `:161–177` emits reorder/input/change every call. [Finish](//wsl.localhost/Ubuntu/home/draco/work/gosx-studio/blocklayoutruntime/island_runtime.js:594) · [Renumber](//wsl.localhost/Ubuntu/home/draco/work/gosx-studio/blocklayoutruntime/island_runtime.js:161)
- Existing `blocklayoutruntime_test.ts:719–816` checks the synthetic pointerdown/up lifecycle and expects an engine-list reorder event, but lacks pointercancel and nonmutating no-op coverage and does not assert cross-row geometry. [Test gap](//wsl.localhost/Ubuntu/home/draco/work/gosx-studio/parity_e2e/blocklayoutruntime_test.ts:719)

**Required action:** add cancel, no-op, and real-pointer cross-row regressions before treating this runtime as ready.

### F6b — P1 conditional: Sitemap pointercancel leaves gesture cleanup live

**Confidence:** high for the static paths. **Precondition:** `sitemapruntime` receives a canceled touch/stylus/OS gesture.

- Node drag (`:155–200`), canvas pan (`:205–220`), and marquee (`:245–290`) install document `pointermove`/`pointerup` handlers but no `pointercancel`. A later pointermove can continue the old gesture until pointerup; node coordinates eventually post through `:734–759`. [Sitemap gestures](//wsl.localhost/Ubuntu/home/draco/work/gosx-studio/sitemapruntime/island_runtime.js:155) · [Commit](//wsl.localhost/Ubuntu/home/draco/work/gosx-studio/sitemapruntime/island_runtime.js:734)

**Required action:** add shared cancel teardown for node drag, pan, and marquee. No browser cancellation run was performed.

### F6c — P2 conditional: fragment replacement retains detached runtime controllers

**Confidence:** static retention risk; no measured memory leak. **Precondition:** repeated fragment refresh/navigation removes a bound form or canvas and rebinds a replacement.

- Selection closures capture a form and only check `document.contains` (`selectionruntime/island_runtime.js:223–226,1427–1454`); workbench rail listeners, MutationObserver, and stage scroll listener have no sampled disconnect path (`workbenchruntime/island_runtime.js:225–235,474–526`). [Selection](//wsl.localhost/Ubuntu/home/draco/work/gosx-studio/selectionruntime/island_runtime.js:223) · [Workbench](//wsl.localhost/Ubuntu/home/draco/work/gosx-studio/workbenchruntime/island_runtime.js:225)
- Canvas prevents same-node rebinding but adds fragment-refresh, resize, and beforeunload callbacks per mounted canvas (`canvas_wasm_free_client.js:322–324,913–915`). [Canvas](//wsl.localhost/Ubuntu/home/draco/work/gosx-studio/canvaswasmfreeruntime/canvas_wasm_free_client.js:322)

Per-node guards and normal pointerup cleanup are mitigations. The concern is retention and growing dispatch work across replacement cycles, not a measured leak. **Required action:** define teardown ownership and add bounded repeated-fragment coverage.

### F6d — P2 conditional capability gap: content-editor drag is native desktop drag

**Confidence:** high for the capability statement. **Precondition:** a browser/device does not synthesize HTML5 drag events for touch.

- Content movement uses native drag events (`content_editor.js:493–570`); explicit Up/Down buttons (`:1961–1968`) and keyboard movement (`:603–643,692–755`) remain. [Native drag](//wsl.localhost/Ubuntu/home/draco/work/gosx-studio/contenteditorruntime/content_editor.js:493) · [Fallbacks](//wsl.localhost/Ubuntu/home/draco/work/gosx-studio/contenteditorruntime/content_editor.js:1961)

This limits direct touch dragging, not control accessibility: button and keyboard paths remain. No touch-drag test was run.

### F6e — P2 integration/parity risk: CMS and host expose an intentional runtime fork

**Confidence:** high for the byte/parity observation. **Precondition:** a path consumes CMS inline helpers and/or host-served assets.

- CMS embeds same-named assets (`cms/studio/assets.go:9–22,28–62`); hostruntime embeds/serves public assets (`hostruntime/runtime.go:41–42,215–230`). [CMS assets](//wsl.localhost/Ubuntu/home/draco/work/gosx-studio/cms/studio/assets.go:9) · [Host assets](//wsl.localhost/Ubuntu/home/draco/work/gosx-studio/hostruntime/runtime.go:215)
- Workbench bytes differ at this snapshot (`64098c81…42166` vs `71581450…cbc094`); command/state are identical. Existing parity checks cover state only (`modern_state_history_test.go:81–87`). [State parity](//wsl.localhost/Ubuntu/home/draco/work/gosx-studio/hostruntime/modern_state_history_test.go:81)
- Architecture records a deliberate independently-versioned fork with canonicalization deferred to v0.7. [Architecture](//wsl.localhost/Ubuntu/home/draco/work/gosx-studio/docs/ARCHITECTURE.md:296)

This is intentional fork debt and supported-path divergence risk, not proof of accidental duplicates to remove. **Required action:** assign compatibility/version ownership and add a parity or divergence contract.

### F6f — P3: standalone section-order asset has no rendered release version

**Confidence:** high for the HTML shape; no stale-cache execution proved. `workbench_page_canvas.go:184–187` emits unversioned `section-order-runtime.js`; main runtime URLs are versioned in `runtime_config.go:42–47`. [HTML](//wsl.localhost/Ubuntu/home/draco/work/gosx-studio/shell/workbench_page_canvas.go:184) · [Config](//wsl.localhost/Ubuntu/home/draco/work/gosx-studio/shell/runtime_config.go:42)

Stable ETag/no-cache behavior exists (`runtimeasset.go:23–35`), so this is a deterministic release-identity gap, not stale execution evidence. [Asset policy](//wsl.localhost/Ubuntu/home/draco/work/gosx-studio/internal/runtimeasset/runtimeasset.go:23)

### F7 — P2 deferred: media MIME metadata is available but dropped before runtime classification

**Confidence:** high for the traced DTO path. **Precondition:** affected pickers/renderers consume the runtime-inferred image flag.

- `cms/media.Asset.ContentType` exists and `shell/cmsbridge.go:49–61` carries it into the bridge DTO. Picker DTOs/renderers drop it in `shell/backend_editor_page.go` and backoffice `backend_{page,gallery,blog,product}_{index,detail}` plus `backend_settings`. [Asset](//wsl.localhost/Ubuntu/home/draco/work/gosx-studio/cms/media/media.go:26) · [Bridge](//wsl.localhost/Ubuntu/home/draco/work/gosx-studio/shell/cmsbridge.go:49) · [Editor picker](//wsl.localhost/Ubuntu/home/draco/work/gosx-studio/shell/backend_editor_page.go:183) · [Gallery](//wsl.localhost/Ubuntu/home/draco/work/gosx-studio/backoffice/backend_gallery_index.go:22)
- `mediaruntime/media_runtime.js:44–74` classifies `image` from extension. `UploadPolicy` declares content-type/extension maps (`media.go:97–100`), but `Allows(contentType, original, size)` relies on reported metadata and does not read bytes by signature (`:173`); host responsibility to validate actual bytes remains an integration obligation, and no upload bypass was proven. [Runtime](//wsl.localhost/Ubuntu/home/draco/work/gosx-studio/mediaruntime/media_runtime.js:44) · [Policy](//wsl.localhost/Ubuntu/home/draco/work/gosx-studio/cms/media/media.go:97)

**Required action:** defer the end-to-end contract `ContentType → data-media-content-type → runtime safe-URL/MIME classification → host producer integration`; unknown non-images remain files. No arbitrary URL-regex widening. No non-test DTO constructors were found in Studio; add a Noni adapter follow-up. Media is closed this pass with no changes.

### F8 — P2 conditional: file adapter can diverge memory from persistence and accepts permissive snapshot endings

**Confidence:** high for `cms/store/file`. **Precondition:** a host uses the file adapter and a write or reload failure occurs.

- Mutating methods update the memory store before `persistLocked` (`file.go:87–169`); a persistence error can leave the running process changed while returning an error. [Mutations](//wsl.localhost/Ubuntu/home/draco/work/gosx-studio/cms/store/file/file.go:87)
- Root corroborated file-store EOF and rollback behavior. `load` (`:201–217`) treats an existing empty/whitespace file as EOF/empty seed and decodes one JSON value without checking trailing data. A valid first value plus trailing data is accepted; partial nonempty JSON usually errors. This is not “all truncated JSON accepted.” [Loader](//wsl.localhost/Ubuntu/home/draco/work/gosx-studio/cms/store/file/file.go:201)

This is file-adapter scope only: no Noni SQLite data loss was observed and no live persistence failure was induced. **Required action:** specify failure/reload semantics and add adapter-only tests for write failure, empty/whitespace, valid JSON plus trailing data, and partial nonempty JSON.

### F9 — P2 deferred review: renderer URL policy depends on upstream validation/serialization

**Confidence:** medium static concern. `cms/render/render.go:53–73` passes trimmed `stringField` values directly to `img src` and button `href`. [Renderer](//wsl.localhost/Ubuntu/home/draco/work/gosx-studio/cms/render/render.go:53)

This is conditional on upstream validation and GoSX attribute serialization. It is not proof that an unsafe data scheme executes in a browser; no blanket unsafe-URL assertion is made. **Follow-up:** document allowed URL policy and verify producer validation plus serializer behavior with focused cases.

### F10 — P2 conditional integration follow-up: Review/MarkReady lack engine-level capability checks

**Confidence:** high for the interface shape; conditional on an authenticated host/caller reaching these methods without a separate guard. This is not a browser-engine finding and is not evidence of a publish bypass.

- `Engine.Review` (`engine.go:138+`) and `MarkReady` (`:164+`) persist without an engine-level capability guard, while Publish requires `CanPublish` (`:214+`). [Review](//wsl.localhost/Ubuntu/home/draco/work/gosx-studio/cms/lifecycle/engine/engine.go:138) · [Mark ready](//wsl.localhost/Ubuntu/home/draco/work/gosx-studio/cms/lifecycle/engine/engine.go:164) · [Publish guard](//wsl.localhost/Ubuntu/home/draco/work/gosx-studio/cms/lifecycle/engine/engine.go:214)

Host/caller guards may be the intended boundary; no unauthorized transition was observed. **Follow-up:** decide whether capability enforcement belongs in the engine or the authenticated host/caller boundary, fail closed, and add unauthorized Review/MarkReady plus allowed Publish regressions.

### F11 — P2 deferred review: authoring operation shape and style allowlist are separate contracts

**Confidence:** high for the separation. `AuthoringMutation` carries many operation kinds/fields, while style mutations use a curated property allowlist and value rejection. [Mutation shape](//wsl.localhost/Ubuntu/home/draco/work/gosx-studio/authoring/mutations.go:164) · [Style contract](//wsl.localhost/Ubuntu/home/draco/work/gosx-studio/authoring/style_mutations.go:30)

The code shows server-side style validation; static review does not establish that every host adapter preserves operation-shape validation and the style allowlist before persistence. This is not evidence of a bypass. **Follow-up:** verify custom adapters reject unsupported kinds/targets and arbitrary style properties/values; add cross-adapter contract coverage.

### F12 — Gallery polish remains WIP; geometry pass is not visual acceptance

**Confidence:** direct cached-Firefox geometry plus root visual review. **Scope:** Gallery polish only.

- Partial fix: direct cached-Firefox geometry checks were 3 pass / 1 fail. Widths 390, 1280, and 1600 passed; 320 with long/unbroken content failed with `scrollWidth=351`. Title/material-overlap and semantics checks passed.
- Root visual review of the 390 screenshot found the Work text wrapping to 2–4 characters per line in a narrow column, which is not acceptable polished mobile UX. Geometry pass is not visual acceptance: Gallery remains WIP at both 390 and 320 for readable layout, with 320 overflow confirmed. No polished-390 claim is made.
- The exact tracked Playwright suite is blocked before discovery because bundled `@playwright/test/cli.js` is absent and the TypeScript compiler is absent. Go/server/build, Chromium, WebKit, and deploy were not run. New Go and TS tests are unverified.
- Gallery CSS is 58-line source WIP and uncommitted; no new deploy occurred. Temporary config was removed and the browser was closed. The broken-image alt placeholder is a fixture limitation.

## Readiness gates

| Gate | State | Implication |
|---|---|---|
| Investigation coverage | Inventory-wide sampled review complete | UI/data static reviews and root corroborations incorporated |
| Exact Studio candidate identity | Blocked | F2; published Noni may be tested instead of PR |
| Named suites and engine matrix | Blocked/partial | F3 omissions; no automatic three-engine guarantee |
| Released consumer without overlays | Blocked | F4 lane absent |
| Current host instructions | Blocked | F5 stale pins |
| Standalone Room authorization | Conditional high risk | F1 topology and forged-authority tests required |
| BlockLayout pointer cancel/no-op | Open high risk | F6 regression gap |
| Sitemap pointer cancel cleanup | Open conditional | F6b no cancel branch |
| Runtime teardown | Deferred conditional | F6c retention risk, not measured leak |
| Media MIME/policy path | Deferred; closed no-change | F7 end-to-end contract absent |
| File-adapter failure/EOF semantics | Conditional | F8 file adapter only |
| Renderer URL policy | Deferred | F9 upstream/serializer contract |
| Review/MarkReady authorization boundary | Conditional | F10 authenticated host/caller versus engine guard |
| Gallery visual/readable-layout acceptance | Open WIP | F12; 320 overflow and 390 readability reject a polished claim |
| Owner walkthrough/acceptance | Pending required turnover acceptance | Not a prerequisite for completing this investigation |

## Verified, blocked, and historical evidence

**Verified or supplied:** clean starting snapshot; inventory/ownership map; inventory-wide sampled UI and data review; file-store EOF/rollback/persistence scope; typed host boundaries; conformance lifecycle projector; real CMS draft/publish/history; backup/deployment discipline; cached-Firefox geometry passes at 390/1280/1600; Gallery title/material-overlap and semantics checks; no observed Noni breach, live exploit, Noni SQLite data loss, email change, production change, auth change, or customer-data change.

**Blocked or not verified:** exact candidate identity in CI; released-module/no-overlay proof; fresh full-Go/full-browser validation; Linux Go/host execution due to WSL hang; tracked Playwright discovery because bundled `@playwright/test/cli.js` and TypeScript compiler are absent; Go/server/build, Chromium, WebKit, and deploy; new Go/TS tests; 320 overflow and readable 390 layout; live authentication/data-path validation; Review/MarkReady capability-boundary decision; required turnover acceptance. No fresh Linux tests or deployment were performed.

Historical release evidence is not fresh certification evidence: the prior editor-polish record reports three engines, 14 Noni checks, staging `d773`, and a verified backup. [Historical record](//wsl.localhost/Ubuntu/home/draco/work/gosx-studio/docs/superpowers/specs/2026-08-27-editor-polish-round2.md:282) No new live identity check was performed here.

Strengths include typed host boundaries, the conformance lifecycle projector, real CMS draft/publish/history behavior, and backup/deployment discipline. The deliberate CMS/host runtime fork remains documented debt, not an accidental duplicate finding.

## Ranked next actions for root

1. Address F12’s 320 overflow and narrow 390 readable-layout failure, then obtain visual acceptance rather than relying on geometry alone.
2. Close F1 with authenticated actor resolution, ignored client authority, fail-closed behavior, forged actor/capability regressions, and deployment-topology confirmation.
3. Close F2/F3 with exact candidate identity, named suite inclusion, and an automated three-engine contract.
4. Add the released-module/no-overlay lane and refresh host instructions with current versus historical pins.
5. Fix and regress F6/F6b pointer cancellation; define F6c teardown ownership and bounded replacement coverage.
6. Decide F6e/F6f runtime version/parity contracts and F10 Review/MarkReady capability-boundary ownership.
7. Complete F7’s optional MIME/safe-URL/producer path and the Noni adapter follow-up; verify F8 file-adapter failure semantics.
8. Verify F9 URL policy and F11 cross-adapter operation/style validation.
9. Complete the required turnover acceptance; it is not a prerequisite for completing this investigation artifact.
