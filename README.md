<div align="center">

# 🏢 ErpMate

### Procure-to-pay, inventory, production and sales — in one ERP

A modern full-stack ERP that manages the complete **procure-to-pay lifecycle**, **multi-warehouse inventory**, **production receipts**, **sales & delivery**, and **master data**.

[![Go](https://img.shields.io/badge/Go-1.21+-00ADD8?logo=go&logoColor=white)](https://go.dev/)
[![React](https://img.shields.io/badge/React-18-61DAFB?logo=react&logoColor=black)](https://react.dev/)
[![TypeScript](https://img.shields.io/badge/TypeScript-5-3178C6?logo=typescript&logoColor=white)](https://www.typescriptlang.org/)
[![Vite](https://img.shields.io/badge/Vite-646CFF?logo=vite&logoColor=white)](https://vitejs.dev/)
[![PostgreSQL](https://img.shields.io/badge/PostgreSQL-15+-4169E1?logo=postgresql&logoColor=white)](https://www.postgresql.org/)
[![License: MIT](https://img.shields.io/badge/License-MIT-green.svg)](#-license)

[Features](#-features) · [Workflow](#-how-it-works) · [Quick start](#-quick-start) · [API](#-api-overview) · [Troubleshooting](#-troubleshooting) · [Contributing](#-contributing)

</div>

---

## 📸 Screenshots

| Admin Dashboard |
| :---: |
| ![Admin Dashboard](https://github.com/user-attachments/assets/2bf66e81-2c41-4baa-a60f-f2f481a377ff) |

---

## 🚀 Features

### 🛒 Procurement
- **Purchase Requisitions** — create, submit, return for correction, and track full approval history
- **Multi-level approvals** — Production Manager, then Finance (approve / suspend / reject)
- **Purchase Orders** — generated from approved PRs; edit supplier, pricing and terms, then issue
- **Goods Receipt Notes** — receive against POs with batch, expiry and cost; stock updates automatically

### 🧪 Production & Materials
- **Material requests** — department requests with live stock availability
- **Material issues** — issue from the raw-material store, with department acknowledgment
- **Production receipts** — record finished goods with quality status (accepted / rejected), batch and expiry

### 📦 Inventory
- **Stock dashboard** — live balances across warehouses with *In stock / Low stock / Out of stock* status
- **FEFO batches** — earliest-expiry-first allocation with expiry status
- **Stock ledger** — every receipt, issue and movement
- **Valuation** — weighted-average cost and inventory value
- **Smart alerts** — a background worker raises *low stock*, *near expiry* and *expired* alerts and notifies the right store keeper

### 💼 Sales & Delivery
- **Sales orders** — stock is reserved on creation
- **Delivery notes** — partial deliveries, printable and downloadable as **PDF**

### 🗂 Master Data
- Users & roles, items & categories, suppliers & customers
- Warehouses (raw material, finished goods, quarantine)

---

## 🔄 How It Works

```text
Purchase Requisition
        |  PM + Finance approval
        v
  Purchase Order
        |  issued to supplier
        v
 Goods Receipt Note ---> Raw-material store
                               |  material issue
                               v
                          Production
                               |  production receipt
                               v
                     Finished-goods store
                               |  reserve and deliver
                               v
                  Sales Order ---> Delivery Note
```

### 📋 Status lifecycles

| Document | Statuses (in order) |
| --- | --- |
| Purchase requisition | `draft` → `pending_pm` → `pending_finance` → `approved` |
| | Side paths: `returned_for_correction` (PM returns, requester resubmits), `suspended` (Finance, can still be approved), `rejected` |
| Purchase order | `draft` → `issued` → `partially_received` → `received` |
| Material request | `draft` → `pending_pm` → `approved` → `partially_issued` → `fully_issued` (or `rejected`) |
| Sales order | `confirmed` → `partially_delivered` → `delivered` |

### 👥 Roles

| Role | Responsibilities |
| --- | --- |
| `admin` | Full access, user and master-data management |
| `purchasing` | Create POs from approved PRs, set suppliers and prices, issue POs |
| `finance` | Budget approval of purchase requisitions; view POs, GRNs and valuation |
| `production_manager` | First-level approval of purchase and material requests |
| `department_head` | Raise requests, acknowledge received materials |
| `store_keeper` | Raw-material store: goods receipts, material issues, stock alerts |
| `production_store_keeper` | Finished-goods store: production receipts, stock alerts |
| `sales` / `sales_head` | Sales orders and delivery notes |

---

## 🛠️ Tech Stack

| Layer | Technology |
| --- | --- |
| **Frontend** | React, Vite, TypeScript, custom CSS |
| **Backend** | Go, [Chi](https://github.com/go-chi/chi) router, [pgx](https://github.com/jackc/pgx), godotenv, bcrypt |
| **Database** | PostgreSQL |
| **Auth** | JWT with role-based access control |

### Prerequisites

| Tool | Version | Download |
| --- | --- | --- |
| Go | 1.21+ | [go.dev/dl](https://go.dev/dl/) |
| Node.js & npm | 18+ | [nodejs.org](https://nodejs.org/) |
| PostgreSQL | 15+ | [postgresql.org](https://www.postgresql.org/download/) |
| Git | any | [git-scm.com](https://git-scm.com/) |

### Backend dependencies

| Package | Purpose |
| --- | --- |
| [`go-chi/chi/v5`](https://github.com/go-chi/chi) | HTTP router |
| [`go-chi/cors`](https://github.com/go-chi/cors) | CORS middleware for the React frontend |
| [`jackc/pgx/v5`](https://github.com/jackc/pgx) | PostgreSQL driver and connection pool |
| [`golang-jwt/jwt/v5`](https://github.com/golang-jwt/jwt) | JWT creation and validation |
| [`golang.org/x/crypto`](https://pkg.go.dev/golang.org/x/crypto/bcrypt) | `bcrypt` password hashing |
| [`google/uuid`](https://github.com/google/uuid) | UUID primary keys |
| [`joho/godotenv`](https://github.com/joho/godotenv) | Loads `.env` files |

---

## ⚡ Quick Start

### 1. Clone the repository

```bash
git clone https://github.com/Hubcosc/erpmate.git
cd erpmate
```

### 2. Set up the database

1. Open **pgAdmin** or `psql`.
2. Create a database named `erp_inventory`.
3. *(Optional)* create a dedicated user:

```sql
CREATE USER erp_user WITH PASSWORD 'replace-with-a-strong-password';
CREATE DATABASE erp_inventory OWNER erp_user;
```

### 3. Run the backend

```bash
cd backend

# 1. Install the Go dependencies
#    (skip the `go get` lines if go.mod already lists them — `go mod download` is enough)
go get github.com/go-chi/chi/v5
go get github.com/go-chi/cors
go get github.com/jackc/pgx/v5
go get github.com/golang-jwt/jwt/v5
go get golang.org/x/crypto/bcrypt
go get github.com/google/uuid
go get github.com/joho/godotenv
go mod tidy

# 2. Create your environment file, then edit it (see "Configuration" below)
cp .env.example .env          # Windows PowerShell: copy .env.example .env

# 3. Start the server (pending migrations are applied automatically)
go run ./cmd/api
```

The API starts on **http://localhost:8080**.

### 4. Run the frontend

In a second terminal:

```bash
cd frontend
npm install
npm run dev
```

The app starts on **http://localhost:5173**.

### 5. Sign in

On first start the backend creates a bootstrap admin from `BOOTSTRAP_ADMIN_EMAIL` and `BOOTSTRAP_ADMIN_PASSWORD` in your `backend/.env` (see the **Configuration** section below). Log in at http://localhost:5173 with those credentials.

> **Warning:** Change the bootstrap password and set a strong, unique `JWT_SECRET` before deploying anywhere. Never commit your `.env` file.

---

## ⚙️ Configuration

### Backend — `backend/.env`

Copy `backend/.env.example` to `backend/.env` and adjust the values:

```env
# ---------- Server ----------
APP_PORT=8080

# ---------- Database ----------
DB_HOST=localhost
DB_PORT=5432
DB_USER=erp_user
DB_PASSWORD=replace-with-a-strong-password
DB_NAME=erp_inventory
DB_SSLMODE=disable

# ---------- Auth ----------
# Use a long random string, e.g.:  openssl rand -base64 48
JWT_SECRET=replace-with-a-long-random-string

# ---------- Bootstrap admin (created on first start) ----------
BOOTSTRAP_ADMIN_EMAIL=admin@example.test
BOOTSTRAP_ADMIN_PASSWORD=replace-with-a-strong-password
```

> **Note:** Always use the exact variable names listed in `backend/.env.example` — that file is the source of truth for what the backend reads.

### Frontend — `frontend/.env`

```env
VITE_API_BASE_URL=http://localhost:8080/api/v1
```

Restart `npm run dev` after changing this file.

### Variable reference

| Variable | Purpose |
| --- | --- |
| `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD`, `DB_NAME`, `DB_SSLMODE` | PostgreSQL connection |
| `JWT_SECRET` | Secret used to sign auth tokens — **required, keep private** |
| `BOOTSTRAP_ADMIN_EMAIL` / `BOOTSTRAP_ADMIN_PASSWORD` | First admin account, created on startup |
| `VITE_API_BASE_URL` | API base URL used by the frontend (defaults to `http://localhost:8080/api/v1`) |

> **Warning:** Never commit `.env` files. Make sure `.env` is listed in your `.gitignore` and only commit `.env.example` (with placeholder values).

---

## 📂 Project Structure

<details>
<summary><b>Click to expand</b></summary>

```text
erpmate/
├── backend/                     # Go backend
│   ├── cmd/api/                 # Application entry point
│   ├── config/                  # Environment configuration
│   ├── internal/
│   │   ├── common/              # Shared helpers (response envelope, notifications)
│   │   ├── database/            # DB connection
│   │   ├── middleware/          # Auth and role middleware
│   │   └── modules/             # Feature modules
│   │       ├── procurement/     # POs and goods receipts
│   │       ├── inventory/       # Stock, batches, movements, valuation, alerts
│   │       └── …                # purchase, material, production, sales, master
│   ├── migrations/              # Numbered SQL migrations + runner
│   └── .env.example
│
├── frontend/                    # React frontend
│   ├── src/
│   │   ├── api/                 # Typed API clients
│   │   ├── modules/             # Feature UI (purchase, inventory, master, …)
│   │   ├── shared/              # Reusable components (icons, modals)
│   │   ├── store/               # Global state (auth)
│   │   ├── App.tsx              # Routing and layout
│   │   └── main.tsx             # Entry point
│   └── package.json
│
├── screenshots/                 # README images
└── README.md
```

</details>

---

## 🔌 API Overview

All endpoints require authentication. Lists return `{ "data": [...], "meta": { "page", "limit", "total" } }` and support `?page=` and `?limit=` (default 20, max 100). Paths are relative to each module's mount point.

<details>
<summary><b>Procurement</b></summary>

| Method | Path | Roles |
| --- | --- | --- |
| `GET` | `/purchase-orders` — filters: `status`, `supplier_id`, `search` | purchasing, finance, admin |
| `POST` | `/purchase-orders/from-requisition/{prId}` | purchasing, admin |
| `GET` | `/purchase-orders/{id}` | purchasing, finance, admin |
| `PUT` / `PATCH` | `/purchase-orders/{id}` (draft only, partial update) | purchasing, admin |
| `POST` | `/purchase-orders/{id}/issue` | purchasing, admin |
| `GET` | `/goods-receipts` — filter: `purchase_order_id` | store_keeper, purchasing, finance, admin |
| `POST` | `/goods-receipts` | store_keeper, admin |
| `GET` | `/goods-receipts/{id}` | store_keeper, purchasing, finance, admin |

</details>

<details>
<summary><b>Inventory</b> (read-only)</summary>

| Path | Purpose |
| --- | --- |
| `GET /lookups` | Filter values for dropdowns |
| `GET /stock`, `/stock/summary` | Stock table and status counts |
| `GET /stock/{itemID}`, `/stock/{itemID}/batches` | Item detail and FEFO batches |
| `GET /movements`, `/movements/types` | Stock ledger |
| `GET /valuation`, `/valuation/summary` | Inventory valuation |
| `GET /alerts`, `/alerts/summary` | Active low-stock and expiry alerts |

</details>

### Error format

| HTTP | Code | Meaning |
| --- | --- | --- |
| 400 | `VALIDATION_ERROR` | Bad input; `details` holds per-field messages |
| 401 / 403 | `UNAUTHORIZED` / `FORBIDDEN` | Not signed in / role not allowed |
| 404 | `NOT_FOUND` | Record does not exist |
| 409 | `CONFLICT` | Wrong state, e.g. editing an issued PO or over-receiving |
| 500 | `INTERNAL_ERROR` | Generic message; details are only in server logs |

---

## 🛠️ Super administration

Sign in as `admin` and open **Administration → Super admin**. It is a Django-style
site: a registry of every model in the system on the left, the records on the
right.

| | |
| --- | --- |
| **Models** | 23, grouped into Master data, Access, Procurement, Production & materials, Sales, Inventory and System |
| **Master data** | Items, categories, suppliers, customers, warehouses and departments can be created, changed and deleted |
| **Access** | Users can be created and changed, including resetting a password |
| **Everything else** | Workflow documents, stock and the audit trail are read-only, so the approval trail stays intact |

Deleting a record is a soft delete — the row keeps its `deleted_at`, so nothing
disappears from the audit trail.

The site is driven by a model registry rather than a hand-written page per
entity. Each model declares its table, its list columns and its form fields, and
the same registry is served to the frontend at `GET /admin/models`, so the UI
renders exactly the fields the backend accepts. Adding a model means adding one
entry to `backend/internal/modules/admin/registry.go` — no frontend change and
no new endpoint.

| Method | Path | Purpose |
| --- | --- | --- |
| `GET` | `/admin/models` | The model registry |
| `GET` | `/admin/lookups` | Reference lists used by the forms |
| `GET` | `/admin/overview` | Record count for every model |
| `GET` | `/admin/{model}` | List records — supports `?page=`, `?limit=`, `?search=` |
| `GET` | `/admin/{model}/{id}` | One record with every form field |
| `POST` | `/admin/{model}` | Create a record |
| `PUT` | `/admin/{model}/{id}` | Update a record |
| `DELETE` | `/admin/{model}/{id}` | Delete (soft delete where the table has `deleted_at`) |

Every route requires the `admin` role; anything else gets `403`.

---

## 🧰 Development

```bash
# Backend
cd backend
go build ./... && go vet ./...

# Frontend
cd frontend
npm run build        # type-checks and builds
```

### Database migrations

Migrations live in `backend/migrations/` as `NNNN_name.up.sql` files and are applied **in order on startup** by the project's own runner. Applied files are recorded by filename in the `schema_migrations` table.

- Use the **next free number** for a new migration. Two files with the same number cause problems.
- Don't edit a migration that has already been applied; add a new one instead.

### Demo data

A fresh database has no master data, so every screen is empty. `cmd/seed-demo` fills it with a realistic, end-to-end scenario: master data, one user per role, and the full procure-to-pay chain (requisition → approval → purchase order → goods receipt → material issue → production receipt → sales order → delivery note).

It talks to the running API rather than writing to PostgreSQL directly, so every document goes through the real service layer and stock movements, batches, reservations, status history and notifications are all created the way the app creates them. Each step is performed by the role that is allowed to perform it.

```bash
# The API must be running first
cd backend
go run ./cmd/seed-demo -reset -password "your-admin-password"
```

| Flag | Default | Purpose |
| --- | --- | --- |
| `-reset` | off | Truncate all transactional tables first, so the seed always starts from a known state |
| `-api` | `http://localhost:8080/api/v1` | Base URL of the running API |
| `-email` | `BOOTSTRAP_ADMIN_EMAIL` | Administrator email to sign in with |
| `-password` | `BOOTSTRAP_ADMIN_PASSWORD` | Administrator password |

The seed creates these accounts, all with the password `DemoPass12345!`:

| Email | Role |
| --- | --- |
| `merina.adam@example.test` | Department Head |
| `abel.tadesse@example.test` | Production Manager |
| `sara.girma@example.test` | Finance |
| `yonas.bekele@example.test` | Purchasing |
| `marta.tesfaye@example.test` | Store Keeper |
| `dawit.alemu@example.test` | Production Store Keeper |
| `hana.mulugeta@example.test` | Sales |
| `kalkidan.fitsum@example.test` | Sales Head |

> The alert worker scans every 15 minutes. After seeding, restart the API so it scans immediately and raises the low-stock and expiry alerts.

See [`backend/cmd/seed-demo/README.md`](backend/cmd/seed-demo/README.md) for how to extend the seeder with your own demo data.

---

## 🩺 Troubleshooting

<details>
<summary><b>migrate CLI fails with column "dirty" does not exist</b></summary>

The project uses its own migration runner, not golang-migrate, so the `migrate` CLI can't read its `schema_migrations` table. Don't use the CLI — just start the backend (`go run ./cmd/api`) and migrations are applied for you.

</details>

<details>
<summary><b>A new migration was not applied</b></summary>

Check that the file name ends in `.up.sql`, that its number is higher than the existing ones, and that it isn't a duplicate. Then restart the backend and inspect:

```sql
SELECT * FROM schema_migrations ORDER BY version DESC;
```

</details>

<details>
<summary><b>The app can't reach the API</b></summary>

Make sure the backend is running on port 8080 and that `VITE_API_BASE_URL` in `frontend/.env` matches. Restart `npm run dev` after changing `.env`.

</details>

<details>
<summary><b>Cannot sign in</b></summary>

The bootstrap admin is created from `BOOTSTRAP_ADMIN_EMAIL` / `BOOTSTRAP_ADMIN_PASSWORD` the first time the backend starts. If you changed them afterwards, they won't update an existing user.

</details>

---

## 🤝 Contributing

1. **Fork** the project
2. Create a feature branch: `git checkout -b feature/AmazingFeature`
3. Commit your changes: `git commit -m "Add some AmazingFeature"`
4. Push to the branch: `git push origin feature/AmazingFeature`
5. Open a **Pull Request**

---

## 📄 License

This project is open source and available under the **MIT License**.

<div align="center">

⭐ If you find this project useful, consider giving it a star.

</div>
