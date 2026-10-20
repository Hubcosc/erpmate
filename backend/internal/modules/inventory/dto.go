package inventory

import (
	"time"

	"github.com/google/uuid"
)

const (
	defaultLimit             = 20
	maxLimit                 = 100
	defaultExpiryWarningDays = 30
)

// Allowed filter values. Exposed to the frontend via GET /lookups.
var (
	StockStatuses   = []string{"in_stock", "low_stock", "out_of_stock"}
	AlertTypes      = []string{"low_stock", "near_expiry", "expired"}
	AlertSeverities = []string{"critical", "warning", "info"}
	CostSources     = []string{"stock_batches", "inbound_movements", "no_cost_recorded"}
	ExpiryStatuses  = []string{"expired", "near_expiry", "ok", "no_expiry"}
)

// ====================== PAGINATION ======================

type Pagination struct {
	Page  int
	Limit int
}

func (p *Pagination) normalize() {
	if p.Page < 1 {
		p.Page = 1
	}
	if p.Limit < 1 || p.Limit > maxLimit {
		p.Limit = defaultLimit
	}
}

func (p Pagination) offset() int { return (p.Page - 1) * p.Limit }

// ListResult carries the normalized page/limit so response meta is always accurate.
type ListResult[T any] struct {
	Items []T
	Page  int
	Limit int
	Total int64
}

// ====================== FILTERS ======================

// StockFilter: GET /stock
// ?search=&item_type=&warehouse_type=&warehouse_id=&category_id=&status=low_stock,out_of_stock&sort=&order=
type StockFilter struct {
	Pagination
	ItemID        *uuid.UUID // internal: used by item detail
	Search        string
	ItemType      string
	WarehouseType string
	WarehouseID   *uuid.UUID
	CategoryID    *uuid.UUID
	Statuses      []string
	Sort          string
	Order         string
}

// BatchFilter: GET /stock/{itemID}/batches
// ?warehouse_type=&warehouse_id=&expiry_status=expired,near_expiry
type BatchFilter struct {
	ItemID         uuid.UUID
	WarehouseType  string
	WarehouseID    *uuid.UUID
	ExpiryStatuses []string
}

// MovementFilter: GET /movements
// ?item_id=&warehouse_id=&movement_type=receipt,issue&reference_type=&search=&date_from=YYYY-MM-DD&date_to=YYYY-MM-DD
type MovementFilter struct {
	Pagination
	ItemID        *uuid.UUID
	WarehouseID   *uuid.UUID
	MovementTypes []string
	ReferenceType string
	Search        string
	DateFrom      *time.Time
	DateTo        *time.Time // inclusive day
}

// ValuationFilter: GET /valuation and /valuation/summary
// ?search=&item_type=&warehouse_type=&warehouse_id=&cost_source=&low_stock=&sort=&order=
type ValuationFilter struct {
	Pagination
	Search        string
	ItemType      string
	WarehouseType string
	WarehouseID   *uuid.UUID
	CostSources   []string
	LowStock      *bool
	Sort          string
	Order         string
}

// AlertFilter: GET /alerts and /alerts/summary
// ?alert_type=low_stock,expired&severity=critical&warehouse_id=&search=
type AlertFilter struct {
	Pagination
	AlertTypes  []string
	Severities  []string
	WarehouseID *uuid.UUID
	Search      string
}
