CREATE TABLE material_requisitions (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    mr_number           VARCHAR(30) UNIQUE NOT NULL,
    department_id       UUID NOT NULL REFERENCES departments(id),
    requested_by        UUID NOT NULL REFERENCES users(id),
    required_date       DATE,
    purpose             TEXT,
    status              VARCHAR(30) NOT NULL DEFAULT 'draft',
    -- draft, pending_pm, approved, rejected, partially_issued, fully_issued, cancelled
    pm_approved_by      UUID REFERENCES users(id),
    pm_approved_at      TIMESTAMPTZ,
    pm_rejection_reason TEXT,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at          TIMESTAMPTZ
);

CREATE TABLE material_requisition_items (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    mr_id           UUID NOT NULL REFERENCES material_requisitions(id) ON DELETE CASCADE,
    item_id         UUID NOT NULL REFERENCES items(id),
    quantity        NUMERIC(18,4) NOT NULL,
    issued_qty      NUMERIC(18,4) NOT NULL DEFAULT 0,
    notes           TEXT
);

CREATE INDEX idx_mr_status ON material_requisitions(status);
CREATE INDEX idx_mr_requested_by ON material_requisitions(requested_by);
CREATE INDEX idx_mr_department ON material_requisitions(department_id);