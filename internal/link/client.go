package link

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"sync"
	"time"

	"afvi/teach-imgui/internal/snap"
)

// Client is the teaching side of one connection: it follows the scene the studio
// publishes and pulls rendered frames of it. Use Done to notice that the connection ended
// and Dial again; reconnecting is the caller's policy.
type Client struct {
	conn net.Conn
	peer Hello

	wmu sync.Mutex // serialises writes on conn

	mu       sync.Mutex
	scene    Scene
	hasScene bool
	pending  map[uint32]chan reply
	next     uint32
	err      error

	updates chan struct{}
	done    chan struct{}
	once    sync.Once
}

type reply struct {
	msg message
}

// Dial connects to a studio at addr and performs the handshake. app names the program in
// the hello message. timeout bounds both the connection attempt and the handshake.
func Dial(addr, app string, timeout time.Duration) (*Client, error) {
	d := net.Dialer{Timeout: timeout, KeepAlive: 15 * time.Second}
	conn, err := d.DialContext(context.Background(), "tcp", addr)
	if err != nil {
		return nil, err
	}
	_ = conn.SetDeadline(time.Now().Add(timeout))
	r := bufio.NewReaderSize(conn, 1<<16)
	peer, err := handshake(&bufferedConn{Reader: r, Writer: conn}, Hello{Role: RoleTeach, App: app, PID: pid()})
	if err != nil {
		_ = conn.Close()
		return nil, err
	}
	_ = conn.SetDeadline(time.Time{})
	c := &Client{
		conn:    conn,
		peer:    peer,
		pending: map[uint32]chan reply{},
		updates: make(chan struct{}, 1),
		done:    make(chan struct{}),
	}
	go c.readLoop(r)
	return c, nil
}

// Peer is the studio's hello message.
func (c *Client) Peer() Hello { return c.peer }

// Scene returns the newest scene the studio pushed; ok is false before the first push.
func (c *Client) Scene() (Scene, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.scene, c.hasScene
}

// Updates is signalled (without blocking the reader, signals coalesce) whenever a new
// scene arrived.
func (c *Client) Updates() <-chan struct{} { return c.updates }

// Done is closed when the connection has ended.
func (c *Client) Done() <-chan struct{} { return c.done }

// Err is why the connection ended (nil while it is alive).
func (c *Client) Err() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.err
}

// Close ends the connection.
func (c *Client) Close() error {
	c.fail(ErrClosed)
	return nil
}

func (c *Client) fail(err error) {
	c.once.Do(func() {
		c.mu.Lock()
		c.err = err
		pending := c.pending
		c.pending = map[uint32]chan reply{}
		c.mu.Unlock()
		close(c.done)
		_ = c.conn.Close()
		for _, ch := range pending {
			close(ch)
		}
	})
}

func (c *Client) readLoop(r *bufio.Reader) {
	for {
		m, err := readMessage(r)
		if err != nil {
			c.fail(err)
			return
		}
		switch m.typ {
		case typeScene:
			var sc Scene
			if err := json.Unmarshal(m.json, &sc); err != nil {
				c.fail(fmt.Errorf("link: bad scene: %w", err))
				return
			}
			sc.Session = c.peer.Session
			c.mu.Lock()
			c.scene, c.hasScene = sc, true
			c.mu.Unlock()
			select {
			case c.updates <- struct{}{}:
			default:
			}
		case typeFrame, typeError:
			c.mu.Lock()
			ch := c.pending[m.id]
			delete(c.pending, m.id)
			c.mu.Unlock()
			if ch != nil {
				ch <- reply{msg: m}
			}
		}
	}
}

// Frame asks the studio for one rendered image of the scene. rev is the revision the
// caller is working on (0 = the current one): when the studio has moved on, the error
// matches ErrStale and the caller should wait for the next scene. It returns the frame and
// the revision it belongs to.
func (c *Client) Frame(ctx context.Context, rev int64, channel, kind string) (snap.Frame, int64, error) {
	ch := make(chan reply, 1)
	c.mu.Lock()
	if c.err != nil {
		err := c.err
		c.mu.Unlock()
		return snap.Frame{}, 0, err
	}
	c.next++
	if c.next == 0 {
		c.next = 1
	}
	id := c.next
	c.pending[id] = ch
	c.mu.Unlock()

	c.wmu.Lock()
	err := writeJSON(c.conn, typeGetFrame, id, frameRequest{Rev: rev, Channel: channel, Kind: kind}, nil)
	c.wmu.Unlock()
	if err != nil {
		c.fail(err)
		return snap.Frame{}, 0, err
	}
	select {
	case rep, ok := <-ch:
		if !ok {
			return snap.Frame{}, 0, c.Err()
		}
		return decodeFrame(rep.msg)
	case <-ctx.Done():
		c.mu.Lock()
		delete(c.pending, id)
		c.mu.Unlock()
		return snap.Frame{}, 0, ctx.Err()
	case <-c.done:
		return snap.Frame{}, 0, c.Err()
	}
}

func decodeFrame(m message) (snap.Frame, int64, error) {
	if m.typ == typeError {
		var e errorBody
		if err := json.Unmarshal(m.json, &e); err != nil {
			return snap.Frame{}, 0, fmt.Errorf("link: bad error message: %w", err)
		}
		return snap.Frame{}, 0, &RemoteError{Code: e.Code, Message: e.Message}
	}
	var h frameHeader
	if err := json.Unmarshal(m.json, &h); err != nil {
		return snap.Frame{}, 0, fmt.Errorf("link: bad frame header: %w", err)
	}
	if h.Enc != "raw" {
		return snap.Frame{}, 0, fmt.Errorf("link: unsupported frame encoding %q", h.Enc)
	}
	if h.Width <= 0 || h.Height <= 0 || len(m.body) != h.Width*h.Height {
		return snap.Frame{}, 0, fmt.Errorf("link: frame %dx%d has %d bytes", h.Width, h.Height, len(m.body))
	}
	return snap.Frame{Width: h.Width, Height: h.Height, Pix: m.body}, h.Rev, nil
}
