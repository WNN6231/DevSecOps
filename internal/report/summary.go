package report

type SeveritySummary struct {
	TotalFindings int `json:"total_findings"`
	Critical      int `json:"critical"`
	High          int `json:"high"`
	Medium        int `json:"medium"`
	Low           int `json:"low"`
	Info          int `json:"info"`
}

func buildSeveritySummary(aggregated AggregatedResult) SeveritySummary {
	return SeveritySummary{
		TotalFindings: len(aggregated.Findings),
		Critical:      aggregated.Counts["critical"],
		High:          aggregated.Counts["high"],
		Medium:        aggregated.Counts["medium"],
		Low:           aggregated.Counts["low"],
		Info:          aggregated.Counts["info"],
	}
}
