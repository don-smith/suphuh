package process

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

// Info is a single process table row.
type Info struct {
	PID  int
	PPID int
	Comm string
	Args string
}

// LabelsForRoots returns friendly labels for process trees rooted at the given PIDs.
func LabelsForRoots(ctx context.Context, roots []int) (map[int]string, error) {
	procs, err := List(ctx)
	if err != nil {
		return nil, err
	}

	children := make(map[int][]Info)
	byPID := make(map[int]Info)
	for _, proc := range procs {
		byPID[proc.PID] = proc
		children[proc.PPID] = append(children[proc.PPID], proc)
	}

	labels := make(map[int]string, len(roots))
	for _, root := range roots {
		labels[root] = LabelForTree(root, byPID, children)
	}
	return labels, nil
}

// List returns a snapshot of the current process table.
func List(ctx context.Context) ([]Info, error) {
	cmd := exec.CommandContext(ctx, "ps", "-axo", "pid=,ppid=,comm=,args=")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("ps command failed: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	procs := make([]Info, 0, len(lines))
	for _, line := range lines {
		proc, ok := parsePSLine(line)
		if ok {
			procs = append(procs, proc)
		}
	}
	return procs, nil
}

func LabelForTree(root int, byPID map[int]Info, children map[int][]Info) string {
	candidates := descendants(root, children)
	if len(candidates) == 0 {
		if proc, ok := byPID[root]; ok {
			return FriendlyName(proc)
		}
		return ""
	}

	best := candidates[0]
	bestScore := score(best)
	for _, candidate := range candidates[1:] {
		candidateScore := score(candidate)
		if candidateScore >= bestScore {
			best = candidate
			bestScore = candidateScore
		}
	}

	return FriendlyName(best)
}

func FriendlyName(proc Info) string {
	comm := strings.TrimPrefix(strings.ToLower(filepath.Base(proc.Comm)), "-")
	args := strings.ToLower(proc.Args)

	switch {
	case comm == "pi" || strings.Contains(args, "pi-coding-agent"):
		return "pi"
	case comm == "claude" || strings.Contains(args, "claude-code") || strings.Contains(args, "@anthropic-ai/claude"):
		return "claude"
	case comm == "codex" || strings.Contains(args, "openai/codex"):
		return "codex"
	case comm == "aider" || strings.Contains(args, "aider"):
		return "aider"
	case comm == "goose" || strings.Contains(args, "block/goose"):
		return "goose"
	case comm == "opencode" || strings.Contains(args, "opencode"):
		return "opencode"
	}

	if comm != "" {
		return comm
	}
	fields := strings.Fields(proc.Args)
	if len(fields) > 0 {
		return fields[0]
	}
	return ""
}

func score(proc Info) int {
	name := FriendlyName(proc)
	switch name {
	case "pi", "claude", "codex", "aider", "goose", "opencode":
		return 100
	case "node", "go", "python", "python3", "zsh", "bash", "sh":
		return 10
	default:
		return 50
	}
}

func descendants(root int, children map[int][]Info) []Info {
	var out []Info
	queue := append([]Info(nil), children[root]...)
	for len(queue) > 0 {
		proc := queue[0]
		queue = queue[1:]
		out = append(out, proc)
		queue = append(queue, children[proc.PID]...)
	}
	return out
}

func parsePSLine(line string) (Info, bool) {
	fields := strings.Fields(line)
	if len(fields) < 3 {
		return Info{}, false
	}

	pid, err := strconv.Atoi(fields[0])
	if err != nil {
		return Info{}, false
	}
	ppid, err := strconv.Atoi(fields[1])
	if err != nil {
		return Info{}, false
	}

	args := ""
	if len(fields) > 3 {
		args = strings.Join(fields[3:], " ")
	}

	return Info{PID: pid, PPID: ppid, Comm: fields[2], Args: args}, true
}
