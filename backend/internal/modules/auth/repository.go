package auth

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) GetByEmail(ctx context.Context, email string) (*User, error) {
	query := `
		SELECT u.id, u.employee_id, u.full_name, u.email, u.phone,
		       u.password_hash, u.department_id, d.name as department_name,
		       u.role_id, r.name as role_name,
		       u.is_active, u.last_login_at, u.created_at
		FROM users u
		JOIN roles r ON r.id = u.role_id
		LEFT JOIN departments d ON d.id = u.department_id
		WHERE u.email = $1 AND u.deleted_at IS NULL
	`
	var u User
	err := r.db.QueryRow(ctx, query, email).Scan(
		&u.ID, &u.EmployeeID, &u.FullName, &u.Email, &u.Phone,
		&u.PasswordHash, &u.DepartmentID, &u.DepartmentName,
		&u.RoleID, &u.RoleName,
		&u.IsActive, &u.LastLoginAt, &u.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *Repository) GetByID(ctx context.Context, id uuid.UUID) (*User, error) {
	query := `
		SELECT u.id, u.employee_id, u.full_name, u.email, u.phone,
		       u.password_hash, u.department_id, d.name as department_name,
		       u.role_id, r.name as role_name,
		       u.is_active, u.last_login_at, u.created_at
		FROM users u
		JOIN roles r ON r.id = u.role_id
		LEFT JOIN departments d ON d.id = u.department_id
		WHERE u.id = $1 AND u.deleted_at IS NULL
	`
	var u User
	err := r.db.QueryRow(ctx, query, id).Scan(
		&u.ID, &u.EmployeeID, &u.FullName, &u.Email, &u.Phone,
		&u.PasswordHash, &u.DepartmentID, &u.DepartmentName,
		&u.RoleID, &u.RoleName,
		&u.IsActive, &u.LastLoginAt, &u.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *Repository) UpdateLastLogin(ctx context.Context, userID uuid.UUID) error {
	_, err := r.db.Exec(ctx, `UPDATE users SET last_login_at = NOW() WHERE id = $1`, userID)
	return err
}