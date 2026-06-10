# Product notes

## Core workflow

1. User presses a tmux binding, for example `prefix + A` or a global tmux key.
2. `suphuh` opens in a tmux popup.
3. Left pane shows tracked agent instances.
4. Right pane previews the selected tmux pane output.
5. User presses:
   - `j`/`k` or arrow keys to move.
   - `Enter` to jump to the selected agent pane.
   - `Esc`/`q` to close.

## Design principles

- Tmux is the operating environment.
- The UI should be discoverable; avoid requiring memorized CLI flags.
- CLI commands may exist, but the happy path should be the popup UI.
- Manual registration should be easy from inside the popup.
- Automatic discovery is desirable, but false confidence is worse than explicit tracking.
- The app should augment existing workflows, not replace tmux, lazygit, vim, or agent CLIs.

## First useful version

The first useful version can be simple:

- Discover tmux panes.
- Allow marking/unmarking panes as agent panes.
- Store tracked pane IDs in a small local state file.
- Show `tmux capture-pane` output for the selected pane.
- Jump with `tmux switch-client -t <session>` and `tmux select-window` / `select-pane`.

## Open product questions

- Should registration be global, per tmux server, per repo, or per user?
- Should the popup list all panes by default with filters, or only explicitly tracked panes?
- What agent status signals are reliable enough across Claude Code, Aider, Codex, Goose, etc.?
- Should status initially be heuristic-based from recent pane output/activity?
- Should there be optional per-agent adapters later?
- Can we derive better process labels than `node`/`go` from pane TTY process trees?
- Should the right preview always reserve a visible metadata header for path/session/command, regardless of captured pane content?
