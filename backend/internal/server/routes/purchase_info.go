package routes

import (
	"github.com/Wei-Shaw/sub2api/internal/handler"

	"github.com/gin-gonic/gin"
)

func RegisterPurchaseInfoRoutes(authenticated *gin.RouterGroup, h *handler.Handlers) {
	purchaseInfo := authenticated.Group("/purchase-info")
	{
		purchaseInfo.GET("", h.PurchaseInfo.ListVisible)
		purchaseInfo.GET("/manage", h.PurchaseInfo.ListManaged)
		purchaseInfo.POST("/manage", h.PurchaseInfo.Create)
		purchaseInfo.PUT("/manage/:id", h.PurchaseInfo.Update)
		purchaseInfo.DELETE("/manage/:id", h.PurchaseInfo.Delete)
	}
}
