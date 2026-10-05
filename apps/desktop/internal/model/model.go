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
	SessionID     string         `json:"sessionId"`
	Connection    Connection     `json:"connection"`
	ServerVersion string         `json:"serverVersion"`
	Schemas       []string       `json:"schemas"`
	DefaultSchema string         `json:"defaultSchema"`
	Engine        EngineFeatures `json:"engine"`
}

type EngineFeatures struct {
	Syntax             SQLSyntax `json:"syntax"`
	BooleanType        bool      `json:"booleanType"`
	CanUpdateToDefault bool      `json:"canUpdateToDefault"`
	CanAlterColumns    bool      `json:"canAlterColumns"`
	TransactionalDDL   bool      `json:"transactionalDDL"`
	ColumnTypes        []string  `json:"columnTypes"`
}

type SQLSyntax struct {
	HashComments          bool `json:"hashComments"`
	DashCommentNeedsSpace bool `json:"dashCommentNeedsSpace"`
	BackslashEscapes      bool `json:"backslashEscapes"`
	EscapeStrings         bool `json:"escapeStrings"`
	DoubleQuotedStrings   bool `json:"doubleQuotedStrings"`
	DollarQuotes          bool `json:"dollarQuotes"`
	ExecutableComments    bool `json:"executableComments"`
}

type AppError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Detail  string `json:"detail,omitempty"`
}

func NewID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
