package report

import (
	"encoding/json"
	"os"

	"devsecops-platform/pkg/common"
)

type JSONReport struct {
	ScannedPath     string          `json:"scanned_path"`
	EnabledScanners []string        `json:"enabled_scanners"`
	Findings        []FindingReport `json:"findings"`
	Summary         SeveritySummary `json:"summary"`
	TotalRiskScore  int             `json:"total_risk_score"`
}

type FindingReport struct {
	Scanner        string `json:"scanner"`
	Severity       string `json:"severity"`
	RuleID         string `json:"rule_id"`
	Title          string `json:"title"`
	Description    string `json:"description"`
	FilePath       string `json:"file_path"`
	LineNumber     int    `json:"line_number"`
	Evidence       string `json:"evidence"`
	Recommendation string `json:"recommendation"`
	Hash           string `json:"hash"`
}

func BuildJSONReport(repoPath string, enabledScanners []string, aggregated AggregatedResult) JSONReport {
	findings := make([]FindingReport, 0, len(aggregated.Findings))
	for _, finding := range aggregated.Findings {
		findings = append(findings, findingReportFromFinding(finding))
	}

	return JSONReport{
		ScannedPath:     repoPath,
		EnabledScanners: append([]string(nil), enabledScanners...),
		Findings:        findings,
		Summary:         buildSeveritySummary(aggregated),
		TotalRiskScore:  aggregated.TotalRiskScore,
	}
}

func WriteJSONReport(reportDir, filename string, report JSONReport) (string, error) {
	return writeStructuredReport(reportDir, filename, report)
}

func findingReportFromFinding(finding common.Finding) FindingReport {
	return FindingReport{
		Scanner:        finding.Scanner,
		Severity:       finding.Severity,
		RuleID:         finding.RuleID,
		Title:          finding.Title,
		Description:    finding.Description,
		FilePath:       finding.FilePath,
		LineNumber:     finding.LineNumber,
		Evidence:       finding.Evidence,
		Recommendation: finding.Recommendation,
		Hash:           finding.Hash,
	}
}

func writeStructuredReport(reportDir, filename string, report interface{}) (string, error) {
	path, err := reportPath(reportDir, filename)
	if err != nil {
		return "", err
	}

	if err := os.MkdirAll(reportDir, 0o755); err != nil {
		return "", err
	}

	content, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return "", err
	}

	if err := os.WriteFile(path, append(content, '\n'), 0o644); err != nil {
		return "", err
	}

	return path, nil
}
