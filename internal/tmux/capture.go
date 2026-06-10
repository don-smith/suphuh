package tmux

import (
	"context"
	"fmt"
	"os/exec"
)

// CapturePane returns recent visible/output history for a tmux pane.
func CapturePane(ctx context.Context, paneID string, lines int) (string, error) {
	if lines <= 0 {
		lines = 80
	}

	start := fmt.Sprintf("-%d", lines)
	cmd := exec.CommandContext(ctx, "tmux", "capture-pane", "-p", "-t", paneID, "-S", start)
	out, err := cmd.Output()
	if err != nil {
		return "", tmuxCommandError(err)
	}

	return string(out), nil
}
