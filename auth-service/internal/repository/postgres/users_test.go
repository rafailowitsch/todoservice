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
func setupPostgres(t *testing.T) (*pgx.Conn, func()) {
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
		CREATE TABLE users (
			id UUID PRIMARY KEY,
			name VARCHAR(100),
			email VARCHAR(100) UNIQUE,
			password_hash VARCHAR(100),
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

func TestUserDB_Create(t *testing.T) {
	conn, teardown := setupPostgres(t)
	defer teardown()

	userDB := NewUserDB(conn)

	user := &domain.User{
		Name:         "Alice",
		Email:        "alice@example.com",
		PasswordHash: "hashedpassword",
	}

	err := userDB.Create(context.Background(), user)
	assert.NoError(t, err)

	// Verify the user was inserted
	var insertedUser domain.User
	err = conn.QueryRow(context.Background(), `SELECT id, name, email, password_hash, created_at, updated_at FROM users WHERE id = $1`, user.ID).Scan(
		&insertedUser.ID,
		&insertedUser.Name,
		&insertedUser.Email,
		&insertedUser.PasswordHash,
		&insertedUser.CreatedAt,
		&insertedUser.UpdatedAt,
	)
	assert.NoError(t, err)
	assert.Equal(t, user.Name, insertedUser.Name)
	assert.Equal(t, user.Email, insertedUser.Email)
	assert.Equal(t, user.PasswordHash, insertedUser.PasswordHash)
}

func TestUserDB_Read(t *testing.T) {
	conn, teardown := setupPostgres(t)
	defer teardown()

	userID := uuid.New()
	_, err := conn.Exec(context.Background(), `INSERT INTO users (id, name, email, password_hash, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6)`,
		userID, "Alice", "alice@example.com", "hashedpassword", time.Now(), time.Now())
	assert.NoError(t, err)

	userDB := NewUserDB(conn)

	user, err := userDB.Read(context.Background(), userID)
	assert.NoError(t, err)
	assert.NotNil(t, user)
	assert.Equal(t, "Alice", user.Name)
	assert.Equal(t, "alice@example.com", user.Email)
	assert.Equal(t, "hashedpassword", user.PasswordHash)
}

func TestUserDB_ReadByEmail(t *testing.T) {
	conn, teardown := setupPostgres(t)
	defer teardown()

	userID := uuid.New()
	_, err := conn.Exec(context.Background(), `INSERT INTO users (id, name, email, password_hash, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6)`,
		userID, "Alice", "alice@example.com", "hashedpassword", time.Now(), time.Now())
	assert.NoError(t, err)

	userDB := NewUserDB(conn)

	user, err := userDB.ReadByEmail(context.Background(), "alice@example.com")
	assert.NoError(t, err)
	assert.NotNil(t, user)
	assert.Equal(t, "Alice", user.Name)
	assert.Equal(t, "alice@example.com", user.Email)
	assert.Equal(t, "hashedpassword", user.PasswordHash)
}

func TestUserDB_Update(t *testing.T) {
	conn, teardown := setupPostgres(t)
	defer teardown()

	userID := uuid.New()
	_, err := conn.Exec(context.Background(), `INSERT INTO users (id, name, email, password_hash, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6)`,
		userID, "Alice", "alice@example.com", "hashedpassword", time.Now(), time.Now())
	assert.NoError(t, err)

	userDB := NewUserDB(conn)

	updatedUser := &domain.User{
		ID:           userID,
		Name:         "Alice Updated",
		Email:        "alice_updated@example.com",
		PasswordHash: "newhashedpassword",
		UpdatedAt:    time.Now(),
	}

	err = userDB.Update(context.Background(), updatedUser)
	assert.NoError(t, err)

	// Verify the user was updated
	var user domain.User
	err = conn.QueryRow(context.Background(), `SELECT id, name, email, password_hash, created_at, updated_at FROM users WHERE id = $1`, userID).Scan(
		&user.ID,
		&user.Name,
		&user.Email,
		&user.PasswordHash,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	assert.NoError(t, err)
	assert.Equal(t, updatedUser.Name, user.Name)
	assert.Equal(t, updatedUser.Email, user.Email)
	assert.Equal(t, updatedUser.PasswordHash, user.PasswordHash)
}

func TestUserDB_Delete(t *testing.T) {
	conn, teardown := setupPostgres(t)
	defer teardown()

	userID := uuid.New()
	_, err := conn.Exec(context.Background(), `INSERT INTO users (id, name, email, password_hash, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6)`,
		userID, "Alice", "alice@example.com", "hashedpassword", time.Now(), time.Now())
	assert.NoError(t, err)

	userDB := NewUserDB(conn)

	err = userDB.Delete(context.Background(), userID)
	assert.NoError(t, err)

	// Verify the user was deleted
	var user domain.User
	err = conn.QueryRow(context.Background(), `SELECT id, name, email, password_hash, created_at, updated_at FROM users WHERE id = $1`, userID).Scan(
		&user.ID,
		&user.Name,
		&user.Email,
		&user.PasswordHash,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, pgx.ErrNoRows))
}
