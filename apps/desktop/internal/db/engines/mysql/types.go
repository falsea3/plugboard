package mysql

var columnTypes = []string{"int", "bigint", "smallint", "tinyint", "tinyint(1)", "decimal(12,2)", "float", "double", "varchar(255)", "char(10)", "text", "mediumtext", "longtext", "date", "time", "datetime", "timestamp", "year", "json", "blob", "longblob", "binary(16)"}

func (Dialect) ColumnTypes() []string { return columnTypes }
