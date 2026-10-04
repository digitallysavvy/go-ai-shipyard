# Server

**Summary.** `server/` is one Go package (`package main`), about 1,000 lines, one concern per file. `main.go` wires config and routes. `chat.go` holds the endpoint and the agent. `tools.go` defines the tools. `coder.go` runs the coding agent. `workspace.go` owns the project on disk.

| File | Key symbols | Read it to |
| --- | --- | --- |
| `server/main.go` | `main`, `config`, `loadConfig`, `repoRoot`, `withCORS` | change routes, ports, env vars or CORS |
| `server/chat.go` | `handleChat`, `chatModel`, `systemPrompt`, `agentKind`, `chatRequest`, `handleReset`, `handleStatus` | change the agent, its models or prompt, or the request body |
| `server/tools.go` | `tools`, `objectSchema` | add or change a tool |
| `server/coder.go` | `coder`, `newHarnessAgent`, `Run`, `collectChanges`, `activityLog`, `commandResult`, `describeToolCall`, `unifiedDiff` | change how Claude Code or Codex runs or how their activity is shown |
| `server/workspace.go` | `workspace`, `Reset`, `Files`, `ReadFile`, `Apply`, `RunTests`, `resolve` | change where the project lives or how tests run |
| `server/dotenv.go` | `loadDotEnv` | change `.env` loading |

## main.go

`main` finds the repo root (`repoRoot` walks up to the directory holding `workspace-template/`), loads `.env`, builds a `config` with `loadConfig`, resets the workspace and serves three routes:

| Route | Handler | Does |
| --- | --- | --- |
| `POST /api/chat` | `handleChat` | Streams the agent's reply. See below. |
| `POST /api/reset` | `handleReset` | Restores the workspace to the failing template |
| `GET /api/status` | `handleStatus` | Which pairings have a key, plus model names, for the UI toggle |

Environment variables are read only in `loadConfig`. The full list is in [`.env.example`](../.env.example). If `TOOL_APPROVAL_SECRET` is unset, `loadConfig` generates a random approval secret per process.

## chat.go

`handleChat` is the core of the app:

1. Decode `chatRequest` (`id`, raw `messages`, `agent`), limited to 8 MB.
2. `chatModel` returns the Anthropic or OpenAI model for the `agentKind`, or a 400 if its key is missing.
3. `ai.CreateUIMessageStreamWithOptions` gets an `Execute` func that builds the `ToolLoopAgent` (model, `systemPrompt`, `s.tools(kind, writer)`, `ai.IsStepCount(12)`, `ExperimentalToolApprovalSecret`), starts it with `agent.CreateAgentUIStreamFromUIMessages`, and merges the stream into the writer.
4. `ai.PipeUIMessageChunksToResponse` writes the headers and the SSE body, with a 15-second keep-alive.

`OnError` returns the real error text so failures show in the UI. That's fine for a local demo; return a generic message in production.

`agentKind` is the pairing (`claude` or `codex`). `CoderName` gives its display name.

## tools.go

`tools` builds the tool list per request so `delegate_to_coding_agent` can capture that request's stream writer. Parameters are JSON Schema maps built with `objectSchema`, which disallows extra properties.

| Tool | Approval | Calls |
| --- | --- | --- |
| `run_tests` | no | `workspace.RunTests` |
| `list_files` | no | `workspace.Files` |
| `read_file` | no | `workspace.ReadFile` |
| `delegate_to_coding_agent` | **yes** (`ToolApproval: true`) | `coder.Run` with the tool call ID and writer |

To add one, see [extending.md](extending.md#add-a-tool).

## coder.go

`coder.Run(ctx, kind, task, toolCallID, writer)` is everything that happens after the user approves:

1. Snapshot the workspace (`workspace.Files`).
2. `newHarnessAgent` builds a `harness.Agent` for the pairing: `claudecode.New` or `codex.New`, the local sandbox provider with a fixed bridge port (4319 for Claude Code, 4318 for Codex), `PermissionModeAllowAll`, and an `OnSession` hook.
3. `CreateSession` with an explicit session ID. `OnSession` writes the snapshot into the sandbox and commits a git baseline so the agent can use `git diff`.
4. `Stream` the task and read `FullStream()`. `activityLog.apply` folds each chunk into a short log: text becomes `say` events, tool calls become `run`, `read`, `edit` or `tool` events (`describeToolCall`), and command results fill in exit codes and output (`commandResult`).
5. After each change, emit the `coderState` as a `data-coder` part, throttled to about 8 updates a second.
6. `collectChanges` lists and reads the sandbox's files and keeps the ones that differ from the snapshot. `workspace.Apply` writes them back, and `unifiedDiff` renders the diff with the system `diff` tool.
7. Return a `coderResult` (agent, summary, files changed, diff), which is what the chat model sees.

Claude Code and Codex report command results differently: Codex as `{exitCode, output}`, Claude Code as `{stdout, stderr, interrupted}` with failure on the result. `commandResult` handles both. Paths are shown relative to the sandbox: `activityLog.strip` removes the sandbox directory, and markdown links are flattened.

The session is destroyed when `Run` returns. `coder.mu` allows one run at a time because the bridge ports are fixed.

## workspace.go

`workspace` is the project on disk (`.data/workspace`). Every path from a tool goes through `resolve`, which rejects absolute paths and `..`. Files over 256 KB (`maxFileBytes`) are skipped. `RunTests` runs `go test -count=1 ./...` with a two-minute timeout and returns a `testRun`.

## Tests

| Test file | Covers |
| --- | --- |
| `server/coder_test.go` | `activityLog` (path stripping across deltas, command results), `describeToolCall`, `commandResult` for both result shapes |
| `server/workspace_test.go` | path escapes rejected, `Reset` restores the template |
| `server/docs_test.go` | every file and server symbol named in the docs exists |

Run them with `go test -race ./server`.
