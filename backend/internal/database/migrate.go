package database

import (
	"context"

	"github.com/Ethio-ERP/erpmate/migrations"
	"github.com/jackc/pgx/v5/pgxpool"
)

func Migrate(ctx context.Context, pool *pgxpool.Pool) error {
	return migrations.Apply(ctx, pool)
}
