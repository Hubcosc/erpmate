# Procurement API (frontend contract)

Mount: `r.Mount("/procurement", procurement.NewHandler(db).Routes())`
All routes need auth. Dates are `YYYY-MM-DD`. Money/quantities are JSON numbers.

## Response envelope
Success: `{ "data": ..., "meta": { "page", "limit", "total" } }` (meta on lists only; uses your `response` package).
Error codes returned by this module:

| HTTP | code | meaning |
|------|------|---------|
| 400 | VALIDATION_ERROR | bad input - show `message` next to the form |
| 401 | UNAUTHORIZED | not logged in |
| 403 | FORBIDDEN | role not allowed - hide the button |
| 404 | NOT_FOUND | record missing |
| 409 | CONFLICT | wrong state (e.g. editing a non-draft PO, over-receipt) |
| 500 | INTERNAL_ERROR | generic; details are only in server logs |

## PO lifecycle
`draft -> issued -> partially_received -> received`

## Purchase orders - `/procurement/purchase-orders`
| Method & path | Roles | Notes |
|---|---|---|
| `GET /` | purchasing, finance, admin | `?status=&supplier_id=&search=&page=&limit=` |
| `POST /from-requisition/{prId}` | purchasing, admin | 201 when created, 200 if the PO already existed |
| `GET /{id}` | purchasing, finance, admin | includes `items[]` |
| `PUT` or `PATCH /{id}` | purchasing, admin | draft only. Partial: omitted field = unchanged, `""` = clear |
| `POST /{id}/issue` | purchasing, admin | needs supplier + a price on every line |

Update body:
```json
{
  "supplier_id": "uuid",
  "expected_delivery_date": "2026-11-30",
  "payment_terms": "Net 30",
  "delivery_terms": "FOB",
  "notes": "",
  "items": [{ "id": "po-item-uuid", "unit_price": 125.50 }]
}
```
Each PO item returns `quantity`, `received_quantity`, `remaining_quantity`,
`is_batch_tracked`, `unit_price`, `line_total`. Use `is_batch_tracked` to
show/require the batch field and `remaining_quantity` as the max on the receipt form.

## Goods receipts - `/procurement/goods-receipts`
| Method & path | Roles |
|---|---|
| `GET /` (`?purchase_order_id=&page=&limit=`) | store_keeper, purchasing, finance, admin |
| `GET /{id}` | same |
| `POST /` | store_keeper, admin |

Create body (`unit_cost` optional - defaults to the PO line price):
```json
{
  "purchase_order_id": "uuid",
  "warehouse_id": "uuid (raw_material warehouse)",
  "supplier_invoice_reference": "INV-1001",
  "notes": "",
  "items": [{
    "purchase_order_item_id": "uuid",
    "quantity_received": 10,
    "batch_number": "B-2026-01",
    "manufacture_date": "2026-09-01",
    "expiry_date": "2027-09-01",
    "unit_cost": 125.5,
    "location": "Rack A1"
  }]
}
```
