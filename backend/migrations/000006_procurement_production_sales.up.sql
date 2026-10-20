-- Procurement, production receipts, and sales/delivery workflows.
ALTER TABLE stock_batches ALTER COLUMN unit_cost TYPE NUMERIC(18,2);
ALTER TABLE stock_movements ALTER COLUMN unit_cost TYPE NUMERIC(18,2);

CREATE SEQUENCE IF NOT EXISTS purchase_order_number_seq START 1;
CREATE SEQUENCE IF NOT EXISTS goods_receipt_number_seq START 1;
CREATE SEQUENCE IF NOT EXISTS production_receipt_number_seq START 1;
CREATE SEQUENCE IF NOT EXISTS sales_order_number_seq START 1;
CREATE SEQUENCE IF NOT EXISTS delivery_note_number_seq START 1;

CREATE TABLE purchase_orders (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    po_number VARCHAR(30) UNIQUE NOT NULL,
    purchase_requisition_id UUID NOT NULL UNIQUE REFERENCES purchase_requisitions(id),
    supplier_id UUID REFERENCES suppliers(id),
    status VARCHAR(30) NOT NULL DEFAULT 'draft',
    order_date DATE NOT NULL DEFAULT CURRENT_DATE,
    expected_delivery_date DATE,
    payment_terms TEXT,
    delivery_terms TEXT,
    notes TEXT,
    total_amount NUMERIC(18,2) NOT NULL DEFAULT 0,
    issued_by UUID REFERENCES users(id),
    issued_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by UUID REFERENCES users(id),
    updated_by UUID REFERENCES users(id),
    deleted_at TIMESTAMPTZ
);
CREATE INDEX idx_purchase_orders_status ON purchase_orders(status);
CREATE INDEX idx_purchase_orders_supplier ON purchase_orders(supplier_id);

CREATE TABLE purchase_order_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    purchase_order_id UUID NOT NULL REFERENCES purchase_orders(id) ON DELETE CASCADE,
    purchase_requisition_item_id UUID REFERENCES purchase_requisition_items(id),
    item_id UUID REFERENCES items(id),
    description TEXT,
    quantity NUMERIC(18,4) NOT NULL CHECK (quantity > 0),
    received_quantity NUMERIC(18,4) NOT NULL DEFAULT 0 CHECK (received_quantity >= 0),
    unit_of_measure VARCHAR(20),
    unit_price NUMERIC(18,2),
    line_total NUMERIC(18,2) NOT NULL DEFAULT 0,
    notes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by UUID REFERENCES users(id),
    updated_by UUID REFERENCES users(id)
);
CREATE INDEX idx_purchase_order_items_po ON purchase_order_items(purchase_order_id);

CREATE TABLE goods_receipt_notes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    grn_number VARCHAR(30) UNIQUE NOT NULL,
    purchase_order_id UUID NOT NULL REFERENCES purchase_orders(id),
    warehouse_id UUID NOT NULL REFERENCES warehouses(id),
    received_by UUID NOT NULL REFERENCES users(id),
    receipt_date TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    supplier_invoice_reference VARCHAR(120),
    notes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by UUID REFERENCES users(id),
    updated_by UUID REFERENCES users(id),
    deleted_at TIMESTAMPTZ
);
CREATE INDEX idx_grn_purchase_order ON goods_receipt_notes(purchase_order_id);

CREATE TABLE goods_receipt_note_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    goods_receipt_note_id UUID NOT NULL REFERENCES goods_receipt_notes(id) ON DELETE CASCADE,
    purchase_order_item_id UUID NOT NULL REFERENCES purchase_order_items(id),
    item_id UUID NOT NULL REFERENCES items(id),
    batch_id UUID REFERENCES stock_batches(id),
    quantity_received NUMERIC(18,4) NOT NULL CHECK (quantity_received > 0),
    batch_number VARCHAR(100),
    manufacture_date DATE,
    expiry_date DATE,
    unit_cost NUMERIC(18,2) NOT NULL CHECK (unit_cost >= 0),
    location VARCHAR(120),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by UUID REFERENCES users(id),
    updated_by UUID REFERENCES users(id)
);

CREATE TABLE production_receipts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    receipt_number VARCHAR(30) UNIQUE NOT NULL,
    warehouse_id UUID NOT NULL REFERENCES warehouses(id),
    recorded_by UUID NOT NULL REFERENCES users(id),
    receipt_date TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    notes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by UUID REFERENCES users(id),
    updated_by UUID REFERENCES users(id),
    deleted_at TIMESTAMPTZ
);
CREATE INDEX idx_production_receipts_date ON production_receipts(receipt_date DESC);

CREATE TABLE production_receipt_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    production_receipt_id UUID NOT NULL REFERENCES production_receipts(id) ON DELETE CASCADE,
    item_id UUID NOT NULL REFERENCES items(id),
    batch_id UUID REFERENCES stock_batches(id),
    quantity_produced NUMERIC(18,4) NOT NULL CHECK (quantity_produced > 0),
    batch_number VARCHAR(100),
    manufacture_date DATE,
    expiry_date DATE,
    unit_cost NUMERIC(18,2),
    quality_status VARCHAR(20) NOT NULL CHECK (quality_status IN ('accepted', 'rejected')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by UUID REFERENCES users(id),
    updated_by UUID REFERENCES users(id)
);

CREATE TABLE sales_orders (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_number VARCHAR(30) UNIQUE NOT NULL,
    customer_id UUID NOT NULL REFERENCES customers(id),
    status VARCHAR(30) NOT NULL DEFAULT 'confirmed',
    order_date DATE NOT NULL DEFAULT CURRENT_DATE,
    required_date DATE,
    notes TEXT,
    total_amount NUMERIC(18,2) NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by UUID REFERENCES users(id),
    updated_by UUID REFERENCES users(id),
    deleted_at TIMESTAMPTZ
);
CREATE INDEX idx_sales_orders_customer ON sales_orders(customer_id);
CREATE INDEX idx_sales_orders_status ON sales_orders(status);

CREATE TABLE sales_order_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    sales_order_id UUID NOT NULL REFERENCES sales_orders(id) ON DELETE CASCADE,
    item_id UUID NOT NULL REFERENCES items(id),
    quantity NUMERIC(18,4) NOT NULL CHECK (quantity > 0),
    reserved_quantity NUMERIC(18,4) NOT NULL DEFAULT 0 CHECK (reserved_quantity >= 0),
    delivered_quantity NUMERIC(18,4) NOT NULL DEFAULT 0 CHECK (delivered_quantity >= 0),
    unit_price NUMERIC(18,2) NOT NULL CHECK (unit_price >= 0),
    line_total NUMERIC(18,2) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by UUID REFERENCES users(id),
    updated_by UUID REFERENCES users(id)
);

CREATE TABLE sales_order_item_allocations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    sales_order_item_id UUID NOT NULL REFERENCES sales_order_items(id) ON DELETE CASCADE,
    warehouse_id UUID NOT NULL REFERENCES warehouses(id),
    batch_id UUID REFERENCES stock_batches(id),
    reserved_quantity NUMERIC(18,4) NOT NULL CHECK (reserved_quantity > 0),
    delivered_quantity NUMERIC(18,4) NOT NULL DEFAULT 0 CHECK (delivered_quantity >= 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by UUID REFERENCES users(id)
);

CREATE TABLE delivery_notes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    delivery_number VARCHAR(30) UNIQUE NOT NULL,
    sales_order_id UUID NOT NULL REFERENCES sales_orders(id),
    delivered_by UUID NOT NULL REFERENCES users(id),
    delivery_date TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    recipient_name VARCHAR(150),
    recipient_signature_reference TEXT,
    notes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by UUID REFERENCES users(id),
    updated_by UUID REFERENCES users(id),
    deleted_at TIMESTAMPTZ
);
CREATE INDEX idx_delivery_notes_order ON delivery_notes(sales_order_id);

CREATE TABLE delivery_note_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    delivery_note_id UUID NOT NULL REFERENCES delivery_notes(id) ON DELETE CASCADE,
    sales_order_item_id UUID NOT NULL REFERENCES sales_order_items(id),
    item_id UUID NOT NULL REFERENCES items(id),
    batch_id UUID REFERENCES stock_batches(id),
    quantity_delivered NUMERIC(18,4) NOT NULL CHECK (quantity_delivered > 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by UUID REFERENCES users(id)
);
