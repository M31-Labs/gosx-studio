package sitehost

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"
)

func init() { passwordCost = bcrypt.MinCost }

func cookieNamed(rec *httptest.ResponseRecorder, name string) *http.Cookie {
	for _, cookie := range rec.Result().Cookies() {
		if cookie.Name == name {
			return cookie
		}
	}
	return nil
}

// csrfWith reads the token from an admin page fetched with a session.
func csrfWith(t *testing.T, handler http.Handler, session *http.Cookie) string {
	t.Helper()
	body := getWithCookie(t, handler, "/admin", session).Body.String()
	marker := `<meta name="csrf-token" content="`
	start := strings.Index(body, marker)
	if start < 0 {
		t.Fatalf("no csrf token on /admin: %s", body[:200])
	}
	rest := body[start+len(marker):]
	return rest[:strings.Index(rest, `"`)]
}

func postAs(t *testing.T, handler http.Handler, session *http.Cookie, path string, form url.Values) *httptest.ResponseRecorder {
	t.Helper()
	form.Set("_csrf", csrfWith(t, handler, session))
	return postWithCookie(t, handler, path, form, session)
}

func newGuardedHost(t *testing.T) (*Host, http.Handler) {
	t.Helper()
	host, err := Open(Options{DataPath: filepath.Join(t.TempDir(), "site.json"), SiteTitle: "Wildflower", SiteKind: "food", Seed: true, NoBackups: true, AdminPassword: "server-secret", BaseURL: "https://wildflower.example"})
	if err != nil {
		t.Fatal(err)
	}
	return host, host.Handler()
}

func TestOwnerAccountIsBootstrappedFromTheServerPassword(t *testing.T) {
	host, handler := newGuardedHost(t)
	rec := get(t, handler, "/admin")
	if rec.Code != http.StatusSeeOther || rec.Header().Get("Location") != loginPath {
		t.Fatalf("admin before any account = %d %q", rec.Code, rec.Header().Get("Location"))
	}
	page := get(t, handler, loginPath).Body.String()
	mustContain(t, page, "Create your account", "the first visit creates the owner")
	mustContain(t, page, `name="serverPassword"`, "and asks for the server password")
	if strings.Contains(page, "csrf-token") {
		t.Fatal("a public page must not carry the CSRF token")
	}

	rec = post(t, handler, loginPath, url.Values{"name": {"Ana"}, "email": {"ana@example.com"}, "password": {"correct horse battery"}, "serverPassword": {"wrong"}})
	mustContain(t, rec.Body.String(), "isn&#39;t the password this site was started with", "the wrong server password is refused")
	rec = post(t, handler, loginPath, url.Values{"name": {"Ana"}, "email": {"ana@example.com"}, "password": {"short"}, "serverPassword": {"server-secret"}})
	mustContain(t, rec.Body.String(), "at least 10 characters", "weak passwords are refused")
	rec = post(t, handler, loginPath, url.Values{"name": {"Ana"}, "email": {"Ana@Example.com"}, "password": {"correct horse battery"}, "serverPassword": {"server-secret"}})
	if rec.Code != http.StatusSeeOther || !strings.HasPrefix(rec.Header().Get("Location"), "/admin") {
		t.Fatalf("create owner = %d %q %s", rec.Code, rec.Header().Get("Location"), rec.Body.String())
	}
	session := cookieNamed(rec, sessionCookie)
	if session == nil || !session.HttpOnly || session.Path != "/admin" || session.SameSite != http.SameSiteLaxMode {
		t.Fatalf("session cookie = %+v", session)
	}
	owner, _ := host.users.byEmail("ana@example.com")
	if owner.Role != roleOwner || owner.Name != "Ana" {
		t.Fatalf("owner = %+v", owner)
	}

	// Signed in: the admin works, and the basics of the old way are gone.
	if code := getWithCookie(t, handler, "/admin", session).Code; code != http.StatusOK {
		t.Fatalf("admin with session = %d", code)
	}
	mustContain(t, getWithCookie(t, handler, "/admin", session).Body.String(), "Sign out", "the nav offers sign out")
	if rec := get(t, handler, "/admin/pages"); rec.Code != http.StatusSeeOther || !strings.Contains(rec.Header().Get("Location"), "next=%2Fadmin%2Fpages") {
		t.Fatalf("admin without session = %d %q", rec.Code, rec.Header().Get("Location"))
	}
	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	req.SetBasicAuth("admin", "server-secret")
	basic := httptest.NewRecorder()
	handler.ServeHTTP(basic, req)
	if basic.Code != http.StatusSeeOther {
		t.Fatalf("basic auth must no longer open the admin: %d", basic.Code)
	}
	// A save through the editor without a session is refused as JSON.
	rec = postJSON(t, handler, "/admin/api/pages/x", `{}`)
	if rec.Code != http.StatusUnauthorized || !strings.Contains(rec.Body.String(), "signed out") {
		t.Fatalf("api without session = %d %s", rec.Code, rec.Body.String())
	}

	// Sign out, sign back in; wrong passwords lock out after ten tries.
	rec = postAs(t, handler, session, logoutPath, url.Values{})
	if cookieNamed(rec, sessionCookie).MaxAge != -1 {
		t.Fatal("sign out must clear the cookie")
	}
	if code := getWithCookie(t, handler, "/admin", session).Code; code != http.StatusSeeOther {
		t.Fatal("an old session must not work after sign out")
	}
	mustContain(t, get(t, handler, loginPath).Body.String(), "Sign in to Wildflower", "the login page now signs in")
	rec = post(t, handler, loginPath, url.Values{"email": {"ana@example.com"}, "password": {"nope nope nope"}})
	mustContain(t, rec.Body.String(), "don&#39;t match", "a wrong password is refused")
	for i := 0; i < 12; i++ {
		post(t, handler, loginPath, url.Values{"email": {"ana@example.com"}, "password": {"nope nope nope"}})
	}
	if rec := post(t, handler, loginPath, url.Values{"email": {"ana@example.com"}, "password": {"correct horse battery"}}); rec.Code != http.StatusTooManyRequests {
		t.Fatalf("lockout = %d", rec.Code)
	}
	host.authFailures.clear("192.0.2.1")
	rec = post(t, handler, loginPath, url.Values{"email": {"ana@example.com"}, "password": {"correct horse battery"}, "next": {"/admin/pages"}})
	if rec.Code != http.StatusSeeOther || rec.Header().Get("Location") != "/admin/pages" {
		t.Fatalf("sign in = %d %q", rec.Code, rec.Header().Get("Location"))
	}
	mustContain(t, getWithCookie(t, handler, "/admin/activity", cookieNamed(rec, sessionCookie)).Body.String(), "Signed in", "sign-ins are in the activity log")
}

func TestInvitesRolesAndTwoStep(t *testing.T) {
	host, handler := newGuardedHost(t)
	rec := post(t, handler, loginPath, url.Values{"name": {"Ana"}, "email": {"ana@example.com"}, "password": {"correct horse battery"}, "serverPassword": {"server-secret"}})
	owner := cookieNamed(rec, sessionCookie)

	// Invite an editor.
	rec = postAs(t, handler, owner, peoplePath, url.Values{"action": {"invite"}, "email": {"sam@example.com"}, "role": {"editor"}})
	body := rec.Body.String()
	mustContain(t, body, "Send them this link", "the invite link is shown")
	start := strings.Index(body, "https://wildflower.example/admin/join/")
	link := body[start:]
	link = link[:strings.Index(link, "<")]
	token := strings.TrimPrefix(link, "https://wildflower.example/admin/join/")
	mustContain(t, get(t, handler, joinPrefix+token).Body.String(), "invited as editor for sam@example.com", "the join page names the role")
	rec = post(t, handler, joinPrefix+token, url.Values{"name": {"Sam"}, "password": {"another long password"}})
	editor := cookieNamed(rec, sessionCookie)
	if editor == nil {
		t.Fatalf("join = %d %s", rec.Code, rec.Body.String())
	}
	if code := get(t, handler, joinPrefix+token).Code; code != http.StatusNotFound {
		t.Fatal("an invite works once")
	}

	// An editor edits but does not administer.
	if code := getWithCookie(t, handler, "/admin/pages", editor).Code; code != http.StatusOK {
		t.Fatalf("editor pages = %d", code)
	}
	if code := getWithCookie(t, handler, "/admin/settings", editor).Code; code != http.StatusForbidden {
		t.Fatalf("editor settings = %d, want 403", code)
	}
	mustContain(t, getWithCookie(t, handler, "/admin/settings", editor).Body.String(), "This part is for admins", "with an explanation")
	if code := postWithCookie(t, handler, "/admin/api/theme", url.Values{}, editor).Code; code != http.StatusForbidden {
		t.Fatal("editors cannot change the Look")
	}
	id := firstPageID(t, host, "menu")
	if strings.Contains(getWithCookie(t, handler, "/admin/edit/"+id, editor).Body.String(), `id="look"`) {
		t.Fatal("editors do not see the Look panel")
	}
	mustContain(t, getWithCookie(t, handler, "/admin/edit/"+id, owner).Body.String(), `id="look"`, "owners do")
	if code := getWithCookie(t, handler, peoplePath, editor).Code; code != http.StatusForbidden {
		t.Fatal("editors cannot manage people")
	}

	// Roles: promote, demote, and the last owner stays.
	sam, _ := host.users.byEmail("sam@example.com")
	postAs(t, handler, owner, peoplePath, url.Values{"action": {"role:admin"}, "user": {sam.ID}})
	sam, _ = host.users.byEmail("sam@example.com")
	if sam.Role != roleAdmin {
		t.Fatalf("sam = %+v", sam)
	}
	if code := getWithCookie(t, handler, "/admin/settings", editor).Code; code != http.StatusOK {
		t.Fatal("an admin reaches settings")
	}
	ana, _ := host.users.byEmail("ana@example.com")
	rec = postAs(t, handler, editor, peoplePath, url.Values{"action": {"remove"}, "user": {ana.ID}})
	mustContain(t, rec.Body.String(), "Only an owner can remove an owner", "admins cannot remove the owner")
	people := getWithCookie(t, handler, peoplePath, owner).Body.String()
	mustContain(t, people, "sam@example.com", "people are listed")
	mustContain(t, people, ">Admin<", "with their role")

	// Two-step sign-in on the owner's account.
	rec = postAs(t, handler, owner, accountPath, url.Values{"action": {"2fa-start"}})
	body = rec.Body.String()
	start = strings.Index(body, `name="secret" value="`)
	secret := body[start+len(`name="secret" value="`):]
	secret = secret[:strings.Index(secret, `"`)]
	mustContain(t, body, "otpauth://totp/", "the app link is offered")
	rec = postAs(t, handler, owner, accountPath, url.Values{"action": {"2fa-confirm"}, "secret": {secret}, "code": {"000000"}})
	mustContain(t, rec.Body.String(), "didn&#39;t match", "a wrong code does not turn it on")
	rec = postAs(t, handler, owner, accountPath, url.Values{"action": {"2fa-confirm"}, "secret": {secret}, "code": {totpCode(secret, timeNow())}})
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("2fa confirm = %d %s", rec.Code, rec.Body.String())
	}
	ana, _ = host.users.byEmail("ana@example.com")
	if ana.TOTPSecret != secret {
		t.Fatal("secret not stored")
	}
	rec = post(t, handler, loginPath, url.Values{"email": {"ana@example.com"}, "password": {"correct horse battery"}})
	if !strings.HasPrefix(rec.Header().Get("Location"), loginCodePath) || cookieNamed(rec, sessionCookie) != nil {
		t.Fatalf("password alone must not sign in with 2fa on: %q", rec.Header().Get("Location"))
	}
	pending := cookieNamed(rec, pendingCookie)
	rec = postWithCookie(t, handler, loginCodePath, url.Values{"code": {"123456"}}, pending)
	mustContain(t, rec.Body.String(), "didn&#39;t match", "a wrong code is refused")
	rec = postWithCookie(t, handler, loginCodePath, url.Values{"code": {totpCode(secret, timeNow().Add(-30*time.Second))}}, pending)
	if cookieNamed(rec, sessionCookie) == nil {
		t.Fatalf("a code from the previous step should sign in: %d %s", rec.Code, rec.Body.String())
	}
	if !verifyTOTP(secret, totpCode(secret, timeNow()), timeNow()) || verifyTOTP(secret, "12345", timeNow()) {
		t.Fatal("totp basics")
	}
}

func TestALaptopSiteStaysOpenUntilSomeoneCreatesAnAccount(t *testing.T) {
	_, handler := newTestHost(t)
	if code := get(t, handler, "/admin").Code; code != http.StatusOK {
		t.Fatal("no password, no accounts: open, as before")
	}
	mustContain(t, get(t, handler, loginPath).Body.String(), "Create your account", "an account can still be created")
	if strings.Contains(get(t, handler, loginPath).Body.String(), "serverPassword") {
		t.Fatal("no server password to ask for")
	}
	rec := post(t, handler, loginPath, url.Values{"name": {"Ana"}, "email": {"ana@example.com"}, "password": {"correct horse battery"}})
	session := cookieNamed(rec, sessionCookie)
	if session == nil {
		t.Fatalf("create owner on a laptop = %d", rec.Code)
	}
	if code := get(t, handler, "/admin").Code; code != http.StatusSeeOther {
		t.Fatal("once an account exists, sign-in is required")
	}
	if code := getWithCookie(t, handler, "/admin", session).Code; code != http.StatusOK {
		t.Fatal("and the new session works")
	}
}
