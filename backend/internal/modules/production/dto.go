package production

import "github.com/google/uuid"

type CreateReceiptRequest struct {
	WarehouseID uuid.UUID           `json:"warehouse_id"`
	Notes       *string             `json:"notes"`
	Items       []CreateReceiptItem `json:"items"`
}

type CreateReceiptItem struct {
	ItemID           uuid.UUID `json:"item_id"`
	QuantityProduced float64   `json:"quantity_produced"`
	BatchNumber      *string   `json:"batch_number"`
	ManufactureDate  *string   `json:"manufacture_date"`
	ExpiryDate       *string   `json:"expiry_date"`
	UnitCost         *float64  `json:"unit_cost"`
	QualityStatus    string    `json:"quality_status"`
}
