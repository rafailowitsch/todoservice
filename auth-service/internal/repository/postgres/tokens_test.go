package postgres

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"todoservice/auth-service/internal/domain"
)

// Helper function to setup PostgreSQL container
func setupPostgresTokens(t *testing.T) (*pgx.Conn, func()) {
	ctx := context.Background()

	req := testcontainers.ContainerRequest{
		Image:        "postgres:13",
		ExposedPorts: []string{"5432/tcp"},
		Env: map[string]string{
			"POSTGRES_PASSWORD": "password",
			"POSTGRES_USER":     "user",
			"POSTGRES_DB":       "testdb",
		},
		WaitingFor: wait.ForListeningPort("5432/tcp"),
	}
	postgresContainer, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	assert.NoError(t, err)

	host, err := postgresContainer.Host(ctx)
	assert.NoError(t, err)

	port, err := postgresContainer.MappedPort(ctx, "5432")
	assert.NoError(t, err)

	dsn := "postgres://user:password@" + host + ":" + port.Port() + "/testdb?sslmode=disable"
	conn, err := pgx.Connect(context.Background(), dsn)
	assert.NoError(t, err)

	_, err = conn.Exec(ctx, `
		CREATE TABLE refresh_tokens (
			id UUID PRIMARY KEY,
			user_id UUID NOT NULL,
			refresh_token TEXT NOT NULL,
			expires_at TIMESTAMP NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);
	`)
	assert.NoError(t, err)

	teardown := func() {
		conn.Close(ctx)
		postgresContainer.Terminate(ctx)
	}

	return conn, teardown
}

func TestRefreshTokenDB_Create(t *testing.T) {
	conn, teardown := setupPostgresTokens(t)
	defer teardown()

	tokenDB := NewRefreshTokenDB(conn)

	token := &domain.RefreshToken{
		UserID:       uuid.New(),
		RefreshToken: "example_refresh_token",
		ExpiresAt:    time.Now().Add(24 * time.Hour).UTC(),
	}

	err := tokenDB.Create(context.Background(), token)
	assert.NoError(t, err)

	// Verify the refresh token was inserted
	var insertedToken domain.RefreshToken
	err = conn.QueryRow(context.Background(), `SELECT id, user_id, refresh_token, expires_at, created_at, updated_at FROM refresh_tokens WHERE id = $1`, token.ID).Scan(
		&insertedToken.ID,
		&insertedToken.UserID,
		&insertedToken.RefreshToken,
		&insertedToken.ExpiresAt,
		&insertedToken.CreatedAt,
		&insertedToken.UpdatedAt,
	)
	assert.NoError(t, err)
	assert.Equal(t, token.UserID, insertedToken.UserID)
	assert.Equal(t, token.RefreshToken, insertedToken.RefreshToken)
	assert.WithinDuration(t, token.ExpiresAt, insertedToken.ExpiresAt, time.Second)
}

func TestRefreshTokenDB_Read(t *testing.T) {
	conn, teardown := setupPostgresTokens(t)
	defer teardown()

	tokenID := uuid.New()
	_, err := conn.Exec(context.Background(), `INSERT INTO refresh_tokens (id, user_id, refresh_token, expires_at, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6)`,
		tokenID, uuid.New(), "example_refresh_token", time.Now().Add(24*time.Hour), time.Now(), time.Now())
	assert.NoError(t, err)

	tokenDB := NewRefreshTokenDB(conn)

	token, err := tokenDB.Read(context.Background(), tokenID)
	assert.NoError(t, err)
	assert.NotNil(t, token)
	assert.Equal(t, "example_refresh_token", token.RefreshToken)
}

func TestRefreshTokenDB_ReadByRefreshToken(t *testing.T) {
	conn, teardown := setupPostgresTokens(t)
	defer teardown()

	tokenID := uuid.New()
	_, err := conn.Exec(context.Background(), `INSERT INTO refresh_tokens (id, user_id, refresh_token, expires_at, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6)`,
		tokenID, uuid.New(), "example_refresh_token", time.Now().Add(24*time.Hour), time.Now(), time.Now())
	assert.NoError(t, err)

	tokenDB := NewRefreshTokenDB(conn)

	token, err := tokenDB.ReadByRefreshToken(context.Background(), "example_refresh_token")
	assert.NoError(t, err)
	assert.NotNil(t, token)
	assert.Equal(t, "example_refresh_token", token.RefreshToken)
}

func TestRefreshTokenDB_Delete(t *testing.T) {
	conn, teardown := setupPostgresTokens(t)
	defer teardown()

	tokenID := uuid.New()
	_, err := conn.Exec(context.Background(), `INSERT INTO refresh_tokens (id, user_id, refresh_token, expires_at, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6)`,
		tokenID, uuid.New(), "example_refresh_token", time.Now().Add(24*time.Hour), time.Now(), time.Now())
	assert.NoError(t, err)

	tokenDB := NewRefreshTokenDB(conn)

	err = tokenDB.Delete(context.Background(), tokenID)
	assert.NoError(t, err)

	// Verify the refresh token was deleted
	var token domain.RefreshToken
	err = conn.QueryRow(context.Background(), `SELECT id, user_id, refresh_token, expires_at, created_at, updated_at FROM refresh_tokens WHERE id = $1`, tokenID).Scan(
		&token.ID,
		&token.UserID,
		&token.RefreshToken,
		&token.ExpiresAt,
		&token.CreatedAt,
		&token.UpdatedAt,
	)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, pgx.ErrNoRows))
}
