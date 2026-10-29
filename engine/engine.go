// Package engine adapts the AFVI desktop inspection code to the web build: it turns image bytes
// into the inputs internal/inspect expects, learns accepted ranges from a good sample the way the
// desktop program does, and returns a JSON report the browser and the report store both understand.
//
// This package is the public face of the algorithm: internal/inspect is internal to this module, so
// anything outside it (the web build in ../afvi_web, the WebAssembly module it serves) goes through
// here. There is one copy of the algorithm, not one per front end.
//
// It stays standard library only, like internal/inspect itself, so the same source also compiles
// to js/wasm.
package engine

import (
	"errors"
	"fmt"
	"image"
	"image/color"
	_ "image/jpeg"
	_ "image/png"

	"afvi/teach-imgui/internal/inspect"
)

// Channel selects the inspection polarity the preview uses; the engine itself is grey only.
type Channel string

// Channels.
const (
	Red   Channel = "red"
	Green Channel = "green"
	Blue  Channel = "blue"
)

// ParseChannel maps a request value to a channel, defaulting to red.
func ParseChannel(s string) Channel {
	switch Channel(s) {
	case Green:
		return Green
	case Blue:
		return Blue
	}
	return Red
}

// Errors.
var (
	ErrNoImage   = errors.New("engine: no image")
	ErrBadImage  = errors.New("engine: the image could not be decoded")
	ErrSizeMatch = errors.New("engine: the image and the material map disagree about the size")
)

// Gray is an 8-bit grey image.
type Gray struct {
	W, H int
	Data []uint8
}

// FromImage converts any decoded image to 8-bit grey with the stdlib weighting (0.299/0.587/0.114).
func FromImage(src image.Image) *Gray {
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	out := &Gray{W: w, H: h, Data: make([]uint8, w*h)}
	switch g := src.(type) {
	case *image.Gray:
		for y := 0; y < h; y++ {
			copy(out.Data[y*w:(y+1)*w], g.Pix[y*g.Stride:y*g.Stride+w])
		}
		return out
	case *image.Gray16:
		for y := 0; y < h; y++ {
			row := g.Pix[y*g.Stride:]
			for x := 0; x < w; x++ {
				out.Data[y*w+x] = uint8(row[2*x] >> 8)
			}
		}
		return out
	}
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			r, gg, bb, _ := src.At(b.Min.X+x, b.Min.Y+y).RGBA()
			g := color.GrayModel.Convert(color.RGBA{
				R: uint8(r >> 8), G: uint8(gg >> 8), B: uint8(bb >> 8), A: 255,
			}).(color.Gray)
			out.Data[y*w+x] = g.Y
		}
	}
	return out
}

// Decode reads a PNG or JPEG into a grey image.
func Decode(data []byte) (*Gray, error) {
	if len(data) == 0 {
		return nil, ErrNoImage
	}
	src, _, err := image.Decode(bytesReader(data))
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrBadImage, err)
	}
	return FromImage(src), nil
}

// Params is the inspection configuration the browser sends (and the page shows).
type Params struct {
	// EdgeMargin ignores pixels close to a material boundary (the machine's "Margine").
	EdgeMargin int `json:"edgeMargin"`
	// MinArea is the smallest accepted defect in pixels.
	MinArea int `json:"minArea"`
	// Materials maps a material name to its accepted GV range.
	Materials map[string]MaterialRange `json:"materials"`
}

// MaterialRange is one material's accepted interval.
//
// The fields are declared one per line on purpose: a shared tag ("Low, High float64 `json:\"low\"`")
// would apply "low" to both and "high" would never come back from JSON.
type MaterialRange struct {
	Low     float64 `json:"low"`
	High    float64 `json:"high"`
	Enabled bool    `json:"enabled"`
}

// materialNames are the engine's material indices, in the order internal/inspect uses them.
var materialNames = [inspect.NumMaterials]string{"base", "sr", "open", "metal"}

// indexOf maps a name to an engine material index.
func indexOf(name string) (int, bool) {
	for i, n := range materialNames {
		if n == name {
			return i, true
		}
	}
	return 0, false
}

// ToInspect builds the library parameters. Materials the caller did not mention keep the library
// default (off), so a partial request cannot widen an inspection by accident.
func (p Params) ToInspect() inspect.Params {
	out := inspect.DefaultParams()
	if p.MinArea > 0 {
		for i := range out.Materials {
			out.Materials[i].MinArea = p.MinArea
		}
	}
	if p.EdgeMargin >= 0 && p.EdgeMargin <= inspect.MaxEdgeMargin {
		out.EdgeMargin = p.EdgeMargin
	}
	for name, r := range p.Materials {
		i, ok := indexOf(name)
		if !ok {
			continue
		}
		mp := out.Materials[i]
		mp.Enabled = r.Enabled
		mp.Low, mp.High = r.Low, r.High
		out.Materials[i] = mp
	}
	return out
}

// Defect is one reported defect, flattened for JSON and for the table the page draws.
type Defect struct {
	Material string  `json:"material"`
	Polarity string  `json:"polarity"` // bright | dark
	Area     int     `json:"area"`
	X        int     `json:"x"`
	Y        int     `json:"y"`
	W        int     `json:"w"`
	H        int     `json:"h"`
	CX       float64 `json:"cx"`
	CY       float64 `json:"cy"`
	PeakGV   int     `json:"peak"`
	MeanGV   float64 `json:"mean"`
}

// MaterialStat is the per-material summary the page shows next to the image.
type MaterialStat struct {
	Material string  `json:"material"`
	Pixels   int     `json:"pixels"`
	Mean     float64 `json:"mean"`
	Std      float64 `json:"std"`
	MinGV    int     `json:"min"`
	MaxGV    int     `json:"max"`
	Low      float64 `json:"low"`
	High     float64 `json:"high"`
	Enabled  bool    `json:"enabled"`
}

// Result is the whole answer of one inspection.
type Result struct {
	Verdict       string         `json:"verdict"` // ok | ng
	Defects       []Defect       `json:"defects"`
	DefectCount   int            `json:"defectCount"`
	FlaggedPixels int            `json:"flaggedPixels"`
	EdgeIgnored   int            `json:"edgeIgnored"`
	Materials     []MaterialStat `json:"materials"`
	Milliseconds  float64        `json:"ms"`
}

// Inspect runs one inspection over a grey image and a material map.
func Inspect(img *Gray, material []uint8, p Params) (*Result, error) {
	if img == nil || len(img.Data) == 0 {
		return nil, ErrNoImage
	}
	if len(material) != img.W*img.H {
		return nil, ErrSizeMatch
	}
	regions := inspect.Regions{Width: img.W, Height: img.H, Material: material}
	params := p.ToInspect()
	started := nowMillis()
	rep, err := inspect.Inspect(inspect.Image{Width: img.W, Height: img.H, Pix: img.Data}, regions, params)
	if err != nil {
		return nil, err
	}
	res := &Result{
		Verdict:       "ok",
		FlaggedPixels: rep.FlaggedPixels,
		EdgeIgnored:   rep.EdgeIgnored,
		DefectCount:   len(rep.Defects),
		Milliseconds:  nowMillis() - started,
	}
	if len(rep.Defects) > 0 {
		res.Verdict = "ng"
	}
	res.Defects = make([]Defect, 0, len(rep.Defects))
	for _, d := range rep.Defects {
		mat := "base"
		if int(d.Material) < len(materialNames) {
			mat = materialNames[d.Material]
		}
		pol := "bright"
		if d.Polarity == inspect.Dark {
			pol = "dark"
		}
		res.Defects = append(res.Defects, Defect{
			Material: mat,
			Polarity: pol,
			Area:     d.Area,
			X:        d.BBox.X, Y: d.BBox.Y, W: d.BBox.W, H: d.BBox.H,
			CX: d.CentroidX, CY: d.CentroidY,
			PeakGV: int(d.PeakGV),
			MeanGV: d.MeanGV,
		})
	}
	for i := 0; i < inspect.NumMaterials; i++ {
		st := rep.PerMaterial[i]
		mp := params.Materials[i]
		res.Materials = append(res.Materials, MaterialStat{
			Material: materialNames[i],
			Pixels:   st.Pixels,
			Mean:     st.Mean,
			Std:      st.Std,
			MinGV:    int(st.Min),
			MaxGV:    int(st.Max),
			Low:      mp.Low,
			High:     mp.High,
			Enabled:  mp.Enabled,
		})
	}
	return res, nil
}

// Learn derives the accepted range of every material from a defect-free sample: internal/inspect
// computes mean -/+ k standard deviations per material and returns the whole parameter set, so the
// web build learns exactly what the desktop program learns.
func Learn(good *Gray, material []uint8, k float64) (Params, error) {
	if good == nil || len(good.Data) == 0 {
		return Params{}, ErrNoImage
	}
	if len(material) != good.W*good.H {
		return Params{}, ErrSizeMatch
	}
	if k <= 0 {
		k = 4
	}
	regions := inspect.Regions{Width: good.W, Height: good.H, Material: material}
	base := inspect.DefaultParams()
	base.EdgeMargin = 6 // keep boundaries out of the statistics, as the desktop default does
	learned, err := inspect.Learn(inspect.Image{Width: good.W, Height: good.H, Pix: good.Data}, regions, k, base)
	if err != nil {
		return Params{}, err
	}
	out := Params{EdgeMargin: learned.EdgeMargin, MinArea: base.Materials[0].MinArea,
		Materials: map[string]MaterialRange{}}
	for i := 0; i < inspect.NumMaterials; i++ {
		mp := learned.Materials[i]
		if !mp.Enabled {
			continue
		}
		out.Materials[materialNames[i]] = MaterialRange{
			Low:     round1(mp.Low),
			High:    round1(mp.High),
			Enabled: true,
		}
	}
	return out, nil
}

func round1(v float64) float64 { return float64(int(v*10+0.5)) / 10 }
