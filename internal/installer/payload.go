package installer

import (
	"archive/zip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
)

// File is one file of the installation: its path inside the install folder (slash separated)
// and where it comes from while the payload is built.
type File struct {
	Name string
	Src  string
}

// Build writes the setup executable: the stub program, then the zip payload (manifest.json
// first, then the files in name order). The zip records its offsets relative to the start of the
// executable, so the result is read back with zip.NewReader like a self-extracting archive.
func Build(out, stub string, m Manifest, files []File) error {
	m.Format = ManifestFormat
	if err := m.validate(); err != nil {
		return err
	}
	for _, f := range files {
		if !validRel(f.Name) || f.Name == manifestName || f.Name == LogName || f.Name == UninstallerName {
			return fmt.Errorf("invalid payload name %q", f.Name)
		}
	}
	sorted := append([]File(nil), files...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Name < sorted[j].Name })
	for i := 1; i < len(sorted); i++ {
		if sorted[i].Name == sorted[i-1].Name {
			return fmt.Errorf("payload name %q is used twice", sorted[i].Name)
		}
	}

	stubData, err := os.ReadFile(stub)
	if err != nil {
		return err
	}
	m.StubSize = int64(len(stubData))
	tmp := out + ".tmp"
	dst, err := os.Create(tmp)
	if err != nil {
		return err
	}
	ok := false
	defer func() {
		dst.Close()
		if !ok {
			os.Remove(tmp)
		}
	}()
	if _, err := dst.Write(stubData); err != nil {
		return err
	}
	zw := zip.NewWriter(dst)
	zw.SetOffset(int64(len(stubData)))
	mw, err := zw.CreateHeader(&zip.FileHeader{Name: manifestName, Method: zip.Deflate})
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	if _, err := mw.Write(data); err != nil {
		return err
	}
	for _, f := range sorted {
		if err := addFile(zw, f); err != nil {
			return err
		}
	}
	if err := zw.Close(); err != nil {
		return err
	}
	if err := dst.Close(); err != nil {
		return err
	}
	ok = true
	return os.Rename(tmp, out)
}

func addFile(zw *zip.Writer, f File) error {
	src, err := os.Open(f.Src)
	if err != nil {
		return err
	}
	defer src.Close()
	st, err := src.Stat()
	if err != nil {
		return err
	}
	if !st.Mode().IsRegular() {
		return fmt.Errorf("%s is not a regular file", f.Src)
	}
	h, err := zip.FileInfoHeader(st)
	if err != nil {
		return err
	}
	h.Name = f.Name
	h.Method = zip.Deflate
	w, err := zw.CreateHeader(h)
	if err != nil {
		return err
	}
	_, err = io.Copy(w, src)
	return err
}

func (m *Manifest) validate() error {
	switch {
	case m.Product == "":
		return errors.New("manifest: product is empty")
	case m.Version == "":
		return errors.New("manifest: version is empty")
	case m.Folder == "" || !validRel(m.Folder):
		return fmt.Errorf("manifest: invalid folder %q", m.Folder)
	case m.Key == "":
		return errors.New("manifest: key is empty")
	case m.Main == "" || !validRel(m.Main):
		return fmt.Errorf("manifest: invalid main program %q", m.Main)
	}
	for _, u := range m.UserData {
		if !validRel(u) || u == "." {
			return fmt.Errorf("manifest: invalid user data path %q", u)
		}
	}
	return nil
}

// Payload is an opened setup executable.
type Payload struct {
	Manifest Manifest
	zr       *zip.ReadCloser
	files    []*zip.File
}

// Open reads the payload that is appended to the executable at path.
func Open(path string) (*Payload, error) {
	zr, err := zip.OpenReader(path)
	if err != nil {
		return nil, fmt.Errorf("no installation data in %s: %w", path, err)
	}
	p := &Payload{zr: zr}
	found := false
	for _, f := range zr.File {
		if f.Name == manifestName {
			rc, err := f.Open()
			if err != nil {
				zr.Close()
				return nil, err
			}
			err = json.NewDecoder(rc).Decode(&p.Manifest)
			rc.Close()
			if err != nil {
				zr.Close()
				return nil, fmt.Errorf("manifest: %w", err)
			}
			found = true
			continue
		}
		p.files = append(p.files, f)
	}
	if !found || p.Manifest.Format != ManifestFormat {
		zr.Close()
		return nil, errors.New("the installation data has no valid manifest")
	}
	if err := p.Manifest.validate(); err != nil {
		zr.Close()
		return nil, err
	}
	for _, f := range p.files {
		if !validRel(f.Name) {
			zr.Close()
			return nil, fmt.Errorf("unsafe path %q in the installation data", f.Name)
		}
	}
	return p, nil
}

// Close releases the executable.
func (p *Payload) Close() error { return p.zr.Close() }

// Names lists the files of the installation.
func (p *Payload) Names() []string {
	out := make([]string, 0, len(p.files))
	for _, f := range p.files {
		if f.FileInfo().IsDir() {
			continue
		}
		out = append(out, f.Name)
	}
	return out
}

// Size is the total size of the files once extracted.
func (p *Payload) Size() int64 {
	var n int64
	for _, f := range p.files {
		n += int64(f.UncompressedSize64)
	}
	return n
}
