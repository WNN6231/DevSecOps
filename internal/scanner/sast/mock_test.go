package sast

import (
	"os"
	"path/filepath"
	"testing"
)

func TestScanDetectsSecretAndExecUsage(t *testing.T) {
	repoPath := t.TempDir()
	source := `package main

import "os/exec"

func main() {
	apiToken := "super-secret-token"
	_ = exec.Command("sh", "-c", "echo test")
}`

	filePath := filepath.Join(repoPath, "main.go")
	if err := os.WriteFile(filePath, []byte(source), 0o644); err != nil {
		t.Fatalf("write source file: %v", err)
	}

	findings, err := NewScanner().Scan(repoPath, "main")
	if err != nil {
		t.Fatalf("scan repo: %v", err)
	}

	if len(findings) != 2 {
		t.Fatalf("expected 2 findings, got %d", len(findings))
	}

	if findings[0].RuleID != "GO_SAST_SECRET_001" {
		t.Fatalf("expected first finding to be secret rule, got %s", findings[0].RuleID)
	}

	if findings[0].FilePath != "main.go" {
		t.Fatalf("expected relative file path, got %s", findings[0].FilePath)
	}

	if findings[1].RuleID != "GO_SAST_EXEC_001" {
		t.Fatalf("expected second finding to be exec rule, got %s", findings[1].RuleID)
	}
}

func TestScanFallsBackToMockFindingWhenRepoIsMissing(t *testing.T) {
	findings, err := NewScanner().Scan(filepath.Join(t.TempDir(), "missing"), "main")
	if err != nil {
		t.Fatalf("scan missing repo: %v", err)
	}

	if len(findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(findings))
	}

	if findings[0].RuleID != "GO_MOCK_SCAN_001" {
		t.Fatalf("expected mock scan rule, got %s", findings[0].RuleID)
	}

	if findings[0].Severity != "info" {
		t.Fatalf("expected info severity, got %s", findings[0].Severity)
	}
}
