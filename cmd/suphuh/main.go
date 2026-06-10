package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"suphuh/internal/tmux"
	"suphuh/internal/tui"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	panes, err := tmux.ListPanes(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "suphuh: %v\n", err)
		os.Exit(1)
	}

	if len(os.Args) > 1 && os.Args[1] == "--list" {
		printPanes(panes)
		return
	}

	if err := tui.Run(panes); err != nil {
		fmt.Fprintf(os.Stderr, "suphuh: %v\n", err)
		os.Exit(1)
	}
}

func printPanes(panes []tmux.Pane) {
	if len(panes) == 0 {
		fmt.Println("No tmux panes found.")
		return
	}

	fmt.Printf("%-8s %-18s %-7s %-18s %-8s %s\n", "PANE", "SESSION", "WINDOW", "COMMAND", "STATE", "PATH")
	fmt.Printf("%-8s %-18s %-7s %-18s %-8s %s\n", strings.Repeat("-", 8), strings.Repeat("-", 18), strings.Repeat("-", 7), strings.Repeat("-", 18), strings.Repeat("-", 8), strings.Repeat("-", 30))

	for _, pane := range panes {
		state := "alive"
		if pane.Dead {
			state = "dead"
		}

		window := fmt.Sprintf("%d.%d", pane.WindowIndex, pane.PaneIndex)
		fmt.Printf("%-8s %-18s %-7s %-18s %-8s %s\n",
			pane.PaneID,
			truncate(pane.SessionName, 18),
			window,
			truncate(pane.CurrentCommand, 18),
			state,
			shortPath(pane.CurrentPath),
		)
	}
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	if max <= 1 {
		return s[:max]
	}
	return s[:max-1] + "…"
}

func shortPath(path string) string {
	home, err := os.UserHomeDir()
	if err == nil && strings.HasPrefix(path, home) {
		path = "~" + strings.TrimPrefix(path, home)
	}

	base := filepath.Base(path)
	parent := filepath.Base(filepath.Dir(path))
	if parent == "." || parent == string(filepath.Separator) || parent == "~" {
		return path
	}
	return filepath.Join("…", parent, base)
}
