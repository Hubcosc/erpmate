package inventory

import (
	"time"

	"github.com/google/uuid"
)

// ====================== STOCK ======================

type StockBalance struct {
	ItemID            uuid.UUID `json:"item_id"`
	ItemCode          string    `json:"item_code"`
	ItemName          string    `json:"item_name"`
	ItemType          string    `json:"item_type"`
	UnitOfMeasure     string    `json:"unit_of_measure"`
	CategoryName      *string   `json:"category_name"`
	WarehouseID       uuid.UUID `json:"warehouse_id"`
	WarehouseCode     string    `json:"warehouse_code"`
	WarehouseName     string    `json:"warehouse_name"`
	WarehouseType     string    `json:"warehouse_type"`
	Quantity          float64   `json:"quantity"`
	ReservedQuantity  float64   `json:"reserved_quantity"`
	AvailableQuantity float64   `json:"available_quantity"`
	MinStockLevel     float64   `json:"min_stock_level"`
	MaxStockLevel     *float64  `json:"max_stock_level"`
	ReorderLevel      *float64  `json:"reorder_level"`
	// Threshold is the effective alert level: reorder_level, falling back to min_stock_level.
	Threshold float64 `json:"threshold"`
	// LowStock is true when Threshold > 0 and available <= Threshold (includes zero stock).
	LowStock bool `json:"low_stock"`
	// StockStatus is one of: in_stock, low_stock, out_of_stock.
	StockStatus string `json:"stock_status"`
}

// ItemStock is the detail view for one item across all warehouses.
type ItemStock struct {
	ItemID          uuid.UUID      `json:"item_id"`
	ItemCode        string         `json:"item_code"`
	ItemName        string         `json:"item_name"`
	ItemType        string         `json:"item_type"`
	UnitOfMeasure   string         `json:"unit_of_measure"`
	CategoryName    *string        `json:"category_name"`
	MinStockLevel   float64        `json:"min_stock_level"`
	MaxStockLevel   *float64       `json:"max_stock_level"`
	ReorderLevel    *float64       `json:"reorder_level"`
	IsBatchTracked  bool           `json:"is_batch_tracked"`
	IsExpiryTracked bool           `json:"is_expiry_tracked"`
	TotalQuantity   float64        `json:"total_quantity"`
	TotalReserved   float64        `json:"total_reserved"`
	TotalAvailable  float64        `json:"total_available"`
	Warehouses      []StockBalance `json:"warehouses"`
}

// StockSummary feeds the status tabs / KPI cards above the stock table.
type StockSummary struct {
	TotalLines int64 `json:"total_lines"` // item x warehouse rows
	TotalItems int64 `json:"total_items"` // distinct items
	InStock    int64 `json:"in_stock"`
	LowStock   int64 `json:"low_stock"`
	OutOfStock int64 `json:"out_of_stock"`
}

type StockBatch struct {
	ID            uuid.UUID `json:"id"`
	ItemID        uuid.UUID `json:"item_id"`
	WarehouseID   uuid.UUID `json:"warehouse_id"`
	WarehouseCode string    `json:"warehouse_code"`
	WarehouseName string    `json:"warehouse_name"`
	BatchNumber   string    `json:"batch_number"`
	Quantity      float64   `json:"quantity"`
	// AvailableQuantity is how much of this batch can still be picked, allocating the
	// item's available balance in FEFO order (earliest expiry first).
	AvailableQuantity float64   `json:"available_quantity"`
	ManufactureDate   *string   `json:"manufacture_date"`
	ExpiryDate        *string   `json:"expiry_date"`
	DaysUntilExpiry   *int      `json:"days_until_expiry"`
	ExpiryStatus      string    `json:"expiry_status"` // expired, near_expiry, ok, no_expiry
	UnitCost          *float64  `json:"unit_cost"`
	InventoryValue    *float64  `json:"inventory_value"`
	CreatedAt         time.Time `json:"created_at"`
}

// ====================== MOVEMENTS ======================

type StockMovement struct {
	ID              uuid.UUID  `json:"id"`
	MovementType    string     `json:"movement_type"`
	ReferenceType   *string    `json:"reference_type"`
	ReferenceID     *uuid.UUID `json:"reference_id"`
	ItemID          uuid.UUID  `json:"item_id"`
	ItemCode        string     `json:"item_code"`
	ItemName        string     `json:"item_name"`
	UnitOfMeasure   string     `json:"unit_of_measure"`
	WarehouseID     uuid.UUID  `json:"warehouse_id"`
	WarehouseName   string     `json:"warehouse_name"`
	BatchID         *uuid.UUID `json:"batch_id"`
	BatchNumber     *string    `json:"batch_number"`
	Quantity        float64    `json:"quantity"`
	UnitCost        *float64   `json:"unit_cost"`
	TotalValue      *float64   `json:"total_value"`
	Notes           *string    `json:"notes"`
	PerformedBy     *uuid.UUID `json:"performed_by"`
	PerformedByName *string    `json:"performed_by_name"`
	MovementDate    time.Time  `json:"movement_date"`
}

// ====================== VALUATION ======================

type StockValuation struct {
	ItemID            uuid.UUID `json:"item_id"`
	ItemCode          string    `json:"item_code"`
	ItemName          string    `json:"item_name"`
	ItemType          string    `json:"item_type"`
	UnitOfMeasure     string    `json:"unit_of_measure"`
	WarehouseID       uuid.UUID `json:"warehouse_id"`
	WarehouseName     string    `json:"warehouse_name"`
	WarehouseType     string    `json:"warehouse_type"`
	Quantity          float64   `json:"quantity"`
	ReservedQuantity  float64   `json:"reserved_quantity"`
	AvailableQuantity float64   `json:"available_quantity"`
	ReorderLevel      *float64  `json:"reorder_level"`
	LowStock          bool      `json:"low_stock"`
	StockStatus       string    `json:"stock_status"`
	AverageUnitCost   float64   `json:"average_unit_cost"`
	InventoryValue    float64   `json:"inventory_value"`
	CostSource        string    `json:"cost_source"` // stock_batches, inbound_movements, no_cost_recorded
}

type ValuationGroup struct {
	WarehouseType string  `json:"warehouse_type"`
	Lines         int64   `json:"lines"`
	Value         float64 `json:"value"`
}

type ValuationSummary struct {
	TotalValue       float64          `json:"total_value"`
	TotalLines       int64            `json:"total_lines"`
	LinesWithoutCost int64            `json:"lines_without_cost"`
	ByWarehouseType  []ValuationGroup `json:"by_warehouse_type"`
}

// ====================== ALERTS ======================

type InventoryAlert struct {
	ID            uuid.UUID `json:"id"`
	AlertType     string    `json:"alert_type"` // low_stock, near_expiry, expired
	Severity      string    `json:"severity"`   // critical, warning, info
	ItemID        uuid.UUID `json:"item_id"`
	ItemCode      string    `json:"item_code"`
	ItemName      string    `json:"item_name"`
	UnitOfMeasure string    `json:"unit_of_measure"`
	WarehouseID   uuid.UUID `json:"warehouse_id"`
	WarehouseCode string    `json:"warehouse_code"`
	WarehouseName string    `json:"warehouse_name"`

	// Always present in the JSON (null when not applicable) so frontend types stay stable.
	BatchNumber     *string   `json:"batch_number"`
	Quantity        *float64  `json:"quantity"`
	Threshold       *float64  `json:"threshold"`
	ExpiryDate      *string   `json:"expiry_date"` // YYYY-MM-DD, same format as StockBatch
	DaysUntilExpiry *int      `json:"days_until_expiry"`
	Message         string    `json:"message"`
	CreatedAt       time.Time `json:"created_at"`
}

type AlertSummary struct {
	Total      int64 `json:"total"`
	LowStock   int64 `json:"low_stock"`
	NearExpiry int64 `json:"near_expiry"`
	Expired    int64 `json:"expired"`
	Critical   int64 `json:"critical"`
	Warning    int64 `json:"warning"`
	Info       int64 `json:"info"`
}
