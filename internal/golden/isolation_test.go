package golden

import (
	"go/parser"
	"go/token"
	"os"
	"strconv"
	"strings"
	"testing"
)

// TestStdlibOnly fails when any file of this package imports something that
// is not the standard library.
func TestStdlibOnly(t *testing.T) {
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	fset := token.NewFileSet()
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".go") {
			continue
		}
		f, err := parser.ParseFile(fset, name, nil, parser.ImportsOnly)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		for _, imp := range f.Imports {
			path, err := strconv.Unquote(imp.Path.Value)
			if err != nil {
				t.Fatalf("%s: bad import %s", name, imp.Path.Value)
			}
			first := path
			if i := strings.IndexByte(path, '/'); i >= 0 {
				first = path[:i]
			}
			if strings.Contains(first, ".") || strings.HasPrefix(path, "afvi/") {
				t.Fatalf("%s imports non-standard package %q", name, path)
			}
		}
	}
}
