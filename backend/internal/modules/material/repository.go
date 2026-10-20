package material

import (
	"context"
	"fmt"
	"time"

	"github.com/Ethio-ERP/erpmate/internal/common/statusnotify"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) GenerateMRNumber(ctx context.Context) (string, error) {
	var count int
	err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM material_requisitions`).Scan(&count)
	if err != nil {
		return "", err
	}
	year := time.Now().Year()
	return fmt.Sprintf("MR-%d-%04d", year, count+1), nil
}

func (r *Repository) CheckStock(ctx context.Context, itemID uuid.UUID) (float64, float64, error) {
	var available, reserved float64
	err := r.db.QueryRow(ctx, `SELECT COALESCE(SUM(sb.available_qty),0),COALESCE(SUM(sb.reserved_qty),0) FROM stock_balances sb JOIN warehouses w ON w.id=sb.warehouse_id WHERE sb.item_id=$1 AND w.warehouse_type='raw_material' AND w.is_active=TRUE`, itemID).Scan(&available, &reserved)
	return available, reserved, err
}

func (r *Repository) Create(ctx context.Context, mr *MaterialRequisition) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	query := `
		INSERT INTO material_requisitions (
			id, mr_number, department_id, requested_by, required_date, purpose, status, created_by, updated_by
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
		RETURNING created_at, updated_at
	`
	err = tx.QueryRow(ctx, query,
		mr.ID, mr.MRNumber, mr.DepartmentID, mr.RequestedBy,
		mr.RequiredDate, mr.Purpose, mr.Status, mr.RequestedBy, mr.RequestedBy,
	).Scan(&mr.CreatedAt, &mr.UpdatedAt)
	if err != nil {
		return err
	}

	for i := range mr.Items {
		item := &mr.Items[i]
		item.ID = uuid.New()
		item.MRID = mr.ID

		_, err = tx.Exec(ctx, `
			INSERT INTO material_requisition_items (id, mr_id, item_id, quantity, notes, created_by)
			VALUES ($1,$2,$3,$4,$5,$6)
		`, item.ID, item.MRID, item.ItemID, item.Quantity, item.Notes, mr.RequestedBy)
		if err != nil {
			return err
		}
	}

	// status history
	_, err = tx.Exec(ctx, `
		INSERT INTO status_history (entity_type, entity_id, new_status, changed_by)
		VALUES ('material_requisition', $1, $2, $3)
	`, mr.ID, mr.Status, mr.RequestedBy)
	if err != nil {
		return err
	}
	if err := statusnotify.Create(ctx, tx, "material_requisition", mr.ID, mr.RequestedBy, mr.RequestedBy, mr.Status); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (r *Repository) GetByID(ctx context.Context, id uuid.UUID) (*MaterialRequisition, error) {
	query := `
		SELECT mr.id, mr.mr_number, mr.department_id, d.name, mr.requested_by, u.full_name,
		       mr.required_date, mr.purpose, mr.status,
		       mr.pm_approved_by, mr.pm_approved_at, mr.pm_rejection_reason,
		       mr.created_at, mr.updated_at
		FROM material_requisitions mr
		LEFT JOIN departments d ON d.id = mr.department_id
		LEFT JOIN users u ON u.id = mr.requested_by
		WHERE mr.id = $1 AND mr.deleted_at IS NULL
	`
	var mr MaterialRequisition
	err := r.db.QueryRow(ctx, query, id).Scan(
		&mr.ID, &mr.MRNumber, &mr.DepartmentID, &mr.DepartmentName,
		&mr.RequestedBy, &mr.RequesterName, &mr.RequiredDate, &mr.Purpose, &mr.Status,
		&mr.PMApprovedBy, &mr.PMApprovedAt, &mr.PMRejectionReason,
		&mr.CreatedAt, &mr.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	rows, err := r.db.Query(ctx, `
		SELECT mri.id, mri.mr_id, mri.item_id, i.code, i.name,
		       mri.quantity, mri.issued_qty, mri.notes, COALESCE(stock.available_qty,0), COALESCE(stock.reserved_qty,0)
		FROM material_requisition_items mri
		JOIN items i ON i.id = mri.item_id
		LEFT JOIN LATERAL (
			SELECT SUM(sb.available_qty) AS available_qty, SUM(sb.reserved_qty) AS reserved_qty
			FROM stock_balances sb JOIN warehouses w ON w.id=sb.warehouse_id
			WHERE sb.item_id=mri.item_id AND w.warehouse_type='raw_material' AND w.is_active=TRUE
		) stock ON TRUE
		WHERE mri.mr_id = $1
	`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var item MRItem
		err := rows.Scan(
			&item.ID, &item.MRID, &item.ItemID, &item.ItemCode, &item.ItemName,
			&item.Quantity, &item.IssuedQty, &item.Notes, &item.AvailableQty, &item.ReservedQty,
		)
		if err != nil {
			return nil, err
		}
		item.StockSufficient = item.AvailableQty >= item.Quantity
		item.SuggestPurchaseRequisition = !item.StockSufficient
		mr.Items = append(mr.Items, item)
	}

	return &mr, nil
}

func (r *Repository) List(ctx context.Context, userID uuid.UUID, role, status string, page, limit int) ([]MaterialRequisition, int64, error) {
	offset := (page - 1) * limit
	var args []interface{}
	where := "WHERE mr.deleted_at IS NULL"
	argPos := 1

	if role == "department_head" {
		where += fmt.Sprintf(" AND mr.requested_by = $%d", argPos)
		args = append(args, userID)
		argPos++
	}

	if status != "" {
		where += fmt.Sprintf(" AND mr.status = $%d", argPos)
		args = append(args, status)
		argPos++
	}

	var total int64
	countQuery := "SELECT COUNT(*) FROM material_requisitions mr " + where
	if err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	query := fmt.Sprintf(`
		SELECT mr.id, mr.mr_number, mr.department_id, d.name, mr.requested_by, u.full_name,
		       mr.required_date, mr.purpose, mr.status, mr.created_at
		FROM material_requisitions mr
		LEFT JOIN departments d ON d.id = mr.department_id
		LEFT JOIN users u ON u.id = mr.requested_by
		%s
		ORDER BY mr.created_at DESC
		LIMIT $%d OFFSET $%d
	`, where, argPos, argPos+1)

	args = append(args, limit, offset)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var list []MaterialRequisition
	for rows.Next() {
		var mr MaterialRequisition
		err := rows.Scan(
			&mr.ID, &mr.MRNumber, &mr.DepartmentID, &mr.DepartmentName,
			&mr.RequestedBy, &mr.RequesterName, &mr.RequiredDate, &mr.Purpose,
			&mr.Status, &mr.CreatedAt,
		)
		if err != nil {
			return nil, 0, err
		}
		list = append(list, mr)
	}

	return list, total, nil
}

func (r *Repository) UpdateStatus(ctx context.Context, id uuid.UUID, newStatus string, changedBy uuid.UUID, reason *string, fields map[string]interface{}) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var oldStatus string
	var requesterID uuid.UUID
	err = tx.QueryRow(ctx, `SELECT status, requested_by FROM material_requisitions WHERE id = $1 AND deleted_at IS NULL FOR UPDATE`, id).Scan(&oldStatus, &requesterID)
	if err != nil {
		return err
	}
	if err := statusnotify.Create(ctx, tx, "material_requisition", id, requesterID, changedBy, newStatus); err != nil {
		return err
	}

	query := "UPDATE material_requisitions SET status = $1, updated_at = NOW(), updated_by=$2"
	args := []interface{}{newStatus, changedBy}
	pos := 3

	for k, v := range fields {
		query += fmt.Sprintf(", %s = $%d", k, pos)
		args = append(args, v)
		pos++
	}
	query += fmt.Sprintf(" WHERE id = $%d", pos)
	args = append(args, id)

	_, err = tx.Exec(ctx, query, args...)
	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO status_history (entity_type, entity_id, old_status, new_status, changed_by, reason)
		VALUES ('material_requisition', $1, $2, $3, $4, $5)
	`, id, oldStatus, newStatus, changedBy, reason)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (r *Repository) GenerateIssueNumber(ctx context.Context) (string, error) {
	var count int
	err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM material_issues`).Scan(&count)
	if err != nil {
		return "", err
	}
	year := time.Now().Year()
	return fmt.Sprintf("MI-%d-%04d", year, count+1), nil
}

func (r *Repository) CreateIssue(ctx context.Context, issue *MaterialIssue) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var mrStatus string
	var requesterID uuid.UUID
	err = tx.QueryRow(ctx, `SELECT status, requested_by FROM material_requisitions WHERE id=$1 AND deleted_at IS NULL FOR UPDATE`, issue.MRID).Scan(&mrStatus, &requesterID)
	if err != nil {
		return err
	}
	if mrStatus != "approved" && mrStatus != "partially_issued" {
		return fmt.Errorf("material requisition is not issuable in status %s", mrStatus)
	}
	var oldStatus string
	oldStatus = mrStatus
	err = tx.QueryRow(ctx, `
		INSERT INTO material_issues (id, issue_number, mr_id, issued_by, issued_to, notes,created_by,updated_by)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
		RETURNING issued_date, created_at
	`, issue.ID, issue.IssueNumber, issue.MRID, issue.IssuedBy, issue.IssuedTo, issue.Notes, issue.IssuedBy, issue.IssuedBy).
		Scan(&issue.IssuedDate, &issue.CreatedAt)
	if err != nil {
		return err
	}
	if issue.IssuedTo != nil {
		if _, err = tx.Exec(ctx, `INSERT INTO notifications (user_id,title,message,entity_type,entity_id,created_by) VALUES ($1,'Material issued','Material has been issued and is awaiting receipt acknowledgment.','material_issue',$2,$3)`, *issue.IssuedTo, issue.ID, issue.IssuedBy); err != nil {
			return err
		}
	}

	for i := range issue.Items {
		item := &issue.Items[i]
		item.ID = uuid.New()
		item.IssueID = issue.ID
		var requisitionItemID, requestedItemID uuid.UUID
		var requestedQty, issuedQty float64
		err = tx.QueryRow(ctx, `SELECT id,item_id,quantity,issued_qty FROM material_requisition_items WHERE id=$1 AND mr_id=$2 FOR UPDATE`, item.MRItemID, issue.MRID).Scan(&requisitionItemID, &requestedItemID, &requestedQty, &issuedQty)
		if err != nil {
			return fmt.Errorf("material requisition item not found: %w", err)
		}
		if item.ItemID != requestedItemID {
			return fmt.Errorf("issue item does not match the requisition item")
		}
		if item.Quantity <= 0 || issuedQty+item.Quantity > requestedQty {
			return fmt.Errorf("issue quantity exceeds the remaining requisition quantity")
		}
		var batchTracked bool
		if err = tx.QueryRow(ctx, `SELECT is_batch_tracked FROM items WHERE id=$1 AND deleted_at IS NULL`, item.ItemID).Scan(&batchTracked); err != nil {
			return err
		}
		if batchTracked && item.BatchID == nil {
			return fmt.Errorf("batch_id is required for batch-tracked item")
		}

		_, err = tx.Exec(ctx, `
			INSERT INTO material_issue_items (id, issue_id, mr_item_id, item_id, batch_id, quantity,created_by)
			VALUES ($1,$2,$3,$4,$5,$6,$7)
		`, item.ID, item.IssueID, item.MRItemID, item.ItemID, item.BatchID, item.Quantity, issue.IssuedBy)
		if err != nil {
			return err
		}
		if item.BatchID != nil {
			var warehouseID uuid.UUID
			var batchQty float64
			err = tx.QueryRow(ctx, `SELECT warehouse_id,quantity FROM stock_batches WHERE id=$1 AND item_id=$2 FOR UPDATE`, *item.BatchID, item.ItemID).Scan(&warehouseID, &batchQty)
			if err != nil {
				return fmt.Errorf("stock batch not found for item")
			}
			if batchQty < item.Quantity {
				return fmt.Errorf("insufficient quantity in selected batch")
			}
			var available float64
			err = tx.QueryRow(ctx, `SELECT sb.available_qty FROM stock_balances sb JOIN warehouses w ON w.id=sb.warehouse_id WHERE sb.item_id=$1 AND sb.warehouse_id=$2 AND w.warehouse_type='raw_material' AND w.is_active=TRUE FOR UPDATE OF sb`, item.ItemID, warehouseID).Scan(&available)
			if err != nil || available < item.Quantity {
				return fmt.Errorf("insufficient available raw-material stock")
			}
			if _, err = tx.Exec(ctx, `UPDATE stock_batches SET quantity=quantity-$1 WHERE id=$2`, item.Quantity, *item.BatchID); err != nil {
				return err
			}
			if _, err = tx.Exec(ctx, `UPDATE stock_balances SET quantity=quantity-$1,updated_at=NOW() WHERE item_id=$2 AND warehouse_id=$3 AND quantity-reserved_qty >= $1`, item.Quantity, item.ItemID, warehouseID); err != nil {
				return err
			}
			if _, err = tx.Exec(ctx, `INSERT INTO stock_movements (movement_type,reference_type,reference_id,item_id,warehouse_id,batch_id,quantity,performed_by,notes) VALUES ('issue','material_issue',$1,$2,$3,$4,$5,$6,$7)`, issue.ID, item.ItemID, warehouseID, item.BatchID, -item.Quantity, issue.IssuedBy, issue.Notes); err != nil {
				return err
			}
		} else {
			type balanceRow struct {
				id, warehouseID uuid.UUID
				available       float64
			}
			rows, queryErr := tx.Query(ctx, `SELECT sb.id,sb.warehouse_id,sb.available_qty FROM stock_balances sb JOIN warehouses w ON w.id=sb.warehouse_id WHERE sb.item_id=$1 AND w.warehouse_type='raw_material' AND w.is_active=TRUE AND sb.available_qty>0 ORDER BY w.code FOR UPDATE OF sb`, item.ItemID)
			if queryErr != nil {
				return queryErr
			}
			balances := make([]balanceRow, 0)
			var totalAvailable float64
			for rows.Next() {
				var b balanceRow
				if scanErr := rows.Scan(&b.id, &b.warehouseID, &b.available); scanErr != nil {
					rows.Close()
					return scanErr
				}
				balances = append(balances, b)
				totalAvailable += b.available
			}
			rows.Close()
			if totalAvailable < item.Quantity {
				return fmt.Errorf("insufficient available raw-material stock")
			}
			remaining := item.Quantity
			for _, b := range balances {
				if remaining <= 0 {
					break
				}
				take := b.available
				if take > remaining {
					take = remaining
				}
				if _, err = tx.Exec(ctx, `UPDATE stock_balances SET quantity=quantity-$1,updated_at=NOW() WHERE id=$2`, take, b.id); err != nil {
					return err
				}
				if _, err = tx.Exec(ctx, `INSERT INTO stock_movements (movement_type,reference_type,reference_id,item_id,warehouse_id,quantity,performed_by,notes) VALUES ('issue','material_issue',$1,$2,$3,$4,$5,$6)`, issue.ID, item.ItemID, b.warehouseID, -take, issue.IssuedBy, issue.Notes); err != nil {
					return err
				}
				remaining -= take
			}
		}

		// Update issued_qty on the material_requisition_item
		result, updateErr := tx.Exec(ctx, `
			UPDATE material_requisition_items
			SET issued_qty = issued_qty + $1, updated_by=$3
			WHERE id = $2 AND issued_qty + $1 <= quantity
		`, item.Quantity, item.MRItemID, issue.IssuedBy)
		if updateErr != nil {
			return updateErr
		}
		if result.RowsAffected() != 1 {
			return fmt.Errorf("issue quantity exceeds the remaining requisition quantity")
		}
	}

	// Check if the Material Requisition is fully issued
	var remaining float64
	err = tx.QueryRow(ctx, `
		SELECT COALESCE(SUM(quantity - issued_qty), 0)
		FROM material_requisition_items
		WHERE mr_id = $1
	`, issue.MRID).Scan(&remaining)
	if err != nil {
		return err
	}

	newStatus := "partially_issued"
	if remaining <= 0 {
		newStatus = "fully_issued"
	}

	_, err = tx.Exec(ctx, `
		UPDATE material_requisitions SET status = $1, updated_at = NOW(), updated_by=$3 WHERE id = $2
	`, newStatus, issue.MRID, issue.IssuedBy)
	if err != nil {
		return err
	}

	// Status history
	_, err = tx.Exec(ctx, `
		INSERT INTO status_history (entity_type, entity_id, old_status, new_status, changed_by)
		VALUES ('material_requisition', $1, $2, $3, $4)
	`, issue.MRID, oldStatus, newStatus, issue.IssuedBy)
	if err != nil {
		return err
	}
	if newStatus != oldStatus {
		if err = statusnotify.Create(ctx, tx, "material_requisition", issue.MRID, requesterID, issue.IssuedBy, newStatus); err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

func (r *Repository) GetIssueByID(ctx context.Context, id uuid.UUID) (*MaterialIssue, error) {
	query := `
		SELECT mi.id, mi.issue_number, mi.mr_id, mr.mr_number,
		       mi.issued_by, u1.full_name, mi.issued_to, u2.full_name,
		       mi.issued_date, mi.notes, mi.created_at, mi.acknowledged_by, mi.acknowledged_at
		FROM material_issues mi
		JOIN material_requisitions mr ON mr.id = mi.mr_id
		JOIN users u1 ON u1.id = mi.issued_by
		LEFT JOIN users u2 ON u2.id = mi.issued_to
		WHERE mi.id = $1
	`
	var issue MaterialIssue
	err := r.db.QueryRow(ctx, query, id).Scan(
		&issue.ID, &issue.IssueNumber, &issue.MRID, &issue.MRNumber,
		&issue.IssuedBy, &issue.IssuedByName, &issue.IssuedTo, &issue.IssuedToName,
		&issue.IssuedDate, &issue.Notes, &issue.CreatedAt, &issue.AcknowledgedBy, &issue.AcknowledgedAt,
	)
	if err != nil {
		return nil, err
	}

	rows, err := r.db.Query(ctx, `
		SELECT mii.id, mii.issue_id, mii.mr_item_id, mii.item_id, i.code, i.name,
		       mii.batch_id, mii.quantity
		FROM material_issue_items mii
		JOIN items i ON i.id = mii.item_id
		WHERE mii.issue_id = $1
	`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var item MaterialIssueItem
		err := rows.Scan(
			&item.ID, &item.IssueID, &item.MRItemID, &item.ItemID,
			&item.ItemCode, &item.ItemName, &item.BatchID, &item.Quantity,
		)
		if err != nil {
			return nil, err
		}
		issue.Items = append(issue.Items, item)
	}

	return &issue, nil
}

func (r *Repository) ListIssuesByMR(ctx context.Context, mrID uuid.UUID) ([]MaterialIssue, error) {
	rows, err := r.db.Query(ctx, `
		SELECT mi.id, mi.issue_number, mi.mr_id, mi.issued_by, u.full_name,
		       mi.issued_to, mi.acknowledged_by, mi.acknowledged_at,
		       mi.issued_date, mi.notes, mi.created_at
		FROM material_issues mi
		JOIN users u ON u.id = mi.issued_by
		WHERE mi.mr_id = $1
		ORDER BY mi.created_at DESC
	`, mrID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []MaterialIssue
	for rows.Next() {
		var issue MaterialIssue
		err := rows.Scan(
			&issue.ID, &issue.IssueNumber, &issue.MRID, &issue.IssuedBy, &issue.IssuedByName,
			&issue.IssuedTo, &issue.AcknowledgedBy, &issue.AcknowledgedAt,
			&issue.IssuedDate, &issue.Notes, &issue.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		list = append(list, issue)
	}
	return list, nil
}

func (r *Repository) GetStatusHistory(ctx context.Context, id uuid.UUID) ([]StatusHistory, error) {
	rows, err := r.db.Query(ctx, `SELECT sh.id,sh.entity_type,sh.entity_id,sh.old_status,sh.new_status,sh.changed_by,u.full_name,sh.reason,sh.created_at FROM status_history sh LEFT JOIN users u ON u.id=sh.changed_by WHERE sh.entity_type='material_requisition' AND sh.entity_id=$1 ORDER BY sh.created_at`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	history := make([]StatusHistory, 0)
	for rows.Next() {
		var entry StatusHistory
		if err = rows.Scan(&entry.ID, &entry.EntityType, &entry.EntityID, &entry.OldStatus, &entry.NewStatus, &entry.ChangedBy, &entry.ChangedByName, &entry.Reason, &entry.CreatedAt); err != nil {
			return nil, err
		}
		history = append(history, entry)
	}
	return history, rows.Err()
}

func (r *Repository) IsRequester(ctx context.Context, id, userID uuid.UUID) (bool, error) {
	var allowed bool
	err := r.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM material_requisitions WHERE id=$1 AND requested_by=$2 AND deleted_at IS NULL)`, id, userID).Scan(&allowed)
	return allowed, err
}

func (r *Repository) CanViewIssue(ctx context.Context, id, userID uuid.UUID) (bool, error) {
	var allowed bool
	err := r.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM material_issues mi JOIN material_requisitions mr ON mr.id=mi.mr_id WHERE mi.id=$1 AND (mi.issued_by=$2 OR mi.issued_to=$2 OR mr.requested_by=$2))`, id, userID).Scan(&allowed)
	return allowed, err
}

func (r *Repository) AcknowledgeIssue(ctx context.Context, id, userID uuid.UUID) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	var issuedBy uuid.UUID
	result, err := tx.Query(ctx, `UPDATE material_issues mi SET acknowledged_by=$1, acknowledged_at=NOW(), updated_by=$1 WHERE mi.id=$2 AND mi.acknowledged_at IS NULL AND (mi.issued_to=$1 OR EXISTS (SELECT 1 FROM material_requisitions mr WHERE mr.id=mi.mr_id AND mr.requested_by=$1)) RETURNING mi.issued_by,mi.issue_number`, userID, id)
	if err != nil {
		return err
	}
	defer result.Close()
	var issueNumber string
	if !result.Next() {
		return fmt.Errorf("issue not found, already acknowledged, or user is not the recipient/requester")
	}
	if err = result.Scan(&issuedBy, &issueNumber); err != nil {
		return err
	}
	if err = result.Err(); err != nil {
		return err
	}
	result.Close()
	if _, err = tx.Exec(ctx, `INSERT INTO notifications (user_id,title,message,entity_type,entity_id,created_by) VALUES ($1,'Material receipt acknowledged',$2,'material_issue',$3,$4)`, issuedBy, "Receipt acknowledged for material issue "+issueNumber+".", id, userID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
