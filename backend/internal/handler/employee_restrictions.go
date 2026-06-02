package handler

import (
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

func abortIfEmployeeFeatureRestricted(c *gin.Context) bool {
	role, ok := middleware2.GetUserRoleFromContext(c)
	if !ok || !service.IsEmployeeRole(role) {
		return false
	}
	response.ErrorFrom(c, service.ErrEmployeeFeatureRestricted)
	return true
}
