package studio

import (
	"strings"
	"testing"

	"m31labs.dev/gosx"
)

func TestRenderBackendContactDetailRendersReadOnlyPanels(t *testing.T) {
	html := gosx.RenderHTML(RenderBackendContactDetail(BackendContactDetailProps{
		Kicker:  "CMS",
		Title:   "Contact",
		Summary: "Ari <Blue>",
		Message: BackendContactMessage{
			Name:        "Ari <Blue>",
			Email:       "ari@example.com",
			Message:     "Can I reserve the bowl?",
			StatusLabel: "New",
			StatusClass: "status status--new",
		},
		Replies: []BackendContactReply{{
			Subject:   "Re: Bowl",
			Message:   "Yes, it is available.",
			SentLabel: "Sent",
			Created:   BackendContactTime{Label: "Jun 28, 2026 4:45 PM", Machine: "2026-06-28T16:45:00Z"},
		}},
		Submission: BackendContactSubmission{
			Created:   BackendContactTime{Label: "Jun 27, 2026 8:00 AM", Machine: "2026-06-27T08:00:00Z"},
			Updated:   BackendContactTime{Label: "Jun 28, 2026 4:45 PM", Machine: "2026-06-28T16:45:00Z"},
			IPAddress: "127.0.0.1",
			UserAgent: "Go test",
		},
	}))

	for _, fragment := range []string{
		`<div class="admin-page" data-gosx-studio-backend-contact-detail-renderer="gosx-studio">`,
		`<section class="admin-heading"><p class="kicker">CMS</p><h1>Contact</h1><p>Ari &lt;Blue&gt;</p></section>`,
		`<div class="panel"><div class="panel__header"><h2>Message</h2><span class="status status--new">New</span></div><div class="message-list"><article><strong>Ari &lt;Blue&gt;</strong><span><a href="mailto:ari@example.com">ari@example.com</a></span><p>Can I reserve the bowl?</p></article></div></div>`,
		`<section class="panel"><div class="panel__header"><h2>Follow-up history</h2></div><ul class="field-list field-list--stacked"><li><strong>Re: Bowl</strong><time datetime="2026-06-28T16:45:00Z" data-viewer-time="datetime">Jun 28, 2026 4:45 PM</time><span>Sent</span><p>Yes, it is available.</p></li></ul></section>`,
		`<section class="panel"><div class="panel__header"><h2>Submission</h2></div><dl class="spec-list"><div><dt>Created</dt><dd><time datetime="2026-06-27T08:00:00Z" data-viewer-time="datetime">Jun 27, 2026 8:00 AM</time></dd></div><div><dt>Updated</dt><dd><time datetime="2026-06-28T16:45:00Z" data-viewer-time="datetime">Jun 28, 2026 4:45 PM</time></dd></div><div><dt>IP</dt><dd>127.0.0.1</dd></div><div><dt>User agent</dt><dd>Go test</dd></div></dl></section>`,
	} {
		if !strings.Contains(html, fragment) {
			t.Fatalf("contact detail html missing %q:\n%s", fragment, html)
		}
	}
}

func TestRenderBackendContactDetailSplitNodes(t *testing.T) {
	heading := gosx.RenderHTML(RenderBackendContactDetailHeading(BackendContactDetailProps{
		Kicker:  "CMS",
		Title:   "Contact",
		Summary: "Ari",
	}))
	message := gosx.RenderHTML(RenderBackendContactMessage(BackendContactMessage{}))
	history := gosx.RenderHTML(RenderBackendContactReplyHistory(nil))
	submission := gosx.RenderHTML(RenderBackendContactSubmission(BackendContactSubmission{}))
	content := gosx.RenderHTML(RenderBackendContactDetailContent(BackendContactDetailProps{}))

	if !strings.Contains(heading, `<section class="admin-heading">`) || strings.Contains(heading, `message-list`) {
		t.Fatalf("heading node should render only contact heading:\n%s", heading)
	}
	if !strings.Contains(message, `<div class="panel"><div class="panel__header"><h2>Message</h2><span class="status"></span>`) {
		t.Fatalf("message node should tolerate missing status class:\n%s", message)
	}
	if strings.Contains(history, `Follow-up history`) {
		t.Fatalf("empty reply history should render no section:\n%s", history)
	}
	if !strings.Contains(submission, `<h2>Submission</h2>`) {
		t.Fatalf("submission node should render submission panel:\n%s", submission)
	}
	if strings.Contains(content, `data-gosx-studio-backend-contact-detail-renderer="gosx-studio"`) {
		t.Fatalf("content renderer should not render its own page wrapper:\n%s", content)
	}
}
