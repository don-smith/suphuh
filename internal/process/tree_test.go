package process

import "testing"

func TestLabelForTreePrefersAgentChild(t *testing.T) {
	byPID := map[int]Info{
		1: {PID: 1, PPID: 0, Comm: "zsh", Args: "-zsh"},
		2: {PID: 2, PPID: 1, Comm: "pi", Args: "pi"},
	}
	children := map[int][]Info{
		1: {byPID[2]},
	}

	if got := LabelForTree(1, byPID, children); got != "pi" {
		t.Fatalf("LabelForTree() = %q, want %q", got, "pi")
	}
}

func TestFriendlyNameRecognizesClaudeNodeArgs(t *testing.T) {
	proc := Info{PID: 2, PPID: 1, Comm: "node", Args: "node /x/@anthropic-ai/claude-code/cli.js"}
	if got := FriendlyName(proc); got != "claude" {
		t.Fatalf("FriendlyName() = %q, want %q", got, "claude")
	}
}
