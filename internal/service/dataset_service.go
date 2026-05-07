package service

import (
	"context"
	"fmt"

	dbgen "github.com/8bitShinobix/mini-databricks/internal/db/generated"
	"github.com/8bitShinobix/mini-databricks/internal/storage"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

type DatasetService struct {
	queries     *dbgen.Queries
	minioClient *storage.MinioClient
}

func NewDatasetService(queries *dbgen.Queries, minioClient *storage.MinioClient) *DatasetService {
	return &DatasetService{queries: queries, minioClient: minioClient}
}

func (s *DatasetService) InitiateUpload(ctx context.Context, workspaceID, createdBy, name string, fileFormat dbgen.FileFormat) (dbgen.Dataset, string, error) {
	workspaceUUID, err := uuid.Parse(workspaceID)
	if err != nil {
		return dbgen.Dataset{}, "", fmt.Errorf("invalid workspace id: %w", err)
	}

	createdByUUID, err := uuid.Parse(createdBy)
	if err != nil {
		return dbgen.Dataset{}, "", fmt.Errorf("invalid user id: %w", err)
	}

	dataset, err := s.queries.InitiateDataset(ctx, dbgen.InitiateDatasetParams{
		WorkspaceID: workspaceUUID,
		CreatedBy:   createdByUUID,
		Name:        name,
		FileFormat:  fileFormat,
	})
	if err != nil {
		return dbgen.Dataset{}, "", fmt.Errorf("failed to create dataset: %w", err)
	}

	objectKey := fmt.Sprintf("%s/%s/%s.%s", workspaceID, dataset.ID, name, fileFormat)
	uploadURL, err := s.minioClient.GenerateUploadURL(ctx, objectKey)
	if err != nil {
		return dbgen.Dataset{}, "", fmt.Errorf("failed to generate upload url: %w", err)
	}

	return dataset, uploadURL, nil
}

func (s *DatasetService) CompleteUpload(ctx context.Context, datasetID, storagePath string, sizeBytes int64) (dbgen.Dataset, error) {
	datasetUUID, err := uuid.Parse(datasetID)
	if err != nil {
		return dbgen.Dataset{}, fmt.Errorf("invalid dataset id: %w", err)
	}

	dataset, err := s.queries.UpdateDatasetStoragePath(ctx, dbgen.UpdateDatasetStoragePathParams{
		ID: datasetUUID,
		StoragePath: pgtype.Text{
			String: storagePath,
			Valid:  true,
		},
		SizeBytes: pgtype.Int8{
			Int64: sizeBytes,
			Valid: true,
		},
	})
	if err != nil {
		return dbgen.Dataset{}, fmt.Errorf("failed to complete upload: %w", err)
	}

	return dataset, nil
}

func (s *DatasetService) GetDataset(ctx context.Context, datasetID, workspaceID string) (dbgen.Dataset, error) {
	datasetUUID, err := uuid.Parse(datasetID)
	if err != nil {
		return dbgen.Dataset{}, fmt.Errorf("invalid dataset id: %w", err)
	}

	workspaceUUID, err := uuid.Parse(workspaceID)
	if err != nil {
		return dbgen.Dataset{}, fmt.Errorf("invalid workspace id: %w", err)
	}

	return s.queries.GetDatasetByID(ctx, dbgen.GetDatasetByIDParams{
		ID:          datasetUUID,
		WorkspaceID: workspaceUUID,
	})
}

func (s *DatasetService) ListDatasets(ctx context.Context, workspaceID string) ([]dbgen.Dataset, error) {
	workspaceUUID, err := uuid.Parse(workspaceID)
	if err != nil {
		return nil, fmt.Errorf("invalid workspace id: %w", err)
	}

	return s.queries.ListDatasetsByWorkspace(ctx, workspaceUUID)
}

func (s *DatasetService) DeleteDataset(ctx context.Context, datasetID, workspaceID string) error {
	datasetUUID, err := uuid.Parse(datasetID)
	if err != nil {
		return fmt.Errorf("invalid dataset id: %w", err)
	}

	workspaceUUID, err := uuid.Parse(workspaceID)
	if err != nil {
		return fmt.Errorf("invalid workspace id: %w", err)
	}

	return s.queries.DeleteDataset(ctx, dbgen.DeleteDatasetParams{
		ID:          datasetUUID,
		WorkspaceID: workspaceUUID,
	})
}
