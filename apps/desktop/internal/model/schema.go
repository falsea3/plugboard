package model

type TableInfo struct {
	Schema string `json:"schema"`
	Name   string `json:"name"`
	Kind   string `json:"kind"`
}

type Column struct {
	Name       string     `json:"name"`
	Type       string     `json:"type"`
	Nullable   bool       `json:"nullable"`
	Default    *string    `json:"default"`
	PrimaryKey bool       `json:"primaryKey"`
	Enum       []string   `json:"enum"`
	Kind       ColumnKind `json:"kind"`
	CastType   string     `json:"-"`
}

type Relation struct {
	Name       string   `json:"name"`
	Table      string   `json:"table"`
	Columns    []string `json:"columns"`
	RefSchema  string   `json:"refSchema"`
	RefTable   string   `json:"refTable"`
	RefColumns []string `json:"refColumns"`
	OnDelete   string   `json:"onDelete"`
	OnUpdate   string   `json:"onUpdate"`
}

type Index struct {
	Name       string   `json:"name"`
	Columns    []string `json:"columns"`
	Unique     bool     `json:"unique"`
	Primary    bool     `json:"primary"`
	Method     string   `json:"method"`
	Where      string   `json:"where"`
	Definition string   `json:"definition"`
}

type DiagramTable struct {
	Name    string   `json:"name"`
	Kind    string   `json:"kind"`
	Columns []Column `json:"columns"`
}

type Diagram struct {
	Schema    string         `json:"schema"`
	Tables    []DiagramTable `json:"tables"`
	Relations []Relation     `json:"relations"`
}

type ColumnKind string

const (
	KindOther    ColumnKind = ""
	KindNumber   ColumnKind = "number"
	KindBool     ColumnKind = "bool"
	KindDateTime ColumnKind = "datetime"
	KindText     ColumnKind = "text"
	KindJSON     ColumnKind = "json"
	KindBinary   ColumnKind = "binary"
)

type DBObject struct {
	Schema string   `json:"schema"`
	Name   string   `json:"name"`
	Kind   string   `json:"kind"`
	Detail string   `json:"detail,omitempty"`
	Values []string `json:"values,omitempty"`
}

type NewIndex struct {
	Name    string   `json:"name"`
	Columns []string `json:"columns"`
	Unique  bool     `json:"unique"`
}
