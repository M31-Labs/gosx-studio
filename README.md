# GoSX Studio

GoSX Studio is the web-editing admin portal for no-code, user-authored
websites. It is the entire surface a site owner lives in day to day: canvas
design, content, commerce, media, and settings back-office, all in one
GoSX-rendered shell. Think Webflow-shaped — host-configured, GoSX-rendered,
one editor, one back-office, one portal.

Reference deployments: **Muddy Noni** (commerce) and **Pajaritos** (school
site).

## Run a site without writing Go

```sh
go run m31labs.dev/gosx-studio/cmd/gosx-site
```

Open `http://127.0.0.1:8080/admin` and a three-question wizard asks what the
business is called, what it does, and how people reach it. It then builds a
real, published website — a bakery gets a menu and a visit page, a consultant
gets services and about — which you edit and publish from there. Nothing else
is required: no adapters, no host application, no configuration file.

```
  Visit your site      http://127.0.0.1:8080/
  Edit your site       http://127.0.0.1:8080/admin
  Saved in             ./data/site.db
```

Editing happens on the page itself. The canvas renders the same markup a
visitor sees; click any text to change it, use the hover controls to move,
copy, or delete a section, and press `+` between sections to add one. Changes
save as you type and stay private until you press Publish — a live page keeps
serving its published version while you work on the next one.

The server binds to localhost by default and refuses to listen on a public
address unless you set `-admin-password`, so it never exposes an unprotected
admin area by accident.

Every site gets a working contact form on its contact page. Messages land in
the admin area's inbox — no mail server to configure.

### Accounts

On a laptop with no `-admin-password`, the admin is open, as before. On a
server, the password you start with unlocks one thing: the form at
`/admin/login` that creates the owner account. After that, everyone signs in
with an email and password, and the password on the command line is only a
bootstrap.

**People** in the admin invites others by link, as an editor (writes and
publishes, reads messages) or an admin (everything except handing over the
site). Anyone can turn on two-step sign-in with an authenticator app under
**Account**. **Activity** shows who did what, and can be downloaded.

### Single sign-on

**Settings → Single sign-on** connects an OpenID Connect provider such as
Google Workspace, Microsoft 365, Okta, or Keycloak. Register the site there
with the redirect address `https://yoursite.com/admin/sso/callback`, paste
the issuer address, client ID, and secret, and a "Sign in with…" button
appears on the sign-in page. Someone whose email already has an account
signs in as that account; someone from the allowed email domain gets an
editor account; everyone else is turned away.

### Review before publishing

Turn on **Settings → Team → Publishing needs approval** and editors' Publish
button becomes Request review. Requests wait under **Review**, where an admin
looks at the draft as visitors would see it, then approves and publishes it
or sends it back with a note the editor sees in the editor. Anyone signed in
can make a **preview link** for a draft from the editor's sidebar: it shows
the current draft to whoever has it for three days, without an account.

### Running it for a team

- **Locked sections.** An admin can lock any block from its toolbar. Editors
  see it and cannot change or remove it; a save that touches a locked block
  is refused with an explanation.
- **Request logs and metrics.** Every response carries an `X-Request-ID`,
  and each request is one JSON line on stderr (`-log-requests=false` to
  stop). `/admin/metrics` serves Prometheus counters and content gauges to a
  signed-in admin.
- **Privacy.** Messages export as CSV, delete one by one or by age, and can
  expire automatically (**Settings → Privacy**). One click writes a
  plain-language privacy page for the owner to check and publish.

### Use your own domain

Open **Settings → Your own domain** in the admin. Type the domain, add the two
DNS records the screen shows, and press **Check DNS now** until it reports that
the domain reaches the server.

For HTTPS, start the site with the `-https` flag. It then listens on ports 80
and 443, fetches a certificate from Let's Encrypt the first time someone opens
the secure address, and renews it on its own:

```sh
gosx-site -https -data /srv/site/site.db -admin-password 'a long passphrase'
```

Certificates are cached in a `certs` folder beside the data file. Pass
`-public-ip` to show the server's address in the DNS instructions when it
cannot be detected.

### Where the site lives

Everything is in one SQLite file, `data/site.db` by default, with pictures in
an `uploads` folder beside it. A site that was started on an older version
with `data/site.json` is moved into the database on the next start, and the
JSON file is kept as `site.json.migrated`. Pass a `.json` path to `-data` to
keep using the one-file snapshot instead.

### Selling

**Shop** in the admin holds products with pictures, options, and stock. A
product on sale appears at `/shop` and in the menu, and can be placed on any
page with the Product block. Visitors fill a cart; to let them pay, connect
Stripe under **Settings → Payments**:

1. Paste your Stripe secret key.
2. In Stripe, add a webhook endpoint for `https://yoursite.com/stripe/webhook`
   sending `checkout.session.completed`, and paste its signing secret.
3. Set a flat shipping charge, a free-shipping threshold, the countries you
   ship to, and whether Stripe Tax works out tax.

Payment happens on Stripe's hosted page; card details never reach the site.
Paid orders appear under **Shop → Orders** with the buyer's address, a link
to the payment in Stripe for refunds, and a "Mark as sent" button.

A product is one of four kinds. **Something you ship** is the default.
**A download** takes a file up to 100 MB; the buyer gets a link that works
for seven days, on the thank-you page and in the receipt. **A subscription**
charges every week, month, or year through Stripe; renewals and
cancellations flow back through the webhook. **An appointment** offers time
slots from the days, hours, and slot length you set; free appointments book
without checkout, paid ones go through the cart. Bookings appear under
**Shop → Bookings**, where you can cancel one.

Buyers have no passwords. The **Your orders** link in the footer asks for
their email and sends a sign-in link good for thirty minutes; the page then
shows their orders, downloads, and subscriptions, with a button that opens
Stripe's billing portal to change or cancel a subscription. A visitor who is
not ready can ask the cart page to email them a link to their cart; if they
have not bought within two hours, one reminder goes out, and never a second.
Both need an email transport (`-mail`).

### Staging and publishing everything at once

Saving in the editor never changes what visitors see; publishing does. The
**Staging** page lists every page and post with unpublished changes and
publishes them one at a time or all together with one button.

To walk through the whole site with those changes in place, give it a
staging address such as `staging.yourbusiness.com` on the Staging page and
point that name at the server like the main domain. The staging address
shows drafts, menus and all, under a banner. It opens only for people who
have the link, which carries a key you can renew. Search engines are told to
stay away, nothing is counted, and anything that writes or pays is sent to
the real site.

### Backups and export

**Settings → Backups and export** downloads the whole site as one zip: pages,
posts, pictures, messages, forms, and visitor counts. A backup of the same
kind is written once a day into a `backups` folder beside the data file, and
the last fourteen are kept. Turn that off with `-no-backups`.

To restore, stop the site, unzip the backup, and start the site from the
`site.db` inside it. Certificates are never included; HTTPS issues new ones.

### Run it in a container

```sh
docker build -t gosx-site .
docker run -p 8080:8080 -v gosx-site-data:/data \
  -e GOSX_SITE_ADMIN_PASSWORD=change-me gosx-site
```

Pages, pictures, and messages all live in the `/data` volume. The admin
password is required in a container because the server is reachable from the
network; the image will not start without one.

`cmd/gosx-site` is the default host: it assembles the `sitehost`, `cms`, and
`hostruntime` packages into a program that runs. Applications that need more
control still configure the packages directly, as the reference deployments
do. The default host is a floor, not a ceiling.

Earlier docs described Studio as "the authoring layer, intentionally separate
from `gosx-cms` and `gosx-admin`." That framing is retired — the code moved
past it. Studio is the portal; `gosx-admin` is a generic back-office toolkit
dependency (Studio consumes only `gosx-admin/blockstudio`); the former
`gosx-cms` content-storage module is folded into this module as `cms/*` (see
[docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) §"Release model" and
[cms/PROVENANCE.md](cms/PROVENANCE.md) for the fold-in record). The standalone
`m31labs.dev/gosx-cms` repository is frozen and tombstoned as of its final
`v0.2.1` tag; new code should import `m31labs.dev/gosx-studio/cms/...`.

## What hosts provide vs. what Studio provides

Hosts supply adapters, persistence, permissions, routes, and copy. Studio
supplies contracts, chrome, panels, canvas engines, back-office pages, and
runtime islands.

Concretely, the host application supplies:

- content adapters
- shell labels, modes, panels, and resource links
- permission adapters
- server actions
- route bindings
- product copy
- feature flags
- design tokens

Studio supplies:

- `.gsx` surfaces for visible editor UI
- GoSX engines for heavy client-side interactions
- GoSX islands for focused reactive controls
- extension points for plugins
- common authoring language for non-technical operators

## Package map

The module is a strict import DAG: `core` sits at the bottom, `shell`
composes everything above it, and peers never import each other. See
[docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) for the full diagram.

| Package | Responsibility |
|---|---|
| `core` | Pure data contracts and defaults: the site-map/canvas/composition/flow/engine/resource type system, zero UI or host imports. |
| `authoring` | The typed server-action mutation boundary: operations, form codec, adapter interface, action handler, authoring surface assembly, and the style/appearance/section-field mutation family. |
| `hostruntime` | Embedded runtime assets, bundle concatenation, and the public `/_gosx/studio/*` paths and HTTP mounting. |
| `canvas` | Page-canvas engine hosting and server-rendered surface markup (artboards, thumbnails, block-layout DOM contract). |
| `sitemap` | The visual site-map board: graph engine render, board/view projections, authoring panels/forms, site navigator. |
| `panels` | Editor inspector/designer panels and their `.gsx` islands (right/left rail content units). |
| `backoffice` | The CRUD portal surfaces — dashboard, per-domain index/detail pages, media library, settings, search, storefront preview. First-class per the portal definition. |
| `shell` | The portal shell: host-facing `ShellConfig`/`Shell`/`Store` contracts, readiness rail, workbench chrome renderers, and the editor workbench composition root. |
| `*runtime` islands (`fieldruntime`, `selectionruntime`, `workbenchruntime`, `styleruntime`, `previewruntime`, `blocklayoutruntime`, `brandruntime`, `inspectorruntime`, `sitemapruntime`, `authoringruntime`, `inlineeditruntime`, `canvas*runtime`) | The go:embed browser-runtime bundles the shell/canvas/sitemap/panels packages mount — one package per engine's client-side JS/WASM surface. |
| `plugins/showcase3d` | The CMS/Studio contract for source photos, generated model artifacts, provenance, moderation, lifecycle readiness, no-code placement controls, and Scene3D viewer descriptors. |
| `studio` (root) | Deprecated compatibility facade — type aliases and forwarding wrappers only, no logic. See "Versioning & compatibility" below. |
| `cms/{blocks, content, flows, lifecycle{,/sqlstore}, media, render, store{,/file,/memory}, style}` | Folded-in content storage: block catalogs, revision/draft/publish lifecycle, media assets, generic block rendering, and CMS store contracts (+ in-memory/file/sqlite-backed implementations). Bottom tier — stdlib, `gosx`, `gosx-admin` only, zero imports of `cms/studio` or the root facade. See [cms/PROVENANCE.md](cms/PROVENANCE.md). |
| `cms/studio` (+ `cms/studio/collab`) | Folded-in portal UI (three-pane authoring shell model, canvas/preview/panels/actions, realtime collaboration) that used to live in the standalone `gosx-cms` module. Top tier — imports `cms/{lifecycle,flows,style}` and `cms/studio/collab`; nothing in this module imports `cms/studio` back (that would close a cycle through the facade — see ARCHITECTURE.md). Ships its own forked runtime assets — see "Release model" below. |

## Quick start

1. Mount the embedded runtime assets and stylesheet: `hostruntime.MountRuntimes` /
   `hostruntime.DefaultRuntimeConfig`.
2. Configure the portal shell: `shell.ShellConfig` / `shell.DefaultShellConfig`
   (labels, modes, panels, resource links, engine globals, feature flags).
3. Wire the mutation boundary: `authoring.AuthoringActionHandler` against your
   `authoring.AuthoringAdapter` implementation.
4. Render the editor route with `shell.RenderBackendEditorPage`, and the CRUD
   routes with the `backoffice` package's per-domain `Render*` pages.

The canonical host adapter is muddy-noni-commerce's
`internal/studiohost/adapter.go`, which builds a `hostruntime.RuntimeConfig`
and `shell.ShellConfig` (host labels, resource links, panels, actions) end to
end.

## Versioning & compatibility

The root `studio` package is a **deprecated compatibility facade** for one
release cycle (v0.6.x): every symbol muddy, pajaritos, and `cms/studio` (the
former `gosx-cms/studio`) used before the package restructure still resolves
through a type alias or thin forwarding wrapper, each marked `// Deprecated:`.
New code should import the subpackages above directly (`core`, `authoring`,
`shell`, `canvas`, `sitemap`, `panels`, `backoffice`, `hostruntime`) rather
than the root.

The `gosx-cms` module fold-in (previously "Phase 2") landed: all 13 `gosx-cms`
packages now live under `cms/` in this module, and `gosx-studio`'s own
`go.mod` no longer requires `gosx-cms` — see
[docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) §"Release model" for the folded
DAG shape and [cms/PROVENANCE.md](cms/PROVENANCE.md) for the path mapping and
source commit.

See `docs/WEBFLOW_CLASS_EDITOR_GAP_INVENTORY.md` for the detailed gap
inventory and work plan toward a Webflow-class no-code editor.
