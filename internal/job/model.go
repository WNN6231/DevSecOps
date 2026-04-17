package job

import (
	"time"

	"devsecops-platform/internal/store"
)

const (
	StatusPending = "pending"
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
}

type JobResponse struct {
	JobID       int64     `json:"job_id"`
	Status      string    `json:"status"`
	RepoURL     string    `json:"repo_url"`
	Branch      string    `json:"branch"`
	ScanType    []string  `json:"scan_type"`
	BlockOnHigh bool      `json:"block_on_high"`
	CreatedAt   time.Time `json:"created_at"`
}

func fromRecord(record store.JobRecord) Job {
	return Job{
		ID:          record.ID,
		RepoURL:     record.RepoURL,
		Branch:      record.Branch,
		ScanType:    append([]string(nil), record.ScanType...),
		BlockOnHigh: record.BlockOnHigh,
		Status:      record.Status,
		CreatedAt:   record.CreatedAt,
	}
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
	}
}
