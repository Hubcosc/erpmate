package procurement

import (
	"time"

	"github.com/google/uuid"
)

type PurchaseOrder struct {
	ID                    uuid.UUID           `json:"id"`
	PONumber              string              `json:"po_number"`
	PurchaseRequisitionID uuid.UUID           `json:"purchase_requisition_id"`
	PRNumber              string              `json:"pr_number,omitempty"`
	SupplierID            *uuid.UUID          `json:"supplier_id"`
	SupplierName          *string             `json:"supplier_name,omitempty"`
	Status                string              `json:"status"`
	OrderDate             time.Time           `json:"order_date"`
	ExpectedDeliveryDate  *time.Time          `json:"expected_delivery_date"`
	PaymentTerms          *string             `json:"payment_terms"`
	DeliveryTerms         *string             `json:"delivery_terms"`
	Notes                 *string             `json:"notes"`
	TotalAmount           float64             `json:"total_amount"`
	IssuedBy              *uuid.UUID          `json:"issued_by"`
	IssuedAt              *time.Time          `json:"issued_at"`
	CreatedAt             time.Time           `json:"created_at"`
	UpdatedAt             time.Time           `json:"updated_at"`
	Items                 []PurchaseOrderItem `json:"items,omitempty"`
}

type PurchaseOrderItem struct {
	ID                        uuid.UUID  `json:"id"`
	PurchaseOrderID           uuid.UUID  `json:"purchase_order_id"`
	PurchaseRequisitionItemID *uuid.UUID `json:"purchase_requisition_item_id"`
	ItemID                    *uuid.UUID `json:"item_id"`
	ItemCode                  *string    `json:"item_code,omitempty"`
	ItemName                  *string    `json:"item_name,omitempty"`
	Description               *string    `json:"description"`
	Quantity                  float64    `json:"quantity"`
	ReceivedQuantity          float64    `json:"received_quantity"`
	RemainingQuantity         float64    `json:"remaining_quantity"`
	IsBatchTracked            bool       `json:"is_batch_tracked"`
	UnitOfMeasure             *string    `json:"unit_of_measure"`
	UnitPrice                 *float64   `json:"unit_price"`
	LineTotal                 float64    `json:"line_total"`
	Notes                     *string    `json:"notes"`
}

type GoodsReceiptNote struct {
	ID                       uuid.UUID              `json:"id"`
	GRNNumber                string                 `json:"grn_number"`
	PurchaseOrderID          uuid.UUID              `json:"purchase_order_id"`
	PONumber                 string                 `json:"po_number,omitempty"`
	WarehouseID              uuid.UUID              `json:"warehouse_id"`
	WarehouseName            string                 `json:"warehouse_name,omitempty"`
	ReceivedBy               uuid.UUID              `json:"received_by"`
	ReceivedByName           string                 `json:"received_by_name,omitempty"`
	ReceiptDate              time.Time              `json:"receipt_date"`
	SupplierInvoiceReference *string                `json:"supplier_invoice_reference"`
	Notes                    *string                `json:"notes"`
	CreatedAt                time.Time              `json:"created_at"`
	Items                    []GoodsReceiptNoteItem `json:"items,omitempty"`
}

type GoodsReceiptNoteItem struct {
	ID                  uuid.UUID  `json:"id"`
	GoodsReceiptNoteID  uuid.UUID  `json:"goods_receipt_note_id"`
	PurchaseOrderItemID uuid.UUID  `json:"purchase_order_item_id"`
	ItemID              uuid.UUID  `json:"item_id"`
	ItemCode            string     `json:"item_code,omitempty"`
	ItemName            string     `json:"item_name,omitempty"`
	BatchID             *uuid.UUID `json:"batch_id"`
	QuantityReceived    float64    `json:"quantity_received"`
	BatchNumber         *string    `json:"batch_number"`
	ManufactureDate     *time.Time `json:"manufacture_date"`
	ExpiryDate          *time.Time `json:"expiry_date"`
	UnitCost            float64    `json:"unit_cost"`
	Location            *string    `json:"location"`
}
