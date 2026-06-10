package tui

import (
	"fmt"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"suphuh/internal/tmux"
)

func TestViewMaintainsStableDimensions(t *testing.T) {
	panes := []tmux.Pane{
		{SessionName: "short", CurrentCommand: "zsh", WindowIndex: 1, PaneIndex: 1, CurrentPath: "/tmp", PaneID: "%1"},
		{SessionName: "very-long-session-name-that-will-truncate", CurrentCommand: "node", WindowIndex: 1, PaneIndex: 2, CurrentPath: "/tmp", PaneID: "%2"},
	}

	m := New(panes)
	m.width = 100
	m.height = 30
	m.preview = strings.Repeat("short line\n", 3)
	m.updatePreviewViewport()
	viewA := m.View()

	m.selected = 1
	m.preview = strings.Repeat("this is a very very very very very very very very very very very long line\n", 80)
	m.updatePreviewViewport()
	m.previewViewport.GotoBottom()
	viewB := m.View()

	assertStableView(t, viewA, 100, 30)
	assertStableView(t, viewB, 100, 30)

	if lineCount(viewA) != lineCount(viewB) {
		t.Fatalf("line count changed: %d vs %d\nA:\n%s\nB:\n%s", lineCount(viewA), lineCount(viewB), viewA, viewB)
	}
}

func assertStableView(t *testing.T, view string, width int, height int) {
	t.Helper()
	lines := strings.Split(strings.TrimRight(view, "\n"), "\n")
	if len(lines) != height {
		t.Fatalf("view height mismatch: got %d lines, want %d\n%s", len(lines), height, numbered(view))
	}
	for i, line := range lines {
		if w := ansi.StringWidth(line); w > width {
			t.Fatalf("line %d too wide: got %d, want <= %d\n%s", i+1, w, width, numbered(view))
		}
	}
}

func lineCount(s string) int {
	return len(strings.Split(strings.TrimRight(s, "\n"), "\n"))
}

func numbered(s string) string {
	lines := strings.Split(strings.TrimRight(s, "\n"), "\n")
	var b strings.Builder
	for i, line := range lines {
		fmt.Fprintf(&b, "%02d %03d %s\n", i+1, ansi.StringWidth(line), line)
	}
	return b.String()
}
