package service

import (
	"context"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/robfig/cron/v3"
)

const scheduledTestDefaultMaxWorkers = 10
const autoHealthProbeMaxWorkers = 1
const autoHealthProbeMaxPerTick = 2
const autoHealthScanPageSize = 100

// ScheduledTestRunnerService periodically scans due test plans and executes them.
type ScheduledTestRunnerService struct {
	planRepo       ScheduledTestPlanRepository
	scheduledSvc   *ScheduledTestService
	accountTestSvc *AccountTestService
	rateLimitSvc   *RateLimitService
	cfg            *config.Config

	cron      *cron.Cron
	startOnce sync.Once
	stopOnce  sync.Once

	autoHealthMu   sync.Mutex
	autoHealthPage int
}

// NewScheduledTestRunnerService creates a new runner.
func NewScheduledTestRunnerService(
	planRepo ScheduledTestPlanRepository,
	scheduledSvc *ScheduledTestService,
	accountTestSvc *AccountTestService,
	rateLimitSvc *RateLimitService,
	cfg *config.Config,
) *ScheduledTestRunnerService {
	return &ScheduledTestRunnerService{
		planRepo:       planRepo,
		scheduledSvc:   scheduledSvc,
		accountTestSvc: accountTestSvc,
		rateLimitSvc:   rateLimitSvc,
		cfg:            cfg,
	}
}

// Start begins the cron ticker (every minute).
func (s *ScheduledTestRunnerService) Start() {
	if s == nil {
		return
	}
	s.startOnce.Do(func() {
		loc := time.Local
		if s.cfg != nil {
			if parsed, err := time.LoadLocation(s.cfg.Timezone); err == nil && parsed != nil {
				loc = parsed
			}
		}

		c := cron.New(cron.WithParser(scheduledTestCronParser), cron.WithLocation(loc))
		_, err := c.AddFunc("* * * * *", func() { s.runScheduled() })
		if err != nil {
			logger.LegacyPrintf("service.scheduled_test_runner", "[ScheduledTestRunner] not started (invalid schedule): %v", err)
			return
		}
		s.cron = c
		s.cron.Start()
		logger.LegacyPrintf("service.scheduled_test_runner", "[ScheduledTestRunner] started (tick=every minute)")
	})
}

// Stop gracefully shuts down the cron scheduler.
func (s *ScheduledTestRunnerService) Stop() {
	if s == nil {
		return
	}
	s.stopOnce.Do(func() {
		if s.cron != nil {
			ctx := s.cron.Stop()
			select {
			case <-ctx.Done():
			case <-time.After(3 * time.Second):
				logger.LegacyPrintf("service.scheduled_test_runner", "[ScheduledTestRunner] cron stop timed out")
			}
		}
	})
}

func (s *ScheduledTestRunnerService) runScheduled() {
	// Delay 10s so execution lands at ~:10 of each minute instead of :00.
	time.Sleep(10 * time.Second)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	now := time.Now()
	plans, err := s.planRepo.ListDue(ctx, now)
	if err != nil {
		logger.LegacyPrintf("service.scheduled_test_runner", "[ScheduledTestRunner] ListDue error: %v", err)
	} else if len(plans) > 0 {
		logger.LegacyPrintf("service.scheduled_test_runner", "[ScheduledTestRunner] found %d due plans", len(plans))
		s.runScheduledPlans(ctx, plans)
	}

	s.runAutoHealthPolicies(ctx, now)
}

func (s *ScheduledTestRunnerService) runScheduledPlans(ctx context.Context, plans []*ScheduledTestPlan) {
	sem := make(chan struct{}, scheduledTestDefaultMaxWorkers)
	var wg sync.WaitGroup

	for _, plan := range plans {
		sem <- struct{}{}
		wg.Add(1)
		go func(p *ScheduledTestPlan) {
			defer wg.Done()
			defer func() { <-sem }()
			s.runOnePlan(ctx, p)
		}(plan)
	}

	wg.Wait()
}

func (s *ScheduledTestRunnerService) runAutoHealthPolicies(ctx context.Context, now time.Time) {
	if s == nil || s.rateLimitSvc == nil || s.rateLimitSvc.accountRepo == nil || s.accountTestSvc == nil {
		return
	}
	page := s.nextAutoHealthScanPage()
	accounts, result, err := s.rateLimitSvc.accountRepo.List(ctx, pagination.PaginationParams{
		Page:      page,
		PageSize:  autoHealthScanPageSize,
		SortBy:    "id",
		SortOrder: pagination.SortOrderAsc,
	})
	if err != nil {
		logger.LegacyPrintf("service.scheduled_test_runner", "[AutoHealth] list accounts error: %v", err)
		return
	}
	if result != nil && (result.Pages <= 0 || page >= result.Pages) {
		s.resetAutoHealthScanPage()
	}

	candidates := make([]Account, 0, autoHealthProbeMaxPerTick)
	for _, account := range accounts {
		if len(candidates) >= autoHealthProbeMaxPerTick {
			break
		}
		policy := account.AutoHealthPolicy()
		if !autoHealthNeedsProbe(&account, policy, now) {
			continue
		}
		candidates = append(candidates, account)
	}
	if len(candidates) == 0 {
		return
	}

	sem := make(chan struct{}, autoHealthProbeMaxWorkers)
	var wg sync.WaitGroup
	for i := range candidates {
		account := candidates[i]
		sem <- struct{}{}
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer func() { <-sem }()
			s.runOneAutoHealthProbe(ctx, &account, now)
		}()
	}
	wg.Wait()
}

func (s *ScheduledTestRunnerService) nextAutoHealthScanPage() int {
	s.autoHealthMu.Lock()
	defer s.autoHealthMu.Unlock()
	if s.autoHealthPage <= 0 {
		s.autoHealthPage = 1
	}
	page := s.autoHealthPage
	s.autoHealthPage++
	return page
}

func (s *ScheduledTestRunnerService) resetAutoHealthScanPage() {
	s.autoHealthMu.Lock()
	defer s.autoHealthMu.Unlock()
	s.autoHealthPage = 1
}

func autoHealthNeedsProbe(account *Account, policy AccountAutoHealthPolicy, now time.Time) bool {
	if account == nil || !policy.Enabled {
		return false
	}
	if policy.NextProbeAt != nil && now.Before(*policy.NextProbeAt) {
		return false
	}
	if account.Status == StatusError {
		return policy.RecoverStatusError
	}
	if account.TempUnschedulableUntil != nil && now.Before(*account.TempUnschedulableUntil) {
		return true
	}
	return false
}

func (s *ScheduledTestRunnerService) runOneAutoHealthProbe(ctx context.Context, account *Account, now time.Time) {
	if account == nil || s.rateLimitSvc == nil || s.accountTestSvc == nil {
		return
	}
	policy := account.AutoHealthPolicy()
	if !policy.Enabled {
		return
	}
	probeCtx, cancel := context.WithTimeout(ctx, 90*time.Second)
	defer cancel()

	result, err := s.accountTestSvc.RunTestBackground(probeCtx, account.ID, policy.ProbeModel)
	nextProbe := now.Add(time.Duration(policy.ProbeIntervalMinutes) * time.Minute)
	if err != nil {
		s.rateLimitSvc.setAutoHealthTempUnschedulable(ctx, account, 0, "probe_error", []byte(err.Error()), policy.ErrorPauseMinutes, policy)
		logger.LegacyPrintf("service.scheduled_test_runner", "[AutoHealth] account=%d probe failed: %v", account.ID, err)
		return
	}
	if result == nil || result.Status != "success" {
		status := "probe_failed"
		message := ""
		if result != nil {
			message = result.ErrorMessage
		}
		s.rateLimitSvc.setAutoHealthTempUnschedulable(ctx, account, 0, status, []byte(message), policy.ErrorPauseMinutes, policy)
		return
	}
	if policy.SlowFirstTokenMs > 0 && result.LatencyMs > int64(policy.SlowFirstTokenMs) {
		message := "auto health probe is still slow"
		s.rateLimitSvc.setAutoHealthTempUnschedulable(ctx, account, 0, "slow_probe", []byte(message), policy.SlowPauseMinutes, policy)
		return
	}
	recovery, recoverErr := s.rateLimitSvc.RecoverAccountAfterSuccessfulTest(ctx, account.ID)
	if recoverErr != nil {
		logger.LegacyPrintf("service.scheduled_test_runner", "[AutoHealth] account=%d recovery failed: %v", account.ID, recoverErr)
		s.rateLimitSvc.persistAutoHealthProbeState(ctx, account.ID, now, nextProbe, "recover_failed")
		return
	}
	_ = recovery
	s.rateLimitSvc.persistAutoHealthProbeState(ctx, account.ID, now, nextProbe, "success")
	logger.LegacyPrintf("service.scheduled_test_runner", "[AutoHealth] account=%d recovered by successful probe", account.ID)
}

func (s *ScheduledTestRunnerService) runOnePlan(ctx context.Context, plan *ScheduledTestPlan) {
	result, err := s.accountTestSvc.RunTestBackground(ctx, plan.AccountID, plan.ModelID)
	if err != nil {
		logger.LegacyPrintf("service.scheduled_test_runner", "[ScheduledTestRunner] plan=%d RunTestBackground error: %v", plan.ID, err)
		return
	}

	if err := s.scheduledSvc.SaveResult(ctx, plan.ID, plan.MaxResults, result); err != nil {
		logger.LegacyPrintf("service.scheduled_test_runner", "[ScheduledTestRunner] plan=%d SaveResult error: %v", plan.ID, err)
	}

	// Auto-recover account if test succeeded and auto_recover is enabled.
	if result.Status == "success" && plan.AutoRecover {
		s.tryRecoverAccount(ctx, plan.AccountID, plan.ID)
	}

	nextRun, err := computeNextRun(plan.CronExpression, time.Now())
	if err != nil {
		logger.LegacyPrintf("service.scheduled_test_runner", "[ScheduledTestRunner] plan=%d computeNextRun error: %v", plan.ID, err)
		return
	}

	if err := s.planRepo.UpdateAfterRun(ctx, plan.ID, time.Now(), nextRun); err != nil {
		logger.LegacyPrintf("service.scheduled_test_runner", "[ScheduledTestRunner] plan=%d UpdateAfterRun error: %v", plan.ID, err)
	}
}

// tryRecoverAccount attempts to recover an account from recoverable runtime state.
func (s *ScheduledTestRunnerService) tryRecoverAccount(ctx context.Context, accountID int64, planID int64) {
	if s.rateLimitSvc == nil {
		return
	}

	recovery, err := s.rateLimitSvc.RecoverAccountAfterSuccessfulTest(ctx, accountID)
	if err != nil {
		logger.LegacyPrintf("service.scheduled_test_runner", "[ScheduledTestRunner] plan=%d auto-recover failed: %v", planID, err)
		return
	}
	if recovery == nil {
		return
	}

	if recovery.ClearedError {
		logger.LegacyPrintf("service.scheduled_test_runner", "[ScheduledTestRunner] plan=%d auto-recover: account=%d recovered from error status", planID, accountID)
	}
	if recovery.ClearedRateLimit {
		logger.LegacyPrintf("service.scheduled_test_runner", "[ScheduledTestRunner] plan=%d auto-recover: account=%d cleared rate-limit/runtime state", planID, accountID)
	}
}
