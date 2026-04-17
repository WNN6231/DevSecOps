package job

import (
	"encoding/json"
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
	RepoURL     string   `json:"repo_url" binding:"required,url"`
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
