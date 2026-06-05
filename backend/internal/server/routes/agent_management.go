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
		agentManagement.GET("/direct-users/groups/assigned", h.AgentManagement.ListDirectUsersWithGroup)
		agentManagement.GET("/direct-users/groups/unassigned", h.AgentManagement.ListDirectUsersWithoutGroup)
		agentManagement.PUT("/direct-users/groups/batch", h.AgentManagement.SetDirectUsersGroupDelegationsBatch)
		agentManagement.PUT("/direct-users/groups/existing", h.AgentManagement.UpdateDirectUsersExistingGroupDelegations)
		agentManagement.POST("/direct-users/groups/reclaim", h.AgentManagement.ReclaimDirectUsersGroupDelegations)
		agentManagement.GET("/direct-agents", h.AgentManagement.ListDirectAgents)
		agentManagement.GET("/direct-agents/groups/assigned", h.AgentManagement.ListDirectAgentsWithGroup)
		agentManagement.GET("/direct-agents/groups/unassigned", h.AgentManagement.ListDirectAgentsWithoutGroup)
		agentManagement.PUT("/direct-agents/groups/batch", h.AgentManagement.SetDirectAgentsGroupDelegationsBatch)
		agentManagement.PUT("/direct-agents/groups/existing", h.AgentManagement.UpdateDirectAgentsExistingGroupDelegations)
		agentManagement.POST("/direct-agents/groups/reclaim", h.AgentManagement.ReclaimDirectAgentsGroupDelegations)
		agentManagement.GET("/direct-enterprises", h.AgentManagement.ListDirectEnterprises)
		agentManagement.GET("/direct-enterprises/groups/assigned", h.AgentManagement.ListDirectEnterprisesWithGroup)
		agentManagement.GET("/direct-enterprises/groups/unassigned", h.AgentManagement.ListDirectEnterprisesWithoutGroup)
		agentManagement.PUT("/direct-enterprises/groups/batch", h.AgentManagement.SetDirectEnterprisesGroupDelegationsBatch)
		agentManagement.PUT("/direct-enterprises/groups/existing", h.AgentManagement.UpdateDirectEnterprisesExistingGroupDelegations)
		agentManagement.POST("/direct-enterprises/groups/reclaim", h.AgentManagement.ReclaimDirectEnterprisesGroupDelegations)
		agentManagement.GET("/admin-agent-tree", h.AgentManagement.AdminAgentTree)
		agentManagement.GET("/structure", h.AgentManagement.SubordinateStructure)
		agentManagement.GET("/usage", h.AgentManagement.ListUsage)
		agentManagement.GET("/usage/stats", h.AgentManagement.UsageStats)
		agentManagement.GET("/usage/users", h.AgentManagement.ListUsageUsers)
		agentManagement.PUT("/children/:id/allocation", h.AgentManagement.UpdateAllocation)
		agentManagement.PUT("/children/:id/agent-income", h.AgentManagement.SetAgentIncome)
		agentManagement.PUT("/invite-defaults", h.AgentManagement.UpdateInviteDefaults)
		agentManagement.POST("/children/:id/upgrade", h.AgentManagement.UpgradeDirectUser)
		agentManagement.DELETE("/children/:id", h.AgentManagement.DeleteDirectChild)
		agentManagement.GET("/groups", h.AgentManagement.ListMyGroups)
		agentManagement.GET("/invite-default-groups", h.AgentManagement.ListInviteGroupDefaultOptions)
		agentManagement.PUT("/invite-default-groups/batch", h.AgentManagement.SetInviteGroupDefaultsBatch)
		agentManagement.PUT("/invite-default-groups/:group_id", h.AgentManagement.SetInviteGroupDefault)
		agentManagement.DELETE("/invite-default-groups/:group_id", h.AgentManagement.RemoveInviteGroupDefault)
		agentManagement.GET("/children/:id/groups", h.AgentManagement.ListChildGroupDelegationOptions)
		agentManagement.PUT("/children/:id/groups/batch", h.AgentManagement.SetChildGroupDelegationsBatch)
		agentManagement.PUT("/children/:id/groups/:group_id", h.AgentManagement.SetChildGroupDelegation)
		agentManagement.DELETE("/children/:id/groups/:group_id", h.AgentManagement.RemoveChildGroupDelegation)
	}
}
