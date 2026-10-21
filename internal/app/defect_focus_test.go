package app

import (
	"regexp"
	"testing"

	"afvi/teach-imgui/internal/core"
	"afvi/teach-imgui/internal/golden"
	"afvi/teach-imgui/internal/inspect"
)

// testMask builds a w*h mask with pix set.
func testMask(w, h int, pix ...int) *golden.Mask {
	m := golden.NewMask(w, h)
	for _, i := range pix {
		m.Pix[i] = 1
	}
	return m
}

// testGray builds a w*h grey image filled with vals.
func testGray(w, h int, vals ...uint8) *core.GrayImage {
	img := core.NewGray(w, h, 0)
	copy(img.Data, vals)
	return img
}

func TestPackGrayTint(t *testing.T) {
	// 6x4 image, grey = index*5.
	const w, h = 6, 4
	vals := make([]uint8, w*h)
	for i := range vals {
		vals[i] = uint8(i * 5)
	}
	img := testGray(w, h, vals...)
	plain := PackGray(img)
	mask := testMask(w, h, 0, 7, 23)
	col := [3]uint8{255, 0, 255}

	tests := []struct {
		name     string
		mask     *golden.Mask
		col      [3]uint8
		op       int
		wantSame bool // byte-identical to PackGray
		check    func(t *testing.T, got []byte)
	}{
		{
			name: "op 0 mask untouched", mask: mask, col: col, op: 0,
			check: func(t *testing.T, got []byte) {
				for _, i := range []int{0, 7, 23} {
					o := 4 * i
					if got[o] != vals[i] || got[o+1] != vals[i] || got[o+2] != vals[i] {
						t.Fatalf("px %d op 0 changed: %v", i, got[o:o+3])
					}
				}
			},
		},
		{
			name: "op 35 blend", mask: mask, col: col, op: 35,
			check: func(t *testing.T, got []byte) {
				for _, i := range []int{0, 7, 23} {
					o := 4 * i
					want := goldenBlend([3]uint8{vals[i], vals[i], vals[i]}, col, 35)
					if got[o] != want[0] || got[o+1] != want[1] || got[o+2] != want[2] {
						t.Fatalf("px %d op 35 = %v, want %v", i, got[o:o+3], want)
					}
					if got[o+3] != 255 {
						t.Fatalf("px %d alpha = %d", i, got[o+3])
					}
				}
			},
		},
		{
			name: "op 100 full colour", mask: mask, col: col, op: 100,
			check: func(t *testing.T, got []byte) {
				o := 4 * 0
				if got[o] != 255 || got[o+1] != 0 || got[o+2] != 255 {
					t.Fatalf("op 100 = %v, want 255,0,255", got[o:o+3])
				}
			},
		},
		{
			name: "op 150 clamped", mask: mask, col: col, op: 150,
			check: func(t *testing.T, got []byte) {
				o := 4 * 0
				if got[o] != 255 || got[o+1] != 0 || got[o+2] != 255 {
					t.Fatalf("op 150 = %v, want clamped 255,0,255", got[o:o+3])
				}
			},
		},
		{
			name: "out of mask untouched", mask: mask, col: col, op: 35,
			check: func(t *testing.T, got []byte) {
				for i := 0; i < w*h; i++ {
					if i == 0 || i == 7 || i == 23 {
						continue
					}
					o := 4 * i
					if got[o] != vals[i] || got[o+1] != vals[i] || got[o+2] != vals[i] {
						t.Fatalf("out-of-mask px %d changed: %v", i, got[o:o+3])
					}
				}
			},
		},
		{name: "nil mask", mask: nil, col: col, op: 35, wantSame: true},
		{name: "size mismatch", mask: golden.NewMask(w+1, h), col: col, op: 35, wantSame: true},
		{name: "invalid mask", mask: &golden.Mask{W: w, H: h}, col: col, op: 35, wantSame: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := PackGrayTint(img, tc.mask, tc.col, tc.op)
			defer PutRGBA(got)
			if len(got) != len(plain) {
				t.Fatalf("len = %d, want %d", len(got), len(plain))
			}
			if tc.wantSame {
				for i := range plain {
					if got[i] != plain[i] {
						t.Fatalf("byte %d = %d, want %d (PackGray)", i, got[i], plain[i])
					}
				}
				return
			}
			if tc.check != nil {
				tc.check(t, got)
			}
		})
	}
}

func TestPackGrayTintNilImage(t *testing.T) {
	if got := PackGrayTint(nil, testMask(2, 2), [3]uint8{1, 2, 3}, 50); got != nil {
		t.Fatalf("nil image = %v, want nil", got)
	}
}

// sampleReport builds a report with two Metal and two SR defects plus one Open defect.
func sampleReport() *inspect.Report {
	return &inspect.Report{Defects: []inspect.Defect{
		{Material: inspect.Metal, Polarity: inspect.Bright},
		{Material: inspect.SR, Polarity: inspect.Dark},
		{Material: inspect.Metal, Polarity: inspect.Dark},
		{Material: inspect.Open, Polarity: inspect.Bright},
		{Material: inspect.SR, Polarity: inspect.Bright},
	}}
}

func TestDefectFocusMaterial(t *testing.T) {
	tests := []struct {
		target golden.Target
		want   inspect.Material
	}{
		{golden.TargetMetal, inspect.Metal},
		{golden.TargetSR, inspect.SR},
	}
	for _, tc := range tests {
		if got := (defectFocus{target: tc.target}).material(); got != tc.want {
			t.Fatalf("target %d material = %v, want %v", tc.target, got, tc.want)
		}
	}
}

func TestDefectFocusMatch(t *testing.T) {
	rep := sampleReport()
	tests := []struct {
		target  golden.Target
		pol     focusPolarity
		wantIdx []int
	}{
		{golden.TargetMetal, focusAll, []int{0, 2}},
		{golden.TargetMetal, focusBright, []int{0}},
		{golden.TargetMetal, focusDark, []int{2}},
		{golden.TargetSR, focusAll, []int{1, 4}},
		{golden.TargetSR, focusBright, []int{4}},
		{golden.TargetSR, focusDark, []int{1}},
	}
	for _, tc := range tests {
		f := defectFocus{target: tc.target, polarity: tc.pol}
		var got []int
		for i := range rep.Defects {
			if f.match(&rep.Defects[i]) {
				got = append(got, i)
			}
		}
		if len(got) != len(tc.wantIdx) {
			t.Fatalf("target %d pol %d = %v, want %v", tc.target, tc.pol, got, tc.wantIdx)
		}
		for i := range got {
			if got[i] != tc.wantIdx[i] {
				t.Fatalf("target %d pol %d = %v, want %v", tc.target, tc.pol, got, tc.wantIdx)
			}
		}
	}
}

func TestDefectFocusIndices(t *testing.T) {
	rep := sampleReport()
	if got := (defectFocus{target: golden.TargetMetal}).indices(nil); got != nil {
		t.Fatalf("nil report indices = %v, want nil", got)
	}
	get := (defectFocus{target: golden.TargetSR, polarity: focusBright}).indices(rep)
	if len(get) != 1 || get[0] != 4 {
		t.Fatalf("indices = %v, want [4]", get)
	}
	// Fresh slice: mutating it does not affect a later call.
	get[0] = 0
	again := (defectFocus{target: golden.TargetSR, polarity: focusBright}).indices(rep)
	if len(again) != 1 || again[0] != 4 {
		t.Fatalf("fresh slice broken: %v", again)
	}
}

func TestFocusCounts(t *testing.T) {
	rep := sampleReport()
	tests := []struct {
		m                 inspect.Material
		all, bright, dark int
	}{
		{inspect.Metal, 2, 1, 1},
		{inspect.SR, 2, 1, 1},
		{inspect.Open, 1, 1, 0},
		{inspect.Base, 0, 0, 0},
	}
	for _, tc := range tests {
		a, b, d := focusCounts(rep, tc.m)
		if a != tc.all || b != tc.bright || d != tc.dark {
			t.Fatalf("material %v = %d/%d/%d, want %d/%d/%d", tc.m, a, b, d, tc.all, tc.bright, tc.dark)
		}
		if a != b+d {
			t.Fatalf("all != bright+dark for %v", tc.m)
		}
	}
	if a, b, d := focusCounts(nil, inspect.Metal); a != 0 || b != 0 || d != 0 {
		t.Fatalf("nil report counts = %d/%d/%d", a, b, d)
	}
}

func TestFocusColor(t *testing.T) {
	if got := focusColor(golden.TargetMetal); got != [3]uint8{255, 0, 255} {
		t.Fatalf("metal colour = %v", got)
	}
	if got := focusColor(golden.TargetSR); got != goldenSRColor {
		t.Fatalf("sr colour = %v, want %v", got, goldenSRColor)
	}
}

func TestGoldenDerivedFocusMask(t *testing.T) {
	var nilDerived *goldenDerived
	if got := nilDerived.focusMask(golden.TargetMetal); got != nil {
		t.Fatalf("nil receiver focusMask = %v", got)
	}
	metal := testMask(2, 2, 0)
	sr := testMask(2, 2, 1)
	d := &goldenDerived{metalInspect: metal, sr: sr}
	if got := d.focusMask(golden.TargetMetal); got != metal {
		t.Fatalf("metal mask wrong")
	}
	if got := d.focusMask(golden.TargetSR); got != sr {
		t.Fatalf("sr mask wrong")
	}
}

func TestNewTintSpec(t *testing.T) {
	mask := testMask(2, 2, 0)
	tests := []struct {
		name   string
		der    *goldenDerived
		target golden.Target
		wantOn bool
	}{
		{"nil derived", nil, golden.TargetMetal, false},
		{"nil mask", &goldenDerived{}, golden.TargetMetal, false},
		{"size-less mask", &goldenDerived{metalInspect: &golden.Mask{W: 2, H: 2}}, golden.TargetMetal, false},
		{"valid metal", &goldenDerived{metalInspect: mask}, golden.TargetMetal, true},
		{"valid sr", &goldenDerived{sr: mask}, golden.TargetSR, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			spec := newTintSpec(tc.der, tc.target)
			if spec.on() != tc.wantOn {
				t.Fatalf("on = %v, want %v", spec.on(), tc.wantOn)
			}
			if tc.wantOn && spec.target != tc.target {
				t.Fatalf("target = %d, want %d", spec.target, tc.target)
			}
		})
	}
}

func TestTintSpecPack(t *testing.T) {
	img := testGray(2, 2, 0, 10, 20, 30)
	plain := PackGray(img)
	defer PutRGBA(plain)
	spec := tintSpec{}
	got := spec.pack(img)
	defer PutRGBA(got)
	for i := range plain {
		if got[i] != plain[i] {
			t.Fatalf("zero tintSpec pack differs from PackGray at %d", i)
		}
	}

	mask := testMask(2, 2, 0)
	spec = tintSpec{mask: mask, col: focusColor(golden.TargetMetal), target: golden.TargetMetal}
	got2 := spec.pack(img)
	defer PutRGBA(got2)
	o := 0
	if got2[o] == plain[o] && got2[o+1] == plain[o+1] {
		t.Fatalf("tinted pixel unchanged")
	}
	o = 4
	if got2[o] != plain[o] || got2[o+1] != plain[o+1] || got2[o+2] != plain[o+2] {
		t.Fatalf("out-of-mask pixel changed")
	}
}

var focusPlaceholder = regexp.MustCompile(`\{([a-zA-Z0-9_]+)\}`)

func TestDefectFocusMessages(t *testing.T) {
	keys := []string{
		"defect.focus.hint",
		"defect.focus.note.on",
		"defect.focus.note.off",
		"defect.focus.color.metal",
		"defect.focus.color.sr",
		"defect.filter.label",
		"defect.filter.all",
		"defect.filter.bright",
		"defect.filter.dark",
		"defect.filter.pol.all",
		"defect.filter.showing",
		"defect.filter.empty",
		"defect.usermask.target",
		"defect.insp.focusNow",
		"defect.insp.focusMark",
	}
	defer SetLang(LangZh)
	langs := []Lang{LangZh, LangEn, LangKo}
	for _, key := range keys {
		var want []string
		for i, l := range langs {
			SetLang(l)
			text := T(key)
			if text == key || text == "" {
				t.Fatalf("%s missing in %s", key, l)
			}
			ph := focusPlaceholder.FindAllString(text, -1)
			if i == 0 {
				want = ph
				continue
			}
			if len(ph) != len(want) {
				t.Fatalf("%s placeholders %v in %s, want %v", key, ph, l, want)
			}
			seen := map[string]bool{}
			for _, p := range ph {
				seen[p] = true
			}
			for _, p := range want {
				if !seen[p] {
					t.Fatalf("%s placeholder %s missing in %s", key, p, l)
				}
			}
		}
	}
}
