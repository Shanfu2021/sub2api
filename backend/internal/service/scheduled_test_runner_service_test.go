//go:build unit

package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type scheduledTestRunnerPlanRepoStub struct {
	updatedAfterRun bool
	lastRunAt       time.Time
	nextRunAt       time.Time
}

func (r *scheduledTestRunnerPlanRepoStub) Create(ctx context.Context, plan *ScheduledTestPlan) (*ScheduledTestPlan, error) {
	return plan, nil
}

func (r *scheduledTestRunnerPlanRepoStub) GetByID(ctx context.Context, id int64) (*ScheduledTestPlan, error) {
	return nil, errors.New("not implemented")
}

func (r *scheduledTestRunnerPlanRepoStub) ListByAccountID(ctx context.Context, accountID int64) ([]*ScheduledTestPlan, error) {
	return nil, nil
}

func (r *scheduledTestRunnerPlanRepoStub) ListDue(ctx context.Context, now time.Time) ([]*ScheduledTestPlan, error) {
	return nil, nil
}

func (r *scheduledTestRunnerPlanRepoStub) Update(ctx context.Context, plan *ScheduledTestPlan) (*ScheduledTestPlan, error) {
	return plan, nil
}

func (r *scheduledTestRunnerPlanRepoStub) Delete(ctx context.Context, id int64) error {
	return nil
}

func (r *scheduledTestRunnerPlanRepoStub) UpdateAfterRun(ctx context.Context, id int64, lastRunAt time.Time, nextRunAt time.Time) error {
	r.updatedAfterRun = true
	r.lastRunAt = lastRunAt
	r.nextRunAt = nextRunAt
	return nil
}

type scheduledTestRunnerResultRepoStub struct {
	results []*ScheduledTestResult
}

func (r *scheduledTestRunnerResultRepoStub) Create(ctx context.Context, result *ScheduledTestResult) (*ScheduledTestResult, error) {
	copied := *result
	copied.ID = int64(len(r.results) + 1)
	r.results = append(r.results, &copied)
	return &copied, nil
}

func (r *scheduledTestRunnerResultRepoStub) ListByPlanID(ctx context.Context, planID int64, limit int) ([]*ScheduledTestResult, error) {
	return r.results, nil
}

func (r *scheduledTestRunnerResultRepoStub) PruneOldResults(ctx context.Context, planID int64, keepCount int) error {
	return nil
}

type scheduledTestRunnerAccountRepoStub struct {
	mockAccountRepoForGemini
	schedulableCalls []bool
	schedulableIDs   []int64
}

func (r *scheduledTestRunnerAccountRepoStub) SetSchedulable(ctx context.Context, id int64, schedulable bool) error {
	r.schedulableIDs = append(r.schedulableIDs, id)
	r.schedulableCalls = append(r.schedulableCalls, schedulable)
	return nil
}

func newScheduledTestRunnerForDecision(result *ScheduledTestResult) (*ScheduledTestRunnerService, *scheduledTestRunnerResultRepoStub, *scheduledTestRunnerAccountRepoStub) {
	planRepo := &scheduledTestRunnerPlanRepoStub{}
	resultRepo := &scheduledTestRunnerResultRepoStub{}
	accountRepo := &scheduledTestRunnerAccountRepoStub{}
	accountRepo.accountsByID = map[int64]*Account{
		42: {ID: 42, Status: StatusActive, Schedulable: true},
		43: {ID: 43, Status: StatusActive, Schedulable: true},
		44: {ID: 44, Status: StatusActive, Schedulable: false},
	}
	scheduledSvc := NewScheduledTestService(planRepo, resultRepo)
	accountTestSvc := &AccountTestService{
		runTestBackgroundFunc: func(ctx context.Context, accountID int64, modelID string) (*ScheduledTestResult, error) {
			copied := *result
			return &copied, nil
		},
	}
	rateLimitSvc := NewRateLimitService(accountRepo, nil, nil, nil, nil)
	return NewScheduledTestRunnerService(planRepo, scheduledSvc, accountTestSvc, rateLimitSvc, nil), resultRepo, accountRepo
}

func TestScheduledTestRunner_AutoSchedulableControlDisablesOnFailure(t *testing.T) {
	runner, resultRepo, accountRepo := newScheduledTestRunnerForDecision(&ScheduledTestResult{
		Status:       "failed",
		ErrorMessage: "upstream failed",
		StartedAt:    time.Now(),
		FinishedAt:   time.Now(),
	})
	plan := &ScheduledTestPlan{
		ID:                     10,
		AccountID:              42,
		ModelID:                "claude-sonnet-4-5",
		CronExpression:         "*/5 * * * *",
		MaxResults:             50,
		AutoSchedulableControl: true,
		FirstTokenTimeoutMs:    10_000,
	}

	runner.runOnePlan(context.Background(), plan)

	require.Equal(t, []int64{42}, accountRepo.schedulableIDs)
	require.Equal(t, []bool{false}, accountRepo.schedulableCalls)
	require.Len(t, resultRepo.results, 1)
	require.Equal(t, ScheduledTestDecisionDisabledFailure, resultRepo.results[0].Decision)
	require.Contains(t, resultRepo.results[0].DecisionReason, "upstream failed")
}

func TestScheduledTestRunner_AutoSchedulableControlDisablesOnSlowFirstToken(t *testing.T) {
	runner, resultRepo, accountRepo := newScheduledTestRunnerForDecision(&ScheduledTestResult{
		Status:       "success",
		FirstTokenMs: scheduledTestInt64Ptr(12_500),
		StartedAt:    time.Now(),
		FinishedAt:   time.Now(),
	})
	plan := &ScheduledTestPlan{
		ID:                     11,
		AccountID:              43,
		ModelID:                "claude-sonnet-4-5",
		CronExpression:         "*/5 * * * *",
		MaxResults:             50,
		AutoSchedulableControl: true,
		FirstTokenTimeoutMs:    10_000,
	}

	runner.runOnePlan(context.Background(), plan)

	require.Equal(t, []int64{43}, accountRepo.schedulableIDs)
	require.Equal(t, []bool{false}, accountRepo.schedulableCalls)
	require.Len(t, resultRepo.results, 1)
	require.Equal(t, ScheduledTestDecisionDisabledSlowFirstToken, resultRepo.results[0].Decision)
	require.Contains(t, resultRepo.results[0].DecisionReason, "12500ms")
}

func TestScheduledTestRunner_AutoSchedulableControlEnablesOnHealthySuccess(t *testing.T) {
	runner, resultRepo, accountRepo := newScheduledTestRunnerForDecision(&ScheduledTestResult{
		Status:       "success",
		FirstTokenMs: scheduledTestInt64Ptr(800),
		StartedAt:    time.Now(),
		FinishedAt:   time.Now(),
	})
	plan := &ScheduledTestPlan{
		ID:                     12,
		AccountID:              44,
		ModelID:                "claude-sonnet-4-5",
		CronExpression:         "*/5 * * * *",
		MaxResults:             50,
		AutoSchedulableControl: true,
		FirstTokenTimeoutMs:    10_000,
	}

	runner.runOnePlan(context.Background(), plan)

	require.Equal(t, []int64{44}, accountRepo.schedulableIDs)
	require.Equal(t, []bool{true}, accountRepo.schedulableCalls)
	require.Len(t, resultRepo.results, 1)
	require.Equal(t, ScheduledTestDecisionEnabled, resultRepo.results[0].Decision)
}

func scheduledTestInt64Ptr(v int64) *int64 {
	return &v
}
