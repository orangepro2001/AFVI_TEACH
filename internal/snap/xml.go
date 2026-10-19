package snap

import (
	"bytes"
	"compress/gzip"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"afvi/teach-imgui/internal/core"
)

// 本文件实现场景文件（afvi-scenario，XML）的编码与解码。类型定义见 scenario.go。

// LooksLikeXML 报告 data 去掉 UTF-8 BOM 与前导空白后是否以 '<' 开头。
func LooksLikeXML(data []byte) bool {
	for len(data) > 0 {
		switch {
		case bytes.HasPrefix(data, []byte{0xEF, 0xBB, 0xBF}):
			data = data[3:]
		case data[0] == ' ' || data[0] == '\t' || data[0] == '\r' || data[0] == '\n':
			data = data[1:]
		default:
			return data[0] == '<'
		}
	}
	return false
}

// MarshalScenario 校验 s 并返回带 XML 声明、两空格缩进、结尾换行的 XML。
func MarshalScenario(s *Scenario) ([]byte, error) {
	if err := validateScenario(s); err != nil {
		return nil, err
	}
	root := scenarioToXML(s)
	var buf bytes.Buffer
	buf.WriteString(xml.Header)
	enc := xml.NewEncoder(&buf)
	enc.Indent("", "  ")
	if err := enc.Encode(root); err != nil {
		return nil, fmt.Errorf("snap: xml: %w", err)
	}
	if err := enc.Flush(); err != nil {
		return nil, fmt.Errorf("snap: xml: %w", err)
	}
	buf.WriteByte('\n')
	return buf.Bytes(), nil
}

// UnmarshalScenario 解析场景文件；以 gzip 魔数开头则先解压；结果经过校验。
func UnmarshalScenario(data []byte) (*Scenario, error) {
	if len(data) >= 2 && data[0] == 0x1f && data[1] == 0x8b {
		zr, err := gzip.NewReader(bytes.NewReader(data))
		if err != nil {
			return nil, fmt.Errorf("snap: gzip: %w", err)
		}
		defer zr.Close()
		plain, err := io.ReadAll(zr)
		if err != nil {
			return nil, fmt.Errorf("snap: gzip: %w", err)
		}
		data = plain
	}
	var root xmlScenario
	if err := xml.Unmarshal(data, &root); err != nil {
		return nil, fmt.Errorf("snap: xml: %w", err)
	}
	if root.XMLName.Local != "afvi-scenario" {
		return nil, fmt.Errorf("snap: xml: root element %q, want %q", root.XMLName.Local, "afvi-scenario")
	}
	if root.Format != ScenarioFormat {
		return nil, fmt.Errorf("snap: bad format %q, want %q", root.Format, ScenarioFormat)
	}
	if root.Version <= 0 || root.Version > ScenarioVersion {
		return nil, fmt.Errorf("snap: unsupported version %d, want 1..%d", root.Version, ScenarioVersion)
	}
	s, err := scenarioFromXML(&root)
	if err != nil {
		return nil, err
	}
	if err := validateScenario(s); err != nil {
		return nil, err
	}
	return s, nil
}

// WriteScenario 原子写入场景文件。路径以 ".gz" 结尾时写紧凑 XML 并 gzip 压缩。
func WriteScenario(path string, s *Scenario) error {
	if err := validateScenario(s); err != nil {
		return err
	}
	var data []byte
	if strings.HasSuffix(strings.ToLower(path), ".gz") {
		root := scenarioToXML(s)
		var buf bytes.Buffer
		enc := xml.NewEncoder(&buf)
		if err := enc.Encode(root); err != nil {
			return fmt.Errorf("snap: xml: %w", err)
		}
		if err := enc.Flush(); err != nil {
			return fmt.Errorf("snap: xml: %w", err)
		}
		var zbuf bytes.Buffer
		zw, err := gzip.NewWriterLevel(&zbuf, gzip.BestCompression)
		if err != nil {
			return err
		}
		if _, err := zw.Write(buf.Bytes()); err != nil {
			zw.Close()
			return err
		}
		if err := zw.Close(); err != nil {
			return err
		}
		data = zbuf.Bytes()
	} else {
		indented, err := MarshalScenario(s)
		if err != nil {
			return err
		}
		data = indented
	}
	return writeFileAtomic(path, data)
}

// ReadScenario 读取场景文件；path 是目录时读取其中的 ScenarioFile。
func ReadScenario(path string) (*Scenario, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	file := path
	if info.IsDir() {
		file = filepath.Join(path, ScenarioFile)
	}
	data, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}
	return UnmarshalScenario(data)
}

// validateScenario 校验场景：基础校验来自 Meta().Validate()，另加笔画与工作室状态检查。
func validateScenario(s *Scenario) error {
	if s == nil {
		return fmt.Errorf("snap: nil scenario")
	}
	m := s.Meta()
	if err := m.Validate(); err != nil {
		return err
	}
	for i, e := range s.Strokes {
		if (e.GV == nil) == (e.Scene == nil) {
			return fmt.Errorf("snap: stroke %d: exactly one of GV/Scene must be set", i)
		}
		if e.GV != nil {
			if err := validateDefect(*e.GV); err != nil {
				return fmt.Errorf("snap: stroke %d: %w", i, err)
			}
		}
		if e.Scene != nil {
			sc := e.Scene
			if sc.Mode < 0 || sc.Mode > 1 {
				return fmt.Errorf("snap: stroke %d: bad mode %d", i, sc.Mode)
			}
			if sc.Shape < 0 || sc.Shape > 4 {
				return fmt.Errorf("snap: stroke %d: bad shape %d", i, sc.Shape)
			}
			if sc.Material < 0 || sc.Material > 3 {
				return fmt.Errorf("snap: stroke %d: bad material %d", i, sc.Material)
			}
		}
	}
	if st := s.Studio; st != nil {
		switch st.Channel {
		case "", ChannelRed, ChannelGreen, ChannelBlue:
		default:
			return fmt.Errorf("snap: studio: bad channel %q", st.Channel)
		}
		switch st.View {
		case "", "test", "golden", "diff":
		default:
			return fmt.Errorf("snap: studio: bad view %q", st.View)
		}
	}
	return nil
}

// ---- XML 结构 ----

type xmlScenario struct {
	XMLName  xml.Name    `xml:"afvi-scenario"`
	Format   string      `xml:"format,attr"`
	Version  int         `xml:"version,attr"`
	Producer string      `xml:"producer,attr"`
	Created  string      `xml:"created,attr"`
	Note     string      `xml:"note,attr"`
	Model    xmlModel    `xml:"model"`
	Product  *xmlProduct `xml:"product"`
	Light    *xmlLight   `xml:"light"`
	Noise    *xmlNoise   `xml:"noise"`
	Warp     *xmlWarp    `xml:"warp"`
	Strokes  *xmlStrokes `xml:"strokes"`
	Studio   *xmlStudio  `xml:"studio"`
}

type xmlModel struct {
	ID      string `xml:"id,attr"`
	Host    string `xml:"host,attr"`
	Model   string `xml:"model,attr"`
	Side    string `xml:"side,attr"`
	Version string `xml:"version,attr"`
	Metal   string `xml:"metal,attr"`
	SR      string `xml:"sr,attr"`
	Open    string `xml:"open,attr"`
	Median  string `xml:"median,attr"`
	Width   int    `xml:"width,attr"`
	Height  int    `xml:"height,attr"`
}

type xmlMaterial struct {
	Name   string  `xml:"name,attr"`
	Sigma  float64 `xml:"sigma,attr"`
	Length float64 `xml:"length,attr"`
	Grain  float64 `xml:"grain,attr"`
}

type xmlProduct struct {
	Enabled   bool          `xml:"enabled,attr"`
	Seed      uint32        `xml:"seed,attr"`
	Strength  float64       `xml:"strength,attr"`
	Materials []xmlMaterial `xml:"material"`
}

type xmlChannel struct {
	Color  string  `xml:"color,attr"`
	Value  float64 `xml:"value,attr"`
	Enable bool    `xml:"enable,attr"`
}

type xmlRow struct {
	Material string  `xml:"material,attr"`
	R        float64 `xml:"r,attr"`
	G        float64 `xml:"g,attr"`
	B        float64 `xml:"b,attr"`
}

type xmlLight struct {
	Gain            float64      `xml:"gain,attr"`
	Offset          float64      `xml:"offset,attr"`
	TextureStrength float64      `xml:"textureStrength,attr"`
	Channels        []xmlChannel `xml:"channel"`
	Reflectance     *xmlRows     `xml:"reflectance"`
	Crosstalk       *xmlRows     `xml:"crosstalk"`
}

type xmlRows struct {
	Rows []xmlRow `xml:"row"`
}

type xmlNoise struct {
	ReadSigma float64 `xml:"readSigma,attr"`
	ShotGain  float64 `xml:"shotGain,attr"`
	Seed      uint32  `xml:"seed,attr"`
}

type xmlWarp struct {
	Tx             float64 `xml:"tx,attr"`
	Ty             float64 `xml:"ty,attr"`
	ThetaDeg       float64 `xml:"thetaDeg,attr"`
	Scale          float64 `xml:"scale,attr"`
	ShearX         float64 `xml:"shearX,attr"`
	ShearY         float64 `xml:"shearY,attr"`
	TiltX          float64 `xml:"tiltX,attr"`
	TiltY          float64 `xml:"tiltY,attr"`
	BendAmplitude  float64 `xml:"bendAmplitude,attr"`
	BendWavelength float64 `xml:"bendWavelength,attr"`
}

type xmlPoint struct {
	X float64 `xml:"x,attr"`
	Y float64 `xml:"y,attr"`
}

type xmlDefect struct {
	Enabled     *bool      `xml:"enabled,attr"`
	Auto        bool       `xml:"auto,attr"`
	Kind        string     `xml:"kind,attr"`
	Shape       string     `xml:"shape,attr"`
	X           float64    `xml:"x,attr"`
	Y           float64    `xml:"y,attr"`
	X2          float64    `xml:"x2,attr"`
	Y2          float64    `xml:"y2,attr"`
	Radius      float64    `xml:"radius,attr"`
	DeltaGv     float64    `xml:"deltaGv,attr"`
	Softness    float64    `xml:"softness,attr"`
	Amplitude   string     `xml:"amplitude,attr"`
	RelativePct float64    `xml:"relativePct,attr"`
	TargetGv    float64    `xml:"targetGv,attr"`
	HasTarget   bool       `xml:"hasTarget,attr"`
	Profile     string     `xml:"profile,attr"`
	Grain       float64    `xml:"grain,attr"`
	Seed        uint32     `xml:"seed,attr"`
	Color       string     `xml:"color,attr"`
	Points      []xmlPoint `xml:"p"`
}

type xmlScene struct {
	Enabled  *bool   `xml:"enabled,attr"`
	Auto     bool    `xml:"auto,attr"`
	Mode     string  `xml:"mode,attr"`
	Shape    string  `xml:"shape,attr"`
	X        float64 `xml:"x,attr"`
	Y        float64 `xml:"y,attr"`
	X2       float64 `xml:"x2,attr"`
	Y2       float64 `xml:"y2,attr"`
	Radius   float64 `xml:"radius,attr"`
	Material string  `xml:"material,attr"`
	Factor   float64 `xml:"factor,attr"`
}

// xmlStroke 用 ,any 保序解码 <defect>/<scene>。
type xmlStroke struct {
	XMLName xml.Name
	Defect  *xmlDefect
	Scene   *xmlScene
}

func (s *xmlStroke) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	switch start.Name.Local {
	case "defect":
		var v xmlDefect
		if err := d.DecodeElement(&v, &start); err != nil {
			return err
		}
		s.Defect = &v
	case "scene":
		var v xmlScene
		if err := d.DecodeElement(&v, &start); err != nil {
			return err
		}
		s.Scene = &v
	default:
		if err := d.Skip(); err != nil {
			return err
		}
	}
	s.XMLName = start.Name
	return nil
}

func (s xmlStroke) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	if s.Defect != nil {
		return e.EncodeElement(s.Defect, xml.StartElement{Name: xml.Name{Local: "defect"}})
	}
	if s.Scene != nil {
		return e.EncodeElement(s.Scene, xml.StartElement{Name: xml.Name{Local: "scene"}})
	}
	return nil
}

type xmlStrokes struct {
	Items []xmlStroke `xml:",any"`
}

type xmlBrush struct {
	Radius            float64 `xml:"radius,attr"`
	DeltaGv           float64 `xml:"deltaGv,attr"`
	Softness          float64 `xml:"softness,attr"`
	Amplitude         string  `xml:"amplitude,attr"`
	RelativePct       float64 `xml:"relativePct,attr"`
	TargetBright      float64 `xml:"targetBright,attr"`
	TargetDark        float64 `xml:"targetDark,attr"`
	Profile           string  `xml:"profile,attr"`
	Grain             float64 `xml:"grain,attr"`
	ForeignMaterial   int     `xml:"foreignMaterial,attr"`
	ContaminationFact float64 `xml:"contaminationFact,attr"`
	UseColor          bool    `xml:"useColor,attr"`
	Color             string  `xml:"color,attr"`
}

type xmlRandom struct {
	Count     int     `xml:"count,attr"`
	Seed      uint32  `xml:"seed,attr"`
	Kind      int     `xml:"kind,attr"`
	Shape     int     `xml:"shape,attr"`
	RadiusMin float64 `xml:"radiusMin,attr"`
	RadiusMax float64 `xml:"radiusMax,attr"`
	ValueMin  float64 `xml:"valueMin,attr"`
	ValueMax  float64 `xml:"valueMax,attr"`
	Material  int     `xml:"material,attr"`
}

type xmlAuto struct {
	Count           int     `xml:"count,attr"`
	Seed            uint32  `xml:"seed,attr"`
	StrokeSeed      uint32  `xml:"strokeSeed,attr"`
	ProductLenScale float64 `xml:"productLenScale,attr"`
	ProductDraw     int     `xml:"productDraw,attr"`
}

type xmlOutput struct {
	Dir        string `xml:"dir,attr"`
	Channels   string `xml:"channels,attr"`
	Good       bool   `xml:"good,attr"`
	RecipeOnly bool   `xml:"recipeOnly,attr"`
}

type xmlStudio struct {
	View      string     `xml:"view,attr"`
	Channel   string     `xml:"channel,attr"`
	ShowMarks bool       `xml:"showMarks,attr"`
	Tool      int        `xml:"tool,attr"`
	Brush     *xmlBrush  `xml:"brush"`
	Random    *xmlRandom `xml:"random"`
	Auto      *xmlAuto   `xml:"auto"`
	Output    *xmlOutput `xml:"output"`
}

// ---- 编码 ----

var materialNames = [4]string{"base", "sr", "open", "metal"}

func scenarioToXML(s *Scenario) *xmlScenario {
	root := &xmlScenario{
		Format:   ScenarioFormat,
		Version:  ScenarioVersion,
		Producer: s.Producer,
		Created:  s.Created,
		Note:     s.Note,
		Model: xmlModel{
			ID:      s.Model.ID,
			Host:    s.Model.Host,
			Model:   s.Model.Model,
			Side:    s.Model.Side,
			Version: s.Model.Version,
			Metal:   s.Model.Metal,
			SR:      s.Model.SR,
			Open:    s.Model.Open,
			Median:  s.Model.Median,
			Width:   s.Width,
			Height:  s.Height,
		},
	}
	p := &xmlProduct{
		Enabled:  s.Recipe.Product.Enabled,
		Seed:     s.Recipe.Product.Seed,
		Strength: s.Recipe.Product.Strength,
	}
	for m := 0; m < 4; m++ {
		mat := s.Recipe.Product.Materials[m]
		p.Materials = append(p.Materials, xmlMaterial{
			Name:   materialNames[m],
			Sigma:  mat.Sigma,
			Length: mat.Length,
			Grain:  mat.Grain,
		})
	}
	root.Product = p

	l := &xmlLight{
		Gain:            s.Recipe.Light.Gain,
		Offset:          s.Recipe.Light.Offset,
		TextureStrength: s.Recipe.Light.TextureStrength,
	}
	for _, c := range s.Recipe.Light.Channels {
		l.Channels = append(l.Channels, xmlChannel{
			Color:  lightColorName(c.Color),
			Value:  c.Value,
			Enable: c.Enable,
		})
	}
	l.Reflectance = rowsToXML(s.Recipe.Light.Reflectance)
	if s.Recipe.Light.Crosstalk != nil {
		l.Crosstalk = rowsToXML(*s.Recipe.Light.Crosstalk)
	}
	root.Light = l

	root.Noise = &xmlNoise{
		ReadSigma: s.Recipe.Noise.ReadSigma,
		ShotGain:  s.Recipe.Noise.ShotGain,
		Seed:      s.Recipe.Noise.Seed,
	}

	if w := s.Recipe.Warp; w != nil {
		root.Warp = &xmlWarp{
			Tx:             w.Tx,
			Ty:             w.Ty,
			ThetaDeg:       w.ThetaDeg,
			Scale:          w.Scale,
			ShearX:         w.ShearX,
			ShearY:         w.ShearY,
			TiltX:          w.TiltX,
			TiltY:          w.TiltY,
			BendAmplitude:  w.BendAmplitude,
			BendWavelength: w.BendWavelength,
		}
	}

	st := &xmlStrokes{Items: make([]xmlStroke, 0, len(s.Strokes))}
	for _, e := range s.Strokes {
		if e.GV != nil {
			st.Items = append(st.Items, xmlStroke{Defect: defectToXML(e.Enabled, e.Auto, e.GV)})
		} else if e.Scene != nil {
			st.Items = append(st.Items, xmlStroke{Scene: sceneToXML(e.Enabled, e.Auto, e.Scene)})
		}
	}
	root.Strokes = st

	if studio := s.Studio; studio != nil {
		xs := &xmlStudio{
			View:      studio.View,
			Channel:   studio.Channel,
			ShowMarks: studio.ShowMarks,
			Tool:      studio.Tool,
		}
		b := studio.Brush
		xs.Brush = &xmlBrush{
			Radius:            b.Radius,
			DeltaGv:           b.DeltaGv,
			Softness:          b.Softness,
			Amplitude:         amplitudeName(b.Amplitude),
			RelativePct:       b.RelativePct,
			TargetBright:      b.TargetBright,
			TargetDark:        b.TargetDark,
			Profile:           profileName(b.Profile),
			Grain:             b.Grain,
			ForeignMaterial:   b.ForeignMaterial,
			ContaminationFact: b.ContaminationFact,
			UseColor:          b.UseColor,
			Color:             formatColor(b.Color),
		}
		r := studio.Random
		xs.Random = &xmlRandom{
			Count:     r.Count,
			Seed:      r.Seed,
			Kind:      r.Kind,
			Shape:     r.Shape,
			RadiusMin: r.RadiusMin,
			RadiusMax: r.RadiusMax,
			ValueMin:  r.ValueMin,
			ValueMax:  r.ValueMax,
			Material:  r.Material,
		}
		a := studio.Auto
		xs.Auto = &xmlAuto{
			Count:           a.Count,
			Seed:            a.Seed,
			StrokeSeed:      a.StrokeSeed,
			ProductLenScale: a.ProductLenScale,
			ProductDraw:     a.ProductDraw,
		}
		o := studio.Output
		xs.Output = &xmlOutput{
			Dir:        o.Dir,
			Channels:   channelsString(o.Channels),
			Good:       o.Good,
			RecipeOnly: o.RecipeOnly,
		}
		root.Studio = xs
	}
	return root
}

func rowsToXML(rows [4][3]float64) *xmlRows {
	out := &xmlRows{Rows: make([]xmlRow, 0, 4)}
	for m := 0; m < 4; m++ {
		out.Rows = append(out.Rows, xmlRow{
			Material: materialNames[m],
			R:        rows[m][0],
			G:        rows[m][1],
			B:        rows[m][2],
		})
	}
	return out
}

func defectToXML(enabled, auto bool, d *Defect) *xmlDefect {
	out := &xmlDefect{
		Enabled:     &enabled,
		Auto:        auto,
		Kind:        defectKindName(d.Kind),
		Shape:       defectShapeName(d.Shape),
		X:           d.X,
		Y:           d.Y,
		X2:          d.X2,
		Y2:          d.Y2,
		Radius:      d.Radius,
		DeltaGv:     d.DeltaGv,
		Softness:    d.Softness,
		Amplitude:   amplitudeName(d.Amplitude),
		RelativePct: d.RelativePct,
		TargetGv:    d.TargetGv,
		HasTarget:   d.HasTarget,
		Profile:     profileName(d.Profile),
		Grain:       d.Grain,
		Seed:        d.Seed,
	}
	if d.HasColor {
		out.Color = formatColor(d.Color)
	}
	if d.Shape == 4 || len(d.Points) > 0 {
		out.Points = make([]xmlPoint, 0, len(d.Points))
		for _, p := range d.Points {
			out.Points = append(out.Points, xmlPoint{X: p.X, Y: p.Y})
		}
	}
	return out
}

func sceneToXML(enabled, auto bool, s *SceneStroke) *xmlScene {
	return &xmlScene{
		Enabled:  &enabled,
		Auto:     auto,
		Mode:     sceneModeName(s.Mode),
		Shape:    defectShapeName(s.Shape),
		X:        s.X,
		Y:        s.Y,
		X2:       s.X2,
		Y2:       s.Y2,
		Radius:   s.Radius,
		Material: materialName(int(s.Material)),
		Factor:   s.Factor,
	}
}

// ---- 解码 ----

func scenarioFromXML(root *xmlScenario) (*Scenario, error) {
	s := &Scenario{
		Producer: root.Producer,
		Created:  root.Created,
		Note:     root.Note,
		Model: ModelRef{
			ID:      root.Model.ID,
			Host:    root.Model.Host,
			Model:   root.Model.Model,
			Side:    root.Model.Side,
			Version: root.Model.Version,
			Metal:   root.Model.Metal,
			SR:      root.Model.SR,
			Open:    root.Model.Open,
			Median:  root.Model.Median,
		},
		Width:  root.Model.Width,
		Height: root.Model.Height,
	}
	s.Recipe.Light.Channels = []LightChannel{}
	s.Recipe.Defects = []Defect{}
	s.Recipe.SceneStrokes = []SceneStroke{}
	s.Strokes = []StrokeEntry{}

	if p := root.Product; p != nil {
		s.Recipe.Product.Enabled = p.Enabled
		s.Recipe.Product.Seed = p.Seed
		s.Recipe.Product.Strength = p.Strength
		for _, mat := range p.Materials {
			idx, err := materialIndex(mat.Name)
			if err != nil {
				return nil, err
			}
			s.Recipe.Product.Materials[idx] = Material{
				Sigma:  mat.Sigma,
				Length: mat.Length,
				Grain:  mat.Grain,
			}
		}
	}

	if l := root.Light; l != nil {
		s.Recipe.Light.Gain = l.Gain
		s.Recipe.Light.Offset = l.Offset
		s.Recipe.Light.TextureStrength = l.TextureStrength
		for _, c := range l.Channels {
			col, err := parseLightColor(c.Color)
			if err != nil {
				return nil, err
			}
			s.Recipe.Light.Channels = append(s.Recipe.Light.Channels, LightChannel{
				Color:  col,
				Value:  c.Value,
				Enable: c.Enable,
			})
		}
		if l.Reflectance != nil {
			rows, err := rowsFromXML(l.Reflectance)
			if err != nil {
				return nil, err
			}
			s.Recipe.Light.Reflectance = rows
		}
		if l.Crosstalk != nil {
			rows, err := rowsFromXML(l.Crosstalk)
			if err != nil {
				return nil, err
			}
			ct := rows
			s.Recipe.Light.Crosstalk = &ct
		}
	}

	if n := root.Noise; n != nil {
		s.Recipe.Noise = Noise{
			ReadSigma: n.ReadSigma,
			ShotGain:  n.ShotGain,
			Seed:      n.Seed,
		}
	}

	if w := root.Warp; w != nil {
		s.Recipe.Warp = &Warp{
			Tx:             w.Tx,
			Ty:             w.Ty,
			ThetaDeg:       w.ThetaDeg,
			Scale:          w.Scale,
			ShearX:         w.ShearX,
			ShearY:         w.ShearY,
			TiltX:          w.TiltX,
			TiltY:          w.TiltY,
			BendAmplitude:  w.BendAmplitude,
			BendWavelength: w.BendWavelength,
		}
	}

	if st := root.Strokes; st != nil {
		for _, item := range st.Items {
			switch {
			case item.Defect != nil:
				d, err := defectFromXML(item.Defect)
				if err != nil {
					return nil, err
				}
				s.Strokes = append(s.Strokes, StrokeEntry{
					Enabled: item.Defect.Enabled == nil || *item.Defect.Enabled,
					Auto:    item.Defect.Auto,
					GV:      d,
				})
			case item.Scene != nil:
				sc, err := sceneFromXML(item.Scene)
				if err != nil {
					return nil, err
				}
				s.Strokes = append(s.Strokes, StrokeEntry{
					Enabled: item.Scene.Enabled == nil || *item.Scene.Enabled,
					Auto:    item.Scene.Auto,
					Scene:   sc,
				})
			}
		}
	}
	for _, e := range s.Strokes {
		if !e.Enabled {
			continue
		}
		if e.GV != nil {
			s.Recipe.Defects = append(s.Recipe.Defects, *e.GV)
		} else if e.Scene != nil {
			s.Recipe.SceneStrokes = append(s.Recipe.SceneStrokes, *e.Scene)
		}
	}

	if xs := root.Studio; xs != nil {
		studio := &Studio{
			View:      xs.View,
			Channel:   xs.Channel,
			ShowMarks: xs.ShowMarks,
			Tool:      xs.Tool,
		}
		if b := xs.Brush; b != nil {
			amp, err := parseAmplitude(b.Amplitude)
			if err != nil {
				return nil, err
			}
			prof, err := parseProfile(b.Profile)
			if err != nil {
				return nil, err
			}
			col, err := parseColor(b.Color)
			if err != nil {
				return nil, err
			}
			studio.Brush = StudioBrush{
				Radius:            b.Radius,
				DeltaGv:           b.DeltaGv,
				Softness:          b.Softness,
				Amplitude:         amp,
				RelativePct:       b.RelativePct,
				TargetBright:      b.TargetBright,
				TargetDark:        b.TargetDark,
				Profile:           prof,
				Grain:             b.Grain,
				ForeignMaterial:   b.ForeignMaterial,
				ContaminationFact: b.ContaminationFact,
				UseColor:          b.UseColor,
				Color:             col,
			}
		}
		if r := xs.Random; r != nil {
			studio.Random = StudioRandom{
				Count:     r.Count,
				Seed:      r.Seed,
				Kind:      r.Kind,
				Shape:     r.Shape,
				RadiusMin: r.RadiusMin,
				RadiusMax: r.RadiusMax,
				ValueMin:  r.ValueMin,
				ValueMax:  r.ValueMax,
				Material:  r.Material,
			}
		}
		if a := xs.Auto; a != nil {
			studio.Auto = StudioAuto{
				Count:           a.Count,
				Seed:            a.Seed,
				StrokeSeed:      a.StrokeSeed,
				ProductLenScale: a.ProductLenScale,
				ProductDraw:     a.ProductDraw,
			}
		}
		if o := xs.Output; o != nil {
			studio.Output = StudioOutput{
				Dir:        o.Dir,
				Channels:   parseChannels(o.Channels),
				Good:       o.Good,
				RecipeOnly: o.RecipeOnly,
			}
		}
		s.Studio = studio
	}
	return s, nil
}

func rowsFromXML(x *xmlRows) ([4][3]float64, error) {
	var out [4][3]float64
	for _, row := range x.Rows {
		idx, err := materialIndex(row.Material)
		if err != nil {
			return out, err
		}
		out[idx] = [3]float64{row.R, row.G, row.B}
	}
	return out, nil
}

func defectFromXML(x *xmlDefect) (*Defect, error) {
	kind, err := parseDefectKind(x.Kind)
	if err != nil {
		return nil, err
	}
	shape, err := parseDefectShape(x.Shape)
	if err != nil {
		return nil, err
	}
	amp, err := parseAmplitude(x.Amplitude)
	if err != nil {
		return nil, err
	}
	prof, err := parseProfile(x.Profile)
	if err != nil {
		return nil, err
	}
	d := &Defect{
		Kind:        kind,
		Shape:       shape,
		X:           x.X,
		Y:           x.Y,
		X2:          x.X2,
		Y2:          x.Y2,
		Radius:      x.Radius,
		DeltaGv:     x.DeltaGv,
		Softness:    x.Softness,
		Amplitude:   amp,
		RelativePct: x.RelativePct,
		TargetGv:    x.TargetGv,
		HasTarget:   x.HasTarget,
		Profile:     prof,
		Grain:       x.Grain,
		Seed:        x.Seed,
	}
	if x.Color != "" {
		col, err := parseColor(x.Color)
		if err != nil {
			return nil, err
		}
		d.HasColor = true
		d.Color = col
	}
	if len(x.Points) > 0 {
		d.Points = make([]Point, 0, len(x.Points))
		for _, p := range x.Points {
			d.Points = append(d.Points, Point{X: p.X, Y: p.Y})
		}
	}
	return d, nil
}

func sceneFromXML(x *xmlScene) (*SceneStroke, error) {
	mode, err := parseSceneMode(x.Mode)
	if err != nil {
		return nil, err
	}
	shape, err := parseDefectShape(x.Shape)
	if err != nil {
		return nil, err
	}
	mat, err := materialIndex(x.Material)
	if err != nil {
		return nil, err
	}
	return &SceneStroke{
		Mode:     mode,
		Shape:    shape,
		X:        x.X,
		Y:        x.Y,
		X2:       x.X2,
		Y2:       x.Y2,
		Radius:   x.Radius,
		Material: mat,
		Factor:   x.Factor,
	}, nil
}

// ---- 名字与数字 ----

func defectKindName(k int) string {
	if k == 1 {
		return "dark"
	}
	return "bright"
}

func parseDefectKind(s string) (int, error) {
	switch strings.ToLower(s) {
	case "", "bright", "0":
		return 0, nil
	case "dark", "1":
		return 1, nil
	}
	return 0, fmt.Errorf("snap: unknown defect kind %q", s)
}

func defectShapeName(sh int) string {
	switch sh {
	case 1:
		return "line"
	case 2:
		return "rect"
	case 3:
		return "ellipse"
	case 4:
		return "polygon"
	}
	return "spot"
}

func parseDefectShape(s string) (int, error) {
	switch strings.ToLower(s) {
	case "", "spot", "0":
		return 0, nil
	case "line", "1":
		return 1, nil
	case "rect", "2":
		return 2, nil
	case "ellipse", "3":
		return 3, nil
	case "polygon", "4":
		return 4, nil
	}
	return 0, fmt.Errorf("snap: unknown shape %q", s)
}

func amplitudeName(a int) string {
	switch a {
	case 1:
		return "relative"
	case 2:
		return "target"
	}
	return "absolute"
}

func parseAmplitude(s string) (int, error) {
	switch strings.ToLower(s) {
	case "", "absolute", "0":
		return 0, nil
	case "relative", "1":
		return 1, nil
	case "target", "2":
		return 2, nil
	}
	return 0, fmt.Errorf("snap: unknown amplitude %q", s)
}

func profileName(p int) string {
	if p == 1 {
		return "gaussian"
	}
	return "linear"
}

func parseProfile(s string) (int, error) {
	switch strings.ToLower(s) {
	case "", "linear", "0":
		return 0, nil
	case "gaussian", "1":
		return 1, nil
	}
	return 0, fmt.Errorf("snap: unknown profile %q", s)
}

func sceneModeName(m int) string {
	if m == 1 {
		return "texture"
	}
	return "material"
}

func parseSceneMode(s string) (int, error) {
	switch strings.ToLower(s) {
	case "", "material", "0":
		return 0, nil
	case "texture", "1":
		return 1, nil
	}
	return 0, fmt.Errorf("snap: unknown scene mode %q", s)
}

func materialName(m int) string {
	if m >= 0 && m < len(materialNames) {
		return materialNames[m]
	}
	return strconv.Itoa(m)
}

func materialIndex(s string) (int, error) {
	switch strings.ToLower(s) {
	case "", "base", "0":
		return 0, nil
	case "sr", "1":
		return 1, nil
	case "open", "2":
		return 2, nil
	case "metal", "3":
		return 3, nil
	}
	if n, err := strconv.Atoi(s); err == nil && n >= 0 && n < 4 {
		return n, nil
	}
	return 0, fmt.Errorf("snap: unknown material %q", s)
}

func lightColorName(c int) string {
	switch core.LightColor(c) {
	case core.LightRed:
		return "red"
	case core.LightGreen:
		return "green"
	case core.LightBlue:
		return "blue"
	case core.LightWhite:
		return "white"
	}
	return strconv.Itoa(c)
}

func parseLightColor(s string) (int, error) {
	switch strings.ToLower(s) {
	case "", "red", "0":
		return int(core.LightRed), nil
	case "green", "1":
		return int(core.LightGreen), nil
	case "blue", "2":
		return int(core.LightBlue), nil
	case "white", "3":
		return int(core.LightWhite), nil
	}
	return 0, fmt.Errorf("snap: unknown light color %q", s)
}

func formatColor(c [3]float64) string {
	return strconv.FormatFloat(c[0], 'g', -1, 64) + "," +
		strconv.FormatFloat(c[1], 'g', -1, 64) + "," +
		strconv.FormatFloat(c[2], 'g', -1, 64)
}

func parseColor(s string) ([3]float64, error) {
	var out [3]float64
	if s == "" {
		return out, nil
	}
	parts := strings.Split(s, ",")
	if len(parts) != 3 {
		return out, fmt.Errorf("snap: bad color %q, want r,g,b", s)
	}
	for i, p := range parts {
		v, err := strconv.ParseFloat(strings.TrimSpace(p), 64)
		if err != nil {
			return out, fmt.Errorf("snap: bad color %q: %w", s, err)
		}
		out[i] = v
	}
	return out, nil
}

func channelsString(ch [3]bool) string {
	var b strings.Builder
	if ch[0] {
		b.WriteByte('r')
	}
	if ch[1] {
		b.WriteByte('g')
	}
	if ch[2] {
		b.WriteByte('b')
	}
	return b.String()
}

func parseChannels(s string) [3]bool {
	var out [3]bool
	for _, r := range strings.ToLower(s) {
		switch r {
		case 'r':
			out[0] = true
		case 'g':
			out[1] = true
		case 'b':
			out[2] = true
		}
	}
	return out
}
