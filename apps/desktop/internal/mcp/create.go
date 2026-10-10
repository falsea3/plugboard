package mcp

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/relay-client/plugboard/apps/desktop/internal/apperr"
	"github.com/relay-client/plugboard/apps/desktop/internal/db"
	"github.com/relay-client/plugboard/apps/desktop/internal/db/engines"
	"github.com/relay-client/plugboard/apps/desktop/internal/model"
	"github.com/relay-client/plugboard/apps/desktop/internal/store"
)

var drivers = []model.Driver{model.Postgres, model.MySQL, model.SQLite, model.ClickHouse, model.Cassandra, model.Redis}

type createArg struct {
	Name     string `json:"name" jsonschema:"a name for the connection, unique in Plugboard"`
	Engine   string `json:"engine" jsonschema:"postgres, mysql, sqlite, clickhouse, cassandra or redis"`
	Host     string `json:"host,omitempty" jsonschema:"server host; not for sqlite"`
	Port     int    `json:"port,omitempty" jsonschema:"server port; the engine's default when 0"`
	User     string `json:"user,omitempty" jsonschema:"user name"`
	Password string `json:"password,omitempty" jsonschema:"password; Plugboard keeps it in the system keychain"`
	Database string `json:"database,omitempty" jsonschema:"database, keyspace or Redis database number"`
	File     string `json:"file,omitempty" jsonschema:"path of the database file, for sqlite"`
	SSLMode  string `json:"ssl_mode,omitempty" jsonschema:"empty, disable, require or verify-full"`
	Env      string `json:"env,omitempty" jsonschema:"local, dev, staging or prod"`
	ReadOnly *bool  `json:"read_only,omitempty" jsonschema:"read-only connection; true when left out"`
}

func (a *Agent) createConnection(ctx context.Context, _ *sdk.CallToolRequest, in createArg) (*sdk.CallToolResult, any, error) {
	c, err := a.newConnection(in)
	if err != nil {
		return nil, nil, err
	}
	all, err := a.store.List()
	if err != nil {
		return nil, nil, err
	}
	if slices.ContainsFunc(all, func(o model.Connection) bool { return strings.EqualFold(o.Name, c.Name) }) {
		return nil, nil, apperr.New("exists", "a connection named "+c.Name+" already exists — pick another name")
	}
	a.audit("create_connection", c.Name, fmt.Sprintf("%s %s:%d %s", c.Driver, c.Host, c.Port, c.Database))
	tctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	s, err := engines.Open(tctx, "mcp-test", c, db.OpenOptions{})
	if err != nil {
		return nil, nil, fmt.Errorf("couldn't connect, so nothing was saved: %w", err)
	}
	version := s.ServerVersion()
	s.Close()
	saved, err := a.store.Save(c)
	if err != nil {
		return nil, nil, err
	}
	return text(map[string]any{"name": saved.Name, "engine": saved.Driver, "server": version, "access": access(a, saved)})
}

func (a *Agent) newConnection(in createArg) (model.Connection, error) {
	driver := model.Driver(strings.ToLower(strings.TrimSpace(in.Engine)))
	if !slices.Contains(drivers, driver) {
		return model.Connection{}, apperr.New("bad_engine", "engine is one of postgres, mysql, sqlite, clickhouse, cassandra, redis")
	}
	env := strings.ToLower(strings.TrimSpace(in.Env))
	if !slices.Contains([]string{"", "local", "dev", "staging", "prod"}, env) {
		return model.Connection{}, apperr.New("bad_env", "env is local, dev, staging or prod")
	}
	c := model.Connection{
		Name: strings.TrimSpace(in.Name), Driver: driver, Host: strings.TrimSpace(in.Host), Port: in.Port,
		User: in.User, Password: in.Password, SavePassword: true, Database: strings.TrimSpace(in.Database),
		File: strings.TrimSpace(in.File), SSLMode: in.SSLMode, Env: env, ReadOnly: in.ReadOnly == nil || *in.ReadOnly,
		AIAccess: true, SSH: model.SSHTunnel{Port: 22, Auth: model.SSHAuthPassword},
	}
	return c, store.Validate(c)
}
