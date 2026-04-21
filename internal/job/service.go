package job

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"devsecops-platform/internal/report"
	"devsecops-platform/internal/store"
	"gorm.io/gorm"
)

var ErrJobNotFound = errors.New("job not found")
var ErrInvalidJobStatus = errors.New("invalid job status")
var ErrReportNotFound = errors.New("report not found")

type Service struct {
	reportDir string
	db        *gorm.DB
	logger    *slog.Logger
}

func NewService(db *gorm.DB, logger *slog.Logger, reportDir string) *Service {
	return &Service{
		reportDir: reportDir,
		db:        db,
		logger:    logger,
	}
}

func (s *Service) CreateJob(repoURL, branch string) (Job, error) {
	return s.createJob(context.Background(), repoURL, branch, nil, false)
}

func (s *Service) GetJob(id int64) (Job, error) {
	return s.getJob(context.Background(), id)
}

func (s *Service) UpdateJobStatus(id int64, status string) (Job, error) {
	return s.updateJobStatus(context.Background(), id, status)
}

func (s *Service) Create(ctx context.Context, req CreateJobRequest) (Job, error) {
	return s.createJob(ctx, req.RepoURL, req.Branch, req.ScanType, req.BlockOnHigh)
}

func (s *Service) GetByID(ctx context.Context, id int64) (Job, error) {
	return s.getJob(ctx, id)
}

func (s *Service) GetResults(ctx context.Context, id int64, req ListResultsRequest) (ResultsResponse, error) {
	normalizedReq, err := normalizeListResultsRequest(req)
	if err != nil {
		return ResultsResponse{}, err
	}

	job, err := s.getJob(ctx, id)
	if err != nil {
		return ResultsResponse{}, err
	}

	offset := (normalizedReq.Page - 1) * normalizedReq.PageSize
	results, total, err := store.ListScanResults(ctx, s.db, job.ID, offset, normalizedReq.PageSize)
	if err != nil {
		return ResultsResponse{}, err
	}

	return toResultsResponse(job.ID, results, normalizedReq.Page, normalizedReq.PageSize, total), nil
}

func (s *Service) GetReport(ctx context.Context, id int64) (ReportResponse, error) {
	job, err := s.getJob(ctx, id)
	if err != nil {
		return ReportResponse{}, err
	}

	content, err := report.ReadMarkdownReport(s.reportDir, job.ID)
	if err != nil {
		if errors.Is(err, report.ErrReportNotFound) {
			return ReportResponse{}, ErrReportNotFound
		}

		return ReportResponse{}, err
	}

	return ReportResponse{
		JobID:   job.ID,
		Content: content,
	}, nil
}

func (s *Service) createJob(ctx context.Context, repoURL, branch string, scanType []string, blockOnHigh bool) (Job, error) {
	encodedScanType, err := encodeScanType(scanType)
	if err != nil {
		return Job{}, err
	}

	record := store.ScanJob{
		RepoURL:     repoURL,
		Branch:      branch,
		ScanType:    encodedScanType,
		BlockOnHigh: blockOnHigh,
		Status:      StatusPending,
	}

	if err := s.db.WithContext(ctx).Create(&record).Error; err != nil {
		return Job{}, err
	}

	job, err := fromRecord(record)
	if err != nil {
		return Job{}, err
	}

	s.logger.InfoContext(ctx, "job created",
		slog.Int64("job_id", job.ID),
		slog.String("status", job.Status),
		slog.String("repo_url", job.RepoURL),
	)

	return job, nil
}

func (s *Service) getJob(ctx context.Context, id int64) (Job, error) {
	var record store.ScanJob
	if err := s.db.WithContext(ctx).First(&record, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return Job{}, ErrJobNotFound
		}

		return Job{}, err
	}

	job, err := fromRecord(record)
	if err != nil {
		return Job{}, err
	}

	s.logger.InfoContext(ctx, "job loaded",
		slog.Int64("job_id", job.ID),
		slog.String("status", job.Status),
	)

	return job, nil
}

func (s *Service) updateJobStatus(ctx context.Context, id int64, status string) (Job, error) {
	if !isValidStatus(status) {
		return Job{}, ErrInvalidJobStatus
	}

	var record store.ScanJob
	if err := s.db.WithContext(ctx).First(&record, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return Job{}, ErrJobNotFound
		}

		return Job{}, err
	}

	applyStatusTimestamps(&record, status)

	if err := s.db.WithContext(ctx).Model(&record).Updates(map[string]interface{}{
		"status":      record.Status,
		"started_at":  record.StartedAt,
		"finished_at": record.FinishedAt,
	}).Error; err != nil {
		return Job{}, err
	}

	job, err := fromRecord(record)
	if err != nil {
		return Job{}, err
	}

	s.logger.InfoContext(ctx, "job status updated",
		slog.Int64("job_id", job.ID),
		slog.String("status", job.Status),
	)

	return job, nil
}

func isValidStatus(status string) bool {
	switch status {
	case StatusPending, StatusRunning, StatusSuccess, StatusFailed, StatusBlocked:
		return true
	default:
		return false
	}
}

func applyStatusTimestamps(record *store.ScanJob, status string) {
	now := record.CreatedAt
	if now.IsZero() {
		now = timeNowUTC()
	}

	record.Status = status

	switch status {
	case StatusPending:
		record.StartedAt = nil
		record.FinishedAt = nil
	case StatusRunning:
		if record.StartedAt == nil {
			startedAt := timeNowUTC()
			record.StartedAt = &startedAt
		}
		record.FinishedAt = nil
	case StatusSuccess, StatusFailed, StatusBlocked:
		if record.StartedAt == nil {
			startedAt := now
			record.StartedAt = &startedAt
		}
		finishedAt := timeNowUTC()
		record.FinishedAt = &finishedAt
	}
}

func timeNowUTC() time.Time {
	return time.Now().UTC()
}
