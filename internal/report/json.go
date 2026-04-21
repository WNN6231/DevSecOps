package report

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type JSONReport struct {
	ScannedPath     string          `json:"scanned_path"`
	EnabledScanners []string        `json:"enabled_scanners"`
	Findings        []FindingReport `json:"findings"`
	Summary         map[string]int  `json:"summary"`
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

func WriteJSONReport(reportDir, filename string, report JSONReport) (string, error) {
	if !filepath.IsAbs(reportDir) {
		return "", ErrInvalidReportDir
	}

	if err := os.MkdirAll(reportDir, 0o755); err != nil {
		return "", err
	}

	reportPath := filepath.Join(filepath.Clean(reportDir), filename)
	content, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return "", err
	}

	if err := os.WriteFile(reportPath, append(content, '\n'), 0o644); err != nil {
		return "", err
	}

	return reportPath, nil
}
