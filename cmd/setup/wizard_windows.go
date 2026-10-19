//go:build windows

package main

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
	"unsafe"

	"afvi/teach-imgui/internal/installer"
)

// Window messages, styles and control classes used by the wizard.
const (
	wmDestroy       = 0x0002
	wmClose         = 0x0010
	wmCommand       = 0x0111
	wmSetFont       = 0x0030
	wmApp           = 0x8000
	wmProgress      = wmApp + 1
	wmFinished      = wmApp + 2
	bmGetCheck      = 0x00F0
	bmSetCheck      = 0x00F1
	pbmSetRange32   = 0x0406
	pbmSetPos       = 0x0402
	wsChild         = 0x40000000
	wsVisible       = 0x10000000
	wsTabStop       = 0x00010000
	wsCaption       = 0x00C00000
	wsSysMenu       = 0x00080000
	wsMinimizeBox   = 0x00020000
	wsExClientEdge  = 0x00000200
	esAutoHScroll   = 0x0080
	bsPushButton    = 0x0
	bsDefPushButton = 0x1
	bsAutoCheckBox  = 0x3
	ssNoPrefix      = 0x80
	pbsSmooth       = 0x1
	swHide          = 0
	swShow          = 5
	idButtonOK      = 1
	idButtonCancel  = 2
	idEdit          = 101
	idBrowse        = 102
	idMenu          = 103
	idDesktop       = 104
	idPath          = 105
	idLaunch        = 106
)

type wndClassEx struct {
	size       uint32
	style      uint32
	wndProc    uintptr
	clsExtra   int32
	wndExtra   int32
	instance   uintptr
	icon       uintptr
	cursor     uintptr
	background uintptr
	menuName   *uint16
	className  *uint16
	iconSm     uintptr
}

type point struct{ X, Y int32 }

type msgT struct {
	hwnd    uintptr
	message uint32
	wParam  uintptr
	lParam  uintptr
	time    uint32
	pt      point
}

type initCommonControls struct{ size, icc uint32 }

const (
	pageOptions = iota
	pageProgress
	pageDone
)

// wizard is the setup window: an options page, a progress page and a result page.
type wizard struct {
	hwnd     uintptr
	instance uintptr
	p        *installer.Payload
	lang     string
	dir      string // folder offered on the first page
	prevVer  string // version of the installed copy, "" for a fresh install
	dpi      int
	font     uintptr
	bold     uintptr
	page     int

	edit, chkMenu, chkDesktop, chkPath uintptr
	statusLbl, bar, fileLbl            uintptr
	resultTitle, resultBody, noteLbl   uintptr
	chkLaunch                          uintptr
	btnOK, btnCancel                   uintptr
	pages                              [3][]uintptr

	mu       sync.Mutex
	done     int64
	total    int64
	file     string
	busy     atomic.Bool
	opts     installOptions
	warnings []string
	err      error
	ok       bool
	finished chan struct{} // closed when the result page is shown
}

var wiz *wizard

// loopMsg receives the messages of the message loop. It is a package variable on purpose: GetMessage
// writes into it while a window procedure (a Go callback on this goroutine) may grow and move the
// goroutine stack, which a stack variable would not survive.
var loopMsg msgT

func wndProc(hwnd, msg, wparam, lparam uintptr) uintptr {
	if w := wiz; w != nil {
		switch msg {
		case wmCommand:
			w.command(int(wparam & 0xffff))
			return 0
		case wmProgress:
			w.updateProgress()
			return 0
		case wmFinished:
			w.showResult()
			return 0
		case wmClose:
			if w.busy.Load() {
				return 0
			}
			call(procDestroyWindow, hwnd)
			return 0
		case wmDestroy:
			call(procPostQuitMessage, 0)
			return 0
		}
	}
	return call(procDefWindowProc, hwnd, msg, wparam, lparam)
}

var wndProcPtr = syscall.NewCallback(wndProc)

func (w *wizard) sc(v int) int { return v * w.dpi / 96 }

// ctl creates a child control; the coordinates are in 96-dpi units.
func (w *wizard) ctl(class, text string, style, exStyle uint32, x, y, cx, cy, id int, page int, font uintptr) uintptr {
	h := call(procCreateWindowEx, uintptr(exStyle), utf16Ptr(class), utf16Ptr(text), uintptr(style|wsChild|wsVisible),
		w.sc(x), w.sc(y), w.sc(cx), w.sc(cy), w.hwnd, uintptr(id), w.instance, 0)
	call(procSendMessage, h, wmSetFont, font, 1)
	if page >= 0 {
		w.pages[page] = append(w.pages[page], h)
	}
	return h
}

func (w *wizard) static(text string, x, y, cx, cy, page int, font uintptr) uintptr {
	return w.ctl("STATIC", text, ssNoPrefix, 0, x, y, cx, cy, 0, page, font)
}

func (w *wizard) checkbox(text string, x, y, cx, id, page int, checked bool) uintptr {
	h := w.ctl("BUTTON", text, bsAutoCheckBox|wsTabStop, 0, x, y, cx, 20, id, page, w.font)
	if checked {
		call(procSendMessage, h, bmSetCheck, 1, 0)
	}
	return h
}

func (w *wizard) text(h uintptr) string {
	n := int(call(procGetWindowTextLength, h))
	buf := make([]uint16, n+1)
	call(procGetWindowText, h, unsafe.Pointer(&buf[0]), n+1)
	return syscall.UTF16ToString(buf)
}

func setText(h uintptr, s string) { call(procSetWindowText, h, utf16Ptr(s)) }

func checked(h uintptr) bool { return call(procSendMessage, h, bmGetCheck, 0, 0) == 1 }

func (w *wizard) fontFace() string {
	switch w.lang {
	case "zh":
		return "Microsoft YaHei UI"
	case "ko":
		return "Malgun Gothic"
	}
	return "Segoe UI"
}

func (w *wizard) makeFont(weight int) uintptr {
	height := -(9 * w.dpi / 72)
	return call(procCreateFont, uintptr(height), 0, 0, 0, uintptr(weight), 0, 0, 0, 1, 0, 0, 5, 0, utf16Ptr(w.fontFace()))
}

func (w *wizard) t(key string, kv ...string) string {
	m := &w.p.Manifest
	return installer.T(w.lang, key, append([]string{"product", m.Product, "version", m.Version}, kv...)...)
}

func (w *wizard) create() error {
	w.instance = call(procGetModuleHandle, 0)
	call(procInitCommonControlsEx, unsafe.Pointer(&initCommonControls{size: 8, icc: 0x20 | 0x4000}))
	hdc := call(procGetDC, 0)
	w.dpi = int(call(procGetDeviceCaps, hdc, 88)) // LOGPIXELSX
	call(procReleaseDC, 0, hdc)
	if w.dpi < 96 {
		w.dpi = 96
	}
	w.font, w.bold = w.makeFont(400), w.makeFont(700)

	className := utf16Ptr("AFVISetupWindow")
	wc := wndClassEx{
		size:       uint32(unsafe.Sizeof(wndClassEx{})),
		style:      0x3, // CS_HREDRAW | CS_VREDRAW
		wndProc:    wndProcPtr,
		instance:   w.instance,
		icon:       call(procLoadIcon, w.instance, 1), // the icon resource of this executable
		cursor:     call(procLoadCursor, 0, 32512),    // IDC_ARROW
		background: 16,                                // COLOR_BTNFACE + 1
		className:  className,
	}
	if call(procRegisterClassEx, unsafe.Pointer(&wc)) == 0 {
		return errors.New("RegisterClassEx failed")
	}
	style := uint32(wsCaption | wsSysMenu | wsMinimizeBox)
	rect := winRect{0, 0, int32(w.sc(540)), int32(w.sc(424))}
	call(procAdjustWindowRectEx, unsafe.Pointer(&rect), uintptr(style), 0, 0)
	cx, cy := int(rect.Right-rect.Left), int(rect.Bottom-rect.Top)
	x := (int(call(procGetSystemMetrics, 0)) - cx) / 2
	y := (int(call(procGetSystemMetrics, 1)) - cy) / 2
	w.hwnd = call(procCreateWindowEx, 0, className, utf16Ptr(w.t("title")), uintptr(style), x, y, cx, cy, 0, 0, w.instance, 0)
	if w.hwnd == 0 {
		return errors.New("CreateWindowEx failed")
	}
	m := &w.p.Manifest

	// Page 1: where to install and what.
	w.static(m.Product+"  "+w.t("version"), 16, 14, 508, 22, pageOptions, w.bold)
	if w.prevVer != "" {
		w.static(w.t("upgrade", "old", w.prevVer), 16, 40, 508, 34, pageOptions, w.font)
	}
	w.static(w.t("location"), 16, 80, 300, 18, pageOptions, w.font)
	w.edit = w.ctl("EDIT", w.dir, esAutoHScroll|wsTabStop, wsExClientEdge, 16, 100, 412, 24, idEdit, pageOptions, w.font)
	w.ctl("BUTTON", w.t("browse"), bsPushButton|wsTabStop, 0, 436, 99, 88, 26, idBrowse, pageOptions, w.font)
	w.static(w.t("writableNote"), 16, 130, 508, 34, pageOptions, w.font)
	w.static(w.t("components"), 16, 172, 508, 18, pageOptions, w.font)
	var comps []string
	for _, c := range m.Components {
		comps = append(comps, "• "+c.For(w.lang))
	}
	w.static(strings.Join(comps, "\r\n"), 28, 192, 496, 112, pageOptions, w.font)
	w.chkMenu = w.checkbox(w.t("optMenu"), 16, 312, 508, idMenu, pageOptions, true)
	w.chkDesktop = w.checkbox(w.t("optDesktop"), 16, 334, 508, idDesktop, pageOptions, true)
	w.chkPath = w.checkbox(w.t("optPath"), 16, 356, 508, idPath, pageOptions, false)

	// Page 2: progress.
	w.statusLbl = w.static(w.t("installing"), 16, 150, 508, 20, pageProgress, w.font)
	w.bar = w.ctl("msctls_progress32", "", pbsSmooth, 0, 16, 176, 508, 22, 0, pageProgress, w.font)
	call(procSendMessage, w.bar, pbmSetRange32, 0, 1000)
	w.fileLbl = w.static("", 16, 206, 508, 20, pageProgress, w.font)

	// Page 3: result.
	w.resultTitle = w.static("", 16, 20, 508, 24, pageDone, w.bold)
	w.resultBody = w.static("", 16, 54, 508, 52, pageDone, w.font)
	w.noteLbl = w.static("", 16, 116, 508, 180, pageDone, w.font)
	w.chkLaunch = w.checkbox("", 16, 316, 508, idLaunch, pageDone, true)

	w.btnOK = w.ctl("BUTTON", w.t("install"), bsDefPushButton|wsTabStop, 0, 330, 384, 96, 28, idButtonOK, -1, w.font)
	w.btnCancel = w.ctl("BUTTON", w.t("cancel"), bsPushButton|wsTabStop, 0, 432, 384, 92, 28, idButtonCancel, -1, w.font)
	w.showPage(pageOptions)
	return nil
}

func (w *wizard) showPage(n int) {
	w.page = n
	for i, list := range w.pages {
		cmd := uintptr(swHide)
		if i == n {
			cmd = swShow
		}
		for _, h := range list {
			call(procShowWindow, h, cmd)
		}
	}
	switch n {
	case pageOptions:
		setText(w.btnOK, w.t("install"))
		call(procEnableWindow, w.btnOK, 1)
		call(procEnableWindow, w.btnCancel, 1)
		call(procShowWindow, w.btnCancel, swShow)
	case pageProgress:
		call(procEnableWindow, w.btnOK, 0)
		call(procEnableWindow, w.btnCancel, 0)
	case pageDone:
		setText(w.btnOK, w.t("finish"))
		call(procEnableWindow, w.btnOK, 1)
		call(procShowWindow, w.btnCancel, swHide)
		call(procSetFocus, w.btnOK)
	}
}

func (w *wizard) command(id int) {
	switch id {
	case idBrowse:
		if dir := browseForFolder(w.hwnd, w.t("browseTitle")); dir != "" {
			setText(w.edit, dir)
		}
	case idButtonOK:
		switch w.page {
		case pageOptions:
			w.startInstall()
		case pageDone:
			w.finish()
		}
	case idButtonCancel:
		if w.page == pageOptions {
			call(procDestroyWindow, w.hwnd)
		}
	}
}

// writableUnder reports whether a file can be created in dir or, if it does not exist yet, in
// its nearest existing parent.
func writableUnder(dir string) error {
	d := dir
	for {
		if st, err := os.Stat(d); err == nil {
			if !st.IsDir() {
				return fmt.Errorf("%s is not a folder", d)
			}
			break
		}
		parent := filepath.Dir(d)
		if parent == d {
			return fmt.Errorf("%s does not exist", dir)
		}
		d = parent
	}
	f, err := os.CreateTemp(d, ".afvi-setup-test-*")
	if err != nil {
		return err
	}
	name := f.Name()
	f.Close()
	return os.Remove(name)
}

// foreignFolder reports whether dir exists, holds files and is not an installation of ours.
func foreignFolder(dir string) bool {
	entries, err := os.ReadDir(dir)
	if err != nil || len(entries) == 0 {
		return false
	}
	_, err = installer.ReadLog(dir)
	return errors.Is(err, fs.ErrNotExist)
}

func (w *wizard) startInstall() {
	dir := strings.TrimSpace(w.text(w.edit))
	dir = strings.Trim(dir, `"`)
	if !filepath.IsAbs(dir) || strings.HasPrefix(dir, `\\`) && len(dir) < 5 {
		example := defaultInstallDir(w.p.Manifest.Folder)
		messageBox(w.hwnd, w.t("badDir", "example", example), w.t("title"), mbOK|mbIconWarning)
		return
	}
	dir = filepath.Clean(dir)
	if err := writableUnder(dir); err != nil {
		messageBox(w.hwnd, w.t("notWritable", "dir", dir, "err", err.Error()), w.t("title"), mbOK|mbIconWarning)
		return
	}
	if foreignFolder(dir) {
		if messageBox(w.hwnd, w.t("nonEmpty", "dir", dir), w.t("title"), mbYesNo|mbIconQuest|mbDefButton2) != idYes {
			return
		}
	}
	w.opts = installOptions{
		dir:     dir,
		menu:    checked(w.chkMenu),
		desktop: checked(w.chkDesktop),
		path:    checked(w.chkPath),
		lang:    w.lang,
	}
	w.busy.Store(true)
	w.showPage(pageProgress)
	go w.worker(w.opts)
}

func (w *wizard) worker(o installOptions) {
	warnings, err := doInstall(w.p, o, func(done, total int64, name string) {
		w.mu.Lock()
		w.done, w.total, w.file = done, total, name
		w.mu.Unlock()
		call(procPostMessage, w.hwnd, wmProgress, 0, 0)
	})
	w.mu.Lock()
	w.warnings, w.err = warnings, err
	w.mu.Unlock()
	if err != nil {
		logf("install to %s failed: %v", o.dir, err)
	}
	for _, warn := range warnings {
		logf("install to %s: %s", o.dir, warn)
	}
	call(procPostMessage, w.hwnd, wmFinished, 0, 0)
}

func (w *wizard) updateProgress() {
	w.mu.Lock()
	done, total, file := w.done, w.total, w.file
	w.mu.Unlock()
	if total > 0 {
		call(procSendMessage, w.bar, pbmSetPos, uintptr(done*1000/total), 0)
	}
	if file != "" {
		setText(w.fileLbl, w.t("copying", "name", file))
	}
}

func (w *wizard) showResult() {
	w.mu.Lock()
	warnings, err := w.warnings, w.err
	w.mu.Unlock()
	w.busy.Store(false)
	m := &w.p.Manifest
	if err != nil {
		w.ok = false
		w.err = err
		setText(w.resultTitle, w.t("failedTitle"))
		msg := err.Error()
		var inUse *installer.InUseError
		if errors.As(err, &inUse) {
			setText(w.resultBody, w.t("inUse", "path", inUse.Path))
		} else {
			setText(w.resultBody, w.t("failed", "err", msg))
		}
		call(procShowWindow, w.chkLaunch, swHide)
	} else {
		w.ok = true
		setText(w.resultTitle, w.t("doneTitle"))
		setText(w.resultBody, w.t("done", "dir", w.opts.dir))
		var notes []string
		for _, n := range m.Notes {
			notes = append(notes, n.For(w.lang))
		}
		notes = append(notes, warnings...)
		setText(w.noteLbl, strings.Join(notes, "\r\n\r\n"))
		setText(w.chkLaunch, w.t("optLaunch", "program", shortcutProgramName(m, w.lang)))
	}
	w.showPage(pageDone)
	if err != nil {
		call(procShowWindow, w.chkLaunch, swHide)
	}
	select {
	case <-w.finished:
	default:
		close(w.finished)
	}
}

// shortcutProgramName is the name of the main program as the first shortcut calls it.
func shortcutProgramName(m *installer.Manifest, lang string) string {
	for _, s := range m.Shortcuts {
		if s.Target == m.Main {
			return s.Title.For(lang)
		}
	}
	return m.Main
}

func (w *wizard) finish() {
	if w.ok && checked(w.chkLaunch) {
		m := &w.p.Manifest
		if err := startDetached(filepath.Join(w.opts.dir, filepath.FromSlash(m.Main)), w.opts.dir); err != nil {
			logf("start %s: %v", m.Main, err)
		}
	}
	call(procDestroyWindow, w.hwnd)
}

// runWizard shows the setup window and returns the exit code: 0 installed, 1 failed, 3 cancelled.
func runWizard(p *installer.Payload, lang, dir, prevVer string, selftestDir string) int {
	w := &wizard{p: p, lang: lang, dir: dir, prevVer: prevVer, finished: make(chan struct{})}
	wiz = w
	if err := w.create(); err != nil {
		messageBox(0, err.Error(), p.Manifest.Product, mbOK|mbIconError)
		return 1
	}
	call(procShowWindow, w.hwnd, swShow)
	call(procUpdateWindow, w.hwnd)
	call(procSetForegroundWindow, w.hwnd)
	if selftestDir != "" {
		go w.selftest(selftestDir)
	}
	msg := &loopMsg
	for {
		if int32(call(procGetMessage, unsafe.Pointer(msg), 0, 0, 0)) <= 0 {
			break
		}
		if call(procIsDialogMessage, w.hwnd, unsafe.Pointer(msg)) != 0 {
			continue
		}
		call(procTranslateMessage, unsafe.Pointer(msg))
		call(procDispatchMessage, unsafe.Pointer(msg))
	}
	switch {
	case w.ok:
		return 0
	case w.err != nil:
		return 1
	}
	return 3
}

// selftest drives the wizard without a person: it photographs the first page, presses Install,
// photographs the result page and presses Finish. The folder to install to comes from -dir.
func (w *wizard) selftest(out string) {
	fail := func(format string, args ...any) {
		logf("selftest: "+format, args...)
		fmt.Fprintf(os.Stderr, "selftest: "+format+"\n", args...)
		w.mu.Lock()
		w.err = fmt.Errorf(format, args...)
		w.mu.Unlock()
		w.busy.Store(false)
		call(procPostMessage, w.hwnd, wmClose, 0, 0)
	}
	if err := os.MkdirAll(out, 0o755); err != nil {
		fail("%v", err)
		return
	}
	time.Sleep(700 * time.Millisecond)
	if err := captureWindow(w.hwnd, filepath.Join(out, "setup-options.png")); err != nil {
		fail("%v", err)
		return
	}
	call(procPostMessage, w.hwnd, wmCommand, idButtonOK, 0)
	select {
	case <-w.finished:
	case <-time.After(120 * time.Second):
		fail("the installation did not finish within 120 s")
		return
	}
	time.Sleep(400 * time.Millisecond)
	if err := captureWindow(w.hwnd, filepath.Join(out, "setup-done.png")); err != nil {
		fail("%v", err)
		return
	}
	call(procSendMessage, w.chkLaunch, bmSetCheck, 0, 0)
	call(procPostMessage, w.hwnd, wmCommand, idButtonOK, 0)
}
