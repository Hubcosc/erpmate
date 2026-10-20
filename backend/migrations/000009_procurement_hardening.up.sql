-- Procurement hardening (renumber if 000007 is already taken).
-- Do NOT edit 000006 if it has already been applied.

-- Allowed PO states.
ALTER TABLE purchase_orders
    ADD CONSTRAINT chk_purchase_orders_status
    CHECK (status IN ('draft','issued','partially_received','received','cancelled'));

-- DB-level safety net against over-receipt.
ALTER TABLE purchase_order_items
    ADD CONSTRAINT chk_poi_received_lte_quantity CHECK (received_quantity <= quantity);

-- Missing indexes for joins / lookups used by the module.
CREATE INDEX IF NOT EXISTS idx_grn_items_grn ON goods_receipt_note_items(goods_receipt_note_id);
CREATE INDEX IF NOT EXISTS idx_grn_items_po_item ON goods_receipt_note_items(purchase_order_item_id);
CREATE INDEX IF NOT EXISTS idx_purchase_orders_created_at ON purchase_orders(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_grn_receipt_date ON goods_receipt_notes(receipt_date DESC);

-- The receipt code relies on these ON CONFLICT targets. They are created here
-- only if your earlier inventory migrations did not already define them.
CREATE UNIQUE INDEX IF NOT EXISTS uq_stock_batches_item_wh_batch
    ON stock_batches(item_id, warehouse_id, batch_number);
CREATE UNIQUE INDEX IF NOT EXISTS uq_stock_balances_item_wh
    ON stock_balances(item_id, warehouse_id);
