package installer

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// importsOf lists the import paths of the non-test Go files matching the glob.
func importsOf(t *testing.T, glob string) map[string][]string {
	t.Helper()
	files, err := filepath.Glob(glob)
	if err != nil {
		t.Fatal(err)
	}
	out := map[string][]string{}
	fset := token.NewFileSet()
	for _, name := range files {
		if strings.HasSuffix(name, "_test.go") {
			continue
		}
		f, err := parser.ParseFile(fset, name, nil, parser.ImportsOnly)
		if err != nil {
			t.Fatalf("parse %s: %v", name, err)
		}
		for _, imp := range f.Imports {
			out[name] = append(out[name], strings.Trim(imp.Path.Value, `"`))
		}
	}
	return out
}

// The installer package is standard library only, and the two programs around it (the setup
// stub and the payload builder) use nothing of the application either: the setup program must stay
// small and buildable without cgo, and it must never depend on what it installs.
func TestInstallerImportsOnlyTheStandardLibrary(t *testing.T) {
	const self = "afvi/teach-imgui/internal/installer"
	for _, glob := range []string{"*.go", filepath.Join("..", "..", "cmd", "setup", "*.go"), filepath.Join("..", "..", "cmd", "mkinstaller", "*.go")} {
		found := importsOf(t, glob)
		if len(found) == 0 {
			t.Fatalf("no Go files match %s", glob)
		}
		for name, paths := range found {
			for _, p := range paths {
				if strings.HasPrefix(p, "afvi/teach-imgui/") && p != self {
					t.Errorf("%s imports %s", name, p)
				}
				if strings.Contains(p, ".") && strings.Split(p, "/")[0] != "afvi" {
					t.Errorf("%s imports %s, which is not in the standard library", name, p)
				}
			}
		}
	}
	if _, err := os.Stat(filepath.Join("..", "..", "go.mod")); err != nil {
		t.Fatal(err)
	}
}
