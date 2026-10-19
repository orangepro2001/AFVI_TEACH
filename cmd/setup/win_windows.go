//go:build windows

package main

import (
	"errors"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"unicode/utf16"
	"unsafe"

	"afvi/teach-imgui/internal/installer"
)

var (
	user32   = syscall.NewLazyDLL("user32.dll")
	kernel32 = syscall.NewLazyDLL("kernel32.dll")
	gdi32    = syscall.NewLazyDLL("gdi32.dll")
	shell32  = syscall.NewLazyDLL("shell32.dll")
	ole32    = syscall.NewLazyDLL("ole32.dll")
	advapi32 = syscall.NewLazyDLL("advapi32.dll")
	comctl32 = syscall.NewLazyDLL("comctl32.dll")

	procRegisterClassEx        = user32.NewProc("RegisterClassExW")
	procCreateWindowEx         = user32.NewProc("CreateWindowExW")
	procDefWindowProc          = user32.NewProc("DefWindowProcW")
	procGetMessage             = user32.NewProc("GetMessageW")
	procTranslateMessage       = user32.NewProc("TranslateMessage")
	procDispatchMessage        = user32.NewProc("DispatchMessageW")
	procIsDialogMessage        = user32.NewProc("IsDialogMessageW")
	procPostQuitMessage        = user32.NewProc("PostQuitMessage")
	procShowWindow             = user32.NewProc("ShowWindow")
	procUpdateWindow           = user32.NewProc("UpdateWindow")
	procSendMessage            = user32.NewProc("SendMessageW")
	procPostMessage            = user32.NewProc("PostMessageW")
	procSendMessageTimeout     = user32.NewProc("SendMessageTimeoutW")
	procSetWindowText          = user32.NewProc("SetWindowTextW")
	procGetWindowText          = user32.NewProc("GetWindowTextW")
	procGetWindowTextLength    = user32.NewProc("GetWindowTextLengthW")
	procEnableWindow           = user32.NewProc("EnableWindow")
	procMessageBox             = user32.NewProc("MessageBoxW")
	procGetSystemMetrics       = user32.NewProc("GetSystemMetrics")
	procLoadCursor             = user32.NewProc("LoadCursorW")
	procLoadIcon               = user32.NewProc("LoadIconW")
	procDestroyWindow          = user32.NewProc("DestroyWindow")
	procSetForegroundWindow    = user32.NewProc("SetForegroundWindow")
	procSetFocus               = user32.NewProc("SetFocus")
	procGetDC                  = user32.NewProc("GetDC")
	procReleaseDC              = user32.NewProc("ReleaseDC")
	procGetClientRect          = user32.NewProc("GetClientRect")
	procAdjustWindowRectEx     = user32.NewProc("AdjustWindowRectEx")
	procPrintWindow            = user32.NewProc("PrintWindow")
	procCreateFont             = gdi32.NewProc("CreateFontW")
	procGetDeviceCaps          = gdi32.NewProc("GetDeviceCaps")
	procDeleteObject           = gdi32.NewProc("DeleteObject")
	procCreateCompatibleDC     = gdi32.NewProc("CreateCompatibleDC")
	procCreateCompatibleBitmap = gdi32.NewProc("CreateCompatibleBitmap")
	procSelectObject           = gdi32.NewProc("SelectObject")
	procGetDIBits              = gdi32.NewProc("GetDIBits")
	procDeleteDC               = gdi32.NewProc("DeleteDC")
	procGetModuleHandle        = kernel32.NewProc("GetModuleHandleW")
	procGetUserDefaultUILang   = kernel32.NewProc("GetUserDefaultUILanguage")
	procInitCommonControlsEx   = comctl32.NewProc("InitCommonControlsEx")
	procSHBrowseForFolder      = shell32.NewProc("SHBrowseForFolderW")
	procSHGetPathFromIDList    = shell32.NewProc("SHGetPathFromIDListW")
	procSHGetKnownFolderPath   = shell32.NewProc("SHGetKnownFolderPath")
	procCoInitializeEx         = ole32.NewProc("CoInitializeEx")
	procCoUninitialize         = ole32.NewProc("CoUninitialize")
	procCoCreateInstance       = ole32.NewProc("CoCreateInstance")
	procCoTaskMemFree          = ole32.NewProc("CoTaskMemFree")
	procRegCreateKeyEx         = advapi32.NewProc("RegCreateKeyExW")
	procRegSetValueEx          = advapi32.NewProc("RegSetValueExW")
	procRegDeleteKey           = advapi32.NewProc("RegDeleteKeyW")
	procRegCloseKey            = advapi32.NewProc("RegCloseKey")
)

// raw converts call arguments to the words the Windows API takes. Pointers are passed as
// unsafe.Pointer or *uint16 and stay referenced by the argument slice until the call returns, so
// the garbage collector cannot free what the API is reading.
func raw(args []any) []uintptr {
	out := make([]uintptr, len(args))
	for i, a := range args {
		switch v := a.(type) {
		case uintptr:
			out[i] = v
		case int:
			out[i] = uintptr(v)
		case uint32:
			out[i] = uintptr(v)
		case *uint16:
			out[i] = uintptr(unsafe.Pointer(v))
		case unsafe.Pointer:
			out[i] = uintptr(v)
		default:
			panic(fmt.Sprintf("setup: unsupported call argument %T", a))
		}
	}
	return out
}

func call(p *syscall.LazyProc, args ...any) uintptr {
	r, _, _ := p.Call(raw(args)...)
	runtime.KeepAlive(args)
	return r
}

func utf16Ptr(s string) *uint16 {
	p, err := syscall.UTF16PtrFromString(s)
	if err != nil {
		p, _ = syscall.UTF16PtrFromString(strings.ReplaceAll(s, "\x00", ""))
	}
	return p
}

func utf16PtrToString(p *uint16) string {
	if p == nil {
		return ""
	}
	var buf []uint16
	for i := uintptr(0); ; i += 2 {
		c := *(*uint16)(unsafe.Add(unsafe.Pointer(p), i))
		if c == 0 {
			break
		}
		buf = append(buf, c)
	}
	return string(utf16.Decode(buf))
}

// --- message boxes ------------------------------------------------------------------------

const (
	mbOK          = 0x0
	mbYesNo       = 0x4
	mbRetryCancel = 0x5
	mbIconError   = 0x10
	mbIconQuest   = 0x20
	mbIconWarning = 0x30
	mbIconInfo    = 0x40
	mbDefButton2  = 0x100
	idOK          = 1
	idCancel      = 2
	idRetry       = 4
	idYes         = 6
	idNo          = 7
)

func messageBox(owner uintptr, text, title string, flags uint32) int {
	return int(call(procMessageBox, owner, utf16Ptr(text), utf16Ptr(title), uintptr(flags)))
}

// --- known folders ------------------------------------------------------------------------

type guid struct {
	Data1 uint32
	Data2 uint16
	Data3 uint16
	Data4 [8]byte
}

var (
	folderLocalAppData = guid{0xF1B32785, 0x6FBA, 0x4FCF, [8]byte{0x9D, 0x55, 0x7B, 0x8E, 0x7F, 0x15, 0x70, 0x91}}
	folderPrograms     = guid{0xA77F5D77, 0x2E2B, 0x44C3, [8]byte{0xA6, 0xA2, 0xAB, 0xA6, 0x01, 0x05, 0x4A, 0x51}}
	folderDesktop      = guid{0xB4BFCC3A, 0xDB2C, 0x424C, [8]byte{0xB0, 0x29, 0x7F, 0xE9, 0x9A, 0x87, 0xC6, 0x41}}
)

// knownFolder returns a shell folder of the current user (redirected folders such as a OneDrive
// desktop are honoured).
func knownFolder(id *guid) (string, error) {
	var p *uint16
	hr := call(procSHGetKnownFolderPath, unsafe.Pointer(id), 0, 0, unsafe.Pointer(&p))
	if int32(hr) < 0 {
		return "", fmt.Errorf("SHGetKnownFolderPath: %#x", uint32(hr))
	}
	defer call(procCoTaskMemFree, unsafe.Pointer(p))
	return utf16PtrToString(p), nil
}

// defaultInstallDir is %LOCALAPPDATA%\Programs\<folder>: writable without administrator rights,
// which matters because the programs write their configuration and captures next to themselves.
func defaultInstallDir(folder string) string {
	base, err := knownFolder(&folderLocalAppData)
	if err != nil || base == "" {
		base = filepath.Join(os.Getenv("USERPROFILE"), "AppData", "Local")
	}
	return filepath.Join(base, "Programs", folder)
}

func userLangID() uint16 { return uint16(call(procGetUserDefaultUILang)) }

// --- registry (HKEY_CURRENT_USER) ---------------------------------------------------------

const (
	hkeyCurrentUser = 0x80000001
	keyWrite        = 0x20006
	regSZ           = 1
	regExpandSZ     = 2
	regDWord        = 4
)

type regKey syscall.Handle

func regCreate(path string) (regKey, error) {
	var h syscall.Handle
	r := call(procRegCreateKeyEx, hkeyCurrentUser, utf16Ptr(path), 0, 0, 0, keyWrite, 0, unsafe.Pointer(&h), 0)
	if r != 0 {
		return 0, fmt.Errorf("registry %s: %w", path, syscall.Errno(r))
	}
	return regKey(h), nil
}

func (k regKey) close() { call(procRegCloseKey, uintptr(k)) }

func (k regKey) setString(name, value string, expand bool) error {
	data := utf16.Encode([]rune(value + "\x00"))
	typ := uintptr(regSZ)
	if expand {
		typ = regExpandSZ
	}
	r := call(procRegSetValueEx, uintptr(k), utf16Ptr(name), 0, typ, unsafe.Pointer(&data[0]), uintptr(len(data)*2))
	if r != 0 {
		return fmt.Errorf("registry value %s: %w", name, syscall.Errno(r))
	}
	return nil
}

func (k regKey) setDWord(name string, v uint32) error {
	r := call(procRegSetValueEx, uintptr(k), utf16Ptr(name), 0, regDWord, unsafe.Pointer(&v), 4)
	if r != 0 {
		return fmt.Errorf("registry value %s: %w", name, syscall.Errno(r))
	}
	return nil
}

// regGetString reads a string value of HKEY_CURRENT_USER\path and reports its type.
func regGetString(path, name string) (value string, typ uint32, ok bool) {
	var h syscall.Handle
	if syscall.RegOpenKeyEx(syscall.HKEY_CURRENT_USER, utf16Ptr(path), 0, syscall.KEY_READ, &h) != nil {
		return "", 0, false
	}
	defer syscall.RegCloseKey(h)
	n := uint32(0)
	if err := syscall.RegQueryValueEx(h, utf16Ptr(name), nil, &typ, nil, &n); err != nil || n == 0 {
		return "", 0, false
	}
	if typ != regSZ && typ != regExpandSZ {
		return "", 0, false
	}
	buf := make([]uint16, n/2+1)
	n = uint32(len(buf) * 2)
	if err := syscall.RegQueryValueEx(h, utf16Ptr(name), nil, &typ, (*byte)(unsafe.Pointer(&buf[0])), &n); err != nil {
		return "", 0, false
	}
	return syscall.UTF16ToString(buf), typ, true
}

func regDeleteTreeKey(path string) {
	call(procRegDeleteKey, hkeyCurrentUser, utf16Ptr(path))
}

const environmentKey = `Environment`

// editUserPath adds dir to or removes it from the user's PATH and tells running programs; it
// reports whether the value changed.
func editUserPath(dir string, add bool) (bool, error) {
	cur, typ, ok := regGetString(environmentKey, "Path")
	if !ok {
		cur, typ = "", regExpandSZ
	}
	var next string
	var changed bool
	if add {
		next, changed = installer.PathAdd(cur, dir)
	} else {
		next, changed = installer.PathRemove(cur, dir)
	}
	if !changed {
		return false, nil
	}
	k, err := regCreate(environmentKey)
	if err != nil {
		return false, err
	}
	defer k.close()
	if err := k.setString("Path", next, typ == regExpandSZ); err != nil {
		return false, err
	}
	broadcastEnvironmentChange()
	return true, nil
}

func broadcastEnvironmentChange() {
	var result uintptr
	call(procSendMessageTimeout, 0xffff, 0x1A, 0, utf16Ptr("Environment"), 2, 3000, unsafe.Pointer(&result))
}

// --- Add/Remove Programs entry ------------------------------------------------------------

func uninstallKeyPath(key string) string {
	return `Software\Microsoft\Windows\CurrentVersion\Uninstall\` + key
}

func writeUninstallEntry(m *installer.Manifest, dir string, sizeKB uint32, date string) error {
	k, err := regCreate(uninstallKeyPath(m.Key))
	if err != nil {
		return err
	}
	defer k.close()
	unins := `"` + filepath.Join(dir, installer.UninstallerName) + `"`
	for name, value := range map[string]string{
		"DisplayName":          m.Product,
		"DisplayVersion":       m.Version,
		"Publisher":            m.Publisher,
		"InstallLocation":      dir,
		"InstallDate":          date,
		"DisplayIcon":          filepath.Join(dir, filepath.FromSlash(m.Main)),
		"UninstallString":      unins + " -uninstall",
		"QuietUninstallString": unins + " -uninstall -silent",
	} {
		if err := k.setString(name, value, false); err != nil {
			return err
		}
	}
	for name, v := range map[string]uint32{"NoModify": 1, "NoRepair": 1, "EstimatedSize": sizeKB} {
		if err := k.setDWord(name, v); err != nil {
			return err
		}
	}
	return nil
}

// existingInstall returns the folder of an installed copy (from the Add/Remove entry, and only
// when its log is still there) and its version.
func existingInstall(m *installer.Manifest) (dir, version string) {
	d, _, ok := regGetString(uninstallKeyPath(m.Key), "InstallLocation")
	if !ok || d == "" {
		return "", ""
	}
	if _, err := installer.ReadLog(d); err != nil {
		return "", ""
	}
	version, _, _ = regGetString(uninstallKeyPath(m.Key), "DisplayVersion")
	return d, version
}

// --- shortcuts (IShellLinkW through COM) --------------------------------------------------

var (
	clsidShellLink  = guid{0x00021401, 0, 0, [8]byte{0xC0, 0, 0, 0, 0, 0, 0, 0x46}}
	iidIShellLinkW  = guid{0x000214F9, 0, 0, [8]byte{0xC0, 0, 0, 0, 0, 0, 0, 0x46}}
	iidIPersistFile = guid{0x0000010B, 0, 0, [8]byte{0xC0, 0, 0, 0, 0, 0, 0, 0x46}}
)

// comCall calls method idx of the COM object obj (slot numbers count IUnknown's three first).
func comCall(obj unsafe.Pointer, idx int, args ...any) uintptr {
	vtbl := *(*unsafe.Pointer)(obj)
	fn := *(*uintptr)(unsafe.Add(vtbl, idx*int(unsafe.Sizeof(uintptr(0)))))
	all := make([]uintptr, 0, len(args)+1)
	all = append(all, uintptr(obj))
	all = append(all, raw(args)...)
	r, _, _ := syscall.SyscallN(fn, all...)
	runtime.KeepAlive(args)
	return r
}

const (
	coinitApartment = 0x2
	clsctxInproc    = 0x1
)

func comOK(hr uintptr) bool { return int32(hr) >= 0 }

// createShortcut writes a .lnk file.
func createShortcut(lnk, target, args, workDir, iconFile, description string) error {
	// COM is initialised per OS thread: stay on this one until CoUninitialize.
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	hr := call(procCoInitializeEx, 0, coinitApartment)
	if comOK(hr) {
		defer call(procCoUninitialize)
	}
	var link unsafe.Pointer
	hr = call(procCoCreateInstance, unsafe.Pointer(&clsidShellLink), 0, clsctxInproc, unsafe.Pointer(&iidIShellLinkW), unsafe.Pointer(&link))
	if !comOK(hr) || link == nil {
		return fmt.Errorf("CoCreateInstance(ShellLink): %#x", uint32(hr))
	}
	defer comCall(link, 2) // Release

	str := utf16Ptr
	const (
		slotSetDescription      = 7
		slotSetWorkingDirectory = 9
		slotSetArguments        = 11
		slotSetIconLocation     = 17
		slotSetPath             = 20
	)
	if hr := comCall(link, slotSetPath, str(target)); !comOK(hr) {
		return fmt.Errorf("IShellLink.SetPath: %#x", uint32(hr))
	}
	if args != "" {
		comCall(link, slotSetArguments, str(args))
	}
	if workDir != "" {
		comCall(link, slotSetWorkingDirectory, str(workDir))
	}
	if description != "" {
		comCall(link, slotSetDescription, str(description))
	}
	if iconFile != "" {
		comCall(link, slotSetIconLocation, str(iconFile), 0)
	}
	var file unsafe.Pointer
	if hr := comCall(link, 0, unsafe.Pointer(&iidIPersistFile), unsafe.Pointer(&file)); !comOK(hr) || file == nil {
		return fmt.Errorf("IShellLink.QueryInterface(IPersistFile): %#x", uint32(hr))
	}
	defer comCall(file, 2)
	const slotSave = 6
	hr = comCall(file, slotSave, str(lnk), 1)
	if !comOK(hr) {
		return fmt.Errorf("IPersistFile.Save(%s): %#x", lnk, uint32(hr))
	}
	return nil
}

// shortcutFileName turns a title into a file name.
func shortcutFileName(title string) string {
	bad := strings.NewReplacer("<", " ", ">", " ", ":", " ", "\"", " ", "/", " ", "\\", " ", "|", " ", "?", " ", "*", " ")
	return strings.TrimSpace(bad.Replace(title)) + ".lnk"
}

// --- folder chooser -----------------------------------------------------------------------

type browseInfo struct {
	owner       uintptr
	root        uintptr
	displayName *uint16
	title       *uint16
	flags       uint32
	callback    uintptr
	param       uintptr
	image       int32
}

// browseForFolder shows the shell's folder dialog; it returns "" when the user cancels.
func browseForFolder(owner uintptr, title string) string {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	hr := call(procCoInitializeEx, 0, coinitApartment)
	if comOK(hr) {
		defer call(procCoUninitialize)
	}
	name := make([]uint16, 260)
	bi := browseInfo{owner: owner, displayName: &name[0], title: utf16Ptr(title), flags: 0x1 | 0x10 | 0x40} // folders only, edit box, new dialog style
	pidl := call(procSHBrowseForFolder, unsafe.Pointer(&bi))
	if pidl == 0 {
		return ""
	}
	defer call(procCoTaskMemFree, pidl)
	path := make([]uint16, 520)
	if call(procSHGetPathFromIDList, pidl, unsafe.Pointer(&path[0])) == 0 {
		return ""
	}
	return syscall.UTF16ToString(path)
}

// --- window capture (for the self-test screenshots) ---------------------------------------

type winRect struct{ Left, Top, Right, Bottom int32 }

type bitmapInfoHeader struct {
	Size          uint32
	Width         int32
	Height        int32
	Planes        uint16
	BitCount      uint16
	Compression   uint32
	SizeImage     uint32
	XPelsPerMeter int32
	YPelsPerMeter int32
	ClrUsed       uint32
	ClrImportant  uint32
}

// captureWindow saves the client area of a window as PNG.
func captureWindow(hwnd uintptr, path string) error {
	var r winRect
	call(procGetClientRect, hwnd, unsafe.Pointer(&r))
	w, h := int(r.Right-r.Left), int(r.Bottom-r.Top)
	if w <= 0 || h <= 0 {
		return fmt.Errorf("empty client rect %dx%d", w, h)
	}
	hdc := call(procGetDC, hwnd)
	defer call(procReleaseDC, hwnd, hdc)
	mem := call(procCreateCompatibleDC, hdc)
	defer call(procDeleteDC, mem)
	bmp := call(procCreateCompatibleBitmap, hdc, uintptr(w), uintptr(h))
	defer call(procDeleteObject, bmp)
	old := call(procSelectObject, mem, bmp)
	const pwClientOnly, pwRenderFullContent = 1, 2
	ok := call(procPrintWindow, hwnd, mem, pwClientOnly|pwRenderFullContent)
	call(procSelectObject, mem, old)
	if ok == 0 {
		return errors.New("PrintWindow failed")
	}
	bi := bitmapInfoHeader{Size: 40, Width: int32(w), Height: -int32(h), Planes: 1, BitCount: 32}
	buf := make([]byte, w*h*4)
	if call(procGetDIBits, mem, bmp, 0, uintptr(h), unsafe.Pointer(&buf[0]), unsafe.Pointer(&bi), 0) == 0 {
		return errors.New("GetDIBits failed")
	}
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for i := 0; i < w*h; i++ {
		img.Pix[4*i] = buf[4*i+2]
		img.Pix[4*i+1] = buf[4*i+1]
		img.Pix[4*i+2] = buf[4*i]
		img.Pix[4*i+3] = 255
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img)
}
