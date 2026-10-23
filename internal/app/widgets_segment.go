package app

import "afvi/teach-imgui/internal/ui"

// SubTabItem is one entry of the second-level tab strip of a page.
type SubTabItem struct {
	Label string // already translated
	Color uint32 // rgba of the chip drawn left of the label, 0 = no chip
}

// SubTabPanel is implemented by a page that wants a second-level tab strip under the page tabs.
type SubTabPanel interface {
	SubTabs() (items []SubTabItem, active int) // no items = no strip this frame
	SelectSubTab(i int)                        // called with the clicked index
	SubTabNote() string                        // short dim text right of the tabs, "" = none
}

// SubTabStrip draws the segmented strip and returns the clicked index or -1.
func SubTabStrip(id string, items []SubTabItem, active int, note string) int {
	out := make([]ui.SubTabItem, len(items))
	for i, it := range items {
		out[i] = ui.SubTabItem{Label: it.Label}
		if it.Color != 0 {
			out[i].Color = css(it.Color)
		}
	}
	return ui.SubTabStrip(id, out, active, note)
}
