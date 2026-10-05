package dialect

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/relay-client/plugboard/apps/desktop/internal/model"
)

func QueryObjects(ctx context.Context, db *sql.DB, schema, query string, args ...any) ([]model.DBObject, error) {
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.DBObject{}
	for rows.Next() {
		var o model.DBObject
		var detail, values, owner sql.NullString
		if err := rows.Scan(&o.Name, &o.Kind, &detail, &values, &owner); err != nil {
			return nil, err
		}
		o.Schema, o.Detail = schema, detail.String
		if owner.Valid {
			o.Schema = owner.String
		}
		if values.Valid {
			if err := json.Unmarshal([]byte(values.String), &o.Values); err != nil {
				return nil, fmt.Errorf("values of %s: %w", o.Name, err)
			}
		}
		out = append(out, o)
	}
	return out, rows.Err()
}

func QueryLines(ctx context.Context, db *sql.DB, query string, args ...any) ([]string, error) {
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []string{}
	for rows.Next() {
		var s sql.NullString
		if err := rows.Scan(&s); err != nil {
			return nil, err
		}
		if s.Valid && s.String != "" {
			out = append(out, s.String)
		}
	}
	return out, rows.Err()
}

func ShowCreate(ctx context.Context, db *sql.DB, obj model.DBObject, query string, column int) (string, error) {
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return "", err
	}
	defer rows.Close()
	cols, err := rows.Columns()
	if err != nil {
		return "", err
	}
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return "", err
		}
		return "", Gone(obj)
	}
	vals := make([]sql.NullString, len(cols))
	dest := make([]any, len(cols))
	for i := range vals {
		dest[i] = &vals[i]
	}
	if err := rows.Scan(dest...); err != nil {
		return "", err
	}
	if column >= len(vals) || !vals[column].Valid {
		return "", fmt.Errorf("the server didn't return the definition of %s — the user may lack the privilege to see it", obj.Name)
	}
	return Statement(vals[column].String), nil
}

func Gone(obj model.DBObject) error {
	return fmt.Errorf("%s %s isn't there any more — reload the tree", obj.Kind, obj.Name)
}

func NoRows(err error, obj model.DBObject) error {
	if errors.Is(err, sql.ErrNoRows) {
		return Gone(obj)
	}
	return err
}

func Statement(s string) string {
	s = strings.TrimRight(strings.TrimSpace(s), ";")
	return s + ";"
}
