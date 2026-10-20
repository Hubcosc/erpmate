package master

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

// ====================== SHARED HELPERS ======================

// mapErr converts driver errors into the typed errors the handler understands.
func mapErr(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505": // unique_violation
			return conflict("a record with the same code or unique value already exists")
		case "23503": // foreign_key_violation
			return conflict("the operation conflicts with related records")
		}
	}
	return err
}

// whereBuilder builds WHERE clauses with positional args. Use "?" as the placeholder.
type whereBuilder struct {
	clauses []string
	args    []interface{}
}

func (w *whereBuilder) add(clause string, args ...interface{}) {
	for _, a := range args {
		w.args = append(w.args, a)
		clause = strings.Replace(clause, "?", fmt.Sprintf("$%d", len(w.args)), 1)
	}
	w.clauses = append(w.clauses, clause)
}

func (w *whereBuilder) sql() string {
	return "WHERE " + strings.Join(w.clauses, " AND ")
}

func escapeLike(s string) string {
	r := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)
	return r.Replace(s)
}

func applySearch(w *whereBuilder, search, prefix string, cols ...string) {
	if search == "" {
		return
	}
	pat := "%" + escapeLike(search) + "%"
	parts := make([]string, len(cols))
	args := make([]interface{}, len(cols))
	for i, c := range cols {
		parts[i] = prefix + c + " ILIKE ?"
		args[i] = pat
	}
	w.add("("+strings.Join(parts, " OR ")+")", args...)
}

func applyActive(w *whereBuilder, active *bool, prefix string) {
	if active != nil {
		w.add(prefix+"is_active = ?", *active)
	}
}

func collect[T any](rows pgx.Rows, scan func(pgx.Row) (T, error)) ([]T, error) {
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (T, error) { return scan(row) })
}

func getOne[T any](ctx context.Context, db *pgxpool.Pool, query string, id uuid.UUID, scan func(pgx.Row) (T, error)) (*T, error) {
	v, err := scan(db.QueryRow(ctx, query, id))
	if err != nil {
		return nil, mapErr(err)
	}
	return &v, nil
}

// listPage runs a COUNT and a paged SELECT with the same filters. Always returns a non-nil slice.
func listPage[T any](
	ctx context.Context, db *pgxpool.Pool,
	cols, from string, w *whereBuilder, order string, limit, offset int,
	scan func(pgx.Row) (T, error),
) ([]T, int64, error) {
	var total int64
	if err := db.QueryRow(ctx, "SELECT COUNT(*) FROM "+from+" "+w.sql(), w.args...).Scan(&total); err != nil {
		return nil, 0, mapErr(err)
	}

	n := len(w.args)
	query := fmt.Sprintf("SELECT %s FROM %s %s ORDER BY %s LIMIT $%d OFFSET $%d",
		cols, from, w.sql(), order, n+1, n+2)
	args := append(append([]interface{}{}, w.args...), limit, offset)

	rows, err := db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, mapErr(err)
	}
	list, err := collect(rows, scan)
	if err != nil {
		return nil, 0, mapErr(err)
	}
	return list, total, nil
}

// updatableColumns whitelists what updateTable may touch, so a map key can never become SQL injection.
var updatableColumns = map[string]map[string]bool{
	"departments":     {"name": true, "code": true, "is_active": true},
	"item_categories": {"name": true, "parent_id": true},
	"items": {
		"name": true, "description": true, "category_id": true, "unit_of_measure": true,
		"min_stock_level": true, "max_stock_level": true, "reorder_level": true,
		"is_batch_tracked": true, "is_expiry_tracked": true, "is_active": true,
	},
	"suppliers": {
		"code": true, "name": true, "contact_person": true, "phone": true,
		"email": true, "address": true, "tin_number": true, "is_active": true,
	},
	"customers": {
		"code": true, "name": true, "customer_type": true, "contact_person": true,
		"phone": true, "email": true, "address": true, "tin_number": true, "is_active": true,
	},
	"warehouses": {"name": true, "warehouse_type": true, "location": true, "is_active": true},
}

func (r *Repository) updateTable(ctx context.Context, table string, id uuid.UUID, fields map[string]interface{}) error {
	if len(fields) == 0 {
		return nil
	}
	allowed, ok := updatableColumns[table]
	if !ok {
		return fmt.Errorf("table %q is not updatable", table)
	}

	keys := make([]string, 0, len(fields))
	for k := range fields {
		if !allowed[k] {
			return fmt.Errorf("column %q is not updatable on %s", k, table)
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var sb strings.Builder
	sb.WriteString("UPDATE " + table + " SET updated_at = NOW()")
	args := make([]interface{}, 0, len(keys)+1)
	for i, k := range keys {
		sb.WriteString(fmt.Sprintf(", %s = $%d", k, i+1))
		args = append(args, fields[k])
	}
	sb.WriteString(fmt.Sprintf(" WHERE id = $%d AND deleted_at IS NULL", len(keys)+1))
	args = append(args, id)

	tag, err := r.db.Exec(ctx, sb.String(), args...)
	if err != nil {
		return mapErr(err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// softDelete marks the row deleted; every read query filters on deleted_at IS NULL.
func (r *Repository) softDelete(ctx context.Context, table string, id uuid.UUID) error {
	if _, ok := updatableColumns[table]; !ok {
		return fmt.Errorf("table %q is not deletable", table)
	}
	tag, err := r.db.Exec(ctx,
		"UPDATE "+table+" SET deleted_at = NOW(), updated_at = NOW() WHERE id = $1 AND deleted_at IS NULL", id)
	if err != nil {
		return mapErr(err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *Repository) exists(ctx context.Context, query string, args ...interface{}) (bool, error) {
	var ok bool
	if err := r.db.QueryRow(ctx, "SELECT EXISTS("+query+")", args...).Scan(&ok); err != nil {
		return false, mapErr(err)
	}
	return ok, nil
}

// ====================== DEPARTMENTS ======================

const departmentCols = "id, name, code, is_active, created_at, updated_at"

func scanDepartment(row pgx.Row) (Department, error) {
	var d Department
	err := row.Scan(&d.ID, &d.Name, &d.Code, &d.IsActive, &d.CreatedAt, &d.UpdatedAt)
	return d, err
}

func (r *Repository) CreateDepartment(ctx context.Context, d *Department) error {
	return mapErr(r.db.QueryRow(ctx, `
		INSERT INTO departments (id, name, code, is_active)
		VALUES ($1, $2, $3, $4)
		RETURNING created_at, updated_at
	`, d.ID, d.Name, d.Code, d.IsActive).Scan(&d.CreatedAt, &d.UpdatedAt))
}

func (r *Repository) GetDepartmentByID(ctx context.Context, id uuid.UUID) (*Department, error) {
	return getOne(ctx, r.db, "SELECT "+departmentCols+" FROM departments WHERE id = $1 AND deleted_at IS NULL", id, scanDepartment)
}

func (r *Repository) ListDepartments(ctx context.Context, p ListParams) ([]Department, int64, error) {
	w := &whereBuilder{}
	w.add("deleted_at IS NULL")
	applySearch(w, p.Search, "", "name", "code")
	applyActive(w, p.IsActive, "")
	return listPage(ctx, r.db, departmentCols, "departments", w, "name, id", p.Limit, p.offset(), scanDepartment)
}

func (r *Repository) UpdateDepartment(ctx context.Context, id uuid.UUID, fields map[string]interface{}) error {
	return r.updateTable(ctx, "departments", id, fields)
}

func (r *Repository) DeleteDepartment(ctx context.Context, id uuid.UUID) error {
	return r.softDelete(ctx, "departments", id)
}

// ====================== ITEM CATEGORIES ======================

const categoryCols = "id, name, parent_id, created_at, updated_at"

func scanCategory(row pgx.Row) (ItemCategory, error) {
	var c ItemCategory
	err := row.Scan(&c.ID, &c.Name, &c.ParentID, &c.CreatedAt, &c.UpdatedAt)
	return c, err
}

func (r *Repository) CreateCategory(ctx context.Context, c *ItemCategory) error {
	return mapErr(r.db.QueryRow(ctx, `
		INSERT INTO item_categories (id, name, parent_id)
		VALUES ($1, $2, $3)
		RETURNING created_at, updated_at
	`, c.ID, c.Name, c.ParentID).Scan(&c.CreatedAt, &c.UpdatedAt))
}

func (r *Repository) GetCategoryByID(ctx context.Context, id uuid.UUID) (*ItemCategory, error) {
	return getOne(ctx, r.db, "SELECT "+categoryCols+" FROM item_categories WHERE id = $1 AND deleted_at IS NULL", id, scanCategory)
}

func (r *Repository) ListCategories(ctx context.Context, p ListParams) ([]ItemCategory, int64, error) {
	w := &whereBuilder{}
	w.add("deleted_at IS NULL")
	applySearch(w, p.Search, "", "name")
	return listPage(ctx, r.db, categoryCols, "item_categories", w, "name, id", p.Limit, p.offset(), scanCategory)
}

func (r *Repository) UpdateCategory(ctx context.Context, id uuid.UUID, fields map[string]interface{}) error {
	return r.updateTable(ctx, "item_categories", id, fields)
}

func (r *Repository) DeleteCategory(ctx context.Context, id uuid.UUID) error {
	return r.softDelete(ctx, "item_categories", id)
}

func (r *Repository) CategoryHasChildren(ctx context.Context, id uuid.UUID) (bool, error) {
	return r.exists(ctx, "SELECT 1 FROM item_categories WHERE parent_id = $1 AND deleted_at IS NULL", id)
}

func (r *Repository) CategoryHasItems(ctx context.Context, id uuid.UUID) (bool, error) {
	return r.exists(ctx, "SELECT 1 FROM items WHERE category_id = $1 AND deleted_at IS NULL", id)
}

// WouldCreateCycle reports whether making newParentID the parent of categoryID creates a loop,
// i.e. categoryID is newParentID itself or one of its ancestors.
func (r *Repository) WouldCreateCycle(ctx context.Context, categoryID, newParentID uuid.UUID) (bool, error) {
	return r.exists(ctx, `
		WITH RECURSIVE ancestors AS (
			SELECT id, parent_id FROM item_categories WHERE id = $1
			UNION
			SELECT c.id, c.parent_id FROM item_categories c JOIN ancestors a ON c.id = a.parent_id
		)
		SELECT 1 FROM ancestors WHERE id = $2
	`, newParentID, categoryID)
}

// ====================== ITEMS ======================

const (
	itemCols = `i.id, i.code, i.name, i.description, i.category_id, c.name,
		i.item_type, i.unit_of_measure, i.min_stock_level, i.max_stock_level,
		i.reorder_level, i.is_batch_tracked, i.is_expiry_tracked, i.is_active,
		i.created_at, i.updated_at`
	itemFrom = "items i LEFT JOIN item_categories c ON c.id = i.category_id"
)

func scanItem(row pgx.Row) (Item, error) {
	var i Item
	err := row.Scan(
		&i.ID, &i.Code, &i.Name, &i.Description, &i.CategoryID, &i.CategoryName,
		&i.ItemType, &i.UnitOfMeasure, &i.MinStockLevel, &i.MaxStockLevel,
		&i.ReorderLevel, &i.IsBatchTracked, &i.IsExpiryTracked, &i.IsActive,
		&i.CreatedAt, &i.UpdatedAt,
	)
	return i, err
}

func (r *Repository) CreateItem(ctx context.Context, item *Item) error {
	return mapErr(r.db.QueryRow(ctx, `
		INSERT INTO items (
			id, code, name, description, category_id, item_type, unit_of_measure,
			min_stock_level, max_stock_level, reorder_level, is_batch_tracked, is_expiry_tracked, is_active
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)
		RETURNING created_at, updated_at
	`, item.ID, item.Code, item.Name, item.Description, item.CategoryID, item.ItemType, item.UnitOfMeasure,
		item.MinStockLevel, item.MaxStockLevel, item.ReorderLevel, item.IsBatchTracked, item.IsExpiryTracked, item.IsActive,
	).Scan(&item.CreatedAt, &item.UpdatedAt))
}

func (r *Repository) GetItemByID(ctx context.Context, id uuid.UUID) (*Item, error) {
	return getOne(ctx, r.db,
		"SELECT "+itemCols+" FROM "+itemFrom+" WHERE i.id = $1 AND i.deleted_at IS NULL", id, scanItem)
}

func (r *Repository) ListItems(ctx context.Context, p ItemListParams) ([]Item, int64, error) {
	w := &whereBuilder{}
	w.add("i.deleted_at IS NULL")
	applySearch(w, p.Search, "i.", "name", "code")
	applyActive(w, p.IsActive, "i.")
	if p.ItemType != "" {
		w.add("i.item_type = ?", p.ItemType)
	}
	if p.CategoryID != nil {
		w.add("i.category_id = ?", *p.CategoryID)
	}
	return listPage(ctx, r.db, itemCols, itemFrom, w, "i.name, i.id", p.Limit, p.offset(), scanItem)
}

func (r *Repository) UpdateItem(ctx context.Context, id uuid.UUID, fields map[string]interface{}) error {
	return r.updateTable(ctx, "items", id, fields)
}

func (r *Repository) DeleteItem(ctx context.Context, id uuid.UUID) error {
	return r.softDelete(ctx, "items", id)
}

// ====================== SUPPLIERS ======================

const supplierCols = "id, code, name, contact_person, phone, email, address, tin_number, is_active, created_at, updated_at"

func scanSupplier(row pgx.Row) (Supplier, error) {
	var s Supplier
	err := row.Scan(&s.ID, &s.Code, &s.Name, &s.ContactPerson, &s.Phone, &s.Email, &s.Address, &s.TinNumber, &s.IsActive, &s.CreatedAt, &s.UpdatedAt)
	return s, err
}

func (r *Repository) CreateSupplier(ctx context.Context, s *Supplier) error {
	return mapErr(r.db.QueryRow(ctx, `
		INSERT INTO suppliers (id, code, name, contact_person, phone, email, address, tin_number, is_active)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
		RETURNING created_at, updated_at
	`, s.ID, s.Code, s.Name, s.ContactPerson, s.Phone, s.Email, s.Address, s.TinNumber, s.IsActive).
		Scan(&s.CreatedAt, &s.UpdatedAt))
}

func (r *Repository) GetSupplierByID(ctx context.Context, id uuid.UUID) (*Supplier, error) {
	return getOne(ctx, r.db, "SELECT "+supplierCols+" FROM suppliers WHERE id = $1 AND deleted_at IS NULL", id, scanSupplier)
}

func (r *Repository) ListSuppliers(ctx context.Context, p ListParams) ([]Supplier, int64, error) {
	w := &whereBuilder{}
	w.add("deleted_at IS NULL")
	applySearch(w, p.Search, "", "name", "code", "contact_person", "tin_number")
	applyActive(w, p.IsActive, "")
	return listPage(ctx, r.db, supplierCols, "suppliers", w, "name, id", p.Limit, p.offset(), scanSupplier)
}

func (r *Repository) UpdateSupplier(ctx context.Context, id uuid.UUID, fields map[string]interface{}) error {
	return r.updateTable(ctx, "suppliers", id, fields)
}

func (r *Repository) DeleteSupplier(ctx context.Context, id uuid.UUID) error {
	return r.softDelete(ctx, "suppliers", id)
}

// ====================== CUSTOMERS ======================

const customerCols = "id, code, name, customer_type, contact_person, phone, email, address, tin_number, is_active, created_at, updated_at"

func scanCustomer(row pgx.Row) (Customer, error) {
	var c Customer
	err := row.Scan(&c.ID, &c.Code, &c.Name, &c.CustomerType, &c.ContactPerson, &c.Phone, &c.Email, &c.Address, &c.TinNumber, &c.IsActive, &c.CreatedAt, &c.UpdatedAt)
	return c, err
}

func (r *Repository) CreateCustomer(ctx context.Context, c *Customer) error {
	return mapErr(r.db.QueryRow(ctx, `
		INSERT INTO customers (id, code, name, customer_type, contact_person, phone, email, address, tin_number, is_active)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
		RETURNING created_at, updated_at
	`, c.ID, c.Code, c.Name, c.CustomerType, c.ContactPerson, c.Phone, c.Email, c.Address, c.TinNumber, c.IsActive).
		Scan(&c.CreatedAt, &c.UpdatedAt))
}

func (r *Repository) GetCustomerByID(ctx context.Context, id uuid.UUID) (*Customer, error) {
	return getOne(ctx, r.db, "SELECT "+customerCols+" FROM customers WHERE id = $1 AND deleted_at IS NULL", id, scanCustomer)
}

func (r *Repository) ListCustomers(ctx context.Context, p ListParams) ([]Customer, int64, error) {
	w := &whereBuilder{}
	w.add("deleted_at IS NULL")
	applySearch(w, p.Search, "", "name", "code", "contact_person", "tin_number")
	applyActive(w, p.IsActive, "")
	return listPage(ctx, r.db, customerCols, "customers", w, "name, id", p.Limit, p.offset(), scanCustomer)
}

func (r *Repository) UpdateCustomer(ctx context.Context, id uuid.UUID, fields map[string]interface{}) error {
	return r.updateTable(ctx, "customers", id, fields)
}

func (r *Repository) DeleteCustomer(ctx context.Context, id uuid.UUID) error {
	return r.softDelete(ctx, "customers", id)
}

// ====================== WAREHOUSES ======================

const warehouseCols = "id, code, name, warehouse_type, location, is_active, created_at, updated_at"

func scanWarehouse(row pgx.Row) (Warehouse, error) {
	var w Warehouse
	err := row.Scan(&w.ID, &w.Code, &w.Name, &w.WarehouseType, &w.Location, &w.IsActive, &w.CreatedAt, &w.UpdatedAt)
	return w, err
}

func (r *Repository) CreateWarehouse(ctx context.Context, w *Warehouse) error {
	return mapErr(r.db.QueryRow(ctx, `
		INSERT INTO warehouses (id, code, name, warehouse_type, location, is_active)
		VALUES ($1,$2,$3,$4,$5,$6)
		RETURNING created_at, updated_at
	`, w.ID, w.Code, w.Name, w.WarehouseType, w.Location, w.IsActive).Scan(&w.CreatedAt, &w.UpdatedAt))
}

func (r *Repository) GetWarehouseByID(ctx context.Context, id uuid.UUID) (*Warehouse, error) {
	return getOne(ctx, r.db, "SELECT "+warehouseCols+" FROM warehouses WHERE id = $1 AND deleted_at IS NULL", id, scanWarehouse)
}

func (r *Repository) ListWarehouses(ctx context.Context, p ListParams) ([]Warehouse, int64, error) {
	w := &whereBuilder{}
	w.add("deleted_at IS NULL")
	applySearch(w, p.Search, "", "name", "code")
	applyActive(w, p.IsActive, "")
	return listPage(ctx, r.db, warehouseCols, "warehouses", w, "name, id", p.Limit, p.offset(), scanWarehouse)
}

func (r *Repository) UpdateWarehouse(ctx context.Context, id uuid.UUID, fields map[string]interface{}) error {
	return r.updateTable(ctx, "warehouses", id, fields)
}

func (r *Repository) DeleteWarehouse(ctx context.Context, id uuid.UUID) error {
	return r.softDelete(ctx, "warehouses", id)
}
