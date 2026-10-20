package purchase

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
)

const (
	maxItems         = 200
	maxJustification = 2000
	maxDescription   = 500
	maxNotes         = 1000
	maxUnit          = 50
	maxReason        = 1000
)

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

// ---------------------------------------------------------------------------
// Validation helpers
// ---------------------------------------------------------------------------

func round2(v float64) float64 { return math.Round(v*100) / 100 }

// cleanText trims s; empty becomes nil; too long is a validation error.
func cleanText(s *string, max int, field string) (*string, error) {
	if s == nil {
		return nil, nil
	}
	v := strings.TrimSpace(*s)
	if v == "" {
		return nil, nil
	}
	if utf8.RuneCountInString(v) > max {
		return nil, Validation(fmt.Sprintf("%s must be at most %d characters", field, max))
	}
	return &v, nil
}

// parseRequiredDate parses YYYY-MM-DD and rejects dates in the past, unless the
// date is unchanged from the existing value (so old drafts stay editable).
func parseRequiredDate(s *string, existing *time.Time) (*time.Time, error) {
	if s == nil || strings.TrimSpace(*s) == "" {
		return nil, nil
	}
	t, err := time.Parse("2006-01-02", strings.TrimSpace(*s))
	if err != nil {
		return nil, Validation("invalid required_date format, use YYYY-MM-DD")
	}
	today := time.Now().Format("2006-01-02")
	unchanged := existing != nil && existing.Format("2006-01-02") == t.Format("2006-01-02")
	if t.Format("2006-01-02") < today && !unchanged {
		return nil, Validation("required_date cannot be in the past")
	}
	return &t, nil
}

// buildItems validates and normalises the request items and computes totals.
func buildItems(src []CreatePRItem) ([]PRItem, float64, error) {
	if len(src) == 0 {
		return nil, 0, Validation("at least one item is required")
	}
	if len(src) > maxItems {
		return nil, 0, Validation(fmt.Sprintf("a requisition can have at most %d items", maxItems))
	}

	items := make([]PRItem, 0, len(src))
	var total float64
	for i, in := range src {
		n := i + 1
		desc, err := cleanText(in.Description, maxDescription, fmt.Sprintf("item %d description", n))
		if err != nil {
			return nil, 0, err
		}
		unit, err := cleanText(in.UnitOfMeasure, maxUnit, fmt.Sprintf("item %d unit_of_measure", n))
		if err != nil {
			return nil, 0, err
		}
		notes, err := cleanText(in.Notes, maxNotes, fmt.Sprintf("item %d notes", n))
		if err != nil {
			return nil, 0, err
		}

		itemID := in.ItemID
		if itemID != nil && *itemID == uuid.Nil {
			itemID = nil
		}
		if itemID == nil && desc == nil {
			return nil, 0, Validation(fmt.Sprintf("item %d: select an item or enter a description", n))
		}
		if !(in.Quantity > 0) || math.IsInf(in.Quantity, 0) {
			return nil, 0, Validation(fmt.Sprintf("item %d: quantity must be greater than 0", n))
		}
		if in.EstimatedUnitPrice != nil && (*in.EstimatedUnitPrice < 0 || math.IsInf(*in.EstimatedUnitPrice, 0)) {
			return nil, 0, Validation(fmt.Sprintf("item %d: estimated unit price cannot be negative", n))
		}

		var lineTotal *float64
		if in.EstimatedUnitPrice != nil {
			v := round2(in.Quantity * *in.EstimatedUnitPrice)
			lineTotal = &v
			total += v
		}
		items = append(items, PRItem{
			ItemID:             itemID,
			Description:        desc,
			Quantity:           in.Quantity,
			UnitOfMeasure:      unit,
			EstimatedUnitPrice: in.EstimatedUnitPrice,
			EstimatedTotal:     lineTotal,
			Notes:              notes,
		})
	}
	return items, round2(total), nil
}

// stockRequests aggregates the catalogue items of a requisition (the same item
// on two lines is checked against its combined quantity), keeping line order.
func stockRequests(items []PRItem) []StockRequest {
	reqs := make([]StockRequest, 0, len(items))
	index := make(map[uuid.UUID]int, len(items))
	for _, it := range items {
		if it.ItemID == nil {
			continue
		}
		if pos, ok := index[*it.ItemID]; ok {
			reqs[pos].Quantity += it.Quantity
			continue
		}
		index[*it.ItemID] = len(reqs)
		reqs = append(reqs, StockRequest{ItemID: *it.ItemID, Quantity: it.Quantity})
	}
	return reqs
}

type preparedPR struct {
	requiredDate  *time.Time
	justification *string
	items         []PRItem
	total         float64
}

// prepare validates the shared parts of create/update. It also verifies that
// every catalogue item exists, so the user gets a clear message instead of a
// foreign-key error.
func (s *Service) prepare(ctx context.Context, requiredDate *string, existingDate *time.Time, justification *string, src []CreatePRItem) (*preparedPR, error) {
	date, err := parseRequiredDate(requiredDate, existingDate)
	if err != nil {
		return nil, err
	}
	just, err := cleanText(justification, maxJustification, "justification")
	if err != nil {
		return nil, err
	}
	items, total, err := buildItems(src)
	if err != nil {
		return nil, err
	}
	if _, err := s.repo.CheckStock(ctx, stockRequests(items)); err != nil {
		return nil, err
	}
	return &preparedPR{requiredDate: date, justification: just, items: items, total: total}, nil
}

// ---------------------------------------------------------------------------
// Permissions and UI helpers
// ---------------------------------------------------------------------------

func (a Actor) isAdmin() bool { return a.Role == RoleAdmin }

// availableActions tells the UI which buttons to show for a requisition.
func availableActions(status string, requestedBy uuid.UUID, a Actor) []string {
	actions := make([]string, 0, 4)
	isOwner := requestedBy == a.UserID

	if isOwner && (a.Role == RoleDepartmentHead || a.isAdmin()) && IsEditableStatus(status) {
		actions = append(actions, ActionEdit, ActionSubmit, ActionDelete)
	}
	if isOwner { // nobody reviews their own requisition
		return actions
	}
	if a.Role == RoleProductionManager || a.isAdmin() {
		if status == StatusPendingPM {
			actions = append(actions, ActionPMApprove, ActionPMReturn, ActionPMReject)
		}
	}
	if a.Role == RoleFinance || a.isAdmin() {
		switch status {
		case StatusPendingFinance:
			actions = append(actions, ActionFinanceApprove, ActionFinanceSuspend, ActionFinanceReject)
		case StatusSuspended:
			actions = append(actions, ActionFinanceApprove, ActionFinanceReject)
		}
	}
	return actions
}

func (s *Service) attachStock(ctx context.Context, pr *PurchaseRequisition) error {
	checks, err := s.repo.CheckStock(ctx, stockRequests(pr.Items))
	if err != nil {
		var perr *Error
		if errors.As(err, &perr) && perr.Kind == KindNotFound {
			// A catalogue item was deleted after the PR was created: keep the
			// requisition viewable, just without stock information.
			pr.StockCheck = []PRStockCheck{}
			pr.SuggestMaterialRequisition = false
			return nil
		}
		return err
	}
	pr.StockCheck = checks
	// Suggest a material requisition only when everything is already in stock.
	pr.SuggestMaterialRequisition = len(checks) > 0
	for _, c := range checks {
		if !c.Sufficient {
			pr.SuggestMaterialRequisition = false
			break
		}
	}
	return nil
}

func (s *Service) authorizeView(ctx context.Context, id uuid.UUID, a Actor) error {
	if a.Role != RoleDepartmentHead {
		if _, err := s.repo.GetRequester(ctx, id); err != nil {
			return err // 404 for unknown ids
		}
		return nil
	}
	requester, err := s.repo.GetRequester(ctx, id)
	if err != nil {
		return err
	}
	if requester != a.UserID {
		return Forbidden("you cannot view this requisition")
	}
	return nil
}

// ---------------------------------------------------------------------------
// Queries
// ---------------------------------------------------------------------------

func (s *Service) Get(ctx context.Context, actor Actor, id uuid.UUID) (*PurchaseRequisition, error) {
	pr, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if actor.Role == RoleDepartmentHead && pr.RequestedBy != actor.UserID {
		return nil, Forbidden("you cannot view this requisition")
	}
	if err := s.attachStock(ctx, pr); err != nil {
		return nil, err
	}
	pr.AvailableActions = availableActions(pr.Status, pr.RequestedBy, actor)
	return pr, nil
}

func (s *Service) List(ctx context.Context, actor Actor, f ListFilter) ([]PRSummary, int64, error) {
	if f.Page < 1 {
		f.Page = 1
	}
	if f.Limit < 1 || f.Limit > 100 {
		f.Limit = 20
	}
	list, total, err := s.repo.List(ctx, actor, f)
	if err != nil {
		return nil, 0, err
	}
	for i := range list {
		list[i].AvailableActions = availableActions(list[i].Status, list[i].RequestedBy, actor)
	}
	return list, total, nil
}

// Summary returns per-status counts plus how many items wait on the actor.
func (s *Service) Summary(ctx context.Context, actor Actor) (*StatusSummary, error) {
	counts, err := s.repo.StatusCounts(ctx, actor)
	if err != nil {
		return nil, err
	}
	sum := &StatusSummary{Counts: counts}
	for _, n := range counts {
		sum.Total += n
	}
	switch actor.Role {
	case RoleProductionManager:
		sum.ActionRequired = counts[StatusPendingPM]
	case RoleFinance:
		sum.ActionRequired = counts[StatusPendingFinance] + counts[StatusSuspended]
	case RoleDepartmentHead:
		sum.ActionRequired = counts[StatusReturned] + counts[StatusDraft]
	case RoleAdmin:
		sum.ActionRequired = counts[StatusPendingPM] + counts[StatusPendingFinance] + counts[StatusSuspended]
	}
	return sum, nil
}

func (s *Service) GetHistory(ctx context.Context, actor Actor, id uuid.UUID) ([]StatusHistory, error) {
	if err := s.authorizeView(ctx, id, actor); err != nil {
		return nil, err
	}
	return s.repo.GetStatusHistory(ctx, id)
}

// ---------------------------------------------------------------------------
// Commands
// ---------------------------------------------------------------------------

func (s *Service) Create(ctx context.Context, actor Actor, req CreatePRRequest) (*PurchaseRequisition, error) {
	if req.DepartmentID == uuid.Nil {
		return nil, Validation("department_id is required")
	}
	p, err := s.prepare(ctx, req.RequiredDate, nil, req.Justification, req.Items)
	if err != nil {
		return nil, err
	}

	pr := &PurchaseRequisition{
		ID:                  uuid.New(),
		DepartmentID:        req.DepartmentID,
		PreferredSupplierID: req.PreferredSupplierID,
		RequestedBy:         actor.UserID,
		RequiredDate:        p.requiredDate,
		Justification:       p.justification,
		Status:              StatusDraft,
		TotalEstimated:      p.total,
		Items:               p.items,
	}
	if err := s.repo.Create(ctx, pr); err != nil {
		return nil, err
	}
	return s.Get(ctx, actor, pr.ID)
}

func (s *Service) Update(ctx context.Context, actor Actor, id uuid.UUID, req UpdatePRRequest) (*PurchaseRequisition, error) {
	existing, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if existing.RequestedBy != actor.UserID {
		return nil, Forbidden("you can only update your own requisitions")
	}
	if !IsEditableStatus(existing.Status) {
		return nil, Conflict("only draft or returned requisitions can be updated")
	}

	p, err := s.prepare(ctx, req.RequiredDate, existing.RequiredDate, req.Justification, req.Items)
	if err != nil {
		return nil, err
	}
	deptID := req.DepartmentID
	if deptID == uuid.Nil {
		deptID = existing.DepartmentID
	}

	err = s.repo.Update(ctx, UpdateParams{
		ID:            id,
		UserID:        actor.UserID,
		DepartmentID:  deptID,
		SupplierID:    req.PreferredSupplierID,
		RequiredDate:  p.requiredDate,
		Justification: p.justification,
		Total:         p.total,
		Items:         p.items,
	})
	if err != nil {
		return nil, err
	}
	return s.Get(ctx, actor, id)
}

func (s *Service) Delete(ctx context.Context, actor Actor, id uuid.UUID) error {
	return s.repo.SoftDelete(ctx, id, actor.UserID)
}

// ---------------------------------------------------------------------------
// Workflow transitions
// ---------------------------------------------------------------------------

type transition struct {
	from          []string
	to            string
	ownerOnly     bool   // only the requester may perform it (submit)
	requireReason bool   // reject / return / suspend
	reasonLabel   string // used in the "reason is required" message
	stateMsg      string // shown when the current status does not allow it
	fields        func(actor Actor, reason *string, now time.Time) map[string]any
}

var (
	trSubmit = transition{
		from:      []string{StatusDraft, StatusReturned},
		to:        StatusPendingPM,
		ownerOnly: true,
		stateMsg:  "only draft or returned requisitions can be submitted",
		fields: func(Actor, *string, time.Time) map[string]any {
			return map[string]any{"return_reason": nil}
		},
	}
	trPMApprove = transition{
		from:     []string{StatusPendingPM},
		to:       StatusPendingFinance,
		stateMsg: "only pending_pm requisitions can be approved by PM",
		fields: func(a Actor, _ *string, now time.Time) map[string]any {
			return map[string]any{"pm_approved_by": a.UserID, "pm_approved_at": now}
		},
	}
	trPMReject = transition{
		from:          []string{StatusPendingPM},
		to:            StatusRejected,
		requireReason: true,
		reasonLabel:   "rejection",
		stateMsg:      "only pending_pm requisitions can be rejected by PM",
		fields: func(_ Actor, r *string, _ time.Time) map[string]any {
			return map[string]any{"pm_rejection_reason": r}
		},
	}
	trPMReturn = transition{
		from:          []string{StatusPendingPM},
		to:            StatusReturned,
		requireReason: true,
		reasonLabel:   "return",
		stateMsg:      "only pending_pm requisitions can be returned",
		fields: func(_ Actor, r *string, _ time.Time) map[string]any {
			return map[string]any{"return_reason": r}
		},
	}
	trFinanceApprove = transition{
		from:     []string{StatusPendingFinance, StatusSuspended},
		to:       StatusApproved,
		stateMsg: "only pending_finance or suspended requisitions can be approved by Finance",
		fields: func(a Actor, _ *string, now time.Time) map[string]any {
			return map[string]any{"finance_approved_by": a.UserID, "finance_approved_at": now, "suspension_reason": nil}
		},
	}
	trFinanceSuspend = transition{
		from:          []string{StatusPendingFinance},
		to:            StatusSuspended,
		requireReason: true,
		reasonLabel:   "suspension",
		stateMsg:      "only pending_finance requisitions can be suspended",
		fields: func(_ Actor, r *string, _ time.Time) map[string]any {
			return map[string]any{"suspension_reason": r}
		},
	}
	trFinanceReject = transition{
		from:          []string{StatusPendingFinance, StatusSuspended},
		to:            StatusRejected,
		requireReason: true,
		reasonLabel:   "rejection",
		stateMsg:      "only pending_finance or suspended requisitions can be rejected by Finance",
		fields: func(_ Actor, r *string, _ time.Time) map[string]any {
			return map[string]any{"finance_rejection_reason": r}
		},
	}
)

// transit runs one workflow transition and returns the refreshed requisition.
func (s *Service) transit(ctx context.Context, actor Actor, id uuid.UUID, reason *string, t transition) (*PurchaseRequisition, error) {
	reason, err := cleanText(reason, maxReason, "reason")
	if err != nil {
		return nil, err
	}
	if t.requireReason && reason == nil {
		return nil, Validation(t.reasonLabel + " reason is required")
	}

	pr, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if t.ownerOnly && pr.RequestedBy != actor.UserID {
		return nil, Forbidden("you can only submit your own requisitions")
	}
	if !t.ownerOnly && pr.RequestedBy == actor.UserID {
		return nil, Forbidden("you cannot review your own requisition")
	}
	if !containsString(t.from, pr.Status) {
		return nil, Conflict(t.stateMsg)
	}

	var fields map[string]any
	if t.fields != nil {
		fields = t.fields(actor, reason, time.Now())
	}
	if err := s.repo.UpdateStatus(ctx, id, t.from, t.to, actor.UserID, reason, fields); err != nil {
		return nil, err
	}
	return s.Get(ctx, actor, id)
}

func (s *Service) Submit(ctx context.Context, actor Actor, id uuid.UUID, reason *string) (*PurchaseRequisition, error) {
	return s.transit(ctx, actor, id, reason, trSubmit)
}

func (s *Service) PMApprove(ctx context.Context, actor Actor, id uuid.UUID, reason *string) (*PurchaseRequisition, error) {
	return s.transit(ctx, actor, id, reason, trPMApprove)
}

func (s *Service) PMReject(ctx context.Context, actor Actor, id uuid.UUID, reason *string) (*PurchaseRequisition, error) {
	return s.transit(ctx, actor, id, reason, trPMReject)
}

func (s *Service) PMReturn(ctx context.Context, actor Actor, id uuid.UUID, reason *string) (*PurchaseRequisition, error) {
	return s.transit(ctx, actor, id, reason, trPMReturn)
}

func (s *Service) FinanceApprove(ctx context.Context, actor Actor, id uuid.UUID, reason *string) (*PurchaseRequisition, error) {
	return s.transit(ctx, actor, id, reason, trFinanceApprove)
}

func (s *Service) FinanceSuspend(ctx context.Context, actor Actor, id uuid.UUID, reason *string) (*PurchaseRequisition, error) {
	return s.transit(ctx, actor, id, reason, trFinanceSuspend)
}

func (s *Service) FinanceReject(ctx context.Context, actor Actor, id uuid.UUID, reason *string) (*PurchaseRequisition, error) {
	return s.transit(ctx, actor, id, reason, trFinanceReject)
}
