package database

import (
	"context"
	"fmt"
	"log"

	"github.com/Ethio-ERP/erpmate/config"
	"github.com/jackc/pgx/v5/pgxpool"
)

func Connect(cfg *config.Config) *pgxpool.Pool {
	dsn := fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=disable",
		cfg.DBUser, cfg.DBPassword, cfg.DBHost, cfg.DBPort, cfg.DBName,
	)

	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		log.Fatalf("Unable to connect to database: %v", err)
	}

	if err := pool.Ping(context.Background()); err != nil {
		log.Fatalf("Unable to ping database: %v", err)
	}
	if err := Migrate(context.Background(), pool); err != nil {
		pool.Close()
		log.Fatalf("Unable to apply database migrations: %v", err)
	}

	log.Println("Connected to PostgreSQL and verified database migrations")
	return pool
}
