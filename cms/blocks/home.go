package blocks

import "m31labs.dev/gosx-admin/blockstudio"

func HomeCatalog() []blockstudio.Definition {
	return []blockstudio.Definition{
		{
			Key:       "hero",
			Label:     "Hero",
			Summary:   "Opening statement, image, and primary call to action.",
			Kind:      "homepage",
			DefaultOn: true,
			Icon:      "layout",
			Fields: []blockstudio.FieldDefinition{
				{Name: "headline", Label: "Headline", Kind: blockstudio.FieldText},
				{Name: "subhead", Label: "Subhead", Kind: blockstudio.FieldTextarea},
				{Name: "ctaLabel", Label: "CTA label", Kind: blockstudio.FieldText},
				{Name: "ctaUrl", Label: "CTA URL", Kind: blockstudio.FieldURL},
			},
		},
		{
			Key:       "products",
			Label:     "Available work",
			Summary:   "Current pieces that can be purchased or requested.",
			Kind:      "commerce",
			DefaultOn: true,
			Icon:      "grid",
		},
		{
			Key:       "gallery",
			Label:     "Gallery archive",
			Summary:   "Featured archive works, studies, and older pieces.",
			Kind:      "content",
			DefaultOn: true,
			Icon:      "image",
		},
		{
			Key:       "blog",
			Label:     "Studio notes",
			Summary:   "Recent writing from the blog surface.",
			Kind:      "content",
			DefaultOn: true,
			Icon:      "text",
		},
	}
}
