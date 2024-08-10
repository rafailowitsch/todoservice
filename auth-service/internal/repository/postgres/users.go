package postgres

import (
	"context"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"time"
	"todoservice/auth-service/internal/domain"
)

type UserDB struct {
	db *pgx.Conn
}

func NewUserDB(db *pgx.Conn) *UserDB {
	return &UserDB{
		db: db,
	}
}

func (u *UserDB) Create(ctx context.Context, user *domain.User) error {
	user.ID = uuid.New()
	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()

	query := `INSERT INTO users (id, name, email, password_hash, created_at, updated_at)
              VALUES ($1, $2, $3, $4, $5, $6)`

	_, err := u.db.Exec(ctx, query, user.ID, user.Name, user.Email, user.PasswordHash, user.CreatedAt, user.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to insert user: %w", err)
	}

	return nil
}

func (u *UserDB) Read(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	query := `SELECT id, name, email, password_hash, created_at, updated_at
              FROM users WHERE id = $1`
	row := u.db.QueryRow(ctx, query, id)

	var user domain.User
	err := row.Scan(&user.ID, &user.Name, &user.Email, &user.PasswordHash, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to read user: %w", err)
	}

	return &user, nil
}

func (u *UserDB) ReadByEmail(ctx context.Context, email string) (*domain.User, error) {
	query := `SELECT id, name, email, password_hash, created_at, updated_at
	          FROM users WHERE email=$1`
	row := u.db.QueryRow(ctx, query, email)

	var user domain.User
	err := row.Scan(&user.ID, &user.Name, &user.Email, &user.PasswordHash, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("user not found: %w", err)
		}
		return nil, fmt.Errorf("failed to read user: %w", err)
	}

	return &user, nil
}

func (u *UserDB) Update(ctx context.Context, user *domain.User) error {
	user.UpdatedAt = time.Now()

	query := `UPDATE users SET name = $1, email = $2, password_hash = $3, updated_at = $4 WHERE id = $5`
	result, err := u.db.Exec(ctx, query, user.Name, user.Email, user.PasswordHash, user.UpdatedAt, user.ID)
	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	rowsAffected := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("user not found")
	}

	return nil
}

func (u *UserDB) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM users WHERE id = $1`
	result, err := u.db.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	rowsAffected := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("user not found")
	}

	return nil
}
