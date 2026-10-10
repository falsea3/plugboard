package mcp

import (
	"context"
	"flag"
	"io"
	"strings"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

const instructions = `Plugboard gives read access to the databases its user opened to AI agents.
Start with list_connections, then list_tables and describe_table before writing a query.
Connections marked read-only refuse anything that writes; don't try to work around it.
Keep queries small: add LIMIT, and filter large tables by an indexed column.`

func ParseFlags(args []string, stderr io.Writer) (Options, error) {
	fs := flag.NewFlagSet("plugboard mcp", flag.ContinueOnError)
	fs.SetOutput(stderr)
	env := fs.String("env", "", "only connections with these environment tags, comma separated (local, dev, staging, prod)")
	write := fs.Bool("write", false, "let agents write to connections that aren't read-only (never Production)")
	create := fs.Bool("create", false, "let agents add connections; they are tested first and open to agents")
	if err := fs.Parse(args); err != nil {
		return Options{}, err
	}
	var envs []string
	for _, e := range strings.Split(*env, ",") {
		if e = strings.ToLower(strings.TrimSpace(e)); e != "" {
			envs = append(envs, e)
		}
	}
	return Options{Envs: envs, Write: *write, Create: *create}, nil
}

func NewServer(a *Agent, version string) *sdk.Server {
	s := sdk.NewServer(&sdk.Implementation{Name: "plugboard", Title: "Plugboard", Version: version}, &sdk.ServerOptions{Instructions: instructions})
	readOnly := &sdk.ToolAnnotations{ReadOnlyHint: true}
	sdk.AddTool(s, &sdk.Tool{Name: "list_connections", Description: "Connections the user opened to AI agents, with their engine, environment and whether they are read-only.", Annotations: readOnly}, a.listConnections)
	sdk.AddTool(s, &sdk.Tool{Name: "list_schemas", Description: "Schemas, databases or keyspaces of a connection, and which one is the default.", Annotations: readOnly}, a.listSchemas)
	sdk.AddTool(s, &sdk.Tool{Name: "list_tables", Description: "Tables and views in a schema.", Annotations: readOnly}, a.listTables)
	sdk.AddTool(s, &sdk.Tool{Name: "describe_table", Description: "A table's columns (type, nullable, default, primary key), indexes and CREATE statement.", Annotations: readOnly}, a.describeTable)
	sdk.AddTool(s, &sdk.Tool{Name: "run_query", Description: "Runs SQL (CQL for Cassandra, commands for Redis) and returns the rows. Read-only connections refuse writes."}, a.runQuery)
	sdk.AddTool(s, &sdk.Tool{Name: "scan_keys", Description: "Redis: keys matching a pattern, a page at a time.", Annotations: readOnly}, a.scanKeys)
	sdk.AddTool(s, &sdk.Tool{Name: "read_key", Description: "Redis: a key's type, TTL and value.", Annotations: readOnly}, a.readKey)
	if a.opts.Create {
		sdk.AddTool(s, &sdk.Tool{Name: "create_connection", Description: "Adds a connection to Plugboard, open to agents. Plugboard connects first and saves it only if that works; the password goes to the system keychain. Read-only unless read_only is false."}, a.createConnection)
	}
	return s
}

func Serve(ctx context.Context, a *Agent, version string) error {
	defer a.Close()
	return NewServer(a, version).Run(ctx, &sdk.StdioTransport{})
}
