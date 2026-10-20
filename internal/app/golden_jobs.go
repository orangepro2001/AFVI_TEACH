package app

import (
	"errors"
	"fmt"

	"afvi/teach-imgui/internal/core"
	"afvi/teach-imgui/internal/golden"
)

// goldenGeometry is the design material map moved onto the real (golden sample)
// picture. Immutable once built.
type goldenGeometry struct {
	scene      *Scene
	fit        core.Similarity // DesignInfo.Fit of the registration ({Scale: 1} when not registered)
	registered bool
	srReal     *golden.Mask // design SR layer on the real picture
	srPic      *golden.Mask // srReal snapped to the edges of the golden sample picture (nil: use srReal)
	metalReal  *golden.Mask // design Metal layer on the real picture
	reg        []uint8      // the design material map on the real picture (the scene's own map when not registered)
}

// srMask returns the SR layer the automatic SR mask is made from: the picture-snapped
// one when there is one, otherwise the registered design layer.
func (g *goldenGeometry) srMask() *golden.Mask {
	if g == nil {
		return nil
	}
	if g.srPic != nil {
		return g.srPic
	}
	return g.srReal
}

// goldenRegScale is the working resolution divisor of the registration.
const goldenRegScale = 2

// goldenSourceImage renders the defect-free golden sample picture of channel ch
// on the design grid.
func goldenSourceImage(scene *Scene, light core.LightSetting, noise core.SensorNoise, ch core.CameraChannel) *core.GrayImage {
	if scene == nil {
		return nil
	}
	return core.RenderGoodSample(scene.Light, &light, ch, goodSampleNoise(noise))
}

// goldenBuildGeometry registers the design layers to the picture ref (the golden
// sample picture of any channel).
func goldenBuildGeometry(scene *Scene, ref *core.GrayImage) (*goldenGeometry, error) {
	if scene == nil {
		return nil, errors.New("golden: no scene")
	}
	if ref == nil {
		return nil, errors.New("golden: no reference picture")
	}
	if scene.W <= 0 || scene.H <= 0 {
		return nil, fmt.Errorf("golden: invalid scene size %dx%d", scene.W, scene.H)
	}
	if ref.Width != scene.W || ref.Height != scene.H {
		return nil, fmt.Errorf("golden: reference size %dx%d does not match design %dx%d",
			ref.Width, ref.Height, scene.W, scene.H)
	}
	_, info, err := core.DesignTemplate(scene.Material, ref, goldenRegScale)
	if err != nil {
		return nil, err
	}
	geo := &goldenGeometry{scene: scene, fit: core.Similarity{Scale: 1}}
	material := scene.Material
	if info.Registered {
		geo.registered = true
		geo.fit = info.Fit
		material = core.WarpMaterialToReference(scene.Material, scene.W, scene.H, info.Fit, goldenRegScale)
	}
	geo.reg = material
	geo.srReal = golden.MaskFromMaterial(material, scene.W, scene.H, core.MatSR)
	geo.srPic = goldenSnapSR(ref, geo.srReal)
	geo.metalReal = golden.MaskFromMaterial(material, scene.W, scene.H, core.MatMetal)
	return geo, nil
}

// goldenAuto is the result of the automatic Metal mask.
type goldenAuto struct {
	Low, High int
	OutRange  bool
	Metal     *golden.Mask
	Fallback  bool // the range could not be derived (flat channel / empty class): Metal is the design layer itself
}

// goldenAutoMetal derives a first Metal mask from the picture and the registered
// design layers.
func goldenAutoMetal(src *core.GrayImage, geo *goldenGeometry) goldenAuto {
	if src == nil {
		return goldenAuto{Low: 128, High: 255, Metal: golden.NewMask(0, 0), Fallback: true}
	}
	w, h := src.Width, src.Height
	fallback := func() goldenAuto {
		out := goldenAuto{Low: 128, High: 255, Fallback: true}
		if geo != nil && geo.metalReal != nil {
			out.Metal = geo.metalReal.Clone()
		} else {
			out.Metal = golden.NewMask(w, h)
		}
		return out
	}
	if geo == nil || geo.metalReal == nil || geo.srReal == nil {
		return fallback()
	}
	if geo.metalReal.W != w || geo.metalReal.H != h || geo.srReal.W != w || geo.srReal.H != h {
		return fallback()
	}
	metalMean, okMetal := goldenMaskMean(src.Data, w, h, geo.metalReal)
	srMean, okSR := goldenMaskMean(src.Data, w, h, geo.srReal)
	lo, ok := golden.AutoRange(src.Data, w, h, geo.metalReal, geo.srReal)
	if !ok || !okMetal || !okSR {
		return fallback()
	}
	out := goldenAuto{}
	if metalMean >= srMean {
		out.Low = int(lo)
		out.High = 255
	} else {
		out.Low = 0
		hi := int(lo) - 1
		if hi < 0 {
			hi = 0
		}
		out.High = hi
	}
	out.Metal = golden.Open(golden.Threshold(src.Data, w, h, uint8(out.Low), uint8(out.High), false), 1)
	return out
}

// goldenMaskMean returns the mean grey of src over the set pixels of m.
func goldenMaskMean(pix []uint8, w, h int, m *golden.Mask) (float64, bool) {
	if m == nil || !m.Valid() || len(pix) < w*h {
		return 0, false
	}
	sum, n := 0, 0
	for i := 0; i < w*h; i++ {
		if m.Pix[i] != 0 {
			sum += int(pix[i])
			n++
		}
	}
	if n == 0 {
		return 0, false
	}
	return float64(sum) / float64(n), true
}

// goldenSnapSR returns the design SR layer limited to the pixels that look like SR in the golden
// sample picture. It never grows beyond the layer (growing would pull base pixels into the
// inspected area) and gives the layer back unchanged when the picture contradicts it.
func goldenSnapSR(src *core.GrayImage, sr *golden.Mask) *golden.Mask {
	if sr == nil {
		return nil
	}
	snapped := golden.And(sr, goldenRefineSR(src, sr))
	if snapped == nil || snapped.Count()*10 < sr.Count()*7 {
		return sr.Clone()
	}
	return snapped
}
