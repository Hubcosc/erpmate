# Database Schema Conventions

This document is the shared source of truth for database design. Schema details will be added as modules are implemented.

- Use `snake_case` for table and column names.
- Use `id UUID` as the primary key.
- Include `deleted_at TIMESTAMP NULL` for soft deletion where applicable.
- Include `created_at`, `updated_at`, `created_by`, and `updated_by` audit columns on operational tables.
- Store monetary values as `NUMERIC(18,2)` and quantities as `NUMERIC(18,4)`.
- Use foreign keys and indexes for relationships and common filters.
- Preserve inventory changes as traceable stock movements; do not update stock without recording the movement.

Migrations are managed from `backend/migrations/` and are applied in filename order at API startup. The baseline defines shared identity, master-data, warehouse, and stock tables; later migrations add the requisition, notification, procurement, production, and sales workflows.

The migration runner tracks applied versions in `schema_migrations` and adopts existing workflow schemas when their identifying tables are already present. Migration `000006` defines PO/GRN, finished-goods receipts, sales orders, delivery notes, reservation allocations, and their number sequences. Migration `000007` adds the active inventory-alert register used to deduplicate and resolve automated low-stock/expiry notifications. Down migrations exist, but rollback is currently a manual DBA operation.
