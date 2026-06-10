package tmux

import (
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// Pane describes a tmux pane as reported by `tmux list-panes`.
type Pane struct {
	SessionID      string
	SessionName    string
	WindowID       string
	WindowIndex    int
	WindowName     string
	PaneID         string
	PaneIndex      int
	PaneTitle      string
	CurrentCommand string
	CurrentPath    string
	Dead           bool
}

const listPanesFormat = "#{session_id}\t#{session_name}\t#{window_id}\t#{window_index}\t#{window_name}\t#{pane_id}\t#{pane_index}\t#{pane_title}\t#{pane_current_command}\t#{pane_current_path}\t#{pane_dead}"

// ListPanes returns all panes known to the current tmux server.
func ListPanes(ctx context.Context) ([]Pane, error) {
	cmd := exec.CommandContext(ctx, "tmux", "list-panes", "-a", "-F", listPanesFormat)
	out, err := cmd.Output()
	if err != nil {
		return nil, tmuxCommandError(err)
	}

	text := strings.TrimRight(string(out), "\n")
	if text == "" {
		return nil, nil
	}

	lines := strings.Split(text, "\n")
	panes := make([]Pane, 0, len(lines))
	for _, line := range lines {
		pane, err := parsePaneLine(line)
		if err != nil {
			return nil, err
		}
		panes = append(panes, pane)
	}

	return panes, nil
}

func parsePaneLine(line string) (Pane, error) {
	fields := strings.Split(line, "\t")
	if len(fields) != 11 {
		return Pane{}, fmt.Errorf("unexpected tmux pane line with %d fields: %q", len(fields), line)
	}

	windowIndex, err := strconv.Atoi(fields[3])
	if err != nil {
		return Pane{}, fmt.Errorf("parse window index %q: %w", fields[3], err)
	}

	paneIndex, err := strconv.Atoi(fields[6])
	if err != nil {
		return Pane{}, fmt.Errorf("parse pane index %q: %w", fields[6], err)
	}

	dead, err := strconv.ParseBool(fields[10])
	if err != nil {
		return Pane{}, fmt.Errorf("parse pane dead flag %q: %w", fields[10], err)
	}

	return Pane{
		SessionID:      fields[0],
		SessionName:    fields[1],
		WindowID:       fields[2],
		WindowIndex:    windowIndex,
		WindowName:     fields[4],
		PaneID:         fields[5],
		PaneIndex:      paneIndex,
		PaneTitle:      fields[7],
		CurrentCommand: fields[8],
		CurrentPath:    fields[9],
		Dead:           dead,
	}, nil
}

func tmuxCommandError(err error) error {
	if exitErr, ok := err.(*exec.ExitError); ok {
		stderr := strings.TrimSpace(string(exitErr.Stderr))
		if stderr != "" {
			return fmt.Errorf("tmux command failed: %s", stderr)
		}
	}
	return fmt.Errorf("tmux command failed: %w", err)
}
