package sqlite

import (
	"strconv"

	"github.com/falsea3/plugboard/apps/desktop/internal/model"
)

func (Dialect) CutValue(expr string, kind model.ColumnKind, chars int) string {
	if kind != model.KindText && kind != model.KindJSON {
		return ""
	}
	return "substr(" + expr + ", 1, " + strconv.Itoa(chars) + ")"
}
