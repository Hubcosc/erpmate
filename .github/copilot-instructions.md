# ERP project guidance

- Keep database table and column names in snake_case. Use UUID primary keys, soft deletion via `deleted_at`, and audit columns (`created_at`, `updated_at`, `created_by`, `updated_by`) where applicable.
- Use `NUMERIC(18,2)` for monetary values and `NUMERIC(18,4)` for quantities.
- Keep the API under `/api/v1`, use JSON, JWT Bearer authentication, and `page`/`limit` pagination.
- Treat `docs/` as the shared contract source. Do not invent final status enums or a standard API response envelope until the team agrees on them.
- Keep backend features organized by module using handler/service/repository/model/DTO layers. Keep frontend features in their matching module folders.
- This scaffold is a foundation only; do not imply workflows or database persistence exist until implemented and tested.
