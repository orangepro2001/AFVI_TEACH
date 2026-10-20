package golden

import (
	"encoding/json"
	"errors"
	"image"
	"image/color"
	"image/png"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// sampleBoard returns a small board for the store tests.
func sampleBoard() *Board {
	m := NewMask(8, 6)
	fillRect(m, 2, 1, 6, 5)
	return &Board{
		Model: "model-1",
		Settings: Settings{
			Channel:  2,
			Low:      40,
			High:     180,
			OutRange: true,
			Padding:  2,
			SRInset:  4,
		},
		Metal: m,
	}
}

func TestSaveLoadRoundTrip(t *testing.T) {
	dir := t.TempDir()
	b := sampleBoard()
	if err := Save(dir, b); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got, err := Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got.Model != b.Model || got.Settings != b.Settings {
		t.Fatalf("round trip mismatch: %+v vs %+v", got.Settings, b.Settings)
	}
	if !got.Metal.Equal(b.Metal) {
		t.Fatal("metal mask mismatch after round trip")
	}

	// A second Save overwrites the first.
	b.Settings.Padding = 5
	if err := Save(dir, b); err != nil {
		t.Fatalf("second Save: %v", err)
	}
	got, err = Load(dir)
	if err != nil {
		t.Fatalf("Load after overwrite: %v", err)
	}
	if got.Settings.Padding != 5 {
		t.Fatalf("padding = %d, want 5", got.Settings.Padding)
	}

	// No temp files left behind.
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".tmp") {
			t.Fatalf("leftover temp file %q", e.Name())
		}
	}
}

func TestLoadMissing(t *testing.T) {
	_, err := Load(t.TempDir())
	if !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("Load of an empty dir error = %v, want fs.ErrNotExist", err)
	}
}

func TestLoadHashMismatch(t *testing.T) {
	dir := t.TempDir()
	if err := Save(dir, sampleBoard()); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, FileMask)
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	img, err := png.Decode(f)
	f.Close()
	if err != nil {
		t.Fatal(err)
	}
	bounds := img.Bounds()
	g, ok := img.(*image.Gray)
	if !ok {
		// Re-encode through a gray picture to flip a pixel.
		g = image.NewGray(bounds)
		for y := 0; y < bounds.Dy(); y++ {
			for x := 0; x < bounds.Dx(); x++ {
				r, _, _, _ := img.At(bounds.Min.X+x, bounds.Min.Y+y).RGBA()
				g.SetGray(x, y, color.Gray{Y: uint8(r >> 8)})
			}
		}
	}
	flipped := image.NewGray(bounds)
	copy(flipped.Pix, g.Pix)
	flipped.Pix[0] ^= 255
	out, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(out, flipped); err != nil {
		out.Close()
		t.Fatal(err)
	}
	out.Close()

	if _, err := Load(dir); err == nil {
		t.Fatal("Load must fail on a hash mismatch")
	}
}

// editDoc rewrites the JSON sidecar with the given mutation.
func editDoc(t *testing.T, dir string, mutate func(*map[string]any)) {
	t.Helper()
	path := filepath.Join(dir, FileJSON)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatal(err)
	}
	mutate(&doc)
	data, err = json.MarshalIndent(doc, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestLoadRejectsBadJSON(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(*map[string]any)
	}{
		{"format", func(d *map[string]any) { (*d)["format"] = "other" }},
		{"version", func(d *map[string]any) { (*d)["version"] = float64(99) }},
		{"width", func(d *map[string]any) { (*d)["width"] = float64(0) }},
		{"height", func(d *map[string]any) { (*d)["height"] = float64(70000) }},
		{"low", func(d *map[string]any) { (*d)["low"] = float64(300) }},
		{"high", func(d *map[string]any) { (*d)["high"] = float64(-1) }},
		{"padding", func(d *map[string]any) { (*d)["padding"] = float64(MaxPad + 1) }},
		{"srInset", func(d *map[string]any) { (*d)["srInset"] = float64(-1) }},
		{"metalPath", func(d *map[string]any) { (*d)["metal"] = "sub/golden-metal.png" }},
		{"metalDotDot", func(d *map[string]any) { (*d)["metal"] = "../golden-metal.png" }},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			dir := t.TempDir()
			if err := Save(dir, sampleBoard()); err != nil {
				t.Fatal(err)
			}
			editDoc(t, dir, c.mutate)
			if _, err := Load(dir); err == nil {
				t.Fatalf("Load accepted a bad %s", c.name)
			}
		})
	}
}

func TestLoadLowAboveHigh(t *testing.T) {
	dir := t.TempDir()
	if err := Save(dir, sampleBoard()); err != nil {
		t.Fatal(err)
	}
	editDoc(t, dir, func(d *map[string]any) {
		(*d)["low"] = float64(200)
		(*d)["high"] = float64(10)
	})
	if _, err := Load(dir); err == nil {
		t.Fatal("Load accepted low > high")
	}
}

func TestSaveLoadUserMask(t *testing.T) {
	dir := t.TempDir()
	b := sampleBoard()
	b.User = sampleUser()
	if err := Save(dir, b); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got, err := Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !reflect.DeepEqual(got.User, b.User) {
		t.Fatalf("user mask round trip mismatch:\n got %+v\nwant %+v", got.User, b.User)
	}
}

func TestSaveOmitsEmptyUserMask(t *testing.T) {
	dir := t.TempDir()
	if err := Save(dir, sampleBoard()); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(dir, FileJSON))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "userMask") {
		t.Fatal("golden.json of a board without regions contains userMask")
	}
}

func TestLoadMissingUserMask(t *testing.T) {
	dir := t.TempDir()
	if err := Save(dir, sampleBoard()); err != nil {
		t.Fatal(err)
	}
	got, err := Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !got.User.Empty() {
		t.Fatal("missing userMask must load as an empty UserMask")
	}
}

func TestLoadRejectsBadUserMask(t *testing.T) {
	cases := []struct {
		name string
		doc  string
	}{
		{"unknown-target", `[{"target":"other","kind":"rect","p0":[1,1],"p1":[2,2]}]`},
		{"unknown-kind", `[{"target":"metal","kind":"circle","p0":[1,1],"p1":[2,2]}]`},
		{"zero-area", `[{"target":"metal","kind":"rect","p0":[1,1],"p1":[1,2]}]`},
		{"short-polygon", `[{"target":"sr","kind":"polygon","points":[[0,0],[1,1]]}]`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			dir := t.TempDir()
			if err := Save(dir, sampleBoard()); err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(dir, FileJSON)
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			var doc map[string]any
			if err := json.Unmarshal(data, &doc); err != nil {
				t.Fatal(err)
			}
			var regions any
			if err := json.Unmarshal([]byte(c.doc), &regions); err != nil {
				t.Fatal(err)
			}
			doc["userMask"] = regions
			data, err = json.Marshal(doc)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, data, 0o644); err != nil {
				t.Fatal(err)
			}
			_, err = Load(dir)
			if err == nil || !strings.HasPrefix(err.Error(), "golden: ") {
				t.Fatalf("Load error = %v, want a golden: error", err)
			}
		})
	}
}

func TestLoadRejectsTooManyUserMask(t *testing.T) {
	dir := t.TempDir()
	if err := Save(dir, sampleBoard()); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, FileJSON)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatal(err)
	}
	regions := make([]any, MaxRegions+1)
	for i := range regions {
		regions[i] = map[string]any{"target": "metal", "kind": "rect", "p0": []float64{1, 1}, "p1": []float64{2, 2}}
	}
	doc["userMask"] = regions
	data, err = json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
	_, err = Load(dir)
	if err == nil || !strings.HasPrefix(err.Error(), "golden: ") {
		t.Fatalf("Load error = %v, want a golden: error", err)
	}
}

func TestSaveRejectsBadUserMask(t *testing.T) {
	dir := t.TempDir()
	bad := sampleBoard()
	bad.User = UserMask{Regions: []Region{rectRegion(TargetMetal, 1, 1, 1, 1)}}
	if err := Save(dir, bad); err == nil {
		t.Fatal("Save with an invalid user region must fail")
	}
	entries, err := os.ReadDir(dir)
	if err == nil && len(entries) != 0 {
		t.Fatal("Save wrote files before validating the user mask")
	}

	full := sampleBoard()
	full.User = UserMask{Regions: make([]Region, MaxRegions+1)}
	for i := range full.User.Regions {
		full.User.Regions[i] = rectRegion(TargetSR, 0, 0, 2, 2)
	}
	if err := Save(dir, full); err == nil {
		t.Fatal("Save with more than MaxRegions must fail")
	}
}

func TestSaveRejectsBadBoard(t *testing.T) {
	dir := t.TempDir()
	if err := Save(dir, nil); err == nil {
		t.Fatal("Save(nil) must fail")
	}
	bad := sampleBoard()
	bad.Metal = NewMask(0, 0)
	if err := Save(dir, bad); err == nil {
		t.Fatal("Save with an invalid mask must fail")
	}
}
