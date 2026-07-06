package blocks

import "m31labs.dev/gosx-admin/blockstudio"

const BodyKind = "content.body"

func BodyCatalog() []blockstudio.Definition {
	return []blockstudio.Definition{
		{
			Key:        "paragraph",
			Label:      "Paragraph",
			Summary:    "Body copy with rich text-friendly line breaks.",
			Kind:       BodyKind,
			DefaultOn:  true,
			Repeatable: true,
			Icon:       "text",
			Fields: []blockstudio.FieldDefinition{
				{Name: "text", Label: "Text", Kind: blockstudio.FieldTextarea, Required: true},
			},
		},
		{
			Key:        "heading",
			Label:      "Heading",
			Summary:    "Section heading for long-form content.",
			Kind:       BodyKind,
			Repeatable: true,
			Icon:       "heading",
			Fields: []blockstudio.FieldDefinition{
				{Name: "text", Label: "Text", Kind: blockstudio.FieldText, Required: true},
				{Name: "level", Label: "Level", Kind: blockstudio.FieldSelect, Default: blockstudio.Value{Kind: blockstudio.FieldSelect, String: "2"}, Options: []blockstudio.FieldOption{
					{Value: "2", Label: "H2"},
					{Value: "3", Label: "H3"},
					{Value: "4", Label: "H4"},
				}},
			},
		},
		{
			Key:        "quote",
			Label:      "Quote",
			Summary:    "Pull quote with optional attribution.",
			Kind:       BodyKind,
			Repeatable: true,
			Icon:       "quote",
			Fields: []blockstudio.FieldDefinition{
				{Name: "text", Label: "Quote", Kind: blockstudio.FieldTextarea, Required: true},
			},
		},
		{
			Key:        "image",
			Label:      "Image",
			Summary:    "Single media asset with caption.",
			Kind:       BodyKind,
			Repeatable: true,
			Icon:       "image",
			Fields: []blockstudio.FieldDefinition{
				{Name: "url", Label: "URL", Kind: blockstudio.FieldURL, Required: true},
				{Name: "alt", Label: "Alt text", Kind: blockstudio.FieldText},
			},
		},
		{
			Key:        "gallery",
			Label:      "Gallery",
			Summary:    "Small image set for body content.",
			Kind:       BodyKind,
			Repeatable: true,
			Icon:       "images",
			Fields: []blockstudio.FieldDefinition{
				{Name: "images", Label: "Images", Kind: blockstudio.FieldTextarea, Required: true},
			},
		},
		{
			Key:        "button",
			Label:      "Button",
			Summary:    "Inline call to action.",
			Kind:       BodyKind,
			Repeatable: true,
			Icon:       "mouse-pointer-click",
			Fields: []blockstudio.FieldDefinition{
				{Name: "label", Label: "Label", Kind: blockstudio.FieldText, Required: true},
				{Name: "href", Label: "Href", Kind: blockstudio.FieldURL, Required: true},
			},
		},
		{
			Key:        "product",
			Label:      "Product",
			Summary:    "Product reference placeholder.",
			Kind:       BodyKind,
			Repeatable: true,
			Icon:       "package",
			Fields: []blockstudio.FieldDefinition{
				{Name: "productRef", Label: "Product reference", Kind: blockstudio.FieldText, Required: true},
			},
		},
		{
			Key:        "flow",
			Label:      "Flow placeholder",
			Summary:    "Named dynamic insertion point for app-owned rendering.",
			Kind:       BodyKind,
			Repeatable: true,
			Icon:       "workflow",
			Fields: []blockstudio.FieldDefinition{
				{Name: "flowKey", Label: "Flow key", Kind: blockstudio.FieldText, Required: true},
			},
		},
	}
}
