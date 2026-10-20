# Demo data seeder

`cmd/seed-demo` fills an empty ErpMate database with a realistic, end-to-end
scenario so every screen has something to show.

A fresh database has no master data at all — no items, no warehouses, no
departments — so every module renders an empty state and there is nothing to
click through. This command fixes that in one step.

## What it creates

| Area | Records |
| --- | --- |
| Master data | 3 departments, 3 item categories, 6 items, 3 suppliers, 3 customers, 3 warehouses |
| Users | 11 — one per role, plus two more department heads |
| Procurement | A requisition that runs all the way to a goods receipt note, and a second one left awaiting approval |
| Materials | A requisition that is issued from the store and acknowledged |
| Production | A production receipt with an accepted and a rejected line |
| Sales | Three orders covering `confirmed`, `partially_delivered` and `delivered`, with two delivery notes |
| Stock | Receipts and issues produce real batches, balances and stock movements |

The document numbers (`PR-2026-0001`, `PO-2026-000002`, …) and the dates are
generated relative to today, so the demo always looks current and the expiry
alerts are meaningful.

## Running it

The API has to be running first, because the seeder talks to it over HTTP.

```bash
# terminal 1 — the API
cd backend
go run ./cmd/api

# terminal 2 — the seed
cd backend
go run ./cmd/seed-demo -reset -password "your-admin-password"
```

Then open http://localhost:5173 and sign in.

### Flags

| Flag | Default | Purpose |
| --- | --- | --- |
| `-reset` | off | Truncate every transactional table first, so the seed always starts from a known state. Roles and the migration history are kept. |
| `-api` | `http://localhost:8080/api/v1` | Base URL of the running API. |
| `-email` | `$BOOTSTRAP_ADMIN_EMAIL` | The administrator to sign in with. |
| `-password` | `$BOOTSTRAP_ADMIN_PASSWORD` | The administrator password. Required. |

`-reset` also recreates the administrator if the reset removed it, so you never
have to restart the API between a reset and a seed.

### The demo accounts

Every seeded account has the password `DemoPass12345!`.

| Email | Role | What to look at |
| --- | --- | --- |
| `merina.adam@example.test` | Department Head | Purchase requisitions, material requests |
| `abel.tadesse@example.test` | Production Manager | Approval queues, production receipts, reports |
| `sara.girma@example.test` | Finance | Requisition approvals, purchase orders, reports |
| `yonas.bekele@example.test` | Purchasing | Purchase orders, goods receipts |
| `marta.tesfaye@example.test` | Store Keeper | Main Store, goods receipts, stock alerts |
| `dawit.alemu@example.test` | Production Store Keeper | Finished Goods Store, production receipts |
| `hana.mulugeta@example.test` | Sales | Sales orders, delivery notes |
| `kalkidan.fitsum@example.test` | Sales Head | Sales reports |
| `tewodros.girma@example.test` | Department Head | A requisition waiting for approval |
| `selam.worku@example.test` | Department Head | Quality Assurance |

The administrator is `admin@example.test` with the password from your `.env`.

## Why it calls the API instead of writing SQL

Every document is posted through the real endpoints, so the demo data is
created exactly the way the application creates it:

- stock movements, batches and balances are written by the same service code
  that handles a real goods receipt
- reservations on sales orders are made and released properly
- status history and in-app notifications are generated as a side effect
- each step is performed **by the role that is allowed to perform it**, which
  means running the seeder also exercises the role checks

Writing the rows directly with SQL would have been faster, but it would have
produced a database the application could never have made.

## Adding your own demo data

The seeder is a plain Go program, so adding data means adding a function and
calling it from `main`.

### 1. Master data

Use the helpers in `main.go`. They look for an existing record first, so they
are safe to call again:

```go
productionDept := createDepartment("Production", "PROD")
rawCategory := createCategory("Raw Materials")
steelSheet := createItem(map[string]any{
    "code": "RM-001", "name": "Steel Sheet 2mm", "category_id": rawCategory,
    "item_type": "raw_material", "unit_of_measure": "kg",
    "min_stock_level": 100, "max_stock_level": 800, "reorder_level": 150,
    "is_batch_tracked": true, "is_expiry_tracked": false,
})
metalSupplies := createSupplier("SUP-001", "Metal Supplies Ltd")
mesfin := createCustomer("CUS-001", "Mesfin Construction", "local")
mainStore := createWarehouse("WH-RM", "Main Raw Material Store", "raw_material", "Building A")
```

### 2. Users

```go
createUser(roleID["store_keeper"], "Marta Tesfaye", "marta.tesfaye@example.test", demoPassword, "")
```

`roleID` is built from `GET /users/roles` at the start of the run, so a new role
in the database is available immediately.

### 3. A workflow

Write one function per document, sign in as the right user, and call the
endpoint:

```go
func seedMaterialIssue(mrID, issuedToUserID string) {
    // The store keeper issues the material from the raw material store.
    must("sign in as store keeper", c.loginAs("marta.tesfaye@example.test", demoPassword))

    var mr materialRequisition
    must("load material requisition", c.request(http.MethodGet, "/material-requisitions/"+mrID, nil, &mr))

    items := make([]map[string]any, 0, len(mr.Items))
    for _, line := range mr.Items {
        outstanding := line.Quantity - line.IssuedQty
        if outstanding <= 0 {
            continue
        }
        entry := map[string]any{
            "mr_item_id": line.ID,
            "item_id":    line.ItemID,
            "quantity":   outstanding,
        }
        // Batch-tracked items must be issued from a specific batch.
        if batchID, _ := pickBatch(line.ItemID); batchID != "" {
            entry["batch_id"] = batchID
        }
        items = append(items, entry)
    }

    body := map[string]any{
        "mr_id":     mrID,
        "issued_to": issuedToUserID,
        "items":     items,
    }
    var issue materialIssue
    must("create material issue", c.request(http.MethodPost, "/material-requisitions/issues", body, &issue))
    step("posted material issue %s", issue.IssueNumber)
}
```

Then call it from `main`:

```go
mrID := seedMaterialRequisition(steelSheet, aluminiumRod, productionDept)
seedMaterialIssue(mrID, departmentHeadID)
```

### Things worth knowing

- **`must` stops the run on the first error.** Use it for anything that must
  succeed, and `step` to report progress.
- **The shapes in `main.go` mirror the API**, not the database. The `item`,
  `purchaseRequisition`, `salesOrder` and similar structs hold the JSON the
  endpoints return. If you add a field to a response, add it to the struct.
- **`pickBatch` allocates earliest-expiry-first.** Batch-tracked items cannot
  be issued without a `batch_id`, so always use it.
- **Dates come from `day(offset)`**, which returns a `YYYY-MM-DD` string
  relative to today. Use negative offsets for things that happened in the past.
- **`issued_to` takes a user id**, not a department id.
- **Add the new account to the list printed at the end of `main`** so the
  password is easy to find.

### Running it against another database

```bash
DB_NAME=erp_test go run ./cmd/seed-demo -reset -password "..."
```

`config.Load()` reads the same `.env` and environment variables the API uses.
