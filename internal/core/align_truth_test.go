package core

import (
	"math"
	"testing"
)

func TestTruthPoseOfAShift(t *testing.T) {
	p := WarpParams{Tx: 10, Ty: -5, Scale: 1, Gain: 1}
	got := TruthPose(&p, 200, 100)
	if math.Abs(got.ShiftX-10) > 1e-9 || math.Abs(got.ShiftY+5) > 1e-9 || got.ThetaDeg != 0 || got.Scale != 1 {
		t.Fatalf("pose %+v, want shift (10,-5)", got)
	}
	// 零值 Scale 当 1。
	if got := TruthPose(&WarpParams{Tx: 3}, 200, 100); got.Scale != 1 || got.ShiftX != 3 {
		t.Fatalf("zero scale: %+v", got)
	}
}

func TestFitPoseMatchesTheTruthOfTheSameSimilarity(t *testing.T) {
	const w, h = 320, 240
	p := WarpParams{Tx: 12, Ty: 7, ThetaDeg: 1.5, Scale: 1.02, Gain: 1}
	// 与 p 等价的相似变换：以图像中心为基准旋转/缩放后再平移。
	cx, cy := float64(w-1)/2, float64(h-1)/2
	th := degToRad(p.ThetaDeg)
	cos, sin := math.Cos(th), math.Sin(th)
	fit := Similarity{
		ThetaDeg: p.ThetaDeg,
		Scale:    p.Scale,
		Tx:       cx + p.Tx - p.Scale*(cos*cx-sin*cy),
		Ty:       cy + p.Ty - p.Scale*(sin*cx+cos*cy),
	}
	want := TruthPose(&p, w, h)
	got := FitPose(fit, w, h)
	if math.Abs(got.ShiftX-want.ShiftX) > 1e-9 || math.Abs(got.ShiftY-want.ShiftY) > 1e-9 ||
		got.ThetaDeg != want.ThetaDeg || got.Scale != want.Scale {
		t.Fatalf("fit pose %+v, truth pose %+v", got, want)
	}
}

func TestTruthPoseIncludesBendAndTilt(t *testing.T) {
	const w, h = 321, 241
	// 弯曲把中心的 y 推到 A·sin(2π·cx/λ)：cx = 160，λ = 640 时 sin(π/2) = 1。
	bend := WarpParams{Scale: 1, Gain: 1, BendAmplitude: 8, BendWavelength: 640}
	if got := TruthPose(&bend, w, h); math.Abs(got.ShiftY-8) > 1e-9 || math.Abs(got.ShiftX) > 1e-9 {
		t.Fatalf("bend pose %+v, want shift (0,8)", got)
	}
	// 透视倾斜不移动中心点本身。
	tilt := WarpParams{Scale: 1, Gain: 1, TiltX: 0.2, TiltY: -0.1}
	if got := TruthPose(&tilt, w, h); math.Abs(got.ShiftX) > 1e-9 || math.Abs(got.ShiftY) > 1e-9 {
		t.Fatalf("tilt moved the centre: %+v", got)
	}
}

func TestTruthLevelThresholds(t *testing.T) {
	cfg := AlignConfig{OkResidualPx: 1.5, NgResidualPx: 5}
	for _, tc := range []struct {
		err  float64
		want VerdictLevel
	}{
		{0, VerdictOK},
		{1.5, VerdictOK},
		{1.6, VerdictManual},
		{5, VerdictManual},
		{5.01, VerdictNG},
		{math.NaN(), VerdictNG},
	} {
		if got := TruthLevel(tc.err, cfg); got != tc.want {
			t.Errorf("TruthLevel(%v) = %q, want %q", tc.err, got, tc.want)
		}
	}
}

func TestJudgeAgainstTruth(t *testing.T) {
	ok, manual, ng := VerdictOK, VerdictManual, VerdictNG
	for _, tc := range []struct {
		level, truth VerdictLevel
		want         TruthJudgement
	}{
		{ok, ok, JudgeCorrect},
		{manual, manual, JudgeCorrect},
		{ng, ng, JudgeCorrect},
		{ok, manual, JudgeMissed},
		{ok, ng, JudgeMissed},
		{ng, ok, JudgeFalseAlarm},
		{ng, manual, JudgeStrict},
		{manual, ok, JudgeCautious},
		{manual, ng, JudgeCautious},
	} {
		if got := JudgeAgainstTruth(tc.level, tc.truth); got != tc.want {
			t.Errorf("JudgeAgainstTruth(%s,%s) = %s, want %s", tc.level, tc.truth, got, tc.want)
		}
	}
}
