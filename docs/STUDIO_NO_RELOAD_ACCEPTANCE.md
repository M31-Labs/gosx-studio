# Studio No-Reload Acceptance

Date: 2026-09-05

## Current Verified Slice

This slice verifies the shared managed authoring path for JSON responses and fragment refreshes. Managed authoring forms now prevent the browser default submit for every marked control before method, target, or action checks can trigger a main-frame navigation or popup. Same-origin `POST` submits stay on `fetch` with multipart form data, `X-Requested-With`, `X-CSRF-Token`, submitter data, and same-origin credentials. Same-origin `GET` submits refresh the preview iframe URL instead of navigating the editor document. Unsupported methods or unsafe actions stay in-page and move the form to an error state. A delayed save response cannot erase text typed after submit: fragment replacement restores post-submit field edits, leaves the form dirty, and guards delayed success timers from overwriting the unsaved state.

JSON validation and conflict-style failures now remain visible in the editor. Non-2xx JSON responses set form/workbench error state, preserve status and message attributes, mark named fields with `aria-invalid` and `data-gosx-studio-authoring-field-error`, update matching error text nodes, emit `gosxstudio:authoring-error`, and clear the original pending baseline. Successful fragment refreshes remount the editor runtimes that can be replaced by returned HTML: preview, workbench, block layout, operation, content editor, sitemap, media, style, and selection. Focus is restored after fragment replacement when the refreshed document still contains the same id, authoring binding, studio field, or name.

Operation runtime saves now queue per effective target. The queue key includes explicit route, page, field, component, control, style property, breakpoint, and state values, so two different targets can commit independently while same-target operations cannot race their cursor updates. Same-target queued payloads are built after the previous request settles, so the second POST uses the target head and document revision returned by the first. If the predecessor fails, already queued same-target edits are rejected without another network mutation; the queue is still cleaned so a later explicit retry can proceed. Document revision updates are monotonic, so an older out-of-order distinct-target response cannot downgrade the form's next expected revision. Structured non-2xx operation responses are parsed before the runtime throws; the error event carries status/body detail and safe returned cursors are recovered for retry.

Verified commands in this slice:

- `GOWORK=off go test ./authoringruntime ./operationruntime`
- `PATH=/home/draco/.cache/codex-tools/node-v22.23.1/bin:$PATH npx playwright test authoringruntime_test.ts operationruntime_test.ts` — 12/12 Chromium
- `GOWORK=off go test ./authoringruntime ./operationruntime ./hostruntime`
- `PATH=/home/draco/.cache/codex-tools/node-v22.23.1/bin:$PATH npx tsc --noEmit`
- `git diff --check`
- `GOWORK=off go test ./...`

The prior 369/375 matrix is historical scratch evidence and was not freshly run for this document.


## STUDIO-HOST-02 Real Host Lifecycle Evidence

This follow-up verifies one concrete reference-host lifecycle workflow against the real Pajaritos admin editor and handlers, using mock authentication and temporary reference-app data. Pajaritos lifecycle actions now return the managed Studio JSON authoring payload instead of redirect-shaped success for JSON requests. The state runtime bridges that payload into the existing authoring fragment/preview refresh path, so approve, schedule, process-due publish, and restore complete without navigating the editor document.

The verified Chromium journey covers: structured 422 schedule validation with preserved status, error text, and `publishAt` field errors; durable direct-edit save to draft; approve and schedule with refreshed lifecycle fragments and no preview refresh; public storefront remaining unchanged after approve and schedule; process-due publish refreshing fragments and preview; a second draft/publish proving V1/V2 live separation; restore to the restore checkpoint whose snapshot contains the V1 headline; visible direct-edit control and preview frame returning to V1 before any manual reload; public storefront returning to V1; and a final explicit reload proving persistence.

The Pajaritos host slice also adds focused server-side coverage for the route boundaries behind that browser proof: admin action CSRF protection is enforced only for `/admin/.../__actions/` posts, public-family/contact action routes are left outside the new shim, local mock auth seeds an admin session only under `PAJARITOS_MOCK_AUTH=1`, approve/schedule/process publish require publish capability at the handler/engine boundary, restore continues through the existing engine `CanRestore` authority check, lifecycle JSON responses include the fragment selectors the runtime consumes, and restore points expose captured headline identity so tests can choose a checkpoint by durable content rather than brittle row order.

Verified STUDIO-HOST-02 commands:

- `PATH=/home/draco/.cache/codex-tools/node-v22.23.1/bin:$PATH npx tsc --noEmit` in `parity_e2e`
- `GOWORK=off go test ./hostruntime ./cms/studio` in `gosx-studio`
- `/home/draco/.cache/codex-tools/node-v22.23.1/bin/node --check hostruntime/assets/state_runtime.js`
- `/home/draco/.cache/codex-tools/node-v22.23.1/bin/node --check cms/studio/assets/state_runtime.js`
- `GOWORK=off go test ./app/admin/editor ./internal/site` in `pajaritos-forest-school`
- `GOSX_STUDIO_REFERENCE_APP_E2E=1 GOSX_STUDIO_CANDIDATE_REPO=/home/draco/work/gosx-studio GOSX_STUDIO_CANDIDATE_SHA=219989a588ee5a5443eb1d7f0c44dbdcf1877b04 npx playwright test reference_apps_lifecycle_no_reload_test.ts --project=chromium --workers=1` — 1/1 Chromium, real Pajaritos host
- `git diff --check` in both repositories

This is not yet a three-browser real-host lifecycle gate. The prior 36/36 Chromium/Firefox/WebKit result belongs to route-fixture no-reload coverage, not this Pajaritos real-host lifecycle journey.


## STUDIO-HOST-03 Compatibility and Preview Timing Evidence

This compatibility follow-up keeps the Pajaritos lifecycle proof on the current GoSX document contract instead of replacing it with a host-owned HTML shell. The canonical Pajaritos module now pins `m31labs.dev/gosx v0.53.8`, so `cmd/pajaritos` compiles with `server.HTMLDocument(ctx.Document(title, body))` and retains framework-managed document metadata/head behavior. The earlier concern that `app/admin/editor/page.css` was missing from the route was reclassified after inspection: GoSX attaches the sidecar stylesheet inline and scoped as `data-gosx-file-css="page.css"`, which the editor mode browser proof exercises through compact default layout and reachable Advanced/Look panels.

The workbench preview runtime now treats an iframe document without `head`, `documentElement`, or `body` as not-yet-bindable. It appends preview patch styles only after a valid host element exists, and it does not set `frame.__gosxStudioPreviewDocument` until style injection succeeds. That preserves same-document retry on the later iframe load event and avoids the Windows timing error where `(doc.head || doc.documentElement).appendChild(style)` could dereference `null`.

Verified STUDIO-HOST-03 commands:

- `GOWORK=off go test ./cmd/pajaritos ./app/admin/editor ./internal/site` in `pajaritos-forest-school`
- `GOWORK=off go test ./workbenchruntime ./hostruntime ./cms/studio` in `gosx-studio`
- `PATH=/home/draco/.cache/codex-tools/node-v22.23.1/bin:$PATH npx tsc --noEmit` in `parity_e2e`
- `/home/draco/.cache/codex-tools/node-v22.23.1/bin/node --check cms/studio/assets/workbench_runtime.js`
- `GOSX_STUDIO_REFERENCE_APP_E2E=1 GOSX_STUDIO_CANDIDATE_REPO=/home/draco/work/gosx-studio GOSX_STUDIO_CANDIDATE_SHA=219989a588ee5a5443eb1d7f0c44dbdcf1877b04 GOSX_STUDIO_PAJARITOS_REPO=/home/draco/work/pajaritos-forest-school npx playwright test reference_apps_lifecycle_no_reload_test.ts reference_apps_editor_modes_no_reload_test.ts --project=chromium --workers=1` — 2/2 Chromium, real Pajaritos host candidate path


## STUDIO-UI-05 Pajaritos Editor Presentation Evidence

This UI follow-up verifies the Pajaritos editor presentation after the host compatibility and preview-timing fixes. The host-scoped editor stylesheet now keeps the default editor compact, gives the primary preview iframe a usable editing viewport, styles the mode and home-layer controls as Paper & Ink pills instead of native browser buttons, and makes the Properties scope strip readable with visible separators. The supplemental controls remain mode-gated: Advanced reveals site tools, Look reveals component style controls, and returning to Home/Edit hides both supplemental panels without leaving focus inside hidden content.

The final isolated Pajaritos preview was rebuilt from the current HOST03 source plus UI05 stylesheet changes on WSL port 4322, preserving the temporary preview data. Windows Chrome verified `http://studio.localhost:4323/admin/editor` with no page errors, no captured bad network responses, no desktop/mobile horizontal overflow, scoped inline `page.css`, and loaded preview content. Desktop default measured `scrollHeight=1321` with a `774.84px × 580px` preview iframe whose document was complete and contained the hero copy. Mobile default measured `scrollHeight=2310` with a `307.66px × 523.27px` preview iframe and the same loaded hero content.

Verified STUDIO-UI-05 commands and artifacts:

- `GOWORK=off go test ./app/admin/editor` in `pajaritos-forest-school`
- `GOSX_STUDIO_REFERENCE_APP_E2E=1 GOSX_STUDIO_CANDIDATE_REPO=/home/draco/work/gosx-studio GOSX_STUDIO_PAJARITOS_REPO=/home/draco/work/pajaritos-forest-school npx playwright test reference_apps_lifecycle_no_reload_test.ts reference_apps_editor_modes_no_reload_test.ts --project=chromium --workers=1 --reporter=line` — 2/2 Chromium
- `rg --pcre2 -n "#[0-9a-fA-F]{3,8}|rgba\(|hsla?\(|font-family: (?!var\()|font-size: (?!var\()|!important" /home/draco/work/pajaritos-forest-school/app/admin/editor/page.css` — no matches
- Windows Chrome proof JSON: `/mnt/c/Users/odvce/.codex/visualizations/2026/09/05/01a0705a-0ed2-73b2-8dec-48736fc4cb12/studio-ui-05-windows-proof.json`
- Scratch report: `/home/draco/work/gosx-studio/.tiller/scratch/codex/studio-ui-05-report.md`

Hero layer-chip selection was still pending at the end of UI05 and was not counted as passed by the UI05 mode/presentation test. The original trigger was that clicking `[data-studio-home-layer-pick="hero"]` left `aria-pressed="false"`, the workbench form `data-studio-selection` unset, and visible selection labels at `No selection`. That pending item is closed by STUDIO-SELECT-01 and the final STUDIO-REVIEW-06 preview proof: clicking Hero now sets `data-studio-selection="hero"`, `data-studio-selection-kind="block"`, updates visible selection labels to `Hero`, and keeps the editor document continuous.


## STUDIO-SELECT-01 Hero Layer Selection Evidence

This final bounded selection fix closes the Pajaritos Hero layer-chip mismatch reported after UI05. The shared home-layer renderer now emits each layer chip as the block selection row expected by the existing selection runtime, using `data-block-studio-block` and `data-studio-block-label` on the same button that carries `data-studio-home-layer-pick`. No host-specific selection runtime branch was added.

The focused Chromium proof now clicks Programs first, verifies the shared workbench selection moves to `programs`, then clicks Hero and verifies the workbench selection becomes `hero`, selection kind is `block`, the Hero chip becomes the selected row, selection readouts update from the Hero label, the Properties inspector for Hero is visible, the preview iframe exposes the matching Hero block identity, and the editor document does not reload. The existing Pajaritos lifecycle no-reload test remains green in the same browser run.

Verified STUDIO-SELECT-01 commands:

- `GOWORK=off go test ./panels ./selectionruntime ./shell` in `gosx-studio`
- `GOWORK=off go test ./cmd/pajaritos ./app/admin/editor ./internal/site` in `pajaritos-forest-school`
- `PATH=/home/draco/.cache/codex-tools/node-v22.23.1/bin:$PATH npx tsc --noEmit` in `parity_e2e`
- `GOSX_STUDIO_REFERENCE_APP_E2E=1 GOSX_STUDIO_CANDIDATE_REPO=/home/draco/work/gosx-studio GOSX_STUDIO_CANDIDATE_SHA=219989a588ee5a5443eb1d7f0c44dbdcf1877b04 GOSX_STUDIO_PAJARITOS_REPO=/home/draco/work/pajaritos-forest-school npx playwright test reference_apps_editor_modes_no_reload_test.ts reference_apps_lifecycle_no_reload_test.ts --project=chromium --workers=1` — 2/2 Chromium, real Pajaritos host candidate path
- `git diff --check` in both repositories

## Gates Still Required

No universal Studio no-reload claim is made yet. The remaining acceptance gates are:

- All control pages prove no main-frame document navigation on normal save, validation failure, conflict, and recovery.
- JSON validation and recovery are verified against real host handlers, including 422 field errors and 409 conflict details.
- Save concurrency is covered for repeated same-target saves, distinct-target saves, stale responses, failed latest response, and recovery after error.
- Draft, publish, and restore coverage expands from the verified Pajaritos reference-host path to the remaining real hosts and host-specific lifecycle surfaces; media upload, navigation, and commerce actions adopt the same managed no-reload contract or are explicitly classified as intentional navigations.
- Temporary-data reference hosts cover mock-auth flows without depending on owner production data, including at least one host beyond Pajaritos.
- The real-host lifecycle no-reload suite passes in Chromium, Firefox, and WebKit.
- Actual owner login, owner walkthrough, and physical touch acceptance are recorded separately.
- The first-run site wizard remains missing and is outside this verified slice.

## Next Implementation Order

1. Stabilize the Pajaritos source-ready preview with the UI containment work from STUDIO-UI-03, then run the same lifecycle journey through the user review server.
2. Promote the Pajaritos lifecycle browser proof to Chromium, Firefox, and WebKit, using the shared document-continuity helper and the real host handlers.
3. Add a second temporary-data reference host so lifecycle no-reload behavior is not proven only by Pajaritos.
4. Extend the managed host adapter to media upload, navigation, and commerce actions, keeping intentional external/public-preview navigations explicit.
5. Add all-control main-frame navigation assertions across reference pages and real handlers, including 422, 409, retry-after-error, and stale response recovery.
6. Build the first-run site wizard product milestone: create site from template, brand/theme, pages/navigation, content/media, responsive preview/readiness, and publish review. Acceptance evidence must show resumable draft progress, explicit host ownership and permissions, validation errors, back/forward behavior, and no document reload through the wizard flow.
7. Run actual owner login/touch acceptance after the technical gates are green.
