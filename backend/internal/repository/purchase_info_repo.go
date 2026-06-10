package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

type PurchaseInfoRepository struct {
	db *sql.DB
}

func NewPurchaseInfoRepository(db *sql.DB) service.PurchaseInfoRepository {
	return &PurchaseInfoRepository{db: db}
}

func (r *PurchaseInfoRepository) GetUserByID(ctx context.Context, userID int64) (*service.User, error) {
	var user service.User
	var parent sql.NullInt64
	err := r.db.QueryRowContext(ctx, `
		SELECT id, email, role, parent_user_id, status
		FROM users
		WHERE id = $1 AND deleted_at IS NULL
	`, userID).Scan(&user.ID, &user.Email, &user.Role, &parent, &user.Status)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, service.ErrUserNotFound
		}
		return nil, err
	}
	if parent.Valid {
		user.ParentUserID = &parent.Int64
	}
	return &user, nil
}

func (r *PurchaseInfoRepository) ListCardsByOwner(ctx context.Context, ownerUserID *int64, onlyEnabled bool) ([]service.PurchaseInfoCard, error) {
	whereOwner := "owner_user_id IS NULL"
	args := []any{}
	if ownerUserID != nil {
		whereOwner = "owner_user_id = $1"
		args = append(args, *ownerUserID)
	}
	enabledClause := ""
	if onlyEnabled {
		enabledClause = " AND enabled = TRUE"
	}
	query := `
		SELECT id, owner_user_id, title, description, purchase_url, contact, sort_order, enabled, created_at, updated_at
		FROM purchase_info_cards
		WHERE deleted_at IS NULL AND ` + whereOwner + enabledClause + `
		ORDER BY sort_order ASC, id ASC
	`
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	cards := []service.PurchaseInfoCard{}
	for rows.Next() {
		card, err := scanPurchaseInfoCard(rows)
		if err != nil {
			return nil, err
		}
		cards = append(cards, *card)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return cards, nil
}

func (r *PurchaseInfoRepository) GetCardByID(ctx context.Context, cardID int64) (*service.PurchaseInfoCard, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, owner_user_id, title, description, purchase_url, contact, sort_order, enabled, created_at, updated_at
		FROM purchase_info_cards
		WHERE id = $1 AND deleted_at IS NULL
	`, cardID)
	card, err := scanPurchaseInfoCard(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, service.ErrPurchaseInfoCardNotFound
		}
		return nil, err
	}
	return card, nil
}

func (r *PurchaseInfoRepository) CreateCard(ctx context.Context, card *service.PurchaseInfoCard) error {
	var owner sql.NullInt64
	if card.OwnerUserID != nil {
		owner.Valid = true
		owner.Int64 = *card.OwnerUserID
	}
	row := r.db.QueryRowContext(ctx, `
		INSERT INTO purchase_info_cards (owner_user_id, title, description, purchase_url, contact, sort_order, enabled)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at, updated_at
	`, owner, card.Title, card.Description, card.PurchaseURL, card.Contact, card.SortOrder, card.Enabled)
	return row.Scan(&card.ID, &card.CreatedAt, &card.UpdatedAt)
}

func (r *PurchaseInfoRepository) UpdateCard(ctx context.Context, card *service.PurchaseInfoCard) error {
	var owner sql.NullInt64
	if card.OwnerUserID != nil {
		owner.Valid = true
		owner.Int64 = *card.OwnerUserID
	}
	err := r.db.QueryRowContext(ctx, `
		UPDATE purchase_info_cards
		SET owner_user_id = $2,
		    title = $3,
		    description = $4,
		    purchase_url = $5,
		    contact = $6,
		    sort_order = $7,
		    enabled = $8,
		    updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL
		RETURNING updated_at
	`, card.ID, owner, card.Title, card.Description, card.PurchaseURL, card.Contact, card.SortOrder, card.Enabled).Scan(&card.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return service.ErrPurchaseInfoCardNotFound
		}
		return err
	}
	return nil
}

func (r *PurchaseInfoRepository) DeleteCard(ctx context.Context, cardID int64) error {
	result, err := r.db.ExecContext(ctx, `
		UPDATE purchase_info_cards
		SET deleted_at = NOW(), updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL
	`, cardID)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return service.ErrPurchaseInfoCardNotFound
	}
	return nil
}

type purchaseInfoScanner interface {
	Scan(dest ...any) error
}

func scanPurchaseInfoCard(row purchaseInfoScanner) (*service.PurchaseInfoCard, error) {
	var card service.PurchaseInfoCard
	var owner sql.NullInt64
	var createdAt time.Time
	var updatedAt time.Time
	if err := row.Scan(
		&card.ID,
		&owner,
		&card.Title,
		&card.Description,
		&card.PurchaseURL,
		&card.Contact,
		&card.SortOrder,
		&card.Enabled,
		&createdAt,
		&updatedAt,
	); err != nil {
		return nil, err
	}
	if owner.Valid {
		card.OwnerUserID = &owner.Int64
	}
	card.CreatedAt = createdAt
	card.UpdatedAt = updatedAt
	return &card, nil
}
