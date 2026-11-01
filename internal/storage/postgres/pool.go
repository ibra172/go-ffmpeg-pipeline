package postgres

import (
	"context"
	"fmt"
	"net"
	"net/url"

	"github.com/ibra172/go-ffmpeg-pipeline/internal/config"
	"github.com/jackc/pgx/v5/pgxpool"
)

func NewPool(ctx context.Context, cfg config.Config) (*pgxpool.Pool, error) {
	connectionURL := url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword(cfg.PostgresUser, cfg.PostgresPassword),
		Host:     net.JoinHostPort(cfg.PostgresHost, cfg.PostgresPort),
		Path:     cfg.PostgresDatabase,
		RawQuery: "sslmode=disable",
	}

	pgxconfig, err := pgxpool.ParseConfig(connectionURL.String())
	if err != nil {
		return nil, fmt.Errorf("parse pgxconfig: %w", err)
	}

	pool, err := pgxpool.NewWithConfig(ctx, pgxconfig)
	if err != nil {
		return nil, fmt.Errorf("create pgxpool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("pgxpool ping: %w", err)
	}

	return pool, nil
}
