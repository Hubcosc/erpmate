package user

import "github.com/google/uuid"

type CreateUserRequest struct {
	EmployeeID   *string    `json:"employee_id"`
	FullName     string     `json:"full_name"`
	Email        string     `json:"email"`
	Phone        *string    `json:"phone"`
	Password     string     `json:"password"`
	DepartmentID *uuid.UUID `json:"department_id"`
	RoleID       uuid.UUID  `json:"role_id"`
}

type UpdateUserRequest struct {
	EmployeeID   *string    `json:"employee_id"`
	FullName     *string    `json:"full_name"`
	Phone        *string    `json:"phone"`
	DepartmentID *uuid.UUID `json:"department_id"`
	RoleID       *uuid.UUID `json:"role_id"`
	IsActive     *bool      `json:"is_active"`
}

type ChangePasswordRequest struct {
	NewPassword string `json:"new_password"`
}