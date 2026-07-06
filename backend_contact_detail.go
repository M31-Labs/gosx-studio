package studio

import "m31labs.dev/gosx"

type BackendContactDetailProps struct {
	Kicker     string
	Title      string
	Summary    string
	Class      string
	Message    BackendContactMessage
	Actions    BackendContactActions
	Replies    []BackendContactReply
	Submission BackendContactSubmission
}

type BackendContactMessage struct {
	Name        string
	Email       string
	Message     string
	StatusLabel string
	StatusClass string
}

type BackendContactReply struct {
	Subject   string
	Message   string
	SentLabel string
	Created   BackendContactTime
}

type BackendContactSubmission struct {
	Created   BackendContactTime
	Updated   BackendContactTime
	IPAddress string
	UserAgent string
}

type BackendContactActions struct {
	ContactID          string
	CSRFToken          string
	BackHref           string
	Paths              BackendContactActionPaths
	Status             string
	ReplySubject       string
	ReplyMessage       string
	ReplyEmailReady    bool
	ReplyEmailDisabled bool
	ReplyAction        BackendContactReplyActionState
}

type BackendContactActionPaths struct {
	SaveStatus string
	SaveReply  string
}

type BackendContactReplyActionState struct {
	Submitted   bool
	Message     string
	FieldErrors map[string]string
}

type BackendContactTime struct {
	Label   string
	Machine string
}

func RenderBackendContactDetail(props BackendContactDetailProps) gosx.Node {
	className := props.Class
	if className == "" {
		className = "admin-page"
	}
	return gosx.El("div", gosx.Attrs(
		gosx.Attr("class", className),
		gosx.Attr("data-gosx-studio-backend-contact-detail-renderer", "gosx-studio"),
	),
		RenderBackendContactDetailContent(props),
	)
}

func RenderBackendContactDetailContent(props BackendContactDetailProps) gosx.Node {
	return gosx.Fragment(
		RenderBackendContactDetailHeading(props),
		RenderBackendContactMessage(props.Message),
		RenderBackendContactReplyHistory(props.Replies),
		RenderBackendContactSubmission(props.Submission),
	)
}

func RenderBackendContactDetailPage(props BackendContactDetailProps) gosx.Node {
	className := props.Class
	if className == "" {
		className = "admin-page"
	}
	return gosx.El("div", gosx.Attrs(
		gosx.Attr("class", className),
		gosx.Attr("data-gosx-studio-backend-contact-detail-renderer", "gosx-studio"),
	),
		RenderBackendContactDetailPageContent(props),
	)
}

func RenderBackendContactDetailPageContent(props BackendContactDetailProps) gosx.Node {
	return gosx.Fragment(
		RenderBackendContactDetailHeading(props),
		gosx.El("section", gosx.Attrs(gosx.Attr("class", "admin-grid")),
			RenderBackendContactMessage(props.Message),
			RenderBackendContactStatusForm(props.Actions),
			RenderBackendContactReplyForm(props.Actions),
		),
		RenderBackendContactReplyHistory(props.Replies),
		RenderBackendContactSubmission(props.Submission),
	)
}

func RenderBackendContactDetailHeading(props BackendContactDetailProps) gosx.Node {
	return gosx.El("section", gosx.Attrs(gosx.Attr("class", "admin-heading")),
		gosx.El("p", gosx.Attrs(gosx.Attr("class", "kicker")), gosx.Text(props.Kicker)),
		gosx.El("h1", nil, gosx.Text(props.Title)),
		gosx.El("p", nil, gosx.Text(props.Summary)),
	)
}

func RenderBackendContactMessage(message BackendContactMessage) gosx.Node {
	statusClass := message.StatusClass
	if statusClass == "" {
		statusClass = "status"
	}
	return gosx.El("div", gosx.Attrs(gosx.Attr("class", "panel")),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "panel__header")),
			gosx.El("h2", nil, gosx.Text("Message")),
			gosx.El("span", gosx.Attrs(gosx.Attr("class", statusClass)), gosx.Text(message.StatusLabel)),
		),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "message-list")),
			gosx.El("article", nil,
				gosx.El("strong", nil, gosx.Text(message.Name)),
				gosx.El("span", nil,
					gosx.El("a", gosx.Attrs(gosx.Attr("href", "mailto:"+message.Email)), gosx.Text(message.Email)),
				),
				gosx.El("p", nil, gosx.Text(message.Message)),
			),
		),
	)
}

func RenderBackendContactStatusForm(actions BackendContactActions) gosx.Node {
	backHref := actions.BackHref
	if backHref == "" {
		backHref = "/admin/contacts"
	}
	return gosx.El("form", gosx.Attrs(
		gosx.Attr("class", "panel admin-form admin-form--single"),
		gosx.Attr("method", "post"),
		gosx.Attr("action", actions.Paths.SaveStatus),
	),
		renderBackendContactHiddenInputs(actions.CSRFToken, actions.ContactID),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "field-row")),
			gosx.El("label", gosx.Attrs(gosx.Attr("for", "status")), gosx.Text("Status")),
			gosx.El("select", gosx.Attrs(gosx.Attr("id", "status"), gosx.Attr("name", "status")),
				renderBackendContactStatusOption(actions.Status, "new", "New"),
				renderBackendContactStatusOption(actions.Status, "responded", "Responded"),
				renderBackendContactStatusOption(actions.Status, "archived", "Archived"),
			),
		),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "button-row")),
			gosx.El("button", gosx.Attrs(gosx.Attr("class", "button button--primary"), gosx.Attr("type", "submit")), gosx.Text("Save status")),
			gosx.El("a", gosx.Attrs(gosx.Attr("class", "button button--secondary"), gosx.Attr("href", backHref), gosx.Attr("data-gosx-link", "true")), gosx.Text("Back to contacts")),
		),
	)
}

func RenderBackendContactReplyForm(actions BackendContactActions) gosx.Node {
	headerChildren := []gosx.Node{gosx.El("h2", nil, gosx.Text("Follow up"))}
	if actions.ReplyEmailReady {
		headerChildren = append(headerChildren, gosx.El("span", gosx.Attrs(gosx.Attr("class", "status status--ready")), gosx.Text("Email ready")))
	}
	if actions.ReplyEmailDisabled {
		headerChildren = append(headerChildren, gosx.El("span", gosx.Attrs(gosx.Attr("class", "status status--request")), gosx.Text("Record only")))
	}
	children := []gosx.Node{
		renderBackendContactHiddenInputs(actions.CSRFToken, actions.ContactID),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "panel__header")), gosx.Fragment(headerChildren...)),
	}
	if actions.ReplyAction.Submitted {
		children = append(children, gosx.El("p", gosx.Attrs(gosx.Attr("class", "form-status form-status--error")), gosx.Text(actions.ReplyAction.Message)))
	}
	children = append(children,
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "field-row")),
			gosx.El("label", gosx.Attrs(gosx.Attr("for", "replySubject")), gosx.Text("Subject")),
			gosx.El("input", gosx.Attrs(gosx.Attr("id", "replySubject"), gosx.Attr("name", "replySubject"), gosx.Attr("value", actions.ReplySubject))),
		),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "field-row")),
			gosx.El("label", gosx.Attrs(gosx.Attr("for", "replyMessage")), gosx.Text("Message")),
			gosx.El("textarea", gosx.Attrs(gosx.Attr("id", "replyMessage"), gosx.Attr("name", "replyMessage"), gosx.Attr("rows", "7")), gosx.Text(actions.ReplyMessage)),
			gosx.El("p", gosx.Attrs(gosx.Attr("class", "form-error")), gosx.Text(actions.ReplyAction.FieldErrors["replyMessage"])),
		),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "check-row")),
			gosx.El("label", nil,
				gosx.El("input", gosx.Attrs(gosx.Attr("type", "checkbox"), gosx.Attr("name", "sendEmail"), gosx.Attr("checked", actions.ReplyEmailReady))),
				gosx.Text(" Send email"),
			),
		),
		gosx.El("button", gosx.Attrs(gosx.Attr("class", "button button--primary"), gosx.Attr("type", "submit")), gosx.Text("Save follow-up")),
	)
	return gosx.El("form", gosx.Attrs(
		gosx.Attr("class", "panel admin-form admin-form--single"),
		gosx.Attr("method", "post"),
		gosx.Attr("action", actions.Paths.SaveReply),
	), gosx.Fragment(children...))
}

func RenderBackendContactReplyHistory(replies []BackendContactReply) gosx.Node {
	if len(replies) == 0 {
		return gosx.Fragment()
	}
	items := make([]gosx.Node, 0, len(replies))
	for _, reply := range replies {
		items = append(items, gosx.El("li", nil,
			gosx.El("strong", nil, gosx.Text(reply.Subject)),
			renderBackendContactTime(reply.Created),
			gosx.El("span", nil, gosx.Text(reply.SentLabel)),
			gosx.El("p", nil, gosx.Text(reply.Message)),
		))
	}
	return gosx.El("section", gosx.Attrs(gosx.Attr("class", "panel")),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "panel__header")),
			gosx.El("h2", nil, gosx.Text("Follow-up history")),
		),
		gosx.El("ul", gosx.Attrs(gosx.Attr("class", "field-list field-list--stacked")), gosx.Fragment(items...)),
	)
}

func RenderBackendContactSubmission(submission BackendContactSubmission) gosx.Node {
	return gosx.El("section", gosx.Attrs(gosx.Attr("class", "panel")),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "panel__header")),
			gosx.El("h2", nil, gosx.Text("Submission")),
		),
		gosx.El("dl", gosx.Attrs(gosx.Attr("class", "spec-list")),
			renderBackendContactSpecTime("Created", submission.Created),
			renderBackendContactSpecTime("Updated", submission.Updated),
			renderBackendContactSpecText("IP", submission.IPAddress),
			renderBackendContactSpecText("User agent", submission.UserAgent),
		),
	)
}

func renderBackendContactHiddenInputs(csrfToken, contactID string) gosx.Node {
	return gosx.Fragment(
		gosx.El("input", gosx.Attrs(gosx.Attr("type", "hidden"), gosx.Attr("name", "csrf_token"), gosx.Attr("value", csrfToken))),
		gosx.El("input", gosx.Attrs(gosx.Attr("type", "hidden"), gosx.Attr("name", "id"), gosx.Attr("value", contactID))),
	)
}

func renderBackendContactStatusOption(status, value, label string) gosx.Node {
	return gosx.El("option", gosx.Attrs(
		gosx.Attr("value", value),
		gosx.Attr("selected", status == value),
	), gosx.Text(label))
}

func renderBackendContactSpecTime(label string, value BackendContactTime) gosx.Node {
	return gosx.El("div", nil,
		gosx.El("dt", nil, gosx.Text(label)),
		gosx.El("dd", nil, renderBackendContactTime(value)),
	)
}

func renderBackendContactSpecText(label, value string) gosx.Node {
	return gosx.El("div", nil,
		gosx.El("dt", nil, gosx.Text(label)),
		gosx.El("dd", nil, gosx.Text(value)),
	)
}

func renderBackendContactTime(value BackendContactTime) gosx.Node {
	return gosx.El("time", gosx.Attrs(
		gosx.Attr("datetime", value.Machine),
		gosx.Attr("data-viewer-time", "datetime"),
	), gosx.Text(value.Label))
}
