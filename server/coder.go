package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/digitallysavvy/go-ai/pkg/agent"
	"github.com/digitallysavvy/go-ai/pkg/ai"
	"github.com/digitallysavvy/go-ai/pkg/harness"
	"github.com/digitallysavvy/go-ai/pkg/harness/claudecode"
	"github.com/digitallysavvy/go-ai/pkg/harness/codex"
	"github.com/digitallysavvy/go-ai/pkg/harness/sandbox/local"
	"github.com/digitallysavvy/go-ai/pkg/provider"
	"github.com/digitallysavvy/go-ai/pkg/providerutils"
)

// coderEvent is one line in the coding agent's live activity log.
type coderEvent struct {
	ID     string `json:"id"`
	Kind   string `json:"kind"` // say | run | read | edit | tool
	Text   string `json:"text"`
	Exit   *int   `json:"exit,omitempty"`
	Output string `json:"output,omitempty"`
}

// coderState is streamed to the browser as a `data-coder` UI message part.
// Every update reuses the tool call ID as the part ID, so useChat replaces
// the part in place instead of appending a new one.
type coderState struct {
	Agent        string       `json:"agent"`
	Status       string       `json:"status"` // starting | running | done | error
	Events       []coderEvent `json:"events"`
	FilesChanged []string     `json:"filesChanged,omitempty"`
	Diff         string       `json:"diff,omitempty"`
	Summary      string       `json:"summary,omitempty"`
	Error        string       `json:"error,omitempty"`
	ElapsedMs    int64        `json:"elapsedMs"`
}

// coderResult is what the chat model sees as the tool output.
type coderResult struct {
	Agent        string   `json:"agent"`
	Summary      string   `json:"summary"`
	FilesChanged []string `json:"filesChanged"`
	Diff         string   `json:"diff"`
}

type coder struct {
	cfg config
	ws  *workspace
	mu  sync.Mutex // one coding session at a time: the bridge port is fixed
}

func (c *coder) newHarnessAgent(kind agentKind, onSession func(context.Context, harness.SandboxSessionContext) error) (*harness.Agent, error) {
	settings := harness.AgentSettings{
		PermissionMode: harness.PermissionModeAllowAll,
		SandboxConfig:  harness.AgentSandboxConfig{OnSession: onSession},
		Instructions: "You are fixing a small Go module. Keep changes minimal, never edit *_test.go files, " +
			"and run `go test ./...` before you finish. End with one or two sentences describing the fix.",
	}
	sandboxRoot := filepath.Join(c.cfg.DataDir, "sandboxes")
	switch kind {
	case agentClaude:
		if c.cfg.AnthropicKey == "" {
			return nil, errors.New("ANTHROPIC_API_KEY is not set, so Claude Code can't run")
		}
		cc, err := claudecode.New(claudecode.Settings{MaxTurns: 30})
		if err != nil {
			return nil, err
		}
		settings.Harness = cc
		settings.Model = c.cfg.ClaudeCodeModel
		settings.Sandbox = local.NewProvider(local.Options{RootDir: sandboxRoot, Ports: []int{4319}})
	case agentCodex:
		if c.cfg.OpenAIKey == "" {
			return nil, errors.New("OPENAI_API_KEY is not set, so Codex can't run")
		}
		settings.Harness = codex.New(codex.Settings{})
		settings.Model = os.Getenv("CODEX_MODEL")
		settings.Sandbox = local.NewProvider(local.Options{RootDir: sandboxRoot, Ports: []int{4318}})
	default:
		return nil, fmt.Errorf("unknown coding agent %q", kind)
	}
	return harness.NewAgent(settings)
}

// Run hands task to Claude Code or Codex, streams its activity into the chat
// as a data part, and syncs the resulting file changes back into the
// workspace.
func (c *coder) Run(ctx context.Context, kind agentKind, task, toolCallID string, writer ai.UIMessageStreamWriter) (coderResult, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	start := time.Now()
	state := &coderState{Agent: kind.CoderName(), Status: "starting", Events: []coderEvent{}}
	var lastEmit time.Time
	emit := func(force bool) {
		if !force && time.Since(lastEmit) < 120*time.Millisecond {
			return
		}
		lastEmit = time.Now()
		state.ElapsedMs = time.Since(start).Milliseconds()
		snapshot := *state
		snapshot.Events = make([]coderEvent, len(state.Events)) // never nil: JSON null would break the client
		copy(snapshot.Events, state.Events)
		writer.Write(ai.UIMessageChunk{"type": "data-coder", "id": toolCallID, "data": snapshot})
	}
	fail := func(err error) (coderResult, error) {
		state.Status, state.Error = "error", err.Error()
		emit(true)
		return coderResult{}, err
	}
	emit(true)

	before, err := c.ws.Files()
	if err != nil {
		return fail(err)
	}

	// Seed the sandbox with the current workspace before the agent starts.
	seed := func(ctx context.Context, sc harness.SandboxSessionContext) error {
		for _, rel := range sortedKeys(before) {
			if err := sc.Session.WriteTextFile(ctx, providerutils.SandboxWriteTextFileOptions{
				Path:    path.Join(sc.SessionWorkDir, rel),
				Content: before[rel],
			}); err != nil {
				return err
			}
		}
		// A baseline commit lets the coding agent use git status/diff as usual.
		res, err := sc.Session.Run(ctx, providerutils.SandboxProcessOptions{
			Command: "git init -q && git add -A && " +
				"git -c user.name=shipyard -c user.email=shipyard@localhost commit -qm baseline",
			WorkingDirectory: sc.SessionWorkDir,
		})
		if err != nil {
			return err
		}
		if res.ExitCode != 0 {
			return fmt.Errorf("git init in sandbox: %s", res.Stderr)
		}
		return nil
	}

	coding, err := c.newHarnessAgent(kind, seed)
	if err != nil {
		return fail(err)
	}
	session, err := coding.CreateSession(ctx, harness.CreateSessionOptions{SessionID: "demo-" + newShortID()})
	if err != nil {
		return fail(err)
	}
	defer func() { _ = session.Destroy(context.WithoutCancel(ctx)) }()
	workDir := session.GetSessionWorkDir()

	result, err := coding.Stream(ctx, agent.AgentStreamOptions{AgentGenerateOptions: agent.AgentGenerateOptions{
		HarnessSession: session,
		Prompt:         task,
	}})
	if err != nil {
		return fail(err)
	}

	state.Status = "running"
	emit(true)
	log := newActivityLog(state, workDir)
	stream := result.FullStream()
	for {
		chunk, err := stream.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return fail(err)
		}
		if log.apply(chunk) {
			emit(chunk.Type != provider.ChunkTypeText)
		}
	}
	if err := stream.Err(); err != nil {
		return fail(err)
	}

	changed, err := c.collectChanges(ctx, session.GetSandboxSession(), workDir, before)
	if err != nil {
		return fail(fmt.Errorf("sync changes: %w", err))
	}
	if err := c.ws.Apply(changed); err != nil {
		return fail(fmt.Errorf("apply changes: %w", err))
	}

	state.Status = "done"
	state.FilesChanged = sortedKeys(changed)
	state.Diff = unifiedDiff(before, changed)
	state.Summary = log.finalMessage()
	emit(true)

	return coderResult{
		Agent:        state.Agent,
		Summary:      state.Summary,
		FilesChanged: state.FilesChanged,
		Diff:         state.Diff,
	}, nil
}

// collectChanges reads every text file back out of the sandbox and returns
// the ones that differ from the workspace snapshot taken before the run.
func (c *coder) collectChanges(ctx context.Context, sb providerutils.SandboxSession, workDir string, before map[string]string) (map[string]string, error) {
	res, err := sb.Run(ctx, providerutils.SandboxProcessOptions{
		Command:          `find . -type f -not -path '*/.*' -size -256k`,
		WorkingDirectory: workDir,
	})
	if err != nil {
		return nil, err
	}
	if res.ExitCode != 0 {
		return nil, fmt.Errorf("list files: %s", res.Stderr)
	}
	changed := map[string]string{}
	for _, line := range strings.Split(strings.TrimSpace(res.Stdout), "\n") {
		rel := strings.TrimPrefix(strings.TrimSpace(line), "./")
		if rel == "" {
			continue
		}
		content, err := sb.ReadTextFile(ctx, providerutils.SandboxReadTextFileOptions{Path: path.Join(workDir, rel)})
		if err != nil || content == nil {
			continue
		}
		if old, ok := before[rel]; !ok || old != *content {
			changed[rel] = *content
		}
	}
	return changed, nil
}

// activityLog turns harness stream chunks into a short, readable log.
type activityLog struct {
	state   *coderState
	workDir string
	byCall  map[string]int // tool call ID -> event index
	textIdx int            // index of the event receiving text deltas, or -1
	rawText string         // unstripped text of that event; paths can span deltas
}

func newActivityLog(state *coderState, workDir string) *activityLog {
	return &activityLog{state: state, workDir: workDir, byCall: map[string]int{}, textIdx: -1}
}

// apply folds one chunk into the log and reports whether anything changed.
func (l *activityLog) apply(chunk *provider.StreamChunk) bool {
	switch chunk.Type {
	case provider.ChunkTypeText:
		if chunk.Text == "" {
			return false
		}
		if l.textIdx < 0 {
			l.state.Events = append(l.state.Events, coderEvent{ID: fmt.Sprintf("say-%d", len(l.state.Events)), Kind: "say"})
			l.textIdx = len(l.state.Events) - 1
			l.rawText = ""
		}
		l.rawText += chunk.Text
		l.state.Events[l.textIdx].Text = markdownLink.ReplaceAllString(l.strip(l.rawText), "$1")
		return true
	case provider.ChunkTypeTextEnd:
		l.textIdx = -1
		return false
	case provider.ChunkTypeToolCall:
		l.textIdx = -1
		if chunk.ToolCall == nil {
			return false
		}
		kind, text := describeToolCall(chunk.ToolCall.ToolName, chunk.ToolCall.Arguments)
		if text == "" {
			return false
		}
		l.state.Events = append(l.state.Events, coderEvent{ID: chunk.ToolCall.ID, Kind: kind, Text: l.strip(text)})
		l.byCall[chunk.ToolCall.ID] = len(l.state.Events) - 1
		return true
	case provider.ChunkTypeToolResult:
		if chunk.ToolResult == nil {
			return false
		}
		idx, ok := l.byCall[chunk.ToolResult.ToolCallID]
		if !ok || l.state.Events[idx].Kind != "run" {
			return false
		}
		code, out := commandResult(chunk.ToolResult.Result, chunk.ToolResult.Error != nil)
		l.state.Events[idx].Exit = &code
		l.state.Events[idx].Output = lastLines(l.strip(out), 8)
		return true
	}
	return false
}

// commandResult reads a shell command's exit status and output from a
// harness tool result. Codex reports {exitCode, output}; Claude Code passes
// through the Agent SDK's Bash result {stdout, stderr, interrupted} and marks
// failures on the result instead of with an exit code.
func commandResult(result interface{}, failed bool) (exit int, output string) {
	exit = 0
	if failed {
		exit = 1
	}
	switch r := result.(type) {
	case map[string]interface{}:
		if code, ok := toInt(r["exitCode"]); ok {
			exit = code
		}
		if out, ok := r["output"].(string); ok {
			return exit, out
		}
		stdout, _ := r["stdout"].(string)
		stderr, _ := r["stderr"].(string)
		if interrupted, _ := r["interrupted"].(bool); interrupted && exit == 0 {
			exit = 1
		}
		return exit, strings.TrimSpace(strings.TrimSpace(stdout) + "\n" + strings.TrimSpace(stderr))
	case string:
		return exit, r
	}
	return exit, ""
}

func (l *activityLog) finalMessage() string {
	for i := len(l.state.Events) - 1; i >= 0; i-- {
		if e := l.state.Events[i]; e.Kind == "say" && strings.TrimSpace(e.Text) != "" {
			return strings.TrimSpace(e.Text)
		}
	}
	return "The coding agent finished without a summary."
}

// strip removes the sandbox work dir so paths read as repo-relative.
func (l *activityLog) strip(s string) string {
	if l.workDir == "" {
		return s
	}
	s = strings.ReplaceAll(s, l.workDir+"/", "")
	return strings.ReplaceAll(s, l.workDir, ".")
}

// markdownLink matches [text](target); the log shows just the text.
var markdownLink = regexp.MustCompile(`\[([^\]]+)\]\([^)]*\)`)

var shellWrapper = regexp.MustCompile(`^/bin/\w+ -l?c (?:'(.*)'|"(.*)")$`)

func describeToolCall(name string, args map[string]interface{}) (kind, text string) {
	str := func(keys ...string) string {
		for _, k := range keys {
			if v, ok := args[k].(string); ok && v != "" {
				return v
			}
		}
		return ""
	}
	lower := strings.ToLower(name)
	switch {
	case strings.Contains(lower, "bash") || lower == "shell":
		cmd := str("command", "cmd")
		if m := shellWrapper.FindStringSubmatch(cmd); m != nil {
			cmd = m[1] + m[2]
		}
		return "run", cmd
	case strings.Contains(lower, "edit") || strings.Contains(lower, "write") || lower == "filechange":
		return "edit", str("file_path", "path")
	case strings.Contains(lower, "read"):
		return "read", str("file_path", "path")
	case lower == "apply_patch":
		return "", "" // reported by the fileChange call that follows it
	default:
		if detail := str("pattern", "query", "path", "file_path"); detail != "" {
			return "tool", name + " " + detail
		}
		return "tool", name
	}
}

func unifiedDiff(before, changed map[string]string) string {
	if len(changed) == 0 {
		return ""
	}
	dir, err := os.MkdirTemp("", "go-ai-demo-diff-")
	if err != nil {
		return ""
	}
	defer os.RemoveAll(dir)

	var out strings.Builder
	for _, rel := range sortedKeys(changed) {
		oldPath, newPath := filepath.Join(dir, "old"), filepath.Join(dir, "new")
		_ = os.WriteFile(oldPath, []byte(before[rel]), 0o600)
		_ = os.WriteFile(newPath, []byte(changed[rel]), 0o600)
		// diff exits 1 when the files differ; only the output matters here.
		b, _ := exec.Command("diff", "-u", "--label", "a/"+rel, "--label", "b/"+rel, oldPath, newPath).Output()
		out.Write(b)
	}
	return out.String()
}

func lastLines(s string, n int) string {
	lines := strings.Split(strings.TrimRight(s, "\n"), "\n")
	if len(lines) > n {
		lines = append([]string{"…"}, lines[len(lines)-n:]...)
	}
	return strings.Join(lines, "\n")
}

func toInt(v interface{}) (int, bool) {
	switch n := v.(type) {
	case int:
		return n, true
	case float64:
		return int(n), true
	case int64:
		return int(n), true
	}
	return 0, false
}
