package sales

import (
	"context"
	"errors"
	"github.com/google/uuid"
	"strings"
	"time"
)

type WorkflowService struct{ repo *WorkflowRepository }

func NewWorkflowService(repo *WorkflowRepository) *WorkflowService {
	return &WorkflowService{repo: repo}
}
func (s *WorkflowService) CreateOrder(ctx context.Context, actor uuid.UUID, req CreateSalesOrderRequest) (*SalesOrder, error) {
	if req.CustomerID == uuid.Nil {
		return nil, errors.New("customer_id is required")
	}
	if len(req.Items) == 0 {
		return nil, errors.New("at least one order item is required")
	}
	return s.repo.CreateOrder(ctx, actor, req)
}
func (s *WorkflowService) GetOrder(ctx context.Context, id uuid.UUID) (*SalesOrder, error) {
	return s.repo.GetOrder(ctx, id)
}
func (s *WorkflowService) ListOrders(ctx context.Context, status string, page, limit int) ([]SalesOrder, int64, error) {
	return s.repo.ListOrders(ctx, status, page, limit)
}
func (s *WorkflowService) CreateDelivery(ctx context.Context, orderID, actor uuid.UUID, req CreateDeliveryNoteRequest) (*DeliveryNote, error) {
	return s.repo.CreateDelivery(ctx, orderID, actor, req)
}
func (s *WorkflowService) GetDelivery(ctx context.Context, id uuid.UUID) (*DeliveryNote, error) {
	return s.repo.GetDelivery(ctx, id)
}
func (s *WorkflowService) ListDeliveries(ctx context.Context, orderID *uuid.UUID, page, limit int) ([]DeliveryNote, int64, error) {
	return s.repo.ListDeliveries(ctx, orderID, page, limit)
}
func parseSalesDate(raw *string) (*time.Time, error) {
	if raw == nil || strings.TrimSpace(*raw) == "" {
		return nil, nil
	}
	value, err := time.Parse("2006-01-02", *raw)
	if err != nil {
		return nil, errors.New("date must use YYYY-MM-DD")
	}
	return &value, nil
}
