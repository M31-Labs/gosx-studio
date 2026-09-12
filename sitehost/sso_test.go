package sitehost

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	cmsstore "m31labs.dev/gosx-studio/cms/store"
)

// fakeProvider is an OpenID Connect provider that issues real RS256 tokens.
type fakeProvider struct {
	server    *httptest.Server
	key       *rsa.PrivateKey
	nonce     string
	challenge string
	email     string
	name      string
	badSig    bool
	tokenHits int
}

func newFakeProvider(t *testing.T) *fakeProvider {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	p := &fakeProvider{key: key, email: "pat@example.org", name: "Pat Example"}
	mux := http.NewServeMux()
	mux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]string{
			"issuer": p.server.URL, "authorization_endpoint": p.server.URL + "/authorize",
			"token_endpoint": p.server.URL + "/token", "jwks_uri": p.server.URL + "/jwks",
		})
	})
	mux.HandleFunc("/jwks", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"keys": []map[string]string{{
			"kty": "RSA", "kid": "k1", "use": "sig",
			"n": base64.RawURLEncoding.EncodeToString(key.N.Bytes()),
			"e": base64.RawURLEncoding.EncodeToString(big.NewInt(int64(key.E)).Bytes()),
		}}})
	})
	mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		p.tokenHits++
		_ = r.ParseForm()
		if r.PostFormValue("code") != "good" || r.PostFormValue("grant_type") != "authorization_code" {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid_grant"})
			return
		}
		if pkceChallenge(r.PostFormValue("code_verifier")) != p.challenge {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "bad_verifier"})
			return
		}
		header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"RS256","kid":"k1"}`))
		claims, _ := json.Marshal(map[string]any{"iss": p.server.URL, "sub": "123", "aud": "app", "exp": time.Now().Add(time.Hour).Unix(), "nonce": p.nonce, "email": p.email, "email_verified": true, "name": p.name})
		payload := base64.RawURLEncoding.EncodeToString(claims)
		digest := sha256.Sum256([]byte(header + "." + payload))
		signature, _ := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, digest[:])
		if p.badSig {
			signature[0] ^= 0xff
		}
		_ = json.NewEncoder(w).Encode(map[string]string{"id_token": header + "." + payload + "." + base64.RawURLEncoding.EncodeToString(signature)})
	})
	p.server = httptest.NewServer(mux)
	t.Cleanup(p.server.Close)
	return p
}

// startSSO begins the flow and hands back what the provider will need.
func startSSO(t *testing.T, handler http.Handler, next string) (state string, cookie *http.Cookie, authURL *url.URL) {
	t.Helper()
	rec := get(t, handler, ssoStartPath+"?next="+url.QueryEscape(next))
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("sso start = %d %s", rec.Code, rec.Body.String())
	}
	authURL, err := url.Parse(rec.Header().Get("Location"))
	if err != nil {
		t.Fatal(err)
	}
	return authURL.Query().Get("state"), cookieNamed(rec, ssoCookie), authURL
}

func TestSignInWithOpenIDConnect(t *testing.T) {
	allowInsecureSSO = true
	t.Cleanup(func() { allowInsecureSSO = false })
	provider := newFakeProvider(t)
	host, handler := newGuardedHost(t)
	post(t, handler, loginPath, url.Values{"name": {"Ana"}, "email": {"ana@example.com"}, "password": {"correct horse battery"}, "serverPassword": {"server-secret"}})
	if err := host.updateSettingsMetadata(func(m cmsstore.Metadata) {
		m[ssoIssuerKey] = provider.server.URL
		m[ssoClientIDKey] = "app"
		m[ssoSecretKey] = "shh"
		m[ssoDomainKey] = "example.org"
		m[ssoLabelKey] = "Sign in with Fake"
	}); err != nil {
		t.Fatal(err)
	}
	login := get(t, handler, loginPath).Body.String()
	mustContain(t, login, `href="/admin/sso/start"`, "the login page offers single sign-on")
	mustContain(t, login, `>Sign in with Fake</a>`, "with the owner's label")

	// Start: state, nonce, PKCE, and a redirect to the provider.
	state, cookie, authURL := startSSO(t, handler, "/admin/pages")
	q := authURL.Query()
	if !strings.HasPrefix(authURL.String(), provider.server.URL+"/authorize?") || q.Get("client_id") != "app" || q.Get("response_type") != "code" ||
		q.Get("code_challenge_method") != "S256" || q.Get("redirect_uri") != "https://wildflower.example/admin/sso/callback" || !strings.Contains(q.Get("scope"), "openid") {
		t.Fatalf("authorize url = %s", authURL)
	}
	provider.nonce, provider.challenge = q.Get("nonce"), q.Get("code_challenge")

	// A wrong state is refused before anything is exchanged.
	rec := getWithCookie(t, handler, ssoCallback+"?code=good&state=nope", cookie)
	mustContain(t, rec.Body.String(), "didn&#39;t start here", "a foreign state is refused")
	if provider.tokenHits != 0 {
		t.Fatal("no token exchange without a matching state")
	}

	// A stranger from the allowed domain becomes an editor.
	rec = getWithCookie(t, handler, ssoCallback+"?code=good&state="+state, cookie)
	if rec.Code != http.StatusSeeOther || rec.Header().Get("Location") != "/admin/pages" {
		t.Fatalf("callback = %d %q %s", rec.Code, rec.Header().Get("Location"), rec.Body.String())
	}
	session := cookieNamed(rec, sessionCookie)
	if session == nil {
		t.Fatal("no session after sso")
	}
	pat, ok := host.users.byEmail("pat@example.org")
	if !ok || pat.Role != roleEditor || pat.Name != "Pat Example" {
		t.Fatalf("pat = %+v %v", pat, ok)
	}
	if code := getWithCookie(t, handler, "/admin/settings", session).Code; code != http.StatusForbidden {
		t.Fatal("an sso editor is still an editor")
	}

	// The owner signs in through the provider and keeps their role.
	provider.email, provider.name = "ana@example.com", "Ana L"
	state, cookie, authURL = startSSO(t, handler, "")
	provider.nonce, provider.challenge = authURL.Query().Get("nonce"), authURL.Query().Get("code_challenge")
	rec = getWithCookie(t, handler, ssoCallback+"?code=good&state="+state, cookie)
	if code := getWithCookie(t, handler, "/admin/settings", cookieNamed(rec, sessionCookie)).Code; code != http.StatusOK {
		t.Fatalf("owner through sso = %d", code)
	}
	ana, _ := host.users.byEmail("ana@example.com")
	if ana.Role != roleOwner || ana.Name != "Ana" {
		t.Fatal("sso must not change an existing account")
	}

	// Outside the domain: no account, no entry.
	provider.email = "eve@other.com"
	state, cookie, authURL = startSSO(t, handler, "")
	provider.nonce, provider.challenge = authURL.Query().Get("nonce"), authURL.Query().Get("code_challenge")
	rec = getWithCookie(t, handler, ssoCallback+"?code=good&state="+state, cookie)
	mustContain(t, rec.Body.String(), "no account for eve@other.com", "unknown people are turned away")
	if _, ok := host.users.byEmail("eve@other.com"); ok {
		t.Fatal("no account must be created")
	}

	// A forged token is refused.
	provider.email, provider.badSig = "pat@example.org", true
	state, cookie, authURL = startSSO(t, handler, "")
	provider.nonce, provider.challenge = authURL.Query().Get("nonce"), authURL.Query().Get("code_challenge")
	rec = getWithCookie(t, handler, ssoCallback+"?code=good&state="+state, cookie)
	mustContain(t, rec.Body.String(), "signature does not check out", "a bad signature is refused")
	if cookieNamed(rec, sessionCookie) != nil {
		t.Fatal("no session on a forged token")
	}

	// A used or expired start cookie cannot be replayed.
	provider.badSig = false
	rec = getWithCookie(t, handler, ssoCallback+"?code=good&state="+state, &http.Cookie{Name: ssoCookie, Value: "garbage"})
	mustContain(t, rec.Body.String(), "didn&#39;t start here", "a tampered start cookie is refused")

	mustContain(t, getWithCookie(t, handler, "/admin/activity", cookieNamed(post(t, handler, loginPath, url.Values{"email": {"ana@example.com"}, "password": {"correct horse battery"}}), sessionCookie)).Body.String(), "Joined through single sign-on", "sso joins are logged")
}
