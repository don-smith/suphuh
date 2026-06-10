# Backlog

## Near-term

- Add a stable metadata header to the right preview pane with path/session/command.
- Add stale-status handling for old status report files.
- Explore lightweight animations/spinners for active agents.
- Add explicit tracking/marking so the list can focus on agent panes instead of every pane.
- Improve process identification with per-agent adapters.
- Add a `suphuh install-hook pi` helper for installing the Pi status extension.

## Design pass

- Refine color palette beyond the current purple/gray prototype.
- Make agent rows more visually scannable.
- Consider icons or short labels for agent type/status.
- Keep the popup fast, calm, and modal rather than turning it into a full dashboard.

## Agent adapters to investigate

- Pi
- Claude Code
- Codex
- Aider
- Goose
- OpenCode

For each adapter, learn:

- how to identify the process reliably
- whether it exposes hooks/events/status files
- what output patterns indicate waiting for user input
- what failure/completion states look like
