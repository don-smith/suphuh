package appstate

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

type State struct {
	SelectedPaneID string `json:"selected_pane_id,omitempty"`
	View           string `json:"view,omitempty"`
	ArtIndex       int    `json:"art_index,omitempty"`
}

func Load() State {
	path, err := path()
	if err != nil {
		return State{}
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return State{}
	}

	var state State
	if err := json.Unmarshal(data, &state); err != nil {
		return State{}
	}
	return state
}

func Save(state State) error {
	path, err := path()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')

	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func path() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	if home == "" {
		return "", errors.New("empty home directory")
	}
	return filepath.Join(home, ".suphuh", "state.json"), nil
}
