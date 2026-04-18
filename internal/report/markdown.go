package report

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"devsecops-platform/pkg/common"
)

const reportsDir = "reports"

var severityOrder = []string{"critical", "high", "medium", "low", "info"}

func WriteMarkdownReport(jobID int64, result AggregatedResult) (string, error) {
	if err := os.MkdirAll(reportsDir, 0o755); err != nil {
		return "", err
	}

	reportPath := filepath.Join(reportsDir, fmt.Sprintf("%d.md", jobID))
	content := buildMarkdownReport(jobID, result)

	if err := os.WriteFile(reportPath, []byte(content), 0o644); err != nil {
		return "", err
	}

	return reportPath, nil
}

func buildMarkdownReport(jobID int64, result AggregatedResult) string {
	var builder strings.Builder

	builder.WriteString(fmt.Sprintf("# Scan Report: Job %d\n\n", jobID))
	builder.WriteString("## Summary\n\n")
	builder.WriteString(fmt.Sprintf("- Total Findings: %d\n", len(result.Findings)))
	builder.WriteString(fmt.Sprintf("- Total Risk Score: %d\n", result.TotalRiskScore))
	for _, severity := range severityOrder {
		builder.WriteString(fmt.Sprintf("- %s: %d\n", strings.Title(severity), result.Counts[severity]))
	}

	builder.WriteString("\n## Detailed Findings\n\n")

	groupedFindings := groupFindingsBySeverity(result.Findings)
	hasFindings := false
	for _, severity := range severityOrder {
		findings := groupedFindings[severity]
		if len(findings) == 0 {
			continue
		}

		hasFindings = true
		builder.WriteString(fmt.Sprintf("### %s\n\n", strings.Title(severity)))
		for index, finding := range findings {
			builder.WriteString(fmt.Sprintf("#### %d. %s\n\n", index+1, finding.Title))
			builder.WriteString(fmt.Sprintf("- Scanner: `%s`\n", finding.Scanner))
			builder.WriteString(fmt.Sprintf("- Rule ID: `%s`\n", finding.RuleID))
			builder.WriteString(fmt.Sprintf("- Severity: `%s`\n", finding.Severity))
			builder.WriteString(fmt.Sprintf("- File: `%s`\n", finding.FilePath))
			builder.WriteString(fmt.Sprintf("- Line: `%d`\n", finding.LineNumber))
			builder.WriteString(fmt.Sprintf("- Hash: `%s`\n\n", finding.Hash))
			builder.WriteString("**Description**\n\n")
			builder.WriteString(finding.Description + "\n\n")
			builder.WriteString("**Evidence**\n\n")
			builder.WriteString("```\n")
			builder.WriteString(finding.Evidence + "\n")
			builder.WriteString("```\n\n")
			builder.WriteString("**Recommendation**\n\n")
			builder.WriteString(finding.Recommendation + "\n\n")
		}
	}

	if !hasFindings {
		builder.WriteString("No findings.\n")
	}

	return builder.String()
}

func groupFindingsBySeverity(findings []common.Finding) map[string][]common.Finding {
	grouped := map[string][]common.Finding{
		"critical": {},
		"high":     {},
		"medium":   {},
		"low":      {},
		"info":     {},
	}

	for _, finding := range findings {
		if _, ok := grouped[finding.Severity]; !ok {
			continue
		}

		grouped[finding.Severity] = append(grouped[finding.Severity], finding)
	}

	return grouped
}
