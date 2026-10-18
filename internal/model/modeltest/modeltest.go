// Package modeltest writes a tiny synthetic reference tree that the model scanner
// accepts, so tests in several packages can load a real Scene without the git-ignored
// reference/ directory.
package modeltest

import (
	"image"
	"image/color"
	"os"
	"path/filepath"

	"golang.org/x/image/tiff"
)

// Reference describes a synthetic reference tree written by WriteReference.
type Reference struct {
	Root   string // reference root; pass it as the -ref directory (contains FM1)
	Host   string // "FM1"
	Model  string // "synthetic-1"
	ID     string // "FM1/synthetic-1"
	Width  int    // 96
	Height int    // 64
}

// WriteReference creates <root>/FM1/PxRepository/synthetic-1/{Metal,SR,Open}.tif and
// <root>/FM1/PxRepository/synthetic-1/2.0/Line/Top/Layer/MEDIAN/Median_0_1_0.tif (8-bit gray TIFF,
// golang.org/x/image/tiff Encode) and returns the description. Layers (96x64, threshold 128 means "present"):
//
//	Metal = 255 where x < 16, else 0
//	Open  = 255 where 16 <= x < 32, else 0
//	SR    = 255 where 36 <= x < 92 and 8 <= y < 56, else 0
//
// (the material map priority is metal > open > sr > base, so the rest is base)
// MEDIAN = 40 for base, 90 for SR, 150 for open, 220 for metal, plus (x*3+y*5)%7 so it is not flat.
func WriteReference(root string) (Reference, error) {
	const (
		width  = 96
		height = 64
	)
	ref := Reference{
		Root:   root,
		Host:   "FM1",
		Model:  "synthetic-1",
		ID:     "FM1/synthetic-1",
		Width:  width,
		Height: height,
	}
	modelDir := filepath.Join(root, "FM1", "PxRepository", ref.Model)
	if err := os.MkdirAll(modelDir, 0o755); err != nil {
		return ref, err
	}
	medianDir := filepath.Join(modelDir, "2.0", "Line", "Top", "Layer", "MEDIAN")
	if err := os.MkdirAll(medianDir, 0o755); err != nil {
		return ref, err
	}

	metal := image.NewGray(image.Rect(0, 0, width, height))
	sr := image.NewGray(image.Rect(0, 0, width, height))
	open := image.NewGray(image.Rect(0, 0, width, height))
	median := image.NewGray(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			var metalV, srV, openV uint8
			if x < 16 {
				metalV = 255
			}
			if x >= 16 && x < 32 {
				openV = 255
			}
			if x >= 36 && x < 92 && y >= 8 && y < 56 {
				srV = 255
			}
			metal.SetGray(x, y, color.Gray{Y: metalV})
			open.SetGray(x, y, color.Gray{Y: openV})
			sr.SetGray(x, y, color.Gray{Y: srV})

			var base uint8
			switch {
			case metalV >= 128:
				base = 220
			case openV >= 128:
				base = 150
			case srV >= 128:
				base = 90
			default:
				base = 40
			}
			base += uint8((x*3 + y*5) % 7)
			median.SetGray(x, y, color.Gray{Y: base})
		}
	}

	files := []struct {
		path string
		img  *image.Gray
	}{
		{filepath.Join(modelDir, "Metal.tif"), metal},
		{filepath.Join(modelDir, "SR.tif"), sr},
		{filepath.Join(modelDir, "Open.tif"), open},
		{filepath.Join(medianDir, "Median_0_1_0.tif"), median},
	}
	for _, f := range files {
		if err := writeGray(f.path, f.img); err != nil {
			return ref, err
		}
	}
	return ref, nil
}

func writeGray(path string, img *image.Gray) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	if err := tiff.Encode(f, img, nil); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}
