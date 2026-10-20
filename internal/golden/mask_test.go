package golden

import (
	"math/rand"
	"testing"
)

// bruteRef is the brute-force reference straight from the definition.
func bruteRef(m *Mask, n int, dilate bool) *Mask {
	if m == nil {
		return nil
	}
	if n <= 0 {
		return m.Clone()
	}
	if n > MaxMorph {
		n = MaxMorph
	}
	out := NewMask(m.W, m.H)
	for y := 0; y < m.H; y++ {
		for x := 0; x < m.W; x++ {
			var any, all bool
			for dy := -n; dy <= n; dy++ {
				for dx := -n; dx <= n; dx++ {
					px, py := x+dx, y+dy
					if px < 0 || py < 0 || px >= m.W || py >= m.H {
						continue
					}
					in := m.Pix[py*m.W+px] != 0
					if in {
						any = true
					} else {
						all = true
					}
				}
			}
			if dilate && any {
				out.Pix[y*m.W+x] = 1
			}
			if !dilate && !all {
				out.Pix[y*m.W+x] = 1
			}
		}
	}
	return out
}

func randomMask(r *rand.Rand, w, h int, density int) *Mask {
	m := NewMask(w, h)
	for i := range m.Pix {
		if r.Intn(100) < density {
			m.Pix[i] = uint8(1 + r.Intn(255))
		}
	}
	return m
}

func TestErodeDilateMatchBrute(t *testing.T) {
	sizes := [][2]int{{1, 1}, {7, 5}, {17, 13}, {40, 31}}
	radii := []int{0, 1, 2, 3, 5, 40}
	for seed, size := range sizes {
		for _, dense := range []int{30, 70} {
			r := rand.New(rand.NewSource(int64(seed*1000 + dense)))
			m := randomMask(r, size[0], size[1], dense)
			for _, n := range radii {
				wantE := bruteRef(m, n, false)
				gotE := Erode(m, n)
				if !gotE.Equal(wantE) {
					t.Fatalf("Erode size %v dense %d n %d mismatch", size, dense, n)
				}
				wantD := bruteRef(m, n, true)
				gotD := Dilate(m, n)
				if !gotD.Equal(wantD) {
					t.Fatalf("Dilate size %v dense %d n %d mismatch", size, dense, n)
				}
				// Open / Close equal their definitions.
				open := Dilate(Erode(m, n), n)
				if !Open(m, n).Equal(open) {
					t.Fatalf("Open size %v dense %d n %d mismatch", size, dense, n)
				}
				closeM := Erode(Dilate(m, n), n)
				if !Close(m, n).Equal(closeM) {
					t.Fatalf("Close size %v dense %d n %d mismatch", size, dense, n)
				}
			}
		}
	}
}

func TestMorphNilAndZero(t *testing.T) {
	if Erode(nil, 2) != nil {
		t.Fatal("Erode(nil) must be nil")
	}
	if Dilate(nil, 2) != nil {
		t.Fatal("Dilate(nil) must be nil")
	}
	m := randomMask(rand.New(rand.NewSource(1)), 7, 5, 50)
	if !Erode(m, 0).Equal(m) {
		t.Fatal("Erode n=0 must clone")
	}
	if !Dilate(m, -3).Equal(m) {
		t.Fatal("Dilate n<0 must clone")
	}
	if !Erode(m, MaxMorph+10).Equal(Erode(m, MaxMorph)) {
		t.Fatal("n must be clamped to MaxMorph")
	}
}

func TestThreshold(t *testing.T) {
	pix := []uint8{0, 10, 50, 100, 200, 255}
	m := Threshold(pix, 3, 2, 50, 200, false)
	want := []uint8{0, 0, 1, 1, 1, 0}
	for i := range want {
		if m.Pix[i] != want[i] {
			t.Fatalf("in-range threshold at %d = %d, want %d", i, m.Pix[i], want[i])
		}
	}
	m = Threshold(pix, 3, 2, 50, 200, true)
	want = []uint8{1, 1, 0, 0, 0, 1}
	for i := range want {
		if m.Pix[i] != want[i] {
			t.Fatalf("out-range threshold at %d = %d, want %d", i, m.Pix[i], want[i])
		}
	}
	// lo > hi: in-range empty, out-of-range full.
	m = Threshold(pix, 3, 2, 200, 50, false)
	if m.Count() != 0 {
		t.Fatal("lo>hi in-range must be empty")
	}
	m = Threshold(pix, 3, 2, 200, 50, true)
	if m.Count() != 6 {
		t.Fatal("lo>hi out-of-range must be full")
	}
	// Short slice: empty w*h mask.
	m = Threshold(pix[:2], 3, 2, 0, 255, false)
	if m.W != 3 || m.H != 2 || m.Count() != 0 {
		t.Fatal("short slice must yield empty w*h mask")
	}
}

func TestAndAndNot(t *testing.T) {
	a := NewMask(3, 2)
	a.Pix[0] = 1
	a.Pix[1] = 1
	a.Pix[4] = 1
	b := NewMask(3, 2)
	b.Pix[1] = 1
	b.Pix[4] = 1
	b.Pix[5] = 1
	and := And(a, b)
	if and.Pix[0] != 0 || and.Pix[1] != 1 || and.Pix[4] != 1 || and.Pix[5] != 0 {
		t.Fatalf("And mismatch: %v", and.Pix)
	}
	andNot := AndNot(a, b)
	if andNot.Pix[0] != 1 || andNot.Pix[1] != 0 || andNot.Pix[4] != 0 {
		t.Fatalf("AndNot mismatch: %v", andNot.Pix)
	}
	// nil b: And -> empty, AndNot -> clone.
	if And(a, nil).Count() != 0 {
		t.Fatal("And nil b must be empty")
	}
	if !AndNot(a, nil).Equal(a) {
		t.Fatal("AndNot nil b must be a clone")
	}
	// Mismatched sizes: And -> empty of a's size, AndNot -> clone.
	c := NewMask(2, 2)
	c.Pix[0] = 1
	em := And(a, c)
	if em.W != 3 || em.H != 2 || em.Count() != 0 {
		t.Fatal("And mismatched sizes must be empty with a's size")
	}
	if !AndNot(a, c).Equal(a) {
		t.Fatal("AndNot mismatched sizes must be a clone")
	}
}

func TestCloneNoAlias(t *testing.T) {
	m := NewMask(4, 4)
	m.Pix[0] = 1
	c := m.Clone()
	c.Pix[0] = 0
	c.Pix[1] = 1
	if m.Pix[0] != 1 || m.Pix[1] != 0 {
		t.Fatal("Clone must not alias")
	}
	if (*Mask)(nil).Clone() != nil {
		t.Fatal("nil Clone must be nil")
	}
}

func TestCountAtEqualValid(t *testing.T) {
	m := NewMask(3, 3)
	m.Pix[0] = 1
	m.Pix[4] = 2
	m.Pix[8] = 255
	if m.Count() != 3 {
		t.Fatalf("Count = %d, want 3", m.Count())
	}
	if (*Mask)(nil).Count() != 0 {
		t.Fatal("nil Count must be 0")
	}
	if !m.At(0, 0) || !m.At(1, 1) || !m.At(2, 2) {
		t.Fatal("At must report set pixels")
	}
	if m.At(-1, 0) || m.At(0, -1) || m.At(3, 0) || m.At(0, 3) {
		t.Fatal("At outside must be false")
	}
	if (*Mask)(nil).At(0, 0) {
		t.Fatal("nil At must be false")
	}
	o := m.Clone()
	if !m.Equal(o) {
		t.Fatal("clones must be equal")
	}
	o.Pix[1] = 1
	if m.Equal(o) {
		t.Fatal("changed masks must differ")
	}
	if !m.Equal(m) {
		t.Fatal("mask must equal itself")
	}
	if !(*Mask)(nil).Equal(nil) {
		t.Fatal("nil equals nil")
	}
	if (*Mask)(nil).Equal(m) || m.Equal(nil) {
		t.Fatal("nil differs from non-nil")
	}
	if !m.Valid() {
		t.Fatal("mask must be valid")
	}
	if (*Mask)(nil).Valid() {
		t.Fatal("nil must not be valid")
	}
	bad := &Mask{W: 3, H: 3, Pix: make([]uint8, 5)}
	if bad.Valid() {
		t.Fatal("wrong length must not be valid")
	}
	if (&Mask{W: 0, H: 3, Pix: nil}).Valid() {
		t.Fatal("zero width must not be valid")
	}
}
