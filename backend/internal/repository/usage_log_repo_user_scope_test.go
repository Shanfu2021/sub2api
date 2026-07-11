package repository

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/pkg/usagestats"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/lib/pq"
	"github.com/stretchr/testify/require"
)

func TestUsageLogRepositoryListWithFiltersAppliesUserIDsToCountAndList(t *testing.T) {
	db, mock := newSQLMock(t)
	repo := &usageLogRepository{sql: db}
	userIDs := []int64{22, 11}

	mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM usage_logs WHERE user_id = ANY\\(\\$1\\)").
		WithArgs(pq.Array(userIDs)).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(int64(0)))
	mock.ExpectQuery("SELECT .* FROM usage_logs WHERE user_id = ANY\\(\\$1\\) ORDER BY id DESC LIMIT \\$2 OFFSET \\$3").
		WithArgs(pq.Array(userIDs), 20, 0).
		WillReturnRows(sqlmock.NewRows([]string{"id"}))

	logs, page, err := repo.ListWithFilters(
		context.Background(),
		pagination.PaginationParams{Page: 1, PageSize: 20},
		usagestats.UsageLogFilters{UserIDs: []int64{22, 11, 22, -1}, ExactTotal: true},
	)
	require.NoError(t, err)
	require.Empty(t, logs)
	require.Equal(t, int64(0), page.Total)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUsageLogRepositoryGetStatsWithFiltersPropagatesUserIDs(t *testing.T) {
	db, mock := newSQLMock(t)
	repo := &usageLogRepository{sql: db}
	userIDs := []int64{31, 32}

	mock.ExpectQuery("FROM usage_logs\\s+WHERE user_id = ANY\\(\\$1\\)").
		WithArgs(pq.Array(userIDs)).
		WillReturnRows(usageStatsSummaryRows())
	mock.ExpectQuery("SELECT COALESCE\\(NULLIF\\(TRIM\\(inbound_endpoint\\)").
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), pq.Array(userIDs)).
		WillReturnRows(endpointStatsRows())
	mock.ExpectQuery("SELECT COALESCE\\(NULLIF\\(TRIM\\(upstream_endpoint\\)").
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), pq.Array(userIDs)).
		WillReturnRows(endpointStatsRows())
	mock.ExpectQuery("SELECT CONCAT\\(").
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), pq.Array(userIDs)).
		WillReturnRows(endpointStatsRows())

	stats, err := repo.GetStatsWithFilters(context.Background(), usagestats.UsageLogFilters{UserIDs: userIDs})
	require.NoError(t, err)
	require.Equal(t, int64(1), stats.TotalRequests)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUsageLogRepositoryUsageFilterWrappersPropagateUserIDs(t *testing.T) {
	start := time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC)
	end := start.Add(24 * time.Hour)
	userIDs := []int64{41, 42}
	filters := usagestats.UsageLogFilters{UserIDs: userIDs}

	t.Run("trend", func(t *testing.T) {
		db, mock := newSQLMock(t)
		repo := &usageLogRepository{sql: db}
		mock.ExpectQuery("FROM usage_logs\\s+WHERE created_at >= \\$1 AND created_at < \\$2 AND user_id = ANY\\(\\$3\\)").
			WithArgs(start, end, pq.Array(userIDs)).
			WillReturnRows(sqlmock.NewRows([]string{"date", "requests", "input_tokens", "output_tokens", "cache_creation_tokens", "cache_read_tokens", "total_tokens", "cost", "actual_cost"}))

		result, err := repo.GetUsageTrendWithUsageFilters(context.Background(), start, end, "day", filters)
		require.NoError(t, err)
		require.Empty(t, result)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("model", func(t *testing.T) {
		db, mock := newSQLMock(t)
		repo := &usageLogRepository{sql: db}
		mock.ExpectQuery("FROM usage_logs\\s+WHERE created_at >= \\$1 AND created_at < \\$2 AND user_id = ANY\\(\\$3\\)").
			WithArgs(start, end, pq.Array(userIDs)).
			WillReturnRows(sqlmock.NewRows([]string{"model", "requests", "input_tokens", "output_tokens", "cache_creation_tokens", "cache_read_tokens", "total_tokens", "cost", "actual_cost", "account_cost"}))

		result, err := repo.GetModelStatsWithUsageFiltersBySource(context.Background(), start, end, filters, usagestats.ModelSourceRequested)
		require.NoError(t, err)
		require.Empty(t, result)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("group", func(t *testing.T) {
		db, mock := newSQLMock(t)
		repo := &usageLogRepository{sql: db}
		mock.ExpectQuery("(?s)FROM usage_logs ul.*WHERE ul.created_at >= \\$1 AND ul.created_at < \\$2 AND ul.user_id = ANY\\(\\$3\\)").
			WithArgs(start, end, pq.Array(userIDs)).
			WillReturnRows(sqlmock.NewRows([]string{"group_id", "group_name", "requests", "total_tokens", "cost", "actual_cost", "account_cost"}))

		result, err := repo.GetGroupStatsWithUsageFilters(context.Background(), start, end, filters)
		require.NoError(t, err)
		require.Empty(t, result)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestPrepareUsageLogInsertPreservesAgentSnapshotAndNegativeIncome(t *testing.T) {
	ownerID := int64(73)
	prepared := prepareUsageLogInsert(&service.UsageLog{
		UserID:                  1,
		APIKeyID:                2,
		AccountID:               3,
		RequestID:               "req-negative-agent-income",
		Model:                   "gpt-5",
		AgentOwnerUserID:        &ownerID,
		AgentUserRateMultiplier: 0.8,
		AgentCostRateMultiplier: 1.2,
		AgentIncome:             -0.25,
		CreatedAt:               time.Date(2025, 2, 2, 0, 0, 0, 0, time.UTC),
	})

	require.Equal(t, &ownerID, prepared.args[24])
	require.Equal(t, 0.8, prepared.args[25])
	require.Equal(t, 1.2, prepared.args[26])
	require.Equal(t, -0.25, prepared.args[27])

	batchQuery, batchArgs := buildUsageLogBatchInsertQuery(
		[]string{usageLogBatchKey(prepared.requestID, 2)},
		map[string]usageLogInsertPrepared{usageLogBatchKey(prepared.requestID, 2): prepared},
	)
	bestEffortQuery, bestEffortArgs := buildUsageLogBestEffortInsertQuery([]usageLogInsertPrepared{prepared})
	for _, query := range []string{batchQuery, bestEffortQuery} {
		require.Contains(t, query, "agent_owner_user_id")
		require.Contains(t, query, "agent_user_rate_multiplier")
		require.Contains(t, query, "agent_cost_rate_multiplier")
		require.Contains(t, query, "agent_income")
	}
	require.Contains(t, batchArgs, -0.25)
	require.Contains(t, bestEffortArgs, -0.25)
	require.Equal(t, 3, strings.Count(batchQuery, "agent_income"))
	require.Equal(t, 3, strings.Count(bestEffortQuery, "agent_income"))
}

func usageStatsSummaryRows() *sqlmock.Rows {
	return sqlmock.NewRows([]string{
		"total_requests",
		"total_input_tokens",
		"total_output_tokens",
		"total_cache_tokens",
		"total_cache_creation_tokens",
		"total_cache_read_tokens",
		"total_cost",
		"total_actual_cost",
		"total_account_cost",
		"avg_duration_ms",
	}).AddRow(int64(1), int64(2), int64(3), int64(4), int64(1), int64(3), 1.2, 1.0, 1.2, 20.0)
}

func endpointStatsRows() *sqlmock.Rows {
	return sqlmock.NewRows([]string{"endpoint", "requests", "total_tokens", "cost", "actual_cost"})
}
