package inventory

import (
	"context"
	"errors"
	"math"
	"strings"

	"github.com/google/uuid"
)

// ErrNotFound is returned when the requested record does not exist.
var ErrNotFound = errors.New("not found")

// ValidationError carries per-field messages so the frontend can show them next to inputs.
type ValidationError struct {
	Fields map[string]string
}

func (e *ValidationError) Error() string { return "validation failed" }

func invalid(field, msg string) error {
	return &ValidationError{Fields: map[string]string{field: msg}}
}

type validator struct {
	fields map[string]string
}

func (v *validator) add(field, msg string) {
	if v.fields == nil {
		v.fields = map[string]string{}
	}
	if _, exists := v.fields[field]; !exists {
		v.fields[field] = msg
	}
}

func (v *validator) err() error {
	if len(v.fields) == 0 {
		return nil
	}
	return &ValidationError{Fields: v.fields}
}

func round4(v float64) float64 { return math.Round(v*10000) / 10000 }

func contains(list []string, v string) bool {
	for _, x := range list {
		if x == v {
			return true
		}
	}
	return false
}

func (v *validator) oneOf(field string, values, allowed []string) {
	for _, val := range values {
		if !contains(allowed, val) {
			v.add(field, "must be one of: "+strings.Join(allowed, ", "))
			return
		}
	}
}

func (v *validator) sortOrder(sort, order string, allowed map[string]string) {
	if sort != "" {
		if _, ok := allowed[sort]; !ok {
			keys := make([]string, 0, len(allowed))
			for k := range allowed {
				keys = append(keys, k)
			}
			v.add("sort", "unsupported sort field; allowed: "+strings.Join(sortedKeys(keys), ", "))
		}
	}
	if order != "" && order != "asc" && order != "desc" {
		v.add("order", "must be asc or desc")
	}
}

func sortedKeys(keys []string) []string {
	for i := 1; i < len(keys); i++ {
		for j := i; j > 0 && keys[j] < keys[j-1]; j-- {
			keys[j], keys[j-1] = keys[j-1], keys[j]
		}
	}
	return keys
}

// ====================== SERVICE ======================

type StockService struct {
	repo *StockRepository
}

func NewStockService(repo *StockRepository) *StockService {
	return &StockService{repo: repo}
}

func newResult[T any](items []T, total int64, p Pagination) ListResult[T] {
	return ListResult[T]{Items: items, Page: p.Page, Limit: p.Limit, Total: total}
}

// ---------- stock ----------

func (s *StockService) ListBalances(ctx context.Context, f StockFilter) (ListResult[StockBalance], error) {
	f.normalize()
	v := &validator{}
	v.oneOf("status", f.Statuses, StockStatuses)
	v.sortOrder(f.Sort, f.Order, stockSorts)
	if err := v.err(); err != nil {
		return ListResult[StockBalance]{}, err
	}
	list, total, err := s.repo.ListBalances(ctx, f)
	if err != nil {
		return ListResult[StockBalance]{}, err
	}
	return newResult(list, total, f.Pagination), nil
}

func (s *StockService) StockSummary(ctx context.Context, f StockFilter) (*StockSummary, error) {
	return s.repo.StockSummary(ctx, f)
}

// GetItemStock returns one item with its per-warehouse balances and totals.
func (s *StockService) GetItemStock(ctx context.Context, itemID uuid.UUID) (*ItemStock, error) {
	info, err := s.repo.GetItemInfo(ctx, itemID)
	if err != nil {
		return nil, err
	}
	rows, _, err := s.repo.ListBalances(ctx, StockFilter{
		Pagination: Pagination{Page: 1, Limit: 500},
		ItemID:     &itemID,
	})
	if err != nil {
		return nil, err
	}
	for _, b := range rows {
		info.TotalQuantity += b.Quantity
		info.TotalReserved += b.ReservedQuantity
		info.TotalAvailable += b.AvailableQuantity
	}
	// Quantities are NUMERIC(18,4); round away float drift from summing.
	info.TotalQuantity = round4(info.TotalQuantity)
	info.TotalReserved = round4(info.TotalReserved)
	info.TotalAvailable = round4(info.TotalAvailable)
	info.Warehouses = rows
	return info, nil
}

func (s *StockService) ListBatches(ctx context.Context, f BatchFilter) ([]StockBatch, error) {
	v := &validator{}
	v.oneOf("expiry_status", f.ExpiryStatuses, ExpiryStatuses)
	if err := v.err(); err != nil {
		return nil, err
	}
	// No implicit warehouse-type default: an item's batches are returned across every
	// warehouse unless the caller filters. (Defaulting to raw_material hid finished-goods batches.)
	return s.repo.ListBatches(ctx, f)
}

// ---------- movements ----------

func (s *StockService) ListMovements(ctx context.Context, f MovementFilter) (ListResult[StockMovement], error) {
	f.normalize()
	v := &validator{}
	if f.DateFrom != nil && f.DateTo != nil && f.DateFrom.After(*f.DateTo) {
		v.add("date_from", "cannot be after date_to")
	}
	if err := v.err(); err != nil {
		return ListResult[StockMovement]{}, err
	}
	list, total, err := s.repo.ListMovements(ctx, f)
	if err != nil {
		return ListResult[StockMovement]{}, err
	}
	return newResult(list, total, f.Pagination), nil
}

func (s *StockService) MovementTypes(ctx context.Context) ([]string, error) {
	return s.repo.MovementTypes(ctx)
}

// ---------- valuation ----------

func (s *StockService) ListValuation(ctx context.Context, f ValuationFilter) (ListResult[StockValuation], error) {
	f.normalize()
	v := &validator{}
	v.oneOf("cost_source", f.CostSources, CostSources)
	v.sortOrder(f.Sort, f.Order, valuationSorts)
	if err := v.err(); err != nil {
		return ListResult[StockValuation]{}, err
	}
	list, total, err := s.repo.ListValuation(ctx, f)
	if err != nil {
		return ListResult[StockValuation]{}, err
	}
	return newResult(list, total, f.Pagination), nil
}

func (s *StockService) ValuationSummary(ctx context.Context, f ValuationFilter) (*ValuationSummary, error) {
	v := &validator{}
	v.oneOf("cost_source", f.CostSources, CostSources)
	if err := v.err(); err != nil {
		return nil, err
	}
	return s.repo.ValuationSummary(ctx, f)
}

// ---------- alerts ----------

func (s *StockService) ListAlerts(ctx context.Context, f AlertFilter) (ListResult[InventoryAlert], error) {
	f.normalize()
	v := &validator{}
	v.oneOf("alert_type", f.AlertTypes, AlertTypes)
	v.oneOf("severity", f.Severities, AlertSeverities)
	if err := v.err(); err != nil {
		return ListResult[InventoryAlert]{}, err
	}
	list, total, err := s.repo.ListAlerts(ctx, f)
	if err != nil {
		return ListResult[InventoryAlert]{}, err
	}
	return newResult(list, total, f.Pagination), nil
}

func (s *StockService) AlertSummary(ctx context.Context, f AlertFilter) (*AlertSummary, error) {
	return s.repo.AlertSummary(ctx, f)
}
