package model

type DumpOptions struct {
	Schemas   []string `json:"schemas"`
	Structure bool     `json:"structure"`
	Data      bool     `json:"data"`
	Clean     bool     `json:"clean"`
	Inserts   bool     `json:"inserts"`
	Owners    bool     `json:"owners"`
}

type DumpProgress struct {
	Phase  string `json:"phase"`
	Object string `json:"object"`
	Tables int    `json:"tables"`
	Done   int    `json:"done"`
	Rows   int64  `json:"rows"`
	Bytes  int64  `json:"bytes"`
}

type ImportOptions struct {
	OneTransaction bool `json:"oneTransaction"`
	StopOnError    bool `json:"stopOnError"`
}

type ImportProgress struct {
	Bytes      int64  `json:"bytes"`
	Total      int64  `json:"total"`
	Statements int    `json:"statements"`
	Errors     int    `json:"errors"`
	Current    string `json:"current"`
}

type ImportError struct {
	Index     int    `json:"index"`
	Statement string `json:"statement"`
	Message   string `json:"message"`
}

type ImportResult struct {
	Statements int           `json:"statements"`
	Rows       int64         `json:"rows"`
	Skipped    []string      `json:"skipped"`
	Errors     []ImportError `json:"errors"`
	Stopped    bool          `json:"stopped"`
	RolledBack bool          `json:"rolledBack"`
}

type DumpResult struct {
	Path   string `json:"path"`
	Bytes  int64  `json:"bytes"`
	Rows   int64  `json:"rows"`
	Tables int    `json:"tables"`
}

type DumpFile struct {
	Path string `json:"path"`
	Name string `json:"name"`
	Size int64  `json:"size"`
}
