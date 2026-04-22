package job

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"devsecops-platform/internal/report"
	"devsecops-platform/internal/scanner"
	"devsecops-platform/internal/store"
	"devsecops-platform/pkg/common"

	"gorm.io/gorm"
)

type Worker struct {
	service      *Service
	logger       *slog.Logger
	pollInterval time.Duration
	scanFunc     func(context.Context, Job, *slog.Logger) ([]common.Finding, error)
}

func NewWorker(service *Service, logger *slog.Logger, pollInterval time.Duration) *Worker {
	return &Worker{
		service:      service,
		logger:       logger,
		pollInterval: pollInterval,
		scanFunc:     runScanWithContext,
	}
}

func (w *Worker) Run(ctx context.Context) {
	ticker := time.NewTicker(w.pollInterval)
	defer ticker.Stop()

	w.runCycle(ctx)

	for {
		select {
		case <-ctx.Done():
			w.logger.Info("worker loop stopped")
			return
		case <-ticker.C:
			w.runCycle(ctx)
		}
	}
}

func (w *Worker) runCycle(ctx context.Context) {
	defer func() {
		if recovered := recover(); recovered != nil {
			w.logger.Error("worker cycle panicked", slog.Any("panic", recovered))
		}
	}()

	if err := w.service.markTimedOutRunningJobs(ctx, timeNowUTC()); err != nil {
		w.logger.Error("worker timed-out job recovery failed", slog.String("error", err.Error()))
		return
	}

	_, err := w.ProcessNext(ctx)
	if err != nil {
		w.logger.Error("worker cycle failed", slog.String("error", err.Error()))
		return
	}
}

func (w *Worker) ProcessNext(ctx context.Context) (bool, error) {
	job, found, err := w.service.claimNextPendingJob(ctx)
	if err != nil || !found {
		return false, err
	}

	jobLogger := w.logger.With(slog.Int64("job_id", job.ID))
	finalStatus, aggregated, reportPath, attemptCount, scanDuration, err := w.processJobWithRetry(ctx, job, jobLogger)
	if err != nil {
		jobLogger.ErrorContext(ctx, "job processing failed",
			slog.Int("attempt_count", attemptCount),
			slog.Duration("total_scan_duration", scanDuration),
			slog.String("error", err.Error()),
			slog.Any("metrics", w.service.MetricsSnapshot()),
		)
		return true, err
	}

	jobLogger.InfoContext(ctx, "job risk summary",
		slog.Any("severity_counts", aggregated.Counts),
		slog.Int("risk_score", aggregated.TotalRiskScore),
		slog.String("status", finalStatus),
	)

	jobLogger.InfoContext(ctx, "job processed",
		slog.String("status", finalStatus),
		slog.Int("attempt_count", attemptCount),
		slog.Int("result_count", len(aggregated.Findings)),
		slog.Duration("total_scan_duration", scanDuration),
		slog.String("report_path", reportPath),
		slog.Any("metrics", w.service.MetricsSnapshot()),
	)

	return true, nil
}

func (w *Worker) processJobWithRetry(ctx context.Context, job Job, logger *slog.Logger) (string, report.AggregatedResult, string, int, time.Duration, error) {
	maxExecutionTimeSec := job.MaxExecutionTimeSec
	if maxExecutionTimeSec <= 0 {
		maxExecutionTimeSec = defaultMaxExecutionTimeSec
	}

	jobStartedAt := time.Now()
	jobCtx, cancel := context.WithTimeout(ctx, time.Duration(maxExecutionTimeSec)*time.Second)
	defer cancel()

	for attempt := 1; attempt <= maxJobAttempts; attempt++ {
		if err := w.service.incrementAttemptCount(ctx, job.ID); err != nil {
			return "", report.AggregatedResult{}, "", attempt, time.Since(jobStartedAt), w.failJob(ctx, job.ID, err)
		}
		job.AttemptCount = attempt

		logger.InfoContext(ctx, "job attempt started",
			slog.Int("attempt", attempt),
			slog.Int("max_attempts", maxJobAttempts),
			slog.Int("max_execution_time_sec", maxExecutionTimeSec),
		)

		aggregated, reportPath, err := w.processSingleAttempt(jobCtx, job, logger)
		if err == nil {
			if err := w.service.finalizeJobSuccess(ctx, job, reportPath, aggregated); err != nil {
				return "", report.AggregatedResult{}, "", attempt, time.Since(jobStartedAt), w.failJob(ctx, job.ID, err)
			}
			return determineFinalStatus(job, aggregated.Findings), aggregated, reportPath, attempt, time.Since(jobStartedAt), nil
		}

		if errors.Is(err, context.DeadlineExceeded) || errors.Is(jobCtx.Err(), context.DeadlineExceeded) {
			logger.ErrorContext(ctx, "job attempt timed out",
				slog.Int("attempt", attempt),
				slog.Duration("elapsed", time.Since(jobStartedAt)),
				slog.String("error", err.Error()),
			)
			return "", report.AggregatedResult{}, "", attempt, time.Since(jobStartedAt), w.failJob(ctx, job.ID, err)
		}

		if attempt == maxJobAttempts {
			return "", report.AggregatedResult{}, "", attempt, time.Since(jobStartedAt), w.failJob(ctx, job.ID, err)
		}

		logger.WarnContext(ctx, "job attempt failed, retrying",
			slog.Int("attempt", attempt),
			slog.Int("max_attempts", maxJobAttempts),
			slog.String("error", err.Error()),
		)
	}

	return "", report.AggregatedResult{}, "", maxJobAttempts, time.Since(jobStartedAt), w.failJob(ctx, job.ID, errors.New("job retry loop exited unexpectedly"))
}

func (w *Worker) processSingleAttempt(ctx context.Context, job Job, logger *slog.Logger) (aggregated report.AggregatedResult, reportPath string, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			err = fmt.Errorf("worker panic recovered: %v", recovered)
		}
	}()

	findings, err := w.scan(ctx, job, logger)
	if err != nil {
		return report.AggregatedResult{}, "", err
	}

	aggregated = report.Aggregate(findings)
	reportPath, err = report.WriteMarkdownReport(w.service.reportDir, job.ID, aggregated)
	if err != nil {
		return report.AggregatedResult{}, "", err
	}

	return aggregated, reportPath, nil
}

func (w *Worker) scan(ctx context.Context, job Job, logger *slog.Logger) ([]common.Finding, error) {
	return w.scanFunc(ctx, job, logger)
}

func runScanWithContext(ctx context.Context, job Job, logger *slog.Logger) ([]common.Finding, error) {
	return scanner.RunScan(ctx, scanner.Job{
		RepoURL:  job.RepoURL,
		Branch:   job.Branch,
		ScanType: job.ScanType,
		ObserveScanner: func(name string, duration time.Duration, findingCount int, err error) {
			attrs := []any{
				slog.String("scanner", name),
				slog.Duration("duration", duration),
				slog.Int("finding_count", findingCount),
			}
			if err != nil {
				attrs = append(attrs, slog.String("error", err.Error()))
				logger.ErrorContext(ctx, "scanner execution completed", attrs...)
				return
			}

			logger.InfoContext(ctx, "scanner execution completed", attrs...)
		},
	})
}

func (w *Worker) failJob(ctx context.Context, jobID int64, scanErr error) error {
	if _, err := w.service.updateJobStatus(ctx, jobID, StatusFailed); err != nil && !errors.Is(err, ErrInvalidJobStatusTransition) {
		return errors.Join(scanErr, err)
	}

	return scanErr
}

func (s *Service) claimNextPendingJob(ctx context.Context) (Job, bool, error) {
	var record store.ScanJob
	if err := s.db.WithContext(ctx).
		Where("status = ?", StatusPending).
		Order("id ASC").
		First(&record).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return Job{}, false, nil
		}

		return Job{}, false, err
	}

	startedAt := timeNowUTC()
	result := s.db.WithContext(ctx).
		Model(&store.ScanJob{}).
		Where("id = ? AND status = ?", record.ID, StatusPending).
		Updates(map[string]interface{}{
			"status":      StatusRunning,
			"started_at":  startedAt,
			"finished_at": nil,
		})
	if result.Error != nil {
		return Job{}, false, result.Error
	}

	if result.RowsAffected == 0 {
		return Job{}, false, nil
	}

	record.Status = StatusRunning
	record.StartedAt = &startedAt
	record.FinishedAt = nil

	job, err := fromRecord(record)
	if err != nil {
		return Job{}, false, err
	}

	s.logger.InfoContext(ctx, "pending job claimed",
		slog.Int64("job_id", job.ID),
		slog.String("status", job.Status),
	)

	return job, true, nil
}

func (s *Service) saveResults(ctx context.Context, jobID int64, findings []common.Finding) error {
	return s.saveResultsWithDB(ctx, s.db, jobID, findings)
}

func (s *Service) saveResultsWithDB(ctx context.Context, db *gorm.DB, jobID int64, findings []common.Finding) error {
	if len(findings) == 0 {
		return nil
	}

	records := buildScanResults(jobID, findings)

	return db.WithContext(ctx).Create(&records).Error
}

func (s *Service) saveReport(ctx context.Context, jobID int64, reportPath string, aggregated report.AggregatedResult) error {
	return s.saveReportWithDB(ctx, s.db, jobID, reportPath, aggregated)
}

func (s *Service) saveReportWithDB(ctx context.Context, db *gorm.DB, jobID int64, reportPath string, aggregated report.AggregatedResult) error {
	record, err := buildScanReport(jobID, reportPath, aggregated)
	if err != nil {
		return err
	}

	return db.WithContext(ctx).Create(&record).Error
}

func buildScanResults(jobID int64, findings []common.Finding) []store.ScanResult {
	records := make([]store.ScanResult, 0, len(findings))
	for _, finding := range findings {
		records = append(records, store.ScanResult{
			JobID:          uint64(jobID),
			ScannerName:    finding.Scanner,
			Severity:       finding.Severity,
			RuleID:         finding.RuleID,
			FilePath:       finding.FilePath,
			LineNumber:     finding.LineNumber,
			Title:          finding.Title,
			Description:    finding.Description,
			Evidence:       finding.Evidence,
			Recommendation: finding.Recommendation,
			Hash:           finding.Hash,
		})
	}

	return records
}

func buildScanReport(jobID int64, reportPath string, aggregated report.AggregatedResult) (store.ScanReport, error) {
	summaryJSON, err := json.Marshal(aggregated.Counts)
	if err != nil {
		return store.ScanReport{}, err
	}

	return store.ScanReport{
		JobID:       uint64(jobID),
		ReportPath:  reportPath,
		SummaryJSON: string(summaryJSON),
		HighCount:   aggregated.Counts["high"],
		MediumCount: aggregated.Counts["medium"],
		LowCount:    aggregated.Counts["low"],
		RiskScore:   aggregated.TotalRiskScore,
	}, nil
}

func determineFinalStatus(job Job, findings []common.Finding) string {
	if !job.BlockOnHigh {
		return StatusSuccess
	}

	for _, finding := range findings {
		if finding.Severity == "critical" || finding.Severity == "high" {
			return StatusBlocked
		}
	}

	return StatusSuccess
}
