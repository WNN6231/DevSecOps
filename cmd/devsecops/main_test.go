package main

import (
	"encoding/json"
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

	jsonReport := report.BuildJSONReport("C:/repo", []string{"sast"}, aggregated)
	if jsonReport.ScannedPath != "C:/repo" {
		t.Fatalf("expected scanned path C:/repo, got %s", jsonReport.ScannedPath)
	}
	if len(jsonReport.Findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(jsonReport.Findings))
	}
	if jsonReport.Summary.TotalFindings != 1 || jsonReport.Summary.High != 1 {
		t.Fatalf("unexpected summary: %+v", jsonReport.Summary)
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

	jsonReportPath := filepath.Join(cfg.ReportDir, jsonReportFilename)
	if _, err := os.Stat(jsonReportPath); err != nil {
		t.Fatalf("expected json report at %s: %v", jsonReportPath, err)
	}

	sarifReportPath := filepath.Join(cfg.ReportDir, sarifReportFilename)
	content, err := os.ReadFile(sarifReportPath)
	if err != nil {
		t.Fatalf("expected sarif report at %s: %v", sarifReportPath, err)
	}

	var sarifReport report.SARIFReport
	if err := json.Unmarshal(content, &sarifReport); err != nil {
		t.Fatalf("unmarshal sarif report: %v", err)
	}

	if len(sarifReport.Runs) != 1 {
		t.Fatalf("expected 1 sarif run, got %d", len(sarifReport.Runs))
	}
}
