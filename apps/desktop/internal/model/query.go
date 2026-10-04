package model

type ResultColumn struct {
	Name string     `json:"name"`
	Type string     `json:"type"`
	Kind ColumnKind `json:"kind"`
}

type ResultSet struct {
	Statement    string         `json:"statement"`
	Columns      []ResultColumn `json:"columns"`
	Rows         [][]any        `json:"rows"`
	RowsAffected int64          `json:"rowsAffected"`
	HasRows      bool           `json:"hasRows"`
	Truncated    bool           `json:"truncated"`
	Pageable     bool           `json:"pageable"`
	HasMore      bool           `json:"hasMore"`
	Offset       int            `json:"offset"`
	DurationMs   float64        `json:"durationMs"`
}

type QueryRun struct {
	Results       []ResultSet `json:"results"`
	Error         string      `json:"error,omitempty"`
	ErrorIndex    int         `json:"errorIndex"`
	ErrorPosition int         `json:"errorPosition"`
	Cancelled     bool        `json:"cancelled"`
	RolledBack    bool        `json:"rolledBack"`
}

type SyntaxProblem struct {
	Index    int    `json:"index"`
	Position int    `json:"position"`
	Message  string `json:"message"`
}

type TableQuery struct {
	Schema    string   `json:"schema"`
	Table     string   `json:"table"`
	Offset    int      `json:"offset"`
	Limit     int      `json:"limit"`
	OrderBy   string   `json:"orderBy"`
	OrderDesc bool     `json:"orderDesc"`
	Filters   []Filter `json:"filters"`
	After     []any    `json:"after"`
	Before    []any    `json:"before"`
	Last      bool     `json:"last"`
}

type Filter struct {
	Column string `json:"column"`
	Op     string `json:"op"`
	Value  string `json:"value"`
}

type TablePage struct {
	Result       ResultSet `json:"result"`
	HasMore      bool      `json:"hasMore"`
	DefaultOrder []string  `json:"defaultOrder"`
	HasPrev      bool      `json:"hasPrev"`
	Keyset       bool      `json:"keyset"`
	Offset       int       `json:"offset"`
}

type RowCount struct {
	Count int64 `json:"count"`
	Exact bool  `json:"exact"`
	Known bool  `json:"known"`
}
