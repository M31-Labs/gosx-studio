package sitehost

import (
	"strconv"

	"m31labs.dev/gosx-admin/blockstudio"
	cmsstore "m31labs.dev/gosx-studio/cms/store"
	"m31labs.dev/gosx-studio/cms/content"
)

// seed.go gives a fresh install real content instead of an empty database.
//
// The audit found that Studio ships no page blueprints, no component
// templates, and no default composition library, so a new operator lands on a
// headers-only table with nothing to click. A default host that boots into a
// blank site reproduces that problem, so this one boots into a small, real,
// editable site the operator can rename.

func text(value string) blockstudio.Value {
	return blockstudio.Value{Kind: blockstudio.FieldText, String: value}
}

func block(order int, key string, values blockstudio.Values) blockstudio.BlockInstance {
	return blockstudio.BlockInstance{
		ID:      key + "-" + strconv.Itoa(order),
		Key:     key,
		Enabled: true,
		Order:   order,
		Values:  values,
	}
}

func document(blocks ...blockstudio.BlockInstance) blockstudio.Document {
	return blockstudio.Document{Version: 1, Kind: "body", Blocks: blocks}
}

// StarterPage describes one page in the starter site.
type StarterPage struct {
	Slug        string
	Title       string
	Description string
	Body        blockstudio.Document
	Publish     bool
}

// StarterSettings is the site-wide starter configuration.
func StarterSettings(title, description, baseURL string) cmsstore.SiteSettingsInput {
	if title == "" {
		title = "My site"
	}
	if description == "" {
		description = "A new site built with GoSX Studio."
	}
	return cmsstore.SiteSettingsInput{
		Title:       title,
		Description: description,
		BaseURL:     baseURL,
		Locale:      "en",
	}
}

// StarterPages is the content a fresh install begins with. Every page is
// published, so the public site works the moment the server starts.
func StarterPages(siteTitle string) []StarterPage {
	if siteTitle == "" {
		siteTitle = "My site"
	}
	return []StarterPage{
		{
			Slug:        "home",
			Title:       siteTitle,
			Description: "Welcome to " + siteTitle + ".",
			Publish:     true,
			Body: document(
				block(0, content.BlockHeading, blockstudio.Values{
					"text":  text("Welcome to " + siteTitle),
					"level": text("2"),
				}),
				block(1, content.BlockParagraph, blockstudio.Values{
					"text": text("This is your home page. Open the admin area to change this text, add pages, and publish when you are ready."),
				}),
				block(2, content.BlockButton, blockstudio.Values{
					"label": text("See what we do"),
					"url":   text("/about"),
				}),
			),
		},
		{
			Slug:        "about",
			Title:       "About",
			Description: "What " + siteTitle + " does and who it is for.",
			Publish:     true,
			Body: document(
				block(0, content.BlockHeading, blockstudio.Values{
					"text":  text("About us"),
					"level": text("2"),
				}),
				block(1, content.BlockParagraph, blockstudio.Values{
					"text": text("Tell people who you are, what you make, and why it matters to them. Two or three short paragraphs is plenty."),
				}),
				block(2, content.BlockQuote, blockstudio.Values{
					"text": text("Replace this with something a customer said about you."),
				}),
			),
		},
		{
			Slug:        "contact",
			Title:       "Contact",
			Description: "How to reach " + siteTitle + ".",
			Publish:     true,
			Body: document(
				block(0, content.BlockHeading, blockstudio.Values{
					"text":  text("Get in touch"),
					"level": text("2"),
				}),
				block(1, content.BlockParagraph, blockstudio.Values{
					"text": text("Add your email address, phone number, or opening hours here so people can reach you."),
				}),
			),
		},
	}
}

// SeedStarterSite writes the starter site into an empty store. It is a no-op
// when the store already holds pages, so restarting never overwrites work.
func SeedStarterSite(store LifecycleContentStore, title, description, baseURL string) error {
	pages, err := store.ListPages(cmsstore.PageFilter{})
	if err != nil {
		return err
	}
	if len(pages) > 0 {
		return nil
	}

	if _, err := store.SaveSiteSettings(StarterSettings(title, description, baseURL)); err != nil {
		return err
	}
	if _, _, err := store.PublishSiteSettings(); err != nil {
		return err
	}

	for _, starter := range StarterPages(title) {
		page, err := store.CreatePage(cmsstore.PageInput{
			Slug:        starter.Slug,
			Title:       starter.Title,
			Description: starter.Description,
			Body:        starter.Body,
		})
		if err != nil {
			return err
		}
		if !starter.Publish {
			continue
		}
		if _, _, err := store.PublishPage(page.ID); err != nil {
			return err
		}
	}
	return nil
}
