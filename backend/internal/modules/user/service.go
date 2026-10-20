package user

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Create(ctx context.Context, req CreateUserRequest) (*User, error) {
	if req.FullName == "" || req.Email == "" || req.Password == "" {
		return nil, errors.New("full_name, email and password are required")
	}

	// Check if email already exists
	existing, _ := s.repo.GetByEmail(ctx, req.Email)
	if existing != nil {
		return nil, errors.New("email already exists")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	u := &User{
		ID:           uuid.New(),
		EmployeeID:   req.EmployeeID,
		FullName:     req.FullName,
		Email:        req.Email,
		Phone:        req.Phone,
		DepartmentID: req.DepartmentID,
		RoleID:       req.RoleID,
		IsActive:     true,
	}

	if err := s.repo.Create(ctx, u, string(hash)); err != nil {
		return nil, err
	}

	return s.repo.GetByID(ctx, u.ID)
}

func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (*User, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *Service) List(ctx context.Context, search string, roleID *uuid.UUID, page, limit int) ([]User, int64, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}
	return s.repo.List(ctx, search, roleID, page, limit)
}

func (s *Service) Update(ctx context.Context, id uuid.UUID, req UpdateUserRequest) (*User, error) {
	fields := map[string]interface{}{}

	if req.EmployeeID != nil {
		fields["employee_id"] = *req.EmployeeID
	}
	if req.FullName != nil {
		fields["full_name"] = *req.FullName
	}
	if req.Phone != nil {
		fields["phone"] = *req.Phone
	}
	if req.DepartmentID != nil {
		fields["department_id"] = *req.DepartmentID
	}
	if req.RoleID != nil {
		fields["role_id"] = *req.RoleID
	}
	if req.IsActive != nil {
		fields["is_active"] = *req.IsActive
	}

	if err := s.repo.Update(ctx, id, fields); err != nil {
		return nil, err
	}

	return s.repo.GetByID(ctx, id)
}

func (s *Service) ChangePassword(ctx context.Context, id uuid.UUID, newPassword string) error {
	if len(newPassword) < 6 {
		return errors.New("password must be at least 6 characters")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	return s.repo.Update(ctx, id, map[string]interface{}{
		"password_hash": string(hash),
	})
}

func (s *Service) Delete(ctx context.Context, id uuid.UUID) error {
	return s.repo.SoftDelete(ctx, id)
}

func (s *Service) ListRoles(ctx context.Context) ([]Role, error) {
	return s.repo.ListRoles(ctx)
}