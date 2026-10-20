package purchase

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/Ethio-ERP/erpmate/internal/common/statusnotify"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

// sortColumns whitelists the columns the list endpoint may sort by.
var sortColumns = map[string]string{
	"created_at":      "pr.created_at",
	"required_date":   "pr.required_date",
	"total_estimated": "pr.total_estimated",
	"pr_number":       "pr.pr_number",
	"status":          "pr.status",
}

// statusColumns whitelists the extra columns UpdateStatus may set.
var statusColumns = map[string]bool{
	"pm_approved_by":           true,
	"pm_approved_at":           true,
	"pm_rejection_reason":      true,
	"return_reason":            true,
	"finance_approved_by":      true,
	"finance_approved_at":      true,
	"finance_rejection_reason": true,
	"suspension_reason":        true,
}

const listFrom = `
	FROM purchase_requisitions pr
	LEFT JOIN departments d ON d.id = pr.department_id
	LEFT JOIN users u ON u.id = pr.requested_by`

// nextPRNumber generates PR-<year>-<seq>. It runs inside the creating
// transaction under an advisory lock, so concurrent creations can never get the
// same number (the old COUNT(*)+1 approach could, and also reused numbers).
func nextPRNumber(ctx context.Context, tx pgx.Tx) (string, error) {
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtext('purchase_requisitions.pr_number'))`); err != nil {
		return "", err
	}
	prefix := fmt.Sprintf("PR-%d-", time.Now().Year())
	var maxSeq int
	err := tx.QueryRow(ctx, `
		SELECT COALESCE(MAX(substring(pr_number FROM '[0-9]+$')::int), 0)
		FROM purchase_requisitions
		WHERE pr_number LIKE $1
	`, prefix+"%").Scan(&maxSeq)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s%04d", prefix, maxSeq+1), nil
}

func (r *Repository) Create(ctx context.Context, pr *PurchaseRequisition) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	pr.PRNumber, err = nextPRNumber(ctx, tx)
	if err != nil {
		return err
	}

	err = tx.QueryRow(ctx, `
		INSERT INTO purchase_requisitions (
			id, pr_number, department_id, requested_by, preferred_supplier_id, required_date, created_by, updated_by,
			justification, status, total_estimated
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
		RETURNING created_at, updated_at
	`,
		pr.ID, pr.PRNumber, pr.DepartmentID, pr.RequestedBy, pr.PreferredSupplierID, pr.RequiredDate, pr.RequestedBy, pr.RequestedBy,
		pr.Justification, pr.Status, pr.TotalEstimated,
	).Scan(&pr.CreatedAt, &pr.UpdatedAt)
	if err != nil {
		return err
	}

	for i := range pr.Items {
		item := &pr.Items[i]
		item.ID = uuid.New()
		item.PRID = pr.ID
		if err = insertItem(ctx, tx, item, pr.RequestedBy); err != nil {
			return err
		}
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO status_history (entity_type, entity_id, new_status, changed_by)
		VALUES ('purchase_requisition', $1, $2, $3)
	`, pr.ID, pr.Status, pr.RequestedBy)
	if err != nil {
		return err
	}
	if err = statusnotify.Create(ctx, tx, "purchase_requisition", pr.ID, pr.RequestedBy, pr.RequestedBy, pr.Status); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func insertItem(ctx context.Context, tx pgx.Tx, item *PRItem, userID uuid.UUID) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO purchase_requisition_items (
			id, pr_id, item_id, description, quantity, unit_of_measure, created_by,
			estimated_unit_price, estimated_total, notes
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
	`, item.ID, item.PRID, item.ItemID, item.Description, item.Quantity,
		item.UnitOfMeasure, userID, item.EstimatedUnitPrice, item.EstimatedTotal, item.Notes)
	return err
}

// CheckStock checks raw-material stock for all requested items with ONE query
// (the previous implementation ran one query per item). The result keeps the
// order of reqs. An unknown / deleted item yields a NotFound error.
func (r *Repository) CheckStock(ctx context.Context, reqs []StockRequest) ([]PRStockCheck, error) {
	checks := make([]PRStockCheck, 0, len(reqs))
	if len(reqs) == 0 {
		return checks, nil
	}

	ids := make([]uuid.UUID, len(reqs))
	for i, q := range reqs {
		ids[i] = q.ItemID
	}

	rows, err := r.db.Query(ctx, `
		SELECT i.id, i.code, i.name, COALESCE(s.qty, 0)
		FROM items i
		LEFT JOIN (
			SELECT sb.item_id, SUM(sb.available_qty) AS qty
			FROM stock_balances sb
			JOIN warehouses w ON w.id = sb.warehouse_id
			     AND w.warehouse_type = 'raw_material' AND w.is_active = TRUE
			WHERE sb.item_id = ANY($1)
			GROUP BY sb.item_id
		) s ON s.item_id = i.id
		WHERE i.id = ANY($1) AND i.deleted_at IS NULL
	`, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	found := make(map[uuid.UUID]PRStockCheck, len(reqs))
	for rows.Next() {
		var c PRStockCheck
		if err := rows.Scan(&c.ItemID, &c.ItemCode, &c.ItemName, &c.Available); err != nil {
			return nil, err
		}
		found[c.ItemID] = c
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	for _, q := range reqs {
		c, ok := found[q.ItemID]
		if !ok {
			return nil, NotFound(fmt.Sprintf("item %s was not found", q.ItemID))
		}
		c.Requested = q.Quantity
		c.Sufficient = c.Available >= q.Quantity
		if !c.Sufficient {
			c.Shortfall = round4(q.Quantity - c.Available)
		}
		checks = append(checks, c)
	}
	return checks, nil
}

func round4(v float64) float64 {
	return float64(int64(v*10000+0.5)) / 10000
}

// UpdateParams groups the arguments of Update.
type UpdateParams struct {
	ID            uuid.UUID
	UserID        uuid.UUID
	DepartmentID  uuid.UUID
	SupplierID    *uuid.UUID
	RequiredDate  *time.Time
	Justification *string
	Total         float64
	Items         []PRItem
}

// Update replaces the editable fields and all items. The row is locked and the
// owner / status are re-checked inside the transaction, so a concurrent submit
// can no longer be overwritten by a stale edit.
func (r *Repository) Update(ctx context.Context, p UpdateParams) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var status string
	var requester uuid.UUID
	err = tx.QueryRow(ctx, `SELECT status, requested_by FROM purchase_requisitions WHERE id=$1 AND deleted_at IS NULL FOR UPDATE`, p.ID).Scan(&status, &requester)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		return err
	}
	if requester != p.UserID {
		return Forbidden("you can only update your own requisitions")
	}
	if !IsEditableStatus(status) {
		return Conflict("only draft or returned requisitions can be updated")
	}

	_, err = tx.Exec(ctx, `
		UPDATE purchase_requisitions
		SET department_id=$1, preferred_supplier_id=$2, required_date=$3, justification=$4,
		    total_estimated=$5, return_reason=NULL, updated_at=NOW(), updated_by=$6
		WHERE id=$7
	`, p.DepartmentID, p.SupplierID, p.RequiredDate, p.Justification, p.Total, p.UserID, p.ID)
	if err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `DELETE FROM purchase_requisition_items WHERE pr_id=$1`, p.ID); err != nil {
		return err
	}
	for i := range p.Items {
		item := &p.Items[i]
		item.ID = uuid.New()
		item.PRID = p.ID
		if err = insertItem(ctx, tx, item, p.UserID); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

// SoftDelete removes a draft / returned requisition owned by userID.
func (r *Repository) SoftDelete(ctx context.Context, id, userID uuid.UUID) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var status string
	var requester uuid.UUID
	err = tx.QueryRow(ctx, `SELECT status, requested_by FROM purchase_requisitions WHERE id=$1 AND deleted_at IS NULL FOR UPDATE`, id).Scan(&status, &requester)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		return err
	}
	if requester != userID {
		return Forbidden("you can only delete your own requisitions")
	}
	if !IsEditableStatus(status) {
		return Conflict("only draft or returned requisitions can be deleted")
	}
	if _, err = tx.Exec(ctx, `UPDATE purchase_requisitions SET deleted_at=NOW(), updated_at=NOW(), updated_by=$1 WHERE id=$2`, userID, id); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *Repository) GetByID(ctx context.Context, id uuid.UUID) (*PurchaseRequisition, error) {
	query := `
		SELECT pr.id, pr.pr_number, pr.department_id, COALESCE(d.name,''), pr.requested_by, COALESCE(u.full_name,''),
		       pr.required_date, pr.justification, pr.status,
		       pr.pm_approved_by, pr.pm_approved_at, pr.pm_rejection_reason, pr.return_reason, pr.preferred_supplier_id,
		       pr.finance_approved_by, pr.finance_approved_at, pr.finance_rejection_reason,
		       pr.suspension_reason, pr.total_estimated, pr.created_at, pr.updated_at
		FROM purchase_requisitions pr
		LEFT JOIN departments d ON d.id = pr.department_id
		LEFT JOIN users u ON u.id = pr.requested_by
		WHERE pr.id = $1 AND pr.deleted_at IS NULL
	`
	var pr PurchaseRequisition
	err := r.db.QueryRow(ctx, query, id).Scan(
		&pr.ID, &pr.PRNumber, &pr.DepartmentID, &pr.DepartmentName, &pr.RequestedBy, &pr.RequesterName,
		&pr.RequiredDate, &pr.Justification, &pr.Status,
		&pr.PMApprovedBy, &pr.PMApprovedAt, &pr.PMRejectionReason, &pr.ReturnReason, &pr.PreferredSupplierID,
		&pr.FinanceApprovedBy, &pr.FinanceApprovedAt, &pr.FinanceRejectionReason,
		&pr.SuspensionReason, &pr.TotalEstimated, &pr.CreatedAt, &pr.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	rows, err := r.db.Query(ctx, `
		SELECT pri.id, pri.pr_id, pri.item_id, i.code, i.name, pri.description,
		       pri.quantity, pri.unit_of_measure, pri.estimated_unit_price,
		       pri.estimated_total, pri.notes
		FROM purchase_requisition_items pri
		LEFT JOIN items i ON i.id = pri.item_id
		WHERE pri.pr_id = $1
	`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	pr.Items = make([]PRItem, 0)
	for rows.Next() {
		var item PRItem
		if err := rows.Scan(
			&item.ID, &item.PRID, &item.ItemID, &item.ItemCode, &item.ItemName, &item.Description,
			&item.Quantity, &item.UnitOfMeasure, &item.EstimatedUnitPrice,
			&item.EstimatedTotal, &item.Notes,
		); err != nil {
			return nil, err
		}
		pr.Items = append(pr.Items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	pr.ItemCount = len(pr.Items)
	return &pr, nil
}

// GetRequester returns who created the requisition (used for view permission).
func (r *Repository) GetRequester(ctx context.Context, id uuid.UUID) (uuid.UUID, error) {
	var requester uuid.UUID
	err := r.db.QueryRow(ctx, `SELECT requested_by FROM purchase_requisitions WHERE id=$1 AND deleted_at IS NULL`, id).Scan(&requester)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return uuid.Nil, ErrNotFound
		}
		return uuid.Nil, err
	}
	return requester, nil
}

func escapeLike(s string) string {
	return strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(s)
}

func buildListWhere(actor Actor, f ListFilter) (string, []any) {
	conds := []string{"pr.deleted_at IS NULL"}
	var args []any
	add := func(cond string, v any) {
		args = append(args, v)
		conds = append(conds, strings.ReplaceAll(cond, "?", fmt.Sprintf("$%d", len(args))))
	}

	if actor.Role == RoleDepartmentHead || f.MineOnly {
		add("pr.requested_by = ?", actor.UserID)
	}
	if len(f.Statuses) > 0 {
		add("pr.status::text = ANY(?)", f.Statuses)
	}
	if f.DepartmentID != nil {
		add("pr.department_id = ?", *f.DepartmentID)
	}
	if f.DateFrom != nil {
		add("pr.created_at >= ?", *f.DateFrom)
	}
	if f.DateTo != nil {
		add("pr.created_at < ?", f.DateTo.AddDate(0, 0, 1))
	}
	if f.Search != "" {
		add(`(pr.pr_number ILIKE ? OR pr.justification ILIKE ? OR u.full_name ILIKE ? OR d.name ILIKE ?)`, "%"+escapeLike(f.Search)+"%")
	}
	return "WHERE " + strings.Join(conds, " AND "), args
}

func (r *Repository) List(ctx context.Context, actor Actor, f ListFilter) ([]PRSummary, int64, error) {
	where, args := buildListWhere(actor, f)

	var total int64
	if err := r.db.QueryRow(ctx, "SELECT COUNT(*) "+listFrom+" "+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	list := make([]PRSummary, 0)
	if total == 0 {
		return list, 0, nil
	}

	col, ok := sortColumns[f.SortBy]
	if !ok {
		col = sortColumns["created_at"]
	}
	dir := "DESC"
	if strings.EqualFold(f.SortDir, "asc") {
		dir = "ASC"
	}

	args = append(args, f.Limit, (f.Page-1)*f.Limit)
	query := fmt.Sprintf(`
		SELECT pr.id, pr.pr_number, pr.department_id, COALESCE(d.name,''), pr.requested_by, COALESCE(u.full_name,''),
		       pr.required_date, pr.justification, pr.status, pr.preferred_supplier_id, pr.total_estimated,
		       (SELECT COUNT(*) FROM purchase_requisition_items pri WHERE pri.pr_id = pr.id),
		       pr.created_at, pr.updated_at
		%s
		%s
		ORDER BY %s %s NULLS LAST, pr.id DESC
		LIMIT $%d OFFSET $%d
	`, listFrom, where, col, dir, len(args)-1, len(args))

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	for rows.Next() {
		var pr PRSummary
		if err := rows.Scan(
			&pr.ID, &pr.PRNumber, &pr.DepartmentID, &pr.DepartmentName, &pr.RequestedBy, &pr.RequesterName,
			&pr.RequiredDate, &pr.Justification, &pr.Status, &pr.PreferredSupplierID, &pr.TotalEstimated,
			&pr.ItemCount, &pr.CreatedAt, &pr.UpdatedAt,
		); err != nil {
			return nil, 0, err
		}
		list = append(list, pr)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

// StatusCounts returns requisition counts per status visible to the actor.
func (r *Repository) StatusCounts(ctx context.Context, actor Actor) (map[string]int, error) {
	query := `SELECT pr.status::text, COUNT(*) FROM purchase_requisitions pr WHERE pr.deleted_at IS NULL`
	var args []any
	if actor.Role == RoleDepartmentHead {
		query += ` AND pr.requested_by = $1`
		args = append(args, actor.UserID)
	}
	query += ` GROUP BY pr.status`

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	counts := make(map[string]int, len(AllStatuses))
	for _, s := range AllStatuses {
		counts[s] = 0
	}
	for rows.Next() {
		var status string
		var n int
		if err := rows.Scan(&status, &n); err != nil {
			return nil, err
		}
		counts[status] = n
	}
	return counts, rows.Err()
}

func containsString(list []string, v string) bool {
	for _, s := range list {
		if s == v {
			return true
		}
	}
	return false
}

// UpdateStatus moves a requisition to newStatus. The row is locked and the
// current status is verified against allowedFrom INSIDE the transaction, so two
// people acting at the same time can no longer both succeed (the old code only
// checked the status before taking the lock).
func (r *Repository) UpdateStatus(ctx context.Context, id uuid.UUID, allowedFrom []string, newStatus string, changedBy uuid.UUID, reason *string, fields map[string]any) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var oldStatus string
	var requesterID uuid.UUID
	err = tx.QueryRow(ctx, `SELECT status, requested_by FROM purchase_requisitions WHERE id = $1 AND deleted_at IS NULL FOR UPDATE`, id).Scan(&oldStatus, &requesterID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		return err
	}
	if !containsString(allowedFrom, oldStatus) {
		return Conflict(fmt.Sprintf("this requisition is %s and can no longer be changed by this action", strings.ReplaceAll(oldStatus, "_", " ")))
	}

	keys := make([]string, 0, len(fields))
	for k := range fields {
		if !statusColumns[k] {
			return fmt.Errorf("purchase: column %q cannot be updated through UpdateStatus", k)
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)

	query := "UPDATE purchase_requisitions SET status = $1, updated_at = NOW(), updated_by = $2"
	args := []any{newStatus, changedBy}

	for _, k := range keys {
		val := fields[k]
		// A bare nil has no type, so give it one.
		if val == nil {
			switch k {
			case "pm_approved_at", "finance_approved_at":
				val = (*time.Time)(nil)
			case "pm_approved_by", "finance_approved_by":
				val = (*uuid.UUID)(nil)
			default:
				val = (*string)(nil)
			}
		}
		args = append(args, val)
		query += fmt.Sprintf(", %s = $%d", k, len(args))
	}
	args = append(args, id)
	query += fmt.Sprintf(" WHERE id = $%d", len(args))

	if _, err = tx.Exec(ctx, query, args...); err != nil {
		return err
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO status_history (entity_type, entity_id, old_status, new_status, changed_by, reason)
		VALUES ('purchase_requisition', $1, $2, $3, $4, $5::text)
	`, id, oldStatus, newStatus, changedBy, reason)
	if err != nil {
		return err
	}
	if err = statusnotify.Create(ctx, tx, "purchase_requisition", id, requesterID, changedBy, newStatus); err != nil {
		return err
	}

	// Finance approval creates the procurement draft atomically. Purchasing fills in
	// the confirmed supplier, final prices, delivery terms, and then issues the PO.
	if newStatus == StatusApproved && oldStatus != StatusApproved {
		var poID uuid.UUID
		err = tx.QueryRow(ctx, `
			INSERT INTO purchase_orders (po_number,purchase_requisition_id,supplier_id,status,created_by,updated_by)
			SELECT 'PO-'||to_char(CURRENT_DATE,'YYYY')||'-'||lpad(nextval('purchase_order_number_seq')::text,6,'0'),
			       pr.id,pr.preferred_supplier_id,'draft',$2,$2
			FROM purchase_requisitions pr WHERE pr.id=$1
			ON CONFLICT (purchase_requisition_id) DO UPDATE SET purchase_requisition_id=EXCLUDED.purchase_requisition_id
			RETURNING id
		`, id, changedBy).Scan(&poID)
		if err != nil {
			return err
		}
		var existingItems int
		if err = tx.QueryRow(ctx, `SELECT COUNT(*) FROM purchase_order_items WHERE purchase_order_id=$1`, poID).Scan(&existingItems); err != nil {
			return err
		}
		if existingItems == 0 {
			if _, err = tx.Exec(ctx, `
				INSERT INTO purchase_order_items (purchase_order_id,purchase_requisition_item_id,item_id,description,quantity,unit_of_measure,unit_price,line_total,notes,created_by,updated_by)
				SELECT $1,pri.id,pri.item_id,pri.description,pri.quantity,pri.unit_of_measure,pri.estimated_unit_price,COALESCE(pri.estimated_total,0),pri.notes,$2,$2
				FROM purchase_requisition_items pri WHERE pri.pr_id=$3
			`, poID, changedBy, id); err != nil {
				return err
			}
			if _, err = tx.Exec(ctx, `UPDATE purchase_orders SET total_amount=COALESCE((SELECT SUM(line_total) FROM purchase_order_items WHERE purchase_order_id=$1),0) WHERE id=$1`, poID); err != nil {
				return err
			}
		}
		if _, err = tx.Exec(ctx, `
			INSERT INTO notifications(user_id, title, message, entity_type, entity_id, created_by) 
			SELECT DISTINCT u.id, 'Purchase order ready for purchasing', 'An approved purchase requisition is ready for supplier and price finalization.', 'purchase_order', $1::uuid, $2::uuid 
			FROM users u 
			JOIN roles r ON r.id=u.role_id 
			WHERE r.name='purchasing' AND u.is_active=TRUE AND u.deleted_at IS NULL
		`, poID, changedBy); err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

func (r *Repository) GetStatusHistory(ctx context.Context, prID uuid.UUID) ([]StatusHistory, error) {
	rows, err := r.db.Query(ctx, `
		SELECT sh.id, sh.entity_type, sh.entity_id, sh.old_status, sh.new_status,
		       sh.changed_by, COALESCE(u.full_name,''), sh.reason, sh.created_at
		FROM status_history sh
		LEFT JOIN users u ON u.id = sh.changed_by
		WHERE sh.entity_type = 'purchase_requisition' AND sh.entity_id = $1
		ORDER BY sh.created_at ASC
	`, prID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	history := make([]StatusHistory, 0)
	for rows.Next() {
		var h StatusHistory
		if err := rows.Scan(&h.ID, &h.EntityType, &h.EntityID, &h.OldStatus, &h.NewStatus,
			&h.ChangedBy, &h.ChangedByName, &h.Reason, &h.CreatedAt); err != nil {
			return nil, err
		}
		history = append(history, h)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return history, nil
}
