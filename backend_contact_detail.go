package studio

import "m31labs.dev/gosx"

type BackendContactDetailProps struct {
	Kicker     string
	Title      string
	Summary    string
	Class      string
	Message    BackendContactMessage
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
