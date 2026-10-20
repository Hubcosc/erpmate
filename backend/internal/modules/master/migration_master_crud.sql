-- Run once. Safe to re-run (IF NOT EXISTS).
-- Needed because the new code soft-deletes every master entity and updates
-- updated_at on every PUT/PATCH.

ALTER TABLE item_categories ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();
ALTER TABLE item_categories ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ;

ALTER TABLE warehouses ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();
ALTER TABLE warehouses ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ;

-- These already have deleted_at (your queries used it); included for completeness.
ALTER TABLE departments ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ;
ALTER TABLE items       ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ;
ALTER TABLE suppliers   ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ;
ALTER TABLE customers   ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ;
