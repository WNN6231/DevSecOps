package report

import "devsecops-platform/pkg/common"

const sarifSchema = "https://json.schemastore.org/sarif-2.1.0.json"

type SARIFReport struct {
	Schema  string     `json:"$schema"`
	Version string     `json:"version"`
	Runs    []SARIFRun `json:"runs"`
}

type SARIFRun struct {
	Tool       SARIFTool          `json:"tool"`
	Results    []SARIFResult      `json:"results"`
	Properties SARIFRunProperties `json:"properties"`
}

type SARIFTool struct {
	Driver SARIFDriver `json:"driver"`
}

type SARIFDriver struct {
	Name  string      `json:"name"`
	Rules []SARIFRule `json:"rules,omitempty"`
}

type SARIFRule struct {
	ID               string            `json:"id"`
	Name             string            `json:"name"`
	ShortDescription SARIFMessage      `json:"shortDescription"`
	Properties       SARIFRuleMetadata `json:"properties"`
}

type SARIFRuleMetadata struct {
	Scanner  string `json:"scanner"`
	Severity string `json:"severity"`
}

type SARIFResult struct {
	RuleID              string                `json:"ruleId"`
	Level               string                `json:"level"`
	Message             SARIFMessage          `json:"message"`
	Locations           []SARIFLocation       `json:"locations,omitempty"`
	PartialFingerprints map[string]string     `json:"partialFingerprints,omitempty"`
	Properties          SARIFResultProperties `json:"properties"`
}

type SARIFMessage struct {
	Text string `json:"text"`
}

type SARIFLocation struct {
	PhysicalLocation SARIFPhysicalLocation `json:"physicalLocation"`
}

type SARIFPhysicalLocation struct {
	ArtifactLocation SARIFArtifactLocation `json:"artifactLocation"`
	Region           SARIFRegion           `json:"region,omitempty"`
}

type SARIFArtifactLocation struct {
	URI string `json:"uri"`
}

type SARIFRegion struct {
	StartLine int `json:"startLine,omitempty"`
}

type SARIFRunProperties struct {
	ScannedPath     string          `json:"scannedPath"`
	EnabledScanners []string        `json:"enabledScanners"`
	Summary         SeveritySummary `json:"summary"`
	TotalRiskScore  int             `json:"totalRiskScore"`
}

type SARIFResultProperties struct {
	Scanner        string `json:"scanner"`
	Severity       string `json:"severity"`
	Description    string `json:"description,omitempty"`
	Evidence       string `json:"evidence,omitempty"`
	Recommendation string `json:"recommendation,omitempty"`
	Hash           string `json:"hash,omitempty"`
}

func BuildSARIFReport(repoPath string, enabledScanners []string, aggregated AggregatedResult) SARIFReport {
	rules := make([]SARIFRule, 0, len(aggregated.Findings))
	results := make([]SARIFResult, 0, len(aggregated.Findings))
	seenRules := make(map[string]struct{}, len(aggregated.Findings))

	for _, finding := range aggregated.Findings {
		ruleID := findingRuleID(finding)
		if _, seen := seenRules[ruleID]; !seen {
			seenRules[ruleID] = struct{}{}
			rules = append(rules, buildSARIFRule(ruleID, finding))
		}

		results = append(results, buildSARIFResult(ruleID, finding))
	}

	return SARIFReport{
		Schema:  sarifSchema,
		Version: "2.1.0",
		Runs: []SARIFRun{
			{
				Tool: SARIFTool{
					Driver: SARIFDriver{
						Name:  "devsecops-platform",
						Rules: rules,
					},
				},
				Results: results,
				Properties: SARIFRunProperties{
					ScannedPath:     repoPath,
					EnabledScanners: append([]string(nil), enabledScanners...),
					Summary:         buildSeveritySummary(aggregated),
					TotalRiskScore:  aggregated.TotalRiskScore,
				},
			},
		},
	}
}

func WriteSARIFReport(reportDir, filename string, report SARIFReport) (string, error) {
	return writeStructuredReport(reportDir, filename, report)
}

func buildSARIFRule(ruleID string, finding common.Finding) SARIFRule {
	name := finding.Title
	if name == "" {
		name = ruleID
	}

	description := finding.Description
	if description == "" {
		description = name
	}

	return SARIFRule{
		ID:               ruleID,
		Name:             name,
		ShortDescription: SARIFMessage{Text: description},
		Properties: SARIFRuleMetadata{
			Scanner:  finding.Scanner,
			Severity: finding.Severity,
		},
	}
}

func buildSARIFResult(ruleID string, finding common.Finding) SARIFResult {
	result := SARIFResult{
		RuleID:  ruleID,
		Level:   sarifLevelForSeverity(finding.Severity),
		Message: SARIFMessage{Text: sarifMessageText(finding, ruleID)},
		Properties: SARIFResultProperties{
			Scanner:        finding.Scanner,
			Severity:       finding.Severity,
			Description:    finding.Description,
			Evidence:       finding.Evidence,
			Recommendation: finding.Recommendation,
			Hash:           finding.Hash,
		},
	}

	if finding.FilePath != "" {
		location := SARIFLocation{
			PhysicalLocation: SARIFPhysicalLocation{
				ArtifactLocation: SARIFArtifactLocation{URI: finding.FilePath},
			},
		}
		if finding.LineNumber > 0 {
			location.PhysicalLocation.Region = SARIFRegion{StartLine: finding.LineNumber}
		}
		result.Locations = []SARIFLocation{location}
	}

	if finding.Hash != "" {
		result.PartialFingerprints = map[string]string{
			"primaryLocationLineHash": finding.Hash,
		}
	}

	return result
}

func findingRuleID(finding common.Finding) string {
	if finding.RuleID != "" {
		return finding.RuleID
	}
	if finding.Hash != "" {
		return finding.Hash
	}
	if finding.Title != "" {
		return finding.Title
	}
	return "unknown-rule"
}

func sarifMessageText(finding common.Finding, ruleID string) string {
	if finding.Title != "" {
		return finding.Title
	}
	if finding.Description != "" {
		return finding.Description
	}
	return ruleID
}

func sarifLevelForSeverity(severity string) string {
	switch severity {
	case "critical", "high":
		return "error"
	case "medium", "low":
		return "warning"
	case "info":
		return "note"
	default:
		return "warning"
	}
}
