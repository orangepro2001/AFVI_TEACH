package app

import "testing"

func TestGoldenSnapSRNeverGrows(t *testing.T) {
	img := refinePicture(60, 40, 40)
	refineFill(img, 10, 8, 50, 32, 16)
	sr := refineBlock(60, 40, 13, 8, 53, 32)

	got := goldenSnapSR(img, sr)
	want := refineBlock(60, 40, 13, 8, 50, 32)
	if !got.Equal(want) {
		t.Fatalf("snapped SR = %d px, want %d px (intersection of layer and picture)", got.Count(), want.Count())
	}
	if &got.Pix[0] == &sr.Pix[0] {
		t.Fatalf("snapped SR must be a copy, not the layer itself")
	}
}

func TestGoldenSnapSRFlatPictureKeepsLayer(t *testing.T) {
	img := refinePicture(60, 40, 100)
	sr := refineBlock(60, 40, 13, 8, 53, 32)

	got := goldenSnapSR(img, sr)
	if !got.Equal(sr) {
		t.Fatalf("flat picture should keep the layer: got %d px, want %d px", got.Count(), sr.Count())
	}
	if &got.Pix[0] == &sr.Pix[0] {
		t.Fatalf("flat picture must return a copy, not the layer itself")
	}
}

func TestGoldenSnapSRNil(t *testing.T) {
	img := refinePicture(60, 40, 40)
	if got := goldenSnapSR(img, nil); got != nil {
		t.Fatalf("nil layer should yield nil, got %+v", got)
	}
}

func TestGoldenSnapSRTooSmallFallsBack(t *testing.T) {
	img := refinePicture(60, 40, 40)
	refineFill(img, 5, 5, 15, 15, 16)
	sr := refineBlock(60, 40, 5, 5, 15, 15)

	got := goldenSnapSR(img, sr)
	if !got.Equal(sr) {
		t.Fatalf("small layer should be preserved: got %d px, want %d px", got.Count(), sr.Count())
	}
}

func TestAlignTemplateGoldenFollowsBoard(t *testing.T) {
	a := newTestApp(t, ProfileTeach)
	dp := defectPanelOf(a)
	if dp == nil {
		t.Fatal("no defect panel")
	}
	g := a.golden
	if g == nil {
		t.Fatal("teach app has no golden board")
	}
	rois := pipelineRois(a.Scene)
	a.align.SetRois(rois)
	if !a.align.Arm() {
		t.Fatal("Arm() failed with two valid ROIs")
	}
	dp.warp = smallWarp()
	dp.renderRev++
	dp.requestRun()
	pollJobs(t, a)

	if got := dp.AlignState(); got != alignDone {
		t.Fatalf("AlignState() = %v, want alignDone\n%s", got, explainAlign(dp.AlignOutcome(), len(rois)))
	}
	out := dp.AlignOutcome()
	if out == nil {
		t.Fatal("AlignOutcome() is nil")
	}
	if out.GoldenRev != g.Rev() {
		t.Fatalf("out.GoldenRev = %d, want %d", out.GoldenRev, g.Rev())
	}
	if dp.tpl == nil || dp.tpl.goldenRev != g.Rev() {
		t.Fatalf("cached template does not follow the board: tpl %+v, board rev %d", dp.tpl, g.Rev())
	}

	rev1 := g.Rev()
	g.SetPadding(g.Padding() + 1)
	pollJobs(t, a)
	if g.Rev() == rev1 {
		t.Fatalf("board revision did not change after SetPadding (%d)", g.Rev())
	}
	if got := dp.AlignState(); got != alignDone {
		t.Fatalf("AlignState() = %v after the board change, want alignDone\n%s", got, explainAlign(dp.AlignOutcome(), len(rois)))
	}
	out2 := dp.AlignOutcome()
	if out2 == nil {
		t.Fatal("AlignOutcome() is nil after the board change")
	}
	if out2.GoldenRev != g.Rev() {
		t.Fatalf("out.GoldenRev = %d, want %d", out2.GoldenRev, g.Rev())
	}
	if dp.tpl == nil || dp.tpl.goldenRev != g.Rev() {
		t.Fatalf("cached template does not follow the board: tpl %+v, board rev %d", dp.tpl, g.Rev())
	}
}

func TestAlignTemplateGoldenStaleWhileBoardChanges(t *testing.T) {
	a := newTestApp(t, ProfileTeach)
	dp := defectPanelOf(a)
	if dp == nil {
		t.Fatal("no defect panel")
	}
	g := a.golden
	if g == nil {
		t.Fatal("teach app has no golden board")
	}
	rois := pipelineRois(a.Scene)
	a.align.SetRois(rois)
	if !a.align.Arm() {
		t.Fatal("Arm() failed with two valid ROIs")
	}
	dp.warp = smallWarp()
	dp.renderRev++
	dp.requestRun()
	pollJobs(t, a)

	if got := dp.AlignState(); got != alignDone {
		t.Fatalf("AlignState() = %v, want alignDone\n%s", got, explainAlign(dp.AlignOutcome(), len(rois)))
	}
	if dp.alignOut == nil {
		t.Fatal("AlignOutcome() is nil")
	}

	// Simulate an outcome made for an older board.
	saved := dp.alignOut.GoldenRev
	dp.alignOut.GoldenRev = g.Rev() - 1
	if got := dp.AlignState(); got != alignWorking {
		t.Fatalf("AlignState() = %v with a stale outcome, want alignWorking", got)
	}

	dp.alignOut.GoldenRev = saved
	if got := dp.AlignState(); got != alignDone {
		t.Fatalf("AlignState() = %v after restoring the rev, want alignDone", got)
	}
}
