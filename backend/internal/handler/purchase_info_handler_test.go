//go:build unit

package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type fakePurchaseInfoService struct {
	listVisibleActorID int64
	listManagedActorID int64
	createActorID      int64
	createInput        service.PurchaseInfoCardInput
	updateActorID      int64
	updateCardID       int64
	updateInput        service.PurchaseInfoCardInput
	deleteActorID      int64
	deleteCardID       int64
	err                error
}

func (s *fakePurchaseInfoService) ListVisibleCards(_ context.Context, actorID int64) ([]service.PurchaseInfoCard, error) {
	s.listVisibleActorID = actorID
	if s.err != nil {
		return nil, s.err
	}
	return []service.PurchaseInfoCard{{ID: 1, Title: "Visible", PurchaseURL: "https://example.com/buy", Enabled: true}}, nil
}

func (s *fakePurchaseInfoService) ListManagedCards(_ context.Context, actorID int64) ([]service.PurchaseInfoCard, error) {
	s.listManagedActorID = actorID
	if s.err != nil {
		return nil, s.err
	}
	return []service.PurchaseInfoCard{{ID: 2, Title: "Managed", Enabled: false}}, nil
}

func (s *fakePurchaseInfoService) CreateCard(_ context.Context, actorID int64, input service.PurchaseInfoCardInput) (*service.PurchaseInfoCard, error) {
	s.createActorID = actorID
	s.createInput = input
	if s.err != nil {
		return nil, s.err
	}
	return &service.PurchaseInfoCard{ID: 3, Title: input.Title, PurchaseURL: input.PurchaseURL, Enabled: input.Enabled}, nil
}

func (s *fakePurchaseInfoService) UpdateCard(_ context.Context, actorID int64, cardID int64, input service.PurchaseInfoCardInput) (*service.PurchaseInfoCard, error) {
	s.updateActorID = actorID
	s.updateCardID = cardID
	s.updateInput = input
	if s.err != nil {
		return nil, s.err
	}
	return &service.PurchaseInfoCard{ID: cardID, Title: input.Title, PurchaseURL: input.PurchaseURL, Enabled: input.Enabled}, nil
}

func (s *fakePurchaseInfoService) DeleteCard(_ context.Context, actorID int64, cardID int64) error {
	s.deleteActorID = actorID
	s.deleteCardID = cardID
	return s.err
}

func newPurchaseInfoHandlerTestRouter(svc *fakePurchaseInfoService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	h := newPurchaseInfoHandler(svc)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: 42})
		c.Next()
	})
	r.GET("/purchase-info", h.ListVisible)
	r.GET("/purchase-info/manage", h.ListManaged)
	r.POST("/purchase-info/manage", h.Create)
	r.PUT("/purchase-info/manage/:id", h.Update)
	r.DELETE("/purchase-info/manage/:id", h.Delete)
	return r
}

func TestPurchaseInfoHandlerListsVisibleCards(t *testing.T) {
	svc := &fakePurchaseInfoService{}
	router := newPurchaseInfoHandlerTestRouter(svc)

	req := httptest.NewRequest(http.MethodGet, "/purchase-info", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, int64(42), svc.listVisibleActorID)
	require.Contains(t, rec.Body.String(), `"title":"Visible"`)
	require.Contains(t, rec.Body.String(), `"purchase_url":"https://example.com/buy"`)
}

func TestPurchaseInfoHandlerListsManagedCards(t *testing.T) {
	svc := &fakePurchaseInfoService{}
	router := newPurchaseInfoHandlerTestRouter(svc)

	req := httptest.NewRequest(http.MethodGet, "/purchase-info/manage", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, int64(42), svc.listManagedActorID)
	require.Contains(t, rec.Body.String(), `"title":"Managed"`)
}

func TestPurchaseInfoHandlerCreatesCard(t *testing.T) {
	svc := &fakePurchaseInfoService{}
	router := newPurchaseInfoHandlerTestRouter(svc)

	req := httptest.NewRequest(http.MethodPost, "/purchase-info/manage", strings.NewReader(`{
		"title": "Shop",
		"description": "Buy codes here",
		"purchase_url": "https://example.com/shop",
		"contact": "support",
		"sort_order": 3,
		"enabled": true
	}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, int64(42), svc.createActorID)
	require.Equal(t, service.PurchaseInfoCardInput{
		Title:       "Shop",
		Description: "Buy codes here",
		PurchaseURL: "https://example.com/shop",
		Contact:     "support",
		SortOrder:   3,
		Enabled:     true,
	}, svc.createInput)
}

func TestPurchaseInfoHandlerUpdatesAndDeletesCard(t *testing.T) {
	svc := &fakePurchaseInfoService{}
	router := newPurchaseInfoHandlerTestRouter(svc)

	req := httptest.NewRequest(http.MethodPut, "/purchase-info/manage/9", strings.NewReader(`{
		"title": "Updated",
		"purchase_url": "https://example.com/new",
		"enabled": true
	}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, int64(42), svc.updateActorID)
	require.Equal(t, int64(9), svc.updateCardID)
	require.Equal(t, "Updated", svc.updateInput.Title)

	req = httptest.NewRequest(http.MethodDelete, "/purchase-info/manage/9", nil)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, int64(42), svc.deleteActorID)
	require.Equal(t, int64(9), svc.deleteCardID)
}

func TestPurchaseInfoHandlerReturnsForbiddenFromService(t *testing.T) {
	svc := &fakePurchaseInfoService{err: service.ErrPurchaseInfoForbidden}
	router := newPurchaseInfoHandlerTestRouter(svc)

	req := httptest.NewRequest(http.MethodGet, "/purchase-info", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusForbidden, rec.Code)
}
