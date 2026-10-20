package app

import (
	"strings"

	"afvi/teach-imgui/internal/core"
)

// explainAlign turns an alignment outcome into a plain-language explanation for the
// operator. drawn is the number of taught (drawn) ROIs, which is what the operator sees,
// while the report only counts the ROIs that were located. It returns "" when there is
// nothing to say.
func explainAlign(out *AlignOutcome, drawn int) string {
	if out == nil {
		return ""
	}
	if out.Err != "" {
		return T("alignx.error", "msg", out.Err)
	}
	if out.Usable {
		return joinReasons(out.Report.Reasons)
	}

	ch := channelName(out.Channel)
	located := 0
	for _, m := range out.Report.Matches {
		if m.Located {
			located++
		}
	}

	lines := make([]string, 0, 8)
	lines = append(lines, T("alignx.summary", "drawn", drawn, "n", located, "ch", ch))
	// A dark channel is the single cause: state it once instead of repeating the capture flat,
	// the template flat and every per-box consequence.
	if out.RawFlat && out.TplFlat {
		lines = append(lines, T("alignx.bothFlat", "ch", ch))
	} else if out.RawFlat {
		lines = append(lines, T("alignx.rawFlat", "ch", ch))
	} else if out.TplFlat {
		lines = append(lines, T("alignx.tplFlat", "ch", ch))
	}

	// Per-box causes over the ROIs that were not located, only when no flat flag is set (they
	// are consequences of a dark channel rather than information). An unknown or empty reason
	// is treated as "low-ncc": the box exists but the match was too weak.
	if !out.RawFlat && !out.TplFlat {
		flat := make([]string, 0)
		lowNcc := make([]string, 0)
		rng := make([]string, 0)
		for _, m := range out.Report.Matches {
			if m.Located {
				continue
			}
			switch m.Reason {
			case "flat":
				flat = append(flat, m.Name)
			case "out-of-range":
				rng = append(rng, m.Name)
			default:
				lowNcc = append(lowNcc, m.Name)
			}
		}
		if len(flat) > 0 {
			lines = append(lines, T("alignx.roiFlat", "k", len(flat), "names", joinNames(flat)))
		}
		if len(lowNcc) > 0 {
			lines = append(lines, T("alignx.roiLowNcc", "k", len(lowNcc), "names", joinNames(lowNcc)))
		}
		if len(rng) > 0 {
			lines = append(lines, T("alignx.roiRange", "k", len(rng), "names", joinNames(rng)))
		}
	}

	if located >= 2 {
		for _, r := range out.Report.Reasons {
			if r.Code == "too-few-located" {
				continue
			}
			if r.Message != "" {
				lines = append(lines, r.Message)
			}
		}
	}

	if out.RawFlat || out.TplFlat {
		lines = append(lines, T("alignx.hintLight"))
	} else {
		lines = append(lines, T("alignx.hintBoxes"))
	}
	return strings.Join(lines, "\n")
}

// joinReasons joins the messages of reasons with a newline, skipping empty ones.
func joinReasons(reasons []core.AlignVerdictReason) string {
	msgs := make([]string, 0, len(reasons))
	for _, r := range reasons {
		if r.Message != "" {
			msgs = append(msgs, r.Message)
		}
	}
	return strings.Join(msgs, "\n")
}

// joinNames joins ROI names separated by "、" and keeps at most four of them, then "…".
func joinNames(names []string) string {
	const max = 4
	if len(names) > max {
		return strings.Join(names[:max], "、") + "…"
	}
	return strings.Join(names, "、")
}

// channelName translates a render channel for the explanation texts.
func channelName(ch core.CameraChannel) string {
	switch ch {
	case core.ChGreen:
		return T("alignx.ch.green")
	case core.ChBlue:
		return T("alignx.ch.blue")
	}
	return T("alignx.ch.red")
}

func init() {
	Messages(map[string]string{
		"alignx.error":     "对位出错：{msg}",
		"alignx.summary":   "已画 {drawn} 个框，在当前{ch}通道的取像里只定位成功 {n} 个，至少需要 2 个。",
		"alignx.ch.red":    "红(R)",
		"alignx.ch.green":  "绿(G)",
		"alignx.ch.blue":   "蓝(B)",
		"alignx.rawFlat":   "{ch}通道的取像几乎没有明暗变化（可能是全黑：这个通道的光源没有开）。",
		"alignx.tplFlat":   "{ch}通道的设计图没有明暗变化（这个通道的光源没有开，设计图是全黑的）。",
		"alignx.roiFlat":   "{k} 个框（{names}）落在没有明暗变化的位置。",
		"alignx.roiLowNcc": "{k} 个框（{names}）与设计图对不上（相似度太低）。",
		"alignx.roiRange":  "{k} 个框（{names}）超出了图像或搜索范围。",
		"alignx.bothFlat":  "{ch}通道没有光：取像和设计图都是全黑（这个通道的光源没有开）。",
		"alignx.hintLight": "请检查光源（R/G/B）是否都已开启，或者先换到红色通道再点“确定对位”；仍然不行请联系工程师。",
		"alignx.hintBoxes": "请把框移到有明显边缘或图案的位置（避开大块平坦区域，不要贴着图像边缘），再点“确定对位”。",
	}, map[string]string{
		"alignx.error":     "Alignment error: {msg}",
		"alignx.summary":   "You drew {drawn} box(es), but only {n} could be located in the {ch} channel capture; at least 2 are required.",
		"alignx.ch.red":    "red (R)",
		"alignx.ch.green":  "green (G)",
		"alignx.ch.blue":   "blue (B)",
		"alignx.rawFlat":   "The {ch} channel capture has almost no contrast (it may be all black: the light source of this channel is off).",
		"alignx.tplFlat":   "The {ch} channel design picture has no contrast (the light source of this channel is off, so the design picture is all black).",
		"alignx.roiFlat":   "{k} box(es) ({names}) sit on a location without contrast.",
		"alignx.roiLowNcc": "{k} box(es) ({names}) do not match the design picture (similarity too low).",
		"alignx.roiRange":  "{k} box(es) ({names}) fall outside the image or the search range.",
		"alignx.bothFlat":  "The {ch} channel has no light: both the capture and the design picture are all black (its light source is off).",
		"alignx.hintLight": "Check that the R/G/B light sources are all on, or switch to the red channel and press Confirm again; if that does not help, call an engineer.",
		"alignx.hintBoxes": "Move the boxes onto locations with clear edges or patterns (avoid large flat areas and do not place them against the image border), then press Confirm again.",
	}, map[string]string{
		"alignx.error":     "정렬 오류: {msg}",
		"alignx.summary":   "박스를 {drawn} 개 그렸지만 {ch} 채널 촬영 이미지에서 {n} 개만 위치를 찾았습니다. 최소 2 개가 필요합니다.",
		"alignx.ch.red":    "빨강(R)",
		"alignx.ch.green":  "초록(G)",
		"alignx.ch.blue":   "파랑(B)",
		"alignx.rawFlat":   "{ch} 채널 촬영 이미지에 거의 명암 변화가 없습니다 (전부 검은색일 수 있습니다: 이 채널의 광원이 꺼져 있음).",
		"alignx.tplFlat":   "{ch} 채널 설계도에 명암 변화가 없습니다 (이 채널의 광원이 꺼져 있어 설계도가 전부 검은색입니다).",
		"alignx.roiFlat":   "{k} 개의 박스({names})가 명암 변화가 없는 위치에 있습니다.",
		"alignx.roiLowNcc": "{k} 개의 박스({names})가 설계도와 일치하지 않습니다 (유사도가 너무 낮음).",
		"alignx.roiRange":  "{k} 개의 박스({names})가 이미지나 탐색 범위를 벗어났습니다.",
		"alignx.bothFlat":  "{ch} 채널에 빛이 없습니다: 촬영 이미지와 설계도가 모두 검은색입니다 (이 채널의 광원이 꺼져 있음).",
		"alignx.hintLight": "R/G/B 광원이 모두 켜져 있는지 확인하거나 빨강 채널로 바꾼 뒤 '정렬 확인' 을 다시 누르세요. 그래도 안 되면 엔지니어에게 문의하세요.",
		"alignx.hintBoxes": "박스를 뚜렷한 가장자리나 패턴이 있는 위치로 옮기고(큰 평탄 영역과 이미지 가장자리를 피하세요) '정렬 확인' 을 다시 누르세요.",
	})
}
