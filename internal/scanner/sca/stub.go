package sca

import "devsecops-platform/pkg/common"

// Scanner is a placeholder for the SCA (Software Composition Analysis) engine.
type Scanner struct{}

func NewScanner() *Scanner { return &Scanner{} }

func (s *Scanner) Scan(_, _ string) ([]common.Finding, error) {
	return []common.Finding{}, nil
}
