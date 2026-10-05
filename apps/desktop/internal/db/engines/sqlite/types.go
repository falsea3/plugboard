package sqlite

var columnTypes = []string{"INTEGER", "REAL", "NUMERIC", "TEXT", "BLOB"}

func (Dialect) ColumnTypes() []string { return columnTypes }
