package job

import (
	"context"
	"encoding/json"
	"errors"
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
}

func NewWorker(service *Service, logger *slog.Logger, pollInterval time.Duration) *Worker {
	return &Worker{
		service:      service,
		logger:       logger,
		pollInterval: pollInterval,
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
	processed, err := w.ProcessNext(ctx)
	if err != nil {
		w.logger.Error("worker cycle failed", slog.String("error", err.Error()))
		return
	}

	if processed {
		w.logger.Info("worker cycle completed")
	}
}

func (w *Worker) ProcessNext(ctx context.Context) (bool, error) {
	job, found, err := w.service.claimNextPendingJob(ctx)
	if err != nil || !found {
		return false, err
	}

	findings, err := w.scan(job)
	if err != nil {
		return true, w.failJob(ctx, job.ID, err)
	}

	aggregated := report.Aggregate(findings)

	if err := w.service.saveResults(ctx, job.ID, aggregated.Findings); err != nil {
		return true, w.failJob(ctx, job.ID, err)
	}

	reportPath, err := report.WriteMarkdownReport(w.service.reportDir, job.ID, aggregated)
	if err != nil {
		return true, w.failJob(ctx, job.ID, err)
	}

	if err := w.service.saveReport(ctx, job.ID, reportPath, aggregated); err != nil {
		return true, w.failJob(ctx, job.ID, err)
	}

	finalStatus := determineFinalStatus(job, aggregated.Findings)
	if _, err := w.service.updateJobStatus(ctx, job.ID, finalStatus); err != nil {
		return true, err
	}

	w.logger.InfoContext(ctx, "job processed",
		slog.Int64("job_id", job.ID),
		slog.String("status", finalStatus),
		slog.Int("result_count", len(aggregated.Findings)),
		slog.Any("severity_counts", aggregated.Counts),
		slog.Int("risk_score", aggregated.TotalRiskScore),
		slog.String("report_path", reportPath),
	)

	return true, nil
}

func (w *Worker) scan(job Job) ([]common.Finding, error) {
	return scanner.RunScan(scanner.Job{
		RepoURL:  job.RepoURL,
		Branch:   job.Branch,
		ScanType: job.ScanType,
	})
}

func (w *Worker) failJob(ctx context.Context, jobID int64, scanErr error) error {
	if _, err := w.service.updateJobStatus(ctx, jobID, StatusFailed); err != nil {
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
	if len(findings) == 0 {
		return nil
	}

	records := buildScanResults(jobID, findings)

	return s.db.WithContext(ctx).Create(&records).Error
}

func (s *Service) saveReport(ctx context.Context, jobID int64, reportPath string, aggregated report.AggregatedResult) error {
	record, err := buildScanReport(jobID, reportPath, aggregated)
	if err != nil {
		return err
	}

	return s.db.WithContext(ctx).Create(&record).Error
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
