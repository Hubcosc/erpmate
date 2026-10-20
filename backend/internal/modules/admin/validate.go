package admin

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Validate checks the registry against the live schema.
//
// Every column and field the registry names has to exist, otherwise a list
// view would fail with a 500 the first time somebody opened it. Running this
// at startup turns a mistake in the registry into an obvious error message.
func Validate(ctx context.Context, pool *pgxpool.Pool) error {
	problems := []string{}

	for _, m := range models {
		aliases := aliasTables(m)
		cache := map[string]map[string]bool{}

		columnsOf := func(alias string) (map[string]bool, error) {
			table, ok := aliases[alias]
			if !ok {
				return nil, fmt.Errorf("unknown table alias %q", alias)
			}
			if cached, ok := cache[table]; ok {
				return cached, nil
			}
			existing, err := tableColumns(ctx, pool, table)
			if err != nil {
				return nil, err
			}
			cache[table] = existing
			return existing, nil
		}

		for _, column := range m.Columns {
			alias, name := splitReference(column.SQL)
			existing, err := columnsOf(alias)
			if err != nil {
				problems = append(problems, fmt.Sprintf("%s: list column %q uses %q", m.Key, column.Key, column.SQL))
				continue
			}
			if !existing[name] {
				problems = append(problems, fmt.Sprintf("%s: list column %q references %q, which %s does not have", m.Key, column.Key, name, aliases[alias]))
			}
		}

		for _, field := range m.Fields {
			alias, name := splitReference(field.SQL)
			existing, err := columnsOf(alias)
			if err != nil {
				problems = append(problems, fmt.Sprintf("%s: form field %q uses %q", m.Key, field.Name, field.SQL))
				continue
			}
			if !existing[name] {
				problems = append(problems, fmt.Sprintf("%s: form field %q references %q, which %s does not have", m.Key, field.Name, name, aliases[alias]))
			}
		}
	}

	if len(problems) > 0 {
		return fmt.Errorf("the admin model registry is out of date:\n  - %s", strings.Join(problems, "\n  - "))
	}
	return nil
}

// aliasTables maps every table alias a model can use to its table.
func aliasTables(m Model) map[string]string {
	aliases := map[string]string{"m": m.Table}
	for _, join := range m.Joins {
		// The join looks like "LEFT JOIN items i ON i.id = m.category_id".
		fields := strings.Fields(join)
		for i, field := range fields {
			if strings.EqualFold(field, "JOIN") && i+2 < len(fields) {
				table := fields[i+1]
				alias := fields[i+2]
				// Stop at the ON keyword.
				if alias == "ON" {
					break
				}
				aliases[alias] = table
				break
			}
		}
	}
	return aliases
}

// splitReference splits "m.code" into the alias "m" and the column "code".
func splitReference(expression string) (string, string) {
	if idx := strings.LastIndex(expression, "."); idx >= 0 {
		return expression[:idx], expression[idx+1:]
	}
	return "m", expression
}

// tableColumns returns the column names of a table.
func tableColumns(ctx context.Context, pool *pgxpool.Pool, table string) (map[string]bool, error) {
	rows, err := pool.Query(ctx, `
		SELECT column_name FROM information_schema.columns
		WHERE table_schema = 'public' AND table_name = $1`, table)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	columns := map[string]bool{}
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		columns[name] = true
	}
	return columns, rows.Err()
}
