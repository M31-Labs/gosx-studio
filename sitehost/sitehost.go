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
	"net/http"
	"strings"

	"m31labs.dev/gosx"
	cmsstore "m31labs.dev/gosx-studio/cms/store"
	"m31labs.dev/gosx-studio/cms/store/file"
	"m31labs.dev/gosx-studio/cms/store/memory"
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
	// DataPath is the JSON snapshot the site is stored in. Required.
	DataPath string
	// SiteTitle seeds a new site and names it in the document shell.
	SiteTitle string
	// SiteDescription seeds the default meta description.
	SiteDescription string
	// BaseURL is the public origin, used for canonical and share URLs.
	BaseURL string
	// SkipSeed leaves an empty store empty instead of writing starter content.
	SkipSeed bool
	// AdminPassword protects every /admin path with HTTP basic auth when set.
	// Leave it empty only for a server bound to localhost.
	AdminPassword string
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
	store LifecycleContentStore
	opts  Options
}

// Open loads or creates the site at Options.DataPath.
func Open(opts Options) (*Host, error) {
	opts = opts.normalize()
	if opts.DataPath == "" {
		return nil, errors.New("sitehost: DataPath is required")
	}

	store, err := file.Open(opts.DataPath)
	if err != nil {
		store, err = file.New(opts.DataPath, memory.Seed{})
		if err != nil {
			return nil, fmt.Errorf("sitehost: open site data: %w", err)
		}
	}

	host := &Host{store: store, opts: opts}
	if !opts.SkipSeed {
		if err := SeedStarterSite(store, opts.SiteTitle, opts.SiteDescription, opts.BaseURL); err != nil {
			return nil, fmt.Errorf("sitehost: seed starter site: %w", err)
		}
	}
	return host, nil
}

// NewWithStore builds a host around an existing store. Tests and embedders use
// this to supply in-memory storage.
func NewWithStore(store LifecycleContentStore, opts Options) *Host {
	return &Host{store: store, opts: opts.normalize()}
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
	mux := http.NewServeMux()

	// Studio's own editor runtime assets and stylesheet.
	hostruntime.MountRuntimes(muxMounter{mux})
	mux.Handle("GET "+publicStylesheetPath, publicStylesheetHandler())

	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	h.mountAdmin(mux)
	h.mountPublic(mux)

	return h.guardAdmin(mux)
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
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	_, _ = w.Write([]byte(RenderDocument(meta, body)))
}
