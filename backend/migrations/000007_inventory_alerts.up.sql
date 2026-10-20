CREATE TABLE inventory_alerts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    alert_key VARCHAR(200) UNIQUE NOT NULL,
    alert_type VARCHAR(30) NOT NULL CHECK (alert_type IN ('low_stock', 'near_expiry', 'expired')),
    item_id UUID NOT NULL REFERENCES items(id),
    warehouse_id UUID NOT NULL REFERENCES warehouses(id),
    batch_id UUID REFERENCES stock_batches(id),
    message TEXT NOT NULL,
    active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    resolved_at TIMESTAMPTZ,
    created_by UUID REFERENCES users(id),
    updated_by UUID REFERENCES users(id)
);

CREATE INDEX idx_inventory_alerts_active ON inventory_alerts(active, alert_type, updated_at DESC);