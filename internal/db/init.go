package db

import (
	"context"
	"github.com/jackc/pgx/v4/pgxpool"
)

func InitDB(config *pgxpool.Config) (*pgxpool.Pool, error) {
	pgPool, err := pgxpool.ConnectConfig(context.Background(), config)
	if err != nil {
		return nil, err
	}

	return pgPool, nil
}
