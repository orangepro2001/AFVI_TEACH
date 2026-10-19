package snap

import (
	"bytes"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

const xtXMLDecl = `<?xml version="1.0" encoding="UTF-8"?>`

// xtFullScenario 构造契约测试用的完整场景：3 个 GV 笔画（斑点并带颜色、多边形并自动、禁用矩形）、
// 2 个场景笔画（启用材质、禁用纹理）、非恒等 warp、带 crosstalk 的 4 通道光、非零 studio。
// Recipe 的缺陷/场景列表由 Strokes 中启用者按序推导（与 Meta() 一致）。
func xtFullScenario() *Scenario {
	ct := [4][3]float64{{1, 0, 0}, {0, 1, 0}, {0, 0, 1}, {1, 1, 1}}
	strokes := []StrokeEntry{
		{Enabled: true, GV: &Defect{Kind: 0, Shape: 0, X: 10.5, Y: 20.25, Radius: 4, DeltaGv: 30, Softness: 0.5, Amplitude: 0, Profile: 0, Grain: 0.1, Seed: 3, HasColor: true, Color: [3]float64{1, 0.5, 0}}},
		{Enabled: true, Auto: true, GV: &Defect{Kind: 1, Shape: 4, X: 5, Y: 5, X2: 9, Y2: 9, Radius: 2, DeltaGv: -20, Softness: 0.25, Amplitude: 2, TargetGv: 40, HasTarget: true, Profile: 1, Grain: 0.2, Seed: 9, Points: []Point{{X: 1, Y: 1}, {X: 8, Y: 1}, {X: 8, Y: 6}, {X: 1, Y: 6}}}},
		{Enabled: false, GV: &Defect{Kind: 0, Shape: 2, X: 0, Y: 0, X2: 4, Y2: 4, Radius: 1, DeltaGv: 5, Softness: 0, Amplitude: 0, Profile: 0, Grain: 0, Seed: 0}},
		{Enabled: true, Scene: &SceneStroke{Mode: 0, Shape: 2, X: 2, Y: 3, X2: 12, Y2: 9, Radius: 1, Material: 0, Factor: 1.5}},
		{Enabled: false, Scene: &SceneStroke{Mode: 1, Shape: 3, X: 1, Y: 1, X2: 3, Y2: 3, Radius: 0.5, Material: 2, Factor: 0.5}},
	}
	defects, scenes := xtDeriveRecipeByStrokes(strokes)

	s := &Scenario{}
	s.Producer = "teach-prod"
	s.Created = "2026-03-05T12:00:00Z"
	s.Note = "full scenario for xml round trip"
	s.Model = ModelRef{ID: "FM1/25fbo015-15", Host: "host-1", Model: "25fbo015-15", Side: "front", Version: "1.0"}
	s.Width = 512
	s.Height = 384
	s.Recipe = Recipe{}
	s.Recipe.Product = Product{Enabled: true, Seed: 1, Strength: 1}
	s.Recipe.Product.Materials = [4]Material{
		{Sigma: 0.03, Length: 32, Grain: 0.02},
		{Sigma: 0.10, Length: 24, Grain: 0.06},
		{Sigma: 0.04, Length: 32, Grain: 0.03},
		{Sigma: 0.04, Length: 20, Grain: 0.03},
	}
	s.Recipe.Light = Light{}
	s.Recipe.Light.Channels = []LightChannel{
		{Color: 0, Value: 512, Enable: true},
		{Color: 1, Value: 128, Enable: true},
		{Color: 2, Value: 64, Enable: true},
		{Color: 3, Value: 256, Enable: false},
	}
	s.Recipe.Light.Reflectance = [4][3]float64{{0.05, 0.05, 0.05}, {0.12, 0.35, 0.15}, {0.5, 0.45, 0.35}, {0.9, 0.7, 0.45}}
	s.Recipe.Light.Crosstalk = &ct
	s.Recipe.Light.Gain = 1.2
	s.Recipe.Light.Offset = 0.1
	s.Recipe.Light.TextureStrength = 0.9
	s.Recipe.Noise = Noise{ReadSigma: 1.5, ShotGain: 0.25, Seed: 7}
	s.Recipe.Warp = &Warp{Tx: 12.5, Ty: -7, ThetaDeg: 0.6, Scale: 1.01, ShearX: 0.01, ShearY: -0.02, TiltX: 0.12, TiltY: -0.08, BendAmplitude: 9, BendWavelength: 1100}
	s.Recipe.Defects = defects
	s.Recipe.SceneStrokes = scenes
	s.Strokes = strokes
	s.Studio = &Studio{}
	s.Studio.View = "diff"
	s.Studio.Channel = "blue"
	s.Studio.ShowMarks = true
	s.Studio.Tool = 2
	s.Studio.Brush = StudioBrush{Radius: 4, DeltaGv: 20, Softness: 0.5, Amplitude: 1, RelativePct: 30, TargetBright: 200, TargetDark: 55, Profile: 1, Grain: 0.1, ForeignMaterial: 1, ContaminationFact: 0.5, UseColor: true, Color: [3]float64{0.5, 0.25, 0.75}}
	s.Studio.Random = StudioRandom{Count: 5, Seed: 42, Kind: 1, Shape: 0, RadiusMin: 2, RadiusMax: 6, ValueMin: 10, ValueMax: 200, Material: 1}
	s.Studio.Auto = StudioAuto{Count: 10, Seed: 1, StrokeSeed: 2, ProductLenScale: 1.5, ProductDraw: 1}
	s.Studio.Output = StudioOutput{Dir: "captures/FM1-25fbo015-15", Channels: [3]bool{true, false, true}, Good: true, RecipeOnly: false}
	return s
}

// xtDeriveRecipeByStrokes 按 Strokes 中启用笔画推导 Recipe 的缺陷与场景列表（与 Meta() 一致；切片非 nil）。
func xtDeriveRecipeByStrokes(strokes []StrokeEntry) (defects []Defect, scenes []SceneStroke) {
	defects = []Defect{}
	scenes = []SceneStroke{}
	for _, e := range strokes {
		if !e.Enabled {
			continue
		}
		if e.GV != nil {
			d := *e.GV
			if len(d.Points) > 0 {
				d.Points = append([]Point(nil), d.Points...)
			}
			defects = append(defects, d)
		} else if e.Scene != nil {
			scenes = append(scenes, *e.Scene)
		}
	}
	return
}

// xtRemoveBlock 去掉 data 中第一个以 open 开始、以 close 结束的连续片段（用于从合法 XML 里“删除”元素）。
func xtRemoveBlock(data []byte, open, close string) []byte {
	i := bytes.Index(data, []byte(open))
	if i < 0 {
		return data
	}
	j := bytes.Index(data[i:], []byte(close))
	if j < 0 {
		return data
	}
	j += i + len(close)
	out := make([]byte, 0, len(data)-(j-i))
	out = append(out, data[:i]...)
	out = append(out, data[j:]...)
	return out
}

func TestScenarioMarshalUnmarshalRoundTrip(t *testing.T) {
	want := xtFullScenario()
	data, err := MarshalScenario(want)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	// 契约：XML 声明头、两空格缩进、以换行结尾
	if !bytes.HasPrefix(data, []byte(xtXMLDecl)) {
		t.Errorf("missing xml declaration, head=%q", data[:60])
	}
	if !bytes.HasSuffix(data, []byte("\n")) {
		t.Errorf("xml does not end with newline")
	}
	if !bytes.Contains(data, []byte("\n  ")) {
		t.Errorf("no two-space indent found")
	}
	got, err := UnmarshalScenario(data)
	if err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !reflect.DeepEqual(*got, *want) {
		t.Errorf("round trip mismatch:\n got %+v\nwant %+v", *got, *want)
	}
	// Recipe 的笔画列表只含启用者：2 个 GV（索引 0、1）+ 1 个场景（索引 3）
	if len(got.Recipe.Defects) != 2 {
		t.Errorf("got %d defects, want 2 (enabled GV only)", len(got.Recipe.Defects))
	}
	if len(got.Recipe.SceneStrokes) != 1 {
		t.Errorf("got %d scene strokes, want 1 (enabled scene only)", len(got.Recipe.SceneStrokes))
	}
	// Strokes 完整保留，含禁用与自动
	if len(got.Strokes) != 5 {
		t.Errorf("got %d strokes, want 5 (all kept)", len(got.Strokes))
	}
}

func TestScenarioStrokesOrder(t *testing.T) {
	want := xtFullScenario()
	data, err := MarshalScenario(want)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	got, err := UnmarshalScenario(data)
	if err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	// GV 与 Scene 的交错顺序不变（与原始 Strokes 一一对应）
	if len(got.Strokes) != len(want.Strokes) {
		t.Fatalf("stroke count %d != %d", len(got.Strokes), len(want.Strokes))
	}
	for i := range want.Strokes {
		ge, gs := got.Strokes[i].GV != nil, got.Strokes[i].Scene != nil
		we, ws := want.Strokes[i].GV != nil, want.Strokes[i].Scene != nil
		if ge != we || gs != ws {
			t.Errorf("stroke %d: got gv=%v scene=%v, want gv=%v scene=%v", i, ge, gs, we, ws)
		}
		if !reflect.DeepEqual(got.Strokes[i], want.Strokes[i]) {
			t.Errorf("stroke %d mismatch: got %+v, want %+v", i, got.Strokes[i], want.Strokes[i])
		}
	}
}

func TestScenarioWriteReadFile(t *testing.T) {
	dir := t.TempDir()
	want := xtFullScenario()
	xmlPath := filepath.Join(dir, "scenario.xml")
	gzPath := filepath.Join(dir, "scenario.xml.gz")
	if err := WriteScenario(xmlPath, want); err != nil {
		t.Fatalf("write xml: %v", err)
	}
	if err := WriteScenario(gzPath, want); err != nil {
		t.Fatalf("write gz: %v", err)
	}
	backXML, err := ReadScenario(xmlPath)
	if err != nil {
		t.Fatalf("read xml: %v", err)
	}
	if !reflect.DeepEqual(*backXML, *want) {
		t.Errorf("xml round trip mismatch:\n got %+v\nwant %+v", *backXML, *want)
	}
	backGZ, err := ReadScenario(gzPath)
	if err != nil {
		t.Fatalf("read gz: %v", err)
	}
	if !reflect.DeepEqual(*backGZ, *want) {
		t.Errorf("gz round trip mismatch:\n got %+v\nwant %+v", *backGZ, *want)
	}
	xi, err := os.Stat(xmlPath)
	if err != nil {
		t.Fatalf("stat xml: %v", err)
	}
	gi, err := os.Stat(gzPath)
	if err != nil {
		t.Fatalf("stat gz: %v", err)
	}
	// gzip 文件应小于对应缩进 XML
	if gi.Size() >= xi.Size() {
		t.Errorf("gz %d bytes not smaller than xml %d bytes", gi.Size(), xi.Size())
	}
	// 目录路径读其中的 scenario.xml
	backDir, err := ReadScenario(dir)
	if err != nil {
		t.Fatalf("read dir: %v", err)
	}
	if !reflect.DeepEqual(*backDir, *want) {
		t.Errorf("directory round trip mismatch:\n got %+v\nwant %+v", *backDir, *want)
	}
}

func TestScenarioEnumNumericAndUnknown(t *testing.T) {
	want := xtFullScenario()
	data, err := MarshalScenario(want)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	// 数字写法：kind="1" shape="3"（dark / ellipse），第一个笔画原为 bright spot
	num := bytes.Replace(data, []byte(`kind="bright"`), []byte(`kind="1"`), 1)
	num = bytes.Replace(num, []byte(`shape="spot"`), []byte(`shape="3"`), 1)
	dec, err := UnmarshalScenario(num)
	if err != nil {
		t.Fatalf("numeric enum unmarshal: %v", err)
	}
	d := dec.Strokes[0].GV
	if d == nil {
		t.Errorf("numeric enum: gv is nil, want kind=1 shape=3")
	} else if d.Kind != 1 || d.Shape != 3 {
		t.Errorf("numeric enum: got kind=%d shape=%d, want kind=1 shape=3", d.Kind, d.Shape)
	}
	// 未知名字报错，且错误信息含属性名 kind
	unk := bytes.Replace(data, []byte(`kind="bright"`), []byte(`kind="purple"`), 1)
	if _, err := UnmarshalScenario(unk); err == nil {
		t.Fatalf("expected error for unknown kind name")
	} else if !strings.Contains(err.Error(), "kind") {
		t.Errorf("error should name the kind attribute, got: %v", err)
	}
}

func TestScenarioForwardCompat(t *testing.T) {
	want := xtFullScenario()
	data, err := MarshalScenario(want)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	// 多余的未知属性与未知子元素被忽略
	withExtra := bytes.Replace(data, []byte(`format="afvi-scenario"`), []byte(`format="afvi-scenario" xt-unknown="1"`), 1)
	withExtra = bytes.Replace(withExtra, []byte(`<model `), []byte(`<xt-unknown-el xt-attr="2"/><model `), 1)
	dec, err := UnmarshalScenario(withExtra)
	if err != nil {
		t.Fatalf("unknown element/attribute should be ignored: %v", err)
	}
	if !reflect.DeepEqual(*dec, *want) {
		t.Errorf("forward compat mismatch:\n got %+v\nwant %+v", *dec, *want)
	}
	// 缺 <studio> -> Studio==nil
	noStudio := xtFullScenario()
	noStudio.Studio = nil
	dataNS, err := MarshalScenario(noStudio)
	if err != nil {
		t.Fatalf("marshal no studio: %v", err)
	}
	decNS, err := UnmarshalScenario(dataNS)
	if err != nil {
		t.Fatalf("unmarshal no studio: %v", err)
	}
	if decNS.Studio != nil {
		t.Errorf("missing studio should decode to Studio==nil, got %+v", decNS.Studio)
	}
	// 缺 <warp> -> Recipe.Warp==nil
	noWarp := xtFullScenario()
	noWarp.Recipe.Warp = nil
	dataNW, err := MarshalScenario(noWarp)
	if err != nil {
		t.Fatalf("marshal no warp: %v", err)
	}
	decNW, err := UnmarshalScenario(dataNW)
	if err != nil {
		t.Fatalf("unmarshal no warp: %v", err)
	}
	if decNW.Recipe.Warp != nil {
		t.Errorf("missing warp should decode to Recipe.Warp==nil, got %+v", decNW.Recipe.Warp)
	}
}

func TestScenarioErrors(t *testing.T) {
	// 非 XML
	if _, err := UnmarshalScenario([]byte("this is not xml")); err == nil {
		t.Fatalf("expected error for non-xml")
	}
	data, err := MarshalScenario(xtFullScenario())
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	// format 不对
	bad := bytes.Replace(data, []byte(`format="afvi-scenario"`), []byte(`format="bogus"`), 1)
	if _, err := UnmarshalScenario(bad); err == nil {
		t.Fatalf("expected error for bad format")
	}
	// version="2"
	bad = bytes.Replace(data, []byte(` version="1"`), []byte(` version="2"`), 1)
	if _, err := UnmarshalScenario(bad); err == nil {
		t.Fatalf("expected error for version 2")
	}
	// 缺 <light>
	nolight := xtRemoveBlock(data, `<light `, `</light>`)
	if _, err := UnmarshalScenario(nolight); err == nil {
		t.Fatalf("expected error for missing light")
	}
	// 多边形只有两个顶点
	s := xtFullScenario()
	s.Strokes[1].GV.Points = s.Strokes[1].GV.Points[:2]
	s.Recipe.Defects[1].Points = s.Recipe.Defects[1].Points[:2]
	if _, err := MarshalScenario(s); err == nil {
		t.Fatalf("expected error for polygon with 2 points")
	}
	// Strokes 项 GV 与 Scene 都为 nil
	s2 := xtFullScenario()
	s2.Strokes = append(s2.Strokes, StrokeEntry{Enabled: true})
	if _, err := MarshalScenario(s2); err == nil {
		t.Fatalf("expected error for stroke with both gv and scene nil")
	}
}

func TestScenarioHooks(t *testing.T) {
	dir := t.TempDir()
	s := xtFullScenario()
	meta := s.Meta()

	// Unmarshal 对 XML 字节直接可用，返回 Scenario.Meta()
	xmlBytes, err := MarshalScenario(s)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	m, err := Unmarshal(xmlBytes)
	if err != nil {
		t.Fatalf("unmarshal xml bytes: %v", err)
	}
	if !reflect.DeepEqual(*m, meta) {
		t.Errorf("unmarshal(xml) != meta():\n got %+v\nwant %+v", *m, meta)
	}
	if m.Images == nil || len(m.Images) != 0 {
		t.Errorf("meta from xml should have empty non-nil images, got %v", m.Images)
	}

	// WriteMeta/ReadMeta 对 .xml 与 .xml.gz：Recipe/Model/Size 相等，Images 为空切片
	for _, ext := range []string{".xml", ".xml.gz"} {
		path := filepath.Join(dir, "meta"+ext)
		if err := WriteMeta(path, &meta); err != nil {
			t.Fatalf("write meta %s: %v", ext, err)
		}
		back, _, err := ReadMeta(path)
		if err != nil {
			t.Fatalf("read meta %s: %v", ext, err)
		}
		if !reflect.DeepEqual(back.Recipe, meta.Recipe) {
			t.Errorf("meta %s recipe mismatch:\n got %+v\nwant %+v", ext, back.Recipe, meta.Recipe)
		}
		if back.Model != meta.Model {
			t.Errorf("meta %s model mismatch: got %+v want %+v", ext, back.Model, meta.Model)
		}
		if back.Width != meta.Width || back.Height != meta.Height {
			t.Errorf("meta %s size: got %dx%d want %dx%d", ext, back.Width, back.Height, meta.Width, meta.Height)
		}
		if back.Images == nil || len(back.Images) != 0 {
			t.Errorf("meta %s images: got %v, want empty non-nil", ext, back.Images)
		}
	}

	// .json 与 .json.gz 行为不变
	jsonMeta := xtValidMeta()
	for _, ext := range []string{".json", ".json.gz"} {
		path := filepath.Join(dir, "meta"+ext)
		if err := WriteMeta(path, jsonMeta); err != nil {
			t.Fatalf("write json meta %s: %v", ext, err)
		}
		back, _, err := ReadMeta(path)
		if err != nil {
			t.Fatalf("read json meta %s: %v", ext, err)
		}
		if !reflect.DeepEqual(*back, *jsonMeta) {
			t.Errorf("json %s round trip mismatch:\n got %+v\nwant %+v", ext, *back, *jsonMeta)
		}
	}

	// Meta() 返回的笔画切片与 Scenario 不共享内存
	m2 := s.Meta()
	if len(m2.Recipe.Defects) == 0 {
		t.Fatalf("Meta() has no defects")
	}
	var origGV Defect
	for _, e := range s.Strokes {
		if e.GV != nil && e.Enabled {
			origGV = *e.GV
			break
		}
	}
	m2.Recipe.Defects[0] = Defect{}
	for _, e := range s.Strokes {
		if e.GV != nil && e.Enabled {
			if !reflect.DeepEqual(*e.GV, origGV) {
				t.Errorf("Meta() defects share memory with Scenario: modifying copy affected original")
			}
			break
		}
	}
}

func TestScenarioFloatBits(t *testing.T) {
	s := xtFullScenario()
	// 将特殊浮点值分布到不同字段，往返后逐 bit 比较
	s.Strokes[0].GV.DeltaGv = 0.1
	s.Strokes[0].GV.Softness = 1.0 / 3
	s.Strokes[0].GV.Radius = math.Copysign(0, -1) // -0.0
	s.Strokes[3].Scene.Factor = 1e-300
	s.Recipe.Product.Materials[0].Sigma = 123456789.123456789
	s.Recipe.Light.Gain = 0.1
	s.Recipe.Noise.ReadSigma = 1.0 / 3
	s.Studio.Brush.Radius = math.Copysign(0, -1)
	s.Studio.Brush.TargetBright = 1e-300

	data, err := MarshalScenario(s)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	got, err := UnmarshalScenario(data)
	if err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	check := func(name string, gotBits, wantBits uint64) {
		if gotBits != wantBits {
			t.Errorf("float bits %s: got %#x want %#x", name, gotBits, wantBits)
		}
	}
	check("defect deltaGv", math.Float64bits(got.Strokes[0].GV.DeltaGv), math.Float64bits(0.1))
	check("defect softness", math.Float64bits(got.Strokes[0].GV.Softness), math.Float64bits(1.0/3))
	check("defect radius -0.0", math.Float64bits(got.Strokes[0].GV.Radius), math.Float64bits(math.Copysign(0, -1)))
	check("scene factor", math.Float64bits(got.Strokes[3].Scene.Factor), math.Float64bits(1e-300))
	check("product sigma", math.Float64bits(got.Recipe.Product.Materials[0].Sigma), math.Float64bits(123456789.123456789))
	check("light gain", math.Float64bits(got.Recipe.Light.Gain), math.Float64bits(0.1))
	check("noise readSigma", math.Float64bits(got.Recipe.Noise.ReadSigma), math.Float64bits(1.0/3))
	check("brush radius -0.0", math.Float64bits(got.Studio.Brush.Radius), math.Float64bits(math.Copysign(0, -1)))
	check("brush targetBright", math.Float64bits(got.Studio.Brush.TargetBright), math.Float64bits(1e-300))
}

func TestScenarioLooksLikeXML(t *testing.T) {
	cases := []struct {
		data []byte
		want bool
	}{
		{[]byte(xtXMLDecl + "<x/>"), true},
		{[]byte("<x/>"), true},
		{[]byte("  <x/>"), true},
		{[]byte("\t\n<x/>"), true},
		{[]byte("\xef\xbb\xbf<x/>"), true},    // UTF-8 BOM
		{[]byte("  \xef\xbb\xbf <x/>"), true}, // BOM 与空白混合
		{[]byte("{\"x\":1}"), false},
		{[]byte(""), false},
		{[]byte("   "), false},
		{[]byte("x<y"), false},
	}
	for _, c := range cases {
		if got := LooksLikeXML(c.data); got != c.want {
			t.Errorf("LooksLikeXML(%q) = %v, want %v", c.data, got, c.want)
		}
	}
}

// xtValidMeta 构造可通过 Validate 的最小 meta（用于 json 回归）。
func xtValidMeta() *Meta {
	m := &Meta{}
	m.Format = Format
	m.Version = Version
	m.Producer = "p"
	m.Created = "2026-01-01T00:00:00Z"
	m.Model = ModelRef{ID: "m", Host: "h", Model: "x", Side: "s", Version: "1"}
	m.Width = 8
	m.Height = 8
	m.Recipe = Recipe{}
	m.Recipe.Product = Product{Enabled: true, Seed: 1, Strength: 1}
	m.Recipe.Light = Light{}
	m.Recipe.Light.Channels = []LightChannel{{Color: 0, Value: 512, Enable: true}}
	m.Recipe.Light.Reflectance = [4][3]float64{{0, 0, 0}, {0, 0, 0}, {0, 0, 0}, {0, 0, 0}}
	m.Recipe.Light.Gain = 1
	m.Recipe.Light.Offset = 0
	m.Recipe.Light.TextureStrength = 1
	m.Recipe.Noise = Noise{Seed: 1}
	m.Images = []Image{{Channel: ChannelRed, Kind: KindTest, File: "red.png", Width: 8, Height: 8, SHA256: strings.Repeat("a", 64)}}
	return m
}
