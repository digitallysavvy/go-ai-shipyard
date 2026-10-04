package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/digitallysavvy/go-ai/pkg/agent"
	"github.com/digitallysavvy/go-ai/pkg/ai"
	"github.com/digitallysavvy/go-ai/pkg/provider"
	"github.com/digitallysavvy/go-ai/pkg/providers/anthropic"
	"github.com/digitallysavvy/go-ai/pkg/providers/openai"
)

// agentKind is the pairing the UI toggles between: the chat model's provider
// and the coding agent it delegates to.
type agentKind string

const (
	agentClaude agentKind = "claude" // Anthropic chat model + Claude Code
	agentCodex  agentKind = "codex"  // OpenAI chat model + Codex
)

// CoderName is the coding agent's display name.
func (k agentKind) CoderName() string {
	switch k {
	case agentClaude:
		return "Claude Code"
	case agentCodex:
		return "Codex"
	}
	return string(k)
}

// apiKeyEnv is the environment variable that enables the pairing.
func (k agentKind) apiKeyEnv() string {
	switch k {
	case agentClaude:
		return "ANTHROPIC_API_KEY"
	case agentCodex:
		return "OPENAI_API_KEY"
	}
	return ""
}

const systemPrompt = `You are Shipyard, an engineering assistant. You run on a Go backend built with the Go AI SDK.

The user's project is a small Go module in your workspace. Work like this:
1. Investigate with run_tests, list_files and read_file. Be quick: one or two calls.
2. You never edit code yourself. When a change is needed, call delegate_to_coding_agent once with a precise task: the failing behavior, the file, and the constraint that tests must not change.
3. After the coding agent finishes, call run_tests to verify.
4. Reply in two or three short sentences: what was wrong, what changed, and that tests pass.

Keep every message brief; the user is watching the work happen live.`

type server struct {
	cfg   config
	ws    *workspace
	coder *coder
}

type chatRequest struct {
	ID       string          `json:"id"`
	Messages json.RawMessage `json:"messages"`
	Agent    agentKind       `json:"agent"`
}

func (s *server) handleChat(w http.ResponseWriter, r *http.Request) {
	var req chatRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 8<<20)).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.Agent == "" {
		req.Agent = agentClaude
	}
	model, err := s.chatModel(req.Agent)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	chunks, _ := ai.CreateUIMessageStreamWithOptions(ctx, ai.UIMessageStreamOptions{
		// Surface real error text in the UI; this is a local demo.
		OnError: func(err error) string {
			log.Printf("chat error: %v", err)
			return err.Error()
		},
		Execute: func(writer ai.UIMessageStreamWriter) {
			shipyard := agent.NewToolLoopAgent(agent.AgentConfig{
				Model:                          model,
				System:                         systemPrompt,
				Tools:                          s.tools(req.Agent, writer),
				StopWhen:                       []ai.StopCondition{ai.IsStepCount(12)},
				ExperimentalToolApprovalSecret: s.cfg.ApprovalSecret,
			})

			// Validates the browser's UI messages against the agent's tools,
			// converts them to model messages (including any approval the
			// user just gave) and streams the agent's reply back as UI chunks.
			stream, errs, err := agent.CreateAgentUIStreamFromUIMessages(ctx, shipyard, agent.CreateAgentUIStreamFromUIMessagesOptions{
				UIMessages: []byte(req.Messages),
			})
			if err != nil {
				writer.Write(ai.UIMessageChunk{"type": "error", "errorText": err.Error()})
				return
			}
			writer.Merge(stream)
			go func() {
				for err := range errs {
					if err != nil {
						log.Printf("agent stream: %v", err)
					}
				}
			}()
		},
	})

	keepAlive := 15 * time.Second // holds the connection open through quiet stretches of a coding run
	if err := ai.PipeUIMessageChunksToResponse(chunks, w, &ai.UIMessageStreamResponseInit{KeepAliveMs: &keepAlive}); err != nil {
		log.Printf("write stream: %v", err)
	}
}

func (s *server) chatModel(kind agentKind) (provider.LanguageModel, error) {
	switch kind {
	case agentClaude:
		if s.cfg.AnthropicKey == "" {
			return nil, fmt.Errorf("set %s in .env and restart the server to use %s", kind.apiKeyEnv(), kind.CoderName())
		}
		return anthropic.New(anthropic.Config{APIKey: s.cfg.AnthropicKey}).LanguageModel(s.cfg.ClaudeChatModel)
	case agentCodex:
		if s.cfg.OpenAIKey == "" {
			return nil, fmt.Errorf("set %s in .env and restart the server to use %s", kind.apiKeyEnv(), kind.CoderName())
		}
		return openai.New(openai.Config{APIKey: s.cfg.OpenAIKey}).LanguageModel(s.cfg.OpenAIChatModel)
	}
	return nil, fmt.Errorf("unknown agent %q (want %q or %q)", kind, agentClaude, agentCodex)
}

func (s *server) handleReset(w http.ResponseWriter, r *http.Request) {
	if err := s.ws.Reset(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *server) handleStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"agents": map[string]interface{}{
			"claude": map[string]interface{}{"available": s.cfg.AnthropicKey != "", "chatModel": s.cfg.ClaudeChatModel, "coder": agentClaude.CoderName(), "apiKeyEnv": agentClaude.apiKeyEnv()},
			"codex":  map[string]interface{}{"available": s.cfg.OpenAIKey != "", "chatModel": s.cfg.OpenAIChatModel, "coder": agentCodex.CoderName(), "apiKeyEnv": agentCodex.apiKeyEnv()},
		},
	})
}

func newShortID() string {
	b := make([]byte, 6)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
