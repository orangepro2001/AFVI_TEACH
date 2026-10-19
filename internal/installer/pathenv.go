package installer

import (
	"path/filepath"
	"strings"
)

// pathKey is how a PATH entry is compared: no surrounding quotes or spaces, no trailing
// separator, case-insensitive (Windows paths).
func pathKey(entry string) string {
	e := strings.TrimSpace(entry)
	e = strings.Trim(e, `"`)
	e = strings.TrimRight(e, `\/`)
	return strings.ToLower(filepath.ToSlash(e))
}

// PathContains reports whether the PATH value lists dir.
func PathContains(path, dir string) bool {
	want := pathKey(dir)
	for _, e := range strings.Split(path, ";") {
		if pathKey(e) == want && want != "" {
			return true
		}
	}
	return false
}

// PathAdd appends dir to a PATH value unless it is already listed; it reports whether the value
// changed.
func PathAdd(path, dir string) (string, bool) {
	if PathContains(path, dir) {
		return path, false
	}
	if strings.TrimSpace(path) == "" {
		return dir, true
	}
	return strings.TrimRight(path, ";") + ";" + dir, true
}

// PathRemove drops every entry that is dir from a PATH value; it reports whether the value
// changed. The other entries are kept exactly as they are.
func PathRemove(path, dir string) (string, bool) {
	want := pathKey(dir)
	parts := strings.Split(path, ";")
	out := parts[:0:0]
	changed := false
	for _, e := range parts {
		if want != "" && pathKey(e) == want {
			changed = true
			continue
		}
		out = append(out, e)
	}
	if !changed {
		return path, false
	}
	return strings.Join(out, ";"), true
}
