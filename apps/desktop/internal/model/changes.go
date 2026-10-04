package model

type ChangeKind string

const (
	ChangeUpdate ChangeKind = "update"
	ChangeInsert ChangeKind = "insert"
	ChangeDelete ChangeKind = "delete"
)

type RowChange struct {
	Kind   ChangeKind     `json:"kind"`
	Key    map[string]any `json:"key"`
	Values map[string]any `json:"values"`
}

type ChangeSet struct {
	Schema  string      `json:"schema"`
	Table   string      `json:"table"`
	Changes []RowChange `json:"changes"`
}

type ColumnChange struct {
	Kind       ChangeKind `json:"kind"`
	Column     string     `json:"column"`
	Name       *string    `json:"name,omitempty"`
	Type       *string    `json:"type,omitempty"`
	Nullable   *bool      `json:"nullable,omitempty"`
	DefaultSet bool       `json:"defaultSet"`
	Default    *string    `json:"default"`
}

type StructureChange struct {
	Schema  string         `json:"schema"`
	Table   string         `json:"table"`
	Changes []ColumnChange `json:"changes"`
}

type ApplyResult struct {
	Applied     int    `json:"applied"`
	Error       string `json:"error,omitempty"`
	FailedIndex int    `json:"failedIndex"`
	Partial     bool   `json:"partial"`
	Cancelled   bool   `json:"cancelled"`
}
