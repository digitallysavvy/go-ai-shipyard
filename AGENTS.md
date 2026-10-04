# AGENTS.md

Shipyard is a chat app that fixes a failing Go project: a Next.js `useChat` frontend, a Go backend built with the [Go AI SDK](https://goaisdk.com) (`github.com/digitallysavvy/go-ai`), and Claude Code or Codex doing the coding through the go-ai harness. It is the reference app for the SDK, so code here should stay small, readable and copyable.

Start here, then open only the doc you need. Every doc lists the files and symbols it covers, so you can go straight to the code.

## Docs map

| You want to | Read |
| --- | --- |
| Understand the whole request path in one page | [docs/architecture.md](docs/architecture.md) |
| Change the Go backend: chat endpoint, tools, coding agent, workspace | [docs/server.md](docs/server.md) |
| Change the UI: chat, tool cards, approval card, pipeline header | [docs/web.md](docs/web.md) |
| Know what goes over the wire: request body, stream chunks, the `data-coder` part, approvals | [docs/protocol.md](docs/protocol.md) |
| Add a tool, a pairing, a model, a sandbox or a sample project | [docs/extending.md](docs/extending.md) |
| Fix something that doesn't run | [docs/troubleshooting.md](docs/troubleshooting.md) |

All docs: [docs/README.md](docs/README.md).

## Layout

```
server/              Go backend (package main), one concern per file
  main.go            config, HTTP routes, CORS, startup
  chat.go            POST /api/chat: the agent, models, system prompt
  tools.go           the agent's tools (run_tests, list_files, read_file, delegate_to_coding_agent)
  coder.go           runs Claude Code or Codex via the go-ai harness; activity log; diff
  workspace.go       the project the agents work on (.data/workspace)
  dotenv.go          .env loader
web/                 Next.js app (React 19, Tailwind 4, @ai-sdk/react)
  app/page.tsx       the chat page: useChat, pairing toggle, phase for the pipeline header
  components/        ToolPart (tool rows, approval card), CoderCard (live log, diff), Pipeline, InlineText
  lib/types.ts       TypeScript mirrors of the server's data shapes
workspace-template/  the failing sample project, copied to .data/workspace on start and reset
docs/                developer docs (this map's targets)
```

## Commands

| Task | Command |
| --- | --- |
| First setup (creates `.env`, installs web deps) | `make setup` |
| Run server (:8080) and web (:3000) | `make dev` |
| Test and type-check everything | `make test` |
| Server tests only | `go test -race ./server` |
| Web type-check only | `cd web && pnpm typecheck` |
| Reset the sample project and sandboxes | `make reset` |

`make test` must pass before you commit. CI (`.github/workflows/ci.yml`) runs the same checks plus `pnpm build`.

## Rules

- **Keep the server and web shapes in sync.** `coderState`/`coderEvent` in `server/coder.go` and `CoderState`/`CoderEvent` in `web/lib/types.ts` describe the same JSON. Change both together. See [docs/protocol.md](docs/protocol.md).
- **Tool approval is a security boundary.** `delegate_to_coding_agent` must keep `ToolApproval: true`, and the agent must keep `ExperimentalToolApprovalSecret`. Don't add a code path that runs the coding agent without an approved call.
- **No new Go dependencies** beyond go-ai without a strong reason. The server is meant to be read in one sitting.
- **Never commit `.env` or anything under `.data/`.** Both are git-ignored.
- **The local sandbox is not isolated.** Coding agents run as the current user. Keep the sample project small and don't point the workspace at real repos.
- **Use the released go-ai.** `go.mod` pins a tagged version. To test unreleased SDK changes, use a local `go.work` (git-ignored); see [docs/extending.md](docs/extending.md#use-an-unreleased-go-ai).
- Go: `gofmt`, standard library first, errors wrapped with context. TypeScript: strict mode, design tokens from `web/app/globals.css`, no new UI libraries.

## Docs stay true

`server/docs_test.go` checks that every file path and `server` symbol named in this file and in `docs/` exists. If you rename or move code, update the docs in the same change, or the test fails.
