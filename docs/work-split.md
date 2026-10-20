# Work Split

Modules are separated so contributors can work independently while sharing the database and conventions.

- Shared foundations: authentication, users/roles, master data, API response/pagination utilities, database, migrations, router.
- Phase 1: purchase requisitions and approvals, material requisitions and approvals, raw-material inventory/GRN/issue, basic notifications.
- Phase 2: purchase orders, finished-goods production receipts, sales orders, delivery documents.
- Phase 3: reporting, dashboards, stock valuation, low-stock and expiry alerts. Initial reporting, weighted-average valuation estimates, and in-app alert automation are now available; advanced accounting valuation, regional sales reports, and external email/SMS alerts remain future work.

Coordinate shared model, status, and API contract changes through `docs/` before implementing dependent modules.

Assigned ownership includes Purchase Requisition, Material Requisition, Material Issue, history for those flows, and basic in-app notifications. Finance approval now creates a draft PO atomically; coordinate any changes to that shared handoff with Procurement.

The paired workflows now include procurement PO/GRN, production receipts, sales orders, and delivery notes. Finance approval creates a draft PO in the same database transaction; Purchasing confirms supplier and line prices, issues the PO, and the raw-material Store Keeper posts GRNs. Production receipts add accepted finished goods to a finished-goods warehouse. Sales order creation reserves finished goods and delivery-note posting reduces stock. These modules use migration `000006_procurement_production_sales` after the earlier migrations.

Inventory alerts are stored in `inventory_alerts`, sent as in-app notifications to the relevant store-keeper role and admins on activation, and automatically resolved when the condition clears. The background scan starts immediately and repeats every 15 minutes, with a 30-day near-expiry window. Inventory valuation uses a weighted-average estimate from remaining priced batches, falling back to inbound movement costs; it is not a perpetual accounting ledger. Reports include delivery-derived revenue, material issues, and outstanding issued PO lines. Delivery notes can be downloaded as PDFs; sales invoices and packing lists remain future documents.
