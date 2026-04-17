package main

import (
	"context"
	"log/slog"
	"os/signal"
	"syscall"

	"devsecops-platform/internal/job"
	"devsecops-platform/internal/scanner/sast"
	"devsecops-platform/internal/store"
	"devsecops-platform/pkg/common"
)

func main() {
	cfg, err := common.LoadConfig()
	if err != nil {
		panic(err)
	}

	logger, err := common.NewLogger(cfg.LogLevel)
	if err != nil {
		panic(err)
	}

	if err := store.InitDB(cfg.Database); err != nil {
		logger.Error("database initialization failed", slog.String("error", err.Error()))
		panic(err)
	}

	jobService := job.NewService(store.GetDB(), logger)
	worker := job.NewWorker(jobService, sast.NewScanner(), logger, cfg.WorkerPollInterval)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	logger.Info("worker starting", slog.Duration("poll_interval", cfg.WorkerPollInterval))

	go worker.Run(ctx)

	<-ctx.Done()
	logger.Info("worker shutdown complete")
}
