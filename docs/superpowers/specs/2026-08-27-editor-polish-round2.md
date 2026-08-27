# Editor Polish Round 2 — proof record

Date: 2026-08-27

Status: deployed to staging; this is a bounded latest-turnover record, not
enterprise certification, all-browser support, or owner acceptance.

## Released identity

- Studio runtime source: `06854e5ecfe960d8b7e63071b8bca3e897061cd2`.
- Exact Studio module: `v0.6.2-0.20260827230336-06854e5ecfe9` (no replace overlay).
- Noni source that built the image: clean pushed commit
  `453a7f6dfaa2915c34e35a8f0630cb479cfde562`.
- Current staging image: `sha256:d773cbebfd966816a12cc19538c77b0a095e438df9c35e09a075de8731055b55`,
  tag `20260827-163610-453a7f6-staging`; Deployment 177/177, Ready 1,
  pod `muddy-noni-gosx-staging-85c49c645b-mhjnx`; Deployment/PVC/PV UIDs
  unchanged. GoSX runtime is `v0.53.8`; lifecycle worker is off.
- The source commit above is the artifact build identity. A later
  documentation-only HEAD is separate and must not be presented as that source.
  STAGE16 remains preserved as historical evidence in the prior spec.

## Latest two-fix delta

- History focus: undo/redo restores selected-block focus when a block is
  present; otherwise it focuses an enabled visible instance-local Undo, Redo,
  or Add fallback. Native text undo remains native.
- Media Filter Enter: single-URL and Gallery media-lines view-only filters
  prevent implicit form submission. Filter Enter leaves URL/gallery values and
  dirty state unchanged; asset-button Enter selection remains available.

## Verification

- Studio QA: 282 tests in 9 files, 94 Chromium + 94 Firefox + 94 WebKit;
  282 passed, 0 failed, 0 skipped. Full Go passed (47 test-bearing packages
  plus 2 no-test packages); focused race passed for 4 packages plus
  `internal/runtimeasset` with no tests; Node syntax and parity TypeScript
  no-emit passed.
- Exact-pin Noni integration: host test-only checkpoint
  `b5a07a040566966e9d855713ea8e53a6f8538d2d`, fixture SHA
  `df10378b15339c3d65ee250177d81664566c7f73861db0b0a13a5de17ae00fc0`,
  14/14 Chromium, 0 failed/skipped. Noni Go/race/module/CMS TypeScript,
  production build, and clean normal server build passed. The separate normal
  server binary SHA is `fbb36461c07865b0332e49adbaf1fd4276649c240cddaea40b7bc7bb75f3e6b7`
  with `vcs.modified=false`; it is not the native image identity.
- Production `gosx build --prod` observed 89 warnings; no comparative baseline
  is asserted. Optional legacy Astro checks reported two `ImportMeta.env`
  diagnostics and missing checker/types tooling; they are outside the native
  GoSX release and are not relabeled as baseline failures or fixes.
- Deployment asset gate passed its scoped checks: 6 versioned managed/alias
  assets, 2 public routes, and 6 manifest runtime assets. Ten bare/future-IMS
  probes were separately recorded as current HTTP 200 with correct hashes,
  not as release pass criteria. Public browser smoke passed 10/10 routes,
  2/2 keyboard cases, 2/2 unsigned boundaries, and 0 asset/console/page
  errors. Evidence: `integration/report.md`, `release/report.md`, and
  `public-browser/polish2-public-browser-report.md` under
  `.tiller/scratch/codex/polish-round2-20260827/`.
- Asset bytes matched the pinned runtime: content-editor JS
  `ef805b9256eee785e6d6f63de9126923c9545596805847ff3cefb81e35e933d5`, media
  runtime JS `9a374c3cd619532c03f98e2e44c6dcc135b7d225375acee39d5315cb81892434`,
  and deployment CSS `f911bb4af23d7686bab95955ac4bd29c52ebebfe6833dc5688f68066b69363b8`;
  outer-media assets and two aliases also passed.
- Postdata receipt `/home/draco/backups/muddy-noni/postdeploy-20260827T234025Z/postdeploy-data-verify-report.json`
  is `status=verified`, SHA-256
  `fd5b6c0d11d146ef457fc5a31badef9e1e756e827414624f8fe829d4e3a8ee3a`;
  image, all 4 files/289303 bytes, manifest
  `82529b5ecd4ef7b73e6b949a4349c3e7ce48fb555d80cabb6fad88a4bce51256`, both
  SQLite checks, live identity, and helper cleanup matched. Its fresh baseline
  is the 2d3f backup receipt `dcb220784dca11bbe55d8c51a6d414a436b0d1ae37f7f09018795f6a0708310e`.

## Residuals and boundaries

- Latest bare-CSS and future-IMS probes are current HTTP 200 with correct
  hashes. Cloudflare `max-age=14400` remains an unmodified policy residual;
  versioned URLs and save/reconcile-before-full-refresh remain the accepted
  path. This does not claim the CDN policy is fixed.
- The Gallery 390px evidence shows the existing index table's Work-title/Year
  crowding above the create form; filter controls fit and functional checks
  pass. This is a follow-up, not a broad mobile visual pass. Extensionless
  remote URLs are not classified as gallery images; that remains follow-up.
- Owner-authenticated sign-in, walkthrough, and acceptance remain pending;
  email configuration is accepted and unchanged. Production, DNS, auth, and
  customer content remain unchanged; checkout/payments and lifecycle worker
  remain off. No arbitrary freeform canvas, physical-touch, all-browser, or
  enterprise-certification claim is made.
