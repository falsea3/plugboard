package model

import (
	"crypto/rand"
	"encoding/hex"
	"strings"
)

type Driver string

const (
	Postgres Driver = "postgres"
	MySQL    Driver = "mysql"
	SQLite   Driver = "sqlite"
)

type Connection struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Driver       Driver    `json:"driver"`
	Host         string    `json:"host"`
	Port         int       `json:"port"`
	User         string    `json:"user"`
	Password     string    `json:"password,omitempty"`
	SavePassword bool      `json:"savePassword"`
	Database     string    `json:"database"`
	File         string    `json:"file"`
	SSLMode      string    `json:"sslMode"`
	Env          string    `json:"env"`
	Color        string    `json:"color"`
	ReadOnly     bool      `json:"readOnly"`
	SSH          SSHTunnel `json:"ssh"`
}

type SSHAuth string

const (
	SSHAuthPassword SSHAuth = "password"
	SSHAuthKey      SSHAuth = "key"
	SSHAuthAgent    SSHAuth = "agent"
)

func (c Connection) WithoutSecrets() Connection {
	c.Password = ""
	c.SSH.Password = ""
	c.SSH.Passphrase = ""
	return c
}

func (c Connection) SSHHost() string {
	if strings.TrimSpace(c.SSH.Host) == "" {
		return c.Host
	}
	return c.SSH.Host
}

type SSHTunnel struct {
	Enabled    bool    `json:"enabled"`
	Host       string  `json:"host"`
	Port       int     `json:"port"`
	User       string  `json:"user"`
	Auth       SSHAuth `json:"auth"`
	KeyFile    string  `json:"keyFile"`
	Password   string  `json:"password,omitempty"`
	Passphrase string  `json:"passphrase,omitempty"`
}

type TestResult struct {
	Ok            bool    `json:"ok"`
	Error         string  `json:"error,omitempty"`
	ServerVersion string  `json:"serverVersion,omitempty"`
	LatencyMs     float64 `json:"latencyMs"`
}

type ConnectSecrets struct {
	Password      string `json:"password"`
	SSHPassword   string `json:"sshPassword"`
	SSHPassphrase string `json:"sshPassphrase"`
}

type ConnectResult struct {
	Session       *SessionInfo   `json:"session,omitempty"`
	HostKeyChange *HostKeyChange `json:"hostKeyChange,omitempty"`
}

type HostKeyChange struct {
	Host        string `json:"host"`
	Fingerprint string `json:"fingerprint"`
}

type TunnelState struct {
	SessionID string `json:"sessionId"`
	State     string `json:"state"`
	Error     string `json:"error,omitempty"`
}

type SessionInfo struct {
	SessionID     string     `json:"sessionId"`
	Connection    Connection `json:"connection"`
	ServerVersion string     `json:"serverVersion"`
	Schemas       []string   `json:"schemas"`
	DefaultSchema string     `json:"defaultSchema"`
}

type TableInfo struct {
	Schema string `json:"schema"`
	Name   string `json:"name"`
	Kind   string `json:"kind"`
}

type Column struct {
	Name       string   `json:"name"`
	Type       string   `json:"type"`
	Nullable   bool     `json:"nullable"`
	Default    *string  `json:"default"`
	PrimaryKey bool     `json:"primaryKey"`
	Enum       []string `json:"enum"`
	Binary     bool     `json:"binary"`
	CastType   string   `json:"-"`
}

type ResultColumn struct {
	Name string `json:"name"`
	Type string `json:"type"`
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
	Results    []ResultSet `json:"results"`
	Error      string      `json:"error,omitempty"`
	ErrorIndex int         `json:"errorIndex"`
	Cancelled  bool        `json:"cancelled"`
	RolledBack bool        `json:"rolledBack"`
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

type AppInfo struct {
	Name      string `json:"name"`
	Version   string `json:"version"`
	GoVersion string `json:"goVersion"`
	Platform  string `json:"platform"`
	DataDir   string `json:"dataDir"`
	Copyright string `json:"copyright"`
}

type UpdateInfo struct {
	Version     string `json:"version"`
	Notes       string `json:"notes"`
	PublishedAt string `json:"publishedAt"`
	ReleaseURL  string `json:"releaseUrl"`
}

type UpdateCheck struct {
	Available *UpdateInfo `json:"available,omitempty"`
	Error     string      `json:"error,omitempty"`
}

type Theme string

const (
	ThemeSystem Theme = "system"
	ThemeLight  Theme = "light"
	ThemeDark   Theme = "dark"
)

type Settings struct {
	Theme             Theme `json:"theme"`
	PageSize          int   `json:"pageSize"`
	EditorFontSize    int   `json:"editorFontSize"`
	ConfirmProdWrites bool  `json:"confirmProdWrites"`
	AutoUpdate        bool  `json:"autoUpdate"`
}

func DefaultSettings() Settings {
	return Settings{Theme: ThemeSystem, PageSize: 300, EditorFontSize: 13, ConfirmProdWrites: true}
}

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

func NewID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
