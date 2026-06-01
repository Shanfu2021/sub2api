package routes

import (
	"github.com/Wei-Shaw/sub2api/internal/handler"

	"github.com/gin-gonic/gin"
)

func RegisterAgentManagementRoutes(authenticated *gin.RouterGroup, h *handler.Handlers) {
	agentManagement := authenticated.Group("/agent-management")
	{
		agentManagement.GET("/summary", h.AgentManagement.Summary)
		agentManagement.GET("/direct-users", h.AgentManagement.ListDirectUsers)
		agentManagement.GET("/direct-agents", h.AgentManagement.ListDirectAgents)
		agentManagement.GET("/direct-enterprises", h.AgentManagement.ListDirectEnterprises)
		agentManagement.PUT("/children/:id/allocation", h.AgentManagement.UpdateAllocation)
		agentManagement.POST("/children/:id/upgrade", h.AgentManagement.UpgradeDirectUser)
		agentManagement.DELETE("/children/:id", h.AgentManagement.DeleteDirectChild)
		agentManagement.GET("/groups", h.AgentManagement.ListMyGroups)
		agentManagement.PUT("/children/:id/groups/:group_id", h.AgentManagement.SetChildGroupDelegation)
		agentManagement.DELETE("/children/:id/groups/:group_id", h.AgentManagement.RemoveChildGroupDelegation)
	}
}
