package app

import "afvi/teach-imgui/internal/ui"

// PaletteTool is one tool of the floating viewer palette.
type PaletteTool int

const (
	PaletteNone PaletteTool = iota // no tool: left drags pan the view
	PalettePick                    // point select
	PaletteRect
	PaletteEllipse
	PalettePolygon
)

// PaletteHost is implemented by panels that use frame tools. The palette is only a view over
// the panel's own tool state, so a panel keeps full control of what each tool does.
type PaletteHost interface {
	PaletteSupports(t PaletteTool) bool // false => the button is greyed out and ignores clicks
	PaletteTool() PaletteTool           // currently active tool (PaletteNone if none)
	SetPaletteTool(t PaletteTool)       // user clicked an enabled button; PaletteNone = back to pan
}

// toggleFocus flips focus mode and refits the image to the new canvas.
func (a *App) toggleFocus() {
	a.focusView = !a.focusView
	a.Viewer.Fit()
}

// buildPalette records the floating tool palette (drawn by the front end at the top right
// of the viewer): four frame tools plus the focus toggle.
func (a *App) buildPalette(panel Panel) {
	if h, ok := panel.(interface{ HidePalette() bool }); ok && h.HidePalette() {
		return
	}
	host, _ := panel.(PaletteHost)
	type item struct {
		ID      string `json:"id"`
		Icon    string `json:"icon"`
		Active  bool   `json:"active"`
		Enabled bool   `json:"enabled"`
		Tip     string `json:"tip"`
		Sep     bool   `json:"sep,omitempty"`
	}
	var items []item
	for _, e := range []struct {
		t    PaletteTool
		name string
	}{{PalettePick, "pick"}, {PaletteRect, "rect"}, {PaletteEllipse, "ellipse"}, {PalettePolygon, "polygon"}} {
		supported := host != nil && host.PaletteSupports(e.t)
		active := supported && host.PaletteTool() == e.t
		tip := T("palette.tool." + e.name)
		if !supported {
			tip += "\n" + T("palette.unsupported")
		}
		id := "pal-" + e.name
		if ui.Clicked(id) && supported {
			if active {
				host.SetPaletteTool(PaletteNone)
			} else {
				host.SetPaletteTool(e.t)
			}
		}
		items = append(items, item{ID: id, Icon: e.name, Active: active, Enabled: supported, Tip: tip})
	}
	if ui.Clicked("pal-focus") {
		a.toggleFocus()
	}
	items = append(items, item{ID: "pal-focus", Icon: "focus", Active: a.focusView, Enabled: true, Tip: T("palette.tool.focus"), Sep: true})
	ui.Add(&ui.Node{K: "palette", P: map[string]any{"items": items}})
}

func init() {
	Messages(map[string]string{
		"palette.tool.pick":    "点选",
		"palette.tool.rect":    "矩形框",
		"palette.tool.ellipse": "椭圆框",
		"palette.tool.polygon": "多边形框",
		"palette.tool.focus":   "专注模式 (F11)：隐藏两侧面板，放大中央图像",
		"palette.unsupported":  "当前页面不支持该工具",
		"menu.focus":           "专注模式",
	}, map[string]string{
		"palette.tool.pick":    "Pick",
		"palette.tool.rect":    "Rectangle frame",
		"palette.tool.ellipse": "Ellipse frame",
		"palette.tool.polygon": "Polygon frame",
		"palette.tool.focus":   "Focus mode (F11): hide the side panels to enlarge the centre image",
		"palette.unsupported":  "Not supported on this page",
		"menu.focus":           "Focus mode",
	}, map[string]string{
		"palette.tool.pick":    "선택",
		"palette.tool.rect":    "사각형 프레임",
		"palette.tool.ellipse": "타원 프레임",
		"palette.tool.polygon": "다각형 프레임",
		"palette.tool.focus":   "집중 모드 (F11): 양쪽 패널을 숨기고 중앙 이미지를 확대",
		"palette.unsupported":  "이 페이지에서는 지원되지 않습니다",
		"menu.focus":           "집중 모드",
	})
}
