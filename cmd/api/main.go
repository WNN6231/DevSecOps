package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"

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

	gin.SetMode(cfg.GinMode)

	if err := store.InitDB(cfg.Database); err != nil {
		logger.Error("database initialization failed", slog.String("error", err.Error()))
		return err
	}

	jobService := job.NewService(store.GetDB(), logger, cfg.ReportDir)
	jobHandler := job.NewHandler(jobService)

	server := &http.Server{
		Addr:         cfg.HTTPAddr,
		Handler:      newRouter(logger, jobHandler),
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
	}

	logger.Info("api server starting", slog.String("addr", cfg.HTTPAddr))
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Error("api server stopped", slog.String("error", err.Error()))
		return err
	}

	return nil
}
