package inventory

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type StockRepository struct {
	db         *pgxpool.Pool
	expiryDays int
}

func NewStockRepository(db *pgxpool.Pool) *StockRepository {
	return &StockRepository{db: db, expiryDays: defaultExpiryWarningDays}
}

// SetExpiryWarningDays makes batch expiry_status agree with the alert worker's window.
// Call it with the same value you pass to RunAlertWorker.
func (r *StockRepository) SetExpiryWarningDays(days int) {
	if days > 0 {
		r.expiryDays = days
	}
}

// ====================== SHARED SQL ======================

// Stock status rules (aliases: sb = stock_balances, i = items). One definition, used by
// the stock list, summary and valuation so they can never disagree.
//
//	threshold  = reorder_level, falling back to min_stock_level
//	low        = threshold > 0 AND available <= threshold   (threshold 0 means "no alert level")
//	status     = out_of_stock | low_stock | in_stock
const (
	thresholdExpr = `COALESCE(i.reorder_level, i.min_stock_level, 0)`
	lowExpr       = `(` + thresholdExpr + ` > 0 AND sb.available_qty <= ` + thresholdExpr + `)`
	statusExpr    = `CASE WHEN sb.available_qty <= 0 THEN 'out_of_stock' WHEN ` + lowExpr + ` THEN 'low_stock' ELSE 'in_stock' END`
)

// severityExpr (aliases: a = inventory_alerts, b = stock_batches, sb = stock_balances).
const severityExpr = `CASE
	WHEN a.alert_type = 'expired' THEN 'critical'
	WHEN a.alert_type = 'low_stock' AND COALESCE(sb.available_qty, 0) <= 0 THEN 'critical'
	WHEN a.alert_type = 'low_stock' THEN 'warning'
	WHEN b.expiry_date IS NOT NULL AND b.expiry_date - CURRENT_DATE <= 7 THEN 'warning'
	ELSE 'info' END`

func mapErr(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	return err
}

// whereBuilder builds WHERE clauses with positional args. Use "?" as the placeholder.
type whereBuilder struct {
	clauses []string
	args    []interface{}
}

func (w *whereBuilder) add(clause string, args ...interface{}) {
	for _, a := range args {
		w.args = append(w.args, a)
		clause = strings.Replace(clause, "?", fmt.Sprintf("$%d", len(w.args)), 1)
	}
	w.clauses = append(w.clauses, clause)
}

func (w *whereBuilder) sql() string {
	if len(w.clauses) == 0 {
		return ""
	}
	return "WHERE " + strings.Join(w.clauses, " AND ")
}

func likePattern(s string) string {
	r := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)
	return "%" + r.Replace(s) + "%"
}

// buildOrder maps a user-supplied sort key to a whitelisted column. Unknown keys fall back.
func buildOrder(sorts map[string]string, key, order, fallback, tiebreak string) string {
	col, ok := sorts[key]
	if !ok {
		return fallback
	}
	dir := "ASC"
	if strings.EqualFold(order, "desc") {
		dir = "DESC"
	}
	return col + " " + dir + ", " + tiebreak
}

func collect[T any](rows pgx.Rows, scan func(pgx.Row) (T, error)) ([]T, error) {
	list, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (T, error) { return scan(row) })
	if err == nil && list == nil {
		list = []T{} // JSON must be [] not null
	}
	return list, err
}

// listPage runs COUNT + paged SELECT with identical filters. Result slice is never nil.
func listPage[T any](
	ctx context.Context, db *pgxpool.Pool,
	with, cols, from string, w *whereBuilder, order string, limit, offset int,
	scan func(pgx.Row) (T, error),
) ([]T, int64, error) {
	var total int64
	countSQL := with + " SELECT COUNT(*) FROM " + from + " " + w.sql()
	if err := db.QueryRow(ctx, countSQL, w.args...).Scan(&total); err != nil {
		return nil, 0, mapErr(err)
	}

	n := len(w.args)
	query := fmt.Sprintf("%s SELECT %s FROM %s %s ORDER BY %s LIMIT $%d OFFSET $%d",
		with, cols, from, w.sql(), order, n+1, n+2)
	args := append(append([]interface{}{}, w.args...), limit, offset)

	rows, err := db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, mapErr(err)
	}
	list, err := collect(rows, scan)
	if err != nil {
		return nil, 0, mapErr(err)
	}
	return list, total, nil
}

// ====================== STOCK BALANCES ======================

const balanceFrom = `stock_balances sb
	JOIN items i ON i.id = sb.item_id
	JOIN warehouses w ON w.id = sb.warehouse_id
	LEFT JOIN item_categories c ON c.id = i.category_id AND c.deleted_at IS NULL`

const balanceCols = `i.id, i.code, i.name, i.item_type, i.unit_of_measure, c.name,
	w.id, w.code, w.name, w.warehouse_type,
	sb.quantity, sb.reserved_qty, sb.available_qty,
	COALESCE(i.min_stock_level, 0), i.max_stock_level, i.reorder_level,
	` + thresholdExpr + `, ` + lowExpr + `, ` + statusExpr

var stockSorts = map[string]string{
	"name":      "i.name",
	"code":      "i.code",
	"warehouse": "w.name",
	"quantity":  "sb.quantity",
	"reserved":  "sb.reserved_qty",
	"available": "sb.available_qty",
}

func scanBalance(row pgx.Row) (StockBalance, error) {
	var b StockBalance
	err := row.Scan(
		&b.ItemID, &b.ItemCode, &b.ItemName, &b.ItemType, &b.UnitOfMeasure, &b.CategoryName,
		&b.WarehouseID, &b.WarehouseCode, &b.WarehouseName, &b.WarehouseType,
		&b.Quantity, &b.ReservedQuantity, &b.AvailableQuantity,
		&b.MinStockLevel, &b.MaxStockLevel, &b.ReorderLevel,
		&b.Threshold, &b.LowStock, &b.StockStatus,
	)
	return b, err
}

// balanceWhere never lists stock for inactive/soft-deleted items or warehouses.
func balanceWhere(f StockFilter, withStatus bool) *whereBuilder {
	w := &whereBuilder{}
	w.add("i.deleted_at IS NULL AND i.is_active = TRUE AND w.is_active = TRUE AND w.deleted_at IS NULL")
	if f.ItemID != nil {
		w.add("i.id = ?", *f.ItemID)
	}
	if f.Search != "" {
		p := likePattern(f.Search)
		w.add("(i.code ILIKE ? OR i.name ILIKE ?)", p, p)
	}
	if f.ItemType != "" {
		w.add("i.item_type = ?", f.ItemType)
	}
	if f.WarehouseType != "" {
		w.add("w.warehouse_type = ?", f.WarehouseType)
	}
	if f.WarehouseID != nil {
		w.add("w.id = ?", *f.WarehouseID)
	}
	if f.CategoryID != nil {
		w.add("i.category_id = ?", *f.CategoryID)
	}
	if withStatus && len(f.Statuses) > 0 {
		w.add("("+statusExpr+") = ANY(?)", f.Statuses)
	}
	return w
}

func (r *StockRepository) ListBalances(ctx context.Context, f StockFilter) ([]StockBalance, int64, error) {
	const def = "i.name, w.name, i.id, w.id"
	order := buildOrder(stockSorts, f.Sort, f.Order, def, def)
	return listPage(ctx, r.db, "", balanceCols, balanceFrom, balanceWhere(f, true), order, f.Limit, f.offset(), scanBalance)
}

// StockSummary ignores the status filter so tab counts stay stable while a tab is selected.
func (r *StockRepository) StockSummary(ctx context.Context, f StockFilter) (*StockSummary, error) {
	w := balanceWhere(f, false)
	query := `SELECT COUNT(*), COUNT(DISTINCT sb.item_id),
		COUNT(*) FILTER (WHERE ` + statusExpr + ` = 'in_stock'),
		COUNT(*) FILTER (WHERE ` + statusExpr + ` = 'low_stock'),
		COUNT(*) FILTER (WHERE ` + statusExpr + ` = 'out_of_stock')
		FROM ` + balanceFrom + ` ` + w.sql()

	var s StockSummary
	if err := r.db.QueryRow(ctx, query, w.args...).Scan(&s.TotalLines, &s.TotalItems, &s.InStock, &s.LowStock, &s.OutOfStock); err != nil {
		return nil, mapErr(err)
	}
	return &s, nil
}

// GetItemInfo returns the item header for the detail view (404 if missing/inactive/deleted).
func (r *StockRepository) GetItemInfo(ctx context.Context, id uuid.UUID) (*ItemStock, error) {
	var s ItemStock
	err := r.db.QueryRow(ctx, `
		SELECT i.id, i.code, i.name, i.item_type, i.unit_of_measure, c.name,
		       COALESCE(i.min_stock_level, 0), i.max_stock_level, i.reorder_level,
		       i.is_batch_tracked, i.is_expiry_tracked
		FROM items i
		LEFT JOIN item_categories c ON c.id = i.category_id AND c.deleted_at IS NULL
		WHERE i.id = $1 AND i.deleted_at IS NULL AND i.is_active = TRUE
	`, id).Scan(&s.ItemID, &s.ItemCode, &s.ItemName, &s.ItemType, &s.UnitOfMeasure, &s.CategoryName,
		&s.MinStockLevel, &s.MaxStockLevel, &s.ReorderLevel, &s.IsBatchTracked, &s.IsExpiryTracked)
	if err != nil {
		return nil, mapErr(err)
	}
	s.Warehouses = []StockBalance{}
	return &s, nil
}

// ====================== BATCHES ======================

func scanBatch(row pgx.Row) (StockBatch, error) {
	var b StockBatch
	err := row.Scan(
		&b.ID, &b.ItemID, &b.WarehouseID, &b.WarehouseCode, &b.WarehouseName, &b.BatchNumber,
		&b.Quantity, &b.AvailableQuantity, &b.ManufactureDate, &b.ExpiryDate, &b.DaysUntilExpiry,
		&b.ExpiryStatus, &b.UnitCost, &b.InventoryValue, &b.CreatedAt,
	)
	return b, err
}

// ListBatches returns batches in FEFO order. AvailableQuantity allocates the item's available
// balance across batches (earliest expiry first), so the per-batch values never add up to more
// than the warehouse's real available stock. Expired/near-expiry batches are flagged, not hidden.
func (r *StockRepository) ListBatches(ctx context.Context, f BatchFilter) ([]StockBatch, error) {
	w := &whereBuilder{}
	w.add("b.item_id = ?", f.ItemID)
	w.add("b.quantity > 0 AND i.deleted_at IS NULL AND w.is_active = TRUE AND w.deleted_at IS NULL")
	if f.WarehouseType != "" {
		w.add("w.warehouse_type = ?", f.WarehouseType)
	}
	if f.WarehouseID != nil {
		w.add("w.id = ?", *f.WarehouseID)
	}

	inner := fmt.Sprintf(`SELECT b.id, b.item_id, b.warehouse_id, w.code AS warehouse_code, w.name AS warehouse_name,
			b.batch_number, b.quantity,
			GREATEST(0, LEAST(b.quantity,
				COALESCE(sb.available_qty, b.quantity) - COALESCE(SUM(b.quantity) OVER (
					PARTITION BY b.warehouse_id
					ORDER BY b.expiry_date NULLS LAST, b.created_at, b.id
					ROWS BETWEEN UNBOUNDED PRECEDING AND 1 PRECEDING), 0))) AS available_quantity,
			b.manufacture_date::text AS manufacture_date,
			b.expiry_date::text AS expiry_date,
			b.expiry_date AS expiry_sort,
			(b.expiry_date - CURRENT_DATE)::integer AS days_until_expiry,
			CASE WHEN b.expiry_date IS NULL THEN 'no_expiry'
			     WHEN b.expiry_date < CURRENT_DATE THEN 'expired'
			     WHEN b.expiry_date <= CURRENT_DATE + %d THEN 'near_expiry'
			     ELSE 'ok' END AS expiry_status,
			b.unit_cost, ROUND(b.quantity * b.unit_cost, 2) AS inventory_value, b.created_at
		FROM stock_batches b
		JOIN items i ON i.id = b.item_id
		JOIN warehouses w ON w.id = b.warehouse_id
		LEFT JOIN stock_balances sb ON sb.item_id = b.item_id AND sb.warehouse_id = b.warehouse_id
		%s`, r.expiryDays, w.sql())

	outerWhere := ""
	if len(f.ExpiryStatuses) > 0 {
		w.args = append(w.args, f.ExpiryStatuses)
		outerWhere = fmt.Sprintf(" WHERE expiry_status = ANY($%d)", len(w.args))
	}

	query := `SELECT id, item_id, warehouse_id, warehouse_code, warehouse_name, batch_number, quantity,
			available_quantity, manufacture_date, expiry_date, days_until_expiry, expiry_status,
			unit_cost, inventory_value, created_at
		FROM (` + inner + `) t` + outerWhere + ` ORDER BY expiry_sort NULLS LAST, created_at, id`

	rows, err := r.db.Query(ctx, query, w.args...)
	if err != nil {
		return nil, mapErr(err)
	}
	list, err := collect(rows, scanBatch)
	return list, mapErr(err)
}

// ====================== MOVEMENTS ======================

const movementFrom = `stock_movements sm
	JOIN items i ON i.id = sm.item_id
	JOIN warehouses w ON w.id = sm.warehouse_id
	LEFT JOIN stock_batches b ON b.id = sm.batch_id
	LEFT JOIN users u ON u.id = sm.performed_by`

const movementCols = `sm.id, sm.movement_type, sm.reference_type, sm.reference_id,
	sm.item_id, i.code, i.name, i.unit_of_measure, sm.warehouse_id, w.name,
	sm.batch_id, b.batch_number, sm.quantity, sm.unit_cost, ROUND(sm.quantity * sm.unit_cost, 2),
	sm.notes, sm.performed_by, u.full_name, sm.movement_date`

func scanMovement(row pgx.Row) (StockMovement, error) {
	var m StockMovement
	err := row.Scan(
		&m.ID, &m.MovementType, &m.ReferenceType, &m.ReferenceID,
		&m.ItemID, &m.ItemCode, &m.ItemName, &m.UnitOfMeasure, &m.WarehouseID, &m.WarehouseName,
		&m.BatchID, &m.BatchNumber, &m.Quantity, &m.UnitCost, &m.TotalValue,
		&m.Notes, &m.PerformedBy, &m.PerformedByName, &m.MovementDate,
	)
	return m, err
}

// ListMovements keeps history for soft-deleted items/warehouses on purpose: an audit trail must not vanish.
func (r *StockRepository) ListMovements(ctx context.Context, f MovementFilter) ([]StockMovement, int64, error) {
	w := &whereBuilder{}
	w.add("TRUE")
	if f.ItemID != nil {
		w.add("sm.item_id = ?", *f.ItemID)
	}
	if f.WarehouseID != nil {
		w.add("sm.warehouse_id = ?", *f.WarehouseID)
	}
	if len(f.MovementTypes) > 0 {
		w.add("sm.movement_type = ANY(?)", f.MovementTypes)
	}
	if f.ReferenceType != "" {
		w.add("sm.reference_type = ?", f.ReferenceType)
	}
	if f.Search != "" {
		p := likePattern(f.Search)
		w.add("(i.code ILIKE ? OR i.name ILIKE ? OR b.batch_number ILIKE ? OR sm.notes ILIKE ?)", p, p, p, p)
	}
	if f.DateFrom != nil {
		w.add("sm.movement_date >= ?", *f.DateFrom)
	}
	if f.DateTo != nil {
		w.add("sm.movement_date < ?", f.DateTo.AddDate(0, 0, 1))
	}
	return listPage(ctx, r.db, "", movementCols, movementFrom, w,
		"sm.movement_date DESC, sm.id DESC", f.Limit, f.offset(), scanMovement)
}

func (r *StockRepository) MovementTypes(ctx context.Context) ([]string, error) {
	rows, err := r.db.Query(ctx, `SELECT DISTINCT movement_type FROM stock_movements ORDER BY movement_type`)
	if err != nil {
		return nil, mapErr(err)
	}
	list, err := collect(rows, func(row pgx.Row) (string, error) {
		var s string
		err := row.Scan(&s)
		return s, err
	})
	return list, mapErr(err)
}

// ====================== VALUATION ======================

const valuationFrom = `valuation v`

const valuationWith = `WITH batch_cost AS (
		SELECT item_id, warehouse_id, SUM(quantity * unit_cost) / NULLIF(SUM(quantity), 0) AS average_cost
		FROM stock_batches WHERE quantity > 0 AND unit_cost IS NOT NULL GROUP BY item_id, warehouse_id
	), movement_cost AS (
		SELECT item_id, warehouse_id, SUM(quantity * unit_cost) / NULLIF(SUM(quantity), 0) AS average_cost
		FROM stock_movements WHERE quantity > 0 AND unit_cost IS NOT NULL
		AND movement_type IN ('receipt', 'production_receipt', 'opening_balance') GROUP BY item_id, warehouse_id
	), valuation AS (
		SELECT i.id AS item_id, i.code AS item_code, i.name AS item_name, i.item_type, i.unit_of_measure,
		       w.id AS warehouse_id, w.name AS warehouse_name, w.warehouse_type,
		       sb.quantity, sb.reserved_qty, sb.available_qty, i.reorder_level,
		       ` + lowExpr + ` AS low_stock,
		       ` + statusExpr + ` AS stock_status,
		       ROUND(COALESCE(bc.average_cost, mc.average_cost, 0), 2) AS average_unit_cost,
		       ROUND(sb.quantity * COALESCE(bc.average_cost, mc.average_cost, 0), 2) AS inventory_value,
		       CASE WHEN bc.average_cost IS NOT NULL THEN 'stock_batches'
		            WHEN mc.average_cost IS NOT NULL THEN 'inbound_movements'
		            ELSE 'no_cost_recorded' END AS cost_source
		FROM stock_balances sb
		JOIN items i ON i.id = sb.item_id
		JOIN warehouses w ON w.id = sb.warehouse_id
		LEFT JOIN batch_cost bc ON bc.item_id = sb.item_id AND bc.warehouse_id = sb.warehouse_id
		LEFT JOIN movement_cost mc ON mc.item_id = sb.item_id AND mc.warehouse_id = sb.warehouse_id
		WHERE i.deleted_at IS NULL AND i.is_active = TRUE AND w.is_active = TRUE AND w.deleted_at IS NULL
	)`

const valuationCols = `v.item_id, v.item_code, v.item_name, v.item_type, v.unit_of_measure,
	v.warehouse_id, v.warehouse_name, v.warehouse_type,
	v.quantity, v.reserved_qty, v.available_qty, v.reorder_level, v.low_stock, v.stock_status,
	v.average_unit_cost, v.inventory_value, v.cost_source`

var valuationSorts = map[string]string{
	"name":      "v.item_name",
	"code":      "v.item_code",
	"warehouse": "v.warehouse_name",
	"quantity":  "v.quantity",
	"value":     "v.inventory_value",
	"unit_cost": "v.average_unit_cost",
}

func scanValuation(row pgx.Row) (StockValuation, error) {
	var v StockValuation
	err := row.Scan(
		&v.ItemID, &v.ItemCode, &v.ItemName, &v.ItemType, &v.UnitOfMeasure,
		&v.WarehouseID, &v.WarehouseName, &v.WarehouseType,
		&v.Quantity, &v.ReservedQuantity, &v.AvailableQuantity, &v.ReorderLevel, &v.LowStock, &v.StockStatus,
		&v.AverageUnitCost, &v.InventoryValue, &v.CostSource,
	)
	return v, err
}

func valuationWhere(f ValuationFilter) *whereBuilder {
	w := &whereBuilder{}
	w.add("TRUE")
	if f.Search != "" {
		p := likePattern(f.Search)
		w.add("(v.item_code ILIKE ? OR v.item_name ILIKE ?)", p, p)
	}
	if f.ItemType != "" {
		w.add("v.item_type = ?", f.ItemType)
	}
	if f.WarehouseType != "" {
		w.add("v.warehouse_type = ?", f.WarehouseType)
	}
	if f.WarehouseID != nil {
		w.add("v.warehouse_id = ?", *f.WarehouseID)
	}
	if len(f.CostSources) > 0 {
		w.add("v.cost_source = ANY(?)", f.CostSources)
	}
	if f.LowStock != nil {
		w.add("v.low_stock = ?", *f.LowStock)
	}
	return w
}

func (r *StockRepository) ListValuation(ctx context.Context, f ValuationFilter) ([]StockValuation, int64, error) {
	const def = "v.item_name, v.warehouse_name, v.item_id, v.warehouse_id"
	order := buildOrder(valuationSorts, f.Sort, f.Order, def, def)
	return listPage(ctx, r.db, valuationWith, valuationCols, valuationFrom, valuationWhere(f), order, f.Limit, f.offset(), scanValuation)
}

func (r *StockRepository) ValuationSummary(ctx context.Context, f ValuationFilter) (*ValuationSummary, error) {
	w := valuationWhere(f)
	var s ValuationSummary
	err := r.db.QueryRow(ctx, valuationWith+`
		SELECT COALESCE(SUM(v.inventory_value), 0), COUNT(*),
		       COUNT(*) FILTER (WHERE v.cost_source = 'no_cost_recorded')
		FROM `+valuationFrom+` `+w.sql(), w.args...).Scan(&s.TotalValue, &s.TotalLines, &s.LinesWithoutCost)
	if err != nil {
		return nil, mapErr(err)
	}

	rows, err := r.db.Query(ctx, valuationWith+`
		SELECT v.warehouse_type, COUNT(*), COALESCE(SUM(v.inventory_value), 0)
		FROM `+valuationFrom+` `+w.sql()+`
		GROUP BY v.warehouse_type ORDER BY SUM(v.inventory_value) DESC, v.warehouse_type`, w.args...)
	if err != nil {
		return nil, mapErr(err)
	}
	s.ByWarehouseType, err = collect(rows, func(row pgx.Row) (ValuationGroup, error) {
		var g ValuationGroup
		err := row.Scan(&g.WarehouseType, &g.Lines, &g.Value)
		return g, err
	})
	if err != nil {
		return nil, mapErr(err)
	}
	return &s, nil
}

// ====================== ALERTS ======================

const alertFrom = `inventory_alerts a
	JOIN items i ON i.id = a.item_id AND i.deleted_at IS NULL
	JOIN warehouses w ON w.id = a.warehouse_id AND w.deleted_at IS NULL
	LEFT JOIN stock_batches b ON b.id = a.batch_id
	LEFT JOIN stock_balances sb ON sb.item_id = a.item_id AND sb.warehouse_id = a.warehouse_id`

const alertCols = `a.id, a.alert_type, (` + severityExpr + `),
	a.item_id, i.code, i.name, i.unit_of_measure,
	a.warehouse_id, w.code, w.name, b.batch_number,
	CASE WHEN a.alert_type IN ('near_expiry', 'expired') THEN b.quantity ELSE sb.available_qty END,
	CASE WHEN a.alert_type = 'low_stock' THEN COALESCE(i.reorder_level, i.min_stock_level, 0) END,
	b.expiry_date::text, (b.expiry_date - CURRENT_DATE)::integer, a.message, a.created_at`

func scanAlert(row pgx.Row) (InventoryAlert, error) {
	var a InventoryAlert
	err := row.Scan(
		&a.ID, &a.AlertType, &a.Severity,
		&a.ItemID, &a.ItemCode, &a.ItemName, &a.UnitOfMeasure,
		&a.WarehouseID, &a.WarehouseCode, &a.WarehouseName, &a.BatchNumber,
		&a.Quantity, &a.Threshold,
		&a.ExpiryDate, &a.DaysUntilExpiry, &a.Message, &a.CreatedAt,
	)
	return a, err
}

func alertWhere(f AlertFilter, withTypeFilters bool) *whereBuilder {
	w := &whereBuilder{}
	w.add("a.active = TRUE")
	if f.WarehouseID != nil {
		w.add("a.warehouse_id = ?", *f.WarehouseID)
	}
	if f.Search != "" {
		p := likePattern(f.Search)
		w.add("(i.code ILIKE ? OR i.name ILIKE ? OR b.batch_number ILIKE ?)", p, p, p)
	}
	if withTypeFilters {
		if len(f.AlertTypes) > 0 {
			w.add("a.alert_type = ANY(?)", f.AlertTypes)
		}
		if len(f.Severities) > 0 {
			w.add("("+severityExpr+") = ANY(?)", f.Severities)
		}
	}
	return w
}

// ListAlerts returns active alerts, most severe first, then newest.
func (r *StockRepository) ListAlerts(ctx context.Context, f AlertFilter) ([]InventoryAlert, int64, error) {
	order := `CASE (` + severityExpr + `) WHEN 'critical' THEN 0 WHEN 'warning' THEN 1 ELSE 2 END, a.created_at DESC, a.id`
	return listPage(ctx, r.db, "", alertCols, alertFrom, alertWhere(f, true), order, f.Limit, f.offset(), scanAlert)
}

// AlertSummary ignores type/severity filters so tab counts stay stable while a tab is selected.
func (r *StockRepository) AlertSummary(ctx context.Context, f AlertFilter) (*AlertSummary, error) {
	w := alertWhere(f, false)
	query := `SELECT COUNT(*),
		COUNT(*) FILTER (WHERE a.alert_type = 'low_stock'),
		COUNT(*) FILTER (WHERE a.alert_type = 'near_expiry'),
		COUNT(*) FILTER (WHERE a.alert_type = 'expired'),
		COUNT(*) FILTER (WHERE (` + severityExpr + `) = 'critical'),
		COUNT(*) FILTER (WHERE (` + severityExpr + `) = 'warning'),
		COUNT(*) FILTER (WHERE (` + severityExpr + `) = 'info')
		FROM ` + alertFrom + ` ` + w.sql()

	var s AlertSummary
	if err := r.db.QueryRow(ctx, query, w.args...).Scan(
		&s.Total, &s.LowStock, &s.NearExpiry, &s.Expired, &s.Critical, &s.Warning, &s.Info,
	); err != nil {
		return nil, mapErr(err)
	}
	return &s, nil
}
