package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/don-smith/suphuh/internal/integrations"
	"github.com/don-smith/suphuh/internal/tmux"
	"github.com/don-smith/suphuh/internal/tui"
)

func main() {
	if handled := handleCommand(os.Args[1:]); handled {
		return
	}

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

func handleCommand(args []string) bool {
	if len(args) == 0 {
		return false
	}

	switch args[0] {
	case "install-hook":
		if len(args) != 2 {
			fmt.Fprintln(os.Stderr, "usage: suphuh install-hook pi")
			os.Exit(2)
		}
		path, err := integrations.Install(args[1])
		if err != nil {
			fmt.Fprintf(os.Stderr, "suphuh: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Installed %s status hook: %s\n", args[1], path)
		fmt.Println("Restart Pi or run /reload in existing Pi sessions to load it.")
		return true
	case "help", "--help", "-h":
		printUsage()
		return true
	default:
		return false
	}
}

func printUsage() {
	fmt.Println("suphuh - tmux popup monitor for coding agents")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  suphuh                 Open TUI")
	fmt.Println("  suphuh --list          List tmux panes")
	fmt.Println("  suphuh install-hook pi Install Pi status extension")
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
			truncate(displayCommand(pane), 18),
			state,
			shortPath(pane.CurrentPath),
		)
	}
}

func displayCommand(pane tmux.Pane) string {
	if pane.DisplayCommand != "" {
		return pane.DisplayCommand
	}
	return pane.CurrentCommand
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
