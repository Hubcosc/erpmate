package procurement

import "github.com/google/uuid"

// UpdatePurchaseOrderRequest is a partial update of a DRAFT purchase order.
//
//   - A nil field is left unchanged.
//   - For text/date fields, an empty string clears the stored value.
//   - Every entry in Items must include its id and unit_price.
type UpdatePurchaseOrderRequest struct {
	SupplierID           *uuid.UUID                `json:"supplier_id"`
	ExpectedDeliveryDate *string                   `json:"expected_delivery_date"` // YYYY-MM-DD
	PaymentTerms         *string                   `json:"payment_terms"`
	DeliveryTerms        *string                   `json:"delivery_terms"`
	Notes                *string                   `json:"notes"`
	Items                []UpdatePurchaseOrderItem `json:"items"`
}

type UpdatePurchaseOrderItem struct {
	ID        uuid.UUID `json:"id"`
	UnitPrice *float64  `json:"unit_price"` // required
}

type CreateGoodsReceiptNoteRequest struct {
	PurchaseOrderID          uuid.UUID                    `json:"purchase_order_id"`
	WarehouseID              uuid.UUID                    `json:"warehouse_id"`
	SupplierInvoiceReference *string                      `json:"supplier_invoice_reference"`
	Notes                    *string                      `json:"notes"`
	Items                    []CreateGoodsReceiptNoteItem `json:"items"`
}

type CreateGoodsReceiptNoteItem struct {
	PurchaseOrderItemID uuid.UUID `json:"purchase_order_item_id"`
	QuantityReceived    float64   `json:"quantity_received"`
	BatchNumber         *string   `json:"batch_number"`     // required for batch-tracked items
	ManufactureDate     *string   `json:"manufacture_date"` // YYYY-MM-DD
	ExpiryDate          *string   `json:"expiry_date"`      // YYYY-MM-DD
	UnitCost            *float64  `json:"unit_cost"`        // defaults to the PO line unit price
	Location            *string   `json:"location"`
}

// PurchaseOrderFilter are the optional list filters.
type PurchaseOrderFilter struct {
	Status     string
	SupplierID *uuid.UUID
	Search     string // matches PO number, PR number or supplier name
}
