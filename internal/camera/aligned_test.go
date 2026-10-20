package camera

import (
	"path/filepath"
	"testing"

	"afvi/teach-imgui/internal/core"
	"afvi/teach-imgui/internal/model"
	"afvi/teach-imgui/internal/model/modeltest"
	"afvi/teach-imgui/internal/snap"
)

// alignedScene builds the synthetic model the other camera tests use.
func alignedScene(t *testing.T) *model.Scene {
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

func readFrame(t *testing.T, dir, channel, kind string) snap.Frame {
	t.Helper()
	meta, _, err := snap.ReadMeta(dir)
	if err != nil {
		t.Fatalf("ReadMeta(%s): %v", dir, err)
	}
	f, err := snap.ReadImage(dir, meta, channel, kind)
	if err != nil {
		t.Fatalf("ReadImage(%s, %s/%s): %v", dir, channel, kind, err)
	}
	return f
}

func TestAlignedDiffersFromDefault(t *testing.T) {
	root := writeRef(t)
	defaultDir := t.TempDir()
	alignedDir := t.TempDir()
	if code, _, errOut := run(t, "-ref", root, "-model", "FM1/synthetic-1", "-out", defaultDir, "-count", "3"); code != 0 {
		t.Fatalf("default: code=%d stderr=%s", code, errOut)
	}
	if code, _, errOut := run(t, "-ref", root, "-model", "FM1/synthetic-1", "-out", alignedDir, "-count", "3", "-aligned"); code != 0 {
		t.Fatalf("aligned: code=%d stderr=%s", code, errOut)
	}
	def := readFrame(t, defaultDir, snap.ChannelRed, snap.KindTest)
	ali := readFrame(t, alignedDir, snap.ChannelRed, snap.KindTest)
	if snap.PixelHash(def) == snap.PixelHash(ali) {
		t.Fatalf("the default capture must be misaligned, but equals the aligned one")
	}
}

func TestAlignedMatchesIdentityRecipe(t *testing.T) {
	scene := alignedScene(t)
	recipe := DefaultRecipe(scene, 3, 7).WithWarp(core.IdentityWarp)
	res, err := Render(scene, recipe, Options{Channels: []core.CameraChannel{core.ChRed}})
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	want := snap.PixelHash(res.Frames[ImageKey{Channel: snap.ChannelRed, Kind: snap.KindTest}])

	root := writeRef(t)
	dir := t.TempDir()
	if code, _, errOut := run(t, "-ref", root, "-model", "FM1/synthetic-1", "-out", dir, "-channels", "red", "-count", "3", "-aligned"); code != 0 {
		t.Fatalf("aligned: code=%d stderr=%s", code, errOut)
	}
	got := readFrame(t, dir, snap.ChannelRed, snap.KindTest)
	if snap.PixelHash(got) != want {
		t.Fatalf("aligned render differs from the identity recipe")
	}
}

func TestAlignedDescriptionWarp(t *testing.T) {
	root := writeRef(t)
	defaultDir := t.TempDir()
	alignedDir := t.TempDir()
	if code, _, errOut := run(t, "-ref", root, "-model", "FM1/synthetic-1", "-out", defaultDir, "-count", "3"); code != 0 {
		t.Fatalf("default: code=%d stderr=%s", code, errOut)
	}
	if code, _, errOut := run(t, "-ref", root, "-model", "FM1/synthetic-1", "-out", alignedDir, "-count", "3", "-aligned"); code != 0 {
		t.Fatalf("aligned: code=%d stderr=%s", code, errOut)
	}
	defMeta, _, err := snap.ReadMeta(defaultDir)
	if err != nil {
		t.Fatalf("ReadMeta(default): %v", err)
	}
	if defMeta.Recipe.Warp == nil {
		t.Fatalf("the default description has no warp")
	}
	want := core.DefaultCaptureWarp()
	w := *defMeta.Recipe.Warp
	if w.Tx != want.Tx || w.Ty != want.Ty || w.ThetaDeg != want.ThetaDeg || w.Scale != want.Scale ||
		w.ShearX != want.ShearX || w.ShearY != want.ShearY || w.TiltX != want.TiltX || w.TiltY != want.TiltY ||
		w.BendAmplitude != want.BendAmplitude || w.BendWavelength != want.BendWavelength {
		t.Fatalf("default warp %+v, want %+v", w, want)
	}
	aliMeta, _, err := snap.ReadMeta(alignedDir)
	if err != nil {
		t.Fatalf("ReadMeta(aligned): %v", err)
	}
	if aliMeta.Recipe.Warp != nil {
		t.Fatalf("the aligned description has a warp: %+v", aliMeta.Recipe.Warp)
	}
}

func TestAlignedIgnoredWithInput(t *testing.T) {
	root := writeRef(t)
	srcDir := t.TempDir()
	if code, _, errOut := run(t, "-ref", root, "-model", "FM1/synthetic-1", "-out", srcDir, "-count", "3",
		"-dx", "6", "-dy", "-4", "-rot", "0.5"); code != 0 {
		t.Fatalf("source: code=%d stderr=%s", code, errOut)
	}
	srcMeta, _, err := snap.ReadMeta(srcDir)
	if err != nil {
		t.Fatalf("ReadMeta(source): %v", err)
	}
	if srcMeta.Recipe.Warp == nil {
		t.Fatalf("the source description has no warp")
	}
	want := *srcMeta.Recipe.Warp

	outDir := t.TempDir()
	if code, _, errOut := run(t, "-in", filepath.Join(srcDir, snap.MetaFile), "-out", outDir, "-aligned"); code != 0 {
		t.Fatalf("re-render: code=%d stderr=%s", code, errOut)
	}
	outMeta, _, err := snap.ReadMeta(outDir)
	if err != nil {
		t.Fatalf("ReadMeta(out): %v", err)
	}
	if outMeta.Recipe.Warp == nil {
		t.Fatalf("-aligned with -in removed the warp")
	}
	if *outMeta.Recipe.Warp != want {
		t.Fatalf("warp changed with -aligned -in: %+v vs %+v", *outMeta.Recipe.Warp, want)
	}
}

func TestAlignedWithShiftKeepsOtherFields(t *testing.T) {
	root := writeRef(t)
	dir := t.TempDir()
	if code, _, errOut := run(t, "-ref", root, "-model", "FM1/synthetic-1", "-out", dir, "-count", "3",
		"-aligned", "-dx", "3"); code != 0 {
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
	want := snap.Warp{Tx: 3, Scale: 1}
	if *w != want {
		t.Fatalf("warp %+v, want %+v", *w, want)
	}
}
