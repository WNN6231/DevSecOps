package report

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"devsecops-platform/pkg/common"
)

func TestWriteJSONReportWritesFile(t *testing.T) {
	reportDir := filepath.Join(t.TempDir(), "reports")
	input := JSONReport{
		ScannedPath:     "C:/repo",
		EnabledScanners: []string{"sast"},
		Summary: SeveritySummary{
			TotalFindings: 1,
			High:          1,
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

	if decoded.Summary.TotalFindings != 1 || decoded.Summary.High != 1 {
		t.Fatalf("unexpected summary contents: %+v", decoded.Summary)
	}
}

func TestWriteJSONReportRejectsRelativeDir(t *testing.T) {
	_, err := WriteJSONReport("reports", "cli-scan.json", JSONReport{})
	if !errors.Is(err, ErrInvalidReportDir) {
		t.Fatalf("expected ErrInvalidReportDir, got %v", err)
	}
}

func TestBuildJSONReportIncludesSeveritySummary(t *testing.T) {
	aggregated := AggregatedResult{
		Findings: []common.Finding{
			{
				Scanner:  "sast",
				Severity: "critical",
				RuleID:   "RULE-1",
				Hash:     "hash-1",
			},
			{
				Scanner:  "sca",
				Severity: "medium",
				RuleID:   "RULE-2",
				Hash:     "hash-2",
			},
		},
		Counts: map[string]int{
			"critical": 1,
			"high":     0,
			"medium":   1,
			"low":      0,
			"info":     0,
		},
		TotalRiskScore: 8,
	}

	report := BuildJSONReport("C:/repo", []string{"sast", "sca"}, aggregated)

	if report.Summary.TotalFindings != 2 {
		t.Fatalf("expected total findings 2, got %d", report.Summary.TotalFindings)
	}
	if report.Summary.Critical != 1 || report.Summary.Medium != 1 {
		t.Fatalf("unexpected summary counts: %+v", report.Summary)
	}
	if len(report.Findings) != 2 {
		t.Fatalf("expected 2 findings, got %d", len(report.Findings))
	}
}
