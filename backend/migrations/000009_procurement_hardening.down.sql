DROP INDEX IF EXISTS uq_stock_balances_item_wh;
DROP INDEX IF EXISTS uq_stock_batches_item_wh_batch;
DROP INDEX IF EXISTS idx_grn_receipt_date;
DROP INDEX IF EXISTS idx_purchase_orders_created_at;
DROP INDEX IF EXISTS idx_grn_items_po_item;
DROP INDEX IF EXISTS idx_grn_items_grn;
ALTER TABLE purchase_order_items DROP CONSTRAINT IF EXISTS chk_poi_received_lte_quantity;
ALTER TABLE purchase_orders DROP CONSTRAINT IF EXISTS chk_purchase_orders_status;
