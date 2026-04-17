package sast

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"devsecops-platform/pkg/common"
)

type Scanner struct{}

func NewScanner() *Scanner {
	return &Scanner{}
}

func (s *Scanner) Scan(repoURL, branch string) ([]common.Finding, error) {
	hash := sha256.Sum256([]byte(repoURL + ":" + branch + ":mock-sast"))

	finding := common.Finding{
		Scanner:        "sast",
		Severity:       "medium",
		RuleID:         "GO_MOCK_001",
		Title:          "Mock hardcoded secret pattern",
		Description:    "Mock SAST scanner generated a deterministic finding for the queued repository.",
		FilePath:       "main.go",
		LineNumber:     42,
		Evidence:       fmt.Sprintf("repo=%s branch=%s", repoURL, branch),
		Recommendation: "Replace the mock scanner with a real Go SAST rule in the next phase.",
		Hash:           hex.EncodeToString(hash[:]),
	}

	return []common.Finding{finding}, nil
}
