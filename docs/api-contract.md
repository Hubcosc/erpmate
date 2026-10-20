# API Contract

- Base path: `/api/v1`
- Authentication: JWT in `Authorization: Bearer <token>`
- Request/response content type: `application/json`
- Pagination: `?page=1&limit=20`
- Filtering: query parameters, for example `?status=approved&department_id=<uuid>`

The standard response envelope was not specified yet. Define the success and error envelope before implementing module endpoints. The starter health endpoint is an infrastructure check only and is not a finalized business API contract.

The implemented requisition endpoints currently use the envelope in `backend/internal/common/response/response.go`. Protected notification endpoints are mounted at `/notifications`; list supports `page`, `limit`, and `unread=true`, and authenticated users can mark their notifications read.

## Procurement, production, and sales endpoints

- `GET /purchase-orders?page=&limit=&status=`; finance-approved requisitions create a draft PO automatically.
- `GET /purchase-orders/{id}`, `PUT /purchase-orders/{id}` (draft only), `POST /purchase-orders/{id}/issue`.
- `POST /purchase-orders/from-requisition/{pr_id}` is an idempotent approved-PR handoff.
- `GET /goods-receipt-notes?page=&limit=&purchase_order_id=`, `POST /goods-receipt-notes`, `GET /goods-receipt-notes/{id}`.
- `GET /production-receipts?page=&limit=`, `POST /production-receipts`, `GET /production-receipts/{id}`.
- `GET /sales-orders?page=&limit=&status=`, `POST /sales-orders`, `GET /sales-orders/{id}`, `POST /sales-orders/{id}/deliveries`.
- `GET /delivery-notes?page=&limit=&sales_order_id=`, `GET /delivery-notes/{id}`, `GET /delivery-notes/{id}/pdf` (authenticated PDF download).
- `GET /inventory/valuation?page=&limit=&search=&warehouse_type=` returns on-hand/reserved/available quantities, a weighted-average unit cost estimate, and current inventory value. `cost_source` identifies batch costs, inbound movement costs, or missing cost data.
- `GET /inventory/alerts?page=&limit=` lists active low-stock and expiry alerts. The API starts a scanner at startup and runs it every 15 minutes; low-stock thresholds use item reorder/minimum levels and expiry warnings default to 30 days.
- `GET /reports/summary?from=YYYY-MM-DD&to=YYYY-MM-DD` returns sales, open purchase commitments, pending requisitions, and inventory alert counts.
- `GET /reports/sales?from=&to=&group_by=product|customer`, `GET /reports/material-consumption?from=&to=`, and `GET /reports/purchase-commitments` provide detail reports. Sales revenue is derived from delivered quantities and sales-order prices; commitments are based on outstanding issued PO lines.

Report access is role-limited in handlers. The current customer schema has no region field, so regional sales breakdowns are not available. Automated notifications are in-app only; email/SMS delivery is not included.

GRN posting and production-receipt acceptance atomically update stock balances, batches, and stock movements. Sales order creation reserves finished-goods stock; delivery notes atomically release reservations, decrement stock, and record stock movements. Delivery-note PDF export is a printable dispatch document, not an invoice or tax/e-invoice.
