package procurement

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	defaultLimit = 20
	maxLimit     = 100
	maxGRNLines  = 200
	maxPrice     = 9_999_999_999_999.99 // fits NUMERIC(18,2)
	maxQuantity  = 99_999_999_999_999.0 // fits NUMERIC(18,4)
)

var validPOStatuses = map[string]bool{
	"draft": true, "issued": true, "partially_received": true, "received": true, "cancelled": true,
}

type Service struct{ repo *WorkflowRepository }

func NewService(repo *WorkflowRepository) *Service { return &Service{repo: repo} }

// NormalizePaging clamps page/limit to sane values. The handler uses it too so
// the pagination meta always matches what was queried.
func NormalizePaging(page, limit int) (int, int) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > maxLimit {
		limit = defaultLimit
	}
	return page, limit
}

// ---------- purchase orders ----------

// CreateFromApprovedPR returns the PO and whether it was newly created.
func (s *Service) CreateFromApprovedPR(ctx context.Context, prID, actorID uuid.UUID) (*PurchaseOrder, bool, error) {
	return s.repo.CreateFromApprovedPR(ctx, prID, actorID)
}

func (s *Service) GetPurchaseOrder(ctx context.Context, id uuid.UUID) (*PurchaseOrder, error) {
	return s.repo.GetPurchaseOrder(ctx, id)
}

func (s *Service) ListPurchaseOrders(ctx context.Context, f PurchaseOrderFilter, page, limit int) ([]PurchaseOrder, int64, error) {
	f.Status = strings.ToLower(strings.TrimSpace(f.Status))
	if f.Status != "" && !validPOStatuses[f.Status] {
		return nil, 0, invalid("invalid status filter")
	}
	f.Search = strings.TrimSpace(f.Search)
	if len(f.Search) > 100 {
		f.Search = f.Search[:100]
	}
	page, limit = NormalizePaging(page, limit)
	return s.repo.ListPurchaseOrders(ctx, f, page, limit)
}

// UpdateDraftPurchaseOrder validates, applies the update and returns the fresh PO.
func (s *Service) UpdateDraftPurchaseOrder(ctx context.Context, id, actor uuid.UUID, req UpdatePurchaseOrderRequest) (*PurchaseOrder, error) {
	in := poUpdate{
		SupplierID:    req.SupplierID,
		PaymentTerms:  trimmed(req.PaymentTerms),
		DeliveryTerms: trimmed(req.DeliveryTerms),
		Notes:         trimmed(req.Notes),
	}
	if req.SupplierID != nil && *req.SupplierID == uuid.Nil {
		return nil, invalid("supplier_id is invalid")
	}
	if req.ExpectedDeliveryDate != nil {
		d, err := parseDate("expected_delivery_date", req.ExpectedDeliveryDate)
		if err != nil {
			return nil, err
		}
		in.SetExpected, in.Expected = true, d
	}
	seen := make(map[uuid.UUID]bool, len(req.Items))
	for _, it := range req.Items {
		if it.ID == uuid.Nil {
			return nil, invalid("every item requires an id")
		}
		if seen[it.ID] {
			return nil, invalid("duplicate item id in request")
		}
		seen[it.ID] = true
		if it.UnitPrice == nil {
			return nil, invalid("unit_price is required for every item")
		}
		p := *it.UnitPrice
		if math.IsNaN(p) || math.IsInf(p, 0) || p < 0 || p > maxPrice {
			return nil, invalid("unit_price must be a non-negative amount")
		}
		in.Items = append(in.Items, poPrice{ID: it.ID, Price: round2(p)})
	}
	if err := s.repo.UpdateDraftPurchaseOrder(ctx, id, actor, in); err != nil {
		return nil, err
	}
	return s.repo.GetPurchaseOrder(ctx, id)
}

// IssuePurchaseOrder issues a draft PO and returns the fresh PO.
func (s *Service) IssuePurchaseOrder(ctx context.Context, id, actor uuid.UUID) (*PurchaseOrder, error) {
	if err := s.repo.IssuePurchaseOrder(ctx, id, actor); err != nil {
		return nil, err
	}
	return s.repo.GetPurchaseOrder(ctx, id)
}

// ---------- goods receipt notes ----------

func (s *Service) CreateGRN(ctx context.Context, actor uuid.UUID, req CreateGoodsReceiptNoteRequest) (*GoodsReceiptNote, error) {
	if req.PurchaseOrderID == uuid.Nil {
		return nil, invalid("purchase_order_id is required")
	}
	if req.WarehouseID == uuid.Nil {
		return nil, invalid("warehouse_id is required")
	}
	if len(req.Items) == 0 {
		return nil, invalid("at least one receipt item is required")
	}
	if len(req.Items) > maxGRNLines {
		return nil, invalid(fmt.Sprintf("a receipt can have at most %d lines", maxGRNLines))
	}
	invoice := emptyToNil(trimmed(req.SupplierInvoiceReference))
	if invoice != nil && len(*invoice) > 120 {
		return nil, invalid("supplier_invoice_reference must be at most 120 characters")
	}
	in := grnInput{
		PurchaseOrderID: req.PurchaseOrderID,
		WarehouseID:     req.WarehouseID,
		InvoiceRef:      invoice,
		Notes:           emptyToNil(trimmed(req.Notes)),
	}
	today := time.Now().UTC().Truncate(24 * time.Hour)
	for i, l := range req.Items {
		n := i + 1
		if l.PurchaseOrderItemID == uuid.Nil {
			return nil, invalid(fmt.Sprintf("line %d: purchase_order_item_id is required", n))
		}
		if math.IsNaN(l.QuantityReceived) || math.IsInf(l.QuantityReceived, 0) || l.QuantityReceived <= 0 || l.QuantityReceived > maxQuantity {
			return nil, invalid(fmt.Sprintf("line %d: received quantity must be greater than 0", n))
		}
		qty := round4(l.QuantityReceived)
		if qty <= 0 {
			return nil, invalid(fmt.Sprintf("line %d: received quantity must be greater than 0", n))
		}
		line := grnLine{PurchaseOrderItemID: l.PurchaseOrderItemID, Quantity: qty}
		if l.UnitCost != nil {
			c := *l.UnitCost
			if math.IsNaN(c) || math.IsInf(c, 0) || c < 0 || c > maxPrice {
				return nil, invalid(fmt.Sprintf("line %d: unit_cost cannot be negative", n))
			}
			c = round2(c)
			line.UnitCost = &c
		}
		if b := emptyToNil(trimmed(l.BatchNumber)); b != nil {
			if len(*b) > 100 {
				return nil, invalid(fmt.Sprintf("line %d: batch_number must be at most 100 characters", n))
			}
			line.BatchNumber = *b
		}
		if loc := emptyToNil(trimmed(l.Location)); loc != nil {
			if len(*loc) > 120 {
				return nil, invalid(fmt.Sprintf("line %d: location must be at most 120 characters", n))
			}
			line.Location = loc
		}
		var err error
		if line.Manufacture, err = parseDate(fmt.Sprintf("line %d: manufacture_date", n), l.ManufactureDate); err != nil {
			return nil, err
		}
		if line.Expiry, err = parseDate(fmt.Sprintf("line %d: expiry_date", n), l.ExpiryDate); err != nil {
			return nil, err
		}
		if line.Expiry != nil {
			if line.Manufacture != nil && line.Expiry.Before(*line.Manufacture) {
				return nil, invalid(fmt.Sprintf("line %d: expiry_date cannot be before manufacture_date", n))
			}
			if line.Expiry.Before(today) {
				return nil, invalid(fmt.Sprintf("line %d: item has already expired", n))
			}
		}
		in.Lines = append(in.Lines, line)
	}
	return s.repo.CreateGRN(ctx, actor, in)
}

func (s *Service) GetGRN(ctx context.Context, id uuid.UUID) (*GoodsReceiptNote, error) {
	return s.repo.GetGRN(ctx, id)
}

func (s *Service) ListGRNs(ctx context.Context, poID *uuid.UUID, page, limit int) ([]GoodsReceiptNote, int64, error) {
	page, limit = NormalizePaging(page, limit)
	return s.repo.ListGRNs(ctx, poID, page, limit)
}

// ---------- helpers ----------

// parseDate parses an optional YYYY-MM-DD value. nil/blank returns (nil, nil).
func parseDate(field string, value *string) (*time.Time, error) {
	if value == nil || strings.TrimSpace(*value) == "" {
		return nil, nil
	}
	d, err := time.Parse("2006-01-02", strings.TrimSpace(*value))
	if err != nil {
		return nil, invalid(field + " must use YYYY-MM-DD")
	}
	return &d, nil
}

// trimmed returns a trimmed copy; nil stays nil (meaning "not supplied").
func trimmed(p *string) *string {
	if p == nil {
		return nil
	}
	v := strings.TrimSpace(*p)
	return &v
}

func emptyToNil(p *string) *string {
	if p == nil || *p == "" {
		return nil
	}
	return p
}
