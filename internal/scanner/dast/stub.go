package dast

import "devsecops-platform/pkg/common"

// Scanner is a placeholder for the DAST (Dynamic Application Security Testing) engine.
type Scanner struct{}

func NewScanner() *Scanner { return &Scanner{} }

func (s *Scanner) Scan(_, _ string) ([]common.Finding, error) {
	return []common.Finding{}, nil
}
