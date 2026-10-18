package snap

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"afvi/teach-imgui/internal/core"
)

func sampleRecipe() (core.ProductOptions, core.LightSetting, core.SensorNoise, []core.DefectStroke, []core.SceneStroke) {
	product := core.DefaultProductOptions()
	light := core.DefaultLightSetting()
	light.Channels = []core.LightChannel{
		{Color: core.LightRed, Value: 512, Enable: true},
		{Color: core.LightGreen, Value: 128, Enable: true},
		{Color: core.LightBlue, Value: 64, Enable: true},
	}
	ct := core.DefaultCrosstalk
	light.Crosstalk = &ct
	noise := core.SensorNoise{ReadSigma: 1.5, ShotGain: 0.25, Seed: 7}
	defects := []core.DefectStroke{
		{
			Kind:      core.Bright,
			Shape:     core.ShapeSpot,
			X:         10.5,
			Y:         20.25,
			Radius:    4,
			DeltaGv:   30,
			Softness:  0.5,
			Amplitude: core.AmpAbsolute,
			Profile:   core.ProfileLinear,
			Grain:     0.1,
			Seed:      3,
		},
		{
			Kind:      core.Dark,
			Shape:     core.ShapePolygon,
			Softness:  0.25,
			Amplitude: core.AmpTarget,
			TargetGv:  40,
			HasTarget: true,
			Profile:   core.ProfileGaussian,
			Grain:     0.2,
			Seed:      9,
			Points:    []core.Point{{X: 1, Y: 1}, {X: 8, Y: 1}, {X: 8, Y: 6}, {X: 1, Y: 6}},
			HasColor:  true,
			Color:     [3]float64{1, 0.5, 0},
		},
	}
	scene := []core.SceneStroke{
		{Mode: core.SceneTexture, Shape: core.ShapeRect, X: 2, Y: 3, X2: 12, Y2: 9, Factor: 1.5},
	}
	return product, light, noise, defects, scene
}

func TestRecipeRoundTrip(t *testing.T) {
	product, light, noise, defects, scene := sampleRecipe()
	r := NewRecipe(product, light, noise, defects, scene)
	data, err := json.Marshal(r)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var back Recipe
	if err := json.Unmarshal(data, &back); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	got := back.Core()
	if !reflect.DeepEqual(got.Product, product) {
		t.Errorf("product mismatch:\n got %+v\nwant %+v", got.Product, product)
	}
	if !reflect.DeepEqual(got.Light, light) {
		t.Errorf("light mismatch:\n got %+v\nwant %+v", got.Light, light)
	}
	if !reflect.DeepEqual(got.Noise, noise) {
		t.Errorf("noise mismatch:\n got %+v\nwant %+v", got.Noise, noise)
	}
	if !reflect.DeepEqual(got.Defects, defects) {
		t.Errorf("defects mismatch:\n got %+v\nwant %+v", got.Defects, defects)
	}
	if !reflect.DeepEqual(got.Scene, scene) {
		t.Errorf("scene mismatch:\n got %+v\nwant %+v", got.Scene, scene)
	}
}

func TestNewRecipeEmptySlices(t *testing.T) {
	product := core.DefaultProductOptions()
	light := core.DefaultLightSetting()
	r := NewRecipe(product, light, core.NoNoise, nil, nil)
	data, err := json.Marshal(r)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	s := string(data)
	if !strings.Contains(s, `"defects":[]`) {
		t.Errorf("missing empty defects array in %s", s)
	}
	if !strings.Contains(s, `"sceneStrokes":[]`) {
		t.Errorf("missing empty sceneStrokes array in %s", s)
	}
}

func validMeta() *Meta {
	product, light, noise, defects, scene := sampleRecipe()
	return &Meta{
		Format:   Format,
		Version:  Version,
		Producer: "test",
		Created:  "2026-01-01T00:00:00Z",
		Model:    ModelRef{ID: "m1", Host: "h", Model: "x", Side: "front", Version: "1"},
		Width:    64,
		Height:   48,
		Recipe:   NewRecipe(product, light, noise, defects, scene),
		Images: []Image{
			{Channel: ChannelRed, Kind: KindTest, File: "red.png", Width: 64, Height: 48, SHA256: strings.Repeat("a", 64)},
		},
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*Meta)
	}{
		{"valid", func(m *Meta) {}},
		{"bad format", func(m *Meta) { m.Format = "other" }},
		{"version 2", func(m *Meta) { m.Version = 2 }},
		{"zero width", func(m *Meta) { m.Width = 0 }},
		{"polygon 2 points", func(m *Meta) { m.Recipe.Defects[1].Points = m.Recipe.Defects[1].Points[:2] }},
		{"bad shape", func(m *Meta) { m.Recipe.Defects[0].Shape = 9 }},
		{"image channel alpha", func(m *Meta) { m.Images[0].Channel = "alpha" }},
		{"image file parent", func(m *Meta) { m.Images[0].File = "../x.png" }},
		{"image file subdir", func(m *Meta) { m.Images[0].File = "a/b.png" }},
		{"sha256 short", func(m *Meta) { m.Images[0].SHA256 = "abc" }},
		{"image size mismatch", func(m *Meta) { m.Images[0].Width = 32 }},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m := validMeta()
			tc.mutate(m)
			err := m.Validate()
			if tc.name == "valid" {
				if err != nil {
					t.Fatalf("valid meta rejected: %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("expected error for %s", tc.name)
			}
		})
	}
}

func TestSetImageOrder(t *testing.T) {
	var m Meta
	m.SetImage(Image{Channel: ChannelGreen, Kind: KindTest, File: "green.png", SHA256: "g"})
	m.SetImage(Image{Channel: ChannelRed, Kind: KindGood, File: "red-good.png", SHA256: "rg"})
	m.SetImage(Image{Channel: ChannelRed, Kind: KindTest, File: "red.png", SHA256: "rt"})
	m.SetImage(Image{Channel: ChannelBlue, Kind: KindTest, File: "blue.png", SHA256: "bt"})
	m.SetImage(Image{Channel: ChannelRed, Kind: KindTest, File: "red.png", SHA256: "rt2"})
	if len(m.Images) != 4 {
		t.Fatalf("got %d images, want 4", len(m.Images))
	}
	want := []struct {
		channel string
		kind    string
		hash    string
	}{
		{ChannelRed, KindTest, "rt2"},
		{ChannelRed, KindGood, "rg"},
		{ChannelGreen, KindTest, "g"},
		{ChannelBlue, KindTest, "bt"},
	}
	for i, w := range want {
		got := m.Images[i]
		if got.Channel != w.channel || got.Kind != w.kind || got.SHA256 != w.hash {
			t.Errorf("image %d = %s/%s/%s, want %s/%s/%s", i, got.Channel, got.Kind, got.SHA256, w.channel, w.kind, w.hash)
		}
	}
}

func TestChannelNames(t *testing.T) {
	pairs := []struct {
		ch   core.CameraChannel
		name string
	}{
		{core.ChRed, ChannelRed},
		{core.ChGreen, ChannelGreen},
		{core.ChBlue, ChannelBlue},
	}
	for _, p := range pairs {
		if got := ChannelName(p.ch); got != p.name {
			t.Errorf("ChannelName(%d) = %q, want %q", p.ch, got, p.name)
		}
		back, err := ParseChannel(p.name)
		if err != nil {
			t.Fatalf("ParseChannel(%q): %v", p.name, err)
		}
		if back != p.ch {
			t.Errorf("ParseChannel(%q) = %d, want %d", p.name, back, p.ch)
		}
	}
	if _, err := ParseChannel("RED"); err != nil {
		t.Errorf("ParseChannel(RED): %v", err)
	}
	if _, err := ParseChannel("x"); err == nil {
		t.Errorf("ParseChannel(x) should fail")
	}
	if got := ImageFile("red", "test"); got != "red.png" {
		t.Errorf("ImageFile(red,test) = %q", got)
	}
	if got := ImageFile("blue", "good"); got != "blue-good.png" {
		t.Errorf("ImageFile(blue,good) = %q", got)
	}
}
