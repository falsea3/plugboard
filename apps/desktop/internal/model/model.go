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

// Connection is a saved connection profile. Password travels from the frontend
// but is never written to connections.json — it lives in the secret store.
type Connection struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Driver       Driver `json:"driver"`
	Host         string `json:"host"`
	Port         int    `json:"port"`
	User         string `json:"user"`
	Password     string `json:"password,omitempty"`
	SavePassword bool   `json:"savePassword"`
	Database     string `json:"database"`
	File         string `json:"file"`
	SSLMode      string `json:"sslMode"`
	Env          string `json:"env"`
	Color        string `json:"color"`
	// ReadOnly opens the session in a read-only transaction mode and makes
	// the backend refuse writing statements before they reach the server.
	ReadOnly bool      `json:"readOnly"`
	SSH      SSHTunnel `json:"ssh"`
}

type SSHAuth string

const (
	SSHAuthPassword SSHAuth = "password"
	SSHAuthKey      SSHAuth = "key"
	SSHAuthAgent    SSHAuth = "agent"
)

// WithoutSecrets returns c with its database password, SSH password and key
// passphrase blanked, for sending to the UI.
func (c Connection) WithoutSecrets() Connection {
	c.Password = ""
	c.SSH.Password = ""
	c.SSH.Passphrase = ""
	return c
}

// SSHHost is the server the tunnel connects to. An empty SSH host means the
// database server itself: its database port is closed, its SSH port open.
func (c Connection) SSHHost() string {
	if strings.TrimSpace(c.SSH.Host) == "" {
		return c.Host
	}
	return c.SSH.Host
}

// SSHTunnel describes an optional jump host. Password and Passphrase travel
// like Connection.Password: never written to connections.json.
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

// ConnectSecrets are typed at connect time for profiles that don't save them.
type ConnectSecrets struct {
	Password      string `json:"password"`
	SSHPassword   string `json:"sshPassword"`
	SSHPassphrase string `json:"sshPassphrase"`
}

// ConnectResult is an open session, or the SSH host key change that stopped
// it opening, for the user to look at and trust or not.
type ConnectResult struct {
	Session       *SessionInfo   `json:"session,omitempty"`
	HostKeyChange *HostKeyChange `json:"hostKeyChange,omitempty"`
}

// HostKeyChange is an SSH server presenting a different key than the one
// remembered for it.
type HostKeyChange struct {
	Host        string `json:"host"`
	Fingerprint string `json:"fingerprint"`
}

// TunnelState is sent when an open session's SSH tunnel drops or comes back.
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
	Kind   string `json:"kind"` // "table" | "view"
}

type Column struct {
	Name       string  `json:"name"`
	Type       string  `json:"type"`
	Nullable   bool    `json:"nullable"`
	Default    *string `json:"default"`
	PrimaryKey bool    `json:"primaryKey"`
	// Enum lists the allowed values of an enum column (PostgreSQL enum
	// types, MySQL ENUM), in declaration order; nil for other columns.
	Enum []string `json:"enum"`
	// Binary columns hold bytes the grid shows as a hex preview, so it
	// can't write them back.
	Binary bool `json:"binary"`
	// CastType is the PostgreSQL type bound text is cast to: the column's
	// type without length or precision, so an over-long value fails on write
	// instead of being cut to fit, and filters compare the value as typed.
	CastType string `json:"-"`
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
	// Pageable results came from a wrapped read: HasMore says another chunk
	// exists, fetched with RunMore from Offset+len(Rows).
	Pageable   bool    `json:"pageable"`
	HasMore    bool    `json:"hasMore"`
	Offset     int     `json:"offset"`
	DurationMs float64 `json:"durationMs"`
}

// QueryRun is the outcome of a script from the SQL editor: results of the
// statements that succeeded, and the error that stopped the rest.
type QueryRun struct {
	Results    []ResultSet `json:"results"`
	Error      string      `json:"error,omitempty"`
	ErrorIndex int         `json:"errorIndex"`
	Cancelled  bool        `json:"cancelled"`
	// RolledBack says the editor's connection was closed with a transaction
	// open (a cancel or a dropped connection), so that transaction is gone.
	RolledBack bool `json:"rolledBack"`
}

type TableQuery struct {
	Schema    string   `json:"schema"`
	Table     string   `json:"table"`
	Offset    int      `json:"offset"`
	Limit     int      `json:"limit"`
	OrderBy   string   `json:"orderBy"`
	OrderDesc bool     `json:"orderDesc"`
	Filters   []Filter `json:"filters"`
	// Keyset paging in primary-key order: the key of the last row of the
	// previous page (After) or the first row of the next one (Before).
	After  []any `json:"after"`
	Before []any `json:"before"`
	// Last asks for the final page.
	Last bool `json:"last"`
}

// Filter is one condition of the table filter bar; conditions are ANDed.
type Filter struct {
	Column string `json:"column"`
	Op     string `json:"op"`
	Value  string `json:"value"`
}

type TablePage struct {
	Result  ResultSet `json:"result"`
	HasMore bool      `json:"hasMore"`
	// DefaultOrder lists the columns rows were sorted by when no sort was
	// asked for: the primary key, so edited rows keep their place.
	DefaultOrder []string `json:"defaultOrder"`
	// HasPrev says rows exist before this page.
	HasPrev bool `json:"hasPrev"`
	// Keyset is true when pages are found by primary key rather than OFFSET;
	// the row number of a keyset page is only known if the client tracked it.
	Keyset bool `json:"keyset"`
	// Offset is the page's first row index when the server worked it out
	// (the last page in OFFSET mode), else -1.
	Offset int `json:"offset"`
}

// RowCount is a table's row count, exact or the engine's estimate.
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

// UpdateInfo is a newer release than the running app.
type UpdateInfo struct {
	Version     string `json:"version"`
	Notes       string `json:"notes"`
	PublishedAt string `json:"publishedAt"`
	// ReleaseURL is the release's GitHub page, to download it by hand.
	ReleaseURL string `json:"releaseUrl"`
}

// UpdateCheck is the answer to "is there a newer release?": Available is nil
// when this is the newest, or when the check failed and Error says why.
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
	// AutoUpdate installs a new release as soon as it is found; the app
	// switches to it on the next start. Off, the user installs it.
	AutoUpdate bool `json:"autoUpdate"`
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

// RowChange is one edited row from the grid. Key holds the primary key as it
// was read (update, delete); Values the new column values (update, insert),
// where a nil value means NULL and a missing column keeps its value/default.
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

type ApplyResult struct {
	Applied     int    `json:"applied"`
	Error       string `json:"error,omitempty"`
	FailedIndex int    `json:"failedIndex"`
}

// NewID returns a random id for connection profiles and sessions.
func NewID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
