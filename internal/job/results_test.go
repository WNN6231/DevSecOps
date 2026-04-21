package job

import (
	"testing"

	"devsecops-platform/internal/store"
)

func TestNormalizeListResultsRequest(t *testing.T) {
	tests := []struct {
		name      string
		input     ListResultsRequest
		want      ListResultsRequest
		wantError bool
	}{
		{
			name:  "defaults when empty",
			input: ListResultsRequest{},
			want: ListResultsRequest{
				Page:     defaultResultsPage,
				PageSize: defaultResultsPageSize,
			},
		},
		{
			name: "caps page size",
			input: ListResultsRequest{
				Page:     2,
				PageSize: 1000,
			},
			want: ListResultsRequest{
				Page:     2,
				PageSize: maxResultsPageSize,
			},
		},
		{
			name: "rejects invalid page",
			input: ListResultsRequest{
				Page:     -1,
				PageSize: 10,
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := normalizeListResultsRequest(tt.input)
			if tt.wantError {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if got != tt.want {
				t.Fatalf("expected %+v, got %+v", tt.want, got)
			}
		})
	}
}

func TestToResultsResponseMapsScanResults(t *testing.T) {
	results := []store.ScanResult{
		{
			JobID:          42,
			ScannerName:    "sast",
			Severity:       "high",
			RuleID:         "RULE-001",
			Title:          "Possible hardcoded secret",
			Description:    "Detected a secret-like literal.",
			FilePath:       "main.go",
			LineNumber:     12,
			Evidence:       `token := "secret"`,
			Recommendation: "Move it to env.",
			Hash:           "abc123",
		},
	}

	response := toResultsResponse(42, results, 2, 10, 11)

	if response.JobID != 42 {
		t.Fatalf("expected job id 42, got %d", response.JobID)
	}
	if len(response.Findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(response.Findings))
	}
	if response.Findings[0].Scanner != "sast" {
		t.Fatalf("expected scanner sast, got %s", response.Findings[0].Scanner)
	}
	if response.Findings[0].RuleID != "RULE-001" {
		t.Fatalf("expected rule id RULE-001, got %s", response.Findings[0].RuleID)
	}
	if response.Pagination.Page != 2 {
		t.Fatalf("expected page 2, got %d", response.Pagination.Page)
	}
	if response.Pagination.PageSize != 10 {
		t.Fatalf("expected page size 10, got %d", response.Pagination.PageSize)
	}
	if response.Pagination.Total != 11 {
		t.Fatalf("expected total 11, got %d", response.Pagination.Total)
	}
}
