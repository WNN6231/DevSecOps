package report

import (
	"testing"

	"devsecops-platform/pkg/common"
)

func TestAggregateDeduplicatesByHash(t *testing.T) {
	findings := []common.Finding{
		{
			Hash:     "dup",
			Severity: "high",
			RuleID:   "RULE-1",
		},
		{
			Hash:     "dup",
			Severity: "critical",
			RuleID:   "RULE-2",
		},
		{
			Hash:     "unique",
			Severity: "low",
			RuleID:   "RULE-3",
		},
	}

	result := Aggregate(findings)

	if len(result.Findings) != 2 {
		t.Fatalf("expected 2 deduplicated findings, got %d", len(result.Findings))
	}

	if result.Findings[0].RuleID != "RULE-1" {
		t.Fatalf("expected first finding to win hash deduplication, got %s", result.Findings[0].RuleID)
	}

	if result.Findings[1].RuleID != "RULE-3" {
		t.Fatalf("expected unique finding to remain, got %s", result.Findings[1].RuleID)
	}
}

func TestAggregateBuildsSeverityCountsAndRiskScore(t *testing.T) {
	findings := []common.Finding{
		{Hash: "critical", Severity: "critical"},
		{Hash: "high", Severity: "high"},
		{Hash: "medium", Severity: "medium"},
		{Hash: "low", Severity: "low"},
		{Hash: "info", Severity: "info"},
	}

	result := Aggregate(findings)

	expectedCounts := map[string]int{
		"critical": 1,
		"high":     1,
		"medium":   1,
		"low":      1,
		"info":     1,
	}

	for severity, expected := range expectedCounts {
		if result.Counts[severity] != expected {
			t.Fatalf("expected %s count %d, got %d", severity, expected, result.Counts[severity])
		}
	}

	if result.TotalRiskScore != 14 {
		t.Fatalf("expected total risk score 14, got %d", result.TotalRiskScore)
	}
}
