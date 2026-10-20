package config

import (
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
)

// Config holds all application configuration loaded from environment variables.
type Config struct {
	// Database
	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string

	// Server
	Port string

	// Auth
	JWTSecret string

	// Bootstrap admin (created on first run if not exists)
	BootstrapAdminName     string
	BootstrapAdminEmail    string
	BootstrapAdminPassword string
}

// Load reads configuration from a .env file (if present) and OS environment variables.
// Priority: OS env vars > .env file > fallback defaults.
func Load() *Config {
	// Try to load .env; ignore error if it doesn't exist.
	if err := godotenv.Load(); err != nil {
		log.Println("no .env file found, using system environment variables")
	}

	return &Config{
		// Support both DB_* and POSTGRES_* prefixes for Docker compatibility
		DBHost:     getFirstEnv("DB_HOST", "POSTGRES_HOST", "localhost"),
		DBPort:     getFirstEnv("DB_PORT", "POSTGRES_PORT", "5432"),
		DBUser:     getFirstEnv("DB_USER", "POSTGRES_USER", "erp_user"),
		DBPassword: getFirstEnv("DB_PASSWORD", "POSTGRES_PASSWORD", "replace-with-a-strong-password"),
		DBName:     getFirstEnv("DB_NAME", "POSTGRES_DB", "erp_inventory"),

		Port: getFirstEnv("PORT", "API_PORT", "8080"),

		JWTSecret: getEnv("JWT_SECRET", "super-secret-key-change-me"),

		BootstrapAdminName:     getEnv("BOOTSTRAP_ADMIN_NAME", "System Administrator"),
		BootstrapAdminEmail:    getEnv("BOOTSTRAP_ADMIN_EMAIL", "admin@example.test"),
		BootstrapAdminPassword: getEnv("BOOTSTRAP_ADMIN_PASSWORD", ""),
	}
}

// Validate checks that required secrets are set and safe.
// Call this from main() right after Load().
func (c *Config) Validate() error {
	if c.JWTSecret == "" || c.JWTSecret == "super-secret-key-change-me" {
		return fmt.Errorf("JWT_SECRET must be set to a secure value in .env")
	}
	if len(c.JWTSecret) < 16 {
		return fmt.Errorf("JWT_SECRET must be at least 16 characters long")
	}

	if c.BootstrapAdminPassword != "" && len(c.BootstrapAdminPassword) < 12 {
		return fmt.Errorf("BOOTSTRAP_ADMIN_PASSWORD must be at least 12 characters long")
	}

	if c.DBHost == "" || c.DBPort == "" || c.DBUser == "" || c.DBName == "" {
		return fmt.Errorf("database configuration is incomplete (check DB_HOST, DB_PORT, DB_USER, DB_NAME)")
	}

	return nil
}

// DSN returns a PostgreSQL connection string.
// Example: host=localhost port=5432 user=erp_user password=xxx dbname=erp_inventory sslmode=disable
func (c *Config) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		c.DBHost, c.DBPort, c.DBUser, c.DBPassword, c.DBName,
	)
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// getEnv returns the value of key, or fallback if unset/empty.
func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

// getFirstEnv returns the first non-empty env var between primary and alternate,
// or fallback if both are unset.
func getFirstEnv(primary, alternate, fallback string) string {
	if value := os.Getenv(primary); value != "" {
		return value
	}
	if value := os.Getenv(alternate); value != "" {
		return value
	}
	return fallback
}