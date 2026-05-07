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
	"github.com/8bitShinobix/mini-databricks/pkg/logger"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	cfg := config.Load()
	logger.Init(cfg.Env)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	pool, err := db.Connect(ctx, cfg.DBUrl)
	if err != nil {
		slog.Error("could not connect to database", "error", err)
		os.Exit(1)
	}
	defer pool.Close()
	slog.Info("autoscaler connected to database")

	k8sController, err := k8s.NewController()
	if err != nil {
		slog.Warn("k8s controller not available", "error", err)
	}

	queries := dbgen.New(pool)

	// start metrics server
	go func() {
		http.Handle("/metrics", promhttp.Handler())
		http.ListenAndServe(":9093", nil)
	}()

	autoscaler := NewAutoscaler(queries, k8sController, AutoscalerConfig{
		MinWorkers:          1,
		MaxWorkers:          10,
		ScaleUpThreshold:    3,
		ScaleDownThreshold:  0,
		CooldownSeconds:     30,
		PollIntervalSeconds: 10,
	})

	slog.Info("autoscaler starting")
	autoscaler.Start(ctx)
}
