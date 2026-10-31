package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"sort"
	"strings"
	"sync"
)

const (
	serverName = "deepseek-mcp"
	// serverVersion is shown in initialize/serverInfo and in list_models.
	serverVersion = "0.1.0"
	// protocolLatest is what we answer when the client asks for an unknown version.
	protocolLatest = "2025-06-18"
)

// supportedVersions are the MCP protocol revisions this server speaks.
var supportedVersions = []string{"2025-06-18", "2025-03-26", "2024-11-05", "2024-10-07"}

// rpcRequest is one JSON-RPC 2.0 message.
type rpcRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

// Tool is one MCP tool.
type Tool struct {
	Name        string
	Description string
	Schema      map[string]any
	// Handler runs the tool. rep lets it send notifications/progress so clients
	// never hit their per-request timeout during a slow completion.
	Handler func(ctx context.Context, args json.RawMessage, rep progressReporter) (string, error)
}

// emitFunc writes one JSON-RPC message to the client. The transport supplies it:
// stdio writes to stdout, Streamable HTTP writes SSE events.
type emitFunc func(payload []byte) error

// progressReporter sends notifications/progress for one in-flight request. It is
// inert when the client did not send `_meta.progressToken` (most clients only
// believe progress for a token they issued).
type progressReporter struct {
	token json.RawMessage
	emit  emitFunc
}

func (p progressReporter) enabled() bool {
	return p.emit != nil && len(p.token) > 0 && string(p.token) != "null"
}

// report sends one progress notification. progress must not decrease; the
// message is what the client shows to the user.
func (p progressReporter) report(progress, total float64, message string) {
	if !p.enabled() {
		return
	}
	params := map[string]any{"progressToken": p.token, "progress": progress}
	if total > 0 {
		params["total"] = total
	}
	if message != "" {
		params["message"] = message
	}
	payload, err := json.Marshal(map[string]any{
		"jsonrpc": "2.0",
		"method":  "notifications/progress",
		"params":  params,
	})
	if err != nil {
		return
	}
	_ = p.emit(payload)
}

// Server holds the configuration and answers MCP messages. Both transports feed
// Handle.
type Server struct {
	cfg      *Config
	client   *Client
	sessions *sessionStore
	cancels  *cancelRegistry
	log      *log.Logger
	tools    []Tool
}

// NewServer wires the tools once.
func NewServer(cfg *Config, client *Client, logger *log.Logger) *Server {
	s := &Server{
		cfg:      cfg,
		client:   client,
		sessions: newSessionStore(),
		cancels:  newCancelRegistry(),
		log:      logger,
	}
	s.tools = []Tool{
		s.toolAsk(),
		s.toolChat(),
		s.toolListModels(),
		s.toolResetSession(),
	}
	return s
}

// Handle answers one JSON-RPC message without a notification channel.
func (s *Server) Handle(ctx context.Context, raw []byte) (out []byte, has bool) {
	return s.HandleEmit(ctx, raw, nil)
}

// HandleEmit answers one JSON-RPC message; emit (optional) carries mid-request
// notifications such as notifications/progress.
func (s *Server) HandleEmit(ctx context.Context, raw []byte, emit emitFunc) (out []byte, has bool) {
	var req rpcRequest
	if err := json.Unmarshal(raw, &req); err != nil {
		return marshalResponse(nil, nil, &rpcError{Code: -32700,
			Message: "parse error: " + err.Error()}), true
	}
	notification := len(req.ID) == 0 || string(req.ID) == "null"
	if req.Method == "" {
		if notification {
			return nil, false
		}
		return marshalResponse(req.ID, nil, &rpcError{Code: -32600,
			Message: "invalid request: missing method"}), true
	}
	reqCtx := ctx
	if !notification {
		// Requests are cancellable through notifications/cancelled.
		var cancel context.CancelFunc
		reqCtx, cancel = context.WithCancel(ctx)
		defer cancel()
		key := idKey(req.ID)
		s.cancels.add(key, cancel)
		defer s.cancels.remove(key)
	}
	result, rerr := s.dispatch(reqCtx, req.Method, req.Params, emit)
	if notification {
		return nil, false
	}
	return marshalResponse(req.ID, result, rerr), true
}

func (s *Server) dispatch(ctx context.Context, method string, params json.RawMessage, emit emitFunc) (any, *rpcError) {
	switch method {
	case "initialize":
		return s.initialize(params), nil
	case "ping":
		return map[string]any{}, nil
	case "tools/list":
		defs := make([]map[string]any, 0, len(s.tools))
		for _, t := range s.tools {
			defs = append(defs, map[string]any{
				"name":        t.Name,
				"description": t.Description,
				"inputSchema": t.Schema,
			})
		}
		return map[string]any{"tools": defs}, nil
	case "tools/call":
		return s.callTool(ctx, params, emit)
	case "prompts/list":
		return map[string]any{"prompts": []any{}}, nil
	case "resources/list":
		return map[string]any{"resources": []any{}}, nil
	case "resources/templates/list":
		return map[string]any{"resourceTemplates": []any{}}, nil
	case "completion/complete":
		return map[string]any{"completion": map[string]any{"values": []string{}}}, nil
	case "logging/setLevel":
		return map[string]any{}, nil
	// Notifications carry no answer.
	case "notifications/cancelled":
		s.cancelRequest(params)
		return nil, nil
	case "notifications/initialized", "initialized", "notifications/progress",
		"notifications/roots/list_changed":
		return nil, nil
	}
	return nil, &rpcError{Code: -32601, Message: "method not found: " + method}
}

func (s *Server) initialize(params json.RawMessage) any {
	var p struct {
		ProtocolVersion string `json:"protocolVersion"`
	}
	_ = json.Unmarshal(params, &p)
	version := protocolLatest
	for _, v := range supportedVersions {
		if v == p.ProtocolVersion {
			version = v
			break
		}
	}
	return map[string]any{
		"protocolVersion": version,
		"capabilities": map[string]any{
			"tools": map[string]any{"listChanged": false},
		},
		"serverInfo":   map[string]any{"name": serverName, "version": serverVersion},
		"instructions": s.instructions(),
	}
}

// instructions is the short usage text every MCP client shows to its model.
func (s *Server) instructions() string {
	var b strings.Builder
	b.WriteString("This server relays questions to external LLM endpoints. ")
	b.WriteString("Credentials and model names live on the server: the client never sends an API key, and `model` is only an optional override. ")
	b.WriteString("Use `ask` for a single self-contained question (no tool access, no repo context - ")
	b.WriteString("put everything it needs into the prompt), `chat` for a multi-turn transcript, ")
	b.WriteString("`list_models` to see which targets are reachable, `reset_session` to drop a conversation. ")
	b.WriteString("The remote model cannot read files or run commands.\n\n")
	b.WriteString(s.routingPolicy())
	return b.String()
}

// routingPolicy is the model-selection guidance, generated from the configuration
// (Provider.Cost / Provider.Note), so the calling agent knows which target to use
// for which kind of task.
func (s *Server) routingPolicy() string {
	var b strings.Builder
	def := s.cfg.DefaultProvider()
	b.WriteString("Model routing (server-side configuration):\n")
	for _, d := range s.cfg.Describe() {
		b.WriteString("  - " + d.Name + " -> " + d.DefaultModel)
		if len(d.Models) > 0 {
			keys := make([]string, 0, len(d.Models))
			for k := range d.Models {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			pairs := make([]string, 0, len(keys))
			for _, k := range keys {
				pairs = append(pairs, k+"="+d.Models[k])
			}
			b.WriteString(" (aliases: " + strings.Join(pairs, ", ") + ")")
		}
		if d.Cost != "" {
			b.WriteString(" [cost: " + d.Cost + "]")
		}
		if d.Note != "" {
			b.WriteString("\n      " + d.Note)
		}
		if d.Default {
			b.WriteString("\n      (default: used when `model` is omitted)")
		}
		b.WriteString("\n")
	}
	if def != nil {
		b.WriteString("Rule of thumb: leave `model` empty for ordinary and medium tasks (the default above); ")
		if expensive := s.expensiveProvider(); expensive != "" {
			b.WriteString("pass model=\"" + expensive + "\" only when the task is genuinely hard, high-stakes or needs the most reliable answer - it is slower and much more expensive.\n")
		} else {
			b.WriteString("pass a specific `model` only when that target fits the task better.\n")
		}
	}
	return b.String()
}

// expensiveProvider names the configured high-cost target, if any.
func (s *Server) expensiveProvider() string {
	for _, d := range s.cfg.Describe() {
		if !d.Default && (strings.EqualFold(d.Cost, "high") || strings.EqualFold(d.Cost, "expensive")) {
			return d.Name
		}
	}
	return ""
}

// routingHint is the compact version of routingPolicy used in the tool
// descriptions, so the calling model picks a target without reading instructions.
func (s *Server) routingHint() string {
	var parts []string
	if def := s.cfg.DefaultProvider(); def != nil {
		parts = append(parts, "Model routing: omit `model` for ordinary and medium tasks (default "+
			def.Name+" -> "+def.DefaultModel()+").")
	}
	if e := s.expensiveProvider(); e != "" {
		parts = append(parts, "Pass model=\""+e+"\" for genuinely hard or high-stakes work: the most reliable but slowest and most expensive target.")
	}
	for _, d := range s.cfg.Describe() {
		if d.Note != "" {
			parts = append(parts, d.Name+": "+d.Note)
		}
	}
	return strings.Join(parts, " ")
}

func (s *Server) callTool(ctx context.Context, params json.RawMessage, emit emitFunc) (any, *rpcError) {
	var p struct {
		Name      string          `json:"name"`
		Arguments json.RawMessage `json:"arguments"`
		Meta      struct {
			ProgressToken json.RawMessage `json:"progressToken"`
		} `json:"_meta"`
	}
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, &rpcError{Code: -32602, Message: "invalid params: " + err.Error()}
	}
	for _, t := range s.tools {
		if t.Name != p.Name {
			continue
		}
		args := p.Arguments
		if len(args) == 0 || string(args) == "null" {
			args = json.RawMessage("{}")
		}
		rep := progressReporter{token: p.Meta.ProgressToken, emit: emit}
		if rep.enabled() {
			rep.report(0, 0, "已提交给模型…")
		}
		text, err := t.Handler(ctx, args, rep)
		if err != nil {
			msg := errText(err)
			s.log.Printf("tool %s failed: %s", t.Name, oneLine(msg))
			return toolResult(msg, true), nil
		}
		return toolResult(text, false), nil
	}
	return nil, &rpcError{Code: -32602, Message: "unknown tool: " + p.Name}
}

func toolResult(text string, isErr bool) map[string]any {
	res := map[string]any{
		"content": []any{map[string]any{"type": "text", "text": text}},
	}
	if isErr {
		res["isError"] = true
	}
	return res
}

func marshalResponse(id json.RawMessage, result any, rerr *rpcError) []byte {
	msg := map[string]any{"jsonrpc": "2.0"}
	if len(id) == 0 {
		msg["id"] = nil
	} else {
		msg["id"] = id
	}
	if rerr != nil {
		msg["error"] = rerr
	} else {
		msg["result"] = result
	}
	b, err := json.Marshal(msg)
	if err != nil {
		// Should never happen for our own values; keep the client in sync anyway.
		return []byte(`{"jsonrpc":"2.0","id":null,"error":{"code":-32603,"message":"internal error"}}`)
	}
	return b
}

// errText renders a handler error with the actionable hint when we have one.
func errText(err error) string {
	var ae *APIError
	if errors.As(err, &ae) {
		if h := ae.Hint(); h != "" {
			return fmt.Sprintf("%v\nhint: %s", ae, h)
		}
	}
	return err.Error()
}

// batch handles a JSON-RPC batch (an array of messages). Responses keep the input
// order; notifications are dropped.
func (s *Server) batch(ctx context.Context, raw []byte, emit emitFunc) (out []byte, has bool) {
	var items []json.RawMessage
	if err := json.Unmarshal(raw, &items); err != nil {
		return s.HandleEmit(ctx, raw, emit)
	}
	responses := make([]json.RawMessage, 0, len(items))
	for _, item := range items {
		if resp, ok := s.HandleEmit(ctx, item, emit); ok {
			responses = append(responses, resp)
		}
	}
	if len(responses) == 0 {
		return nil, false
	}
	b, err := json.Marshal(responses)
	if err != nil {
		return nil, false
	}
	return b, true
}

// sessionStore keeps short conversations by id so a client can continue a thread
// without resending it. In-memory, per process.
type sessionStore struct {
	mu      sync.Mutex
	m       map[string]*session
	maxKeep int
}

type session struct {
	messages []Message
}

const (
	maxSessions     = 512
	maxSessionMsgs  = 60
	maxSessionRunes = 120000
)

func newSessionStore() *sessionStore {
	return &sessionStore{m: map[string]*session{}, maxKeep: maxSessions}
}

// Get returns a copy of the session's messages.
func (s *sessionStore) Get(id string) []Message {
	s.mu.Lock()
	defer s.mu.Unlock()
	sess := s.m[id]
	if sess == nil {
		return nil
	}
	out := make([]Message, len(sess.messages))
	copy(out, sess.messages)
	return out
}

// Append adds messages and trims the history from the front.
func (s *sessionStore) Append(id string, msgs ...Message) {
	if id == "" || len(msgs) == 0 {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	sess := s.m[id]
	if sess == nil {
		if len(s.m) >= s.maxKeep {
			s.dropOldestLocked()
		}
		sess = &session{}
		s.m[id] = sess
	}
	sess.messages = append(sess.messages, msgs...)
	for len(sess.messages) > maxSessionMsgs {
		sess.messages = sess.messages[1:]
	}
	for runeLen(sess.messages) > maxSessionRunes && len(sess.messages) > 2 {
		sess.messages = sess.messages[1:]
	}
}

// Reset drops one session and reports whether it existed.
func (s *sessionStore) Reset(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.m[id]
	delete(s.m, id)
	return ok
}

// Len is the number of live sessions.
func (s *sessionStore) Len() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.m)
}

// dropOldestLocked evicts an arbitrary session when the table is full; sessions
// are a convenience, not durable state.
func (s *sessionStore) dropOldestLocked() {
	for id := range s.m {
		delete(s.m, id)
		return
	}
}

func runeLen(msgs []Message) int {
	n := 0
	for _, m := range msgs {
		n += len([]rune(m.Content))
	}
	return n
}

func oneLine(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

// cancelRegistry maps a request id to the cancel func of the running call, so a
// notifications/cancelled message can stop a long completion.
type cancelRegistry struct {
	mu sync.Mutex
	m  map[string]context.CancelFunc
}

func newCancelRegistry() *cancelRegistry {
	return &cancelRegistry{m: map[string]context.CancelFunc{}}
}

func (r *cancelRegistry) add(key string, cancel context.CancelFunc) {
	if key == "" {
		return
	}
	r.mu.Lock()
	r.m[key] = cancel
	r.mu.Unlock()
}

func (r *cancelRegistry) remove(key string) {
	r.mu.Lock()
	delete(r.m, key)
	r.mu.Unlock()
}

func (r *cancelRegistry) cancel(key string) bool {
	r.mu.Lock()
	cancel, ok := r.m[key]
	r.mu.Unlock()
	if ok {
		cancel()
	}
	return ok
}

// cancelRequest handles notifications/cancelled: {"requestId": 12}.
func (s *Server) cancelRequest(params json.RawMessage) {
	var p struct {
		RequestID json.RawMessage `json:"requestId"`
		Reason    string          `json:"reason"`
	}
	if err := json.Unmarshal(params, &p); err != nil {
		return
	}
	key := idKey(p.RequestID)
	if s.cancels.cancel(key) {
		s.log.Printf("请求 %s 已被客户端取消", key)
	} else {
		s.log.Printf("取消请求 %s：没有在执行的调用", key)
	}
}

// idKey normalises a JSON-RPC id for the cancel map.
func idKey(id json.RawMessage) string {
	return strings.TrimSpace(string(id))
}
