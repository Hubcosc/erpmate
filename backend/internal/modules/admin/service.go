package admin

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

// ErrNotFound is returned when a record does not exist.
var ErrNotFound = errors.New("record not found")

// ErrValidation is returned when the submitted values do not satisfy the model.
var ErrValidation = errors.New("validation failed")

// Service runs the generic queries behind the admin site.
type Service struct {
	pool *pgxpool.Pool
}

// NewService builds an admin service.
func NewService(pool *pgxpool.Pool) *Service {
	return &Service{pool: pool}
}

// columnName turns a registry column expression into its physical column name.
// The expression always comes from the registry, never from a request.
func columnName(sql string) string {
	if idx := strings.LastIndex(sql, "."); idx >= 0 {
		return sql[idx+1:]
	}
	return sql
}

// dbType is the PostgreSQL type a bind parameter is cast to. Fields that do not
// declare one are treated as text.
func dbType(f Field) string {
	if f.DBType == "" {
		return "text"
	}
	return f.DBType
}

// fromClause builds the FROM and JOIN parts of a query.
func fromClause(m Model) string {
	clause := m.Table + " m"
	for _, join := range m.Joins {
		clause += " " + join
	}
	return clause
}

// whereClause builds the WHERE parts shared by the list and count queries.
func whereClause(m Model, search string) (string, []any) {
	var conditions []string
	var args []any

	if m.SoftDelete {
		conditions = append(conditions, "m.deleted_at IS NULL")
	}
	if search != "" && len(m.Search) > 0 {
		parts := make([]string, 0, len(m.Search))
		for _, col := range m.Search {
			args = append(args, "%"+search+"%")
			parts = append(parts, fmt.Sprintf("%s ILIKE $%d", col, len(args)))
		}
		conditions = append(conditions, "("+strings.Join(parts, " OR ")+")")
	}
	if len(conditions) == 0 {
		return "", args
	}
	return " WHERE " + strings.Join(conditions, " AND "), args
}

// recordFilter excludes soft deleted rows from single-record lookups.
func recordFilter(m Model) string {
	if m.SoftDelete {
		return " AND m.deleted_at IS NULL"
	}
	return ""
}

// List returns a page of records for the model.
func (s *Service) List(ctx context.Context, m Model, search string, page, limit int) ([]map[string]any, int64, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 200 {
		limit = 50
	}

	where, args := whereClause(m, search)

	selectParts := []string{"m.id AS id"}
	for _, col := range m.Columns {
		selectParts = append(selectParts, col.SQL+" AS "+col.Key)
	}
	query := "SELECT " + strings.Join(selectParts, ", ") + " FROM " + fromClause(m) + where

	orderBy := m.OrderBy
	if orderBy == "" {
		orderBy = "m.id"
	}
	direction := "ASC"
	if m.OrderDesc {
		direction = "DESC"
	}
	query += " ORDER BY " + orderBy + " " + direction + ", m.id " + direction

	query += fmt.Sprintf(" LIMIT $%d OFFSET $%d", len(args)+1, len(args)+2)
	args = append(args, limit, (page-1)*limit)

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list %s: %w", m.Key, err)
	}
	defer rows.Close()

	items, err := scanRows(rows)
	if err != nil {
		return nil, 0, err
	}

	countQuery := "SELECT count(*) FROM " + fromClause(m) + where
	countArgs := args[:len(args)-2]
	var total int64
	if err := s.pool.QueryRow(ctx, countQuery, countArgs...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count %s: %w", m.Key, err)
	}

	return items, total, nil
}

// Get returns a single record with every form field populated.
func (s *Service) Get(ctx context.Context, m Model, id string) (map[string]any, error) {
	parts := []string{"m.id AS id"}
	for _, f := range m.Fields {
		if f.WriteOnly {
			continue
		}
		parts = append(parts, f.SQL+" AS "+f.Name)
	}
	query := "SELECT " + strings.Join(parts, ", ") + " FROM " + fromClause(m) + " WHERE m.id = $1" + recordFilter(m)

	rows, err := s.pool.Query(ctx, query, id)
	if err != nil {
		return nil, fmt.Errorf("get %s: %w", m.Key, err)
	}
	defer rows.Close()

	items, err := scanRows(rows)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, ErrNotFound
	}
	return items[0], nil
}

// Create inserts a new record from the submitted values.
func (s *Service) Create(ctx context.Context, m Model, values map[string]any) (map[string]any, error) {
	if !m.Editable {
		return nil, fmt.Errorf("%s cannot be edited", m.Key)
	}

	columns := []string{}
	placeholders := []string{}
	args := []any{}

	for _, f := range m.Fields {
		if f.ReadOnly {
			continue
		}
		value, ok := values[f.Name]
		if !ok || value == nil {
			if f.Required {
				return nil, fmt.Errorf("%s is required", f.Label)
			}
			continue
		}
		if str, isString := value.(string); isString && str == "" {
			// An empty string clears a nullable text column.
			if f.Required {
				return nil, fmt.Errorf("%s is required", f.Label)
			}
			value = nil
		}
		if f.Required && value == nil {
			return nil, fmt.Errorf("%s is required", f.Label)
		}

		column := columnName(f.SQL)
		if f.WriteOnly {
			// The password is stored hashed, never in the clear.
			hash, err := bcrypt.GenerateFromPassword([]byte(value.(string)), bcrypt.DefaultCost)
			if err != nil {
				return nil, fmt.Errorf("hash password: %w", err)
			}
			value = string(hash)
			column = "password_hash"
		}

		args = append(args, value)
		columns = append(columns, column)
		placeholders = append(placeholders, fmt.Sprintf("$%d::%s", len(args), dbType(f)))
	}

	if len(columns) == 0 {
		return nil, errors.New("nothing to save")
	}

	query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s) RETURNING id",
		m.Table, strings.Join(columns, ", "), strings.Join(placeholders, ", "))

	var id string
	if err := s.pool.QueryRow(ctx, query, args...).Scan(&id); err != nil {
		return nil, fmt.Errorf("create %s: %w", m.Key, err)
	}
	return s.Get(ctx, m, id)
}

// Update changes an existing record from the submitted values.
func (s *Service) Update(ctx context.Context, m Model, id string, values map[string]any) (map[string]any, error) {
	if !m.Editable {
		return nil, fmt.Errorf("%s cannot be edited", m.Key)
	}

	sets := []string{}
	args := []any{}

	for _, f := range m.Fields {
		if f.ReadOnly {
			continue
		}
		value, ok := values[f.Name]
		if !ok {
			continue
		}
		if str, isString := value.(string); isString && str == "" {
			if f.Required {
				return nil, fmt.Errorf("%s is required", f.Label)
			}
			value = nil
		}
		if f.WriteOnly {
			secret, isString := value.(string)
			if !isString || secret == "" {
				continue
			}
			if len(secret) < 12 {
				return nil, errors.New("password must be at least 12 characters")
			}
			hash, err := bcrypt.GenerateFromPassword([]byte(secret), bcrypt.DefaultCost)
			if err != nil {
				return nil, fmt.Errorf("hash password: %w", err)
			}
			args = append(args, string(hash))
			sets = append(sets, "password_hash = $"+fmt.Sprint(len(args))+"::text")
			continue
		}

		args = append(args, value)
		sets = append(sets, columnName(f.SQL)+" = $"+fmt.Sprint(len(args))+"::"+dbType(f))
	}

	if len(sets) == 0 {
		return s.Get(ctx, m, id)
	}

	args = append(args, id)
	query := fmt.Sprintf("UPDATE %s SET %s, updated_at = now() WHERE id = $%d AND deleted_at IS NULL",
		m.Table, strings.Join(sets, ", "), len(args))

	tag, err := s.pool.Exec(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("update %s: %w", m.Key, err)
	}
	if tag.RowsAffected() == 0 {
		// The row may exist but already be soft deleted.
		return nil, ErrNotFound
	}
	return s.Get(ctx, m, id)
}

// Delete removes a record. Tables with a deleted_at column are soft deleted so
// the audit trail stays intact.
func (s *Service) Delete(ctx context.Context, m Model, id string) error {
	if !m.Editable {
		return fmt.Errorf("%s cannot be deleted", m.Key)
	}

	var query string
	if m.SoftDelete {
		query = fmt.Sprintf("UPDATE %s SET deleted_at = now() WHERE id = $1 AND deleted_at IS NULL", m.Table)
	} else {
		query = fmt.Sprintf("DELETE FROM %s WHERE id = $1", m.Table)
	}

	tag, err := s.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete %s: %w", m.Key, err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// ModelCount is the number of records in one model.
type ModelCount struct {
	Key   string `json:"key"`
	Label string `json:"label"`
	Group string `json:"group"`
	Icon  string `json:"icon"`
	Count int64  `json:"count"`
}

// Overview counts the records in every registered model.
func (s *Service) Overview(ctx context.Context) ([]ModelCount, error) {
	out := make([]ModelCount, 0, len(models))
	for _, m := range models {
		where, args := whereClause(m, "")
		query := "SELECT count(*) FROM " + fromClause(m) + where
		var count int64
		if err := s.pool.QueryRow(ctx, query, args...).Scan(&count); err != nil {
			return nil, fmt.Errorf("count %s: %w", m.Key, err)
		}
		out = append(out, ModelCount{Key: m.Key, Label: m.LabelPlural, Group: m.Group, Icon: m.Icon, Count: count})
	}
	return out, nil
}

// Lookups are the reference lists the admin forms use to populate choices.
type Lookups struct {
	Roles         []Option `json:"roles"`
	Departments   []Option `json:"departments"`
	ItemCategories []Option `json:"item_categories"`
	Items         []Option `json:"items"`
	Warehouses    []Option `json:"warehouses"`
	Suppliers     []Option `json:"suppliers"`
	Customers     []Option `json:"customers"`
}

// BuildLookups collects the reference lists used by the admin forms.
func (s *Service) BuildLookups(ctx context.Context) (Lookups, error) {
	var l Lookups
	queries := []struct {
		target *[]Option
		query  string
	}{
		{&l.Roles, "SELECT id, name FROM roles ORDER BY name"},
		{&l.Departments, "SELECT id, name FROM departments WHERE deleted_at IS NULL ORDER BY name"},
		{&l.ItemCategories, "SELECT id, name FROM item_categories WHERE deleted_at IS NULL ORDER BY name"},
		{&l.Items, "SELECT id, code || ' — ' || name AS name FROM items WHERE deleted_at IS NULL ORDER BY code"},
		{&l.Warehouses, "SELECT id, code || ' — ' || name AS name FROM warehouses WHERE deleted_at IS NULL ORDER BY code"},
		{&l.Suppliers, "SELECT id, name FROM suppliers WHERE deleted_at IS NULL ORDER BY name"},
		{&l.Customers, "SELECT id, name FROM customers WHERE deleted_at IS NULL ORDER BY name"},
	}
	for _, q := range queries {
		rows, err := s.pool.Query(ctx, q.query)
		if err != nil {
			return l, fmt.Errorf("lookups: %w", err)
		}
		for rows.Next() {
			var id string
			var label string
			if err := rows.Scan(&id, &label); err != nil {
				rows.Close()
				return l, fmt.Errorf("lookups: %w", err)
			}
			*q.target = append(*q.target, Option{Value: id, Label: label})
		}
		rows.Close()
	}
	return l, nil
}

// scanRows reads a result set into a slice of maps keyed by column name.
func scanRows(rows pgx.Rows) ([]map[string]any, error) {
	fields := rows.FieldDescriptions()
	items := make([]map[string]any, 0, 16)

	for rows.Next() {
		values := make([]any, len(fields))
		targets := make([]any, len(fields))
		for i := range values {
			targets[i] = &values[i]
		}
		if err := rows.Scan(targets...); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		row := make(map[string]any, len(fields))
		for i, field := range fields {
			row[string(field.Name)] = normalise(values[i])
		}
		items = append(items, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows: %w", err)
	}
	return items, nil
}

// normalise converts driver values into JSON-friendly values.
func normalise(value any) any {
	switch v := value.(type) {
	case time.Time:
		return v
	case uuid.UUID:
		return v.String()
	case [16]byte:
		// pgx returns a uuid column as a byte array when the target is `any`.
		return uuid.UUID(v).String()
	case []byte:
		return string(v)
	default:
		return value
	}
}
