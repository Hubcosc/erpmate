package sales

import (
	"context"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type WorkflowRepository struct{ db *pgxpool.Pool }

func NewWorkflowRepository(db *pgxpool.Pool) *WorkflowRepository { return &WorkflowRepository{db: db} }

type stockAllocation struct {
	warehouseID uuid.UUID
	batchID     *uuid.UUID
	quantity    float64
}

func (r *WorkflowRepository) CreateOrder(ctx context.Context, actor uuid.UUID, req CreateSalesOrderRequest) (*SalesOrder, error) {
	if len(req.Items) == 0 {
		return nil, errors.New("at least one order line is required")
	}
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	var customerActive bool
	if err = tx.QueryRow(ctx, `SELECT is_active FROM customers WHERE id=$1 AND deleted_at IS NULL`, req.CustomerID).Scan(&customerActive); err != nil {
		return nil, errors.New("active customer not found")
	}
	if !customerActive {
		return nil, errors.New("customer is inactive")
	}
	var required *time.Time
	if req.RequiredDate != nil && *req.RequiredDate != "" {
		d, e := time.Parse("2006-01-02", *req.RequiredDate)
		if e != nil {
			return nil, errors.New("required_date must use YYYY-MM-DD")
		}
		required = &d
	}
	var number string
	if err = tx.QueryRow(ctx, `SELECT 'SO-'||to_char(CURRENT_DATE,'YYYY')||'-'||lpad(nextval('sales_order_number_seq')::text,6,'0')`).Scan(&number); err != nil {
		return nil, err
	}
	orderID := uuid.New()
	if err = tx.QueryRow(ctx, `INSERT INTO sales_orders(id,order_number,customer_id,status,required_date,notes,created_by,updated_by) VALUES($1,$2,$3,'confirmed',$4,$5,$6,$6) RETURNING order_date,created_at`, orderID, number, req.CustomerID, required, req.Notes, actor).Scan(new(time.Time), new(time.Time)); err != nil {
		return nil, err
	}
	var total float64
	for _, line := range req.Items {
		if line.ItemID == uuid.Nil || line.Quantity <= 0 || line.UnitPrice < 0 {
			return nil, errors.New("each line requires an item, positive quantity, and non-negative unit price")
		}
		var itemType string
		var batchTracked bool
		if err = tx.QueryRow(ctx, `SELECT item_type,is_batch_tracked FROM items WHERE id=$1 AND is_active=TRUE AND deleted_at IS NULL`, line.ItemID).Scan(&itemType, &batchTracked); err != nil {
			return nil, errors.New("active sales item not found")
		}
		if itemType != "finished_good" {
			return nil, errors.New("sales orders can contain finished goods only")
		}
		lineTotal := math.Round(line.Quantity*line.UnitPrice*100) / 100
		total += lineTotal
		orderItemID := uuid.New()
		if _, err = tx.Exec(ctx, `INSERT INTO sales_order_items(id,sales_order_id,item_id,quantity,unit_price,line_total,created_by,updated_by) VALUES($1,$2,$3,$4,$5,$6,$7,$7)`, orderItemID, orderID, line.ItemID, line.Quantity, line.UnitPrice, lineTotal, actor); err != nil {
			return nil, err
		}
		allocations, e := reserveFinishedGoods(ctx, tx, orderItemID, line.ItemID, line.Quantity, batchTracked, actor)
		if e != nil {
			return nil, e
		}
		if allocations <= 0 {
			return nil, fmt.Errorf("no allocatable stock for item %s", line.ItemID)
		}
	}
	if _, err = tx.Exec(ctx, `UPDATE sales_orders SET total_amount=$1,updated_at=NOW(),updated_by=$2 WHERE id=$3`, math.Round(total*100)/100, actor, orderID); err != nil {
		return nil, err
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, err
	}
	return r.GetOrder(ctx, orderID)
}

func reserveFinishedGoods(ctx context.Context, tx pgx.Tx, orderItemID, itemID uuid.UUID, requested float64, batchTracked bool, actor uuid.UUID) (float64, error) {
	allocs := make([]stockAllocation, 0)
	if batchTracked {
		type row struct {
			batchID, warehouseID uuid.UUID
			batchQty, available  float64
			expiry               *time.Time
		}
		rows, err := tx.Query(ctx, `SELECT sb.id,sb.warehouse_id,sb.quantity,bal.available_qty,sb.expiry_date FROM stock_batches sb JOIN stock_balances bal ON bal.item_id=sb.item_id AND bal.warehouse_id=sb.warehouse_id JOIN warehouses w ON w.id=sb.warehouse_id WHERE sb.item_id=$1 AND w.warehouse_type='finished_goods' AND w.is_active=TRUE AND sb.quantity>0 AND bal.available_qty>0 ORDER BY sb.expiry_date NULLS LAST,sb.created_at FOR UPDATE OF sb,bal`, itemID)
		if err != nil {
			return 0, err
		}
		var rowsData []row
		for rows.Next() {
			var v row
			if err = rows.Scan(&v.batchID, &v.warehouseID, &v.batchQty, &v.available, &v.expiry); err != nil {
				rows.Close()
				return 0, err
			}
			rowsData = append(rowsData, v)
		}
		rows.Close()
		if err = rows.Err(); err != nil {
			return 0, err
		}
		warehouseReserved := map[uuid.UUID]float64{}
		remaining := requested
		for _, v := range rowsData {
			if remaining <= 0 {
				break
			}
			capacity := v.available - warehouseReserved[v.warehouseID]
			if capacity <= 0 {
				continue
			}
			take := math.Min(remaining, math.Min(v.batchQty, capacity))
			batchID := v.batchID
			allocs = append(allocs, stockAllocation{v.warehouseID, &batchID, take})
			warehouseReserved[v.warehouseID] += take
			remaining -= take
		}
	} else {
		type row struct {
			id, warehouseID uuid.UUID
			available       float64
		}
		rows, err := tx.Query(ctx, `SELECT bal.id,bal.warehouse_id,bal.available_qty FROM stock_balances bal JOIN warehouses w ON w.id=bal.warehouse_id WHERE bal.item_id=$1 AND w.warehouse_type='finished_goods' AND w.is_active=TRUE AND bal.available_qty>0 ORDER BY w.code FOR UPDATE OF bal`, itemID)
		if err != nil {
			return 0, err
		}
		var data []row
		for rows.Next() {
			var v row
			if err = rows.Scan(&v.id, &v.warehouseID, &v.available); err != nil {
				rows.Close()
				return 0, err
			}
			data = append(data, v)
		}
		rows.Close()
		if err = rows.Err(); err != nil {
			return 0, err
		}
		remaining := requested
		for _, v := range data {
			if remaining <= 0 {
				break
			}
			take := math.Min(remaining, v.available)
			allocs = append(allocs, stockAllocation{v.warehouseID, nil, take})
			remaining -= take
		}
	}
	var reserved float64
	for _, a := range allocs {
		tag, err := tx.Exec(ctx, `UPDATE stock_balances SET reserved_qty=reserved_qty+$1,updated_at=NOW() WHERE item_id=$2 AND warehouse_id=$3 AND available_qty >= $1`, a.quantity, itemID, a.warehouseID)
		if err != nil {
			return 0, err
		}
		if tag.RowsAffected() != 1 {
			return 0, errors.New("finished-goods stock changed while reserving")
		}
		if _, err = tx.Exec(ctx, `INSERT INTO sales_order_item_allocations(sales_order_item_id,warehouse_id,batch_id,reserved_quantity,created_by) VALUES($1,$2,$3,$4,$5)`, orderItemID, a.warehouseID, a.batchID, a.quantity, actor); err != nil {
			return 0, err
		}
		reserved += a.quantity
	}
	if reserved+1e-8 < requested {
		return 0, fmt.Errorf("insufficient finished-goods availability: requested %.4f, available %.4f", requested, reserved)
	}
	if _, err := tx.Exec(ctx, `UPDATE sales_order_items SET reserved_quantity=$1 WHERE id=$2`, reserved, orderItemID); err != nil {
		return 0, err
	}
	return reserved, nil
}

func (r *WorkflowRepository) GetOrder(ctx context.Context, id uuid.UUID) (*SalesOrder, error) {
	o := &SalesOrder{}
	err := r.db.QueryRow(ctx, `SELECT so.id,so.order_number,so.customer_id,c.name,so.status,so.order_date,so.required_date,so.notes,so.total_amount,so.created_at FROM sales_orders so JOIN customers c ON c.id=so.customer_id WHERE so.id=$1 AND so.deleted_at IS NULL`, id).Scan(&o.ID, &o.OrderNumber, &o.CustomerID, &o.CustomerName, &o.Status, &o.OrderDate, &o.RequiredDate, &o.Notes, &o.TotalAmount, &o.CreatedAt)
	if err != nil {
		return nil, err
	}
	rows, err := r.db.Query(ctx, `SELECT soi.id,soi.sales_order_id,soi.item_id,i.code,i.name,soi.quantity,soi.reserved_quantity,soi.delivered_quantity,soi.unit_price,soi.line_total,COALESCE((SELECT SUM(bal.available_qty) FROM stock_balances bal JOIN warehouses w ON w.id=bal.warehouse_id WHERE bal.item_id=soi.item_id AND w.warehouse_type='finished_goods' AND w.is_active=TRUE),0) FROM sales_order_items soi JOIN items i ON i.id=soi.item_id WHERE soi.sales_order_id=$1 ORDER BY soi.created_at`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var item SalesOrderItem
		if err = rows.Scan(&item.ID, &item.SalesOrderID, &item.ItemID, &item.ItemCode, &item.ItemName, &item.Quantity, &item.ReservedQuantity, &item.DeliveredQuantity, &item.UnitPrice, &item.LineTotal, &item.AvailableQuantity); err != nil {
			return nil, err
		}
		o.Items = append(o.Items, item)
	}
	return o, rows.Err()
}

func (r *WorkflowRepository) ListOrders(ctx context.Context, status string, page, limit int) ([]SalesOrder, int64, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}
	where := "WHERE so.deleted_at IS NULL"
	args := []any{}
	if status != "" {
		where += " AND so.status=$1"
		args = append(args, status)
	}
	var total int64
	if err := r.db.QueryRow(ctx, "SELECT COUNT(*) FROM sales_orders so "+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	q := fmt.Sprintf(`SELECT so.id,so.order_number,so.customer_id,c.name,so.status,so.order_date,so.required_date,so.notes,so.total_amount,so.created_at FROM sales_orders so JOIN customers c ON c.id=so.customer_id %s ORDER BY so.created_at DESC LIMIT $%d OFFSET $%d`, where, len(args)+1, len(args)+2)
	args = append(args, limit, (page-1)*limit)
	rows, err := r.db.Query(ctx, q, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	list := make([]SalesOrder, 0)
	for rows.Next() {
		var o SalesOrder
		if err = rows.Scan(&o.ID, &o.OrderNumber, &o.CustomerID, &o.CustomerName, &o.Status, &o.OrderDate, &o.RequiredDate, &o.Notes, &o.TotalAmount, &o.CreatedAt); err != nil {
			return nil, 0, err
		}
		list = append(list, o)
	}
	return list, total, rows.Err()
}

func (r *WorkflowRepository) CreateDelivery(ctx context.Context, orderID, actor uuid.UUID, req CreateDeliveryNoteRequest) (*DeliveryNote, error) {
	if len(req.Items) == 0 {
		return nil, errors.New("at least one delivery line is required")
	}
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	var orderStatus string
	var orderCreator uuid.UUID
	if err = tx.QueryRow(ctx, `SELECT status,created_by FROM sales_orders WHERE id=$1 AND deleted_at IS NULL FOR UPDATE`, orderID).Scan(&orderStatus, &orderCreator); err != nil {
		return nil, err
	}
	if orderStatus != "confirmed" && orderStatus != "partially_delivered" {
		return nil, errors.New("sales order is not open for delivery")
	}
	var number string
	if err = tx.QueryRow(ctx, `SELECT 'DN-'||to_char(CURRENT_DATE,'YYYY')||'-'||lpad(nextval('delivery_note_number_seq')::text,6,'0')`).Scan(&number); err != nil {
		return nil, err
	}
	noteID := uuid.New()
	note := &DeliveryNote{ID: noteID, DeliveryNumber: number, SalesOrderID: orderID, DeliveredBy: actor, RecipientName: req.RecipientName, RecipientSignatureReference: req.RecipientSignatureReference, Notes: req.Notes}
	if err = tx.QueryRow(ctx, `INSERT INTO delivery_notes(id,delivery_number,sales_order_id,delivered_by,recipient_name,recipient_signature_reference,notes,created_by,updated_by) VALUES($1,$2,$3,$4,$5,$6,$7,$4,$4) RETURNING delivery_date,created_at`, noteID, number, orderID, actor, req.RecipientName, req.RecipientSignatureReference, req.Notes).Scan(&note.DeliveryDate, &note.CreatedAt); err != nil {
		return nil, err
	}
	for _, line := range req.Items {
		if line.QuantityDelivered <= 0 {
			return nil, errors.New("quantity_delivered must be greater than 0")
		}
		var itemID uuid.UUID
		var remaining float64
		if err = tx.QueryRow(ctx, `SELECT item_id,reserved_quantity-delivered_quantity FROM sales_order_items WHERE id=$1 AND sales_order_id=$2 FOR UPDATE`, line.SalesOrderItemID, orderID).Scan(&itemID, &remaining); err != nil {
			return nil, errors.New("sales order line not found")
		}
		if line.QuantityDelivered > remaining+1e-8 {
			return nil, errors.New("delivery quantity exceeds the undelivered reservation")
		}
		type allocation struct {
			id, warehouseID uuid.UUID
			batchID         *uuid.UUID
			remaining       float64
		}
		rows, err := tx.Query(ctx, `SELECT id,warehouse_id,batch_id,reserved_quantity-delivered_quantity FROM sales_order_item_allocations WHERE sales_order_item_id=$1 AND reserved_quantity>delivered_quantity ORDER BY created_at,id FOR UPDATE`, line.SalesOrderItemID)
		if err != nil {
			return nil, err
		}
		var allocs []allocation
		for rows.Next() {
			var a allocation
			if err = rows.Scan(&a.id, &a.warehouseID, &a.batchID, &a.remaining); err != nil {
				rows.Close()
				return nil, err
			}
			allocs = append(allocs, a)
		}
		rows.Close()
		if err = rows.Err(); err != nil {
			return nil, err
		}
		toDeliver := line.QuantityDelivered
		for _, a := range allocs {
			if toDeliver <= 1e-8 {
				break
			}
			take := math.Min(toDeliver, a.remaining)
			tag, e := tx.Exec(ctx, `UPDATE stock_balances SET quantity=quantity-$1,reserved_qty=reserved_qty-$1,updated_at=NOW() WHERE item_id=$2 AND warehouse_id=$3 AND quantity >= $1 AND reserved_qty >= $1`, take, itemID, a.warehouseID)
			if e != nil {
				return nil, e
			}
			if tag.RowsAffected() != 1 {
				return nil, errors.New("reserved finished-goods stock is no longer available")
			}
			if a.batchID != nil {
				tag, e = tx.Exec(ctx, `UPDATE stock_batches SET quantity=quantity-$1 WHERE id=$2 AND quantity >= $1`, take, *a.batchID)
				if e != nil {
					return nil, e
				}
				if tag.RowsAffected() != 1 {
					return nil, errors.New("reserved batch quantity is no longer available")
				}
			}
			if _, e = tx.Exec(ctx, `UPDATE sales_order_item_allocations SET delivered_quantity=delivered_quantity+$1 WHERE id=$2`, take, a.id); e != nil {
				return nil, e
			}
			if _, e = tx.Exec(ctx, `INSERT INTO delivery_note_items(delivery_note_id,sales_order_item_id,item_id,batch_id,quantity_delivered,created_by) VALUES($1,$2,$3,$4,$5,$6)`, noteID, line.SalesOrderItemID, itemID, a.batchID, take, actor); e != nil {
				return nil, e
			}
			if _, e = tx.Exec(ctx, `INSERT INTO stock_movements(movement_type,reference_type,reference_id,item_id,warehouse_id,batch_id,quantity,performed_by,notes) VALUES('sale','delivery_note',$1,$2,$3,$4,$5,$6,$7)`, noteID, itemID, a.warehouseID, a.batchID, -take, actor, req.Notes); e != nil {
				return nil, e
			}
			toDeliver -= take
		}
		if toDeliver > 1e-8 {
			return nil, errors.New("reserved allocation is insufficient for delivery quantity")
		}
		if _, err = tx.Exec(ctx, `UPDATE sales_order_items SET delivered_quantity=delivered_quantity+$1,updated_by=$2,updated_at=NOW() WHERE id=$3`, line.QuantityDelivered, actor, line.SalesOrderItemID); err != nil {
			return nil, err
		}
	}
	var remaining float64
	if err = tx.QueryRow(ctx, `SELECT COALESCE(SUM(quantity-delivered_quantity),0) FROM sales_order_items WHERE sales_order_id=$1`, orderID).Scan(&remaining); err != nil {
		return nil, err
	}
	newStatus := "partially_delivered"
	if remaining <= 1e-8 {
		newStatus = "delivered"
	}
	if _, err = tx.Exec(ctx, `UPDATE sales_orders SET status=$1,updated_by=$2,updated_at=NOW() WHERE id=$3`, newStatus, actor, orderID); err != nil {
		return nil, err
	}
	if _, err = tx.Exec(ctx, `INSERT INTO notifications(user_id,title,message,entity_type,entity_id,created_by) VALUES($1,'Sales order delivery recorded',$2,'delivery_note',$3,$4)`, orderCreator, "A delivery note was created for your sales order "+number+".", noteID, actor); err != nil {
		return nil, err
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, err
	}
	return r.GetDelivery(ctx, noteID)
}

func (r *WorkflowRepository) GetDelivery(ctx context.Context, id uuid.UUID) (*DeliveryNote, error) {
	n := &DeliveryNote{}
	err := r.db.QueryRow(ctx, `SELECT dn.id,dn.delivery_number,dn.sales_order_id,so.order_number,dn.delivered_by,u.full_name,dn.delivery_date,dn.recipient_name,dn.recipient_signature_reference,dn.notes,dn.created_at FROM delivery_notes dn JOIN sales_orders so ON so.id=dn.sales_order_id JOIN users u ON u.id=dn.delivered_by WHERE dn.id=$1 AND dn.deleted_at IS NULL`, id).Scan(&n.ID, &n.DeliveryNumber, &n.SalesOrderID, &n.OrderNumber, &n.DeliveredBy, &n.DeliveredByName, &n.DeliveryDate, &n.RecipientName, &n.RecipientSignatureReference, &n.Notes, &n.CreatedAt)
	if err != nil {
		return nil, err
	}
	rows, err := r.db.Query(ctx, `SELECT dni.id,dni.delivery_note_id,dni.sales_order_item_id,dni.item_id,i.code,i.name,dni.batch_id,b.batch_number,dni.quantity_delivered FROM delivery_note_items dni JOIN items i ON i.id=dni.item_id LEFT JOIN stock_batches b ON b.id=dni.batch_id WHERE dni.delivery_note_id=$1 ORDER BY dni.created_at`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var item DeliveryNoteItem
		if err = rows.Scan(&item.ID, &item.DeliveryNoteID, &item.SalesOrderItemID, &item.ItemID, &item.ItemCode, &item.ItemName, &item.BatchID, &item.BatchNumber, &item.QuantityDelivered); err != nil {
			return nil, err
		}
		n.Items = append(n.Items, item)
	}
	return n, rows.Err()
}

func (r *WorkflowRepository) ListDeliveries(ctx context.Context, orderID *uuid.UUID, page, limit int) ([]DeliveryNote, int64, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}
	where := "WHERE dn.deleted_at IS NULL"
	args := []any{}
	if orderID != nil {
		where += " AND dn.sales_order_id=$1"
		args = append(args, *orderID)
	}
	var total int64
	if err := r.db.QueryRow(ctx, "SELECT COUNT(*) FROM delivery_notes dn "+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	q := fmt.Sprintf(`SELECT dn.id,dn.delivery_number,dn.sales_order_id,so.order_number,dn.delivered_by,u.full_name,dn.delivery_date,dn.recipient_name,dn.recipient_signature_reference,dn.notes,dn.created_at FROM delivery_notes dn JOIN sales_orders so ON so.id=dn.sales_order_id JOIN users u ON u.id=dn.delivered_by %s ORDER BY dn.delivery_date DESC LIMIT $%d OFFSET $%d`, where, len(args)+1, len(args)+2)
	args = append(args, limit, (page-1)*limit)
	rows, err := r.db.Query(ctx, q, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	list := make([]DeliveryNote, 0)
	for rows.Next() {
		var n DeliveryNote
		if err = rows.Scan(&n.ID, &n.DeliveryNumber, &n.SalesOrderID, &n.OrderNumber, &n.DeliveredBy, &n.DeliveredByName, &n.DeliveryDate, &n.RecipientName, &n.RecipientSignatureReference, &n.Notes, &n.CreatedAt); err != nil {
			return nil, 0, err
		}
		list = append(list, n)
	}
	return list, total, rows.Err()
}
