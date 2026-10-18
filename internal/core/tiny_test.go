package core

import (
	"math"
	"reflect"
	"testing"
)

func TestInteriorMaterial(t *testing.T) {
	const w, h = 20, 20
	material := make([]uint8, w*h)
	for i := range material {
		material[i] = MatSR
	}
	material[10*w+10] = MatMetal

	if !InteriorMaterial(material, w, h, 5, 5, 2, MatSR) {
		t.Fatalf("(5,5,r=2) should be interior SR")
	}
	if InteriorMaterial(material, w, h, 10, 10, 0, MatSR) {
		t.Fatalf("(10,10,r=0) holds metal, should not be interior SR")
	}
	if !InteriorMaterial(material, w, h, 10, 10, 0, MatMetal) {
		t.Fatalf("(10,10,r=0) should be interior metal")
	}
	if InteriorMaterial(material, w, h, 10, 10, 1, MatSR) {
		t.Fatalf("(10,10,r=1) should not be interior SR")
	}
	if InteriorMaterial(material, w, h, 1, 1, 2, MatSR) {
		t.Fatalf("(1,1,r=2) leaves the image")
	}
	if InteriorMaterial(material, w, h, 5, 5, -1, MatSR) {
		t.Fatalf("r=-1 should be false")
	}
}

func TestDefaultTinyDefects(t *testing.T) {
	const w, h = 240, 200
	material := make([]uint8, w*h)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			if x < 40 {
				material[y*w+x] = MatMetal
			} else {
				material[y*w+x] = MatSR
			}
		}
	}
	strokes := DefaultTinyDefects(material, w, h, 10, 7, DefaultTinyEdgeMargin)
	if len(strokes) != 10 {
		t.Fatalf("got %d strokes, want 10", len(strokes))
	}
	for i, s := range strokes {
		if s.Kind != Bright {
			t.Fatalf("stroke %d kind %v, want Bright", i, s.Kind)
		}
		if s.Shape != ShapeSpot {
			t.Fatalf("stroke %d shape %v, want ShapeSpot", i, s.Shape)
		}
		if s.Radius < 1.5 || s.Radius > 3 {
			t.Fatalf("stroke %d radius %v out of [1.5,3]", i, s.Radius)
		}
		if s.DeltaGv < 45 || s.DeltaGv > 90 {
			t.Fatalf("stroke %d deltaGv %v out of [45,90]", i, s.DeltaGv)
		}
		if s.Amplitude != AmpAbsolute {
			t.Fatalf("stroke %d amplitude %v, want AmpAbsolute", i, s.Amplitude)
		}
		r := DefaultTinyEdgeMargin + int(math.Ceil(s.Radius)) + 3
		if !InteriorMaterial(material, w, h, int(math.Round(s.X)), int(math.Round(s.Y)), r, MatSR) {
			t.Fatalf("stroke %d not interior SR", i)
		}
	}
	again := DefaultTinyDefects(material, w, h, 10, 7, DefaultTinyEdgeMargin)
	if !reflect.DeepEqual(strokes, again) {
		t.Fatalf("same seed should be deterministic")
	}
	other := DefaultTinyDefects(material, w, h, 10, 8, DefaultTinyEdgeMargin)
	if reflect.DeepEqual(strokes, other) {
		t.Fatalf("different seed should differ")
	}
	if got := DefaultTinyDefects(material, w, h, 0, 7, DefaultTinyEdgeMargin); len(got) != 0 {
		t.Fatalf("count 0 gave %d strokes", len(got))
	}
	noSR := make([]uint8, w*h)
	for i := range noSR {
		noSR[i] = MatBase
	}
	if got := DefaultTinyDefects(noSR, w, h, 10, 7, DefaultTinyEdgeMargin); len(got) != 0 {
		t.Fatalf("no SR gave %d strokes", len(got))
	}
}

func TestGoodSampleNoise(t *testing.T) {
	in := SensorNoise{ReadSigma: 1.5, ShotGain: 2.5, Seed: 42}
	out := GoodSampleNoise(in)
	if out.Seed != in.Seed+1000 {
		t.Fatalf("seed %d, want %d", out.Seed, in.Seed+1000)
	}
	if out.ReadSigma != in.ReadSigma || out.ShotGain != in.ShotGain {
		t.Fatalf("other fields changed: %+v", out)
	}
}
