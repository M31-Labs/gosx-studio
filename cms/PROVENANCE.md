# cms/ Provenance

Packages under `gosx-studio/cms/` are plain-file copies from the standalone
`m31labs.dev/gosx-cms` repository, folded in per
`.tiller/scratch/cms-foldin-spec-v0.1.md` (copy-with-attribution, §5). Blame
for pre-fold history stays in the source repo — `git log --follow` does not
cross repositories, so this file is the provenance anchor.

- **Source repo:** `git@github.com:odvcencio/gosx-cms.git` (`https://github.com/odvcencio/gosx-cms`)
- **Source SHA (frozen baseline, all copies in this fold-in are taken from this commit):** `81dc8b001d1d877b1abe9679dbe52727e7f1a749`
- **Final source tag (post fold-in, S5 tombstone):** `v0.2.1` (not yet cut at the time of this slice; gosx-cms is frozen at the SHA above for the duration of the fold-in)
- **Copy method:** `git -C gosx-cms show <source-SHA>:<path>` per file (never the working tree — the checkout had unrelated uncommitted WIP at copy time), then a mechanical import-path rewrite (`m31labs.dev/gosx-cms/` → `m31labs.dev/gosx-studio/cms/`) — no other content changes.

## Path mapping

| Old path (gosx-cms) | New path (gosx-studio) | Status |
|---|---|---|
| `blocks/` | `cms/blocks/` | copied (S2) |
| `content/` | `cms/content/` | copied (S2) |
| `flows/` | `cms/flows/` | copied (S2) |
| `lifecycle/` | `cms/lifecycle/` | copied (S2) |
| `lifecycle/sqlstore/` | `cms/lifecycle/sqlstore/` | copied (S2) |
| `media/` | `cms/media/` | copied (S2) |
| `render/` | `cms/render/` | copied (S2) |
| `store/` | `cms/store/` | copied (S2) |
| `store/file/` | `cms/store/file/` | copied (S2) |
| `store/memory/` | `cms/store/memory/` | copied (S2) |
| `style/` | `cms/style/` | copied (S2) |
| `studio/` | `cms/studio/` | copied (S3) |
| `studio/collab/` | `cms/studio/collab/` | copied (S3) |
| `studio/assets/` | `cms/studio/assets/` | copied (S3) |

S2 copied the 11 packages above with zero `studio` imports (bottom-tier, DAG
gate: `go list -deps` shows none of core/authoring/hostruntime/canvas/sitemap/
panels/backoffice/shell import `cms/studio` — n/a until S3 lands `cms/studio`
itself). The old `m31labs.dev/gosx-cms` module require + replace stay in
`go.mod` through S3; `shell/{cmsbridge.go,options.go}` still import the old
module paths until S4's cutover.

S3 copied `studio/` + `studio/collab/` (64 .go files) and `studio/assets/`
(5 embedded JS runtimes) — 69 files total, byte-identical to the frozen
source SHA modulo the `m31labs.dev/gosx-cms/` → `m31labs.dev/gosx-studio/cms/`
import rewrite (verified file-by-file, diff-based, both against the S1 hash
manifest and directly against the frozen source commit). `go.mod` bumped
`m31labs.dev/gosx-admin` `v0.1.1` → `v0.2.0` per spec (folded code needs
`workbench`, `calendar`, `blockstudio/collab`, all present in the local
`../gosx-admin` checkout at `v0.2.0`); `go mod tidy` picked up
`github.com/gorilla/websocket` as a new indirect dependency.

DAG gate (S3): `go list -deps ./core/... ./authoring/... ./hostruntime/...
./canvas/... ./sitemap/... ./panels/... ./backoffice/... ./shell/...` contains
zero references to `m31labs.dev/gosx-studio/cms/studio` — confirmed no cycle
reappeared through the facade.

**Note (deviation from spec narrative):** the spec's DAG description states
`cms/studio` imports the `gosx-studio` root facade (`gosxstudio
"m31labs.dev/gosx-studio"`) and that this import must be kept as-is. At the
frozen source SHA (`81dc8b0`), no `.go` file anywhere in gosx-cms actually
imports the bare `m31labs.dev/gosx-studio` package — `go.mod` requires it
(inert/vestigial), but `go list -deps ./cms/studio/...` shows its dependency
closure is only `cms/{lifecycle,flows,media,style}` +
`cms/studio/collab` + external deps, no root facade. Nothing needed
preserving; there was no such import to rewrite or keep. Confirmed via
repo-wide grep of the frozen SHA before and after the copy.
