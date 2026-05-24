package showcase3d

import (
	"strings"

	studio "github.com/M31-Labs/gosx-studio"
)

const (
	FeatureKey = "showcase-3d"
	PluginName = "Showcase 3D"
)

type AssetSource string

const (
	AssetSourcePhotoGenerated AssetSource = "photo-generated"
	AssetSourceUploadedModel  AssetSource = "uploaded-model"
)

type LifecycleState string

const (
	LifecycleDraft      LifecycleState = "draft"
	LifecycleGenerating LifecycleState = "generating"
	LifecycleReview     LifecycleState = "review"
	LifecycleApproved   LifecycleState = "approved"
	LifecyclePublished  LifecycleState = "published"
	LifecycleRejected   LifecycleState = "rejected"
	LifecycleReplaced   LifecycleState = "replaced"
)

type ModerationStatus string

const (
	ModerationPending  ModerationStatus = "pending"
	ModerationApproved ModerationStatus = "approved"
	ModerationFlagged  ModerationStatus = "flagged"
	ModerationRejected ModerationStatus = "rejected"
)

type ArtifactFormat string

const (
	ArtifactGLB    ArtifactFormat = "glb"
	ArtifactUSDZ   ArtifactFormat = "usdz"
	ArtifactPoster ArtifactFormat = "poster"
)

type Config struct {
	Enabled             bool
	AssetSource         AssetSource
	PopoutEnabled       bool
	ProviderAdapterKey  string
	PlacementPolicyRef  string
	DefaultComponent    string
	DefaultModelBinding string
}

func DefaultConfig() Config {
	return Config{
		Enabled:             true,
		AssetSource:         AssetSourcePhotoGenerated,
		PopoutEnabled:       true,
		ProviderAdapterKey:  "showcase3d.provider",
		PlacementPolicyRef:  "showcase3d.placement-policy",
		DefaultComponent:    "Showcase3DViewer",
		DefaultModelBinding: "showcase3d.model",
	}
}

func Feature() studio.Feature {
	return studio.Feature{
		Key:     FeatureKey,
		Label:   PluginName,
		Surface: studio.SurfaceShowcase3D,
		Summary: "CMS showcase plugin for AI-generated 3D piece assets, Studio placement controls, and public pop-out viewing.",
	}
}

type SourcePhoto struct {
	Key           string
	Label         string
	URI           string
	AltText       string
	Hash          string
	PermissionRef string
}

type ModelArtifact struct {
	Key         string
	Label       string
	URI         string
	Format      ArtifactFormat
	ContentType string
	Hash        string
	ByteSize    int64
}

type Provenance struct {
	Provider        string
	ProviderModel   string
	RequestID       string
	GeneratedAt     string
	SourcePhotoHash string
	PromptSummary   string
}

type ShowcaseAsset struct {
	Key          string
	PieceID      string
	Label        string
	Summary      string
	SourcePhotos []SourcePhoto
	Artifacts    []ModelArtifact
	Provenance   Provenance
	Moderation   ModerationStatus
	Lifecycle    LifecycleState
}

type ReadinessCheck struct {
	Key     string
	Label   string
	Summary string
	Status  studio.ReadinessStatus
}

type PlacementControls struct {
	ModelBinding     string
	PhotoBinding     string
	PlacementBinding string
	PopoutBinding    string
	Component        string
}

type ViewerSurface struct {
	Key          string
	Label        string
	Component    string
	Engine       studio.EngineKind
	Inline       bool
	Popout       bool
	ModelURI     string
	PosterURI    string
	FallbackText string
}

func DefaultPlacementControls() PlacementControls {
	config := DefaultConfig()
	return PlacementControls{
		ModelBinding:     config.DefaultModelBinding,
		PhotoBinding:     "showcase3d.photos",
		PlacementBinding: "showcase3d.placement",
		PopoutBinding:    "showcase3d.popout",
		Component:        config.DefaultComponent,
	}
}

func ComponentTemplate() studio.ComponentTemplate {
	controls := DefaultPlacementControls()
	return studio.ComponentTemplate{
		Key:            FeatureKey,
		Label:          PluginName,
		Summary:        "Place an approved generated model with inline and pop-out viewing.",
		Category:       "Plugins",
		GoSXComponent:  controls.Component,
		Source:         studio.ComponentSourcePlugin,
		DefaultBinding: controls.ModelBinding,
		Status:         "Plugin",
		AddLabel:       "Add 3D viewer",
		Controls:       AuthoringControls(controls),
	}
}

func AuthoringControls(controls PlacementControls) []studio.Control {
	controls = controls.Normalize()
	return []studio.Control{
		{Key: "model", Label: "Model", Kind: studio.ControlScene3D, Binding: controls.ModelBinding, Required: true},
		{Key: "photos", Label: "Source photos", Kind: studio.ControlMedia, Binding: controls.PhotoBinding, Required: true},
		{Key: "placement", Label: "Placement", Kind: studio.ControlChoice, Binding: controls.PlacementBinding, Options: []studio.ControlOption{
			{Value: "inline", Label: "Inline"},
			{Value: "feature", Label: "Feature section"},
			{Value: "product-popout", Label: "Product pop-out"},
		}},
		{Key: "popout", Label: "Pop-out viewer", Kind: studio.ControlToggle, Binding: controls.PopoutBinding},
	}
}

func (controls PlacementControls) Normalize() PlacementControls {
	defaults := DefaultPlacementControls()
	controls.ModelBinding = firstNonEmpty(controls.ModelBinding, defaults.ModelBinding)
	controls.PhotoBinding = firstNonEmpty(controls.PhotoBinding, defaults.PhotoBinding)
	controls.PlacementBinding = firstNonEmpty(controls.PlacementBinding, defaults.PlacementBinding)
	controls.PopoutBinding = firstNonEmpty(controls.PopoutBinding, defaults.PopoutBinding)
	controls.Component = firstNonEmpty(controls.Component, defaults.Component)
	return controls
}

func (asset ShowcaseAsset) Normalize() ShowcaseAsset {
	asset.Key = strings.TrimSpace(asset.Key)
	asset.PieceID = strings.TrimSpace(asset.PieceID)
	asset.Label = strings.TrimSpace(asset.Label)
	asset.Summary = strings.TrimSpace(asset.Summary)
	asset.SourcePhotos = normalizeSourcePhotos(asset.SourcePhotos)
	asset.Artifacts = normalizeModelArtifacts(asset.Artifacts)
	asset.Provenance = asset.Provenance.Normalize()
	asset.Moderation = normalizeModerationStatus(asset.Moderation)
	asset.Lifecycle = normalizeLifecycleState(asset.Lifecycle)
	return asset
}

func (asset ShowcaseAsset) ReadinessChecks() []ReadinessCheck {
	asset = asset.Normalize()
	return []ReadinessCheck{
		{
			Key:     "photos",
			Label:   "Source photos",
			Summary: sourcePhotoSummary(asset.SourcePhotos),
			Status:  readinessFromBool(len(asset.SourcePhotos) > 0),
		},
		{
			Key:     "model",
			Label:   "Generated model",
			Summary: modelArtifactSummary(asset.Artifacts),
			Status:  readinessFromBool(asset.ModelArtifact().URI != ""),
		},
		{
			Key:     "moderation",
			Label:   "Moderation",
			Summary: moderationSummary(asset.Moderation),
			Status:  moderationReadiness(asset.Moderation),
		},
		{
			Key:     "lifecycle",
			Label:   "Lifecycle",
			Summary: lifecycleSummary(asset.Lifecycle),
			Status:  lifecycleReadiness(asset.Lifecycle),
		},
	}
}

func (asset ShowcaseAsset) PublishReady() bool {
	for _, check := range asset.ReadinessChecks() {
		if check.Status != studio.ReadinessReady {
			return false
		}
	}
	return true
}

func (asset ShowcaseAsset) ModelArtifact() ModelArtifact {
	for _, artifact := range asset.Normalize().Artifacts {
		if artifact.Format == ArtifactGLB {
			return artifact
		}
	}
	return ModelArtifact{}
}

func (asset ShowcaseAsset) PosterArtifact() ModelArtifact {
	for _, artifact := range asset.Normalize().Artifacts {
		if artifact.Format == ArtifactPoster {
			return artifact
		}
	}
	return ModelArtifact{}
}

func Viewer(asset ShowcaseAsset, controls PlacementControls, popout bool) ViewerSurface {
	asset = asset.Normalize()
	controls = controls.Normalize()
	model := asset.ModelArtifact()
	poster := asset.PosterArtifact()
	return ViewerSurface{
		Key:          firstNonEmpty(asset.Key, asset.PieceID, FeatureKey),
		Label:        firstNonEmpty(asset.Label, PluginName),
		Component:    controls.Component,
		Engine:       studio.EngineScene3D,
		Inline:       true,
		Popout:       popout,
		ModelURI:     model.URI,
		PosterURI:    poster.URI,
		FallbackText: "3D viewer unavailable. Show the approved poster image instead.",
	}
}

func (photo SourcePhoto) Normalize() SourcePhoto {
	photo.Key = strings.TrimSpace(photo.Key)
	photo.Label = strings.TrimSpace(photo.Label)
	photo.URI = strings.TrimSpace(photo.URI)
	photo.AltText = strings.TrimSpace(photo.AltText)
	photo.Hash = strings.TrimSpace(photo.Hash)
	photo.PermissionRef = strings.TrimSpace(photo.PermissionRef)
	return photo
}

func (artifact ModelArtifact) Normalize() ModelArtifact {
	artifact.Key = strings.TrimSpace(artifact.Key)
	artifact.Label = strings.TrimSpace(artifact.Label)
	artifact.URI = strings.TrimSpace(artifact.URI)
	artifact.Format = normalizeArtifactFormat(artifact.Format)
	artifact.ContentType = strings.TrimSpace(artifact.ContentType)
	artifact.Hash = strings.TrimSpace(artifact.Hash)
	return artifact
}

func (provenance Provenance) Normalize() Provenance {
	provenance.Provider = strings.TrimSpace(provenance.Provider)
	provenance.ProviderModel = strings.TrimSpace(provenance.ProviderModel)
	provenance.RequestID = strings.TrimSpace(provenance.RequestID)
	provenance.GeneratedAt = strings.TrimSpace(provenance.GeneratedAt)
	provenance.SourcePhotoHash = strings.TrimSpace(provenance.SourcePhotoHash)
	provenance.PromptSummary = strings.TrimSpace(provenance.PromptSummary)
	return provenance
}

func normalizeSourcePhotos(photos []SourcePhoto) []SourcePhoto {
	out := make([]SourcePhoto, 0, len(photos))
	for _, photo := range photos {
		photo = photo.Normalize()
		if photo.Key == "" || photo.URI == "" {
			continue
		}
		out = append(out, photo)
	}
	return out
}

func normalizeModelArtifacts(artifacts []ModelArtifact) []ModelArtifact {
	out := make([]ModelArtifact, 0, len(artifacts))
	for _, artifact := range artifacts {
		artifact = artifact.Normalize()
		if artifact.Key == "" || artifact.URI == "" {
			continue
		}
		out = append(out, artifact)
	}
	return out
}

func normalizeArtifactFormat(format ArtifactFormat) ArtifactFormat {
	switch ArtifactFormat(strings.TrimSpace(string(format))) {
	case ArtifactUSDZ:
		return ArtifactUSDZ
	case ArtifactPoster:
		return ArtifactPoster
	default:
		return ArtifactGLB
	}
}

func normalizeModerationStatus(status ModerationStatus) ModerationStatus {
	switch ModerationStatus(strings.TrimSpace(string(status))) {
	case ModerationApproved:
		return ModerationApproved
	case ModerationFlagged:
		return ModerationFlagged
	case ModerationRejected:
		return ModerationRejected
	default:
		return ModerationPending
	}
}

func normalizeLifecycleState(state LifecycleState) LifecycleState {
	switch LifecycleState(strings.TrimSpace(string(state))) {
	case LifecycleGenerating:
		return LifecycleGenerating
	case LifecycleReview:
		return LifecycleReview
	case LifecycleApproved:
		return LifecycleApproved
	case LifecyclePublished:
		return LifecyclePublished
	case LifecycleRejected:
		return LifecycleRejected
	case LifecycleReplaced:
		return LifecycleReplaced
	default:
		return LifecycleDraft
	}
}

func readinessFromBool(ok bool) studio.ReadinessStatus {
	if ok {
		return studio.ReadinessReady
	}
	return studio.ReadinessBlocked
}

func sourcePhotoSummary(photos []SourcePhoto) string {
	if len(photos) == 0 {
		return "Add source photos before generating or publishing a viewer."
	}
	return "Source photos are recorded for provenance and moderation."
}

func modelArtifactSummary(artifacts []ModelArtifact) string {
	for _, artifact := range artifacts {
		if artifact.Format == ArtifactGLB && artifact.URI != "" {
			return "Approved GLB model artifact is available for Scene3D."
		}
	}
	return "Generate or upload a GLB model artifact before publishing."
}

func moderationSummary(status ModerationStatus) string {
	switch status {
	case ModerationApproved:
		return "Model is approved for public viewing."
	case ModerationFlagged:
		return "Model needs moderation review."
	case ModerationRejected:
		return "Model was rejected and cannot be published."
	default:
		return "Model is waiting for moderation."
	}
}

func moderationReadiness(status ModerationStatus) studio.ReadinessStatus {
	switch status {
	case ModerationApproved:
		return studio.ReadinessReady
	case ModerationFlagged:
		return studio.ReadinessWatch
	default:
		return studio.ReadinessBlocked
	}
}

func lifecycleSummary(state LifecycleState) string {
	switch state {
	case LifecycleApproved, LifecyclePublished:
		return "Showcase asset is approved for placement."
	case LifecycleGenerating:
		return "Generation is still running."
	case LifecycleReview:
		return "Review the generated model before publishing."
	case LifecycleRejected:
		return "Rejected showcase assets cannot be published."
	case LifecycleReplaced:
		return "This showcase asset has been replaced."
	default:
		return "Showcase asset is still a draft."
	}
}

func lifecycleReadiness(state LifecycleState) studio.ReadinessStatus {
	switch state {
	case LifecycleApproved, LifecyclePublished:
		return studio.ReadinessReady
	case LifecycleReview:
		return studio.ReadinessWatch
	default:
		return studio.ReadinessBlocked
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if normalized := strings.TrimSpace(value); normalized != "" {
			return normalized
		}
	}
	return ""
}
