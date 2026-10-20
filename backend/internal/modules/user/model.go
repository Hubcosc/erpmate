package user

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID             uuid.UUID  `json:"id"`
	EmployeeID     *string    `json:"employee_id"`
	FullName       string     `json:"full_name"`
	Email          string     `json:"email"`
	Phone          *string    `json:"phone"`
	DepartmentID   *uuid.UUID `json:"department_id"`
	DepartmentName *string    `json:"department_name,omitempty"`
	RoleID         uuid.UUID  `json:"role_id"`
	RoleName       string     `json:"role_name"`
	IsActive       bool       `json:"is_active"`
	LastLoginAt    *time.Time `json:"last_login_at"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

type Role struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Description *string   `json:"description"`
}