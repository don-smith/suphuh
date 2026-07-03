package tui

import (
	"fmt"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/muesli/termenv"

	"github.com/don-smith/suphuh/internal/status"
	"github.com/don-smith/suphuh/internal/tmux"
)

func TestViewContainsDesignElements(t *testing.T) {
	m := New([]tmux.Pane{
		{SessionName: "suphuh", CurrentCommand: "pi", DisplayCommand: "pi", CurrentPath: "/Users/don/projects/suphuh", PaneID: "%1"},
		{SessionName: "notes", CurrentCommand: "zsh", DisplayCommand: "zsh", CurrentPath: "/Users/don", PaneID: "%2"},
	})
	m.width = 100
	m.height = 30
	m.preview = "hello\nworld\n"
	m.updatePreviewViewport()

	plain := ansi.Strip(m.View())
	for _, want := range []string{"sup?huh?", "2 panes", "pi", "suphuh", "/Users/don/projects/suphuh"} {
		if !strings.Contains(plain, want) {
			t.Fatalf("view missing %q\n%s", want, numbered(plain))
		}
	}
}

func TestViewVisualSnapshot(t *testing.T) {
	m := New([]tmux.Pane{
		{SessionName: "suphuh", CurrentCommand: "pi", DisplayCommand: "pi", CurrentPath: "/Users/don/projects/suphuh", PaneID: "%1"},
		{SessionName: "lois", CurrentCommand: "pi", DisplayCommand: "pi", CurrentPath: "/Users/don/hypr/clients/lois", PaneID: "%2"},
		{SessionName: "zsh", CurrentCommand: "zsh", DisplayCommand: "zsh", CurrentPath: "/Users/don", PaneID: "%3"},
	})
	m.width = 100
	m.height = 30
	m.viewMode = ViewAgentsFirst
	m.preview = strings.Repeat("preview line\n", 8)
	m.updatePreviewViewport()

	view := m.View()
	assertStableView(t, view, 100, 30)
	t.Logf("rendered TUI snapshot:\n%s", numbered(ansi.Strip(view)))
}

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

func TestStatusGlyphRendersWaitingQuestionMark(t *testing.T) {
	m := New([]tmux.Pane{{
		SessionName:    "suphuh",
		CurrentCommand: "pi",
		DisplayCommand: "pi",
		PaneID:         "%1",
		HasStatus:      true,
		Status:         status.Report{State: status.Waiting},
	}})

	got := ansi.Strip(m.statusGlyph(m.panes[0], false))
	if got != "?" {
		t.Fatalf("statusGlyph() = %q, want ?", got)
	}
}

func TestViewRendersWaitingStatusWithoutChangingDimensions(t *testing.T) {
	m := New([]tmux.Pane{
		{SessionName: "suphuh", CurrentCommand: "pi", DisplayCommand: "pi", CurrentPath: "/Users/don/projects/suphuh", PaneID: "%1", HasStatus: true, Status: status.Report{State: status.Waiting}},
		{SessionName: "notes", CurrentCommand: "zsh", DisplayCommand: "zsh", CurrentPath: "/Users/don", PaneID: "%2"},
	})
	m.width = 100
	m.height = 30
	m.preview = "waiting\n"
	m.updatePreviewViewport()

	view := m.View()
	assertStableView(t, view, 100, 30)
	if !strings.Contains(ansi.Strip(view), "? suphuh") {
		t.Fatalf("view missing waiting question mark\n%s", numbered(ansi.Strip(view)))
	}
}

func TestRenderListShowsPiSessionNameInProcessColumn(t *testing.T) {
	m := New([]tmux.Pane{{
		SessionName:    "work",
		CurrentCommand: "pi",
		DisplayCommand: "pi",
		PaneID:         "%1",
		HasStatus:      true,
		Status:         status.Report{Agent: "pi", State: status.Idle, SessionName: "API review"},
	}})

	plain := ansi.Strip(m.renderList(60, 5))
	if !strings.Contains(plain, "pi API review") {
		t.Fatalf("list missing pi session name\n%s", numbered(plain))
	}
}

func TestRenderListFallsBackToNonMainBranchForUnnamedPiSession(t *testing.T) {
	m := New([]tmux.Pane{{
		SessionName:    "work",
		CurrentCommand: "pi",
		DisplayCommand: "pi",
		PaneID:         "%1",
		HasStatus:      true,
		Status:         status.Report{Agent: "pi", State: status.Idle, Branch: "feature/api-review"},
	}})

	plain := ansi.Strip(m.renderList(60, 5))
	if !strings.Contains(plain, "pi feature/api-review") {
		t.Fatalf("list missing pi branch fallback\n%s", numbered(plain))
	}
}

func TestRenderListLeavesMainBranchUnnamedPiSessionBlank(t *testing.T) {
	m := New([]tmux.Pane{{
		SessionName:    "work",
		CurrentCommand: "pi",
		DisplayCommand: "pi",
		PaneID:         "%1",
		HasStatus:      true,
		Status:         status.Report{Agent: "pi", State: status.Idle, Branch: "main"},
	}})

	plain := ansi.Strip(m.renderList(60, 5))
	if strings.Contains(plain, "main") {
		t.Fatalf("list should not show main branch for unnamed Pi session\n%s", numbered(plain))
	}
}

func TestRenderListUsesPaneBranchWhenStatusBranchMissing(t *testing.T) {
	m := New([]tmux.Pane{{
		SessionName:    "work",
		CurrentCommand: "pi",
		DisplayCommand: "pi",
		PaneID:         "%1",
		HasStatus:      true,
		CurrentBranch:  "feature/from-pane-path",
		Status:         status.Report{Agent: "pi", State: status.Idle},
	}})

	plain := ansi.Strip(m.renderList(60, 5))
	if !strings.Contains(plain, "pi feature/from-pane-path") {
		t.Fatalf("list missing pane branch fallback\n%s", numbered(plain))
	}
}

func TestRenderListDimsPiBranchSuffix(t *testing.T) {
	lipgloss.SetColorProfile(termenv.ANSI256)
	defer lipgloss.SetColorProfile(termenv.Ascii)

	m := New([]tmux.Pane{
		{SessionName: "shell", CurrentCommand: "zsh", DisplayCommand: "zsh", PaneID: "%1"},
		{
			SessionName:    "work",
			CurrentCommand: "pi",
			DisplayCommand: "pi",
			PaneID:         "%2",
			HasStatus:      true,
			CurrentBranch:  "feature/branch-label",
			Status:         status.Report{Agent: "pi", State: status.Idle},
		},
	})

	for i, pane := range m.panes {
		if displayCommand(pane) != "pi" {
			m.selected = i
			break
		}
	}

	rendered := m.renderList(80, 5)
	if !strings.Contains(rendered, "pi "+headerMetaStyle.Render("feature/branch-label")) {
		t.Fatalf("list should dim Pi branch suffix\nraw:\n%q\nplain:\n%s", rendered, numbered(ansi.Strip(rendered)))
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
