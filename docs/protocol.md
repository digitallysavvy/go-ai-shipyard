# Protocol

The browser and server speak the AI SDK UI message stream protocol (v1): the browser POSTs the whole conversation as UI messages, and the server answers with Server-Sent Events, one JSON chunk per event, ending with `data: [DONE]`. Shipyard adds one request field (`agent`) and one custom part (`data-coder`). Everything else is standard, which is why the frontend is plain `useChat`.

## Request

`POST /api/chat`, `Content-Type: application/json`:

```json
{
  "id": "chat-id-from-useChat",
  "messages": [{ "id": "u1", "role": "user", "parts": [{ "type": "text", "text": "CI is red on shortlink. Find out why and get it fixed." }] }],
  "agent": "claude"
}
```

| Field | Set by | Meaning |
| --- | --- | --- |
| `id`, `messages` | `useChat` | Standard. `messages` is the full history, including approval responses |
| `agent` | the transport's `body` in `page.tsx` | `"claude"` or `"codex"`, parsed into `agentKind` in `server/chat.go`. Defaults to `claude` |

Bad JSON gets a 400. A pairing without its API key gets a 400 with the variable to set. Errors after streaming starts arrive as `error` chunks.

## Response

Status 200 with `Content-Type: text/event-stream` and `X-Vercel-AI-UI-Message-Stream: v1`, set by `ai.PipeUIMessageChunksToResponse`. A `: keep-alive` comment goes out every 15 seconds without other output.

Chunks Shipyard relies on:

| Chunk `type` | When | Used by |
| --- | --- | --- |
| `start`, `start-step`, `finish-step`, `finish` | Message and step boundaries | `useChat` |
| `text-start`, `text-delta`, `text-end` | Agent text | `MessageView` |
| `tool-input-available` (after `tool-input-start` and `tool-input-delta` when the model streams arguments) | A tool call and its arguments | `ToolPart` |
| `tool-output-available` | A tool result | `ToolPart`, `TestsPart` |
| `tool-approval-request` | `delegate_to_coding_agent` needs approval; carries `approvalId` and `signature` | `DelegatePart` |
| `tool-output-denied` | The user denied | `DelegatePart` |
| `data-coder` | Coding-agent progress | `CoderCard` |
| `error` | Anything that failed after streaming started | `useChat` (`error`) |

The full chunk list is in the SDK reference: [UI message chunks](https://goaisdk.com/docs/reference/ai/ui-message-chunks).

## The data-coder part

`coder.Run` writes this chunk for every state change of a coding run:

```json
{ "type": "data-coder", "id": "<tool call id>", "data": { "...": "coderState" } }
```

Because `id` is the tool call ID, each write replaces the previous part in the message instead of adding one, and `ToolPart` can match the part to its tool call. The `data` object is `coderState` in `server/coder.go`, mirrored by `CoderState` in `web/lib/types.ts`:

| Field | Type | Notes |
| --- | --- | --- |
| `agent` | string | `"Claude Code"` or `"Codex"` |
| `status` | `starting` \| `running` \| `done` \| `error` | |
| `events` | `CoderEvent[]` | Never null; an empty run sends `[]` |
| `filesChanged` | string[] | When `done` |
| `diff` | string | Unified diff, when `done` |
| `summary` | string | The coding agent's last message, when `done` |
| `error` | string | When `error` |
| `elapsedMs` | number | Since the run started |

A `CoderEvent` (`coderEvent` in Go) has `id`, `kind` (`say`, `run`, `read`, `edit` or `tool`), `text`, and for `run` events `exit` and `output`. `output` keeps the last 8 lines, preceded by a `…` line when it was cut.

Change the Go struct and the TypeScript type together.

## Approval round trip

1. Turn 1 ends with `tool-approval-request` (`toolCallId`, `approvalId`, `signature`). `useChat` puts the tool part in state `approval-requested`.
2. The user clicks Approve or Deny. `addToolApprovalResponse({ id: approvalId, approved })` sets the part to `approval-responded`.
3. `lastAssistantMessageIsCompleteWithApprovalResponses` sees every approval answered and sends turn 2 with the updated messages.
4. The server validates the messages and verifies `signature` with the approval secret. A forged or tampered approval is rejected. Approved calls run before the model is called; denied calls become `tool-output-denied`. In go-ai, signing and verification are `ai.SignToolApproval` and `ai.VerifyToolApprovalSignature` ([tool_approval_signature.go](https://github.com/digitallysavvy/go-ai/blob/main/pkg/ai/tool_approval_signature.go)), and resuming is `ai.ResumeToolApprovals` ([tool_approval_resume.go](https://github.com/digitallysavvy/go-ai/blob/main/pkg/ai/tool_approval_resume.go)).

If the server restarts between the two turns without `TOOL_APPROVAL_SECRET` set, the new random secret can't verify the old signature, and the approval is rejected. Set the variable to keep approvals valid across restarts.

## Status

`GET /api/status` tells the UI which pairings are usable:

```json
{
  "agents": {
    "claude": { "available": true, "chatModel": "claude-sonnet-5-5", "coder": "Claude Code", "apiKeyEnv": "ANTHROPIC_API_KEY" },
    "codex": { "available": false, "chatModel": "gpt-6-astra", "coder": "Codex", "apiKeyEnv": "OPENAI_API_KEY" }
  }
}
```

`handleStatus` in `server/chat.go` builds it; `AgentStatus` in `web/lib/types.ts` mirrors one entry.
