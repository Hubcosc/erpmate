package sales

import "github.com/google/uuid"

type CreateSalesOrderRequest struct {
	CustomerID   uuid.UUID              `json:"customer_id"`
	RequiredDate *string                `json:"required_date"`
	Notes        *string                `json:"notes"`
	Items        []CreateSalesOrderItem `json:"items"`
}

type CreateSalesOrderItem struct {
	ItemID    uuid.UUID `json:"item_id"`
	Quantity  float64   `json:"quantity"`
	UnitPrice float64   `json:"unit_price"`
}

type CreateDeliveryNoteRequest struct {
	RecipientName               *string                  `json:"recipient_name"`
	RecipientSignatureReference *string                  `json:"recipient_signature_reference"`
	Notes                       *string                  `json:"notes"`
	Items                       []CreateDeliveryNoteItem `json:"items"`
}

type CreateDeliveryNoteItem struct {
	SalesOrderItemID  uuid.UUID `json:"sales_order_item_id"`
	QuantityDelivered float64   `json:"quantity_delivered"`
}
