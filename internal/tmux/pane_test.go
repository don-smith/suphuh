package tmux

import (
	"context"
	"os/exec"
	"testing"
)

func TestParsePaneLine(t *testing.T) {
	line := "$27\tsuphuh\t@31\t1\tnode\t%45\t2\tπ - suphuh\t15516\t/dev/ttys008\tnode\t/Users/don/projects/suphuh\t0"

	pane, err := parsePaneLine(line)
	if err != nil {
		t.Fatalf("parsePaneLine() error = %v", err)
	}

	if pane.SessionID != "$27" || pane.SessionName != "suphuh" || pane.WindowID != "@31" || pane.WindowIndex != 1 || pane.WindowName != "node" || pane.PaneID != "%45" || pane.PaneIndex != 2 || pane.PaneTitle != "π - suphuh" || pane.PanePID != 15516 || pane.PaneTTY != "/dev/ttys008" || pane.CurrentCommand != "node" || pane.DisplayCommand != "node" || pane.CurrentPath != "/Users/don/projects/suphuh" || pane.Dead {
		t.Fatalf("unexpected pane: %#v", pane)
	}
}

func TestParsePaneLineRejectsUnexpectedFieldCount(t *testing.T) {
	_, err := parsePaneLine("too\tfew")
	if err == nil {
		t.Fatal("parsePaneLine() error = nil, want error")
	}
}

func TestEnrichGitBranchesPopulatesBranchFromPanePath(t *testing.T) {
	repo := t.TempDir()
	runGit(t, repo, "init")
	runGit(t, repo, "checkout", "-b", "feature/branch-label")

	panes := []Pane{{
		DisplayCommand: "pi",
		CurrentPath:    repo,
	}}

	enrichGitBranches(context.Background(), panes)

	if panes[0].CurrentBranch != "feature/branch-label" {
		t.Fatalf("CurrentBranch = %q, want %q", panes[0].CurrentBranch, "feature/branch-label")
	}
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, out)
	}
}
