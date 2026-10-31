package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestInitialize(t *testing.T) {
	u := newUpstream(t, "x")
	s := newTestServer(t, u)

	resp, ok := rpc(t, s, `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-03-26","clientInfo":{"name":"t","version":"1"}}}`)
	if !ok {
		t.Fatal("no response")
	}
	result, _ := resp["result"].(map[string]any)
	if result["protocolVersion"] != "2025-03-26" {
		t.Errorf("protocolVersion = %v, want the client's version echoed", result["protocolVersion"])
	}
	info, _ := result["serverInfo"].(map[string]any)
	if info["name"] != serverName {
		t.Errorf("serverInfo = %v", info)
	}
	caps, _ := result["capabilities"].(map[string]any)
	if _, ok := caps["tools"]; !ok {
		t.Errorf("capabilities lacks tools: %v", caps)
	}
	if instr, _ := result["instructions"].(string); !strings.Contains(instr, "fake") {
		t.Errorf("instructions should list the configured providers: %q", instr)
	}
}

func TestInitializeUnknownVersionFallsBack(t *testing.T) {
	s := newTestServer(t, newUpstream(t, "x"))
	resp, _ := rpc(t, s, `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"1999-01-01"}}`)
	result, _ := resp["result"].(map[string]any)
	if result["protocolVersion"] != protocolLatest {
		t.Errorf("protocolVersion = %v, want %q", result["protocolVersion"], protocolLatest)
	}
}

func TestToolsList(t *testing.T) {
	s := newTestServer(t, newUpstream(t, "x"))
	resp, _ := rpc(t, s, `{"jsonrpc":"2.0","id":2,"method":"tools/list"}`)
	result, _ := resp["result"].(map[string]any)
	tools, _ := result["tools"].([]any)
	if len(tools) != 4 {
		t.Fatalf("tool count = %d, want 4: %v", len(tools), result)
	}
	seen := map[string]bool{}
	for _, raw := range tools {
		tool := raw.(map[string]any)
		name, _ := tool["name"].(string)
		seen[name] = true
		if d, _ := tool["description"].(string); d == "" {
			t.Errorf("tool %s has no description", name)
		}
		schema, _ := tool["inputSchema"].(map[string]any)
		if schema["type"] != "object" {
			t.Errorf("tool %s schema = %v", name, schema)
		}
	}
	for _, want := range []string{"ask", "chat", "list_models", "reset_session"} {
		if !seen[want] {
			t.Errorf("missing tool %q", want)
		}
	}
}

func TestAskTool(t *testing.T) {
	u := newUpstream(t, "第二个意见：可以")
	s := newTestServer(t, u)
	text, isErr := callTool(t, s, "ask", `{"prompt":"这个函数有问题吗？","system":"你是代码审查员"}`)
	if isErr {
		t.Fatalf("unexpected tool error: %s", text)
	}
	if text != "第二个意见：可以" {
		t.Errorf("text = %q", text)
	}
	body := u.last().Body
	if body["model"] != "fake-chat" {
		t.Errorf("model = %v, want the provider default", body["model"])
	}
	if body["max_tokens"] != float64(256) {
		t.Errorf("max_tokens = %v, want the provider default 256", body["max_tokens"])
	}
	msgs := u.messages()
	if len(msgs) != 2 || msgs[0]["role"] != "system" || msgs[1]["role"] != "user" {
		t.Fatalf("messages = %v", msgs)
	}
	if msgs[1]["content"] != "这个函数有问题吗？" {
		t.Errorf("user content = %v", msgs[1]["content"])
	}
}

func TestAskToolVerboseFooter(t *testing.T) {
	s := newTestServer(t, newUpstream(t, "ok"))
	text, _ := callTool(t, s, "ask", `{"prompt":"hi","verbose":true}`)
	if !strings.Contains(text, "[model=fake-chat") || !strings.Contains(text, "tokens=11/7") {
		t.Errorf("verbose text = %q", text)
	}
}

func TestAskToolModelSelection(t *testing.T) {
	u := newUpstream(t, "ok")
	s := newTestServer(t, u)
	if _, isErr := callTool(t, s, "ask", `{"prompt":"hi","model":"big"}`); isErr {
		t.Fatal("alias call failed")
	}
	if got := u.last().Body["model"]; got != "fake-chat-big" {
		t.Errorf("model = %v, want the alias target fake-chat-big", got)
	}
	if _, isErr := callTool(t, s, "ask", `{"prompt":"hi","model":"fake-chat-other"}`); isErr {
		t.Fatal("literal model call failed")
	}
	if got := u.last().Body["model"]; got != "fake-chat-other" {
		t.Errorf("model = %v, want the literal id", got)
	}
}

func TestAskToolSessionKeepsHistory(t *testing.T) {
	u := newUpstream(t, "回答")
	s := newTestServer(t, u)
	if _, isErr := callTool(t, s, "ask", `{"prompt":"我叫 Tony","session":"s1"}`); isErr {
		t.Fatal("first call failed")
	}
	if _, isErr := callTool(t, s, "ask", `{"prompt":"我叫什么？","session":"s1"}`); isErr {
		t.Fatal("second call failed")
	}
	msgs := u.messages()
	if len(msgs) != 3 {
		t.Fatalf("second request should carry history, got %v", msgs)
	}
	if msgs[0]["role"] != "user" || msgs[1]["role"] != "assistant" || msgs[2]["content"] != "我叫什么？" {
		t.Errorf("history = %v", msgs)
	}
	if s.sessions.Len() != 1 {
		t.Errorf("sessions = %d", s.sessions.Len())
	}
	text, _ := callTool(t, s, "reset_session", `{"session":"s1"}`)
	if !strings.Contains(text, "dropped") || s.sessions.Len() != 0 {
		t.Errorf("reset = %q, sessions = %d", text, s.sessions.Len())
	}
}

func TestStatelessAskDoesNotKeepHistory(t *testing.T) {
	u := newUpstream(t, "回答")
	s := newTestServer(t, u)
	callTool(t, s, "ask", `{"prompt":"a"}`)
	callTool(t, s, "ask", `{"prompt":"b"}`)
	if len(u.messages()) != 1 {
		t.Errorf("a stateless call must send a single message, got %v", u.messages())
	}
	if s.sessions.Len() != 0 {
		t.Errorf("sessions = %d, want 0", s.sessions.Len())
	}
}

func TestChatTool(t *testing.T) {
	u := newUpstream(t, "assistant answer")
	s := newTestServer(t, u)
	text, isErr := callTool(t, s, "chat",
		`{"messages":[{"role":"user","content":"问题一"},{"role":"assistant","content":"答一"},{"role":"user","content":"问题二"}]}`)
	if isErr {
		t.Fatalf("chat failed: %s", text)
	}
	if text != "assistant answer" {
		t.Errorf("text = %q", text)
	}
	if got := len(u.messages()); got != 3 {
		t.Errorf("messages = %d", got)
	}
}

func TestChatToolValidatesMessages(t *testing.T) {
	s := newTestServer(t, newUpstream(t, "x"))
	text, isErr := callTool(t, s, "chat", `{"messages":[]}`)
	if !isErr || !strings.Contains(text, "messages is required") {
		t.Errorf("text = %q, isErr = %v", text, isErr)
	}
	text, isErr = callTool(t, s, "ask", `{"prompt":"  "}`)
	if !isErr || !strings.Contains(text, "prompt is required") {
		t.Errorf("text = %q, isErr = %v", text, isErr)
	}
}

func TestUpstreamErrorBecomesToolError(t *testing.T) {
	u := newUpstream(t, "")
	u.set(func(u *upstream) { u.status = http.StatusUnauthorized })
	s := newTestServer(t, u)
	text, isErr := callTool(t, s, "ask", `{"prompt":"hi"}`)
	if !isErr {
		t.Fatalf("want an isError result, got %q", text)
	}
	if !strings.Contains(text, "invalid api key") || !strings.Contains(text, "hint:") {
		t.Errorf("error text = %q", text)
	}
}

func TestEmptyAnswerIsReported(t *testing.T) {
	s := newTestServer(t, newUpstream(t, ""))
	text, isErr := callTool(t, s, "ask", `{"prompt":"hi"}`)
	if !isErr || !strings.Contains(text, "empty answer") {
		t.Errorf("text = %q, isErr = %v", text, isErr)
	}
}

func TestMissingKeyIsRefusedBeforeCallingOut(t *testing.T) {
	u := newUpstream(t, "x")
	cfg, err := ParseConfig([]byte(`{"providers":[{"name":"fake","base_url":"` + u.URL + `/v1","model":"m"}]}`))
	if err != nil {
		t.Fatal(err)
	}
	s := NewServer(cfg, NewClient(0, 1), log.New(io.Discard, "", 0))
	text, isErr := callTool(t, s, "ask", `{"prompt":"hi"}`)
	if !isErr || !strings.Contains(text, "no API key") {
		t.Errorf("text = %q, isErr = %v", text, isErr)
	}
	if u.count() != 0 {
		t.Errorf("upstream was called %d times", u.count())
	}
}

func TestListModelsTool(t *testing.T) {
	s := newTestServer(t, newUpstream(t, "x"))
	text, isErr := callTool(t, s, "list_models", `{}`)
	if isErr {
		t.Fatalf("list_models failed: %s", text)
	}
	var out struct {
		Server    string     `json:"server"`
		Providers []Describe `json:"providers"`
	}
	if err := json.Unmarshal([]byte(text), &out); err != nil {
		t.Fatalf("not JSON: %v\n%s", err, text)
	}
	if out.Server != serverName || len(out.Providers) != 1 {
		t.Fatalf("out = %+v", out)
	}
	p := out.Providers[0]
	if p.DefaultModel != "fake-chat" || !p.HasKey || p.Models["big"] != "fake-chat-big" {
		t.Errorf("provider = %+v", p)
	}
	if strings.Contains(text, "sk-test") {
		t.Error("list_models leaked the API key")
	}
}

func TestProtocolErrors(t *testing.T) {
	s := newTestServer(t, newUpstream(t, "x"))
	resp, _ := rpc(t, s, `{"jsonrpc":"2.0","id":3,"method":"nope/nothing"}`)
	errBox, _ := resp["error"].(map[string]any)
	if errBox["code"] != float64(-32601) {
		t.Errorf("error = %v", errBox)
	}
	resp, _ = rpc(t, s, `{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"nope"}}`)
	errBox, _ = resp["error"].(map[string]any)
	if errBox["code"] != float64(-32602) {
		t.Errorf("error = %v", errBox)
	}
	resp, _ = rpc(t, s, `not json at all`)
	errBox, _ = resp["error"].(map[string]any)
	if errBox["code"] != float64(-32700) {
		t.Errorf("parse error = %v", errBox)
	}
}

func TestNotificationGetsNoResponse(t *testing.T) {
	s := newTestServer(t, newUpstream(t, "x"))
	for _, msg := range []string{
		`{"jsonrpc":"2.0","method":"notifications/initialized"}`,
		`{"jsonrpc":"2.0","method":"notifications/cancelled","params":{"requestId":9}}`,
	} {
		if resp, has := s.Handle(context.Background(), []byte(msg)); has {
			t.Errorf("%s answered %s", msg, resp)
		}
	}
}

func TestPing(t *testing.T) {
	s := newTestServer(t, newUpstream(t, "x"))
	resp, ok := rpc(t, s, `{"jsonrpc":"2.0","id":5,"method":"ping"}`)
	if !ok {
		t.Fatal("no response")
	}
	if _, ok := resp["result"]; !ok {
		t.Errorf("ping = %v", resp)
	}
}

func TestCancelRegistryStopsCall(t *testing.T) {
	stopped := make(chan struct{})
	reg := newCancelRegistry()
	reg.add("5", func() { close(stopped) })
	s := &Server{cancels: reg, log: log.New(io.Discard, "", 0)}
	s.cancelRequest(json.RawMessage(`{"requestId":5,"reason":"user"}`))
	select {
	case <-stopped:
	case <-time.After(time.Second):
		t.Fatal("cancel was not propagated")
	}
	reg.remove("5")
	if reg.cancel("5") {
		t.Fatal("removed id must not cancel anything")
	}
}

func TestBatch(t *testing.T) {
	s := newTestServer(t, newUpstream(t, "x"))
	out, has := s.batch(context.Background(), []byte(`[
	  {"jsonrpc":"2.0","id":1,"method":"ping"},
	  {"jsonrpc":"2.0","method":"notifications/initialized"},
	  {"jsonrpc":"2.0","id":2,"method":"tools/list"}
	]`), nil)
	if !has {
		t.Fatal("batch produced no answer")
	}
	var items []map[string]any
	if err := json.Unmarshal(out, &items); err != nil {
		t.Fatalf("not an array: %v\n%s", err, out)
	}
	if len(items) != 2 {
		t.Fatalf("items = %d, want 2 (the notification is dropped)", len(items))
	}
}

func TestHTTPTransport(t *testing.T) {
	u := newUpstream(t, "http 回答")
	s := newTestServer(t, u)
	ts := httptest.NewServer(NewHandler(s, "secret-token"))
	defer ts.Close()

	post := func(path, body string, headers map[string]string) *http.Response {
		req, err := http.NewRequest(http.MethodPost, ts.URL+path, strings.NewReader(body))
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("Content-Type", "application/json")
		for k, v := range headers {
			req.Header.Set(k, v)
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		return resp
	}

	req := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"ask","arguments":{"prompt":"hi"}}}`
	resp := post("/mcp", req, nil)
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("without token: status = %d, want 401", resp.StatusCode)
	}
	resp.Body.Close()

	resp = post("/mcp", req, map[string]string{"Authorization": "Bearer secret-token"})
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("with token: status = %d, body = %s", resp.StatusCode, body)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if !strings.Contains(string(body), "http 回答") {
		t.Errorf("body = %s", body)
	}

	// SSE-only clients get an event stream.
	resp = post("/mcp", `{"jsonrpc":"2.0","id":2,"method":"ping"}`,
		map[string]string{"Authorization": "Bearer secret-token", "Accept": "text/event-stream"})
	if ct := resp.Header.Get("Content-Type"); !strings.Contains(ct, "text/event-stream") {
		t.Errorf("content-type = %q", ct)
	}
	sse, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if !strings.HasPrefix(string(sse), "event: message\ndata: {") {
		t.Errorf("sse = %q", sse)
	}

	// Notifications only -> 202.
	resp = post("/mcp", `{"jsonrpc":"2.0","method":"notifications/initialized"}`,
		map[string]string{"Authorization": "Bearer secret-token"})
	if resp.StatusCode != http.StatusAccepted {
		t.Errorf("notification status = %d, want 202", resp.StatusCode)
	}
	resp.Body.Close()

	// GET /mcp is not a transport we support (auth is checked first).
	getReq, _ := http.NewRequest(http.MethodGet, ts.URL+"/mcp", nil)
	getReq.Header.Set("Authorization", "Bearer secret-token")
	getResp, err := http.DefaultClient.Do(getReq)
	if err != nil {
		t.Fatal(err)
	}
	if getResp.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("GET /mcp = %d", getResp.StatusCode)
	}
	getResp.Body.Close()

	// ... and without a token it is rejected before routing.
	unauthGet, err := http.Get(ts.URL + "/mcp")
	if err != nil {
		t.Fatal(err)
	}
	if unauthGet.StatusCode != http.StatusUnauthorized {
		t.Errorf("unauthenticated GET /mcp = %d, want 401", unauthGet.StatusCode)
	}
	unauthGet.Body.Close()

	// /healthz stays open so a supervisor can probe it.
	health, err := http.Get(ts.URL + "/healthz")
	if err != nil {
		t.Fatal(err)
	}
	if health.StatusCode != http.StatusOK {
		t.Errorf("healthz = %d", health.StatusCode)
	}
	health.Body.Close()
}

func TestHTTPWithoutToken(t *testing.T) {
	s := newTestServer(t, newUpstream(t, "x"))
	ts := httptest.NewServer(NewHandler(s, ""))
	defer ts.Close()
	resp, err := http.Post(ts.URL+"/mcp", "application/json",
		strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"ping"}`))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", resp.StatusCode)
	}
}

func TestHTTPBatch(t *testing.T) {
	s := newTestServer(t, newUpstream(t, "x"))
	ts := httptest.NewServer(NewHandler(s, ""))
	defer ts.Close()
	resp, err := http.Post(ts.URL+"/mcp", "application/json", strings.NewReader(
		`[{"jsonrpc":"2.0","id":1,"method":"ping"},{"jsonrpc":"2.0","id":2,"method":"tools/list"}]`))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if !strings.HasPrefix(strings.TrimSpace(string(body)), "[") {
		t.Errorf("batch body = %s", body)
	}
}

func TestStdioRoundTrip(t *testing.T) {
	s := newTestServer(t, newUpstream(t, "stdio 回答"))
	in := strings.NewReader(strings.Join([]string{
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18"}}`,
		``,
		`{"jsonrpc":"2.0","method":"notifications/initialized"}`,
		`{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"ask","arguments":{"prompt":"hi"}}}`,
	}, "\n") + "\n")
	var out bytes.Buffer
	if err := runStdio(context.Background(), in, &out, s, log.New(io.Discard, "", 0)); err != nil {
		t.Fatalf("runStdio: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	if len(lines) != 2 {
		t.Fatalf("stdout lines = %d, want 2:\n%s", len(lines), out.String())
	}
	for _, line := range lines {
		var msg map[string]any
		if err := json.Unmarshal([]byte(line), &msg); err != nil {
			t.Fatalf("stdout is not pure JSON-RPC: %v\n%s", err, line)
		}
	}
	if !strings.Contains(lines[1], "stdio 回答") {
		t.Errorf("second response = %s", lines[1])
	}
}

func TestNormalizeAddr(t *testing.T) {
	cases := map[string]string{
		"8765":           "127.0.0.1:8765",
		"127.0.0.1:9000": "127.0.0.1:9000",
		":9000":          ":9000",
		" 127.0.0.1:1 ":  "127.0.0.1:1",
	}
	for in, want := range cases {
		if got := normalizeAddr(in); got != want {
			t.Errorf("normalizeAddr(%q) = %q, want %q", in, got, want)
		}
	}
}
