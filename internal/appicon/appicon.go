// Package appicon renders the application icons of the two GUI programs
// (teach-imgui.exe and camera-gui.exe) in the Win32 classic look and packs them
// as PNG-in-ICO files. It uses the standard library only, so the picture is
// reproducible code: cmd/mkicon writes the .ico files that the Windows resource
// scripts embed into the executables.
package appicon

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"math"
)

// Kind selects which icon is rendered.
type Kind int

const (
	// Teach is the teaching program: a magnifier over a metal pad with a red
	// defect speck and an R-G-B bar.
	Teach Kind = iota
	// Studio is the camera studio: a camera lens with additive red, green and
	// blue light discs.
	Studio
)

// String returns the file-name stem of the kind ("teach" or "studio").
func (k Kind) String() string {
	switch k {
	case Teach:
		return "teach"
	case Studio:
		return "studio"
	default:
		return fmt.Sprintf("kind(%d)", int(k))
	}
}

// Sizes are the icon sizes rendered by ICO, in pixels.
var Sizes = []int{16, 24, 32, 48, 64, 128, 256}

// layer is one painting step of the painter's algorithm. A layer paints when
// shape (and clip, when set) contains the sample point: a normal layer replaces
// the sample colour, an additive layer adds its colour channel-wise.
type layer struct {
	shape func(x, y float64) bool
	clip  func(x, y float64) bool
	r     float64
	g     float64
	b     float64
	add   bool
}

// rect is the half-open rectangle x0 <= x < x1 && y0 <= y < y1.
func rect(x0, y0, x1, y1 float64) func(x, y float64) bool {
	return func(x, y float64) bool {
		return x >= x0 && x < x1 && y >= y0 && y < y1
	}
}

// disc is the closed disc of radius r around (cx, cy).
func disc(cx, cy, r float64) func(x, y float64) bool {
	return func(x, y float64) bool {
		dx, dy := x-cx, y-cy
		return dx*dx+dy*dy <= r*r
	}
}

// capsule is the set of points within w/2 of the segment (x0,y0)-(x1,y1).
func capsule(x0, y0, x1, y1, w float64) func(x, y float64) bool {
	half := w / 2
	dx, dy := x1-x0, y1-y0
	len2 := dx*dx + dy*dy
	return func(x, y float64) bool {
		t := 0.0
		if len2 > 0 {
			t = ((x-x0)*dx + (y-y0)*dy) / len2
			if t < 0 {
				t = 0
			} else if t > 1 {
				t = 1
			}
		}
		px, py := x0+t*dx, y0+t*dy
		ex, ey := x-px, y-py
		return ex*ex+ey*ey <= half*half
	}
}

// polygon is the even-odd fill of the closed polygon through points.
func polygon(points [][2]float64) func(x, y float64) bool {
	return func(x, y float64) bool {
		inside := false
		n := len(points)
		for i := 0; i < n; i++ {
			xi, yi := points[i][0], points[i][1]
			xj, yj := points[(i+1)%n][0], points[(i+1)%n][1]
			if (yi > y) != (yj > y) && x < (xj-xi)*(y-yi)/(yj-yi)+xi {
				inside = !inside
			}
		}
		return inside
	}
}

// tile returns the common Win32-classic tile: black frame, silver face, white
// highlight and #808080 shadow. t0 and b are the frame and bevel widths in unit
// coordinates.
func tile(t0, b float64) []layer {
	x0, y0 := t0, t0
	x1, y1 := 1-t0, 1-t0
	return []layer{
		{shape: rect(0, 0, 1, 1), r: 0x00, g: 0x00, b: 0x00},
		{shape: rect(t0, t0, 1-t0, 1-t0), r: 0xC0, g: 0xC0, b: 0xC0},
		{shape: polygon([][2]float64{
			{x0, y0}, {x1, y0}, {x1 - b, y0 + b}, {x0 + b, y0 + b},
			{x0 + b, y1 - b}, {x0, y1},
		}), r: 0xFF, g: 0xFF, b: 0xFF},
		{shape: polygon([][2]float64{
			{x1, y0}, {x1, y1}, {x0, y1}, {x0 + b, y1 - b},
			{x1 - b, y1 - b}, {x1 - b, y0 + b},
		}), r: 0x80, g: 0x80, b: 0x80},
	}
}

// teachLayers appends the magnifier, the metal pad band, the red defect speck
// and the R-G-B bar to the tile.
func teachLayers(t0, b float64) []layer {
	ls := tile(t0, b)
	const cx, cy = 0.43, 0.42
	ls = append(ls,
		layer{shape: rect(0.15, 0.79, 0.31, 0.87), r: 0xE0, g: 0x20, b: 0x20},
		layer{shape: rect(0.31, 0.79, 0.47, 0.87), r: 0x20, g: 0xA0, b: 0x20},
		layer{shape: rect(0.47, 0.79, 0.63, 0.87), r: 0x20, g: 0x40, b: 0xE0},
		layer{
			shape: capsule(cx+0.27*0.7071, cy+0.27*0.7071, 0.80, 0.79, 0.11),
			r:     0x00, g: 0x00, b: 0x80,
		},
		layer{shape: disc(cx, cy, 0.285), r: 0x00, g: 0x00, b: 0x80},
		layer{shape: disc(cx, cy, 0.215), r: 0xF4, g: 0xF8, b: 0xFF},
		layer{
			shape: rect(cx-0.215, cy+0.02, cx+0.215, cy+0.12),
			clip:  disc(cx, cy, 0.215),
			r:     0xA0, g: 0xA0, b: 0xA0,
		},
		layer{
			shape: disc(cx+0.06, cy+0.05, 0.05),
			clip:  disc(cx, cy, 0.215),
			r:     0xE0, g: 0x20, b: 0x20,
		},
	)
	return ls
}

// studioLayers appends the camera lens and the additive red, green and blue
// light discs to the tile.
func studioLayers(t0, b float64) []layer {
	ls := tile(t0, b)
	ls = append(ls,
		layer{shape: disc(0.5, 0.5, 0.385), r: 0x40, g: 0x40, b: 0x40},
		layer{shape: disc(0.5, 0.5, 0.345), r: 0x00, g: 0x00, b: 0x00},
		layer{shape: disc(0.405, 0.435, 0.205), r: 0xFF, g: 0x00, b: 0x00, add: true},
		layer{shape: disc(0.595, 0.435, 0.205), r: 0x00, g: 0xFF, b: 0x00, add: true},
		layer{shape: disc(0.5, 0.6, 0.205), r: 0x00, g: 0x00, b: 0xFF, add: true},
	)
	return ls
}

// layersFor returns the layer stack of kind for a size in pixels.
func layersFor(kind Kind, size int) []layer {
	t0 := math.Max(1, math.Round(0.02*float64(size))) / float64(size)
	b := math.Max(1, math.Round(0.045*float64(size))) / float64(size)
	if kind == Studio {
		return studioLayers(t0, b)
	}
	return teachLayers(t0, b)
}

// Render paints the icon of kind at size x size pixels with 4x4 supersampling.
// A size below 1 yields a 1x1 transparent image.
func Render(kind Kind, size int) *image.NRGBA {
	if size < 1 {
		size = 1
	}
	img := image.NewNRGBA(image.Rect(0, 0, size, size))
	ls := layersFor(kind, size)
	s := float64(size)
	for py := 0; py < size; py++ {
		for px := 0; px < size; px++ {
			var sr, sg, sb float64
			count := 0
			for sy := 0; sy < 4; sy++ {
				for sx := 0; sx < 4; sx++ {
					x := (float64(px) + (float64(sx)+0.5)/4) / s
					y := (float64(py) + (float64(sy)+0.5)/4) / s
					r, g, b, painted := sample(ls, x, y)
					if !painted {
						continue
					}
					sr += r
					sg += g
					sb += b
					count++
				}
			}
			if count == 0 {
				img.SetNRGBA(px, py, color.NRGBA{})
				continue
			}
			img.SetNRGBA(px, py, color.NRGBA{
				R: uint8(sr/float64(count) + 0.5),
				G: uint8(sg/float64(count) + 0.5),
				B: uint8(sb/float64(count) + 0.5),
				A: uint8(255*float64(count)/16 + 0.5),
			})
		}
	}
	return img
}

// sample walks the layers in order and returns the colour of one sub-sample.
func sample(ls []layer, x, y float64) (r, g, b float64, painted bool) {
	for _, l := range ls {
		if l.shape == nil || !l.shape(x, y) {
			continue
		}
		if l.clip != nil && !l.clip(x, y) {
			continue
		}
		if l.add && painted {
			r = clamp255(r + l.r)
			g = clamp255(g + l.g)
			b = clamp255(b + l.b)
			continue
		}
		r, g, b = l.r, l.g, l.b
		painted = true
	}
	return r, g, b, painted
}

// clamp255 clamps v to 0..255.
func clamp255(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 255 {
		return 255
	}
	return v
}

// WriteICO writes images as a PNG-in-ICO file (little endian). Every image must
// be square and at most 256 pixels wide.
func WriteICO(w io.Writer, images []*image.NRGBA) error {
	if len(images) == 0 {
		return errors.New("appicon: no images")
	}
	blobs := make([][]byte, len(images))
	for i, img := range images {
		if img == nil {
			return fmt.Errorf("appicon: image %d is nil", i)
		}
		b := img.Bounds()
		if b.Dx() != b.Dy() {
			return fmt.Errorf("appicon: image %d is not square (%dx%d)", i, b.Dx(), b.Dy())
		}
		if b.Dx() > 256 {
			return fmt.Errorf("appicon: image %d is larger than 256 (%d)", i, b.Dx())
		}
		var buf bytes.Buffer
		enc := png.Encoder{CompressionLevel: png.BestCompression}
		if err := enc.Encode(&buf, img); err != nil {
			return fmt.Errorf("appicon: encoding image %d: %w", i, err)
		}
		blobs[i] = buf.Bytes()
	}
	offset := uint32(6 + 16*len(images))
	head := make([]byte, 6)
	binary.LittleEndian.PutUint16(head[0:], 0)
	binary.LittleEndian.PutUint16(head[2:], 1)
	binary.LittleEndian.PutUint16(head[4:], uint16(len(images)))
	if _, err := w.Write(head); err != nil {
		return err
	}
	for i, img := range images {
		size := img.Bounds().Dx()
		entry := make([]byte, 16)
		if size < 256 {
			entry[0] = byte(size)
			entry[1] = byte(size)
		}
		entry[2] = 0
		entry[3] = 0
		binary.LittleEndian.PutUint16(entry[4:], 1)
		binary.LittleEndian.PutUint16(entry[6:], 32)
		binary.LittleEndian.PutUint32(entry[8:], uint32(len(blobs[i])))
		binary.LittleEndian.PutUint32(entry[12:], offset)
		if _, err := w.Write(entry); err != nil {
			return err
		}
		offset += uint32(len(blobs[i]))
	}
	for _, blob := range blobs {
		if _, err := w.Write(blob); err != nil {
			return err
		}
	}
	return nil
}

// ICO renders every size in Sizes and returns the ICO file of kind.
func ICO(kind Kind) ([]byte, error) {
	images := make([]*image.NRGBA, 0, len(Sizes))
	for _, size := range Sizes {
		images = append(images, Render(kind, size))
	}
	var buf bytes.Buffer
	if err := WriteICO(&buf, images); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
