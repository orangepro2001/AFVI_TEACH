package core

import (
	"math"
	"testing"
)

func TestEllipseWeight(t *testing.T) {
	stroke := DefectStroke{Shape: ShapeEllipse, X: 10, Y: 10, X2: 30, Y2: 20, Softness: 0}
	if w := strokeWeight(stroke, 20, 15, 30, 20); w != 1 {
		t.Fatalf("centre weight=%v want 1", w)
	}
	if w := strokeWeight(stroke, 31, 15, 30, 20); w != 0 {
		t.Fatalf("outside weight=%v want 0", w)
	}
	if w := strokeWeight(stroke, 10, 10, 30, 20); w != 0 {
		t.Fatalf("corner weight=%v want 0", w)
	}
	// q≈0.5: cx=20, cy=15, rx=10, ry=5 → x=25, y=15 gives q=0.5
	if w := strokeWeight(stroke, 25, 15, 30, 20); w != 1 {
		t.Fatalf("q=0.5 weight=%v want 1", w)
	}

	soft := DefectStroke{Shape: ShapeEllipse, X: 10, Y: 10, X2: 30, Y2: 20, Softness: 1}
	if w := strokeWeight(soft, 20, 15, 30, 20); w != 1 {
		t.Fatalf("soft centre weight=%v want 1", w)
	}
	want := shapeWeight(0.5, 1, 1)
	got := strokeWeight(soft, 25, 15, 30, 20)
	if got != want {
		t.Fatalf("soft q=0.5 weight=%v want %v", got, want)
	}
	if !(got > 0 && got < 1) {
		t.Fatalf("soft q=0.5 weight=%v want in (0,1)", got)
	}
}

func TestPolygonWeight(t *testing.T) {
	tri := DefectStroke{Shape: ShapePolygon, Points: []Point{{X: 0, Y: 0}, {X: 20, Y: 0}, {X: 0, Y: 20}}}
	if w := strokeWeight(tri, 3, 3, 0, 0); w != 1 {
		t.Fatalf("inside weight=%v want 1", w)
	}
	if w := strokeWeight(tri, 15, 15, 0, 0); w != 0 {
		t.Fatalf("outside weight=%v want 0", w)
	}
	if w := strokeWeight(tri, -1, 5, 0, 0); w != 0 {
		t.Fatalf("negative weight=%v want 0", w)
	}

	// Concave "L": (0,0),(10,0),(10,4),(4,4),(4,10),(0,10)
	l := DefectStroke{Shape: ShapePolygon, Points: []Point{
		{X: 0, Y: 0}, {X: 10, Y: 0}, {X: 10, Y: 4}, {X: 4, Y: 4}, {X: 4, Y: 10}, {X: 0, Y: 10},
	}}
	if w := strokeWeight(l, 7, 7, 0, 0); w != 0 {
		t.Fatalf("notch weight=%v want 0", w)
	}
	if w := strokeWeight(l, 2, 2, 0, 0); w != 1 {
		t.Fatalf("arm weight=%v want 1", w)
	}

	// Softness: 1px inside an edge → 0<w<1; centre of big square → 1.
	sq := DefectStroke{Shape: ShapePolygon, Softness: 0.5, Points: []Point{
		{X: 0, Y: 0}, {X: 20, Y: 0}, {X: 20, Y: 20}, {X: 0, Y: 20},
	}}
	edge := strokeWeight(sq, 1, 10, 0, 0)
	if !(edge > 0 && edge < 1) {
		t.Fatalf("soft edge weight=%v want in (0,1)", edge)
	}
	if w := strokeWeight(sq, 10, 10, 0, 0); w != 1 {
		t.Fatalf("soft centre weight=%v want 1", w)
	}

	// <3 points → 0
	short := DefectStroke{Shape: ShapePolygon, Points: []Point{{X: 0, Y: 0}, {X: 1, Y: 1}}}
	if w := strokeWeight(short, 0, 0, 0, 0); w != 0 {
		t.Fatalf("short weight=%v want 0", w)
	}
}

func TestStrokeBoundsEllipseAndPolygon(t *testing.T) {
	ell := DefectStroke{Shape: ShapeEllipse, X: 10, Y: 20, X2: 30, Y2: 40}
	b := strokeBounds(ell)
	if b.X != 10 || b.Y != 20 || b.W != 20 || b.H != 20 {
		t.Fatalf("ellipse bounds=%+v", b)
	}
	poly := DefectStroke{Shape: ShapePolygon, Points: []Point{{X: 5, Y: 7}, {X: 15, Y: 3}, {X: 9, Y: 20}}}
	b = strokeBounds(poly)
	if b.X != 5 || b.Y != 3 || b.W != 10 || b.H != 17 {
		t.Fatalf("polygon bounds=%+v", b)
	}
	empty := DefectStroke{Shape: ShapePolygon}
	if b := strokeBounds(empty); b != (Rect{}) {
		t.Fatalf("empty bounds=%+v", b)
	}
}

func TestChannelFactor(t *testing.T) {
	plain := DefectStroke{}
	if f := plain.ChannelFactor(ChRed); f != 1 {
		t.Fatalf("plain factor=%v want 1", f)
	}
	col := DefectStroke{HasColor: true, Color: [3]float64{-0.5, 1.7, 0.5}}
	if f := col.ChannelFactor(ChRed); f != 0 {
		t.Fatalf("clamp low=%v want 0", f)
	}
	if f := col.ChannelFactor(ChGreen); f != 1 {
		t.Fatalf("clamp high=%v want 1", f)
	}
	if f := col.ChannelFactor(ChBlue); f != 0.5 {
		t.Fatalf("mid=%v want 0.5", f)
	}
	nan := DefectStroke{HasColor: true, Color: [3]float64{math.NaN(), 0, 0}}
	if f := nan.ChannelFactor(ChRed); f != 0 {
		t.Fatalf("nan=%v want 0", f)
	}
	if f := col.ChannelFactor(CameraChannel(9)); f != 1 {
		t.Fatalf("oob=%v want 1", f)
	}
}

func testColorScene(t *testing.T) (*LightScene, LightSetting) {
	t.Helper()
	scene := testUniformScene(40, 40, MatSR, 100)
	setting := LightSetting{
		Channels: []LightChannel{
			{Color: LightRed, Value: 1024, Enable: true},
			{Color: LightGreen, Value: 1024, Enable: true},
			{Color: LightBlue, Value: 1024, Enable: true},
		},
		Reflectance:     DefaultReflectance,
		Gain:            1,
		TextureStrength: 1,
	}
	return scene, setting
}

func TestRenderDefectPairColorChannels(t *testing.T) {
	scene, setting := testColorScene(t)
	rect := DefectStroke{
		Kind: Bright, Shape: ShapeRect, X: 10, Y: 10, X2: 20, Y2: 20,
		DeltaGv: 40, Softness: 0, HasColor: true, Color: [3]float64{1, 0, 0},
	}
	channels := []CameraChannel{ChRed, ChGreen, ChBlue}
	goldens := make([]*GrayImage, 3)
	tests := make([]*GrayImage, 3)
	for i, ch := range channels {
		goldens[i], tests[i] = RenderDefectPair(scene, &setting, ch, nil, []DefectStroke{rect}, NoNoise)
	}
	idx := 15*40 + 15
	if d := int(tests[0].Data[idx]) - int(goldens[0].Data[idx]); d != 40 {
		t.Fatalf("red delta=%d want 40", d)
	}
	for i := 1; i < 3; i++ {
		for j := range goldens[i].Data {
			if goldens[i].Data[j] != tests[i].Data[j] {
				t.Fatalf("channel %d changed at %d", i, j)
			}
		}
	}

	half := rect
	half.Color = [3]float64{0.5, 0, 0}
	hg, ht := RenderDefectPair(scene, &setting, ChRed, nil, []DefectStroke{half}, NoNoise)
	if d := int(ht.Data[idx]) - int(hg.Data[idx]); d != 20 {
		t.Fatalf("half red delta=%d want 20", d)
	}

	green := rect
	green.Color = [3]float64{0, 1, 0}
	gg, gt := RenderDefectPair(scene, &setting, ChGreen, nil, []DefectStroke{green}, NoNoise)
	if d := int(gt.Data[idx]) - int(gg.Data[idx]); d != 40 {
		t.Fatalf("green delta=%d want 40", d)
	}
	rg, rt := RenderDefectPair(scene, &setting, ChRed, nil, []DefectStroke{green}, NoNoise)
	for j := range rg.Data {
		if rg.Data[j] != rt.Data[j] {
			t.Fatalf("green stroke changed red at %d", j)
		}
	}

	plain := rect
	plain.HasColor = false
	white := rect
	white.Color = [3]float64{1, 1, 1}
	for _, ch := range channels {
		_, pt := RenderDefectPair(scene, &setting, ch, nil, []DefectStroke{plain}, NoNoise)
		_, wt := RenderDefectPair(scene, &setting, ch, nil, []DefectStroke{white}, NoNoise)
		for j := range pt.Data {
			if pt.Data[j] != wt.Data[j] {
				t.Fatalf("plain vs white differ at %d", j)
			}
		}
	}
}

func TestNewShapesTouchOnlyBoundingBox(t *testing.T) {
	scene, setting := testColorScene(t)
	ellipse := DefectStroke{
		Kind: Bright, Shape: ShapeEllipse, X: 10, Y: 10, X2: 30, Y2: 20,
		DeltaGv: 40, Softness: 0,
	}
	poly := DefectStroke{
		Kind: Bright, Shape: ShapePolygon,
		Points:  []Point{{X: 12, Y: 12}, {X: 28, Y: 12}, {X: 20, Y: 28}},
		DeltaGv: 40, Softness: 0,
	}
	for _, stroke := range []DefectStroke{ellipse, poly} {
		golden, test := RenderDefectPair(scene, &setting, ChRed, nil, []DefectStroke{stroke}, NoNoise)
		b := strokeBounds(stroke)
		x0 := int(math.Floor(b.X))
		y0 := int(math.Floor(b.Y))
		x1 := int(math.Floor(b.X + b.W))
		y1 := int(math.Floor(b.Y + b.H))
		for y := 0; y < 40; y++ {
			for x := 0; x < 40; x++ {
				idx := y*40 + x
				if golden.Data[idx] == test.Data[idx] {
					continue
				}
				if x < x0 || x > x1 || y < y0 || y > y1 {
					t.Fatalf("shape %d touched outside bbox at (%d,%d)", stroke.Shape, x, y)
				}
			}
		}
	}
}

func TestGenerateRandomDefectsCopiesColor(t *testing.T) {
	base := RandomDefectOptions{
		Count: 5, Seed: 7, Width: 40, Height: 40,
		Kind: RandomMixedKind, Shape: RandomMixedShape,
		RadiusMin: 2, RadiusMax: 5, ValueMin: 20, ValueMax: 60,
		Amplitude: AmpAbsolute, Profile: ProfileLinear, Softness: 0.2,
		LineLengthMax: 20,
	}
	plain := GenerateRandomDefects(base)
	colored := base
	colored.HasColor = true
	colored.Color = [3]float64{1, 0.5, 0}
	withColor := GenerateRandomDefects(colored)
	if len(plain) != len(withColor) {
		t.Fatalf("len mismatch %d vs %d", len(plain), len(withColor))
	}
	for i := range plain {
		if plain[i].X != withColor[i].X || plain[i].Y != withColor[i].Y {
			t.Fatalf("position %d differs", i)
		}
		if !withColor[i].HasColor {
			t.Fatalf("stroke %d missing HasColor", i)
		}
		if withColor[i].Color != colored.Color {
			t.Fatalf("stroke %d color=%v want %v", i, withColor[i].Color, colored.Color)
		}
	}
}
