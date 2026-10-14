package clickhouse

import (
	"context"
	"database/sql"
	"strings"

	"github.com/falsea3/plugboard/apps/desktop/internal/apperr"
	"github.com/falsea3/plugboard/apps/desktop/internal/db/sql/dialect"
	"github.com/falsea3/plugboard/apps/desktop/internal/model"
)

var _ dialect.ServerConfig = Dialect{}

const serverSettings = "system.server_settings"

func (Dialect) ConfigFiles(context.Context, *sql.DB) ([]model.ConfigFile, error) {
	return []model.ConfigFile{
		{Name: "Server settings", Path: serverSettings},
		{Name: "config.xml", Path: "/etc/clickhouse-server/config.xml"},
		{Name: "users.xml", Path: "/etc/clickhouse-server/users.xml"},
	}, nil
}

func (Dialect) ReadServerFile(ctx context.Context, db *sql.DB, file string) (model.ConfigText, error) {
	if file != serverSettings {
		return model.ConfigText{}, needsSSH(file)
	}
	rows, err := db.QueryContext(ctx, `SELECT name, value, changed, changeable_without_restart FROM system.server_settings ORDER BY name`)
	if err != nil {
		return model.ConfigText{}, err
	}
	defer rows.Close()
	var b strings.Builder
	b.WriteString("# What the server runs with now. Read-only: change config.xml (or a file in config.d) and save it to reload.\n")
	b.WriteString("# changed: set in the config files · restart: needs a restart to change\n")
	for rows.Next() {
		var name, value, live string
		var changed uint8
		if err := rows.Scan(&name, &value, &changed, &live); err != nil {
			return model.ConfigText{}, err
		}
		var marks []string
		if changed == 1 {
			marks = append(marks, "changed")
		}
		if live == "No" {
			marks = append(marks, "restart")
		}
		line := name + ": " + value
		if len(marks) > 0 {
			line += "  # " + strings.Join(marks, ", ")
		}
		b.WriteString(line + "\n")
	}
	return model.ConfigText{Path: file, Text: b.String(), Via: "live", ReadOnly: true}, rows.Err()
}

func needsSSH(file string) error {
	return apperr.New("not_supported", "ClickHouse can't read or write files for a client — connect through SSH to the machine the server runs on to open "+file)
}

func (Dialect) WriteServerFile(_ context.Context, _ *sql.DB, file, _ string) error {
	if file == serverSettings {
		return apperr.New("not_supported", "these settings are read-only — change config.xml and save it")
	}
	return needsSSH(file)
}

func (Dialect) CheckConfig(context.Context, *sql.DB, string) ([]model.ConfigProblem, []string) {
	return []model.ConfigProblem{}, nil
}

func (Dialect) ReloadConfig(ctx context.Context, db *sql.DB) error {
	_, err := db.ExecContext(ctx, `SYSTEM RELOAD CONFIG`)
	return err
}
