package app

import (
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"

	"afvi/teach-imgui/internal/core"
	"afvi/teach-imgui/internal/golden"
	"afvi/teach-imgui/internal/inspect"
)

// The golden board is the inspection area of the teach program: a Metal mask drawn on the picture of
// a golden sample (the machine's "mask editor") plus an automatic SR mask, both on the design grid.
// Alignment moves a capture onto that grid, so the masks, not the design layers, decide what is
// inspected. See docs/GOLDEN.md.

const (
	goldenPageID  = "golden" // page id of the golden board
	goldenHistory = 6        // earlier masks that Undo can bring back
)

// goldenMorph selects the morphology of a Morph request.
type goldenMorph int

const (
	morphOpen goldenMorph = iota
	morphClose
	morphErode
	morphDilate
)

// goldenDerived is what the inspection and the page use of a board: immutable once built.
type goldenDerived struct {
	rev          int
	metalInspect *golden.Mask // the drawn Metal mask eroded by the padding
	sr           *golden.Mask // automatic SR mask
	material     []uint8      // inspection material map: 3 Metal, 1 SR, 0 elsewhere
	classes      []uint8      // class map of the whole golden picture (3 drawn Metal, 1 SR layer, 2 Open layer, 0 base): the alignment template is painted from it
	metalPx      int          // pixels of metalInspect
	bandPx       int          // pixels of the padding band (drawn minus inspected)
	srPx         int          // pixels of sr
}

// goldenSnap is one entry of the undo history.
type goldenSnap struct {
	metal  *golden.Mask
	lo, hi int
	out    bool
}

type goldenOpKind int

const (
	opInit   goldenOpKind = iota // load the saved board or derive the automatic one
	opAuto                       // derive the automatic board again
	opRange                      // mask = threshold of the picture
	opMorph                      // open / close / erode / dilate
	opAdd                        // add a shape
	opErase                      // erase a shape
	opArea                       // area threshold
	opUndo                       // bring the previous mask back
	opLoad                       // load the saved board
	opSave                       // save the board
	opDerive                     // padding or SR inset changed
	opSource                     // channel changed: new golden sample picture
	opRepack                     // repaint the picture
)

// goldenOp is one queued request. Values that the page edits live on the board (padding, inset,
// channel) and are read when the op starts.
type goldenOp struct {
	kind    goldenOpKind
	lo, hi  int
	out     bool
	morph   goldenMorph
	n       int
	shape   golden.Shape
	minGV   int
	minArea int
	fill    bool
}

// goldenInput is everything a golden job may use. All of it is captured on the UI thread before
// the job starts; masks and pictures are immutable.
type goldenInput struct {
	op      goldenOp
	scene   *Scene
	model   string
	light   core.LightSetting
	noise   core.SensorNoise
	dir     string // "" = do not touch the disk
	channel core.CameraChannel
	src     *core.GrayImage
	srcCh   core.CameraChannel
	geo     *goldenGeometry
	metal   *golden.Mask
	appLo   int // the range that belongs to metal
	appHi   int
	appOut  bool
	padding int
	srInset int
	derived *goldenDerived
	snap    *goldenSnap
	view    goldenView
	wantPic bool
}

// goldenResult is what a golden job returns to the UI thread.
type goldenResult struct {
	src      *core.GrayImage
	srcCh    core.CameraChannel
	geo      *goldenGeometry
	metal    *golden.Mask
	lo, hi   int
	out      bool
	channel  core.CameraChannel
	padding  int
	srInset  int
	derived  *goldenDerived
	pic      []byte
	changed  bool // the mask or its range changed
	restore  bool // an undo: no history entry
	loaded   bool // came from the saved board
	saved    bool
	msgKey   string
	msgText  string
	msgErr   bool
	fallback bool
}

// goldenBoard is the golden board of one loaded model. All fields belong to the UI thread; masks and
// pictures are never modified after they are published.
type goldenBoard struct {
	a     *App
	scene *Scene
	gen   int // bumped when a model is loaded: results of older jobs are dropped
	rev   int // bumped whenever the masks, padding or inset changed what is inspected

	channel  core.CameraChannel
	low      int // requested range (what the page shows)
	high     int
	out      bool
	appLo    int // range that belongs to metal
	appHi    int
	appOut   bool
	padding  int
	srInset  int
	metal    *golden.Mask
	src      *core.GrayImage
	srcCh    core.CameraChannel
	geo      *goldenGeometry
	derived  *goldenDerived
	view     goldenView
	history  []goldenSnap
	lastKind goldenOpKind // kind of the last applied mask change (range drags share one undo entry)
	hasLast  bool

	confirmed bool
	dirOver   string // directory override (self-test, tests)
	msgKey    string
	msgText   string
	msgErr    bool

	queue   []goldenOp
	running bool
}

// newGoldenBoard returns an empty board with the default settings.
func newGoldenBoard(a *App) *goldenBoard {
	return &goldenBoard{
		a:       a,
		channel: core.ChRed,
		low:     128, high: 255,
		appLo: 128, appHi: 255,
		padding: golden.DefaultPadding,
		srInset: golden.DefaultSRInset,
		view:    defaultGoldenView(),
	}
}

// goldenBoard returns the golden board of the app (nil in the studio).
func (a *App) goldenBoard() *goldenBoard { return a.golden }

// inspectRegionsFor returns the regions to inspect for scene: the material map of the golden board
// when it is ready, else the design layers.
func (a *App) inspectRegionsFor(scene *Scene) inspect.Regions {
	if g := a.goldenBoard(); g != nil && g.scene == scene && g.Ready() {
		return inspect.Regions{Width: scene.W, Height: scene.H, Material: g.derived.material}
	}
	return inspectRegions(scene)
}

// goldenChanged tells the detection that what is inspected changed.
func (a *App) goldenChanged() {
	if dp := defectPanelOf(a); dp != nil {
		dp.RequestCapture()
	}
}

// modelLoaded resets the board for the new scene and queues the load of the saved board or the
// automatic generation.
func (g *goldenBoard) modelLoaded(scene *Scene) {
	if g == nil {
		return
	}
	g.gen++
	g.rev++
	g.scene = scene
	g.metal, g.src, g.geo, g.derived = nil, nil, nil, nil
	g.history = nil
	g.hasLast = false
	g.queue = nil
	g.running = false
	g.confirmed = false
	g.msgKey, g.msgText, g.msgErr = "", "", false
	if scene == nil {
		return
	}
	g.enqueue(goldenOp{kind: opInit})
}

// Ready reports whether the inspection regions of the board are built and current.
func (g *goldenBoard) Ready() bool {
	return g != nil && g.metal != nil && g.derived != nil && g.derived.rev == g.rev && g.derived.material != nil
}

// Empty reports whether the built board inspects no pixel at all (an erased mask would otherwise let
// every unit pass).
func (g *goldenBoard) Empty() bool {
	return g.Ready() && g.derived.metalPx+g.derived.srPx == 0
}

// Rev counts the changes of what is inspected.
func (g *goldenBoard) Rev() int {
	if g == nil {
		return 0
	}
	return g.rev
}

// Busy reports a running or queued request.
func (g *goldenBoard) Busy() bool {
	return g != nil && (g.running || len(g.queue) > 0)
}

// Metal returns the drawn Metal mask (nil until built).
func (g *goldenBoard) Metal() *golden.Mask {
	if g == nil {
		return nil
	}
	return g.metal
}

// Derived returns the derived masks (nil until built).
func (g *goldenBoard) Derived() *goldenDerived {
	if g == nil {
		return nil
	}
	return g.derived
}

// Source returns the golden sample picture of the current channel (nil until rendered).
func (g *goldenBoard) Source() *core.GrayImage {
	if g == nil {
		return nil
	}
	return g.src
}

// Geometry returns the design layers moved onto the picture (nil until built).
func (g *goldenBoard) Geometry() *goldenGeometry {
	if g == nil {
		return nil
	}
	return g.geo
}

// Padding is the requested padding in pixels.
func (g *goldenBoard) Padding() int {
	if g == nil {
		return golden.DefaultPadding
	}
	return g.padding
}

// SRInset is the requested SR inset in pixels.
func (g *goldenBoard) SRInset() int {
	if g == nil {
		return golden.DefaultSRInset
	}
	return g.srInset
}

// Channel is the channel of the golden sample picture.
func (g *goldenBoard) Channel() core.CameraChannel {
	if g == nil {
		return core.ChRed
	}
	return g.channel
}

// Range is the requested grey range of the binary threshold.
func (g *goldenBoard) Range() (lo, hi int, out bool) {
	if g == nil {
		return 128, 255, false
	}
	return g.low, g.high, g.out
}

// Confirmed reports a board that was saved (or loaded) and not edited since.
func (g *goldenBoard) Confirmed() bool { return g != nil && g.confirmed }

// View returns the display settings.
func (g *goldenBoard) View() goldenView {
	if g == nil {
		return defaultGoldenView()
	}
	return g.view
}

// SetView changes the display settings; only the picture is repainted.
func (g *goldenBoard) SetView(v goldenView) {
	if g == nil || v == g.view {
		return
	}
	if v.Opacity < 0 {
		v.Opacity = 0
	} else if v.Opacity > 100 {
		v.Opacity = 100
	}
	g.view = v
	g.enqueue(goldenOp{kind: opRepack})
}

// CanUndo reports an earlier mask in the history.
func (g *goldenBoard) CanUndo() bool { return g != nil && len(g.history) > 0 }

// SetPadding sets the padding (pixels) between the drawn Metal mask and the inspected area.
func (g *goldenBoard) SetPadding(n int) {
	if g == nil {
		return
	}
	n = golden.ClampPad(n)
	if n == g.padding {
		return
	}
	g.padding = n
	g.enqueue(goldenOp{kind: opDerive})
}

// SetSRInset sets the inset (pixels) of the automatic SR mask.
func (g *goldenBoard) SetSRInset(n int) {
	if g == nil {
		return
	}
	n = golden.ClampPad(n)
	if n == g.srInset {
		return
	}
	g.srInset = n
	g.enqueue(goldenOp{kind: opDerive})
}

// SetRange sets the grey range of the binary threshold: the Metal mask becomes the threshold of the
// golden sample picture.
func (g *goldenBoard) SetRange(lo, hi int, out bool) {
	if g == nil {
		return
	}
	lo, hi = clampInt(lo, 0, 255), clampInt(hi, 0, 255)
	if lo > hi {
		hi = lo
	}
	g.low, g.high, g.out = lo, hi, out
	g.enqueue(goldenOp{kind: opRange, lo: lo, hi: hi, out: out})
}

// SetChannel renders the golden sample picture of another channel (the mask stays).
func (g *goldenBoard) SetChannel(ch core.CameraChannel) {
	if g == nil || ch == g.channel {
		return
	}
	g.channel = ch
	g.enqueue(goldenOp{kind: opSource})
}

// Morph applies Open / Close / Erode / Dilate with radius n to the Metal mask.
func (g *goldenBoard) Morph(kind goldenMorph, n int) {
	if g == nil || n <= 0 {
		return
	}
	g.enqueue(goldenOp{kind: opMorph, morph: kind, n: clampInt(n, 1, golden.MaxMorph)})
}

// AddShape adds a shape to the Metal mask; with minGV > 0 only pixels at least that bright.
func (g *goldenBoard) AddShape(sh golden.Shape, minGV int) {
	if g == nil {
		return
	}
	g.enqueue(goldenOp{kind: opAdd, shape: sh, minGV: clampInt(minGV, 0, 255)})
}

// EraseShape removes a shape from the Metal mask.
func (g *goldenBoard) EraseShape(sh golden.Shape) {
	if g == nil {
		return
	}
	g.enqueue(goldenOp{kind: opErase, shape: sh})
}

// AreaFilter removes Metal blobs smaller than minArea pixels and, with fill, fills holes smaller
// than minArea pixels.
func (g *goldenBoard) AreaFilter(minArea int, fill bool) {
	if g == nil || minArea <= 1 {
		return
	}
	g.enqueue(goldenOp{kind: opArea, minArea: minArea, fill: fill})
}

// Auto derives the automatic board again from the design layers.
func (g *goldenBoard) Auto() {
	if g == nil {
		return
	}
	g.enqueue(goldenOp{kind: opAuto})
}

// Undo brings the previous mask back (up to goldenHistory steps).
func (g *goldenBoard) Undo() bool {
	if g == nil || !g.CanUndo() {
		return false
	}
	g.enqueue(goldenOp{kind: opUndo})
	return true
}

// Dir is the folder the board is saved in.
func (g *goldenBoard) Dir() string {
	if g == nil {
		return ""
	}
	if g.dirOver != "" {
		return g.dirOver
	}
	return filepath.Join(defaultCaptureDir(g.scene), "golden")
}

// Save writes the board (queued; the page shows the result through Message).
func (g *goldenBoard) Save() error {
	if g == nil || g.scene == nil {
		return errors.New("golden: no model")
	}
	g.enqueue(goldenOp{kind: opSave})
	return nil
}

// Load reads the saved board back (queued).
func (g *goldenBoard) Load() error {
	if g == nil || g.scene == nil {
		return errors.New("golden: no model")
	}
	g.enqueue(goldenOp{kind: opLoad})
	return nil
}

// Activate repaints the picture because the page is shown.
func (g *goldenBoard) Activate() {
	if g == nil || g.scene == nil {
		return
	}
	g.enqueue(goldenOp{kind: opRepack})
}

// Message returns the status line of the page: an i18n key (or "" with a text) and whether it is an error.
func (g *goldenBoard) Message() (key string, text string, isErr bool) {
	if g == nil {
		return "", "", false
	}
	return g.msgKey, g.msgText, g.msgErr
}

// PixelInfo describes the picture pixel (x, y).
func (g *goldenBoard) PixelInfo(x, y int) (gv int, inMetal, inInspect, inSR, ok bool) {
	if g == nil || g.src == nil || x < 0 || y < 0 || x >= g.src.Width || y >= g.src.Height {
		return 0, false, false, false, false
	}
	gv = int(g.src.Data[y*g.src.Width+x])
	inMetal = g.metal.At(x, y)
	if d := g.derived; d != nil {
		inInspect = d.metalInspect.At(x, y)
		inSR = d.sr.At(x, y)
	}
	return gv, inMetal, inInspect, inSR, true
}

// pageActive reports whether the golden page is the visible page.
func (g *goldenBoard) pageActive() bool {
	p := g.a.ActivePanel()
	return p != nil && p.ID() == goldenPageID
}

// enqueue adds a request (coalescing consecutive repeats of the cheap ones) and starts the pump.
func (g *goldenBoard) enqueue(op goldenOp) {
	if g.scene == nil {
		return
	}
	if n := len(g.queue); n > 0 {
		last := &g.queue[n-1]
		switch op.kind {
		case opRange, opDerive, opSource, opRepack:
			if last.kind == op.kind {
				*last = op
				g.pump()
				return
			}
		}
	}
	g.queue = append(g.queue, op)
	g.pump()
}

// pump starts the next queued request unless one is running. One job at a time (key "golden"), so
// a newer submission never replaces a pending one.
func (g *goldenBoard) pump() {
	for !g.running && len(g.queue) > 0 {
		op := g.queue[0]
		g.queue = g.queue[1:]
		if g.start(op) {
			return
		}
	}
}

// start submits the job of op; false when the op turned out to be a no-op.
func (g *goldenBoard) start(op goldenOp) bool {
	a := g.a
	scene := g.scene
	in := goldenInput{
		op:      op,
		scene:   scene,
		model:   scene.Info.ID,
		light:   a.Light.Clone(),
		channel: g.channel,
		src:     g.src,
		srcCh:   g.srcCh,
		geo:     g.geo,
		metal:   g.metal,
		appLo:   g.appLo,
		appHi:   g.appHi,
		appOut:  g.appOut,
		padding: g.padding,
		srInset: g.srInset,
		derived: g.derived,
		view:    g.view,
		wantPic: g.pageActive(),
	}
	if dp := defectPanelOf(a); dp != nil {
		in.noise = dp.noise
	}
	switch op.kind {
	case opInit:
		if a.selftest == nil {
			in.dir = g.Dir()
		}
	case opLoad, opSave:
		in.dir = g.Dir()
	case opUndo:
		if len(g.history) == 0 {
			return false
		}
		s := g.history[len(g.history)-1]
		g.history = g.history[:len(g.history)-1]
		in.snap = &s
	}
	gen := g.gen
	g.running = true
	a.Jobs.Run("golden", func() func() {
		var res *goldenResult
		var err error
		func() {
			defer func() {
				if r := recover(); r != nil {
					err = fmt.Errorf("golden: %v", r)
				}
			}()
			res, err = goldenRun(in)
		}()
		return func() { g.apply(gen, in, res, err) }
	})
	return true
}

// apply installs the result of a job on the UI thread.
func (g *goldenBoard) apply(gen int, in goldenInput, r *goldenResult, err error) {
	if g.gen != gen || g.a.Scene != in.scene {
		if r != nil && r.pic != nil {
			PutRGBA(r.pic)
		}
		if g.gen == gen {
			// The scene went away without a new model (machine switch): nothing queued is valid any more,
			// and the board must not stay busy for ever.
			g.running = false
			g.queue = nil
		}
		return
	}
	g.running = false
	defer g.pump()
	if err != nil || r == nil {
		if err == nil {
			err = errors.New("golden: no result")
		}
		g.msgKey, g.msgText, g.msgErr = "golden.msg.error", err.Error(), true
		g.a.SetHint(T("common.error", "msg", err.Error()))
		return
	}
	if in.op.kind != opRepack {
		// A repaint keeps the status line; every other request replaces it.
		g.msgKey, g.msgText, g.msgErr = r.msgKey, r.msgText, r.msgErr
	}
	g.src, g.srcCh, g.geo = r.src, r.srcCh, r.geo
	if r.changed {
		if !r.restore && in.metal != nil && !(in.op.kind == opRange && g.hasLast && g.lastKind == opRange) {
			g.history = append(g.history, goldenSnap{metal: in.metal, lo: in.appLo, hi: in.appHi, out: in.appOut})
			if len(g.history) > goldenHistory {
				g.history = g.history[len(g.history)-goldenHistory:]
			}
		}
		g.metal = r.metal
		g.appLo, g.appHi, g.appOut = r.lo, r.hi, r.out
		if r.loaded || r.restore || in.op.kind == opAuto || in.op.kind == opInit {
			g.low, g.high, g.out = r.lo, r.hi, r.out
		}
		g.lastKind, g.hasLast = in.op.kind, true
		g.confirmed = r.loaded
	} else if r.metal != nil && g.metal == nil {
		g.metal = r.metal
	}
	if r.loaded {
		g.padding, g.srInset = r.padding, r.srInset
		g.channel = r.channel
	}
	if r.saved {
		g.confirmed = true
	}
	derivedChanged := r.derived != nil
	if derivedChanged {
		g.rev++
		r.derived.rev = g.rev
		g.derived = r.derived
	}
	if r.pic != nil {
		if g.pageActive() {
			g.a.Viewer.SetImage(r.src.Width, r.src.Height, r.pic)
		} else {
			PutRGBA(r.pic)
		}
	} else if !in.wantPic && g.pageActive() && r.src != nil && in.op.kind != opSave {
		g.queue = append(g.queue, goldenOp{kind: opRepack})
	}
	if derivedChanged {
		g.a.goldenChanged()
	}
}

// goldenRun is the job: it uses only the captured input. It renders the golden sample picture and the
// geometry when they are missing, applies the op, derives the inspection masks and paints the picture.
func goldenRun(in goldenInput) (*goldenResult, error) {
	scene := in.scene
	if scene == nil || scene.W <= 0 || scene.H <= 0 {
		return nil, errors.New("golden: no model")
	}
	r := &goldenResult{
		src: in.src, srcCh: in.srcCh, geo: in.geo, metal: in.metal,
		lo: in.appLo, hi: in.appHi, out: in.appOut,
		channel: in.channel, padding: in.padding, srInset: in.srInset,
	}
	op := in.op

	// A saved board decides the channel, so it is read first.
	var saved *golden.Board
	if op.kind == opInit || op.kind == opLoad {
		if in.dir != "" {
			b, err := golden.Load(in.dir)
			switch {
			case err == nil && b.Model == in.model && b.Metal.W == scene.W && b.Metal.H == scene.H:
				saved = b
			case err == nil:
				r.msgKey = "golden.msg.mismatch"
				r.msgErr = op.kind == opLoad
			case errors.Is(err, fs.ErrNotExist):
				if op.kind == opLoad {
					return nil, fmt.Errorf("golden: nothing saved in %s", in.dir)
				}
			default:
				if op.kind == opLoad {
					return nil, err
				}
				r.msgKey, r.msgText, r.msgErr = "golden.msg.error", err.Error(), true
			}
		} else if op.kind == opLoad {
			return nil, errors.New("golden: no folder")
		}
		if op.kind == opLoad && saved == nil {
			return nil, errors.New("golden: the saved board does not match this model")
		}
		if saved != nil {
			ch := core.CameraChannel(saved.Settings.Channel)
			if ch < core.ChRed || ch > core.ChBlue {
				ch = core.ChRed
			}
			r.channel = ch
		}
	}

	// Golden sample picture of the channel.
	if r.src == nil || r.srcCh != r.channel {
		r.src = goldenSourceImage(scene, in.light, in.noise, r.channel)
		r.srcCh = r.channel
	}
	if r.src == nil || r.src.Width != scene.W || r.src.Height != scene.H {
		return nil, errors.New("golden: the golden sample picture has the wrong size")
	}
	if r.geo == nil {
		geo, err := goldenBuildGeometry(scene, r.src)
		if err != nil {
			return nil, err
		}
		r.geo = geo
	}

	setAuto := func() {
		auto := goldenAutoMetal(r.src, r.geo)
		r.metal = auto.Metal
		r.lo, r.hi, r.out = auto.Low, auto.High, auto.OutRange
		r.changed = true
		r.fallback = auto.Fallback
		if auto.Fallback {
			r.msgKey = "golden.msg.fallback"
		} else {
			r.msgKey = "golden.msg.auto"
		}
	}
	if r.metal == nil && saved == nil {
		setAuto()
	}

	switch op.kind {
	case opInit, opLoad:
		if saved != nil {
			r.metal = saved.Metal
			r.lo, r.hi, r.out = saved.Settings.Low, saved.Settings.High, saved.Settings.OutRange
			r.padding, r.srInset = saved.Settings.Padding, saved.Settings.SRInset
			r.changed, r.loaded = true, true
			r.msgKey = "golden.msg.loaded"
			r.msgErr = false
		}
	case opAuto:
		setAuto()
	case opRange:
		r.metal = golden.Threshold(r.src.Data, scene.W, scene.H, uint8(op.lo), uint8(op.hi), op.out)
		r.lo, r.hi, r.out = op.lo, op.hi, op.out
		r.changed = true
	case opMorph:
		switch op.morph {
		case morphOpen:
			r.metal = golden.Open(r.metal, op.n)
		case morphClose:
			r.metal = golden.Close(r.metal, op.n)
		case morphErode:
			r.metal = golden.Erode(r.metal, op.n)
		default:
			r.metal = golden.Dilate(r.metal, op.n)
		}
		r.changed = true
	case opAdd:
		m := r.metal.Clone()
		golden.Add(m, op.shape, r.src.Data, op.minGV)
		r.metal = m
		r.changed = true
	case opErase:
		m := r.metal.Clone()
		golden.Erase(m, op.shape)
		r.metal = m
		r.changed = true
	case opArea:
		r.metal = golden.AreaFilter(r.metal, op.minArea, op.fill)
		r.changed = true
	case opUndo:
		if in.snap == nil || in.snap.metal == nil {
			return nil, errors.New("golden: nothing to undo")
		}
		r.metal = in.snap.metal
		r.lo, r.hi, r.out = in.snap.lo, in.snap.hi, in.snap.out
		r.changed, r.restore = true, true
	case opSave:
		b := &golden.Board{
			Model: in.model,
			Settings: golden.Settings{
				Channel: int(r.channel), Low: r.lo, High: r.hi, OutRange: r.out,
				Padding: in.padding, SRInset: in.srInset,
			},
			Metal: r.metal,
		}
		if err := golden.Save(in.dir, b); err != nil {
			return nil, err
		}
		r.saved = true
		r.msgKey = "golden.msg.saved"
		return r, nil
	}

	// Derive the inspection masks when the mask, the padding or the inset changed.
	if r.changed || op.kind == opDerive || in.derived == nil {
		d, err := goldenDerive(r.metal, r.geo, r.padding, r.srInset)
		if err != nil {
			return nil, err
		}
		r.derived = d
	}
	if in.wantPic {
		d := r.derived
		if d == nil {
			d = in.derived
		}
		var inspectMask, sr *golden.Mask
		if d != nil {
			inspectMask, sr = d.metalInspect, d.sr
		}
		r.pic = PackGolden(r.src, r.metal, inspectMask, sr, in.view)
	}
	return r, nil
}

// goldenDerive builds the inspection masks and the material map of a drawn Metal mask.
func goldenDerive(metal *golden.Mask, geo *goldenGeometry, padding, srInset int) (*goldenDerived, error) {
	if !metal.Valid() {
		return nil, errors.New("golden: no Metal mask")
	}
	d := &goldenDerived{}
	d.metalInspect = golden.InspectMetal(metal, padding)
	if sr := geo.srMask(); sr != nil {
		d.sr = golden.AutoSR(sr, metal, srInset)
	}
	if d.sr == nil {
		d.sr = golden.NewMask(metal.W, metal.H)
	}
	d.material = golden.Material(d.metalInspect, d.sr)
	if d.material == nil {
		return nil, errors.New("golden: cannot build the material map")
	}
	d.classes = goldenClasses(metal, geo)
	d.metalPx = d.metalInspect.Count()
	d.srPx = d.sr.Count()
	d.bandPx = metal.Count() - d.metalPx
	return d, nil
}

// goldenClasses paints the class map of the golden picture: the drawn Metal mask first, then the SR
// layer, then the Open layer of the design moved onto the picture; everything else is base. The
// alignment template is painted from it, so the aligned capture sits on the geometry of the mask.
func goldenClasses(metal *golden.Mask, geo *goldenGeometry) []uint8 {
	out := make([]uint8, len(metal.Pix))
	sr := geo.srMask()
	for i := range out {
		switch {
		case metal.Pix[i] != 0:
			out[i] = core.MatMetal
		case sr != nil && sr.Pix[i] != 0:
			out[i] = core.MatSR
		case geo != nil && len(geo.reg) == len(out) && geo.reg[i] == core.MatOpen:
			out[i] = core.MatOpen
		}
	}
	return out
}

// clampInt limits v to lo..hi.
func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
