package camera

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"afvi/teach-imgui/internal/core"
	"afvi/teach-imgui/internal/snap"
)

func TestRenderWarpMovesTheProduct(t *testing.T) {
	scene := newScene(t)
	plain := DefaultRecipe(scene, 3, 7).WithWarp(core.IdentityWarp)
	moved := plain.WithWarp(core.WarpParams{Tx: 6, Ty: 4, Scale: 1})
	if moved.Warp == nil {
		t.Fatalf("warp not stored")
	}
	res0, err := Render(scene, plain, Options{Good: true})
	if err != nil {
		t.Fatal(err)
	}
	res1, err := Render(scene, moved, Options{Good: true})
	if err != nil {
		t.Fatal(err)
	}
	k := ImageKey{Channel: snap.ChannelRed, Kind: snap.KindTest}
	a, b := res0.Frames[k], res1.Frames[k]
	w, h := a.Width, a.Height
	if b.Width != w || b.Height != h {
		t.Fatalf("size changed: %dx%d vs %dx%d", b.Width, b.Height, w, h)
	}
	for y := 0; y < h-4; y++ {
		for x := 0; x < w-6; x++ {
			if b.Pix[(y+4)*w+(x+6)] != a.Pix[y*w+x] {
				t.Fatalf("pixel (%d,%d) did not move by (6,4)", x, y)
			}
		}
	}
	if snap.PixelHash(a) == snap.PixelHash(b) {
		t.Fatalf("a warped capture must differ from the plain one")
	}
	// 良品样本是示教基准，始终在标称位置。
	gk := ImageKey{Channel: snap.ChannelRed, Kind: snap.KindGood}
	if !bytes.Equal(res0.Frames[gk].Pix, res1.Frames[gk].Pix) {
		t.Fatalf("the good sample must not move")
	}
	// 与进程内直接调用核心一致。
	cr := moved.Core()
	ls, err := BuildLightScene(scene, cr.Product)
	if err != nil {
		t.Fatal(err)
	}
	light := cr.Light
	_, want := core.RenderDefectPairWarped(ls, &light, core.ChRed, cr.Scene, cr.Defects, cr.Noise, cr.Warp)
	if !bytes.Equal(b.Pix, want.Data) {
		t.Fatalf("camera differs from core.RenderDefectPairWarped")
	}
	// 带 warp 的配方存盘后可读，且 warp 随描述保存。
	dir := t.TempDir()
	if err := Save(dir, res1); err != nil {
		t.Fatal(err)
	}
	meta, _, err := snap.ReadMeta(dir)
	if err != nil {
		t.Fatalf("ReadMeta: %v", err)
	}
	if meta.Recipe.Warp == nil || meta.Recipe.Warp.Tx != 6 || meta.Recipe.Warp.Ty != 4 {
		t.Fatalf("warp lost: %+v", meta.Recipe.Warp)
	}
}

func TestPlainRecipeJSONHasNoWarp(t *testing.T) {
	scene := newScene(t)
	data, err := json.Marshal(DefaultRecipe(scene, 3, 7).WithWarp(core.IdentityWarp))
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(data, []byte("warp")) {
		t.Fatalf("a recipe without warp must not mention it: %s", data)
	}
}

func TestMainWarpFlags(t *testing.T) {
	root := writeRef(t)
	dir := t.TempDir()
	code, _, errOut := run(t, "-ref", root, "-model", "FM1/synthetic-1", "-out", dir,
		"-dx", "6", "-dy", "-4", "-rot", "0.5", "-scale", "1.01", "-shear-x", "0.01",
		"-tilt-x", "0.1", "-tilt-y", "-0.05", "-bend", "5", "-bend-wave", "800")
	if code != 0 {
		t.Fatalf("code=%d stderr=%s", code, errOut)
	}
	meta, _, err := snap.ReadMeta(dir)
	if err != nil {
		t.Fatalf("ReadMeta: %v", err)
	}
	w := meta.Recipe.Warp
	if w == nil {
		t.Fatalf("no warp in the description")
	}
	want := snap.Warp{Tx: 6, Ty: -4, ThetaDeg: 0.5, Scale: 1.01, ShearX: 0.01, TiltX: 0.1, TiltY: -0.05, BendAmplitude: 5, BendWavelength: 800}
	if *w != want {
		t.Fatalf("warp %+v, want %+v", *w, want)
	}
	code, out, errOut := run(t, "-info", dir)
	if code != 0 {
		t.Fatalf("info: code=%d stderr=%s", code, errOut)
	}
	if !strings.Contains(out, "warp: dx=6 dy=-4 rot=0.5 scale=1.01") {
		t.Fatalf("info lacks the warp line: %q", out)
	}
	// 没有任何 warp 参数、但带 -aligned 时描述里没有 warp。
	plain := t.TempDir()
	if code, _, errOut := run(t, "-ref", root, "-model", "FM1/synthetic-1", "-out", plain, "-aligned"); code != 0 {
		t.Fatalf("plain: code=%d stderr=%s", code, errOut)
	}
	pm, _, err := snap.ReadMeta(plain)
	if err != nil {
		t.Fatal(err)
	}
	if pm.Recipe.Warp != nil {
		t.Fatalf("plain capture has a warp: %+v", pm.Recipe.Warp)
	}
	// 只有显式给出的参数才改写配方：-in 重新渲染描述并改位置。
	again := t.TempDir()
	code, _, errOut = run(t, "-in", dir, "-out", again, "-dx", "0", "-dy", "0", "-rot", "0", "-scale", "1",
		"-shear-x", "0", "-tilt-x", "0", "-tilt-y", "0", "-bend", "0")
	if code != 0 {
		t.Fatalf("re-render: code=%d stderr=%s", code, errOut)
	}
	rm, _, err := snap.ReadMeta(again)
	if err != nil {
		t.Fatal(err)
	}
	if rm.Recipe.Warp != nil {
		t.Fatalf("zeroing every warp flag must remove the warp, got %+v", rm.Recipe.Warp)
	}
}

func TestMainWarpFlagErrors(t *testing.T) {
	root := writeRef(t)
	for _, args := range [][]string{
		{"-tilt-x", "0.7"},
		{"-dx", "5000"},
		{"-rot", "40"},
		{"-scale", "3"},
		{"-bend", "5", "-bend-wave", "10"},
	} {
		full := append([]string{"-ref", root, "-model", "FM1/synthetic-1", "-out", t.TempDir()}, args...)
		code, _, errOut := run(t, full...)
		if code != 2 {
			t.Fatalf("%v: code=%d, want 2 (stderr %q)", args, code, errOut)
		}
		if !strings.Contains(errOut, "warp") {
			t.Fatalf("%v: stderr %q does not mention the warp", args, errOut)
		}
	}
}
