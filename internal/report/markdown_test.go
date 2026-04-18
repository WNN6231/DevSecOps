package report

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"devsecops-platform/pkg/common"
)

func TestBuildMarkdownReportIncludesSummaryAndGroupedFindings(t *testing.T) {
	result := AggregatedResult{
		Findings: []common.Finding{
			{
				Scanner:        "sast",
				Severity:       "high",
				RuleID:         "RULE-HIGH",
				Title:          "High finding",
				Description:    "High severity issue.",
				FilePath:       "main.go",
				LineNumber:     12,
				Evidence:       `token := "secret"`,
				Recommendation: "Remove the secret.",
				Hash:           "hash-high",
			},
			{
				Scanner:        "sast",
				Severity:       "low",
				RuleID:         "RULE-LOW",
				Title:          "Low finding",
				Description:    "Low severity issue.",
				FilePath:       "helper.go",
				LineNumber:     3,
				Evidence:       "fmt.Println(value)",
				Recommendation: "Review logging.",
				Hash:           "hash-low",
			},
		},
		Counts: map[string]int{
			"critical": 0,
			"high":     1,
			"medium":   0,
			"low":      1,
			"info":     0,
		},
		TotalRiskScore: 6,
	}

	content := buildMarkdownReport(42, result)

	expectedParts := []string{
		"# Scan Report: Job 42",
		"## Summary",
		"- Total Findings: 2",
		"- Total Risk Score: 6",
		"- High: 1",
		"- Low: 1",
		"## Detailed Findings",
		"### High",
		"### Low",
		"#### 1. High finding",
		"#### 1. Low finding",
		"**Evidence**",
		"hash-high",
	}

	for _, expected := range expectedParts {
		if !strings.Contains(content, expected) {
			t.Fatalf("expected markdown report to contain %q", expected)
		}
	}
}

func TestWriteMarkdownReportWritesFileToReportsDirectory(t *testing.T) {
	workdir := t.TempDir()
	previousDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("get working directory: %v", err)
	}

	if err := os.Chdir(workdir); err != nil {
		t.Fatalf("change directory: %v", err)
	}
	defer func() {
		if err := os.Chdir(previousDir); err != nil {
			t.Fatalf("restore working directory: %v", err)
		}
	}()

	result := AggregatedResult{
		Counts: map[string]int{
			"critical": 0,
			"high":     0,
			"medium":   0,
			"low":      0,
			"info":     0,
		},
	}

	reportPath, err := WriteMarkdownReport(7, result)
	if err != nil {
		t.Fatalf("write markdown report: %v", err)
	}

	expectedPath := filepath.Join("reports", "7.md")
	if reportPath != expectedPath {
		t.Fatalf("expected report path %s, got %s", expectedPath, reportPath)
	}

	content, err := os.ReadFile(reportPath)
	if err != nil {
		t.Fatalf("read markdown report: %v", err)
	}

	if !strings.Contains(string(content), "# Scan Report: Job 7") {
		t.Fatalf("expected report file content to include job heading")
	}
}
