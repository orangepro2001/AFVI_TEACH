package app

import (
	"math"

	"afvi/teach-imgui/internal/core"
	"afvi/teach-imgui/internal/golden"
)

// goldenRefineSR snaps the registered SR layer sr to the golden sample
// picture src and returns a NEW mask (never sr itself).
func goldenRefineSR(src *core.GrayImage, sr *golden.Mask) *golden.Mask {
	if src == nil || !sr.Valid() || sr.W != src.Width || sr.H != src.Height {
		if sr == nil {
			return nil
		}
		return sr.Clone()
	}
	w, h := sr.W, sr.H
	interior := golden.Erode(sr, 8)
	if interior.Count() < 100 {
		return sr.Clone()
	}
	mu, sigma := refineStats(src.Data, w, h, interior)
	half := 4 * sigma
	if half < 6 {
		half = 6
	}
	lo := refineClamp(refineRound(mu - half))
	hi := refineClamp(refineRound(mu + half))
	if lo > hi {
		lo, hi = hi, lo
	}
	cand := golden.And(
		golden.Threshold(src.Data, w, h, uint8(lo), uint8(hi), false),
		golden.Dilate(sr, 10),
	)
	cand = golden.Open(cand, 1)
	cand = golden.AreaFilter(cand, 64, true)
	base := sr.Count()
	n := cand.Count()
	if base <= 0 || n*10 < base*7 || n*10 > base*13 {
		return sr.Clone()
	}
	return cand
}

// refineStats returns the mean and the population standard deviation of pix
// over the set pixels of m.
func refineStats(pix []uint8, w, h int, m *golden.Mask) (float64, float64) {
	sum, sq := 0.0, 0.0
	n := 0
	for i := 0; i < w*h; i++ {
		if m.Pix[i] == 0 {
			continue
		}
		v := float64(pix[i])
		sum += v
		sq += v * v
		n++
	}
	if n == 0 {
		return 0, 0
	}
	mean := sum / float64(n)
	variance := sq/float64(n) - mean*mean
	if variance < 0 {
		variance = 0
	}
	return mean, math.Sqrt(variance)
}

// refineClamp clamps a grey value into 0..255.
func refineClamp(v int) int {
	if v < 0 {
		return 0
	}
	if v > 255 {
		return 255
	}
	return v
}

// refineRound rounds to the nearest integer.
func refineRound(x float64) int {
	return int(math.Floor(x + 0.5))
}
