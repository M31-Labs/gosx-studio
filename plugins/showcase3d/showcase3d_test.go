package showcase3d

import (
	"testing"

	studio "github.com/M31-Labs/gosx-studio"
)

func TestFeatureDeclaresShowcaseSurface(t *testing.T) {
	feature := Feature()

	if feature.Key != FeatureKey {
		t.Fatalf("feature key = %q, want %q", feature.Key, FeatureKey)
	}
	if feature.Surface != studio.SurfaceShowcase3D {
		t.Fatalf("feature surface = %q, want %q", feature.Surface, studio.SurfaceShowcase3D)
	}
	if feature.Summary == "" {
		t.Fatal("feature summary should explain the plugin boundary")
	}
}

func TestDefaultConfigEnablesPhotoGeneratedPopout(t *testing.T) {
	config := DefaultConfig()

	if !config.Enabled {
		t.Fatal("default config should enable the plugin")
	}
	if config.AssetSource != AssetSourcePhotoGenerated {
		t.Fatalf("asset source = %q, want %q", config.AssetSource, AssetSourcePhotoGenerated)
	}
	if !config.PopoutEnabled {
		t.Fatal("default config should enable public pop-out viewing")
	}
	if config.ProviderAdapterKey == "" || config.PlacementPolicyRef == "" {
		t.Fatalf("config should reference host-owned provider and placement policy keys: %#v", config)
	}
}

func TestShowcaseAssetTracksPhotosModelProvenanceAndReadiness(t *testing.T) {
	asset := ShowcaseAsset{
		Key:     " piece-1 ",
		PieceID: " product-1 ",
		Label:   " Small bowl ",
		SourcePhotos: []SourcePhoto{
			{Key: " front ", Label: " Front ", URI: " /media/front.jpg ", AltText: " Bowl front ", Hash: " sha256:front ", PermissionRef: " product.media.release "},
			{Label: "ignored"},
		},
		Artifacts: []ModelArtifact{
			{Key: " poster ", URI: " /models/bowl.webp ", Format: ArtifactPoster, ContentType: " image/webp "},
			{Key: " model ", URI: " /models/bowl.glb ", Format: ArtifactGLB, ContentType: " model/gltf-binary ", ByteSize: 42000},
			{Label: "ignored"},
		},
		Provenance: Provenance{
			Provider:        " Generator ",
			ProviderModel:   " photo-to-3d-v1 ",
			RequestID:       " req_123 ",
			GeneratedAt:     " 2026-05-24T12:00:00Z ",
			SourcePhotoHash: " sha256:set ",
			PromptSummary:   " ceramic bowl ",
		},
		Moderation: ModerationApproved,
		Lifecycle:  LifecycleApproved,
	}

	asset = asset.Normalize()
	if asset.Key != "piece-1" || asset.SourcePhotos[0].URI != "/media/front.jpg" {
		t.Fatalf("asset normalization = %#v", asset)
	}
	if len(asset.SourcePhotos) != 1 || len(asset.Artifacts) != 2 {
		t.Fatalf("asset should filter incomplete photos/artifacts: %#v", asset)
	}
	if asset.ModelArtifact().URI != "/models/bowl.glb" || asset.PosterArtifact().URI != "/models/bowl.webp" {
		t.Fatalf("artifacts = model %#v poster %#v", asset.ModelArtifact(), asset.PosterArtifact())
	}
	if !asset.PublishReady() {
		t.Fatalf("asset should be publish-ready: %#v", asset.ReadinessChecks())
	}
}

func TestShowcaseAssetReadinessBlocksUnmoderatedDrafts(t *testing.T) {
	asset := ShowcaseAsset{
		Key:       "piece-1",
		Label:     "Small bowl",
		Lifecycle: LifecycleDraft,
	}

	if asset.PublishReady() {
		t.Fatal("draft without photos, model, and moderation should not publish")
	}
	checks := asset.ReadinessChecks()
	if len(checks) != 4 {
		t.Fatalf("checks = %#v", checks)
	}
	for _, check := range checks {
		if check.Status == studio.ReadinessReady {
			t.Fatalf("check should not be ready: %#v", check)
		}
	}
}

func TestAuthoringControlsAndTemplateExposeNoCodeScene3DPlacement(t *testing.T) {
	controls := AuthoringControls(PlacementControls{})
	if len(controls) != 4 {
		t.Fatalf("controls = %#v", controls)
	}
	if controls[0].Kind != studio.ControlScene3D || controls[0].Binding != "showcase3d.model" || !controls[0].Required {
		t.Fatalf("model control = %#v", controls[0])
	}
	if controls[1].Kind != studio.ControlMedia || controls[2].Kind != studio.ControlChoice || controls[3].Kind != studio.ControlToggle {
		t.Fatalf("control kinds = %#v", controls)
	}

	template := ComponentTemplate()
	if template.GoSXComponent != "Showcase3DViewer" || template.Source != studio.ComponentSourcePlugin {
		t.Fatalf("template = %#v", template)
	}
	if template.ControlCount() != 4 || template.DefaultBinding != "showcase3d.model" {
		t.Fatalf("template controls = %#v", template)
	}
}

func TestViewerDeclaresScene3DSurfaceWithoutProviderSecrets(t *testing.T) {
	asset := ShowcaseAsset{
		Key:        "piece-1",
		Label:      "Small bowl",
		Moderation: ModerationApproved,
		Lifecycle:  LifecycleApproved,
		SourcePhotos: []SourcePhoto{
			{Key: "front", URI: "/media/front.jpg"},
		},
		Artifacts: []ModelArtifact{
			{Key: "model", URI: "/models/bowl.glb", Format: ArtifactGLB},
			{Key: "poster", URI: "/models/bowl.webp", Format: ArtifactPoster},
		},
	}

	viewer := Viewer(asset, PlacementControls{}, true)
	if viewer.Engine != studio.EngineScene3D || viewer.Component != "Showcase3DViewer" {
		t.Fatalf("viewer surface = %#v", viewer)
	}
	if !viewer.Inline || !viewer.Popout || viewer.ModelURI != "/models/bowl.glb" || viewer.PosterURI != "/models/bowl.webp" {
		t.Fatalf("viewer asset refs = %#v", viewer)
	}
}
