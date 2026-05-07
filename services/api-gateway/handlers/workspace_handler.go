package handlers

import (
	"net/http"

	dbgen "github.com/8bitShinobix/mini-databricks/internal/db/generated"
	"github.com/8bitShinobix/mini-databricks/internal/service"
	"github.com/gin-gonic/gin"
)

type WorkspaceHandler struct {
	workspaceService *service.WorkspaceService
}

func NewWorkspaceHandler(workspaceService *service.WorkspaceService) *WorkspaceHandler {
	return &WorkspaceHandler{workspaceService: workspaceService}
}

func (h *WorkspaceHandler) Create(c *gin.Context) {
	var req struct {
		Name string              `json:"name" binding:"required"`
		Plan dbgen.WorkspacePlan `json:"plan" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ownerID := c.GetString("user_id")

	workspace, err := h.workspaceService.CreateWorkspace(c.Request.Context(), req.Name, ownerID, req.Plan)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, workspace)
}

func (h *WorkspaceHandler) GetByID(c *gin.Context) {
	workspaceID := c.Param("id")

	workspace, err := h.workspaceService.GetWorkspaceByID(c.Request.Context(), workspaceID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "workspace not found"})
		return
	}

	c.JSON(http.StatusOK, workspace)
}

func (h *WorkspaceHandler) ListMyWorkspaces(c *gin.Context) {
	ownerID := c.GetString("user_id")

	workspaces, err := h.workspaceService.GetWorkspacesByOwner(c.Request.Context(), ownerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, workspaces)
}

func (h *WorkspaceHandler) Update(c *gin.Context) {
	workspaceID := c.Param("id")
	ownerID := c.GetString("user_id")

	var req struct {
		Name *string              `json:"name"`
		Plan *dbgen.WorkspacePlan `json:"plan"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	workspace, err := h.workspaceService.UpdateWorkspace(c.Request.Context(), workspaceID, ownerID, req.Name, req.Plan)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, workspace)
}

func (h *WorkspaceHandler) Delete(c *gin.Context) {
	workspaceID := c.Param("id")
	ownerID := c.GetString("user_id")

	err := h.workspaceService.DeleteWorkspace(c.Request.Context(), workspaceID, ownerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "workspace deleted"})
}
