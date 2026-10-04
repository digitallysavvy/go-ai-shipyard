# Troubleshooting

**Summary.** The common failures, what causes each, and the fix. Start with the server log: `make dev` prints it in the same terminal as the web app.

## A pairing button is greyed out

That pairing has no API key. The server log shows which pairings loaded, for example `chat agents available: [claude codex]`. Add `ANTHROPIC_API_KEY` (Claude Code) or `OPENAI_API_KEY` (Codex) to `.env` and restart. Keys exported in your shell take precedence over `.env`.

## "The Go server returned an error … Check that it's running"

The browser can't reach `http://localhost:8080`. Check that the server started (`make dev` or `make server`). If you changed `PORT`, set `NEXT_PUBLIC_API_URL` for the web app to match. If you changed where the web app runs, set `WEB_ORIGIN` so CORS allows it.

## The first coding run takes a minute or more

On its first run, each pairing installs its bridge (the Claude Code or Codex CLI and their dependencies) into the sandbox with pnpm. Later runs reuse the install and start in a few seconds. Do one warm-up run per pairing before recording.

## The coding agent fails to start: port already in use

Each pairing's bridge listens on a fixed port: 4319 for Claude Code, 4318 for Codex (`newHarnessAgent` in `server/coder.go`). Stop whatever else is using the port, or change the port there.

## The approval is rejected after a server restart

Approval requests are signed with a secret that is random per process unless `TOOL_APPROVAL_SECRET` is set. An approval card from before a restart can't be verified by the new process. Click **Start over**, or set `TOOL_APPROVAL_SECRET` in `.env`.

## The coding agent changed nothing, or the tests still fail

The agent only sees the task text the chat model wrote. Read the task on the approval card; if it's vague, the fix will be too. Check the live log for the commands the agent ran. **Start over** resets the project to its original failing state.

## Output arrives all at once instead of streaming

A proxy between the browser and the server is buffering the response. The server already sends `X-Accel-Buffering: no` and a keep-alive every 15 seconds. Turn off response buffering in the proxy, or connect to the server directly.

## `make test` fails on `server/docs_test.go`

A doc names a file or Go symbol that no longer exists. The test output lists each one. Update the doc in the same change as the rename.

## Something else

Run `go test -race ./server` and `cd web && pnpm typecheck` to rule out a broken build, then check the server log for the first error.
