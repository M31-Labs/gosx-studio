package sitehost

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"m31labs.dev/gosx"
)

// auth.go is sign-in.
//
// A site starts with no accounts. On a laptop with no server password it is
// open, as before. On a real server the -admin-password gates one thing: the
// form that creates the owner account. From then on every admin request
// needs a session cookie, and the password on the command line is nothing
// but a bootstrap. Admin-only paths are listed in one table; everything
// else under /admin is for every signed-in person.

const (
	loginPath     = "/admin/login"
	loginCodePath = "/admin/login/code"
	logoutPath    = "/admin/logout"
	joinPrefix    = "/admin/join/"
	accountPath   = "/admin/account"
	peoplePath    = "/admin/users"
	pendingCookie = "gosx_2fa"
	pendingTTL    = 10 * time.Minute
)

// AdminUser is kept for the command's welcome text; sign-in is by email.
const AdminUser = "admin"

// adminOnlyPrefixes need an admin or the owner. Everything else under
// /admin is open to editors.
var adminOnlyPrefixes = []string{
	"/admin/settings", "/admin/domain", "/admin/backups", "/admin/export.zip",
	"/admin/users", "/admin/orders", agentsAdminPath, stagingAdminPath, "/admin/api/theme", "/admin/metrics", "/admin/messages/prune", "/admin/messages/export.csv", "/admin/privacy-page",
}

func adminOnly(path string) bool {
	for _, prefix := range adminOnlyPrefixes {
		if path == prefix || strings.HasPrefix(path, prefix+"/") {
			return true
		}
	}
	return false
}

type userKey struct{}

// currentUser is the signed-in person, if any.
func (h *Host) currentUser(r *http.Request) (User, bool) {
	user, ok := r.Context().Value(userKey{}).(User)
	return user, ok
}

// roleAtLeast reports whether the request may do what the role may do. With
// no accounts yet (a laptop site) the answer is always yes.
func (h *Host) roleAtLeast(r *http.Request, role string) bool {
	if h.users.count() == 0 {
		return true
	}
	user, ok := h.currentUser(r)
	return ok && roleRank(user.Role) >= roleRank(role)
}

func isPublicAdminPath(path string) bool {
	return path == loginPath || path == loginCodePath || strings.HasPrefix(path, joinPrefix) || strings.HasPrefix(path, "/admin/sso/")
}

// guardAdmin decides who gets into /admin.
func (h *Host) guardAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !isAdminPath(r.URL.Path) || isPublicAdminPath(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}
		if h.users.count() == 0 {
			if strings.TrimSpace(h.opts.AdminPassword) == "" {
				next.ServeHTTP(w, r) // a laptop site: open, as before
				return
			}
			h.sendToLogin(w, r)
			return
		}
		user, ok := h.sessionUser(r)
		if !ok {
			h.sendToLogin(w, r)
			return
		}
		if adminOnly(r.URL.Path) && roleRank(user.Role) < roleRank(roleAdmin) {
			h.writeForbidden(w, r, user)
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), userKey{}, user)))
	})
}

func (h *Host) sessionUser(r *http.Request) (User, bool) {
	cookie, err := r.Cookie(sessionCookie)
	if err != nil {
		return User{}, false
	}
	return h.users.userForSession(cookie.Value)
}

func (h *Host) sendToLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		if strings.HasPrefix(r.URL.Path, "/admin/api/") {
			writeJSON(w, http.StatusUnauthorized, editorSaveResult{Message: "You're signed out. Sign in again to keep editing."})
			return
		}
		http.Error(w, "Sign in to manage this site.", http.StatusUnauthorized)
		return
	}
	next := ""
	if r.URL.Path != adminPathPrefix && r.URL.Path != adminPathPrefix+"/" {
		next = "?next=" + url.QueryEscape(r.URL.RequestURI())
	}
	http.Redirect(w, r, loginPath+next, http.StatusSeeOther)
}

func (h *Host) writeForbidden(w http.ResponseWriter, r *http.Request, user User) {
	if strings.HasPrefix(r.URL.Path, "/admin/api/") || r.Method != http.MethodGet {
		http.Error(w, "That part of the site is for admins.", http.StatusForbidden)
		return
	}
	body := h.renderAdminShell("", "Admins only",
		"", adminStatus{}, gosx.El("section", gosx.Attrs(gosx.Attr("class", "admin-empty")),
			gosx.El("h2", nil, gosx.Text("This part is for admins")),
			gosx.El("p", nil, gosx.Text("You're signed in as "+user.Name+" ("+user.roleLabel()+"). Settings, payments, the domain, orders, and people are managed by the site's admins. Ask them if you need a change here.")),
			gosx.El("p", nil, gosx.El("a", gosx.Attrs(gosx.Attr("class", "admin-button"), gosx.Attr("href", "/admin")), gosx.Text("Back to your dashboard"))),
		))
	w.WriteHeader(http.StatusForbidden)
	_, _ = w.Write([]byte(RenderDocument(h.withDefaults(h.adminMeta("Admins only")), body)))
}

// withDefaults fills the parts of PageMeta writeDocument would.
func (h *Host) withDefaults(meta PageMeta) PageMeta {
	meta.Theme = h.theme()
	meta.CSRF = h.csrfToken()
	settings := h.settings()
	meta.Favicon = brandFromSettings(settings).FaviconHref(meta.Theme, firstNonEmpty(settings.Title, h.opts.SiteTitle))
	return meta
}

func (h *Host) mountAuth(mux *http.ServeMux) {
	mux.HandleFunc("GET "+loginPath, h.handleLogin)
	mux.HandleFunc("POST "+loginPath, h.handleLoginSubmit)
	mux.HandleFunc("GET "+loginCodePath, h.handleLoginCode)
	mux.HandleFunc("POST "+loginCodePath, h.handleLoginCodeSubmit)
	mux.HandleFunc("POST "+logoutPath, h.handleLogout)
	mux.HandleFunc("GET "+joinPrefix+"{token}", h.handleJoin)
	mux.HandleFunc("POST "+joinPrefix+"{token}", h.handleJoinSubmit)
	mux.HandleFunc("GET "+accountPath, h.handleAccount)
	mux.HandleFunc("POST "+accountPath, h.handleAccountAction)
	mux.HandleFunc("GET "+peoplePath, h.handlePeople)
	mux.HandleFunc("POST "+peoplePath, h.handlePeopleAction)
}

func setSessionCookie(w http.ResponseWriter, r *http.Request, token string, maxAge int) {
	http.SetCookie(w, &http.Cookie{
		Name: sessionCookie, Value: token, Path: "/admin", HttpOnly: true, SameSite: http.SameSiteLaxMode,
		Secure: r.TLS != nil || strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https"), MaxAge: maxAge,
	})
}

func (h *Host) signIn(w http.ResponseWriter, r *http.Request, user User) error {
	token, err := h.users.newSession(user.ID)
	if err != nil {
		return err
	}
	setSessionCookie(w, r, token, int(sessionTTL/time.Second))
	return nil
}

func safeNext(raw string) string {
	raw = strings.TrimSpace(raw)
	if !strings.HasPrefix(raw, "/admin") || strings.HasPrefix(raw, "//") || strings.ContainsAny(raw, "\r\n") {
		return "/admin"
	}
	return raw
}

// ---------- the sign-in page ----------

func (h *Host) renderAuthPage(w http.ResponseWriter, status int, title string, problem string, panel gosx.Node) {
	meta := h.adminMeta(title)
	meta.AdminChrome = false
	meta.NoIndex = true
	body := gosx.El("div", gosx.Attrs(gosx.Attr("class", "wz")),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "wz-card wz-card--narrow")),
			problemNote(problem),
			panel,
		),
	)
	h.writeDocument(w, status, meta, body)
}

func (h *Host) handleLogin(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.sessionUser(r); ok {
		http.Redirect(w, r, "/admin", http.StatusSeeOther)
		return
	}
	if h.users.count() == 0 {
		h.renderCreateOwner(w, http.StatusOK, "", r.FormValue("email"), r.FormValue("name"))
		return
	}
	h.renderLogin(w, http.StatusOK, "", r.URL.Query().Get("email"), r.URL.Query().Get("next"))
}

func (h *Host) renderLogin(w http.ResponseWriter, status int, problem, email, next string) {
	siteTitle := firstNonEmpty(h.settings().Title, h.opts.SiteTitle)
	panel := gosx.El("form", gosx.Attrs(gosx.Attr("method", "post"), gosx.Attr("action", loginPath), gosx.Attr("class", "wz-form")),
		gosx.El("h1", nil, gosx.Text("Sign in to "+siteTitle)),
		wizardField("email", "Email", email, "you@example.com", false),
		gosx.El("label", gosx.Attrs(gosx.Attr("class", "wz-field"), gosx.Attr("for", "wz-password")),
			gosx.El("span", nil, gosx.Text("Password")),
			gosx.El("input", gosx.Attrs(gosx.Attr("type", "password"), gosx.Attr("id", "wz-password"), gosx.Attr("name", "password"), gosx.Attr("autocomplete", "current-password"), gosx.Attr("required", "required")))),
		hidden("next", next),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "wz-actions")),
			gosx.El("button", gosx.Attrs(gosx.Attr("class", "wz-btn"), gosx.Attr("type", "submit")), gosx.Text("Sign in"))),
		h.ssoButton(next),
		gosx.El("p", gosx.Attrs(gosx.Attr("class", "wz-hint")), gosx.Text("Forgotten your password? Another admin can send you a fresh invite from People.")),
	)
	h.renderAuthPage(w, status, "Sign in", problem, panel)
}

func (h *Host) renderCreateOwner(w http.ResponseWriter, status int, problem, email, name string) {
	siteTitle := firstNonEmpty(h.settings().Title, h.opts.SiteTitle)
	fields := []gosx.Node{
		gosx.El("h1", nil, gosx.Text("Create your account")),
		gosx.El("p", gosx.Attrs(gosx.Attr("class", "wz-lede")), gosx.Text("This is the owner account for "+siteTitle+". You can invite others afterwards.")),
		wizardField("name", "Your name", name, "Ana Lopez", false),
		wizardField("email", "Email", email, "you@example.com", false),
		passwordField("password", "Password", "new-password", "At least "+strconv.Itoa(minPasswordLen)+" characters."),
	}
	if strings.TrimSpace(h.opts.AdminPassword) != "" {
		fields = append(fields, passwordField("serverPassword", "The password this site was started with", "off", "The one given on the command line, as -admin-password. It proves you run this server."))
	}
	fields = append(fields, gosx.El("div", gosx.Attrs(gosx.Attr("class", "wz-actions")),
		gosx.El("button", gosx.Attrs(gosx.Attr("class", "wz-btn"), gosx.Attr("type", "submit")), gosx.Text("Create account"))))
	panel := gosx.El("form", gosx.Attrs(gosx.Attr("method", "post"), gosx.Attr("action", loginPath), gosx.Attr("class", "wz-form")), gosx.Fragment(fields...))
	h.renderAuthPage(w, status, "Create your account", problem, panel)
}

func passwordField(name, label, autocomplete, hint string) gosx.Node {
	return gosx.El("label", gosx.Attrs(gosx.Attr("class", "wz-field"), gosx.Attr("for", "wz-"+name)),
		gosx.El("span", nil, gosx.Text(label)),
		gosx.El("input", gosx.Attrs(gosx.Attr("type", "password"), gosx.Attr("id", "wz-"+name), gosx.Attr("name", name), gosx.Attr("autocomplete", autocomplete), gosx.Attr("required", "required"))),
		gosx.El("small", gosx.Attrs(gosx.Attr("class", "wz-hint")), gosx.Text(hint)),
	)
}

func (h *Host) handleLoginSubmit(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	host := remoteHost(r)
	now := timeNow()
	email := normalizeEmail(r.PostFormValue("email"))
	password := r.PostFormValue("password")
	if h.authFailures != nil && h.authFailures.blocked(host, now) {
		const locked = "Too many tries. Wait ten minutes and try again."
		if h.users.count() == 0 {
			h.renderCreateOwner(w, http.StatusTooManyRequests, locked, email, r.PostFormValue("name"))
		} else {
			h.renderLogin(w, http.StatusTooManyRequests, locked, email, r.PostFormValue("next"))
		}
		return
	}

	if h.users.count() == 0 {
		name := strings.TrimSpace(r.PostFormValue("name"))
		switch {
		case name == "":
			h.renderCreateOwner(w, http.StatusOK, "Add your name.", email, name)
			return
		case !validEmail(email):
			h.renderCreateOwner(w, http.StatusOK, "That email address doesn't look right.", email, name)
			return
		case passwordProblem(password) != "":
			h.renderCreateOwner(w, http.StatusOK, passwordProblem(password), email, name)
			return
		}
		if server := strings.TrimSpace(h.opts.AdminPassword); server != "" && !secureEqual(r.PostFormValue("serverPassword"), server) {
			if h.authFailures != nil {
				h.authFailures.record(host, now)
			}
			h.renderCreateOwner(w, http.StatusOK, "That isn't the password this site was started with.", email, name)
			return
		}
		hash, err := hashPassword(password)
		if err != nil {
			h.renderCreateOwner(w, http.StatusOK, "Something went wrong. Try again.", email, name)
			return
		}
		user, err := h.users.put(User{Email: email, Name: name, Role: roleOwner, PasswordHash: hash})
		if err != nil {
			h.renderCreateOwner(w, http.StatusOK, "Something went wrong. Try again.", email, name)
			return
		}
		if err := h.signIn(w, r, user); err != nil {
			h.renderCreateOwner(w, http.StatusOK, "Something went wrong. Try again.", email, name)
			return
		}
		h.audit(r, user, "account.created", "Created the owner account")
		http.Redirect(w, r, "/admin?welcome=1", http.StatusSeeOther)
		return
	}

	user, ok := h.users.byEmail(email)
	if !ok || !checkPassword(user.PasswordHash, password) {
		if h.authFailures != nil {
			h.authFailures.record(host, now)
		}
		h.renderLogin(w, http.StatusOK, "That email and password don't match.", email, r.PostFormValue("next"))
		return
	}
	if h.authFailures != nil {
		h.authFailures.clear(host)
	}
	next := safeNext(r.PostFormValue("next"))
	if user.TOTPSecret != "" {
		h.setPending(w, r, user.ID)
		http.Redirect(w, r, loginCodePath+"?next="+url.QueryEscape(next), http.StatusSeeOther)
		return
	}
	if err := h.signIn(w, r, user); err != nil {
		h.renderLogin(w, http.StatusOK, "Something went wrong. Try again.", email, next)
		return
	}
	h.audit(r, user, "signin", "Signed in")
	http.Redirect(w, r, next, http.StatusSeeOther)
}

// The pending cookie carries "user:expiry:mac" between the password step
// and the code step; it is signed with the install secret.
func (h *Host) setPending(w http.ResponseWriter, r *http.Request, userID string) {
	expiry := strconv.FormatInt(timeNow().Add(pendingTTL).Unix(), 10)
	value := userID + ":" + expiry + ":" + h.sign(userID+":"+expiry)
	http.SetCookie(w, &http.Cookie{Name: pendingCookie, Value: value, Path: "/admin", HttpOnly: true, SameSite: http.SameSiteLaxMode,
		Secure: r.TLS != nil || strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https"), MaxAge: int(pendingTTL / time.Second)})
}

func (h *Host) pendingUser(r *http.Request) (User, bool) {
	cookie, err := r.Cookie(pendingCookie)
	if err != nil {
		return User{}, false
	}
	parts := strings.Split(cookie.Value, ":")
	if len(parts) != 3 || !tokensEqual(parts[2], h.sign(parts[0]+":"+parts[1])) {
		return User{}, false
	}
	expiry, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil || timeNow().Unix() > expiry {
		return User{}, false
	}
	return h.users.byID(parts[0])
}

func (h *Host) sign(value string) string {
	mac := hmac.New(sha256.New, h.installSecret())
	mac.Write([]byte(value))
	return hex.EncodeToString(mac.Sum(nil))
}

func (h *Host) handleLoginCode(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.pendingUser(r); !ok {
		http.Redirect(w, r, loginPath, http.StatusSeeOther)
		return
	}
	h.renderLoginCode(w, http.StatusOK, "", r.URL.Query().Get("next"))
}

func (h *Host) renderLoginCode(w http.ResponseWriter, status int, problem, next string) {
	panel := gosx.El("form", gosx.Attrs(gosx.Attr("method", "post"), gosx.Attr("action", loginCodePath), gosx.Attr("class", "wz-form")),
		gosx.El("h1", nil, gosx.Text("Enter your code")),
		gosx.El("p", gosx.Attrs(gosx.Attr("class", "wz-lede")), gosx.Text("Open your authenticator app and type the six-digit code it shows for this site.")),
		gosx.El("label", gosx.Attrs(gosx.Attr("class", "wz-field"), gosx.Attr("for", "wz-code")),
			gosx.El("span", nil, gosx.Text("Code")),
			gosx.El("input", gosx.Attrs(gosx.Attr("type", "text"), gosx.Attr("id", "wz-code"), gosx.Attr("name", "code"), gosx.Attr("inputmode", "numeric"), gosx.Attr("autocomplete", "one-time-code"), gosx.Attr("required", "required"), gosx.Attr("autofocus", "autofocus")))),
		hidden("next", next),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "wz-actions")),
			gosx.El("button", gosx.Attrs(gosx.Attr("class", "wz-btn"), gosx.Attr("type", "submit")), gosx.Text("Sign in"))),
	)
	h.renderAuthPage(w, status, "Enter your code", problem, panel)
}

func (h *Host) handleLoginCodeSubmit(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	user, ok := h.pendingUser(r)
	if !ok {
		http.Redirect(w, r, loginPath, http.StatusSeeOther)
		return
	}
	host := remoteHost(r)
	now := timeNow()
	if h.authFailures != nil && h.authFailures.blocked(host, now) {
		h.renderLoginCode(w, http.StatusTooManyRequests, "Too many tries. Wait ten minutes and try again.", r.PostFormValue("next"))
		return
	}
	if !verifyTOTP(user.TOTPSecret, r.PostFormValue("code"), now) {
		if h.authFailures != nil {
			h.authFailures.record(host, now)
		}
		h.renderLoginCode(w, http.StatusOK, "That code didn't match. Codes change every 30 seconds; try the current one.", r.PostFormValue("next"))
		return
	}
	http.SetCookie(w, &http.Cookie{Name: pendingCookie, Value: "", Path: "/admin", MaxAge: -1})
	if err := h.signIn(w, r, user); err != nil {
		h.renderLoginCode(w, http.StatusOK, "Something went wrong. Try again.", r.PostFormValue("next"))
		return
	}
	h.audit(r, user, "signin", "Signed in with a code")
	http.Redirect(w, r, safeNext(r.PostFormValue("next")), http.StatusSeeOther)
}

func (h *Host) handleLogout(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie(sessionCookie); err == nil {
		h.users.dropSession(cookie.Value)
	}
	setSessionCookie(w, r, "", -1)
	http.Redirect(w, r, loginPath, http.StatusSeeOther)
}

// ---------- joining by invite ----------

func (h *Host) handleJoin(w http.ResponseWriter, r *http.Request) {
	invite, ok := h.users.inviteByToken(r.PathValue("token"))
	if !ok {
		h.renderAuthPage(w, http.StatusNotFound, "Invite not found", "", gosx.Fragment(
			gosx.El("h1", nil, gosx.Text("That invite has expired")),
			gosx.El("p", gosx.Attrs(gosx.Attr("class", "wz-lede")), gosx.Text("Invites last seven days and work once. Ask whoever invited you to send another."))))
		return
	}
	h.renderJoin(w, http.StatusOK, "", r.PathValue("token"), invite, "")
}

func (h *Host) renderJoin(w http.ResponseWriter, status int, problem, token string, invite Invite, name string) {
	siteTitle := firstNonEmpty(h.settings().Title, h.opts.SiteTitle)
	panel := gosx.El("form", gosx.Attrs(gosx.Attr("method", "post"), gosx.Attr("action", joinPrefix+token), gosx.Attr("class", "wz-form")),
		gosx.El("h1", nil, gosx.Text("Join "+siteTitle)),
		gosx.El("p", gosx.Attrs(gosx.Attr("class", "wz-lede")), gosx.Text("You've been invited as "+strings.ToLower(User{Role: invite.Role}.roleLabel())+" for "+invite.Email+". Choose a password to finish.")),
		wizardField("name", "Your name", name, "Ana Lopez", false),
		passwordField("password", "Password", "new-password", "At least "+strconv.Itoa(minPasswordLen)+" characters."),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "wz-actions")),
			gosx.El("button", gosx.Attrs(gosx.Attr("class", "wz-btn"), gosx.Attr("type", "submit")), gosx.Text("Join"))),
	)
	h.renderAuthPage(w, status, "Join", problem, panel)
}

func (h *Host) handleJoinSubmit(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	token := r.PathValue("token")
	invite, ok := h.users.inviteByToken(token)
	if !ok {
		http.Redirect(w, r, joinPrefix+token, http.StatusSeeOther)
		return
	}
	name := strings.TrimSpace(r.PostFormValue("name"))
	password := r.PostFormValue("password")
	if name == "" {
		h.renderJoin(w, http.StatusOK, "Add your name.", token, invite, name)
		return
	}
	if problem := passwordProblem(password); problem != "" {
		h.renderJoin(w, http.StatusOK, problem, token, invite, name)
		return
	}
	hash, err := hashPassword(password)
	if err != nil {
		h.renderJoin(w, http.StatusOK, "Something went wrong. Try again.", token, invite, name)
		return
	}
	user := User{Email: invite.Email, Name: name, Role: invite.Role, PasswordHash: hash}
	if existing, found := h.users.byEmail(invite.Email); found {
		// A re-invite of someone who forgot their password: new password,
		// same person, sessions elsewhere ended.
		user = existing
		user.Name, user.Role, user.PasswordHash, user.TOTPSecret = name, invite.Role, hash, ""
		h.users.dropUserSessions(existing.ID)
	}
	saved, err := h.users.put(user)
	if err != nil {
		h.renderJoin(w, http.StatusOK, "Something went wrong. Try again.", token, invite, name)
		return
	}
	h.users.consumeInvite(token)
	if err := h.signIn(w, r, saved); err != nil {
		http.Redirect(w, r, loginPath, http.StatusSeeOther)
		return
	}
	h.audit(r, saved, "account.joined", "Joined as "+saved.roleLabel())
	http.Redirect(w, r, "/admin?welcome=1", http.StatusSeeOther)
}

// ---------- your account ----------

func (h *Host) handleAccount(w http.ResponseWriter, r *http.Request) {
	user, ok := h.currentUser(r)
	if !ok {
		h.renderAdminInfo(w, "Your account", "No accounts yet", "This site has no accounts, so anyone who can reach it can edit it. Start the site with -admin-password on a real server, and the first visit creates the owner account.")
		return
	}
	h.renderAccount(w, user, adminStatus{Message: r.URL.Query().Get("status")}, "")
}

func (h *Host) renderAdminInfo(w http.ResponseWriter, title, heading, text string) {
	body := h.renderAdminShell("", title, "", adminStatus{}, gosx.El("section", gosx.Attrs(gosx.Attr("class", "admin-empty")),
		gosx.El("h2", nil, gosx.Text(heading)), gosx.El("p", nil, gosx.Text(text))))
	h.writeDocument(w, http.StatusOK, h.adminMeta(title), body)
}

func (h *Host) renderAccount(w http.ResponseWriter, user User, status adminStatus, newSecret string) {
	profile := gosx.El("section", gosx.Attrs(gosx.Attr("class", "admin-panel")),
		gosx.El("h2", nil, gosx.Text("You")),
		gosx.El("form", gosx.Attrs(gosx.Attr("method", "post"), gosx.Attr("action", accountPath)),
			h.csrfField(), hidden("action", "profile"),
			adminTextField("name", "Name", user.Name, "How you appear to others on this site."),
			gosx.El("p", gosx.Attrs(gosx.Attr("class", "admin-hint")), gosx.Text(user.Email+" · "+user.roleLabel())),
			gosx.El("div", gosx.Attrs(gosx.Attr("class", "admin-actions")),
				gosx.El("button", gosx.Attrs(gosx.Attr("class", "admin-button"), gosx.Attr("type", "submit")), gosx.Text("Save"))),
		),
	)
	password := gosx.El("section", gosx.Attrs(gosx.Attr("class", "admin-panel")),
		gosx.El("h2", nil, gosx.Text("Password")),
		gosx.El("form", gosx.Attrs(gosx.Attr("method", "post"), gosx.Attr("action", accountPath)),
			h.csrfField(), hidden("action", "password"),
			passwordAdminField("current", "Current password", "current-password"),
			passwordAdminField("password", "New password", "new-password"),
			gosx.El("div", gosx.Attrs(gosx.Attr("class", "admin-actions")),
				gosx.El("button", gosx.Attrs(gosx.Attr("class", "admin-button"), gosx.Attr("type", "submit")), gosx.Text("Change password"))),
		),
	)
	var twoFactor gosx.Node
	switch {
	case newSecret != "":
		twoFactor = gosx.El("form", gosx.Attrs(gosx.Attr("method", "post"), gosx.Attr("action", accountPath)),
			h.csrfField(), hidden("action", "2fa-confirm"), hidden("secret", newSecret),
			gosx.El("p", nil, gosx.Text("In your authenticator app (Google Authenticator, Authy, 1Password, and others), add an account by entering this key:")),
			gosx.El("pre", gosx.Attrs(gosx.Attr("class", "admin-code")), gosx.Text(spacedSecret(newSecret))),
			gosx.El("p", gosx.Attrs(gosx.Attr("class", "admin-hint")), gosx.Text("Or open this link on a phone that has the app: "),
				gosx.El("a", gosx.Attrs(gosx.Attr("href", otpauthURL(h, user, newSecret))), gosx.Text("add to authenticator"))),
			adminTextField("code", "Then type the code it shows", "", "Six digits. This proves the app is set up before we turn it on."),
			gosx.El("div", gosx.Attrs(gosx.Attr("class", "admin-actions")),
				gosx.El("button", gosx.Attrs(gosx.Attr("class", "admin-button"), gosx.Attr("type", "submit")), gosx.Text("Turn on"))),
		)
	case user.TOTPSecret != "":
		twoFactor = gosx.El("form", gosx.Attrs(gosx.Attr("method", "post"), gosx.Attr("action", accountPath)),
			h.csrfField(), hidden("action", "2fa-off"),
			gosx.El("p", nil, gosx.El("span", gosx.Attrs(gosx.Attr("class", "admin-badge"), gosx.Attr("data-state", "published")), gosx.Text("On")),
				gosx.Text(" Signing in asks for a code from your app.")),
			adminTextField("code", "Code from your app", "", "Needed to turn it off."),
			gosx.El("div", gosx.Attrs(gosx.Attr("class", "admin-actions")),
				gosx.El("button", gosx.Attrs(gosx.Attr("class", "admin-secondary"), gosx.Attr("type", "submit")), gosx.Text("Turn off"))),
		)
	default:
		twoFactor = gosx.El("form", gosx.Attrs(gosx.Attr("method", "post"), gosx.Attr("action", accountPath)),
			h.csrfField(), hidden("action", "2fa-start"),
			gosx.El("p", nil, gosx.Text("A second step at sign-in: a code from an app on your phone. Strongly recommended for anyone who can change settings or take payments.")),
			gosx.El("div", gosx.Attrs(gosx.Attr("class", "admin-actions")),
				gosx.El("button", gosx.Attrs(gosx.Attr("class", "admin-button"), gosx.Attr("type", "submit")), gosx.Text("Set up two-step sign-in"))),
		)
	}
	security := gosx.El("section", gosx.Attrs(gosx.Attr("class", "admin-panel")),
		gosx.El("h2", nil, gosx.Text("Two-step sign-in")), twoFactor,
		gosx.El("h3", gosx.Attrs(gosx.Attr("class", "admin-subhead")), gosx.Text("Sessions")),
		gosx.El("form", gosx.Attrs(gosx.Attr("method", "post"), gosx.Attr("action", accountPath)),
			h.csrfField(), hidden("action", "signout-all"),
			gosx.El("p", gosx.Attrs(gosx.Attr("class", "admin-hint")), gosx.Text("Lost a laptop or shared a computer? Sign out everywhere, including here.")),
			gosx.El("button", gosx.Attrs(gosx.Attr("class", "admin-secondary"), gosx.Attr("type", "submit")), gosx.Text("Sign out everywhere"))),
	)
	body := h.renderAdminShell("account", "Your account", "", status, profile, password, security)
	h.writeDocument(w, http.StatusOK, h.adminMeta("Your account"), body)
}

func passwordAdminField(name, label, autocomplete string) gosx.Node {
	return gosx.El("div", gosx.Attrs(gosx.Attr("class", "admin-field")),
		gosx.El("label", gosx.Attrs(gosx.Attr("for", name)), gosx.Text(label)),
		gosx.El("input", gosx.Attrs(gosx.Attr("type", "password"), gosx.Attr("id", name), gosx.Attr("name", name), gosx.Attr("autocomplete", autocomplete), gosx.Attr("required", "required"))),
	)
}

func otpauthURL(h *Host, user User, secret string) string {
	issuer := firstNonEmpty(h.settings().Title, h.opts.SiteTitle)
	return "otpauth://totp/" + url.PathEscape(issuer+":"+user.Email) + "?secret=" + secret + "&issuer=" + url.QueryEscape(issuer) + "&digits=6&period=30"
}

func (h *Host) handleAccountAction(w http.ResponseWriter, r *http.Request) {
	user, ok := h.currentUser(r)
	if !ok {
		http.Redirect(w, r, accountPath, http.StatusSeeOther)
		return
	}
	_ = r.ParseForm()
	fail := func(message string) { h.renderAccount(w, user, adminStatus{Message: message, Error: true}, "") }
	switch r.PostFormValue("action") {
	case "profile":
		user.Name = strings.TrimSpace(r.PostFormValue("name"))
		if user.Name == "" {
			fail("Add your name.")
			return
		}
		if _, err := h.users.put(user); err != nil {
			fail("We couldn't save that.")
			return
		}
		http.Redirect(w, r, accountPath+"?status="+queryEscape("Saved."), http.StatusSeeOther)
	case "password":
		if !checkPassword(user.PasswordHash, r.PostFormValue("current")) {
			fail("That isn't your current password.")
			return
		}
		if problem := passwordProblem(r.PostFormValue("password")); problem != "" {
			fail(problem)
			return
		}
		hash, err := hashPassword(r.PostFormValue("password"))
		if err != nil {
			fail("We couldn't save that.")
			return
		}
		user.PasswordHash = hash
		if _, err := h.users.put(user); err != nil {
			fail("We couldn't save that.")
			return
		}
		h.audit(r, user, "account.password", "Changed their password")
		http.Redirect(w, r, accountPath+"?status="+queryEscape("Password changed."), http.StatusSeeOther)
	case "2fa-start":
		h.renderAccount(w, user, adminStatus{}, newTOTPSecret())
	case "2fa-confirm":
		secret := strings.TrimSpace(r.PostFormValue("secret"))
		if !verifyTOTP(secret, r.PostFormValue("code"), timeNow()) {
			h.renderAccount(w, user, adminStatus{Message: "That code didn't match. Check the key was entered exactly, then try the current code.", Error: true}, secret)
			return
		}
		user.TOTPSecret = secret
		if _, err := h.users.put(user); err != nil {
			fail("We couldn't save that.")
			return
		}
		h.audit(r, user, "account.2fa", "Turned on two-step sign-in")
		http.Redirect(w, r, accountPath+"?status="+queryEscape("Two-step sign-in is on."), http.StatusSeeOther)
	case "2fa-off":
		if !verifyTOTP(user.TOTPSecret, r.PostFormValue("code"), timeNow()) {
			fail("That code didn't match.")
			return
		}
		user.TOTPSecret = ""
		if _, err := h.users.put(user); err != nil {
			fail("We couldn't save that.")
			return
		}
		h.audit(r, user, "account.2fa", "Turned off two-step sign-in")
		http.Redirect(w, r, accountPath+"?status="+queryEscape("Two-step sign-in is off."), http.StatusSeeOther)
	case "signout-all":
		h.users.dropUserSessions(user.ID)
		setSessionCookie(w, r, "", -1)
		http.Redirect(w, r, loginPath, http.StatusSeeOther)
	default:
		http.Redirect(w, r, accountPath, http.StatusSeeOther)
	}
}

// ---------- people ----------

func (h *Host) handlePeople(w http.ResponseWriter, r *http.Request) {
	h.renderPeople(w, r, adminStatus{Message: r.URL.Query().Get("status")}, "")
}

func (h *Host) renderPeople(w http.ResponseWriter, r *http.Request, status adminStatus, inviteLink string) {
	me, _ := h.currentUser(r)
	users := h.users.list()
	rows := make([]gosx.Node, 0, len(users))
	for _, user := range users {
		last := "Never"
		if !user.LastLogin.IsZero() {
			last = formatWhen(user.LastLogin)
		}
		twoStep := gosx.El("span", gosx.Attrs(gosx.Attr("class", "admin-badge")), gosx.Text("Off"))
		if user.TOTPSecret != "" {
			twoStep = gosx.El("span", gosx.Attrs(gosx.Attr("class", "admin-badge"), gosx.Attr("data-state", "published")), gosx.Text("On"))
		}
		actions := []gosx.Node{}
		if user.ID != me.ID {
			if roleRank(me.Role) >= roleRank(roleOwner) || user.Role != roleOwner {
				for _, role := range []string{roleEditor, roleAdmin, roleOwner} {
					if role == user.Role || (role == roleOwner && me.Role != roleOwner) {
						continue
					}
					actions = append(actions, h.peopleButton(user.ID, "role:"+role, "Make "+strings.ToLower(User{Role: role}.roleLabel())))
				}
				actions = append(actions, h.peopleButton(user.ID, "remove", "Remove"))
			}
		}
		name := gosx.Text(user.Name)
		if user.ID == me.ID {
			name = gosx.Fragment(gosx.Text(user.Name+" "), gosx.El("span", gosx.Attrs(gosx.Attr("class", "admin-badge")), gosx.Text("you")))
		}
		rows = append(rows, gosx.El("tr", nil,
			gosx.El("td", nil, name, gosx.El("br", nil), gosx.El("small", gosx.Attrs(gosx.Attr("class", "admin-hint")), gosx.Text(user.Email))),
			gosx.El("td", nil, gosx.Text(user.roleLabel())),
			gosx.El("td", nil, twoStep),
			gosx.El("td", nil, gosx.Text(last)),
			gosx.El("td", gosx.Attrs(gosx.Attr("class", "admin-row-actions")), gosx.Fragment(actions...)),
		))
	}
	list := gosx.El("section", gosx.Attrs(gosx.Attr("class", "admin-panel")),
		gosx.El("h2", nil, gosx.Text("Who can sign in")),
		gosx.El("table", gosx.Attrs(gosx.Attr("class", "admin-table")),
			gosx.El("thead", nil, gosx.El("tr", nil, gosx.El("th", nil, gosx.Text("Person")), gosx.El("th", nil, gosx.Text("Role")), gosx.El("th", nil, gosx.Text("Two-step")), gosx.El("th", nil, gosx.Text("Last sign-in")), gosx.El("th", nil, gosx.Text("")))),
			gosx.El("tbody", nil, gosx.Fragment(rows...))),
		gosx.El("p", gosx.Attrs(gosx.Attr("class", "admin-hint")), gosx.Text("Owners can do everything. Admins can do everything except hand the site to someone else. Editors write, publish, and read messages, but don't touch settings, payments, the domain, orders, or people.")),
	)
	var linkPanel gosx.Node = gosx.Fragment()
	if inviteLink != "" {
		linkPanel = gosx.El("section", gosx.Attrs(gosx.Attr("class", "admin-panel")),
			gosx.El("h2", nil, gosx.Text("Send them this link")),
			gosx.El("p", nil, gosx.Text("It works once and expires in seven days. Copy it into a message to them:")),
			gosx.El("pre", gosx.Attrs(gosx.Attr("class", "admin-code")), gosx.Text(inviteLink)),
		)
	}
	roleOptions := make([]gosx.Node, 0, 3)
	for _, role := range []string{roleEditor, roleAdmin} {
		attrs := []any{gosx.Attr("type", "radio"), gosx.Attr("name", "role"), gosx.Attr("value", role)}
		if role == roleEditor {
			attrs = append(attrs, gosx.Attr("checked", "checked"))
		}
		roleOptions = append(roleOptions, gosx.El("label", gosx.Attrs(gosx.Attr("class", "admin-radio")),
			gosx.El("input", gosx.Attrs(attrs...)),
			gosx.Text(" "+User{Role: role}.roleLabel())))
	}
	invite := gosx.El("section", gosx.Attrs(gosx.Attr("class", "admin-panel")),
		gosx.El("h2", nil, gosx.Text("Invite someone")),
		gosx.El("form", gosx.Attrs(gosx.Attr("method", "post"), gosx.Attr("action", peoplePath)),
			h.csrfField(), hidden("action", "invite"),
			adminTextField("email", "Their email", "", "They'll set their own password. Inviting an existing address gives that person a fresh way in if they forgot theirs."),
			gosx.El("div", gosx.Attrs(gosx.Attr("class", "admin-field admin-radios")), gosx.El("span", gosx.Attrs(gosx.Attr("class", "admin-radios__label")), gosx.Text("As")), gosx.Fragment(roleOptions...)),
			gosx.El("div", gosx.Attrs(gosx.Attr("class", "admin-actions")),
				gosx.El("button", gosx.Attrs(gosx.Attr("class", "admin-button"), gosx.Attr("type", "submit")), gosx.Text("Create invite link"))),
		),
	)
	body := h.renderAdminShell("users", "People", "Everyone who can sign in to this site, and what they can do.", status, linkPanel, list, invite)
	h.writeDocument(w, http.StatusOK, h.adminMeta("People"), body)
}

func (h *Host) peopleButton(id, action, label string) gosx.Node {
	return gosx.El("form", gosx.Attrs(gosx.Attr("method", "post"), gosx.Attr("action", peoplePath), gosx.Attr("class", "admin-inline-form")),
		h.csrfField(), hidden("action", action), hidden("user", id),
		gosx.El("button", gosx.Attrs(gosx.Attr("class", "admin-row-btn"), gosx.Attr("type", "submit")), gosx.Text(label)),
	)
}

func (h *Host) handlePeopleAction(w http.ResponseWriter, r *http.Request) {
	me, _ := h.currentUser(r)
	_ = r.ParseForm()
	action := r.PostFormValue("action")
	fail := func(message string) { h.renderPeople(w, r, adminStatus{Message: message, Error: true}, "") }
	switch {
	case action == "invite":
		email := normalizeEmail(r.PostFormValue("email"))
		if !validEmail(email) {
			fail("That email address doesn't look right.")
			return
		}
		role := normalizeRole(r.PostFormValue("role"))
		if role == roleOwner {
			role = roleAdmin
		}
		token, err := h.users.invite(email, role, me.ID)
		if err != nil {
			fail("We couldn't create the invite.")
			return
		}
		link := h.absoluteBase(r) + joinPrefix + token
		h.notify(Mail{To: email, Subject: "You're invited to edit " + firstNonEmpty(h.settings().Title, h.opts.SiteTitle),
			Text: me.Name + " invited you to help run " + firstNonEmpty(h.settings().Title, h.opts.SiteTitle) + ".\n\nOpen this link to choose a password and get in:\n" + link + "\n\nIt works once and expires in seven days.\n"})
		h.audit(r, me, "people.invited", "Invited "+email+" as "+User{Role: role}.roleLabel())
		h.renderPeople(w, r, adminStatus{Message: "Invite created for " + email + ". If email is set up, it's on its way; either way, the link is below."}, link)
	case strings.HasPrefix(action, "role:"):
		target, ok := h.users.byID(r.PostFormValue("user"))
		if !ok || target.ID == me.ID {
			fail("We couldn't find that person.")
			return
		}
		role := normalizeRole(strings.TrimPrefix(action, "role:"))
		if (role == roleOwner || target.Role == roleOwner) && me.Role != roleOwner {
			fail("Only an owner can change who owns the site.")
			return
		}
		target.Role = role
		if _, err := h.users.put(target); err != nil {
			fail("The site needs at least one owner.")
			return
		}
		h.audit(r, me, "people.role", "Made "+target.Email+" "+strings.ToLower(target.roleLabel()))
		http.Redirect(w, r, peoplePath+"?status="+queryEscape(target.Name+" is now "+strings.ToLower(target.roleLabel())+"."), http.StatusSeeOther)
	case action == "remove":
		target, ok := h.users.byID(r.PostFormValue("user"))
		if !ok || target.ID == me.ID {
			fail("We couldn't find that person.")
			return
		}
		if target.Role == roleOwner && me.Role != roleOwner {
			fail("Only an owner can remove an owner.")
			return
		}
		if err := h.users.remove(target.ID); err != nil {
			fail("The site needs at least one owner.")
			return
		}
		h.audit(r, me, "people.removed", "Removed "+target.Email)
		http.Redirect(w, r, peoplePath+"?status="+queryEscape(target.Name+" can no longer sign in."), http.StatusSeeOther)
	default:
		http.Redirect(w, r, peoplePath, http.StatusSeeOther)
	}
}

// audit records who did what; audit.go keeps the log. Until it exists this
// is a no-op hook so every action site already names its event.
func (h *Host) audit(r *http.Request, user User, event, summary string) {
	if h.auditLog != nil {
		h.auditLog.record(user, event, summary, remoteHost(r))
	}
}

func isAdminPath(path string) bool {
	return path == adminPathPrefix || strings.HasPrefix(path, adminPathPrefix+"/")
}

func secureEqual(got, want string) bool {
	return subtle.ConstantTimeCompare([]byte(got), []byte(want)) == 1
}
