package job

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"devsecops-platform/internal/store"
)

var ErrJobNotFound = errors.New("job not found")

type Service struct {
	store  *store.JobStore
	logger *slog.Logger
}

func NewService(store *store.JobStore, logger *slog.Logger) *Service {
	return &Service{
		store:  store,
		logger: logger,
	}
}

func (s *Service) Create(ctx context.Context, req CreateJobRequest) (Job, error) {
	record, err := s.store.Create(store.JobRecord{
		RepoURL:     req.RepoURL,
		Branch:      req.Branch,
		ScanType:    append([]string(nil), req.ScanType...),
		BlockOnHigh: req.BlockOnHigh,
		Status:      StatusPending,
		CreatedAt:   time.Now().UTC(),
	})
	if err != nil {
		return Job{}, err
	}

	job := fromRecord(record)

	s.logger.InfoContext(ctx, "job created",
		slog.Int64("job_id", job.ID),
		slog.String("status", job.Status),
		slog.String("repo_url", job.RepoURL),
	)

	return job, nil
}

func (s *Service) GetByID(ctx context.Context, id int64) (Job, error) {
	record, ok := s.store.GetByID(id)
	if !ok {
		return Job{}, ErrJobNotFound
	}

	job := fromRecord(record)

	s.logger.InfoContext(ctx, "job loaded",
		slog.Int64("job_id", job.ID),
		slog.String("status", job.Status),
	)

	return job, nil
}
