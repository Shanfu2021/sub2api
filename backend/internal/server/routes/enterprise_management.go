package routes

import (
	"github.com/Wei-Shaw/sub2api/internal/handler"
	"github.com/gin-gonic/gin"
)

func RegisterEnterpriseManagementRoutes(authenticated *gin.RouterGroup, h *handler.Handlers) {
	enterpriseManagement := authenticated.Group("/enterprise-management")
	{
		enterpriseManagement.GET("/summary", h.EnterpriseManagement.Summary)
		enterpriseManagement.GET("/employees", h.EnterpriseManagement.ListEmployees)
		enterpriseManagement.POST("/employees", h.EnterpriseManagement.CreateEmployee)
		enterpriseManagement.POST("/employees/import", h.EnterpriseManagement.ImportEmployees)
		enterpriseManagement.PUT("/employees/balances/initialize", h.EnterpriseManagement.InitializeEmployeeBalances)
		enterpriseManagement.PUT("/employees/:id/allocation", h.EnterpriseManagement.UpdateEmployeeAllocation)
		enterpriseManagement.DELETE("/employees/:id", h.EnterpriseManagement.DeleteEmployee)
		enterpriseManagement.GET("/groups", h.EnterpriseManagement.ListMyGroups)
		enterpriseManagement.GET("/employees/:id/groups", h.EnterpriseManagement.ListEmployeeGroupOptions)
		enterpriseManagement.PUT("/employees/:id/groups/:group_id", h.EnterpriseManagement.SetEmployeeGroup)
		enterpriseManagement.DELETE("/employees/:id/groups/:group_id", h.EnterpriseManagement.RemoveEmployeeGroup)
	}
}
