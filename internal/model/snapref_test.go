package model_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"afvi/teach-imgui/internal/model"
	"afvi/teach-imgui/internal/model/modeltest"
)

func TestRefRoundTrip(t *testing.T) {
	ref, err := modeltest.WriteReference(t.TempDir())
	if err != nil {
		t.Fatalf("WriteReference: %v", err)
	}
	infos := model.ScanHost("FM1", filepath.Join(ref.Root, "FM1"))
	if len(infos) == 0 {
		t.Fatalf("no models scanned")
	}
	info := infos[0]
	back := model.InfoFromRef(info.Ref())
	if back.ID != info.ID || back.Host != info.Host || back.Model != info.Model ||
		back.Side != info.Side || back.Version != info.Version {
		t.Fatalf("identity changed: %+v vs %+v", back, info)
	}
	if back.Metal != info.Metal || back.SR != info.SR || back.Open != info.Open || back.Median != info.Median {
		t.Fatalf("paths changed: %+v vs %+v", back, info)
	}
}

func TestLayersExist(t *testing.T) {
	ref, err := modeltest.WriteReference(t.TempDir())
	if err != nil {
		t.Fatalf("WriteReference: %v", err)
	}
	infos := model.ScanHost("FM1", filepath.Join(ref.Root, "FM1"))
	if len(infos) == 0 {
		t.Fatalf("no models scanned")
	}
	info := infos[0]
	if err := info.LayersExist(); err != nil {
		t.Fatalf("LayersExist: %v", err)
	}
	if err := os.Remove(info.SR); err != nil {
		t.Fatalf("remove SR: %v", err)
	}
	err = info.LayersExist()
	if err == nil {
		t.Fatalf("LayersExist should fail after removing SR")
	}
	if !strings.Contains(err.Error(), "SR") {
		t.Fatalf("error %q does not mention SR", err)
	}
	empty := model.ModelInfo{}
	if err := empty.LayersExist(); err == nil {
		t.Fatalf("empty paths should be an error")
	}
}
