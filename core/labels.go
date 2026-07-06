package core

func PageGroupLabel(group PageGroup) string {
	switch normalizePageGroup(group) {
	case PageGroupCommerce:
		return "Store"
	case PageGroupContent:
		return "Content"
	case PageGroupFlows:
		return "Flows"
	case PageGroupUtility:
		return "Utility"
	default:
		return "Site"
	}
}

func ComponentSourceLabel(source ComponentSource) string {
	switch normalizeComponentSource(source) {
	case ComponentSourceCMS:
		return "CMS"
	case ComponentSourcePlugin:
		return "Plugin"
	case ComponentSourceStudio:
		return "Studio"
	default:
		return "Site"
	}
}

func ControlKindLabel(kind ControlKind) string {
	switch NormalizeControlKind(kind) {
	case ControlRichText:
		return "Rich text"
	case ControlMedia:
		return "Media"
	case ControlChoice:
		return "Choice"
	case ControlToggle:
		return "Toggle"
	case ControlNumber:
		return "Number"
	case ControlLink:
		return "Link"
	case ControlColor:
		return "Color"
	case ControlSource:
		return "Source"
	case ControlFlow:
		return "Flow"
	case ControlScene3D:
		return "Scene 3D"
	default:
		return "Text"
	}
}

func WorkspaceNodeKindLabel(kind WorkspaceNodeKind) string {
	switch normalizeWorkspaceNodeKind(kind) {
	case WorkspaceNodeComponent:
		return "Component"
	case WorkspaceNodeResource:
		return "Resource"
	default:
		return "Page"
	}
}

func WorkspaceLinkKindLabel(kind WorkspaceLinkKind) string {
	switch normalizeWorkspaceLinkKind(kind) {
	case WorkspaceLinkBinds:
		return "Binding"
	default:
		return "Contains"
	}
}

func ReadinessStatusLabel(status ReadinessStatus) string {
	switch normalizeReadinessStatus(status) {
	case ReadinessBlocked:
		return "Needs setup"
	case ReadinessWatch:
		return "Review"
	default:
		return "Ready"
	}
}

func ResourceKindLabel(kind ResourceKind) string {
	switch normalizeResourceKind(kind) {
	case ResourcePages:
		return "Pages"
	case ResourceProducts:
		return "Products"
	case ResourceOrders:
		return "Orders"
	case ResourceContacts:
		return "Contacts"
	case ResourceSettings:
		return "Settings"
	case ResourceRevisions:
		return "Revisions"
	case ResourceLifecycle:
		return "Lifecycle"
	case ResourceFlows:
		return "Flows"
	default:
		return "Media"
	}
}

func CanvasActionKindLabel(kind CanvasActionKind) string {
	switch normalizeCanvasActionKind(kind) {
	case CanvasActionReveal:
		return "Reveal"
	case CanvasActionMoveUp:
		return "Move up"
	case CanvasActionMoveDown:
		return "Move down"
	case CanvasActionInlineText:
		return "Edit text"
	case CanvasActionContent:
		return "Content"
	case CanvasActionStyle:
		return "Style"
	case CanvasActionToggleVisibility:
		return "Hide"
	default:
		return "Action"
	}
}
