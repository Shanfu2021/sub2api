package handler

import (
	"context"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type purchaseInfoService interface {
	ListVisibleCards(ctx context.Context, actorID int64) ([]service.PurchaseInfoCard, error)
	ListManagedCards(ctx context.Context, actorID int64) ([]service.PurchaseInfoCard, error)
	CreateCard(ctx context.Context, actorID int64, input service.PurchaseInfoCardInput) (*service.PurchaseInfoCard, error)
	UpdateCard(ctx context.Context, actorID int64, cardID int64, input service.PurchaseInfoCardInput) (*service.PurchaseInfoCard, error)
	DeleteCard(ctx context.Context, actorID int64, cardID int64) error
}

type PurchaseInfoHandler struct {
	service purchaseInfoService
}

func NewPurchaseInfoHandler(svc *service.PurchaseInfoService) *PurchaseInfoHandler {
	return newPurchaseInfoHandler(svc)
}

func newPurchaseInfoHandler(svc purchaseInfoService) *PurchaseInfoHandler {
	return &PurchaseInfoHandler{service: svc}
}

func (h *PurchaseInfoHandler) ListVisible(c *gin.Context) {
	actorID, ok := currentActorID(c)
	if !ok {
		return
	}
	cards, err := h.service.ListVisibleCards(c.Request.Context(), actorID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, purchaseInfoCardsFromService(cards))
}

func (h *PurchaseInfoHandler) ListManaged(c *gin.Context) {
	actorID, ok := currentActorID(c)
	if !ok {
		return
	}
	cards, err := h.service.ListManagedCards(c.Request.Context(), actorID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, purchaseInfoCardsFromService(cards))
}

func (h *PurchaseInfoHandler) Create(c *gin.Context) {
	actorID, ok := currentActorID(c)
	if !ok {
		return
	}
	var req service.PurchaseInfoCardInput
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	card, err := h.service.CreateCard(c.Request.Context(), actorID, req)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, purchaseInfoCardFromService(card))
}

func (h *PurchaseInfoHandler) Update(c *gin.Context) {
	actorID, ok := currentActorID(c)
	if !ok {
		return
	}
	cardID, ok := parsePositiveID(c, "id", "Invalid card ID")
	if !ok {
		return
	}
	var req service.PurchaseInfoCardInput
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	card, err := h.service.UpdateCard(c.Request.Context(), actorID, cardID, req)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, purchaseInfoCardFromService(card))
}

func (h *PurchaseInfoHandler) Delete(c *gin.Context) {
	actorID, ok := currentActorID(c)
	if !ok {
		return
	}
	cardID, ok := parsePositiveID(c, "id", "Invalid card ID")
	if !ok {
		return
	}
	if err := h.service.DeleteCard(c.Request.Context(), actorID, cardID); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"id": cardID})
}

type purchaseInfoCardResponse struct {
	ID          int64  `json:"id"`
	OwnerUserID *int64 `json:"owner_user_id,omitempty"`
	Title       string `json:"title"`
	Description string `json:"description"`
	PurchaseURL string `json:"purchase_url"`
	Contact     string `json:"contact"`
	SortOrder   int    `json:"sort_order"`
	Enabled     bool   `json:"enabled"`
}

func purchaseInfoCardsFromService(cards []service.PurchaseInfoCard) []purchaseInfoCardResponse {
	out := make([]purchaseInfoCardResponse, 0, len(cards))
	for i := range cards {
		out = append(out, purchaseInfoCardFromService(&cards[i]))
	}
	return out
}

func purchaseInfoCardFromService(card *service.PurchaseInfoCard) purchaseInfoCardResponse {
	if card == nil {
		return purchaseInfoCardResponse{}
	}
	return purchaseInfoCardResponse{
		ID:          card.ID,
		OwnerUserID: card.OwnerUserID,
		Title:       card.Title,
		Description: card.Description,
		PurchaseURL: card.PurchaseURL,
		Contact:     card.Contact,
		SortOrder:   card.SortOrder,
		Enabled:     card.Enabled,
	}
}
