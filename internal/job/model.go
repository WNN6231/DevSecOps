package job

import (
	"encoding/json"
	"errors"
	"time"

	"devsecops-platform/internal/store"
)

const (
	StatusPending = "pending"
	StatusRunning = "running"
	StatusSuccess = "success"
	StatusFailed  = "failed"
	StatusBlocked = "blocked"
)

type CreateJobRequest struct {
	RepoURL     string   `json:"repo_url"`
	Branch      string   `json:"branch" binding:"required"`
	ScanType    []string `json:"scan_type" binding:"required,min=1,dive,required,oneof=sast sca secret dast"`
	BlockOnHigh bool     `json:"block_on_high"`
}

type Job struct {
	ID          int64
	RepoURL     string
	Branch      string
	ScanType    []string
	BlockOnHigh bool
	Status      string
	CreatedAt   time.Time
	StartedAt   *time.Time
	FinishedAt  *time.Time
}

type JobResponse struct {
	JobID       int64      `json:"job_id"`
	Status      string     `json:"status"`
	RepoURL     string     `json:"repo_url"`
	Branch      string     `json:"branch"`
	ScanType    []string   `json:"scan_type"`
	BlockOnHigh bool       `json:"block_on_high"`
	CreatedAt   time.Time  `json:"created_at"`
	StartedAt   *time.Time `json:"started_at,omitempty"`
	FinishedAt  *time.Time `json:"finished_at,omitempty"`
}

type ListResultsRequest struct {
	Page     int `form:"page"`
	PageSize int `form:"page_size"`
}

type ResultFindingResponse struct {
	Scanner        string `json:"scanner"`
	Severity       string `json:"severity"`
	RuleID         string `json:"rule_id"`
	Title          string `json:"title"`
	Description    string `json:"description"`
	FilePath       string `json:"file_path"`
	LineNumber     int    `json:"line_number"`
	Evidence       string `json:"evidence"`
	Recommendation string `json:"recommendation"`
	Hash           string `json:"hash"`
}

type ResultsPagination struct {
	Page     int   `json:"page"`
	PageSize int   `json:"page_size"`
	Total    int64 `json:"total"`
}

type ResultsResponse struct {
	JobID      int64                   `json:"job_id"`
	Findings   []ResultFindingResponse `json:"findings"`
	Pagination ResultsPagination       `json:"pagination"`
}

type ReportResponse struct {
	JobID   int64  `json:"job_id"`
	Content string `json:"content"`
}

const (
	defaultResultsPage     = 1
	defaultResultsPageSize = 20
	maxResultsPageSize     = 100
)

var ErrInvalidPagination = errors.New("invalid pagination")

func fromRecord(record store.ScanJob) (Job, error) {
	scanType, err := decodeScanType(record.ScanType)
	if err != nil {
		return Job{}, err
	}

	return Job{
		ID:          int64(record.ID),
		RepoURL:     record.RepoURL,
		Branch:      record.Branch,
		ScanType:    scanType,
		BlockOnHigh: record.BlockOnHigh,
		Status:      record.Status,
		CreatedAt:   record.CreatedAt,
		StartedAt:   record.StartedAt,
		FinishedAt:  record.FinishedAt,
	}, nil
}

func (j Job) toResponse() JobResponse {
	return JobResponse{
		JobID:       j.ID,
		Status:      j.Status,
		RepoURL:     j.RepoURL,
		Branch:      j.Branch,
		ScanType:    append([]string(nil), j.ScanType...),
		BlockOnHigh: j.BlockOnHigh,
		CreatedAt:   j.CreatedAt,
		StartedAt:   j.StartedAt,
		FinishedAt:  j.FinishedAt,
	}
}

func normalizeListResultsRequest(req ListResultsRequest) (ListResultsRequest, error) {
	if req.Page == 0 {
		req.Page = defaultResultsPage
	}
	if req.PageSize == 0 {
		req.PageSize = defaultResultsPageSize
	}
	if req.Page < 1 || req.PageSize < 1 {
		return ListResultsRequest{}, ErrInvalidPagination
	}
	if req.PageSize > maxResultsPageSize {
		req.PageSize = maxResultsPageSize
	}

	return req, nil
}

func toResultsResponse(jobID int64, results []store.ScanResult, page, pageSize int, total int64) ResultsResponse {
	findings := make([]ResultFindingResponse, 0, len(results))
	for _, result := range results {
		findings = append(findings, ResultFindingResponse{
			Scanner:        result.ScannerName,
			Severity:       result.Severity,
			RuleID:         result.RuleID,
			Title:          result.Title,
			Description:    result.Description,
			FilePath:       result.FilePath,
			LineNumber:     result.LineNumber,
			Evidence:       result.Evidence,
			Recommendation: result.Recommendation,
			Hash:           result.Hash,
		})
	}

	return ResultsResponse{
		JobID:    jobID,
		Findings: findings,
		Pagination: ResultsPagination{
			Page:     page,
			PageSize: pageSize,
			Total:    total,
		},
	}
}

func encodeScanType(scanType []string) (string, error) {
	if len(scanType) == 0 {
		return "[]", nil
	}

	encoded, err := json.Marshal(scanType)
	if err != nil {
		return "", err
	}

	return string(encoded), nil
}

func decodeScanType(value string) ([]string, error) {
	if value == "" {
		return []string{}, nil
	}

	var scanType []string
	if err := json.Unmarshal([]byte(value), &scanType); err != nil {
		return nil, err
	}

	return scanType, nil
}
