package link

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// TestOnlyAppImportsLink guards the other direction: the live link is reached from the
// UI package only, and there only from the files that implement it.
func TestOnlyAppImportsLink(t *testing.T) {
	allowedApp := map[string]bool{"live.go": true, "live_studio.go": true, "live_teach.go": true, "teach_save.go": true}
	root := filepath.Join("..", "..")
	fset := token.NewFileSet()
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if d.Name() == ".git" || d.Name() == "reference" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		f, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
		if err != nil {
			return err
		}
		for _, imp := range f.Imports {
			if p, _ := strconv.Unquote(imp.Path.Value); p != "afvi/teach-imgui/internal/link" {
				continue
			}
			rel, _ := filepath.Rel(root, path)
			rel = filepath.ToSlash(rel)
			if !strings.HasPrefix(rel, "internal/app/") || !allowedApp[filepath.Base(rel)] {
				t.Errorf("%s imports internal/link: only the live link files of internal/app may", rel)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

// TestLinkImportsOnlyStdlibAndSnap guards the contract package: it may use the standard
// library and internal/snap, never the simulation, the inspection, the camera or the UI.
func TestLinkImportsOnlyStdlibAndSnap(t *testing.T) {
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	fset := token.NewFileSet()
	for _, name := range files {
		if _, err := os.Stat(name); err != nil {
			continue
		}
		f, err := parser.ParseFile(fset, name, nil, parser.ImportsOnly)
		if err != nil {
			t.Fatalf("parse %s: %v", name, err)
		}
		for _, imp := range f.Imports {
			path, _ := strconv.Unquote(imp.Path.Value)
			if !strings.Contains(strings.SplitN(path, "/", 2)[0], ".") && !strings.HasPrefix(path, "afvi/") {
				continue // standard library
			}
			if path == "afvi/teach-imgui/internal/snap" {
				continue
			}
			t.Errorf("%s imports %q: internal/link may only import the standard library and internal/snap", name, path)
		}
	}
}
