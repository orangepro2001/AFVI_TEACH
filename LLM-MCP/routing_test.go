package main

import (
	"encoding/json"
	"io"
	"log"
	"strings"
	"testing"
)

// routingServer has a cheap default and an expensive high-effort target, the
// shape the deployment uses (deepseek by default, gpt for hard tasks).
func routingServer(t *testing.T, u *upstream) (*Server, *Config) {
	t.Helper()
	cfg, err := ParseConfig([]byte(`{
	  "default": "deepseek",
	  "providers": [
	    {"name": "deepseek", "base_url": "` + u.URL + `/v1", "api_key": "k1", "model": "ds-medium",
	     "cost": "low", "note": "默认：中等任务用它"},
	    {"name": "gpt", "base_url": "` + u.URL + `/v1", "api_key": "k2", "model": "gpt-astra",
	     "models": {"hard": "gpt-astra"}, "params": {"reasoning_effort": "max"},
	     "cost": "high", "note": "可靠但高价：复杂任务用它"}
	  ]
	}`))
	if err != nil {
		t.Fatalf("ParseConfig: %v", err)
	}
	return NewServer(cfg, NewClient(0, 1), log.New(io.Discard, "", 0)), cfg
}

func TestInstructionsExplainRouting(t *testing.T) {
	s, _ := routingServer(t, newUpstream(t, "x"))
	resp, _ := rpc(t, s, `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18"}}`)
	result, _ := resp["result"].(map[string]any)
	instr, _ := result["instructions"].(string)

	for _, want := range []string{
		"Model routing",
		"默认：中等任务用它",
		"可靠但高价：复杂任务用它",
		"cost: high",
		"pass model=\"gpt\"",
		"much more expensive",
		"(default: used when `model` is omitted)",
	} {
		if !strings.Contains(instr, want) {
			t.Errorf("instructions missing %q:\n%s", want, instr)
		}
	}
	if strings.Index(instr, "deepseek") > strings.Index(instr, "pass model=\"gpt\"") {
		t.Errorf("the default target should be listed before the expensive hint:\n%s", instr)
	}
}

func TestToolDescriptionsExplainRouting(t *testing.T) {
	s, _ := routingServer(t, newUpstream(t, "x"))
	resp, _ := rpc(t, s, `{"jsonrpc":"2.0","id":2,"method":"tools/list"}`)
	tools := resp["result"].(map[string]any)["tools"].([]any)
	for _, raw := range tools {
		tool := raw.(map[string]any)
		switch tool["name"] {
		case "ask", "chat":
			desc, _ := tool["description"].(string)
			for _, want := range []string{"Model routing", `model=\"gpt\"`, "中等任务"} {
				if !strings.Contains(desc, strings.ReplaceAll(want, `\"`, `"`)) {
					t.Errorf("%s description missing %q:\n%s", tool["name"], want, desc)
				}
			}
			props := tool["inputSchema"].(map[string]any)["properties"].(map[string]any)
			modelProp, _ := props["model"].(map[string]any)
			if desc, _ := modelProp["description"].(string); !strings.Contains(desc, "expensive") {
				t.Errorf("%s model param should mention the expensive target: %s", tool["name"], desc)
			}
		}
	}
}

func TestListModelsShowsCostAndNote(t *testing.T) {
	s, _ := routingServer(t, newUpstream(t, "x"))
	text, isErr := callTool(t, s, "list_models", `{}`)
	if isErr {
		t.Fatalf("list_models failed: %s", text)
	}
	var out struct {
		Providers []Describe `json:"providers"`
	}
	if err := json.Unmarshal([]byte(text), &out); err != nil {
		t.Fatalf("not JSON: %v\n%s", err, text)
	}
	if len(out.Providers) != 2 {
		t.Fatalf("providers = %d", len(out.Providers))
	}
	if out.Providers[0].Name != "deepseek" || !out.Providers[0].Default || out.Providers[0].Cost != "low" {
		t.Errorf("first provider should be the default cheap one: %+v", out.Providers[0])
	}
	gpt := out.Providers[1]
	if gpt.Cost != "high" || !strings.Contains(gpt.Note, "复杂任务") {
		t.Errorf("gpt entry = %+v", gpt)
	}
	if gpt.Params["reasoning_effort"] != "max" {
		t.Errorf("reasoning_effort not surfaced: %+v", gpt.Params)
	}
}

func TestExpensiveTargetSendsEffortAndItsOwnKey(t *testing.T) {
	u := newUpstream(t, "深思熟虑的回答")
	s, _ := routingServer(t, u)

	text, isErr := callTool(t, s, "ask", `{"prompt":"一个很难的问题","model":"gpt"}`)
	if isErr {
		t.Fatalf("ask failed: %s", text)
	}
	if text != "深思熟虑的回答" {
		t.Errorf("text = %q", text)
	}
	body := u.last().Body
	if body["model"] != "gpt-astra" {
		t.Errorf("model = %v, want gpt-astra", body["model"])
	}
	if body["reasoning_effort"] != "max" {
		t.Errorf("reasoning_effort = %v, want max", body["reasoning_effort"])
	}
	if got := u.last().Auth; got != "Bearer k2" {
		t.Errorf("Authorization = %q, want the gpt key", got)
	}

	// The default target must not inherit the expensive effort setting.
	if _, isErr := callTool(t, s, "ask", `{"prompt":"简单问题"}`); isErr {
		t.Fatal("default ask failed")
	}
	body = u.last().Body
	if body["model"] != "ds-medium" {
		t.Errorf("model = %v, want ds-medium", body["model"])
	}
	if _, ok := body["reasoning_effort"]; ok {
		t.Errorf("reasoning_effort leaked into the default provider: %v", body["reasoning_effort"])
	}
	if got := u.last().Auth; got != "Bearer k1" {
		t.Errorf("Authorization = %q, want the deepseek key", got)
	}

	// An alias of the expensive provider works the same way.
	if _, isErr := callTool(t, s, "ask", `{"prompt":"x","model":"hard"}`); isErr {
		t.Fatal("alias ask failed")
	}
	if u.last().Body["model"] != "gpt-astra" {
		t.Errorf("alias model = %v", u.last().Body["model"])
	}
}
