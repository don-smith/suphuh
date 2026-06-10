package integrations

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
)

//go:embed pi/suphuh-status.ts
var files embed.FS

func Install(agent string) (string, error) {
	switch agent {
	case "pi":
		return InstallPi()
	default:
		return "", fmt.Errorf("unsupported agent %q", agent)
	}
}

func InstallPi() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	data, err := files.ReadFile("pi/suphuh-status.ts")
	if err != nil {
		return "", err
	}

	path := filepath.Join(home, ".pi", "agent", "extensions", "suphuh-status.ts")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", err
	}

	if err := os.WriteFile(path, data, 0o644); err != nil {
		return "", err
	}
	return path, nil
}
