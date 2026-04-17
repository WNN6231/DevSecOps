package main

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"devsecops-platform/internal/job"
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

	gin.SetMode(cfg.GinMode)

	if err := store.InitDB(cfg.Database); err != nil {
		logger.Error("database initialization failed", slog.String("error", err.Error()))
		panic(err)
	}

	jobService := job.NewService(store.GetDB(), logger)
	jobHandler := job.NewHandler(jobService)

	router := newRouter(logger, jobHandler)

	server := &http.Server{
		Addr:         cfg.HTTPAddr,
		Handler:      router,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
	}

	logger.Info("api server starting", slog.String("addr", cfg.HTTPAddr))
	if err := server.ListenAndServe(); err != nil {
		logger.Error("api server stopped", slog.String("error", err.Error()))
	}
}
