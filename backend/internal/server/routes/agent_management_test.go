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

func TestAgentManagementRoutesAreRegisteredUnderAuthenticatedUserRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	v1 := router.Group("/api/v1")
	RegisterUserRoutes(
		v1,
		&handler.Handlers{AgentManagement: handler.NewAgentManagementHandler(&service.AgentManagementService{})},
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
		{http.MethodGet, "/api/v1/agent-management/summary"},
		{http.MethodGet, "/api/v1/agent-management/direct-users"},
		{http.MethodPost, "/api/v1/agent-management/direct-users"},
		{http.MethodGet, "/api/v1/agent-management/direct-agents"},
		{http.MethodGet, "/api/v1/agent-management/direct-enterprises"},
		{http.MethodPut, "/api/v1/agent-management/children/:id/allocation"},
		{http.MethodPut, "/api/v1/agent-management/invite-defaults"},
		{http.MethodPost, "/api/v1/agent-management/children/:id/upgrade"},
		{http.MethodDelete, "/api/v1/agent-management/children/:id"},
		{http.MethodGet, "/api/v1/agent-management/groups"},
		{http.MethodPut, "/api/v1/agent-management/children/:id/groups/:group_id"},
		{http.MethodDelete, "/api/v1/agent-management/children/:id/groups/:group_id"},
	} {
		_, ok := registered[tc.method+" "+tc.path]
		require.True(t, ok, "%s %s", tc.method, tc.path)
	}
}
