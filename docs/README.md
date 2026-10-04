# Shipyard docs

Read [architecture.md](architecture.md) first, then only the page for the area you're changing. The task-to-page map, the terms (Shipyard, coding agent, pairing) and the rules are in [AGENTS.md](../AGENTS.md). Setup is in the [README](../README.md).

| Page | Covers |
| --- | --- |
| [architecture.md](architecture.md) | One request end to end, where state lives, design choices |
| [server.md](server.md) | `server/`: endpoint, tools, coding-agent runner, workspace |
| [web.md](web.md) | `web/`: chat page, tool rows, approval card, coding-agent card, header |
| [protocol.md](protocol.md) | Request body, stream chunks, the `data-coder` part, approvals, `/api/status` |
| [extending.md](extending.md) | Recipes: tools, approval, models, pairings, sandboxes, sample project, unreleased go-ai |
| [troubleshooting.md](troubleshooting.md) | Failures, causes and fixes |

## Go AI SDK guides

Shipyard follows these guides:

- [Serve a useChat frontend from Go](https://goaisdk.com/docs/build-a-chat-app/serve-usechat-from-go)
- [Tool approval end to end](https://goaisdk.com/docs/build-a-chat-app/tool-approval)
- [Coding agents with the harness](https://goaisdk.com/docs/build-a-chat-app/coding-agents-harness)
