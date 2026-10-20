package core

import (
	"math"
	"testing"
)

// alignCaptureDesign is a synthetic piecewise-constant 640x480 "design" picture: a plain base,
// one diagonal half-plane edge and five rectangles of distinct brightness. Every rectangle
// corner gives a structurally unique window, so ROI matching has a single clear peak.
func alignCaptureDesign() *GrayImage {
	const width, height = 640, 480
	img := NewGray(width, height, 0)
	for y := 0; y < height; y++ {
		row := y * width
		for x := 0; x < width; x++ {
			v := uint8(50)
			if float64(y) >= 100.0+0.5*float64(x) {
				v = 85
			}
			img.Data[row+x] = v
		}
	}
	fill := func(x0, y0, x1, y1 int, v uint8) {
		for y := y0; y < y1; y++ {
			for x := x0; x < x1; x++ {
				img.Data[y*width+x] = v
			}
		}
	}
	fill(40, 40, 160, 120, 140)
	fill(470, 40, 600, 120, 190)
	fill(50, 300, 190, 390, 110)
	fill(480, 330, 600, 430, 160)
	fill(280, 190, 360, 270, 220)
	return img
}

// alignCaptureRois are FULL-resolution design coordinates centred on rectangle corners so each
// window contains a unique corner pattern; together they cover the four quarters and the middle.
func alignCaptureRois() []AlignRoi {
	return []AlignRoi{
		{Name: "TL", Rect: Rect{X: 10, Y: 20, W: 80, H: 60}},
		{Name: "TR", Rect: Rect{X: 440, Y: 20, W: 70, H: 60}},
		{Name: "BL", Rect: Rect{X: 20, Y: 270, W: 90, H: 70}},
		{Name: "BR", Rect: Rect{X: 450, Y: 300, W: 70, H: 80}},
		{Name: "C1", Rect: Rect{X: 250, Y: 160, W: 80, H: 50}},
		{Name: "C2", Rect: Rect{X: 330, Y: 240, W: 100, H: 60}},
	}
}

func alignCaptureFixtures(t *testing.T) (design, capture, template *GrayImage, wp WarpParams) {
	t.Helper()
	design = alignCaptureDesign()
	wp = DefaultCaptureWarp()
	capture = ApplyWarp(design, &wp, 0)
	work, err := DownsampleArea(design, 2)
	if err != nil {
		t.Fatalf("DownsampleArea: %v", err)
	}
	template = GaussianBlur(work, 1.0)
	return design, capture, template, wp
}

func TestAlignCaptureRecoversTheDefaultMisalignment(t *testing.T) {
	const width, height = 640, 480
	design, capture, template, wp := alignCaptureFixtures(t)
	out, err := AlignCapture(template, capture, alignCaptureRois(), 2, DefaultAlignConfig)
	if err != nil {
		t.Fatalf("AlignCapture: %v", err)
	}
	if !out.Usable {
		t.Fatalf("not usable: level %s reasons %+v matches %+v", out.Report.Level, out.Report.Reasons, describeMatches(out.Report.Matches))
	}
	if out.Scale != 2 {
		t.Fatalf("Scale %d, want 2", out.Scale)
	}
	if out.Image == nil {
		t.Fatalf("aligned image missing")
	}
	if out.Report.Matches == nil || len(out.Report.Matches) != 6 {
		t.Fatalf("got %d matches, want 6", len(out.Report.Matches))
	}
	for _, m := range out.Report.Matches {
		if !m.Located {
			t.Fatalf("ROI %s not located (reason %q)", m.Name, m.Reason)
		}
	}
	if !(out.ElapsedMs > 0) {
		t.Fatalf("ElapsedMs %v, want > 0", out.ElapsedMs)
	}

	// The full-resolution fit maps design points onto the warp truth: the fit sends a design
	// (template) point to the capture point, exactly like WarpPoint does.
	for _, p := range []Point{{X: 0, Y: 0}, {X: width - 1, Y: 0}, {X: 0, Y: height - 1}, {X: width - 1, Y: height - 1}, {X: 320, Y: 240}} {
		pred := ApplySimilarity(out.Fit, p)
		truth := WarpPoint(&wp, p, width, height)
		d := math.Hypot(pred.X-truth.X, pred.Y-truth.Y)
		if d > 1.5 {
			t.Errorf("design point %+v: fit maps to %+v, truth is %+v (error %.3f px, want <= 1.5)", p, pred, truth, d)
		}
	}

	// The aligned capture is far closer to the design picture than the raw capture.
	rawDiff, _ := MeanAbsDiff(capture, design, nil)
	alignedDiff, _ := MeanAbsDiff(out.Image, design, nil)
	if !(alignedDiff*4 < rawDiff) {
		t.Fatalf("mean difference raw %.3f aligned %.3f, want at least a four-fold drop", rawDiff, alignedDiff)
	}

	// The image is ReplicateNoData of the resampled capture.
	re := RealignInput(capture, out.Fit, nil)
	again := ReplicateNoData(re)
	if again.Width != width || again.Height != height {
		t.Fatalf("aligned image is %dx%d, want %dx%d", again.Width, again.Height, width, height)
	}
	for i := range again.Data {
		if again.Data[i] != out.Image.Data[i] {
			t.Fatalf("pixel %d differs from ReplicateNoData(RealignInput(capture, Fit, nil))", i)
		}
	}
	// The shift must have opened a no-data border in the plain resample.
	noData := 0
	for _, v := range re.Valid {
		if !v {
			noData++
		}
	}
	if noData == 0 {
		t.Fatalf("expected a no-data border from the shift")
	}
	// The outcome reports the same border as its NoData mask (the inspection skips it).
	if len(out.NoData) != len(re.Valid) {
		t.Fatalf("NoData has %d entries, want %d", len(out.NoData), len(re.Valid))
	}
	for i, v := range re.Valid {
		x, y := i%width, i/width
		frame := x < AlignBorderPx || y < AlignBorderPx || x >= width-AlignBorderPx || y >= height-AlignBorderPx
		if want := !v || frame; out.NoData[i] != want {
			t.Fatalf("NoData[%d] = %v, want %v (valid %v, frame %v)", i, out.NoData[i], want, v, frame)
		}
	}
	// Valid pixels keep their resampled value; the replicated border must match the nearest
	// valid pixel instead of being black.
	for y := 0; y < height; y++ {
		base := y * width
		if !rowHasData(re, y) {
			continue // rows the shift left without any data are checked below
		}
		// right border: nearest valid pixel of the same row
		rx := -1
		for x := width - 1; x >= 0; x-- {
			if re.Valid[base+x] {
				rx = x
				break
			}
		}
		if rx < 0 {
			t.Fatalf("row %d has no valid pixel", y)
		}
		for x := rx + 1; x < width; x++ {
			if out.Image.Data[base+x] != out.Image.Data[base+rx] {
				t.Fatalf("(%d,%d) is %d, want the neighbouring valid pixel's %d", x, y, out.Image.Data[base+x], out.Image.Data[base+rx])
			}
		}
	}
	// The top rows with no data at all are copies of their nearest source row; they must not be
	// the black frame the plain resample leaves.
	for y := 0; y < height; y++ {
		base := y * width
		if re.Valid[base+width/2] {
			continue
		}
		if out.Image.Data[base+width/2] == 0 {
			t.Fatalf("(%d,%d): replicated border is black", width/2, y)
		}
		src := -1
		for d := 1; d < height; d++ {
			if y-d >= 0 && rowHasData(re, y-d) {
				src = y - d
				break
			}
			if y+d < height && rowHasData(re, y+d) {
				src = y + d
				break
			}
		}
		if src < 0 {
			continue
		}
		sbase := src * width
		for x := 0; x < width; x++ {
			if out.Image.Data[base+x] != out.Image.Data[sbase+x] {
				t.Fatalf("(%d,%d) is %d, want the source row %d's %d", x, y, out.Image.Data[base+x], src, out.Image.Data[sbase+x])
			}
		}
	}
}

func rowHasData(re *Realigned, y int) bool {
	for x := 0; x < re.Image.Width; x++ {
		if re.Valid[y*re.Image.Width+x] {
			return true
		}
	}
	return false
}

func describeMatches(ms []RoiMatch) []string {
	out := make([]string, 0, len(ms))
	for _, m := range ms {
		out = append(out, m.Name+":"+m.Reason)
	}
	return out
}

func TestAlignCaptureFullResolutionPath(t *testing.T) {
	const width, height = 640, 480
	_, capture, _, wp := alignCaptureFixtures(t)
	// scale 0 counts as 1: the template is the full-resolution design picture.
	out, err := AlignCapture(alignCaptureDesign(), capture, alignCaptureRois(), 0, DefaultAlignConfig)
	if err != nil {
		t.Fatalf("AlignCapture: %v", err)
	}
	if !out.Usable {
		t.Fatalf("not usable: level %s reasons %+v", out.Report.Level, out.Report.Reasons)
	}
	if out.Scale != 1 {
		t.Fatalf("Scale %d, want 1", out.Scale)
	}
	for _, p := range []Point{{X: 0, Y: 0}, {X: width - 1, Y: height - 1}, {X: 320, Y: 240}} {
		pred := ApplySimilarity(out.Fit, p)
		truth := WarpPoint(&wp, p, width, height)
		if d := math.Hypot(pred.X-truth.X, pred.Y-truth.Y); d > 1.5 {
			t.Errorf("design point %+v: error %.3f px, want <= 1.5", p, d)
		}
	}
}

func TestAlignCaptureErrors(t *testing.T) {
	_, capture, template, _ := alignCaptureFixtures(t)
	rois := alignCaptureRois()
	if _, err := AlignCapture(template, capture, rois[:1], 2, DefaultAlignConfig); err == nil {
		t.Fatalf("one ROI must be an error")
	}
	if _, err := AlignCapture(template, capture, nil, 2, DefaultAlignConfig); err == nil {
		t.Fatalf("no ROIs must be an error")
	}
	if _, err := AlignCapture(NewGray(100, 80, 50), capture, rois, 2, DefaultAlignConfig); err == nil {
		t.Fatalf("a template of the wrong working size must be an error")
	}
	if _, err := AlignCapture(nil, capture, rois, 2, DefaultAlignConfig); err == nil {
		t.Fatalf("a nil template must be an error")
	}
	if _, err := AlignCapture(template, nil, rois, 2, DefaultAlignConfig); err == nil {
		t.Fatalf("a nil capture must be an error")
	}
}

func TestAlignCaptureNotUsableOnAFitlessTemplate(t *testing.T) {
	_, capture, _, _ := alignCaptureFixtures(t)
	// A flat template of the right working size: every ROI is flat, so there is no fit.
	flat := NewGray(capture.Width/2, capture.Height/2, 50)
	out, err := AlignCapture(flat, capture, alignCaptureRois(), 2, DefaultAlignConfig)
	if err != nil {
		t.Fatalf("AlignCapture: %v", err)
	}
	if out.Usable {
		t.Fatalf("flat template must not be usable")
	}
	if out.Report.Level != VerdictNG {
		t.Fatalf("level %s, want ng", out.Report.Level)
	}
	if out.Image != nil {
		t.Fatalf("Image must be nil when not usable")
	}
	if out.Fit != (Similarity{Scale: 1}) {
		t.Fatalf("Fit %+v, want {Scale: 1}", out.Fit)
	}
}

func TestFitToFullResIdentity(t *testing.T) {
	f := Similarity{Tx: 3, Ty: -2, ThetaDeg: 0.7, Scale: 1.01}
	for _, s := range []int{-3, 0, 1} {
		if got := FitToFullRes(f, s); got != f {
			t.Fatalf("scale %d: got %+v, want the fit unchanged", s, got)
		}
	}
}

func TestFitToFullResHandComputed(t *testing.T) {
	f := Similarity{Tx: 10, Ty: -6, ThetaDeg: 0, Scale: 1}
	// theta 0: a=1, b=0, so both correction factors (1-(a-b)) and (1-(a+b)) are 0.
	g2 := FitToFullRes(f, 2)
	if math.Abs(g2.Tx-20) > 1e-12 || math.Abs(g2.Ty+12) > 1e-12 || g2.ThetaDeg != 0 || math.Abs(g2.Scale-1) > 1e-12 {
		t.Fatalf("scale 2: %+v, want Tx 20 Ty -12", g2)
	}
	g3 := FitToFullRes(f, 3)
	if math.Abs(g3.Tx-30) > 1e-12 || math.Abs(g3.Ty+18) > 1e-12 {
		t.Fatalf("scale 3: %+v, want Tx 30 Ty -18", g3)
	}
	// theta 90, scale 2: a = 2*cos(90) = 0, b = 2*sin(90) = 2; s=2, c=(s-1)/2=0.5:
	// Tx_full = 2*7 + 0.5*(1-(0-2)) = 14 + 1.5 = 15.5; Ty_full = 2*3 + 0.5*(1-(0+2)) = 6 - 0.5 = 5.5.
	r := Similarity{Tx: 7, Ty: 3, ThetaDeg: 90, Scale: 2}
	r2 := FitToFullRes(r, 2)
	if math.Abs(r2.Tx-15.5) > 1e-12 || math.Abs(r2.Ty-5.5) > 1e-12 || r2.ThetaDeg != 90 || r2.Scale != 2 {
		t.Fatalf("rotated scale 2: %+v, want Tx 15.5 Ty 5.5", r2)
	}
	// s=3, c=1: Tx_full = 21 + (1+2) = 24; Ty_full = 9 + (1-2) = 8.
	r3 := FitToFullRes(r, 3)
	if math.Abs(r3.Tx-24) > 1e-12 || math.Abs(r3.Ty-8) > 1e-12 {
		t.Fatalf("rotated scale 3: %+v, want Tx 24 Ty 8", r3)
	}
}

func TestFitToFullResMapsPixelCentres(t *testing.T) {
	fit := Similarity{Tx: 4.25, Ty: -3.5, ThetaDeg: 0.8, Scale: 1.004}
	for _, s := range []int{2, 3, 4} {
		sd := float64(s)
		c := (sd - 1) / 2
		full := FitToFullRes(fit, s)
		for _, p := range []Point{{X: 0, Y: 0}, {X: 7, Y: 11}, {X: 31.25, Y: 40}, {X: 160, Y: 120}} {
			q := ApplySimilarity(fit, p)
			got := ApplySimilarity(full, Point{X: sd*p.X + c, Y: sd*p.Y + c})
			want := Point{X: sd*q.X + c, Y: sd*q.Y + c}
			if math.Abs(got.X-want.X) > 1e-9 || math.Abs(got.Y-want.Y) > 1e-9 {
				t.Fatalf("scale %d at working point %+v: got %+v, want %+v", s, p, got, want)
			}
		}
	}
}

func TestReplicateNoDataNearestInRow(t *testing.T) {
	const w, h = 6, 4
	img := &GrayImage{Width: w, Height: h, Data: []uint8{
		0, 0, 0, 0, 0, 0,
		0, 0, 20, 30, 40, 50,
		10, 0, 20, 30, 0, 50,
		1, 2, 3, 4, 5, 6,
	}}
	valid := []bool{
		false, false, false, false, false, false,
		false, false, true, true, true, true,
		true, false, true, true, false, true,
		true, true, true, true, true, true,
	}
	out := ReplicateNoData(&Realigned{Image: img, Valid: valid})
	want := []uint8{
		20, 20, 20, 30, 40, 50, // empty row copied from the nearest row with data (row 1, repaired)
		20, 20, 20, 30, 40, 50, // x0, x1 take the nearer right side (x2)
		10, 10, 20, 30, 30, 50, // x1 and x4 are ties: equal distance takes the left one
		1, 2, 3, 4, 5, 6,
	}
	for i, v := range want {
		if out.Data[i] != v {
			t.Fatalf("pixel %d is %d, want %d", i, out.Data[i], v)
		}
	}
	// The input is not touched.
	if img.Data[1] != 0 || img.Data[7] != 0 || img.Data[13] != 0 || img.Data[16] != 0 {
		t.Fatalf("ReplicateNoData must not modify the input image")
	}
}

func TestReplicateNoDataNothingValidIsAPlainCopy(t *testing.T) {
	img := &GrayImage{Width: 3, Height: 2, Data: []uint8{7, 0, 9, 0, 0, 4}}
	out := ReplicateNoData(&Realigned{Image: img, Valid: []bool{false, false, false, false, false, false}})
	if out.Width != 3 || out.Height != 2 {
		t.Fatalf("size %dx%d, want 3x2", out.Width, out.Height)
	}
	want := []uint8{7, 0, 9, 0, 0, 4}
	for i, v := range want {
		if out.Data[i] != v {
			t.Fatalf("pixel %d is %d, want %d (plain copy)", i, out.Data[i], v)
		}
	}
}
