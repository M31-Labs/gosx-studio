package lifecycle

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"
)

const (
	PreviewTokenVersion = "v1"
	DefaultPreviewTTL   = 72 * time.Hour
	MaxPreviewTTL       = 30 * 24 * time.Hour
)

type PreviewLink struct {
	ResourceKind string    `json:"resourceKind"`
	ResourceID   string    `json:"resourceId"`
	Route        string    `json:"route,omitempty"`
	Audience     string    `json:"audience,omitempty"`
	Created      time.Time `json:"created"`
	Expires      time.Time `json:"expires"`
	Issuer       string    `json:"issuer,omitempty"`
	Nonce        string    `json:"nonce,omitempty"`
}

type PreviewLinkInput struct {
	ResourceKind string
	ResourceID   string
	Route        string
	Audience     string
	Created      time.Time
	Expires      time.Time
	TTL          time.Duration
	Issuer       string
	Nonce        string
}

type PreviewLinkOptions struct {
	BaseURL    string
	Path       string
	QueryParam string
}

func NewPreviewLink(input PreviewLinkInput, now time.Time) (PreviewLink, error) {
	if now.IsZero() {
		now = time.Now()
	}
	created := input.Created
	if created.IsZero() {
		created = now
	}
	ttl := input.TTL
	if ttl == 0 {
		ttl = DefaultPreviewTTL
	}
	expires := input.Expires
	if expires.IsZero() {
		expires = created.Add(ttl)
	}
	link := PreviewLink{
		ResourceKind: strings.TrimSpace(input.ResourceKind),
		ResourceID:   strings.TrimSpace(input.ResourceID),
		Route:        strings.TrimSpace(input.Route),
		Audience:     strings.TrimSpace(input.Audience),
		Created:      created.UTC(),
		Expires:      expires.UTC(),
		Issuer:       strings.TrimSpace(input.Issuer),
		Nonce:        strings.TrimSpace(input.Nonce),
	}
	if err := validatePreviewLink(link, now.UTC(), true); err != nil {
		return PreviewLink{}, err
	}
	return link, nil
}

func SignPreviewLink(link PreviewLink, secret []byte) (string, error) {
	if len(secret) == 0 {
		return "", fmt.Errorf("preview link secret is required")
	}
	link = normalizePreviewLink(link)
	if err := validatePreviewLink(link, time.Time{}, false); err != nil {
		return "", err
	}
	payload, err := json.Marshal(link)
	if err != nil {
		return "", fmt.Errorf("preview link payload: %w", err)
	}
	payloadPart := base64.RawURLEncoding.EncodeToString(payload)
	signingInput := PreviewTokenVersion + "." + payloadPart
	signature := signPreviewPayload([]byte(signingInput), secret)
	return signingInput + "." + base64.RawURLEncoding.EncodeToString(signature), nil
}

func VerifyPreviewLink(token string, secret []byte, now time.Time) (PreviewLink, error) {
	if len(secret) == 0 {
		return PreviewLink{}, fmt.Errorf("preview link secret is required")
	}
	if now.IsZero() {
		now = time.Now()
	}
	parts := strings.Split(strings.TrimSpace(token), ".")
	if len(parts) != 3 || parts[0] != PreviewTokenVersion {
		return PreviewLink{}, fmt.Errorf("unsupported preview token")
	}
	signature, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return PreviewLink{}, fmt.Errorf("decode preview signature: %w", err)
	}
	expected := signPreviewPayload([]byte(parts[0]+"."+parts[1]), secret)
	if !hmac.Equal(signature, expected) {
		return PreviewLink{}, fmt.Errorf("preview token signature mismatch")
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return PreviewLink{}, fmt.Errorf("decode preview payload: %w", err)
	}
	var link PreviewLink
	if err := json.Unmarshal(payload, &link); err != nil {
		return PreviewLink{}, fmt.Errorf("decode preview link: %w", err)
	}
	link = normalizePreviewLink(link)
	if err := validatePreviewLink(link, now.UTC(), true); err != nil {
		return PreviewLink{}, err
	}
	if !now.UTC().Before(link.Expires) {
		return PreviewLink{}, fmt.Errorf("preview token expired")
	}
	return link, nil
}

func PreviewURL(link PreviewLink, token string, options PreviewLinkOptions) (string, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return "", fmt.Errorf("preview token is required")
	}
	queryParam := strings.TrimSpace(options.QueryParam)
	if queryParam == "" {
		queryParam = "preview_token"
	}
	path := strings.TrimSpace(options.Path)
	if path == "" {
		path = strings.TrimSpace(link.Route)
	}
	if path == "" {
		path = "/"
	}
	base := strings.TrimSpace(options.BaseURL)
	var parsed *url.URL
	var err error
	if base == "" {
		parsed, err = url.Parse(path)
	} else {
		baseURL, err := url.Parse(base)
		if err != nil {
			return "", fmt.Errorf("parse preview base url: %w", err)
		}
		ref, err := url.Parse(path)
		if err != nil {
			return "", fmt.Errorf("parse preview path: %w", err)
		}
		parsed = baseURL.ResolveReference(ref)
	}
	if err != nil {
		return "", fmt.Errorf("parse preview path: %w", err)
	}
	q := parsed.Query()
	q.Set(queryParam, token)
	parsed.RawQuery = q.Encode()
	return parsed.String(), nil
}

func normalizePreviewLink(link PreviewLink) PreviewLink {
	link.ResourceKind = strings.TrimSpace(link.ResourceKind)
	link.ResourceID = strings.TrimSpace(link.ResourceID)
	link.Route = strings.TrimSpace(link.Route)
	link.Audience = strings.TrimSpace(link.Audience)
	link.Issuer = strings.TrimSpace(link.Issuer)
	link.Nonce = strings.TrimSpace(link.Nonce)
	if !link.Created.IsZero() {
		link.Created = link.Created.UTC()
	}
	if !link.Expires.IsZero() {
		link.Expires = link.Expires.UTC()
	}
	return link
}

func validatePreviewLink(link PreviewLink, now time.Time, enforceWindow bool) error {
	if strings.TrimSpace(link.ResourceKind) == "" || strings.TrimSpace(link.ResourceID) == "" {
		return fmt.Errorf("preview link requires resource kind and resource id")
	}
	if link.Created.IsZero() || link.Expires.IsZero() {
		return fmt.Errorf("preview link requires created and expires timestamps")
	}
	if !link.Expires.After(link.Created) {
		return fmt.Errorf("preview link expires before it becomes valid")
	}
	if link.Expires.Sub(link.Created) > MaxPreviewTTL {
		return fmt.Errorf("preview link exceeds maximum ttl")
	}
	if enforceWindow && !now.IsZero() && now.Before(link.Created.Add(-time.Minute)) {
		return fmt.Errorf("preview token is not valid yet")
	}
	return nil
}

func signPreviewPayload(payload, secret []byte) []byte {
	mac := hmac.New(sha256.New, secret)
	mac.Write(payload)
	return mac.Sum(nil)
}
