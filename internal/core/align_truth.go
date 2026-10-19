package core

import "math"

// AlignPose 是整体位姿：图像中心被移动的量、旋转角和缩放。教学界面用它把"设定的错位"
// 和"对齐算法测得的错位"摆在一起比较（相似变换的 Tx/Ty 是相对图像原点的，不能直接和
// 以图像中心为基准的 WarpParams.Tx/Ty 比，所以统一换算成中心位移）。
type AlignPose struct {
	ShiftX, ShiftY float64 // 图像中心移动的像素数，向右、向下为正
	ThetaDeg       float64
	Scale          float64
}

// TruthPose 返回扭曲参数 p（真值）对应的整体位姿；翘起和弯曲也算进中心位移里。
func TruthPose(p *WarpParams, width, height int) AlignPose {
	centre := Point{X: float64(width-1) / 2, Y: float64(height-1) / 2}
	moved := WarpPoint(p, centre, width, height)
	scale := p.Scale
	if scale == 0 {
		scale = 1
	}
	return AlignPose{ShiftX: moved.X - centre.X, ShiftY: moved.Y - centre.Y, ThetaDeg: p.ThetaDeg, Scale: scale}
}

// FitPose 返回对齐算法拟合出的相似变换对应的整体位姿。
func FitPose(fit Similarity, width, height int) AlignPose {
	centre := Point{X: float64(width-1) / 2, Y: float64(height-1) / 2}
	moved := ApplySimilarity(fit, centre)
	return AlignPose{ShiftX: moved.X - centre.X, ShiftY: moved.Y - centre.Y, ThetaDeg: fit.ThetaDeg, Scale: fit.Scale}
}

// TruthLevel 是按真值误差（各 ROI 中心的预测位置与真实位置的最大距离）给出的"应有的"判定：
// 不超过 OK 阈值为 OK，超过 NG 阈值为 NG，之间为 MANUAL；没有拟合（NaN）按 NG。
func TruthLevel(maxErrorPx float64, cfg AlignConfig) VerdictLevel {
	switch {
	case math.IsNaN(maxErrorPx) || maxErrorPx > cfg.NgResidualPx:
		return VerdictNG
	case maxErrorPx <= cfg.OkResidualPx:
		return VerdictOK
	}
	return VerdictManual
}

// TruthJudgement 说明算法的判定相对于真值是否正确。
type TruthJudgement string

// 判定与真值的关系。
const (
	JudgeCorrect    TruthJudgement = "correct"     // 判定等级与真值等级一致
	JudgeMissed     TruthJudgement = "missed"      // 漏判：判 OK，但真值不是 OK（放行了实际有偏差的板）
	JudgeFalseAlarm TruthJudgement = "false-alarm" // 误判：判 NG，但真值是 OK（拒绝了实际对齐良好的板）
	JudgeStrict     TruthJudgement = "strict"      // 偏严：判 NG，真值只是 MANUAL
	JudgeCautious   TruthJudgement = "cautious"    // 保守：判 MANUAL（转人工），真值是 OK 或 NG
)

// JudgeAgainstTruth 比较算法的判定 level 与真值判定 truth。
func JudgeAgainstTruth(level, truth VerdictLevel) TruthJudgement {
	switch {
	case level == truth:
		return JudgeCorrect
	case level == VerdictOK:
		return JudgeMissed
	case level == VerdictNG && truth == VerdictOK:
		return JudgeFalseAlarm
	case level == VerdictNG:
		return JudgeStrict
	}
	return JudgeCautious
}

// RoiDiagnosis 用真值解释一个 ROI 的匹配结果，供教学界面说明"为什么这个 ROI 不可靠"。
// 所有长度和 AlignReport 一样是工作分辨率的像素。
type RoiDiagnosis struct {
	MatchErrPx  float64 // 匹配位置到真实位置的距离；没有匹配为 NaN（注意：不是整体拟合的误差）
	FitDevPx    float64 // 匹配位置到整体拟合预测位置的距离，超过 RANSAC 阈值就是外点；没有匹配或拟合为 NaN
	OverhangPx  float64 // 真实位置的窗口伸出图像边缘的最大距离，0 表示完整在图内
	InsideShare float64 // 真实位置的窗口落在图像内的面积比例 0..1，和 AlignConfig.MinOverlap 比较
	Occluded    float64 // 真实位置的窗口里被遮挡盖住的比例 0..1（只数图内部分）
}

// DiagnoseRois 对 rep 里的每个匹配（和 rois 一一对应）给出诊断。p 是真值扭曲参数，
// 其中的 Occlusions 必须和生成输入图时用的一致（工作分辨率）。
func DiagnoseRois(rep *AlignReport, rois []AlignRoi, p *WarpParams, width, height int) []RoiDiagnosis {
	out := make([]RoiDiagnosis, len(rep.Matches))
	for i, m := range rep.Matches {
		d := RoiDiagnosis{MatchErrPx: math.NaN(), FitDevPx: math.NaN()}
		if i < len(rois) {
			roi := rois[i]
			truth := WarpPoint(p, m.Template, width, height)
			if m.Found != nil {
				d.MatchErrPx = math.Hypot(m.Found.X-truth.X, m.Found.Y-truth.Y)
				if rep.Fit != nil {
					pred := ApplySimilarity(*rep.Fit, m.Template)
					d.FitDevPx = math.Hypot(m.Found.X-pred.X, m.Found.Y-pred.Y)
				}
			}
			x0, y0 := truth.X-roi.W/2, truth.Y-roi.H/2
			x1, y1 := x0+roi.W, y0+roi.H
			d.OverhangPx = math.Max(0, math.Max(math.Max(-x0, x1-float64(width)), math.Max(-y0, y1-float64(height))))
			inX := math.Max(0, math.Min(x1, float64(width))-math.Max(x0, 0))
			inY := math.Max(0, math.Min(y1, float64(height))-math.Max(y0, 0))
			if roi.W > 0 && roi.H > 0 {
				d.InsideShare = (inX / roi.W) * (inY / roi.H)
			}
			d.Occluded = occludedShare(p.Occlusions, x0, y0, x1, y1, width, height)
		}
		out[i] = d
	}
	return out
}

// occludedShare 返回窗口 [x0,x1)×[y0,y1) 落在图像内的部分里被遮挡盖住的比例。
// 遮挡的像素范围和 ApplyWarp 一致（向外取整）。
func occludedShare(occs []Occlusion, x0, y0, x1, y1 float64, width, height int) float64 {
	if len(occs) == 0 {
		return 0
	}
	wx0, wy0 := maxInt(int(math.Floor(x0)), 0), maxInt(int(math.Floor(y0)), 0)
	wx1, wy1 := int(math.Ceil(x1)), int(math.Ceil(y1))
	if wx1 > width {
		wx1 = width
	}
	if wy1 > height {
		wy1 = height
	}
	if wx1 <= wx0 || wy1 <= wy0 {
		return 0
	}
	covered := 0
	for y := wy0; y < wy1; y++ {
		for x := wx0; x < wx1; x++ {
			for _, o := range occs {
				if x >= int(math.Floor(o.X)) && x < int(math.Ceil(o.X+o.W)) && y >= int(math.Floor(o.Y)) && y < int(math.Ceil(o.Y+o.H)) {
					covered++
					break
				}
			}
		}
	}
	return float64(covered) / float64((wx1-wx0)*(wy1-wy0))
}
