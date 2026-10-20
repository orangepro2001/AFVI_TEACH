// Package camera renders a capture (test and good images) from a model and a recipe.
package camera

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"afvi/teach-imgui/internal/core"
	"afvi/teach-imgui/internal/model"
	"afvi/teach-imgui/internal/snap"
)

// Producer identifies the program that wrote a capture.
const Producer = "afvi-camera 1"

// ImageKey identifies one frame of a capture.
type ImageKey struct {
	Channel string
	Kind    string
}

// Options selects what Render produces.
type Options struct {
	Channels []core.CameraChannel // empty means red only; duplicates are ignored
	Good     bool                 // also render the defect-free good sample of every channel
}

// Result is a rendered capture: the description (Images filled in) and the pixels.
type Result struct {
	Meta   snap.Meta
	Frames map[ImageKey]snap.Frame
}

// DefaultRecipe is the camera's default scenario for a scene. Like a real camera it is slightly
// misaligned: the product is shifted by a few pixels and rotated by a fraction of a degree
// (core.DefaultCaptureWarp). The good sample stays at the nominal position because Render always
// renders it un-warped; only the test image moves.
func DefaultRecipe(scene *model.Scene, count int, seed uint32) snap.Recipe {
	light := core.DefaultLightSetting()
	light.Channels = []core.LightChannel{
		{Color: core.LightRed, Value: 512, Enable: true},
		{Color: core.LightGreen, Value: 512, Enable: true},
		{Color: core.LightBlue, Value: 512, Enable: true},
		{Color: core.LightWhite, Value: 0, Enable: false},
	}
	product := core.DefaultProductOptions()
	noise := core.SensorNoise{Seed: 1}
	var defects []core.DefectStroke
	if scene != nil {
		defects = core.DefaultTinyDefects(scene.Material, scene.W, scene.H, count, seed, core.DefaultTinyEdgeMargin)
	}
	return snap.NewRecipe(product, light, noise, defects, nil).WithWarp(core.DefaultCaptureWarp())
}

// NewMeta builds the description without images.
func NewMeta(scene *model.Scene, recipe snap.Recipe) snap.Meta {
	m := snap.Meta{
		Format:   snap.Format,
		Version:  snap.Version,
		Producer: Producer,
		Created:  time.Now().UTC().Format(time.RFC3339),
		Recipe:   recipe,
		Images:   make([]snap.Image, 0),
	}
	if scene != nil {
		m.Model = scene.Info.Ref()
		m.Width = scene.W
		m.Height = scene.H
	}
	return m
}

// BuildLightScene returns core.NewLightScene(scene.Median, scene.Material) with core.ApplyProductVariation(.., product) applied.
func BuildLightScene(scene *model.Scene, product core.ProductOptions) (*core.LightScene, error) {
	if scene == nil {
		return nil, fmt.Errorf("camera: nil scene")
	}
	ls, err := core.NewLightScene(scene.Median, scene.Material)
	if err != nil {
		return nil, err
	}
	core.ApplyProductVariation(ls, product)
	return ls, nil
}

// Render synthesises the images the recipe describes.
func Render(scene *model.Scene, recipe snap.Recipe, opt Options) (*Result, error) {
	if scene == nil {
		return nil, fmt.Errorf("camera: nil scene")
	}
	cr := recipe.Core()
	ls, err := BuildLightScene(scene, cr.Product)
	if err != nil {
		return nil, err
	}
	channels := opt.Channels
	if len(channels) == 0 {
		channels = []core.CameraChannel{core.ChRed}
	}
	seen := make(map[core.CameraChannel]bool)
	meta := NewMeta(scene, recipe)
	frames := make(map[ImageKey]snap.Frame)
	for _, ch := range channels {
		if seen[ch] {
			continue
		}
		seen[ch] = true
		name := snap.ChannelName(ch)
		if name == "" {
			return nil, fmt.Errorf("camera: unknown channel %d", ch)
		}
		light := cr.Light
		// The product sits where the recipe's warp says (misalignment, lift); the defects move with it.
		golden, test := core.RenderDefectPairWarped(ls, &light, ch, cr.Scene, cr.Defects, cr.Noise, cr.Warp)
		_ = golden
		testFrame := snap.Frame{Width: test.Width, Height: test.Height, Pix: test.Data}
		key := ImageKey{Channel: name, Kind: snap.KindTest}
		frames[key] = testFrame
		meta.SetImage(snap.Image{
			Channel: name,
			Kind:    snap.KindTest,
			File:    snap.ImageFile(name, snap.KindTest),
			Width:   testFrame.Width,
			Height:  testFrame.Height,
			SHA256:  snap.PixelHash(testFrame),
		})
		if opt.Good {
			// The good sample is the taught reference: it always sits at the nominal position.
			good := core.RenderGoodSample(ls, &light, ch, core.GoodSampleNoise(cr.Noise))
			goodFrame := snap.Frame{Width: good.Width, Height: good.Height, Pix: good.Data}
			goodKey := ImageKey{Channel: name, Kind: snap.KindGood}
			frames[goodKey] = goodFrame
			meta.SetImage(snap.Image{
				Channel: name,
				Kind:    snap.KindGood,
				File:    snap.ImageFile(name, snap.KindGood),
				Width:   goodFrame.Width,
				Height:  goodFrame.Height,
				SHA256:  snap.PixelHash(goodFrame),
			})
		}
	}
	return &Result{Meta: meta, Frames: frames}, nil
}

// Save writes res into dir (created if needed).
func Save(dir string, res *Result) error {
	if res == nil {
		return fmt.Errorf("camera: nil result")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	meta := res.Meta
	old, oldDir, err := snap.ReadMeta(dir)
	if err == nil && oldDir == dir && old.Model.ID == meta.Model.ID &&
		old.Width == meta.Width && old.Height == meta.Height && sameRecipe(old.Recipe, meta.Recipe) {
		replaced := make(map[ImageKey]bool)
		for k := range res.Frames {
			replaced[k] = true
		}
		for _, img := range old.Images {
			key := ImageKey{Channel: img.Channel, Kind: img.Kind}
			if replaced[key] {
				continue
			}
			if _, err := os.Stat(filepath.Join(dir, img.File)); err != nil {
				continue
			}
			meta.SetImage(img)
		}
	}
	for key, frame := range res.Frames {
		if err := snap.WriteImage(dir, &meta, key.Channel, key.Kind, frame); err != nil {
			return err
		}
	}
	res.Meta = meta
	return snap.WriteMeta(filepath.Join(dir, snap.MetaFile), &meta)
}

func sameRecipe(a, b snap.Recipe) bool {
	ab, errA := json.Marshal(a)
	bb, errB := json.Marshal(b)
	if errA != nil || errB != nil {
		return false
	}
	return string(ab) == string(bb)
}

// SceneFromMeta loads the model the description refers to and checks that the loaded size equals m.Width x m.Height.
func SceneFromMeta(m *snap.Meta) (*model.Scene, error) {
	if m == nil {
		return nil, fmt.Errorf("camera: nil meta")
	}
	info := model.InfoFromRef(m.Model)
	if err := info.LayersExist(); err != nil {
		return nil, err
	}
	scene, err := model.LoadScene(info)
	if err != nil {
		return nil, err
	}
	if scene.W != m.Width || scene.H != m.Height {
		return nil, fmt.Errorf("camera: scene is %dx%d, meta says %dx%d", scene.W, scene.H, m.Width, m.Height)
	}
	return scene, nil
}
