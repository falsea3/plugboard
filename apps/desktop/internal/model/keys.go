package model

type KeyInfo struct {
	Name string `json:"name"`
	Key  string `json:"key"`
	Type string `json:"type"`
}

type KeyPage struct {
	Keys   []KeyInfo `json:"keys"`
	Cursor string    `json:"cursor"`
	Done   bool      `json:"done"`
}

type KeyItem struct {
	Field  string `json:"field"`
	Label  string `json:"label,omitempty"`
	Value  string `json:"value"`
	Score  string `json:"score,omitempty"`
	Binary bool   `json:"binary,omitempty"`
}

type KeyValue struct {
	Key       string    `json:"key"`
	Type      string    `json:"type"`
	TTL       int64     `json:"ttl"`
	Size      int64     `json:"size"`
	Text      string    `json:"text"`
	Binary    bool      `json:"binary,omitempty"`
	Truncated bool      `json:"truncated,omitempty"`
	Items     []KeyItem `json:"items"`
	Cursor    string    `json:"cursor"`
	Done      bool      `json:"done"`
}

type KeyEdit struct {
	Key   string `json:"key"`
	Op    string `json:"op"`
	Field string `json:"field"`
	Value string `json:"value"`
	Score string `json:"score"`
	Old   string `json:"old"`
	Index int64  `json:"index"`
}
