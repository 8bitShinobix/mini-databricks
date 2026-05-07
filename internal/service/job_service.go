package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	dbgen "github.com/8bitShinobix/mini-databricks/internal/db/generated"
	"github.com/8bitShinobix/mini-databricks/internal/ratelimit"
	"github.com/8bitShinobix/mini-databricks/internal/storage"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
)

type JobService struct {
	queries     *dbgen.Queries
	pool        *pgxpool.Pool
	rateLimiter *ratelimit.RateLimiter
	storage     *storage.MinioClient
}

func NewJobService(queries *dbgen.Queries, pool *pgxpool.Pool, rateLimiter *ratelimit.RateLimiter, storage *storage.MinioClient) *JobService {
	return &JobService{queries: queries, pool: pool, rateLimiter: rateLimiter, storage: storage}
}

func (s *JobService) CreateJob(
	ctx context.Context,
	workspaceID string,
	createdBy string,
	datasetID string,
	entrypoint string,
	parameters []byte,
	compute []byte,
	maxRetries int32,
	idempotencyKey string,
) (dbgen.Job, error) {
	allowed, err := s.rateLimiter.AllowJobSubmission(ctx, workspaceID)
	if err != nil {
		return dbgen.Job{}, fmt.Errorf("rate limit check failed: %w", err)
	}
	if !allowed {
		return dbgen.Job{}, fmt.Errorf("rate limit exceeded: max 10 concurrent jobs per workspace")
	}

	workspaceUUID, err := uuid.Parse(workspaceID)
	if err != nil {
		return dbgen.Job{}, fmt.Errorf("invalid workspace id: %w", err)
	}

	createdByUUID, err := uuid.Parse(createdBy)
	if err != nil {
		return dbgen.Job{}, fmt.Errorf("invalid user id: %w", err)
	}

	datasetUUID, err := uuid.Parse(datasetID)
	if err != nil {
		return dbgen.Job{}, fmt.Errorf("invalid dataset id: %w", err)
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return dbgen.Job{}, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	qtx := s.queries.WithTx(tx)

	job, err := qtx.CreateJob(ctx, dbgen.CreateJobParams{
		WorkspaceID: workspaceUUID,
		CreatedBy:   createdByUUID,
		DatasetID:   datasetUUID,
		Entrypoint:  entrypoint,
		Parameters:  parameters,
		Compute:     compute,
		MaxRetries:  maxRetries,
		IdempotencyKey: pgtype.Text{
			String: idempotencyKey,
			Valid:  idempotencyKey != "",
		},
	})
	if err != nil {
		return dbgen.Job{}, fmt.Errorf("failed to create job: %w", err)
	}

	carrier := propagation.MapCarrier{}
	otel.GetTextMapPropagator().Inject(ctx, carrier)

	payload, err := json.Marshal(map[string]string{
		"job_id":       job.ID.String(),
		"workspace_id": workspaceID,
		"traceparent":  carrier["traceparent"],
	})

	if err != nil {
		return dbgen.Job{}, fmt.Errorf("failed to marshal payload: %w", err)
	}

	_, err = qtx.CreateOutboxEvent(ctx, dbgen.CreateOutboxEventParams{
		EventType:   "job.submitted",
		AggregateID: job.ID,
		Payload:     payload,
	})
	if err != nil {
		return dbgen.Job{}, fmt.Errorf("failed to create outbox event: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return dbgen.Job{}, fmt.Errorf("failed to commit transaction: %w", err)
	}

	if err := s.rateLimiter.IncrementRunningJobs(ctx, workspaceID); err != nil {
		return dbgen.Job{}, fmt.Errorf("failed to increment rate limit: %w", err)
	}

	return job, nil
}

func (s *JobService) GetJob(ctx context.Context, jobID, workspaceID string) (dbgen.Job, error) {
	jobUUID, err := uuid.Parse(jobID)
	if err != nil {
		return dbgen.Job{}, fmt.Errorf("invalid job id: %w", err)
	}

	workspaceUUID, err := uuid.Parse(workspaceID)
	if err != nil {
		return dbgen.Job{}, fmt.Errorf("invalid workspace id: %w", err)
	}

	return s.queries.GetJobByID(ctx, dbgen.GetJobByIDParams{
		ID:          jobUUID,
		WorkspaceID: workspaceUUID,
	})
}

func (s *JobService) ListJobs(ctx context.Context, workspaceID string) ([]dbgen.Job, error) {
	workspaceUUID, err := uuid.Parse(workspaceID)
	if err != nil {
		return nil, fmt.Errorf("invalid workspace id: %w", err)
	}

	return s.queries.ListJobsByWorkspace(ctx, workspaceUUID)
}

func (s *JobService) CancelJob(ctx context.Context, jobID string) (dbgen.Job, error) {
	jobUUID, err := uuid.Parse(jobID)
	if err != nil {
		return dbgen.Job{}, fmt.Errorf("invalid job id: %w", err)
	}

	// cancel the job
	job, err := s.queries.UpdateJobState(ctx, dbgen.UpdateJobStateParams{
		ID:    jobUUID,
		State: dbgen.JobStateCANCELLED,
	})
	if err != nil {
		return dbgen.Job{}, fmt.Errorf("failed to cancel job: %w", err)
	}

	// cancel all runs for this job so workers stop processing
	runs, err := s.queries.ListRunsByJob(ctx, jobUUID)
	if err != nil {
		return job, fmt.Errorf("failed to list runs: %w", err)
	}
	for _, run := range runs {
		if _, err := s.queries.UpdateRunState(ctx, dbgen.UpdateRunStateParams{
			ID:    run.ID,
			State: dbgen.RunStateCANCELLED,
		}); err != nil {
			slog.Error("failed to cancel run", "run_id", run.ID, "error", err)
		}
	}

	return job, nil
}
func (s *JobService) GetJobProgress(ctx context.Context, jobID, workspaceID string) (dbgen.GetRunProgressRow, error) {
	jobUUID, err := uuid.Parse(jobID)
	if err != nil {
		return dbgen.GetRunProgressRow{}, fmt.Errorf("invalid job id: %w", err)
	}
	workspaceUUID, err := uuid.Parse(workspaceID)
	if err != nil {
		return dbgen.GetRunProgressRow{}, fmt.Errorf("invalid workspace id: %w", err)
	}

	job, err := s.queries.GetJobByID(ctx, dbgen.GetJobByIDParams{
		ID:          jobUUID,
		WorkspaceID: workspaceUUID,
	})
	if err != nil {
		return dbgen.GetRunProgressRow{}, fmt.Errorf("job not found: %w", err)
	}

	runs, err := s.queries.ListRunsByJob(ctx, job.ID)
	if err != nil || len(runs) == 0 {
		return dbgen.GetRunProgressRow{}, fmt.Errorf("no runs found for job: %w", err)
	}

	return s.queries.GetRunProgress(ctx, runs[0].ID)
}

func (s *JobService) GetJobArtifacts(ctx context.Context, jobID, workspaceID string) ([]dbgen.Artifact, error) {
	jobUUID, err := uuid.Parse(jobID)
	if err != nil {
		return nil, fmt.Errorf("invalid job id: %w", err)
	}
	// verify job belongs to workspace
	_, err = s.queries.GetJobByID(ctx, dbgen.GetJobByIDParams{
		ID:          jobUUID,
		WorkspaceID: uuid.MustParse(workspaceID),
	})
	if err != nil {
		return nil, fmt.Errorf("job not found: %w", err)
	}
	return s.queries.ListArtifactsByJob(ctx, jobUUID)
}

func (s *JobService) GetArtifactDownloadURL(ctx context.Context, artifactID string) (string, error) {
	artifactUUID, err := uuid.Parse(artifactID)
	if err != nil {
		return "", fmt.Errorf("invalid artifact id: %w", err)
	}
	artifact, err := s.queries.GetArtifact(ctx, artifactUUID)
	if err != nil {
		return "", fmt.Errorf("artifact not found: %w", err)
	}
	return s.storage.GenerateDownloadURL(ctx, artifact.StoragePath)
}
