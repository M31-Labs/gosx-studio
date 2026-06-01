# gosx-studio parity test harness

Reusable Playwright harness for Phase 3 of the gosx-studio runtime burn-down.
Shared by all seven slices that replace JavaScript runtime contracts with
`.gsx`-authored islands and engines.

See:

- `~/.hyphae/spaces/m31labs-gosx/plans/gosx-vm-unification-and-editor-bridge-burn-down.md`
- `~/.hyphae/spaces/m31labs-gosx/specs/gosx-studio-runtime-parity-matrix.md`
- `~/.hyphae/spaces/m31labs-gosx/plans/2026-05-25-phase-3-slice-1-fieldruntime.md`

## How the harness works

Each Phase 3 slice ships its runtime contract additively — the legacy JS
implementation in `assets/studio-engines.js` keeps running, and the new
`.gsx`-authored island ships beside it. A bridge shim picks which path
runs based on a feature flag declared in `studio.ShellConfig.FeatureFlags`
(`field-runtime-islands`, `selection-runtime-islands`, etc.).

The consumer (muddy-noni-commerce editor route) honors a URL parameter
`?gosx-studio-parity=baseline|candidate` that forces the flag off (legacy
JS) or on (island path) for that request. The harness drives the same
process in both modes through that parameter; parity assertions compare
observable state (DOM after-state, signal store contents, dispatched
events, clipboard contents) between the two boots.

This avoids the original plan's two-server-process design — both impls
already coexist in one process by Section C of every slice. See
`harness.ts` header comment for the trade-off and
`~/.hyphae/spaces/m31labs-gosx/lessons/` for the design note.

## Running

Prerequisites:

- Node 20+
- Chromium installed via `npx playwright install chromium`
- A running muddy-noni-commerce dev server on the URL below (or override
  via `GOSX_STUDIO_PARITY_BASE_URL`).
- **An authenticated session for the editor route, OR the auth-bypass env
  var.** `/admin/editor` redirects unauthenticated requests to `/login`,
  which makes the parity harness fail at the canvas-attached wait. Pick one:

  - **For CI and one-shot runs (recommended)** — boot muddy-noni with
    `MUDDY_MOCK_AUTH=1`. This bypasses the `/admin/*` auth guard for the
    lifetime of that process only (mirrors `MUDDY_MOCK_CHECKOUT=1`). See
    `cmd/muddy-noni/main.go` (`adminSessionMiddleware`) and
    `.github/workflows/parity.yml` in muddy-noni-commerce. DO NOT enable
    in production deployments.
  - **For interactive debugging against a real auth flow** — configure
    `ADMIN_EMAILS=<your-email>@<domain>` and sign in via Google OAuth,
    or thread a pre-issued session cookie into the Playwright context
    (see Playwright's `storageState`).
- **The gosx-studio module bump must include the fieldruntime island
  bundle.** Until muddy-noni-commerce's `go.mod` pins a gosx-studio
  version that contains `m31labs.dev/gosx-studio/fieldruntime` (or a
  replace directive routes the import there), the candidate mode boots
  the same legacy bundle as baseline. The `@coverage` test in
  `fieldruntime_test.ts` detects this and marks itself skipped with a
  descriptive reason so CI doesn't ship false-positive parity claims.

```bash
cd ~/work/muddy-noni-commerce
MUDDY_MOCK_AUTH=1 PORT=3010 go run ./cmd/muddy-noni &

cd ~/work/gosx-studio/parity_e2e   # or the active worktree
npm install
npx playwright install chromium
npm run smoke          # harness boots both modes
npm test               # full parity suite
```

## CI integration

Authoritative CI runner: `.github/workflows/parity.yml` in
`muddy-noni-commerce`. The workflow:

1. Checks out `muddy-noni-commerce` (the consumer) and `gosx-studio` at
   the tag this README ships with.
2. Installs the `gosx` CLI (`go install m31labs.dev/gosx/cmd/gosx@<tag>`)
   and runs `gosx build .` to populate `dist/` (where the WASM + bootstrap
   assets land).
3. Boots muddy-noni in the background with `MUDDY_MOCK_AUTH=1` +
   `MUDDY_MOCK_CHECKOUT=1` and waits for `/admin/editor` to return 200.
4. Runs the parity suite from this directory pointed at
   `http://localhost:3010` via `GOSX_STUDIO_PARITY_BASE_URL`.
5. Captures the muddy-noni log + playwright report as build artifacts on
   failure.

If you add a new parity test that needs additional muddy-noni env vars or
a different boot path, edit both the workflow and the **Prerequisites**
list above so interactive runs stay in sync with CI.

Override the base URL when running against a different host:

```bash
GOSX_STUDIO_PARITY_BASE_URL=http://localhost:8080 npm test
```

## Reference App Authoring E2E

`reference_apps_authoring_test.ts` is an opt-in browser workflow for the two
current reference apps. It boots Muddy/Noni and Pajaritos from sibling worktrees
with temporary data directories, opens their real admin editors, verifies the
visible authoring buttons receive pointer events, and submits create-page,
add-section, duplicate-section, and save-field actions through the real GoSX
action/CSRF flow. Muddy assertions include browser-reloaded duplicate and field
state. Pajaritos assertions now verify native create-page, add-section,
duplicate, and save-field payloads, 303 redirects, browser-visible success
feedback, persisted page/section reflection, staging/production readiness
copy, preview-refresh instrumentation, and a visible restore-point rollback
after the editor reload.

It now runs in default `gosx-studio` CI via
`.github/workflows/reference-apps.yml`, which checks out the sibling repos in
the same layout used locally. Run it locally with:

```bash
npm run test:reference-apps
```

Override sibling locations if needed:

```bash
GOSX_STUDIO_REFERENCE_APP_E2E=1 \
GOSX_STUDIO_MUDDY_REPO=~/work/muddy-noni-commerce \
GOSX_STUDIO_PAJARITOS_REPO=~/work/pajaritos-forest-school \
  npx playwright test reference_apps_authoring_test.ts
```

## File layout

| File | Purpose |
|------|---------|
| `harness.ts` | Shared boot + assert helpers. Used by every per-contract test file. Cap at ~200 lines. |
| `playwright.config.ts` | Playwright config (chromium, serial, base URL). |
| `smoke_test.ts` | Smoke test for the harness itself. Tag `@smoke`. |
| `<contract>_test.ts` | Per-contract parity suite (one per Phase 3 slice). |
| `tsconfig.json` | TypeScript config for tooling. Tests do not emit. |

## Adding a parity suite for a new contract

1. Add a feature flag (`<contract>-runtime-islands`) to
   `studio.ShellConfig.FeatureFlags` and to the consumer's URL-param
   handling.
2. Ship the island implementation additively (Section B + C of the
   slice plan).
3. Create `<contract>_test.ts` next to this README. Import
   `bootBaseline` / `bootCandidate` / `disposeBoot` from `./harness`.
4. For each contract method, write a test that drives the same user
   interaction in both boots and asserts observable equivalence.
5. Wire the suite into CI as a required check on PRs touching the
   contract's files.
6. After 7 days of CI green on the candidate path, delete the JS
   implementation (Section E of the slice plan).
