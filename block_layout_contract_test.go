package studio

import (
	"testing"

	"m31labs.dev/gosx-studio/blocklayoutruntime"
)

func TestBlockLayoutContractReexportsRuntimeConstants(t *testing.T) {
	checks := map[string][2]string{
		"BlockLayoutRowAttr":       {BlockLayoutRowAttr, blocklayoutruntime.RowAttr},
		"BlockLayoutOrderAttr":     {BlockLayoutOrderAttr, blocklayoutruntime.OrderAttr},
		"BlockLayoutIndexAttr":     {BlockLayoutIndexAttr, blocklayoutruntime.IndexAttr},
		"BlockLayoutHandleAttr":    {BlockLayoutHandleAttr, blocklayoutruntime.HandleAttr},
		"BlockLayoutMoveAttr":      {BlockLayoutMoveAttr, blocklayoutruntime.MoveAttr},
		"BlockLayoutAddBlockAttr":  {BlockLayoutAddBlockAttr, blocklayoutruntime.AddBlockAttr},
		"BlockLayoutVisibleAttr":   {BlockLayoutVisibleAttr, blocklayoutruntime.VisibleAttr},
		"BlockLayoutPillAttr":      {BlockLayoutPillAttr, blocklayoutruntime.PillAttr},
		"BlockLayoutSelectedClass": {BlockLayoutSelectedClass, blocklayoutruntime.SelectedClass},
		"BlockLayoutReorderEvent":  {BlockLayoutReorderEvent, blocklayoutruntime.ReorderEvent},
		"BlockLayoutSelectEvent":   {BlockLayoutSelectEvent, blocklayoutruntime.SelectEvent},
	}
	for name, pair := range checks {
		if pair[0] != pair[1] {
			t.Fatalf("%s = %q, want runtime constant %q", name, pair[0], pair[1])
		}
	}
}
