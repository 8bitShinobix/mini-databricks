package service

import (
	"context"
	"fmt"

	dbgen "github.com/8bitShinobix/mini-databricks/internal/db/generated"
	"github.com/google/uuid"
)

type WorkspaceService struct {
	queries *dbgen.Queries
}

func NewWorkspaceService(queries *dbgen.Queries) *WorkspaceService {
	return &WorkspaceService{queries: queries}
}

func (s *WorkspaceService) CreateWorkspace(ctx context.Context, name string, ownerID string, plan dbgen.WorkspacePlan) (dbgen.Workspace, error) {
	ownerUUID, err := uuid.Parse(ownerID)
	if err != nil {
		return dbgen.Workspace{}, fmt.Errorf("invalid owner id: %w", err)
	}

	workspace, err := s.queries.CreateWorkspace(ctx, dbgen.CreateWorkspaceParams{
		Name:    name,
		OwnerID: ownerUUID,
		Plan:    plan,
	})
	if err != nil {
		return dbgen.Workspace{}, err
	}
	return workspace, nil
}

func (s *WorkspaceService) GetWorkspacesByOwner(ctx context.Context, ownerID string) ([]dbgen.Workspace, error) {
	ownerUUID, err := uuid.Parse(ownerID)
	if err != nil {
		return nil, fmt.Errorf("invalid owner id: %w", err)
	}

	workspaces, err := s.queries.GetWorkspacesByOwner(ctx, ownerUUID)
	if err != nil {
		return nil, err
	}
	return workspaces, nil
}

func (s *WorkspaceService) GetWorkspaceByID(ctx context.Context, workspaceID string) (dbgen.Workspace, error) {
	workspaceUUID, err := uuid.Parse(workspaceID)
	if err != nil {
		return dbgen.Workspace{}, fmt.Errorf("invalid workspace id: %w", err)
	}

	workspace, err := s.queries.GetWorkspaceByID(ctx, workspaceUUID)
	if err != nil {
		return dbgen.Workspace{}, err
	}
	return workspace, nil
}

func (s *WorkspaceService) DeleteWorkspace(ctx context.Context, workspaceID string, ownerID string) error {
	workspaceUUID, err := uuid.Parse(workspaceID)
	if err != nil {
		return fmt.Errorf("invalid workspace id: %w", err)
	}

	ownerUUID, err := uuid.Parse(ownerID)
	if err != nil {
		return fmt.Errorf("invalid owner id: %w", err)
	}

	return s.queries.DeleteWorkspace(ctx, dbgen.DeleteWorkspaceParams{
		ID:      workspaceUUID,
		OwnerID: ownerUUID,
	})
}

func (s *WorkspaceService) UpdateWorkspace(ctx context.Context, workspaceID string, ownerID string, name *string, plan *dbgen.WorkspacePlan) (dbgen.Workspace, error) {
	workspaceUUID, err := uuid.Parse(workspaceID)
	if err != nil {
		return dbgen.Workspace{}, fmt.Errorf("invalid workspace id: %w", err)
	}

	ownerUUID, err := uuid.Parse(ownerID)
	if err != nil {
		return dbgen.Workspace{}, fmt.Errorf("invalid owner id: %w", err)
	}

	existing, err := s.queries.GetWorkspaceByID(ctx, workspaceUUID)
	if err != nil {
		return dbgen.Workspace{}, fmt.Errorf("workspace not found: %w", err)
	}

	updatedName := existing.Name
	if name != nil {
		updatedName = *name
	}

	updatedPlan := existing.Plan
	if plan != nil {
		updatedPlan = *plan
	}

	return s.queries.UpdateWorkspace(ctx, dbgen.UpdateWorkspaceParams{
		ID:      workspaceUUID,
		OwnerID: ownerUUID,
		Name:    updatedName,
		Plan:    updatedPlan,
	})
}
