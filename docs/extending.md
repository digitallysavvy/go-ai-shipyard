# Extending Shipyard

**Summary.** Short recipes for the common changes. Each lists the files to touch, in order. Run `make test` after each.

| Recipe | Files |
| --- | --- |
| [Add a tool](#add-a-tool) | `server/tools.go`, maybe `web/components/ToolPart.tsx` |
| [Make a tool ask for approval](#make-a-tool-ask-for-approval) | `server/tools.go` |
| [Change the models](#change-the-models) | `.env` |
| [Add a pairing](#add-a-pairing) | `server/chat.go`, `server/coder.go`, `web/lib/types.ts`, `web/app/page.tsx` |
| [Run the coding agent in a remote sandbox](#run-the-coding-agent-in-a-remote-sandbox) | `server/coder.go` |
| [Use your own sample project](#use-your-own-sample-project) | `workspace-template/`, `server/chat.go`, `server/workspace.go`, `web/app/page.tsx` |
| [Use an unreleased go-ai](#use-an-unreleased-go-ai) | `go.work` (local only) |

## Add a tool

Append a `types.Tool` to the slice in `tools` (`server/tools.go`):

```go
{
	Name:        "git_log",
	Description: "Show the last commits in the workspace.",
	Parameters: objectSchema(map[string]interface{}{
		"limit": map[string]interface{}{"type": "integer", "description": "How many commits, at most 20."},
	}, "limit"),
	Execute: func(ctx context.Context, input map[string]interface{}, _ types.ToolExecutionOptions) (interface{}, error) {
		// Read input values, call into the workspace, return any JSON-able value.
		return map[string]interface{}{"commits": []string{}}, nil
	},
},
```

The model sees the name, description and schema; the browser sees the call and result. Without a matching `case` in `ToolPart`, the row shows the tool name only. Add a `case` there to render something better. If the agent should use the tool in a particular way, say so in `systemPrompt` (`server/chat.go`).

Tools receive model-chosen input. Validate paths with `workspace.resolve` (as `ReadFile` does) and never pass input to a shell.

## Make a tool ask for approval

Set `ToolApproval` on the tool. `true` always asks:

```go
ToolApproval: true,
```

To ask only for some inputs, use a function. The SDK also accepts a bare literal with this signature; the named type documents intent:

```go
ToolApproval: types.ToolNeedsApprovalFunc(func(ctx context.Context, input map[string]interface{}, opts types.ToolNeedsApprovalOptions) bool {
	limit, _ := input["limit"].(float64)
	return limit > 10
}),
```

Nothing else changes: the agent already signs approval requests, and `DelegatePart`'s approval card is generic enough to copy for a new tool. Unknown approval values fail closed (the tool reports an error instead of running).

## Change the models

Set these in `.env` and restart:

| Variable | Default |
| --- | --- |
| `CLAUDE_CHAT_MODEL` | `claude-sonnet-5-5` |
| `CLAUDE_CODE_MODEL` | `claude-sonnet-5-5` |
| `OPENAI_CHAT_MODEL` | `gpt-6-astra` |
| `CODEX_MODEL` | the Codex default |

Model IDs are listed in go-ai's [Anthropic](https://github.com/digitallysavvy/go-ai/blob/main/pkg/providers/anthropic/model_ids.go) and [OpenAI](https://github.com/digitallysavvy/go-ai/blob/main/pkg/providers/openai/model_ids.go) model ID files.

## Add a pairing

A pairing is a chat model plus a coding agent. To add one, for example the OpenCode harness:

1. `server/chat.go`: add a constant next to `agentClaude` and `agentCodex`, give it a name in `CoderName`, return its chat model in `chatModel`, and add it to `handleStatus`.
2. `server/coder.go`: add a `case` in `newHarnessAgent` with the harness constructor (see `pkg/harness/*` in go-ai) and a bridge port that no other pairing uses.
3. `server/main.go`: if it needs a new API key, read it in `loadConfig` and list it in `availableAgents`.
4. `web/lib/types.ts`: add it to `AgentKind`.
5. `web/app/page.tsx`: add it to `PAIRINGS`.
6. `.env.example` and the README: document any new variable.

## Run the coding agent in a remote sandbox

`newHarnessAgent` uses the go-ai local sandbox provider, which runs the coding agent on your machine with your permissions. To isolate it, replace `local.NewProvider(...)` with another `harness.SandboxProvider`, such as Vercel Sandbox (`pkg/harness/sandbox/vercel`; see the [Vercel sandbox reference](https://goaisdk.com/docs/reference/ai/harness-sandbox-vercel)). `Run` already seeds and reads files through the sandbox API, so nothing else in `coder.go` changes. Remote sandboxes take longer to start; warm one up before recording.

## Use your own sample project

1. Replace the files in `workspace-template/`. Keep it small: every coding run copies all of it into the sandbox.
2. `server/workspace.go`: `RunTests` runs `go test -count=1 ./...`. Change the command for another language.
3. `server/chat.go`: update `systemPrompt`, which describes the project as a Go module.
4. `web/app/page.tsx`: update `SUGGESTION` and the text in `EmptyState`.

`make reset`, or **Start over** in the app, copies the new template into `.data/workspace`.

## Use an unreleased go-ai

`go.mod` pins a released go-ai. To try SDK changes before they're tagged, clone go-ai next to this repo, check out the branch, and add a workspace file:

```bash
go work init . ../go-ai
```

`go.work` is git-ignored. Delete it to go back to the released version. Don't commit a `replace` directive or a pseudo-version to `main`.
