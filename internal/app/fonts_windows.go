//go:build windows

package app

func fontPaths() []string {
	return []string{`C:\Windows\Fonts\tahoma.ttf`, `C:\Windows\Fonts\msyh.ttc`, `C:\Windows\Fonts\malgun.ttf`}
}
