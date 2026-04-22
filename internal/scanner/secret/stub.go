package secret

import "context"
import "devsecops-platform/pkg/common"

// Scanner is a placeholder for the Secret scanning engine.
type Scanner struct{}

func NewScanner() *Scanner { return &Scanner{} }

func (s *Scanner) Scan(ctx context.Context, _, _ string) ([]common.Finding, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	return []common.Finding{}, nil
}
