package job

import (
	"testing"

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
