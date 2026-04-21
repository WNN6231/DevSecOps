package common

import (
	"path/filepath"
	"testing"
)

func TestResolveReportDirDefaultsToProjectRootReports(t *testing.T) {
	t.Setenv("REPORT_DIR", "")

	projectRoot, err := projectRootDir()
	if err != nil {
		t.Fatalf("project root dir: %v", err)
	}

	reportDir, err := resolveReportDir()
	if err != nil {
		t.Fatalf("resolve report dir: %v", err)
	}

	expected := filepath.Join(projectRoot, "reports")
	if reportDir != expected {
		t.Fatalf("expected report dir %s, got %s", expected, reportDir)
	}
}

func TestResolveReportDirResolvesRelativeValueAgainstProjectRoot(t *testing.T) {
	t.Setenv("REPORT_DIR", "custom-reports")

	projectRoot, err := projectRootDir()
	if err != nil {
		t.Fatalf("project root dir: %v", err)
	}

	reportDir, err := resolveReportDir()
	if err != nil {
		t.Fatalf("resolve report dir: %v", err)
	}

	expected := filepath.Join(projectRoot, "custom-reports")
	if reportDir != expected {
		t.Fatalf("expected report dir %s, got %s", expected, reportDir)
	}
}
