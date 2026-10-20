package sales

import (
	"github.com/google/uuid"
	"time"
)

type SalesOrder struct {
	ID           uuid.UUID        `json:"id"`
	OrderNumber  string           `json:"order_number"`
	CustomerID   uuid.UUID        `json:"customer_id"`
	CustomerName string           `json:"customer_name,omitempty"`
	Status       string           `json:"status"`
	OrderDate    time.Time        `json:"order_date"`
	RequiredDate *time.Time       `json:"required_date"`
	Notes        *string          `json:"notes"`
	TotalAmount  float64          `json:"total_amount"`
	CreatedAt    time.Time        `json:"created_at"`
	Items        []SalesOrderItem `json:"items,omitempty"`
}

type SalesOrderItem struct {
	ID                uuid.UUID `json:"id"`
	SalesOrderID      uuid.UUID `json:"sales_order_id"`
	ItemID            uuid.UUID `json:"item_id"`
	ItemCode          string    `json:"item_code,omitempty"`
	ItemName          string    `json:"item_name,omitempty"`
	Quantity          float64   `json:"quantity"`
	ReservedQuantity  float64   `json:"reserved_quantity"`
	DeliveredQuantity float64   `json:"delivered_quantity"`
	AvailableQuantity float64   `json:"available_quantity,omitempty"`
	UnitPrice         float64   `json:"unit_price"`
	LineTotal         float64   `json:"line_total"`
}

type DeliveryNote struct {
	ID                          uuid.UUID          `json:"id"`
	DeliveryNumber              string             `json:"delivery_number"`
	SalesOrderID                uuid.UUID          `json:"sales_order_id"`
	OrderNumber                 string             `json:"order_number,omitempty"`
	DeliveredBy                 uuid.UUID          `json:"delivered_by"`
	DeliveredByName             string             `json:"delivered_by_name,omitempty"`
	DeliveryDate                time.Time          `json:"delivery_date"`
	RecipientName               *string            `json:"recipient_name"`
	RecipientSignatureReference *string            `json:"recipient_signature_reference"`
	Notes                       *string            `json:"notes"`
	CreatedAt                   time.Time          `json:"created_at"`
	Items                       []DeliveryNoteItem `json:"items,omitempty"`
}

type DeliveryNoteItem struct {
	ID                uuid.UUID  `json:"id"`
	DeliveryNoteID    uuid.UUID  `json:"delivery_note_id"`
	SalesOrderItemID  uuid.UUID  `json:"sales_order_item_id"`
	ItemID            uuid.UUID  `json:"item_id"`
	ItemCode          string     `json:"item_code,omitempty"`
	ItemName          string     `json:"item_name,omitempty"`
	BatchID           *uuid.UUID `json:"batch_id"`
	BatchNumber       *string    `json:"batch_number,omitempty"`
	QuantityDelivered float64    `json:"quantity_delivered"`
}
