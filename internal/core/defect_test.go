package core

import (
	"math"
	"reflect"
	"testing"
)

func testCreateGray(width, height int, fill uint8) *GrayImage {
	return NewGray(width, height, fill)
}

func testUniformScene(width, height int, materialIndex uint8, medianValue uint8) *LightScene {
	median := testCreateGray(width, height, medianValue)
	material := make([]uint8, width*height)
	for i := range material {
		material[i] = materialIndex
	}
	scene, err := NewLightScene(median, material)
	if err != nil {
		panic(err)
	}
	return scene
}

func testRedSetting(value float64, gain float64) LightSetting {
	return LightSetting{
		Channels:    []LightChannel{{Color: LightRed, Value: value, Enable: true}},
		Reflectance: DefaultReflectance,
		Gain:        gain,
	}
}

func TestHashGrainMatchesTS(t *testing.T) {
	cases := []struct {
		x, y int
		seed uint32
		want float64
	}{
		{3, 7, 1, 0.6075038067065179},
		{-5, 12, 42, 0.9535387260839343},
		{1000, 2000, 9, 0.6104653077200055},
	}
	for _, c := range cases {
		got := hashGrain(c.x, c.y, c.seed)
		if got != c.want {
			t.Fatalf("hashGrain(%d,%d,%d)=%v want %v", c.x, c.y, c.seed, got, c.want)
		}
	}
}

func TestPaintDefectsBrightensAndDarkens(t *testing.T) {
	img := testCreateGray(20, 20, 100)
	bright := PaintDefects(img, []DefectStroke{
		{Kind: Bright, Shape: ShapeSpot, X: 10, Y: 10, Radius: 3, DeltaGv: 50, Softness: 0},
	})
	if bright.Data[10*20+10] != 150 {
		t.Fatalf("bright=%d want 150", bright.Data[10*20+10])
	}
	dark := PaintDefects(img, []DefectStroke{
		{Kind: Dark, Shape: ShapeSpot, X: 10, Y: 10, Radius: 3, DeltaGv: 50, Softness: 0},
	})
	if dark.Data[10*20+10] != 50 {
		t.Fatalf("dark=%d want 50", dark.Data[10*20+10])
	}
}

func TestPaintDefectsLineWeightFallsOff(t *testing.T) {
	img := testCreateGray(20, 20, 0)
	out := PaintDefects(img, []DefectStroke{
		{Kind: Bright, Shape: ShapeLine, X: 2, Y: 10, X2: 18, Y2: 10, Radius: 4, DeltaGv: 100, Softness: 0.5},
	})
	center := out.Data[10*20+10]
	edge := out.Data[13*20+10]
	if center != 100 {
		t.Fatalf("center=%d want 100", center)
	}
	if edge != 50 {
		t.Fatalf("edge=%d want 50", edge)
	}
}

func TestPaintSceneMaterialContrast(t *testing.T) {
	median := testCreateGray(40, 40, 100)
	material := make([]uint8, 40*40)
	for i := range material {
		material[i] = 1
	}
	scene, err := NewLightScene(median, material)
	if err != nil {
		t.Fatal(err)
	}
	withForeign := PaintScene(scene, []SceneStroke{
		{Mode: SceneMaterial, Shape: ShapeSpot, X: 20, Y: 20, Radius: 5, Material: MatMetal},
	})
	// Go has no optional fields: TS's implicit gain/textureStrength of 1 must be explicit.
	redSetting := LightSetting{
		Channels:    []LightChannel{{Color: LightRed, Value: 600, Enable: true}},
		Reflectance: DefaultReflectance, Gain: 1, TextureStrength: 1,
	}
	blueSetting := LightSetting{
		Channels:    []LightChannel{{Color: LightBlue, Value: 600, Enable: true}},
		Reflectance: DefaultReflectance, Gain: 1, TextureStrength: 1,
	}
	redGolden := RenderChannel(scene, &redSetting, ChRed)
	redTest := RenderChannel(withForeign, &redSetting, ChRed)
	blueGolden := RenderChannel(scene, &blueSetting, ChBlue)
	blueTest := RenderChannel(withForeign, &blueSetting, ChBlue)
	idx := 20*40 + 20
	redDelta := math.Abs(float64(redTest.Data[idx]) - float64(redGolden.Data[idx]))
	blueDelta := math.Abs(float64(blueTest.Data[idx]) - float64(blueGolden.Data[idx]))
	if redDelta == blueDelta {
		t.Fatalf("redDelta=%v blueDelta=%v", redDelta, blueDelta)
	}
}

func TestRenderDefectPairNoStrokesMatchesGolden(t *testing.T) {
	scene := testUniformScene(30, 30, 3, 100)
	setting := testRedSetting(512, 1)
	golden, test := RenderDefectPair(scene, &setting, ChRed, nil, nil, NoNoise)
	if golden.Width != test.Width || golden.Height != test.Height {
		t.Fatal("size mismatch")
	}
	for i := range golden.Data {
		if golden.Data[i] != test.Data[i] {
			t.Fatalf("mismatch at %d: %d vs %d", i, golden.Data[i], test.Data[i])
		}
	}
}

func TestRenderDefectPairPaintsBeforeClamping(t *testing.T) {
	scene := testUniformScene(40, 40, 3, 100)
	setting := testRedSetting(1024, 3)
	golden := RenderChannel(scene, &setting, ChRed)
	if golden.Data[20*40+20] != 255 {
		t.Fatalf("golden=%d want 255", golden.Data[20*40+20])
	}
	_, test := RenderDefectPair(scene, &setting, ChRed, nil, []DefectStroke{
		{Kind: Dark, Shape: ShapeSpot, X: 20, Y: 20, Radius: 5, DeltaGv: 60, Softness: 0},
	}, NoNoise)
	if test.Data[20*40+20] != 255 {
		t.Fatalf("test=%d want 255", test.Data[20*40+20])
	}
}

func TestRelativeAmplitudeScalesWithIllumination(t *testing.T) {
	scene := testUniformScene(40, 40, 2, 100)
	stroke := DefectStroke{
		Kind: Bright, Shape: ShapeSpot, X: 20, Y: 20, Radius: 5,
		DeltaGv: 0, Softness: 0, Amplitude: AmpRelative, RelativePct: 50,
	}
	lowSetting := testRedSetting(512, 1)
	highSetting := testRedSetting(1024, 1)
	lowGolden, lowTest := RenderDefectPair(scene, &lowSetting, ChRed, nil, []DefectStroke{stroke}, NoNoise)
	highGolden, highTest := RenderDefectPair(scene, &highSetting, ChRed, nil, []DefectStroke{stroke}, NoNoise)
	idx := 20*40 + 20
	lowDelta := float64(lowTest.Data[idx]) - float64(lowGolden.Data[idx])
	highDelta := float64(highTest.Data[idx]) - float64(highGolden.Data[idx])
	if lowDelta <= 0 {
		t.Fatalf("lowDelta=%v", lowDelta)
	}
	if math.Abs(highDelta-2*lowDelta) > 2 {
		t.Fatalf("highDelta=%v lowDelta=%v", highDelta, lowDelta)
	}
}

func TestTargetAmplitudeClamps(t *testing.T) {
	dark := PaintDefects(testCreateGray(20, 20, 200), []DefectStroke{
		{Kind: Bright, Shape: ShapeSpot, X: 10, Y: 10, Radius: 4, DeltaGv: 0, Softness: 0, Amplitude: AmpTarget, TargetGv: 250, HasTarget: true},
	})
	if dark.Data[10*20+10] != 250 {
		t.Fatalf("dark=%d want 250", dark.Data[10*20+10])
	}
	bright := PaintDefects(testCreateGray(20, 20, 252), []DefectStroke{
		{Kind: Bright, Shape: ShapeSpot, X: 10, Y: 10, Radius: 4, DeltaGv: 0, Softness: 0, Amplitude: AmpTarget, TargetGv: 250, HasTarget: true},
	})
	if bright.Data[10*20+10] != 252 {
		t.Fatalf("bright=%d want 252", bright.Data[10*20+10])
	}
}

func TestPaintSceneLineFollowsSecondEndpoint(t *testing.T) {
	scene := testUniformScene(40, 40, 1, 100)
	painted := PaintScene(scene, []SceneStroke{
		{Mode: SceneTexture, Shape: ShapeLine, X: 5, Y: 5, X2: 30, Y2: 30, Radius: 2, Factor: 2},
	})
	onLine := painted.Texture[25*40+25]
	offLine := painted.Texture[5*40+25]
	base := scene.Texture[25*40+25]
	if float64(onLine)/float64(base) <= 1.9 {
		t.Fatalf("onLine=%v base=%v", onLine, base)
	}
	if offLine != scene.Texture[5*40+25] {
		t.Fatalf("offLine=%v want %v", offLine, scene.Texture[5*40+25])
	}
}

func TestRectStrokesFill(t *testing.T) {
	golden := testCreateGray(20, 20, 100)
	test := PaintDefects(golden, []DefectStroke{
		{Kind: Bright, Shape: ShapeRect, X: 5, Y: 5, X2: 14, Y2: 9, Radius: 0, DeltaGv: 50, Softness: 0},
	})
	if test.Data[7*20+7] != 150 {
		t.Fatalf("inside=%d want 150", test.Data[7*20+7])
	}
	if test.Data[2*20+2] != 100 {
		t.Fatalf("outside=%d want 100", test.Data[2*20+2])
	}
	changed := 0
	for i := range test.Data {
		if test.Data[i] != golden.Data[i] {
			changed++
		}
	}
	if changed != 50 {
		t.Fatalf("changed=%d want 50", changed)
	}
}

func TestRenderGoodSample(t *testing.T) {
	scene := testUniformScene(30, 30, 3, 100)
	setting := testRedSetting(512, 1)
	clean := RenderChannel(scene, &setting, ChRed)
	good := RenderGoodSample(scene, &setting, ChRed, NoNoise)
	if good.Width != clean.Width || good.Height != clean.Height {
		t.Fatal("size mismatch")
	}
	for i := range clean.Data {
		if good.Data[i] != clean.Data[i] {
			t.Fatalf("mismatch at %d: %d vs %d", i, good.Data[i], clean.Data[i])
		}
	}

	noise := SensorNoise{ReadSigma: 2, ShotGain: 0, Seed: 7}
	first := RenderGoodSample(scene, &setting, ChRed, noise)
	second := RenderGoodSample(scene, &setting, ChRed, noise)
	for i := range first.Data {
		if first.Data[i] != second.Data[i] {
			t.Fatalf("noise not deterministic at %d", i)
		}
	}
	differs := false
	for i := range first.Data {
		if first.Data[i] != good.Data[i] {
			differs = true
			break
		}
	}
	if !differs {
		t.Fatal("noisy image should differ from noise-free image")
	}
	other := RenderGoodSample(scene, &setting, ChRed, SensorNoise{ReadSigma: 2, ShotGain: 0, Seed: 8})
	same := true
	for i := range first.Data {
		if first.Data[i] != other.Data[i] {
			same = false
			break
		}
	}
	if same {
		t.Fatal("different seeds should differ")
	}
}

func TestGaussianProfileLowersBand(t *testing.T) {
	golden := testCreateGray(40, 40, 100)
	linear := PaintDefects(golden, []DefectStroke{
		{Kind: Bright, Shape: ShapeSpot, X: 20, Y: 20, Radius: 6, DeltaGv: 100, Softness: 0.5, Profile: ProfileLinear},
	})
	gaussian := PaintDefects(golden, []DefectStroke{
		{Kind: Bright, Shape: ShapeSpot, X: 20, Y: 20, Radius: 6, DeltaGv: 100, Softness: 0.5, Profile: ProfileGaussian},
	})
	center := 20*40 + 20
	if linear.Data[center] != gaussian.Data[center] {
		t.Fatalf("center mismatch %d vs %d", linear.Data[center], gaussian.Data[center])
	}
	band := 20*40 + 24
	if gaussian.Data[band] >= linear.Data[band] {
		t.Fatalf("gaussian=%d linear=%d", gaussian.Data[band], linear.Data[band])
	}
}

func TestGrainDeterministicAndInert(t *testing.T) {
	golden := testCreateGray(40, 40, 100)
	base := DefectStroke{Kind: Bright, Shape: ShapeSpot, X: 20, Y: 20, Radius: 8, DeltaGv: 80, Softness: 0.2}
	a := PaintDefects(golden, []DefectStroke{func() DefectStroke { s := base; s.Grain = 0.5; s.Seed = 7; return s }()})
	b := PaintDefects(golden, []DefectStroke{func() DefectStroke { s := base; s.Grain = 0.5; s.Seed = 7; return s }()})
	c := PaintDefects(golden, []DefectStroke{func() DefectStroke { s := base; s.Grain = 0.5; s.Seed = 8; return s }()})
	plain := PaintDefects(golden, []DefectStroke{base})
	for i := range a.Data {
		if a.Data[i] != b.Data[i] {
			t.Fatalf("a!=b at %d", i)
		}
	}
	same := true
	for i := range a.Data {
		if a.Data[i] != c.Data[i] {
			same = false
			break
		}
	}
	if same {
		t.Fatal("a==c should differ")
	}
	for i := range plain.Data {
		if plain.Data[i] != PaintDefects(golden, []DefectStroke{base}).Data[i] {
			t.Fatalf("plain mismatch at %d", i)
		}
	}
}

func TestSensorNoiseDeterministicAndScales(t *testing.T) {
	width := 40
	height := 40
	median := testCreateGray(width, height, 100)
	material := make([]uint8, width*height)
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			if x < width/2 {
				material[y*width+x] = 1
			} else {
				material[y*width+x] = 3
			}
		}
	}
	scene, err := NewLightScene(median, material)
	if err != nil {
		t.Fatal(err)
	}
	setting := testRedSetting(1024, 1)
	noise := SensorNoise{ReadSigma: 0, ShotGain: 1, Seed: 3}
	firstGolden, firstTest := RenderDefectPair(scene, &setting, ChRed, nil, nil, noise)
	_, secondTest := RenderDefectPair(scene, &setting, ChRed, nil, nil, noise)
	for i := range firstTest.Data {
		if firstTest.Data[i] != secondTest.Data[i] {
			t.Fatalf("noise not deterministic at %d", i)
		}
	}

	std := func(from, to int) float64 {
		sum := 0.0
		sumSq := 0.0
		count := 0
		for y := 0; y < height; y++ {
			for x := from; x < to; x++ {
				idx := y*width + x
				d := float64(firstTest.Data[idx]) - float64(firstGolden.Data[idx])
				sum += d
				sumSq += d * d
				count++
			}
		}
		mean := sum / float64(count)
		v := sumSq/float64(count) - mean*mean
		if v < 0 {
			v = 0
		}
		return math.Sqrt(v)
	}
	if !(std(0, width/2) < std(width/2, width)) {
		t.Fatal("noise should scale with signal")
	}
}

func TestGenerateRandomDefectsDeterministicAndConstrained(t *testing.T) {
	width := 60
	height := 60
	material := make([]uint8, width*height)
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			if x < width/2 {
				material[y*width+x] = 1
			} else {
				material[y*width+x] = 3
			}
		}
	}
	options := RandomDefectOptions{
		Count:         8,
		Seed:          42,
		Width:         width,
		Height:        height,
		Kind:          RandomMixedKind,
		Shape:         RandomMixedShape,
		RadiusMin:     2,
		RadiusMax:     6,
		ValueMin:      30,
		ValueMax:      80,
		Amplitude:     AmpAbsolute,
		Profile:       ProfileLinear,
		Softness:      0.3,
		Grain:         0,
		LineLengthMax: 60,
	}
	a := GenerateRandomDefects(options)
	b := GenerateRandomDefects(options)
	if len(a) != 8 {
		t.Fatalf("len=%d want 8", len(a))
	}
	if len(a) != len(b) {
		t.Fatal("length mismatch")
	}
	for i := range a {
		if !reflect.DeepEqual(a[i], b[i]) {
			t.Fatalf("stroke %d differs", i)
		}
	}
	for _, stroke := range a {
		if !(stroke.X >= stroke.Radius && stroke.X <= float64(width-1)-stroke.Radius) {
			t.Fatalf("x out of range: %v", stroke.X)
		}
		if !(stroke.Y >= stroke.Radius && stroke.Y <= float64(height-1)-stroke.Radius) {
			t.Fatalf("y out of range: %v", stroke.Y)
		}
	}

	constrained := GenerateRandomDefects(RandomDefectOptions{
		Count:         options.Count,
		Seed:          options.Seed,
		Width:         options.Width,
		Height:        options.Height,
		Kind:          options.Kind,
		Shape:         options.Shape,
		RadiusMin:     options.RadiusMin,
		RadiusMax:     options.RadiusMax,
		ValueMin:      options.ValueMin,
		ValueMax:      options.ValueMax,
		Amplitude:     options.Amplitude,
		Profile:       options.Profile,
		Softness:      options.Softness,
		Grain:         options.Grain,
		LineLengthMax: options.LineLengthMax,
		Material:      material,
		Materials:     []uint8{3},
	})
	if len(constrained) == 0 {
		t.Fatal("constrained empty")
	}
	for _, stroke := range constrained {
		ix := int(math.Round(stroke.X))
		iy := int(math.Round(stroke.Y))
		if material[iy*width+ix] != 3 {
			t.Fatalf("material=%d want 3", material[iy*width+ix])
		}
	}
}
