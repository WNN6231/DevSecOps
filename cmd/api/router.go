package main

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"devsecops-platform/internal/job"
	"devsecops-platform/pkg/common"
)

func newRouter(logger *slog.Logger, jobHandler *job.Handler) *gin.Engine {
	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(common.RequestLogger(logger))

	router.GET("/health", func(c *gin.Context) {
		common.WriteSuccess(c, gin.H{
			"status": "ok",
		})
	})

	v1 := router.Group("/api/v1")
	jobHandler.RegisterRoutes(v1)

	router.NoRoute(func(c *gin.Context) {
		common.WriteError(c, http.StatusNotFound, "not found")
	})

	return router
}
