package production

import (
	"context"
	"errors"
	"github.com/google/uuid"
	"strings"
)

type WorkflowService struct{ repo *WorkflowRepository }

func NewWorkflowService(repo *WorkflowRepository) *WorkflowService {
	return &WorkflowService{repo: repo}
}
func (s *WorkflowService) Create(ctx context.Context, actor uuid.UUID, req CreateReceiptRequest) (*Receipt, error) {
	if req.WarehouseID == uuid.Nil {
		return nil, errors.New("warehouse_id is required")
	}
	if len(req.Items) == 0 {
		return nil, errors.New("at least one receipt item is required")
	}
	for i := range req.Items {
		req.Items[i].QualityStatus = strings.ToLower(strings.TrimSpace(req.Items[i].QualityStatus))
		if req.Items[i].QualityStatus == "" {
			req.Items[i].QualityStatus = "accepted"
		}
	}
	return s.repo.Create(ctx, actor, req)
}
func (s *WorkflowService) GetByID(ctx context.Context, id uuid.UUID) (*Receipt, error) {
	return s.repo.GetByID(ctx, id)
}
func (s *WorkflowService) List(ctx context.Context, page, limit int) ([]Receipt, int64, error) {
	return s.repo.List(ctx, page, limit)
}
