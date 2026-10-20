// Package golden: user-mask regions excluded from the inspected areas.
// Standard library only.
package golden

import "math"

// Target selects which inspected area a region removes pixels from.
type Target int

const (
	// TargetMetal removes pixels of the inspected Metal area only.
	TargetMetal Target = iota
	// TargetSR removes pixels of the inspected SR area only.
	TargetSR
)

// String returns "metal" or "sr"; any other value yields "metal".
func (t Target) String() string {
	if t == TargetSR {
		return "sr"
	}
	return "metal"
}

// ParseTarget is the inverse of String; only the exact lower-case names
// "metal" and "sr" are accepted.
func ParseTarget(s string) (Target, bool) {
	switch s {
	case "metal":
		return TargetMetal, true
	case "sr":
		return TargetSR, true
	}
	return TargetMetal, false
}

const (
	// MaxRegions is the largest number of regions of one board.
	MaxRegions = 256
	// MaxRegionPoints is the largest number of points of one polygon.
	MaxRegionPoints = 4096
)

// Region is one excluded area: a shape in design-grid image coordinates and
// the area it applies to.
type Region struct {
	Target Target
	Shape  Shape
}

// Valid reports whether the region may be used: a known target, finite
// coordinates, a non-zero-area Rect/Ellipse or a 3..MaxRegionPoints polygon.
func (r Region) Valid() bool {
	if r.Target != TargetMetal && r.Target != TargetSR {
		return false
	}
	s := r.Shape
	switch s.Kind {
	case ShapeRect, ShapeEllipse:
		return finite(s.P0.X) && finite(s.P0.Y) && finite(s.P1.X) && finite(s.P1.Y) &&
			s.P0.X != s.P1.X && s.P0.Y != s.P1.Y
	case ShapePolygon:
		n := len(s.Points)
		if n < 3 || n > MaxRegionPoints {
			return false
		}
		for _, p := range s.Points {
			if !finite(p.X) || !finite(p.Y) {
				return false
			}
		}
		return true
	}
	return false
}

// finite reports a non-NaN, non-infinite coordinate.
func finite(v float64) bool {
	return !math.IsNaN(v) && !math.IsInf(v, 0)
}

// UserMask is the list of excluded regions. The zero value is empty. All
// methods are value receivers and never modify the receiver nor share the
// Points slices of the result with it, so a UserMask may be captured by a job.
type UserMask struct {
	Regions []Region
}

// cloneRegion returns a deep copy of r.
func cloneRegion(r Region) Region {
	out := Region{Target: r.Target, Shape: Shape{
		Kind: r.Shape.Kind,
		P0:   r.Shape.P0,
		P1:   r.Shape.P1,
	}}
	if r.Shape.Points != nil {
		out.Shape.Points = make([]Pt, len(r.Shape.Points))
		copy(out.Shape.Points, r.Shape.Points)
	}
	return out
}

// Clone returns a deep copy of the region (the Points slice is not shared).
func (r Region) Clone() Region { return cloneRegion(r) }

// Clone returns a deep copy. The zero value yields the zero value.
func (u UserMask) Clone() UserMask {
	if u.Regions == nil {
		return UserMask{}
	}
	out := UserMask{Regions: make([]Region, len(u.Regions))}
	for i, r := range u.Regions {
		out.Regions[i] = cloneRegion(r)
	}
	return out
}

// Empty reports whether there are no regions.
func (u UserMask) Empty() bool {
	return len(u.Regions) == 0
}

// Count returns the number of regions of target t.
func (u UserMask) Count(t Target) int {
	n := 0
	for _, r := range u.Regions {
		if r.Target == t {
			n++
		}
	}
	return n
}

// With returns a copy with r appended. An invalid r or a full list returns a
// plain clone.
func (u UserMask) With(r Region) UserMask {
	out := u.Clone()
	if !r.Valid() || len(out.Regions) >= MaxRegions {
		return out
	}
	out.Regions = append(out.Regions, cloneRegion(r))
	return out
}

// Without returns a copy without index i. An out-of-range i returns a plain
// clone.
func (u UserMask) Without(i int) UserMask {
	out := u.Clone()
	if i < 0 || i >= len(out.Regions) {
		return out
	}
	out.Regions = append(out.Regions[:i], out.Regions[i+1:]...)
	return out
}

// WithoutTarget returns a copy without every region of target t.
func (u UserMask) WithoutTarget(t Target) UserMask {
	out := u.Clone()
	if len(out.Regions) == 0 {
		return out
	}
	kept := make([]Region, 0, len(out.Regions))
	for _, r := range out.Regions {
		if r.Target != t {
			kept = append(kept, r)
		}
	}
	if len(kept) == 0 {
		out.Regions = nil
		return out
	}
	out.Regions = kept
	return out
}

// Rasterize returns the union of the shapes of target t on a w*h mask using
// the pixel-centre rule of Add/Erase. It returns nil when there is no region
// of t.
func (u UserMask) Rasterize(w, h int, t Target) *Mask {
	found := false
	for _, r := range u.Regions {
		if r.Target == t {
			found = true
			break
		}
	}
	if !found {
		return nil
	}
	m := NewMask(w, h)
	if w <= 0 || h <= 0 {
		return m
	}
	for _, r := range u.Regions {
		if r.Target != t {
			continue
		}
		Add(m, r.Shape, nil, 0)
	}
	return m
}

// ApplyUser removes the regions from the inspected areas. metalInspect loses
// the pixels of the Metal regions, sr loses the pixels of the SR regions,
// nothing else. Inputs are not modified; a mask without regions of its kind is
// returned as is (same pointer). removedMetal and removedSR count the pixels
// that were really cleared. Invalid inputs stay as they are (counts 0).
func ApplyUser(metalInspect, sr *Mask, u UserMask) (metal, srOut *Mask, removedMetal, removedSR int) {
	metal, srOut = metalInspect, sr
	if u.Count(TargetMetal) > 0 && metalInspect != nil && metalInspect.Valid() {
		metal = metalInspect.Clone()
		removedMetal = eraseRegions(metal, u, TargetMetal)
	}
	if u.Count(TargetSR) > 0 && sr != nil && sr.Valid() {
		srOut = sr.Clone()
		removedSR = eraseRegions(srOut, u, TargetSR)
	}
	return metal, srOut, removedMetal, removedSR
}

// eraseRegions clears the pixels inside the regions of target t on m and
// returns the number of pixels that were set before. Only the shape bounds are
// visited.
func eraseRegions(m *Mask, u UserMask, t Target) int {
	removed := 0
	for _, r := range u.Regions {
		if r.Target != t {
			continue
		}
		x0, y0, x1, y1 := r.Shape.Bounds(m.W, m.H)
		if x1 <= x0 || y1 <= y0 {
			continue
		}
		for y := y0; y < y1; y++ {
			cy := float64(y) + 0.5
			row := y * m.W
			for x := x0; x < x1; x++ {
				if m.Pix[row+x] == 0 {
					continue
				}
				if r.Shape.Contains(float64(x)+0.5, cy) {
					m.Pix[row+x] = 0
					removed++
				}
			}
		}
	}
	return removed
}
