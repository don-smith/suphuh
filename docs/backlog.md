# Backlog

## Near-term

- Add stale-status handling for old status report files.
- Add explicit tracking/marking so the list can focus on agent panes instead of every pane.
- Improve process identification with per-agent adapters.

## Done

- Install Pi status extension with `suphuh install-hook pi`.
- Live refresh pane previews and status while popup is open.
- Persist selected pane and view mode between popup invocations.
- Add `all` and `agents first` views.
- Highlight the full selected row in the pane list.
- Hide status glyphs for non-agent panes, even when stale status files exist.
- Add a stable metadata header to the right preview pane with path/session/command.
- Add lightweight spinner animation for working agents.

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
