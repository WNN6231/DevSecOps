package sast

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"devsecops-platform/pkg/common"
)

var (
	secretPattern = regexp.MustCompile("(?i)\\b[a-z0-9_]*(password|passwd|pwd|secret|token|api_?key|access_?key)[a-z0-9_]*\\b\\s*(:=|=|:)\\s*(\"([^\"\\\\]|\\\\.){4,}\"|`[^`]{4,}`)")
	execPattern   = regexp.MustCompile(`\bexec\.(Command|CommandContext)\s*\(`)
)

type Scanner struct{}

func NewScanner() *Scanner {
	return &Scanner{}
}

func (s *Scanner) Scan(ctx context.Context, repoURL, branch string) ([]common.Finding, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	repoPath, ok := resolveLocalRepoPath(repoURL)
	if !ok {
		return []common.Finding{mockFinding(repoURL, branch)}, nil
	}

	return scanRepo(ctx, repoPath)
}

func resolveLocalRepoPath(repoURL string) (string, bool) {
	if repoURL == "" {
		return "", false
	}

	info, err := os.Stat(repoURL)
	if err != nil || !info.IsDir() {
		return "", false
	}

	return repoURL, true
}

func scanRepo(ctx context.Context, repoPath string) ([]common.Finding, error) {
	findings := make([]common.Finding, 0)

	err := filepath.WalkDir(repoPath, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if err := ctx.Err(); err != nil {
			return err
		}

		if entry.IsDir() {
			return nil
		}

		if filepath.Ext(path) != ".go" {
			return nil
		}

		fileFindings, err := scanGoFile(ctx, repoPath, path)
		if err != nil {
			return err
		}

		findings = append(findings, fileFindings...)
		return nil
	})
	if err != nil {
		return nil, err
	}

	return findings, nil
}

func scanGoFile(ctx context.Context, repoPath, path string) ([]common.Finding, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	relativePath, err := filepath.Rel(repoPath, path)
	if err != nil {
		return nil, err
	}

	findings := make([]common.Finding, 0)
	scanner := bufio.NewScanner(file)
	lineNumber := 0

	for scanner.Scan() {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		lineNumber++
		line := scanner.Text()
		evidence := strings.TrimSpace(line)

		if secretPattern.MatchString(line) {
			findings = append(findings, newFinding(
				"GO_SAST_SECRET_001",
				"high",
				"Possible hardcoded secret",
				"This mock SAST scanner detected a hardcoded secret pattern in Go source.",
				filepath.ToSlash(relativePath),
				lineNumber,
				evidence,
				"Move the secret to environment variables or a secret manager.",
			))
		}

		if execPattern.MatchString(line) {
			findings = append(findings, newFinding(
				"GO_SAST_EXEC_001",
				"high",
				"Command execution usage",
				"This mock SAST scanner detected exec.Command usage in Go source.",
				filepath.ToSlash(relativePath),
				lineNumber,
				evidence,
				"Validate inputs strictly or avoid shelling out from application code.",
			))
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return findings, nil
}

func newFinding(ruleID, severity, title, description, filePath string, lineNumber int, evidence, recommendation string) common.Finding {
	hash := sha256.Sum256([]byte(strings.Join([]string{
		"sast",
		ruleID,
		filePath,
		fmt.Sprintf("%d", lineNumber),
		evidence,
	}, ":")))

	return common.Finding{
		Scanner:        "sast",
		Severity:       severity,
		RuleID:         ruleID,
		Title:          title,
		Description:    description,
		FilePath:       filePath,
		LineNumber:     lineNumber,
		Evidence:       evidence,
		Recommendation: recommendation,
		Hash:           hex.EncodeToString(hash[:]),
	}
}

func mockFinding(repoURL, branch string) common.Finding {
	hash := sha256.Sum256([]byte(repoURL + ":" + branch + ":mock-sast"))

	return common.Finding{
		Scanner:        "sast",
		Severity:       "info",
		RuleID:         "GO_MOCK_SCAN_001",
		Title:          "Mock repository scan completed",
		Description:    "The repository was not available locally, so the mock SAST scanner skipped source inspection.",
		FilePath:       "",
		LineNumber:     0,
		Evidence:       fmt.Sprintf("repo=%s branch=%s", repoURL, branch),
		Recommendation: "Provide a local repository path to enable Go file scanning in the mock scanner.",
		Hash:           hex.EncodeToString(hash[:]),
	}
}
