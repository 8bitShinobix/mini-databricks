package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/8bitShinobix/mini-databricks/internal/config"
	"github.com/8bitShinobix/mini-databricks/internal/db"
	dbgen "github.com/8bitShinobix/mini-databricks/internal/db/generated"
	"github.com/8bitShinobix/mini-databricks/pkg/logger"
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

	slog.Info("cleanup controller connected to database")

	queries := dbgen.New(pool)

	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	// run once immediately on startup
	runCleanup(ctx, queries)

	for {
		select {
		case <-ctx.Done():
			slog.Info("cleanup controller shutting down")
			return
		case <-ticker.C:
			runCleanup(ctx, queries)
		}
	}
}

func runCleanup(ctx context.Context, queries *dbgen.Queries) {
	slog.Info("running cleanup...")

	if err := queries.CleanupStaleDatasetsInitiated(ctx); err != nil {
		slog.Error("failed to cleanup stale datasets", "error", err)
	} else {
		slog.Info("cleaned up stale initiated datasets")
	}

	if err := queries.CleanupDeadTasks(ctx); err != nil {
		slog.Error("failed to cleanup dead tasks", "error", err)
	} else {
		slog.Info("cleaned up dead tasks")
	}

	if err := queries.CleanupDeliveredOutboxEvents(ctx); err != nil {
		slog.Error("failed to cleanup outbox events", "error", err)
	} else {
		slog.Info("cleaned up delivered outbox events")
	}

	slog.Info("cleanup complete")
}
