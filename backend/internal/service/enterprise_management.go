package service

import (
	"context"
	"fmt"
	"sort"
	"strings"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
)

var (
	ErrEnterpriseManagementForbidden          = infraerrors.Forbidden("ENTERPRISE_MANAGEMENT_FORBIDDEN", "enterprise management operation is not allowed")
	ErrEnterpriseManagementNotEmployee        = infraerrors.Forbidden("ENTERPRISE_MANAGEMENT_NOT_EMPLOYEE", "target user is not an enterprise employee")
	ErrEnterpriseManagementInvalidAllocation  = infraerrors.BadRequest("ENTERPRISE_MANAGEMENT_INVALID_ALLOCATION", "allocation must be non-negative")
	ErrEnterpriseManagementAllocationExceeded = infraerrors.BadRequest("ENTERPRISE_MANAGEMENT_ALLOCATION_EXCEEDED", "allocation exceeds enterprise remaining capacity")
	ErrEnterpriseManagementBalanceExceeded    = infraerrors.BadRequest("ENTERPRISE_MANAGEMENT_BALANCE_EXCEEDED", "employee balance exceeds enterprise available balance")
	ErrEnterpriseManagementInvalidGroup       = infraerrors.BadRequest("ENTERPRISE_MANAGEMENT_INVALID_GROUP", "only active exclusive groups visible to enterprise can be assigned")
	ErrEnterpriseEmployeeImportQuotaExceeded  = infraerrors.BadRequest("ENTERPRISE_EMPLOYEE_IMPORT_QUOTA_EXCEEDED", "employee import exceeds enterprise remaining capacity")
	ErrEnterpriseEmployeeBalanceInitExceeded  = infraerrors.BadRequest("ENTERPRISE_EMPLOYEE_BALANCE_INIT_EXCEEDED", "enterprise balance is not enough to initialize employee balances")
)

type EnterpriseProfile struct {
	UserID          int64 `json:"user_id"`
	PoolConcurrency int   `json:"pool_concurrency"`
	PoolRPM         int   `json:"pool_rpm"`
}

type EmployeeCreateInput struct {
	Email       string  `json:"email"`
	Password    string  `json:"password"`
	Username    string  `json:"username"`
	Balance     float64 `json:"balance"`
	Concurrency int     `json:"concurrency"`
	RPM         int     `json:"rpm"`
}

type EmployeeAllocationUpdate struct {
	Balance     float64 `json:"balance"`
	Concurrency int     `json:"concurrency"`
	RPM         int     `json:"rpm"`
}

type EmployeeImportRecord struct {
	Email         *string  `json:"email"`
	Username      *string  `json:"username"`
	Password      *string  `json:"password"`
	Concurrency   *int     `json:"concurrency"`
	RPM           *int     `json:"rpm"`
	InvalidFields []string `json:"-"`
}

type EmployeeImportInput struct {
	Employees []EmployeeImportRecord `json:"employees"`
}

type EmployeeImportSkip struct {
	Row    int    `json:"row"`
	Email  string `json:"email,omitempty"`
	Reason string `json:"reason"`
}

type EmployeeImportResult struct {
	Created      []User               `json:"created"`
	CreatedCount int                  `json:"created_count"`
	Skipped      []EmployeeImportSkip `json:"skipped"`
	SkippedCount int                  `json:"skipped_count"`
	Allocation   AllocationSummary    `json:"allocation"`
}

type EmployeeBalanceInitializationInput struct {
	Balance float64 `json:"balance"`
}

type EmployeeBalanceInitializationResult struct {
	EmployeeCount           int     `json:"employee_count"`
	TargetBalance           float64 `json:"target_balance"`
	CurrentBalance          float64 `json:"current_balance"`
	RequiredBalance         float64 `json:"required_balance"`
	EnterpriseBalanceBefore float64 `json:"enterprise_balance_before"`
	EnterpriseBalanceAfter  float64 `json:"enterprise_balance_after"`
	AffectedUserIDs         []int64 `json:"-"`
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
	CreateEmployees(ctx context.Context, enterpriseID int64, operatorID int64, users []*User) error
	UpdateEmployeeAllocation(ctx context.Context, enterpriseID int64, employeeID int64, operatorID int64, target EmployeeAllocationUpdate) (*EmployeeAllocationResult, error)
	InitializeEmployeeBalances(ctx context.Context, enterpriseID int64, operatorID int64, targetBalance float64) (*EmployeeBalanceInitializationResult, error)
	DeleteEmployeeAndReturnAllocation(ctx context.Context, enterpriseID int64, employeeID int64, operatorID int64) ([]int64, error)
	HardDeleteEnterpriseWithEmployees(ctx context.Context, enterpriseID int64) ([]int64, error)
	CascadeEnterpriseStatus(ctx context.Context, enterpriseID int64, targetStatus string) ([]int64, error)
	ListDirectChildren(ctx context.Context, parentID int64, roles []string, params pagination.PaginationParams) ([]User, *pagination.PaginationResult, error)
	ListDirectChildrenWithSearch(ctx context.Context, parentID int64, roles []string, params pagination.PaginationParams, search string) ([]User, *pagination.PaginationResult, error)
	ListGroupDelegationsForChild(ctx context.Context, childID int64) ([]AgentGroupDelegation, error)
	GetGroupDelegation(ctx context.Context, managerID int64, childID int64, groupID int64) (*AgentGroupDelegation, error)
	UpsertGroupDelegation(ctx context.Context, managerID int64, childID int64, groupID int64, rateMultiplier float64, canDelegate bool) error
	DeleteGroupDelegation(ctx context.Context, managerID int64, childID int64, groupID int64) error
}

type EnterpriseAdminCleanupRepository interface {
	DeleteEmployeeAndReturnAllocation(ctx context.Context, enterpriseID int64, employeeID int64, operatorID int64) ([]int64, error)
	HardDeleteEnterpriseWithEmployees(ctx context.Context, enterpriseID int64) ([]int64, error)
	CascadeEnterpriseStatus(ctx context.Context, enterpriseID int64, targetStatus string) ([]int64, error)
}

type EnterpriseManagementService struct {
	repo                 EnterpriseManagementRepository
	userRepo             UserRepository
	groupRepo            GroupRepository
	userGroupRateRepo    UserGroupRateRepository
	authCacheInvalidator APIKeyAuthCacheInvalidator
}

func NewEnterpriseManagementService(repo EnterpriseManagementRepository, userRepo UserRepository, groupRepo GroupRepository, userGroupRateRepo UserGroupRateRepository, authCacheInvalidator APIKeyAuthCacheInvalidator) *EnterpriseManagementService {
	return &EnterpriseManagementService{
		repo:                 repo,
		userRepo:             userRepo,
		groupRepo:            groupRepo,
		userGroupRateRepo:    userGroupRateRepo,
		authCacheInvalidator: authCacheInvalidator,
	}
}

func (s *EnterpriseManagementService) ListEmployeesWithQuery(ctx context.Context, actorID int64, query DirectChildrenQuery) (*DirectChildrenResult, error) {
	actor, err := s.requireEnterpriseActor(ctx, actorID)
	if err != nil {
		return nil, err
	}
	if query.Pagination.PageSize == 0 {
		query.Pagination = pagination.DefaultPagination()
	}
	users, page, err := s.repo.ListDirectChildrenWithSearch(ctx, actor.ID, []string{RoleEmployee}, query.Pagination, query.Search)
	if err != nil {
		return nil, err
	}
	return &DirectChildrenResult{Users: users, Pagination: page}, nil
}

func (s *EnterpriseManagementService) CreateEmployee(ctx context.Context, actorID int64, input EmployeeCreateInput) (*User, error) {
	if input.Balance < 0 || input.Concurrency < 1 || input.RPM < 0 {
		return nil, ErrEnterpriseManagementInvalidAllocation
	}
	actor, err := s.requireEnterpriseActor(ctx, actorID)
	if err != nil {
		return nil, err
	}
	if input.Balance > actor.Balance {
		return nil, ErrEnterpriseManagementBalanceExceeded
	}
	if err := s.ensureEnterpriseQuotaAvailable(ctx, actor.ID, nil, input.Concurrency, input.RPM); err != nil {
		return nil, err
	}

	parentID := actor.ID
	employee := &User{
		Email:                input.Email,
		Username:             input.Username,
		Role:                 RoleEmployee,
		ParentUserID:         &parentID,
		Balance:              input.Balance,
		Concurrency:          input.Concurrency,
		AllocatedConcurrency: input.Concurrency,
		RPMLimit:             input.RPM,
		AllocatedRPM:         input.RPM,
		Status:               StatusActive,
	}
	if err := employee.SetPassword(input.Password); err != nil {
		return nil, err
	}
	if err := s.repo.CreateEmployee(ctx, actor.ID, actor.ID, employee); err != nil {
		return nil, err
	}
	s.invalidateUser(ctx, actor.ID)
	s.invalidateUser(ctx, employee.ID)
	if fresh, err := s.userRepo.GetByID(ctx, employee.ID); err == nil {
		return fresh, nil
	}
	return employee, nil
}

func (s *EnterpriseManagementService) ImportEmployees(ctx context.Context, actorID int64, input EmployeeImportInput) (*EmployeeImportResult, error) {
	actor, err := s.requireEnterpriseActor(ctx, actorID)
	if err != nil {
		return nil, err
	}

	validUsers, skipped, err := s.prepareEmployeeImportRows(ctx, actor.ID, input.Employees)
	if err != nil {
		return nil, err
	}
	if len(validUsers) == 0 {
		summary, err := s.GetAllocationSummary(ctx, actor.ID)
		if err != nil {
			return nil, err
		}
		return &EmployeeImportResult{
			Created:      []User{},
			CreatedCount: 0,
			Skipped:      skipped,
			SkippedCount: len(skipped),
			Allocation:   *summary,
		}, nil
	}

	requestedConcurrency, requestedRPM, requestedUnlimitedRPM := employeeImportRequestedQuota(validUsers)
	if err := s.ensureEnterpriseImportQuotaAvailable(ctx, actor.ID, requestedConcurrency, requestedRPM, requestedUnlimitedRPM); err != nil {
		return nil, err
	}
	if err := s.repo.CreateEmployees(ctx, actor.ID, actor.ID, validUsers); err != nil {
		return nil, err
	}

	created := make([]User, 0, len(validUsers))
	affectedUserIDs := []int64{actor.ID}
	for _, user := range validUsers {
		if user == nil {
			continue
		}
		created = append(created, *user)
		affectedUserIDs = append(affectedUserIDs, user.ID)
	}
	s.invalidateUsers(ctx, affectedUserIDs)
	summary, err := s.GetAllocationSummary(ctx, actor.ID)
	if err != nil {
		return nil, err
	}
	return &EmployeeImportResult{
		Created:      created,
		CreatedCount: len(created),
		Skipped:      skipped,
		SkippedCount: len(skipped),
		Allocation:   *summary,
	}, nil
}

func (s *EnterpriseManagementService) UpdateEmployeeAllocation(ctx context.Context, actorID int64, employeeID int64, input EmployeeAllocationUpdate) (*AllocationSummary, error) {
	if input.Balance < 0 || input.Concurrency < 1 || input.RPM < 0 {
		return nil, ErrEnterpriseManagementInvalidAllocation
	}
	actor, err := s.requireEnterpriseActor(ctx, actorID)
	if err != nil {
		return nil, err
	}
	employee, err := s.requireEmployee(ctx, actor.ID, employeeID)
	if err != nil {
		return nil, err
	}
	if input.Balance > employee.Balance+actor.Balance {
		return nil, ErrEnterpriseManagementBalanceExceeded
	}
	if err := s.ensureEnterpriseQuotaAvailable(ctx, actor.ID, &employee.ID, input.Concurrency, input.RPM); err != nil {
		return nil, err
	}
	result, err := s.repo.UpdateEmployeeAllocation(ctx, actor.ID, employee.ID, actor.ID, input)
	if err != nil {
		return nil, err
	}
	s.invalidateUser(ctx, actor.ID)
	s.invalidateUser(ctx, employee.ID)
	if result != nil {
		s.invalidateUsers(ctx, result.AffectedUserIDs)
		return &result.Allocation, nil
	}
	return s.GetAllocationSummary(ctx, actor.ID)
}

func (s *EnterpriseManagementService) InitializeEmployeeBalances(ctx context.Context, actorID int64, input EmployeeBalanceInitializationInput) (*EmployeeBalanceInitializationResult, error) {
	if input.Balance < 0 {
		return nil, ErrEnterpriseManagementInvalidAllocation
	}
	actor, err := s.requireEnterpriseActor(ctx, actorID)
	if err != nil {
		return nil, err
	}
	result, err := s.repo.InitializeEmployeeBalances(ctx, actor.ID, actor.ID, input.Balance)
	if err != nil {
		return nil, err
	}
	if result != nil {
		s.invalidateUsers(ctx, append([]int64{actor.ID}, result.AffectedUserIDs...))
	}
	return result, nil
}

func (s *EnterpriseManagementService) DeleteEmployee(ctx context.Context, actorID int64, employeeID int64) error {
	actor, err := s.requireEnterpriseActor(ctx, actorID)
	if err != nil {
		return err
	}
	employee, err := s.requireEmployee(ctx, actor.ID, employeeID)
	if err != nil {
		return err
	}
	affected, err := s.repo.DeleteEmployeeAndReturnAllocation(ctx, actor.ID, employee.ID, actor.ID)
	if err != nil {
		return err
	}
	s.invalidateUsers(ctx, append([]int64{actor.ID, employee.ID}, affected...))
	return nil
}

func (s *EnterpriseManagementService) GetSummary(ctx context.Context, actorID int64) (*AgentManagementSummary, error) {
	actor, err := s.requireEnterpriseActor(ctx, actorID)
	if err != nil {
		return nil, err
	}
	summary, err := s.GetAllocationSummary(ctx, actor.ID)
	if err != nil {
		return nil, err
	}
	return &AgentManagementSummary{Allocation: *summary}, nil
}

func (s *EnterpriseManagementService) GetAllocationSummary(ctx context.Context, enterpriseID int64) (*AllocationSummary, error) {
	profile, err := s.repo.GetEnterpriseProfile(ctx, enterpriseID)
	if err != nil {
		return nil, err
	}
	if profile == nil {
		profile = &EnterpriseProfile{UserID: enterpriseID}
	}
	usage, err := s.repo.GetEnterpriseEmployeeQuotaUsage(ctx, enterpriseID, nil)
	if err != nil {
		return nil, err
	}
	summary := buildEnterpriseAllocationSummary(profile, usage)
	return &summary, nil
}

func (s *EnterpriseManagementService) ListMyGroups(ctx context.Context, actorID int64) ([]AgentGroupRate, error) {
	actor, err := s.requireEnterpriseActor(ctx, actorID)
	if err != nil {
		return nil, err
	}
	if s.groupRepo == nil || s.repo == nil {
		return []AgentGroupRate{}, nil
	}
	groups, err := s.groupRepo.ListActive(ctx)
	if err != nil {
		return nil, err
	}
	groupsByID := make(map[int64]Group, len(groups))
	out := make([]AgentGroupRate, 0, len(groups))
	for i := range groups {
		groupsByID[groups[i].ID] = groups[i]
		if groups[i].IsExclusive {
			continue
		}
		out = append(out, AgentGroupRate{
			Group:         groups[i],
			EffectiveRate: groups[i].RateMultiplier,
			CanDelegate:   false,
			Source:        "public",
		})
	}
	delegations, err := s.repo.ListGroupDelegationsForChild(ctx, actor.ID)
	if err != nil {
		return nil, err
	}
	for i := range delegations {
		group, ok := groupsByID[delegations[i].GroupID]
		if !ok && delegations[i].Group != nil {
			group = *delegations[i].Group
			ok = true
		}
		if !ok || !group.IsActive() || !group.IsExclusive {
			continue
		}
		group.RateMultiplier = delegations[i].RateMultiplier
		out = append(out, AgentGroupRate{
			Group:         group,
			EffectiveRate: delegations[i].RateMultiplier,
			CanDelegate:   delegations[i].CanDelegate,
			Source:        "delegated",
		})
	}
	return out, nil
}

func (s *EnterpriseManagementService) ListEmployeeGroupOptions(ctx context.Context, actorID int64, employeeID int64) ([]ChildGroupDelegationOption, error) {
	actor, err := s.requireEnterpriseActor(ctx, actorID)
	if err != nil {
		return nil, err
	}
	employee, err := s.requireEmployee(ctx, actor.ID, employeeID)
	if err != nil {
		return nil, err
	}
	groups, err := s.ListMyGroups(ctx, actor.ID)
	if err != nil {
		return nil, err
	}
	delegations, err := s.repo.ListGroupDelegationsForChild(ctx, employee.ID)
	if err != nil {
		return nil, err
	}
	assignedByGroupID := make(map[int64]AgentGroupDelegation, len(delegations))
	for i := range delegations {
		if delegations[i].ManagerUserID == actor.ID {
			assignedByGroupID[delegations[i].GroupID] = delegations[i]
		}
	}

	out := make([]ChildGroupDelegationOption, 0, len(groups))
	for i := range groups {
		if !groups[i].Group.IsExclusive || !groups[i].CanDelegate {
			continue
		}
		group := groups[i].Group
		group.RateMultiplier = groups[i].EffectiveRate
		option := ChildGroupDelegationOption{
			Group:               group,
			EffectiveRate:       groups[i].EffectiveRate,
			CanDelegate:         false,
			Source:              groups[i].Source,
			ChildRateMultiplier: groups[i].EffectiveRate,
			ChildCanDelegate:    false,
		}
		if delegation, ok := assignedByGroupID[groups[i].Group.ID]; ok {
			option.Assigned = true
			option.ChildRateMultiplier = delegation.RateMultiplier
			option.ChildCanDelegate = false
		}
		out = append(out, option)
	}
	return out, nil
}

func (s *EnterpriseManagementService) SetEmployeeGroup(ctx context.Context, actorID int64, employeeID int64, groupID int64, assigned bool) error {
	actor, err := s.requireEnterpriseActor(ctx, actorID)
	if err != nil {
		return err
	}
	employee, err := s.requireEmployee(ctx, actor.ID, employeeID)
	if err != nil {
		return err
	}
	if !assigned {
		if err := s.repo.DeleteGroupDelegation(ctx, actor.ID, employee.ID, groupID); err != nil {
			return err
		}
		if s.userRepo != nil {
			if err := s.userRepo.RemoveGroupFromUserAllowedGroups(ctx, employee.ID, groupID); err != nil {
				return err
			}
		}
		if err := s.syncEmployeeUserGroupRate(ctx, employee.ID, groupID, nil); err != nil {
			return err
		}
		s.invalidateUser(ctx, employee.ID)
		return nil
	}
	rate, err := s.enterpriseDelegableGroupRate(ctx, actor.ID, groupID)
	if err != nil {
		return err
	}
	if err := s.repo.UpsertGroupDelegation(ctx, actor.ID, employee.ID, groupID, rate, false); err != nil {
		return err
	}
	if s.userRepo != nil {
		if err := s.userRepo.AddGroupToAllowedGroups(ctx, employee.ID, groupID); err != nil {
			return err
		}
	}
	if err := s.syncEmployeeUserGroupRate(ctx, employee.ID, groupID, &rate); err != nil {
		return err
	}
	s.invalidateUser(ctx, employee.ID)
	return nil
}

func (s *EnterpriseManagementService) syncEmployeeUserGroupRate(ctx context.Context, userID int64, groupID int64, rate *float64) error {
	if s == nil || s.userGroupRateRepo == nil || userID <= 0 || groupID <= 0 {
		return nil
	}
	return s.userGroupRateRepo.SyncUserGroupRates(ctx, userID, map[int64]*float64{groupID: rate})
}

func (s *EnterpriseManagementService) ensureEnterpriseQuotaAvailable(ctx context.Context, enterpriseID int64, excludeEmployeeID *int64, requestedConcurrency int, requestedRPM int) error {
	profile, err := s.repo.GetEnterpriseProfile(ctx, enterpriseID)
	if err != nil {
		return err
	}
	if profile == nil {
		profile = &EnterpriseProfile{UserID: enterpriseID}
	}
	usage, err := s.repo.GetEnterpriseEmployeeQuotaUsage(ctx, enterpriseID, excludeEmployeeID)
	if err != nil {
		return err
	}
	if concurrencyRequestExceedsCapacity(profile.PoolConcurrency, usage.Concurrency, requestedConcurrency) ||
		rpmRequestExceedsCapacity(profile.PoolRPM, usage.RPM, usage.UnlimitedRPM, requestedRPM) {
		return ErrEnterpriseManagementAllocationExceeded
	}
	return nil
}

func (s *EnterpriseManagementService) ensureEnterpriseImportQuotaAvailable(ctx context.Context, enterpriseID int64, requestedConcurrency int, requestedRPM int, requestedUnlimitedRPM bool) error {
	profile, err := s.repo.GetEnterpriseProfile(ctx, enterpriseID)
	if err != nil {
		return err
	}
	if profile == nil {
		profile = &EnterpriseProfile{UserID: enterpriseID}
	}
	usage, err := s.repo.GetEnterpriseEmployeeQuotaUsage(ctx, enterpriseID, nil)
	if err != nil {
		return err
	}
	effectiveRequestedRPM := requestedRPM
	if requestedUnlimitedRPM {
		effectiveRequestedRPM = 0
	}
	if concurrencyRequestExceedsCapacity(profile.PoolConcurrency, usage.Concurrency, requestedConcurrency) ||
		rpmRequestExceedsCapacity(profile.PoolRPM, usage.RPM, usage.UnlimitedRPM, effectiveRequestedRPM) {
		return ErrEnterpriseEmployeeImportQuotaExceeded.WithMetadata(map[string]string{
			"available_concurrency": fmt.Sprintf("%d", concurrencyRemaining(profile.PoolConcurrency, usage.Concurrency)),
			"required_concurrency":  fmt.Sprintf("%d", requestedConcurrency),
			"available_rpm":         fmt.Sprintf("%d", rpmRemaining(profile.PoolRPM, usage.RPM, usage.UnlimitedRPM)),
			"required_rpm":          fmt.Sprintf("%d", effectiveRequestedRPM),
		})
	}
	return nil
}

func (s *EnterpriseManagementService) requireEnterpriseActor(ctx context.Context, actorID int64) (*User, error) {
	actor, err := s.userRepo.GetByID(ctx, actorID)
	if err != nil {
		return nil, err
	}
	if actor.Role != RoleEnterprise {
		return nil, ErrEnterpriseManagementForbidden
	}
	return actor, nil
}

func (s *EnterpriseManagementService) requireEmployee(ctx context.Context, enterpriseID int64, employeeID int64) (*User, error) {
	employee, err := s.userRepo.GetByID(ctx, employeeID)
	if err != nil {
		return nil, err
	}
	if employee.Role != RoleEmployee || employee.ParentUserID == nil || *employee.ParentUserID != enterpriseID {
		return nil, ErrEnterpriseManagementNotEmployee
	}
	return employee, nil
}

func (s *EnterpriseManagementService) enterpriseDelegableGroupRate(ctx context.Context, enterpriseID int64, groupID int64) (float64, error) {
	groups, err := s.ListMyGroups(ctx, enterpriseID)
	if err != nil {
		return 0, err
	}
	for i := range groups {
		if groups[i].Group.ID == groupID && groups[i].Group.IsExclusive && groups[i].CanDelegate {
			return groups[i].EffectiveRate, nil
		}
	}
	return 0, ErrEnterpriseManagementInvalidGroup
}

func (s *EnterpriseManagementService) prepareEmployeeImportRows(ctx context.Context, enterpriseID int64, records []EmployeeImportRecord) ([]*User, []EmployeeImportSkip, error) {
	valid := make([]*User, 0, len(records))
	skipped := make([]EmployeeImportSkip, 0)
	seenEmails := map[string]struct{}{}

	for i := range records {
		rowNumber := i + 1
		record := records[i]
		email := stringPtrValueOrEmpty(record.Email)
		username := stringPtrValueOrEmpty(record.Username)
		password := stringPtrValueOrEmpty(record.Password)
		missing := missingEmployeeImportFields(record)
		invalid := invalidEmployeeImportFields(record)
		if len(record.InvalidFields) > 0 {
			invalid = append(invalid, record.InvalidFields...)
		}
		if len(missing) > 0 {
			skipped = append(skipped, EmployeeImportSkip{Row: rowNumber, Email: email, Reason: "missing required fields: " + strings.Join(missing, ", ")})
			continue
		}
		if len(invalid) > 0 {
			skipped = append(skipped, EmployeeImportSkip{Row: rowNumber, Email: email, Reason: "invalid fields: " + strings.Join(uniqueStrings(invalid), ", ")})
			continue
		}
		normalizedEmail := strings.ToLower(strings.TrimSpace(email))
		if _, exists := seenEmails[normalizedEmail]; exists {
			skipped = append(skipped, EmployeeImportSkip{Row: rowNumber, Email: email, Reason: "duplicate email in import file"})
			continue
		}
		seenEmails[normalizedEmail] = struct{}{}
		if s.userRepo != nil {
			exists, err := s.userRepo.ExistsByEmail(ctx, email)
			if err != nil {
				return nil, nil, err
			}
			if exists {
				skipped = append(skipped, EmployeeImportSkip{Row: rowNumber, Email: email, Reason: "email already exists"})
				continue
			}
		}

		user := &User{
			Email:                strings.TrimSpace(email),
			Username:             strings.TrimSpace(username),
			Role:                 RoleEmployee,
			ParentUserID:         &enterpriseID,
			Balance:              0,
			Concurrency:          *record.Concurrency,
			AllocatedConcurrency: *record.Concurrency,
			RPMLimit:             *record.RPM,
			AllocatedRPM:         *record.RPM,
			Status:               StatusActive,
		}
		if err := user.SetPassword(password); err != nil {
			skipped = append(skipped, EmployeeImportSkip{Row: rowNumber, Email: email, Reason: "invalid password"})
			continue
		}
		valid = append(valid, user)
	}
	return valid, skipped, nil
}

func (s *EnterpriseManagementService) invalidateUser(ctx context.Context, userID int64) {
	if s.authCacheInvalidator != nil {
		s.authCacheInvalidator.InvalidateAuthCacheByUserID(ctx, userID)
	}
}

func (s *EnterpriseManagementService) invalidateUsers(ctx context.Context, userIDs []int64) {
	seen := make(map[int64]struct{}, len(userIDs))
	for _, userID := range userIDs {
		if _, ok := seen[userID]; ok {
			continue
		}
		seen[userID] = struct{}{}
		s.invalidateUser(ctx, userID)
	}
}

func missingEmployeeImportFields(record EmployeeImportRecord) []string {
	var missing []string
	if record.Email == nil || strings.TrimSpace(*record.Email) == "" {
		missing = append(missing, "email")
	}
	if record.Username == nil || strings.TrimSpace(*record.Username) == "" {
		missing = append(missing, "username")
	}
	if record.Password == nil || strings.TrimSpace(*record.Password) == "" {
		missing = append(missing, "password")
	}
	if record.Concurrency == nil {
		missing = append(missing, "concurrency")
	}
	if record.RPM == nil {
		missing = append(missing, "rpm")
	}
	return missing
}

func invalidEmployeeImportFields(record EmployeeImportRecord) []string {
	var invalid []string
	if record.Concurrency != nil && *record.Concurrency < 1 {
		invalid = append(invalid, "concurrency")
	}
	if record.RPM != nil && *record.RPM < 0 {
		invalid = append(invalid, "rpm")
	}
	return invalid
}

func employeeImportRequestedQuota(users []*User) (int, int, bool) {
	var concurrency int
	var rpm int
	var unlimitedRPM bool
	for _, user := range users {
		if user == nil {
			continue
		}
		concurrency += user.Concurrency
		if user.RPMLimit == 0 {
			unlimitedRPM = true
			continue
		}
		if !unlimitedRPM {
			rpm += user.RPMLimit
		}
	}
	if unlimitedRPM {
		return concurrency, 0, true
	}
	return concurrency, rpm, false
}

func stringPtrValueOrEmpty(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}

func uniqueStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

func buildEnterpriseAllocationSummary(profile *EnterpriseProfile, usage QuotaUsageSummary) AllocationSummary {
	if profile == nil {
		return AllocationSummary{}
	}
	return AllocationSummary{
		TotalConcurrency:     profile.PoolConcurrency,
		AllocatedConcurrency: usage.Concurrency,
		RemainingConcurrency: concurrencyRemaining(profile.PoolConcurrency, usage.Concurrency),
		TotalRPM:             profile.PoolRPM,
		AllocatedRPM:         usage.RPM,
		RemainingRPM:         rpmRemaining(profile.PoolRPM, usage.RPM, usage.UnlimitedRPM),
		UnlimitedCapacity:    false,
		UnlimitedConcurrency: false,
		UnlimitedRPM:         profile.PoolRPM == 0,
	}
}
