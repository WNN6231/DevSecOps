package main

import (
	"os"
	"path/filepath"
	"testing"

	"devsecops-platform/internal/report"
	"devsecops-platform/pkg/common"
)

func TestHasBlockingFindings(t *testing.T) {
	if !hasBlockingFindings(report.AggregatedResult{
		Counts: map[string]int{"critical": 0, "high": 1},
	}) {
		t.Fatal("expected high findings to block")
	}

	if hasBlockingFindings(report.AggregatedResult{
		Counts: map[string]int{"critical": 0, "high": 0},
	}) {
		t.Fatal("expected no blocking findings")
	}
}

func TestBuildJSONReportMapsAggregatedResult(t *testing.T) {
	aggregated := report.AggregatedResult{
		Findings: []common.Finding{
			{
				Scanner: "sast",
				RuleID:  "RULE-1",
				Hash:    "abc",
			},
		},
		Counts:         map[string]int{"high": 1},
		TotalRiskScore: 4,
	}

	jsonReport := buildJSONReport("C:/repo", []string{"sast"}, aggregated)
	if jsonReport.ScannedPath != "C:/repo" {
		t.Fatalf("expected scanned path C:/repo, got %s", jsonReport.ScannedPath)
	}
	if len(jsonReport.Findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(jsonReport.Findings))
	}
	if jsonReport.TotalRiskScore != 4 {
		t.Fatalf("expected risk score 4, got %d", jsonReport.TotalRiskScore)
	}
}

func TestRunScanFailsOnHigh(t *testing.T) {
	cfg := common.Config{
		ReportDir: filepath.Join(t.TempDir(), "reports"),
	}

	repoPath := t.TempDir()
	source := "package main\n\nfunc main() {\n\tapiToken := \"super-secret-token\"\n}\n"
	if err := os.WriteFile(filepath.Join(repoPath, "main.go"), []byte(source), 0o644); err != nil {
		t.Fatalf("write test repo file: %v", err)
	}

	previousDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("get working directory: %v", err)
	}

	if err := os.Chdir(repoPath); err != nil {
		t.Fatalf("change directory: %v", err)
	}
	defer func() {
		if err := os.Chdir(previousDir); err != nil {
			t.Fatalf("restore working directory: %v", err)
		}
	}()

	exitCode, err := run([]string{"scan", "--fail-on-high"}, cfg)
	if err != nil {
		t.Fatalf("run scan: %v", err)
	}

	if exitCode != 1 {
		t.Fatalf("expected exit code 1, got %d", exitCode)
	}
}
