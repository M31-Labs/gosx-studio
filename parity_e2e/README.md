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

```bash
cd ~/work/muddy-noni-commerce
PORT=3010 go run ./cmd/muddy-noni &

cd ~/work/gosx-studio/parity_e2e   # or the active worktree
npm install
npx playwright install chromium
npm run smoke          # harness boots both modes
npm test               # full parity suite
```

Override the base URL when running against a different host:

```bash
GOSX_STUDIO_PARITY_BASE_URL=http://localhost:8080 npm test
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
