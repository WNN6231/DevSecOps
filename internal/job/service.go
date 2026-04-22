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
var ErrInvalidJobStatusTransition = errors.New("invalid job status transition")
var ErrInvalidRepoURL = errors.New("invalid repo url")
var ErrInvalidBranch = errors.New("invalid branch")
var ErrInvalidScanType = errors.New("invalid scan type")
var ErrInvalidMaxExecutionTime = errors.New("invalid max execution time")
var ErrReportNotFound = errors.New("report not found")

type Service struct {
	reportDir string
	db        *gorm.DB
	logger    *slog.Logger
	metrics   *Metrics
}

func NewService(db *gorm.DB, logger *slog.Logger, reportDir string) *Service {
	return &Service{
		reportDir: reportDir,
		db:        db,
		logger:    logger,
		metrics:   &Metrics{},
	}
}

func (s *Service) CreateJob(repoURL, branch string) (Job, error) {
	return s.createJob(context.Background(), repoURL, branch, nil, false, 0)
}

func (s *Service) GetJob(id int64) (Job, error) {
	return s.getJob(context.Background(), id)
}

func (s *Service) UpdateJobStatus(id int64, status string) (Job, error) {
	return s.updateJobStatus(context.Background(), id, status)
}

func (s *Service) Create(ctx context.Context, req CreateJobRequest) (Job, error) {
	return s.createJob(ctx, req.RepoURL, req.Branch, req.ScanType, req.BlockOnHigh, req.MaxExecutionTimeSec)
}

func (s *Service) GetByID(ctx context.Context, id int64) (Job, error) {
	return s.getJob(ctx, id)
}

func (s *Service) MetricsSnapshot() MetricsSnapshot {
	return s.metrics.Snapshot()
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

func (s *Service) createJob(ctx context.Context, repoURL, branch string, scanType []string, blockOnHigh bool, maxExecutionTimeSec int) (Job, error) {
	repoURL, branch, scanType, err := sanitizeCreateJobInput(repoURL, branch, scanType)
	if err != nil {
		return Job{}, err
	}

	encodedScanType, err := encodeScanType(scanType)
	if err != nil {
		return Job{}, err
	}
	normalizedTimeoutSec, err := normalizeMaxExecutionTimeSec(maxExecutionTimeSec)
	if err != nil {
		return Job{}, err
	}

	record := store.ScanJob{
		RepoURL:             repoURL,
		Branch:              branch,
		ScanType:            encodedScanType,
		BlockOnHigh:         blockOnHigh,
		Status:              StatusPending,
		MaxExecutionTimeSec: normalizedTimeoutSec,
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
		slog.String("repo_url", redactRepoURLForLog(job.RepoURL)),
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

	if err := validateStatusTransition(record.Status, status); err != nil {
		return Job{}, err
	}

	previousStatus := record.Status
	applyStatusTimestamps(&record, status)

	if err := s.db.WithContext(ctx).Model(&record).Updates(map[string]interface{}{
		"status":      record.Status,
		"started_at":  record.StartedAt,
		"finished_at": record.FinishedAt,
	}).Error; err != nil {
		return Job{}, err
	}

	if previousStatus != record.Status {
		s.recordStatusMetrics(record.Status)
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

func (s *Service) incrementAttemptCount(ctx context.Context, jobID int64) error {
	result := s.db.WithContext(ctx).
		Model(&store.ScanJob{}).
		Where("id = ? AND status = ? AND attempt_count < ?", jobID, StatusRunning, maxJobAttempts).
		Update("attempt_count", gorm.Expr("attempt_count + 1"))
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrInvalidJobStatusTransition
	}

	return nil
}

func (s *Service) finalizeJobSuccess(ctx context.Context, job Job, reportPath string, aggregated report.AggregatedResult) error {
	finalStatus := determineFinalStatus(job, aggregated.Findings)

	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := s.saveResultsWithDB(ctx, tx, job.ID, aggregated.Findings); err != nil {
			return err
		}
		if err := s.saveReportWithDB(ctx, tx, job.ID, reportPath, aggregated); err != nil {
			return err
		}
		_, err := s.updateJobStatusWithDB(ctx, tx, job.ID, finalStatus)
		return err
	})
}

func (s *Service) markTimedOutRunningJobs(ctx context.Context, now time.Time) error {
	var records []store.ScanJob
	if err := s.db.WithContext(ctx).
		Where("status = ?", StatusRunning).
		Find(&records).Error; err != nil {
		return err
	}

	for _, record := range records {
		if !jobExecutionTimedOut(record, now) {
			continue
		}

		attrs := []any{
			slog.Uint64("job_id", record.ID),
			slog.Int("max_execution_time_sec", record.MaxExecutionTimeSec),
		}
		if record.StartedAt != nil {
			attrs = append(attrs, slog.Time("started_at", *record.StartedAt))
		}
		s.logger.WarnContext(ctx, "timed-out running job marked failed", attrs...)

		if _, err := s.updateJobStatus(ctx, int64(record.ID), StatusFailed); err != nil && !errors.Is(err, ErrInvalidJobStatusTransition) {
			return err
		}
	}

	return nil
}

func (s *Service) updateJobStatusWithDB(ctx context.Context, db *gorm.DB, id int64, status string) (Job, error) {
	if !isValidStatus(status) {
		return Job{}, ErrInvalidJobStatus
	}

	var record store.ScanJob
	if err := db.WithContext(ctx).First(&record, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return Job{}, ErrJobNotFound
		}

		return Job{}, err
	}

	if err := validateStatusTransition(record.Status, status); err != nil {
		return Job{}, err
	}

	previousStatus := record.Status
	applyStatusTimestamps(&record, status)

	if err := db.WithContext(ctx).Model(&record).Updates(map[string]interface{}{
		"status":      record.Status,
		"started_at":  record.StartedAt,
		"finished_at": record.FinishedAt,
	}).Error; err != nil {
		return Job{}, err
	}

	if previousStatus != record.Status {
		s.recordStatusMetrics(record.Status)
	}

	return fromRecord(record)
}

func (s *Service) recordStatusMetrics(status string) {
	switch status {
	case StatusSuccess, StatusFailed, StatusBlocked:
		s.metrics.recordTerminalStatus(status)
	}
}

func isValidStatus(status string) bool {
	switch status {
	case StatusPending, StatusRunning, StatusSuccess, StatusFailed, StatusBlocked:
		return true
	default:
		return false
	}
}

func validateStatusTransition(current, next string) error {
	if current == next {
		return nil
	}

	switch current {
	case StatusPending:
		if next == StatusRunning {
			return nil
		}
	case StatusRunning:
		if next == StatusSuccess || next == StatusFailed || next == StatusBlocked {
			return nil
		}
	}

	return ErrInvalidJobStatusTransition
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

func jobExecutionTimedOut(record store.ScanJob, now time.Time) bool {
	if record.Status != StatusRunning {
		return false
	}
	if record.StartedAt == nil {
		return true
	}

	maxExecutionTimeSec := record.MaxExecutionTimeSec
	if maxExecutionTimeSec <= 0 {
		maxExecutionTimeSec = defaultMaxExecutionTimeSec
	}

	deadline := record.StartedAt.Add(time.Duration(maxExecutionTimeSec) * time.Second)
	return !now.Before(deadline)
}
