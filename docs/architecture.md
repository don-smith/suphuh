# Architecture notes

## Recommended stack

Use Go with Charmbracelet Bubble Tea/Lip Gloss/Bubbles for the TUI.

Reasons:

- Bubble Tea is the library behind many polished terminal apps.
- Go produces a single portable binary with fast startup, which matters for tmux popups.
- Tmux integration is naturally done by shelling out to `tmux` commands.
- Python TUI libraries are viable, but packaging/startup/dependency isolation tend to create more friction for this specific use case.

## Main components

- `internal/tmux`: thin wrapper around tmux commands.
- `internal/model`: agent instance and persisted state types.
- `internal/status`: status heuristics for pane activity/output.
- `internal/tui`: Bubble Tea application.
- `cmd/suphuh`: CLI entry point.

## Tmux data model

A tracked agent points at a tmux pane identity:

- session name/id
- window index/id/name
- pane id/index/title/current command/current path

Prefer tmux pane IDs like `%12` for stability while the pane lives. Persist enough metadata to render helpful labels and to detect stale entries.

## Candidate tmux commands

List panes:

```sh
tmux list-panes -a -F '#{session_id}\t#{session_name}\t#{window_id}\t#{window_index}\t#{window_name}\t#{pane_id}\t#{pane_index}\t#{pane_title}\t#{pane_current_command}\t#{pane_current_path}\t#{pane_dead}'
```

Preview selected pane:

```sh
tmux capture-pane -p -t '%12' -S -80
```

Jump to pane:

```sh
tmux switch-client -t '<session_name>'
tmux select-window -t '<session_name>:<window_index>'
tmux select-pane -t '%12'
```

Open as popup:

```tmux
bind-key A display-popup -E -w 90% -h 80% 'suphuh'
```

## Status strategy

Start with pragmatic heuristics:

- `gone`: tracked pane no longer exists.
- `active`: pane output changed recently or command appears busy.
- `waiting`: output contains common prompts such as permission/confirmation/input-needed text.
- `idle`: no output change for a configured threshold.
- `unknown`: no reliable signal yet.

Later, add optional adapters for specific agents.
