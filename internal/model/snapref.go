package model

import (
	"fmt"
	"os"

	"afvi/teach-imgui/internal/snap"
)

// Ref converts the model description to the capture meta's model reference (absolute layer paths included).
func (i ModelInfo) Ref() snap.ModelRef {
	return snap.ModelRef{
		ID:      i.ID,
		Host:    i.Host,
		Model:   i.Model,
		Side:    i.Side,
		Version: i.Version,
		Metal:   i.Metal,
		SR:      i.SR,
		Open:    i.Open,
		Median:  i.Median,
	}
}

// InfoFromRef converts back; Pattern and AlignSpec stay empty.
func InfoFromRef(r snap.ModelRef) ModelInfo {
	return ModelInfo{
		ID:      r.ID,
		Host:    r.Host,
		Model:   r.Model,
		Side:    r.Side,
		Version: r.Version,
		Metal:   r.Metal,
		SR:      r.SR,
		Open:    r.Open,
		Median:  r.Median,
	}
}

// LayersExist reports an error naming the first of Metal, SR, Open, Median that is empty or not a regular file.
func (i ModelInfo) LayersExist() error {
	layers := []struct {
		name string
		path string
	}{
		{"Metal", i.Metal},
		{"SR", i.SR},
		{"Open", i.Open},
		{"Median", i.Median},
	}
	for _, l := range layers {
		if l.path == "" {
			return fmt.Errorf("model: %s path is empty", l.name)
		}
		st, err := os.Stat(l.path)
		if err != nil {
			return fmt.Errorf("model: %s: %w", l.name, err)
		}
		if !st.Mode().IsRegular() {
			return fmt.Errorf("model: %s is not a regular file", l.name)
		}
	}
	return nil
}
