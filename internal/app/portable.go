package app

import (
	"os"
	"path/filepath"
	"strings"
)

// The green (no-install) distribution is a folder the user unpacks anywhere: nothing is
// registered and nothing is written outside it. Its marker file - portable.txt, shipped in the zip
// by scripts/package-portable.sh - is also how the program knows which update package to ask for
// and how to apply it:
//
//	installed copy   -> platform win-x64,          setup.exe -silent -dir <dir> -launch
//	portable copy    -> platform win-x64-portable,  a new zip, unpacked over this folder while the
//	                                                program is closed, then started again
//
// The two never offer each other's package, so an installed copy can never be replaced by a bare
// folder of exes (and lose its shortcuts or registry entry), and a portable copy can never become
// installed by surprise.

// portableMarker is the file that marks a green distribution.
const portableMarker = "portable.txt"

// portablePlatform is the update platform of the green distribution.
const portablePlatform = updatePlat + "-portable"

// executableDir is the folder the program runs from (the folder an update may write to).
func executableDir() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	return filepath.Dir(exe), nil
}

// isPortable reports whether this program runs from an unpacked green distribution.
func isPortable() bool {
	dir, err := executableDir()
	if err != nil {
		return false
	}
	return portableAt(dir)
}

// portableAt reports whether dir holds the marker of a green distribution.
func portableAt(dir string) bool {
	_, err := os.Stat(filepath.Join(dir, portableMarker))
	return err == nil
}

// swapScript is the batch file that replaces the files of a green distribution after this program
// has closed: robocopy copies over the folder (never mirroring, so afvi.toml and captures stay),
// retrying for a minute while the old exe is still locked, then starts the new program and deletes
// itself. It lives here, not in the Windows-only file, because the text is what has to be right and
// this way any platform can check it.
func swapScript(stage, dir, exe string) string {
	q := func(s string) string { return `"` + strings.ReplaceAll(s, `"`, `""`) + `"` }
	return "@echo off\r\n" +
		"rem written by AFVI Teach to update the green distribution in place\r\n" +
		"setlocal\r\n" +
		"set tries=0\r\n" +
		":retry\r\n" +
		"robocopy " + q(stage) + " " + q(dir) + " /e /nfl /ndl /njh /njs /nc /ns /np >nul\r\n" +
		"if %ERRORLEVEL% LSS 8 goto done\r\n" +
		"set /a tries+=1\r\n" +
		"if %tries% GEQ 60 goto done\r\n" +
		"timeout /t 1 /nobreak >nul\r\n" +
		"goto retry\r\n" +
		":done\r\n" +
		"rmdir /s /q " + q(stage) + " >nul 2>nul\r\n" +
		"start \"\" " + q(exe) + "\r\n" +
		"del \"%~f0\"\r\n"
}
