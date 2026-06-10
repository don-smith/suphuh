# suphuh

A tmux-first terminal popup for quickly checking on coding agents.

The goal is intentionally small: press one tmux key binding, see every tracked agent instance, preview its pane output, and jump to it if needed.

## Current prototype

```sh
go run ./cmd/suphuh
```

Inside the TUI:

- `j`/`k` or arrow keys move through tmux panes.
- `v` switches between `all` and `agents first` views.
- The right side previews the selected pane via `tmux capture-pane`.
- Selection and view mode persist between popup invocations.
- `Enter` jumps to the selected pane.
- `q`/`Esc` closes.

Plain listing mode is also available:

```sh
go run ./cmd/suphuh --list
```

Install and use from tmux:

```sh
go install ./cmd/suphuh
```

From a shell, invoke tmux commands with the `tmux` executable:

```sh
tmux display-popup -E -w 55% -h 65% '/Users/don/go/bin/suphuh'
```

In `.tmux.conf`, omit the leading `tmux`:

```tmux
bind-key K display-popup -E -w 55% -h 65% '/Users/don/go/bin/suphuh'
```

See [`docs/product.md`](docs/product.md), [`docs/architecture.md`](docs/architecture.md), [`docs/testing.md`](docs/testing.md), [`docs/adapter-interface.md`](docs/adapter-interface.md), [`docs/status.md`](docs/status.md), and [`docs/backlog.md`](docs/backlog.md).
