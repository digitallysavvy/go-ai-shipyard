# Web

`web/` is a Next.js app with one page. `web/app/page.tsx` owns the `useChat` state and renders messages. Tool calls render through `web/components/ToolPart.tsx`, which shows approval cards and hands the coding agent's progress to `web/components/CoderCard.tsx`. `web/components/Pipeline.tsx` is the header that lights up each hop of a request. Shapes shared with the server are in `web/lib/types.ts`.

| File | Key symbols | Read it to |
| --- | --- | --- |
| `web/app/page.tsx` | `Page`, `PAIRINGS`, `MessageView`, `EmptyState`, `derivePhase`, `isNetworkError` | change chat behavior, the pairing toggle, Start over or message layout |
| `web/components/ToolPart.tsx` | `ToolPart`, `ApprovalCard`, `DelegatePart`, `TestsPart` | change how a tool call looks, including approval cards |
| `web/components/CoderCard.tsx` | `CoderCard`, `EventLine`, `Diff` | change the coding agent's live log or the diff view |
| `web/components/Pipeline.tsx` | `Pipeline`, `Phase` | change the header |
| `web/components/InlineText.tsx` | `InlineText` | change how inline code in agent text renders |
| `web/lib/types.ts` | `CoderState`, `CoderEvent`, `TestRun`, `DemoMessage`, `AgentKind`, `AgentStatus`, `API_URL` | change a shape the server sends |

## page.tsx

`Page` sets up `useChat<DemoMessage>` with a `DefaultChatTransport` pointed at `${API_URL}/api/chat`. Its `body` function adds `{ agent }` from a ref, so the pairing toggle applies to every request, including the automatic one after an approval. `sendAutomaticallyWhen` resends the conversation once the user answers an approval; see [protocol.md](protocol.md#approval-round-trip).

On load, `Page` calls `GET /api/status`. Pairings without a key are disabled, with a tooltip naming the variable to set. If the selected pairing has no key, the first available one in `PAIRINGS` is selected. **Start over** calls `POST /api/reset` and clears the messages. Errors show as "Can't reach the server" when `isNetworkError` matches, and as the server's message otherwise.

`MessageView` renders a message's parts in order. It collects `data-coder` parts into a map by ID, so each `delegate_to_coding_agent` tool part finds its live state by tool call ID. Data parts aren't rendered on their own.

`derivePhase` turns the chat status and the last message into a `Phase` for the header:

| Phase | When |
| --- | --- |
| `approval` | The last assistant message has a part in `approval-requested` and the chat is `ready` |
| `coding` | The last assistant message has a `data-coder` part that is `starting` or `running` and the chat is `streaming` |
| `sending` | The chat is `submitted` |
| `thinking` | The chat is `streaming` otherwise |
| `idle` | Anything else |

`API_URL` defaults to `http://localhost:8080`. Set `NEXT_PUBLIC_API_URL` to point the UI at another server, and set the server's `WEB_ORIGIN` to allow the UI's origin.

## ToolPart.tsx

`ToolPart` switches on the tool name (`getToolName`) and renders one row per tool call. Any tool in `approval-requested` gets an `ApprovalCard` with its input and Approve and Deny, so a new approval-gated tool works without UI changes. `TestsPart` shows pass or fail with the output behind a `<details>`. `DelegatePart` covers the coding agent's lifecycle by `part.state`:

| `part.state` | Shows |
| --- | --- |
| `input-streaming`, `input-available` | "Writing a task for …" |
| `approval-requested` | `ApprovalCard` with the task text |
| `approval-responded` (denied) or `output-denied` | "You denied the change …" |
| anything after approval | `CoderCard` with the matching `data-coder` state |

## CoderCard.tsx

`CoderCard` shows a header (status dot, agent name, elapsed time), a log of `CoderEvent`s that auto-scrolls, and, when done, the changed files and the `Diff`. `EventLine` renders each event kind. Command output appears only for test runs and failed commands, so the log stays readable on a recording.

## Pipeline.tsx

`Pipeline` draws four nodes (useChat, `POST /api/chat`, go-ai agent, coding agent) and lights the ones the current `Phase` involves. The animated link (`.link-active` in `web/app/globals.css`) is the app's only animation and stops under `prefers-reduced-motion`.

## Styling

Tailwind 4. Colors, fonts and shadows are tokens in `@theme` in `web/app/globals.css`; use them instead of raw values. Amber is reserved for approval cards. Fonts are Figtree for the UI and Go Mono (bundled in `web/app/fonts/`) for code.
