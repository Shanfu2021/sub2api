package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type employeeRestrictionAffiliateRepoStub struct {
	service.AffiliateRepository
	ensureCalls   int
	transferCalls int
}

func (r *employeeRestrictionAffiliateRepoStub) EnsureUserAffiliate(context.Context, int64) (*service.AffiliateSummary, error) {
	r.ensureCalls++
	return &service.AffiliateSummary{UserID: 42, AffCode: "AFF42"}, nil
}
func (r *employeeRestrictionAffiliateRepoStub) ThawFrozenQuota(context.Context, int64) (float64, error) {
	return 0, nil
}
func (r *employeeRestrictionAffiliateRepoStub) TransferQuotaToBalance(context.Context, int64) (float64, float64, error) {
	r.transferCalls++
	return 10, 20, nil
}
func (r *employeeRestrictionAffiliateRepoStub) ListInvitees(context.Context, int64, int) ([]service.AffiliateInvitee, error) {
	return nil, nil
}

type employeeRestrictionSubscriptionRepoStub struct {
	service.UserSubscriptionRepository
	listByUserCalls       int
	listActiveByUserCalls int
}

func (r *employeeRestrictionSubscriptionRepoStub) ListByUserID(context.Context, int64) ([]service.UserSubscription, error) {
	r.listByUserCalls++
	return []service.UserSubscription{}, nil
}
func (r *employeeRestrictionSubscriptionRepoStub) ListActiveByUserID(context.Context, int64) ([]service.UserSubscription, error) {
	r.listActiveByUserCalls++
	return []service.UserSubscription{}, nil
}
func (r *employeeRestrictionSubscriptionRepoStub) GetByID(context.Context, int64) (*service.UserSubscription, error) {
	return nil, errors.New("unexpected GetByID")
}

func TestEmployeeCannotUseAffiliateEndpoints(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &employeeRestrictionAffiliateRepoStub{}
	h := NewUserHandler(nil, nil, nil, nil, service.NewAffiliateService(repo, nil, nil, nil), nil)

	for _, tc := range []struct {
		name    string
		method  string
		path    string
		invoke  func(*gin.Context)
		counter func() int
	}{
		{
			name:    "detail",
			method:  http.MethodGet,
			path:    "/api/v1/user/aff",
			invoke:  h.GetAffiliate,
			counter: func() int { return repo.ensureCalls },
		},
		{
			name:    "transfer",
			method:  http.MethodPost,
			path:    "/api/v1/user/aff/transfer",
			invoke:  h.TransferAffiliateQuota,
			counter: func() int { return repo.transferCalls },
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(recorder)
			c.Request = httptest.NewRequest(tc.method, tc.path, nil)
			c.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: 42})
			c.Set(string(middleware2.ContextKeyUserRole), service.RoleEmployee)

			tc.invoke(c)

			require.Equal(t, http.StatusForbidden, recorder.Code)
			require.Equal(t, 0, tc.counter(), "employee affiliate request must not call service repository")
			require.Equal(t, "EMPLOYEE_FEATURE_RESTRICTED", responseReason(t, recorder))
		})
	}
}

func TestEmployeeCannotViewSubscriptions(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &employeeRestrictionSubscriptionRepoStub{}
	h := NewSubscriptionHandler(service.NewSubscriptionService(nil, repo, nil, nil, nil))

	for _, tc := range []struct {
		name   string
		path   string
		invoke func(*gin.Context)
	}{
		{name: "list", path: "/api/v1/subscriptions", invoke: h.List},
		{name: "active", path: "/api/v1/subscriptions/active", invoke: h.GetActive},
		{name: "progress", path: "/api/v1/subscriptions/progress", invoke: h.GetProgress},
		{name: "summary", path: "/api/v1/subscriptions/summary", invoke: h.GetSummary},
	} {
		t.Run(tc.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(recorder)
			c.Request = httptest.NewRequest(http.MethodGet, tc.path, nil)
			c.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: 42})
			c.Set(string(middleware2.ContextKeyUserRole), service.RoleEmployee)

			tc.invoke(c)

			require.Equal(t, http.StatusForbidden, recorder.Code)
			require.Equal(t, 0, repo.listByUserCalls)
			require.Equal(t, 0, repo.listActiveByUserCalls)
			require.Equal(t, "EMPLOYEE_FEATURE_RESTRICTED", responseReason(t, recorder))
		})
	}
}

func TestEmployeePaymentHandlerBlocksBeforeCreateOrder(t *testing.T) {
	gin.SetMode(gin.TestMode)

	h := NewPaymentHandler(nil, nil)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/payment/orders", bytes.NewReader([]byte(`{"amount":10,"payment_type":"alipay"}`)))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: 42})
	c.Set(string(middleware2.ContextKeyUserRole), service.RoleEmployee)

	h.CreateOrder(c)

	require.Equal(t, http.StatusForbidden, recorder.Code)
	require.Equal(t, "EMPLOYEE_FEATURE_RESTRICTED", responseReason(t, recorder))
}

func responseReason(t *testing.T, recorder *httptest.ResponseRecorder) string {
	t.Helper()
	var body struct {
		Reason string `json:"reason"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &body))
	return body.Reason
}
