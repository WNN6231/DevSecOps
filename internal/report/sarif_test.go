package report

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"devsecops-platform/pkg/common"
)

func TestBuildSARIFReportIncludesSummaryAndStructuredResults(t *testing.T) {
	aggregated := AggregatedResult{
		Findings: []common.Finding{
			{
				Scanner:        "sast",
				Severity:       "high",
				RuleID:         "RULE-1",
				Title:          "Hardcoded secret",
				Description:    "Secret-like literal detected.",
				FilePath:       "main.go",
				LineNumber:     12,
				Evidence:       `token := "secret"`,
				Recommendation: "Move the secret to an environment variable.",
				Hash:           "hash-1",
			},
		},
		Counts: map[string]int{
			"critical": 0,
			"high":     1,
			"medium":   0,
			"low":      0,
			"info":     0,
		},
		TotalRiskScore: 4,
	}

	report := BuildSARIFReport("C:/repo", []string{"sast"}, aggregated)

	if report.Version != "2.1.0" {
		t.Fatalf("expected SARIF version 2.1.0, got %s", report.Version)
	}
	if len(report.Runs) != 1 {
		t.Fatalf("expected 1 run, got %d", len(report.Runs))
	}

	run := report.Runs[0]
	if run.Properties.Summary.TotalFindings != 1 || run.Properties.Summary.High != 1 {
		t.Fatalf("unexpected run summary: %+v", run.Properties.Summary)
	}
	if len(run.Tool.Driver.Rules) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(run.Tool.Driver.Rules))
	}
	if len(run.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(run.Results))
	}

	result := run.Results[0]
	if result.RuleID != "RULE-1" {
		t.Fatalf("expected rule id RULE-1, got %s", result.RuleID)
	}
	if result.Level != "error" {
		t.Fatalf("expected level error, got %s", result.Level)
	}
	if len(result.Locations) != 1 || result.Locations[0].PhysicalLocation.Region.StartLine != 12 {
		t.Fatalf("unexpected locations: %+v", result.Locations)
	}
	if result.PartialFingerprints["primaryLocationLineHash"] != "hash-1" {
		t.Fatalf("unexpected fingerprints: %+v", result.PartialFingerprints)
	}
}

func TestWriteSARIFReportWritesFile(t *testing.T) {
	reportDir := filepath.Join(t.TempDir(), "reports")
	input := SARIFReport{
		Schema:  sarifSchema,
		Version: "2.1.0",
		Runs: []SARIFRun{
			{
				Properties: SARIFRunProperties{
					Summary: SeveritySummary{TotalFindings: 1, Low: 1},
				},
			},
		},
	}

	reportPath, err := WriteSARIFReport(reportDir, "cli-scan.sarif.json", input)
	if err != nil {
		t.Fatalf("write sarif report: %v", err)
	}

	content, err := os.ReadFile(reportPath)
	if err != nil {
		t.Fatalf("read sarif report: %v", err)
	}

	var decoded SARIFReport
	if err := json.Unmarshal(content, &decoded); err != nil {
		t.Fatalf("unmarshal sarif report: %v", err)
	}

	if decoded.Version != "2.1.0" {
		t.Fatalf("expected version 2.1.0, got %s", decoded.Version)
	}
	if decoded.Runs[0].Properties.Summary.Low != 1 {
		t.Fatalf("unexpected summary contents: %+v", decoded.Runs[0].Properties.Summary)
	}
}

func TestWriteSARIFReportRejectsRelativeDir(t *testing.T) {
	_, err := WriteSARIFReport("reports", "cli-scan.sarif.json", SARIFReport{})
	if !errors.Is(err, ErrInvalidReportDir) {
		t.Fatalf("expected ErrInvalidReportDir, got %v", err)
	}
}
