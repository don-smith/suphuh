package status

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

func RemoveForPane(paneID string) error {
	path, err := reportPath(paneID)
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func Cleanup(activePaneIDs map[string]bool) error {
	dir, err := statusDir()
	if err != nil {
		return err
	}

	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		path := filepath.Join(dir, entry.Name())
		paneID, ok := paneIDFromReport(path)
		if !ok || !activePaneIDs[paneID] {
			_ = os.Remove(path)
		}
	}
	return nil
}

func statusDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".suphuh", "status"), nil
}

func paneIDFromReport(path string) (string, bool) {
	data, err := os.ReadFile(path)
	if err == nil {
		var report Report
		if json.Unmarshal(data, &report) == nil && report.PaneID != "" {
			return report.PaneID, true
		}
	}

	name := strings.TrimSuffix(filepath.Base(path), ".json")
	if strings.HasPrefix(name, "pct_") {
		return "%" + strings.TrimPrefix(name, "pct_"), true
	}
	return "", false
}
