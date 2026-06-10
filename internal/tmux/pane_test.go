package tmux

import "testing"

func TestParsePaneLine(t *testing.T) {
	line := "$27\tsuphuh\t@31\t1\tnode\t%45\t2\tπ - suphuh\tnode\t/Users/don/projects/suphuh\t0"

	pane, err := parsePaneLine(line)
	if err != nil {
		t.Fatalf("parsePaneLine() error = %v", err)
	}

	if pane.SessionID != "$27" || pane.SessionName != "suphuh" || pane.WindowID != "@31" || pane.WindowIndex != 1 || pane.WindowName != "node" || pane.PaneID != "%45" || pane.PaneIndex != 2 || pane.PaneTitle != "π - suphuh" || pane.CurrentCommand != "node" || pane.CurrentPath != "/Users/don/projects/suphuh" || pane.Dead {
		t.Fatalf("unexpected pane: %#v", pane)
	}
}

func TestParsePaneLineRejectsUnexpectedFieldCount(t *testing.T) {
	_, err := parsePaneLine("too\tfew")
	if err == nil {
		t.Fatal("parsePaneLine() error = nil, want error")
	}
}
