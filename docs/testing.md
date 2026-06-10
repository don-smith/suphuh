# Testing notes

## TUI layout snapshot testing

Terminal UIs are plain text plus ANSI escape sequences, so we can test a lot of behavior without visual screenshots.

The useful pattern is:

1. Construct the Bubble Tea model directly.
2. Set a deterministic terminal size on the model.
3. Populate the model with representative data.
4. Call `View()`.
5. Assert properties of the rendered string.

For layout regressions, the most important assertions are:

- rendered line count equals the terminal height
- every rendered line width is less than or equal to the terminal width
- dimensions remain stable when selection/content changes
- long output does not wrap and push other UI elements around

See `internal/tui/app_test.go`.

## Why this matters

This project is a tmux popup app. Small layout bugs are very visible: panes jump, titles disappear, help text moves, or the popup starts scrolling. Snapshot-style TUI tests give us a fast feedback loop for those issues.

The first version of this test caught a real bug where long preview content made the app render 52 lines into a 30-line viewport. That was caused by Lip Gloss wrapping inside a styled box after our layout math had already run.

## Helpful techniques

Use Charm's ANSI width helpers instead of raw string length:

```go
ansi.StringWidth(line)
```

This handles ANSI escape sequences and wide characters better than `len(line)`.

When debugging a rendered view, print line numbers and measured widths:

```text
01 100 sup?huh?                  3 panes • 2 agents • 0 working
02 100 ╭────╮╭────╮
03 100 │ ...││ ...│
```

That makes invisible layout problems obvious.

For a deliberate visual-ish snapshot, run:

```sh
go test ./internal/tui -run TestViewVisualSnapshot -v
```

This logs a stripped, numbered render of the TUI. It is useful for checking title bars, borders, grouping separators, and help text without taking a screenshot.

## What to test after UI changes

Any change to styles, borders, padding, truncation, viewport sizing, or help text should run:

```sh
go test ./...
```

If the UI changes materially, add or update a test case in `internal/tui/app_test.go` with the problematic content shape.
