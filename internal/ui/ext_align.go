package ui

// Extra node kinds of the align pages (internal/wapp/align_view.go, panel_align_flow*.go).

const (
	// KAlignBanner is the big full-width verdict lamp of the align pages; L = title,
	// P.level = ok|manual|ng.
	KAlignBanner = "alignbanner"
	// KToneText is wrapped text in a tone colour; L = text, P.tone = dim|warn|good|bad.
	KToneText = "tonetext"
)

// AlignBanner appends the big verdict lamp: level is ok, manual or ng.
func AlignBanner(title, level string) *Node {
	return cur.add(&Node{K: KAlignBanner, L: title, P: map[string]any{"level": level}})
}

// AlignNoteTone appends wrapped text in the given tone (dim|warn|good|bad).
func AlignNoteTone(text, tone string) *Node {
	return cur.add(&Node{K: KToneText, L: text, P: map[string]any{"tone": tone}})
}
