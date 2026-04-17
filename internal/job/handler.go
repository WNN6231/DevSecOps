package job

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"devsecops-platform/pkg/common"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) RegisterRoutes(router gin.IRoutes) {
	router.POST("/jobs", h.createJob)
	router.GET("/jobs/:id", h.getJob)
}

func (h *Handler) createJob(c *gin.Context) {
	var req CreateJobRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.WriteError(c, http.StatusBadRequest, "invalid request")
		return
	}

	job, err := h.service.Create(c.Request.Context(), req)
	if err != nil {
		common.WriteError(c, http.StatusInternalServerError, "internal error")
		return
	}

	common.WriteOK(c, job.toResponse())
}

func (h *Handler) getJob(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		common.WriteError(c, http.StatusBadRequest, "invalid job id")
		return
	}

	job, err := h.service.GetByID(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, ErrJobNotFound) {
			common.WriteError(c, http.StatusNotFound, "job not found")
			return
		}

		common.WriteError(c, http.StatusInternalServerError, "internal error")
		return
	}

	common.WriteOK(c, job.toResponse())
}
