package camera

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"afvi/teach-imgui/internal/core"
	"afvi/teach-imgui/internal/model"
	"afvi/teach-imgui/internal/model/modeltest"
	"afvi/teach-imgui/internal/snap"
)

func newScene(t *testing.T) *model.Scene {
	t.Helper()
	ref, err := modeltest.WriteReference(t.TempDir())
	if err != nil {
		t.Fatalf("WriteReference: %v", err)
	}
	infos := model.ScanHost("FM1", filepath.Join(ref.Root, "FM1"))
	if len(infos) == 0 {
		t.Fatalf("no models scanned")
	}
	scene, err := model.LoadScene(infos[0])
	if err != nil {
		t.Fatalf("LoadScene: %v", err)
	}
	return scene
}

func TestDefaultRecipe(t *testing.T) {
	scene := newScene(t)
	recipe := DefaultRecipe(scene, 3, 7)
	if len(recipe.Defects) < 1 || len(recipe.Defects) > 3 {
		t.Fatalf("got %d defects, want 1..3", len(recipe.Defects))
	}
	if len(recipe.Light.Channels) != 4 {
		t.Fatalf("got %d light channels, want 4", len(recipe.Light.Channels))
	}
	for i := 0; i < 3; i++ {
		c := recipe.Light.Channels[i]
		if !c.Enable || c.Value != 512 {
			t.Fatalf("channel %d: enable=%v value=%v, want true/512", i, c.Enable, c.Value)
		}
	}
	if recipe.Light.Channels[3].Enable {
		t.Fatalf("white light should be disabled")
	}
	cr := recipe.Core()
	if cr.Product != core.DefaultProductOptions() {
		t.Fatalf("product %+v, want default", cr.Product)
	}
	if cr.Noise.Seed != 1 {
		t.Fatalf("noise seed %d, want 1", cr.Noise.Seed)
	}
}

func TestRenderMatchesCore(t *testing.T) {
	scene := newScene(t)
	recipe := DefaultRecipe(scene, 3, 7)
	res, err := Render(scene, recipe, Options{})
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	cr := recipe.Core()
	ls, err := core.NewLightScene(scene.Median, scene.Material)
	if err != nil {
		t.Fatalf("NewLightScene: %v", err)
	}
	core.ApplyProductVariation(ls, cr.Product)
	light := cr.Light
	_, wantTest := core.RenderDefectPair(ls, &light, core.ChRed, cr.Scene, cr.Defects, cr.Noise)
	wantGood := core.RenderGoodSample(ls, &light, core.ChRed, core.GoodSampleNoise(cr.Noise))

	gotTest := res.Frames[ImageKey{Channel: snap.ChannelRed, Kind: snap.KindTest}]
	if !bytes.Equal(gotTest.Pix, wantTest.Data) {
		t.Fatalf("test frame differs from core")
	}
	goodRes, err := Render(scene, recipe, Options{Good: true})
	if err != nil {
		t.Fatalf("Render good: %v", err)
	}
	gotGood := goodRes.Frames[ImageKey{Channel: snap.ChannelRed, Kind: snap.KindGood}]
	if !bytes.Equal(gotGood.Pix, wantGood.Data) {
		t.Fatalf("good frame differs from core")
	}
}

func TestGoodDiffersOnlyAtDefects(t *testing.T) {
	scene := newScene(t)
	recipe := DefaultRecipe(scene, 3, 7)
	res, err := Render(scene, recipe, Options{Good: true})
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	test := res.Frames[ImageKey{Channel: snap.ChannelRed, Kind: snap.KindTest}]
	good := res.Frames[ImageKey{Channel: snap.ChannelRed, Kind: snap.KindGood}]
	diff := 0
	for i := range test.Pix {
		if test.Pix[i] == good.Pix[i] {
			continue
		}
		diff++
		x := i % test.Width
		y := i / test.Width
		inside := false
		for _, d := range recipe.Defects {
			r := d.Radius + 2
			if float64(x) >= d.X-r && float64(x) <= d.X+r && float64(y) >= d.Y-r && float64(y) <= d.Y+r {
				inside = true
				break
			}
		}
		if !inside {
			t.Fatalf("pixel (%d,%d) differs outside every defect", x, y)
		}
	}
	if diff < 1 || diff >= 600 {
		t.Fatalf("got %d differing pixels, want 1..599", diff)
	}
}

func TestRenderDeterministic(t *testing.T) {
	scene := newScene(t)
	recipe := DefaultRecipe(scene, 3, 7)
	a, err := Render(scene, recipe, Options{Channels: []core.CameraChannel{core.ChRed, core.ChGreen, core.ChBlue}, Good: true})
	if err != nil {
		t.Fatalf("Render a: %v", err)
	}
	b, err := Render(scene, recipe, Options{Channels: []core.CameraChannel{core.ChRed, core.ChGreen, core.ChBlue}, Good: true})
	if err != nil {
		t.Fatalf("Render b: %v", err)
	}
	if len(a.Frames) != len(b.Frames) {
		t.Fatalf("frame count %d vs %d", len(a.Frames), len(b.Frames))
	}
	for key, fa := range a.Frames {
		fb, ok := b.Frames[key]
		if !ok {
			t.Fatalf("missing frame %+v", key)
		}
		if snap.PixelHash(fa) != snap.PixelHash(fb) {
			t.Fatalf("frame %+v differs", key)
		}
	}
}

func TestRenderChannels(t *testing.T) {
	scene := newScene(t)
	recipe := DefaultRecipe(scene, 3, 7)
	res, err := Render(scene, recipe, Options{
		Channels: []core.CameraChannel{core.ChRed, core.ChGreen, core.ChBlue, core.ChRed},
		Good:     true,
	})
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if len(res.Frames) != 6 {
		t.Fatalf("got %d frames, want 6", len(res.Frames))
	}
	if len(res.Meta.Images) != 6 {
		t.Fatalf("got %d meta images, want 6", len(res.Meta.Images))
	}
	want := []struct{ channel, kind string }{
		{snap.ChannelRed, snap.KindTest},
		{snap.ChannelRed, snap.KindGood},
		{snap.ChannelGreen, snap.KindTest},
		{snap.ChannelGreen, snap.KindGood},
		{snap.ChannelBlue, snap.KindTest},
		{snap.ChannelBlue, snap.KindGood},
	}
	for i, w := range want {
		img := res.Meta.Images[i]
		if img.Channel != w.channel || img.Kind != w.kind {
			t.Fatalf("image %d is %s/%s, want %s/%s", i, img.Channel, img.Kind, w.channel, w.kind)
		}
	}
	hashes := map[string]string{}
	for _, ch := range []string{snap.ChannelRed, snap.ChannelGreen, snap.ChannelBlue} {
		f := res.Frames[ImageKey{Channel: ch, Kind: snap.KindTest}]
		hashes[ch] = snap.PixelHash(f)
	}
	if hashes[snap.ChannelRed] == hashes[snap.ChannelGreen] ||
		hashes[snap.ChannelRed] == hashes[snap.ChannelBlue] ||
		hashes[snap.ChannelGreen] == hashes[snap.ChannelBlue] {
		t.Fatalf("test frames should differ per channel")
	}
}

func TestSaveAndMerge(t *testing.T) {
	scene := newScene(t)
	dir := t.TempDir()
	recipe := DefaultRecipe(scene, 3, 7)
	red, err := Render(scene, recipe, Options{Channels: []core.CameraChannel{core.ChRed}})
	if err != nil {
		t.Fatalf("Render red: %v", err)
	}
	if err := Save(dir, red); err != nil {
		t.Fatalf("Save red: %v", err)
	}
	green, err := Render(scene, recipe, Options{Channels: []core.CameraChannel{core.ChGreen}})
	if err != nil {
		t.Fatalf("Render green: %v", err)
	}
	if err := Save(dir, green); err != nil {
		t.Fatalf("Save green: %v", err)
	}
	meta, _, err := snap.ReadMeta(dir)
	if err != nil {
		t.Fatalf("ReadMeta: %v", err)
	}
	if len(meta.Images) != 2 {
		t.Fatalf("got %d images, want 2", len(meta.Images))
	}
	for _, img := range meta.Images {
		if _, err := snap.ReadImage(dir, meta, img.Channel, img.Kind); err != nil {
			t.Fatalf("ReadImage %s/%s: %v", img.Channel, img.Kind, err)
		}
	}
	other := DefaultRecipe(scene, 4, 7)
	blue, err := Render(scene, other, Options{Channels: []core.CameraChannel{core.ChBlue}})
	if err != nil {
		t.Fatalf("Render blue: %v", err)
	}
	if err := Save(dir, blue); err != nil {
		t.Fatalf("Save blue: %v", err)
	}
	meta, _, err = snap.ReadMeta(dir)
	if err != nil {
		t.Fatalf("ReadMeta after: %v", err)
	}
	if len(meta.Images) != 1 || meta.Images[0].Channel != snap.ChannelBlue {
		t.Fatalf("got %+v, want only blue", meta.Images)
	}
}

func TestSaveReadable(t *testing.T) {
	scene := newScene(t)
	dir := t.TempDir()
	recipe := DefaultRecipe(scene, 3, 7)
	res, err := Render(scene, recipe, Options{})
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if err := Save(dir, res); err != nil {
		t.Fatalf("Save: %v", err)
	}
	meta, _, err := snap.ReadMeta(dir)
	if err != nil {
		t.Fatalf("ReadMeta: %v", err)
	}
	if err := meta.Validate(); err != nil {
		t.Fatalf("Validate: %v", err)
	}
	if meta.Producer != Producer {
		t.Fatalf("producer %q, want %q", meta.Producer, Producer)
	}
	if meta.Model.ID != "FM1/synthetic-1" {
		t.Fatalf("model id %q", meta.Model.ID)
	}
	if meta.Width != 96 || meta.Height != 64 {
		t.Fatalf("size %dx%d, want 96x64", meta.Width, meta.Height)
	}
}

func TestRecipeOnlyIsSmall(t *testing.T) {
	scene := newScene(t)
	dir := t.TempDir()
	recipe := DefaultRecipe(scene, 3, 7)
	meta := NewMeta(scene, recipe)
	path := filepath.Join(dir, "d.json.gz")
	if err := snap.WriteMeta(path, &meta); err != nil {
		t.Fatalf("WriteMeta: %v", err)
	}
	st, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	if st.Size() >= 4096 {
		t.Fatalf("recipe file is %d bytes, want < 4096", st.Size())
	}
	back, _, err := snap.ReadMeta(path)
	if err != nil {
		t.Fatalf("ReadMeta: %v", err)
	}
	a, _ := json.Marshal(meta.Recipe)
	b, _ := json.Marshal(back.Recipe)
	if string(a) != string(b) {
		t.Fatalf("recipe changed after round trip")
	}
}

func TestSceneFromMeta(t *testing.T) {
	scene := newScene(t)
	recipe := DefaultRecipe(scene, 3, 7)
	meta := NewMeta(scene, recipe)
	loaded, err := SceneFromMeta(&meta)
	if err != nil {
		t.Fatalf("SceneFromMeta: %v", err)
	}
	if loaded.W != scene.W || loaded.H != scene.H {
		t.Fatalf("size %dx%d, want %dx%d", loaded.W, loaded.H, scene.W, scene.H)
	}
	meta.Model.SR = filepath.Join(t.TempDir(), "missing.tif")
	if _, err := SceneFromMeta(&meta); err == nil {
		t.Fatalf("SceneFromMeta should fail with a missing SR path")
	}
}
