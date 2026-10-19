package inspect

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// 检出参数文件（afvi-inspect）：教学程序把每个相机通道调好的检出参数保存下来，
// 下次直接读取。文件只含检查包自己的数据，不含任何仿真信息；型号 ID 仅作记录。

// 检出参数文件的格式标识、版本和默认文件名。
const (
	ParamsFormat   = "afvi-inspect"
	ParamsVersion  = 1
	ParamsFileName = "inspect.json"
)

// ChannelParams 是一个相机通道的检出参数。
type ChannelParams struct {
	Learned bool   `json:"learned"` // 区间是否已经学习过（false 时程序会在下次运行时学习）
	Params  Params `json:"params"`
}

// ParamsFile 是检出参数文件的内容。Channels 以 "red"、"green"、"blue" 为键。
type ParamsFile struct {
	Format   string                   `json:"format"`
	Version  int                      `json:"version"`
	Producer string                   `json:"producer,omitempty"`
	Saved    string                   `json:"saved,omitempty"` // RFC 3339，UTC
	Model    string                   `json:"model,omitempty"` // 参数所属型号的 ID，仅作记录与核对
	Width    int                      `json:"width,omitempty"`
	Height   int                      `json:"height,omitempty"`
	LearnK   float64                  `json:"learnK,omitempty"` // 学习区间 = 均值 ± k·σ 的系数
	Channels map[string]ChannelParams `json:"channels"`
}

// tileRuleWire 是一条小格规则在文件中的形式。
type tileRuleWire struct {
	Enabled    bool    `json:"enabled"`
	Bright     float64 `json:"bright"`
	Dark       float64 `json:"dark"`
	BrightSize int     `json:"brightSize"`
	DarkSize   int     `json:"darkSize"`
}

// tileRuleToWire 把 TileRule 转成文件形式。
func tileRuleToWire(r TileRule) tileRuleWire {
	return tileRuleWire{
		Enabled:    r.Enabled,
		Bright:     r.Bright,
		Dark:       r.Dark,
		BrightSize: r.BrightSize,
		DarkSize:   r.DarkSize,
	}
}

// tileRuleFromWire 把文件形式转成 TileRule。
func tileRuleFromWire(w tileRuleWire) TileRule {
	return TileRule{
		Enabled:    w.Enabled,
		Bright:     w.Bright,
		Dark:       w.Dark,
		BrightSize: w.BrightSize,
		DarkSize:   w.DarkSize,
	}
}

// materialWire 是单个材质在文件中的形式。
type materialWire struct {
	Enabled         bool         `json:"enabled"`
	Low             float64      `json:"low"`
	High            float64      `json:"high"`
	Size            int          `json:"size"`
	MinArea         int          `json:"minArea"`
	LocalBackground bool         `json:"localBackground"`
	IntervalOff     bool         `json:"intervalOff"`
	Abs             tileRuleWire `json:"abs"`
	Pct             tileRuleWire `json:"pct"`
}

// materialToWire 把 MaterialParams 转成文件形式。
func materialToWire(m MaterialParams) materialWire {
	return materialWire{
		Enabled:         m.Enabled,
		Low:             m.Low,
		High:            m.High,
		Size:            m.Size,
		MinArea:         m.MinArea,
		LocalBackground: m.LocalBackground,
		IntervalOff:     m.IntervalOff,
		Abs:             tileRuleToWire(m.Abs),
		Pct:             tileRuleToWire(m.Pct),
	}
}

// materialFromWire 把文件形式转成 MaterialParams。
func materialFromWire(w materialWire) MaterialParams {
	return MaterialParams{
		Enabled:         w.Enabled,
		Low:             w.Low,
		High:            w.High,
		Size:            w.Size,
		MinArea:         w.MinArea,
		LocalBackground: w.LocalBackground,
		IntervalOff:     w.IntervalOff,
		Abs:             tileRuleFromWire(w.Abs),
		Pct:             tileRuleFromWire(w.Pct),
	}
}

// paramsWire 是 Params 在文件中的形式：材质按名字（base/sr/open/metal）存放。
// Materials 用 json.RawMessage 保存，以便对每个材质以默认值为底叠加文件里的字段。
type paramsWire struct {
	Materials        map[string]json.RawMessage `json:"materials"`
	EdgeMargin       int                        `json:"edgeMargin"`
	BackgroundRadius int                        `json:"backgroundRadius"`
	TileSize         int                        `json:"tileSize"`
	TileSmooth       bool                       `json:"tileSmooth"`
}

// MarshalJSON 按名字写出各材质，文件不依赖材质索引的顺序。
func (p Params) MarshalJSON() ([]byte, error) {
	w := paramsWire{
		Materials:        make(map[string]json.RawMessage, NumMaterials),
		EdgeMargin:       p.EdgeMargin,
		BackgroundRadius: p.BackgroundRadius,
		TileSize:         p.TileSize,
		TileSmooth:       p.TileSmooth,
	}
	for i := 0; i < NumMaterials; i++ {
		data, err := json.Marshal(materialToWire(p.Materials[i]))
		if err != nil {
			return nil, err
		}
		w.Materials[Material(i).String()] = json.RawMessage(data)
	}
	return json.Marshal(w)
}

// UnmarshalJSON 以 DefaultParams 为底读取：文件里缺少的材质或字段保持库默认值，
// 未知字段被忽略。结果是否合法由 Validate 判断。
func (p *Params) UnmarshalJSON(data []byte) error {
	def := DefaultParams()
	w := paramsWire{
		EdgeMargin:       def.EdgeMargin,
		BackgroundRadius: def.BackgroundRadius,
		TileSize:         def.TileSize,
		TileSmooth:       def.TileSmooth,
	}
	if err := json.Unmarshal(data, &w); err != nil {
		return err
	}
	out := def
	out.EdgeMargin = w.EdgeMargin
	out.BackgroundRadius = w.BackgroundRadius
	out.TileSize = w.TileSize
	out.TileSmooth = w.TileSmooth
	for i := 0; i < NumMaterials; i++ {
		raw, ok := w.Materials[Material(i).String()]
		if !ok {
			continue
		}
		// 以该材质的默认值为底，叠加文件里出现的字段。
		mw := materialToWire(def.Materials[i])
		if err := json.Unmarshal(raw, &mw); err != nil {
			return err
		}
		out.Materials[i] = materialFromWire(mw)
	}
	*p = out
	return nil
}

// Validate 校验文件头与每个通道的参数。
func (f *ParamsFile) Validate() error {
	if f.Format != ParamsFormat {
		return fmt.Errorf("检出参数文件：format=%q，应为 %q", f.Format, ParamsFormat)
	}
	if f.Version != ParamsVersion {
		return fmt.Errorf("检出参数文件：不支持的 version=%d（支持 %d）", f.Version, ParamsVersion)
	}
	for name, cp := range f.Channels {
		if err := cp.Params.Validate(); err != nil {
			return fmt.Errorf("检出参数文件：通道 %s：%w", name, err)
		}
	}
	return nil
}

// ResolveParamsPath 返回参数文件的路径：path 是目录时取其中的 inspect.json。
func ResolveParamsPath(path string) string {
	if st, err := os.Stat(path); err == nil && st.IsDir() {
		return filepath.Join(path, ParamsFileName)
	}
	return path
}

// WriteParamsFile 写出参数文件（目录不存在时创建；先写临时文件再替换，不会留下半个文件）。
// path 是目录时写入其中的 inspect.json。Format、Version 和 Saved 由本函数填写。
func WriteParamsFile(path string, f *ParamsFile) error {
	out := *f
	out.Format = ParamsFormat
	out.Version = ParamsVersion
	out.Saved = time.Now().UTC().Format(time.RFC3339)
	if err := out.Validate(); err != nil {
		return err
	}
	data, err := json.MarshalIndent(&out, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	target := path
	if st, err := os.Stat(path); err == nil && st.IsDir() {
		target = filepath.Join(path, ParamsFileName)
	}
	dir := filepath.Dir(target)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".inspect-*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return err
	}
	if err := os.Rename(tmpName, target); err != nil {
		os.Remove(tmpName)
		return err
	}
	return nil
}

// ReadParamsFile 读取并校验参数文件；path 是目录时读取其中的 inspect.json。
func ReadParamsFile(path string) (*ParamsFile, error) {
	data, err := os.ReadFile(ResolveParamsPath(path))
	if err != nil {
		return nil, err
	}
	var f ParamsFile
	if err := json.Unmarshal(data, &f); err != nil {
		return nil, fmt.Errorf("检出参数文件：%w", err)
	}
	if err := f.Validate(); err != nil {
		return nil, err
	}
	return &f, nil
}
