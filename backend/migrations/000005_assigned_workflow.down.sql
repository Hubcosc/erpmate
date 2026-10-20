DROP TABLE IF EXISTS notifications;
DROP TABLE IF EXISTS notifications;
ALTER TABLE material_issue_items DROP COLUMN IF EXISTS updated_at, DROP COLUMN IF EXISTS created_at, DROP COLUMN IF EXISTS updated_by, DROP COLUMN IF EXISTS created_by;
ALTER TABLE material_issues DROP COLUMN IF EXISTS updated_at, DROP COLUMN IF EXISTS updated_by, DROP COLUMN IF EXISTS created_by, DROP COLUMN IF EXISTS acknowledged_at, DROP COLUMN IF EXISTS acknowledged_by;
ALTER TABLE material_requisition_items DROP COLUMN IF EXISTS updated_at, DROP COLUMN IF EXISTS created_at, DROP COLUMN IF EXISTS updated_by, DROP COLUMN IF EXISTS created_by;
ALTER TABLE material_requisitions DROP COLUMN IF EXISTS updated_by, DROP COLUMN IF EXISTS created_by;
ALTER TABLE purchase_requisition_items DROP COLUMN IF EXISTS updated_at, DROP COLUMN IF EXISTS created_at, DROP COLUMN IF EXISTS updated_by, DROP COLUMN IF EXISTS created_by;
ALTER TABLE purchase_requisitions DROP COLUMN IF EXISTS updated_by, DROP COLUMN IF EXISTS created_by, DROP COLUMN IF EXISTS return_reason, DROP COLUMN IF EXISTS preferred_supplier_id;
