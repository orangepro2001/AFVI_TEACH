package core

import "testing"

// paintedRef builds a 64x48 reference: value 40 everywhere, value 200 in a 20x16 block.
func paintedRef() (*GrayImage, []uint8) {
	ref := NewGray(64, 48, 40)
	classes := make([]uint8, 64*48)
	for y := 16; y < 32; y++ {
		for x := 22; x < 42; x++ {
			ref.Data[y*64+x] = 200
			classes[y*64+x] = 3
		}
	}
	return ref, classes
}

func TestPaintedTemplateScale1(t *testing.T) {
	ref, classes := paintedRef()
	out, err := PaintedTemplate(classes, ref, 1)
	if err != nil {
		t.Fatalf("PaintedTemplate: %v", err)
	}
	if out.Width != ref.Width || out.Height != ref.Height {
		t.Fatalf("size = %dx%d, want %dx%d", out.Width, out.Height, ref.Width, ref.Height)
	}
	centre := out.GetPixel(32, 24)
	if centre <= 120 || centre >= 201 {
		t.Errorf("centre = %d, want > 120 and < 201", centre)
	}
	if v := out.GetPixel(0, 0); v >= 45 {
		t.Errorf("corner = %d, want < 45", v)
	}
}

func TestPaintedTemplateScale2(t *testing.T) {
	ref, classes := paintedRef()
	out, err := PaintedTemplate(classes, ref, 2)
	if err != nil {
		t.Fatalf("PaintedTemplate: %v", err)
	}
	if out.Width != 32 || out.Height != 24 {
		t.Fatalf("size = %dx%d, want 32x24", out.Width, out.Height)
	}
}

func TestPaintedTemplateWrongLength(t *testing.T) {
	ref, _ := paintedRef()
	if _, err := PaintedTemplate(make([]uint8, 10), ref, 1); err == nil {
		t.Fatal("want error for wrong class-map length")
	}
}

func TestPaintedTemplateDeterministic(t *testing.T) {
	ref, classes := paintedRef()
	a, err := PaintedTemplate(classes, ref, 1)
	if err != nil {
		t.Fatalf("PaintedTemplate: %v", err)
	}
	b, err := PaintedTemplate(classes, ref, 1)
	if err != nil {
		t.Fatalf("PaintedTemplate: %v", err)
	}
	if a.Width != b.Width || a.Height != b.Height || len(a.Data) != len(b.Data) {
		t.Fatal("sizes differ")
	}
	for i := range a.Data {
		if a.Data[i] != b.Data[i] {
			t.Fatalf("pixel %d differs: %d vs %d", i, a.Data[i], b.Data[i])
		}
	}
}

func TestPaintedTemplateFlat(t *testing.T) {
	ref, classes := paintedRef()
	flat := make([]uint8, len(classes))
	out, err := PaintedTemplate(flat, ref, 1)
	if err != nil {
		t.Fatalf("PaintedTemplate: %v", err)
	}
	stats, err := MaterialStats(ref, flat)
	if err != nil {
		t.Fatalf("MaterialStats: %v", err)
	}
	mean := ClampByte(stats[0].Mean)
	for i, v := range out.Data {
		diff := int(v) - int(mean)
		if diff < -1 || diff > 1 {
			t.Fatalf("pixel %d = %d, want mean %d within 1", i, v, mean)
		}
	}
}
