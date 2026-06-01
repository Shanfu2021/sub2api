package service

import (
	"context"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
)

type AgentManagementRepository interface {
	GetRootAdmin(ctx context.Context) (*User, error)
	ListDirectChildren(ctx context.Context, parentID int64, roles []string, params pagination.PaginationParams) ([]User, *pagination.PaginationResult, error)
	SumDirectChildAllocations(ctx context.Context, parentID int64, excludeChildID *int64) (concurrency int, rpm int, err error)
	SetParent(ctx context.Context, userID int64, parentID *int64) error
	SetRoleAndParent(ctx context.Context, userID int64, role string, parentID *int64) error
	SetAllocation(ctx context.Context, userID int64, concurrency int, rpm int) error
	DeleteLevel1AgentAndMoveChildren(ctx context.Context, agentID int64, rootAdminID int64) error
}
