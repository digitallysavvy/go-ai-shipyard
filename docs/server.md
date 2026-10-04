# Server

`server/` is one Go package (`package main`), about 1,000 lines, one concern per file.

| File | Key symbols | Read it to |
| --- | --- | --- |
| `server/main.go` | `main`, `config`, `loadConfig`, `repoRoot`, `withCORS` | change routes, ports, env vars or CORS |
| `server/chat.go` | `handleChat`, `chatModel`, `systemPrompt`, `agentKind`, `chatRequest`, `handleReset`, `handleStatus` | change the agent, its models or prompt, or the request body |
| `server/tools.go` | `tools`, `objectSchema` | add or change a tool |
| `server/coder.go` | `coder`, `newHarnessAgent`, `Run`, `collectChanges`, `activityLog`, `commandResult`, `describeToolCall`, `unifiedDiff` | change how Claude Code or Codex runs, or how their activity is shown |
| `server/workspace.go` | `workspace`, `Reset`, `Files`, `ReadFile`, `Apply`, `RunTests`, `resolve` | change where the project lives or how tests run |
| `server/dotenv.go` | `loadDotEnv` | change `.env` loading |

## main.go

`main` finds the repo root (`repoRoot` walks up to the directory holding `workspace-template/`), loads `.env`, builds a `config` with `loadConfig`, resets the workspace and serves:

| Route | Handler | Does |
| --- | --- | --- |
| `POST /api/chat` | `handleChat` | Streams Shipyard's reply. See below. |
| `POST /api/reset` | `handleReset` | Restores the workspace to the failing template |
| `GET /api/status` | `handleStatus` | Which pairings have a key, with model names and key variables, for the UI toggle |

`loadConfig` reads every environment variable; [`.env.example`](../.env.example) lists them. If `TOOL_APPROVAL_SECRET` is unset, it generates a random approval secret per process.

## chat.go

`handleChat` is the core of the app:

1. Decode `chatRequest` (`id`, raw `messages`, `agent`), limited to 8 MB.
2. `chatModel` returns the Anthropic or OpenAI model for the `agentKind`, or a 400 naming the missing key variable.
3. `ai.CreateUIMessageStreamWithOptions` gets an `Execute` func that builds the `ToolLoopAgent` (model, `systemPrompt`, `s.tools(kind, writer)`, `ai.IsStepCount(12)`, `ExperimentalToolApprovalSecret`), starts it with `agent.CreateAgentUIStreamFromUIMessages`, and merges the stream into the writer.
4. `ai.PipeUIMessageChunksToResponse` writes the headers and the SSE body, with a 15-second keep-alive.

`OnError` returns the real error text so failures show in the UI. That suits a local demo; return a generic message in production.

`agentKind` is the pairing (`claude` or `codex`). Its methods give the coding agent's display name (`CoderName`) and the key variable that enables it (`apiKeyEnv`).

## tools.go

`tools` builds the tool list per request, so `delegate_to_coding_agent` can capture that request's stream writer. Parameters are JSON Schema maps built with `objectSchema`, which disallows extra properties.

| Tool | Approval | Calls |
| --- | --- | --- |
| `run_tests` | no | `workspace.RunTests` |
| `list_files` | no | `workspace.Files` |
| `read_file` | no | `workspace.ReadFile` |
| `delegate_to_coding_agent` | **yes** (`ToolApproval: true`) | `coder.Run` with the tool call ID and writer |

To add one, see [extending.md](extending.md#add-a-tool).

## coder.go

`coder.Run(ctx, kind, task, toolCallID, writer)` is everything after the user approves:

1. Snapshot the workspace (`workspace.Files`).
2. `newHarnessAgent` builds a `harness.Agent` for the pairing: `claudecode.New` or `codex.New`, the local sandbox provider with a fixed bridge port (4319 for Claude Code, 4318 for Codex), `PermissionModeAllowAll`, and the `seed` closure from `Run` as the `OnSession` hook.
3. `CreateSession` with an explicit session ID. `seed` writes the snapshot into the sandbox and commits a git baseline, so the coding agent can use `git diff`.
4. `Stream` the task and read `FullStream()`. `activityLog.apply` folds each chunk into a short log: text becomes `say` events, tool calls become `run`, `read`, `edit` or `tool` events (`describeToolCall`), and command results fill in the exit code and output (`commandResult`). The stream ends with a finish chunk whose `Usage` holds the run's token counts; `apply` ignores it today.
5. Emit the `coderState` as a `data-coder` part after each change. Text updates are throttled to one per 120 ms; tool calls, results and status changes go out at once.
6. `collectChanges` lists the sandbox's files (skipping dotfiles, dot-directories and files of 256 KB or more), reads each one and keeps those that are new or differ from the snapshot. Deleted files are not synced back. `workspace.Apply` writes the changes, and `unifiedDiff` renders the diff with the system `diff` command.
7. Return a `coderResult` (agent, summary, files changed, diff). That is the tool result the chat model sees.

Claude Code and Codex report command results differently: Codex as `{exitCode, output}`, Claude Code as `{stdout, stderr, interrupted}` with failure marked on the result. `commandResult` handles both. `activityLog.strip` removes the sandbox directory from paths, and markdown links are replaced with their link text.

The session is destroyed when `Run` returns. `coder.mu` allows one run at a time, because the bridge ports are fixed.

## workspace.go

`workspace` is the project on disk (`.data/workspace`). Every path from a tool goes through `resolve`, which rejects absolute paths and `..`. Files over 256 KB (`maxFileBytes`) are skipped. `RunTests` runs `go test -count=1 ./...` with a two-minute timeout and returns a `testRun`. The workspace is not a git repository.

## Tests

| File | Covers |
| --- | --- |
| `server/coder_test.go` | `activityLog`, `describeToolCall`, `commandResult` for both result shapes |
| `server/workspace_test.go` | path escapes rejected, `Reset` restores the template |
| `server/docs_test.go` | links, repo paths and file-and-symbols tables in the docs |
