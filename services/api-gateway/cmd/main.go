package main

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"net/http"

	"github.com/8bitShinobix/mini-databricks/internal/config"
	"github.com/8bitShinobix/mini-databricks/internal/db"
	dbgen "github.com/8bitShinobix/mini-databricks/internal/db/generated"
	"github.com/8bitShinobix/mini-databricks/internal/kafka"
	"github.com/8bitShinobix/mini-databricks/internal/ratelimit"
	"github.com/8bitShinobix/mini-databricks/internal/service"
	"github.com/8bitShinobix/mini-databricks/internal/storage"
	"github.com/8bitShinobix/mini-databricks/internal/telemetry"
	"github.com/8bitShinobix/mini-databricks/pkg/logger"
	"github.com/8bitShinobix/mini-databricks/services/api-gateway/handlers"
	"github.com/8bitShinobix/mini-databricks/services/api-gateway/routes"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
)

func main() {
	cfg := config.Load()
	logger.Init(cfg.Env)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	shutdown := telemetry.Init(ctx, "api-gateway")
	defer shutdown()

	pool, err := db.Connect(ctx, cfg.DBUrl)
	if err != nil {
		slog.Error("could not connect to database", "error", err)
		os.Exit(1)
	}
	defer pool.Close()
	slog.Info("connected to database")

	kafkaProducer := kafka.NewProducer(cfg.KafkaBrokers)
	defer kafkaProducer.Close()

	if err := kafka.CreateTopic(cfg.KafkaBrokers, "job.submitted"); err != nil {
		slog.Warn("topic creation warning", "error", err)
	}
	if err := kafka.CreateTopic(cfg.KafkaBrokers, "job.completed"); err != nil {
		slog.Warn("topic creation warning", "error", err)
	}

	minioClient, err := storage.NewMinioClient(
		cfg.MinioEndpoint,
		cfg.MinioAccessKey,
		cfg.MinioSecretKey,
		cfg.MinioBucket,
	)
	if err != nil {
		slog.Error("could not connect to minio", "error", err)
		os.Exit(1)
	}
	if err := minioClient.EnsureBucketExists(ctx); err != nil {
		slog.Error("could not ensure bucket exists", "error", err)
		os.Exit(1)
	}
	slog.Info("connected to minio")

	rateLimiter, err := ratelimit.NewRateLimiter(cfg.RedisURL, 10)
	if err != nil {
		slog.Error("could not connect to redis", "error", err)
		os.Exit(1)
	}
	slog.Info("connected to redis")

	queries := dbgen.New(pool)

	dispatcher := kafka.NewDispatcher(queries, kafkaProducer)
	go dispatcher.Start(ctx)

	go func() {
		consumer := kafka.NewConsumer(cfg.KafkaBrokers, "job.completed", "api-gateway-group")
		defer consumer.Close()
		consumer.Consume(ctx, func(ctx context.Context, payload []byte) error {
			var event struct {
				JobID       string `json:"job_id"`
				WorkspaceID string `json:"workspace_id"`
			}
			if err := json.Unmarshal(payload, &event); err != nil {
				return err
			}
			slog.Info("job completed, decrementing rate limit", "job_id", event.JobID)
			return rateLimiter.DecrementRunningJobs(ctx, event.WorkspaceID)
		})
	}()

	userService := service.NewUserService(queries)
	userHandler := handlers.NewUserHandler(userService, cfg.JWTSecret)

	workspaceService := service.NewWorkspaceService(queries)
	workspaceHandler := handlers.NewWorkspaceHandler(workspaceService)

	datasetService := service.NewDatasetService(queries, minioClient)
	datasetHandler := handlers.NewDatasetHandler(datasetService)

	jobService := service.NewJobService(queries, pool, rateLimiter, minioClient)
	jobHandler := handlers.NewJobHandler(jobService)

	r := gin.Default()
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:5173"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
	}))
	r.Use(otelgin.Middleware("api-gateway"))
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})
	r.GET("/ready", func(c *gin.Context) {
		if err := pool.Ping(ctx); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"status": "not ready",
				"error":  "database unreachable",
			})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ready"})
	})

	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

	r.GET("/admin/stats", func(c *gin.Context) {
		pending, _ := queries.GetPendingTaskCount(ctx)
		running, _ := queries.GetRunningTaskCount(ctx)
		c.JSON(http.StatusOK, gin.H{
			"pending_tasks": pending,
			"running_tasks": running,
		})
	})

	routes.Setup(r, userHandler, workspaceHandler, datasetHandler, jobHandler, cfg.JWTSecret)

	slog.Info("api gateway starting", "port", cfg.APIPort)
	r.Run(":" + cfg.APIPort)
}
