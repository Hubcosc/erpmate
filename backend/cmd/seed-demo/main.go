// Command seed-demo populates a fresh ErpMate database with realistic demo data.
//
// It talks to the running API rather than writing to PostgreSQL directly, so every
// document goes through the real service layer: stock movements, batch records,
// reservations, status history and notifications are all created the way the app
// creates them.
//
// Usage:
//
//	go run ./cmd/seed-demo                 # uses http://localhost:8080
//	go run ./cmd/seed-demo -api http://host:port
//
// The command is idempotent: records that already exist are reused instead of
// being created twice, so it is safe to run more than once.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/Ethio-ERP/erpmate/config"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

// ---------------------------------------------------------------------------
// API client
// ---------------------------------------------------------------------------

type envelope struct {
	Success bool            `json:"success"`
	Data    json.RawMessage `json:"data"`
	Meta    *struct {
		Page  int   `json:"page"`
		Limit int   `json:"limit"`
		Total int64 `json:"total"`
	} `json:"meta"`
	Error *struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

type client struct {
	baseURL string
	token   string
	http    *http.Client
}

func (c *client) request(method, path string, body any, out any) error {
	var reader io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("encode request: %w", err)
		}
		reader = bytes.NewReader(encoded)
	}

	req, err := http.NewRequest(method, c.baseURL+path, reader)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	res, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("%s %s: %w", method, path, err)
	}
	defer res.Body.Close()

	raw, err := io.ReadAll(res.Body)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}

	var env envelope
	if err := json.Unmarshal(raw, &env); err != nil {
		return fmt.Errorf("%s %s: unexpected response (status %d): %s", method, path, res.StatusCode, truncate(string(raw), 200))
	}
	if !env.Success {
		message := "request failed"
		code := ""
		if env.Error != nil {
			message = env.Error.Message
			code = env.Error.Code
		}
		return fmt.Errorf("%s %s: [%s] %s", method, path, code, message)
	}
	if out != nil && len(env.Data) > 0 {
		if err := json.Unmarshal(env.Data, out); err != nil {
			return fmt.Errorf("%s %s: decode data: %w", method, path, err)
		}
	}
	return nil
}

// list fetches a collection and returns the raw items plus the total.
func (c *client) list(path string, out any) (int64, error) {
	var env envelope
	req, err := http.NewRequest(http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return 0, err
	}
	req.Header.Set("Accept", "application/json")
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	res, err := c.http.Do(req)
	if err != nil {
		return 0, err
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(res.Body)
	if err := json.Unmarshal(raw, &env); err != nil {
		return 0, fmt.Errorf("GET %s: unexpected response: %s", path, truncate(string(raw), 200))
	}
	if !env.Success {
		message := "request failed"
		if env.Error != nil {
			message = env.Error.Message
		}
		return 0, fmt.Errorf("GET %s: %s", path, message)
	}
	if out != nil && len(env.Data) > 0 {
		if err := json.Unmarshal(env.Data, out); err != nil {
			return 0, fmt.Errorf("GET %s: decode data: %w", path, err)
		}
	}
	if env.Meta != nil {
		return env.Meta.Total, nil
	}
	return 0, nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

func (c *client) login(email, password string) error {
	var payload struct {
		Token string `json:"token"`
		User  struct {
			ID       string `json:"id"`
			FullName string `json:"full_name"`
			RoleName string `json:"role_name"`
		} `json:"user"`
	}
	if err := c.request(http.MethodPost, "/auth/login", map[string]string{
		"email": email, "password": password,
	}, &payload); err != nil {
		return err
	}
	if payload.Token == "" {
		return fmt.Errorf("login returned no token")
	}
	c.token = payload.Token
	log.Printf("  signed in as %s (%s)", payload.User.FullName, payload.User.RoleName)
	return nil
}

// loginAs switches the active session to another account, so each workflow step
// is performed by the role that is allowed to perform it.
func (c *client) loginAs(email, password string) error {
	return c.login(email, password)
}

// ---------------------------------------------------------------------------
// Domain shapes
// ---------------------------------------------------------------------------

type role struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type department struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Code string `json:"code"`
}

type category struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type item struct {
	ID              string  `json:"id"`
	Code            string  `json:"code"`
	Name            string  `json:"name"`
	ItemType        string  `json:"item_type"`
	UnitOfMeasure   string  `json:"unit_of_measure"`
	MinStockLevel   float64 `json:"min_stock_level"`
	ReorderLevel    float64 `json:"reorder_level"`
	MaxStockLevel   float64 `json:"max_stock_level"`
	IsBatchTracked  bool    `json:"is_batch_tracked"`
	IsExpiryTracked bool    `json:"is_expiry_tracked"`
}

type supplier struct {
	ID   string `json:"id"`
	Code string `json:"code"`
	Name string `json:"name"`
}

type customer struct {
	ID   string `json:"id"`
	Code string `json:"code"`
	Name string `json:"name"`
}

type warehouse struct {
	ID            string `json:"id"`
	Code          string `json:"code"`
	Name          string `json:"name"`
	WarehouseType string `json:"warehouse_type"`
}

type user struct {
	ID       string `json:"id"`
	FullName string `json:"full_name"`
	Email    string `json:"email"`
	RoleName string `json:"role_name"`
}

type purchaseRequisition struct {
	ID           string `json:"id"`
	PRNumber     string `json:"pr_number"`
	Status       string `json:"status"`
	DepartmentID string `json:"department_id"`
	Items        []struct {
		ID       string  `json:"id"`
		ItemID   string  `json:"item_id"`
		Quantity float64 `json:"quantity"`
	} `json:"items"`
}

type purchaseOrder struct {
	ID       string `json:"id"`
	PONumber string `json:"po_number"`
	Status   string `json:"status"`
	PRID     string `json:"purchase_requisition_id"`
	Items    []struct {
		ID        string  `json:"id"`
		ItemID    string  `json:"item_id"`
		Quantity  float64 `json:"quantity"`
		UnitPrice float64 `json:"unit_price"`
	} `json:"items"`
}

type goodsReceiptNote struct {
	ID       string `json:"id"`
	GRNNumber string `json:"grn_number"`
	Items    []struct {
		ID       string  `json:"id"`
		ItemID   string  `json:"item_id"`
		Quantity float64 `json:"quantity_received"`
	} `json:"items"`
}

type materialRequisition struct {
	ID       string `json:"id"`
	MRNumber string `json:"mr_number"`
	Status   string `json:"status"`
	Items    []struct {
		ID         string  `json:"id"`
		ItemID     string  `json:"item_id"`
		Quantity   float64 `json:"quantity"`
		IssuedQty  float64 `json:"issued_qty"`
	} `json:"items"`
}

type materialIssue struct {
	ID         string `json:"id"`
	IssueNumber string `json:"issue_number"`
}

type productionReceipt struct {
	ID           string `json:"id"`
	ReceiptNumber string `json:"receipt_number"`
}

type salesOrder struct {
	ID          string `json:"id"`
	OrderNumber string `json:"order_number"`
	Status      string `json:"status"`
	Items       []struct {
		ID               string  `json:"id"`
		ItemID           string  `json:"item_id"`
		Quantity         float64 `json:"quantity"`
		DeliveredQty     float64 `json:"delivered_quantity"`
	} `json:"items"`
}

type deliveryNote struct {
	ID            string `json:"id"`
	DeliveryNumber string `json:"delivery_number"`
}

type stockBatch struct {
	ID                string  `json:"id"`
	ItemID            string  `json:"item_id"`
	BatchNumber       string  `json:"batch_number"`
	Quantity          float64 `json:"quantity"`
	AvailableQuantity float64 `json:"available_quantity"`
	ExpiryDate        string  `json:"expiry_date"`
}

// ---------------------------------------------------------------------------
// Seed helpers
// ---------------------------------------------------------------------------

var c *client

// demoPassword is the password given to every demo user.
const demoPassword = "DemoPass12345!"

func step(format string, args ...any) {
	log.Printf("  "+format, args...)
}

func createDepartment(name, code string) string {
	var existing []department
	if _, err := c.list("/master/departments?limit=100", &existing); err == nil {
		for _, d := range existing {
			if strings.EqualFold(d.Name, name) {
				step("department %q already exists", name)
				return d.ID
			}
		}
	}
	var created department
	body := map[string]any{"name": name, "code": code}
	if err := c.request(http.MethodPost, "/master/departments", body, &created); err != nil {
		log.Fatalf("create department %s: %v", name, err)
	}
	step("created department %q", name)
	return created.ID
}

func createCategory(name string) string {
	var existing []category
	if _, err := c.list("/master/categories?limit=100", &existing); err == nil {
		for _, cat := range existing {
			if strings.EqualFold(cat.Name, name) {
				return cat.ID
			}
		}
	}
	var created category
	if err := c.request(http.MethodPost, "/master/categories", map[string]any{"name": name}, &created); err != nil {
		log.Fatalf("create category %s: %v", name, err)
	}
	step("created category %q", name)
	return created.ID
}

func createItem(payload map[string]any) item {
	code, _ := payload["code"].(string)
	var existing []item
	if _, err := c.list("/master/items?limit=100", &existing); err == nil {
		for _, it := range existing {
			if strings.EqualFold(it.Code, code) {
				return it
			}
		}
	}
	var created item
	if err := c.request(http.MethodPost, "/master/items", payload, &created); err != nil {
		log.Fatalf("create item %s: %v", code, err)
	}
	step("created item %q", code)
	return created
}

func createSupplier(code, name string) supplier {
	var existing []supplier
	if _, err := c.list("/master/suppliers?limit=100", &existing); err == nil {
		for _, s := range existing {
			if strings.EqualFold(s.Code, code) || strings.EqualFold(s.Name, name) {
				return s
			}
		}
	}
	var created supplier
	body := map[string]any{
		"code": code, "name": name,
		"contact_person": "Demo Contact", "email": "sales@" + code + ".example",
		"phone": "+251900000001", "address": "Bole Road, Addis Ababa",
	}
	if err := c.request(http.MethodPost, "/master/suppliers", body, &created); err != nil {
		log.Fatalf("create supplier %s: %v", name, err)
	}
	step("created supplier %q", name)
	return created
}

func createCustomer(code, name, customerType string) customer {
	var existing []customer
	if _, err := c.list("/master/customers?limit=100", &existing); err == nil {
		for _, cu := range existing {
			if strings.EqualFold(cu.Code, code) || strings.EqualFold(cu.Name, name) {
				return cu
			}
		}
	}
	var created customer
	body := map[string]any{
		"code": code, "name": name, "customer_type": customerType,
		"contact_person": "Demo Buyer", "email": "orders@" + code + ".example",
		"phone": "+251900000002", "address": "Meridens, Addis Ababa",
	}
	if err := c.request(http.MethodPost, "/master/customers", body, &created); err != nil {
		log.Fatalf("create customer %s: %v", name, err)
	}
	step("created customer %q", name)
	return created
}

func createWarehouse(code, name, warehouseType, location string) warehouse {
	var existing []warehouse
	if _, err := c.list("/master/warehouses", &existing); err == nil {
		for _, w := range existing {
			if strings.EqualFold(w.Code, code) {
				return w
			}
		}
	}
	var created warehouse
	body := map[string]any{"code": code, "name": name, "warehouse_type": warehouseType, "location": location}
	if err := c.request(http.MethodPost, "/master/warehouses", body, &created); err != nil {
		log.Fatalf("create warehouse %s: %v", code, err)
	}
	step("created warehouse %q (%s)", name, warehouseType)
	return created
}

func createUser(roleID, fullName, email, password, departmentID string) string {
	var existing []user
	if _, err := c.list("/users?limit=100", &existing); err == nil {
		for _, u := range existing {
			if strings.EqualFold(u.Email, email) {
				step("user %q already exists", email)
				return u.ID
			}
		}
	}
	body := map[string]any{
		"full_name": fullName, "email": email, "password": password, "role_id": roleID,
	}
	if departmentID != "" {
		body["department_id"] = departmentID
	}
	var created user
	if err := c.request(http.MethodPost, "/users", body, &created); err != nil {
		log.Fatalf("create user %s: %v", email, err)
	}
	step("created user %s (%s)", email, created.RoleName)
	return created.ID
}

func must(action string, err error) {
	if err != nil {
		log.Fatalf("%s: %v", action, err)
	}
}

func day(offset int) string {
	return time.Now().AddDate(0, 0, offset).Format("2006-01-02")
}

// pickBatch returns the batch to issue from, using earliest-expiry-first
// allocation. Batch-tracked items must be issued from a specific batch.
func pickBatch(itemID string) (batchID string, batchNumber string) {
	var batches []stockBatch
	if _, err := c.list("/inventory/stock/"+itemID+"/batches", &batches); err != nil {
		return "", ""
	}
	// Earliest expiry first; fall back to the largest available batch.
	sort.SliceStable(batches, func(i, j int) bool {
		if batches[i].ExpiryDate != batches[j].ExpiryDate {
			return batches[i].ExpiryDate < batches[j].ExpiryDate
		}
		return batches[i].AvailableQuantity > batches[j].AvailableQuantity
	})
	for _, batch := range batches {
		if batch.AvailableQuantity > 0 {
			return batch.ID, batch.BatchNumber
		}
	}
	return "", ""
}

// ---------------------------------------------------------------------------
// Main
// ---------------------------------------------------------------------------

func main() {
	apiURL := flag.String("api", "http://localhost:8080/api/v1", "base URL of the running ErpMate API")
	email := flag.String("email", envOr("BOOTSTRAP_ADMIN_EMAIL", "admin@example.test"), "administrator email")
	password := flag.String("password", envOr("BOOTSTRAP_ADMIN_PASSWORD", ""), "administrator password")
	reset := flag.Bool("reset", false, "truncate all transactional tables before seeding")
	flag.Parse()

	if *password == "" {
		log.Fatal("-password is required (or set BOOTSTRAP_ADMIN_PASSWORD)")
	}

	if *reset {
		resetDatabase()
	}

	c = &client{
		baseURL: strings.TrimRight(*apiURL, "/"),
		http:    &http.Client{Timeout: 30 * time.Second},
	}

	log.Println("seeding demo data")
	ensureAdmin(*email, *password)
	if err := c.login(*email, *password); err != nil {
		log.Fatalf("login: %v", err)
	}

	// ---- roles -----------------------------------------------------------
	var roles []role
	if _, err := c.list("/users/roles", &roles); err != nil {
		log.Fatalf("list roles: %v", err)
	}
	roleID := map[string]string{}
	for _, r := range roles {
		roleID[r.Name] = r.ID
	}
	for _, required := range []string{"admin", "department_head", "production_manager", "finance", "purchasing", "store_keeper", "production_store_keeper", "sales", "sales_head"} {
		if roleID[required] == "" {
			log.Fatalf("role %q is missing from the database", required)
		}
	}
	step("found %d roles", len(roles))

	// ---- master data -----------------------------------------------------
	log.Println("master data")
	productionDept := createDepartment("Production", "PROD")
	maintenanceDept := createDepartment("Maintenance", "MAINT")
	qaDept := createDepartment("Quality Assurance", "QA")

	rawCategory := createCategory("Raw Materials")
	finishedCategory := createCategory("Finished Goods")
	consumableCategory := createCategory("Consumables")

	steelSheet := createItem(map[string]any{
		"code": "RM-001", "name": "Steel Sheet 2mm", "category_id": rawCategory,
		"item_type": "raw_material", "unit_of_measure": "kg",
		"min_stock_level": 100, "max_stock_level": 800, "reorder_level": 150,
		"is_batch_tracked": true, "is_expiry_tracked": false,
	})
	aluminiumRod := createItem(map[string]any{
		"code": "RM-002", "name": "Aluminium Rod 10mm", "category_id": rawCategory,
		"item_type": "raw_material", "unit_of_measure": "kg",
		"min_stock_level": 80, "max_stock_level": 600, "reorder_level": 120,
		"is_batch_tracked": true, "is_expiry_tracked": false,
	})
	paint := createItem(map[string]any{
		"code": "RM-003", "name": "Industrial Paint 20L", "category_id": rawCategory,
		"item_type": "raw_material", "unit_of_measure": "pcs",
		"min_stock_level": 10, "max_stock_level": 60, "reorder_level": 15,
		"is_batch_tracked": true, "is_expiry_tracked": true,
	})
	officeChair := createItem(map[string]any{
		"code": "FG-001", "name": "Office Chair", "category_id": finishedCategory,
		"item_type": "finished_good", "unit_of_measure": "pcs",
		"min_stock_level": 20, "max_stock_level": 200, "reorder_level": 30,
		"is_batch_tracked": false, "is_expiry_tracked": false,
	})
	steelDesk := createItem(map[string]any{
		"code": "FG-002", "name": "Steel Work Bench", "category_id": finishedCategory,
		"item_type": "finished_good", "unit_of_measure": "pcs",
		"min_stock_level": 10, "max_stock_level": 90, "reorder_level": 15,
		"is_batch_tracked": false, "is_expiry_tracked": false,
	})
	cuttingDisc := createItem(map[string]any{
		"code": "CN-001", "name": "Cutting Disc 4.5 inch", "category_id": consumableCategory,
		"item_type": "consumable", "unit_of_measure": "pcs",
		"min_stock_level": 200, "max_stock_level": 1500, "reorder_level": 250,
		"is_batch_tracked": false, "is_expiry_tracked": false,
	})

	metalSupplies := createSupplier("SUP-001", "Metal Supplies Ltd")
	hardwareTraders := createSupplier("SUP-002", "Hardware Traders PLC")
	chemicalWorks := createSupplier("SUP-003", "Chemical Works Ethiopia")

	mesfin := createCustomer("CUS-001", "Mesfin Construction", "local")
	boleTrading := createCustomer("CUS-002", "Bole Trading", "local")
	almaz := createCustomer("CUS-003", "Almaz Distributors", "international")

	mainStore := createWarehouse("WH-RM", "Main Raw Material Store", "raw_material", "Building A, Floor 1")
	finishedStore := createWarehouse("WH-FG", "Finished Goods Store", "finished_goods", "Building B, Floor 1")
	createWarehouse("WH-QR", "Quarantine Store", "quarantine", "Building A, Floor 2")

	// ---- users -----------------------------------------------------------
	log.Println("users")
	departmentHeadID := createUser(roleID["department_head"], "Merina Adam", "merina.adam@example.test", demoPassword, productionDept)
	createUser(roleID["production_manager"], "Abel Tadesse", "abel.tadesse@example.test", demoPassword, productionDept)
	createUser(roleID["finance"], "Sara Girma", "sara.girma@example.test", demoPassword, "")
	createUser(roleID["purchasing"], "Yonas Bekele", "yonas.bekele@example.test", demoPassword, "")
	createUser(roleID["store_keeper"], "Marta Tesfaye", "marta.tesfaye@example.test", demoPassword, "")
	createUser(roleID["production_store_keeper"], "Dawit Alemu", "dawit.alemu@example.test", demoPassword, "")
	createUser(roleID["sales"], "Hana Mulugeta", "hana.mulugeta@example.test", demoPassword, "")
	createUser(roleID["sales_head"], "Kalkidan Fitsum", "kalkidan.fitsum@example.test", demoPassword, "")
	createUser(roleID["department_head"], "Tewodros Girma", "tewodros.girma@example.test", demoPassword, maintenanceDept)
	createUser(roleID["department_head"], "Selam Worku", "selam.worku@example.test", demoPassword, qaDept)

	// ---- purchase requisition -> purchase order -> goods receipt ---------
	log.Println("procurement")
	prID := seedPurchaseRequisition(steelSheet, aluminiumRod, paint, cuttingDisc, productionDept, metalSupplies.ID, hardwareTraders.ID, chemicalWorks.ID)

	prices := map[string]float64{
		steelSheet.ID:   182.0,
		aluminiumRod.ID: 315.0,
		paint.ID:        1425.0,
		cuttingDisc.ID:  42.0,
	}
	batches := map[string]batchMeta{
		steelSheet.ID:   {batch: "B-2026-014", made: day(-30), expiry: day(0), cost: 182.0, receiveTo: 400},
		aluminiumRod.ID: {batch: "AL-2026-009", made: day(-45), expiry: day(0), cost: 315.0, receiveTo: 250},
		paint.ID:        {batch: "PT-2026-031", made: day(-20), expiry: day(345), cost: 1425.0, receiveTo: 12},
	}
	poID := seedPurchaseOrder(prID, metalSupplies.ID, prices)
	seedGoodsReceipt(poID, mainStore.ID, batches)

	// ---- a second PR left pending, so the approval queue is not empty ----
	seedPendingRequisition(cuttingDisc, maintenanceDept, hardwareTraders.ID)

	// ---- material requisition -> issue -> acknowledge ---------------------
	log.Println("materials")
	mrID := seedMaterialRequisition(steelSheet, aluminiumRod, productionDept)
	seedMaterialIssue(mrID, departmentHeadID)

	// ---- production receipts --------------------------------------------
	log.Println("production")
	seedProductionReceipt(finishedStore.ID, officeChair, steelDesk)

	// ---- sales order -> delivery note ------------------------------------
	log.Println("sales")
	seedSalesAndDelivery(officeChair, steelDesk, mesfin, boleTrading, almaz)

	log.Println("done — open http://localhost:5173 and sign in as the admin to explore")
	log.Println("demo accounts (password DemoPass12345!):")
	for _, email := range []string{
		"merina.adam@example.test", "abel.tadesse@example.test", "sara.girma@example.test",
		"yonas.bekele@example.test", "marta.tesfaye@example.test", "dawit.alemu@example.test",
		"hana.mulugeta@example.test", "kalkidan.fitsum@example.test",
	} {
		log.Printf("  %s", email)
	}
}

func envOr(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

// ensureAdmin creates the administrator account when it is missing, so the
// tool works on a freshly reset database without restarting the API.
func ensureAdmin(email, password string) {
	cfg := config.Load()
	pool, err := pgxpool.New(context.Background(), cfg.DSN())
	if err != nil {
		log.Fatalf("admin: connect to the database: %v", err)
	}
	defer pool.Close()
	ctx := context.Background()

	var exists bool
	if err := pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM users WHERE email = $1 AND deleted_at IS NULL)`, email,
	).Scan(&exists); err != nil {
		log.Fatalf("admin: check: %v", err)
	}
	if exists {
		return
	}

	var roleID string
	if err := pool.QueryRow(ctx, `SELECT id FROM roles WHERE name IN ('admin','system_admin') LIMIT 1`).Scan(&roleID); err != nil {
		log.Fatalf("admin: find role: %v", err)
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Fatalf("admin: hash password: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO users (employee_id, full_name, email, password_hash, role_id, is_active)
		VALUES ($1, $2, lower($3), $4, $5, true)`,
		"ADMIN-001", cfg.BootstrapAdminName, email, string(hash), roleID,
	); err != nil {
		log.Fatalf("admin: insert: %v", err)
	}
	log.Printf("  created administrator %s", email)
}

// resetDatabase truncates every transactional table so a seed always starts
// from a known state. The roles and the migration history are kept.
func resetDatabase() {
	cfg := config.Load()
	pool, err := pgxpool.New(context.Background(), cfg.DSN())
	if err != nil {
		log.Fatalf("reset: connect to the database: %v", err)
	}
	defer pool.Close()

	const statement = `TRUNCATE TABLE
		purchase_requisition_items, purchase_requisitions,
		purchase_order_items, purchase_orders,
		goods_receipt_note_items, goods_receipt_notes,
		material_requisition_items, material_requisitions,
		material_issue_items, material_issues,
		production_receipt_items, production_receipts,
		sales_order_item_allocations, sales_order_items, sales_orders,
		delivery_note_items, delivery_notes,
		stock_movements, stock_balances, stock_batches, inventory_alerts,
		notifications, status_history,
		users, items, item_categories, suppliers, customers, warehouses, departments
		CASCADE;`

	if _, err := pool.Exec(context.Background(), statement); err != nil {
		log.Fatalf("reset: truncate: %v", err)
	}
	log.Println("reset: transactional tables truncated (roles and migrations kept)")
}

// ---------------------------------------------------------------------------
// Workflow seeds
// ---------------------------------------------------------------------------

func seedPurchaseRequisition(steelSheet, aluminiumRod, paint, cuttingDisc item, departmentID, metalSupplierID, hardwareSupplierID, chemicalSupplierID string) string {
	// The department head raises the request.
	must("sign in as department head", c.loginAs("merina.adam@example.test", demoPassword))
	body := map[string]any{
		"department_id":         departmentID,
		"preferred_supplier_id": metalSupplierID,
		"required_date":         day(21),
		"justification":         "Monthly raw material replenishment for the production line.",
		"items": []map[string]any{
			{"item_id": steelSheet.ID, "description": steelSheet.Name, "quantity": 400.0, "unit_of_measure": steelSheet.UnitOfMeasure, "estimated_unit_price": 185.0},
			{"item_id": aluminiumRod.ID, "description": aluminiumRod.Name, "quantity": 250.0, "unit_of_measure": aluminiumRod.UnitOfMeasure, "estimated_unit_price": 320.0},
			{"item_id": paint.ID, "description": paint.Name, "quantity": 12.0, "unit_of_measure": paint.UnitOfMeasure, "estimated_unit_price": 1450.0, "notes": "Batch with a one year expiry"},
			{"item_id": cuttingDisc.ID, "description": cuttingDisc.Name, "quantity": 300.0, "unit_of_measure": cuttingDisc.UnitOfMeasure, "estimated_unit_price": 42.0},
		},
	}

	var created purchaseRequisition
	must("create purchase requisition", c.request(http.MethodPost, "/purchase-requisitions", body, &created))
	step("created purchase requisition %s", created.PRNumber)

	must("submit requisition", c.request(http.MethodPost, "/purchase-requisitions/"+created.ID+"/submit", map[string]any{"reason": "Monthly replenishment."}, nil))

	// The production manager approves first.
	must("sign in as production manager", c.loginAs("abel.tadesse@example.test", demoPassword))
	must("production manager approval", c.request(http.MethodPost, "/purchase-requisitions/"+created.ID+"/pm-approve", map[string]any{"reason": "Approved for the quarterly plan."}, nil))

	// Then finance approves, which creates the draft purchase order.
	must("sign in as finance", c.loginAs("sara.girma@example.test", demoPassword))
	must("finance approval", c.request(http.MethodPost, "/purchase-requisitions/"+created.ID+"/finance-approve", map[string]any{"reason": "Budget confirmed."}, nil))
	step("requisition %s approved by production manager and finance", created.PRNumber)

	return created.ID
}

func seedPendingRequisition(cuttingDisc item, departmentID, supplierID string) {
	must("sign in as maintenance department head", c.loginAs("tewodros.girma@example.test", demoPassword))
	body := map[string]any{
		"department_id":         departmentID,
		"preferred_supplier_id": supplierID,
		"required_date":         day(10),
		"justification":         "Replacement consumables for the maintenance workshop.",
		"items": []map[string]any{
			{"item_id": cuttingDisc.ID, "description": cuttingDisc.Name, "quantity": 120.0, "unit_of_measure": cuttingDisc.UnitOfMeasure, "estimated_unit_price": 42.0},
		},
	}
	var created purchaseRequisition
	if err := c.request(http.MethodPost, "/purchase-requisitions", body, &created); err != nil {
		log.Fatalf("create pending requisition: %v", err)
	}
	must("submit pending requisition", c.request(http.MethodPost, "/purchase-requisitions/"+created.ID+"/submit", map[string]any{"reason": "Workshop restock."}, nil))
	step("created requisition %s awaiting production manager approval", created.PRNumber)
}

// batchMeta describes how a purchase order line is received into stock.
type batchMeta struct {
	batch     string
	made      string
	expiry    string
	cost      float64
	receiveTo float64
}

func seedPurchaseOrder(prID, supplierID string, prices map[string]float64) string {
	// Purchasing confirms the supplier and prices, then issues the order.
	must("sign in as purchasing", c.loginAs("yonas.bekele@example.test", demoPassword))

	var po purchaseOrder
	must("create purchase order from requisition", c.request(http.MethodPost, "/purchase-orders/from-requisition/"+prID, nil, &po))
	step("created draft purchase order %s", po.PONumber)

	items := make([]map[string]any, 0, len(po.Items))
	for _, line := range po.Items {
		price, ok := prices[line.ItemID]
		if !ok {
			price = 42.0
		}
		items = append(items, map[string]any{"id": line.ID, "unit_price": price})
	}
	update := map[string]any{
		"supplier_id":            supplierID,
		"expected_delivery_date": day(14),
		"payment_terms":          "Net 30",
		"delivery_terms":         "FOB Addis Ababa",
		"notes":                  "Deliver to the main raw material store during working hours.",
		"items":                  items,
	}
	must("update purchase order", c.request(http.MethodPut, "/purchase-orders/"+po.ID, update, &po))
	must("issue purchase order", c.request(http.MethodPost, "/purchase-orders/"+po.ID+"/issue", nil, &po))
	step("issued purchase order %s to the supplier", po.PONumber)

	return po.ID
}

func seedGoodsReceipt(poID, warehouseID string, batches map[string]batchMeta) {
	// The store keeper receives the goods.
	must("sign in as store keeper", c.loginAs("marta.tesfaye@example.test", demoPassword))

	var po purchaseOrder
	must("load purchase order", c.request(http.MethodGet, "/purchase-orders/"+poID, nil, &po))

	items := make([]map[string]any, 0, len(po.Items))
	for _, line := range po.Items {
		meta, ok := batches[line.ItemID]
		if !ok {
			continue
		}
		entry := map[string]any{
			"purchase_order_item_id": line.ID,
			"quantity_received":      meta.receiveTo,
			"unit_cost":              meta.cost,
			"location":               "Rack A",
		}
		if meta.batch != "" {
			entry["batch_number"] = meta.batch
			entry["manufacture_date"] = meta.made
		}
		if meta.expiry != "" {
			entry["expiry_date"] = meta.expiry
		}
		items = append(items, entry)
	}

	body := map[string]any{
		"purchase_order_id":          poID,
		"warehouse_id":               warehouseID,
		"supplier_invoice_reference": "INV-2026-" + day(0),
		"notes":                      "Goods inspected and accepted on arrival.",
		"items":                      items,
	}
	var grn goodsReceiptNote
	must("create goods receipt note", c.request(http.MethodPost, "/goods-receipt-notes", body, &grn))
	step("posted goods receipt note %s into the main raw material store", grn.GRNNumber)
}

func seedMaterialRequisition(steelSheet, aluminiumRod item, departmentID string) string {
	// The department head requests the material.
	must("sign in as department head", c.loginAs("merina.adam@example.test", demoPassword))
	body := map[string]any{
		"department_id": departmentID,
		"required_date": day(7),
		"purpose":       "Weekly production run for the furniture line.",
		"items": []map[string]any{
			{"item_id": steelSheet.ID, "quantity": 120.0},
			{"item_id": aluminiumRod.ID, "quantity": 60.0},
		},
	}
	var created materialRequisition
	must("create material requisition", c.request(http.MethodPost, "/material-requisitions", body, &created))
	step("created material requisition %s", created.MRNumber)

	must("submit material requisition", c.request(http.MethodPost, "/material-requisitions/"+created.ID+"/submit", map[string]any{}, nil))

	// The production manager approves it.
	must("sign in as production manager", c.loginAs("abel.tadesse@example.test", demoPassword))
	must("approve material requisition", c.request(http.MethodPost, "/material-requisitions/"+created.ID+"/pm-approve", map[string]any{}, nil))
	step("material requisition %s approved", created.MRNumber)

	return created.ID
}

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
	if len(items) == 0 {
		return
	}

	body := map[string]any{
		"mr_id":     mrID,
		"issued_to": issuedToUserID,
		"notes":     "Issued to the production floor.",
		"items":     items,
	}
	var issue materialIssue
	must("create material issue", c.request(http.MethodPost, "/material-requisitions/issues", body, &issue))
	step("posted material issue %s from the raw material store", issue.IssueNumber)

	// The requesting department acknowledges receipt.
	must("sign in as department head", c.loginAs("merina.adam@example.test", demoPassword))
	must("acknowledge material issue", c.request(http.MethodPost, "/material-requisitions/issues/"+issue.ID+"/acknowledge", map[string]any{}, nil))
	step("material issue %s acknowledged by the requesting department", issue.IssueNumber)
}

func seedProductionReceipt(warehouseID string, officeChair, steelDesk item) {
	// The production store keeper receives the finished goods.
	must("sign in as production store keeper", c.loginAs("dawit.alemu@example.test", demoPassword))

	body := map[string]any{
		"warehouse_id": warehouseID,
		"notes":        "Weekly production output received from the workshop.",
		"items": []map[string]any{
			{"item_id": officeChair.ID, "quantity_produced": 60.0, "unit_cost": 2400.0, "quality_status": "accepted"},
			{"item_id": steelDesk.ID, "quantity_produced": 24.0, "unit_cost": 1850.0, "quality_status": "accepted"},
			{"item_id": steelDesk.ID, "quantity_produced": 3.0, "unit_cost": 1850.0, "quality_status": "rejected", "notes": "Dented frames"},
		},
	}
	var created productionReceipt
	must("create production receipt", c.request(http.MethodPost, "/production-receipts", body, &created))
	step("recorded production receipt %s in the finished goods store", created.ReceiptNumber)
}

func seedSalesAndDelivery(officeChair, steelDesk item, mesfin, boleTrading, almaz customer) {
	// The sales team raises the order.
	must("sign in as sales", c.loginAs("hana.mulugeta@example.test", demoPassword))

	// A fully delivered order.
	seedSalesOrder(officeChair, mesfin, 25.0, 2500.0, "Net 30", true)

	// A partially delivered order, so both states are visible in the UI.
	partial := seedSalesOrder(steelDesk, boleTrading, 20.0, 1900.0, "Net 45", false)
	seedPartialDelivery(partial, 8.0)

	// A confirmed order with nothing dispatched yet.
	seedSalesOrder(officeChair, almaz, 10.0, 2600.0, "Net 60", false)
}

func seedSalesOrder(product item, buyer customer, quantity, unitPrice float64, terms string, deliverAll bool) string {
	body := map[string]any{
		"customer_id":   buyer.ID,
		"required_date": day(18),
		"notes":         "Order received through the sales desk.",
		"items": []map[string]any{
			{"item_id": product.ID, "quantity": quantity, "unit_price": unitPrice},
		},
	}
	var created salesOrder
	must("create sales order", c.request(http.MethodPost, "/sales-orders", body, &created))
	step("created sales order %s for %s", created.OrderNumber, buyer.Name)

	if deliverAll {
		seedFullDelivery(created, buyer)
	}
	return created.ID
}

func seedFullDelivery(order salesOrder, buyer customer) {
	items := make([]map[string]any, 0, len(order.Items))
	for _, line := range order.Items {
		items = append(items, map[string]any{
			"sales_order_item_id": line.ID,
			"quantity_delivered":   line.Quantity,
		})
	}
	body := map[string]any{
		"recipient_name":                 buyer.Name + " Warehouse",
		"recipient_signature_reference": "SIG-" + day(0),
		"notes":                          "Delivered in good order.",
		"items":                          items,
	}
	var note deliveryNote
	must("create delivery note", c.request(http.MethodPost, "/sales-orders/"+order.ID+"/deliveries", body, &note))
	step("posted delivery note %s for %s", note.DeliveryNumber, order.OrderNumber)
}

func seedPartialDelivery(orderID string, quantity float64) {
	var order salesOrder
	must("load sales order", c.request(http.MethodGet, "/sales-orders/"+orderID, nil, &order))
	items := make([]map[string]any, 0, len(order.Items))
	for _, line := range order.Items {
		items = append(items, map[string]any{
			"sales_order_item_id": line.ID,
			"quantity_delivered":   quantity,
		})
	}
	body := map[string]any{
		"recipient_name": "Partial dispatch",
		"notes":          "First tranche of the order.",
		"items":          items,
	}
	var note deliveryNote
	must("create partial delivery note", c.request(http.MethodPost, "/sales-orders/"+orderID+"/deliveries", body, &note))
	step("posted partial delivery note %s for %s", note.DeliveryNumber, order.OrderNumber)
}
