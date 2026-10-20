package user

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Create(ctx context.Context, u *User, passwordHash string) error {
	query := `
		INSERT INTO users (
			id, employee_id, full_name, email, phone, password_hash,
			department_id, role_id, is_active
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
		RETURNING created_at, updated_at
	`
	return r.db.QueryRow(ctx, query,
		u.ID, u.EmployeeID, u.FullName, u.Email, u.Phone, passwordHash,
		u.DepartmentID, u.RoleID, u.IsActive,
	).Scan(&u.CreatedAt, &u.UpdatedAt)
}

func (r *Repository) GetByID(ctx context.Context, id uuid.UUID) (*User, error) {
	query := `
		SELECT u.id, u.employee_id, u.full_name, u.email, u.phone,
		       u.department_id, d.name, u.role_id, r.name,
		       u.is_active, u.last_login_at, u.created_at, u.updated_at
		FROM users u
		JOIN roles r ON r.id = u.role_id
		LEFT JOIN departments d ON d.id = u.department_id
		WHERE u.id = $1 AND u.deleted_at IS NULL
	`
	var u User
	err := r.db.QueryRow(ctx, query, id).Scan(
		&u.ID, &u.EmployeeID, &u.FullName, &u.Email, &u.Phone,
		&u.DepartmentID, &u.DepartmentName, &u.RoleID, &u.RoleName,
		&u.IsActive, &u.LastLoginAt, &u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *Repository) GetByEmail(ctx context.Context, email string) (*User, error) {
	query := `
		SELECT u.id, u.employee_id, u.full_name, u.email, u.phone,
		       u.department_id, d.name, u.role_id, r.name,
		       u.is_active, u.last_login_at, u.created_at, u.updated_at
		FROM users u
		JOIN roles r ON r.id = u.role_id
		LEFT JOIN departments d ON d.id = u.department_id
		WHERE u.email = $1 AND u.deleted_at IS NULL
	`
	var u User
	err := r.db.QueryRow(ctx, query, email).Scan(
		&u.ID, &u.EmployeeID, &u.FullName, &u.Email, &u.Phone,
		&u.DepartmentID, &u.DepartmentName, &u.RoleID, &u.RoleName,
		&u.IsActive, &u.LastLoginAt, &u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *Repository) List(ctx context.Context, search string, roleID *uuid.UUID, page, limit int) ([]User, int64, error) {
	offset := (page - 1) * limit
	var args []interface{}
	where := "WHERE u.deleted_at IS NULL"
	argPos := 1

	if search != "" {
		where += fmt.Sprintf(" AND (u.full_name ILIKE $%d OR u.email ILIKE $%d OR u.employee_id ILIKE $%d)", argPos, argPos, argPos)
		args = append(args, "%"+search+"%")
		argPos++
	}

	if roleID != nil {
		where += fmt.Sprintf(" AND u.role_id = $%d", argPos)
		args = append(args, *roleID)
		argPos++
	}

	var total int64
	countQuery := "SELECT COUNT(*) FROM users u " + where
	if err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	query := fmt.Sprintf(`
		SELECT u.id, u.employee_id, u.full_name, u.email, u.phone,
		       u.department_id, d.name, u.role_id, r.name,
		       u.is_active, u.last_login_at, u.created_at, u.updated_at
		FROM users u
		JOIN roles r ON r.id = u.role_id
		LEFT JOIN departments d ON d.id = u.department_id
		%s
		ORDER BY u.created_at DESC
		LIMIT $%d OFFSET $%d
	`, where, argPos, argPos+1)

	args = append(args, limit, offset)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var list []User
	for rows.Next() {
		var u User
		err := rows.Scan(
			&u.ID, &u.EmployeeID, &u.FullName, &u.Email, &u.Phone,
			&u.DepartmentID, &u.DepartmentName, &u.RoleID, &u.RoleName,
			&u.IsActive, &u.LastLoginAt, &u.CreatedAt, &u.UpdatedAt,
		)
		if err != nil {
			return nil, 0, err
		}
		list = append(list, u)
	}

	return list, total, nil
}

func (r *Repository) Update(ctx context.Context, id uuid.UUID, fields map[string]interface{}) error {
	if len(fields) == 0 {
		return nil
	}

	query := "UPDATE users SET updated_at = NOW()"
	args := []interface{}{}
	pos := 1

	for k, v := range fields {
		query += fmt.Sprintf(", %s = $%d", k, pos)
		args = append(args, v)
		pos++
	}

	query += fmt.Sprintf(" WHERE id = $%d AND deleted_at IS NULL", pos)
	args = append(args, id)

	_, err := r.db.Exec(ctx, query, args...)
	return err
}

func (r *Repository) SoftDelete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `
		UPDATE users SET deleted_at = NOW(), is_active = false, updated_at = NOW()
		WHERE id = $1
	`, id)
	return err
}

func (r *Repository) ListRoles(ctx context.Context) ([]Role, error) {
	rows, err := r.db.Query(ctx, `SELECT id, name, description FROM roles ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var roles []Role
	for rows.Next() {
		var role Role
		if err := rows.Scan(&role.ID, &role.Name, &role.Description); err != nil {
			return nil, err
		}
		roles = append(roles, role)
	}
	return roles, nil
}