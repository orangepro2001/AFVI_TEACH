// Package snap defines the capture file format shared with the camera program.
package snap

import (
	"encoding/hex"
	"fmt"
	"sort"
	"strings"

	"afvi/teach-imgui/internal/core"
)

// Format, Version and MetaFile identify a capture description.
const (
	Format   = "afvi-snap"
	Version  = 1
	MetaFile = "snap.json"
)

// Channel and kind names used on the wire.
const (
	ChannelRed   = "red"
	ChannelGreen = "green"
	ChannelBlue  = "blue"
	KindTest     = "test"
	KindGood     = "good"
)

// ModelRef identifies the model the capture was taken from.
type ModelRef struct {
	ID      string `json:"id"`
	Host    string `json:"host"`
	Model   string `json:"model"`
	Side    string `json:"side"`
	Version string `json:"version"`
	Metal   string `json:"metal,omitempty"`
	SR      string `json:"sr,omitempty"`
	Open    string `json:"open,omitempty"`
	Median  string `json:"median,omitempty"`
}

// Image describes one gray PNG of the capture.
type Image struct {
	Channel string `json:"channel"`
	Kind    string `json:"kind"`
	File    string `json:"file"`
	Width   int    `json:"width"`
	Height  int    `json:"height"`
	SHA256  string `json:"sha256"` // hex sha256 of the raw pixel bytes
}

// Material is the surface non-uniformity of one material.
type Material struct {
	Sigma  float64 `json:"sigma"`
	Length float64 `json:"length"`
	Grain  float64 `json:"grain"`
}

// Product describes the virtual good unit.
type Product struct {
	Enabled   bool        `json:"enabled"`
	Seed      uint32      `json:"seed"`
	Strength  float64     `json:"strength"`
	Materials [4]Material `json:"materials"` // base, sr, open, metal
}

// LightChannel is one light source setting.
type LightChannel struct {
	Color  int     `json:"color"` // core.LightColor: 0 red, 1 green, 2 blue, 3 white
	Value  float64 `json:"value"`
	Enable bool    `json:"enable"`
}

// Light is the illumination and camera setting.
type Light struct {
	Channels        []LightChannel `json:"channels"`
	Reflectance     [4][3]float64  `json:"reflectance"`
	Crosstalk       *[4][3]float64 `json:"crosstalk,omitempty"`
	Gain            float64        `json:"gain"`
	Offset          float64        `json:"offset"`
	TextureStrength float64        `json:"textureStrength"`
}

// Noise is the sensor noise setting.
type Noise struct {
	ReadSigma float64 `json:"readSigma"`
	ShotGain  float64 `json:"shotGain"`
	Seed      uint32  `json:"seed"`
}

// Point is a polygon vertex.
type Point struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

// Defect mirrors core.DefectStroke. Kind 0 bright / 1 dark; Shape 0 spot, 1 line, 2 rect,
// 3 ellipse, 4 polygon; Amplitude 0 absolute, 1 relative, 2 target; Profile 0 linear, 1 gaussian.
type Defect struct {
	Kind        int        `json:"kind"`
	Shape       int        `json:"shape"`
	X           float64    `json:"x"`
	Y           float64    `json:"y"`
	X2          float64    `json:"x2"`
	Y2          float64    `json:"y2"`
	Radius      float64    `json:"radius"`
	DeltaGv     float64    `json:"deltaGv"`
	Softness    float64    `json:"softness"`
	Amplitude   int        `json:"amplitude"`
	RelativePct float64    `json:"relativePct"`
	TargetGv    float64    `json:"targetGv"`
	HasTarget   bool       `json:"hasTarget"`
	Profile     int        `json:"profile"`
	Grain       float64    `json:"grain"`
	Seed        uint32     `json:"seed"`
	Points      []Point    `json:"points,omitempty"`
	HasColor    bool       `json:"hasColor"`
	Color       [3]float64 `json:"color"`
}

// SceneStroke mirrors core.SceneStroke. Mode 0 material, 1 texture; Shape as in Defect.
type SceneStroke struct {
	Mode     int     `json:"mode"`
	Shape    int     `json:"shape"`
	X        float64 `json:"x"`
	Y        float64 `json:"y"`
	X2       float64 `json:"x2"`
	Y2       float64 `json:"y2"`
	Radius   float64 `json:"radius"`
	Material int     `json:"material"`
	Factor   float64 `json:"factor"`
}

// Recipe is the full simulation recipe of a capture.
type Recipe struct {
	Product      Product       `json:"product"`
	Light        Light         `json:"light"`
	Noise        Noise         `json:"noise"`
	Defects      []Defect      `json:"defects"`
	SceneStrokes []SceneStroke `json:"sceneStrokes"`
}

// Meta is the content of the description file.
type Meta struct {
	Format   string   `json:"format"`
	Version  int      `json:"version"`
	Producer string   `json:"producer"`
	Created  string   `json:"created"` // RFC 3339, UTC
	Note     string   `json:"note,omitempty"`
	Model    ModelRef `json:"model"`
	Width    int      `json:"width"`
	Height   int      `json:"height"`
	Recipe   Recipe   `json:"recipe"`
	Images   []Image  `json:"images"`
}

// CoreRecipe is a Recipe converted to the simulation types.
type CoreRecipe struct {
	Product core.ProductOptions
	Light   core.LightSetting
	Noise   core.SensorNoise
	Defects []core.DefectStroke
	Scene   []core.SceneStroke
}

// NewRecipe converts simulation settings to the wire format (deep copy; nil slices become empty non-nil slices so the JSON shows [] not null).
func NewRecipe(product core.ProductOptions, light core.LightSetting, noise core.SensorNoise, defects []core.DefectStroke, scene []core.SceneStroke) Recipe {
	r := Recipe{
		Product: Product{
			Enabled:  product.Enabled,
			Seed:     product.Seed,
			Strength: product.Strength,
		},
		Light: Light{
			Channels:        make([]LightChannel, 0, len(light.Channels)),
			Reflectance:     light.Reflectance,
			Gain:            light.Gain,
			Offset:          light.Offset,
			TextureStrength: light.TextureStrength,
		},
		Noise: Noise{
			ReadSigma: noise.ReadSigma,
			ShotGain:  noise.ShotGain,
			Seed:      noise.Seed,
		},
		Defects:      make([]Defect, 0, len(defects)),
		SceneStrokes: make([]SceneStroke, 0, len(scene)),
	}
	for m := 0; m < 4; m++ {
		r.Product.Materials[m] = Material{
			Sigma:  product.Materials[m].Sigma,
			Length: product.Materials[m].Length,
			Grain:  product.Materials[m].Grain,
		}
	}
	for _, c := range light.Channels {
		r.Light.Channels = append(r.Light.Channels, LightChannel{
			Color:  int(c.Color),
			Value:  c.Value,
			Enable: c.Enable,
		})
	}
	if light.Crosstalk != nil {
		ct := [4][3]float64(*light.Crosstalk)
		r.Light.Crosstalk = &ct
	}
	for _, d := range defects {
		r.Defects = append(r.Defects, defectToWire(d))
	}
	for _, s := range scene {
		r.SceneStrokes = append(r.SceneStrokes, SceneStroke{
			Mode:     int(s.Mode),
			Shape:    int(s.Shape),
			X:        s.X,
			Y:        s.Y,
			X2:       s.X2,
			Y2:       s.Y2,
			Radius:   s.Radius,
			Material: int(s.Material),
			Factor:   s.Factor,
		})
	}
	return r
}

func defectToWire(d core.DefectStroke) Defect {
	out := Defect{
		Kind:        int(d.Kind),
		Shape:       int(d.Shape),
		X:           d.X,
		Y:           d.Y,
		X2:          d.X2,
		Y2:          d.Y2,
		Radius:      d.Radius,
		DeltaGv:     d.DeltaGv,
		Softness:    d.Softness,
		Amplitude:   int(d.Amplitude),
		RelativePct: d.RelativePct,
		TargetGv:    d.TargetGv,
		HasTarget:   d.HasTarget,
		Profile:     int(d.Profile),
		Grain:       d.Grain,
		Seed:        d.Seed,
		HasColor:    d.HasColor,
		Color:       d.Color,
	}
	if len(d.Points) > 0 {
		out.Points = make([]Point, len(d.Points))
		for i, p := range d.Points {
			out.Points[i] = Point{X: p.X, Y: p.Y}
		}
	}
	return out
}

// Core converts back (deep copy). NewRecipe followed by Core must reproduce the inputs exactly (compare with reflect.DeepEqual in tests).
func (r Recipe) Core() CoreRecipe {
	out := CoreRecipe{
		Product: core.ProductOptions{
			Enabled:  r.Product.Enabled,
			Seed:     r.Product.Seed,
			Strength: r.Product.Strength,
		},
		Light: core.LightSetting{
			Channels:        make([]core.LightChannel, 0, len(r.Light.Channels)),
			Reflectance:     r.Light.Reflectance,
			Gain:            r.Light.Gain,
			Offset:          r.Light.Offset,
			TextureStrength: r.Light.TextureStrength,
		},
		Noise: core.SensorNoise{
			ReadSigma: r.Noise.ReadSigma,
			ShotGain:  r.Noise.ShotGain,
			Seed:      r.Noise.Seed,
		},
		Defects: make([]core.DefectStroke, 0, len(r.Defects)),
		Scene:   make([]core.SceneStroke, 0, len(r.SceneStrokes)),
	}
	for m := 0; m < 4; m++ {
		out.Product.Materials[m] = core.ProductMaterial{
			Sigma:  r.Product.Materials[m].Sigma,
			Length: r.Product.Materials[m].Length,
			Grain:  r.Product.Materials[m].Grain,
		}
	}
	for _, c := range r.Light.Channels {
		out.Light.Channels = append(out.Light.Channels, core.LightChannel{
			Color:  core.LightColor(c.Color),
			Value:  c.Value,
			Enable: c.Enable,
		})
	}
	if r.Light.Crosstalk != nil {
		ct := core.Crosstalk(*r.Light.Crosstalk)
		out.Light.Crosstalk = &ct
	}
	for _, d := range r.Defects {
		out.Defects = append(out.Defects, defectFromWire(d))
	}
	for _, s := range r.SceneStrokes {
		out.Scene = append(out.Scene, core.SceneStroke{
			Mode:     core.SceneMode(s.Mode),
			Shape:    core.DefectShape(s.Shape),
			X:        s.X,
			Y:        s.Y,
			X2:       s.X2,
			Y2:       s.Y2,
			Radius:   s.Radius,
			Material: uint8(s.Material),
			Factor:   s.Factor,
		})
	}
	return out
}

func defectFromWire(d Defect) core.DefectStroke {
	out := core.DefectStroke{
		Kind:        core.DefectKind(d.Kind),
		Shape:       core.DefectShape(d.Shape),
		X:           d.X,
		Y:           d.Y,
		X2:          d.X2,
		Y2:          d.Y2,
		Radius:      d.Radius,
		DeltaGv:     d.DeltaGv,
		Softness:    d.Softness,
		Amplitude:   core.DefectAmplitude(d.Amplitude),
		RelativePct: d.RelativePct,
		TargetGv:    d.TargetGv,
		HasTarget:   d.HasTarget,
		Profile:     core.DefectProfile(d.Profile),
		Grain:       d.Grain,
		Seed:        d.Seed,
		HasColor:    d.HasColor,
		Color:       d.Color,
	}
	if len(d.Points) > 0 {
		out.Points = make([]core.Point, len(d.Points))
		for i, p := range d.Points {
			out.Points[i] = core.Point{X: p.X, Y: p.Y}
		}
	}
	return out
}

// ChannelName returns the wire name of a camera channel.
func ChannelName(ch core.CameraChannel) string {
	switch ch {
	case core.ChRed:
		return ChannelRed
	case core.ChGreen:
		return ChannelGreen
	case core.ChBlue:
		return ChannelBlue
	}
	return ""
}

// ParseChannel parses a channel name case-insensitively.
func ParseChannel(name string) (core.CameraChannel, error) {
	switch strings.ToLower(name) {
	case ChannelRed:
		return core.ChRed, nil
	case ChannelGreen:
		return core.ChGreen, nil
	case ChannelBlue:
		return core.ChBlue, nil
	}
	return 0, fmt.Errorf("snap: unknown channel %q", name)
}

// ImageFile returns the file name of an image of the given channel and kind.
func ImageFile(channel, kind string) string {
	if kind == KindGood {
		return channel + "-good.png"
	}
	return channel + ".png"
}

// FindImage returns the image entry for channel+kind.
func (m *Meta) FindImage(channel, kind string) (Image, bool) {
	for _, img := range m.Images {
		if img.Channel == channel && img.Kind == kind {
			return img, true
		}
	}
	return Image{}, false
}

// SetImage replaces the entry with the same channel+kind or appends it, then sorts Images by channel (red, green, blue) and kind (test, good).
func (m *Meta) SetImage(img Image) {
	replaced := false
	for i := range m.Images {
		if m.Images[i].Channel == img.Channel && m.Images[i].Kind == img.Kind {
			m.Images[i] = img
			replaced = true
			break
		}
	}
	if !replaced {
		m.Images = append(m.Images, img)
	}
	sort.SliceStable(m.Images, func(i, j int) bool {
		ci := channelRank(m.Images[i].Channel)
		cj := channelRank(m.Images[j].Channel)
		if ci != cj {
			return ci < cj
		}
		return kindRank(m.Images[i].Kind) < kindRank(m.Images[j].Kind)
	})
}

func channelRank(channel string) int {
	switch channel {
	case ChannelRed:
		return 0
	case ChannelGreen:
		return 1
	case ChannelBlue:
		return 2
	}
	return 3
}

func kindRank(kind string) int {
	if kind == KindTest {
		return 0
	}
	if kind == KindGood {
		return 1
	}
	return 2
}

// Validate checks the description for structural problems.
func (m *Meta) Validate() error {
	if m.Format != Format {
		return fmt.Errorf("snap: bad format %q, want %q", m.Format, Format)
	}
	if m.Version <= 0 || m.Version > Version {
		return fmt.Errorf("snap: unsupported version %d, want 1..%d", m.Version, Version)
	}
	if m.Width <= 0 || m.Height <= 0 {
		return fmt.Errorf("snap: bad size %dx%d", m.Width, m.Height)
	}
	if n := len(m.Recipe.Light.Channels); n < 1 || n > 8 {
		return fmt.Errorf("snap: light has %d channels, want 1..8", n)
	}
	if m.Recipe.Product.Strength < 0 {
		return fmt.Errorf("snap: negative product strength %v", m.Recipe.Product.Strength)
	}
	for i, d := range m.Recipe.Defects {
		if err := validateDefect(d); err != nil {
			return fmt.Errorf("snap: defect %d: %w", i, err)
		}
	}
	for i, s := range m.Recipe.SceneStrokes {
		if s.Mode < 0 || s.Mode > 1 {
			return fmt.Errorf("snap: scene stroke %d: bad mode %d", i, s.Mode)
		}
		if s.Shape < 0 || s.Shape > 4 {
			return fmt.Errorf("snap: scene stroke %d: bad shape %d", i, s.Shape)
		}
		if s.Material < 0 || s.Material > 3 {
			return fmt.Errorf("snap: scene stroke %d: bad material %d", i, s.Material)
		}
	}
	for i, img := range m.Images {
		if err := m.validateImage(img); err != nil {
			return fmt.Errorf("snap: image %d: %w", i, err)
		}
	}
	return nil
}

func validateDefect(d Defect) error {
	if d.Kind < 0 || d.Kind > 1 {
		return fmt.Errorf("bad kind %d", d.Kind)
	}
	if d.Shape < 0 || d.Shape > 4 {
		return fmt.Errorf("bad shape %d", d.Shape)
	}
	if d.Amplitude < 0 || d.Amplitude > 2 {
		return fmt.Errorf("bad amplitude %d", d.Amplitude)
	}
	if d.Profile < 0 || d.Profile > 1 {
		return fmt.Errorf("bad profile %d", d.Profile)
	}
	if d.Shape == 4 && len(d.Points) < 3 {
		return fmt.Errorf("polygon needs at least 3 points, got %d", len(d.Points))
	}
	return nil
}

func (m *Meta) validateImage(img Image) error {
	if img.Channel != ChannelRed && img.Channel != ChannelGreen && img.Channel != ChannelBlue {
		return fmt.Errorf("unknown channel %q", img.Channel)
	}
	if img.Kind != KindTest && img.Kind != KindGood {
		return fmt.Errorf("unknown kind %q", img.Kind)
	}
	if img.File == "" {
		return fmt.Errorf("empty file name")
	}
	if img.File == "." || img.File == ".." || strings.ContainsAny(img.File, `/\`) {
		return fmt.Errorf("file %q is not a plain file name", img.File)
	}
	if img.Width <= 0 || img.Height <= 0 {
		return fmt.Errorf("bad size %dx%d", img.Width, img.Height)
	}
	if img.Width != m.Width || img.Height != m.Height {
		return fmt.Errorf("size %dx%d does not match meta %dx%d", img.Width, img.Height, m.Width, m.Height)
	}
	if len(img.SHA256) != 64 {
		return fmt.Errorf("sha256 must be 64 hex characters, got %d", len(img.SHA256))
	}
	if _, err := hex.DecodeString(img.SHA256); err != nil {
		return fmt.Errorf("sha256 is not hex: %w", err)
	}
	return nil
}
