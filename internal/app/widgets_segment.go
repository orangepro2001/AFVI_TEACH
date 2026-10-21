package app

import (
	"strconv"

	"github.com/AllenDang/cimgui-go/imgui"
)

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

// SubTabStrip draws the strip at the current cursor in the tool-button style: entries raised,
// the active one pressed (Light fill, EdgePressed, label shifted 1px), a colour chip left of
// the centred label and the dim note right of the last entry, clipped to the window. The cursor
// is left below the strip. Returns the clicked index or -1.
func SubTabStrip(id string, items []SubTabItem, active int, note string) int {
	h := imgui.FrameHeight() + 2
	start := imgui.CursorScreenPos()
	right := start.X + imgui.ContentRegionAvail().X
	dl := imgui.WindowDrawList()
	p := CurrentPalette()
	clicked := -1
	x := start.X
	last := start.X
	for i, it := range items {
		w := imgui.CalcTextSize(it.Label).X + sc(24)
		if w < sc(96) {
			w = sc(96)
		}
		imgui.SetCursorScreenPos(imgui.NewVec2(x, start.Y))
		if imgui.InvisibleButton("##"+id+strconv.Itoa(i), imgui.NewVec2(w, h)) {
			clicked = i
		}
		min, max := imgui.ItemRectMin(), imgui.ItemRectMax()
		on := i == active
		if on {
			FillRect(dl, min, max, p.Light)
			DrawEdge(dl, min, max, EdgePressed)
		} else {
			FillRect(dl, min, max, p.Face)
			DrawEdge(dl, min, max, EdgeRaised)
		}
		off := float32(0)
		if on {
			off = 1
		}
		// chip + centred label drawn as one centred group
		side := (max.Y - min.Y) * 0.6
		ts := imgui.CalcTextSize(it.Label)
		group := ts.X
		if it.Color != 0 {
			group += side + sc(6)
		}
		gx := min.X + (max.X-min.X-group)/2 + off
		if it.Color != 0 {
			cy := min.Y + (max.Y-min.Y-side)/2 + off
			dl.AddRectFilled(imgui.NewVec2(gx, cy), imgui.NewVec2(gx+side, cy+side), it.Color)
			dl.AddRect(imgui.NewVec2(gx, cy), imgui.NewVec2(gx+side, cy+side), p.DkShadow)
			gx += side + sc(6)
		}
		dl.AddTextVec2(imgui.NewVec2(gx, min.Y+(max.Y-min.Y-ts.Y)/2+off), TextColor(), it.Label)
		x = max.X + sc(4)
		last = max.X
	}
	if note != "" {
		nt := imgui.CalcTextSize(note)
		nx := last + sc(12)
		dl.PushClipRect(imgui.NewVec2(nx, start.Y), imgui.NewVec2(right, start.Y+h))
		dl.AddTextVec2(imgui.NewVec2(nx, start.Y+(h-nt.Y)/2), imgui.ColorU32Vec4(p.TextDim), note)
		dl.PopClipRect()
	}
	imgui.SetCursorScreenPos(imgui.NewVec2(start.X, start.Y+h+sc(2)))
	imgui.Dummy(imgui.NewVec2(0, 0))
	return clicked
}
