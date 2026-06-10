package service

import (
	"context"
	"errors"
	"testing"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/payment"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/stretchr/testify/require"
)

type employeeRestrictionUserRepoStub struct {
	UserRepository
	user      *User
	getByID   int
	updatedBy int
}

func (r *employeeRestrictionUserRepoStub) GetByID(context.Context, int64) (*User, error) {
	r.getByID++
	if r.user == nil {
		return nil, errors.New("missing user")
	}
	cloned := *r.user
	return &cloned, nil
}

func (r *employeeRestrictionUserRepoStub) UpdateBalance(context.Context, int64, float64) error {
	r.updatedBy++
	return nil
}

type employeeRestrictionRedeemRepoStub struct {
	code     *RedeemCode
	useCalls int
}

func (r *employeeRestrictionRedeemRepoStub) Create(context.Context, *RedeemCode) error {
	return nil
}
func (r *employeeRestrictionRedeemRepoStub) CreateBatch(context.Context, []RedeemCode) error {
	return nil
}
func (r *employeeRestrictionRedeemRepoStub) GetByID(context.Context, int64) (*RedeemCode, error) {
	return r.code, nil
}
func (r *employeeRestrictionRedeemRepoStub) GetByCode(context.Context, string) (*RedeemCode, error) {
	if r.code == nil {
		return nil, ErrRedeemCodeNotFound
	}
	cloned := *r.code
	return &cloned, nil
}
func (r *employeeRestrictionRedeemRepoStub) Update(context.Context, *RedeemCode) error {
	return nil
}
func (r *employeeRestrictionRedeemRepoStub) BatchUpdate(context.Context, []int64, RedeemCodeBatchUpdateFields) (int64, error) {
	return 0, nil
}
func (r *employeeRestrictionRedeemRepoStub) Delete(context.Context, int64) error {
	return nil
}
func (r *employeeRestrictionRedeemRepoStub) Use(context.Context, int64, int64) error {
	r.useCalls++
	return nil
}
func (r *employeeRestrictionRedeemRepoStub) List(context.Context, pagination.PaginationParams) ([]RedeemCode, *pagination.PaginationResult, error) {
	return nil, nil, nil
}
func (r *employeeRestrictionRedeemRepoStub) ListWithFilters(context.Context, pagination.PaginationParams, string, string, string) ([]RedeemCode, *pagination.PaginationResult, error) {
	return nil, nil, nil
}
func (r *employeeRestrictionRedeemRepoStub) ListByUser(context.Context, int64, int) ([]RedeemCode, error) {
	return nil, nil
}
func (r *employeeRestrictionRedeemRepoStub) ListByUserPaginated(context.Context, int64, pagination.PaginationParams, string) ([]RedeemCode, *pagination.PaginationResult, error) {
	return nil, nil, nil
}
func (r *employeeRestrictionRedeemRepoStub) SumPositiveBalanceByUser(context.Context, int64) (float64, error) {
	return 0, nil
}

type employeeRestrictionSettingRepoStub struct {
	values map[string]string
}

func (r *employeeRestrictionSettingRepoStub) Get(context.Context, string) (*Setting, error) {
	return nil, nil
}
func (r *employeeRestrictionSettingRepoStub) GetValue(_ context.Context, key string) (string, error) {
	return r.values[key], nil
}
func (r *employeeRestrictionSettingRepoStub) Set(context.Context, string, string) error {
	return nil
}
func (r *employeeRestrictionSettingRepoStub) GetMultiple(_ context.Context, keys []string) (map[string]string, error) {
	out := make(map[string]string, len(keys))
	for _, key := range keys {
		out[key] = r.values[key]
	}
	return out, nil
}
func (r *employeeRestrictionSettingRepoStub) SetMultiple(context.Context, map[string]string) error {
	return nil
}
func (r *employeeRestrictionSettingRepoStub) GetAll(context.Context) (map[string]string, error) {
	return r.values, nil
}
func (r *employeeRestrictionSettingRepoStub) GetAllGrouped(context.Context) (map[string]map[string]string, error) {
	return nil, nil
}
func (r *employeeRestrictionSettingRepoStub) Delete(context.Context, string) error {
	return nil
}

type employeeRestrictionLoadBalancerStub struct {
	selectCalls int
}

func (lb *employeeRestrictionLoadBalancerStub) GetInstanceConfig(context.Context, int64) (map[string]string, error) {
	return nil, nil
}
func (lb *employeeRestrictionLoadBalancerStub) SelectInstance(context.Context, string, payment.PaymentType, payment.Strategy, float64) (*payment.InstanceSelection, error) {
	lb.selectCalls++
	return nil, errors.New("unexpected payment selection")
}

func TestEmployeeCannotRedeemBeforeCodeUse(t *testing.T) {
	redeemRepo := &employeeRestrictionRedeemRepoStub{
		code: &RedeemCode{
			ID:     123,
			Code:   "EMP-REDEEM",
			Type:   RedeemTypeBalance,
			Value:  10,
			Status: StatusUnused,
		},
	}
	userRepo := &employeeRestrictionUserRepoStub{
		user: &User{ID: 42, Role: RoleEmployee, Status: StatusActive},
	}
	svc := NewRedeemService(redeemRepo, userRepo, nil, nil, nil, nil, nil, nil)

	_, err := svc.Redeem(context.Background(), 42, "EMP-REDEEM")

	require.ErrorIs(t, err, ErrEmployeeFeatureRestricted)
	require.Equal(t, 0, redeemRepo.useCalls, "employee redeem must be blocked before Use consumes the code")
	require.Equal(t, 0, userRepo.updatedBy)
}

func TestEmployeeCannotCreatePaymentOrder(t *testing.T) {
	userRepo := &employeeRestrictionUserRepoStub{
		user: &User{ID: 42, Role: RoleEmployee, Status: StatusActive},
	}
	lb := &employeeRestrictionLoadBalancerStub{}
	configSvc := NewPaymentConfigService(nil, &employeeRestrictionSettingRepoStub{
		values: map[string]string{
			SettingPaymentEnabled:      "true",
			SettingEnabledPaymentTypes: payment.TypeAlipay,
		},
	}, nil)
	svc := NewPaymentService(&dbent.Client{}, payment.NewRegistry(), lb, nil, nil, configSvc, userRepo, nil, nil)

	_, err := svc.CreateOrder(context.Background(), CreateOrderRequest{
		UserID:      42,
		Amount:      20,
		PaymentType: payment.TypeAlipay,
	})

	require.ErrorIs(t, err, ErrEmployeeFeatureRestricted)
	require.Equal(t, 1, userRepo.getByID)
	require.Equal(t, 0, lb.selectCalls, "employee payment creation must stop before provider selection")
}

func TestEmployeeFeatureRestrictedErrorShape(t *testing.T) {
	require.True(t, infraerrors.IsForbidden(ErrEmployeeFeatureRestricted))
	require.Equal(t, "EMPLOYEE_FEATURE_RESTRICTED", infraerrors.Reason(ErrEmployeeFeatureRestricted))
	require.True(t, IsEmployeeRole(RoleEmployee))
	require.False(t, IsEmployeeRole(RoleUser))
}
