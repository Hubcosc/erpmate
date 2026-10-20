package procurement

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Ethio-ERP/erpmate/internal/common/statusnotify"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type WorkflowRepository struct{ db *pgxpool.Pool }

func NewWorkflowRepository(db *pgxpool.Pool) *WorkflowRepository { return &WorkflowRepository{db: db} }

// CreateFromApprovedPR creates a draft PO from a finance-approved requisition.
// It is idempotent: if a PO already exists for the requisition it is returned
// with created=false.
func (r *WorkflowRepository) CreateFromApprovedPR(ctx context.Context, prID, actorID uuid.UUID) (*PurchaseOrder, bool, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, false, err
	}
	defer tx.Rollback(ctx)

	var status string
	var supplierID *uuid.UUID
	err = tx.QueryRow(ctx, `SELECT status,preferred_supplier_id FROM purchase_requisitions WHERE id=$1 AND deleted_at IS NULL FOR UPDATE`, prID).Scan(&status, &supplierID)
	if err != nil {
		return nil, false, noRows(err, "purchase requisition not found")
	}

	// The requisition row is locked above, so concurrent callers are serialised here.
	var existingID uuid.UUID
	var existingDeleted bool
	err = tx.QueryRow(ctx, `SELECT id,deleted_at IS NOT NULL FROM purchase_orders WHERE purchase_requisition_id=$1`, prID).Scan(&existingID, &existingDeleted)
	switch {
	case err == nil:
		if existingDeleted {
			return nil, false, conflict("a purchase order for this requisition was previously removed")
		}
		po, e := r.GetPurchaseOrder(ctx, existingID)
		return po, false, e
	case !errors.Is(err, pgx.ErrNoRows):
		return nil, false, err
	}

	if status != "approved" {
		return nil, false, conflict("purchase requisition must be finance-approved before a purchase order can be created")
	}

	var poNumber string
	if err = tx.QueryRow(ctx, `SELECT 'PO-'||to_char(CURRENT_DATE,'YYYY')||'-'||lpad(nextval('purchase_order_number_seq')::text,6,'0')`).Scan(&poNumber); err != nil {
		return nil, false, err
	}
	poID := uuid.New()
	if _, err = tx.Exec(ctx, `INSERT INTO purchase_orders (id,po_number,purchase_requisition_id,supplier_id,status,created_by,updated_by) VALUES ($1,$2,$3,$4,'draft',$5,$5)`, poID, poNumber, prID, supplierID, actorID); err != nil {
		return nil, false, err
	}
	tag, err := tx.Exec(ctx, `INSERT INTO purchase_order_items (purchase_order_id,purchase_requisition_item_id,item_id,description,quantity,unit_of_measure,unit_price,line_total,notes,created_by,updated_by) SELECT $1,pri.id,pri.item_id,pri.description,pri.quantity,pri.unit_of_measure,pri.estimated_unit_price,COALESCE(pri.estimated_total,0),pri.notes,$2,$2 FROM purchase_requisition_items pri WHERE pri.pr_id=$3`, poID, actorID, prID)
	if err != nil {
		return nil, false, err
	}
	if tag.RowsAffected() == 0 {
		return nil, false, invalid("purchase requisition has no items")
	}
	if _, err = tx.Exec(ctx, `UPDATE purchase_orders SET total_amount=COALESCE((SELECT SUM(line_total) FROM purchase_order_items WHERE purchase_order_id=$1),0) WHERE id=$1`, poID); err != nil {
		return nil, false, err
	}
	if err = statusnotify.Create(ctx, tx, "purchase_order", poID, actorID, actorID, "draft"); err != nil {
		return nil, false, err
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, false, err
	}
	po, err := r.GetPurchaseOrder(ctx, poID)
	return po, true, err
}

func (r *WorkflowRepository) GetPurchaseOrder(ctx context.Context, id uuid.UUID) (*PurchaseOrder, error) {
	po := &PurchaseOrder{}
	err := scanPO(r.db.QueryRow(ctx, `SELECT `+poColumns+poFrom+`WHERE po.id=$1 AND po.deleted_at IS NULL`, id), po)
	if err != nil {
		return nil, noRows(err, "purchase order not found")
	}
	rows, err := r.db.Query(ctx, `SELECT poi.id,poi.purchase_order_id,poi.purchase_requisition_item_id,poi.item_id,i.code,i.name,poi.description,poi.quantity,poi.received_quantity,(poi.quantity-poi.received_quantity),COALESCE(i.is_batch_tracked,FALSE),poi.unit_of_measure,poi.unit_price,poi.line_total,poi.notes FROM purchase_order_items poi LEFT JOIN items i ON i.id=poi.item_id WHERE poi.purchase_order_id=$1 ORDER BY poi.created_at,poi.id`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var it PurchaseOrderItem
		if err = rows.Scan(&it.ID, &it.PurchaseOrderID, &it.PurchaseRequisitionItemID, &it.ItemID, &it.ItemCode, &it.ItemName, &it.Description, &it.Quantity, &it.ReceivedQuantity, &it.RemainingQuantity, &it.IsBatchTracked, &it.UnitOfMeasure, &it.UnitPrice, &it.LineTotal, &it.Notes); err != nil {
			return nil, err
		}
		po.Items = append(po.Items, it)
	}
	return po, rows.Err()
}

func (r *WorkflowRepository) ListPurchaseOrders(ctx context.Context, f PurchaseOrderFilter, page, limit int) ([]PurchaseOrder, int64, error) {
	where := []string{"po.deleted_at IS NULL"}
	args := []any{}
	if f.Status != "" {
		args = append(args, f.Status)
		where = append(where, fmt.Sprintf("po.status=$%d", len(args)))
	}
	if f.SupplierID != nil {
		args = append(args, *f.SupplierID)
		where = append(where, fmt.Sprintf("po.supplier_id=$%d", len(args)))
	}
	if f.Search != "" {
		args = append(args, "%"+escapeLike(f.Search)+"%")
		n := len(args)
		where = append(where, fmt.Sprintf("(po.po_number ILIKE $%d OR pr.pr_number ILIKE $%d OR s.name ILIKE $%d)", n, n, n))
	}
	clause := "WHERE " + strings.Join(where, " AND ")

	var total int64
	if err := r.db.QueryRow(ctx, `SELECT COUNT(*)`+poFrom+clause, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	q := fmt.Sprintf(`SELECT %s%s%s ORDER BY po.created_at DESC,po.id LIMIT $%d OFFSET $%d`, poColumns, poFrom, clause, len(args)+1, len(args)+2)
	args = append(args, limit, (page-1)*limit)
	rows, err := r.db.Query(ctx, q, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	list := make([]PurchaseOrder, 0)
	for rows.Next() {
		var po PurchaseOrder
		if err = scanPO(rows, &po); err != nil {
			return nil, 0, err
		}
		list = append(list, po)
	}
	return list, total, rows.Err()
}

func (r *WorkflowRepository) UpdateDraftPurchaseOrder(ctx context.Context, id, actorID uuid.UUID, in poUpdate) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var status string
	var orderDate time.Time
	if err = tx.QueryRow(ctx, `SELECT status,order_date FROM purchase_orders WHERE id=$1 AND deleted_at IS NULL FOR UPDATE`, id).Scan(&status, &orderDate); err != nil {
		return noRows(err, "purchase order not found")
	}
	if status != "draft" {
		return conflict("only draft purchase orders can be edited")
	}
	if in.Expected != nil && in.Expected.Before(orderDate) {
		return invalid("expected_delivery_date cannot be before the order date")
	}
	if in.SupplierID != nil {
		var active bool
		if err = tx.QueryRow(ctx, `SELECT is_active FROM suppliers WHERE id=$1 AND deleted_at IS NULL`, *in.SupplierID).Scan(&active); err != nil {
			return noRows(err, "supplier not found")
		}
		if !active {
			return invalid("supplier is inactive")
		}
	}
	_, err = tx.Exec(ctx, `UPDATE purchase_orders SET
		supplier_id=COALESCE($1,supplier_id),
		expected_delivery_date=CASE WHEN $2::boolean THEN $3::date ELSE expected_delivery_date END,
		payment_terms=CASE WHEN $4::text IS NULL THEN payment_terms ELSE NULLIF($4::text,'') END,
		delivery_terms=CASE WHEN $5::text IS NULL THEN delivery_terms ELSE NULLIF($5::text,'') END,
		notes=CASE WHEN $6::text IS NULL THEN notes ELSE NULLIF($6::text,'') END,
		updated_at=NOW(),updated_by=$7 WHERE id=$8`,
		in.SupplierID, in.SetExpected, in.Expected, in.PaymentTerms, in.DeliveryTerms, in.Notes, actorID, id)
	if err != nil {
		return err
	}
	for _, it := range in.Items {
		tag, e := tx.Exec(ctx, `UPDATE purchase_order_items SET unit_price=$1,line_total=ROUND(quantity*$1,2),updated_at=NOW(),updated_by=$2 WHERE id=$3 AND purchase_order_id=$4`, it.Price, actorID, it.ID, id)
		if e != nil {
			return e
		}
		if tag.RowsAffected() != 1 {
			return notFound("purchase order item " + it.ID.String() + " not found on this purchase order")
		}
	}
	if _, err = tx.Exec(ctx, `UPDATE purchase_orders SET total_amount=COALESCE((SELECT SUM(line_total) FROM purchase_order_items WHERE purchase_order_id=$1),0),updated_at=NOW(),updated_by=$2 WHERE id=$1`, id, actorID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *WorkflowRepository) IssuePurchaseOrder(ctx context.Context, id, actorID uuid.UUID) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var status string
	var supplierID *uuid.UUID
	if err = tx.QueryRow(ctx, `SELECT status,supplier_id FROM purchase_orders WHERE id=$1 AND deleted_at IS NULL FOR UPDATE`, id).Scan(&status, &supplierID); err != nil {
		return noRows(err, "purchase order not found")
	}
	if status != "draft" {
		return conflict("only draft purchase orders can be issued")
	}
	if supplierID == nil {
		return invalid("select a supplier before issuing the purchase order")
	}
	var active bool
	if err = tx.QueryRow(ctx, `SELECT is_active FROM suppliers WHERE id=$1 AND deleted_at IS NULL`, *supplierID).Scan(&active); err != nil {
		return noRows(err, "supplier not found")
	}
	if !active {
		return invalid("supplier is inactive")
	}
	var total, noPrice, noItem int
	if err = tx.QueryRow(ctx, `SELECT COUNT(*),COUNT(*) FILTER (WHERE unit_price IS NULL),COUNT(*) FILTER (WHERE item_id IS NULL) FROM purchase_order_items WHERE purchase_order_id=$1`, id).Scan(&total, &noPrice, &noItem); err != nil {
		return err
	}
	switch {
	case total == 0:
		return invalid("purchase order has no items")
	case noPrice > 0:
		return invalid("finalize a unit price for every purchase order item before issuing")
	case noItem > 0:
		return invalid("every purchase order line must reference a catalog item before issuing")
	}
	if _, err = tx.Exec(ctx, `UPDATE purchase_orders SET status='issued',issued_by=$1,issued_at=NOW(),updated_by=$1,updated_at=NOW() WHERE id=$2`, actorID, id); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `INSERT INTO status_history(entity_type,entity_id,old_status,new_status,changed_by) VALUES('purchase_order',$1,'draft','issued',$2)`, id, actorID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *WorkflowRepository) CreateGRN(ctx context.Context, actorID uuid.UUID, in grnInput) (*GoodsReceiptNote, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	var warehouseType string
	if err = tx.QueryRow(ctx, `SELECT warehouse_type FROM warehouses WHERE id=$1 AND is_active=TRUE`, in.WarehouseID).Scan(&warehouseType); err != nil {
		return nil, noRows(err, "active receiving warehouse not found")
	}
	if warehouseType != "raw_material" {
		return nil, invalid("purchase goods must be received into a raw-material warehouse")
	}
	var poStatus string
	if err = tx.QueryRow(ctx, `SELECT status FROM purchase_orders WHERE id=$1 AND deleted_at IS NULL FOR UPDATE`, in.PurchaseOrderID).Scan(&poStatus); err != nil {
		return nil, noRows(err, "purchase order not found")
	}
	if poStatus != "issued" && poStatus != "partially_received" {
		return nil, conflict("goods can only be received against an issued purchase order")
	}

	var number string
	if err = tx.QueryRow(ctx, `SELECT 'GRN-'||to_char(CURRENT_DATE,'YYYY')||'-'||lpad(nextval('goods_receipt_number_seq')::text,6,'0')`).Scan(&number); err != nil {
		return nil, err
	}
	grnID := uuid.New()
	if _, err = tx.Exec(ctx, `INSERT INTO goods_receipt_notes(id,grn_number,purchase_order_id,warehouse_id,received_by,supplier_invoice_reference,notes,created_by,updated_by) VALUES($1,$2,$3,$4,$5,$6,$7,$5,$5)`, grnID, number, in.PurchaseOrderID, in.WarehouseID, actorID, in.InvoiceRef, in.Notes); err != nil {
		return nil, err
	}

	for _, line := range in.Lines {
		var itemID uuid.UUID
		var itemCode string
		var ordered, received float64
		var linePrice *float64
		var tracked bool
		err = tx.QueryRow(ctx, `SELECT poi.item_id,i.code,poi.quantity,poi.received_quantity,poi.unit_price,i.is_batch_tracked FROM purchase_order_items poi JOIN items i ON i.id=poi.item_id WHERE poi.id=$1 AND poi.purchase_order_id=$2 FOR UPDATE OF poi`, line.PurchaseOrderItemID, in.PurchaseOrderID).Scan(&itemID, &itemCode, &ordered, &received, &linePrice, &tracked)
		if err != nil {
			return nil, noRows(err, "purchase order item "+line.PurchaseOrderItemID.String()+" not found on this purchase order")
		}
		// NUMERIC(18,4): compare at 4 decimals to avoid float drift.
		if round4(received+line.Quantity) > round4(ordered) {
			return nil, conflict(fmt.Sprintf("receipt quantity for item %s exceeds the remaining PO quantity (%.4f)", itemCode, round4(ordered-received)))
		}
		if tracked && line.BatchNumber == "" {
			return nil, invalid(fmt.Sprintf("batch_number is required for batch-tracked item %s", itemCode))
		}
		var unitCost float64
		switch {
		case line.UnitCost != nil:
			unitCost = *line.UnitCost
		case linePrice != nil:
			unitCost = *linePrice
		default:
			return nil, invalid(fmt.Sprintf("unit_cost is required for item %s", itemCode))
		}

		var batchID *uuid.UUID
		if line.BatchNumber != "" {
			var bid uuid.UUID
			// Re-receiving into an existing batch adds quantity and re-averages the cost.
			err = tx.QueryRow(ctx, `INSERT INTO stock_batches(item_id,warehouse_id,batch_number,quantity,manufacture_date,expiry_date,unit_cost) VALUES($1,$2,$3,$4,$5,$6,$7)
				ON CONFLICT(item_id,warehouse_id,batch_number) DO UPDATE SET
				quantity=stock_batches.quantity+EXCLUDED.quantity,
				manufacture_date=COALESCE(stock_batches.manufacture_date,EXCLUDED.manufacture_date),
				expiry_date=COALESCE(stock_batches.expiry_date,EXCLUDED.expiry_date),
				unit_cost=ROUND(((stock_batches.quantity*COALESCE(stock_batches.unit_cost,0))+(EXCLUDED.quantity*EXCLUDED.unit_cost))/NULLIF(stock_batches.quantity+EXCLUDED.quantity,0),2)
				RETURNING id`, itemID, in.WarehouseID, line.BatchNumber, line.Quantity, line.Manufacture, line.Expiry, unitCost).Scan(&bid)
			if err != nil {
				return nil, err
			}
			batchID = &bid
		}
		if _, err = tx.Exec(ctx, `INSERT INTO stock_balances(item_id,warehouse_id,quantity,reserved_qty) VALUES($1,$2,$3,0) ON CONFLICT(item_id,warehouse_id) DO UPDATE SET quantity=stock_balances.quantity+EXCLUDED.quantity,updated_at=NOW()`, itemID, in.WarehouseID, line.Quantity); err != nil {
			return nil, err
		}
		if _, err = tx.Exec(ctx, `INSERT INTO goods_receipt_note_items(id,goods_receipt_note_id,purchase_order_item_id,item_id,batch_id,quantity_received,batch_number,manufacture_date,expiry_date,unit_cost,location,created_by,updated_by) VALUES($1,$2,$3,$4,$5,$6,NULLIF($7,''),$8,$9,$10,$11,$12,$12)`, uuid.New(), grnID, line.PurchaseOrderItemID, itemID, batchID, line.Quantity, line.BatchNumber, line.Manufacture, line.Expiry, unitCost, line.Location, actorID); err != nil {
			return nil, err
		}
		if _, err = tx.Exec(ctx, `UPDATE purchase_order_items SET received_quantity=received_quantity+$1,updated_at=NOW(),updated_by=$2 WHERE id=$3`, line.Quantity, actorID, line.PurchaseOrderItemID); err != nil {
			return nil, err
		}
		if _, err = tx.Exec(ctx, `INSERT INTO stock_movements(movement_type,reference_type,reference_id,item_id,warehouse_id,batch_id,quantity,unit_cost,notes,performed_by) VALUES('receipt','goods_receipt_note',$1,$2,$3,$4,$5,$6,$7,$8)`, grnID, itemID, in.WarehouseID, batchID, line.Quantity, unitCost, in.InvoiceRef, actorID); err != nil {
			return nil, err
		}
	}

	var stillOpen bool
	if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM purchase_order_items WHERE purchase_order_id=$1 AND received_quantity<quantity)`, in.PurchaseOrderID).Scan(&stillOpen); err != nil {
		return nil, err
	}
	newStatus := "received"
	if stillOpen {
		newStatus = "partially_received"
	}
	if _, err = tx.Exec(ctx, `UPDATE purchase_orders SET status=$1,updated_at=NOW(),updated_by=$2 WHERE id=$3`, newStatus, actorID, in.PurchaseOrderID); err != nil {
		return nil, err
	}
	if newStatus != poStatus {
		if _, err = tx.Exec(ctx, `INSERT INTO status_history(entity_type,entity_id,old_status,new_status,changed_by) VALUES('purchase_order',$1,$2,$3,$4)`, in.PurchaseOrderID, poStatus, newStatus, actorID); err != nil {
			return nil, err
		}
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, err
	}
	return r.GetGRN(ctx, grnID)
}

func (r *WorkflowRepository) GetGRN(ctx context.Context, id uuid.UUID) (*GoodsReceiptNote, error) {
	g := &GoodsReceiptNote{}
	if err := scanGRN(r.db.QueryRow(ctx, `SELECT `+grnColumns+grnFrom+`WHERE g.id=$1 AND g.deleted_at IS NULL`, id), g); err != nil {
		return nil, noRows(err, "goods receipt note not found")
	}
	rows, err := r.db.Query(ctx, `SELECT gi.id,gi.goods_receipt_note_id,gi.purchase_order_item_id,gi.item_id,i.code,i.name,gi.batch_id,gi.quantity_received,gi.batch_number,gi.manufacture_date,gi.expiry_date,gi.unit_cost,gi.location FROM goods_receipt_note_items gi JOIN items i ON i.id=gi.item_id WHERE gi.goods_receipt_note_id=$1 ORDER BY gi.created_at,gi.id`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var it GoodsReceiptNoteItem
		if err = rows.Scan(&it.ID, &it.GoodsReceiptNoteID, &it.PurchaseOrderItemID, &it.ItemID, &it.ItemCode, &it.ItemName, &it.BatchID, &it.QuantityReceived, &it.BatchNumber, &it.ManufactureDate, &it.ExpiryDate, &it.UnitCost, &it.Location); err != nil {
			return nil, err
		}
		g.Items = append(g.Items, it)
	}
	return g, rows.Err()
}

func (r *WorkflowRepository) ListGRNs(ctx context.Context, poID *uuid.UUID, page, limit int) ([]GoodsReceiptNote, int64, error) {
	clause := "WHERE g.deleted_at IS NULL"
	args := []any{}
	if poID != nil {
		clause += " AND g.purchase_order_id=$1"
		args = append(args, *poID)
	}
	var total int64
	if err := r.db.QueryRow(ctx, `SELECT COUNT(*)`+grnFrom+clause, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	q := fmt.Sprintf(`SELECT %s%s%s ORDER BY g.receipt_date DESC,g.id LIMIT $%d OFFSET $%d`, grnColumns, grnFrom, clause, len(args)+1, len(args)+2)
	args = append(args, limit, (page-1)*limit)
	rows, err := r.db.Query(ctx, q, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	list := make([]GoodsReceiptNote, 0)
	for rows.Next() {
		var g GoodsReceiptNote
		if err = scanGRN(rows, &g); err != nil {
			return nil, 0, err
		}
		list = append(list, g)
	}
	return list, total, rows.Err()
}
