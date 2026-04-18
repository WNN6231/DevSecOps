package report

import "devsecops-platform/pkg/common"

var severityRiskScore = map[string]int{
	"critical": 5,
	"high":     4,
	"medium":   3,
	"low":      2,
	"info":     0,
}

type AggregatedResult struct {
	Findings       []common.Finding
	Counts         map[string]int
	TotalRiskScore int
}

func Aggregate(findings []common.Finding) AggregatedResult {
	counts := map[string]int{
		"critical": 0,
		"high":     0,
		"medium":   0,
		"low":      0,
		"info":     0,
	}
	seenHashes := make(map[string]struct{}, len(findings))
	deduplicated := make([]common.Finding, 0, len(findings))
	totalRiskScore := 0

	for _, finding := range findings {
		if _, exists := seenHashes[finding.Hash]; exists {
			continue
		}

		seenHashes[finding.Hash] = struct{}{}
		deduplicated = append(deduplicated, finding)

		if _, ok := counts[finding.Severity]; ok {
			counts[finding.Severity]++
			totalRiskScore += severityRiskScore[finding.Severity]
		}
	}

	return AggregatedResult{
		Findings:       deduplicated,
		Counts:         counts,
		TotalRiskScore: totalRiskScore,
	}
}
