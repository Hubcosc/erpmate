# Status Enums

This is the shared location for exact status values. Values should be agreed before implementing state transitions; do not invent alternative spellings in individual modules.

Initial workflow concepts from the requirements:

- Purchase requisition: pending production manager review, returned for correction, rejected, pending finance review, suspended by finance, approved by finance.
- Material requisition: pending production manager review, rejected, approved, issued, receipt acknowledged.
- Procurement, inventory, and sales statuses will be specified when those phases begin.

The currently implemented workflow values are centralized in `backend/internal/common/constants/statuses.go`:

- Purchase requisition: `draft`, `pending_pm`, `returned_for_correction`, `pending_finance`, `suspended`, `approved`, `rejected`.
- Material requisition: `draft`, `pending_pm`, `approved`, `rejected`, `partially_issued`, `fully_issued`.
- Purchase order: `draft`, `issued`, `partially_received`, `received`.
- Sales order: `confirmed`, `partially_delivered`, `delivered`.
- Production receipt line quality: `accepted`, `rejected`.

Use these values consistently. Review any additional state or transition with the workflow owner before adding it.
