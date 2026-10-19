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
