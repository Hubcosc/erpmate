package notification

import (
	"context"
	"errors"

	"github.com/google/uuid"
)

type Service struct{ repo *Repository }

func NewService(repo *Repository) *Service { return &Service{repo: repo} }

func (s *Service) List(ctx context.Context, userID uuid.UUID, unread bool, page, limit int) ([]Notification, int64, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}
	return s.repo.List(ctx, userID, unread, page, limit)
}

func (s *Service) MarkRead(ctx context.Context, userID, id uuid.UUID) error {
	updated, err := s.repo.MarkRead(ctx, userID, id)
	if err != nil {
		return err
	}
	if !updated {
		return errors.New("notification not found or already read")
	}
	return nil
}

func (s *Service) MarkAllRead(ctx context.Context, userID uuid.UUID) error {
	return s.repo.MarkAllRead(ctx, userID)
}
