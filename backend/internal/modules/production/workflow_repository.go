package production

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type WorkflowRepository struct{ db *pgxpool.Pool }

func NewWorkflowRepository(db *pgxpool.Pool) *WorkflowRepository { return &WorkflowRepository{db: db} }

func (r *WorkflowRepository) Create(ctx context.Context, actor uuid.UUID, req CreateReceiptRequest) (*Receipt, error) {
	if len(req.Items) == 0 {
		return nil, errors.New("at least one finished-good line is required")
	}
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	var warehouseType string
	if err = tx.QueryRow(ctx, `SELECT warehouse_type FROM warehouses WHERE id=$1 AND is_active=TRUE`, req.WarehouseID).Scan(&warehouseType); err != nil {
		return nil, errors.New("active production warehouse not found")
	}
	if warehouseType != "finished_goods" {
		return nil, errors.New("production receipts must be recorded in a finished-goods warehouse")
	}
	var receiptNumber string
	if err = tx.QueryRow(ctx, `SELECT 'PROD-'||to_char(CURRENT_DATE,'YYYY')||'-'||lpad(nextval('production_receipt_number_seq')::text,6,'0')`).Scan(&receiptNumber); err != nil {
		return nil, err
	}
	receiptID := uuid.New()
	receipt := &Receipt{ID: receiptID, ReceiptNumber: receiptNumber, WarehouseID: req.WarehouseID, RecordedBy: actor, Notes: req.Notes}
	if err = tx.QueryRow(ctx, `INSERT INTO production_receipts(id,receipt_number,warehouse_id,recorded_by,notes,created_by,updated_by) VALUES($1,$2,$3,$4,$5,$4,$4) RETURNING receipt_date,created_at`, receiptID, receiptNumber, req.WarehouseID, actor, req.Notes).Scan(&receipt.ReceiptDate, &receipt.CreatedAt); err != nil {
		return nil, err
	}
	for _, line := range req.Items {
		if line.QuantityProduced <= 0 {
			return nil, errors.New("quantity_produced must be greater than 0")
		}
		quality := strings.ToLower(strings.TrimSpace(line.QualityStatus))
		if quality != "accepted" && quality != "rejected" {
			return nil, errors.New("quality_status must be accepted or rejected")
		}
		var itemType string
		var tracked, expiryTracked bool
		if err = tx.QueryRow(ctx, `SELECT item_type,is_batch_tracked,is_expiry_tracked FROM items WHERE id=$1 AND deleted_at IS NULL AND is_active=TRUE`, line.ItemID).Scan(&itemType, &tracked, &expiryTracked); err != nil {
			return nil, errors.New("active product not found")
		}
		if itemType != "finished_good" {
			return nil, errors.New("production receipts accept finished_good items only")
		}
		batchNumber := ""
		if line.BatchNumber != nil {
			batchNumber = strings.TrimSpace(*line.BatchNumber)
		}
		if tracked && batchNumber == "" {
			return nil, errors.New("batch_number is required for batch-tracked items")
		}
		if expiryTracked && quality == "accepted" && (line.ExpiryDate == nil || *line.ExpiryDate == "") {
			return nil, errors.New("expiry_date is required for expiry-tracked items")
		}
		if line.UnitCost != nil && *line.UnitCost < 0 {
			return nil, errors.New("unit_cost cannot be negative")
		}
		var manufacture, expiry *time.Time
		if line.ManufactureDate != nil && *line.ManufactureDate != "" {
			d, e := time.Parse("2006-01-02", *line.ManufactureDate)
			if e != nil {
				return nil, errors.New("manufacture_date must use YYYY-MM-DD")
			}
			manufacture = &d
		}
		if line.ExpiryDate != nil && *line.ExpiryDate != "" {
			d, e := time.Parse("2006-01-02", *line.ExpiryDate)
			if e != nil {
				return nil, errors.New("expiry_date must use YYYY-MM-DD")
			}
			expiry = &d
		}
		var batchID *uuid.UUID
		if quality == "accepted" {
			if batchNumber == "" {
				batchNumber = receiptNumber + "-" + line.ItemID.String()
			}
			var id uuid.UUID
			err = tx.QueryRow(ctx, `INSERT INTO stock_batches(item_id,warehouse_id,batch_number,quantity,manufacture_date,expiry_date,unit_cost) VALUES($1,$2,$3,$4,$5,$6,$7) ON CONFLICT(item_id,warehouse_id,batch_number) DO UPDATE SET quantity=stock_batches.quantity+EXCLUDED.quantity,manufacture_date=COALESCE(stock_batches.manufacture_date,EXCLUDED.manufacture_date),expiry_date=COALESCE(stock_batches.expiry_date,EXCLUDED.expiry_date),unit_cost=COALESCE(EXCLUDED.unit_cost,stock_batches.unit_cost) RETURNING id`, line.ItemID, req.WarehouseID, batchNumber, line.QuantityProduced, manufacture, expiry, line.UnitCost).Scan(&id)
			if err != nil {
				return nil, err
			}
			batchID = &id
			if _, err = tx.Exec(ctx, `INSERT INTO stock_balances(item_id,warehouse_id,quantity,reserved_qty) VALUES($1,$2,$3,0) ON CONFLICT(item_id,warehouse_id) DO UPDATE SET quantity=stock_balances.quantity+EXCLUDED.quantity,updated_at=NOW()`, line.ItemID, req.WarehouseID, line.QuantityProduced); err != nil {
				return nil, err
			}
			if _, err = tx.Exec(ctx, `INSERT INTO stock_movements(movement_type,reference_type,reference_id,item_id,warehouse_id,batch_id,quantity,unit_cost,notes,performed_by) VALUES('production_receipt','production_receipt',$1,$2,$3,$4,$5,$6,$7,$8)`, receiptID, line.ItemID, req.WarehouseID, batchID, line.QuantityProduced, line.UnitCost, req.Notes, actor); err != nil {
				return nil, err
			}
		}
		var unitCost any
		if line.UnitCost != nil {
			unitCost = *line.UnitCost
		}
		_, err = tx.Exec(ctx, `INSERT INTO production_receipt_items(production_receipt_id,item_id,batch_id,quantity_produced,batch_number,manufacture_date,expiry_date,unit_cost,quality_status,created_by,updated_by) VALUES($1,$2,$3,$4,NULLIF($5,''),$6,$7,$8,$9,$10,$10)`, receiptID, line.ItemID, batchID, line.QuantityProduced, batchNumber, manufacture, expiry, unitCost, quality, actor)
		if err != nil {
			return nil, err
		}
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, err
	}
	return r.GetByID(ctx, receiptID)
}

func (r *WorkflowRepository) GetByID(ctx context.Context, id uuid.UUID) (*Receipt, error) {
	v := &Receipt{}
	err := r.db.QueryRow(ctx, `SELECT pr.id,pr.receipt_number,pr.warehouse_id,w.name,pr.recorded_by,u.full_name,pr.receipt_date,pr.notes,pr.created_at FROM production_receipts pr JOIN warehouses w ON w.id=pr.warehouse_id JOIN users u ON u.id=pr.recorded_by WHERE pr.id=$1 AND pr.deleted_at IS NULL`, id).Scan(&v.ID, &v.ReceiptNumber, &v.WarehouseID, &v.WarehouseName, &v.RecordedBy, &v.RecordedByName, &v.ReceiptDate, &v.Notes, &v.CreatedAt)
	if err != nil {
		return nil, err
	}
	rows, err := r.db.Query(ctx, `SELECT pri.id,pri.production_receipt_id,pri.item_id,i.code,i.name,pri.batch_id,pri.quantity_produced,pri.batch_number,pri.manufacture_date,pri.expiry_date,pri.unit_cost,pri.quality_status FROM production_receipt_items pri JOIN items i ON i.id=pri.item_id WHERE pri.production_receipt_id=$1 ORDER BY pri.created_at`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var item ReceiptItem
		if err = rows.Scan(&item.ID, &item.ProductionReceiptID, &item.ItemID, &item.ItemCode, &item.ItemName, &item.BatchID, &item.QuantityProduced, &item.BatchNumber, &item.ManufactureDate, &item.ExpiryDate, &item.UnitCost, &item.QualityStatus); err != nil {
			return nil, err
		}
		v.Items = append(v.Items, item)
	}
	return v, rows.Err()
}

func (r *WorkflowRepository) List(ctx context.Context, page, limit int) ([]Receipt, int64, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}
	var total int64
	if err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM production_receipts WHERE deleted_at IS NULL`).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := r.db.Query(ctx, `SELECT pr.id,pr.receipt_number,pr.warehouse_id,w.name,pr.recorded_by,u.full_name,pr.receipt_date,pr.notes,pr.created_at FROM production_receipts pr JOIN warehouses w ON w.id=pr.warehouse_id JOIN users u ON u.id=pr.recorded_by WHERE pr.deleted_at IS NULL ORDER BY pr.receipt_date DESC LIMIT $1 OFFSET $2`, limit, (page-1)*limit)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	list := make([]Receipt, 0)
	for rows.Next() {
		var item Receipt
		if err = rows.Scan(&item.ID, &item.ReceiptNumber, &item.WarehouseID, &item.WarehouseName, &item.RecordedBy, &item.RecordedByName, &item.ReceiptDate, &item.Notes, &item.CreatedAt); err != nil {
			return nil, 0, err
		}
		list = append(list, item)
	}
	return list, total, rows.Err()
}
