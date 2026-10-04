# AGENTS.md

Shipyard is a chat app that fixes a failing Go project. A Next.js `useChat` frontend talks to a Go server built with the Go AI SDK (`github.com/digitallysavvy/go-ai`). The server's agent finds the cause, asks the user to approve a change, and has Claude Code or Codex make it in a sandbox. It is the SDK's reference app, so code here should stay small enough to read in one sitting and copy into another project.

Start here. Then open only the doc for your task; each one names the files and symbols to go to next.

## Docs map

| You want to | Read |
| --- | --- |
| See one request end to end (two HTTP round trips) | [docs/architecture.md](docs/architecture.md) |
| Change the Go server: chat endpoint, tools, coding-agent runner, workspace | [docs/server.md](docs/server.md) |
| Change the UI: chat page, tool rows, approval card, coding-agent card, header | [docs/web.md](docs/web.md) |
| Change what crosses the wire: request body, stream chunks, `data-coder`, approvals, `/api/status` | [docs/protocol.md](docs/protocol.md) |
| Add a tool, approval, model, pairing, sandbox or sample project; test unreleased go-ai | [docs/extending.md](docs/extending.md) |
| Fix something that doesn't run | [docs/troubleshooting.md](docs/troubleshooting.md) |

## Layout

```
server/              Go server (package main)
web/                 Next.js app
workspace-template/  the failing sample project
docs/                these docs
```

## Terms

- **Shipyard** is the chat agent: a go-ai `ToolLoopAgent` running on the chat model.
- **Coding agent** is Claude Code or Codex, run through the go-ai harness.
- **Pairing** is a chat model plus a coding agent: `claude` (Anthropic + Claude Code) or `codex` (OpenAI + Codex). The header toggle picks one.

## Commands

| Task | Command |
| --- | --- |
| First setup (creates `.env`, installs web deps) | `make setup` |
| Run server (:8080) and web (:3000) | `make dev` |
| Vet, test and type-check everything | `make test` |
| Reset the sample project and sandboxes | `make reset` |

`make test` must pass before you commit. CI (`.github/workflows/ci.yml`) runs `go vet`, `go test -race` and `go build` for the server, and `pnpm typecheck` and `pnpm build` for the web app.

## Rules

- **Keep the server and web shapes in sync.** `coderState`/`coderEvent` in `server/coder.go` and `CoderState`/`CoderEvent` in `web/lib/types.ts` describe the same JSON. Change both together.
- **Tool approval is a security boundary.** `delegate_to_coding_agent` keeps `ToolApproval: true`, and the agent keeps `ExperimentalToolApprovalSecret`. No code path may run the coding agent without an approved call.
- **No new Go dependencies** beyond go-ai without a strong reason.
- **Never commit `.env` or anything under `.data/`.** Both are git-ignored.
- **The local sandbox is not isolated.** Coding agents run as the current user. Keep the sample project small and don't point the workspace at a real repo.
- **Pin a released go-ai in `go.mod`.** To test unreleased SDK changes, use a local `go.work`; see [docs/extending.md](docs/extending.md#use-an-unreleased-go-ai).
- Go: `gofmt`, standard library first, errors wrapped with context. TypeScript: strict mode, tokens from `web/app/globals.css`, no new UI libraries.

## go-ai source

The SDK's source is in the module cache: `go list -m -f '{{.Dir}}' github.com/digitallysavvy/go-ai`. Its docs for agents start at https://goaisdk.com/llms.txt.

## Docs stay true

`server/docs_test.go` runs with `make test`. It fails when `AGENTS.md` or a page in `docs/` links to a missing file, names a repo path that doesn't exist, or lists a symbol in a file-and-symbols table row that the file doesn't declare. It doesn't check prose claims such as ports or timeouts, so update those by hand when you change them.
