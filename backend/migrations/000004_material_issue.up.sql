CREATE TABLE material_issues (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    issue_number    VARCHAR(30) UNIQUE NOT NULL,
    mr_id           UUID NOT NULL REFERENCES material_requisitions(id),
    issued_by       UUID NOT NULL REFERENCES users(id),
    issued_to       UUID REFERENCES users(id),
    issued_date     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    notes           TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE material_issue_items (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    issue_id        UUID NOT NULL REFERENCES material_issues(id) ON DELETE CASCADE,
    mr_item_id      UUID NOT NULL REFERENCES material_requisition_items(id),
    item_id         UUID NOT NULL REFERENCES items(id),
    batch_id        UUID REFERENCES stock_batches(id),
    quantity        NUMERIC(18,4) NOT NULL
);

CREATE INDEX idx_material_issues_mr ON material_issues(mr_id);