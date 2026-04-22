package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"devsecops-platform/internal/report"
	"devsecops-platform/internal/scanner"
	"devsecops-platform/pkg/common"
)

const jsonReportFilename = "cli-scan.json"
const sarifReportFilename = "cli-scan.sarif.json"
const cliUsage = "usage: devsecops scan [--fail-on-high]\n       devsecops run --ci-mode"

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
	return runWithWriter(args, cfg, os.Stdout)
}

func runWithWriter(args []string, cfg common.Config, stdout io.Writer) (int, error) {
	if len(args) == 0 {
		return 0, fmt.Errorf(cliUsage)
	}

	switch args[0] {
	case "scan":
		return runScan(args[1:], cfg, stdout)
	case "run":
		return runCI(args[1:], cfg, stdout)
	default:
		return 0, fmt.Errorf("unknown command: %s", args[0])
	}
}

type scanOptions struct {
	failOnHigh bool
	ciMode     bool
}

func runScan(args []string, cfg common.Config, stdout io.Writer) (int, error) {
	flags := flag.NewFlagSet("scan", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)

	failOnHigh := flags.Bool("fail-on-high", false, "exit with code 1 when high or critical findings exist")
	if err := flags.Parse(args); err != nil {
		return 0, err
	}

	return executeScan(cfg, scanOptions{
		failOnHigh: *failOnHigh,
	}, stdout)
}

func runCI(args []string, cfg common.Config, stdout io.Writer) (int, error) {
	flags := flag.NewFlagSet("run", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)

	ciMode := flags.Bool("ci-mode", false, "simulate CI pipeline output and failure behavior")
	if err := flags.Parse(args); err != nil {
		return 0, err
	}

	if !*ciMode {
		return 0, fmt.Errorf("usage: devsecops run --ci-mode")
	}

	return executeScan(cfg, scanOptions{
		failOnHigh: true,
		ciMode:     true,
	}, stdout)
}

func executeScan(cfg common.Config, options scanOptions, stdout io.Writer) (int, error) {
	repoPath, err := filepath.Abs(".")
	if err != nil {
		return 0, err
	}

	enabledScanners := scanner.EnabledScanners(nil)
	if options.ciMode {
		printCIStart(stdout, repoPath, enabledScanners)
	}

	findings, err := runEnabledScanners(repoPath)
	if err != nil {
		return 0, err
	}

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

	if options.ciMode {
		printCISummary(stdout, aggregated, reportPath, sarifReportPath)
	} else {
		printSummary(stdout, repoPath, enabledScanners, aggregated, reportPath, sarifReportPath)
	}

	if options.failOnHigh && hasBlockingFindings(aggregated) {
		if options.ciMode {
			printCIResult(stdout, false, aggregated)
		}
		return 1, nil
	}

	if options.ciMode {
		printCIResult(stdout, true, aggregated)
	}

	return 0, nil
}

func runEnabledScanners(repoPath string) ([]common.Finding, error) {
	return scanner.RunScan(context.Background(), scanner.Job{
		RepoURL:  repoPath,
		Branch:   "local",
		ScanType: nil, // run all
	})
}

func printSummary(stdout io.Writer, repoPath string, enabledScanners []string, aggregated report.AggregatedResult, reportPath string, sarifReportPath string) {
	fmt.Fprintf(stdout, "Scanned Path: %s\n", repoPath)
	fmt.Fprintf(stdout, "Enabled Scanners: %s\n", strings.Join(enabledScanners, ", "))
	fmt.Fprintf(stdout, "Total Findings: %d\n", len(aggregated.Findings))
	fmt.Fprintf(stdout, "Critical: %d\n", aggregated.Counts["critical"])
	fmt.Fprintf(stdout, "High: %d\n", aggregated.Counts["high"])
	fmt.Fprintf(stdout, "Medium: %d\n", aggregated.Counts["medium"])
	fmt.Fprintf(stdout, "Low: %d\n", aggregated.Counts["low"])
	fmt.Fprintf(stdout, "Info: %d\n", aggregated.Counts["info"])
	fmt.Fprintf(stdout, "Total Risk Score: %d\n", aggregated.TotalRiskScore)
	fmt.Fprintf(stdout, "JSON Report: %s\n", reportPath)
	fmt.Fprintf(stdout, "SARIF Report: %s\n", sarifReportPath)
}

func printCIStart(stdout io.Writer, repoPath string, enabledScanners []string) {
	fmt.Fprintln(stdout, "[CI] Pipeline started")
	fmt.Fprintln(stdout, "[CI] Stage: security-scan")
	fmt.Fprintf(stdout, "[CI] Workspace: %s\n", repoPath)
	fmt.Fprintf(stdout, "[CI] Running full scan with scanners: %s\n", strings.Join(enabledScanners, ", "))
}

func printCISummary(stdout io.Writer, aggregated report.AggregatedResult, reportPath string, sarifReportPath string) {
	fmt.Fprintf(stdout, "[CI] Findings summary: critical=%d high=%d medium=%d low=%d info=%d total=%d\n",
		aggregated.Counts["critical"],
		aggregated.Counts["high"],
		aggregated.Counts["medium"],
		aggregated.Counts["low"],
		aggregated.Counts["info"],
		len(aggregated.Findings),
	)
	fmt.Fprintf(stdout, "[CI] Total risk score: %d\n", aggregated.TotalRiskScore)
	fmt.Fprintf(stdout, "[CI] Artifact generated: %s\n", reportPath)
	fmt.Fprintf(stdout, "[CI] Artifact generated: %s\n", sarifReportPath)
}

func printCIResult(stdout io.Writer, passed bool, aggregated report.AggregatedResult) {
	if passed {
		fmt.Fprintln(stdout, "[CI] Result: PASSED")
		return
	}

	fmt.Fprintf(stdout, "[CI] Result: FAILED (blocking findings detected: critical=%d high=%d)\n",
		aggregated.Counts["critical"],
		aggregated.Counts["high"],
	)
}

func hasBlockingFindings(aggregated report.AggregatedResult) bool {
	return aggregated.Counts["critical"] > 0 || aggregated.Counts["high"] > 0
}
