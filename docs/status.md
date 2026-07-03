# Agent status strategy

## ACP/RPC notes

ACP is worth tracking, but it is not the best first integration path for already-running interactive panes.

For Pi specifically, the installed CLI currently documents `--mode rpc`, not ACP. RPC is stdin/stdout JSONL and is excellent when a host application launches and owns the agent process. suphuh's current model is different: it observes agents that are already running interactively inside tmux panes. Sending JSON into those panes would interfere with the user-facing TUI.

So the first status path is hook/event based:

1. Agent-specific extension/hook observes lifecycle events.
2. Hook writes a tiny status report keyed by tmux pane id.
3. suphuh reads those reports and renders status indicators.

This works well with tmux because interactive panes already expose `TMUX_PANE` to child processes.

## Pi integration

Install the Pi extension with:

```sh
suphuh install-hook pi
```

This writes:

```text
~/.pi/agent/extensions/suphuh-status.ts
```

Existing Pi sessions need `/reload` or a restart. New Pi sessions load it automatically.

It writes files like:

```text
~/.suphuh/status/pct_45.json
```

Example report:

```json
{
  "pane_id": "%45",
  "agent": "pi",
  "state": "working",
  "session_name": "API review",
  "branch": "feature/api-review",
  "updated_at": "2026-06-10T12:00:00.000Z"
}
```

States:

- `working`
- `waiting`
- `idle`

Optional display metadata:

- `session_name`: Pi's `/name` or `--name` value, if set.
- `branch`: Git branch for Pi's current working directory, if available. The TUI uses this only when `session_name` is empty, and leaves `main` blank. If this field is missing, suphuh also computes the fallback branch directly from the tmux pane's current path.

For a quick manual test without waiting for an agent run, create a report for a pane:

```sh
mkdir -p ~/.suphuh/status
printf '%s\n' '{"pane_id":"%45","agent":"pi","state":"waiting","updated_at":"'"$(date -u +%Y-%m-%dT%H:%M:%SZ)"'"}' > ~/.suphuh/status/pct_45.json
```

Replace `%45`/`pct_45` with the pane id you want to test.

## UI indicators

Current glyphs:

- animated spinner — `working`
- pulsing `?` — `waiting`
- `✓` — `idle`
- `·` — no status report

## Future adapters

Claude Code, Codex, Aider, Goose, and OpenCode should each get their own adapter strategy. Prefer official hooks/protocols when available. Use pane-output heuristics only as a fallback.
