// Command server is the Go backend for the go-ai demo.
//
// It serves the AI SDK UI message stream protocol on POST /api/chat, so the
// stock useChat hook from @ai-sdk/react talks to it with no adapter code.
package main

import (
	"context"
	"crypto/rand"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"
)

func main() {
	root, err := repoRoot()
	if err != nil {
		log.Fatal(err)
	}
	loadDotEnv(filepath.Join(root, ".env"))
	cfg := loadConfig(root)

	ws, err := newWorkspace(filepath.Join(root, "workspace-template"), filepath.Join(cfg.DataDir, "workspace"))
	if err != nil {
		log.Fatalf("workspace: %v", err)
	}
	if err := ws.Reset(); err != nil {
		log.Fatalf("workspace reset: %v", err)
	}

	srv := &server{cfg: cfg, ws: ws, coder: &coder{cfg: cfg, ws: ws}}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/chat", srv.handleChat)
	mux.HandleFunc("POST /api/reset", srv.handleReset)
	mux.HandleFunc("GET /api/status", srv.handleStatus)

	httpServer := &http.Server{
		Addr:              cfg.Addr,
		Handler:           withCORS(cfg.WebOrigin, mux),
		ReadHeaderTimeout: 10 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = httpServer.Shutdown(shutdownCtx)
	}()

	log.Printf("go-ai demo server listening on http://localhost%s (web origin %s)", cfg.Addr, cfg.WebOrigin)
	log.Printf("chat agents available: %v", cfg.availableAgents())
	if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}

type config struct {
	Addr            string
	WebOrigin       string
	DataDir         string
	AnthropicKey    string
	OpenAIKey       string
	ClaudeChatModel string
	OpenAIChatModel string
	ClaudeCodeModel string
	CodexModel      string // empty uses the Codex default
	ApprovalSecret  []byte
}

func loadConfig(root string) config {
	cfg := config{
		Addr:            ":" + envOr("PORT", "8080"),
		WebOrigin:       envOr("WEB_ORIGIN", "http://localhost:3000"),
		DataDir:         envOr("DEMO_DATA_DIR", filepath.Join(root, ".data")),
		AnthropicKey:    os.Getenv("ANTHROPIC_API_KEY"),
		OpenAIKey:       os.Getenv("OPENAI_API_KEY"),
		ClaudeChatModel: envOr("CLAUDE_CHAT_MODEL", "claude-sonnet-5-5"),
		OpenAIChatModel: envOr("OPENAI_CHAT_MODEL", "gpt-6-astra"),
		ClaudeCodeModel: envOr("CLAUDE_CODE_MODEL", "claude-sonnet-5-5"),
		CodexModel:      os.Getenv("CODEX_MODEL"),
	}
	// Approval requests are HMAC-signed so a client can't forge an approval
	// for a call the server never issued. A random per-process secret is fine
	// for a demo; set TOOL_APPROVAL_SECRET to keep approvals valid across
	// restarts.
	if s := os.Getenv("TOOL_APPROVAL_SECRET"); s != "" {
		cfg.ApprovalSecret = []byte(s)
	} else {
		cfg.ApprovalSecret = make([]byte, 32)
		_, _ = rand.Read(cfg.ApprovalSecret)
	}
	return cfg
}

func (c config) availableAgents() []string {
	var out []string
	if c.AnthropicKey != "" {
		out = append(out, string(agentClaude))
	}
	if c.OpenAIKey != "" {
		out = append(out, string(agentCodex))
	}
	if len(out) == 0 {
		out = append(out, "none (set ANTHROPIC_API_KEY and/or OPENAI_API_KEY in .env)")
	}
	return out
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// repoRoot finds the directory holding workspace-template/ so the server works whether it
// is started from the repo root (`go run ./server`) or from server/.
func repoRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "workspace-template")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", errors.New("run the server from inside the go-ai-shipyard repo")
		}
		dir = parent
	}
}

func withCORS(origin string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
