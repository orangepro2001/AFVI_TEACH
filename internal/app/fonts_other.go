//go:build !windows

package app

// fontPaths lists Latin, CJK and Hangul fonts commonly present on Linux (the self-test
// under Xvfb); missing files are skipped by setupFonts.
func fontPaths() []string {
	return []string{
		"/usr/share/fonts/truetype/dejavu/DejaVuSans.ttf",
		"/usr/share/fonts/opentype/noto/NotoSansCJK-Regular.ttc",
		"/usr/share/fonts/opentype/noto/NotoSerifCJK-Regular.ttc",
		"/usr/share/fonts/truetype/wqy/wqy-microhei.ttc",
	}
}
