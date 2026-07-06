package studio

type FlowDesignerProps struct {
	FormID         string
	PublishAction  string
	DefaultFlowKey string
	Flows          []FlowDesignerFlow
}

type FlowDesignerFlow struct {
	Key                string
	Label              string
	Description        string
	Summary            string
	Status             string
	StatusLabel        string
	CardClass          string
	Selected           bool
	Route              string
	HasRoute           bool
	EmbedTarget        string
	HasEmbedTarget     bool
	HandlerRef         string
	HandlerStatusLabel string
	HandlerDetail      string
	HandlerClass       string
	HandlerInputID     string
	HandlerInputName   string
	HandlerSource      string
	RouteInputID       string
	RouteInputName     string
	EmbedInputID       string
	EmbedInputName     string
	Readiness          FlowDesignerReadiness
	ReadinessChecks    []FlowDesignerCheck
	Nodes              []FlowDesignerNode
	StepCount          int
	ActionCount        int
	FieldCount         int
	RequiredFieldCount int
	CanExecute         bool
	Steps              []FlowDesignerStep
	Actions            []FlowDesignerAction
}

type FlowDesignerReadiness struct {
	Status      string
	Label       string
	Summary     string
	CanPublish  bool
	Class       string
	BadgeClass  string
	StatusLabel string
}

type FlowDesignerCheck struct {
	Key         string
	Label       string
	Summary     string
	Status      string
	StatusLabel string
	Class       string
	TabIndex    string
}

type FlowDesignerNode struct {
	Key         string
	Label       string
	Kind        string
	KindLabel   string
	Status      string
	StatusLabel string
	Summary     string
	Class       string
	TabIndex    string
}

type FlowDesignerStep struct {
	Key            string
	Label          string
	BlockCount     int
	HasBlocks      bool
	BodySummary    string
	LabelInputID   string
	LabelInputName string
	LabelSource    string
	BodySource     string
}

type FlowDesignerAction struct {
	Key        string
	Label      string
	HandlerRef string
	FieldCount int
	CanExecute bool
	Fields     []FlowDesignerField
}

type FlowDesignerField struct {
	Name          string
	Label         string
	Kind          string
	Required      bool
	RequiredLabel string
}
