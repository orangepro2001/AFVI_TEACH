package main

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestChatStreamAccumulates(t *testing.T) {
	u := newUpstream(t, "这是一个被切成四块的回答")
	u.set(func(u *upstream) { u.sse = true })
	p := newProviderConfig(t, u, "")
	c := NewClient(0, 1)

	var updates []string
	res, err := c.ChatStream(context.Background(), Target{Provider: p, Model: "m1"}, ChatOptions{
		Messages: []Message{{Role: "user", Content: "hi"}},
	}, func(d Delta) { updates = append(updates, d.Text) })
	if err != nil {
		t.Fatalf("ChatStream: %v", err)
	}
	if !res.Streamed {
		t.Error("Streamed should be true for an SSE answer")
	}
	if res.Text != "这是一个被切成四块的回答" {
		t.Errorf("Text = %q", res.Text)
	}
	if res.FinishReason != "stop" || res.OutputTokens != 7 || res.PromptTokens != 11 {
		t.Errorf("meta = %+v", res)
	}
	if len(updates) < 3 {
		t.Errorf("expected incremental updates, got %d: %v", len(updates), updates)
	}
	if updates[len(updates)-1] != res.Text {
		t.Errorf("last update %q should equal the final text", updates[len(updates)-1])
	}
	if u.last().Body["stream"] != true || u.last().Body["stream_options"] == nil {
		t.Errorf("request did not ask for a stream with usage: %v", u.last().Body)
	}
}

func TestChatStreamFallsBackWhenEndpointIgnoresStream(t *testing.T) {
	u := newUpstream(t, "整段返回") // sse off: answers application/json
	p := newProviderConfig(t, u, "")
	c := NewClient(0, 1)
	var sawDelta bool
	res, err := c.ChatStream(context.Background(), Target{Provider: p, Model: "m1"}, ChatOptions{
		Messages: []Message{{Role: "user", Content: "hi"}},
	}, func(d Delta) { sawDelta = true })
	if err != nil {
		t.Fatalf("ChatStream: %v", err)
	}
	if res.Streamed || res.Text != "整段返回" || !sawDelta {
		t.Errorf("res = %+v sawDelta=%v", res, sawDelta)
	}
}

// recorder collects everything a transport would push to the client.
type recorder struct {
	mu   sync.Mutex
	msgs []map[string]any
}

func (r *recorder) emit(payload []byte) error {
	var m map[string]any
	if err := json.Unmarshal(payload, &m); err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.msgs = append(r.msgs, m)
	return nil
}

func (r *recorder) progress() []map[string]any {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []map[string]any
	for _, m := range r.msgs {
		if m["method"] == "notifications/progress" {
			out = append(out, m)
		}
	}
	return out
}

func TestProgressNotificationsKeepSlowCallAlive(t *testing.T) {
	old := progressEvery
	progressEvery = 40 * time.Millisecond
	defer func() { progressEvery = old }()

	u := newUpstream(t, strings.Repeat("长回答", 40))
	u.set(func(u *upstream) { u.sse = true; u.chunkDelayMs = 60 })
	s := newTestServer(t, u)

	rec := &recorder{}
	req := `{"jsonrpc":"2.0","id":9,"method":"tools/call",
	  "params":{"name":"ask","arguments":{"prompt":"长任务"},"_meta":{"progressToken":"tok-1"}}}`
	resp, ok := s.HandleEmit(context.Background(), []byte(req), rec.emit)
	if !ok {
		t.Fatal("no response")
	}
	var out map[string]any
	if err := json.Unmarshal(resp, &out); err != nil {
		t.Fatalf("bad response: %v", err)
	}
	if errBox, bad := out["error"]; bad {
		t.Fatalf("tool call failed: %v", errBox)
	}

	progress := rec.progress()
	if len(progress) < 2 {
		t.Fatalf("expected several progress notifications, got %d: %v", len(progress), progress)
	}
	last := 0.0
	for _, p := range progress {
		params, _ := p["params"].(map[string]any)
		if params["progressToken"] != "tok-1" {
			t.Errorf("progressToken = %v, want tok-1", params["progressToken"])
		}
		v, _ := params["progress"].(float64)
		if v < last {
			t.Errorf("progress went backwards: %v after %v", v, last)
		}
		last = v
		if _, ok := p["id"]; ok {
			t.Errorf("a notification must not carry an id: %v", p)
		}
	}
	// The final notification says the call finished.
	finalParams, _ := progress[len(progress)-1]["params"].(map[string]any)
	if msg, _ := finalParams["message"].(string); !strings.Contains(msg, "完成") {
		t.Errorf("last progress message = %q", msg)
	}
}

func TestNoProgressWithoutToken(t *testing.T) {
	u := newUpstream(t, "短回答")
	u.set(func(u *upstream) { u.sse = true })
	s := newTestServer(t, u)
	rec := &recorder{}
	req := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"ask","arguments":{"prompt":"hi"}}}`
	if _, ok := s.HandleEmit(context.Background(), []byte(req), rec.emit); !ok {
		t.Fatal("no response")
	}
	if got := len(rec.progress()); got != 0 {
		t.Errorf("client sent no progressToken, so nothing must be pushed; got %d", got)
	}
}

func TestHTTPStreamsProgressOverSSE(t *testing.T) {
	old := progressEvery
	progressEvery = 30 * time.Millisecond
	defer func() { progressEvery = old }()

	u := newUpstream(t, strings.Repeat("片段", 30))
	u.set(func(u *upstream) { u.sse = true; u.chunkDelayMs = 50 })
	s := newTestServer(t, u)
	ts := httptest.NewServer(NewHandler(s, ""))
	defer ts.Close()

	body := `{"jsonrpc":"2.0","id":1,"method":"tools/call",
	  "params":{"name":"ask","arguments":{"prompt":"hi"},"_meta":{"progressToken":"h1"}}}`
	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/mcp", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream, application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if ct := resp.Header.Get("Content-Type"); !strings.Contains(ct, "text/event-stream") {
		t.Fatalf("content-type = %q, want an event stream", ct)
	}
	raw, _ := io.ReadAll(resp.Body)
	text := string(raw)
	if !strings.Contains(text, "notifications/progress") {
		t.Fatalf("no progress notification in the stream:\n%s", text)
	}
	if !strings.Contains(text, `"id":1`) {
		t.Errorf("the final response is missing from the stream:\n%s", text)
	}
	// Progress must arrive before the result line.
	pi := strings.Index(text, "notifications/progress")
	ri := strings.Index(text, `"id":1`)
	if pi > ri {
		t.Errorf("progress arrived after the result:\n%s", text)
	}
}

func TestStreamErrorFrameIsReported(t *testing.T) {
	s := newTestServer(t, newUpstream(t, "x"))
	_ = s
	frames := "data: {\"error\":{\"message\":\"quota exceeded\"}}\n\ndata: [DONE]\n\n"
	_, err := readSSE("p", strings.NewReader(frames), "m", nil)
	if err == nil || !strings.Contains(err.Error(), "quota exceeded") {
		t.Fatalf("err = %v", err)
	}
}

func TestReadSSEHandlesKeepAliveAndGarbage(t *testing.T) {
	frames := strings.Join([]string{
		": keep-alive",
		"event: message",
		`data: {"choices":[{"delta":{"content":"你"}}]}`,
		`data: {"choices":[{"delta":{"content":"好"}}]}`,
		`data: {not json}`,
		`data: {"choices":[{"delta":{},"finish_reason":"stop"}]}`,
		"data: [DONE]",
	}, "\n\n")
	res, err := readSSE("p", strings.NewReader(frames), "m", nil)
	if err != nil {
		t.Fatalf("readSSE: %v", err)
	}
	if res.Text != "你好" || res.FinishReason != "stop" {
		t.Errorf("res = %+v", res)
	}
}

func TestRenderResultMentionsTruncation(t *testing.T) {
	text := renderResult(ChatResult{Text: "abc", FinishReason: "length"}, false)
	if !strings.Contains(text, "finish_reason=length") {
		t.Errorf("text = %q", text)
	}
}

var _ = log.New

func TestRunStdioStopsOnContextCancel(t *testing.T) {
	s := newTestServer(t, newUpstream(t, "x"))
	pr, pw := io.Pipe()
	defer pw.Close()
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- runStdio(ctx, pr, io.Discard, s, log.New(io.Discard, "", 0)) }()

	// A blocking read must not swallow the shutdown signal.
	time.Sleep(50 * time.Millisecond)
	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("runStdio: %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("runStdio ignored the cancelled context")
	}
}
