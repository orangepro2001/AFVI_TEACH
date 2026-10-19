package snap

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// Marshal validates m and returns indented JSON (2 spaces, trailing newline) for hand editing.
func Marshal(m *Meta) ([]byte, error) {
	if err := m.Validate(); err != nil {
		return nil, err
	}
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(data, '\n'), nil
}

// Unmarshal parses a description; data that starts with the gzip magic 0x1f 0x8b is decompressed first. Unknown JSON fields are ignored; the result is validated.
func Unmarshal(data []byte) (*Meta, error) {
	if len(data) >= 2 && data[0] == 0x1f && data[1] == 0x8b {
		zr, err := gzip.NewReader(bytes.NewReader(data))
		if err != nil {
			return nil, fmt.Errorf("snap: gzip: %w", err)
		}
		defer zr.Close()
		plain, err := io.ReadAll(zr)
		if err != nil {
			return nil, fmt.Errorf("snap: gzip: %w", err)
		}
		data = plain
	}
	if LooksLikeXML(data) {
		s, err := UnmarshalScenario(data)
		if err != nil {
			return nil, err
		}
		m := s.Meta()
		return &m, nil
	}
	var m Meta
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("snap: json: %w", err)
	}
	if err := m.Validate(); err != nil {
		return nil, err
	}
	return &m, nil
}

// WriteMeta writes the description atomically (temp file in the same directory + rename). A path ending in ".gz" gets compact JSON gzip-compressed (the "dense" form); any other path gets Marshal output. Parent directories are created.
func WriteMeta(path string, m *Meta) error {
	if err := m.Validate(); err != nil {
		return err
	}
	lower := strings.ToLower(path)
	if strings.HasSuffix(lower, ".xml") || strings.HasSuffix(lower, ".xml.gz") {
		return WriteScenario(path, ScenarioFromMeta(*m))
	}
	var data []byte
	if strings.HasSuffix(path, ".gz") {
		compact, err := json.Marshal(m)
		if err != nil {
			return err
		}
		var buf bytes.Buffer
		zw, err := gzip.NewWriterLevel(&buf, gzip.BestCompression)
		if err != nil {
			return err
		}
		if _, err := zw.Write(compact); err != nil {
			zw.Close()
			return err
		}
		if err := zw.Close(); err != nil {
			return err
		}
		data = buf.Bytes()
	} else {
		indented, err := Marshal(m)
		if err != nil {
			return err
		}
		data = indented
	}
	return writeFileAtomic(path, data)
}

// ReadMeta reads a description. path may be a capture folder (uses MetaFile) or a file. It returns the meta and the directory that contains the description file (where the images live).
func ReadMeta(path string) (*Meta, string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, "", err
	}
	file := path
	dir := filepath.Dir(path)
	if info.IsDir() {
		file = filepath.Join(path, MetaFile)
		dir = path
		// 没有 snap.json 但有场景文件的文件夹：读场景文件（只有配方，没有图像）。
		if _, err := os.Stat(file); err != nil {
			if _, serr := os.Stat(filepath.Join(path, ScenarioFile)); serr == nil {
				file = filepath.Join(path, ScenarioFile)
			}
		}
	}
	data, err := os.ReadFile(file)
	if err != nil {
		return nil, "", err
	}
	m, err := Unmarshal(data)
	if err != nil {
		return nil, "", err
	}
	return m, dir, nil
}

// Frame is an 8-bit gray image.
type Frame struct {
	Width, Height int
	Pix           []uint8
}

// PixelHash returns the hex sha256 of f.Pix.
func PixelHash(f Frame) string {
	sum := sha256.Sum256(f.Pix)
	return hex.EncodeToString(sum[:])
}

// WritePNG writes an 8-bit gray PNG, png.Encoder with BestSpeed, created via temp file + rename, parent dirs created; error if len(Pix) != Width*Height.
func WritePNG(path string, f Frame) error {
	if f.Width <= 0 || f.Height <= 0 {
		return fmt.Errorf("snap: bad frame size %dx%d", f.Width, f.Height)
	}
	if len(f.Pix) != f.Width*f.Height {
		return fmt.Errorf("snap: frame has %d pixels, want %d", len(f.Pix), f.Width*f.Height)
	}
	img := &image.Gray{Pix: f.Pix, Stride: f.Width, Rect: image.Rect(0, 0, f.Width, f.Height)}
	var buf bytes.Buffer
	enc := png.Encoder{CompressionLevel: png.BestSpeed}
	if err := enc.Encode(&buf, img); err != nil {
		return err
	}
	return writeFileAtomic(path, buf.Bytes())
}

// ReadPNG reads any PNG; converted to 8-bit gray (color.GrayModel) when it is not already *image.Gray; honours the stride.
func ReadPNG(path string) (Frame, error) {
	f, err := os.Open(path)
	if err != nil {
		return Frame{}, err
	}
	defer f.Close()
	img, err := png.Decode(f)
	if err != nil {
		return Frame{}, err
	}
	b := img.Bounds()
	w := b.Dx()
	h := b.Dy()
	if w <= 0 || h <= 0 {
		return Frame{}, fmt.Errorf("snap: empty image %q", path)
	}
	out := make([]uint8, w*h)
	if g, ok := img.(*image.Gray); ok {
		for y := 0; y < h; y++ {
			src := g.PixOffset(b.Min.X, b.Min.Y+y)
			copy(out[y*w:(y+1)*w], g.Pix[src:src+w])
		}
		return Frame{Width: w, Height: h, Pix: out}, nil
	}
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			c := color.GrayModel.Convert(img.At(b.Min.X+x, b.Min.Y+y)).(color.Gray)
			out[y*w+x] = c.Y
		}
	}
	return Frame{Width: w, Height: h, Pix: out}, nil
}

// WriteImage writes dir/ImageFile(channel, kind) and records the entry (size and PixelHash) in m with SetImage. It does not write the description file.
func WriteImage(dir string, m *Meta, channel, kind string, f Frame) error {
	if _, err := ParseChannel(channel); err != nil {
		return err
	}
	if kind != KindTest && kind != KindGood {
		return fmt.Errorf("snap: unknown kind %q", kind)
	}
	name := ImageFile(channel, kind)
	if err := WritePNG(filepath.Join(dir, name), f); err != nil {
		return err
	}
	m.SetImage(Image{
		Channel: channel,
		Kind:    kind,
		File:    name,
		Width:   f.Width,
		Height:  f.Height,
		SHA256:  PixelHash(f),
	})
	return nil
}

// ReadImage loads the image recorded in m for channel+kind from dir and verifies size and sha256; a missing entry, a missing file, a size mismatch or a hash mismatch are distinct, descriptive errors (the hash error contains "sha256").
func ReadImage(dir string, m *Meta, channel, kind string) (Frame, error) {
	entry, ok := m.FindImage(channel, kind)
	if !ok {
		return Frame{}, fmt.Errorf("snap: no image entry for channel %q kind %q", channel, kind)
	}
	f, err := ReadPNG(filepath.Join(dir, entry.File))
	if err != nil {
		return Frame{}, err
	}
	if f.Width != entry.Width || f.Height != entry.Height {
		return Frame{}, fmt.Errorf("snap: image %q is %dx%d, want %dx%d", entry.File, f.Width, f.Height, entry.Width, entry.Height)
	}
	if got := PixelHash(f); got != entry.SHA256 {
		return Frame{}, fmt.Errorf("snap: image %q sha256 %s, want %s", entry.File, got, entry.SHA256)
	}
	return f, nil
}

func writeFileAtomic(path string, data []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, filepath.Base(path)+".*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return err
	}
	if err := os.Rename(tmpName, path); err != nil {
		os.Remove(tmpName)
		return err
	}
	return nil
}
