package routes

import (
	"github.com/8bitShinobix/mini-databricks/internal/middleware"
	"github.com/8bitShinobix/mini-databricks/services/api-gateway/handlers"
	"github.com/gin-gonic/gin"
)

func Setup(r *gin.Engine, userHandler *handlers.UserHandler, workspaceHandler *handlers.WorkspaceHandler, datasetHandler *handlers.DatasetHandler, jobHandler *handlers.JobHandler, jwtSecret string) {
	v1 := r.Group("/api/v1")
	{
		auth := v1.Group("/auth")
		{
			auth.POST("/register", userHandler.Register)
			auth.POST("/login", userHandler.Login)
		}
		protected := v1.Group("/")
		protected.Use(middleware.AuthMiddleware(jwtSecret))
		{
			protected.GET("/me", userHandler.Me)
			protected.POST("/workspaces", workspaceHandler.Create)
			protected.GET("/workspaces", workspaceHandler.ListMyWorkspaces)
			protected.GET("/workspaces/:id", workspaceHandler.GetByID)
			protected.PATCH("/workspaces/:id", workspaceHandler.Update)
			protected.DELETE("/workspaces/:id", workspaceHandler.Delete)
			protected.POST("/datasets/initiate", datasetHandler.Initiate)
			protected.POST("/datasets/:id/complete", datasetHandler.Complete)
			protected.GET("/datasets/:id", datasetHandler.GetByID)
			protected.GET("/datasets", datasetHandler.List)
			protected.DELETE("/datasets/:id", datasetHandler.Delete)
			protected.POST("/jobs", jobHandler.Create)
			protected.GET("/jobs", jobHandler.List)
			protected.GET("/jobs/:id", jobHandler.Get)
			protected.POST("/jobs/:id/cancel", jobHandler.Cancel)
			protected.GET("/jobs/:id/progress", jobHandler.GetProgress)
			protected.GET("/jobs/:id/artifacts", jobHandler.GetArtifacts)
			protected.GET("/jobs/:id/artifacts/:artifact_id/download", jobHandler.DownloadArtifact)
		}

	}
}
