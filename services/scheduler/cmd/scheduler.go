package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	dbgen "github.com/8bitShinobix/mini-databricks/internal/db/generated"
	"github.com/8bitShinobix/mini-databricks/internal/k8s"
	"github.com/8bitShinobix/mini-databricks/internal/metrics"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

type Scheduler struct {
	queries       *dbgen.Queries
	k8sController *k8s.Controller
}

func NewScheduler(queries *dbgen.Queries, k8sController *k8s.Controller) *Scheduler {
	return &Scheduler{queries: queries, k8sController: k8sController}
}

type JobSubmittedEvent struct {
	JobID       string `json:"job_id"`
	WorkspaceID string `json:"workspace_id"`
}

func (s *Scheduler) HandleJobSubmitted(ctx context.Context, payload []byte) error {
	// --- span starts here ---
	ctx, span := otel.Tracer("scheduler").Start(ctx, "handle_job_submitted")
	defer span.End()

	var event JobSubmittedEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to parse event: %w", err)
	}

	span.SetAttributes(
		attribute.String("job_id", event.JobID),
		attribute.String("workspace_id", event.WorkspaceID),
	)

	slog.Info("scheduling job", "job_id", event.JobID)

	jobUUID, err := uuid.Parse(event.JobID)
	if err != nil {
		span.RecordError(err)
		return fmt.Errorf("invalid job id: %w", err)
	}

	workspaceUUID, err := uuid.Parse(event.WorkspaceID)
	if err != nil {
		span.RecordError(err)
		return fmt.Errorf("invalid workspace id: %w", err)
	}

	// child span — DB: update job state
	_, updateSpan := otel.Tracer("scheduler").Start(ctx, "db_update_job_state")
	job, err := s.queries.UpdateJobState(ctx, dbgen.UpdateJobStateParams{
		ID:    jobUUID,
		State: dbgen.JobStateQUEUED,
	})
	updateSpan.End()
	if err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to update job state: %w", err)
	}

	// child span — DB: create run
	_, runSpan := otel.Tracer("scheduler").Start(ctx, "db_create_run")
	run, err := s.queries.CreateRun(ctx, dbgen.CreateRunParams{
		JobID:       jobUUID,
		WorkspaceID: workspaceUUID,
	})
	runSpan.End()
	if err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to create run: %w", err)
	}

	metrics.RunsCreatedTotal.Inc()

	// child span — DB: create tasks
	_, tasksSpan := otel.Tracer("scheduler").Start(ctx, "db_create_tasks")
	numPartitions := int32(3)
	for i := int32(0); i < numPartitions; i++ {
		_, err := s.queries.CreateTask(ctx, dbgen.CreateTaskParams{
			RunID:          run.ID,
			WorkspaceID:    workspaceUUID,
			PartitionIndex: i,
		})
		if err != nil {
			tasksSpan.RecordError(err)
			tasksSpan.End()
			return fmt.Errorf("failed to create task %d: %w", i, err)
		}
	}
	tasksSpan.End()

	metrics.TasksCreatedTotal.Add(float64(numPartitions))

	if _, err := s.queries.UpdateRunTasksTotal(ctx, dbgen.UpdateRunTasksTotalParams{
		ID:         run.ID,
		TasksTotal: numPartitions,
	}); err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to set tasks total: %w", err)
	}

	if _, err := s.queries.InitRunProgress(ctx, run.ID); err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to init run progress: %w", err)
	}

	if s.k8sController != nil {
		if err := s.k8sController.EnsureNamespace(ctx, event.WorkspaceID); err != nil {
			slog.Warn("failed to ensure namespace", "error", err)
		}
		if err := s.k8sController.CreateWorkerJob(ctx, event.JobID, event.WorkspaceID, run.ID.String(), numPartitions); err != nil {
			slog.Warn("failed to create k8s job", "error", err)
		}
	}

	span.SetAttributes(attribute.String("run_id", run.ID.String()))
	slog.Info("job queued", "job_id", job.ID, "run_id", run.ID, "tasks", numPartitions)
	return nil
}
