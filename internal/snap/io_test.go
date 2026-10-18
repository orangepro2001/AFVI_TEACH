package snap

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"afvi/teach-imgui/internal/core"
)

func bigMeta(t *testing.T) *Meta {
	t.Helper()
	m := validMeta()
	m.Recipe.Defects = make([]Defect, 0, 200)
	for i := 0; i < 200; i++ {
		m.Recipe.Defects = append(m.Recipe.Defects, Defect{
			Kind:      i % 2,
			Shape:     0,
			X:         float64(i),
			Y:         float64(i) * 0.5,
			Radius:    3,
			DeltaGv:   10,
			Softness:  0.5,
			Amplitude: 0,
			Profile:   0,
			Seed:      uint32(i),
		})
	}
	return m
}

func TestWriteReadMetaJSONAndGzip(t *testing.T) {
	dir := t.TempDir()
	m := bigMeta(t)
	plain := filepath.Join(dir, "snap.json")
	gz := filepath.Join(dir, "snap.json.gz")
	if err := WriteMeta(plain, m); err != nil {
		t.Fatalf("WriteMeta plain: %v", err)
	}
	if err := WriteMeta(gz, m); err != nil {
		t.Fatalf("WriteMeta gz: %v", err)
	}
	plainData, err := os.ReadFile(plain)
	if err != nil {
		t.Fatalf("read plain: %v", err)
	}
	gzData, err := os.ReadFile(gz)
	if err != nil {
		t.Fatalf("read gz: %v", err)
	}
	if len(gzData) < 2 || gzData[0] != 0x1f || gzData[1] != 0x8b {
		t.Fatalf("gzip magic missing: % x", gzData[:min(2, len(gzData))])
	}
	if len(gzData) >= len(plainData) {
		t.Errorf("gzip %d bytes not smaller than plain %d bytes", len(gzData), len(plainData))
	}
	backPlain, err := Unmarshal(plainData)
	if err != nil {
		t.Fatalf("Unmarshal plain: %v", err)
	}
	backGz, err := Unmarshal(gzData)
	if err != nil {
		t.Fatalf("Unmarshal gz: %v", err)
	}
	if !reflect.DeepEqual(backPlain, m) {
		t.Errorf("plain round trip mismatch")
	}
	if !reflect.DeepEqual(backGz, m) {
		t.Errorf("gzip round trip mismatch")
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read dir: %v", err)
	}
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".tmp") {
			t.Errorf("leftover temp file %s", e.Name())
		}
	}
}

func TestReadMetaFolder(t *testing.T) {
	dir := t.TempDir()
	m := validMeta()
	if err := WriteMeta(filepath.Join(dir, MetaFile), m); err != nil {
		t.Fatalf("WriteMeta: %v", err)
	}
	got, gotDir, err := ReadMeta(dir)
	if err != nil {
		t.Fatalf("ReadMeta folder: %v", err)
	}
	if gotDir != dir {
		t.Errorf("dir = %q, want %q", gotDir, dir)
	}
	if !reflect.DeepEqual(got, m) {
		t.Errorf("meta mismatch")
	}
	empty := t.TempDir()
	if _, _, err := ReadMeta(empty); err == nil {
		t.Errorf("ReadMeta of folder without %s should fail", MetaFile)
	}
}

func TestPNGRoundTrip(t *testing.T) {
	f := Frame{Width: 37, Height: 23, Pix: make([]uint8, 37*23)}
	for y := 0; y < f.Height; y++ {
		for x := 0; x < f.Width; x++ {
			f.Pix[y*f.Width+x] = uint8((x*7 + y*3) % 256)
		}
	}
	path := filepath.Join(t.TempDir(), "frame.png")
	if err := WritePNG(path, f); err != nil {
		t.Fatalf("WritePNG: %v", err)
	}
	back, err := ReadPNG(path)
	if err != nil {
		t.Fatalf("ReadPNG: %v", err)
	}
	if !reflect.DeepEqual(back, f) {
		t.Errorf("frame mismatch")
	}
	h := PixelHash(f)
	if len(h) != 64 {
		t.Errorf("hash length %d, want 64", len(h))
	}
	f.Pix[0] ^= 0xff
	if PixelHash(f) == h {
		t.Errorf("hash did not change after pixel edit")
	}
}

func TestWriteReadImage(t *testing.T) {
	dir := t.TempDir()
	m := validMeta()
	f := Frame{Width: m.Width, Height: m.Height, Pix: make([]uint8, m.Width*m.Height)}
	for i := range f.Pix {
		f.Pix[i] = uint8(i % 251)
	}
	if err := WriteImage(dir, m, ChannelRed, KindTest, f); err != nil {
		t.Fatalf("WriteImage: %v", err)
	}
	entry, ok := m.FindImage(ChannelRed, KindTest)
	if !ok {
		t.Fatalf("entry not recorded")
	}
	if entry.Width != f.Width || entry.Height != f.Height {
		t.Errorf("entry size %dx%d, want %dx%d", entry.Width, entry.Height, f.Width, f.Height)
	}
	if entry.SHA256 != PixelHash(f) {
		t.Errorf("entry hash %s, want %s", entry.SHA256, PixelHash(f))
	}
	back, err := ReadImage(dir, m, ChannelRed, KindTest)
	if err != nil {
		t.Fatalf("ReadImage: %v", err)
	}
	if !reflect.DeepEqual(back, f) {
		t.Errorf("image mismatch")
	}
	if _, err := ReadImage(dir, m, ChannelGreen, KindTest); err == nil {
		t.Errorf("missing entry should fail")
	}
	other := Frame{Width: f.Width, Height: f.Height, Pix: make([]uint8, len(f.Pix))}
	for i := range other.Pix {
		other.Pix[i] = uint8((i + 1) % 251)
	}
	if err := WritePNG(filepath.Join(dir, entry.File), other); err != nil {
		t.Fatalf("overwrite: %v", err)
	}
	if _, err := ReadImage(dir, m, ChannelRed, KindTest); err == nil || !strings.Contains(err.Error(), "sha256") {
		t.Errorf("hash mismatch error = %v, want sha256", err)
	}
	small := Frame{Width: f.Width - 1, Height: f.Height, Pix: make([]uint8, (f.Width-1)*f.Height)}
	if err := WritePNG(filepath.Join(dir, entry.File), small); err != nil {
		t.Fatalf("overwrite small: %v", err)
	}
	if _, err := ReadImage(dir, m, ChannelRed, KindTest); err == nil {
		t.Errorf("size mismatch should fail")
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

var _ = core.ChRed
