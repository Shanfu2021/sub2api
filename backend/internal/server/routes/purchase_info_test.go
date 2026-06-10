//go:build unit

package routes

import (
	"net/http"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/handler"
	servermiddleware "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestPurchaseInfoRoutesAreRegisteredUnderAuthenticatedUserRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	v1 := router.Group("/api/v1")
	RegisterUserRoutes(
		v1,
		&handler.Handlers{
			AgentManagement:      handler.NewAgentManagementHandler(&service.AgentManagementService{}),
			EnterpriseManagement: handler.NewEnterpriseManagementHandler(&service.EnterpriseManagementService{}),
			PurchaseInfo:         handler.NewPurchaseInfoHandler(&service.PurchaseInfoService{}),
		},
		servermiddleware.JWTAuthMiddleware(func(c *gin.Context) {
			c.Set(string(servermiddleware.ContextKeyUser), servermiddleware.AuthSubject{UserID: 42})
			c.Next()
		}),
		nil,
	)

	registered := make(map[string]struct{})
	for _, route := range router.Routes() {
		registered[route.Method+" "+route.Path] = struct{}{}
	}

	for _, tc := range []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/api/v1/purchase-info"},
		{http.MethodGet, "/api/v1/purchase-info/manage"},
		{http.MethodPost, "/api/v1/purchase-info/manage"},
		{http.MethodPut, "/api/v1/purchase-info/manage/:id"},
		{http.MethodDelete, "/api/v1/purchase-info/manage/:id"},
	} {
		_, ok := registered[tc.method+" "+tc.path]
		require.True(t, ok, "%s %s", tc.method, tc.path)
	}
}
