package postgres

var columnTypes = []string{"integer", "bigint", "smallint", "serial", "bigserial", "numeric(12,2)", "real", "double precision", "boolean", "text", "varchar(255)", "char(10)", "uuid", "date", "time", "timestamp", "timestamptz", "interval", "json", "jsonb", "bytea", "inet", "text[]", "integer[]"}

func (Dialect) ColumnTypes() []string { return columnTypes }
