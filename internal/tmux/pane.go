package tmux

import (
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"

	"github.com/don-smith/suphuh/internal/process"
	"github.com/don-smith/suphuh/internal/status"
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
	PanePID        int
	PaneTTY        string
	CurrentCommand string
	DisplayCommand string
	CurrentPath    string
	Status         status.Report
	HasStatus      bool
	Dead           bool
}

const listPanesFormat = "#{session_id}\t#{session_name}\t#{window_id}\t#{window_index}\t#{window_name}\t#{pane_id}\t#{pane_index}\t#{pane_title}\t#{pane_pid}\t#{pane_tty}\t#{pane_current_command}\t#{pane_current_path}\t#{pane_dead}"

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

	cleanupStaleStatuses(panes)
	enrichDisplayCommands(ctx, panes)
	return panes, nil
}

func parsePaneLine(line string) (Pane, error) {
	fields := strings.Split(line, "\t")
	if len(fields) != 13 {
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

	panePID, err := strconv.Atoi(fields[8])
	if err != nil {
		return Pane{}, fmt.Errorf("parse pane pid %q: %w", fields[8], err)
	}

	dead, err := strconv.ParseBool(fields[12])
	if err != nil {
		return Pane{}, fmt.Errorf("parse pane dead flag %q: %w", fields[12], err)
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
		PanePID:        panePID,
		PaneTTY:        fields[9],
		CurrentCommand: fields[10],
		DisplayCommand: fields[10],
		CurrentPath:    fields[11],
		Dead:           dead,
	}, nil
}

func enrichDisplayCommands(ctx context.Context, panes []Pane) {
	roots := make([]int, 0, len(panes))
	for i := range panes {
		panes[i].DisplayCommand = cleanCommandName(panes[i].CurrentCommand)
		if panes[i].PanePID > 0 && isAmbiguousRuntime(panes[i].CurrentCommand) {
			roots = append(roots, panes[i].PanePID)
		}
	}

	labels, err := process.LabelsForRoots(ctx, roots)
	if err != nil {
		return
	}

	for i := range panes {
		if label := labels[panes[i].PanePID]; label != "" && isAmbiguousRuntime(panes[i].CurrentCommand) {
			panes[i].DisplayCommand = label
		}
		if isKnownAgentCommand(panes[i].DisplayCommand) {
			panes[i].Status, panes[i].HasStatus = status.LoadForPane(panes[i].PaneID)
		} else {
			_ = status.RemoveForPane(panes[i].PaneID)
			panes[i].HasStatus = false
		}
	}
}

func cleanupStaleStatuses(panes []Pane) {
	active := make(map[string]bool, len(panes))
	for _, pane := range panes {
		active[pane.PaneID] = true
	}
	_ = status.Cleanup(active)
}

func isKnownAgentCommand(command string) bool {
	switch cleanCommandName(command) {
	case "pi", "claude", "codex", "aider", "goose", "opencode", "gemini":
		return true
	default:
		return false
	}
}

func isAmbiguousRuntime(command string) bool {
	switch cleanCommandName(command) {
	case "node", "go", "python", "python3", "deno", "bun":
		return true
	default:
		return false
	}
}

func cleanCommandName(command string) string {
	return strings.TrimPrefix(command, "-")
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
