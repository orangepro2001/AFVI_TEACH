package core

import (
	"bytes"
	"math"
	"testing"
)

func TestTiltMovesPointsLikeALiftedEdge(t *testing.T) {
	w, h := 200, 150
	p := warpParams(func(p *WarpParams) { p.TiltX = 0.2 })
	// 右边缘（相对中心 +99.5）被压缩 1/1.2，左边缘（-99.5）被拉伸 1/0.8；中心不动。
	right := WarpPoint(&p, Point{X: 199, Y: 74.5}, w, h)
	wantRight := 99.5 + 99.5/1.2
	if math.Abs(right.X-wantRight) > 1e-9 || math.Abs(right.Y-74.5) > 1e-9 {
		t.Fatalf("right edge -> %+v, want x %v", right, wantRight)
	}
	left := WarpPoint(&p, Point{X: 0, Y: 74.5}, w, h)
	wantLeft := 99.5 - 99.5/0.8
	if math.Abs(left.X-wantLeft) > 1e-9 {
		t.Fatalf("left edge -> %+v, want x %v", left, wantLeft)
	}
	centre := WarpPoint(&p, Point{X: 99.5, Y: 74.5}, w, h)
	if math.Abs(centre.X-99.5) > 1e-12 || math.Abs(centre.Y-74.5) > 1e-12 {
		t.Fatalf("centre moved to %+v", centre)
	}
	// TiltY 对称地作用在纵向。
	py := warpParams(func(p *WarpParams) { p.TiltY = -0.1 })
	bottom := WarpPoint(&py, Point{X: 99.5, Y: 149}, w, h)
	wantBottom := 74.5 + 74.5/0.9
	if math.Abs(bottom.Y-wantBottom) > 1e-9 {
		t.Fatalf("bottom edge -> %+v, want y %v", bottom, wantBottom)
	}
}

func TestUnwarpPointInvertsWarpPointWithTilt(t *testing.T) {
	w, h := 5860, 4020
	p := warpParams(func(p *WarpParams) {
		p.Tx = 40
		p.Ty = -25
		p.ThetaDeg = 0.8
		p.Scale = 1.01
		p.ShearX = 0.01
		p.ShearY = -0.02
		p.TiltX = 0.15
		p.TiltY = -0.1
		p.BendAmplitude = 12
		p.BendWavelength = 900
	})
	for _, pt := range []Point{{10, 10}, {5000, 100}, {2930, 2010}, {5800, 4000}, {1234.5, 3333.25}} {
		q := WarpPoint(&p, pt, w, h)
		back, ok := UnwarpPoint(&p, q, w, h)
		if !ok {
			t.Fatalf("point %+v not invertible", pt)
		}
		if math.Abs(back.X-pt.X) > 1e-6 || math.Abs(back.Y-pt.Y) > 1e-6 {
			t.Fatalf("point %+v round-trips to %+v", pt, back)
		}
	}
	// 分母不为正的位置不可逆。
	if _, ok := UnwarpPoint(&p, Point{X: 1e9, Y: 1e9}, w, h); ok {
		t.Fatalf("a point beyond the vanishing line must not be invertible")
	}
}

func TestGeometryIdentity(t *testing.T) {
	if !(WarpParams{}).GeometryIdentity() || !IdentityWarp.GeometryIdentity() {
		t.Fatalf("zero value and IdentityWarp must be geometry identity")
	}
	photometric := warpParams(func(p *WarpParams) {
		p.BlurSigma = 2
		p.NoiseSigma = 5
		p.Gain = 1.4
		p.Offset = 20
		p.Gradient = 0.3
		p.Occlusions = []Occlusion{{Rect: Rect{X: 1, Y: 1, W: 5, H: 5}, Value: 0}}
	})
	if !photometric.GeometryIdentity() {
		t.Fatalf("photometric changes are not geometry")
	}
	if bend := warpParams(func(p *WarpParams) { p.BendAmplitude = 5 }); !bend.GeometryIdentity() {
		t.Fatalf("a bend without wavelength does nothing")
	}
	for name, mod := range map[string]func(p *WarpParams){
		"tx":    func(p *WarpParams) { p.Tx = 1 },
		"ty":    func(p *WarpParams) { p.Ty = -1 },
		"theta": func(p *WarpParams) { p.ThetaDeg = 0.1 },
		"scale": func(p *WarpParams) { p.Scale = 1.01 },
		"shearX": func(p *WarpParams) {
			p.ShearX = 0.01
		},
		"shearY": func(p *WarpParams) { p.ShearY = 0.01 },
		"tiltX":  func(p *WarpParams) { p.TiltX = 0.05 },
		"tiltY":  func(p *WarpParams) { p.TiltY = -0.05 },
		"bend": func(p *WarpParams) {
			p.BendAmplitude = 3
			p.BendWavelength = 500
		},
	} {
		if q := warpParams(mod); q.GeometryIdentity() {
			t.Fatalf("%s must not be geometry identity", name)
		}
	}
	g := photometric.Geometry()
	if g.Gain != 1 || g.Offset != 0 || g.BlurSigma != 0 || g.NoiseSigma != 0 || g.Gradient != 0 || len(g.Occlusions) != 0 || g.Scale != 1 {
		t.Fatalf("Geometry kept photometric fields: %+v", g)
	}
	if zeroScale := (WarpParams{Tx: 3}).Geometry(); zeroScale.Scale != 1 || zeroScale.Tx != 3 {
		t.Fatalf("Geometry of a zero-scale value: %+v", zeroScale)
	}
	if b := (WarpParams{BendAmplitude: 4}).Geometry(); b.BendAmplitude != 0 || b.BendWavelength != 0 {
		t.Fatalf("an ineffective bend must be dropped: %+v", b)
	}
}

func TestValidateCaptureWarp(t *testing.T) {
	good := WarpParams{Tx: 30, Ty: -20, ThetaDeg: 1.5, Scale: 1.01, ShearX: 0.01, TiltX: 0.2, TiltY: -0.1, BendAmplitude: 10, BendWavelength: 800}
	if err := ValidateCaptureWarp(good); err != nil {
		t.Fatalf("good warp rejected: %v", err)
	}
	if err := ValidateCaptureWarp(WarpParams{}); err != nil {
		t.Fatalf("zero warp rejected: %v", err)
	}
	for name, bad := range map[string]WarpParams{
		"shift":    {Tx: 2500},
		"rotation": {ThetaDeg: 20},
		"scale":    {Scale: 2},
		"shear":    {ShearX: 0.5},
		"tilt":     {TiltX: 0.6},
		"tiltsum":  {TiltX: 0.45, TiltY: 0.45},
		"bend":     {BendAmplitude: 500, BendWavelength: 800},
		"bendwave": {BendAmplitude: 5, BendWavelength: 10},
		"nan":      {Tx: math.NaN()},
		"inf":      {BendWavelength: math.Inf(1)},
	} {
		if err := ValidateCaptureWarp(bad); err == nil {
			t.Fatalf("%s should be rejected", name)
		}
	}
}

// warpTestScene 是一个有结构的小场景，使平移后的像素能逐个对应。
func warpTestScene() (*LightScene, LightSetting) {
	w, h := 64, 48
	median := NewGray(w, h, 0)
	material := make([]uint8, w*h)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			median.Data[y*w+x] = uint8(40 + (x*7+y*13)%120)
			material[y*w+x] = 1
		}
	}
	scene, err := NewLightScene(median, material)
	if err != nil {
		panic(err)
	}
	return scene, testRedSetting(512, 1)
}

func TestRenderDefectPairWarpedIdentityEqualsPlain(t *testing.T) {
	scene, light := warpTestScene()
	noise := SensorNoise{ReadSigma: 2, ShotGain: 0.05, Seed: 9}
	strokes := []DefectStroke{{Kind: Bright, Shape: ShapeSpot, X: 20, Y: 20, Radius: 3, DeltaGv: 60, Amplitude: AmpAbsolute}}
	g0, t0 := RenderDefectPair(scene, &light, ChRed, nil, strokes, noise)
	for _, w := range []WarpParams{{}, IdentityWarp, {Scale: 1, Gain: 1.5, BlurSigma: 3}} {
		g1, t1 := RenderDefectPairWarped(scene, &light, ChRed, nil, strokes, noise, w)
		if !bytes.Equal(g0.Data, g1.Data) || !bytes.Equal(t0.Data, t1.Data) {
			t.Fatalf("identity geometry %+v must equal RenderDefectPair", w)
		}
	}
}

func TestRenderDefectPairWarpedShiftsTheWholeProduct(t *testing.T) {
	scene, light := warpTestScene()
	strokes := []DefectStroke{{Kind: Bright, Shape: ShapeSpot, X: 20, Y: 20, Radius: 3, DeltaGv: 60, Amplitude: AmpAbsolute}}
	g0, t0 := RenderDefectPair(scene, &light, ChRed, nil, strokes, SensorNoise{Seed: 1})
	w := WarpParams{Tx: 5, Ty: 3, Scale: 1}
	g1, t1 := RenderDefectPairWarped(scene, &light, ChRed, nil, strokes, SensorNoise{Seed: 1}, w)
	width, height := t0.Width, t0.Height
	for y := 0; y < height-3; y++ {
		for x := 0; x < width-5; x++ {
			if t1.Data[(y+3)*width+(x+5)] != t0.Data[y*width+x] {
				t.Fatalf("test image: pixel (%d,%d) did not move by (5,3)", x, y)
			}
			if g1.Data[(y+3)*width+(x+5)] != g0.Data[y*width+x] {
				t.Fatalf("golden image: pixel (%d,%d) did not move by (5,3)", x, y)
			}
		}
	}
	// 移出视野之外的一侧是黑色。
	for x := 0; x < 5; x++ {
		if t1.Data[10*width+x] != 0 {
			t.Fatalf("uncovered pixel (%d,10) is %d, want 0", x, t1.Data[10*width+x])
		}
	}
}

func TestRenderDefectPairWarpedNoiseIsAddedAfterTheWarp(t *testing.T) {
	scene, light := warpTestScene()
	w := WarpParams{Tx: 2, Scale: 1}
	noise := SensorNoise{ReadSigma: 3, Seed: 4}
	_, clean := RenderDefectPairWarped(scene, &light, ChRed, nil, nil, SensorNoise{Seed: 4}, w)
	g, noisy := RenderDefectPairWarped(scene, &light, ChRed, nil, nil, noise, w)
	_, again := RenderDefectPairWarped(scene, &light, ChRed, nil, nil, noise, w)
	if !bytes.Equal(noisy.Data, again.Data) {
		t.Fatalf("warped noisy render is not deterministic")
	}
	if bytes.Equal(noisy.Data, clean.Data) {
		t.Fatalf("noise had no effect")
	}
	if !bytes.Equal(g.Data, func() []uint8 {
		geo := w.Geometry()
		g0, _ := RenderDefectPair(scene, &light, ChRed, nil, nil, SensorNoise{Seed: 4})
		return ApplyWarp(g0, &geo, 0).Data
	}()) {
		t.Fatalf("golden must stay noise-free")
	}
	// AddSensorNoise 在噪声关闭时只复制。
	copyOnly := AddSensorNoise(clean, SensorNoise{Seed: 4})
	if !bytes.Equal(copyOnly.Data, clean.Data) || &copyOnly.Data[0] == &clean.Data[0] {
		t.Fatalf("AddSensorNoise without noise must return an equal copy")
	}
}
