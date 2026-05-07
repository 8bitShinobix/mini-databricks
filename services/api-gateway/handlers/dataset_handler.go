package handlers

import (
	"net/http"

	dbgen "github.com/8bitShinobix/mini-databricks/internal/db/generated"
	"github.com/8bitShinobix/mini-databricks/internal/service"
	"github.com/gin-gonic/gin"
)

type DatasetHandler struct {
	datasetService *service.DatasetService
}

func NewDatasetHandler(datasetService *service.DatasetService) *DatasetHandler {
	return &DatasetHandler{datasetService: datasetService}
}

func (h *DatasetHandler) Initiate(c *gin.Context) {
	var req struct {
		Name        string           `json:"name" binding:"required"`
		FileFormat  dbgen.FileFormat `json:"file_format" binding:"required"`
		WorkspaceID string           `json:"workspace_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID := c.GetString("user_id")

	dataset, uploadURL, err := h.datasetService.InitiateUpload(
		c.Request.Context(),
		req.WorkspaceID,
		userID,
		req.Name,
		req.FileFormat,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"dataset":    dataset,
		"upload_url": uploadURL,
	})
}

func (h *DatasetHandler) Complete(c *gin.Context) {
	datasetID := c.Param("id")

	var req struct {
		StoragePath string `json:"storage_path" binding:"required"`
		SizeBytes   int64  `json:"size_bytes" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	dataset, err := h.datasetService.CompleteUpload(c.Request.Context(), datasetID, req.StoragePath, req.SizeBytes)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, dataset)
}

func (h *DatasetHandler) GetByID(c *gin.Context) {
	datasetID := c.Param("id")
	workspaceID := c.Query("workspace_id")

	dataset, err := h.datasetService.GetDataset(c.Request.Context(), datasetID, workspaceID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "dataset not found"})
		return
	}

	c.JSON(http.StatusOK, dataset)
}

func (h *DatasetHandler) List(c *gin.Context) {
	workspaceID := c.Query("workspace_id")

	datasets, err := h.datasetService.ListDatasets(c.Request.Context(), workspaceID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, datasets)
}

func (h *DatasetHandler) Delete(c *gin.Context) {
	datasetID := c.Param("id")
	workspaceID := c.Query("workspace_id")

	err := h.datasetService.DeleteDataset(c.Request.Context(), datasetID, workspaceID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "dataset deleted"})
}
