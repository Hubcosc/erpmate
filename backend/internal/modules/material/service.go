package material

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Create(ctx context.Context, userID uuid.UUID, req CreateMRRequest) (*MaterialRequisition, error) {
	if len(req.Items) == 0 {
		return nil, errors.New("at least one item is required")
	}

	mrNumber, err := s.repo.GenerateMRNumber(ctx)
	if err != nil {
		return nil, err
	}

	var requiredDate *time.Time
	if req.RequiredDate != nil && *req.RequiredDate != "" {
		t, err := time.Parse("2006-01-02", *req.RequiredDate)
		if err != nil {
			return nil, errors.New("invalid required_date format, use YYYY-MM-DD")
		}
		requiredDate = &t
	}

	mr := &MaterialRequisition{
		ID:           uuid.New(),
		MRNumber:     mrNumber,
		DepartmentID: req.DepartmentID,
		RequestedBy:  userID,
		RequiredDate: requiredDate,
		Purpose:      req.Purpose,
		Status:       "draft",
	}

	for _, item := range req.Items {
		if item.Quantity <= 0 {
			return nil, errors.New("quantity must be greater than 0")
		}
		mr.Items = append(mr.Items, MRItem{
			ItemID:   item.ItemID,
			Quantity: item.Quantity,
			Notes:    item.Notes,
		})
		available, reserved, err := s.repo.CheckStock(ctx, item.ItemID)
		if err != nil {
			return nil, err
		}
		mr.Items[len(mr.Items)-1].AvailableQty = available
		mr.Items[len(mr.Items)-1].ReservedQty = reserved
		mr.Items[len(mr.Items)-1].StockSufficient = available >= item.Quantity
		mr.Items[len(mr.Items)-1].SuggestPurchaseRequisition = available < item.Quantity
	}

	if err := s.repo.Create(ctx, mr); err != nil {
		return nil, err
	}

	return s.repo.GetByID(ctx, mr.ID)
}

func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (*MaterialRequisition, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *Service) List(ctx context.Context, userID uuid.UUID, role, status string, page, limit int) ([]MaterialRequisition, int64, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}
	return s.repo.List(ctx, userID, role, status, page, limit)
}

func (s *Service) Submit(ctx context.Context, id, userID uuid.UUID) error {
	mr, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return errors.New("material requisition not found")
	}
	if mr.Status != "draft" {
		return errors.New("only draft requisitions can be submitted")
	}
	if mr.RequestedBy != userID {
		return errors.New("you can only submit your own requisitions")
	}
	return s.repo.UpdateStatus(ctx, id, "pending_pm", userID, nil, nil)
}

func (s *Service) PMApprove(ctx context.Context, id, userID uuid.UUID) error {
	mr, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return errors.New("material requisition not found")
	}
	if mr.Status != "pending_pm" {
		return errors.New("only pending_pm requisitions can be approved")
	}

	now := time.Now()
	fields := map[string]interface{}{
		"pm_approved_by": userID,
		"pm_approved_at": now,
	}
	return s.repo.UpdateStatus(ctx, id, "approved", userID, nil, fields)
}

func (s *Service) PMReject(ctx context.Context, id, userID uuid.UUID, reason *string) error {
	if reason == nil || strings.TrimSpace(*reason) == "" {
		return errors.New("rejection reason is required")
	}
	mr, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return errors.New("material requisition not found")
	}
	if mr.Status != "pending_pm" {
		return errors.New("only pending_pm requisitions can be rejected")
	}

	fields := map[string]interface{}{
		"pm_rejection_reason": reason,
	}
	return s.repo.UpdateStatus(ctx, id, "rejected", userID, reason, fields)
}

func (s *Service) GetHistory(ctx context.Context, id uuid.UUID) ([]StatusHistory, error) {
	return s.repo.GetStatusHistory(ctx, id)
}

func (s *Service) CanView(ctx context.Context, id, userID uuid.UUID, role string) (bool, error) {
	if role != "department_head" {
		return true, nil
	}
	return s.repo.IsRequester(ctx, id, userID)
}

func (s *Service) CanViewIssue(ctx context.Context, id, userID uuid.UUID, role string) (bool, error) {
	if role == "admin" || role == "store_keeper" {
		return true, nil
	}
	return s.repo.CanViewIssue(ctx, id, userID)
}

func (s *Service) CanListIssues(ctx context.Context, mrID, userID uuid.UUID, role string) (bool, error) {
	if role == "admin" || role == "store_keeper" {
		return true, nil
	}
	if role != "department_head" {
		return false, nil
	}
	return s.repo.IsRequester(ctx, mrID, userID)
}

func (s *Service) AcknowledgeIssue(ctx context.Context, id, userID uuid.UUID) error {
	return s.repo.AcknowledgeIssue(ctx, id, userID)
}

func (s *Service) CreateIssue(ctx context.Context, userID uuid.UUID, req CreateIssueRequest) (*MaterialIssue, error) {
	if len(req.Items) == 0 {
		return nil, errors.New("at least one item is required")
	}

	// Check MR is approved or partially issued
	mr, err := s.repo.GetByID(ctx, req.MRID)
	if err != nil {
		return nil, errors.New("material requisition not found")
	}
	if mr.Status != "approved" && mr.Status != "partially_issued" {
		return nil, errors.New("only approved or partially_issued requisitions can be issued")
	}

	issueNumber, err := s.repo.GenerateIssueNumber(ctx)
	if err != nil {
		return nil, err
	}

	issue := &MaterialIssue{
		ID:          uuid.New(),
		IssueNumber: issueNumber,
		MRID:        req.MRID,
		IssuedBy:    userID,
		IssuedTo:    req.IssuedTo,
		Notes:       req.Notes,
	}

	for _, item := range req.Items {
		if item.Quantity <= 0 {
			return nil, errors.New("quantity must be greater than 0")
		}
		issue.Items = append(issue.Items, MaterialIssueItem{
			MRItemID: item.MRItemID,
			ItemID:   item.ItemID,
			BatchID:  item.BatchID,
			Quantity: item.Quantity,
		})
	}

	if err := s.repo.CreateIssue(ctx, issue); err != nil {
		return nil, err
	}

	return s.repo.GetIssueByID(ctx, issue.ID)
}

func (s *Service) GetIssueByID(ctx context.Context, id uuid.UUID) (*MaterialIssue, error) {
	return s.repo.GetIssueByID(ctx, id)
}

func (s *Service) ListIssuesByMR(ctx context.Context, mrID uuid.UUID) ([]MaterialIssue, error) {
	return s.repo.ListIssuesByMR(ctx, mrID)
}
