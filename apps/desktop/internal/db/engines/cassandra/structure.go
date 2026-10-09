package cassandra

import (
	"context"
	"fmt"
	"strings"

	"github.com/relay-client/plugboard/apps/desktop/internal/apperr"
	"github.com/relay-client/plugboard/apps/desktop/internal/db"
	"github.com/relay-client/plugboard/apps/desktop/internal/model"
)

func (s *Session) ApplyStructure(context.Context, model.StructureChange) (int, bool, error) {
	return 0, false, db.ErrNotSupported
}

func (s *Session) PreviewStructure(ctx context.Context, sc model.StructureChange) ([]string, error) {
	cols, err := s.Columns(ctx, sc.Schema, sc.Table)
	if err != nil {
		return nil, err
	}
	byName := map[string]model.Column{}
	for _, c := range cols {
		byName[c.Name] = c
	}
	alter := "ALTER TABLE " + qualified(sc.Schema, sc.Table)
	var out []string
	for _, ch := range sc.Changes {
		switch ch.Kind {
		case model.ChangeInsert:
			if ch.Name == nil || ch.Type == nil || strings.TrimSpace(*ch.Name) == "" || strings.TrimSpace(*ch.Type) == "" {
				return nil, apperr.New("bad_column", "a new column needs a name and a type")
			}
			out = append(out, alter+" ADD "+quote(strings.TrimSpace(*ch.Name))+" "+strings.TrimSpace(*ch.Type))
			continue
		}
		cur, ok := byName[ch.Column]
		if !ok {
			return nil, fmt.Errorf("unknown column %s", ch.Column)
		}
		switch ch.Kind {
		case model.ChangeDelete:
			if cur.PrimaryKey {
				return nil, apperr.New("not_supported", "Cassandra can't drop a primary key column")
			}
			out = append(out, alter+" DROP "+quote(cur.Name))
		case model.ChangeUpdate:
			if ch.Type != nil && strings.TrimSpace(*ch.Type) != cur.Type {
				return nil, apperr.New("not_supported", "Cassandra can't change a column's type")
			}
			if ch.Name != nil && *ch.Name != cur.Name {
				if !cur.PrimaryKey {
					return nil, apperr.New("not_supported", "Cassandra only renames primary key columns")
				}
				out = append(out, alter+" RENAME "+quote(cur.Name)+" TO "+quote(strings.TrimSpace(*ch.Name)))
			}
		}
	}
	return out, nil
}

func (s *Session) RenameTableSQL(string, string, string) string { return "" }

func (s *Session) CreateIndexSQL(ks, table string, idx model.NewIndex) string {
	return "CREATE INDEX " + quote(idx.Name) + " ON " + qualified(ks, table) + " (" + quote(idx.Columns[0]) + ")"
}

func (s *Session) DropIndexSQL(ks, _, name string) string {
	return "DROP INDEX " + qualified(ks, name)
}

func (s *Session) TruncateSQL(ks, table string) string {
	return "TRUNCATE " + qualified(ks, table)
}
