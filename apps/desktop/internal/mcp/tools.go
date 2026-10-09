package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"unicode/utf8"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/relay-client/plugboard/apps/desktop/internal/db"
	"github.com/relay-client/plugboard/apps/desktop/internal/model"
)

const (
	defaultRows = 100
	maxRows     = 1000
	maxCell     = 1000
)

type connArg struct {
	Connection string `json:"connection" jsonschema:"connection name from list_connections"`
}

type schemaArg struct {
	Connection string `json:"connection" jsonschema:"connection name from list_connections"`
	Schema     string `json:"schema,omitempty" jsonschema:"schema, database or keyspace; the connection's default when empty"`
}

type tableArg struct {
	Connection string `json:"connection" jsonschema:"connection name from list_connections"`
	Schema     string `json:"schema,omitempty" jsonschema:"schema, database or keyspace; the connection's default when empty"`
	Table      string `json:"table" jsonschema:"table or view name"`
}

type queryArg struct {
	Connection string `json:"connection" jsonschema:"connection name from list_connections"`
	SQL        string `json:"sql" jsonschema:"SQL (or CQL, or Redis commands one per line) to run; several statements run in order"`
	MaxRows    int    `json:"max_rows,omitempty" jsonschema:"rows to return per result, 100 by default, at most 1000"`
}

type keysArg struct {
	Connection string `json:"connection" jsonschema:"connection name from list_connections"`
	DB         string `json:"db,omitempty" jsonschema:"Redis database number; the connection's default when empty"`
	Pattern    string `json:"pattern,omitempty" jsonschema:"glob pattern such as user:*; every key when empty"`
	Cursor     string `json:"cursor,omitempty" jsonschema:"cursor from the previous page"`
}

type keyArg struct {
	Connection string `json:"connection" jsonschema:"connection name from list_connections"`
	DB         string `json:"db,omitempty" jsonschema:"Redis database number; the connection's default when empty"`
	Key        string `json:"key" jsonschema:"the key, as scan_keys listed it (its key field)"`
}

func text(v any) (*sdk.CallToolResult, any, error) {
	b, err := json.MarshalIndent(v, "", " ")
	if err != nil {
		return nil, nil, err
	}
	return &sdk.CallToolResult{Content: []sdk.Content{&sdk.TextContent{Text: string(b)}}}, nil, nil
}

func (a *Agent) defaultSchema(ctx context.Context, s db.Session, schema string) (string, error) {
	if schema != "" {
		return schema, nil
	}
	info, err := s.Info(ctx)
	return info.DefaultSchema, err
}

func access(a *Agent, c model.Connection) string {
	if a.canWrite(c) {
		return "read-write"
	}
	return "read-only"
}

func (a *Agent) listConnections(context.Context, *sdk.CallToolRequest, struct{}) (*sdk.CallToolResult, any, error) {
	list, err := a.Connections()
	if err != nil {
		return nil, nil, err
	}
	type item struct {
		Name     string `json:"name"`
		Engine   string `json:"engine"`
		Env      string `json:"env,omitempty"`
		Database string `json:"database,omitempty"`
		Access   string `json:"access"`
	}
	out := []item{}
	for _, c := range list {
		out = append(out, item{c.Name, string(c.Driver), c.Env, c.Database, access(a, c)})
	}
	a.audit("list_connections", "", "")
	return text(out)
}

func (a *Agent) listSchemas(ctx context.Context, _ *sdk.CallToolRequest, in connArg) (*sdk.CallToolResult, any, error) {
	s, _, err := a.session(ctx, in.Connection)
	if err != nil {
		return nil, nil, err
	}
	info, err := s.Info(ctx)
	if err != nil {
		return nil, nil, err
	}
	a.audit("list_schemas", in.Connection, "")
	return text(map[string]any{"schemas": info.Schemas, "default": info.DefaultSchema, "server": info.ServerVersion})
}

func (a *Agent) listTables(ctx context.Context, _ *sdk.CallToolRequest, in schemaArg) (*sdk.CallToolResult, any, error) {
	s, _, err := a.session(ctx, in.Connection)
	if err != nil {
		return nil, nil, err
	}
	schema, err := a.defaultSchema(ctx, s, in.Schema)
	if err != nil {
		return nil, nil, err
	}
	tables, err := s.Tables(ctx, schema)
	if err != nil {
		return nil, nil, err
	}
	a.audit("list_tables", in.Connection, schema)
	return text(map[string]any{"schema": schema, "tables": tables})
}

func (a *Agent) describeTable(ctx context.Context, _ *sdk.CallToolRequest, in tableArg) (*sdk.CallToolResult, any, error) {
	s, _, err := a.session(ctx, in.Connection)
	if err != nil {
		return nil, nil, err
	}
	schema, err := a.defaultSchema(ctx, s, in.Schema)
	if err != nil {
		return nil, nil, err
	}
	cols, err := s.Columns(ctx, schema, in.Table)
	if err != nil {
		return nil, nil, err
	}
	if len(cols) == 0 {
		return nil, nil, fmt.Errorf("no table %s in %s", in.Table, schema)
	}
	out := map[string]any{"schema": schema, "table": in.Table, "columns": cols}
	if r, ok := s.(db.SchemaReader); ok {
		if idx, err := r.Indexes(ctx, schema, in.Table); err == nil {
			out["indexes"] = idx
		}
		if ddl, err := r.DDL(ctx, model.DBObject{Schema: schema, Name: in.Table, Kind: "table"}); err == nil {
			out["ddl"] = ddl
		}
	}
	a.audit("describe_table", in.Connection, schema+"."+in.Table)
	return text(out)
}

func (a *Agent) runQuery(ctx context.Context, _ *sdk.CallToolRequest, in queryArg) (*sdk.CallToolResult, any, error) {
	s, _, err := a.session(ctx, in.Connection)
	if err != nil {
		return nil, nil, err
	}
	limit := in.MaxRows
	if limit <= 0 {
		limit = defaultRows
	}
	limit = min(limit, maxRows)
	a.audit("run_query", in.Connection, in.SQL)
	results, err := s.Run(ctx, in.SQL)
	out := make([]map[string]any, len(results))
	for i, rs := range results {
		out[i] = shape(rs, limit)
	}
	if err != nil {
		return text(map[string]any{"results": out, "error": err.Error()})
	}
	return text(map[string]any{"results": out})
}

func shape(rs model.ResultSet, limit int) map[string]any {
	names := make([]string, len(rs.Columns))
	for i, c := range rs.Columns {
		names[i] = c.Name
	}
	rows := rs.Rows
	more := rs.HasMore || rs.Truncated
	if len(rows) > limit {
		rows, more = rows[:limit], true
	}
	cut := make([][]any, len(rows))
	for i, r := range rows {
		cut[i] = make([]any, len(r))
		for j, v := range r {
			cut[i][j] = cell(v)
		}
	}
	out := map[string]any{"statement": rs.Statement}
	if rs.HasRows {
		out["columns"], out["rows"], out["more_rows"] = names, cut, more
	} else {
		out["rows_affected"] = rs.RowsAffected
	}
	return out
}

func cell(v any) any {
	s, ok := v.(string)
	if !ok || utf8.RuneCountInString(s) <= maxCell {
		return v
	}
	r := []rune(s)
	return fmt.Sprintf("%s… (%d characters)", string(r[:maxCell]), len(r))
}

func (a *Agent) keyStore(ctx context.Context, conn, dbName string) (db.KeyStore, string, error) {
	s, _, err := a.session(ctx, conn)
	if err != nil {
		return nil, "", err
	}
	k, ok := s.(db.KeyStore)
	if !ok {
		return nil, "", fmt.Errorf("%s isn't a key-value database — use list_tables and run_query", conn)
	}
	name, err := a.defaultSchema(ctx, s, dbName)
	return k, name, err
}

func (a *Agent) scanKeys(ctx context.Context, _ *sdk.CallToolRequest, in keysArg) (*sdk.CallToolResult, any, error) {
	k, dbName, err := a.keyStore(ctx, in.Connection, in.DB)
	if err != nil {
		return nil, nil, err
	}
	page, err := k.ScanKeys(ctx, dbName, in.Pattern, in.Cursor, 200)
	if err != nil {
		return nil, nil, err
	}
	a.audit("scan_keys", in.Connection, dbName+" "+in.Pattern)
	return text(page)
}

func (a *Agent) readKey(ctx context.Context, _ *sdk.CallToolRequest, in keyArg) (*sdk.CallToolResult, any, error) {
	k, dbName, err := a.keyStore(ctx, in.Connection, in.DB)
	if err != nil {
		return nil, nil, err
	}
	v, err := k.ReadKey(ctx, dbName, in.Key, "")
	if err != nil {
		return nil, nil, err
	}
	if t, ok := cell(v.Text).(string); ok {
		v.Text = t
	}
	a.audit("read_key", in.Connection, dbName+" "+in.Key)
	return text(v)
}
