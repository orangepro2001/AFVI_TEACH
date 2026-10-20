package app

import (
	"afvi/teach-imgui/internal/core"
	snapfmt "afvi/teach-imgui/internal/snap"
)

// alignTemplateLight returns the light the design template must be rendered with: the
// light of the adopted capture (meta), because that is what the capture actually shows,
// or the app light when there is no capture to take it from.
func alignTemplateLight(appLight core.LightSetting, meta *snapfmt.Meta) core.LightSetting {
	if meta != nil && len(meta.Recipe.Light.Channels) > 0 {
		return meta.Recipe.Core().Light
	}
	return appLight.Clone()
}

// flatRangeGV is the grey value span between the 5th and the 95th percentile below
// which an image counts as flat.
const flatRangeGV = 3

// imageFlat reports whether img has (almost) no contrast: it is nil/empty or the span
// between the 5th and the 95th percentile of its pixels is below flatRangeGV. The
// percentile measure ignores sparse specks (for example bright defect dots on a black
// frame) that would push a plain standard deviation over the threshold. Pixels are
// sampled with a stride so that a full-resolution frame costs a few milliseconds.
func imageFlat(img *core.GrayImage) bool {
	if img == nil || img.Width <= 0 || img.Height <= 0 || len(img.Data) < img.Width*img.Height {
		return true
	}
	n := img.Width * img.Height
	step := 1
	if n > 1<<20 {
		step = n / (1 << 20)
	}
	var hist [256]int
	count := 0
	for i := 0; i < n; i += step {
		hist[img.Data[i]]++
		count++
	}
	if count == 0 {
		return true
	}
	p5 := histPercentile(hist[:], count, 5)
	p95 := histPercentile(hist[:], count, 95)
	return p95-p5 < flatRangeGV
}

// histPercentile returns the smallest grey value v such that at least pct percent of the
// count samples are <= v.
func histPercentile(hist []int, count int, pct int) int {
	need := (count*pct + 99) / 100
	if need < 1 {
		need = 1
	}
	cum := 0
	for v := 0; v < len(hist); v++ {
		cum += hist[v]
		if cum >= need {
			return v
		}
	}
	return len(hist) - 1
}
