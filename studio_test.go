package studio

import "testing"

func TestDefaultBoundaryNamesStudioAsAuthoringLayer(t *testing.T) {
	boundary := DefaultBoundary()

	if boundary.CMSPackage != PackageCMS {
		t.Fatalf("CMS package = %q, want %q", boundary.CMSPackage, PackageCMS)
	}
	if boundary.AdminPackage != PackageAdmin {
		t.Fatalf("admin package = %q, want %q", boundary.AdminPackage, PackageAdmin)
	}
	if boundary.StudioPackage != PackageStudio {
		t.Fatalf("studio package = %q, want %q", boundary.StudioPackage, PackageStudio)
	}
	if boundary.ProductPackage == "" {
		t.Fatal("product package placeholder should describe the composed product boundary")
	}
}

func TestDefaultFeaturesCoverAuthoringSurfaces(t *testing.T) {
	features := DefaultFeatures()
	if len(features) < 4 {
		t.Fatalf("expected at least 4 default features, got %d", len(features))
	}

	surfaces := map[SurfaceKind]bool{}
	for _, feature := range features {
		if feature.Key == "" || feature.Label == "" || feature.Summary == "" {
			t.Fatalf("feature should include key, label, and summary: %#v", feature)
		}
		surfaces[feature.Surface] = true
	}

	for _, surface := range []SurfaceKind{SurfaceCanvas, SurfaceInspector, SurfaceFlow, SurfacePublish} {
		if !surfaces[surface] {
			t.Fatalf("missing default feature for surface %q", surface)
		}
	}
}
