package appicon

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/color"
	"image/png"
	"testing"
)

func TestRenderSizesAndDeterminism(t *testing.T) {
	for _, kind := range []Kind{Teach, Studio} {
		for _, size := range Sizes {
			a := Render(kind, size)
			b := Render(kind, size)
			if got := a.Bounds(); got != image.Rect(0, 0, size, size) {
				t.Fatalf("%s %d: bounds = %v", kind, size, got)
			}
			if !bytes.Equal(a.Pix, b.Pix) {
				t.Fatalf("%s %d: renders differ", kind, size)
			}
		}
	}
}

func TestTileFrame(t *testing.T) {
	if got := Render(Teach, 16).NRGBAAt(0, 0); got != (color.NRGBA{0, 0, 0, 255}) {
		t.Fatalf("teach 16 (0,0) = %v", got)
	}
	if got := Render(Studio, 256).NRGBAAt(0, 0); got != (color.NRGBA{0, 0, 0, 255}) {
		t.Fatalf("studio 256 (0,0) = %v", got)
	}
	if got := Render(Teach, 16).NRGBAAt(1, 1); got != (color.NRGBA{0xFF, 0xFF, 0xFF, 255}) {
		t.Fatalf("teach 16 (1,1) = %v", got)
	}
	if got := Render(Teach, 16).NRGBAAt(14, 14); got != (color.NRGBA{0x80, 0x80, 0x80, 255}) {
		t.Fatalf("teach 16 (14,14) = %v", got)
	}
}

func TestTeachDefectSpeck(t *testing.T) {
	img := Render(Teach, 256)
	x, y := 125, 120 // (0.43+0.06)*256 and (0.42+0.05)*256, inside the speck disc
	if got := img.NRGBAAt(x, y); got != (color.NRGBA{0xE0, 0x20, 0x20, 255}) {
		t.Fatalf("teach 256 (%d,%d) = %v", x, y, got)
	}
}

func TestStudioLensCentreIsWhite(t *testing.T) {
	img := Render(Studio, 256)
	if got := img.NRGBAAt(128, 128); got != (color.NRGBA{255, 255, 255, 255}) {
		t.Fatalf("studio 256 (128,128) = %v", got)
	}
	if got := img.NRGBAAt(20, 20); got != (color.NRGBA{0xC0, 0xC0, 0xC0, 255}) {
		t.Fatalf("studio 256 (20,20) = %v", got)
	}
}

func TestKindsDiffer(t *testing.T) {
	if bytes.Equal(Render(Teach, 64).Pix, Render(Studio, 64).Pix) {
		t.Fatal("teach and studio 64 px renders are identical")
	}
}

func TestICOStructure(t *testing.T) {
	for _, kind := range []Kind{Teach, Studio} {
		data, err := ICO(kind)
		if err != nil {
			t.Fatalf("%s: %v", kind, err)
		}
		if len(data) < 6 {
			t.Fatalf("%s: file too short", kind)
		}
		if got := binary.LittleEndian.Uint16(data[0:]); got != 0 {
			t.Fatalf("%s: reserved = %d", kind, got)
		}
		if got := binary.LittleEndian.Uint16(data[2:]); got != 1 {
			t.Fatalf("%s: type = %d", kind, got)
		}
		count := int(binary.LittleEndian.Uint16(data[4:]))
		if count != len(Sizes) {
			t.Fatalf("%s: count = %d, want %d", kind, count, len(Sizes))
		}
		prev := uint32(0)
		total := uint32(6 + 16*count)
		for i := 0; i < count; i++ {
			entry := data[6+16*i : 6+16*(i+1)]
			size := Sizes[i]
			want := byte(size)
			if size == 256 {
				want = 0
			}
			if entry[0] != want || entry[1] != want {
				t.Fatalf("%s entry %d: size bytes = %d,%d want %d", kind, i, entry[0], entry[1], want)
			}
			if got := binary.LittleEndian.Uint16(entry[4:]); got != 1 {
				t.Fatalf("%s entry %d: planes = %d", kind, i, got)
			}
			if got := binary.LittleEndian.Uint16(entry[6:]); got != 32 {
				t.Fatalf("%s entry %d: bit count = %d", kind, i, got)
			}
			length := binary.LittleEndian.Uint32(entry[8:])
			offset := binary.LittleEndian.Uint32(entry[12:])
			if offset < prev {
				t.Fatalf("%s entry %d: offset %d < %d", kind, i, offset, prev)
			}
			prev = offset
			if int(offset)+int(length) > len(data) {
				t.Fatalf("%s entry %d: blob outside the file", kind, i)
			}
			img, err := png.Decode(bytes.NewReader(data[offset : offset+length]))
			if err != nil {
				t.Fatalf("%s entry %d: %v", kind, i, err)
			}
			if got := img.Bounds(); got != image.Rect(0, 0, size, size) {
				t.Fatalf("%s entry %d: decoded bounds = %v", kind, i, got)
			}
			total += length
		}
		if int(total) != len(data) {
			t.Fatalf("%s: byte lengths add up to %d, file is %d", kind, total, len(data))
		}
	}
}

func TestWriteICORejects(t *testing.T) {
	if err := WriteICO(&bytes.Buffer{}, nil); err == nil {
		t.Fatal("empty list accepted")
	}
	if err := WriteICO(&bytes.Buffer{}, []*image.NRGBA{image.NewNRGBA(image.Rect(0, 0, 16, 8))}); err == nil {
		t.Fatal("non-square image accepted")
	}
	if err := WriteICO(&bytes.Buffer{}, []*image.NRGBA{image.NewNRGBA(image.Rect(0, 0, 300, 300))}); err == nil {
		t.Fatal("300x300 image accepted")
	}
}
