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
		dataPath = flag.String("data", env("GOSX_SITE_DATA", filepath.Join("data", "site.json")), "path to the site's data file")
		title    = flag.String("title", env("GOSX_SITE_TITLE", "My site"), "site name, used until you change it in Settings")
		desc     = flag.String("description", env("GOSX_SITE_DESCRIPTION", ""), "one-line site description for search results")
		baseURL  = flag.String("base-url", env("GOSX_SITE_BASE_URL", ""), "public address, e.g. https://yourbusiness.com")
		password = flag.String("admin-password", env("GOSX_SITE_ADMIN_PASSWORD", ""), "password for the admin area (required off localhost)")
	)
	flag.Parse()

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

	host, err := sitehost.Open(sitehost.Options{
		DataPath:        *dataPath,
		SiteTitle:       *title,
		SiteDescription: *desc,
		BaseURL:         *baseURL,
		AdminPassword:   *password,
	})
	if err != nil {
		return err
	}

	server := &http.Server{
		Addr:              *addr,
		Handler:           host.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
	}

	printWelcome(listener.Addr().String(), *dataPath, *password != "")

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

func printWelcome(addr, dataPath string, guarded bool) {
	base := "http://" + addr
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
		fmt.Println("  Admin sign-in        username \"" + sitehost.AdminUser + "\" with the password you set")
	} else {
		fmt.Println("  Admin sign-in        not required (this server only accepts local connections)")
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
