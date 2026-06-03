package service

import (
	"context"
	"errors"
	"net/url"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

var (
	ErrPurchaseInfoForbidden    = infraerrors.Forbidden("PURCHASE_INFO_FORBIDDEN", "purchase info is not available for this user")
	ErrPurchaseInfoManageDenied = infraerrors.Forbidden("PURCHASE_INFO_MANAGE_FORBIDDEN", "purchase info management is not allowed")
	ErrPurchaseInfoInvalidInput = infraerrors.BadRequest("PURCHASE_INFO_INVALID_INPUT", "invalid purchase info input")
	ErrPurchaseInfoCardNotFound = infraerrors.NotFound("PURCHASE_INFO_CARD_NOT_FOUND", "purchase info card not found")
)

type PurchaseInfoCard struct {
	ID          int64
	OwnerUserID *int64
	Title       string
	Description string
	PurchaseURL string
	Contact     string
	SortOrder   int
	Enabled     bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type PurchaseInfoCardInput struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	PurchaseURL string `json:"purchase_url"`
	Contact     string `json:"contact"`
	SortOrder   int    `json:"sort_order"`
	Enabled     bool   `json:"enabled"`
}

type PurchaseInfoRepository interface {
	GetUserByID(ctx context.Context, userID int64) (*User, error)
	ListCardsByOwner(ctx context.Context, ownerUserID *int64, onlyEnabled bool) ([]PurchaseInfoCard, error)
	GetCardByID(ctx context.Context, cardID int64) (*PurchaseInfoCard, error)
	CreateCard(ctx context.Context, card *PurchaseInfoCard) error
	UpdateCard(ctx context.Context, card *PurchaseInfoCard) error
	DeleteCard(ctx context.Context, cardID int64) error
}

type PurchaseInfoService struct {
	repo PurchaseInfoRepository
}

func NewPurchaseInfoService(repo PurchaseInfoRepository) *PurchaseInfoService {
	return &PurchaseInfoService{repo: repo}
}

func (s *PurchaseInfoService) ListVisibleCards(ctx context.Context, actorID int64) ([]PurchaseInfoCard, error) {
	actor, err := s.requireActor(ctx, actorID)
	if err != nil {
		return nil, err
	}
	if actor.Role == RoleEmployee {
		return nil, ErrPurchaseInfoForbidden
	}
	ownerID, err := s.visibleOwnerID(ctx, actor)
	if err != nil {
		return nil, err
	}
	return s.repo.ListCardsByOwner(ctx, ownerID, true)
}

func (s *PurchaseInfoService) ListManagedCards(ctx context.Context, actorID int64) ([]PurchaseInfoCard, error) {
	ownerID, err := s.manageOwnerID(ctx, actorID)
	if err != nil {
		return nil, err
	}
	return s.repo.ListCardsByOwner(ctx, ownerID, false)
}

func (s *PurchaseInfoService) CreateCard(ctx context.Context, actorID int64, input PurchaseInfoCardInput) (*PurchaseInfoCard, error) {
	ownerID, err := s.manageOwnerID(ctx, actorID)
	if err != nil {
		return nil, err
	}
	card, err := buildPurchaseInfoCard(ownerID, input)
	if err != nil {
		return nil, err
	}
	if err := s.repo.CreateCard(ctx, card); err != nil {
		return nil, err
	}
	return card, nil
}

func (s *PurchaseInfoService) UpdateCard(ctx context.Context, actorID int64, cardID int64, input PurchaseInfoCardInput) (*PurchaseInfoCard, error) {
	ownerID, err := s.manageOwnerID(ctx, actorID)
	if err != nil {
		return nil, err
	}
	existing, err := s.repo.GetCardByID(ctx, cardID)
	if err != nil {
		return nil, err
	}
	if !purchaseInfoOwnerEqual(existing.OwnerUserID, ownerID) {
		return nil, ErrPurchaseInfoManageDenied
	}
	card, err := buildPurchaseInfoCard(ownerID, input)
	if err != nil {
		return nil, err
	}
	card.ID = existing.ID
	card.CreatedAt = existing.CreatedAt
	if err := s.repo.UpdateCard(ctx, card); err != nil {
		return nil, err
	}
	return card, nil
}

func (s *PurchaseInfoService) DeleteCard(ctx context.Context, actorID int64, cardID int64) error {
	ownerID, err := s.manageOwnerID(ctx, actorID)
	if err != nil {
		return err
	}
	existing, err := s.repo.GetCardByID(ctx, cardID)
	if err != nil {
		return err
	}
	if !purchaseInfoOwnerEqual(existing.OwnerUserID, ownerID) {
		return ErrPurchaseInfoManageDenied
	}
	return s.repo.DeleteCard(ctx, cardID)
}

func (s *PurchaseInfoService) requireActor(ctx context.Context, actorID int64) (*User, error) {
	if s == nil || s.repo == nil {
		return nil, ErrPurchaseInfoForbidden
	}
	return s.repo.GetUserByID(ctx, actorID)
}

func (s *PurchaseInfoService) visibleOwnerID(ctx context.Context, actor *User) (*int64, error) {
	if actor == nil {
		return nil, ErrPurchaseInfoForbidden
	}
	if actor.Role == RoleAdmin {
		return nil, nil
	}
	if actor.ParentUserID == nil {
		return nil, nil
	}
	parent, err := s.repo.GetUserByID(ctx, *actor.ParentUserID)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return nil, nil
		}
		return nil, err
	}
	if parent.Role == RoleAdmin {
		return nil, nil
	}
	if parent.Role == RoleAgentLevel1 || parent.Role == RoleAgentLevel2 {
		parentID := parent.ID
		return &parentID, nil
	}
	return nil, nil
}

func (s *PurchaseInfoService) manageOwnerID(ctx context.Context, actorID int64) (*int64, error) {
	actor, err := s.requireActor(ctx, actorID)
	if err != nil {
		return nil, err
	}
	switch actor.Role {
	case RoleAdmin:
		return nil, nil
	case RoleAgentLevel1, RoleAgentLevel2:
		id := actor.ID
		return &id, nil
	default:
		return nil, ErrPurchaseInfoManageDenied
	}
}

func buildPurchaseInfoCard(ownerID *int64, input PurchaseInfoCardInput) (*PurchaseInfoCard, error) {
	title := strings.TrimSpace(input.Title)
	description := strings.TrimSpace(input.Description)
	purchaseURL := strings.TrimSpace(input.PurchaseURL)
	contact := strings.TrimSpace(input.Contact)

	if title == "" {
		return nil, ErrPurchaseInfoInvalidInput.WithMetadata(map[string]string{"field": "title"})
	}
	if len(title) > 100 {
		return nil, ErrPurchaseInfoInvalidInput.WithMetadata(map[string]string{"field": "title"})
	}
	if len(description) > 2000 {
		return nil, ErrPurchaseInfoInvalidInput.WithMetadata(map[string]string{"field": "description"})
	}
	if len(purchaseURL) > 1024 {
		return nil, ErrPurchaseInfoInvalidInput.WithMetadata(map[string]string{"field": "purchase_url"})
	}
	if len(contact) > 500 {
		return nil, ErrPurchaseInfoInvalidInput.WithMetadata(map[string]string{"field": "contact"})
	}
	if purchaseURL != "" {
		parsed, err := url.ParseRequestURI(purchaseURL)
		if err != nil || parsed.Scheme == "" || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
			return nil, ErrPurchaseInfoInvalidInput.WithMetadata(map[string]string{"field": "purchase_url"})
		}
	}

	card := &PurchaseInfoCard{
		OwnerUserID: clonePurchaseInfoInt64Ptr(ownerID),
		Title:       title,
		Description: description,
		PurchaseURL: purchaseURL,
		Contact:     contact,
		SortOrder:   input.SortOrder,
		Enabled:     input.Enabled,
	}
	return card, nil
}

func purchaseInfoOwnerEqual(a, b *int64) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return *a == *b
}

func clonePurchaseInfoInt64Ptr(v *int64) *int64 {
	if v == nil {
		return nil
	}
	out := *v
	return &out
}
