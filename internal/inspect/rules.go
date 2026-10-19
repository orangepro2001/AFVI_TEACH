package inspect

import (
	"fmt"
	"math"
)

// 小格参考规则（"Offset 1" 绝对值、"Offset 2" 百分比）。
//
// 图像被切成 TileSize x TileSize 像素的小格（默认 400）；某材质的每个有效像素，
// 与"它所在小格内同材质有效像素的平均 GV"（小格平均）比较：
//
//	Offset 1（绝对）：亮 v > 平均 + Bright；暗 v < 平均 - Dark          （Bright/Dark 单位 GV）
//	Offset 2（百分比）：亮 v > 平均 * (1 + Bright/100)；暗 v < 平均 * (1 - Dark/100)
//
// 偏移量 <= 0 表示该极性不检查。每条规则、每个极性有自己的最小连续（8 连通）被标记像素数，
// 因此每个材质新增 2 规则 x 2 极性 x (偏移 + Size) = 8 个参数。

// 小格尺寸常量。TileSize 为 0 表示 DefaultTileSize。
const (
	DefaultTileSize = 400
	MinTileSize     = 16
	MaxTileSize     = 4096
)

// 规则偏移量上限。
const (
	MaxAbsOffset = 255.0  // Offset 1 的上限（GV）
	MaxPctOffset = 1000.0 // Offset 2 的上限（百分比）
)

// 缺陷由哪些规则检出（Defect.Rules 的位）。
const (
	RuleInterval uint8 = 1 << iota // 接受区间 [Low, High]（含局部背景模式）
	RuleAbsolute                   // Offset 1：相对小格平均的绝对 GV 偏差
	RulePercent                    // Offset 2：相对小格平均的百分比偏差
)

// TileRule 是一条相对小格平均的检出规则。
type TileRule struct {
	Enabled      bool    // 关闭的规则不检查任何像素
	Bright, Dark float64 // 亮/暗偏移量：绝对规则为 GV，百分比规则为百分数；<= 0 表示该极性不检查
	BrightSize   int     // 亮缺陷的最小连续被标记像素数，>= 1
	DarkSize     int     // 暗缺陷的最小连续被标记像素数，>= 1
}

// size 返回某极性的最小连续像素数（至少 1）。
func (r TileRule) size(pol Polarity) int {
	n := r.BrightSize
	if pol == Dark {
		n = r.DarkSize
	}
	if n < 1 {
		return 1
	}
	return n
}

// offset 返回某极性的偏移量。
func (r TileRule) offset(pol Polarity) float64 {
	if pol == Dark {
		return r.Dark
	}
	return r.Bright
}

// active 报告该规则对某极性是否起作用。
func (r TileRule) active(pol Polarity) bool {
	return r.Enabled && r.offset(pol) > 0
}

// validate 校验一条已启用规则；max 是偏移量上限。
func (r TileRule) validate(m Material, name string, max float64) error {
	if !r.Enabled {
		return nil
	}
	for _, v := range []struct {
		label string
		val   float64
	}{{"Bright", r.Bright}, {"Dark", r.Dark}} {
		if math.IsNaN(v.val) || math.IsInf(v.val, 0) {
			return fmt.Errorf("材质 %s：%s.%s=%v 必须是有限数", m, name, v.label, v.val)
		}
		if v.val < 0 || v.val > max {
			return fmt.Errorf("材质 %s：%s.%s=%v 超出 [0,%v]", m, name, v.label, v.val, max)
		}
	}
	if r.BrightSize < 1 {
		return fmt.Errorf("材质 %s：%s.BrightSize=%d 必须 >= 1", m, name, r.BrightSize)
	}
	if r.DarkSize < 1 {
		return fmt.Errorf("材质 %s：%s.DarkSize=%d 必须 >= 1", m, name, r.DarkSize)
	}
	return nil
}

// defaultAbsRule / defaultPctRule 是 DefaultParams 使用的规则初值：关闭，但启用后有合理的起点。
func defaultAbsRule() TileRule {
	return TileRule{Enabled: false, Bright: 30, Dark: 30, BrightSize: 3, DarkSize: 3}
}

func defaultPctRule() TileRule {
	return TileRule{Enabled: false, Bright: 15, Dark: 15, BrightSize: 3, DarkSize: 3}
}

// tileSize 返回生效的小格尺寸（0 = 默认）。
func (p Params) tileSize() int {
	if p.TileSize <= 0 {
		return DefaultTileSize
	}
	return p.TileSize
}

// RuleNames 把规则位翻译成短名列表（"interval"、"abs"、"pct"），顺序固定。
func RuleNames(rules uint8) []string {
	var out []string
	if rules&RuleInterval != 0 {
		out = append(out, "interval")
	}
	if rules&RuleAbsolute != 0 {
		out = append(out, "abs")
	}
	if rules&RulePercent != 0 {
		out = append(out, "pct")
	}
	return out
}
