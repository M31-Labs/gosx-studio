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

func TestRenderBackendContactStatusAndReplyForms(t *testing.T) {
	actions := BackendContactActions{
		ContactID:          "contact_123",
		CSRFToken:          "csrf-token",
		Status:             "responded",
		ReplySubject:       "Re: Bowl <available>",
		ReplyMessage:       "Yes, it is available.",
		ReplyEmailReady:    true,
		ReplyEmailDisabled: false,
		Paths: BackendContactActionPaths{
			SaveStatus: "/save-status",
			SaveReply:  "/save-reply",
		},
		ReplyAction: BackendContactReplyActionState{
			Submitted: true,
			Message:   "Reply message is required.",
			FieldErrors: map[string]string{
				"replyMessage": "Write a reply before saving.",
			},
		},
	}

	statusHTML := gosx.RenderHTML(RenderBackendContactStatusForm(actions))
	replyHTML := gosx.RenderHTML(RenderBackendContactReplyForm(actions))

	for _, fragment := range []string{
		`<form class="panel admin-form admin-form--single" method="post" action="/save-status"><input type="hidden" name="csrf_token" value="csrf-token" /><input type="hidden" name="id" value="contact_123" />`,
		`<label for="status">Status</label><select id="status" name="status"><option value="new">New</option><option value="responded" selected>Responded</option><option value="archived">Archived</option></select>`,
		`<button class="button button--primary" type="submit">Save status</button><a class="button button--secondary" href="/admin/contacts" data-gosx-link="true">Back to contacts</a>`,
	} {
		if !strings.Contains(statusHTML, fragment) {
			t.Fatalf("status form html missing %q:\n%s", fragment, statusHTML)
		}
	}
	for _, fragment := range []string{
		`<form class="panel admin-form admin-form--single" method="post" action="/save-reply"><input type="hidden" name="csrf_token" value="csrf-token" /><input type="hidden" name="id" value="contact_123" />`,
		`<div class="panel__header"><h2>Follow up</h2><span class="status status--ready">Email ready</span></div>`,
		`<p class="form-status form-status--error">Reply message is required.</p>`,
		`<input id="replySubject" name="replySubject" value="Re: Bowl &lt;available&gt;" />`,
		`<textarea id="replyMessage" name="replyMessage" rows="7">Yes, it is available.</textarea><p class="form-error">Write a reply before saving.</p>`,
		`<div class="check-row"><label><input type="checkbox" name="sendEmail" checked /> Send email</label></div><button class="button button--primary" type="submit">Save follow-up</button>`,
	} {
		if !strings.Contains(replyHTML, fragment) {
			t.Fatalf("reply form html missing %q:\n%s", fragment, replyHTML)
		}
	}
	if strings.Contains(replyHTML, `Record only`) {
		t.Fatalf("email-ready reply form should not show record-only status:\n%s", replyHTML)
	}
}

func TestRenderBackendContactReplyFormRecordOnly(t *testing.T) {
	html := gosx.RenderHTML(RenderBackendContactReplyForm(BackendContactActions{
		ReplyEmailDisabled: true,
		Paths:              BackendContactActionPaths{SaveReply: "/save-reply"},
	}))

	for _, fragment := range []string{
		`<div class="panel__header"><h2>Follow up</h2><span class="status status--request">Record only</span></div>`,
		`<input type="checkbox" name="sendEmail" /> Send email`,
	} {
		if !strings.Contains(html, fragment) {
			t.Fatalf("record-only reply form missing %q:\n%s", fragment, html)
		}
	}
	if strings.Contains(html, `Email ready`) || strings.Contains(html, `form-status form-status--error`) {
		t.Fatalf("record-only reply form should omit email-ready and unsubmitted error:\n%s", html)
	}
}

func TestRenderBackendContactDetailPageOwnsActionForms(t *testing.T) {
	html := gosx.RenderHTML(RenderBackendContactDetailPage(BackendContactDetailProps{
		Kicker:  "CMS",
		Title:   "Contact",
		Summary: "Ari Blue",
		Message: BackendContactMessage{
			Name:        "Ari Blue",
			Email:       "ari@example.com",
			Message:     "Can I reserve the bowl?",
			StatusLabel: "New",
			StatusClass: "status status--new",
		},
		Actions: BackendContactActions{
			ContactID:       "contact_123",
			CSRFToken:       "csrf-token",
			Status:          "new",
			ReplyMessage:    "Draft reply",
			ReplyEmailReady: true,
			Paths: BackendContactActionPaths{
				SaveStatus: "/save-status",
				SaveReply:  "/save-reply",
			},
		},
		Submission: BackendContactSubmission{
			Created: BackendContactTime{Label: "Jun 27, 2026 8:00 AM", Machine: "2026-06-27T08:00:00Z"},
			Updated: BackendContactTime{Label: "Jun 27, 2026 8:00 AM", Machine: "2026-06-27T08:00:00Z"},
		},
	}))

	for _, fragment := range []string{
		`<div class="admin-page" data-gosx-studio-backend-contact-detail-renderer="gosx-studio">`,
		`<section class="admin-heading"><p class="kicker">CMS</p><h1>Contact</h1><p>Ari Blue</p></section>`,
		`<section class="admin-grid"><div class="panel"><div class="panel__header"><h2>Message</h2>`,
		`<form class="panel admin-form admin-form--single" method="post" action="/save-status">`,
		`<form class="panel admin-form admin-form--single" method="post" action="/save-reply">`,
		`<section class="panel"><div class="panel__header"><h2>Submission</h2>`,
	} {
		if !strings.Contains(html, fragment) {
			t.Fatalf("contact detail page html missing %q:\n%s", fragment, html)
		}
	}
	if strings.Contains(html, `Follow-up history`) {
		t.Fatalf("page should omit empty reply history:\n%s", html)
	}
}
