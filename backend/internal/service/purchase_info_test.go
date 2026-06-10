//go:build unit

package service

import (
	"context"
	"errors"
	"sort"
	"strings"
	"testing"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/stretchr/testify/require"
)

type purchaseInfoRepoStub struct {
	users  map[int64]*User
	cards  map[int64]*PurchaseInfoCard
	nextID int64
}

func newPurchaseInfoRepoStub(users ...*User) *purchaseInfoRepoStub {
	r := &purchaseInfoRepoStub{
		users:  map[int64]*User{},
		cards:  map[int64]*PurchaseInfoCard{},
		nextID: 1,
	}
	for _, user := range users {
		clone := *user
		r.users[user.ID] = &clone
	}
	return r
}

func (r *purchaseInfoRepoStub) GetUserByID(_ context.Context, userID int64) (*User, error) {
	user, ok := r.users[userID]
	if !ok {
		return nil, ErrUserNotFound
	}
	clone := *user
	return &clone, nil
}

func (r *purchaseInfoRepoStub) ListCardsByOwner(_ context.Context, ownerUserID *int64, onlyEnabled bool) ([]PurchaseInfoCard, error) {
	out := []PurchaseInfoCard{}
	for _, card := range r.cards {
		if !sameNullableInt64(card.OwnerUserID, ownerUserID) {
			continue
		}
		if onlyEnabled && !card.Enabled {
			continue
		}
		clone := *card
		out = append(out, clone)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].SortOrder == out[j].SortOrder {
			return out[i].ID < out[j].ID
		}
		return out[i].SortOrder < out[j].SortOrder
	})
	return out, nil
}

func (r *purchaseInfoRepoStub) GetCardByID(_ context.Context, cardID int64) (*PurchaseInfoCard, error) {
	card, ok := r.cards[cardID]
	if !ok {
		return nil, ErrPurchaseInfoCardNotFound
	}
	clone := *card
	return &clone, nil
}

func (r *purchaseInfoRepoStub) CreateCard(_ context.Context, card *PurchaseInfoCard) error {
	if card == nil {
		return nil
	}
	card.ID = r.nextID
	r.nextID++
	now := time.Now()
	card.CreatedAt = now
	card.UpdatedAt = now
	clone := *card
	r.cards[card.ID] = &clone
	return nil
}

func (r *purchaseInfoRepoStub) UpdateCard(_ context.Context, card *PurchaseInfoCard) error {
	if card == nil {
		return nil
	}
	if _, ok := r.cards[card.ID]; !ok {
		return ErrPurchaseInfoCardNotFound
	}
	card.UpdatedAt = time.Now()
	clone := *card
	r.cards[card.ID] = &clone
	return nil
}

func (r *purchaseInfoRepoStub) DeleteCard(_ context.Context, cardID int64) error {
	if _, ok := r.cards[cardID]; !ok {
		return ErrPurchaseInfoCardNotFound
	}
	delete(r.cards, cardID)
	return nil
}

func sameNullableInt64(a, b *int64) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return *a == *b
}

func TestPurchaseInfoVisibleCardsUseRootForAdminDirectUser(t *testing.T) {
	rootID := int64(1)
	userID := int64(2)
	repo := newPurchaseInfoRepoStub(
		&User{ID: rootID, Role: RoleAdmin, Status: StatusActive},
		&User{ID: userID, Role: RoleUser, ParentUserID: &rootID, Status: StatusActive},
	)
	repo.cards[1] = &PurchaseInfoCard{ID: 1, OwnerUserID: nil, Title: "Admin shop", Enabled: true, SortOrder: 2}
	repo.cards[2] = &PurchaseInfoCard{ID: 2, OwnerUserID: nil, Title: "Disabled", Enabled: false, SortOrder: 1}
	svc := NewPurchaseInfoService(repo)

	cards, err := svc.ListVisibleCards(context.Background(), userID)

	require.NoError(t, err)
	require.Len(t, cards, 1)
	require.Equal(t, "Admin shop", cards[0].Title)
}

func TestPurchaseInfoAgentSeesUpstreamCardsAndManagesOwnCards(t *testing.T) {
	rootID := int64(1)
	agentID := int64(2)
	childID := int64(3)
	repo := newPurchaseInfoRepoStub(
		&User{ID: rootID, Role: RoleAdmin, Status: StatusActive},
		&User{ID: agentID, Role: RoleAgentLevel1, ParentUserID: &rootID, Status: StatusActive},
		&User{ID: childID, Role: RoleUser, ParentUserID: &agentID, Status: StatusActive},
	)
	repo.cards[1] = &PurchaseInfoCard{ID: 1, OwnerUserID: nil, Title: "Admin wholesale", Enabled: true}
	repo.cards[2] = &PurchaseInfoCard{ID: 2, OwnerUserID: &agentID, Title: "Agent retail", Enabled: true}
	svc := NewPurchaseInfoService(repo)

	agentVisible, err := svc.ListVisibleCards(context.Background(), agentID)
	require.NoError(t, err)
	require.Len(t, agentVisible, 1)
	require.Equal(t, "Admin wholesale", agentVisible[0].Title)

	childVisible, err := svc.ListVisibleCards(context.Background(), childID)
	require.NoError(t, err)
	require.Len(t, childVisible, 1)
	require.Equal(t, "Agent retail", childVisible[0].Title)

	managed, err := svc.ListManagedCards(context.Background(), agentID)
	require.NoError(t, err)
	require.Len(t, managed, 1)
	require.Equal(t, "Agent retail", managed[0].Title)
}

func TestPurchaseInfoEmployeeCannotViewOrManage(t *testing.T) {
	enterpriseID := int64(10)
	employeeID := int64(11)
	repo := newPurchaseInfoRepoStub(
		&User{ID: enterpriseID, Role: RoleEnterprise, Status: StatusActive},
		&User{ID: employeeID, Role: RoleEmployee, ParentUserID: &enterpriseID, Status: StatusActive},
	)
	svc := NewPurchaseInfoService(repo)

	_, err := svc.ListVisibleCards(context.Background(), employeeID)
	require.Error(t, err)
	require.True(t, infraerrors.IsForbidden(err))

	_, err = svc.ListManagedCards(context.Background(), employeeID)
	require.Error(t, err)
	require.True(t, infraerrors.IsForbidden(err))
}

func TestPurchaseInfoCreateRejectsNonHTTPSURL(t *testing.T) {
	adminID := int64(1)
	repo := newPurchaseInfoRepoStub(&User{ID: adminID, Role: RoleAdmin, Status: StatusActive})
	svc := NewPurchaseInfoService(repo)

	_, err := svc.CreateCard(context.Background(), adminID, PurchaseInfoCardInput{
		Title:       "Bad URL",
		PurchaseURL: "ftp://example.com/code",
		Enabled:     true,
	})

	require.Error(t, err)
	require.True(t, infraerrors.IsBadRequest(err))
}

func TestPurchaseInfoAgentCannotUpdateAnotherOwnersCard(t *testing.T) {
	agentID := int64(2)
	otherID := int64(3)
	repo := newPurchaseInfoRepoStub(
		&User{ID: agentID, Role: RoleAgentLevel1, Status: StatusActive},
		&User{ID: otherID, Role: RoleAgentLevel1, Status: StatusActive},
	)
	repo.cards[1] = &PurchaseInfoCard{ID: 1, OwnerUserID: &otherID, Title: "Other", Enabled: true}
	svc := NewPurchaseInfoService(repo)

	_, err := svc.UpdateCard(context.Background(), agentID, 1, PurchaseInfoCardInput{Title: "Take over", Enabled: true})

	require.Error(t, err)
	require.True(t, infraerrors.IsForbidden(err))
}

func TestPurchaseInfoCreateTrimsInput(t *testing.T) {
	adminID := int64(1)
	repo := newPurchaseInfoRepoStub(&User{ID: adminID, Role: RoleAdmin, Status: StatusActive})
	svc := NewPurchaseInfoService(repo)

	card, err := svc.CreateCard(context.Background(), adminID, PurchaseInfoCardInput{
		Title:       "  Admin service  ",
		Description: "  buy here  ",
		PurchaseURL: " https://example.com/buy ",
		Contact:     " support ",
		Enabled:     true,
		SortOrder:   7,
	})

	require.NoError(t, err)
	require.Equal(t, "Admin service", card.Title)
	require.Equal(t, "buy here", card.Description)
	require.Equal(t, "https://example.com/buy", card.PurchaseURL)
	require.Equal(t, "support", card.Contact)
	require.Nil(t, card.OwnerUserID)
	require.Equal(t, 7, card.SortOrder)
}

func TestPurchaseInfoTitleRequired(t *testing.T) {
	adminID := int64(1)
	repo := newPurchaseInfoRepoStub(&User{ID: adminID, Role: RoleAdmin, Status: StatusActive})
	svc := NewPurchaseInfoService(repo)

	_, err := svc.CreateCard(context.Background(), adminID, PurchaseInfoCardInput{Title: strings.Repeat(" ", 3)})

	require.Error(t, err)
	require.True(t, infraerrors.IsBadRequest(err))
}

func TestPurchaseInfoDeleteRequiresOwnership(t *testing.T) {
	agentID := int64(2)
	otherID := int64(3)
	repo := newPurchaseInfoRepoStub(
		&User{ID: agentID, Role: RoleAgentLevel1, Status: StatusActive},
		&User{ID: otherID, Role: RoleAgentLevel1, Status: StatusActive},
	)
	repo.cards[1] = &PurchaseInfoCard{ID: 1, OwnerUserID: &otherID, Title: "Other", Enabled: true}
	svc := NewPurchaseInfoService(repo)

	err := svc.DeleteCard(context.Background(), agentID, 1)

	require.Error(t, err)
	require.True(t, infraerrors.IsForbidden(err))
}

func TestPurchaseInfoFallbackToRootWhenUserHasNoParent(t *testing.T) {
	userID := int64(2)
	repo := newPurchaseInfoRepoStub(&User{ID: userID, Role: RoleUser, Status: StatusActive})
	repo.cards[1] = &PurchaseInfoCard{ID: 1, OwnerUserID: nil, Title: "Root", Enabled: true}
	svc := NewPurchaseInfoService(repo)

	cards, err := svc.ListVisibleCards(context.Background(), userID)

	require.NoError(t, err)
	require.Len(t, cards, 1)
	require.Equal(t, "Root", cards[0].Title)
}

func TestPurchaseInfoFallbackToRootWhenParentMissing(t *testing.T) {
	parentID := int64(99)
	userID := int64(2)
	repo := newPurchaseInfoRepoStub(&User{ID: userID, Role: RoleUser, ParentUserID: &parentID, Status: StatusActive})
	repo.cards[1] = &PurchaseInfoCard{ID: 1, OwnerUserID: nil, Title: "Root", Enabled: true}
	svc := NewPurchaseInfoService(repo)

	cards, err := svc.ListVisibleCards(context.Background(), userID)

	require.NoError(t, err)
	require.Len(t, cards, 1)
	require.Equal(t, "Root", cards[0].Title)
}

func TestPurchaseInfoRepositoryErrorsPropagate(t *testing.T) {
	errBoom := errors.New("boom")
	repo := &purchaseInfoRepoErrorStub{err: errBoom}
	svc := NewPurchaseInfoService(repo)

	_, err := svc.ListVisibleCards(context.Background(), 1)

	require.ErrorIs(t, err, errBoom)
}

type purchaseInfoRepoErrorStub struct {
	err error
}

func (r *purchaseInfoRepoErrorStub) GetUserByID(context.Context, int64) (*User, error) {
	return nil, r.err
}
func (r *purchaseInfoRepoErrorStub) ListCardsByOwner(context.Context, *int64, bool) ([]PurchaseInfoCard, error) {
	return nil, r.err
}
func (r *purchaseInfoRepoErrorStub) GetCardByID(context.Context, int64) (*PurchaseInfoCard, error) {
	return nil, r.err
}
func (r *purchaseInfoRepoErrorStub) CreateCard(context.Context, *PurchaseInfoCard) error {
	return r.err
}
func (r *purchaseInfoRepoErrorStub) UpdateCard(context.Context, *PurchaseInfoCard) error {
	return r.err
}
func (r *purchaseInfoRepoErrorStub) DeleteCard(context.Context, int64) error {
	return r.err
}
