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
		agentManagement.POST("/direct-users", h.AgentManagement.CreateDirectUser)
		agentManagement.GET("/direct-agents", h.AgentManagement.ListDirectAgents)
		agentManagement.GET("/direct-enterprises", h.AgentManagement.ListDirectEnterprises)
		agentManagement.PUT("/children/:id/allocation", h.AgentManagement.UpdateAllocation)
		agentManagement.PUT("/invite-defaults", h.AgentManagement.UpdateInviteDefaults)
		agentManagement.POST("/children/:id/upgrade", h.AgentManagement.UpgradeDirectUser)
		agentManagement.DELETE("/children/:id", h.AgentManagement.DeleteDirectChild)
		agentManagement.GET("/groups", h.AgentManagement.ListMyGroups)
		agentManagement.GET("/children/:id/groups", h.AgentManagement.ListChildGroupDelegationOptions)
		agentManagement.PUT("/children/:id/groups/:group_id", h.AgentManagement.SetChildGroupDelegation)
		agentManagement.DELETE("/children/:id/groups/:group_id", h.AgentManagement.RemoveChildGroupDelegation)
	}
}
