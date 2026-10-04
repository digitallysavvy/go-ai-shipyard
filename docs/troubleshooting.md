# Troubleshooting

Each entry is a symptom, its cause and the fix. Start with the server log: `make dev` prints it in the same terminal as the web app.

## A pairing button is greyed out

That pairing has no API key. The server log lists the pairings it loaded, for example `chat agents available: [claude codex]`, and the button's tooltip names the variable to set. Add `ANTHROPIC_API_KEY` (Claude Code) or `OPENAI_API_KEY` (Codex) to `.env` and restart the server. Keys exported in your shell take precedence over `.env`.

## "Can't reach the server at http://localhost:8080"

The browser got no response from the server. Check that it started (`make dev` or `make server`). If you changed `PORT`, set `NEXT_PUBLIC_API_URL` for the web app to match. If the web app runs on another origin, set `WEB_ORIGIN` so CORS allows it.

"Something went wrong: …" is different: the server answered with an error, and the text after the colon is its message.

## The first coding run takes a minute or more

On its first run, each pairing installs its bridge (the Claude Code or Codex CLI and their dependencies) into the sandbox with pnpm. Later runs reuse the install and start in a few seconds. Do one warm-up run per pairing before recording.

## The coding agent fails to start: port already in use

Each pairing's bridge listens on a fixed port: 4319 for Claude Code, 4318 for Codex (`newHarnessAgent` in `server/coder.go`). Stop whatever else uses the port, or change it there.

## The approval is rejected after a server restart

Approval requests are signed with a secret that is random per process unless `TOOL_APPROVAL_SECRET` is set. A card from before a restart can't be verified by the new process. Click **Start over**, or set `TOOL_APPROVAL_SECRET` in `.env`.

## The coding agent changed nothing, or the tests still fail

The coding agent sees only the task text the chat model wrote. Read the task on the approval card; if it's vague, the fix will be too. The live log shows the commands the agent ran. **Start over** resets the project to its failing state.

## Output arrives all at once instead of streaming

Check whether the server streams:

```bash
curl -N -X POST localhost:8080/api/chat -H 'Content-Type: application/json' \
  -d '{"agent":"codex","messages":[{"id":"u1","role":"user","parts":[{"type":"text","text":"Run the tests."}]}]}'
```

If `data:` lines arrive one by one, the server is fine, and something between it and the browser is buffering: usually a reverse proxy, or `NEXT_PUBLIC_API_URL` pointing through one. The server already sends `X-Accel-Buffering: no` and a keep-alive every 15 seconds. Turn off response buffering in the proxy, or connect to the server directly. Use `"agent":"claude"` if only the Anthropic key is set.

## `make test` fails on `server/docs_test.go`

A doc names a file, link or symbol that no longer exists. The test output lists each one. Update the doc in the same change as the rename.
