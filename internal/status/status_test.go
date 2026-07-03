package status

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadForPaneReadsCanonicalWaiting(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	path, err := reportPath("%45")
	if err != nil {
		t.Fatalf("reportPath() error = %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	data := fmt.Sprintf(`{"pane_id":"%%45","agent":"pi","state":"%s","updated_at":"2026-06-26T00:00:00Z"}`+"\n", Waiting)
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	report, ok := LoadForPane("%45")
	if !ok {
		t.Fatal("LoadForPane() ok = false, want true")
	}
	if report.State != Waiting {
		t.Fatalf("LoadForPane() State = %q, want %q", report.State, Waiting)
	}
}

func TestLoadForPaneReadsPiDisplayMetadata(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	path, err := reportPath("%45")
	if err != nil {
		t.Fatalf("reportPath() error = %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	data := `{"pane_id":"%45","agent":"pi","state":"idle","session_name":"API review","branch":"feature/api-review","updated_at":"2026-06-26T00:00:00Z"}` + "\n"
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	report, ok := LoadForPane("%45")
	if !ok {
		t.Fatal("LoadForPane() ok = false, want true")
	}
	if report.SessionName != "API review" {
		t.Fatalf("LoadForPane() SessionName = %q, want %q", report.SessionName, "API review")
	}
	if report.Branch != "feature/api-review" {
		t.Fatalf("LoadForPane() Branch = %q, want %q", report.Branch, "feature/api-review")
	}
}
