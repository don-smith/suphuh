# Rename blocked status to waiting with pulsing question-mark glyph

## Quick Reference
- **Date:** 2026-06-26
- **Key files:** `internal/status/status.go`, `internal/integrations/pi/suphuh-status.ts`, `internal/tui/app.go`
- **Key concepts:** waiting state, tool execution debounce, pulsing glyph, Pi lifecycle events
- **One-line summary:** Renamed the human-attention status from `blocked` to `waiting` with a pulsing `?` TUI glyph, detecting waiting state from Pi's `tool_execution_start`/`tool_execution_end` events.

## How It Works Now

The status pipeline has three canonical states: `working`, `waiting`, and `idle`. No legacy aliases — `blocked` is gone entirely.

**Pi adapter** (`internal/integrations/pi/suphuh-status.ts`): Detects waiting state via tool execution timing. On `tool_execution_start`, starts a 200ms debounce timer. If the timer fires before `tool_execution_end`, the tool is likely blocking for user input (interactive tools like `ask_user_question` run until the user responds; fast tools like `read`/`grep` finish in <100ms and never trigger the debounce). When the debounced waiting state activates, the adapter publishes `"waiting"`. When all tools finish, it returns to `"working"`. Publishes `"idle"` at session start and after `agent_end`.

**Go status reader** (`internal/status/status.go`): `LoadForPane` reads JSON, defaults empty fields, returns the report. No normalization needed — only three states exist.

**TUI rendering** (`internal/tui/app.go`): `statusGlyph()` switches on `pane.Status.State` using package constants. `Waiting` renders a single-cell `?` with a pulsing color cycle (muted → amber → accent → amber), driven by the existing `spinnerFrame` refresh tick. `Working` renders the animated spinner, `idle` renders `✓`.

## Key Decisions

### Canonical state name is `waiting`

**What:** Use `waiting` for any state needing human attention (prompts, approvals, questions).
**Why:** `blocked` was misleading — the agent isn't blocked, it's waiting for the user. A separate `question` state would duplicate the same concept.
**Regression risk:** low

### Detect waiting via tool execution timing, not a specific event

**What:** The adapter listens to `tool_execution_start`/`tool_execution_end` with a 200ms debounce. Tools that finish quickly (<200ms) never trigger waiting; tools that hold execution open (e.g. `ask_user_question` waiting for a human response) trigger the debounce and publish `"waiting"`.
**Why:** Pi doesn't emit a "waiting for user" event. The `herdr:blocked` event came from the now-uninstalled Herder tool and was dead code. Tool execution timing is a tool-name-agnostic heuristic that works for any interactive tool.
**Regression risk:** low — the 200ms threshold has been chosen to cleanly separate fast tools from interactive ones.

### No legacy `blocked` compatibility

**What:** Removed the `Blocked` constant, `normalizeState()` function, and all legacy alias code.
**Why:** No legacy `"blocked"` status files exist in any repo. Herder (the only source of `herdr:blocked` events) was uninstalled. There is nothing to be compatible with.
**Regression risk:** low

### Waiting glyph is a pulsing `?`, not a `◆` diamond

**What:** The TUI renders a one-cell `?` that pulses through amber/magenta on the refresh tick.
**Why:** Visually distinct from both the spinner and idle checkmark. Single-cell width preserves the existing pane-list layout.
**Regression risk:** low

## Constraints & Gotchas

- **The 200ms debounce is load-bearing.** If future Pi versions add sub-200ms interactive tools, they won't trigger waiting. The threshold can be tuned but shouldn't be set so low that fast system tools false-positive.
- **`tool_execution_start` and `tool_execution_end` are Pi lifecycle events**, not event-bus channels. They use `pi.on(...)`, not `pi.events.on(...)`.
- **`spinnerFrame` drives the pulsing.** The waiting glyph animation reuses the existing 200ms TUI refresh tick.
- **Selected rows return raw `?` without styling.** Consistent with how spinner and idle glyphs handle selection.
- **Non-agent panes never show status glyphs**, even when stale status files exist. Unchanged.

## Background Context

suphuh's original status model had `working`, `blocked`, and `idle`. The `blocked` state was originally signaled by Herder's `herdr:blocked` event. After Herder was uninstalled, the event listener became dead code — the adapter could only publish `working` and `idle` in practice. This change replaces the dead event with tool execution timing and drops the `blocked` name entirely in favor of `waiting`.