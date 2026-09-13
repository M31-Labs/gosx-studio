package sitehost

import (
	"context"
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"math/big"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"m31labs.dev/gosx"
	cmsstore "m31labs.dev/gosx-studio/cms/store"
)

// sso.go is "Sign in with Google" (or Microsoft, Okta, Keycloak, anything
// that speaks OpenID Connect).
//
// The site is the client: it reads the provider's discovery document, sends
// the person to the provider with a state, a nonce, and a PKCE challenge,
// exchanges the code it gets back for an ID token, and checks that token's
// signature against the provider's published keys before believing the
// email inside. An email that already has an account signs in as that
// account; an email at the allowed domain gets an editor account made for
// it; anything else is turned away. No library, no long-lived provider
// secrets beyond the client secret the owner pastes.

const (
	ssoIssuerKey   = "ssoIssuer"
	ssoClientIDKey = "ssoClientId"
	ssoSecretKey   = "ssoClientSecret"
	ssoDomainKey   = "ssoDomain"
	ssoLabelKey    = "ssoLabel"
	ssoStartPath   = "/admin/sso/start"
	ssoCallback    = "/admin/sso/callback"
	ssoCookie      = "gosx_sso"
	ssoTTL         = 10 * time.Minute
)

// allowInsecureSSO lets tests use a plain-http fake provider.
var allowInsecureSSO = false

// ssoHTTP fetches from the provider. Tests replace it.
var ssoHTTP = &http.Client{Timeout: 15 * time.Second}

type ssoSettings struct {
	Issuer   string
	ClientID string
	Secret   string
	Domain   string
	Label    string
}

func (h *Host) sso() ssoSettings {
	meta := h.settings().Metadata
	return ssoSettings{
		Issuer:   strings.TrimRight(strings.TrimSpace(meta[ssoIssuerKey]), "/"),
		ClientID: strings.TrimSpace(meta[ssoClientIDKey]),
		Secret:   strings.TrimSpace(meta[ssoSecretKey]),
		Domain:   strings.ToLower(strings.TrimPrefix(strings.TrimSpace(meta[ssoDomainKey]), "@")),
		Label:    firstNonEmpty(strings.TrimSpace(meta[ssoLabelKey]), "Sign in with single sign-on"),
	}
}

func (s ssoSettings) ready() bool { return s.Issuer != "" && s.ClientID != "" }

// ---------- discovery and keys, cached ----------

type ssoDiscovery struct {
	Issuer                string `json:"issuer"`
	AuthorizationEndpoint string `json:"authorization_endpoint"`
	TokenEndpoint         string `json:"token_endpoint"`
	JWKSURI               string `json:"jwks_uri"`
}

type ssoCache struct {
	mu        sync.Mutex
	issuer    string
	discovery ssoDiscovery
	fetched   time.Time
	keys      map[string]*rsa.PublicKey
	keysAt    time.Time
}

func (h *Host) ssoDiscover(ctx context.Context, issuer string) (ssoDiscovery, error) {
	h.ssoState.mu.Lock()
	defer h.ssoState.mu.Unlock()
	if h.ssoState.issuer == issuer && timeNow().Sub(h.ssoState.fetched) < time.Hour {
		return h.ssoState.discovery, nil
	}
	var doc ssoDiscovery
	if err := ssoGetJSON(ctx, issuer+"/.well-known/openid-configuration", &doc); err != nil {
		return ssoDiscovery{}, err
	}
	if doc.AuthorizationEndpoint == "" || doc.TokenEndpoint == "" || doc.JWKSURI == "" {
		return ssoDiscovery{}, errors.New("the provider's discovery document is missing endpoints")
	}
	h.ssoState.issuer, h.ssoState.discovery, h.ssoState.fetched = issuer, doc, timeNow()
	h.ssoState.keys = nil
	return doc, nil
}

func ssoGetJSON(ctx context.Context, target string, out any) error {
	if !allowInsecureSSO && !strings.HasPrefix(target, "https://") {
		return errors.New("the provider address must start with https://")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	res, err := ssoHTTP.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode >= 300 {
		return errors.New("the provider answered " + res.Status)
	}
	return json.NewDecoder(io.LimitReader(res.Body, 1<<20)).Decode(out)
}

type jwk struct {
	Kid string `json:"kid"`
	Kty string `json:"kty"`
	N   string `json:"n"`
	E   string `json:"e"`
	Use string `json:"use"`
}

func (h *Host) ssoKey(ctx context.Context, doc ssoDiscovery, kid string) (*rsa.PublicKey, error) {
	h.ssoState.mu.Lock()
	defer h.ssoState.mu.Unlock()
	if h.ssoState.keys != nil && timeNow().Sub(h.ssoState.keysAt) < time.Hour {
		if key, ok := h.ssoState.keys[kid]; ok {
			return key, nil
		}
	}
	var set struct {
		Keys []jwk `json:"keys"`
	}
	if err := ssoGetJSON(ctx, doc.JWKSURI, &set); err != nil {
		return nil, err
	}
	keys := map[string]*rsa.PublicKey{}
	for _, key := range set.Keys {
		if key.Kty != "RSA" || (key.Use != "" && key.Use != "sig") {
			continue
		}
		n, err := base64.RawURLEncoding.DecodeString(key.N)
		if err != nil {
			continue
		}
		e, err := base64.RawURLEncoding.DecodeString(key.E)
		if err != nil {
			continue
		}
		keys[key.Kid] = &rsa.PublicKey{N: new(big.Int).SetBytes(n), E: int(new(big.Int).SetBytes(e).Int64())}
	}
	h.ssoState.keys, h.ssoState.keysAt = keys, timeNow()
	key, ok := keys[kid]
	if !ok {
		return nil, errors.New("the token was signed with a key the provider does not publish")
	}
	return key, nil
}

// ---------- the ID token ----------

type idClaims struct {
	Issuer        string          `json:"iss"`
	Subject       string          `json:"sub"`
	Audience      json.RawMessage `json:"aud"`
	Expires       int64           `json:"exp"`
	Nonce         string          `json:"nonce"`
	Email         string          `json:"email"`
	EmailVerified any             `json:"email_verified"`
	Name          string          `json:"name"`
}

func (c idClaims) audiences() []string {
	var one string
	if json.Unmarshal(c.Audience, &one) == nil {
		return []string{one}
	}
	var many []string
	_ = json.Unmarshal(c.Audience, &many)
	return many
}

// verifyIDToken checks the signature and the claims that matter.
func (h *Host) verifyIDToken(ctx context.Context, doc ssoDiscovery, settings ssoSettings, token, nonce string) (idClaims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return idClaims{}, errors.New("malformed token")
	}
	headerRaw, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return idClaims{}, errors.New("malformed token header")
	}
	var header struct {
		Alg string `json:"alg"`
		Kid string `json:"kid"`
	}
	if json.Unmarshal(headerRaw, &header) != nil || header.Alg != "RS256" {
		return idClaims{}, errors.New("unsupported token signature")
	}
	key, err := h.ssoKey(ctx, doc, header.Kid)
	if err != nil {
		return idClaims{}, err
	}
	signature, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return idClaims{}, errors.New("malformed token signature")
	}
	digest := sha256.Sum256([]byte(parts[0] + "." + parts[1]))
	if err := rsa.VerifyPKCS1v15(key, crypto.SHA256, digest[:], signature); err != nil {
		return idClaims{}, errors.New("the token's signature does not check out")
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return idClaims{}, errors.New("malformed token payload")
	}
	var claims idClaims
	if err := json.Unmarshal(payload, &claims); err != nil {
		return idClaims{}, errors.New("malformed token claims")
	}
	if strings.TrimRight(claims.Issuer, "/") != settings.Issuer && strings.TrimRight(claims.Issuer, "/") != strings.TrimRight(doc.Issuer, "/") {
		return idClaims{}, errors.New("the token is from a different provider")
	}
	if !containsString(claims.audiences(), settings.ClientID) {
		return idClaims{}, errors.New("the token is for a different app")
	}
	if claims.Expires <= timeNow().Unix() {
		return idClaims{}, errors.New("the token has expired")
	}
	if nonce == "" || claims.Nonce != nonce {
		return idClaims{}, errors.New("the sign-in did not match what we started")
	}
	if claims.Email == "" {
		return idClaims{}, errors.New("the provider did not share an email address")
	}
	if verified, ok := claims.EmailVerified.(bool); ok && !verified {
		return idClaims{}, errors.New("the provider says that email is not verified")
	}
	return claims, nil
}

// ---------- the flow ----------

func (h *Host) mountSSO(mux *http.ServeMux) {
	mux.HandleFunc("GET "+ssoStartPath, h.handleSSOStart)
	mux.HandleFunc("GET "+ssoCallback, h.handleSSOCallback)
}

func pkceChallenge(verifier string) string {
	sum := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

func (h *Host) handleSSOStart(w http.ResponseWriter, r *http.Request) {
	settings := h.sso()
	if !settings.ready() {
		http.Redirect(w, r, loginPath, http.StatusSeeOther)
		return
	}
	doc, err := h.ssoDiscover(r.Context(), settings.Issuer)
	if err != nil {
		h.renderLogin(w, http.StatusOK, "Single sign-on isn't reachable right now: "+err.Error(), "", r.URL.Query().Get("next"))
		return
	}
	state, nonce, verifier := randomHex(16), randomHex(16), randomHex(32)
	next := safeNext(r.URL.Query().Get("next"))
	expiry := timeNow().Add(ssoTTL).Unix()
	value := strings.Join([]string{state, nonce, verifier, url.QueryEscape(next), itoa64(expiry)}, ":")
	http.SetCookie(w, &http.Cookie{Name: ssoCookie, Value: value + ":" + h.sign(value), Path: "/admin", HttpOnly: true, SameSite: http.SameSiteLaxMode,
		Secure: r.TLS != nil || strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https"), MaxAge: int(ssoTTL / time.Second)})

	params := url.Values{}
	params.Set("response_type", "code")
	params.Set("client_id", settings.ClientID)
	params.Set("redirect_uri", h.absoluteBase(r)+ssoCallback)
	params.Set("scope", "openid email profile")
	params.Set("state", state)
	params.Set("nonce", nonce)
	params.Set("code_challenge", pkceChallenge(verifier))
	params.Set("code_challenge_method", "S256")
	separator := "?"
	if strings.Contains(doc.AuthorizationEndpoint, "?") {
		separator = "&"
	}
	http.Redirect(w, r, doc.AuthorizationEndpoint+separator+params.Encode(), http.StatusSeeOther)
}

func itoa64(n int64) string { return big.NewInt(n).String() }

func (h *Host) ssoPending(r *http.Request) (state, nonce, verifier, next string, ok bool) {
	cookie, err := r.Cookie(ssoCookie)
	if err != nil {
		return "", "", "", "", false
	}
	parts := strings.Split(cookie.Value, ":")
	if len(parts) != 6 {
		return "", "", "", "", false
	}
	value := strings.Join(parts[:5], ":")
	if !tokensEqual(parts[5], h.sign(value)) {
		return "", "", "", "", false
	}
	expiry, ok := new(big.Int).SetString(parts[4], 10)
	if !ok || timeNow().Unix() > expiry.Int64() {
		return "", "", "", "", false
	}
	unescaped, _ := url.QueryUnescape(parts[3])
	return parts[0], parts[1], parts[2], safeNext(unescaped), true
}

func (h *Host) handleSSOCallback(w http.ResponseWriter, r *http.Request) {
	settings := h.sso()
	fail := func(message string) {
		h.renderLogin(w, http.StatusOK, "Single sign-on didn't work: "+message, "", "")
	}
	if !settings.ready() {
		http.Redirect(w, r, loginPath, http.StatusSeeOther)
		return
	}
	http.SetCookie(w, &http.Cookie{Name: ssoCookie, Value: "", Path: "/admin", MaxAge: -1})
	state, nonce, verifier, next, ok := h.ssoPending(r)
	if !ok || state == "" || !tokensEqual(r.URL.Query().Get("state"), state) {
		fail("the sign-in took too long or didn't start here. Try again.")
		return
	}
	if problem := r.URL.Query().Get("error"); problem != "" {
		fail(firstNonEmpty(r.URL.Query().Get("error_description"), problem))
		return
	}
	code := r.URL.Query().Get("code")
	if code == "" {
		fail("the provider sent no code.")
		return
	}
	doc, err := h.ssoDiscover(r.Context(), settings.Issuer)
	if err != nil {
		fail(err.Error())
		return
	}
	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("code", code)
	form.Set("redirect_uri", h.absoluteBase(r)+ssoCallback)
	form.Set("client_id", settings.ClientID)
	if settings.Secret != "" {
		form.Set("client_secret", settings.Secret)
	}
	form.Set("code_verifier", verifier)
	req, err := http.NewRequestWithContext(r.Context(), http.MethodPost, doc.TokenEndpoint, strings.NewReader(form.Encode()))
	if err != nil {
		fail(err.Error())
		return
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	res, err := ssoHTTP.Do(req)
	if err != nil {
		fail("couldn't reach the provider.")
		return
	}
	defer res.Body.Close()
	var tokens struct {
		IDToken string `json:"id_token"`
		Error   string `json:"error"`
		Desc    string `json:"error_description"`
	}
	if err := json.NewDecoder(io.LimitReader(res.Body, 1<<20)).Decode(&tokens); err != nil || tokens.IDToken == "" {
		fail(firstNonEmpty(tokens.Desc, tokens.Error, "the provider gave no identity token."))
		return
	}
	claims, err := h.verifyIDToken(r.Context(), doc, settings, tokens.IDToken, nonce)
	if err != nil {
		fail(err.Error())
		return
	}

	email := normalizeEmail(claims.Email)
	user, found := h.users.byEmail(email)
	if !found {
		domain := email[strings.LastIndex(email, "@")+1:]
		if settings.Domain == "" || domain != settings.Domain {
			fail("there's no account for " + email + ". Ask an admin to invite you.")
			return
		}
		user, err = h.users.put(User{Email: email, Name: firstNonEmpty(strings.TrimSpace(claims.Name), email), Role: roleEditor, PasswordHash: "sso"})
		if err != nil {
			fail("we couldn't create your account.")
			return
		}
		h.audit(r, user, "account.joined", "Joined through single sign-on as editor")
	}
	if err := h.signIn(w, r, user); err != nil {
		fail("we couldn't sign you in.")
		return
	}
	h.audit(r, user, "signin", "Signed in through single sign-on")
	http.Redirect(w, r, next, http.StatusSeeOther)
}

// ---------- settings and the login button ----------

// ssoFieldsIfPlanned is the settings section, or nothing when the plan
// has no single sign-on.
func (h *Host) ssoFieldsIfPlanned(settings cmsstore.SiteSettings) gosx.Node {
	if !h.featureOn(FeatureSSO) {
		return gosx.Fragment()
	}
	return h.renderSSOFields(settings, h.absoluteBaseFromSettings())
}

func (h *Host) renderSSOFields(settings cmsstore.SiteSettings, base string) gosx.Node {
	sso := h.sso()
	state, label := "draft", "Off"
	if sso.ready() {
		state, label = "published", "On"
	}
	return gosx.Fragment(
		gosx.El("h2", gosx.Attrs(gosx.Attr("class", "admin-subhead"), gosx.Attr("id", "sso")), gosx.Text("Single sign-on"),
			gosx.Text(" "), gosx.El("span", gosx.Attrs(gosx.Attr("class", "admin-badge"), gosx.Attr("data-state", state)), gosx.Text(label))),
		gosx.El("p", gosx.Attrs(gosx.Attr("class", "admin-hint")),
			gosx.Text("Let your team sign in with the account they already have at work: Google Workspace, Microsoft 365, Okta, Keycloak, or any OpenID Connect provider. Register this site there with the redirect address "),
			gosx.El("code", nil, gosx.Text(base+ssoCallback)), gosx.Text(", then paste what it gives you. If your company uses SAML instead, its identity provider (Okta, Entra ID, OneLogin, Keycloak) can publish the same directory as an OpenID Connect app; connect that here. There is no SAML to configure on this side.")),
		adminTextField("ssoIssuer", "Provider address (issuer)", settings.Metadata[ssoIssuerKey], "For Google: https://accounts.google.com. For Microsoft: https://login.microsoftonline.com/YOUR-TENANT-ID/v2.0. Others show it in their settings."),
		adminTextField("ssoClientId", "Client ID", settings.Metadata[ssoClientIDKey], ""),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "admin-field")),
			gosx.El("label", gosx.Attrs(gosx.Attr("for", "ssoClientSecret")), gosx.Text("Client secret")),
			gosx.El("input", gosx.Attrs(gosx.Attr("type", "password"), gosx.Attr("id", "ssoClientSecret"), gosx.Attr("name", "ssoClientSecret"), gosx.Attr("value", settings.Metadata[ssoSecretKey]), gosx.Attr("autocomplete", "off"))),
			gosx.El("small", nil, gosx.Text("Leave blank for providers that use PKCE without a secret.")),
		),
		adminTextField("ssoDomain", "Let anyone from this email domain in as an editor", settings.Metadata[ssoDomainKey], "For example yourbusiness.com. Leave blank so only people you've invited can sign in this way."),
		adminTextField("ssoLabel", "Button text", settings.Metadata[ssoLabelKey], "For example \"Sign in with Google\"."),
	)
}

func applySSOFields(r *http.Request, metadata cmsstore.Metadata) string {
	issuer := strings.TrimRight(strings.TrimSpace(r.PostFormValue("ssoIssuer")), "/")
	if issuer != "" && !strings.HasPrefix(issuer, "https://") && !allowInsecureSSO {
		return "The provider address must start with https://."
	}
	set := func(key, value string) {
		if value = strings.TrimSpace(value); value != "" {
			metadata[key] = value
		} else {
			delete(metadata, key)
		}
	}
	set(ssoIssuerKey, issuer)
	set(ssoClientIDKey, r.PostFormValue("ssoClientId"))
	set(ssoSecretKey, r.PostFormValue("ssoClientSecret"))
	set(ssoDomainKey, strings.ToLower(strings.TrimPrefix(strings.TrimSpace(r.PostFormValue("ssoDomain")), "@")))
	set(ssoLabelKey, r.PostFormValue("ssoLabel"))
	return ""
}

// ssoButton is the extra way in on the sign-in page.
func (h *Host) ssoButton(next string) gosx.Node {
	settings := h.sso()
	if !settings.ready() || !h.featureOn(FeatureSSO) {
		return gosx.Fragment()
	}
	href := ssoStartPath
	if next != "" {
		href += "?next=" + url.QueryEscape(next)
	}
	return gosx.El("p", gosx.Attrs(gosx.Attr("class", "wz-actions wz-actions--alt")),
		gosx.El("a", gosx.Attrs(gosx.Attr("class", "wz-btn wz-btn--ghost"), gosx.Attr("href", href), gosx.Attr("data-sso", "true")), gosx.Text(settings.Label)))
}
