package report

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestWriteJSONReportWritesFile(t *testing.T) {
	reportDir := filepath.Join(t.TempDir(), "reports")
	input := JSONReport{
		ScannedPath:     "C:/repo",
		EnabledScanners: []string{"sast"},
		Summary: map[string]int{
			"high": 1,
		},
		TotalRiskScore: 4,
	}

	reportPath, err := WriteJSONReport(reportDir, "cli-scan.json", input)
	if err != nil {
		t.Fatalf("write json report: %v", err)
	}

	content, err := os.ReadFile(reportPath)
	if err != nil {
		t.Fatalf("read json report: %v", err)
	}

	var decoded JSONReport
	if err := json.Unmarshal(content, &decoded); err != nil {
		t.Fatalf("unmarshal json report: %v", err)
	}

	if decoded.ScannedPath != input.ScannedPath {
		t.Fatalf("expected scanned path %s, got %s", input.ScannedPath, decoded.ScannedPath)
	}
}

func TestWriteJSONReportRejectsRelativeDir(t *testing.T) {
	_, err := WriteJSONReport("reports", "cli-scan.json", JSONReport{})
	if !errors.Is(err, ErrInvalidReportDir) {
		t.Fatalf("expected ErrInvalidReportDir, got %v", err)
	}
}
