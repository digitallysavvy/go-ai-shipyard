package main

import (
	"testing"

	"github.com/digitallysavvy/go-ai/pkg/provider"
	"github.com/digitallysavvy/go-ai/pkg/provider/types"
)

func TestActivityLogStripsWorkDirAcrossDeltas(t *testing.T) {
	state := &coderState{Events: []coderEvent{}}
	log := newActivityLog(state, "/tmp/sbx/work/codex-1")
	for _, delta := range []string{"Fixed [shortlink.go](/tmp/sbx/wo", "rk/codex-1/shortlink.go:54) so ", "links expire."} {
		log.apply(&provider.StreamChunk{Type: provider.ChunkTypeText, Text: delta})
	}
	if got, want := state.Events[0].Text, "Fixed shortlink.go so links expire."; got != want {
		t.Fatalf("text = %q, want %q", got, want)
	}
	if got := log.finalMessage(); got != "Fixed shortlink.go so links expire." {
		t.Fatalf("finalMessage = %q", got)
	}
}

func TestActivityLogRecordsCommandResults(t *testing.T) {
	state := &coderState{Events: []coderEvent{}}
	log := newActivityLog(state, "/w")
	log.apply(&provider.StreamChunk{Type: provider.ChunkTypeToolCall, ToolCall: &types.ToolCall{
		ID: "c1", ToolName: "bash", Arguments: map[string]interface{}{"command": "/bin/zsh -lc 'go test ./...'"},
	}})
	log.apply(&provider.StreamChunk{Type: provider.ChunkTypeToolResult, ToolResult: &types.ToolResult{
		ToolCallID: "c1", ToolName: "bash", Result: map[string]interface{}{"exitCode": float64(1), "output": "--- FAIL: TestX\nFAIL"},
	}})
	e := state.Events[0]
	if e.Kind != "run" || e.Text != "go test ./..." {
		t.Fatalf("event = %+v", e)
	}
	if e.Exit == nil || *e.Exit != 1 || e.Output != "--- FAIL: TestX\nFAIL" {
		t.Fatalf("result not recorded: %+v", e)
	}
}

func TestDescribeToolCall(t *testing.T) {
	cases := []struct {
		name string
		args map[string]interface{}
		kind string
		text string
	}{
		{"Bash", map[string]interface{}{"command": "go vet ./..."}, "run", "go vet ./..."},
		{"Edit", map[string]interface{}{"file_path": "shortlink.go"}, "edit", "shortlink.go"},
		{"fileChange", map[string]interface{}{"path": "shortlink.go"}, "edit", "shortlink.go"},
		{"Read", map[string]interface{}{"file_path": "go.mod"}, "read", "go.mod"},
		{"Grep", map[string]interface{}{"pattern": "ExpiresAt"}, "tool", "Grep ExpiresAt"},
		{"apply_patch", map[string]interface{}{}, "", ""},
	}
	for _, c := range cases {
		kind, text := describeToolCall(c.name, c.args)
		if kind != c.kind || text != c.text {
			t.Errorf("describeToolCall(%s) = %q, %q; want %q, %q", c.name, kind, text, c.kind, c.text)
		}
	}
}

func TestCommandResultShapes(t *testing.T) {
	cases := []struct {
		name   string
		result interface{}
		failed bool
		exit   int
		output string
	}{
		{"codex ok", map[string]interface{}{"exitCode": float64(0), "output": "ok  example.com/x"}, false, 0, "ok  example.com/x"},
		{"codex failure", map[string]interface{}{"exitCode": float64(1), "output": "FAIL"}, false, 1, "FAIL"},
		{"claude code ok", map[string]interface{}{"stdout": "ok  example.com/x\n", "stderr": "", "interrupted": false}, false, 0, "ok  example.com/x"},
		{"claude code failure", map[string]interface{}{"stdout": "--- FAIL: TestX\n", "stderr": "exit status 1"}, true, 1, "--- FAIL: TestX\nexit status 1"},
		{"claude code interrupted", map[string]interface{}{"stdout": "", "stderr": "", "interrupted": true}, false, 1, ""},
		{"plain string", "done", false, 0, "done"},
	}
	for _, c := range cases {
		exit, out := commandResult(c.result, c.failed)
		if exit != c.exit || out != c.output {
			t.Errorf("%s: commandResult = %d, %q; want %d, %q", c.name, exit, out, c.exit, c.output)
		}
	}
}
