package app

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/image/bmp"
	"golang.org/x/image/tiff"

	"afvi/teach-imgui/internal/core"
)

// writePNG encodes img as PNG into dir/name and returns the path.
func writePNG(t *testing.T, dir, name string, img image.Image) string {
	t.Helper()
	path := filepath.Join(dir, name)
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("create %s: %v", name, err)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		t.Fatalf("encode %s: %v", name, err)
	}
	return path
}

func TestReadGrayImage(t *testing.T) {
	dir := t.TempDir()

	gray := image.NewGray(image.Rect(0, 0, 3, 2))
	gray.Pix = []uint8{0, 10, 20, 30, 40, 50}
	grayPath := writePNG(t, dir, "gray.png", gray)
	got, err := readGrayImage(grayPath)
	if err != nil {
		t.Fatalf("readGrayImage(gray): %v", err)
	}
	if got.Width != 3 || got.Height != 2 {
		t.Fatalf("gray size %dx%d, want 3x2", got.Width, got.Height)
	}
	for i, want := range gray.Pix {
		if got.Data[i] != want {
			t.Errorf("gray pixel %d = %d, want %d", i, got.Data[i], want)
		}
	}

	rgb := image.NewNRGBA(image.Rect(0, 0, 3, 1))
	rgb.Set(0, 0, color.NRGBA{R: 255, A: 255})
	rgb.Set(1, 0, color.NRGBA{G: 255, A: 255})
	rgb.Set(2, 0, color.NRGBA{B: 255, A: 255})
	rgbPath := writePNG(t, dir, "rgb.png", rgb)
	got, err = readGrayImage(rgbPath)
	if err != nil {
		t.Fatalf("readGrayImage(rgb): %v", err)
	}
	wantLum := []uint8{
		color.GrayModel.Convert(color.NRGBA{R: 255, A: 255}).(color.Gray).Y,
		color.GrayModel.Convert(color.NRGBA{G: 255, A: 255}).(color.Gray).Y,
		color.GrayModel.Convert(color.NRGBA{B: 255, A: 255}).(color.Gray).Y,
	}
	for i, want := range wantLum {
		if got.Data[i] != want {
			t.Errorf("rgb pixel %d = %d, want %d", i, got.Data[i], want)
		}
	}

	g16 := image.NewGray16(image.Rect(0, 0, 1, 1))
	g16.SetGray16(0, 0, color.Gray16{Y: 0xABCD})
	g16Path := writePNG(t, dir, "gray16.png", g16)
	got, err = readGrayImage(g16Path)
	if err != nil {
		t.Fatalf("readGrayImage(gray16): %v", err)
	}
	if got.Data[0] != 0xAB {
		t.Errorf("gray16 pixel = %#x, want 0xAB", got.Data[0])
	}

	bmpPath := filepath.Join(dir, "img.bmp")
	bf, err := os.Create(bmpPath)
	if err != nil {
		t.Fatalf("create bmp: %v", err)
	}
	if err := bmp.Encode(bf, gray); err != nil {
		t.Fatalf("encode bmp: %v", err)
	}
	bf.Close()
	got, err = readGrayImage(bmpPath)
	if err != nil {
		t.Fatalf("readGrayImage(bmp): %v", err)
	}
	if got.Width != 3 || got.Height != 2 || got.Data[0] != 0 {
		t.Errorf("bmp image %dx%d first %d, want 3x2 first 0", got.Width, got.Height, got.Data[0])
	}

	tiffPath := filepath.Join(dir, "img.tiff")
	tf, err := os.Create(tiffPath)
	if err != nil {
		t.Fatalf("create tiff: %v", err)
	}
	if err := tiff.Encode(tf, gray, nil); err != nil {
		t.Fatalf("encode tiff: %v", err)
	}
	tf.Close()
	got, err = readGrayImage(tiffPath)
	if err != nil {
		t.Fatalf("readGrayImage(tiff): %v", err)
	}
	if got.Width != 3 || got.Height != 2 || got.Data[5] != 50 {
		t.Errorf("tiff image %dx%d last %d, want 3x2 last 50", got.Width, got.Height, got.Data[5])
	}

	if _, err := readGrayImage(filepath.Join(dir, "missing.png")); err == nil {
		t.Error("readGrayImage(missing) = nil error, want an error")
	}
	badPath := filepath.Join(dir, "bad.png")
	if err := os.WriteFile(badPath, []byte("not an image"), 0o644); err != nil {
		t.Fatalf("write bad.png: %v", err)
	}
	if _, err := readGrayImage(badPath); err == nil {
		t.Error("readGrayImage(text file) = nil error, want an error")
	}
}

func TestManualModeDoesNotInspect(t *testing.T) {
	a := newTestApp(t, ProfileTeach)
	dp := defectPanelOf(a)
	if dp == nil {
		t.Fatal("no defect panel")
	}
	dp.manual = true
	dp.requestRun()
	pollJobs(t, a)
	if dp.test == nil {
		t.Fatal("test image is nil in manual mode")
	}
	if dp.lastReport != nil {
		t.Error("lastReport is not nil in manual mode")
	}
	if dp.learned[dp.channel] {
		t.Errorf("learned[%v] = true in manual mode, want false", dp.channel)
	}

	dp.requestInspect()
	pollJobs(t, a)
	if dp.lastReport == nil {
		t.Fatal("lastReport is nil after Inspect")
	}
	if !dp.learned[dp.channel] {
		t.Errorf("learned[%v] = false after Inspect, want true", dp.channel)
	}

	dp.renderRev++
	dp.requestRun()
	pollJobs(t, a)
	if dp.lastReport != nil {
		t.Error("lastReport is not nil after a manual-mode refresh")
	}

	dp.manual = false
	dp.requestRun()
	pollJobs(t, a)
	if dp.lastReport == nil {
		t.Error("lastReport is nil after leaving manual mode")
	}
}

func TestLoadImageReplacesSources(t *testing.T) {
	a := newTestApp(t, ProfileTeach)
	dp := defectPanelOf(a)
	if dp == nil {
		t.Fatal("no defect panel")
	}
	dir := t.TempDir()
	img := image.NewGray(image.Rect(0, 0, a.Scene.W, a.Scene.H))
	for y := 0; y < a.Scene.H; y++ {
		for x := 0; x < a.Scene.W; x++ {
			img.Pix[y*img.Stride+x] = uint8((x + y) % 256)
		}
	}
	path := writePNG(t, dir, "load.png", img)

	dp.liveFollow = true
	dp.requestLoadImage(path)
	pollJobs(t, a)
	if dp.user.img == nil {
		t.Fatalf("user image is nil: %s %s", dp.imgMsg.key, dp.imgMsg.text)
	}
	if dp.user.img.Width != a.Scene.W || dp.user.img.Height != a.Scene.H {
		t.Fatalf("loaded size %dx%d, want %dx%d", dp.user.img.Width, dp.user.img.Height, a.Scene.W, a.Scene.H)
	}
	for i := range img.Pix {
		if dp.user.img.Data[i] != img.Pix[i] {
			t.Fatalf("loaded pixel %d = %d, want %d", i, dp.user.img.Data[i], img.Pix[i])
		}
	}
	if dp.test != dp.user.img {
		t.Error("test image is not the loaded image")
	}
	if dp.liveFollow {
		t.Error("liveFollow is still true after loading an image")
	}
	if dp.snapSrc.on {
		t.Error("snap source is still on after loading an image")
	}
	if dp.imgMsg.key != "defect.manual.loaded" {
		t.Errorf("imgMsg.key = %q, want defect.manual.loaded", dp.imgMsg.key)
	}

	dp.manual = true
	dp.requestInspect()
	pollJobs(t, a)
	if dp.lastReport == nil {
		t.Fatal("lastReport is nil after inspecting the loaded image")
	}
	if !dp.learned[dp.channel] {
		t.Errorf("learned[%v] = false, want true (learned from the loaded image)", dp.channel)
	}

	dp.clearUserImage()
	if dp.user.img != nil {
		t.Error("user image is not nil after clearUserImage")
	}
}

func TestLoadImageRejectsWrongSize(t *testing.T) {
	a := newTestApp(t, ProfileTeach)
	dp := defectPanelOf(a)
	if dp == nil {
		t.Fatal("no defect panel")
	}
	dir := t.TempDir()
	img := image.NewGray(image.Rect(0, 0, a.Scene.W+3, a.Scene.H+2))
	path := writePNG(t, dir, "wrong.png", img)
	dp.requestLoadImage(path)
	pollJobs(t, a)
	if dp.user.img != nil {
		t.Error("user image is not nil after a size mismatch")
	}
	if dp.imgMsg.key != "defect.manual.sizeMismatch" || !dp.imgMsg.err {
		t.Errorf("imgMsg = %q err=%v, want defect.manual.sizeMismatch err=true", dp.imgMsg.key, dp.imgMsg.err)
	}
}

func TestSaveWithLoadedImageWritesOnlyParams(t *testing.T) {
	a := newTestApp(t, ProfileTeach)
	dp := defectPanelOf(a)
	if dp == nil {
		t.Fatal("no defect panel")
	}
	dir := t.TempDir()
	img := image.NewGray(image.Rect(0, 0, a.Scene.W, a.Scene.H))
	path := writePNG(t, dir, "load.png", img)
	dp.requestLoadImage(path)
	pollJobs(t, a)
	if dp.user.img == nil {
		t.Fatalf("user image is nil: %s %s", dp.imgMsg.key, dp.imgMsg.text)
	}
	out := filepath.Join(t.TempDir(), "s")
	dp.snapSrc.path, dp.snapSrc.pathAuto = out, false
	dp.requestSaveSession()
	pollJobs(t, a)
	if _, err := os.Stat(filepath.Join(out, "inspect.json")); err != nil {
		t.Fatalf("inspect.json missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(out, "snap.json")); err == nil {
		t.Error("snap.json was written for a loaded image")
	}
}

func TestManualKeysExist(t *testing.T) {
	keys := []string{
		"defect.section.manual",
		"defect.manual.mode",
		"defect.manual.note",
		"defect.manual.loadImage",
		"defect.manual.inspect",
		"defect.manual.clear",
		"defect.manual.srcImage",
		"defect.manual.learnNote",
		"defect.manual.noGolden",
		"defect.manual.pending",
		"defect.manual.done",
		"defect.manual.loaded",
		"defect.manual.loadError",
		"defect.manual.sizeMismatch",
		"defect.manual.noDialog",
		"defect.manual.dialogError",
	}
	for _, lang := range []Lang{LangZh, LangEn, LangKo} {
		for _, k := range keys {
			if dictionaries[lang][k] == "" {
				t.Errorf("dictionaries[%s][%q] is empty", lang, k)
			}
		}
	}
	_ = core.MatSR
}
