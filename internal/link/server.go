package link

import (
	"bufio"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"sync"
	"time"

	"afvi/teach-imgui/internal/snap"
)

// RenderFunc renders one frame of the scene it was published with. channel is "red",
// "green" or "blue" and kind one of KindTest, KindGood, KindGolden. It runs on a server
// goroutine, so it must not touch state that another goroutine mutates (capture the
// scenario by value when publishing).
type RenderFunc func(channel, kind string) (snap.Frame, error)

// Server is the studio side: it holds the current scene and serves it to every connected
// client. Clients get the scene right after the handshake and again after every Publish.
type Server struct {
	hello Hello
	ln    net.Listener
	sem   chan struct{} // bounds concurrent renders

	mu     sync.Mutex
	rev    int64
	scene  []byte // marshalled Scene of the current revision
	render RenderFunc
	cache  []cached // frames of the current revision, oldest first
	conns  map[*serverConn]struct{}
	closed bool

	wg sync.WaitGroup
}

type cached struct {
	rev           int64
	channel, kind string
	frame         snap.Frame
}

const (
	maxCached        = 4
	maxRenders       = 2
	handshakeTimeout = 5 * time.Second
)

// Listen starts serving on addr (for example DefaultAddr, or "127.0.0.1:0" for a free
// port). app names the program in the hello message.
func Listen(addr, app string) (*Server, error) {
	lc := net.ListenConfig{KeepAlive: 15 * time.Second}
	ln, err := lc.Listen(context.Background(), "tcp", addr)
	if err != nil {
		return nil, err
	}
	var id [8]byte
	_, _ = rand.Read(id[:])
	s := &Server{
		hello: Hello{Protocol: Protocol, Role: RoleStudio, App: app, PID: os.Getpid(), Session: hex.EncodeToString(id[:])},
		ln:    ln,
		sem:   make(chan struct{}, maxRenders),
		conns: map[*serverConn]struct{}{},
	}
	s.wg.Add(1)
	go s.acceptLoop()
	return s, nil
}

// Addr is the address the server listens on.
func (s *Server) Addr() net.Addr { return s.ln.Addr() }

// Session is the identifier of this server instance.
func (s *Server) Session() string { return s.hello.Session }

// Publish makes meta the current scene and render its renderer, replaces the cached
// frames, notifies every client and returns the new revision. The caller must not modify
// meta (or what it references) afterwards.
func (s *Server) Publish(meta snap.Meta, render RenderFunc) int64 {
	s.mu.Lock()
	s.rev++
	rev := s.rev
	js, err := json.Marshal(Scene{Rev: rev, Meta: meta})
	if err != nil {
		// Meta is plain data; a marshalling failure is a programming error, keep the old scene.
		s.rev--
		s.mu.Unlock()
		return s.rev
	}
	s.scene = js
	s.render = render
	s.cache = nil
	conns := make([]*serverConn, 0, len(s.conns))
	for c := range s.conns {
		conns = append(conns, c)
	}
	s.mu.Unlock()
	for _, c := range conns {
		c.notify()
	}
	return rev
}

// Rev is the current revision (0 before the first Publish).
func (s *Server) Rev() int64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.rev
}

// Peers lists the hello messages of the connected clients.
func (s *Server) Peers() []Hello {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]Hello, 0, len(s.conns))
	for c := range s.conns {
		out = append(out, c.peer)
	}
	return out
}

// Close stops listening, disconnects every client and waits for the goroutines.
func (s *Server) Close() error {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return nil
	}
	s.closed = true
	conns := make([]*serverConn, 0, len(s.conns))
	for c := range s.conns {
		conns = append(conns, c)
	}
	s.mu.Unlock()
	err := s.ln.Close()
	for _, c := range conns {
		c.close()
	}
	s.wg.Wait()
	return err
}

func (s *Server) acceptLoop() {
	defer s.wg.Done()
	for {
		conn, err := s.ln.Accept()
		if err != nil {
			return
		}
		sc := &serverConn{s: s, c: conn, push: make(chan struct{}, 1), done: make(chan struct{})}
		s.wg.Add(1)
		go func() {
			defer s.wg.Done()
			sc.serve()
		}()
	}
}

type serverConn struct {
	s    *Server
	c    net.Conn
	peer Hello
	wmu  sync.Mutex // serialises writes on c
	push chan struct{}
	done chan struct{}
	once sync.Once
}

func (sc *serverConn) close() {
	sc.once.Do(func() {
		close(sc.done)
		_ = sc.c.Close()
	})
}

// notify asks the pusher goroutine to send the current scene (coalescing: a burst of
// publishes results in one push of the newest scene).
func (sc *serverConn) notify() {
	select {
	case sc.push <- struct{}{}:
	default:
	}
}

func (sc *serverConn) write(typ msgType, id uint32, js, body []byte) error {
	sc.wmu.Lock()
	defer sc.wmu.Unlock()
	return writeRaw(sc.c, typ, id, js, body)
}

func (sc *serverConn) writeError(id uint32, code, msg string) error {
	js, _ := json.Marshal(errorBody{Code: code, Message: msg})
	return sc.write(typeError, id, js, nil)
}

func (sc *serverConn) serve() {
	defer sc.close()
	s := sc.s
	_ = sc.c.SetDeadline(time.Now().Add(handshakeTimeout))
	r := bufio.NewReaderSize(sc.c, 1<<16)
	// The handshake reads through r so no byte the client sends right after its hello is lost.
	peer, err := handshake(&bufferedConn{Reader: r, Writer: sc.c}, s.hello)
	if err != nil {
		_ = sc.writeError(0, CodeBadRequest, err.Error())
		return
	}
	_ = sc.c.SetDeadline(time.Time{})
	sc.peer = peer

	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return
	}
	s.conns[sc] = struct{}{}
	s.mu.Unlock()
	defer func() {
		s.mu.Lock()
		delete(s.conns, sc)
		s.mu.Unlock()
	}()

	pusherDone := make(chan struct{})
	go func() {
		defer close(pusherDone)
		sc.pushLoop()
	}()
	sc.notify()
	for {
		m, err := readMessage(r)
		if err != nil {
			break
		}
		if m.typ != typeGetFrame {
			continue // hello again, pushes from a confused peer, future message types
		}
		var req frameRequest
		if err := json.Unmarshal(m.json, &req); err != nil {
			if sc.writeError(m.id, CodeBadRequest, "bad frame request: "+err.Error()) != nil {
				break
			}
			continue
		}
		s.wg.Add(1)
		go func(id uint32) {
			defer s.wg.Done()
			sc.handleFrame(id, req)
		}(m.id)
	}
	sc.close()
	<-pusherDone
}

// bufferedConn reads through a bufio.Reader and writes straight to the connection.
type bufferedConn struct {
	*bufio.Reader
	Writer interface{ Write([]byte) (int, error) }
}

func (b *bufferedConn) Write(p []byte) (int, error) { return b.Writer.Write(p) }

// pushLoop sends the current scene whenever it is newer than the last one sent.
func (sc *serverConn) pushLoop() {
	var sent int64
	for {
		select {
		case <-sc.push:
		case <-sc.done:
			return
		}
		sc.s.mu.Lock()
		rev, js := sc.s.rev, sc.s.scene
		sc.s.mu.Unlock()
		if rev == 0 || rev == sent {
			continue
		}
		if err := sc.write(typeScene, 0, js, nil); err != nil {
			sc.close()
			return
		}
		sent = rev
	}
}

func (sc *serverConn) handleFrame(id uint32, req frameRequest) {
	s := sc.s
	if _, err := snap.ParseChannel(req.Channel); err != nil {
		_ = sc.writeError(id, CodeBadRequest, "unknown channel "+req.Channel)
		return
	}
	switch req.Kind {
	case KindTest, KindGood, KindGolden:
	default:
		_ = sc.writeError(id, CodeBadRequest, "unknown kind "+req.Kind)
		return
	}
	s.mu.Lock()
	rev, render := s.rev, s.render
	if rev == 0 || render == nil {
		s.mu.Unlock()
		_ = sc.writeError(id, CodeNoScene, "no scene has been published yet")
		return
	}
	if req.Rev != 0 && req.Rev != rev {
		s.mu.Unlock()
		_ = sc.writeError(id, CodeStale, fmt.Sprintf("revision %d requested, current is %d", req.Rev, rev))
		return
	}
	for _, c := range s.cache {
		if c.rev == rev && c.channel == req.Channel && c.kind == req.Kind {
			s.mu.Unlock()
			sc.sendFrame(id, rev, req, c.frame)
			return
		}
	}
	s.mu.Unlock()
	select {
	case s.sem <- struct{}{}:
	case <-sc.done:
		return
	}
	frame, err := render(req.Channel, req.Kind)
	<-s.sem
	if err != nil {
		_ = sc.writeError(id, CodeRender, err.Error())
		return
	}
	if frame.Width <= 0 || frame.Height <= 0 || len(frame.Pix) != frame.Width*frame.Height {
		_ = sc.writeError(id, CodeRender, "renderer returned an inconsistent frame")
		return
	}
	s.mu.Lock()
	if s.rev != rev {
		cur := s.rev
		s.mu.Unlock()
		_ = sc.writeError(id, CodeStale, fmt.Sprintf("revision %d was replaced by %d while rendering", rev, cur))
		return
	}
	s.cache = append(s.cache, cached{rev: rev, channel: req.Channel, kind: req.Kind, frame: frame})
	if len(s.cache) > maxCached {
		s.cache = s.cache[len(s.cache)-maxCached:]
	}
	s.mu.Unlock()
	sc.sendFrame(id, rev, req, frame)
}

func (sc *serverConn) sendFrame(id uint32, rev int64, req frameRequest, f snap.Frame) {
	js, _ := json.Marshal(frameHeader{Rev: rev, Channel: req.Channel, Kind: req.Kind, Width: f.Width, Height: f.Height, Enc: "raw"})
	if err := sc.write(typeFrame, id, js, f.Pix); err != nil {
		sc.close()
	}
}
