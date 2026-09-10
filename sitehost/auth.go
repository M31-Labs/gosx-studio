package sitehost

import (
	"crypto/subtle"
	"net/http"
	"strings"
)

// auth.go guards the back office.
//
// The default host ships a working admin area, so it must not ship an open one.
// The rule is simple and stated in one place: when AdminPassword is set, every
// /admin path needs it. The binary refuses to listen on a public address
// without one, so the unsafe combination cannot be reached by accident.

const adminRealm = `Basic realm="Site admin", charset="UTF-8"`

// AdminUser is the username the back office expects.
const AdminUser = "admin"

// guardAdmin wraps a handler so /admin paths require the configured password.
func (h *Host) guardAdmin(next http.Handler) http.Handler {
	password := strings.TrimSpace(h.opts.AdminPassword)
	if password == "" {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !isAdminPath(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}
		user, pass, ok := r.BasicAuth()
		if !ok || !secureEqual(user, AdminUser) || !secureEqual(pass, password) {
			w.Header().Set("WWW-Authenticate", adminRealm)
			http.Error(w, "Sign in to manage this site.", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func isAdminPath(path string) bool {
	return path == adminPathPrefix || strings.HasPrefix(path, adminPathPrefix+"/")
}

func secureEqual(got, want string) bool {
	return subtle.ConstantTimeCompare([]byte(got), []byte(want)) == 1
}
