package studio

import (
	"fmt"
	"strings"
	"time"

	"m31labs.dev/gosx"
	"m31labs.dev/gosx-studio/cms/lifecycle"
)

type PreviewShareOptions struct {
	Class       string
	Kicker      string
	Title       string
	Detail      string
	EmptyTitle  string
	EmptyDetail string
	InputLabel  string
	CopyLabel   string
	OpenLabel   string
	Now         time.Time
}

func RenderPreviewSharePanel(link lifecycle.PreviewLink, href string, options PreviewShareOptions) gosx.Node {
	className := firstNonEmpty(options.Class, "gosx-studio-preview-share")
	title := firstNonEmpty(options.Title, "Share preview")
	detail := firstNonEmpty(options.Detail, "Send a signed draft preview without giving editor access.")
	if strings.TrimSpace(href) == "" {
		return gosx.El("section", gosx.Attrs(
			gosx.Attr("class", className+" "+className+"--disabled"),
			gosx.Attr("data-studio-preview-share", "true"),
			gosx.Attr("data-studio-preview-share-state", "disabled"),
		),
			renderPreviewShareHead(className, options.Kicker, title, "Unavailable"),
			gosx.El("article", gosx.Attrs(gosx.Attr("class", className+"__empty")),
				gosx.El("strong", nil, gosx.Text(firstNonEmpty(options.EmptyTitle, "Preview sharing unavailable"))),
				gosx.El("p", nil, gosx.Text(firstNonEmpty(options.EmptyDetail, "Configure a preview signing secret before sharing drafts outside the editor."))),
			),
		)
	}
	meta := previewShareMeta(link, options.Now)
	return gosx.El("section", gosx.Attrs(
		gosx.Attr("class", className),
		gosx.Attr("data-studio-preview-share", "true"),
		gosx.Attr("data-studio-preview-share-state", "ready"),
	),
		renderPreviewShareHead(className, options.Kicker, title, meta),
		gosx.El("p", gosx.Attrs(gosx.Attr("class", className+"__detail")), gosx.Text(detail)),
		gosx.El("label", gosx.Attrs(gosx.Attr("class", className+"__field")),
			gosx.El("span", nil, gosx.Text(firstNonEmpty(options.InputLabel, "Preview URL"))),
			gosx.El("input", gosx.Attrs(
				gosx.Attr("type", "url"),
				gosx.Attr("readonly", "readonly"),
				gosx.Attr("value", href),
				gosx.Attr("data-studio-preview-url", "true"),
				gosx.Attr("aria-label", firstNonEmpty(options.InputLabel, "Preview URL")),
			)),
		),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", className+"__actions")),
			gosx.El("button", gosx.Attrs(
				gosx.Attr("type", "button"),
				gosx.Attr("data-studio-copy-target", "[data-studio-preview-url]"),
			), gosx.Text(firstNonEmpty(options.CopyLabel, "Copy link"))),
			gosx.El("a", gosx.Attrs(
				gosx.Attr("href", href),
				gosx.Attr("target", "_blank"),
				gosx.Attr("rel", "noreferrer"),
			), gosx.Text(firstNonEmpty(options.OpenLabel, "Open"))),
		),
		gosx.El("dl", gosx.Attrs(gosx.Attr("class", className+"__meta")),
			previewMetaPair("Resource", link.ResourceKind+"/"+link.ResourceID),
			previewMetaPair("Audience", firstNonEmpty(link.Audience, "reviewer")),
			previewMetaPair("Expires", previewTimeLabel(link.Expires, options.Now)),
		),
	)
}

func renderPreviewShareHead(className, kicker, title, status string) gosx.Node {
	return gosx.El("div", gosx.Attrs(gosx.Attr("class", className+"__head")),
		gosx.El("div", nil,
			gosx.El("p", gosx.Attrs(gosx.Attr("class", className+"__kicker")), gosx.Text(firstNonEmpty(kicker, "Preview"))),
			gosx.El("h2", nil, gosx.Text(title)),
		),
		gosx.El("output", nil, gosx.Text(status)),
	)
}

func previewMetaPair(label, value string) gosx.Node {
	return gosx.Fragment(
		gosx.El("dt", nil, gosx.Text(label)),
		gosx.El("dd", nil, gosx.Text(value)),
	)
}

func previewShareMeta(link lifecycle.PreviewLink, now time.Time) string {
	if link.Expires.IsZero() {
		return "Ready"
	}
	if now.IsZero() {
		now = time.Now()
	}
	remaining := link.Expires.Sub(now)
	if remaining <= 0 {
		return "Expired"
	}
	if remaining < time.Hour {
		return fmt.Sprintf("%dm left", int(remaining.Minutes()))
	}
	if remaining < 48*time.Hour {
		return fmt.Sprintf("%dh left", int(remaining.Hours()))
	}
	return fmt.Sprintf("%dd left", int(remaining.Hours()/24))
}

func previewTimeLabel(value, now time.Time) string {
	if value.IsZero() {
		return "Not set"
	}
	if now.IsZero() {
		now = time.Now()
	}
	date := value.UTC().Format("Jan 2, 2006 15:04 UTC")
	remaining := previewShareMeta(lifecycle.PreviewLink{Expires: value}, now)
	if remaining == "Expired" {
		return date + " (expired)"
	}
	return date + " (" + remaining + ")"
}
