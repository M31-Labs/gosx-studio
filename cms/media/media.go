package media

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"path/filepath"
	"strings"
	"time"

	"m31labs.dev/gosx-studio/core"
)

type Item struct {
	URL string `json:"url"`
	Alt string `json:"alt"`
}

type Asset struct {
	ID          string     `json:"id"`
	URL         string     `json:"url"`
	Alt         string     `json:"alt"`
	Caption     string     `json:"caption,omitempty"`
	Filename    string     `json:"filename"`
	ContentType string     `json:"contentType"`
	Size        int64      `json:"size"`
	Variants    Variants   `json:"variants,omitempty"`
	FocalX      float64    `json:"focalX,omitempty"`
	FocalY      float64    `json:"focalY,omitempty"`
	ArchivedAt  *time.Time `json:"archivedAt,omitempty"`
	Created     time.Time  `json:"created"`
	Updated     time.Time  `json:"updated"`
}

type Variants map[string]Variant

type Variant struct {
	URL         string `json:"url"`
	Width       int    `json:"width,omitempty"`
	Height      int    `json:"height,omitempty"`
	ContentType string `json:"contentType,omitempty"`
	Size        int64  `json:"size,omitempty"`
}

type FocalPoint struct {
	X float64
	Y float64
}

type Usage struct {
	Kind  string
	Title string
	ID    string
	Href  string
}

type Input struct {
	URL         string
	Alt         string
	Caption     string
	Filename    string
	ContentType string
	Size        int64
	Variants    Variants
	FocalX      float64
	FocalY      float64
}

type Filter struct {
	Archived *bool
}

type Store interface {
	ListMedia(...Filter) []Asset
	MediaByID(string) (Asset, bool)
	MediaUsage(string) []Usage
	CreateMediaAsset(Input) (Asset, error)
	UpdateMediaAsset(string, Input) (Asset, error)
}

type Upload struct {
	Filename    string
	ContentType string
	Size        int64
	Alt         string
	Reader      io.Reader
}

type StoredObject struct {
	URL         string
	Filename    string
	ContentType string
	Size        int64
}

type UploadPolicy struct {
	MaxBytes        int64
	ContentTypes    map[string]string
	ExtensionTypes  map[string]string
	GenerateVariant bool
}

type Storage interface {
	Save(context.Context, Upload) (StoredObject, error)
	Delete(context.Context, StoredObject) error
}

type VariantGenerator interface {
	Generate(context.Context, StoredObject) (Variants, []StoredObject, error)
}

type PickerField struct {
	Name       string
	Label      string
	URL        string
	Alt        string
	Required   bool
	Assets     []Asset
	AssetCount int
	HasAssets  bool
}

func Picker(name, label string, value Item, assets []Asset, required bool) PickerField {
	assets = CloneAssets(assets)
	return PickerField{
		Name:       strings.TrimSpace(name),
		Label:      strings.TrimSpace(label),
		URL:        strings.TrimSpace(value.URL),
		Alt:        strings.TrimSpace(value.Alt),
		Required:   required,
		Assets:     assets,
		AssetCount: len(assets),
		HasAssets:  len(assets) > 0,
	}
}

func DefaultUploadPolicy() UploadPolicy {
	return UploadPolicy{
		MaxBytes: 12 << 20,
		ContentTypes: map[string]string{
			"image/gif":  ".gif",
			"image/jpeg": ".jpg",
			"image/png":  ".png",
			"image/webp": ".webp",
		},
		ExtensionTypes: map[string]string{
			".ico":   "image/x-icon",
			".woff":  "font/woff",
			".woff2": "font/woff2",
			".ttf":   "font/ttf",
			".otf":   "font/otf",
		},
		GenerateVariant: true,
	}
}

func (policy UploadPolicy) AllowedExtension(contentType, original string) (string, bool) {
	contentType = strings.TrimSpace(strings.ToLower(contentType))
	if ext, ok := policy.ContentTypes[contentType]; ok {
		return ext, true
	}
	ext := strings.ToLower(filepath.Ext(original))
	if ext == "" {
		return "", false
	}
	if allowed, ok := policy.ExtensionTypes[ext]; ok && allowed != "" {
		return ext, true
	}
	return "", false
}

func (policy UploadPolicy) Allows(contentType, original string, size int64) bool {
	if policy.MaxBytes > 0 && size > policy.MaxBytes {
		return false
	}
	_, ok := policy.AllowedExtension(contentType, original)
	return ok
}

func InputFromStoredObject(object StoredObject, alt string, variants Variants) Input {
	return Input{
		URL:         strings.TrimSpace(object.URL),
		Alt:         strings.TrimSpace(alt),
		Filename:    strings.TrimSpace(object.Filename),
		ContentType: strings.TrimSpace(object.ContentType),
		Size:        object.Size,
		Variants:    NormalizeVariants(variants),
	}
}

func NormalizeItem(item Item, fallbackTitle string) Item {
	return Item{
		URL: strings.TrimSpace(item.URL),
		Alt: firstNonEmpty(item.Alt, fallbackTitle),
	}
}

func NormalizeList(items []Item, fallbackTitle string, placeholder bool) []Item {
	out := make([]Item, 0, len(items))
	seen := map[string]bool{}
	for _, item := range items {
		normalized := NormalizeItem(item, fallbackTitle)
		if strings.TrimSpace(normalized.URL) == "" {
			continue
		}
		key := strings.ToLower(normalized.URL)
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, normalized)
	}
	if len(out) == 0 && placeholder {
		return []Item{{URL: "/media/placeholder.svg", Alt: firstNonEmpty(fallbackTitle, "Image")}}
	}
	return out
}

func NormalizeAsset(input Input, asset Asset) Asset {
	asset.URL = strings.TrimSpace(input.URL)
	asset.Alt = strings.TrimSpace(input.Alt)
	asset.Caption = strings.TrimSpace(input.Caption)
	asset.Filename = strings.TrimSpace(input.Filename)
	asset.FocalX = input.FocalX
	asset.FocalY = input.FocalY
	if strings.TrimSpace(input.ContentType) != "" {
		asset.ContentType = strings.TrimSpace(input.ContentType)
	}
	if input.Size > 0 {
		asset.Size = input.Size
	}
	if input.Variants != nil {
		asset.Variants = NormalizeVariants(input.Variants)
	}
	if asset.Filename == "" {
		asset.Filename = filepath.Base(strings.TrimSpace(input.URL))
	}
	return asset
}

func NormalizeVariants(variants Variants) Variants {
	if len(variants) == 0 {
		return nil
	}
	out := Variants{}
	for name, variant := range variants {
		name = strings.TrimSpace(name)
		variant.URL = strings.TrimSpace(variant.URL)
		variant.ContentType = strings.TrimSpace(variant.ContentType)
		if name == "" || variant.URL == "" {
			continue
		}
		out[name] = variant
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func NormalizeFocalPoint(x, y float64) FocalPoint {
	if x < 0 || x > 1 {
		x = 0
	}
	if y < 0 || y > 1 {
		y = 0
	}
	return FocalPoint{X: x, Y: y}
}

func ParseLines(value string) []Item {
	out := []Item{}
	for _, line := range strings.Split(value, "\n") {
		parts := strings.SplitN(strings.TrimSpace(line), "|", 2)
		if strings.TrimSpace(parts[0]) == "" {
			continue
		}
		alt := ""
		if len(parts) > 1 {
			alt = strings.TrimSpace(parts[1])
		}
		out = append(out, Item{URL: strings.TrimSpace(parts[0]), Alt: alt})
	}
	return out
}

func FormatLines(items []Item) string {
	lines := make([]string, 0, len(items))
	for _, item := range items {
		item = NormalizeItem(item, "")
		if item.URL == "" {
			continue
		}
		line := item.URL
		if item.Alt != "" {
			line += " | " + item.Alt
		}
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n")
}

func CloneAsset(asset Asset) Asset {
	raw, _ := json.Marshal(asset)
	var out Asset
	_ = json.Unmarshal(raw, &out)
	return out
}

func CloneAssets(assets []Asset) []Asset {
	out := make([]Asset, len(assets))
	for i, asset := range assets {
		out[i] = CloneAsset(asset)
	}
	return out
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

// ErrAssetReferenced signals that a caller attempted to permanently delete an
// asset that MediaUsage still reports as referenced by draft or published
// content. Hosts should surface the returned Usage list ("still used by
// <Product>, <Page>...") instead of deleting, matching Webflow-class editors'
// referenced-asset protection.
var ErrAssetReferenced = errors.New("asset is still referenced by draft or published content")

// UsageLookup is the narrow read capability GuardDeleteMediaAsset needs from a
// host's media store.
type UsageLookup interface {
	MediaUsage(id string) []Usage
}

// AssetDeleter is the narrow write capability GuardDeleteMediaAsset needs from
// a host's media store: a PERMANENT delete (distinct from archive/restore,
// which are reversible and already usage-agnostic).
type AssetDeleter interface {
	DeleteMediaAsset(id string) error
}

// GuardDeleteMediaAsset refuses to delete an asset that lookup.MediaUsage
// still reports as referenced, returning that usage list so the caller can
// render "still used by X, Y" instead of silently deleting a referenced asset
// or leaving a dangling reference behind. Studio owns this guard so every
// host enforces the same referenced-asset protection rather than
// reimplementing (and potentially forgetting) the usage check.
func GuardDeleteMediaAsset(lookup UsageLookup, deleter AssetDeleter, id string) ([]Usage, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, errors.New("asset id is required")
	}
	if lookup != nil {
		if usage := lookup.MediaUsage(id); len(usage) > 0 {
			return usage, ErrAssetReferenced
		}
	}
	if deleter == nil {
		return nil, errors.New("no asset deleter is configured")
	}
	return nil, deleter.DeleteMediaAsset(id)
}

// AssetReadinessCheck reports one publish-readiness concern for a media
// asset, mirroring core.FlowReadinessCheck's shape so hosts can fold media
// diagnostics into the same readiness/publish-review surface.
type AssetReadinessCheck struct {
	Key     string
	Label   string
	Summary string
	Status  core.ReadinessStatus
}

// AssetReadiness computes the accessibility/publish-readiness checks for one
// asset given its current usage. Alt text is only REQUIRED once an asset is
// actually bound to visible content ("alt text required where semantically
// needed" — an unused library upload is Watch, not Blocked, so operators can
// stage assets before wiring them up).
func AssetReadiness(asset Asset, usage []Usage) []AssetReadinessCheck {
	checks := []AssetReadinessCheck{}
	referenced := len(usage) > 0
	hasAlt := strings.TrimSpace(asset.Alt) != ""
	switch {
	case hasAlt:
		checks = append(checks, AssetReadinessCheck{
			Key: "alt-text", Label: "Alt text", Summary: "Alt text is set.", Status: core.ReadinessReady,
		})
	case referenced:
		checks = append(checks, AssetReadinessCheck{
			Key: "alt-text", Label: "Alt text", Summary: "This asset is used on the live site and has no alt text.", Status: core.ReadinessBlocked,
		})
	default:
		checks = append(checks, AssetReadinessCheck{
			Key: "alt-text", Label: "Alt text", Summary: "Add alt text before using this asset on a page.", Status: core.ReadinessWatch,
		})
	}
	return checks
}

// AssetReadinessStatus rolls AssetReadiness up to a single status the way
// core.Flow.ReadinessStatus rolls up flow checks: any Blocked check blocks,
// any Watch (with no Blocked) watches, otherwise ready.
func AssetReadinessStatus(asset Asset, usage []Usage) core.ReadinessStatus {
	status := core.ReadinessReady
	for _, check := range AssetReadiness(asset, usage) {
		switch check.Status {
		case core.ReadinessBlocked:
			return core.ReadinessBlocked
		case core.ReadinessWatch:
			status = core.ReadinessWatch
		}
	}
	return status
}
