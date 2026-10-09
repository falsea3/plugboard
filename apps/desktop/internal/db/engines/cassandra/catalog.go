package cassandra

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/relay-client/plugboard/apps/desktop/internal/model"
)

func isSystem(ks string) bool {
	return strings.HasPrefix(ks, "system") || ks == "dse_system" || ks == "solr_admin"
}

func (s *Session) strings(ctx context.Context, q string, args ...any) ([]string, error) {
	gs, err := s.main()
	if err != nil {
		return nil, err
	}
	iter := gs.Query(q, args...).WithContext(ctx).Iter()
	var out []string
	var v string
	for iter.Scan(&v) {
		out = append(out, v)
	}
	return out, iter.Close()
}

func (s *Session) keyspaces(ctx context.Context) ([]string, error) {
	all, err := s.strings(ctx, `SELECT keyspace_name FROM system_schema.keyspaces`)
	if err != nil {
		return nil, err
	}
	slices.SortFunc(all, func(a, b string) int {
		if isSystem(a) != isSystem(b) {
			if isSystem(a) {
				return 1
			}
			return -1
		}
		return strings.Compare(a, b)
	})
	return all, nil
}

func (s *Session) Tables(ctx context.Context, ks string) ([]model.TableInfo, error) {
	tables, err := s.strings(ctx, `SELECT table_name FROM system_schema.tables WHERE keyspace_name = ?`, ks)
	if err != nil {
		return nil, err
	}
	views, err := s.strings(ctx, `SELECT view_name FROM system_schema.views WHERE keyspace_name = ?`, ks)
	if err != nil {
		return nil, err
	}
	out := []model.TableInfo{}
	for _, t := range tables {
		out = append(out, model.TableInfo{Schema: ks, Name: t, Kind: "table"})
	}
	for _, v := range views {
		out = append(out, model.TableInfo{Schema: ks, Name: v, Kind: "view"})
	}
	slices.SortFunc(out, func(a, b model.TableInfo) int { return strings.Compare(a.Name, b.Name) })
	return out, nil
}

type columnRow struct {
	model.Column
	role     string
	position int
}

func (s *Session) columns(ctx context.Context, ks, table string) ([]columnRow, error) {
	gs, err := s.main()
	if err != nil {
		return nil, err
	}
	iter := gs.Query(`SELECT column_name, type, kind, position FROM system_schema.columns WHERE keyspace_name = ? AND table_name = ?`, ks, table).WithContext(ctx).Iter()
	var rows []columnRow
	var name, typ, role string
	var pos int
	for iter.Scan(&name, &typ, &role, &pos) {
		rows = append(rows, columnRow{Column: model.Column{Name: name, Type: typ, Nullable: role == "regular" || role == "static", PrimaryKey: role == "partition_key" || role == "clustering", Kind: kindOf(typ)}, role: role, position: pos})
	}
	if err := iter.Close(); err != nil {
		return nil, err
	}
	rank := map[string]int{"partition_key": 0, "clustering": 1, "static": 2, "regular": 3}
	slices.SortStableFunc(rows, func(a, b columnRow) int {
		if rank[a.role] != rank[b.role] {
			return rank[a.role] - rank[b.role]
		}
		if a.role == "partition_key" || a.role == "clustering" {
			return a.position - b.position
		}
		return strings.Compare(a.Name, b.Name)
	})
	return rows, nil
}

func (s *Session) Columns(ctx context.Context, ks, table string) ([]model.Column, error) {
	rows, err := s.columns(ctx, ks, table)
	if err != nil {
		return nil, err
	}
	out := make([]model.Column, len(rows))
	for i, r := range rows {
		out[i] = r.Column
	}
	return out, nil
}

func (s *Session) Relations(context.Context, string) ([]model.Relation, error) {
	return []model.Relation{}, nil
}

func (s *Session) Indexes(ctx context.Context, ks, table string) ([]model.Index, error) {
	rows, err := s.columns(ctx, ks, table)
	if err != nil {
		return nil, err
	}
	out := []model.Index{}
	var partition, clustering []string
	for _, r := range rows {
		switch r.role {
		case "partition_key":
			partition = append(partition, r.Name)
		case "clustering":
			clustering = append(clustering, r.Name)
		}
	}
	if len(partition) > 0 {
		def := "PRIMARY KEY ((" + strings.Join(partition, ", ") + ")"
		if len(clustering) > 0 {
			def += ", " + strings.Join(clustering, ", ")
		}
		out = append(out, model.Index{Name: "PRIMARY KEY", Columns: append(slices.Clone(partition), clustering...), Primary: true, Unique: true, Method: "partition", Definition: def + ")"})
	}
	gs, err := s.main()
	if err != nil {
		return nil, err
	}
	iter := gs.Query(`SELECT index_name, kind, options FROM system_schema.indexes WHERE keyspace_name = ? AND table_name = ?`, ks, table).WithContext(ctx).Iter()
	var name, kind string
	var options map[string]string
	for iter.Scan(&name, &kind, &options) {
		target := options["target"]
		method := strings.ToLower(kind)
		if c := options["class_name"]; c != "" {
			method = c[strings.LastIndex(c, ".")+1:]
		}
		out = append(out, model.Index{Name: name, Columns: []string{target}, Method: method, Definition: fmt.Sprintf("CREATE INDEX %s ON %s (%s)", quote(name), qualified(ks, table), target)})
		options = nil
	}
	return out, iter.Close()
}

func (s *Session) Diagram(ctx context.Context, ks string) (model.Diagram, error) {
	tables, err := s.Tables(ctx, ks)
	if err != nil {
		return model.Diagram{}, err
	}
	d := model.Diagram{Schema: ks, Tables: []model.DiagramTable{}, Relations: []model.Relation{}}
	for _, t := range tables {
		cols, err := s.Columns(ctx, ks, t.Name)
		if err != nil {
			return model.Diagram{}, err
		}
		d.Tables = append(d.Tables, model.DiagramTable{Name: t.Name, Kind: t.Kind, Columns: cols})
	}
	return d, nil
}
