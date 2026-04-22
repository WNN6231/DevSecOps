package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"devsecops-platform/internal/job"
	"devsecops-platform/internal/store"
	"devsecops-platform/pkg/common"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := common.LoadConfig()
	if err != nil {
		return err
	}

	logger, err := common.NewLogger(cfg.LogLevel)
	if err != nil {
		return err
	}

	if err := store.InitDB(cfg.Database); err != nil {
		logger.Error("database initialization failed", slog.String("error", err.Error()))
		return err
	}

	jobService := job.NewService(store.GetDB(), logger, cfg.ReportDir)
	worker := job.NewWorker(jobService, logger, cfg.WorkerPollInterval)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	logger.Info("worker starting", slog.Duration("poll_interval", cfg.WorkerPollInterval))

	go worker.Run(ctx)

	<-ctx.Done()
	logger.Info("worker shutdown complete")
	return nil
}
