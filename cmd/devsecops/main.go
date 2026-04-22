package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"devsecops-platform/internal/report"
	"devsecops-platform/internal/scanner"
	"devsecops-platform/pkg/common"
)

const jsonReportFilename = "cli-scan.json"
const sarifReportFilename = "cli-scan.sarif.json"

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

	findings, err := runEnabledScanners(repoPath)
	if err != nil {
		return 0, err
	}

	enabledScanners := scanner.EnabledScanners(nil)

	aggregated := report.Aggregate(findings)
	jsonReport := report.BuildJSONReport(repoPath, enabledScanners, aggregated)
	reportPath, err := report.WriteJSONReport(cfg.ReportDir, jsonReportFilename, jsonReport)
	if err != nil {
		return 0, err
	}

	sarifReport := report.BuildSARIFReport(repoPath, enabledScanners, aggregated)
	sarifReportPath, err := report.WriteSARIFReport(cfg.ReportDir, sarifReportFilename, sarifReport)
	if err != nil {
		return 0, err
	}

	printSummary(repoPath, enabledScanners, aggregated, reportPath, sarifReportPath)

	if *failOnHigh && hasBlockingFindings(aggregated) {
		return 1, nil
	}

	return 0, nil
}

func runEnabledScanners(repoPath string) ([]common.Finding, error) {
	return scanner.RunScan(scanner.Job{
		RepoURL:  repoPath,
		Branch:   "local",
		ScanType: nil, // run all
	})
}

func printSummary(repoPath string, enabledScanners []string, aggregated report.AggregatedResult, reportPath string, sarifReportPath string) {
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
	fmt.Printf("SARIF Report: %s\n", sarifReportPath)
}

func hasBlockingFindings(aggregated report.AggregatedResult) bool {
	return aggregated.Counts["critical"] > 0 || aggregated.Counts["high"] > 0
}
