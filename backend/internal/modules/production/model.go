package production

import (
	"github.com/google/uuid"
	"time"
)

type Receipt struct {
	ID             uuid.UUID     `json:"id"`
	ReceiptNumber  string        `json:"receipt_number"`
	WarehouseID    uuid.UUID     `json:"warehouse_id"`
	WarehouseName  string        `json:"warehouse_name,omitempty"`
	RecordedBy     uuid.UUID     `json:"recorded_by"`
	RecordedByName string        `json:"recorded_by_name,omitempty"`
	ReceiptDate    time.Time     `json:"receipt_date"`
	Notes          *string       `json:"notes"`
	CreatedAt      time.Time     `json:"created_at"`
	Items          []ReceiptItem `json:"items,omitempty"`
}

type ReceiptItem struct {
	ID                  uuid.UUID  `json:"id"`
	ProductionReceiptID uuid.UUID  `json:"production_receipt_id"`
	ItemID              uuid.UUID  `json:"item_id"`
	ItemCode            string     `json:"item_code,omitempty"`
	ItemName            string     `json:"item_name,omitempty"`
	BatchID             *uuid.UUID `json:"batch_id"`
	QuantityProduced    float64    `json:"quantity_produced"`
	BatchNumber         *string    `json:"batch_number"`
	ManufactureDate     *time.Time `json:"manufacture_date"`
	ExpiryDate          *time.Time `json:"expiry_date"`
	UnitCost            *float64   `json:"unit_cost"`
	QualityStatus       string     `json:"quality_status"`
}
