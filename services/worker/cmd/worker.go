package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/8bitShinobix/mini-databricks/internal/circuitbreaker"
	dbgen "github.com/8bitShinobix/mini-databricks/internal/db/generated"
	joberrors "github.com/8bitShinobix/mini-databricks/internal/errors"
	"github.com/8bitShinobix/mini-databricks/internal/kafka"
	"github.com/8bitShinobix/mini-databricks/internal/metrics"
	"github.com/8bitShinobix/mini-databricks/internal/storage"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/sony/gobreaker"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

type Worker struct {
	queries        *dbgen.Queries
	workerID       string
	kafkaProducer  *kafka.Producer
	minioClient    *storage.MinioClient
	minioEndpoint  string
	minioAccessKey string
	minioSecretKey string
	minioBucket    string
	pythonPath     string
	taskRunnerPath string
	taskTimeout    time.Duration
	dbBreaker      *gobreaker.CircuitBreaker
	minioBreaker   *gobreaker.CircuitBreaker
}

func NewWorker(
	queries *dbgen.Queries,
	workerID string,
	kafkaProducer *kafka.Producer,
	minioClient *storage.MinioClient,
	minioEndpoint string,
	minioAccessKey string,
	minioSecretKey string,
	minioBucket string,
	pythonPath string,
	taskRunnerPath string,
	taskTimeout time.Duration,
) *Worker {
	return &Worker{
		queries:        queries,
		workerID:       workerID,
		kafkaProducer:  kafkaProducer,
		minioClient:    minioClient,
		minioEndpoint:  minioEndpoint,
		minioAccessKey: minioAccessKey,
		minioSecretKey: minioSecretKey,
		minioBucket:    minioBucket,
		pythonPath:     pythonPath,
		taskRunnerPath: taskRunnerPath,
		taskTimeout:    taskTimeout,
		dbBreaker:      circuitbreaker.New("postgres"),
		minioBreaker:   circuitbreaker.New("minio"),
	}
}

func (w *Worker) Start(ctx context.Context) {
	slog.Info("worker polling for tasks", "worker_id", w.workerID)
	for {
		select {
		case <-ctx.Done():
			slog.Info("worker shutting down")
			return
		default:
			if err := w.pollAndProcess(ctx); err != nil {
				slog.Error("error processing task", "error", err)
			}
			time.Sleep(2 * time.Second)
		}
	}
}

func (w *Worker) pollAndProcess(ctx context.Context) error {
	var task dbgen.Task
	_, err := w.dbBreaker.Execute(func() (any, error) {
		t, err := w.queries.LeaseTask(ctx, pgtype.Text{
			String: w.workerID,
			Valid:  true,
		})
		if err != nil {
			if strings.Contains(err.Error(), "no rows") {
				return nil, nil // not a failure
			}
			return nil, err // real DB error
		}
		task = t
		return nil, nil
	})

	if err != nil {
		if strings.Contains(err.Error(), "circuit breaker is open") {
			slog.Debug("db circuit open, skipping poll")
		}
		return nil
	}

	// no task was assigned
	if task.RunID == uuid.Nil {
		return nil
	}

	slog.Info("leased task", "worker_id", w.workerID, "task_id", task.ID)

	runState, err := w.queries.GetRunState(ctx, task.RunID)
	if err != nil {
		slog.Error("failed to check run state", "error", err)
		return err
	}
	if runState == dbgen.RunStateCANCELLED {
		slog.Info("run cancelled before processing, marking DEAD", "task_id", task.ID)
		if _, err := w.queries.UpdateTaskState(ctx, dbgen.UpdateTaskStateParams{
			ID:    task.ID,
			State: dbgen.TaskStateDEAD,
		}); err != nil {
			slog.Error("failed to mark task DEAD", "task_id", task.ID, "error", err)
		}
		metrics.TasksProcessedTotal.WithLabelValues("dead").Inc()
		return nil
	}

	if _, err := w.queries.TaskStarted(ctx, task.RunID); err != nil {
		slog.Warn("failed to update run progress (started)", "error", err)
	}

	heartbeatCtx, stopHeartbeat := context.WithCancel(ctx)
	defer stopHeartbeat()

	go w.heartbeat(heartbeatCtx, task.ID.String())

	processCtx, cancelProcess := context.WithTimeout(ctx, w.taskTimeout)
	defer cancelProcess()

	taskStart := time.Now()

	if err := w.process(processCtx, task); err != nil {
		slog.Error("task failed", "task_id", task.ID, "error", err)

		if processCtx.Err() == context.DeadlineExceeded {
			slog.Warn("task timed out", "task_id", task.ID, "timeout", w.taskTimeout)
		}

		job, jobErr := w.queries.GetJobByRunID(ctx, task.RunID)

		canRetry := jobErr == nil &&
			task.RetryCount < job.MaxRetries &&
			joberrors.IsTransient(err)

		if canRetry {
			slog.Info("retrying task",
				"task_id", task.ID,
				"retry_count", task.RetryCount+1,
				"max_retries", job.MaxRetries,
			)
			if _, retryErr := w.queries.RetryTask(ctx, dbgen.RetryTaskParams{
				ID:           task.ID,
				ErrorMessage: pgtype.Text{String: err.Error(), Valid: true},
			}); retryErr != nil {
				slog.Error("failed to reset task for retry", "error", retryErr)
			}
			return nil
		}

		slog.Error("task permanently failed",
			"task_id", task.ID,
			"retry_count", task.RetryCount,
			"max_retries", job.MaxRetries,
		)
		if _, updateErr := w.queries.FailTaskWithError(ctx, dbgen.FailTaskWithErrorParams{
			ID:           task.ID,
			ErrorMessage: pgtype.Text{String: err.Error(), Valid: true},
		}); updateErr != nil {
			slog.Error("failed to mark task as failed", "error", updateErr)
		}
		if _, err := w.queries.TaskFailed(ctx, task.RunID); err != nil {
			slog.Error("failed to update run progress", "error", err)
		}
		metrics.TasksProcessedTotal.WithLabelValues("failed").Inc()
		metrics.TaskDurationSeconds.Observe(time.Since(taskStart).Seconds())
		return err
	}

	runState, err = w.queries.GetRunState(ctx, task.RunID)
	if err != nil {
		slog.Error("failed to check run state after processing", "error", err)
		return err
	}
	if runState == dbgen.RunStateCANCELLED {
		slog.Info("run cancelled during processing, marking DEAD", "task_id", task.ID)
		if _, err := w.queries.UpdateTaskState(ctx, dbgen.UpdateTaskStateParams{
			ID:    task.ID,
			State: dbgen.TaskStateDEAD,
		}); err != nil {
			slog.Error("failed to mark task DEAD", "task_id", task.ID, "error", err)
		}
		metrics.TasksProcessedTotal.WithLabelValues("dead").Inc()
		metrics.TaskDurationSeconds.Observe(time.Since(taskStart).Seconds())
		return nil
	}

	if _, updateErr := w.queries.UpdateTaskState(ctx, dbgen.UpdateTaskStateParams{
		ID:    task.ID,
		State: dbgen.TaskStateSUCCEEDED,
	}); updateErr != nil {
		slog.Error("failed to update task state to SUCCEEDED", "error", updateErr)
		return updateErr
	}

	if _, err := w.queries.TaskSucceeded(ctx, task.RunID); err != nil {
		slog.Error("failed to update run progress (succeeded)", "error", err)
	}

	slog.Info("checking run completion", "task_id", task.ID, "run_id", task.RunID)

	if err := w.checkAndCompleteRun(ctx, task); err != nil {
		slog.Error("error checking run completion", "error", err)
	}

	metrics.TasksProcessedTotal.WithLabelValues("succeeded").Inc()
	metrics.TaskDurationSeconds.Observe(time.Since(taskStart).Seconds())
	slog.Info("task completed successfully", "task_id", task.ID)
	return nil
}

func (w *Worker) process(ctx context.Context, task dbgen.Task) error {
	ctx, span := otel.Tracer("worker").Start(ctx, "process_task")
	defer span.End()

	span.SetAttributes(
		attribute.String("task_id", task.ID.String()),
		attribute.Int("partition_index", int(task.PartitionIndex)),
	)

	slog.Info("processing task", "task_id", task.ID, "partition", task.PartitionIndex)

	job, err := w.queries.GetJobByRunID(ctx, task.RunID)
	if err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to get job: %w", err)
	}

	dataset, err := w.queries.GetDatasetByJobID(ctx, job.ID)
	if err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to get dataset: %w", err)
	}

	totalTasks, err := w.queries.GetRunTaskCount(ctx, task.RunID)
	if err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to get task count: %w", err)
	}

	env := append(os.Environ(),
		fmt.Sprintf("TASK_ID=%s", task.ID),
		fmt.Sprintf("RUN_ID=%s", task.RunID),
		fmt.Sprintf("DATASET_STORAGE_PATH=%s", dataset.StoragePath.String),
		fmt.Sprintf("PARTITION_INDEX=%d", task.PartitionIndex),
		fmt.Sprintf("TOTAL_PARTITIONS=%d", totalTasks),
		fmt.Sprintf("JOB_ENTRYPOINT=%s", job.Entrypoint),
		fmt.Sprintf("JOB_PARAMETERS=%s", string(job.Parameters)),
		fmt.Sprintf("MINIO_ENDPOINT=%s", w.minioEndpoint),
		fmt.Sprintf("MINIO_ACCESS_KEY=%s", w.minioAccessKey),
		fmt.Sprintf("MINIO_SECRET_KEY=%s", w.minioSecretKey),
		fmt.Sprintf("MINIO_BUCKET=%s", w.minioBucket),
	)

	cmd := exec.CommandContext(ctx, w.pythonPath, w.taskRunnerPath)
	cmd.Env = env
	cmd.Stderr = os.Stderr

	// span for just the Python subprocess
	_, subSpan := otel.Tracer("worker").Start(ctx, "python_subprocess")
	start := time.Now()
	output, err := cmd.Output()
	metrics.PythonSubprocessDurationSeconds.Observe(time.Since(start).Seconds())
	subSpan.End()

	if err != nil {
		span.RecordError(err)
		return fmt.Errorf("task runner failed: %w", err)
	}

	rawOutput := strings.TrimSpace(string(output))
	parts := strings.SplitN(rawOutput, ":", 2)
	outputPath := parts[0]
	var sizeBytes int64
	if len(parts) == 2 {
		fmt.Sscanf(parts[1], "%d", &sizeBytes)
	}

	span.SetAttributes(
		attribute.String("output_path", outputPath),
		attribute.Int64("size_bytes", sizeBytes),
	)

	slog.Info("task runner completed", "task_id", task.ID, "output_path", outputPath, "size_bytes", sizeBytes)

	_, err = w.queries.CreateArtifact(ctx, dbgen.CreateArtifactParams{
		RunID:       task.RunID,
		WorkspaceID: task.WorkspaceID,
		Name:        fmt.Sprintf("partition-%d.csv", task.PartitionIndex),
		StoragePath: outputPath,
		ContentType: "text/csv",
		SizeBytes:   sizeBytes,
	})
	if err != nil {
		slog.Warn("failed to register artifact", "task_id", task.ID, "error", err)
	}

	slog.Info("task partition done", "task_id", task.ID, "partition", task.PartitionIndex)
	return nil
}

func (w *Worker) heartbeat(ctx context.Context, taskID string) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	taskUUID, err := uuid.Parse(taskID)
	if err != nil {
		slog.Error("invalid task id for heartbeat", "task_id", taskID, "error", err)
		return
	}

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			_, err := w.queries.HeartbeatTask(ctx, dbgen.HeartbeatTaskParams{
				ID: taskUUID,
				LeaseOwner: pgtype.Text{
					String: w.workerID,
					Valid:  true,
				},
			})
			if err != nil {
				slog.Error("heartbeat failed", "task_id", taskID, "error", err)
				return
			}
			slog.Debug("heartbeat sent", "task_id", taskID)
		}
	}
}

func (w *Worker) checkAndCompleteRun(ctx context.Context, task dbgen.Task) error {
	progress, err := w.queries.GetRunProgress(ctx, task.RunID)
	if err != nil {
		return fmt.Errorf("failed to get run progress: %w", err)
	}

	slog.Info("run progress",
		"run_id", task.RunID,
		"done", progress.TasksDone,
		"total", progress.TasksTotal,
		"failed", progress.TasksFailed,
	)

	allDone := progress.TasksDone+progress.TasksFailed == progress.TasksTotal
	if allDone {
		newRunState := dbgen.RunStateSUCCEEDED
		newJobState := dbgen.JobStateSUCCEEDED
		if progress.TasksFailed > 0 {
			newRunState = dbgen.RunStateFAILED
			newJobState = dbgen.JobStateFAILED
		}

		_, err = w.queries.CompleteRun(ctx, dbgen.CompleteRunParams{
			ID:    task.RunID,
			State: newRunState,
		})
		if err != nil {
			return fmt.Errorf("failed to complete run: %w", err)
		}

		job, err := w.queries.GetJobByRunID(ctx, task.RunID)
		if err != nil {
			return fmt.Errorf("failed to get job: %w", err)
		}

		_, err = w.queries.CompleteJob(ctx, dbgen.CompleteJobParams{
			ID:    job.ID,
			State: newJobState,
		})
		if err != nil {
			return fmt.Errorf("failed to complete job: %w", err)
		}

		if err := w.kafkaProducer.Publish(ctx, "job.completed", job.ID.String(), map[string]string{
			"job_id":       job.ID.String(),
			"workspace_id": job.WorkspaceID.String(),
		}); err != nil {
			slog.Error("failed to publish job.completed event", "error", err)
		}

		slog.Info("job finished", "job_id", job.ID, "state", newJobState)
	}

	return nil
}
