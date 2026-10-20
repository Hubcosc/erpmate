package inventory

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Alert listing/summary queries live in repository.go (ListAlerts, AlertSummary).
// This file only contains the background worker that creates and resolves alerts.

type alertCandidate struct {
	key, alertType, message, warehouseType string
	itemID, warehouseID                    uuid.UUID
	batchID                                *uuid.UUID
}

type existingAlert struct {
	id        uuid.UUID
	alertType string
	message   string
	active    bool
}

// querier is satisfied by both *pgxpool.Pool and pgx.Tx.
type querier interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

// RunAlertWorker starts the background scan and returns immediately.
// Existing signature kept; use RunAlertWorkerContext to stop it on shutdown.
func RunAlertWorker(db *pgxpool.Pool, interval time.Duration, expiryDays int) {
	RunAlertWorkerContext(context.Background(), db, interval, expiryDays)
}

// RunAlertWorkerContext scans immediately (in the background, so startup is not blocked),
// then on every interval, until ctx is cancelled.
func RunAlertWorkerContext(ctx context.Context, db *pgxpool.Pool, interval time.Duration, expiryDays int) {
	if interval <= 0 {
		interval = 15 * time.Minute
	}
	if expiryDays <= 0 {
		expiryDays = defaultExpiryWarningDays
	}
	go func() {
		runAlertScan(ctx, db, expiryDays)
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				runAlertScan(ctx, db, expiryDays)
			}
		}
	}()
}

// runAlertScan never panics: one bad scan must not kill the worker goroutine.
func runAlertScan(parent context.Context, db *pgxpool.Pool, expiryDays int) {
	defer func() {
		if rec := recover(); rec != nil {
			log.Printf("inventory alert scan panicked: %v", rec)
		}
	}()
	ctx, cancel := context.WithTimeout(parent, 2*time.Minute)
	defer cancel()
	if err := evaluateInventoryAlerts(ctx, db, expiryDays); err != nil && parent.Err() == nil {
		log.Printf("inventory alert scan failed: %v", err)
	}
}

// evaluateInventoryAlerts runs one scan inside a single transaction, under an advisory lock so
// several app instances never scan at once. Only rows that changed are written, and all writes
// are sent in one round trip.
func evaluateInventoryAlerts(ctx context.Context, db *pgxpool.Pool, expiryDays int) error {
	tx, err := db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if _, err = tx.Exec(ctx, `SELECT pg_advisory_xact_lock(728194063)`); err != nil {
		return err
	}

	candidates, err := findAlertCandidates(ctx, tx, expiryDays)
	if err != nil {
		return err
	}

	// De-duplicate by key and remember every key that is still a problem.
	activeKeys := make([]string, 0, len(candidates))
	unique := make([]alertCandidate, 0, len(candidates))
	seen := make(map[string]bool, len(candidates))
	for _, c := range candidates {
		if seen[c.key] {
			continue
		}
		seen[c.key] = true
		activeKeys = append(activeKeys, c.key)
		unique = append(unique, c)
	}

	existing, err := loadExistingAlerts(ctx, tx, activeKeys)
	if err != nil {
		return err
	}

	batch := &pgx.Batch{}
	for _, c := range unique {
		ex, found := existing[c.key]
		switch {
		case !found:
			alertID := uuid.New()
			batch.Queue(`INSERT INTO inventory_alerts(id,alert_key,alert_type,item_id,warehouse_id,batch_id,message)
				VALUES($1,$2,$3,$4,$5,$6,$7)`,
				alertID, c.key, c.alertType, c.itemID, c.warehouseID, c.batchID, c.message)
			queueAlertNotifications(batch, alertID, c)
		case !ex.active:
			// Problem came back after being resolved: reactivate and notify again.
			batch.Queue(`UPDATE inventory_alerts
				SET alert_type=$1,message=$2,active=TRUE,resolved_at=NULL,updated_at=NOW() WHERE id=$3`,
				c.alertType, c.message, ex.id)
			queueAlertNotifications(batch, ex.id, c)
		case ex.message != c.message || ex.alertType != c.alertType:
			// Still active, details changed (e.g. quantity dropped): refresh quietly, no new notification.
			batch.Queue(`UPDATE inventory_alerts SET alert_type=$1,message=$2,updated_at=NOW() WHERE id=$3`,
				c.alertType, c.message, ex.id)
		}
	}

	// Anything active that is no longer a candidate has been fixed (or its item/warehouse is gone).
	batch.Queue(`UPDATE inventory_alerts SET active=FALSE,resolved_at=NOW(),updated_at=NOW()
		WHERE active=TRUE AND NOT (alert_key=ANY($1::varchar[]))`, activeKeys)

	results := tx.SendBatch(ctx, batch)
	for i := 0; i < batch.Len(); i++ {
		if _, err = results.Exec(); err != nil {
			results.Close()
			return fmt.Errorf("inventory alert batch statement %d: %w", i, err)
		}
	}
	if err = results.Close(); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func loadExistingAlerts(ctx context.Context, q querier, keys []string) (map[string]existingAlert, error) {
	out := make(map[string]existingAlert, len(keys))
	if len(keys) == 0 {
		return out, nil
	}
	rows, err := q.Query(ctx, `SELECT alert_key,id,alert_type,message,active
		FROM inventory_alerts WHERE alert_key=ANY($1::varchar[])`, keys)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var key string
		var e existingAlert
		if err = rows.Scan(&key, &e.id, &e.alertType, &e.message, &e.active); err != nil {
			return nil, err
		}
		out[key] = e
	}
	return out, rows.Err()
}

func findAlertCandidates(ctx context.Context, q querier, expiryDays int) ([]alertCandidate, error) {
	result := make([]alertCandidate, 0)

	low, err := queryLowStockCandidates(ctx, q)
	if err != nil {
		return nil, err
	}
	result = append(result, low...)

	expiry, err := queryExpiryCandidates(ctx, q, expiryDays)
	if err != nil {
		return nil, err
	}
	return append(result, expiry...), nil
}

// Low stock uses the same rule as the stock list: threshold = reorder_level or min_stock_level,
// threshold must be > 0, and available <= threshold.
func queryLowStockCandidates(ctx context.Context, q querier) ([]alertCandidate, error) {
	rows, err := q.Query(ctx, `SELECT 'low_stock:'||sb.item_id::text||':'||sb.warehouse_id::text,'low_stock',sb.item_id,sb.warehouse_id,w.warehouse_type,
		'Low stock: '||i.name||' ('||sb.available_qty::float8::text||' '||i.unit_of_measure||' available; threshold '||COALESCE(i.reorder_level,i.min_stock_level,0)::float8::text||')'
		FROM stock_balances sb JOIN items i ON i.id=sb.item_id JOIN warehouses w ON w.id=sb.warehouse_id
		WHERE i.deleted_at IS NULL AND i.is_active=TRUE AND w.is_active=TRUE AND w.deleted_at IS NULL
		AND COALESCE(i.reorder_level,i.min_stock_level,0)>0 AND sb.available_qty<=COALESCE(i.reorder_level,i.min_stock_level,0)`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := make([]alertCandidate, 0)
	for rows.Next() {
		var c alertCandidate
		if err = rows.Scan(&c.key, &c.alertType, &c.itemID, &c.warehouseID, &c.warehouseType, &c.message); err != nil {
			return nil, err
		}
		list = append(list, c)
	}
	return list, rows.Err()
}

// A batch moves from near_expiry to expired under a different key, so the old alert resolves
// and a fresh, more severe "expired" alert is raised and notified.
func queryExpiryCandidates(ctx context.Context, q querier, expiryDays int) ([]alertCandidate, error) {
	rows, err := q.Query(ctx, `SELECT 'expiry:'||CASE WHEN b.expiry_date<CURRENT_DATE THEN 'expired:' ELSE 'near_expiry:' END||b.id::text,
		CASE WHEN b.expiry_date<CURRENT_DATE THEN 'expired' ELSE 'near_expiry' END,b.item_id,b.warehouse_id,w.warehouse_type,b.id,
		i.name||' batch '||b.batch_number||' expires on '||b.expiry_date::text
		FROM stock_batches b JOIN items i ON i.id=b.item_id JOIN warehouses w ON w.id=b.warehouse_id
		WHERE b.quantity>0 AND b.expiry_date IS NOT NULL
		AND i.deleted_at IS NULL AND i.is_active=TRUE AND w.is_active=TRUE AND w.deleted_at IS NULL
		AND b.expiry_date<=CURRENT_DATE+$1::integer`, expiryDays)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := make([]alertCandidate, 0)
	for rows.Next() {
		var c alertCandidate
		var batchID uuid.UUID
		if err = rows.Scan(&c.key, &c.alertType, &c.itemID, &c.warehouseID, &c.warehouseType, &batchID, &c.message); err != nil {
			return nil, err
		}
		c.batchID = &batchID
		list = append(list, c)
	}
	return list, rows.Err()
}

// queueAlertNotifications notifies the responsible store keeper role plus admins.
func queueAlertNotifications(batch *pgx.Batch, alertID uuid.UUID, c alertCandidate) {
	role := "store_keeper"
	if c.warehouseType == "finished_goods" {
		role = "production_store_keeper"
	}
	batch.Queue(`INSERT INTO notifications(user_id,title,message,entity_type,entity_id)
		SELECT u.id,$1,$2,'inventory_alert',$3 FROM users u JOIN roles r ON r.id=u.role_id
		WHERE u.is_active=TRUE AND u.deleted_at IS NULL AND r.name IN ($4,'admin')`,
		inventoryAlertTitle(c.alertType), c.message, alertID, role)
}

func inventoryAlertTitle(alertType string) string {
	switch alertType {
	case "low_stock":
		return "Low stock alert"
	case "expired":
		return "Expired stock alert"
	default:
		return "Near-expiry stock alert"
	}
}
