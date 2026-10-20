package procurement

import (
	"errors"
	"math"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// rowScanner is satisfied by both pgx.Row and pgx.Rows.
type rowScanner interface{ Scan(dest ...any) error }

const poColumns = `po.id,po.po_number,po.purchase_requisition_id,pr.pr_number,po.supplier_id,s.name,po.status,po.order_date,po.expected_delivery_date,po.payment_terms,po.delivery_terms,po.notes,po.total_amount,po.issued_by,po.issued_at,po.created_at,po.updated_at`
const poFrom = ` FROM purchase_orders po JOIN purchase_requisitions pr ON pr.id=po.purchase_requisition_id LEFT JOIN suppliers s ON s.id=po.supplier_id `

func scanPO(row rowScanner, po *PurchaseOrder) error {
	return row.Scan(&po.ID, &po.PONumber, &po.PurchaseRequisitionID, &po.PRNumber, &po.SupplierID, &po.SupplierName, &po.Status, &po.OrderDate, &po.ExpectedDeliveryDate, &po.PaymentTerms, &po.DeliveryTerms, &po.Notes, &po.TotalAmount, &po.IssuedBy, &po.IssuedAt, &po.CreatedAt, &po.UpdatedAt)
}

const grnColumns = `g.id,g.grn_number,g.purchase_order_id,po.po_number,g.warehouse_id,w.name,g.received_by,u.full_name,g.receipt_date,g.supplier_invoice_reference,g.notes,g.created_at`
const grnFrom = ` FROM goods_receipt_notes g JOIN purchase_orders po ON po.id=g.purchase_order_id JOIN warehouses w ON w.id=g.warehouse_id JOIN users u ON u.id=g.received_by `

func scanGRN(row rowScanner, g *GoodsReceiptNote) error {
	return row.Scan(&g.ID, &g.GRNNumber, &g.PurchaseOrderID, &g.PONumber, &g.WarehouseID, &g.WarehouseName, &g.ReceivedBy, &g.ReceivedByName, &g.ReceiptDate, &g.SupplierInvoiceReference, &g.Notes, &g.CreatedAt)
}

// poUpdate is the validated, normalised form of UpdatePurchaseOrderRequest.
type poUpdate struct {
	SupplierID    *uuid.UUID
	SetExpected   bool       // true when expected_delivery_date was supplied
	Expected      *time.Time // nil with SetExpected=true clears the date
	PaymentTerms  *string    // nil = unchanged, "" = clear
	DeliveryTerms *string
	Notes         *string
	Items         []poPrice
}

type poPrice struct {
	ID    uuid.UUID
	Price float64
}

// grnInput is the validated, normalised form of CreateGoodsReceiptNoteRequest.
type grnInput struct {
	PurchaseOrderID uuid.UUID
	WarehouseID     uuid.UUID
	InvoiceRef      *string
	Notes           *string
	Lines           []grnLine
}

type grnLine struct {
	PurchaseOrderItemID uuid.UUID
	Quantity            float64
	BatchNumber         string
	Manufacture         *time.Time
	Expiry              *time.Time
	UnitCost            *float64 // nil = use PO line price
	Location            *string
}

func round2(v float64) float64 { return math.Round(v*100) / 100 }
func round4(v float64) float64 { return math.Round(v*10000) / 10000 }

// noRows converts pgx.ErrNoRows into a 404 AppError and passes other errors through.
func noRows(err error, msg string) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return notFound(msg)
	}
	return err
}

// escapeLike escapes LIKE wildcards in user-supplied search text.
func escapeLike(s string) string {
	r := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)
	return r.Replace(s)
}
