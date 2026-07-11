package core

import "strings"

type Flow struct {
	Key                string
	Label              string
	Description        string
	Summary            string
	Route              string
	EmbedTarget        string
	HasRoute           bool
	HasEmbedTarget     bool
	HandlerRef         string
	CanExecute         bool
	Steps              []FlowStep
	Actions            []FlowAction
	FieldCount         int
	RequiredFieldCount int
}

type FlowStep struct {
	Key        string
	Label      string
	Summary    string
	BlockCount int
	HasBlocks  bool
}

type FlowAction struct {
	Key        string
	Label      string
	HandlerRef string
	CanExecute bool
	Fields     []FlowField
}

type FlowField struct {
	Name     string
	Label    string
	Kind     ControlKind
	Required bool
}

type FlowReadinessCheck struct {
	Key     string
	Label   string
	Summary string
	Status  ReadinessStatus
}

type FlowNode struct {
	Key     string
	Label   string
	Summary string
	Kind    string
	Status  ReadinessStatus
}

func (flow Flow) Normalize() Flow {
	flow.Key = strings.TrimSpace(flow.Key)
	flow.Label = strings.TrimSpace(flow.Label)
	flow.Description = strings.TrimSpace(flow.Description)
	flow.Summary = strings.TrimSpace(flow.Summary)
	flow.Route = strings.TrimSpace(flow.Route)
	flow.EmbedTarget = strings.TrimSpace(flow.EmbedTarget)
	flow.HandlerRef = strings.TrimSpace(flow.HandlerRef)
	flow.Steps = normalizeFlowSteps(flow.Steps)
	flow.Actions = normalizeFlowActions(flow.Actions)
	if flow.FieldCount == 0 {
		for _, action := range flow.Actions {
			flow.FieldCount += len(action.Fields)
		}
	}
	if flow.RequiredFieldCount == 0 {
		for _, action := range flow.Actions {
			for _, field := range action.Fields {
				if field.Required {
					flow.RequiredFieldCount++
				}
			}
		}
	}
	return flow
}

func (flow Flow) ReadinessChecks() []FlowReadinessCheck {
	flow = flow.Normalize()
	return []FlowReadinessCheck{
		{
			Key:     "handler",
			Label:   "Submission action",
			Summary: flowHandlerReadinessSummary(flow),
			Status:  flowHandlerReadinessStatus(flow),
		},
		{
			Key:     "placement",
			Label:   "Public placement",
			Summary: flowPlacementReadinessSummary(flow),
			Status:  flowPlacementReadinessStatus(flow),
		},
		{
			Key:     "structure",
			Label:   "Steps and actions",
			Summary: flowStructureReadinessSummary(flow),
			Status:  flowStructureReadinessStatus(flow),
		},
		{
			Key:     "fields",
			Label:   "Editor fields",
			Summary: flowFieldsReadinessSummary(flow),
			Status:  flowFieldsReadinessStatus(flow),
		},
	}
}

func (flow Flow) ReadinessStatus() ReadinessStatus {
	status := ReadinessReady
	for _, check := range flow.ReadinessChecks() {
		switch normalizeReadinessStatus(check.Status) {
		case ReadinessBlocked:
			return ReadinessBlocked
		case ReadinessWatch:
			status = ReadinessWatch
		}
	}
	return status
}

func (flow Flow) ReadinessLabel() string {
	switch flow.ReadinessStatus() {
	case ReadinessBlocked:
		return "Needs setup"
	case ReadinessWatch:
		return "Review before publish"
	default:
		return "Ready to publish"
	}
}

func (flow Flow) Nodes() []FlowNode {
	flow = flow.Normalize()
	nodes := []FlowNode{
		{
			Key:     "placement",
			Label:   "Placement",
			Summary: flowPlacementReadinessSummary(flow),
			Kind:    "placement",
			Status:  flowPlacementReadinessStatus(flow),
		},
	}
	for _, step := range flow.Steps {
		status := ReadinessReady
		summary := strings.TrimSpace(step.Summary)
		if summary == "" {
			summary = flowStepSummary(step)
		}
		if !step.HasBlocks && step.BlockCount == 0 {
			status = ReadinessWatch
		}
		nodes = append(nodes, FlowNode{
			Key:     "step:" + WorkspaceToken(step.Key),
			Label:   step.Label,
			Summary: summary,
			Kind:    "step",
			Status:  status,
		})
	}
	for _, action := range flow.Actions {
		status := ReadinessReady
		summary := flowActionSummary(action)
		if !action.CanExecute || strings.TrimSpace(action.HandlerRef) == "" {
			status = ReadinessBlocked
		}
		nodes = append(nodes, FlowNode{
			Key:     "action:" + WorkspaceToken(action.Key),
			Label:   action.Label,
			Summary: summary,
			Kind:    "action",
			Status:  status,
		})
	}
	nodes = append(nodes, FlowNode{
		Key:     "publish",
		Label:   flow.ReadinessLabel(),
		Summary: "Publish only after the visible checks match the operator intent.",
		Kind:    "publish",
		Status:  flow.ReadinessStatus(),
	})
	return normalizeFlowNodes(nodes)
}

func normalizeReadinessStatus(status ReadinessStatus) ReadinessStatus {
	switch ReadinessStatus(strings.TrimSpace(string(status))) {
	case ReadinessWatch:
		return ReadinessWatch
	case ReadinessBlocked:
		return ReadinessBlocked
	default:
		return ReadinessReady
	}
}

func normalizeFlowSteps(steps []FlowStep) []FlowStep {
	out := make([]FlowStep, 0, len(steps))
	for _, step := range steps {
		step.Key = strings.TrimSpace(step.Key)
		step.Label = strings.TrimSpace(step.Label)
		step.Summary = strings.TrimSpace(step.Summary)
		if step.Key == "" || step.Label == "" {
			continue
		}
		out = append(out, step)
	}
	return out
}

func normalizeFlowActions(actions []FlowAction) []FlowAction {
	out := make([]FlowAction, 0, len(actions))
	for _, action := range actions {
		action.Key = strings.TrimSpace(action.Key)
		action.Label = strings.TrimSpace(action.Label)
		action.HandlerRef = strings.TrimSpace(action.HandlerRef)
		action.Fields = normalizeFlowFields(action.Fields)
		if action.Key == "" || action.Label == "" {
			continue
		}
		out = append(out, action)
	}
	return out
}

func normalizeFlowFields(fields []FlowField) []FlowField {
	out := make([]FlowField, 0, len(fields))
	for _, field := range fields {
		field.Name = strings.TrimSpace(field.Name)
		field.Label = strings.TrimSpace(field.Label)
		field.Kind = NormalizeControlKind(field.Kind)
		if field.Name == "" || field.Label == "" {
			continue
		}
		out = append(out, field)
	}
	return out
}

func normalizeFlowNodes(nodes []FlowNode) []FlowNode {
	out := make([]FlowNode, 0, len(nodes))
	for _, node := range nodes {
		node.Key = strings.TrimSpace(node.Key)
		node.Label = strings.TrimSpace(node.Label)
		node.Summary = strings.TrimSpace(node.Summary)
		node.Kind = strings.TrimSpace(node.Kind)
		node.Status = normalizeReadinessStatus(node.Status)
		if node.Key == "" || node.Label == "" {
			continue
		}
		out = append(out, node)
	}
	return out
}

func flowHandlerReadinessStatus(flow Flow) ReadinessStatus {
	if flow.HandlerRef == "" {
		return ReadinessBlocked
	}
	if !flow.CanExecute {
		return ReadinessWatch
	}
	return ReadinessReady
}

func flowHandlerReadinessSummary(flow Flow) string {
	if flow.HandlerRef == "" {
		return "Connect the site action that receives this flow."
	}
	if !flow.CanExecute {
		return flow.HandlerRef + " is set; review secondary action handlers."
	}
	return flow.HandlerRef + " receives submissions."
}

func flowPlacementReadinessStatus(flow Flow) ReadinessStatus {
	if flow.HasRoute && flow.Route != "" && flow.HasEmbedTarget && flow.EmbedTarget != "" {
		return ReadinessReady
	}
	if (flow.HasRoute && flow.Route != "") || (flow.HasEmbedTarget && flow.EmbedTarget != "") {
		return ReadinessWatch
	}
	return ReadinessBlocked
}

func flowPlacementReadinessSummary(flow Flow) string {
	hasRoute := flow.HasRoute && flow.Route != ""
	hasEmbed := flow.HasEmbedTarget && flow.EmbedTarget != ""
	switch {
	case hasRoute && hasEmbed:
		return "Visible at " + flow.Route + " and embedded in " + flow.EmbedTarget + "."
	case hasRoute:
		return "Visible at " + flow.Route + "; choose an embed target if it also belongs inside a page."
	case hasEmbed:
		return "Embedded in " + flow.EmbedTarget + "; add a route if it needs a direct page."
	default:
		return "Choose where this flow appears on the site."
	}
}

func flowStructureReadinessStatus(flow Flow) ReadinessStatus {
	if len(flow.Steps) > 0 && len(flow.Actions) > 0 {
		return ReadinessReady
	}
	if len(flow.Steps) > 0 || len(flow.Actions) > 0 {
		return ReadinessWatch
	}
	return ReadinessBlocked
}

func flowStructureReadinessSummary(flow Flow) string {
	return CountLabel(len(flow.Steps), "step", "steps") + " and " + CountLabel(len(flow.Actions), "action", "actions") + "."
}

func flowFieldsReadinessStatus(flow Flow) ReadinessStatus {
	if flow.FieldCount > 0 {
		return ReadinessReady
	}
	return ReadinessWatch
}

// BindingResolved reports whether this flow currently resolves as a
// component's resource Binding target (e.g. Component.Binding ==
// "flow."+flow.Key). It composes with BindingDiagnostic/BindingResolver: a
// host's ResourceBindingAdapter for core.ResourceFlows typically wraps this
// method directly, so a flow with no handler or that cannot yet execute
// blocks readiness/publish the same way an unresolved CMS or media binding
// does, with an actionable reason instead of a silent false.
func (flow Flow) BindingResolved() (bool, string) {
	flow = flow.Normalize()
	if flow.HandlerRef == "" {
		return false, "Connect a handler before this flow can accept submissions."
	}
	if !flow.CanExecute {
		return false, "This flow cannot execute yet; review its steps and actions."
	}
	return true, ""
}

// ValidatePayload runs the same required/format checks a real submission
// would (see cms/flows.ValidateActionPayload) but stays pure/core-level so the
// flow designer's isolated test-execution path (authoring.ApplyTestFlowAction)
// can validate test values before ever calling a host handler.
func (action FlowAction) ValidatePayload(values map[string]string) map[string]string {
	actions := normalizeFlowActions([]FlowAction{action})
	errs := map[string]string{}
	if len(actions) == 0 {
		return errs
	}
	action = actions[0]
	for _, field := range action.Fields {
		value := strings.TrimSpace(values[field.Name])
		if field.Required && value == "" {
			errs[field.Name] = "This field is required."
			continue
		}
		if value != "" && looksLikeEmailFieldName(field.Name) && !strings.Contains(value, "@") {
			errs[field.Name] = "Enter a valid email address."
		}
	}
	return errs
}

func looksLikeEmailFieldName(name string) bool {
	name = strings.ToLower(strings.TrimSpace(name))
	return name == "email" || strings.HasSuffix(name, "email")
}

func flowFieldsReadinessSummary(flow Flow) string {
	if flow.FieldCount == 0 {
		return "No fields are exposed to site operators."
	}
	return CountLabel(flow.FieldCount, "field", "fields") + ", " + CountLabel(flow.RequiredFieldCount, "required", "required") + "."
}

func flowStepSummary(step FlowStep) string {
	if step.BlockCount == 0 {
		return "No body blocks yet."
	}
	return CountLabel(step.BlockCount, "body block", "body blocks") + "."
}

func flowActionSummary(action FlowAction) string {
	fieldCount := len(action.Fields)
	if fieldCount == 0 {
		return "No fields; handler " + FirstNonEmpty(action.HandlerRef, "not connected") + "."
	}
	return CountLabel(fieldCount, "field", "fields") + "; handler " + FirstNonEmpty(action.HandlerRef, "not connected") + "."
}
