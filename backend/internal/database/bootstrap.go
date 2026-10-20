package database

import (
	"context"
	"fmt"
	"log"

	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"

	"github.com/Ethio-ERP/erpmate/config"
)

// BootstrapAdmin ensures the initial admin user exists.
// Safe to call on every startup — it's a no-op if the user already exists.
func BootstrapAdmin(ctx context.Context, pool *pgxpool.Pool, cfg *config.Config) error {
	if cfg.BootstrapAdminEmail == "" || cfg.BootstrapAdminPassword == "" {
		log.Println("bootstrap: BOOTSTRAP_ADMIN_EMAIL/PASSWORD not set, skipping")
		return nil
	}

	// 1. Already exists?
	var exists bool
	if err := pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM users WHERE email = $1 AND deleted_at IS NULL)`,
		cfg.BootstrapAdminEmail,
	).Scan(&exists); err != nil {
		return fmt.Errorf("check admin exists: %w", err)
	}
	if exists {
		log.Printf("bootstrap: admin %s already exists", cfg.BootstrapAdminEmail)
		return nil
	}

	// 2. Look up the admin role id (adjust names to match your `roles` table)
	var roleID string
	err := pool.QueryRow(ctx,
		`SELECT id FROM roles WHERE name IN ('admin', 'system_admin') LIMIT 1`,
	).Scan(&roleID)
	if err != nil {
		return fmt.Errorf("find admin role: %w", err)
	}

	// 3. Hash the password with bcrypt (your auth module likely does this too)
	hash, err := bcrypt.GenerateFromPassword([]byte(cfg.BootstrapAdminPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hash admin password: %w", err)
	}

	// 4. Insert the admin. Matches your `users` schema exactly.
	_, err = pool.Exec(ctx, `
		INSERT INTO users (
			employee_id,
			full_name,
			email,
			password_hash,
			role_id,
			is_active
		) VALUES ($1, $2, $3, $4, $5, true)
	`, "ADMIN-001", cfg.BootstrapAdminName, cfg.BootstrapAdminEmail, string(hash), roleID)
	if err != nil {
		return fmt.Errorf("insert admin user: %w", err)
	}

	log.Printf("bootstrap: admin user created → %s", cfg.BootstrapAdminEmail)
	return nil
}