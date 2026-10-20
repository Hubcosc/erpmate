-- Purchase Requisitions
CREATE TABLE purchase_requisitions (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    pr_number               VARCHAR(30) UNIQUE NOT NULL,
    department_id           UUID NOT NULL REFERENCES departments(id),
    requested_by            UUID NOT NULL REFERENCES users(id),
    required_date           DATE,
    justification           TEXT,
    status                  VARCHAR(30) NOT NULL DEFAULT 'draft',
    pm_approved_by          UUID REFERENCES users(id),
    pm_approved_at          TIMESTAMPTZ,
    pm_rejection_reason     TEXT,
    finance_approved_by     UUID REFERENCES users(id),
    finance_approved_at     TIMESTAMPTZ,
    finance_rejection_reason TEXT,
    suspension_reason       TEXT,
    total_estimated         NUMERIC(18,2) NOT NULL DEFAULT 0,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at              TIMESTAMPTZ
);

CREATE TABLE purchase_requisition_items (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    pr_id                   UUID NOT NULL REFERENCES purchase_requisitions(id) ON DELETE CASCADE,
    item_id                 UUID REFERENCES items(id),
    description             TEXT,
    quantity                NUMERIC(18,4) NOT NULL,
    unit_of_measure         VARCHAR(20),
    estimated_unit_price    NUMERIC(18,2),
    estimated_total         NUMERIC(18,2),
    notes                   TEXT
);

CREATE TABLE status_history (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    entity_type     VARCHAR(50) NOT NULL,
    entity_id       UUID NOT NULL,
    old_status      VARCHAR(30),
    new_status      VARCHAR(30) NOT NULL,
    changed_by      UUID REFERENCES users(id),
    reason          TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_pr_status ON purchase_requisitions(status);
CREATE INDEX idx_pr_requested_by ON purchase_requisitions(requested_by);
CREATE INDEX idx_pr_department ON purchase_requisitions(department_id);
CREATE INDEX idx_status_history_entity ON status_history(entity_type, entity_id);