package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"devsecops-platform/internal/report"
	"devsecops-platform/internal/scanner/sast"
	"devsecops-platform/pkg/common"
)

const jsonReportFilename = "cli-scan.json"

func main() {
	cfg, err := common.LoadConfig()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	exitCode, err := run(os.Args[1:], cfg)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	os.Exit(exitCode)
}

func run(args []string, cfg common.Config) (int, error) {
	if len(args) == 0 {
		return 0, fmt.Errorf("usage: devsecops scan [--fail-on-high]")
	}

	switch args[0] {
	case "scan":
		return runScan(args[1:], cfg)
	default:
		return 0, fmt.Errorf("unknown command: %s", args[0])
	}
}

func runScan(args []string, cfg common.Config) (int, error) {
	flags := flag.NewFlagSet("scan", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)

	failOnHigh := flags.Bool("fail-on-high", false, "exit with code 1 when high or critical findings exist")
	if err := flags.Parse(args); err != nil {
		return 0, err
	}

	repoPath, err := filepath.Abs(".")
	if err != nil {
		return 0, err
	}

	findings, enabledScanners, err := runEnabledScanners(repoPath)
	if err != nil {
		return 0, err
	}

	aggregated := report.Aggregate(findings)
	jsonReport := buildJSONReport(repoPath, enabledScanners, aggregated)
	reportPath, err := report.WriteJSONReport(cfg.ReportDir, jsonReportFilename, jsonReport)
	if err != nil {
		return 0, err
	}

	printSummary(repoPath, enabledScanners, aggregated, reportPath)

	if *failOnHigh && hasBlockingFindings(aggregated) {
		return 1, nil
	}

	return 0, nil
}

func runEnabledScanners(repoPath string) ([]common.Finding, []string, error) {
	enabledScanners := []string{"sast"}
	findings, err := sast.NewScanner().Scan(repoPath, "local")
	if err != nil {
		return nil, nil, err
	}

	return findings, enabledScanners, nil
}

func buildJSONReport(repoPath string, enabledScanners []string, aggregated report.AggregatedResult) report.JSONReport {
	findings := make([]report.FindingReport, 0, len(aggregated.Findings))
	for _, finding := range aggregated.Findings {
		findings = append(findings, report.FindingReport{
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
		})
	}

	return report.JSONReport{
		ScannedPath:     repoPath,
		EnabledScanners: append([]string(nil), enabledScanners...),
		Findings:        findings,
		Summary:         aggregated.Counts,
		TotalRiskScore:  aggregated.TotalRiskScore,
	}
}

func printSummary(repoPath string, enabledScanners []string, aggregated report.AggregatedResult, reportPath string) {
	fmt.Printf("Scanned Path: %s\n", repoPath)
	fmt.Printf("Enabled Scanners: %s\n", strings.Join(enabledScanners, ", "))
	fmt.Printf("Total Findings: %d\n", len(aggregated.Findings))
	fmt.Printf("Critical: %d\n", aggregated.Counts["critical"])
	fmt.Printf("High: %d\n", aggregated.Counts["high"])
	fmt.Printf("Medium: %d\n", aggregated.Counts["medium"])
	fmt.Printf("Low: %d\n", aggregated.Counts["low"])
	fmt.Printf("Info: %d\n", aggregated.Counts["info"])
	fmt.Printf("Total Risk Score: %d\n", aggregated.TotalRiskScore)
	fmt.Printf("JSON Report: %s\n", reportPath)
}

func hasBlockingFindings(aggregated report.AggregatedResult) bool {
	return aggregated.Counts["critical"] > 0 || aggregated.Counts["high"] > 0
}
