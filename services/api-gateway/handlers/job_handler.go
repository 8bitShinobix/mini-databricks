package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/8bitShinobix/mini-databricks/internal/metrics"
	"github.com/8bitShinobix/mini-databricks/internal/service"
	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

type JobHandler struct {
	jobService *service.JobService
}

func NewJobHandler(jobService *service.JobService) *JobHandler {
	return &JobHandler{jobService: jobService}
}

func (h *JobHandler) Create(c *gin.Context) {
	ctx, span := otel.Tracer("api-gateway").Start(c.Request.Context(), "create_job")
	defer span.End()

	var req struct {
		WorkspaceID    string          `json:"workspace_id" binding:"required"`
		DatasetID      string          `json:"dataset_id" binding:"required"`
		Entrypoint     string          `json:"entrypoint" binding:"required"`
		Parameters     json.RawMessage `json:"parameters"`
		Compute        json.RawMessage `json:"compute"`
		MaxRetries     int32           `json:"max_retries"`
		IdempotencyKey string          `json:"idempotency_key"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		span.RecordError(err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	span.SetAttributes(
		attribute.String("workspace_id", req.WorkspaceID),
		attribute.String("entrypoint", req.Entrypoint),
	)

	userID := c.GetString("user_id")
	job, err := h.jobService.CreateJob(
		ctx,
		req.WorkspaceID,
		userID,
		req.DatasetID,
		req.Entrypoint,
		req.Parameters,
		req.Compute,
		req.MaxRetries,
		req.IdempotencyKey,
	)
	if err != nil {
		span.RecordError(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	span.SetAttributes(attribute.String("job_id", job.ID.String()))
	metrics.JobsSubmittedTotal.Inc()
	c.JSON(http.StatusCreated, gin.H{"job": job})
}

func (h *JobHandler) Get(c *gin.Context) {
	jobID := c.Param("id")
	workspaceID := c.Query("workspace_id")
	if workspaceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "workspace_id is required"})
		return
	}

	job, err := h.jobService.GetJob(c.Request.Context(), jobID, workspaceID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "job not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"job": job})
}

func (h *JobHandler) List(c *gin.Context) {
	workspaceID := c.Query("workspace_id")
	if workspaceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "workspace_id is required"})
		return
	}

	jobs, err := h.jobService.ListJobs(c.Request.Context(), workspaceID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"jobs": jobs})
}

func (h *JobHandler) Cancel(c *gin.Context) {
	jobID := c.Param("id")
	job, err := h.jobService.CancelJob(c.Request.Context(), jobID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	metrics.JobsCancelledTotal.Inc()
	c.JSON(http.StatusOK, gin.H{"job": job})
}

func (h *JobHandler) GetProgress(c *gin.Context) {
	jobID := c.Param("id")
	workspaceID := c.Query("workspace_id")
	if workspaceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "workspace_id is required"})
		return
	}

	progress, err := h.jobService.GetJobProgress(c.Request.Context(), jobID, workspaceID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"job_id": jobID,
		"run_id": progress.ID,
		"state":  progress.State,
		"progress": gin.H{
			"total":   progress.TasksTotal,
			"pending": progress.TasksPending,
			"running": progress.TasksRunning,
			"done":    progress.TasksDone,
			"failed":  progress.TasksFailed,
			"percent": progress.PercentComplete,
		},
		"started_at":  progress.StartedAt,
		"finished_at": progress.FinishedAt,
	})
}

func (h *JobHandler) GetArtifacts(c *gin.Context) {
	jobID := c.Param("id")
	workspaceID := c.Query("workspace_id")
	if workspaceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "workspace_id is required"})
		return
	}

	artifacts, err := h.jobService.GetJobArtifacts(c.Request.Context(), jobID, workspaceID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"artifacts": artifacts})
}

func (h *JobHandler) DownloadArtifact(c *gin.Context) {
	artifactID := c.Param("artifact_id")

	url, err := h.jobService.GetArtifactDownloadURL(c.Request.Context(), artifactID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"download_url": url})
}
