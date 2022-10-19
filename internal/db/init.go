package db

import (
	"context"
	"database/sql"
	"github.com/BlackRRR/middleware-bot/db"
	"github.com/jackc/pgx/v4/pgxpool"
	_ "github.com/lib/pq"
	"github.com/pressly/goose/v3"
	"log"
)

const dbDriver = "postgres"

func InitDB(cfg *pgxpool.Config, dbConn string) (*pgxpool.Pool, error) {
	dataBase, err := sql.Open(dbDriver, dbConn)
	if err != nil {
		log.Fatalf("Failed open database: %s\n", err.Error())
	}

	goose.SetBaseFS(db.EmbedMigrations)

	if err := goose.SetDialect("postgres"); err != nil {
		panic(err)
	}

	if err := goose.Up(dataBase, "migrations"); err != nil {
		panic(err)
	}

	pgPool, err := pgxpool.ConnectConfig(context.Background(), cfg)
	if err != nil {
		return nil, err
	}

	return pgPool, nil
}
