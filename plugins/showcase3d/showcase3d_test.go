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
}
