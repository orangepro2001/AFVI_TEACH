package app

import (
	"fmt"
	"strings"

	"github.com/AllenDang/cimgui-go/imgui"

	"afvi/teach-imgui/internal/inspect"
)

func init() {
	Messages(map[string]string{
		"defect.insp.intervalOn":     "使用接受区间（Low / High）",
		"defect.insp.tileStats":      "小格平均 GV：{tiles} 个小格，{min} ~ {max}",
		"defect.col.rule":            "规则",
		"defect.rule.short.interval": "区间",
		"defect.rule.short.abs":      "偏移1",
		"defect.rule.short.pct":      "偏移2",
		"defect.rule.note":           "除了接受区间，还可以按「小格平均」检出：图像被切成 400×400 像素的小格，某材质的像素与它所在小格内同材质像素的平均 GV 比较。偏移1 = 绝对值（GV 差），偏移2 = 百分比；亮/暗各有偏移量和最小连续像素数。偏移量为 0 表示不检查该极性。",
		"defect.rule.tileSize":       "小格尺寸（像素）",
		"defect.rule.smooth":         "小格平均平滑插值",
		"defect.rule.tileNote":       "小格尺寸默认 400。平滑插值在相邻小格中心之间做双线性插值，消除亮度渐变在小格边界处的台阶；关闭时严格使用本格平均。修改小格设置不会重新学习区间。",
		"defect.rule.abs.title":      "偏移1（绝对值：比小格平均亮/暗多少 GV）",
		"defect.rule.pct.title":      "偏移2（百分比：比小格平均亮/暗百分之几）",
		"defect.rule.enable":         "启用",
		"defect.rule.bright":         "亮偏移",
		"defect.rule.dark":           "暗偏移",
		"defect.rule.brightSize":     "亮最小像素数",
		"defect.rule.darkSize":       "暗最小像素数",
		"defect.rule.offsetNote":     "亮：像素 > 小格平均 + 亮偏移（偏移1）或 > 小格平均 × (1 + 亮偏移%)（偏移2）；暗同理。每条规则、每个极性各自按「最小像素数」过滤连通块，被接受的像素合并后作为缺陷报告。",
	}, map[string]string{
		"defect.insp.intervalOn":     "Use accepted interval (Low / High)",
		"defect.insp.tileStats":      "Tile mean GV: {tiles} tiles, {min} ~ {max}",
		"defect.col.rule":            "Rule",
		"defect.rule.short.interval": "Intv",
		"defect.rule.short.abs":      "Off1",
		"defect.rule.short.pct":      "Off2",
		"defect.rule.note":           "Besides the accepted interval, defects can be detected against the tile mean: the image is cut into 400×400 px tiles and a material's pixels are compared with the mean GV of the same material inside their tile. Offset 1 = absolute (GV difference), Offset 2 = percentage; bright and dark each have an offset and a minimum contiguous pixel count. An offset of 0 disables that polarity.",
		"defect.rule.tileSize":       "Tile size (px)",
		"defect.rule.smooth":         "Smooth tile mean (bilinear)",
		"defect.rule.tileNote":       "The tile size defaults to 400. Smoothing interpolates bilinearly between adjacent tile centres, removing the steps a brightness gradient leaves at tile borders; when off, the tile's own mean is used strictly. Changing the tile settings does not re-learn the ranges.",
		"defect.rule.abs.title":      "Offset 1 (absolute: how many GV brighter/darker than the tile mean)",
		"defect.rule.pct.title":      "Offset 2 (percent: how many % brighter/darker than the tile mean)",
		"defect.rule.enable":         "Enabled",
		"defect.rule.bright":         "Bright offset",
		"defect.rule.dark":           "Dark offset",
		"defect.rule.brightSize":     "Bright min pixels",
		"defect.rule.darkSize":       "Dark min pixels",
		"defect.rule.offsetNote":     "Bright: pixel > tile mean + bright offset (Offset 1) or > tile mean × (1 + bright offset%) (Offset 2); dark likewise. Each rule and polarity filters its connected blocks by its own minimum pixel count; the accepted pixels are merged and reported as defects.",
	}, map[string]string{
		"defect.insp.intervalOn":     "허용 구간 사용 (Low / High)",
		"defect.insp.tileStats":      "타일 평균 GV: {tiles}개 타일, {min} ~ {max}",
		"defect.col.rule":            "규칙",
		"defect.rule.short.interval": "구간",
		"defect.rule.short.abs":      "오프1",
		"defect.rule.short.pct":      "오프2",
		"defect.rule.note":           "허용 구간 외에도 '타일 평균' 기준으로 검출할 수 있습니다. 이미지를 400×400 픽셀 타일로 나누고, 어떤 재질의 픽셀을 같은 타일 안 같은 재질 픽셀의 평균 GV와 비교합니다. 오프셋1 = 절대값(GV 차이), 오프셋2 = 백분율이며, 밝음/어두움 각각 오프셋과 최소 연속 픽셀 수를 가집니다. 오프셋이 0이면 해당 극성은 검사하지 않습니다.",
		"defect.rule.tileSize":       "타일 크기 (픽셀)",
		"defect.rule.smooth":         "타일 평균 부드러운 보간",
		"defect.rule.tileNote":       "타일 크기 기본값은 400입니다. 부드러운 보간은 인접 타일 중심 사이를 쌍선형 보간하여 밝기 변화가 타일 경계에서 만드는 계단을 없앱니다. 끄면 해당 타일의 평균을 그대로 사용합니다. 타일 설정을 바꿔도 구간은 다시 학습하지 않습니다.",
		"defect.rule.abs.title":      "오프셋1 (절대값: 타일 평균보다 몇 GV 밝/어두운지)",
		"defect.rule.pct.title":      "오프셋2 (백분율: 타일 평균보다 몇 % 밝/어두운지)",
		"defect.rule.enable":         "사용",
		"defect.rule.bright":         "밝음 오프셋",
		"defect.rule.dark":           "어두움 오프셋",
		"defect.rule.brightSize":     "밝음 최소 픽셀 수",
		"defect.rule.darkSize":       "어두움 최소 픽셀 수",
		"defect.rule.offsetNote":     "밝음: 픽셀 > 타일 평균 + 밝음 오프셋(오프셋1) 또는 > 타일 평균 × (1 + 밝음 오프셋%)(오프셋2), 어두움도 동일합니다. 각 규칙과 극성은 자체 최소 픽셀 수로 연결 블록을 걸러내고, 채택된 픽셀을 합쳐 결함으로 보고합니다.",
	})
}

// drawTileGlobal draws the tile-size input and the smooth switch (they apply to every channel); it reports whether anything changed.
func (p *DefectPanel) drawTileGlobal(params *inspect.Params) bool {
	Note(T("defect.rule.note"))
	changed := false
	ts := params.TileSize
	if ts == 0 {
		ts = inspect.DefaultTileSize
	}
	if SliderInputInt(T("defect.rule.tileSize"), "insp-tile", &ts, inspect.MinTileSize, 2000) {
		for i := range p.inspectParams {
			p.inspectParams[i].TileSize = ts
		}
		changed = true
	}
	smooth := params.TileSmooth
	if Check(T("defect.rule.smooth"), "insp-smooth", &smooth) {
		for i := range p.inspectParams {
			p.inspectParams[i].TileSmooth = smooth
		}
		changed = true
	}
	Note(T("defect.rule.tileNote"))
	return changed
}

// drawMaterialRules draws, for one material of the current channel's parameters, the Offset 1 (absolute) and Offset 2 (percent) rules; it reports whether anything changed.
func (p *DefectPanel) drawMaterialRules(m inspect.Material, mp *inspect.MaterialParams) bool {
	changed := false
	imgui.TextUnformatted(T("defect.rule.abs.title"))
	if Check(T("defect.rule.enable"), fmt.Sprintf("insp-abs-en-%d", m), &mp.Abs.Enabled) {
		changed = true
	}
	if SliderInputF64(T("defect.rule.bright"), fmt.Sprintf("insp-abs-b-%d", m), &mp.Abs.Bright, 0, 255, "%.1f") {
		changed = true
	}
	if SliderInputF64(T("defect.rule.dark"), fmt.Sprintf("insp-abs-d-%d", m), &mp.Abs.Dark, 0, 255, "%.1f") {
		changed = true
	}
	bs := mp.Abs.BrightSize
	if SliderInputInt(T("defect.rule.brightSize"), fmt.Sprintf("insp-abs-bs-%d", m), &bs, 1, 64) {
		mp.Abs.BrightSize = bs
		changed = true
	}
	ds := mp.Abs.DarkSize
	if SliderInputInt(T("defect.rule.darkSize"), fmt.Sprintf("insp-abs-ds-%d", m), &ds, 1, 64) {
		mp.Abs.DarkSize = ds
		changed = true
	}

	imgui.TextUnformatted(T("defect.rule.pct.title"))
	if Check(T("defect.rule.enable"), fmt.Sprintf("insp-pct-en-%d", m), &mp.Pct.Enabled) {
		changed = true
	}
	if SliderInputF64(T("defect.rule.bright"), fmt.Sprintf("insp-pct-b-%d", m), &mp.Pct.Bright, 0, 300, "%.1f") {
		changed = true
	}
	if SliderInputF64(T("defect.rule.dark"), fmt.Sprintf("insp-pct-d-%d", m), &mp.Pct.Dark, 0, 100, "%.1f") {
		changed = true
	}
	pbs := mp.Pct.BrightSize
	if SliderInputInt(T("defect.rule.brightSize"), fmt.Sprintf("insp-pct-bs-%d", m), &pbs, 1, 64) {
		mp.Pct.BrightSize = pbs
		changed = true
	}
	pds := mp.Pct.DarkSize
	if SliderInputInt(T("defect.rule.darkSize"), fmt.Sprintf("insp-pct-ds-%d", m), &pds, 1, 64) {
		mp.Pct.DarkSize = pds
		changed = true
	}
	Note(T("defect.rule.offsetNote"))
	return changed
}

// ruleLabel names the rules that detected a defect, joined with "+", e.g. "Offset 1+Offset 2".
func ruleLabel(rules uint8) string {
	if rules == 0 {
		return "-"
	}
	names := inspect.RuleNames(rules)
	out := make([]string, 0, len(names))
	for _, n := range names {
		switch n {
		case "interval":
			out = append(out, T("defect.rule.short.interval"))
		case "abs":
			out = append(out, T("defect.rule.short.abs"))
		case "pct":
			out = append(out, T("defect.rule.short.pct"))
		default:
			out = append(out, n)
		}
	}
	return strings.Join(out, "+")
}
