package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/8bitShinobix/mini-databricks/internal/config"
	"github.com/8bitShinobix/mini-databricks/internal/db"
	dbgen "github.com/8bitShinobix/mini-databricks/internal/db/generated"
	"github.com/8bitShinobix/mini-databricks/internal/k8s"
	"github.com/8bitShinobix/mini-databricks/internal/kafka"
	"github.com/8bitShinobix/mini-databricks/internal/telemetry"
	"github.com/8bitShinobix/mini-databricks/pkg/logger"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	cfg := config.Load()
	logger.Init(cfg.Env)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	shutdown := telemetry.Init(ctx, "scheduler")
	defer shutdown()

	pool, err := db.Connect(ctx, cfg.DBUrl)
	if err != nil {
		slog.Error("could not connect to database", "error", err)
		os.Exit(1)
	}
	defer pool.Close()
	slog.Info("scheduler connected to database")

	k8sController, err := k8s.NewController()
	if err != nil {
		slog.Warn("k8s controller not available", "error", err)
	}

	queries := dbgen.New(pool)

	consumer := kafka.NewConsumer(cfg.KafkaBrokers, "job.submitted", "scheduler-group")
	defer consumer.Close()

	if err := kafka.CreateTopic(cfg.KafkaBrokers, "job.submitted.deadletter"); err != nil {
		slog.Warn("topic creation warning", "error", err)
	}

	// start metrics server
	go func() {
		http.Handle("/metrics", promhttp.Handler())
		http.ListenAndServe(":9091", nil)
	}()

	slog.Info("scheduler listening for jobs...")

	scheduler := NewScheduler(queries, k8sController)
	consumer.Consume(ctx, scheduler.HandleJobSubmitted)
}
