package job

import (
	"errors"
	"testing"
)

func TestSanitizeCreateJobInput(t *testing.T) {
	repoURL, branch, scanType, err := sanitizeCreateJobInput(" https://github.com/acme/repo.git ", " main ", []string{"sast", "sca"})
	if err != nil {
		t.Fatalf("sanitize input: %v", err)
	}

	if repoURL != "https://github.com/acme/repo.git" {
		t.Fatalf("unexpected repo url: %s", repoURL)
	}
	if branch != "main" {
		t.Fatalf("unexpected branch: %s", branch)
	}
	if len(scanType) != 2 || scanType[0] != "sast" || scanType[1] != "sca" {
		t.Fatalf("unexpected scan types: %#v", scanType)
	}
}

func TestSanitizeRepoURLRejectsUnsafeValues(t *testing.T) {
	invalidValues := []string{
		"",
		"https://user:token@example.com/repo.git",
		"https://example.com/repo.git?token=secret",
		"https://example.com/../repo.git",
		"../repo",
		"git@example.com:../repo.git",
		"file:///tmp/repo",
	}

	for _, value := range invalidValues {
		if _, err := sanitizeRepoURL(value); !errors.Is(err, ErrInvalidRepoURL) {
			t.Fatalf("expected repo url %q to be rejected, got %v", value, err)
		}
	}
}

func TestSanitizeBranchRejectsUnsafeValues(t *testing.T) {
	invalidValues := []string{"", "-main", "../main", "feature bad"}

	for _, value := range invalidValues {
		if _, err := sanitizeBranch(value); !errors.Is(err, ErrInvalidBranch) {
			t.Fatalf("expected branch %q to be rejected, got %v", value, err)
		}
	}
}

func TestRedactRepoURLForLogRemovesCredentials(t *testing.T) {
	redacted := redactRepoURLForLog("https://user:token@example.com/repo.git?token=secret")
	if redacted != "https://example.com/repo.git" {
		t.Fatalf("unexpected redacted repo url: %s", redacted)
	}
}
