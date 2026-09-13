// Command gosx-site runs a complete GoSX Studio website with no Go code.
//
//	go run m31labs.dev/gosx-studio/cmd/gosx-site
//
// The first run creates a small published site and prints where to edit it.
// Everything is stored in one JSON file, so a site is a file you can copy,
// back up, or check into version control.
package main

import (
	"context"
	"crypto/tls"
	"errors"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"m31labs.dev/gosx-studio/sitehost"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "gosx-site: "+err.Error())
		os.Exit(1)
	}
}

func run() error {
	var (
		addr     = flag.String("addr", env("GOSX_SITE_ADDR", "127.0.0.1:8080"), "address to listen on")
		dataPath = flag.String("data", env("GOSX_SITE_DATA", filepath.Join("data", "site.db")), "path to the site's data file (a .db is SQLite, the default; a .json is the one-file snapshot)")
		title    = flag.String("title", env("GOSX_SITE_TITLE", "My site"), "site name, used until you change it in Settings")
		desc     = flag.String("description", env("GOSX_SITE_DESCRIPTION", ""), "one-line site description for search results")
		baseURL  = flag.String("base-url", env("GOSX_SITE_BASE_URL", ""), "public address, e.g. https://yourbusiness.com")
		password = flag.String("admin-password", env("GOSX_SITE_ADMIN_PASSWORD", ""), "password for the admin area (required off localhost)")
		mailURL  = flag.String("mail", env("GOSX_SITE_MAIL", ""), "email transport: smtp://user:pass@host:587?from=you@example.com, resend://KEY?from=..., or postmark://TOKEN?from=...")
		https    = flag.Bool("https", env("GOSX_SITE_HTTPS", "") == "true", "serve HTTPS on :443 with automatic Let's Encrypt certificates (and redirect :80); connect a domain in the admin first")
		certDir  = flag.String("cert-dir", env("GOSX_SITE_CERT_DIR", ""), "where certificates are cached (default: a certs folder beside the data file)")
		publicIP = flag.String("public-ip", env("GOSX_SITE_PUBLIC_IP", ""), "this server's public IP address, shown in the domain instructions")
		noBackup = flag.Bool("no-backups", env("GOSX_SITE_NO_BACKUPS", "") == "true", "turn off the daily backup zip written beside the data file")
		logReqs  = flag.Bool("log-requests", env("GOSX_SITE_LOG_REQUESTS", "true") != "false", "write one JSON line per request to stderr")
		features = flag.String("features", env("GOSX_SITE_FEATURES", ""), "what the site's plan includes, comma-separated: blog,forms,shop,stats,domain,team,staging,sso (empty: everything)")
		managed  = flag.String("managed-by", env("GOSX_SITE_MANAGED_BY", ""), "name of the hosting platform running this site, when one does")
		domain   = flag.String("domain", env("GOSX_SITE_DOMAIN", ""), "the owner's domain as connected by the platform")
		opToken  = flag.String("operator-token", env("GOSX_SITE_OPERATOR_TOKEN", ""), "token the platform uses for /platform/status and /platform/export.zip")
		agentKey = flag.String("agent-key", env("GOSX_SITE_AGENT_KEY", ""), "pre-provisioned agent key(s), comma-separated, so an agent can build the site before anyone signs in")
	)
	if len(os.Args) > 1 && os.Args[1] == "mcp" {
		return runMCP(os.Args[2:])
	}
	flag.Parse()

	if *https && *addr == "127.0.0.1:8080" {
		*addr = ":443"
	}
	if err := checkAdminExposure(*addr, *password); err != nil {
		return err
	}

	// Claim the port before touching the site data, so a busy port fails
	// without creating or seeding a half-started site.
	listener, err := net.Listen("tcp", *addr)
	if err != nil {
		return fmt.Errorf("listen on %s: %w", *addr, err)
	}
	defer listener.Close()

	// HTTPS also needs port 80: Let's Encrypt checks the domain there, and
	// anyone typing the address without https:// is sent to the secure one.
	var challengeListener net.Listener
	if *https {
		challengeListener, err = net.Listen("tcp", ":80")
		if err != nil {
			return fmt.Errorf("listen on :80 for certificate checks and http to https redirects: %w", err)
		}
		defer challengeListener.Close()
	}

	host, err := sitehost.Open(sitehost.Options{
		DataPath:        *dataPath,
		SiteTitle:       *title,
		SiteDescription: *desc,
		BaseURL:         *baseURL,
		AdminPassword:   *password,
		MailURL:         *mailURL,
		TLS:             *https,
		CertDir:         *certDir,
		PublicIP:        *publicIP,
		NoBackups:       *noBackup,
		LogRequests:     *logReqs,
		Features:        sitehost.ParseFeatures(*features),
		ManagedBy:       *managed,
		Domain:          *domain,
		OperatorToken:   *opToken,
		AgentKeys:       splitList(*agentKey),
	})
	if err != nil {
		return err
	}

	server := &http.Server{
		Addr:              *addr,
		Handler:           host.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
	}
	var challengeServer *http.Server
	if *https {
		manager := host.TLSManager()
		listener = tls.NewListener(listener, manager.TLSConfig())
		challengeServer = &http.Server{Addr: ":80", Handler: manager.HTTPHandler(nil), ReadHeaderTimeout: 10 * time.Second}
		go func() { _ = challengeServer.Serve(challengeListener) }()
	}

	if host.Migrated != "" {
		fmt.Println("  Moved your site from " + host.Migrated + " into " + *dataPath + " (the old file is kept as " + host.Migrated + ".migrated).")
	}
	printWelcome(listener.Addr().String(), *dataPath, *password != "", *https, host.Domain())

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)
	serveErr := make(chan error, 1)
	go func() {
		if err := server.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serveErr <- err
			return
		}
		serveErr <- nil
	}()

	select {
	case err := <-serveErr:
		return err
	case <-shutdown:
		fmt.Println("\nStopping. Your site is saved.")
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if challengeServer != nil {
			_ = challengeServer.Shutdown(ctx)
		}
		return server.Shutdown(ctx)
	}
}

// checkAdminExposure refuses the one combination that would publish an
// unprotected admin area: a public listen address with no password.
func checkAdminExposure(addr, password string) error {
	if strings.TrimSpace(password) != "" || isLoopbackAddr(addr) {
		return nil
	}
	return errors.New("refusing to start: " + addr + " is reachable from the network but the admin area has no password.\n" +
		"  Set one with -admin-password or GOSX_SITE_ADMIN_PASSWORD,\n" +
		"  or listen on localhost only with -addr 127.0.0.1:8080")
}

func isLoopbackAddr(addr string) bool {
	host, _, err := net.SplitHostPort(strings.TrimSpace(addr))
	if err != nil {
		host = strings.TrimSpace(addr)
	}
	if host == "" {
		return false // ":8080" binds every interface
	}
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func printWelcome(addr, dataPath string, guarded, https bool, domain string) {
	// Bound to every interface, the listener reports "[::]" or "0.0.0.0",
	// which is not an address a person can type. Show one that is.
	if host, port, err := net.SplitHostPort(addr); err == nil {
		if ip := net.ParseIP(host); host == "" || (ip != nil && ip.IsUnspecified()) {
			addr = net.JoinHostPort("localhost", port)
		}
	}
	base := "http://" + addr
	if https {
		base = "https://" + strings.TrimSuffix(addr, ":443")
		if domain != "" {
			base = "https://" + domain
		}
	}
	if abs, err := filepath.Abs(dataPath); err == nil {
		dataPath = abs
	}

	fmt.Println()
	fmt.Println("  Your site is running.")
	fmt.Println()
	fmt.Println("  Visit your site      " + base + "/")
	fmt.Println("  Edit your site       " + base + "/admin")
	fmt.Println("  Saved in             " + dataPath)
	if guarded {
		fmt.Println("  Admin sign-in        open " + base + "/admin/login and create the owner account; the password you set unlocks that form")
	} else {
		fmt.Println("  Admin sign-in        not required (this server only accepts local connections)")
	}
	if https && domain == "" {
		fmt.Println("  HTTPS                on, waiting for a domain: connect one under Settings, Your own domain")
	} else if https {
		fmt.Println("  HTTPS                on, certificates from Let's Encrypt are automatic")
	}
	fmt.Println()
	fmt.Println("  Press Ctrl+C to stop.")
	fmt.Println()
}

func env(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

// runMCP bridges a stdio MCP client (Claude Desktop, an IDE) to a site's
// agent API over HTTP: gosx-site mcp -site https://example.com -key gsk_…
func runMCP(args []string) error {
	set := flag.NewFlagSet("mcp", flag.ContinueOnError)
	site := set.String("site", env("GOSX_SITE_URL", ""), "the site's address, e.g. https://yourbusiness.com")
	key := set.String("key", env("GOSX_SITE_AGENT_KEY", ""), "an agent key from the site's Agents page")
	if err := set.Parse(args); err != nil {
		return err
	}
	if *site == "" {
		return errors.New("mcp: -site is required (or set GOSX_SITE_URL)")
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	return sitehost.RunMCPStdio(ctx, os.Stdin, os.Stdout, *site, *key, &http.Client{Timeout: 90 * time.Second})
}

func splitList(raw string) []string {
	out := []string{}
	for _, part := range strings.Split(raw, ",") {
		if part = strings.TrimSpace(part); part != "" {
			out = append(out, part)
		}
	}
	return out
}
