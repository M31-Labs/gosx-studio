# Pillar 5 — Extraction Completion Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Move ~12,500 lines of editor code from `gosx-cms/studio/` and ~3,050 lines of `.gsx` components from `muddy-noni-commerce` into `gosx-studio`, restoring the architectural boundary so CMS never imports Studio.

**Architecture:** 12 sequential batches by editor surface (chrome → cleanup). Each batch is internally consistent — no shim, no temporary type aliases. After each batch: all four `go test` suites (`gosx-studio`, `gosx-cms`, `muddy-noni-commerce`, `pajaritos-forest-school`) pass. Each batch produces one `buckley commit --yes --minimal-output` per host repo touched.

**Tech Stack:** Go 1.24, `m31labs.dev/gosx-studio`, `m31labs.dev/gosx-cms`, `m31labs.dev/gosx` (`.gsx` components), `buckley` for commits, `hypha` for impact analysis between batches.

**Spec:** `gosx-studio/docs/superpowers/specs/2026-05-28-pillar-5-extraction-design.md`

---

## File Structure Decisions

### Destination layout in `gosx-studio`

```
gosx-studio/
├── studio.go              # existing — all contract types
├── shell.go               # existing — ShellConfig + path constants
├── runtime.go             # existing — runtime asset handlers
├── store.go               # NEW (batch 1) — Store interface from spec §4
├── chrome.go              # NEW (batch 2) — RenderStudioToolbar, RenderModebar, RenderSaveStatus, RenderHistoryControls
├── chrome.gsx             # NEW (batch 2) — StudioWorkbenchShell, StudioToolbar, StudioModebar, StudioSaveStatus, StudioHistoryControls, StudioCommandPalette
├── chrome_test.go         # NEW (batch 2)
├── canvas.go              # NEW (batch 3)
├── canvas.gsx             # NEW (batch 3)
├── canvas_test.go         # NEW (batch 3)
├── ...one file pair per surface (batches 4-11)...
├── *.go from cmsstudio    # arrives via batches; flat package
├── *.gsx from page.gsx    # arrives via batches; flat package
├── blocklayoutruntime/    # existing — kept as sub-package per spec §1
├── brandruntime/
├── fieldruntime/
├── previewruntime/
├── selectionruntime/
├── styleruntime/
├── workbenchruntime/
├── plugins/showcase3d/    # existing
└── assets/studio.css      # existing
```

### What changes in each host repo (per batch)

- **muddy-noni-commerce**:
  - `app/admin/editor/page.gsx` — components removed as their surface moves
  - `app/admin/editor/page.server.go` — `cmsstudio.X` references flipped to `studio.X` per surface
  - `app/admin/editor/page_server_test.go` — same flips for assertions
  - `go.mod` — already has `gosx-studio` dep; no add needed
- **pajaritos-forest-school**:
  - `go.mod` — bumped to `m31labs.dev/*` paths in batch 1; `gosx-studio` added
  - `internal/site/store.go` and `store_test.go` — `cmsstudio.X` → `studio.X` for surface moves
  - `internal/site/lifecycle.go` and `lifecycle_test.go` — same flips for batches 9, 10
  - `app/admin/editor/page.server.go` and `page_server_test.go` — same flips for batches 9, 11
- **gosx-cms**:
  - `studio/<files>.go` removed as they move out
  - `lifecycle/`, `health/`, `publish/` — receive content-shaped types relocated within CMS (batches 9, 10)
  - `flows/`, `style/` — `Studio*` prefix dropped from type names (batches 9, 11)

---

## The Batch Recipe

Every batch after batch 1 follows this recipe. Batches 2 and 12 spell out every step; batches 3-11 invoke the recipe by reference, supplying only their surface-specific manifest.

**Inputs the recipe expects:**
- `SURFACE_NAME` — the editor surface (e.g., "chrome", "canvas", "look")
- `CMSSTUDIO_FILES` — list of files under `gosx-cms/studio/` that move (with their `_test.go` pairs)
- `GSX_COMPONENTS` — list of `func XxxYyy() Node` definitions in Noni's `page.gsx` that move
- `NONI_TOUCH_FILES` — Noni files where `cmsstudio.X` → `studio.X` flips happen
- `PAJARITOS_TOUCH_FILES` — Pajaritos files where same flips happen (often empty for non-flow/non-publish batches)
- `STUDIO_TARGET_FILE` — single `.go` file in `gosx-studio` where the moved functions land (one per surface)
- `STUDIO_GSX_TARGET_FILE` — single `.gsx` file where moved components land

**Recipe steps (every batch, in order):**

1. **Open a trace** for the batch:
   ```bash
   hypha trace start --agent "identity://claude-extraction" --task "pillar-5-batch-<N>-<surface>" --phase "extract <SURFACE_NAME>"
   ```
2. **Verify clean working tree** in all three repos:
   ```bash
   git -C /home/draco/work/gosx-studio status --porcelain
   git -C /home/draco/work/gosx-cms status --porcelain
   git -C /home/draco/work/muddy-noni-commerce status --porcelain
   git -C /home/draco/work/pajaritos-forest-school status --porcelain
   ```
   Expected: all empty. If not, abort and resolve.
3. **Run baseline test on all four suites** to confirm green starting state:
   ```bash
   (cd /home/draco/work/gosx-studio && go test ./... 2>&1 | tail -5)
   (cd /home/draco/work/gosx-cms && go test ./... 2>&1 | tail -5)
   (cd /home/draco/work/muddy-noni-commerce && go test ./... 2>&1 | tail -5)
   (cd /home/draco/work/pajaritos-forest-school && go test ./... 2>&1 | tail -5)
   ```
   Expected: all four end with `ok` lines and no `FAIL`. If any fail, abort and resolve.
4. **Move Go files** for the surface from `gosx-cms/studio/<file>.go` to `gosx-studio/<STUDIO_TARGET_FILE>`. Use `git mv` per file so history is preserved:
   ```bash
   git -C /home/draco/work/gosx-cms mv studio/<file>.go ../gosx-studio/<STUDIO_TARGET_FILE>
   ```
   For each moved file, change the package declaration from `package studio` (it already says this — both packages are named `studio`) — no change needed for the package line itself.
5. **Move `.gsx` components** from Noni's `app/admin/editor/page.gsx` to `gosx-studio/<STUDIO_GSX_TARGET_FILE>`. This is a manual edit (not `git mv`) because we're splitting one source file. Cut the `func XxxYyy() Node { ... }` block from `page.gsx`; paste into the new `.gsx` file. The destination file must start with `package studio` and use the same `Node` and `data.*`/`actions.*` references — verify the destination compiles.
6. **Update imports in Noni**:
   - In each `NONI_TOUCH_FILES` entry, change `cmsstudio.RenderXxx` to `studio.RenderXxx` for the surface's symbols
   - If the file imported `cmsstudio "m31labs.dev/gosx-cms/studio"` but Noni already imports `"m31labs.dev/gosx-studio"` somewhere else in the same file, just rename the alias usage
   - If Noni doesn't already import `gosx-studio` in that file, add the import
   - Run `goimports -w` on each touched file:
     ```bash
     goimports -w /home/draco/work/muddy-noni-commerce/<file>
     ```
7. **Update imports in Pajaritos** (same flip):
   - Same pattern as Noni for each `PAJARITOS_TOUCH_FILES` entry
8. **Run tests after the move**, in dependency order:
   ```bash
   (cd /home/draco/work/gosx-studio && go build ./... && go test ./...)
   (cd /home/draco/work/gosx-cms && go build ./... && go test ./...)
   (cd /home/draco/work/muddy-noni-commerce && go build ./... && go test ./...)
   (cd /home/draco/work/pajaritos-forest-school && go build ./... && go test ./...)
   ```
   Expected: each ends with `ok`-pattern output and no `FAIL`. If any fails, do NOT commit — fix and re-run. Do not move to step 9 until all four are green.
9. **Commit each repo** with buckley:
   ```bash
   cd /home/draco/work/gosx-studio && buckley commit --yes --minimal-output
   cd /home/draco/work/gosx-cms && buckley commit --yes --minimal-output
   cd /home/draco/work/muddy-noni-commerce && buckley commit --yes --minimal-output
   cd /home/draco/work/pajaritos-forest-school && buckley commit --yes --minimal-output
   ```
   Note: never `-graft`, never `--skip-entities` per global commit rules. The cms/studio commit only happens if files moved out of `gosx-cms/studio/` — skip the cms commit if that batch only touched gosx-studio (rare).
10. **Tick the trace**:
    ```bash
    hypha trace tick <trace-id> "batch <N> <surface> green across all four suites"
    ```
11. **Close the trace** when batch is done:
    ```bash
    hypha trace done <trace-id> --status succeeded
    ```

That's the recipe. Now batch by batch.

---

## Batch 1: Prep — Bump Pajaritos to `m31labs.dev/*` paths

**Why this is batch 1 (not 0):** Pajaritos is on `github.com/odvcencio/gosx-cms v0.1.56` (the old path) and `gosx-cms v0.1.56` (a stale version). Batches 2+ rename CMS types and add new gosx-studio APIs that don't exist on v0.1.56. Pajaritos must be on the same module paths and recent versions before any other batch touches it.

This batch only touches pajaritos. No file moves yet.

**Files:**
- Modify: `pajaritos-forest-school/go.mod`
- Modify: 16 `.go` files in `pajaritos-forest-school` (per inventory: `app/contact/page.server.go`, `cmd/pajaritos/main.go`, `internal/site/lifecycle.go`, `app/admin/editor/page_server_test.go`, `app/admin/editor/page.server.go`, `internal/site/lifecycle_test.go`, `internal/site/store_test.go`, `internal/site/store.go`, plus 8 more discovered by full sweep)

### Steps

- [ ] **Step 1: Open the trace**

```bash
hypha trace start --agent "identity://claude-extraction" --task "pillar-5-batch-1-prep" --phase "bump pajaritos module paths"
```

- [ ] **Step 2: Baseline test (sanity)**

```bash
cd /home/draco/work/pajaritos-forest-school && go test ./... 2>&1 | tail -5
```
Expected: `ok` lines for all packages; no `FAIL`. Capture the count of passing packages — Step 14 confirms the same count.

- [ ] **Step 3: Enumerate the exact file list**

```bash
grep -rln "github.com/odvcencio/gosx" /home/draco/work/pajaritos-forest-school --include='*.go' | sort | tee /tmp/pajaritos-bump-files.txt
wc -l /tmp/pajaritos-bump-files.txt
```
Expected: `16` lines.

- [ ] **Step 4: Update go.mod — replace old module paths**

Edit `/home/draco/work/pajaritos-forest-school/go.mod`. Replace these three `require` lines:

```
github.com/odvcencio/gosx v0.18.28
github.com/odvcencio/gosx-admin v0.1.2
github.com/odvcencio/gosx-cms v0.1.56
```

with the current m31labs versions:

```
m31labs.dev/gosx v0.23.1
m31labs.dev/gosx-admin v0.2.0
m31labs.dev/gosx-cms v0.2.0
m31labs.dev/gosx-studio v0.5.2
```

Note: `gosx-studio` is added now even though pajaritos doesn't import it yet — batch 2 will add the first import.

- [ ] **Step 5: Sed-replace import paths across all 16 files**

```bash
for f in $(cat /tmp/pajaritos-bump-files.txt); do
  sed -i \
    -e 's|github.com/odvcencio/gosx-admin|m31labs.dev/gosx-admin|g' \
    -e 's|github.com/odvcencio/gosx-cms|m31labs.dev/gosx-cms|g' \
    -e 's|"github.com/odvcencio/gosx"|"m31labs.dev/gosx"|g' \
    -e 's|github.com/odvcencio/gosx/|m31labs.dev/gosx/|g' \
    "$f"
done
```

Order matters: replace `gosx-cms` and `gosx-admin` BEFORE the bare `gosx` rule so the longer match wins.

- [ ] **Step 6: Verify no leftover old paths remain**

```bash
grep -rn "github.com/odvcencio/gosx" /home/draco/work/pajaritos-forest-school --include='*.go'
```
Expected: empty output (zero matches).

- [ ] **Step 7: go mod tidy**

```bash
cd /home/draco/work/pajaritos-forest-school && go mod tidy
```
Expected: completes silently or with download messages. No errors.

- [ ] **Step 8: Build**

```bash
cd /home/draco/work/pajaritos-forest-school && go build ./...
```
Expected: silent success. If any "undefined" errors appear, a cmsstudio API may have changed shape between v0.1.56 and v0.2.0 — those need per-call-site fixes before this batch can land. If errors are extensive, abort and surface to the user before forcing through.

- [ ] **Step 9: Run tests**

```bash
cd /home/draco/work/pajaritos-forest-school && go test ./... 2>&1 | tail -20
```
Expected: all `ok` lines, no `FAIL`, same package count as Step 2.

- [ ] **Step 10: Sanity-check the diff is just path swaps**

```bash
cd /home/draco/work/pajaritos-forest-school && git diff --stat
git diff -- '*.go' | head -50
```
Expected: each diff line is an import-path swap. No semantic code changes. If any non-import lines appear in the diff, investigate — those came from go mod tidy fixing something interesting.

- [ ] **Step 11: Commit pajaritos**

```bash
cd /home/draco/work/pajaritos-forest-school && buckley commit --yes --minimal-output
```
Buckley should compose a commit message referencing the path migration. Do not pass `-m`, `-graft`, or `--skip-entities`.

- [ ] **Step 12: Tick the trace**

```bash
hypha trace tick <trace-id> "pajaritos on m31labs.dev/* paths; gosx-studio dep added; tests green"
```

- [ ] **Step 13: Close the trace**

```bash
hypha trace done <trace-id> --status succeeded
```

- [ ] **Step 14: Verify baseline still holds**

```bash
cd /home/draco/work/pajaritos-forest-school && go test ./... 2>&1 | grep -c "^ok"
```
Expected: same count as Step 2. If lower, something silently regressed — bisect.

**Batch 1 success criteria:**
- Pajaritos `go.mod` uses `m31labs.dev/*` paths and includes `gosx-studio v0.5.2`
- Zero `github.com/odvcencio/gosx` references remain in pajaritos source
- `go test ./...` passes with same package count as before
- One buckley commit on pajaritos

---

## Batch 1.5: Foundation (added 2026-05-29 after batch 2 surfaced the plan gap)

**Why this exists:** Batch 2 attempted to move chrome alone and discovered the chrome renderers depend on a shared cmsstudio foundation (`Shell`, `Options`, `View`, `Workbench`, `Mode`, `Metric`, `Viewport`, `Panel`, `Action`) plus helpers (`firstNonEmpty`, `boolAttr`, `fmtAny`, `normalizeModes`, etc.) that 22+ other cmsstudio files also use. The plan's appendix correctly identified `studio.go` as belonging in gosx-studio but didn't schedule it as its own batch. This batch fills that gap before any surface lifts.

**Why before batch 2:** With the foundation in gosx-studio, every surface batch (2-11) becomes a true "lift X surface" operation without dependency chasing. Without this batch first, every surface batch hits the same wall.

### Surface manifest

| What | Source | Target |
|---|---|---|
| Foundation types: `Shell`, `Options`, `View`, `Workbench`, `Mode`, `Metric`, `Viewport`, `Panel`, `Action`, `Section`, `ModeConfig` | `gosx-cms/studio/studio.go` (verify the actual file by reading; everything that's a panel-foundation type, not a content type) | `gosx-studio/options.go` (renamed to avoid collision with the existing `gosx-studio/studio.go` which holds CONTRACT types like `SiteMap`/`Page`/`Component`) |
| Helpers: `firstNonEmpty`, `boolAttr`, `fmtAny`, `normalizeModes`, `normalizeMetrics`, `normalizeViewports`, `normalizePanels`, etc. | `gosx-cms/studio/studio.go` + adjacent helper files | `gosx-studio/options.go` (single file is fine) or split to `gosx-studio/options_helpers.go` if grouping makes the file too large |
| Possibly: `gosx-cms/studio/site_editor.go`, `document.go`, `ids.go` | inspect — if they contain shared foundation, move; if they're surface-specific, leave for the right surface batch | `gosx-studio/`-shaped per content |
| NEW: `Store` interface (per spec §4) | author from scratch — signatures already specified in the spec | `gosx-studio/store.go` |
| `cmsstudio` sibling files using moved foundation types | ~22 cmsstudio `.go` files that import `Shell`/`Action`/`Mode`/`Metric`/`Viewport`/`Panel` | flip `Shell` → `studio.Shell`, etc., inside cmsstudio (cmsstudio now imports gosx-studio for these types — this is the legitimate `studio → cms` import direction asserted by the spec but inverted at the type level: cmsstudio's remaining renderer files temporarily depend on gosx-studio. They will be deleted by batch 12) |
| Noni: `editorStudioShell` and any code that constructs `cmsstudio.Shell{...}` / `cmsstudio.Options{...}` | `app/admin/editor/page.server.go`, possibly `page_server_test.go` | flip `cmsstudio.Shell{...}` → `studio.Shell{...}` etc. |
| Pajaritos: `internal/site/store.go` and friends — anywhere a `cmsstudio.Options` or `cmsstudio.Shell` is constructed | inspect | flip prefix |

### Recipe — same as batch 2 with a few specifics

Run all 17 steps of the canonical recipe (trace open → clean working tree → baseline → move + verify → build → test → commit → trace close), with these specifics:

- **Step 4 (file selection)**: read `gosx-cms/studio/studio.go` end-to-end and decide per-type whether it's foundation (moves) or content (stays in CMS at its right home). Spec §Scope's "Stays in CMS, but relocates within CMS" table identifies content-shaped types — don't move them with foundation; they relocate inside CMS during batches 9 and 10.
- **Step 5 (Store interface)**: author `gosx-studio/store.go` per the signatures in spec §4. Method bodies are interfaces only (`type Store interface {...}`). The `HostShell(store, shell)` and `FeatureFlagAttrs(shell)` constructors live alongside.
- **Step 6 (gosx-studio build)**: probably surfaces no naming collisions because the foundation types don't overlap with the existing `gosx-studio/studio.go` contract types. If they do, rename inside `options.go`.
- **Step 7+ (cmsstudio sibling flips)**: this is the biggest single edit in this batch — ~22 files in cmsstudio need their `Shell`/`Action`/`Mode` references prefixed with `studio.`. Use `grep -rln` to find them; use `gofmt`-friendly automated find/replace where types are unambiguous. Add `import studio "m31labs.dev/gosx-studio"` to each touched file.
- **Step 8+ (host flips)**: both hosts construct `cmsstudio.Shell{...}` (Noni's `editorStudioShell`, possibly pajaritos's `internal/site/store.go`). Flip prefix; add gosx-studio import if missing (pajaritos may need `go get m31labs.dev/gosx-studio@v0.5.2` first since batch 1's tidy pruned it).
- **Step 14 (tests)**: all four suites must be green. This batch touches the foundation; if anything's wrong, it shows up everywhere.

### Risk

This is the largest batch by line count and the only one that creates a *temporary inversion* — for the duration of batches 1.5 through 11, remaining cmsstudio files will `import m31labs.dev/gosx-studio` so they can reach the moved foundation types. That's against the spec's stated CMS direction, but it's transitional: by end of batch 12, `gosx-cms/studio/` is deleted entirely and the inversion vanishes.

If this temporary import direction is uncomfortable, the alternative is to do everything in one big-bang move — but the user has chosen branch-by-batch. The inversion is the price.

### Success criteria

- `gosx-cms/studio/studio.go` shrinks to only content-shaped types (`PublishReview`, `HealthReport`, etc. that will move within CMS in batches 9-10) — or is gone entirely if all content types have already been relocated
- `gosx-studio/options.go` contains `Shell`, `Options`, `View`, `Workbench`, `Mode`, `Metric`, `Viewport`, `Panel`, `Action`, and their helpers
- `gosx-studio/store.go` contains the `Store` interface from spec §4
- Remaining cmsstudio files compile by importing `m31labs.dev/gosx-studio` for the moved types
- Noni's `editorStudioShell` constructs `studio.Shell{...}` (not `cmsstudio.Shell{...}`)
- Pajaritos: same flip if it constructs any of these types directly
- All four `go test ./...` suites green
- 4 buckley commits

---

## Batch 2: Chrome — the canonical surface lift

**Why this is THE template batch:** Chrome is the lowest-coupled surface AFTER the foundation is in place (batch 1.5). With Shell/Action/Mode/Metric/Viewport already in gosx-studio, chrome renderers can now move with only their own dependencies. If the recipe works here, it works for every later surface lift.

### Surface manifest

**Chrome surface includes:**

| What | Source | Target |
|---|---|---|
| Renderers: `RenderStudioToolbar`, `RenderSaveStatus`, `RenderHistoryControls`, `RenderCommandPaletteScript`, `RenderStudioStateScript` | `gosx-cms/studio/{chrome.go, save_status.go, history_controls.go, commands.go, command_builder.go}` + their `_test.go` files | `gosx-studio/chrome.go` + `gosx-studio/chrome_test.go` |
| `.gsx` components: `StudioWorkbenchShell`, `StudioToolbar`, `StudioModebar`, `StudioMetricStrip`, `StudioCommandPalette`, `StudioSaveStatus`, `StudioHistoryControls` | `muddy-noni-commerce/app/admin/editor/page.gsx` lines 45-273 (verify with `grep -n "^func StudioWorkbenchShell\|^func StudioToolbar\|^func StudioModebar\|^func StudioMetricStrip\|^func StudioCommandPalette\|^func StudioSaveStatus\|^func StudioHistoryControls" page.gsx`) | `gosx-studio/chrome.gsx` |
| Noni call-site flips | `muddy-noni-commerce/app/admin/editor/page.server.go` (any `cmsstudio.RenderStudioToolbar` / `cmsstudio.RenderSaveStatus` / `cmsstudio.RenderHistoryControls` / `cmsstudio.RenderCommandPaletteScript` / `cmsstudio.RenderStudioStateScript`) and `page_server_test.go` | `studio.<same>` |
| Pajaritos call-site flips | `pajaritos-forest-school/internal/site/store.go` and `store_test.go` — if they reference any of the above | `studio.<same>` |

**Workbench panels NOTE:** `gosx-cms/studio/workbench.go` and `workbench_panels.go` contain a mix of chrome (workbench shell), canvas (preview frame), and inspector (panels). At planning time the implementer must read these files and decide per-function which surface owns each function. Default: anything named `RenderWorkbench<X>` where X is chrome-shaped (toolbar, save, history) goes with this batch; anything panel-shaped goes with batches 5-7; anything canvas-shaped goes with batch 3. If unsure, split the file across batches by copy-paste rather than git-mv (preserving function-level history matters less than commit cleanliness).

### Steps

- [ ] **Step 1: Open the trace**

```bash
hypha trace start --agent "identity://claude-extraction" --task "pillar-5-batch-2-chrome" --phase "extract chrome surface"
```

- [ ] **Step 2: Baseline test on all four suites**

```bash
for d in gosx-studio gosx-cms muddy-noni-commerce pajaritos-forest-school; do
  echo "=== $d ==="
  (cd /home/draco/work/$d && go test ./... 2>&1 | tail -3)
done
```
Expected: every block ends with `ok` lines, no `FAIL`. Abort if any red.

- [ ] **Step 3: Run impact analysis to confirm the move list**

```bash
hypha analyze impact RenderStudioToolbar --space hypha://m31labs/gosx-cms
hypha analyze refs RenderSaveStatus --space hypha://m31labs/gosx-cms
hypha analyze refs RenderHistoryControls --space hypha://m31labs/gosx-cms
hypha analyze refs RenderCommandPaletteScript --space hypha://m31labs/gosx-cms
```
Capture call-site lists into the trace. These are the exact spots step 6 must flip. If the analysis surfaces a call site not in the manifest above, ADD it before proceeding.

- [ ] **Step 4: Move `chrome.go` and its test**

```bash
cd /home/draco/work
git -C gosx-cms mv studio/chrome.go ../gosx-studio/chrome.go
git -C gosx-cms mv studio/chrome_test.go ../gosx-studio/chrome_test.go
```

Verify the package declaration in both moved files is `package studio` (it already is in both source and destination — no change). Run:
```bash
head -1 /home/draco/work/gosx-studio/chrome.go
```
Expected: `package studio`.

- [ ] **Step 5: Move `save_status.go`, `history_controls.go`, `commands.go`, `command_builder.go` and their `_test.go` partners**

```bash
cd /home/draco/work
for f in save_status save_status_test history_controls history_controls_test commands commands_test command_builder; do
  git -C gosx-cms mv studio/${f}.go ../gosx-studio/${f}.go
done
```

Note: there is no `command_builder_test.go` per the file inventory. If a build error reports a missing test helper, look in `commands_test.go`.

- [ ] **Step 6: Build gosx-studio — surface any naming collisions**

```bash
cd /home/draco/work/gosx-studio && go build ./...
```
Expected outcomes:
- ✅ Silent success: proceed
- ❌ "redeclared in this block" — two moved files define a type with the same name. Inspect, rename the inner one (e.g., `ToolbarOptions` if it conflicts with a `Toolbar` elsewhere). Document the rename in the commit message.
- ❌ Import errors: a moved file imports a sibling cmsstudio file that hasn't moved yet. Either (a) move that file too, or (b) revert this batch's moves of files that depend on the unmoved sibling, leaving them for a later batch. Update the manifest if so.

- [ ] **Step 7: Move chrome `.gsx` components from Noni**

Read `/home/draco/work/muddy-noni-commerce/app/admin/editor/page.gsx`. Locate each function in the manifest's GSX components row (lines roughly 45-273 — verify with grep). Use the Edit tool to cut each function block out of `page.gsx` and the Write tool to assemble them into the new file:

```bash
cat /home/draco/work/gosx-studio/chrome.gsx
```
should produce (after the Write tool is done):
```go
package studio

func StudioWorkbenchShell() Node {
    // ...existing body verbatim from page.gsx...
}

func StudioToolbar() Node {
    // ...
}

// ...and so on for StudioModebar, StudioMetricStrip, StudioCommandPalette, StudioSaveStatus, StudioHistoryControls
```

After the cut, `page.gsx` no longer defines those seven functions. Verify with:
```bash
grep -nE "^func (StudioWorkbenchShell|StudioToolbar|StudioModebar|StudioMetricStrip|StudioCommandPalette|StudioSaveStatus|StudioHistoryControls)\(" /home/draco/work/muddy-noni-commerce/app/admin/editor/page.gsx
```
Expected: empty output (none of those functions remain in Noni's page.gsx).

- [ ] **Step 8: Build all four go modules to confirm the .gsx move compiles**

```bash
for d in gosx-studio gosx-cms muddy-noni-commerce pajaritos-forest-school; do
  echo "=== $d ==="
  (cd /home/draco/work/$d && go build ./... 2>&1 | head -20)
done
```
Expected: silent success in all four. If gosx-studio fails with "undefined: Node" or similar, the destination `.gsx` may be missing an import — `gosx-studio` may need `import . "m31labs.dev/gosx"` if its other `.gsx` files use the same pattern. Check `gosx-studio/fieldruntime/field_bind.gsx` for the canonical pattern.

If muddy-noni fails with "undefined: StudioWorkbenchShell" inside Page() or another remaining component, the `Page()` function in `page.gsx` still references the moved components — update it to use `studio.StudioWorkbenchShell` etc., importing `m31labs.dev/gosx-studio`. The `.gsx` import syntax depends on how other admin pages import; check the file's existing imports.

- [ ] **Step 9: Update Noni's `page.server.go` call sites**

Find each `cmsstudio.RenderStudioToolbar(...)`, `cmsstudio.RenderSaveStatus(...)`, `cmsstudio.RenderHistoryControls(...)`, `cmsstudio.RenderCommandPaletteScript(...)`, `cmsstudio.RenderStudioStateScript(...)` in `app/admin/editor/page.server.go` and `page_server_test.go`. Flip the prefix to `studio.`:

```bash
grep -n "cmsstudio\.RenderStudioToolbar\|cmsstudio\.RenderSaveStatus\|cmsstudio\.RenderHistoryControls\|cmsstudio\.RenderCommandPaletteScript\|cmsstudio\.RenderStudioStateScript" \
  /home/draco/work/muddy-noni-commerce/app/admin/editor/page.server.go \
  /home/draco/work/muddy-noni-commerce/app/admin/editor/page_server_test.go
```

For each match line, use the Edit tool to change `cmsstudio.` to `studio.`. If Noni's import block doesn't already alias `studio "m31labs.dev/gosx-studio"`, add it. If only `cmsstudio` was previously imported in that file and ALL its uses are now flipped, remove the `cmsstudio` import entirely (the build will fail with "unused import" if left).

Run `goimports -w` to clean up:
```bash
goimports -w /home/draco/work/muddy-noni-commerce/app/admin/editor/page.server.go
goimports -w /home/draco/work/muddy-noni-commerce/app/admin/editor/page_server_test.go
```

- [ ] **Step 10: Update Pajaritos call sites if any**

```bash
grep -n "cmsstudio\.RenderStudioToolbar\|cmsstudio\.RenderSaveStatus\|cmsstudio\.RenderHistoryControls\|cmsstudio\.RenderCommandPaletteScript\|cmsstudio\.RenderStudioStateScript" \
  /home/draco/work/pajaritos-forest-school/internal/site/store.go \
  /home/draco/work/pajaritos-forest-school/internal/site/store_test.go \
  /home/draco/work/pajaritos-forest-school/app/admin/editor/page.server.go \
  /home/draco/work/pajaritos-forest-school/app/admin/editor/page_server_test.go 2>/dev/null
```

For pajaritos, the chrome renderers are usually invoked indirectly through `cmsstudio.New(Options{...})` → `cmsstudio.View()` rather than direct `Render*` calls. The flip is: any direct `cmsstudio.RenderStudioToolbar` etc. becomes `studio.RenderStudioToolbar`. If pajaritos has zero direct chrome refs, this step is a no-op — proceed.

Apply edits + `goimports -w` per match (same pattern as step 9).

- [ ] **Step 11: Build and test all four suites**

```bash
for d in gosx-studio gosx-cms muddy-noni-commerce pajaritos-forest-school; do
  echo "=== build $d ==="
  (cd /home/draco/work/$d && go build ./... 2>&1 | head -10)
  echo "=== test $d ==="
  (cd /home/draco/work/$d && go test ./... 2>&1 | tail -10)
done
```
Expected: every block ends with `ok` lines, zero `FAIL`. If any fails, DO NOT commit. Debug, fix, re-run step 11 until green.

Common failure modes:
- `gosx-cms` build fails with "RenderStudioToolbar undefined" — a cmsstudio internal references one of the moved functions; that reference must move with the function or it must take an injected callback. Per-case judgment.
- `gosx-studio` test fails — a moved `_test.go` references a helper that stayed in cmsstudio. Move the helper too (typically it's in `panels_test.go` or `studio_test.go`).
- Noni test fails — the test asserts a specific render output and the moved function's output has shifted subtly. Should not happen if the function moved verbatim; investigate any failure carefully.

- [ ] **Step 12: Verify the chrome surface is fully gone from cmsstudio**

```bash
grep -nE "func Render(StudioToolbar|SaveStatus|HistoryControls|CommandPaletteScript|StudioStateScript)\b" /home/draco/work/gosx-cms/studio/*.go
```
Expected: empty (all moved). If anything matches, that file/function still belongs to chrome and didn't move — finish the move.

- [ ] **Step 13: Commit each repo with buckley**

```bash
cd /home/draco/work/gosx-cms && buckley commit --yes --minimal-output
cd /home/draco/work/gosx-studio && buckley commit --yes --minimal-output
cd /home/draco/work/muddy-noni-commerce && buckley commit --yes --minimal-output
cd /home/draco/work/pajaritos-forest-school && buckley commit --yes --minimal-output
```

Order matters: gosx-cms commits FIRST (it loses files), then gosx-studio (it gains them), then hosts (they see both new and removed). If a repo has nothing to commit (e.g., pajaritos chrome refs were already empty), skip it; `buckley commit` will detect empty diff.

- [ ] **Step 14: Tick the trace**

```bash
hypha trace tick <trace-id> "batch 2 chrome: 5 renderers + 7 .gsx components moved; all four suites green; 3-4 buckley commits landed"
```

- [ ] **Step 15: Close the trace**

```bash
hypha trace done <trace-id> --status succeeded
```

**Batch 2 success criteria:**
- `gosx-cms/studio/{chrome,save_status,history_controls,commands,command_builder}.go` no longer exist
- `gosx-studio/{chrome.go, chrome.gsx, chrome_test.go}` exist with the moved content
- Noni's `page.gsx` no longer declares the seven chrome `.gsx` functions
- Noni's `page.server.go` references `studio.RenderStudioToolbar` etc., not `cmsstudio.<same>`
- All four `go test ./...` suites green
- ≥3 buckley commits landed (cms, studio, noni; pajaritos optional)

---

## Batches 3-11: Surface lifts (recipe-driven)

Each of these batches follows the recipe above. The manifests below are the inputs. **Before starting a batch, the implementer must read the actual files and refine the manifest** — file boundaries between surfaces aren't perfect in cmsstudio today, and a few functions will need judgment calls.

### Batch 3 — Canvas

- `STUDIO_TARGET_FILE`: `gosx-studio/canvas.go` + `canvas_test.go`
- `STUDIO_GSX_TARGET_FILE`: `gosx-studio/canvas.gsx`
- `CMSSTUDIO_FILES`: `studio/site_canvas.go`, `site_canvas_test.go`, `preview.go`, `preview_test.go`, `workspace_surface.go`, `workspace_surface_test.go`
- `GSX_COMPONENTS` (from page.gsx): `StudioRailResizer`, `StudioCanvasBar`, `StudioCanvasTools`, `StudioZoomControls`, `StudioZoomControlsIsland`, `StudioPreviewFrame`, `StudioPreviewViewportIsland`, `StudioViewportSwitcher`, `StudioCanvasStatus`, `CanvasInsertShelf`, `CanvasSelectionCommandbar`
- `NONI_TOUCH_FILES`: `app/admin/editor/page.server.go`, `page_server_test.go`
- `PAJARITOS_TOUCH_FILES`: `internal/site/store.go`, `store_test.go`

### Batch 4 — Site Navigator + Site Map

**One-time survey before starting batch 4:** Batches 4, 5, 6, and 8 each take portions of `gosx-cms/studio/panels.go`. Before splitting it across four batches, run this survey first and capture the function-to-batch mapping in the trace:

```bash
grep -nE "^func Render" /home/draco/work/gosx-cms/studio/panels.go
```

Then decide per function which batch owns it. Suggested mapping:
- `RenderSiteNavigator`, `RenderLayerList` → batch 4
- `RenderBlockLibrary` → batch 5
- `RenderInspectorPanel`, `RenderInspectorHeader`, `RenderScopeStrip`, `RenderPanelHeading` → batch 6
- `RenderBrandPanel`, `RenderLinkGridPanel` → batch 8
- `RenderFlowLibrary` → batch 11

Capture the final mapping in the trace as `hypha trace tick <id> "panels.go function-to-batch mapping: ..."` so batches 5, 6, 8, 11 don't re-derive it.

The same survey applies to `workbench.go`, `workbench_composition.go`, and `workbench_panels.go` — these mix chrome (batch 2), canvas (batch 3), and inspector (batch 6) responsibilities. Survey at the start of batch 2 or wherever a workbench-named function first comes up.

- `STUDIO_TARGET_FILE`: `gosx-studio/site_navigator.go` + `site_navigator_test.go`
- `STUDIO_GSX_TARGET_FILE`: `gosx-studio/site_navigator.gsx`
- `CMSSTUDIO_FILES`: portions of `studio/panels.go`, `panels_test.go` matching `RenderSiteNavigator`, `RenderLayerList`
- `GSX_COMPONENTS`: `SiteNavigatorPanel`, `HomeLayersPanel`, `HomeLayerSelectionIsland`, `SiteNavigatorIsland`, `SiteMapEnginePanel`, `SiteMapCanvasIsland`
- `NONI_TOUCH_FILES`: same as batch 3
- `PAJARITOS_TOUCH_FILES`: same as batch 3 (likely heavier — pajaritos uses site navigation)

### Batch 5 — Block Library + Block Layout

- `STUDIO_TARGET_FILE`: `gosx-studio/blocks.go` + `blocks_test.go`
- `STUDIO_GSX_TARGET_FILE`: `gosx-studio/blocks.gsx`
- `CMSSTUDIO_FILES`: `studio/panels.go` portions (`RenderBlockLibrary`, `RenderLayerList` if not already moved in batch 4)
- `GSX_COMPONENTS`: `BlockLibraryPanel`, `BlockLayoutEnginePanel`, `CanvasEnginePanel`
- `NONI_TOUCH_FILES`: same
- `PAJARITOS_TOUCH_FILES`: `internal/site/store.go`, `store_test.go`

### Batch 6 — Home Inspector

- `STUDIO_TARGET_FILE`: `gosx-studio/inspector.go` + `inspector_test.go`
- `STUDIO_GSX_TARGET_FILE`: `gosx-studio/inspector.gsx`
- `CMSSTUDIO_FILES`: `studio/panels.go` portions (`RenderInspectorPanel`, `RenderInspectorHeader`, `RenderScopeStrip`)
- `GSX_COMPONENTS`: `InspectorChromePanel`, `HomeInspectorPanel`, `HomeInspectorIsland`
- `NONI_TOUCH_FILES`: same
- `PAJARITOS_TOUCH_FILES`: same

### Batch 7 — Look

- `STUDIO_TARGET_FILE`: `gosx-studio/look.go` + `look_test.go`
- `STUDIO_GSX_TARGET_FILE`: `gosx-studio/look.gsx`
- `CMSSTUDIO_FILES`: `studio/style_panel.go`, `style_panel_test.go`, `design_panels.go`, `design_panels_test.go`
- `GSX_COMPONENTS`: `LookPanel`, `LookStyleScope`, `LookStyleWorkbench`, `LookStyleImpact`, `LookChoiceGroup`, `LookField`, `LookSelectControl`, `LookRadioGroup`, `LookSwatches`, `LookColorGrid`
- `NONI_TOUCH_FILES`: same
- `PAJARITOS_TOUCH_FILES`: same

### Batch 8 — Brand

- `STUDIO_TARGET_FILE`: `gosx-studio/brand.go` + `brand_test.go`
- `STUDIO_GSX_TARGET_FILE`: `gosx-studio/brand.gsx`
- `CMSSTUDIO_FILES`: portions of `studio/panels.go` matching brand (`RenderBrandPanel` already named) — verify with grep
- `GSX_COMPONENTS`: `BrandPanel`, `BrandFieldList`, `BrandMediaPickerIsland`
- `NONI_TOUCH_FILES`: same
- `PAJARITOS_TOUCH_FILES`: same

### Batch 9 — Publish (HEAVY pajaritos diff)

This batch carries the bulk of Pajaritos's direct `cmsstudio` imports (lifecycle helpers).

- `STUDIO_TARGET_FILE`: `gosx-studio/publish.go` + `publish_test.go`
- `STUDIO_GSX_TARGET_FILE`: `gosx-studio/publish.gsx`
- `CMSSTUDIO_FILES`: `studio/publish_review.go`, `publish_review_test.go`, `lifecycle_review.go`, `lifecycle_review_test.go`, `lifecycle_commands.go`, `lifecycle_commands_test.go`, `assessment.go`, `assessment_test.go`
- `GSX_COMPONENTS`: `PublishPanel`, `RevisionHistoryPanel`, `PreviewSharePanel`
- `NONI_TOUCH_FILES`: same
- `PAJARITOS_TOUCH_FILES`: `internal/site/store.go`, `store_test.go`, `internal/site/lifecycle.go`, `lifecycle_test.go`, `app/admin/editor/page.server.go`, `page_server_test.go`

**Special move in batch 9 (per spec §Scope):** Content-state types relocate within CMS:
- `cmsstudio.PublishReview` → `gosx-cms/publish/publish_review.go` (new sub-package)
- `cmsstudio.Readiness` → `gosx-cms/lifecycle/readiness.go`
- `cmsstudio.RevisionItem` → `gosx-cms/lifecycle/revision_item.go` (deduplicate with existing `lifecycle.Revision` if possible)

Anywhere these types were imported with the `cmsstudio.` prefix, flip to their new package. Update Noni AND pajaritos to use the new package paths. Each rename is a separate step inside the batch.

**Heads-up for the `Store` interface signature shift:** Per spec §4, `Store.Readiness()` currently returns the bridged `studio.Readiness` type (which during batches 2-8 is a thin re-export of `cmsstudio.Readiness`). When `Readiness` lands at `gosx-cms/lifecycle/readiness.go` in this batch, `Store.Readiness()` must change its return type to `lifecycle.Readiness`. Update:
- `gosx-studio/store.go` — change the interface method signature
- `gosx-studio/store_test.go` — the `stubStore` (added in batch 12 step 14, may already be in place if Pillar 5 added it earlier) needs its `Readiness()` method updated
- Noni and pajaritos `Store` implementations — update their method signatures and any callers that read the return type

This is the only signature break in the entire pillar; account for it explicitly so it doesn't show up as a mysterious test failure mid-batch.

### Batch 10 — Activity (medium pajaritos diff)

- `STUDIO_TARGET_FILE`: `gosx-studio/activity.go` + `activity_test.go`
- `STUDIO_GSX_TARGET_FILE`: `gosx-studio/activity.gsx`
- `CMSSTUDIO_FILES`: `studio/activity.go`, `activity_test.go`, `health.go`, `health_test.go`, `performance.go`, `performance_test.go`
- `GSX_COMPONENTS`: `ActivityPanel`, `ActivityHealthPanel`, `ActivityPerformancePanel`, `ActivityEmptyPanel`
- `NONI_TOUCH_FILES`: same
- `PAJARITOS_TOUCH_FILES`: `internal/site/lifecycle.go`, `lifecycle_test.go`

**Special move in batch 10:**
- `cmsstudio.HealthReport` → `gosx-cms/health/report.go` (new sub-package)
- `cmsstudio.PerformanceSignal` → `gosx-cms/health/performance.go`
- `cmsstudio.LifecycleActivityItem` → `gosx-cms/lifecycle/activity.go`

### Batch 11 — Flow (HEAVY pajaritos diff)

This batch carries the rest of Pajaritos's direct `cmsstudio` imports (flow helpers).

- `STUDIO_TARGET_FILE`: `gosx-studio/flow.go` + `flow_test.go`
- `STUDIO_GSX_TARGET_FILE`: `gosx-studio/flow.gsx`
- `CMSSTUDIO_FILES`: `studio/flow_authoring.go`, `flow_authoring_test.go`, `flow_cards.go`, `flow_cards_test.go`, `flow_editor.go`, `flow_editor_test.go`
- `GSX_COMPONENTS`: `FlowDesignerPanel`, `FlowDesignerIsland`
- `NONI_TOUCH_FILES`: same + `app/admin/editor/flow_designer.server.go`
- `PAJARITOS_TOUCH_FILES`: same as batch 9 (heavy)

**Special renames in batch 11 (per spec §Scope):**
- `cmsflows.StudioFlow` → `cmsflows.Flow`
- `cmsflows.StudioStep` → `cmsflows.Step`
- `cmsflows.StudioAction` → `cmsflows.Action`

These are renames within `gosx-cms/flows/`, not moves. Update all callers across all four repos.

### Batch 12 below — Cleanup

---

## Batch 12: Cleanup, delete `gosx-cms/studio/`, doc scrub, retire parity_e2e

This batch only runs after batches 2-11 have all landed.

**Files (cleanup):**
- Delete: `gosx-cms/studio/` (entire directory)
- Modify: `gosx-studio/README.md` (boundary statement)
- Modify (scrub): every `.go`/`.ts`/`.md` file in `gosx-studio` containing the string `"the legacy bundle"` (~110 occurrences)
- Modify (final): `muddy-noni-commerce/internal/gosxstudio/runtime.go` (move 3 handlers to gosx-studio) and `muddy-noni-commerce/app/admin/editor/page.server.go` (`editorStudioFeatureFlagAttrs` → `studio.FeatureFlagAttrs`)
- Delete or convert: `gosx-studio/parity_e2e/`
- Update: hyphae `initiative.gosx-studio-current-state`

### Steps

- [ ] **Step 1: Open trace**

```bash
hypha trace start --agent "identity://claude-extraction" --task "pillar-5-batch-12-cleanup" --phase "final cleanup + doc scrub"
```

- [ ] **Step 2: Confirm gosx-cms/studio/ is empty of non-trivial Go files**

```bash
ls /home/draco/work/gosx-cms/studio/
```
Expected: should contain only `ids.go` (the only non-batched file by inventory) — or be empty. If other `.go` files remain, identify their surface and route to the appropriate batch (2-11) before continuing. If `ids.go` remains, decide: does it belong in Studio (editor-shaped) or CMS root (utility)? Move accordingly.

- [ ] **Step 3: Move the 3 runtime handlers from Noni to gosx-studio**

The handlers live in `muddy-noni-commerce/internal/gosxstudio/runtime.go`:

```bash
grep -n "WorkbenchRuntimeHandler\|CommandRuntimeHandler\|StateRuntimeHandler" /home/draco/work/muddy-noni-commerce/internal/gosxstudio/runtime.go
```

Move each `func WorkbenchRuntimeHandler() http.Handler { ... }`, `func CommandRuntimeHandler()`, `func StateRuntimeHandler()` from Noni's runtime.go into `gosx-studio/runtime.go`. Also move any underlying `*RuntimeScript()` functions they call. The existing path constants in `gosx-studio/shell.go` (`WorkbenchRuntimePath`, `CommandRuntimePath`, `StateRuntimePath`) now have handlers that match them — this resolves the asymmetry from the design audit.

Update Noni's app mount code (where `app.Mount("GET "+WorkbenchRuntimePath, WorkbenchRuntimeHandler())` lives) to call `studio.WorkbenchRuntimeHandler()` instead of the local one. Delete the now-empty `internal/gosxstudio/runtime.go` if everything moved.

- [ ] **Step 4: Replace `editorStudioFeatureFlagAttrs` with `studio.FeatureFlagAttrs`**

In `gosx-studio/shell.go`, add:

```go
// FeatureFlagAttrs returns the data-* attribute map that the workbench root
// must render so the BridgeShim flag gates evaluate to true. Hosts SHOULD
// invoke this rather than hardcoding the flag-attribute map, so adding a
// new runtime island automatically rolls forward to all hosts.
func FeatureFlagAttrs(config ShellConfig) map[string]string {
    out := make(map[string]string, len(config.FeatureFlags))
    for key, enabled := range config.Normalize().FeatureFlags {
        if !enabled {
            continue
        }
        if !strings.HasSuffix(key, "-runtime-islands") {
            continue
        }
        out["data-gosx-studio-feature-flag-"+key] = "true"
    }
    return out
}
```

Add a test in `shell_test.go`:

```go
func TestFeatureFlagAttrsRendersAllEnabledRuntimeFlags(t *testing.T) {
    attrs := FeatureFlagAttrs(DefaultShellConfig())
    expected := []string{
        "data-gosx-studio-feature-flag-field-runtime-islands",
        "data-gosx-studio-feature-flag-selection-runtime-islands",
        "data-gosx-studio-feature-flag-brand-runtime-islands",
        "data-gosx-studio-feature-flag-block-layout-runtime-islands",
        "data-gosx-studio-feature-flag-style-runtime-islands",
        "data-gosx-studio-feature-flag-workbench-runtime-islands",
        "data-gosx-studio-feature-flag-preview-runtime-islands",
    }
    for _, key := range expected {
        if attrs[key] != "true" {
            t.Fatalf("missing or false attr %q in %#v", key, attrs)
        }
    }
}
```

Replace Noni's `editorStudioFeatureFlagAttrs()` body with `return studio.FeatureFlagAttrs(host.ShellConfig())` (or wherever Noni's ShellConfig lives). Delete the now-unused helper.

- [ ] **Step 5: Verify everything still builds and tests**

```bash
for d in gosx-studio gosx-cms muddy-noni-commerce pajaritos-forest-school; do
  (cd /home/draco/work/$d && go build ./... && go test ./... 2>&1 | tail -3)
done
```
Expected: all green.

- [ ] **Step 6: Delete `gosx-cms/studio/` entirely (if confirmed empty in Step 2)**

```bash
git -C /home/draco/work/gosx-cms rm -r studio/
cd /home/draco/work/gosx-cms && go build ./... && go test ./... 2>&1 | tail -3
```
Expected: `gosx-cms` still builds and tests green (it should — the package had no internal callers since the editor code moved out batch-by-batch).

- [ ] **Step 7: Scrub the "the legacy bundle" find-replace artifacts**

```bash
grep -rln "the legacy bundle" /home/draco/work/gosx-studio --include='*.go' --include='*.ts' --include='*.md' > /tmp/legacy-bundle-files.txt
wc -l /tmp/legacy-bundle-files.txt
```
Expected: ~19 files (based on the audit earlier in design).

For each file in the list, read the surrounding context with Read tool. The string almost always appeared where `studio-engines.js` or `preview-runtime.js` used to be referenced as a `file.js:line` cite. Where the original line number is preserved (`"the legacy bundle:912"`), replace with `(original location lost in deletion)`. Where the entire phrase is gibberish (e.g., `"the legacy the legacy bundle"`), remove it cleanly so the surrounding prose reads.

Sample fixes:
- `// Mirrors applyTextUpdate at the legacy bundle:912.` → `// Mirrors applyTextUpdate (legacy bundle removed 2026-05-27).`
- `// the legacy the legacy bundle mount(root) walked` → `// The legacy mount(root) walked`

This is judgment work — no mechanical sed will produce readable output. Plan ~15-30 minutes for the scrub.

Verify no occurrences remain:
```bash
grep -rn "the legacy bundle" /home/draco/work/gosx-studio --include='*.go' --include='*.ts' --include='*.md'
```
Expected: empty.

- [ ] **Step 8: Update `gosx-studio/README.md` boundary statement**

The current README (line 8) lists three peer packages and says `gosx-studio: site canvas, inspectors, no-code authoring, theme controls, flow editing, publish tools, and showcase plugins`. Add a paragraph after the "Initial Scope" section:

```markdown
## Architectural Boundary

Studio imports CMS freely; CMS never imports Studio. Studio is the authoring surface for any content CMS encodes. A host wires both modules together via `studio.HostShell(store, shellConfig)`, where the host implements the `studio.Store` interface to feed Studio with its content.
```

- [ ] **Step 9: Retire `parity_e2e/`**

Pick one of two outcomes (recommended: delete):

(a) Delete:
```bash
git -C /home/draco/work/gosx-studio rm -r parity_e2e/
```
Document the deletion in the commit message: parity dual-path comparison no longer exists; Pillar 2 will choose its own e2e harness.

(b) Convert to smoke test: rewrite each test to only run candidate mode and assert "editor loads, mount points present." Removes ~70% of the harness. Only do this if you're confident a single-mode browser smoke is useful.

- [ ] **Step 10: Update hyphae initiative**

```bash
hypha show hypha://m31labs/gosx-studio/object/initiative.gosx-studio-current-state
```

Author an update spore that:
- Notes Pillar 5 complete
- Lists Pillars 1-4 as sequenced future work (cite the spec at `gosx-studio/docs/superpowers/specs/2026-05-28-pillar-5-extraction-design.md`)
- Updates the "What Is Done" section to reflect Studio now owns the editor
- Submit:
  ```bash
  hypha spore submit <draft.md> --sign --as identity://odvcencio
  hypha graft <spore-id> --as identity://odvcencio
  ```

- [ ] **Step 11: Write the hyphae lesson on the `StudioXxx`-naming smell**

```bash
hypha show hypha://m31labs/gosx-studio/object/space.m31labs-gosx-studio
# pull skeleton
```

Author a lesson:
```markdown
# Lesson: `StudioXxx` prefix in a sibling package is a boundary smell

**Context:** During Pillar 5, gosx-cms held types named `StudioFlow`, `StudioStep`, `StudioAction`, `RecipeView` (etc.) inside its `studio/` sub-package and across `flows/` and `style/`. The prefix was meaningful when CMS owned its own editor; once gosx-studio became the editor module, the prefix was a sign that CMS hadn't actually let go.

**The smell:** When a sibling package needs a prefix matching another sibling's name, the boundary is wrong. Either:
1. The prefixed type belongs in the named sibling (move it), or
2. The two siblings should be one package (merge them).

**The fix in Pillar 5:** Types like `StudioFlow` lost the prefix and became `cmsflows.Flow`. Editor-shaped types like `RecipeView` moved to gosx-studio. Boundary-honoring rename + relocation.

**When to apply:** Any time a type or function carries a sibling-package name as a prefix, ask: should this be in that sibling, or merged with it?
```

Submit + graft same as Step 10.

- [ ] **Step 12: Run all four test suites for the final time**

```bash
for d in gosx-studio gosx-cms muddy-noni-commerce pajaritos-forest-school; do
  echo "=== final $d ==="
  (cd /home/draco/work/$d && go test ./... 2>&1 | tail -3)
done
```
Expected: green across the board.

- [ ] **Step 13: Verify success criteria**

Verify each item from spec §Success Criteria:

```bash
# gosx-cms/studio/ should not exist
ls /home/draco/work/gosx-cms/studio/ 2>&1
# Expected: "No such file or directory"

# gosx-studio.Store interface must exist
grep -n "^type Store interface" /home/draco/work/gosx-studio/store.go
# Expected: one match

# Both hosts must compile + test green (re-run from Step 12)

# Pajaritos editor route should still be ~340 lines or less
wc -l /home/draco/work/pajaritos-forest-school/app/admin/editor/page.gsx \
       /home/draco/work/pajaritos-forest-school/app/admin/editor/page.server.go
# Expected: ≤ 42 + 297 + small buffer for new gosx-studio import lines

# Noni editor route must have shrunk under 600 lines total
wc -l /home/draco/work/muddy-noni-commerce/app/admin/editor/page.gsx \
       /home/draco/work/muddy-noni-commerce/app/admin/editor/page.server.go
# Expected: substantially less than 7060; goal is < 600

# gosx-studio must have an integration test
grep -rn "func TestStudioRendersWithStubHost\|func TestStudioRendersAgainstStubStore" /home/draco/work/gosx-studio/*_test.go
# Expected: one match (added in Step 14 below if missing)

# Three runtime handlers ship in gosx-studio
grep -n "func WorkbenchRuntimeHandler\|func CommandRuntimeHandler\|func StateRuntimeHandler" /home/draco/work/gosx-studio/runtime.go
# Expected: 3 matches

# Zero "the legacy bundle" artifacts
grep -rn "the legacy bundle" /home/draco/work/gosx-studio --include='*.go' --include='*.ts' --include='*.md'
# Expected: empty

# parity_e2e gone or single-mode
ls /home/draco/work/gosx-studio/parity_e2e/ 2>&1
# Expected: "No such file" (if deleted) or harness.ts shows single-mode only
```

If any check fails, finish that piece before declaring Pillar 5 complete.

- [ ] **Step 14: If `TestStudioRendersAgainstStubStore` doesn't yet exist, add it**

Write `gosx-studio/store_test.go`:

```go
package studio

import (
    "testing"

    "m31labs.dev/gosx"
    "m31labs.dev/gosx-cms/lifecycle"
    "m31labs.dev/gosx-cms/media"
)

type stubStore struct{}

func (stubStore) LeftPanels() []Panel            { return []Panel{{Children: []gosx.Node{}}} }
func (stubStore) RightPanels() []Panel           { return []Panel{{Children: []gosx.Node{}}} }
func (stubStore) MediaAssets() []media.Asset     { return nil }
func (stubStore) Revisions() []lifecycle.Revision { return nil }
func (stubStore) Readiness() Readiness            { return Readiness{} }
func (stubStore) Adapters() []ResourceAdapter     { return DefaultResourceAdapters() }

func TestStudioRendersAgainstStubStore(t *testing.T) {
    shell := HostShell(stubStore{}, DefaultShellConfig().Normalize())
    if shell.Labels.ProductName == "" {
        t.Fatalf("expected normalized ProductName, got empty")
    }
    // Add additional render-shape assertions as Studio's surface stabilizes.
}
```

Run:
```bash
cd /home/draco/work/gosx-studio && go test -run TestStudioRendersAgainstStubStore -v ./...
```
Expected: PASS.

This test gates the boundary: if a later change makes Studio import Noni or pajaritos transitively, this test fails because the stub store has no Noni / pajaritos dependency.

- [ ] **Step 15: Commit all four repos**

```bash
cd /home/draco/work/gosx-cms && buckley commit --yes --minimal-output
cd /home/draco/work/gosx-studio && buckley commit --yes --minimal-output
cd /home/draco/work/muddy-noni-commerce && buckley commit --yes --minimal-output
cd /home/draco/work/pajaritos-forest-school && buckley commit --yes --minimal-output
```

- [ ] **Step 16: Tick + close the trace**

```bash
hypha trace tick <trace-id> "pillar 5 complete: gosx-cms/studio deleted; gosx-studio is the editor; both hosts thin; ~110 doc artifacts scrubbed"
hypha trace done <trace-id> --status succeeded
```

**Batch 12 success criteria:**
- All success criteria from spec §Success Criteria pass (verified in Step 13)
- One commit per repo (cms loses studio/; studio gains store_test.go + handlers + FeatureFlagAttrs + scrubbed docs; hosts get FeatureFlagAttrs flip; pajaritos final lint)
- Trace closed succeeded
- Hyphae initiative updated; lesson submitted

---

## Done When

- All 12 batches landed
- Pajaritos's editor route is still ≤ 350 lines (proof no Noni-shape leaked into Studio)
- Noni's editor route is < 600 lines (host glue only)
- The `gosx-studio` integration test against a stub store passes
- Hyphae current-state initiative reflects Pillar 5 complete and Pillars 1-4 sequenced

Time to first user-visible value: end of batch 2 (proves the recipe works and the Store interface is real). Time to full completion: 3-4 weeks of focused work, or faster with worktree fan-out across non-adjacent batches.

---

## Appendix: Full `gosx-cms/studio/` file inventory + batch assignment

Source files in `gosx-cms/studio/` (excluding `_test.go` partners, which always move with their `.go`):

| File | Batch | Notes |
|---|---|---|
| `activity.go` | 10 | Activity panel |
| `assessment.go` | 9 | Publish readiness assessment |
| `assets.go` | 8 | Brand media; check if any portions are publish-shaped (preview asset list) |
| `chrome.go` | 2 | Toolbar/chrome |
| `command_builder.go` | 2 | Command palette builder |
| `commands.go` | 2 | Command palette renderer |
| `design_panels.go` | 7 | Look/Brand style controls; verify by reading |
| `document.go` | 6 | Inspector document field rendering |
| `flow_authoring.go` | 11 | Flow authoring |
| `flow_cards.go` | 11 | Flow cards |
| `flow_editor.go` | 11 | Flow editor |
| `health.go` | 10 | Health panel (data types relocate to `cms/health/`) |
| `history_controls.go` | 2 | History/undo chrome |
| `ids.go` | 12 | Likely just an `ids` helper; if pure utility, move to `cms/` root; if studio-only, move to `gosx-studio` during batch 2 or 12 |
| `lifecycle_commands.go` | 9 | Publish/lifecycle command building |
| `lifecycle_review.go` | 9 | Publish review |
| `panels.go` | 4-8, 11 | Per the batch-4 survey above — splits across multiple batches |
| `performance.go` | 10 | Performance panel (data types relocate to `cms/health/`) |
| `preview.go` | 3 | Preview frame |
| `publish_review.go` | 9 | Publish review (data type relocates to `cms/publish/`) |
| `readiness.go` | 9 | Readiness (relocates to `cms/lifecycle/`) |
| `save_status.go` | 2 | Save-status chrome |
| `site_canvas.go` | 3 | Site canvas |
| `site_editor.go` | TBD | Inspect at planning time — likely batch 6 (inspector) or batch 4 (site nav); judgment per `RenderSiteEditor` signature |
| `studio.go` | 12 (rump) | Studio root types — `Options`, `Shell`, `View`, `Workbench`, `Metric`, `Viewport`, `Panel`, `Action`. The non-content types move to `gosx-studio` during batch 2 alongside the `Store` interface (since `Store` re-exports their shapes). Anything content-shaped stays in CMS. Batch 12 confirms studio.go is empty by end. |
| `style_panel.go` | 7 | Look/style panel |
| `workbench.go` | mixed | Per the survey at batch 2/3 start — chrome (batch 2), canvas (batch 3), inspector (batch 6) portions |
| `workbench_composition.go` | mixed | Same survey |
| `workbench_panels.go` | mixed | Same survey — `LifecycleActivityItem` here relocates to `cms/lifecycle/` in batch 10 |
| `workspace_surface.go` | 3 | Canvas workspace |

If `gosx-cms/studio/` gains new files between when this plan was written (2026-05-28) and when batch 1 starts, the freeze announced in batch 1 covers them — they should be assigned a batch before batch 2 begins. Run:

```bash
ls /home/draco/work/gosx-cms/studio/*.go | xargs -I {} basename {} .go | grep -v _test
```

Compare against the table above. Any new entries get a batch assignment.
