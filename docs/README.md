# Shipyard docs

Shipyard is a chat app that fixes a failing Go project. A Next.js `useChat` frontend talks to a Go server built with the Go AI SDK. The server's agent investigates, asks the user to approve a code change, and has Claude Code or Codex make it in a sandbox.

These docs go from the whole picture to single files. Read the first page, then only the page for the area you're changing. Each page opens with a summary and a table of the files and symbols it covers.

| Page | Read it when you |
| --- | --- |
| [architecture.md](architecture.md) | are new to the code, or need the end-to-end flow of one chat turn and the approval round trip |
| [server.md](server.md) | change anything under `server/`: the chat endpoint, tools, the coding-agent runner, the workspace |
| [web.md](web.md) | change anything under `web/`: the chat page, tool rows, the approval card, the coding-agent card, the pipeline header |
| [protocol.md](protocol.md) | change data that crosses the wire: the request body, stream chunks, the `data-coder` part, approval fields |
| [extending.md](extending.md) | add a tool, a pairing, a model or a sandbox provider, replace the sample project, or test unreleased go-ai |
| [troubleshooting.md](troubleshooting.md) | can't get it to run, or something behaves oddly |

For setup and a tour aimed at people, see the [README](../README.md). For rules every change must follow, see [AGENTS.md](../AGENTS.md).

## SDK docs

Shipyard uses these parts of the Go AI SDK. The guides explain the same patterns in more depth:

- [Serve a useChat frontend from Go](https://goaisdk.com/docs/build-a-chat-app/serve-usechat-from-go)
- [Tool approval end to end](https://goaisdk.com/docs/build-a-chat-app/tool-approval)
- [Coding agents with the harness](https://goaisdk.com/docs/build-a-chat-app/coding-agents-harness)
- Agent-readable index of all SDK docs: https://goaisdk.com/llms.txt
