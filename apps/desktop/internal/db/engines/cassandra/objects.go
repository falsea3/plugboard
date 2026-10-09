package cassandra

import (
	"context"
	"fmt"
	"strings"

	"github.com/relay-client/plugboard/apps/desktop/internal/model"
)

func (s *Session) Objects(ctx context.Context, ks string) ([]model.DBObject, error) {
	gs, err := s.main()
	if err != nil {
		return nil, err
	}
	out := []model.DBObject{}
	iter := gs.Query(`SELECT type_name, field_names, field_types FROM system_schema.types WHERE keyspace_name = ?`, ks).WithContext(ctx).Iter()
	var name string
	var fields, types []string
	for iter.Scan(&name, &fields, &types) {
		parts := make([]string, len(fields))
		for i := range fields {
			parts[i] = fields[i] + " " + types[i]
		}
		out = append(out, model.DBObject{Schema: ks, Name: name, Kind: "domain", Detail: "(" + strings.Join(parts, ", ") + ")"})
	}
	if err := iter.Close(); err != nil {
		return nil, err
	}
	for _, src := range []struct{ table, column, kind string }{{"functions", "function_name", "function"}, {"aggregates", "aggregate_name", "procedure"}} {
		iter := gs.Query(`SELECT `+src.column+`, argument_types FROM system_schema.`+src.table+` WHERE keyspace_name = ?`, ks).WithContext(ctx).Iter()
		var args []string
		for iter.Scan(&name, &args) {
			out = append(out, model.DBObject{Schema: ks, Name: name, Kind: src.kind, Detail: strings.Join(args, ", ")})
		}
		if err := iter.Close(); err != nil {
			return nil, err
		}
	}
	return out, nil
}

func (s *Session) DDL(ctx context.Context, obj model.DBObject) (string, error) {
	gs, err := s.main()
	if err != nil {
		return "", err
	}
	what := map[string]string{"table": "TABLE", "view": "MATERIALIZED VIEW", "domain": "TYPE", "function": "FUNCTION", "procedure": "AGGREGATE"}[obj.Kind]
	if what == "" {
		return "", fmt.Errorf("can't describe a %s", obj.Kind)
	}
	target := qualified(obj.Schema, obj.Name)
	if obj.Kind == "function" || obj.Kind == "procedure" {
		target += "(" + obj.Detail + ")"
	}
	iter := gs.Query("DESCRIBE " + what + " " + target).WithContext(ctx).Iter()
	var b strings.Builder
	var ks, typ, name, stmt string
	for iter.Scan(&ks, &typ, &name, &stmt) {
		if b.Len() > 0 {
			b.WriteString("\n\n")
		}
		b.WriteString(strings.TrimSpace(stmt))
	}
	if err := iter.Close(); err != nil {
		if strings.Contains(err.Error(), "not found") {
			return "", fmt.Errorf("%s %s isn't there any more — reload the tree", obj.Kind, obj.Name)
		}
		return "", err
	}
	return b.String(), nil
}
