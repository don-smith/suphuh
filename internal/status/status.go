package status

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type State string

const (
	Unknown State = "unknown"
	Working State = "working"
	Waiting State = "waiting"
	Idle    State = "idle"
)

type Report struct {
	PaneID    string    `json:"pane_id"`
	Agent     string    `json:"agent"`
	State     State     `json:"state"`
	Message   string    `json:"message,omitempty"`
	UpdatedAt time.Time `json:"updated_at"`
}

func LoadForPane(paneID string) (Report, bool) {
	path, err := reportPath(paneID)
	if err != nil {
		return Report{}, false
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return Report{}, false
	}

	var report Report
	if err := json.Unmarshal(data, &report); err != nil {
		return Report{}, false
	}
	if report.PaneID == "" {
		report.PaneID = paneID
	}
	if report.State == "" {
		report.State = Unknown
	}
	return report, true
}

func reportPath(paneID string) (string, error) {
	if paneID == "" {
		return "", errors.New("empty pane id")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".suphuh", "status", paneFileName(paneID)+".json"), nil
}

func paneFileName(paneID string) string {
	replacer := strings.NewReplacer("%", "pct_", "$", "session_", "/", "_", "\\", "_", ":", "_")
	return replacer.Replace(paneID)
}
