package app

import (
	"strings"
	"testing"

	"afvi/teach-imgui/internal/core"
)

// eachLang runs fn once per selectable language and restores the active language.
func eachLang(t *testing.T, fn func(t *testing.T, lang Lang)) {
	t.Helper()
	prev := currentLang
	defer SetLang(prev)
	for _, l := range []Lang{LangZh, LangEn, LangKo} {
		SetLang(l)
		fn(t, l)
	}
}

func TestExplainAlignNil(t *testing.T) {
	eachLang(t, func(t *testing.T, l Lang) {
		if got := explainAlign(nil, 3); got != "" {
			t.Fatalf("[%s] explainAlign(nil) = %q, want empty", l, got)
		}
	})
}

func TestExplainAlignError(t *testing.T) {
	eachLang(t, func(t *testing.T, l Lang) {
		out := &AlignOutcome{Err: "boom-marker"}
		got := explainAlign(out, 2)
		if !strings.Contains(got, "boom-marker") {
			t.Fatalf("[%s] error text %q does not contain the error message", l, got)
		}
	})
}

func TestExplainAlignUsableNoReasons(t *testing.T) {
	eachLang(t, func(t *testing.T, l Lang) {
		out := &AlignOutcome{Usable: true}
		if got := explainAlign(out, 2); got != "" {
			t.Fatalf("[%s] usable outcome without reasons = %q, want empty", l, got)
		}
	})
}

func TestExplainAlignFlatAllBoxes(t *testing.T) {
	matches := make([]core.RoiMatch, 8)
	for i := range matches {
		matches[i] = core.RoiMatch{Name: "roi" + string(rune('A'+i)), Reason: "flat"}
	}
	out := &AlignOutcome{
		Channel: core.ChGreen,
		TplFlat: true,
		RawFlat: true,
		Report:  core.AlignReport{Matches: matches, Level: core.VerdictNG},
	}
	texts := make(map[Lang]string)
	eachLang(t, func(t *testing.T, l Lang) {
		got := explainAlign(out, 8)
		if got == "" {
			t.Fatalf("[%s] flat outcome produced no text", l)
		}
		if strings.Contains(got, "alignx.") {
			t.Fatalf("[%s] text contains an untranslated key: %q", l, got)
		}
		if !strings.Contains(got, "8") {
			t.Fatalf("[%s] text does not contain the count 8: %q", l, got)
		}
		if hint := T("alignx.hintLight"); !strings.Contains(got, hint) {
			t.Fatalf("[%s] text does not contain the light hint: %q", l, got)
		}
		if both := T("alignx.bothFlat", "ch", channelName(out.Channel)); !strings.Contains(got, both) {
			t.Fatalf("[%s] text does not contain the both-flat line: %q", l, got)
		}
		if raw := T("alignx.rawFlat", "ch", channelName(out.Channel)); strings.Contains(got, raw) {
			t.Fatalf("[%s] text still contains the raw-flat line: %q", l, got)
		}
		if tpl := T("alignx.tplFlat", "ch", channelName(out.Channel)); strings.Contains(got, tpl) {
			t.Fatalf("[%s] text still contains the template-flat line: %q", l, got)
		}
		if roi := T("alignx.roiFlat", "k", 8, "names", "roiA、roiB、roiC、roiD…"); strings.Contains(got, T("alignx.roiFlat", "k", 8, "names", joinNames([]string{"roiA", "roiB", "roiC", "roiD", "roiE", "roiF", "roiG", "roiH"}))) {
			_ = roi
			t.Fatalf("[%s] text still contains the per-box flat line: %q", l, got)
		}
		texts[l] = got
	})
	if texts[LangZh] == texts[LangEn] || texts[LangEn] == texts[LangKo] || texts[LangZh] == texts[LangKo] {
		t.Fatalf("texts do not differ between languages: %v", texts)
	}
}

func TestExplainAlignResidualAndNoLightHint(t *testing.T) {
	matches := []core.RoiMatch{
		{Name: "a", Located: true},
		{Name: "b", Located: true},
		{Name: "c", Located: true},
		{Name: "d", Reason: "low-ncc"},
		{Name: "e", Reason: "low-ncc"},
		{Name: "f", Reason: "low-ncc"},
		{Name: "g", Reason: "low-ncc"},
		{Name: "h", Reason: "low-ncc"},
	}
	out := &AlignOutcome{
		Channel: core.ChRed,
		Report: core.AlignReport{
			Matches: matches,
			Level:   core.VerdictManual,
			Reasons: []core.AlignVerdictReason{{Code: "residual-high", Message: "RES"}},
		},
	}
	eachLang(t, func(t *testing.T, l Lang) {
		got := explainAlign(out, 8)
		if !strings.Contains(got, "RES") {
			t.Fatalf("[%s] text does not contain the report reason: %q", l, got)
		}
		if hint := T("alignx.hintLight"); strings.Contains(got, hint) {
			t.Fatalf("[%s] text contains the light hint but nothing is flat: %q", l, got)
		}
	})
}

func TestExplainAlignBothFlatOnly(t *testing.T) {
	matches := []core.RoiMatch{
		{Name: "a", Reason: "flat"},
		{Name: "b", Reason: "flat"},
		{Name: "c", Reason: "flat"},
	}
	out := &AlignOutcome{
		Channel: core.ChBlue,
		RawFlat: true,
		TplFlat: true,
		Report:  core.AlignReport{Matches: matches, Level: core.VerdictNG},
	}
	eachLang(t, func(t *testing.T, l Lang) {
		got := explainAlign(out, 3)
		ch := channelName(out.Channel)
		if both := T("alignx.bothFlat", "ch", ch); !strings.Contains(got, both) {
			t.Fatalf("[%s] text does not contain the both-flat line: %q", l, got)
		}
		if raw := T("alignx.rawFlat", "ch", ch); strings.Contains(got, raw) {
			t.Fatalf("[%s] text contains the raw-flat line: %q", l, got)
		}
		if tpl := T("alignx.tplFlat", "ch", ch); strings.Contains(got, tpl) {
			t.Fatalf("[%s] text contains the template-flat line: %q", l, got)
		}
		roiFlat := T("alignx.roiFlat", "k", 3, "names", "a、b、c")
		if strings.Contains(got, roiFlat) {
			t.Fatalf("[%s] text contains the per-box flat line: %q", l, got)
		}
	})
}

func TestExplainAlignLowNccNoFlags(t *testing.T) {
	matches := []core.RoiMatch{
		{Name: "a", Located: true},
		{Name: "b", Located: true},
		{Name: "c", Reason: "low-ncc"},
		{Name: "d", Reason: "low-ncc"},
		{Name: "e", Reason: "low-ncc"},
	}
	out := &AlignOutcome{
		Channel: core.ChRed,
		Report:  core.AlignReport{Matches: matches, Level: core.VerdictNG},
	}
	eachLang(t, func(t *testing.T, l Lang) {
		got := explainAlign(out, 5)
		lowNcc := T("alignx.roiLowNcc", "k", 3, "names", "c、d、e")
		if !strings.Contains(got, lowNcc) {
			t.Fatalf("[%s] text does not contain the low-ncc line: %q", l, got)
		}
		if hint := T("alignx.hintBoxes"); !strings.Contains(got, hint) {
			t.Fatalf("[%s] text does not contain the box hint: %q", l, got)
		}
		if hint := T("alignx.hintLight"); strings.Contains(got, hint) {
			t.Fatalf("[%s] text contains the light hint but no flat flag is set: %q", l, got)
		}
	})
}

func TestExplainAlignTruncatesNames(t *testing.T) {
	matches := make([]core.RoiMatch, 6)
	for i := range matches {
		matches[i] = core.RoiMatch{Name: "n" + string(rune('1'+i)), Reason: "flat"}
	}
	out := &AlignOutcome{
		Channel: core.ChRed,
		Report:  core.AlignReport{Matches: matches},
	}
	got := explainAlign(out, 6)
	if !strings.Contains(got, "…") {
		t.Fatalf("more than four names were not truncated: %q", got)
	}
}
