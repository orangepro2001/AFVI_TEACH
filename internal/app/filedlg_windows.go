//go:build windows

package app

import (
	"fmt"
	"os"
	"runtime"
	"syscall"
	"unicode/utf16"
	"unsafe"
)

var (
	comdlg32                 = syscall.NewLazyDLL("comdlg32.dll")
	procGetOpenFileNameW     = comdlg32.NewProc("GetOpenFileNameW")
	procCommDlgExtendedError = comdlg32.NewProc("CommDlgExtendedError")
)

// openFileName mirrors OPENFILENAMEW (amd64 layout, size 152).
type openFileName struct {
	structSize    uint32
	owner         uintptr
	instance      uintptr
	filter        *uint16
	customFilter  *uint16
	maxCustFilter uint32
	filterIndex   uint32
	file          *uint16
	maxFile       uint32
	fileTitle     *uint16
	maxFileTitle  uint32
	initialDir    *uint16
	title         *uint16
	flags         uint32
	fileOffset    uint16
	fileExtension uint16
	defExt        *uint16
	custData      uintptr
	hook          uintptr
	templateName  *uint16
	reserved      uintptr
	reserved2     uint32
	flagsEx       uint32
}

// pickImageFile opens the native open dialog and returns the chosen path ("" when the
// user cancels).
func pickImageFile(initialDir string) (string, error) {
	buf := make([]uint16, 1024)
	filter := utf16.Encode([]rune("Images (*.png;*.jpg;*.jpeg;*.bmp;*.tif;*.tiff)\x00*.png;*.jpg;*.jpeg;*.bmp;*.tif;*.tiff\x00All files (*.*)\x00*.*\x00\x00"))
	title := utf16.Encode([]rune("Load Image\x00"))
	ofn := openFileName{
		structSize:  uint32(unsafe.Sizeof(openFileName{})),
		filter:      &filter[0],
		filterIndex: 1,
		file:        &buf[0],
		maxFile:     uint32(len(buf)),
		title:       &title[0],
		flags:       0x00080000 | 0x00001000 | 0x00000800 | 0x00000008 | 0x00000004,
	}
	if initialDir != "" {
		if st, err := os.Stat(initialDir); err == nil && st.IsDir() {
			if dir, err := syscall.UTF16PtrFromString(initialDir); err == nil {
				ofn.initialDir = dir
			}
		}
	}
	ret, _, _ := procGetOpenFileNameW.Call(uintptr(unsafe.Pointer(&ofn)))
	runtime.KeepAlive(buf)
	runtime.KeepAlive(filter)
	runtime.KeepAlive(title)
	if ret == 0 {
		code, _, _ := procCommDlgExtendedError.Call()
		if code == 0 {
			return "", nil
		}
		return "", fmt.Errorf("file dialog error %#x", code)
	}
	return syscall.UTF16ToString(buf), nil
}
