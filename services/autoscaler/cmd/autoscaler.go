package main

import (
	"context"
	"log/slog"
	"time"

	dbgen "github.com/8bitShinobix/mini-databricks/internal/db/generated"
	"github.com/8bitShinobix/mini-databricks/internal/k8s"
	"github.com/8bitShinobix/mini-databricks/internal/metrics"
)

type AutoscalerConfig struct {
	MinWorkers          int32
	MaxWorkers          int32
	ScaleUpThreshold    int64
	ScaleDownThreshold  int64
	CooldownSeconds     int
	PollIntervalSeconds int
}

type Autoscaler struct {
	queries       *dbgen.Queries
	k8sController *k8s.Controller
	config        AutoscalerConfig
	lastScaleAt   time.Time
}

func NewAutoscaler(queries *dbgen.Queries, k8sController *k8s.Controller, cfg AutoscalerConfig) *Autoscaler {
	return &Autoscaler{
		queries:       queries,
		k8sController: k8sController,
		config:        cfg,
	}
}

func (a *Autoscaler) Start(ctx context.Context) {
	ticker := time.NewTicker(time.Duration(a.config.PollIntervalSeconds) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.Info("autoscaler shutting down")
			return
		case <-ticker.C:
			if err := a.evaluate(ctx); err != nil {
				slog.Error("autoscaler evaluation error", "error", err)
			}
		}
	}
}

func (a *Autoscaler) evaluate(ctx context.Context) error {
	pendingCount, err := a.queries.GetPendingTaskCount(ctx)
	if err != nil {
		return err
	}

	runningCount, err := a.queries.GetRunningTaskCount(ctx)
	if err != nil {
		return err
	}

	metrics.PendingTasksCurrent.Set(float64(pendingCount))
	metrics.RunningTasksCurrent.Set(float64(runningCount))

	slog.Debug("autoscaler tick", "pending", pendingCount, "running", runningCount)

	cooldown := time.Duration(a.config.CooldownSeconds) * time.Second
	if time.Since(a.lastScaleAt) < cooldown {
		slog.Debug("autoscaler in cooldown, skipping")
		return nil
	}

	if a.k8sController == nil {
		a.logScalingDecision(pendingCount)
		return nil
	}

	currentReplicas, err := a.k8sController.GetWorkerReplicas(ctx)
	if err != nil {
		a.logScalingDecision(pendingCount)
		return nil
	}

	metrics.WorkerReplicasCurrent.Set(float64(currentReplicas))
	slog.Debug("autoscaler replicas", "current", currentReplicas)

	if pendingCount > a.config.ScaleUpThreshold && currentReplicas < a.config.MaxWorkers {
		newReplicas := currentReplicas + 1
		slog.Info("autoscaler scaling UP",
			"from", currentReplicas,
			"to", newReplicas,
			"pending", pendingCount,
		)
		if err := a.k8sController.SetWorkerReplicas(ctx, newReplicas); err != nil {
			return err
		}
		metrics.WorkerReplicasCurrent.Set(float64(newReplicas))
		a.lastScaleAt = time.Now()
		return nil
	}

	if pendingCount <= a.config.ScaleDownThreshold &&
		runningCount == 0 &&
		currentReplicas > a.config.MinWorkers {
		newReplicas := currentReplicas - 1
		slog.Info("autoscaler scaling DOWN",
			"from", currentReplicas,
			"to", newReplicas,
		)
		if err := a.k8sController.SetWorkerReplicas(ctx, newReplicas); err != nil {
			return err
		}
		metrics.WorkerReplicasCurrent.Set(float64(newReplicas))
		a.lastScaleAt = time.Now()
		return nil
	}

	return nil
}

func (a *Autoscaler) logScalingDecision(pendingCount int64) {
	if pendingCount > a.config.ScaleUpThreshold {
		slog.Info("autoscaler would scale UP",
			"pending", pendingCount,
			"threshold", a.config.ScaleUpThreshold,
		)
	} else if pendingCount <= a.config.ScaleDownThreshold {
		slog.Debug("autoscaler would scale DOWN", "pending", pendingCount)
	}
}
