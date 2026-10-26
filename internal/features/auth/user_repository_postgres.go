package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/ibra172/go-ffmpeg-pipeline/internal/apperr"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UserPostgresRepository struct {
	pool         *pgxpool.Pool
	queryTimeout time.Duration
}

func NewUserPostgresRepository(
	pool *pgxpool.Pool,
	queryTimeout time.Duration,
) *UserPostgresRepository {
	return &UserPostgresRepository{
		pool:         pool,
		queryTimeout: queryTimeout,
	}
}

func (r *UserPostgresRepository) CreateUser(ctx context.Context, user *User) error {
	queryCtx, cancel := context.WithTimeout(ctx, r.queryTimeout)
	defer cancel()

	query := `
	INSERT INTO ffmpeg_app.users (id, username, password_hash)
    VALUES ($1, $2, $3);
	`

	_, err := r.pool.Exec(
		queryCtx,
		query,
		user.ID, user.Username, user.PasswordHash,
	)
	if err != nil {
		var pgErr *pgconn.PgError

		if errors.As(err, &pgErr) {
			if pgErr.Code == "23505" &&
				pgErr.ConstraintName == "users_username_key" {
				return fmt.Errorf("username is already taken: %w", apperr.ErrConflict)
			}
		}

		return fmt.Errorf("execute create user query: %w", err)
	}

	return nil
}

func (r *UserPostgresRepository) GetUserByID(ctx context.Context, id uuid.UUID) (User, error) {
	queryCtx, cancel := context.WithTimeout(ctx, r.queryTimeout)
	defer cancel()

	query := `
	SELECT id, username, password_hash
	FROM ffmpeg_app.users
	WHERE id = $1
	`

	row := r.pool.QueryRow(queryCtx, query, id)

	var user User
	if err := row.Scan(&user.ID, &user.Username, &user.PasswordHash); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return User{}, fmt.Errorf("no user with ID=`%v`: %w", id, apperr.ErrNotFound)
		}

		return User{}, fmt.Errorf("scan row: %w", err)
	}

	return user, nil
}

func (r *UserPostgresRepository) GetUserByUsername(
	ctx context.Context, username string,
) (User, error) {
	queryCtx, cancel := context.WithTimeout(ctx, r.queryTimeout)
	defer cancel()

	query := `
	SELECT id, username, password_hash
	FROM ffmpeg_app.users
	WHERE username = $1
	`

	row := r.pool.QueryRow(queryCtx, query, username)

	var user User
	if err := row.Scan(&user.ID, &user.Username, &user.PasswordHash); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return User{}, fmt.Errorf("no user with username=`%v`: %w", username, apperr.ErrNotFound)
		}

		return User{}, fmt.Errorf("scan row: %w", err)
	}

	return user, nil
}
