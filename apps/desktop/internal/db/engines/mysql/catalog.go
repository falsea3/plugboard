package mysql

import (
	"context"
	"database/sql"
	"strings"

	"github.com/relay-client/plugboard/apps/desktop/internal/db/sql/dialect"
	"github.com/relay-client/plugboard/apps/desktop/internal/model"
)

func (Dialect) DefaultSchema(ctx context.Context, db *sql.DB, c model.Connection) (string, error) {
	if c.Database != "" {
		return c.Database, nil
	}
	var s sql.NullString
	if err := db.QueryRowContext(ctx, `SELECT DATABASE()`).Scan(&s); err != nil {
		return "", err
	}
	return s.String, nil
}

func (Dialect) ListSchemas(ctx context.Context, db *sql.DB) ([]string, error) {
	return dialect.QueryStrings(ctx, db, `
		SELECT schema_name FROM information_schema.schemata
		WHERE schema_name NOT IN ('information_schema', 'mysql', 'performance_schema', 'sys')
		ORDER BY schema_name`)
}

func (Dialect) ListTables(ctx context.Context, db *sql.DB, schema string) ([]model.TableInfo, error) {
	return dialect.QueryTables(ctx, db, schema, `
		SELECT table_name, table_type FROM information_schema.tables
		WHERE table_schema = ?
		ORDER BY table_name`, schema)
}

func (Dialect) ListRelations(ctx context.Context, db *sql.DB, schema string) ([]model.Relation, error) {
	return dialect.QueryRelations(ctx, db, `
		SELECT k.constraint_name, k.table_name, k.column_name, k.referenced_table_schema, k.referenced_table_name,
		       k.referenced_column_name, r.delete_rule, r.update_rule
		FROM information_schema.key_column_usage k
		JOIN information_schema.referential_constraints r
		  ON r.constraint_schema = k.constraint_schema AND r.constraint_name = k.constraint_name AND r.table_name = k.table_name
		WHERE k.table_schema = ? AND k.referenced_table_name IS NOT NULL
		ORDER BY k.table_name, k.constraint_name, k.ordinal_position`, schema)
}

func (d Dialect) ListColumns(ctx context.Context, db *sql.DB, schema, table string) ([]model.Column, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT column_name, column_type, data_type, is_nullable = 'YES', column_default, extra, column_key = 'PRI'
		FROM information_schema.columns
		WHERE table_schema = ? AND table_name = ?
		ORDER BY ordinal_position`, schema, table)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.Column{}
	for rows.Next() {
		var c model.Column
		var dataType, extra string
		var def sql.NullString
		if err := rows.Scan(&c.Name, &c.Type, &dataType, &c.Nullable, &def, &extra, &c.PrimaryKey); err != nil {
			return nil, err
		}
		c.Default = d.defaultSQL(def, dataType, extra)
		c.Enum = parseEnum(c.Type)
		c.Kind = d.TypeOf(c.Type).Kind
		out = append(out, c)
	}
	return out, rows.Err()
}

func parseEnum(t string) []string {
	if !strings.HasPrefix(strings.ToLower(t), "enum(") || !strings.HasSuffix(t, ")") {
		return nil
	}
	body := t[5 : len(t)-1]
	var out []string
	for i := 0; i < len(body); i++ {
		if body[i] != '\'' {
			continue
		}
		var b strings.Builder
		for i++; i < len(body); i++ {
			if body[i] == '\'' {
				if i+1 < len(body) && body[i+1] == '\'' {
					b.WriteByte('\'')
					i++
					continue
				}
				break
			}
			b.WriteByte(body[i])
		}
		out = append(out, b.String())
	}
	return out
}

var numericTypes = map[string]bool{
	"tinyint": true, "smallint": true, "mediumint": true, "int": true, "integer": true, "bigint": true,
	"decimal": true, "numeric": true, "float": true, "double": true, "real": true, "bit": true, "year": true,
}

func (d Dialect) defaultSQL(raw sql.NullString, dataType, extra string) *string {
	if !raw.Valid {
		return nil
	}
	v := raw.String
	if d.writesDefaultsAsSQL() {
		if v == "NULL" {
			return nil
		}
		return &v
	}
	dataType = strings.ToLower(dataType)
	upper := strings.ToUpper(v)
	now := strings.HasPrefix(upper, "CURRENT_TIMESTAMP") || strings.HasPrefix(upper, "NOW(") || strings.HasPrefix(upper, "LOCALTIME")
	switch {
	case strings.Contains(strings.ToUpper(extra), "DEFAULT_GENERATED"):
		if !now {
			v = "(" + unescape(v) + ")"
		}
	case now && (dataType == "timestamp" || dataType == "datetime"):
	case numericTypes[dataType]:
	case (dataType == "binary" || dataType == "varbinary") && strings.HasPrefix(v, "0x"):
	default:
		v = d.QuoteString(v)
	}
	return &v
}

func unescape(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] == '\\' && i+1 < len(s) && (s[i+1] == '\'' || s[i+1] == '\\') {
			i++
		}
		b.WriteByte(s[i])
	}
	return b.String()
}
