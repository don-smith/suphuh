# Agent adapter interface

suphuh observes agents running in existing tmux panes. The common integration contract between suphuh and an agent adapter is intentionally small.

## Responsibilities

### suphuh

- Discovers tmux panes.
- Identifies likely agent type from process metadata where possible.
- Reads status reports from `~/.suphuh/status`.
- Renders status in the popup.
- Jumps to panes; it does not own or proxy agent stdin/stdout.

### Agent adapter

- Hooks into one coding agent's lifecycle/event system.
- Determines the current tmux pane id, normally via `TMUX_PANE`.
- Publishes status reports using the common JSON schema.
- Cleans up its report on shutdown if possible.
- Fails silently; status integration must never break the coding agent.

## Status report schema

Reports are JSON files in:

```text
~/.suphuh/status/<encoded-pane-id>.json
```

Current schema:

```json
{
  "pane_id": "%45",
  "agent": "pi",
  "state": "working",
  "message": "optional detail",
  "updated_at": "2026-06-10T12:00:00.000Z"
}
```

Required fields:

- `pane_id`: tmux pane id, e.g. `%45`
- `agent`: stable adapter name, e.g. `pi`, `claude`, `codex`
- `state`: one of `working`, `blocked`, `idle`
- `updated_at`: ISO timestamp

Optional fields:

- `message`: short human-readable detail, especially for blocked/error states

## File writing rules

Adapters should write atomically:

1. write `<target>.<pid>.tmp`
2. rename temp file to final path

This prevents suphuh from reading partial JSON.

Adapters should remove their file on session shutdown when possible. suphuh must still tolerate stale files because agents can be killed abruptly.

## Adding a new agent adapter

1. Learn the agent's lifecycle hooks/protocol.
2. Identify how to get the tmux pane id.
3. Map lifecycle events to `working`, `blocked`, and `idle`.
4. Implement an adapter under `internal/integrations/<agent>/` or equivalent packaged location.
5. Add an installer command, e.g. `suphuh install-hook <agent>`.
6. Add docs describing how to reload/restart that agent.
7. Add tests for status rendering and stale/missing reports.

## ACP/RPC note

Protocol integrations like ACP or Pi RPC are useful when a host launches and owns the agent process. suphuh currently observes already-running interactive panes, so hook-based status reporting is safer: it does not interfere with the user's terminal UI.
