# Extending Shipyard

Recipes for the changes people make most. Each lists the files to touch, in order. Run `make test` after each.

| Recipe | Files |
| --- | --- |
| [Add a tool](#add-a-tool) | `server/tools.go`, maybe `web/components/ToolPart.tsx` |
| [Make a tool ask for approval](#make-a-tool-ask-for-approval) | `server/tools.go` |
| [Change the models](#change-the-models) | `.env` |
| [Add a pairing](#add-a-pairing) | `server/chat.go`, `server/coder.go`, `server/main.go`, `web/lib/types.ts`, `web/app/page.tsx` |
| [Run the coding agent in a remote sandbox](#run-the-coding-agent-in-a-remote-sandbox) | `server/coder.go` |
| [Use your own sample project](#use-your-own-sample-project) | `workspace-template/` and the files listed in the recipe |
| [Use an unreleased go-ai](#use-an-unreleased-go-ai) | `go.work` (local only) |

## Add a tool

Append a `types.Tool` to the slice in `tools` (`server/tools.go`):

```go
{
	Name:        "count_lines",
	Description: "Count the lines in a workspace file.",
	Parameters: objectSchema(map[string]interface{}{
		"path": map[string]interface{}{"type": "string", "description": "Path relative to the workspace root."},
	}, "path"),
	Execute: func(ctx context.Context, input map[string]interface{}, _ types.ToolExecutionOptions) (interface{}, error) {
		p, _ := input["path"].(string)
		content, err := s.ws.ReadFile(p) // ReadFile rejects paths outside the workspace
		if err != nil {
			return nil, err
		}
		return map[string]interface{}{"path": p, "lines": strings.Count(content, "\n")}, nil
	},
},
```

The model sees the name, description and schema; the browser sees the call and result. Without a `case` in `ToolPart`, the row shows the tool name only; add one to render something better. If Shipyard should use the tool in a particular way, say so in `systemPrompt` (`server/chat.go`).

Tools receive model-chosen input. Resolve paths through the workspace (as `ReadFile` does with `resolve`) and never pass input to a shell.

The workspace is not a git repository. A tool that runs git needs its own working directory, or has to run inside the coding agent's sandbox, which has a baseline commit.

## Make a tool ask for approval

Set `ToolApproval` on the tool. `true` always asks:

```go
ToolApproval: true,
```

To ask only for some inputs, use a function. Ask when in doubt: here a missing or empty branch also needs approval, so a model that leaves it out can't skip the check.

```go
ToolApproval: types.ToolNeedsApprovalFunc(func(ctx context.Context, input map[string]interface{}, opts types.ToolNeedsApprovalOptions) bool {
	branch, _ := input["branch"].(string)
	return branch == "" || branch == "main"
}),
```

That's all. Approval requests are already signed (`ExperimentalToolApprovalSecret` in `handleChat`), and `ToolPart` shows an `ApprovalCard` with the tool's input for any tool waiting for approval. Add a `case` in `ToolPart` only if you want custom wording. A value the SDK doesn't recognize fails closed: the tool reports an error instead of running.

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

A pairing is a chat model plus a coding agent. To add one, for example with the OpenCode harness:

1. `server/chat.go`: add a constant next to `agentClaude` and `agentCodex`, then add a `case` for it in `CoderName`, `apiKeyEnv` and `chatModel`, and an entry in `handleStatus`. Update the "unknown agent" error in `chatModel` to list it.
2. `server/coder.go`: add a `case` in `newHarnessAgent` with the harness constructor and a bridge port no other pairing uses. The constructors differ:

   | Coding agent | Constructor |
   | --- | --- |
   | Claude Code | `claudecode.New(claudecode.Settings) (*claudecode.Harness, error)` |
   | Codex | `codex.New(codex.Settings) *codex.Harness` |
   | OpenCode | `opencode.CreateOpenCode(...opencode.Settings) (harness.Harness, error)` |

   Each package is under `pkg/harness/` in go-ai (`go list -m -f '{{.Dir}}' github.com/digitallysavvy/go-ai`).
3. `server/main.go`: if it needs a new API key, add it to `config`, read it in `loadConfig` and list the pairing in `availableAgents`.
4. `web/lib/types.ts`: add it to `AgentKind`.
5. `web/app/page.tsx`: add it to `PAIRINGS`. The fallback pairing and the disabled-button tooltip read `PAIRINGS` and `/api/status`, so they need no change.
6. `.env.example` and the README: document any new variable.

## Run the coding agent in a remote sandbox

`newHarnessAgent` uses the go-ai local sandbox provider, which runs the coding agent on your machine with your permissions. To isolate it, replace `local.NewProvider(...)` with another `harness.SandboxProvider`, such as Vercel Sandbox (`pkg/harness/sandbox/vercel` in go-ai; see the [Vercel sandbox reference](https://goaisdk.com/docs/reference/ai/harness-sandbox-vercel)). `Run` already seeds and reads files through the sandbox API, so the rest of `coder.go` stays the same. Remote sandboxes take longer to start; warm one up before recording.

## Use your own sample project

1. Replace the files in `workspace-template/`. Keep it small: every coding run copies all of it into the sandbox.
2. Change the Go-specific strings to your language's test command and wording:

   | File | Symbol | Says |
   | --- | --- | --- |
   | `server/workspace.go` | `RunTests` | runs `go test -count=1 ./...` |
   | `server/tools.go` | `tools` | the `run_tests` description |
   | `server/chat.go` | `systemPrompt` | "a small Go module" |
   | `server/coder.go` | `newHarnessAgent` | the coding agent's `Instructions` |
   | `web/components/ToolPart.tsx` | `TestsPart` | the `go test ./...` label |
   | `web/app/page.tsx` | `EmptyState` | the empty-state text |

3. Update `SUGGESTION` in `web/app/page.tsx`, the suggested first message.

`make reset`, or **Start over** in the app, copies the new template into `.data/workspace`.

## Use an unreleased go-ai

`go.mod` pins a released go-ai. To try SDK changes before they're tagged, clone go-ai next to this repo, check out the branch, and add a workspace file:

```bash
go work init . ../go-ai
```

`go.work` is git-ignored. Delete it to go back to the released version. Don't commit a `replace` directive or a pseudo-version to `main`.
