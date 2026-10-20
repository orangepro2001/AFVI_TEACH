package app

import (
	"reflect"
	"testing"

	"afvi/teach-imgui/internal/core"
	snapfmt "afvi/teach-imgui/internal/snap"
)

func TestAlignTemplateLightNilMeta(t *testing.T) {
	appLight := core.DefaultLightSetting()
	got := alignTemplateLight(appLight, nil)
	if !reflect.DeepEqual(got, appLight) {
		t.Fatalf("light = %+v, want %+v", got, appLight)
	}
	got.Channels[0].Value = 123
	if appLight.Channels[0].Value == 123 {
		t.Fatal("returned light aliases the argument's Channels")
	}
}

func TestAlignTemplateLightFromMeta(t *testing.T) {
	light := core.DefaultLightSetting()
	light.Channels = []core.LightChannel{
		{Color: core.LightRed, Value: 512, Enable: false},
		{Color: core.LightBlue, Value: 300, Enable: true},
	}
	r := snapfmt.NewRecipe(core.DefaultProductOptions(), light, core.SensorNoise{}, nil, nil)
	meta := &snapfmt.Meta{Recipe: r}
	got := alignTemplateLight(core.DefaultCaptureLight(), meta)
	var blue, red *core.LightChannel
	for i := range got.Channels {
		switch got.Channels[i].Color {
		case core.LightBlue:
			blue = &got.Channels[i]
		case core.LightRed:
			red = &got.Channels[i]
		}
	}
	if blue == nil || !blue.Enable || blue.Value != 300 {
		t.Fatalf("blue channel = %+v, want enabled at 300", blue)
	}
	if red == nil || red.Enable {
		t.Fatalf("red channel = %+v, want disabled", red)
	}
}

func TestImageFlat(t *testing.T) {
	if !imageFlat(nil) {
		t.Error("imageFlat(nil) = false, want true")
	}
	zero := core.NewGray(8, 8, 0)
	if !imageFlat(zero) {
		t.Error("all-zero image is not flat")
	}
	constant := core.NewGray(8, 8, 100)
	if !imageFlat(constant) {
		t.Error("constant 100 image is not flat")
	}
	board := core.NewGray(8, 8, 0)
	for y := 0; y < 8; y++ {
		for x := 0; x < 8; x++ {
			if (x+y)%2 == 0 {
				board.Data[y*8+x] = 255
			}
		}
	}
	if imageFlat(board) {
		t.Error("checkerboard 0/255 is flat")
	}

	sparse := core.NewGray(100, 100, 0)
	for i := 0; i < 20; i++ {
		sparse.Data[i*37] = 255
	}
	if !imageFlat(sparse) {
		t.Error("black image with 20 bright specks is not flat")
	}

	halves := core.NewGray(100, 100, 0)
	for y := 0; y < 100; y++ {
		for x := 0; x < 100; x++ {
			if x < 50 {
				halves.Data[y*100+x] = 40
			} else {
				halves.Data[y*100+x] = 80
			}
		}
	}
	if imageFlat(halves) {
		t.Error("image with 40/80 halves is flat")
	}
}

func TestTeachDefaultLightIsRGB(t *testing.T) {
	teach := newTestApp(t, ProfileTeach)
	want := core.DefaultCaptureLight()
	if len(teach.Light.Channels) != len(want.Channels) || !reflect.DeepEqual(teach.Light.Channels, want.Channels) {
		t.Fatalf("teach light channels = %+v, want %+v", teach.Light.Channels, want.Channels)
	}
	if !reflect.DeepEqual(teach.Light.Reflectance, want.Reflectance) ||
		teach.Light.Gain != want.Gain || teach.Light.Offset != want.Offset ||
		teach.Light.TextureStrength != want.TextureStrength {
		t.Errorf("teach light = %+v, want %+v", teach.Light, want)
	}

	studio := newTestApp(t, ProfileStudio)
	// The studio starts with the same light, so a scenario it publishes has contrast in every channel.
	if !reflect.DeepEqual(studio.Light.Channels, want.Channels) {
		t.Fatalf("studio light channels = %+v, want %+v", studio.Light.Channels, want.Channels)
	}
}
