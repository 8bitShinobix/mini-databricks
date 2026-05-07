package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/8bitShinobix/mini-databricks/internal/config"
	"github.com/8bitShinobix/mini-databricks/internal/db"
	dbgen "github.com/8bitShinobix/mini-databricks/internal/db/generated"
	"github.com/8bitShinobix/mini-databricks/internal/kafka"
	"github.com/8bitShinobix/mini-databricks/internal/storage"
	"github.com/8bitShinobix/mini-databricks/internal/telemetry"
	"github.com/8bitShinobix/mini-databricks/pkg/logger"
	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	cfg := config.Load()
	logger.Init(cfg.Env)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	shutdown := telemetry.Init(ctx, "worker")
	defer shutdown()

	pool, err := db.Connect(ctx, cfg.DBUrl)
	if err != nil {
		slog.Error("could not connect to database", "error", err)
		os.Exit(1)
	}
	defer pool.Close()
	slog.Info("worker connected to database")

	kafkaProducer := kafka.NewProducer(cfg.KafkaBrokers)
	defer kafkaProducer.Close()

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
	slog.Info("worker connected to minio")

	queries := dbgen.New(pool)
	workerID := uuid.New().String()
	taskTimeout := time.Duration(cfg.TaskTimeoutSeconds) * time.Second

	// start metrics server
	go func() {
		http.Handle("/metrics", promhttp.Handler())
		http.ListenAndServe(":9095", nil)
	}()

	slog.Info("worker starting", "worker_id", workerID, "task_timeout", taskTimeout)

	worker := NewWorker(
		queries,
		workerID,
		kafkaProducer,
		minioClient,
		cfg.MinioEndpoint,
		cfg.MinioAccessKey,
		cfg.MinioSecretKey,
		cfg.MinioBucket,
		cfg.PythonPath,
		cfg.TaskRunnerPath,
		taskTimeout,
	)
	worker.Start(ctx)
}
