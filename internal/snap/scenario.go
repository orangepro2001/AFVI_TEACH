package snap

import "afvi/teach-imgui/internal/core"

// 场景文件（afvi-scenario，XML）：相机工作室把调好的所有信息高密度地存成一个文件，
// 下次直接读取；教学程序读取它时不需要 camera.exe（在进程内按配方渲染）。
// XML 的编码解码在 xml.go。

// 场景文件的格式标识、版本和默认文件名。
const (
	ScenarioFormat  = "afvi-scenario"
	ScenarioVersion = 1
	ScenarioFile    = "scenario.xml" // 默认文件名
)

// Scenario 是场景文件的内容。
type Scenario struct {
	Producer string
	Created  string // RFC 3339, UTC
	Note     string
	Model    ModelRef
	Width    int
	Height   int
	// Recipe 是能渲染的部分。解码后 Recipe.Defects / Recipe.SceneStrokes 一律由 Strokes 中"启用"的笔画
	// 按顺序推导（GV 笔画进 Defects，场景笔画进 SceneStrokes）；编码时同样以 Strokes 为准。
	Recipe  Recipe
	Strokes []StrokeEntry // 全部笔画（含禁用、自动生成的），保持面板顺序
	Studio  *Studio       // 相机工作室的界面状态；非工作室生成的文件为 nil
}

// StrokeEntry 是一条笔画及其状态。GV 与 Scene 恰有一个非 nil。
type StrokeEntry struct {
	Enabled bool
	Auto    bool // 加载型号时自动生成的默认小缺陷
	GV      *Defect
	Scene   *SceneStroke
}

// Studio 是相机工作室的界面状态（教学程序读取时忽略）。零值字段表示默认。
type Studio struct {
	View      string // "test" | "golden" | "diff"
	Channel   string // "red" | "green" | "blue"
	ShowMarks bool
	Tool      int
	Brush     StudioBrush
	Random    StudioRandom
	Auto      StudioAuto
	Output    StudioOutput
}

// StudioBrush 是笔刷设置。
type StudioBrush struct {
	Radius, DeltaGv, Softness             float64
	Amplitude                             int // 0 absolute, 1 relative, 2 target
	RelativePct, TargetBright, TargetDark float64
	Profile                               int // 0 linear, 1 gaussian
	Grain                                 float64
	ForeignMaterial                       int
	ContaminationFact                     float64
	UseColor                              bool
	Color                                 [3]float64
}

// StudioRandom 是随机缺陷生成器的设置。
type StudioRandom struct {
	Count                                    int
	Seed                                     uint32
	Kind, Shape                              int
	RadiusMin, RadiusMax, ValueMin, ValueMax float64
	Material                                 int // -1 = 全部材质
}

// StudioAuto 是默认小缺陷、笔画种子与虚拟良品绘制的设置。
type StudioAuto struct {
	Count           int
	Seed            uint32
	StrokeSeed      uint32
	ProductLenScale float64
	ProductDraw     int
}

// StudioOutput 是输出栏的设置。
type StudioOutput struct {
	Dir        string
	Channels   [3]bool // red, green, blue
	Good       bool
	RecipeOnly bool
}

// Meta 返回等价的抓拍描述：Format/Version 填好，Producer/Created/Note/Model/Width/Height/Recipe 取自场景，
// Images 为空的非 nil 切片。Recipe 的笔画列表是 Strokes 中启用的那些（复制，不与 Scenario 共享内存）。
func (s *Scenario) Meta() Meta {
	r := s.Recipe
	r.Light.Channels = append([]LightChannel{}, s.Recipe.Light.Channels...)
	if s.Recipe.Light.Crosstalk != nil {
		ct := *s.Recipe.Light.Crosstalk
		r.Light.Crosstalk = &ct
	}
	if s.Recipe.Warp != nil {
		w := *s.Recipe.Warp
		r.Warp = &w
	}
	r.Defects = []Defect{}
	r.SceneStrokes = []SceneStroke{}
	for _, e := range s.Strokes {
		if !e.Enabled {
			continue
		}
		if e.GV != nil {
			d := *e.GV
			if len(d.Points) > 0 {
				d.Points = append([]Point(nil), d.Points...)
			}
			r.Defects = append(r.Defects, d)
		} else if e.Scene != nil {
			r.SceneStrokes = append(r.SceneStrokes, *e.Scene)
		}
	}
	return Meta{
		Format:   Format,
		Version:  Version,
		Producer: s.Producer,
		Created:  s.Created,
		Note:     s.Note,
		Model:    s.Model,
		Width:    s.Width,
		Height:   s.Height,
		Recipe:   r,
		Images:   []Image{},
	}
}

// ScenarioFromMeta 由抓拍描述构造场景：每条 Recipe 笔画成为一个 Enabled=true、Auto=false 的 StrokeEntry
// （先全部 GV 再全部 Scene），Studio 为 nil。
func ScenarioFromMeta(m Meta) *Scenario {
	s := &Scenario{
		Producer: m.Producer,
		Created:  m.Created,
		Note:     m.Note,
		Model:    m.Model,
		Width:    m.Width,
		Height:   m.Height,
		Recipe:   m.Recipe,
		Strokes:  make([]StrokeEntry, 0, len(m.Recipe.Defects)+len(m.Recipe.SceneStrokes)),
	}
	s.Recipe.Light.Channels = append([]LightChannel{}, m.Recipe.Light.Channels...)
	if m.Recipe.Light.Crosstalk != nil {
		ct := *m.Recipe.Light.Crosstalk
		s.Recipe.Light.Crosstalk = &ct
	}
	if m.Recipe.Warp != nil {
		w := *m.Recipe.Warp
		s.Recipe.Warp = &w
	}
	for i := range m.Recipe.Defects {
		d := m.Recipe.Defects[i]
		if len(d.Points) > 0 {
			d.Points = append([]Point(nil), d.Points...)
		}
		s.Strokes = append(s.Strokes, StrokeEntry{Enabled: true, GV: &d})
	}
	for i := range m.Recipe.SceneStrokes {
		sc := m.Recipe.SceneStrokes[i]
		s.Strokes = append(s.Strokes, StrokeEntry{Enabled: true, Scene: &sc})
	}
	s.Recipe.Defects = []Defect{}
	s.Recipe.SceneStrokes = []SceneStroke{}
	for _, e := range s.Strokes {
		if e.GV != nil {
			s.Recipe.Defects = append(s.Recipe.Defects, *e.GV)
		} else {
			s.Recipe.SceneStrokes = append(s.Recipe.SceneStrokes, *e.Scene)
		}
	}
	return s
}

// WireDefect 把模拟缺陷笔画转成文件形式（深拷贝）。
func WireDefect(d core.DefectStroke) Defect { return defectToWire(d) }

// CoreDefect 把文件形式的缺陷笔画转回模拟类型（深拷贝）。
func CoreDefect(d Defect) core.DefectStroke { return defectFromWire(d) }

// WireScene 把模拟场景笔画转成文件形式。
func WireScene(s core.SceneStroke) SceneStroke {
	return SceneStroke{
		Mode:     int(s.Mode),
		Shape:    int(s.Shape),
		X:        s.X,
		Y:        s.Y,
		X2:       s.X2,
		Y2:       s.Y2,
		Radius:   s.Radius,
		Material: int(s.Material),
		Factor:   s.Factor,
	}
}

// CoreScene 把文件形式的场景笔画转回模拟类型。
func CoreScene(s SceneStroke) core.SceneStroke {
	return core.SceneStroke{
		Mode:     core.SceneMode(s.Mode),
		Shape:    core.DefectShape(s.Shape),
		X:        s.X,
		Y:        s.Y,
		X2:       s.X2,
		Y2:       s.Y2,
		Radius:   s.Radius,
		Material: uint8(s.Material),
		Factor:   s.Factor,
	}
}
