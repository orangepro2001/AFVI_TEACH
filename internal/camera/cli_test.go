package camera

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"afvi/teach-imgui/internal/core"
	"afvi/teach-imgui/internal/model/modeltest"
	"afvi/teach-imgui/internal/snap"
)

func run(t *testing.T, args ...string) (int, string, string) {
	t.Helper()
	t.Setenv("AFVI_CONFIG", "")
	var out, errBuf bytes.Buffer
	code := Main(args, &out, &errBuf)
	return code, out.String(), errBuf.String()
}

func writeRef(t *testing.T) string {
	t.Helper()
	ref, err := modeltest.WriteReference(t.TempDir())
	if err != nil {
		t.Fatalf("WriteReference: %v", err)
	}
	return ref.Root
}

func TestMainList(t *testing.T) {
	root := writeRef(t)
	code, out, errOut := run(t, "-ref", root, "-list")
	if code != 0 {
		t.Fatalf("code=%d stderr=%s", code, errOut)
	}
	if !strings.Contains(out, "FM1/synthetic-1") {
		t.Fatalf("stdout=%q", out)
	}
}

func TestMainDefaultImage(t *testing.T) {
	root := writeRef(t)
	dir := t.TempDir()
	code, _, errOut := run(t, "-ref", root, "-model", "FM1/synthetic-1", "-out", dir, "-channels", "red,green", "-good", "-count", "3", "-aligned")
	if code != 0 {
		t.Fatalf("code=%d stderr=%s", code, errOut)
	}
	for _, name := range []string{"snap.json", "red.png", "green.png", "red-good.png", "green-good.png"} {
		if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
			t.Fatalf("missing %s: %v", name, err)
		}
	}
	meta, _, err := snap.ReadMeta(dir)
	if err != nil {
		t.Fatalf("ReadMeta: %v", err)
	}
	if len(meta.Images) != 4 {
		t.Fatalf("images=%d", len(meta.Images))
	}
	if n := len(meta.Recipe.Defects); n < 1 || n > 3 {
		t.Fatalf("defects=%d", n)
	}
	for _, img := range meta.Images {
		if _, err := snap.ReadImage(dir, meta, img.Channel, img.Kind); err != nil {
			t.Fatalf("ReadImage %s/%s: %v", img.Channel, img.Kind, err)
		}
	}
}

func TestMainRecipeOnlyThenRender(t *testing.T) {
	root := writeRef(t)
	dir := t.TempDir()
	desc := filepath.Join(dir, "desc.json.gz")
	code, _, errOut := run(t, "-ref", root, "-model", "FM1/synthetic-1", "-mode", "recipe", "-out", desc, "-count", "3")
	if code != 0 {
		t.Fatalf("code=%d stderr=%s", code, errOut)
	}
	st, err := os.Stat(desc)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if st.Size() >= 4096 {
		t.Fatalf("description is %d bytes", st.Size())
	}
	dir2 := t.TempDir()
	code, _, errOut = run(t, "-in", desc, "-out", dir2, "-channels", "blue")
	if code != 0 {
		t.Fatalf("code=%d stderr=%s", code, errOut)
	}
	meta, _, err := snap.ReadMeta(desc)
	if err != nil {
		t.Fatalf("ReadMeta: %v", err)
	}
	scene, err := SceneFromMeta(meta)
	if err != nil {
		t.Fatalf("SceneFromMeta: %v", err)
	}
	res, err := Render(scene, meta.Recipe, Options{Channels: []core.CameraChannel{core.ChBlue}})
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	want := snap.PixelHash(res.Frames[ImageKey{Channel: snap.ChannelBlue, Kind: snap.KindTest}])
	meta2, _, err := snap.ReadMeta(dir2)
	if err != nil {
		t.Fatalf("ReadMeta(dir2): %v", err)
	}
	got, err := snap.ReadImage(dir2, meta2, snap.ChannelBlue, snap.KindTest)
	if err != nil {
		t.Fatalf("ReadImage: %v", err)
	}
	if snap.PixelHash(got) != want {
		t.Fatalf("hash mismatch")
	}
}

func TestMainRerenderMergesChannels(t *testing.T) {
	root := writeRef(t)
	dir := t.TempDir()
	code, _, errOut := run(t, "-ref", root, "-model", "FM1/synthetic-1", "-out", dir, "-channels", "red", "-count", "3")
	if code != 0 {
		t.Fatalf("code=%d stderr=%s", code, errOut)
	}
	code, _, errOut = run(t, "-in", filepath.Join(dir, "snap.json"), "-out", dir, "-channels", "green")
	if code != 0 {
		t.Fatalf("code=%d stderr=%s", code, errOut)
	}
	meta, _, err := snap.ReadMeta(dir)
	if err != nil {
		t.Fatalf("ReadMeta: %v", err)
	}
	if _, ok := meta.FindImage(snap.ChannelRed, snap.KindTest); !ok {
		t.Fatalf("red test image missing")
	}
	if _, ok := meta.FindImage(snap.ChannelGreen, snap.KindTest); !ok {
		t.Fatalf("green test image missing")
	}
}

func TestMainInfo(t *testing.T) {
	root := writeRef(t)
	dir := t.TempDir()
	code, _, errOut := run(t, "-ref", root, "-model", "FM1/synthetic-1", "-out", dir, "-count", "3")
	if code != 0 {
		t.Fatalf("code=%d stderr=%s", code, errOut)
	}
	code, out, errOut := run(t, "-info", dir)
	if code != 0 {
		t.Fatalf("code=%d stderr=%s", code, errOut)
	}
	if !strings.Contains(out, "FM1/synthetic-1") {
		t.Fatalf("stdout=%q", out)
	}
	if !strings.Contains(out, "96x64") {
		t.Fatalf("stdout=%q", out)
	}
	if !strings.Contains(out, "defects") {
		t.Fatalf("stdout=%q", out)
	}
}

func TestMainOverrides(t *testing.T) {
	root := writeRef(t)
	dir := t.TempDir()
	code, _, errOut := run(t, "-ref", root, "-model", "FM1/synthetic-1", "-mode", "recipe", "-out", filepath.Join(dir, "d.json"), "-no-product", "-noise-read", "2", "-strength", "0.5", "-count", "3")
	if code != 0 {
		t.Fatalf("code=%d stderr=%s", code, errOut)
	}
	meta, _, err := snap.ReadMeta(filepath.Join(dir, "d.json"))
	if err != nil {
		t.Fatalf("ReadMeta: %v", err)
	}
	if meta.Recipe.Product.Enabled {
		t.Fatalf("product should be disabled")
	}
	if meta.Recipe.Noise.ReadSigma != 2 {
		t.Fatalf("readSigma=%g", meta.Recipe.Noise.ReadSigma)
	}
}

func TestMainBareRunUsesFirstModel(t *testing.T) {
	root := writeRef(t)
	dir := t.TempDir()
	code, out, errOut := run(t, "-ref", root, "-out", dir)
	if code != 0 {
		t.Fatalf("code=%d stderr=%s", code, errOut)
	}
	if !strings.Contains(errOut, "no -model given, using FM1/synthetic-1") {
		t.Fatalf("stderr=%q", errOut)
	}
	if !strings.Contains(out, "snap: FM1/synthetic-1") {
		t.Fatalf("stdout=%q", out)
	}
	for _, name := range []string{"snap.json", "red.png"} {
		if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
			t.Fatalf("missing %s: %v", name, err)
		}
	}
}

func TestMainUsageErrors(t *testing.T) {
	root := writeRef(t)
	if code, _, _ := run(t, "-ref", root, "-model", "FM1/synthetic-1", "-mode", "weird"); code != 2 {
		t.Fatalf("bad mode: code=%d", code)
	}
	if code, _, _ := run(t, "-ref", root, "-model", "FM1/synthetic-1", "-channels", "purple"); code != 2 {
		t.Fatalf("bad channels: code=%d", code)
	}
	if code, _, _ := run(t, "-ref", root, "-model", "FM1/synthetic-1", "-mode", "image", "-out", "x.json"); code != 2 {
		t.Fatalf("bad out: code=%d", code)
	}
	code, _, errOut := run(t, "-ref", root, "-model", "FM1/nope")
	if code != 1 {
		t.Fatalf("unknown model: code=%d", code)
	}
	if !strings.Contains(errOut, "not found") {
		t.Fatalf("stderr=%q", errOut)
	}
}
