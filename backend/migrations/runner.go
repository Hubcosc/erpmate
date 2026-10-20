package migrations

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed *.up.sql
var migrationFiles embed.FS

// Apply executes unapplied migrations in filename order. It recognizes existing
// migration objects and records them so databases migrated manually can adopt
// this runner without trying to recreate their schema.
func Apply(ctx context.Context, pool *pgxpool.Pool) error {
	if _, err := pool.Exec(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (
		version VARCHAR(100) PRIMARY KEY,
		applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
	)`); err != nil {
		return fmt.Errorf("create migration tracking table: %w", err)
	}

	files, err := fs.Glob(migrationFiles, "*.up.sql")
	if err != nil {
		return fmt.Errorf("list embedded migrations: %w", err)
	}
	sort.Strings(files)
	for _, name := range files {
		var applied bool
		if err := pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version=$1)`, name).Scan(&applied); err != nil {
			return fmt.Errorf("check migration %s: %w", name, err)
		}
		if applied {
			continue
		}

		marker := migrationMarker(name)
		if marker != "" {
			var exists bool
			if err := pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM information_schema.tables WHERE table_schema=current_schema() AND table_name=$1)`, marker).Scan(&exists); err != nil {
				return fmt.Errorf("check existing migration object for %s: %w", name, err)
			}
			if exists {
				if _, err := pool.Exec(ctx, `INSERT INTO schema_migrations(version) VALUES($1) ON CONFLICT DO NOTHING`, name); err != nil {
					return fmt.Errorf("adopt existing migration %s: %w", name, err)
				}
				continue
			}
		}

		sql, err := migrationFiles.ReadFile(name)
		if err != nil {
			return fmt.Errorf("read migration %s: %w", name, err)
		}
		tx, err := pool.Begin(ctx)
		if err != nil {
			return fmt.Errorf("begin migration %s: %w", name, err)
		}
		if _, err = tx.Exec(ctx, string(sql)); err != nil {
			_ = tx.Rollback(ctx)
			return fmt.Errorf("apply migration %s: %w", name, err)
		}
		if _, err = tx.Exec(ctx, `INSERT INTO schema_migrations(version) VALUES($1)`, name); err != nil {
			_ = tx.Rollback(ctx)
			return fmt.Errorf("record migration %s: %w", name, err)
		}
		if err = tx.Commit(ctx); err != nil {
			return fmt.Errorf("commit migration %s: %w", name, err)
		}
	}
	return nil
}

func migrationMarker(name string) string {
	parts := strings.SplitN(name, "_", 2)
	if len(parts) != 2 {
		return ""
	}
	switch parts[0] {
	case "000001":
		return "roles"
	case "000002":
		return "purchase_requisitions"
	case "000003":
		return "material_requisitions"
	case "000004":
		return "material_issues"
	case "000005":
		return "notifications"
	case "000006":
		return "purchase_orders"
	case "000007":
		return "inventory_alerts"
	default:
		return ""
	}
}
