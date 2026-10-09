package clickhouse

import (
	"context"
	"database/sql"
	"regexp"
	"strings"

	"github.com/relay-client/plugboard/apps/desktop/internal/db/sql/dialect"
	"github.com/relay-client/plugboard/apps/desktop/internal/model"
)

func (Dialect) DefaultSchema(ctx context.Context, db *sql.DB, _ model.Connection) (string, error) {
	var name string
	err := db.QueryRowContext(ctx, `SELECT currentDatabase()`).Scan(&name)
	return name, err
}

func (Dialect) ListSchemas(ctx context.Context, db *sql.DB) ([]string, error) {
	return dialect.QueryStrings(ctx, db, `
		SELECT name FROM system.databases
		WHERE name NOT IN ('INFORMATION_SCHEMA', 'information_schema')
		ORDER BY name = 'system', name`)
}

func (Dialect) ListTables(ctx context.Context, db *sql.DB, schema string) ([]model.TableInfo, error) {
	return dialect.QueryTables(ctx, db, schema, `
		SELECT name, if(engine IN ('View', 'MaterializedView', 'LiveView', 'WindowView'), 'VIEW', 'BASE TABLE')
		FROM system.tables
		WHERE database = ? AND NOT is_temporary AND engine != 'Dictionary'
		ORDER BY name`, schema)
}

func (d Dialect) ListColumns(ctx context.Context, db *sql.DB, schema, table string) ([]model.Column, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT name, type, default_kind, default_expression
		FROM system.columns WHERE database = ? AND table = ?
		ORDER BY position`, schema, table)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.Column{}
	for rows.Next() {
		var c model.Column
		var kind, expr string
		if err := rows.Scan(&c.Name, &c.Type, &kind, &expr); err != nil {
			return nil, err
		}
		c.Nullable = strings.HasPrefix(unwrap(c.Type, "LowCardinality"), "Nullable(")
		c.Kind = d.TypeOf(c.Type).Kind
		c.Enum = enumValues(c.Type)
		if kind != "" {
			def := expr
			if kind != "DEFAULT" {
				def = kind + " " + expr
			}
			c.Default = &def
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

var enumItem = regexp.MustCompile(`'((?:[^'\\]|\\.)*)'\s*=\s*-?\d+`)

func enumValues(t string) []string {
	t = unwrap(unwrap(t, "LowCardinality"), "Nullable")
	if !strings.HasPrefix(t, "Enum") {
		return nil
	}
	var out []string
	for _, m := range enumItem.FindAllStringSubmatch(t, -1) {
		out = append(out, strings.NewReplacer(`\'`, `'`, `\\`, `\`).Replace(m[1]))
	}
	return out
}

func unwrap(t, wrapper string) string {
	if strings.HasPrefix(t, wrapper+"(") && strings.HasSuffix(t, ")") {
		return t[len(wrapper)+1 : len(t)-1]
	}
	return t
}

func (Dialect) ListRelations(context.Context, *sql.DB, string) ([]model.Relation, error) {
	return []model.Relation{}, nil
}

func (Dialect) ListIndexes(ctx context.Context, db *sql.DB, schema, table string) ([]model.Index, error) {
	var engine, sorting, primary string
	err := db.QueryRowContext(ctx, `SELECT engine, sorting_key, primary_key FROM system.tables WHERE database = ? AND name = ?`, schema, table).Scan(&engine, &sorting, &primary)
	if err == sql.ErrNoRows {
		return []model.Index{}, nil
	}
	if err != nil {
		return nil, err
	}
	out := []model.Index{}
	if sorting != "" {
		key := primary
		if key == "" {
			key = sorting
		}
		out = append(out, model.Index{Name: "PRIMARY KEY", Columns: splitKey(key), Primary: true, Method: engine, Definition: "ORDER BY (" + sorting + ")"})
	}
	rows, err := db.QueryContext(ctx, `
		SELECT name, type_full, expr, granularity FROM system.data_skipping_indices
		WHERE database = ? AND table = ? ORDER BY name`, schema, table)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var name, typ, expr string
		var granularity uint64
		if err := rows.Scan(&name, &typ, &expr, &granularity); err != nil {
			return nil, err
		}
		out = append(out, model.Index{Name: name, Columns: splitKey(expr), Method: typ, Definition: "INDEX " + name + " " + expr + " TYPE " + typ + " GRANULARITY " + itoa(granularity)})
	}
	return out, rows.Err()
}

func splitKey(key string) []string {
	var out []string
	depth, start := 0, 0
	for i, r := range key {
		switch r {
		case '(':
			depth++
		case ')':
			depth--
		case ',':
			if depth == 0 {
				out = append(out, strings.TrimSpace(key[start:i]))
				start = i + 1
			}
		}
	}
	if rest := strings.TrimSpace(key[start:]); rest != "" {
		out = append(out, rest)
	}
	return out
}

func (Dialect) SortKey(ctx context.Context, db *sql.DB, schema, table string) ([]string, error) {
	var key string
	err := db.QueryRowContext(ctx, `SELECT sorting_key FROM system.tables WHERE database = ? AND name = ?`, schema, table).Scan(&key)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return splitKey(key), err
}

func (Dialect) EstimateRows(ctx context.Context, db *sql.DB, schema, table string) (int64, bool, error) {
	var n sql.NullInt64
	err := db.QueryRowContext(ctx, `SELECT total_rows FROM system.tables WHERE database = ? AND name = ?`, schema, table).Scan(&n)
	if err == sql.ErrNoRows {
		return 0, false, nil
	}
	return n.Int64, err == nil && n.Valid, err
}
