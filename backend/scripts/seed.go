package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/Ethio-ERP/erpmate/config"
	"github.com/Ethio-ERP/erpmate/internal/database"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	email := strings.TrimSpace(os.Getenv("BOOTSTRAP_ADMIN_EMAIL"))
	password := os.Getenv("BOOTSTRAP_ADMIN_PASSWORD")
	name := strings.TrimSpace(os.Getenv("BOOTSTRAP_ADMIN_NAME"))
	if email == "" || password == "" || name == "" {
		log.Fatal("BOOTSTRAP_ADMIN_EMAIL, BOOTSTRAP_ADMIN_PASSWORD, and BOOTSTRAP_ADMIN_NAME are required")
	}
	if len(password) < 12 {
		log.Fatal("BOOTSTRAP_ADMIN_PASSWORD must be at least 12 characters")
	}

	cfg := config.Load()
	db := database.Connect(cfg)
	defer db.Close()

	ctx := context.Background()
	var existing bool
	if err := db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM users WHERE lower(email)=lower($1) AND deleted_at IS NULL)`, email).Scan(&existing); err != nil {
		log.Fatal(err)
	}
	if existing {
		log.Fatalf("an active account with email %q already exists; no changes made", email)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Fatal(err)
	}
	var userID uuid.UUID
	err = db.QueryRow(ctx, `
		INSERT INTO users (id, full_name, email, password_hash, role_id, is_active)
		SELECT $1, $2, lower($3), $4, r.id, TRUE
		FROM roles r WHERE r.name='admin'
		RETURNING id
	`, uuid.New(), name, email, string(hash)).Scan(&userID)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Created administrator account %s (id %s).\n", email, userID)
}
