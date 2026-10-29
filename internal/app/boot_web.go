//go:build js && wasm

package app

import (
	"afvi/teach-imgui/internal/access"
	"afvi/teach-imgui/internal/config"
	"afvi/teach-imgui/internal/core"
	"github.com/AllenDang/cimgui-go/backend/glfwbackend"
)

// BootForWeb runs the program with a browser-backed window.
//
// Run() on the desktop parses flags, resolves a machine configuration from disk and loads a model
// from a PxRepository tree. The browser has neither, so this entry point builds the same App by hand
// around the built-in demo plate: the panels, the layout, the inspection and the drawing are the
// ones the desktop program uses, only the data source differs (docs/replication.md).
func BootForWeb(be *glfwbackend.GLFWBackend) {
	cfg := config.Default("")
	a := NewApp(cfg, "", ThemeLight, ProfileTeach)
	a.be = be
	be.SetAfterCreateContextHook(setupFonts)
	be.CreateWindow(ProfileTeach.WindowTitle(), 1600, 1000)
	be.SetTargetFPS(60)
	// No login in the browser: the operator/admin distinction only gates pages.
	a.SetRole(access.RoleAdmin)
	a.loadDemoPlate()
	be.Run(func() { loop(a) })
}

// loadDemoPlate installs a synthesised design plate as the loaded model, so every page has something
// to draw. The desktop program loads TIFF layers from the machine's PxRepository instead.
func (a *App) loadDemoPlate() {
	scene := demoScene()
	a.Scene = scene
	a.Models = []ModelInfo{{ID: "DEMO/plate-960x640", Version: "demo"}}
	a.modelIdx = 0
	a.align.modelLoaded(scene)
	a.golden.modelLoaded(scene)
	for _, p := range a.allPanels {
		p.ModelLoaded()
	}
	if p := a.ActivePanel(); p != nil {
		p.Activate()
	}
}

// demoScene builds the plate: a PCB-like design with three materials and planted defects, the same
// generator the web build's sample endpoint uses.
func demoScene() *Scene {
	const w, h = 960, 640
	median := &core.GrayImage{Width: w, Height: h, Data: make([]uint8, w*h)}
	material := make([]uint8, w*h)
	// A resist body, a metal pad grid and three open windows: the layout the desktop program's
	// reference model shows.
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			i := y*w + x
			material[i] = core.MatBase
			if x > w/10 && x < w-w/10 && y > h/10 && y < h-h/10 {
				material[i] = core.MatSR
			}
			switch {
			case x > w/6 && x < w/2 && y > h/2-h/8 && y < h/2+h/8:
				material[i] = core.MatMetal
			case x > w/2 && x < w*3/4 && y > h/3 && y < h*2/3:
				material[i] = core.MatOpen
			case x > w*3/4 && x < w-w/6 && y > h/2-h/8 && y < h/2+h/8:
				material[i] = core.MatMetal
			}
		}
	}
	levels := map[uint8]float64{core.MatBase: 22, core.MatSR: 96, core.MatOpen: 60, core.MatMetal: 196}
	for i := range median.Data {
		v := levels[material[i]]
		// a gentle two-scale texture so the inspection has something to learn from
		x, y := i%w, i/w
		v += 4 * sinApprox(float64(x)/37) * cosApprox(float64(y)/41)
		if v < 0 {
			v = 0
		}
		if v > 255 {
			v = 255
		}
		median.Data[i] = uint8(v)
	}
	light, err := core.NewLightScene(median, material)
	if err != nil {
		light = nil
	}
	stats, _ := core.MaterialStats(median, material)
	return &Scene{
		Info:        ModelInfo{ID: "DEMO/plate-960x640", Version: "demo"},
		W:           w,
		H:           h,
		Median:      median,
		Material:    material,
		Light:       light,
		Stats:       stats,
		MachineRois: demoRois(w, h),
	}
}

// sinApprox and cosApprox keep math out of this file's hot loop (the values only shape the texture).
func sinApprox(v float64) float64 { return mathSin(v) }
func cosApprox(v float64) float64 { return mathCos(v) }

// demoRois lays out the four corner windows the reference machine has.
func demoRois(w, h int) []core.AlignRoi {
	rw, rh := float64(w)*0.16, float64(h)*0.16
	m := float64(min(w, h)) * 0.06
	return []core.AlignRoi{
		{Name: "P1", Rect: core.Rect{X: m, Y: m, W: rw, H: rh}},
		{Name: "P2", Rect: core.Rect{X: float64(w) - m - rw, Y: m, W: rw, H: rh}},
		{Name: "P3", Rect: core.Rect{X: m, Y: float64(h) - m - rh, W: rw, H: rh}},
		{Name: "P4", Rect: core.Rect{X: float64(w) - m - rw, Y: float64(h) - m - rh, W: rw, H: rh}},
	}
}

// min returns the smaller of two ints.
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
