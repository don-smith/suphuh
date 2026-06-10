package tmux

import (
	"context"
	"fmt"
	"os/exec"
)

// JumpToPane switches the attached tmux client to the pane.
func JumpToPane(ctx context.Context, pane Pane) error {
	commands := [][]string{
		{"switch-client", "-t", pane.SessionName},
		{"select-window", "-t", fmt.Sprintf("%s:%d", pane.SessionName, pane.WindowIndex)},
		{"select-pane", "-t", pane.PaneID},
	}

	for _, args := range commands {
		cmd := exec.CommandContext(ctx, "tmux", args...)
		if err := cmd.Run(); err != nil {
			return tmuxCommandError(err)
		}
	}

	return nil
}
