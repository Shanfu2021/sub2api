package service

import (
	"context"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

var (
	ErrEnterpriseManagementForbidden          = infraerrors.Forbidden("ENTERPRISE_MANAGEMENT_FORBIDDEN", "enterprise management operation is not allowed")
	ErrEnterpriseManagementNotEmployee        = infraerrors.Forbidden("ENTERPRISE_MANAGEMENT_NOT_EMPLOYEE", "target user is not an enterprise employee")
	ErrEnterpriseManagementInvalidAllocation  = infraerrors.BadRequest("ENTERPRISE_MANAGEMENT_INVALID_ALLOCATION", "allocation must be non-negative")
	ErrEnterpriseManagementAllocationExceeded = infraerrors.BadRequest("ENTERPRISE_MANAGEMENT_ALLOCATION_EXCEEDED", "allocation exceeds enterprise remaining capacity")
	ErrEnterpriseManagementBalanceExceeded    = infraerrors.BadRequest("ENTERPRISE_MANAGEMENT_BALANCE_EXCEEDED", "employee balance exceeds enterprise available balance")
	ErrEnterpriseManagementInvalidGroup       = infraerrors.BadRequest("ENTERPRISE_MANAGEMENT_INVALID_GROUP", "only active exclusive groups visible to enterprise can be assigned")
)

type EnterpriseProfile struct {
	UserID          int64 `json:"user_id"`
	PoolConcurrency int   `json:"pool_concurrency"`
	PoolRPM         int   `json:"pool_rpm"`
}

type EmployeeAllocationUpdate struct {
	Balance     float64 `json:"balance"`
	Concurrency int     `json:"concurrency"`
	RPM         int     `json:"rpm"`
}

type EmployeeAllocationResult struct {
	User               *User
	BalanceDelta       float64
	EnterpriseBalance  float64
	EmployeeBalance    float64
	Allocation         AllocationSummary
	AffectedUserIDs    []int64
	QuotaUsage         QuotaUsageSummary
	RemainingQuotaUser *User
}

type EnterpriseManagementRepository interface {
	GetEnterpriseProfile(ctx context.Context, userID int64) (*EnterpriseProfile, error)
	UpsertEnterpriseProfile(ctx context.Context, userID int64, poolConcurrency int, poolRPM int) error
	GetEnterpriseEmployeeQuotaUsage(ctx context.Context, enterpriseID int64, excludeEmployeeID *int64) (QuotaUsageSummary, error)
	RecalculateEnterpriseQuota(ctx context.Context, enterpriseID int64) error
	CreateEmployee(ctx context.Context, enterpriseID int64, operatorID int64, user *User) error
	UpdateEmployeeAllocation(ctx context.Context, enterpriseID int64, employeeID int64, operatorID int64, target EmployeeAllocationUpdate) (*EmployeeAllocationResult, error)
	DeleteEmployeeAndReturnAllocation(ctx context.Context, enterpriseID int64, employeeID int64, operatorID int64) ([]int64, error)
	HardDeleteEnterpriseWithEmployees(ctx context.Context, enterpriseID int64) ([]int64, error)
	CascadeEnterpriseStatus(ctx context.Context, enterpriseID int64, targetStatus string) ([]int64, error)
}
