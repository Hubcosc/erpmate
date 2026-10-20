-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- ======================
-- 1. ROLES
-- ======================
CREATE TABLE roles (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        VARCHAR(50) UNIQUE NOT NULL,
    description TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ======================
-- 2. DEPARTMENTS
-- ======================
CREATE TABLE departments (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        VARCHAR(100) NOT NULL,
    code        VARCHAR(20) UNIQUE,
    is_active   BOOLEAN NOT NULL DEFAULT TRUE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at  TIMESTAMPTZ
);

-- ======================
-- 3. USERS
-- ======================
CREATE TABLE users (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    employee_id     VARCHAR(50) UNIQUE,
    full_name       VARCHAR(150) NOT NULL,
    email           VARCHAR(150) UNIQUE NOT NULL,
    phone           VARCHAR(30),
    password_hash   TEXT NOT NULL,
    department_id   UUID REFERENCES departments(id),
    role_id         UUID NOT NULL REFERENCES roles(id),
    is_active       BOOLEAN NOT NULL DEFAULT TRUE,
    last_login_at   TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at      TIMESTAMPTZ
);

-- ======================
-- 4. ITEM CATEGORIES
-- ======================
CREATE TABLE item_categories (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        VARCHAR(100) NOT NULL,
    parent_id   UUID REFERENCES item_categories(id),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ======================
-- 5. ITEMS
-- ======================
CREATE TABLE items (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code                VARCHAR(50) UNIQUE NOT NULL,
    name                VARCHAR(200) NOT NULL,
    description         TEXT,
    category_id         UUID REFERENCES item_categories(id),
    item_type           VARCHAR(30) NOT NULL CHECK (item_type IN ('raw_material', 'finished_good', 'consumable', 'spare_part')),
    unit_of_measure     VARCHAR(20) NOT NULL,
    min_stock_level     NUMERIC(18,4) NOT NULL DEFAULT 0,
    max_stock_level     NUMERIC(18,4),
    reorder_level       NUMERIC(18,4),
    is_batch_tracked    BOOLEAN NOT NULL DEFAULT FALSE,
    is_expiry_tracked   BOOLEAN NOT NULL DEFAULT FALSE,
    is_active           BOOLEAN NOT NULL DEFAULT TRUE,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at          TIMESTAMPTZ
);

-- ======================
-- 6. SUPPLIERS
-- ======================
CREATE TABLE suppliers (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code            VARCHAR(50) UNIQUE,
    name            VARCHAR(200) NOT NULL,
    contact_person  VARCHAR(100),
    phone           VARCHAR(30),
    email           VARCHAR(150),
    address         TEXT,
    tin_number      VARCHAR(50),
    is_active       BOOLEAN NOT NULL DEFAULT TRUE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at      TIMESTAMPTZ
);

-- ======================
-- 7. CUSTOMERS
-- ======================
CREATE TABLE customers (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code            VARCHAR(50) UNIQUE,
    name            VARCHAR(200) NOT NULL,
    customer_type   VARCHAR(20) NOT NULL DEFAULT 'local' CHECK (customer_type IN ('local', 'international')),
    contact_person  VARCHAR(100),
    phone           VARCHAR(30),
    email           VARCHAR(150),
    address         TEXT,
    tin_number      VARCHAR(50),
    is_active       BOOLEAN NOT NULL DEFAULT TRUE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at      TIMESTAMPTZ
);

-- ======================
-- 8. WAREHOUSES
-- ======================
CREATE TABLE warehouses (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code            VARCHAR(30) UNIQUE NOT NULL,
    name            VARCHAR(100) NOT NULL,
    warehouse_type  VARCHAR(30) NOT NULL CHECK (warehouse_type IN ('raw_material', 'finished_goods', 'quarantine')),
    location        TEXT,
    is_active       BOOLEAN NOT NULL DEFAULT TRUE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ======================
-- 9. STOCK BALANCES
-- ======================
CREATE TABLE stock_balances (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    item_id         UUID NOT NULL REFERENCES items(id),
    warehouse_id    UUID NOT NULL REFERENCES warehouses(id),
    quantity        NUMERIC(18,4) NOT NULL DEFAULT 0,
    reserved_qty    NUMERIC(18,4) NOT NULL DEFAULT 0,
    available_qty   NUMERIC(18,4) GENERATED ALWAYS AS (quantity - reserved_qty) STORED,
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (item_id, warehouse_id)
);

-- ======================
-- 10. STOCK BATCHES
-- ======================
CREATE TABLE stock_batches (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    item_id          UUID NOT NULL REFERENCES items(id),
    warehouse_id     UUID NOT NULL REFERENCES warehouses(id),
    batch_number     VARCHAR(100) NOT NULL,
    quantity         NUMERIC(18,4) NOT NULL,
    manufacture_date DATE,
    expiry_date      DATE,
    unit_cost        NUMERIC(18,4),
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (item_id, warehouse_id, batch_number)
);

-- ======================
-- 11. STOCK MOVEMENTS
-- ======================
CREATE TABLE stock_movements (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    movement_type   VARCHAR(30) NOT NULL,
    reference_type  VARCHAR(50),
    reference_id    UUID,
    item_id         UUID NOT NULL REFERENCES items(id),
    warehouse_id    UUID NOT NULL REFERENCES warehouses(id),
    batch_id        UUID REFERENCES stock_batches(id),
    quantity        NUMERIC(18,4) NOT NULL, -- positive = in, negative = out
    unit_cost       NUMERIC(18,4),
    notes           TEXT,
    performed_by    UUID REFERENCES users(id),
    movement_date   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ======================
-- INDEXES
-- ======================
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_department ON users(department_id);
CREATE INDEX idx_items_code ON items(code);
CREATE INDEX idx_items_type ON items(item_type);
CREATE INDEX idx_stock_balances_item ON stock_balances(item_id);
CREATE INDEX idx_stock_movements_item ON stock_movements(item_id);
CREATE INDEX idx_stock_movements_date ON stock_movements(movement_date);

-- ======================
-- SEED DATA (Roles)
-- ======================
INSERT INTO roles (name, description) VALUES
('admin', 'System Administrator'),
('department_head', 'Department Head'),
('production_manager', 'Production Manager'),
('finance', 'Finance'),
('purchasing', 'Purchasing Team'),
('store_keeper', 'Raw Material Store Keeper'),
('production_store_keeper', 'Production Store Keeper'),
('sales', 'Sales Staff'),
('sales_head', 'Sales Head');