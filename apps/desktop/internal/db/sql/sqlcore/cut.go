package sqlcore

import (
	"strings"

	"github.com/falsea3/plugboard/apps/desktop/internal/model"
)

func (s *Session) selectList(cols []model.Column) (string, []bool) {
	cut := make([]bool, len(cols))
	parts := make([]string, len(cols))
	wrapped := false
	for i, c := range cols {
		name := s.Dialect.QuoteIdent(c.Name)
		parts[i] = name
		if c.PrimaryKey {
			continue
		}
		if expr := s.Dialect.CutValue(name, c.Kind, MaxCellBytes+1); expr != "" {
			parts[i] = expr + " AS " + name
			cut[i], wrapped = true, true
		}
	}
	if !wrapped {
		return "*", cut
	}
	return strings.Join(parts, ", "), cut
}

func cutInSQL(rs *model.ResultSet, cols []model.Column, cut []bool) {
	if len(rs.Columns) != len(cols) {
		return
	}
	for i, c := range cols {
		if cut[i] {
			rs.Columns[i].Type, rs.Columns[i].Kind = c.Type, c.Kind
		}
	}
	for i, c := range rs.Cut {
		if cut[c.Col] {
			rs.Cut[i].Size = 0
		}
	}
}
