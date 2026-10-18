package link

import (
	"bytes"
	"context"
	"errors"
	"net"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"afvi/teach-imgui/internal/snap"
)

const testW, testH = 64, 48

func testMeta(id string) snap.Meta {
	return snap.Meta{
		Format:  snap.Format,
		Version: snap.Version,
		Model:   snap.ModelRef{ID: id},
		Width:   testW,
		Height:  testH,
		Recipe:  snap.Recipe{Defects: []snap.Defect{}, SceneStrokes: []snap.SceneStroke{}},
		Images:  []snap.Image{},
	}
}

// testRender returns a frame whose pixels depend on rev, channel and kind, so a test can
// tell which request a frame answers.
func testRender(rev int, count *atomic.Int32) RenderFunc {
	return func(channel, kind string) (snap.Frame, error) {
		if count != nil {
			count.Add(1)
		}
		pix := make([]byte, testW*testH)
		seed := byte(rev*31) + channel[0] + kind[0]
		for i := range pix {
			pix[i] = seed + byte(i)
		}
		return snap.Frame{Width: testW, Height: testH, Pix: pix}, nil
	}
}

func startPair(t *testing.T) (*Server, *Client) {
	t.Helper()
	s, err := Listen("127.0.0.1:0", "test-studio")
	if err != nil {
		t.Fatalf("Listen: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	c, err := Dial(s.Addr().String(), "test-teach", 2*time.Second)
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	t.Cleanup(func() { c.Close() })
	return s, c
}

func waitScene(t *testing.T, c *Client, rev int64) Scene {
	t.Helper()
	deadline := time.After(3 * time.Second)
	for {
		if sc, ok := c.Scene(); ok && sc.Rev >= rev {
			return sc
		}
		select {
		case <-c.Updates():
		case <-deadline:
			t.Fatalf("scene rev %d did not arrive", rev)
		case <-c.Done():
			t.Fatalf("connection ended: %v", c.Err())
		}
	}
}

func TestSceneAndFrameRoundTrip(t *testing.T) {
	s, c := startPair(t)
	if c.Peer().Role != RoleStudio || c.Peer().App != "test-studio" || c.Peer().Session != s.Session() {
		t.Fatalf("peer = %+v", c.Peer())
	}
	if _, ok := c.Scene(); ok {
		t.Fatal("scene before the first Publish")
	}
	rev := s.Publish(testMeta("FM1/abc"), testRender(1, nil))
	if rev != 1 {
		t.Fatalf("first revision = %d, want 1", rev)
	}
	sc := waitScene(t, c, 1)
	if sc.Meta.Model.ID != "FM1/abc" || sc.Meta.Width != testW || sc.Session != s.Session() {
		t.Fatalf("scene = %+v", sc)
	}
	for _, kind := range []string{KindTest, KindGood, KindGolden} {
		f, got, err := c.Frame(context.Background(), 1, "green", kind)
		if err != nil {
			t.Fatalf("Frame %s: %v", kind, err)
		}
		want, _ := testRender(1, nil)("green", kind)
		if got != 1 || f.Width != testW || f.Height != testH || !bytes.Equal(f.Pix, want.Pix) {
			t.Fatalf("Frame %s: rev %d size %dx%d, pixels differ", kind, got, f.Width, f.Height)
		}
	}
	if peers := s.Peers(); len(peers) != 1 || peers[0].Role != RoleTeach || peers[0].App != "test-teach" {
		t.Fatalf("peers = %+v", peers)
	}
}

func TestStaleRevision(t *testing.T) {
	s, c := startPair(t)
	s.Publish(testMeta("m"), testRender(1, nil))
	waitScene(t, c, 1)
	s.Publish(testMeta("m"), testRender(2, nil))
	_, _, err := c.Frame(context.Background(), 1, "red", KindTest)
	if !errors.Is(err, ErrStale) {
		t.Fatalf("frame of rev 1 after rev 2: err = %v, want ErrStale", err)
	}
	f, rev, err := c.Frame(context.Background(), 0, "red", KindTest)
	if err != nil || rev != 2 {
		t.Fatalf("frame of the current rev: rev %d err %v", rev, err)
	}
	want, _ := testRender(2, nil)("red", KindTest)
	if !bytes.Equal(f.Pix, want.Pix) {
		t.Fatal("frame is not rendered by the newest renderer")
	}
}

func TestPublishWhileRendering(t *testing.T) {
	s, c := startPair(t)
	started := make(chan struct{})
	release := make(chan struct{})
	s.Publish(testMeta("m"), func(channel, kind string) (snap.Frame, error) {
		close(started)
		<-release
		return testRender(1, nil)(channel, kind)
	})
	waitScene(t, c, 1)
	errc := make(chan error, 1)
	go func() {
		_, _, err := c.Frame(context.Background(), 1, "red", KindTest)
		errc <- err
	}()
	<-started
	s.Publish(testMeta("m"), testRender(2, nil))
	close(release)
	select {
	case err := <-errc:
		if !errors.Is(err, ErrStale) {
			t.Fatalf("err = %v, want ErrStale", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("frame request did not finish")
	}
}

func TestNoSceneAndBadRequests(t *testing.T) {
	s, c := startPair(t)
	_, _, err := c.Frame(context.Background(), 0, "red", KindTest)
	var re *RemoteError
	if !errors.As(err, &re) || re.Code != CodeNoScene {
		t.Fatalf("before Publish: err = %v, want no-scene", err)
	}
	s.Publish(testMeta("m"), testRender(1, nil))
	waitScene(t, c, 1)
	for _, tc := range []struct{ channel, kind string }{{"cyan", KindTest}, {"red", "sepia"}} {
		_, _, err := c.Frame(context.Background(), 0, tc.channel, tc.kind)
		if !errors.As(err, &re) || re.Code != CodeBadRequest {
			t.Fatalf("Frame(%s, %s): err = %v, want bad-request", tc.channel, tc.kind, err)
		}
	}
	s.Publish(testMeta("m"), func(string, string) (snap.Frame, error) { return snap.Frame{}, errors.New("boom") })
	waitScene(t, c, 2)
	_, _, err = c.Frame(context.Background(), 0, "red", KindTest)
	if !errors.As(err, &re) || re.Code != CodeRender || re.Message != "boom" {
		t.Fatalf("failing renderer: err = %v, want render/boom", err)
	}
}

func TestPublishCoalescesToNewest(t *testing.T) {
	s, c := startPair(t)
	var last int64
	for i := 0; i < 50; i++ {
		last = s.Publish(testMeta("m"), testRender(i, nil))
	}
	sc := waitScene(t, c, last)
	if sc.Rev != last {
		t.Fatalf("client rev %d, want %d", sc.Rev, last)
	}
}

func TestFrameCacheAndPublishInvalidates(t *testing.T) {
	s, c := startPair(t)
	var renders atomic.Int32
	s.Publish(testMeta("m"), testRender(1, &renders))
	waitScene(t, c, 1)
	for i := 0; i < 3; i++ {
		if _, _, err := c.Frame(context.Background(), 1, "red", KindTest); err != nil {
			t.Fatal(err)
		}
	}
	if n := renders.Load(); n != 1 {
		t.Fatalf("renderer ran %d times for 3 identical requests, want 1", n)
	}
	s.Publish(testMeta("m"), testRender(2, &renders))
	waitScene(t, c, 2)
	if _, _, err := c.Frame(context.Background(), 2, "red", KindTest); err != nil {
		t.Fatal(err)
	}
	if n := renders.Load(); n != 2 {
		t.Fatalf("renderer ran %d times after a new scene, want 2", n)
	}
}

func TestSeveralClientsAndConcurrentFrames(t *testing.T) {
	s, err := Listen("127.0.0.1:0", "studio")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	s.Publish(testMeta("m"), testRender(1, nil))
	var wg sync.WaitGroup
	for i := 0; i < 3; i++ {
		c, err := Dial(s.Addr().String(), "teach", 2*time.Second)
		if err != nil {
			t.Fatal(err)
		}
		defer c.Close()
		waitScene(t, c, 1)
		for _, ch := range []string{"red", "green", "blue"} {
			wg.Add(1)
			go func(ch string) {
				defer wg.Done()
				f, _, err := c.Frame(context.Background(), 1, ch, KindTest)
				want, _ := testRender(1, nil)(ch, KindTest)
				if err != nil || !bytes.Equal(f.Pix, want.Pix) {
					t.Errorf("concurrent frame %s: err %v", ch, err)
				}
			}(ch)
		}
	}
	wg.Wait()
	deadline := time.Now().Add(2 * time.Second)
	for len(s.Peers()) != 3 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if n := len(s.Peers()); n != 3 {
		t.Fatalf("%d peers, want 3", n)
	}
}

func TestServerCloseEndsClient(t *testing.T) {
	s, c := startPair(t)
	s.Publish(testMeta("m"), testRender(1, nil))
	waitScene(t, c, 1)
	s.Close()
	select {
	case <-c.Done():
	case <-time.After(3 * time.Second):
		t.Fatal("client did not notice the server closing")
	}
	if _, _, err := c.Frame(context.Background(), 0, "red", KindTest); err == nil {
		t.Fatal("Frame on an ended connection succeeded")
	}
}

func TestClientCancel(t *testing.T) {
	s, c := startPair(t)
	release := make(chan struct{})
	s.Publish(testMeta("m"), func(channel, kind string) (snap.Frame, error) {
		<-release
		return testRender(1, nil)(channel, kind)
	})
	waitScene(t, c, 1)
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	_, _, err := c.Frame(ctx, 1, "red", KindTest)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("err = %v, want deadline exceeded", err)
	}
	close(release)
	// The connection stays usable after an abandoned request.
	if _, _, err := c.Frame(context.Background(), 1, "green", KindTest); err != nil {
		t.Fatalf("frame after a cancelled request: %v", err)
	}
}

func TestLargeFrame(t *testing.T) {
	s, c := startPair(t)
	const w, h = 2048, 1536
	s.Publish(testMeta("m"), func(channel, kind string) (snap.Frame, error) {
		pix := make([]byte, w*h)
		for i := range pix {
			pix[i] = byte(i * 7)
		}
		return snap.Frame{Width: w, Height: h, Pix: pix}, nil
	})
	waitScene(t, c, 1)
	f, _, err := c.Frame(context.Background(), 1, "blue", KindGood)
	if err != nil {
		t.Fatal(err)
	}
	if f.Width != w || f.Height != h || f.Pix[12345] != byte(12345*7%256) {
		t.Fatalf("large frame wrong: %dx%d", f.Width, f.Height)
	}
}

func TestDialRejectsNonLinkPeer(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		_, _ = conn.Write([]byte("HTTP/1.1 400 Bad Request\r\n\r\nnot a link server, padding padding"))
		time.Sleep(200 * time.Millisecond)
	}()
	if _, err := Dial(ln.Addr().String(), "teach", time.Second); err == nil {
		t.Fatal("Dial accepted a peer that does not speak the protocol")
	}
}

func TestServerRejectsOtherProtocolVersion(t *testing.T) {
	s, err := Listen("127.0.0.1:0", "studio")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	conn, err := net.Dial("tcp", s.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	var buf bytes.Buffer
	if err := writeJSON(&buf, typeHello, 0, Hello{Protocol: Protocol, Role: RoleTeach}, nil); err != nil {
		t.Fatal(err)
	}
	raw := buf.Bytes()
	raw[4] = Protocol + 1
	if _, err := conn.Write(raw); err != nil {
		t.Fatal(err)
	}
	_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	// The server answers with its own hello and an error, then closes; reading must end
	// with the connection closed rather than hang.
	for {
		if _, err := readMessage(conn); err != nil {
			break
		}
	}
	if n := len(s.Peers()); n != 0 {
		t.Fatalf("%d peers registered after a bad handshake", n)
	}
}

func TestMessageFraming(t *testing.T) {
	var buf bytes.Buffer
	if err := writeJSON(&buf, typeFrame, 7, frameHeader{Rev: 3, Channel: "red", Kind: KindTest, Width: 2, Height: 2, Enc: "raw"}, []byte{1, 2, 3, 4}); err != nil {
		t.Fatal(err)
	}
	m, err := readMessage(&buf)
	if err != nil {
		t.Fatal(err)
	}
	f, rev, err := decodeFrame(m)
	if err != nil || rev != 3 || m.id != 7 || !bytes.Equal(f.Pix, []byte{1, 2, 3, 4}) {
		t.Fatalf("decoded %+v rev %d id %d err %v", f, rev, m.id, err)
	}
	// A frame whose body does not match its header is rejected.
	m.body = m.body[:3]
	if _, _, err := decodeFrame(m); err == nil {
		t.Fatal("short frame body accepted")
	}
}
