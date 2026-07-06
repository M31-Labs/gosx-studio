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
| `studio/` | `cms/studio/` | pending (S3) |
| `studio/collab/` | `cms/studio/collab/` | pending (S3) |

S2 copied the 11 packages above with zero `studio` imports (bottom-tier, DAG
gate: `go list -deps` shows none of core/authoring/hostruntime/canvas/sitemap/
panels/backoffice/shell import `cms/studio` — n/a until S3 lands `cms/studio`
itself). The old `m31labs.dev/gosx-cms` module require + replace stay in
`go.mod` through S3; `shell/{cmsbridge.go,options.go}` still import the old
module paths until S4's cutover. `studio/` + `studio/collab/` (~60 files) and
`studio/assets/` (5 embedded JS runtimes) move in S3.
