package showcase3d

import studio "github.com/M31-Labs/gosx-studio"

const (
	FeatureKey = "showcase-3d"
	PluginName = "Showcase 3D"
)

type AssetSource string

const (
	AssetSourcePhotoGenerated AssetSource = "photo-generated"
	AssetSourceUploadedModel  AssetSource = "uploaded-model"
)

type Config struct {
	Enabled       bool
	AssetSource   AssetSource
	PopoutEnabled bool
}

func DefaultConfig() Config {
	return Config{
		Enabled:       true,
		AssetSource:   AssetSourcePhotoGenerated,
		PopoutEnabled: true,
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
