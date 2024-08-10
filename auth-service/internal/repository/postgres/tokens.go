package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"
	"todoservice/auth-service/internal/domain"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type RefreshTokenDB struct {
	db *pgx.Conn
}

func NewRefreshTokenDB(db *pgx.Conn) *RefreshTokenDB {
	return &RefreshTokenDB{
		db: db,
	}
}

func (r *RefreshTokenDB) Create(ctx context.Context, token *domain.RefreshToken) error {
	token.ID = uuid.New()
	token.CreatedAt = time.Now()
	token.UpdatedAt = time.Now()

	query := `INSERT INTO refresh_tokens (id, user_id, refresh_token, expires_at, created_at, updated_at)
              VALUES ($1, $2, $3, $4, $5, $6)`

	_, err := r.db.Exec(ctx, query, token.ID, token.UserID, token.RefreshToken, token.ExpiresAt, token.CreatedAt, token.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to insert refresh token: %w", err)
	}

	return nil
}

func (r *RefreshTokenDB) Read(ctx context.Context, id uuid.UUID) (*domain.RefreshToken, error) {
	query := `SELECT id, user_id, refresh_token, expires_at, created_at, updated_at
              FROM refresh_tokens WHERE id = $1`
	row := r.db.QueryRow(ctx, query, id)

	var token domain.RefreshToken
	err := row.Scan(&token.ID, &token.UserID, &token.RefreshToken, &token.ExpiresAt, &token.CreatedAt, &token.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("refresh token not found: %w", err)
		}
		return nil, fmt.Errorf("failed to read refresh token: %w", err)
	}

	return &token, nil
}

func (r *RefreshTokenDB) ReadByRefreshToken(ctx context.Context, refreshToken string) (*domain.RefreshToken, error) {
	query := `SELECT id, user_id, refresh_token, expires_at, created_at, updated_at
	          FROM refresh_tokens WHERE refresh_token=$1`
	row := r.db.QueryRow(ctx, query, refreshToken)

	var token domain.RefreshToken
	err := row.Scan(&token.ID, &token.UserID, &token.RefreshToken, &token.ExpiresAt, &token.CreatedAt, &token.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("refresh token not found: %w", err)
		}
		return nil, fmt.Errorf("failed to read refresh token: %w", err)
	}

	return &token, nil
}

func (r *RefreshTokenDB) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM refresh_tokens WHERE id = $1`
	result, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete refresh token: %w", err)
	}

	rowsAffected := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("refresh token not found")
	}

	return nil
}
