package report

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct{ db *pgxpool.Pool }

func NewRepository(db *pgxpool.Pool) *Repository { return &Repository{db: db} }

type Period struct {
	From time.Time `json:"from"`
	To   time.Time `json:"to"`
}

type Summary struct {
	Period
	SalesRevenue            float64 `json:"sales_revenue"`
	DeliveryCount           int64   `json:"delivery_count"`
	UnitsDelivered          float64 `json:"units_delivered"`
	PurchaseCommitments     float64 `json:"purchase_commitments"`
	PendingPurchaseRequests int64   `json:"pending_purchase_requests"`
	PendingMaterialRequests int64   `json:"pending_material_requests"`
	LowStockItems           int64   `json:"low_stock_items"`
	NearExpiryBatches       int64   `json:"near_expiry_batches"`
	ExpiredBatches          int64   `json:"expired_batches"`
}

type SalesRow struct {
	Key            string  `json:"key"`
	Name           string  `json:"name"`
	UnitsDelivered float64 `json:"units_delivered"`
	Revenue        float64 `json:"revenue"`
}

type ConsumptionRow struct {
	ItemID     string  `json:"item_id"`
	ItemCode   string  `json:"item_code"`
	ItemName   string  `json:"item_name"`
	Unit       string  `json:"unit_of_measure"`
	Quantity   float64 `json:"quantity_issued"`
	IssueCount int64   `json:"issue_count"`
}

type PurchaseCommitmentRow struct {
	PurchaseOrderID  string  `json:"purchase_order_id"`
	PONumber         string  `json:"po_number"`
	SupplierName     *string `json:"supplier_name"`
	Status           string  `json:"status"`
	OrderedValue     float64 `json:"ordered_value"`
	ReceivedValue    float64 `json:"received_value"`
	OutstandingValue float64 `json:"outstanding_value"`
}

func (r *Repository) Summary(ctx context.Context, period Period) (Summary, error) {
	result := Summary{Period: period}
	err := r.db.QueryRow(ctx, `SELECT
		COALESCE((SELECT SUM(dni.quantity_delivered*soi.unit_price) FROM delivery_note_items dni JOIN delivery_notes dn ON dn.id=dni.delivery_note_id JOIN sales_order_items soi ON soi.id=dni.sales_order_item_id WHERE dn.deleted_at IS NULL AND dn.delivery_date::date BETWEEN $1::date AND $2::date),0),
		(SELECT COUNT(*) FROM delivery_notes dn WHERE dn.deleted_at IS NULL AND dn.delivery_date::date BETWEEN $1::date AND $2::date),
		COALESCE((SELECT SUM(dni.quantity_delivered) FROM delivery_note_items dni JOIN delivery_notes dn ON dn.id=dni.delivery_note_id WHERE dn.deleted_at IS NULL AND dn.delivery_date::date BETWEEN $1::date AND $2::date),0),
		COALESCE((SELECT SUM((poi.quantity-poi.received_quantity)*COALESCE(poi.unit_price,0)) FROM purchase_order_items poi JOIN purchase_orders po ON po.id=poi.purchase_order_id WHERE po.deleted_at IS NULL AND po.status IN ('issued','partially_received')),0),
		(SELECT COUNT(*) FROM purchase_requisitions WHERE deleted_at IS NULL AND status IN ('pending_pm','pending_finance','suspended')),
		(SELECT COUNT(*) FROM material_requisitions WHERE deleted_at IS NULL AND status IN ('pending_pm')),
		(SELECT COUNT(*) FROM stock_balances sb JOIN items i ON i.id=sb.item_id WHERE i.is_active=TRUE AND COALESCE(i.reorder_level,i.min_stock_level)>0 AND sb.available_qty<=COALESCE(i.reorder_level,i.min_stock_level)),
		(SELECT COUNT(*) FROM stock_batches WHERE quantity>0 AND expiry_date BETWEEN CURRENT_DATE AND CURRENT_DATE+30),
		(SELECT COUNT(*) FROM stock_batches WHERE quantity>0 AND expiry_date<CURRENT_DATE)`, period.From, period.To).Scan(
		&result.SalesRevenue, &result.DeliveryCount, &result.UnitsDelivered, &result.PurchaseCommitments,
		&result.PendingPurchaseRequests, &result.PendingMaterialRequests, &result.LowStockItems, &result.NearExpiryBatches, &result.ExpiredBatches)
	return result, err
}

func (r *Repository) Sales(ctx context.Context, period Period, groupBy string) ([]SalesRow, error) {
	key, name := "i.id::text", "i.name"
	if groupBy == "customer" {
		key, name = "c.id::text", "c.name"
	}
	query := fmt.Sprintf(`SELECT %s,%s,SUM(dni.quantity_delivered),SUM(dni.quantity_delivered*soi.unit_price)
		FROM delivery_note_items dni JOIN delivery_notes dn ON dn.id=dni.delivery_note_id
		JOIN sales_order_items soi ON soi.id=dni.sales_order_item_id JOIN items i ON i.id=dni.item_id
		JOIN sales_orders so ON so.id=dn.sales_order_id JOIN customers c ON c.id=so.customer_id
		WHERE dn.deleted_at IS NULL AND dn.delivery_date::date BETWEEN $1::date AND $2::date
		GROUP BY %s ORDER BY SUM(dni.quantity_delivered*soi.unit_price) DESC`, key, name, key)
	rows, err := r.db.Query(ctx, query, period.From, period.To)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	list := make([]SalesRow, 0)
	for rows.Next() {
		var item SalesRow
		if err = rows.Scan(&item.Key, &item.Name, &item.UnitsDelivered, &item.Revenue); err != nil {
			return nil, err
		}
		list = append(list, item)
	}
	return list, rows.Err()
}

func (r *Repository) MaterialConsumption(ctx context.Context, period Period) ([]ConsumptionRow, error) {
	rows, err := r.db.Query(ctx, `SELECT i.id::text,i.code,i.name,i.unit_of_measure,SUM(mii.quantity),COUNT(DISTINCT mi.id)
		FROM material_issue_items mii JOIN material_issues mi ON mi.id=mii.issue_id JOIN items i ON i.id=mii.item_id
		WHERE mi.issued_date::date BETWEEN $1::date AND $2::date GROUP BY i.id,i.code,i.name,i.unit_of_measure
		ORDER BY SUM(mii.quantity) DESC`, period.From, period.To)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	list := make([]ConsumptionRow, 0)
	for rows.Next() {
		var item ConsumptionRow
		if err = rows.Scan(&item.ItemID, &item.ItemCode, &item.ItemName, &item.Unit, &item.Quantity, &item.IssueCount); err != nil {
			return nil, err
		}
		list = append(list, item)
	}
	return list, rows.Err()
}

func (r *Repository) PurchaseCommitments(ctx context.Context) ([]PurchaseCommitmentRow, error) {
	rows, err := r.db.Query(ctx, `SELECT po.id::text,po.po_number,s.name,po.status,
		COALESCE(SUM(poi.quantity*COALESCE(poi.unit_price,0)),0),
		COALESCE(SUM(poi.received_quantity*COALESCE(poi.unit_price,0)),0),
		COALESCE(SUM((poi.quantity-poi.received_quantity)*COALESCE(poi.unit_price,0)),0)
		FROM purchase_orders po LEFT JOIN suppliers s ON s.id=po.supplier_id JOIN purchase_order_items poi ON poi.purchase_order_id=po.id
		WHERE po.deleted_at IS NULL AND po.status IN ('issued','partially_received')
		GROUP BY po.id,po.po_number,s.name,po.status ORDER BY po.order_date DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	list := make([]PurchaseCommitmentRow, 0)
	for rows.Next() {
		var item PurchaseCommitmentRow
		if err = rows.Scan(&item.PurchaseOrderID, &item.PONumber, &item.SupplierName, &item.Status, &item.OrderedValue, &item.ReceivedValue, &item.OutstandingValue); err != nil {
			return nil, err
		}
		list = append(list, item)
	}
	return list, rows.Err()
}
