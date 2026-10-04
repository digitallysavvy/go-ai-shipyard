# Web

**Summary.** `web/` is a Next.js app with one page. `web/app/page.tsx` owns the `useChat` state and renders messages. Tool calls render through `web/components/ToolPart.tsx`, which shows the approval card and hands the coding agent's progress to `web/components/CoderCard.tsx`. `web/components/Pipeline.tsx` is the header that lights up each hop of a request. Data shapes shared with the server live in `web/lib/types.ts`.

| File | Key exports | Read it to |
| --- | --- | --- |
| `web/app/page.tsx` | `Page`, `MessageView`, `EmptyState`, `derivePhase` | change chat behavior, the pairing toggle, Start over or message layout |
| `web/components/ToolPart.tsx` | `ToolPart`, `DelegatePart`, `TestsPart` | change how a tool call looks, including the approval card |
| `web/components/CoderCard.tsx` | `CoderCard`, `EventLine`, `Diff` | change the coding agent's live log or the diff view |
| `web/components/Pipeline.tsx` | `Pipeline`, `Phase` | change the header pipeline |
| `web/components/InlineText.tsx` | `InlineText` | change how inline code in agent text renders |
| `web/lib/types.ts` | `CoderState`, `CoderEvent`, `TestRun`, `DemoMessage`, `AgentKind`, `AgentStatus`, `API_URL` | change a shape the server sends |
| `web/app/globals.css` | Tailwind `@theme` tokens | change colors, fonts or shadows |

## page.tsx

`Page` sets up `useChat<DemoMessage>` with:

- a `DefaultChatTransport` pointed at `${API_URL}/api/chat`, whose `body` function adds `{ agent }` from a ref, so the pairing toggle applies to every request, including the automatic one after an approval;
- `sendAutomaticallyWhen: lastAssistantMessageIsCompleteWithApprovalResponses`, which sends the conversation back as soon as the user answers an approval.

On load it calls `GET /api/status` to learn which pairings have an API key and to show the model names. A pairing without a key is disabled. **Start over** calls `POST /api/reset` and clears the messages.

`MessageView` renders a message's parts in order. It collects `data-coder` parts into a map by ID, so each `delegate_to_coding_agent` tool part finds its live state by tool call ID. Data parts aren't rendered on their own.

`derivePhase` turns the chat status and the last message into a `Phase` for the header: `sending` (request in flight), `thinking` (streaming), `approval` (an approval card is waiting), `coding` (a `data-coder` part is `starting` or `running`) or `idle`.

`API_URL` defaults to `http://localhost:8080`. Set `NEXT_PUBLIC_API_URL` to point the UI at another server. The server must allow the web origin (`WEB_ORIGIN`).

## ToolPart.tsx

`ToolPart` switches on the tool name (`getToolName`) and renders one row per tool call. `TestsPart` shows pass or fail with the output behind a `<details>`. `DelegatePart` covers the whole approval lifecycle by `part.state`:

| `part.state` | Shows |
| --- | --- |
| `input-streaming`, `input-available` | "Writing a task for …" |
| `approval-requested` | The approval card: the task text, **Approve** and **Deny**, which call `onApproval(part.approval.id, approved)` |
| `approval-responded` (denied) or `output-denied` | "You denied the change …" |
| anything after approval | `CoderCard` with the matching `data-coder` state |

## CoderCard.tsx

`CoderCard` shows the header (status dot, agent name, elapsed time), a terminal-style log of `CoderEvent`s that auto-scrolls, and, when done, the changed files and the `Diff`. `EventLine` renders each event kind. Command output is shown only for test runs and failed commands, so the log stays readable on a recording.

## Pipeline.tsx

`Pipeline` draws four nodes (useChat, `POST /api/chat`, go-ai agent, coding agent) and lights the ones the current `Phase` involves. The animated link is the one CSS animation in the app (`.link-active` in `globals.css`), and it stops under `prefers-reduced-motion`.

## Styling

Tailwind 4 with tokens in `@theme` in `web/app/globals.css`: `paper`, `ink`, `go`, `go-deep`, `gopher`, `gopher-soft`, `amber` (only for the approval moment), `pass`, `fail`, `muted`. Fonts: Figtree for UI, Go Mono (bundled in `web/app/fonts/`) for code. Use the tokens instead of raw colors.

## Checks

`pnpm typecheck` and `pnpm build` (both in CI). There are no component tests; the end-to-end check is a manual run, see the README.
