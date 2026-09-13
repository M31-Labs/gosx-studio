// Package sitehost is the default host application for GoSX Studio.
//
// Studio has always been a library that a host application configures: the
// README states that hosts supply "adapters, persistence, permissions, routes,
// and copy." That framing is correct for an embedding library and fatal for a
// product aimed at non-technical site owners, because it puts roughly nine
// thousand lines of hand-written Go between a person and their own website.
//
// sitehost inverts the relationship. It assembles the storage, routing,
// document shell, and back-office surfaces Studio already ships into a program
// that runs. A site owner configures it with flags and environment variables;
// nobody writes Go. Applications that need more control keep using the
// packages directly — this is a floor, not a ceiling.
package sitehost

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"

	"m31labs.dev/gosx"
	cmsstore "m31labs.dev/gosx-studio/cms/store"
	"m31labs.dev/gosx-studio/hostruntime"
)

const (
	publicStylesheetPath = "/_gosx/site/site.css"
	adminPathPrefix      = "/admin"
)

// LifecycleContentStore is the storage contract the default host needs: full
// CRUD plus the draft/publish/restore surface. cms/store/file satisfies it.
type LifecycleContentStore interface {
	cmsstore.Store
	cmsstore.LifecycleStore
}

// Options configures the default host.
type Options struct {
	// DataPath is the file the site is stored in. Required. A ".json" path
	// uses the JSON snapshot store; anything else is a SQLite database,
	// created on first start and migrated from a site.json beside it if one
	// exists.
	DataPath string
	// SiteTitle seeds a new site and names it in the document shell.
	SiteTitle string
	// SiteDescription seeds the default meta description.
	SiteDescription string
	// BaseURL is the public origin, used for canonical and share URLs.
	BaseURL string
	// Seed builds a starter site immediately instead of sending the owner
	// through the setup wizard. Tests and scripted installs use it.
	Seed bool
	// SiteKind picks the starter template when Seed is set. See SiteKinds.
	SiteKind string
	// AdminPassword is the bootstrap secret on a real server: it gates the
	// form that creates the owner account, after which sign-in is by email
	// and password. Leave it empty only for a server bound to localhost.
	AdminPassword string
	// UploadDir is where pictures are stored. Defaults to an "uploads" folder
	// beside DataPath.
	UploadDir string
	// MailURL configures how the site sends email: smtp://, smtps://,
	// resend://, or postmark:// — see ParseMailURL. Empty means no email.
	MailURL string
	// Mailer overrides MailURL with a ready transport. Tests use it.
	Mailer Mailer
	// TLS reports that the process serves HTTPS with automatic certificates.
	// The admin's domain screen reads it; cmd/gosx-site sets it for -https.
	TLS bool
	// CertDir caches issued certificates. Defaults to a "certs" folder beside
	// DataPath.
	CertDir string
	// PublicIP is the address DNS records should point at, when known.
	PublicIP string
	// NoBackups turns off the daily backup into a "backups" folder beside
	// DataPath. Export on demand still works.
	NoBackups bool
	// LogRequests writes one JSON line per request to LogWriter (stderr by
	// default). Static assets are not logged.
	LogRequests bool
	// LogWriter receives request log lines when LogRequests is set.
	LogWriter io.Writer
}

func (o Options) normalize() Options {
	o.DataPath = strings.TrimSpace(o.DataPath)
	o.SiteTitle = strings.TrimSpace(o.SiteTitle)
	o.SiteDescription = strings.TrimSpace(o.SiteDescription)
	o.BaseURL = strings.TrimRight(strings.TrimSpace(o.BaseURL), "/")
	if o.SiteTitle == "" {
		o.SiteTitle = "My site"
	}
	return o
}

// Host is a running default site: one store, one HTTP handler.
type Host struct {
	store    LifecycleContentStore
	opts     Options
	messages *messageStore

	securityOnce sync.Once
	csrf         string
	secretOnce   sync.Once
	secret       []byte
	authFailures *rateLimiter

	mailer     Mailer
	mailStatus mailStatus

	media     *mediaIndex
	stats     *statsStore
	forms     *formStore
	products  *productStore
	orders    *orderStore
	bookings  *bookingStore
	carts     *cartStore
	users     *userStore
	auditLog  *auditStore
	ssoState  ssoCache
	metrics   *metrics
	logMu     sync.Mutex
	retention dueChecker
	reminders dueChecker
	due       dueChecker
	backups   backupState

	// draft marks the staging view: livePage and livePost answer with the
	// working record. See staging.go.
	draft bool

	// Migrated is the JSON file a SQLite site was created from on this
	// start, or empty.
	Migrated string
}

// Open loads or creates the site at Options.DataPath.
func Open(opts Options) (*Host, error) {
	opts = opts.normalize()
	if opts.DataPath == "" {
		return nil, errors.New("sitehost: DataPath is required")
	}

	store, migrated, err := openStore(opts.DataPath)
	if err != nil {
		return nil, fmt.Errorf("sitehost: open site data: %w", err)
	}

	host := &Host{store: store, opts: opts, messages: newMessageStore(opts.messagesPath()), authFailures: newRateLimiter(authFailLimit, authFailWindow), media: newMediaIndex(opts.uploadDir()), stats: newStatsStore(opts.statsPath()), forms: newFormStore(opts.formsPath()), products: newProductStore(opts.productsPath()), orders: newOrderStore(opts.ordersPath()), bookings: newBookingStore(opts.bookingsPath()), carts: newCartStore(opts.cartsPath()), users: newUserStore(opts.usersPath()), auditLog: newAuditStore(opts.auditPath()), metrics: newMetrics()}
	host.Migrated = migrated
	if err := host.configureMail(); err != nil {
		return nil, err
	}
	// Seeding is the wizard's job. Options.Seed exists so tests and embedders
	// can skip the wizard and get a site in one call.
	if opts.Seed && !host.SetupComplete() {
		if err := host.CompleteSetup(SetupAnswers{
			SiteTitle: opts.SiteTitle,
			Tagline:   opts.SiteDescription,
			Kind:      opts.SiteKind,
			BaseURL:   opts.BaseURL,
		}); err != nil {
			return nil, fmt.Errorf("sitehost: build starter site: %w", err)
		}
	}
	return host, nil
}

// NewWithStore builds a host around an existing store. Tests and embedders use
// this to supply in-memory storage.
func NewWithStore(store LifecycleContentStore, opts Options) *Host {
	opts = opts.normalize()
	host := &Host{store: store, opts: opts, messages: newMessageStore(opts.messagesPath()), authFailures: newRateLimiter(authFailLimit, authFailWindow), media: newMediaIndex(opts.uploadDir()), stats: newStatsStore(opts.statsPath()), forms: newFormStore(opts.formsPath()), products: newProductStore(opts.productsPath()), orders: newOrderStore(opts.ordersPath()), bookings: newBookingStore(opts.bookingsPath()), carts: newCartStore(opts.cartsPath()), users: newUserStore(opts.usersPath()), auditLog: newAuditStore(opts.auditPath()), metrics: newMetrics()}
	_ = host.configureMail()
	return host
}

// configureMail resolves the transport from Options.
func (h *Host) configureMail() error {
	if h.opts.Mailer != nil {
		h.mailer = h.opts.Mailer
		return nil
	}
	mailer, err := ParseMailURL(h.opts.MailURL)
	if err != nil {
		return fmt.Errorf("sitehost: %w", err)
	}
	h.mailer = mailer
	return nil
}

// Store exposes the underlying content store.
func (h *Host) Store() LifecycleContentStore { return h.store }

// Options returns the normalized configuration.
func (h *Host) Options() Options { return h.opts }

type muxMounter struct{ mux *http.ServeMux }

func (m muxMounter) Mount(pattern string, handler http.Handler) {
	m.mux.Handle(pattern, handler)
}

// Handler builds the complete site: runtime assets, the public site, and the
// back office.
func (h *Host) Handler() http.Handler {
	mux := h.routes()
	drafts := h.draftView().routes()
	// Outermost first: headers on everything, then the staging address,
	// then sign-in, then CSRF on what is signed in, then the setup gate,
	// then the routes. Staging only ever reads content, so it needs none of
	// the sign-in or CSRF layers.
	site := h.guardAdmin(h.requireCSRF(h.requireSetup(mux)))
	return h.observe(h.housekeeping(h.securityHeaders(h.hostRedirect(h.stagingGate(site, h.requireSetup(drafts))))))
}

// routes is every handler on one mux, for the site and for its staging
// view alike.
func (h *Host) routes() *http.ServeMux {
	mux := http.NewServeMux()

	// Studio's own editor runtime assets and stylesheet.
	hostruntime.MountRuntimes(muxMounter{mux})
	mux.Handle("GET "+publicStylesheetPath, publicStylesheetHandler())

	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	h.mountSetup(mux)
	h.mountAdmin(mux)
	h.mountEditor(mux)
	h.mountUploads(mux)
	h.mountMessages(mux)
	h.mountGrowth(mux)
	h.mountMedia(mux)
	h.mountBlog(mux)
	h.mountStats(mux)
	h.mountDomain(mux)
	h.mountHistory(mux)
	h.mountForms(mux)
	h.mountBackups(mux)
	h.mountShop(mux)
	h.mountCheckout(mux)
	h.mountAuth(mux)
	h.mountAudit(mux)
	h.mountReview(mux)
	h.mountSSO(mux)
	h.mountPrivacy(mux)
	h.mountCustomers(mux)
	h.mountStaging(mux)
	h.mountPublic(mux)
	return mux
}

// settings reads site settings, falling back to the configured defaults so the
// document shell always has a title and description.
func (h *Host) settings() cmsstore.SiteSettings {
	settings, ok, err := h.store.SiteSettings()
	if err != nil || !ok {
		return cmsstore.SiteSettings{
			Title:       h.opts.SiteTitle,
			Description: h.opts.SiteDescription,
			BaseURL:     h.opts.BaseURL,
		}
	}
	if strings.TrimSpace(settings.Title) == "" {
		settings.Title = h.opts.SiteTitle
	}
	if strings.TrimSpace(settings.BaseURL) == "" {
		settings.BaseURL = h.opts.BaseURL
	}
	return settings
}

func (h *Host) writeDocument(w http.ResponseWriter, status int, meta PageMeta, body gosx.Node) {
	if meta.Theme.Palette.Key == "" {
		meta.Theme = h.theme()
	}
	if meta.AdminChrome {
		meta.CSRF = h.csrfToken()
	}
	if h.draft && !meta.AdminChrome {
		meta.NoIndex = true
		body = gosx.Fragment(h.stagingBanner(), body)
	}
	// The visitor beacon goes on public pages that were actually served,
	// never on the admin, a 404, or the wizard.
	meta.Stats = !meta.AdminChrome && !meta.NoIndex && status == http.StatusOK && h.SetupComplete() && h.statsEnabled()
	if meta.Favicon == "" {
		settings := h.settings()
		meta.Favicon = brandFromSettings(settings).FaviconHref(meta.Theme, firstNonEmpty(settings.Title, h.opts.SiteTitle))
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	_, _ = w.Write([]byte(RenderDocument(meta, body)))
}
