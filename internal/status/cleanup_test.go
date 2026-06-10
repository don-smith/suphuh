package status

import "testing"

func TestPaneIDFromReportFallsBackToFileName(t *testing.T) {
	paneID, ok := paneIDFromReport("/tmp/pct_45.json")
	if !ok {
		t.Fatal("paneIDFromReport() ok = false, want true")
	}
	if paneID != "%45" {
		t.Fatalf("paneIDFromReport() = %q, want %%45", paneID)
	}
}
