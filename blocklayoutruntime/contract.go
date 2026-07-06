package blocklayoutruntime

const (
	// RowAttr marks a block-layout row and stores the block key.
	RowAttr = "data-block-studio-block"
	// OrderAttr marks the hidden order input updated during renumbering.
	OrderAttr = "data-block-studio-order"
	// IndexAttr stores the zero-based row index after renumbering.
	IndexAttr = "data-block-studio-index"
	// HandleAttr marks the drag handle used by pointer-event reordering.
	HandleAttr = "data-block-studio-handle"
	// MoveAttr marks up/down move buttons and stores their direction.
	MoveAttr = "data-block-studio-move"
	// AddBlockAttr marks section-library buttons and stores the target block key.
	AddBlockAttr = "data-editor-add-block"
	// VisibleAttr marks the block visibility checkbox.
	VisibleAttr = "data-editor-block-visible"
	// PillAttr marks the visible status pill updated with visibility state.
	PillAttr = "data-editor-block-pill"
	// SelectedClass is the row class toggled by selection.
	SelectedClass = "is-selected"
	// ReorderEvent is dispatched when the block list order is renumbered.
	ReorderEvent = "blockstudio:reorder"
	// SelectEvent is dispatched when a block row is selected.
	SelectEvent = "blockstudio:select"
)
