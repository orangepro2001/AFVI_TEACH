//go:build windows

package app

import (
	"fmt"
	"os/exec"
	"syscall"
	"unsafe"
)

// user32 comes from capture_windows.go; only the procedures this file needs are new here.
var (
	procEnumWindows        = user32.NewProc("EnumWindows")
	procGetWindowThreadPID = user32.NewProc("GetWindowThreadProcessId")
	procIsWindowVisible    = user32.NewProc("IsWindowVisible")
	procGetWindowRect      = user32.NewProc("GetWindowRect")
	procSetWindowPos       = user32.NewProc("SetWindowPos")
	procMonitorFromWindow  = user32.NewProc("MonitorFromWindow")
	procGetMonitorInfo     = user32.NewProc("GetMonitorInfoW")
	procGetForegroundWnd   = user32.NewProc("GetForegroundWindow")
)

// spawnPeer starts the same executable as the other face in its own process, detached from this
// one, and tiles the two windows side by side.
func spawnPeer(p Profile) error {
	exe, err := siblingExecutable()
	if err != nil {
		return err
	}
	cmd := exec.Command(exe, peerArgs(p)...)
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: 0x00000008} // DETACHED_PROCESS
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("starting %s: %w", exe, err)
	}
	pid := uint32(cmd.Process.Pid)
	// the child needs a moment to create its window; tiling runs on a goroutine so the UI thread
	// is never blocked, and it gives up quietly when the window never appears
	go func() {
		_ = cmd.Wait()
	}()
	go tileWithPeer(pid)
	return nil
}

// monitorInfo is the Win32 MONITORINFO; the rectangle type is the package's winRect.
type monitorInfo struct {
	cbSize    uint32
	rcMonitor winRect
	rcWork    winRect
	dwFlags   uint32
}

// tileWindows puts self and peer on the left and right half of the monitor that holds this
// window: the face the user was looking at stays where it was (left), the new one appears right.
func tileWindows(self, peer syscall.Handle) {
	work, ok := workAreaOf(self)
	if !ok {
		return
	}
	w := (work.Right - work.Left) / 2
	h := work.Bottom - work.Top
	const swpNoZOrder, swpNoActivate, swpShowWindow = 0x0004, 0x0010, 0x0040
	_, _, _ = procSetWindowPos.Call(uintptr(self), 0,
		uintptr(work.Left), uintptr(work.Top), uintptr(w), uintptr(h),
		swpNoZOrder|swpNoActivate|swpShowWindow)
	_, _, _ = procSetWindowPos.Call(uintptr(peer), 0,
		uintptr(work.Left+w), uintptr(work.Top), uintptr(work.Right-work.Left-w), uintptr(h),
		swpNoZOrder|swpNoActivate|swpShowWindow)
}

// workAreaOf returns the work area (screen minus taskbar) of the monitor holding hwnd.
func workAreaOf(hwnd syscall.Handle) (winRect, bool) {
	const monitorDefaultToNearest = 2
	mon, _, _ := procMonitorFromWindow.Call(uintptr(hwnd), monitorDefaultToNearest)
	if mon == 0 {
		return winRect{}, false
	}
	mi := monitorInfo{cbSize: uint32(unsafe.Sizeof(monitorInfo{}))}
	ok, _, _ := procGetMonitorInfo.Call(mon, uintptr(unsafe.Pointer(&mi)))
	if ok == 0 {
		return winRect{}, false
	}
	return mi.rcWork, true
}

// tileWithPeer waits for the new process to show a window and then tiles both. It polls for a
// few seconds (a cold start on a slow machine takes a moment) and does nothing when the window
// never appears, because a window that is not there cannot be tiled.
func tileWithPeer(pid uint32) {
	self := windowOfProcess(windowsPID())
	if self == 0 {
		return
	}
	deadline := uptimeMS() + 8000
	for uptimeMS() < deadline {
		if peer := windowOfProcess(pid); peer != 0 {
			tileWindows(self, peer)
			return
		}
		sleepMS(200)
	}
}

// windowOfProcess finds the first visible top-level window owned by pid.
func windowOfProcess(pid uint32) syscall.Handle {
	var found syscall.Handle
	cb := syscall.NewCallback(func(hwnd uintptr, _ uintptr) uintptr {
		var owner uint32
		_, _, _ = procGetWindowThreadPID.Call(hwnd, uintptr(unsafe.Pointer(&owner)))
		if owner != pid {
			return 1 // continue
		}
		if vis, _, _ := procIsWindowVisible.Call(hwnd); vis == 0 {
			return 1
		}
		var r winRect
		if ok, _, _ := procGetWindowRect.Call(hwnd, uintptr(unsafe.Pointer(&r))); ok == 0 {
			return 1
		}
		// Skip the 0-size and tool windows ImGui/GLFW create before the real one.
		if r.Right-r.Left < 200 || r.Bottom-r.Top < 120 {
			return 1
		}
		found = syscall.Handle(hwnd)
		return 0 // stop
	})
	_, _, _ = procEnumWindows.Call(cb, 0)
	return found
}
