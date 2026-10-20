// Package golden: persistence of a golden board (mask PNG + JSON sidecar).
// Standard library only.
package golden

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Format identifies the JSON sidecar of a golden board.
const Format = "afvi-golden"

// Version is the current sidecar format version.
const Version = 1

// FileJSON is the JSON sidecar file name inside a board directory.
const FileJSON = "golden.json"

// FileMask is the Metal mask PNG file name inside a board directory.
const FileMask = "golden-metal.png"

// Settings are the numeric board settings.
type Settings struct {
	Channel  int
	Low      int
	High     int
	OutRange bool
	Padding  int
	SRInset  int
}

// Board is a persisted golden board.
type Board struct {
	Model    string
	Settings Settings
	Metal    *Mask
}

// goldenFile is the JSON document of FileJSON. The field order defines the
// key order of the written JSON.
type goldenFile struct {
	Format   string `json:"format"`
	Version  int    `json:"version"`
	Model    string `json:"model"`
	Width    int    `json:"width"`
	Height   int    `json:"height"`
	Channel  int    `json:"channel"`
	Low      int    `json:"low"`
	High     int    `json:"high"`
	OutRange bool   `json:"outRange"`
	Padding  int    `json:"padding"`
	SRInset  int    `json:"srInset"`
	Metal    string `json:"metal"`
	SHA256   string `json:"sha256"`
	Created  string `json:"created"`
}

// Save writes the Metal mask PNG and the JSON sidecar of b into dir, creating
// it when needed. Both files are written atomically (temp file + rename).
func Save(dir string, b *Board) error {
	if b == nil {
		return errors.New("golden: nil board")
	}
	if !b.Metal.Valid() {
		return errors.New("golden: invalid metal mask")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("golden: %w", err)
	}
	w, h := b.Metal.W, b.Metal.H

	// 8-bit gray PNG: 255 inside, 0 outside.
	img := image.NewGray(image.Rect(0, 0, w, h))
	for i, v := range b.Metal.Pix {
		if v != 0 {
			img.Pix[i] = 255
		}
	}
	if err := writeAtomic(filepath.Join(dir, FileMask), func(f *os.File) error {
		return png.Encode(f, img)
	}); err != nil {
		return err
	}

	doc := goldenFile{
		Format:   Format,
		Version:  Version,
		Model:    b.Model,
		Width:    w,
		Height:   h,
		Channel:  b.Settings.Channel,
		Low:      b.Settings.Low,
		High:     b.Settings.High,
		OutRange: b.Settings.OutRange,
		Padding:  b.Settings.Padding,
		SRInset:  b.Settings.SRInset,
		Metal:    FileMask,
		SHA256:   maskDigest(b.Metal),
		Created:  time.Now().UTC().Format(time.RFC3339),
	}
	data, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return fmt.Errorf("golden: %w", err)
	}
	data = append(data, '\n')
	return writeAtomic(filepath.Join(dir, FileJSON), func(f *os.File) error {
		_, err := f.Write(data)
		return err
	})
}

// Load reads and validates the golden board stored in dir.
func Load(dir string) (*Board, error) {
	data, err := os.ReadFile(filepath.Join(dir, FileJSON))
	if err != nil {
		return nil, fmt.Errorf("golden: %w", err)
	}
	var doc goldenFile
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("golden: %w", err)
	}
	if doc.Format != Format {
		return nil, fmt.Errorf("golden: unknown format %q", doc.Format)
	}
	if doc.Version < 1 || doc.Version > Version {
		return nil, fmt.Errorf("golden: unsupported version %d", doc.Version)
	}
	if doc.Width <= 0 || doc.Height <= 0 || doc.Width > 65535 || doc.Height > 65535 {
		return nil, fmt.Errorf("golden: bad size %dx%d", doc.Width, doc.Height)
	}
	if err := checkSettings(doc); err != nil {
		return nil, err
	}
	name := doc.Metal
	if name == "" || strings.Contains(name, "..") ||
		strings.ContainsRune(name, '/') || strings.ContainsRune(name, '\\') ||
		filepath.Base(name) != name {
		return nil, fmt.Errorf("golden: bad metal file name %q", name)
	}
	f, err := os.Open(filepath.Join(dir, name))
	if err != nil {
		return nil, fmt.Errorf("golden: %w", err)
	}
	defer f.Close()
	img, err := png.Decode(f)
	if err != nil {
		return nil, fmt.Errorf("golden: %w", err)
	}
	bounds := img.Bounds()
	if bounds.Dx() != doc.Width || bounds.Dy() != doc.Height {
		return nil, fmt.Errorf("golden: png size %dx%d, want %dx%d",
			bounds.Dx(), bounds.Dy(), doc.Width, doc.Height)
	}
	m := NewMask(doc.Width, doc.Height)
	if g, ok := img.(*image.Gray); ok {
		// Fast path for the 8-bit gray PNG Save writes.
		for y := 0; y < doc.Height; y++ {
			row := g.Pix[y*g.Stride : y*g.Stride+doc.Width]
			dst := m.Pix[y*m.W : (y+1)*m.W]
			for x, v := range row {
				if v >= 128 {
					dst[x] = 1
				}
			}
		}
	} else {
		for y := 0; y < doc.Height; y++ {
			for x := 0; x < doc.Width; x++ {
				r, _, _, _ := img.At(bounds.Min.X+x, bounds.Min.Y+y).RGBA()
				if r>>8 >= 128 {
					m.Pix[y*m.W+x] = 1
				}
			}
		}
	}
	if maskDigest(m) != strings.ToLower(doc.SHA256) {
		return nil, errors.New("golden: mask sha256 mismatch")
	}
	return &Board{
		Model: doc.Model,
		Settings: Settings{
			Channel:  doc.Channel,
			Low:      doc.Low,
			High:     doc.High,
			OutRange: doc.OutRange,
			Padding:  doc.Padding,
			SRInset:  doc.SRInset,
		},
		Metal: m,
	}, nil
}

// checkSettings rejects silently invalid numeric settings.
func checkSettings(doc goldenFile) error {
	if doc.Low < 0 || doc.Low > 255 || doc.High < 0 || doc.High > 255 {
		return fmt.Errorf("golden: range %d..%d out of 0..255", doc.Low, doc.High)
	}
	if doc.Low > doc.High {
		return fmt.Errorf("golden: low %d > high %d", doc.Low, doc.High)
	}
	if doc.Padding < 0 || doc.Padding > MaxPad || doc.SRInset < 0 || doc.SRInset > MaxPad {
		return fmt.Errorf("golden: padding %d / inset %d out of 0..%d",
			doc.Padding, doc.SRInset, MaxPad)
	}
	return nil
}

// writeAtomic writes via a temp file in the same directory and renames it over
// the final name, removing the temp file on error.
func writeAtomic(path string, fill func(*os.File) error) error {
	dir := filepath.Dir(path)
	f, err := os.CreateTemp(dir, ".golden-*.tmp")
	if err != nil {
		return fmt.Errorf("golden: %w", err)
	}
	tmp := f.Name()
	if err := fill(f); err != nil {
		f.Close()
		os.Remove(tmp)
		return fmt.Errorf("golden: %w", err)
	}
	if err := f.Close(); err != nil {
		os.Remove(tmp)
		return fmt.Errorf("golden: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		os.Remove(tmp)
		return fmt.Errorf("golden: %w", err)
	}
	return nil
}

// maskDigest returns the hex sha256 of the 0/1 pixel bytes.
func maskDigest(m *Mask) string {
	buf := make([]byte, len(m.Pix))
	for i, v := range m.Pix {
		if v != 0 {
			buf[i] = 1
		}
	}
	sum := sha256.Sum256(buf)
	return hex.EncodeToString(sum[:])
}
