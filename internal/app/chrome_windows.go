//go:build windows

package app

import (
	"syscall"
	"unsafe"
)

var (
	procShowWindow           = user32.NewProc("ShowWindow")
	procSystemParametersInfo = user32.NewProc("SystemParametersInfoW")
)

// chromeMinimise minimises the window with the given title (SW_MINIMIZE).
func chromeMinimise(title string) {
	t, err := syscall.UTF16PtrFromString(title)
	if err != nil {
		return
	}
	hwnd, _, _ := procFindWindowW.Call(0, uintptr(unsafe.Pointer(t)))
	if hwnd != 0 {
		procShowWindow.Call(hwnd, 6)
	}
}

// chromeWorkArea is the primary monitor's work area (without the task bar).
func chromeWorkArea() (x, y, w, h int, ok bool) {
	var r winRect
	const spiGetWorkArea = 0x0030
	ret, _, _ := procSystemParametersInfo.Call(spiGetWorkArea, 0, uintptr(unsafe.Pointer(&r)), 0)
	if ret == 0 {
		return 0, 0, 0, 0, false
	}
	return int(r.Left), int(r.Top), int(r.Right - r.Left), int(r.Bottom - r.Top), true
}
