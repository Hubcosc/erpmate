# Inventory API (frontend contract)

All routes are read-only `GET`s. Dates are `YYYY-MM-DD`. Quantities have up to 4 decimals, money 2.
List endpoints return `{ "data": [...], "meta": { "page", "limit", "total" } }` (default limit 20, max 100).
Multi-value filters are comma separated: `?status=low_stock,out_of_stock`.

## Errors (same shape as the procurement module)
| HTTP | code | when |
|---|---|---|
| 400 | VALIDATION_ERROR | bad filter / sort / id. `details` is `{ "field": "message" }` - show it next to the input |
| 404 | NOT_FOUND | item detail for an unknown / inactive item |
| 500 | INTERNAL_ERROR | generic; real error is only in server logs |

## Endpoints
| Path | Filters | Use it for |
|---|---|---|
| `GET /lookups` | - | dropdown values: `stock_statuses`, `alert_types`, `alert_severities`, `cost_sources`, `expiry_statuses` |
| `GET /stock` | `search item_type warehouse_type warehouse_id category_id status sort order page limit` | stock table (one row per item x warehouse) |
| `GET /stock/summary` | same filters (`status` ignored) | status tabs / KPI cards: `total_lines total_items in_stock low_stock out_of_stock` |
| `GET /stock/{itemID}` | - | item page: header + totals + `warehouses[]` |
| `GET /stock/{itemID}/batches` | `warehouse_type warehouse_id expiry_status` | FEFO batch table (all warehouses unless filtered) |
| `GET /movements` | `item_id warehouse_id movement_type reference_type search date_from date_to page limit` | stock ledger |
| `GET /movements/types` | - | values for the movement-type filter |
| `GET /valuation` | `search item_type warehouse_type warehouse_id cost_source low_stock sort order page limit` | valuation table |
| `GET /valuation/summary` | same filters | `total_value total_lines lines_without_cost by_warehouse_type[]` |
| `GET /alerts` | `alert_type severity warehouse_id search page limit` | active alerts, most severe first |
| `GET /alerts/summary` | `warehouse_id search` | counts by type and severity |

Sort keys - stock: `name code warehouse quantity reserved available`; valuation: `name code warehouse quantity value unit_cost`. `order=asc|desc`.

## Rules the UI can rely on
- **Stock status** (one rule everywhere): `threshold = reorder_level, else min_stock_level`.
  `out_of_stock` = available <= 0; `low_stock` = threshold > 0 and 0 < available <= threshold; otherwise `in_stock`.
  Use `stock_status` for tabs. The boolean `low_stock` is also true for zero stock with a threshold, so it can overlap `out_of_stock`.
- **Batch `expiry_status`**: `expired`, `near_expiry` (within the configured warning days), `ok`, `no_expiry`.
  `available_quantity` per batch is allocated earliest-expiry-first and never adds up to more than the warehouse's available stock.
- **Alert severity**: `expired` = critical; `low_stock` = critical at zero stock, otherwise warning; `near_expiry` = warning within 7 days, otherwise info.
- Money fields are rounded to 2 decimals. Empty lists are `[]`, never `null`. Nullable fields are always present as `null`.
- Alert `expiry_date` and batch `expiry_date` use the same `YYYY-MM-DD` string format.
