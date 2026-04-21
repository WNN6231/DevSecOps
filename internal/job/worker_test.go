package job

import (
	"encoding/json"
	"testing"

	"devsecops-platform/internal/report"
	"devsecops-platform/pkg/common"
)

func TestBuildScanResultsMapsRequiredFindingFields(t *testing.T) {
	findings := []common.Finding{
		{
			Scanner:        "sast",
			Severity:       "high",
			RuleID:         "GO_SAST_SECRET_001",
			Title:          "Possible hardcoded secret",
			Description:    "secret found",
			FilePath:       "main.go",
			LineNumber:     12,
			Evidence:       `token := "secret"`,
			Recommendation: "move to env",
			Hash:           "abc123",
		},
	}

	records := buildScanResults(42, findings)
	if len(records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(records))
	}

	record := records[0]
	if record.JobID != 42 {
		t.Fatalf("expected job id 42, got %d", record.JobID)
	}

	if record.Severity != "high" {
		t.Fatalf("expected severity high, got %s", record.Severity)
	}

	if record.RuleID != "GO_SAST_SECRET_001" {
		t.Fatalf("expected rule id GO_SAST_SECRET_001, got %s", record.RuleID)
	}

	if record.FilePath != "main.go" {
		t.Fatalf("expected file path main.go, got %s", record.FilePath)
	}

	if record.Evidence != `token := "secret"` {
		t.Fatalf("expected evidence to be persisted, got %s", record.Evidence)
	}
}

func TestBuildScanReportMapsRequiredFields(t *testing.T) {
	aggregated := report.AggregatedResult{
		Counts: map[string]int{
			"critical": 1,
			"high":     2,
			"medium":   3,
			"low":      4,
			"info":     5,
		},
		TotalRiskScore: 22,
	}

	record, err := buildScanReport(42, "reports/42.md", aggregated)
	if err != nil {
		t.Fatalf("build scan report: %v", err)
	}

	if record.JobID != 42 {
		t.Fatalf("expected job id 42, got %d", record.JobID)
	}

	if record.ReportPath != "reports/42.md" {
		t.Fatalf("expected report path reports/42.md, got %s", record.ReportPath)
	}

	if record.HighCount != 2 || record.MediumCount != 3 || record.LowCount != 4 {
		t.Fatalf("unexpected severity counts: high=%d medium=%d low=%d", record.HighCount, record.MediumCount, record.LowCount)
	}

	if record.RiskScore != 22 {
		t.Fatalf("expected risk score 22, got %d", record.RiskScore)
	}

	var summary map[string]int
	if err := json.Unmarshal([]byte(record.SummaryJSON), &summary); err != nil {
		t.Fatalf("unmarshal summary json: %v", err)
	}

	if summary["critical"] != 1 || summary["info"] != 5 {
		t.Fatalf("unexpected summary json contents: %+v", summary)
	}
}
