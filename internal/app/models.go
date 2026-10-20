package app

import (
	"afvi/teach-imgui/internal/config"
	"afvi/teach-imgui/internal/core"
	"afvi/teach-imgui/internal/model"
)

// ModelInfo aliases model.ModelInfo so the rest of the app keeps compiling unchanged.
type ModelInfo = model.ModelInfo

// Scene aliases model.Scene so the rest of the app keeps compiling unchanged.
type Scene = model.Scene

var (
	ErrPathNotSet   = model.ErrPathNotSet
	ErrPathNotFound = model.ErrPathNotFound
	ErrNoRepository = model.ErrNoRepository
)

// FindReferenceRoot resolves the reference directory; see model.FindReferenceRoot.
func FindReferenceRoot(override string) (string, error) { return model.FindReferenceRoot(override) }

// ScanMachine scans every host of one machine; see model.ScanMachine.
func ScanMachine(cfg *config.Config, m config.Machine) []ModelInfo { return model.ScanMachine(cfg, m) }

// CheckHostPath reports how many models a configured host path yields; see model.CheckHostPath.
func CheckHostPath(cfg *config.Config, host, path string) (int, error) {
	return model.CheckHostPath(cfg, host, path)
}

// LoadScene decodes the four layers and builds the light scene; see model.LoadScene.
func LoadScene(info ModelInfo) (*Scene, error) { return model.LoadScene(info) }

// ApplyProduct rebuilds the light scene of the loaded model with a new virtual good unit
// (background job "product"; a newer call replaces a pending one), then publishes a shallow
// copy of the scene so jobs that captured the old scene drop their stale results, and tells
// every panel the light changed. Strokes and other panel state are kept.
func (a *App) ApplyProduct(opts core.ProductOptions) {
	scene := a.Scene
	if scene == nil {
		return
	}
	a.Jobs.Run("product", func() func() {
		light, err := core.NewLightScene(scene.Median, scene.Material)
		if err != nil {
			return func() { a.SetHint(T("common.error", "msg", err.Error())) }
		}
		core.ApplyProductVariation(light, opts)
		return func() {
			if a.Scene != scene {
				return
			}
			next := *scene
			next.Light = light
			next.Product = opts
			a.Scene = &next
			// The golden sample picture is rendered from the baked product, so the board starts over.
			a.golden.modelLoaded(a.Scene)
			a.NotifyLightChanged()
		}
	})
}
