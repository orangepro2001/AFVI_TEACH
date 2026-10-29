package engine

import (
	"afvi/teach-imgui/internal/core"
)

// The light page of the desktop program renders its scene with core.RenderChannel and shows the
// per-material statistics of the design layers. The web copy needs exactly those two operations, so
// they are exposed here - again as a door, not as a reimplementation.

// LightSetting is the light configuration the web page edits: the four lamp values (0..1023, the
// machine's scale) and nothing else. Gain, offset, reflectance and crosstalk stay at the desktop
// program's defaults, which is what the light page shows until the user changes them.
type LightSetting struct {
	R, G, B, White float64
}

// DefaultLight is the light a real capture is taken with, the same default the desktop teaching
// program uses (red, green and blue at 512, white off).
func DefaultLight() LightSetting { return LightSetting{R: 512, G: 512, B: 512} }

// toCore builds the full core.LightSetting from the four lamp values.
func (l LightSetting) toCore() core.LightSetting {
	s := core.DefaultCaptureLight()
	s.Channels = []core.LightChannel{
		{Color: core.LightRed, Value: l.R, Enable: l.R > 0},
		{Color: core.LightGreen, Value: l.G, Enable: l.G > 0},
		{Color: core.LightBlue, Value: l.B, Enable: l.B > 0},
		{Color: core.LightWhite, Value: l.White, Enable: l.White > 0},
	}
	return s
}

// LightScene is the design plate the page renders: the "MEDIAN" grey image and its material map.
// It is built once per plate and reused for every render, because building it is the expensive part.
type LightScene struct {
	scene  *core.LightScene
	median *Gray
	// Width, Height and Material are what the page needs to draw the material view.
	Width, Height int
	Material      []uint8
}

// NewLightScene prepares a plate for rendering. median is the design grey image ("MEDIAN"), material
// is the per-pixel material map (0 base, 1 SR, 2 open, 3 metal).
func NewLightScene(median *Gray, material []uint8) (*LightScene, error) {
	if median == nil || len(median.Data) == 0 {
		return nil, ErrNoImage
	}
	if len(material) != median.W*median.H {
		return nil, ErrSizeMatch
	}
	img := &core.GrayImage{Width: median.W, Height: median.H, Data: median.Data}
	scene, err := core.NewLightScene(img, material)
	if err != nil {
		return nil, err
	}
	return &LightScene{scene: scene, median: median, Width: median.W, Height: median.H, Material: material}, nil
}

// Render returns the grey image for one camera channel, exactly as core.RenderChannel computes it.
func (s *LightScene) Render(l LightSetting, channel Channel) *Gray {
	if s == nil || s.scene == nil {
		return nil
	}
	ls := l.toCore()
	img := core.RenderChannel(s.scene, &ls, channel.toCore())
	return &Gray{W: img.Width, H: img.Height, Data: img.Data}
}

// MaterialStats returns the per-material mean and standard deviation of the plate, the numbers the
// light page's GV table shows.
func (s *LightScene) MaterialStats() []Stat {
	if s == nil || s.scene == nil {
		return nil
	}
	img := &core.GrayImage{Width: s.Width, Height: s.Height, Data: s.median.Data}
	stats, err := core.MaterialStats(img, s.Material)
	if err != nil {
		return nil
	}
	out := make([]Stat, 0, len(stats))
	for i := range stats {
		out = append(out, Stat{Material: materialNames[i], Pixels: stats[i].Count,
			Mean: stats[i].Mean, Std: stats[i].Std})
	}
	return out
}

// Stat is one row of the GV table.
type Stat struct {
	Material string  `json:"material"`
	Pixels   int     `json:"pixels"`
	Mean     float64 `json:"mean"`
	Std      float64 `json:"std"`
}

// toCore maps the channel name to core's enum.
func (c Channel) toCore() core.CameraChannel {
	switch c {
	case Green:
		return core.ChGreen
	case Blue:
		return core.ChBlue
	}
	return core.ChRed
}
