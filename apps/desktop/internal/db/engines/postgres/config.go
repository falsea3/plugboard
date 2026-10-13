package postgres

import (
	"context"
	"database/sql"
	"path"

	"github.com/relay-client/plugboard/apps/desktop/internal/db/sql/dialect"
	"github.com/relay-client/plugboard/apps/desktop/internal/model"
)

var _ dialect.ServerConfig = Dialect{}

const notApplied = "setting could not be applied"

func (Dialect) ConfigFiles(ctx context.Context, db *sql.DB) ([]model.ConfigFile, error) {
	var conf, hba, ident, data string
	err := db.QueryRowContext(ctx, `SELECT current_setting('config_file'), current_setting('hba_file'), current_setting('ident_file'), current_setting('data_directory')`).
		Scan(&conf, &hba, &ident, &data)
	if err != nil {
		return nil, err
	}
	return []model.ConfigFile{
		{Name: "postgresql.conf", Path: conf},
		{Name: "pg_hba.conf", Path: hba},
		{Name: "pg_ident.conf", Path: ident},
		{Name: "postgresql.auto.conf", Path: path.Join(data, "postgresql.auto.conf")},
	}, nil
}

func (Dialect) ReadServerFile(ctx context.Context, db *sql.DB, file string) (string, error) {
	var text string
	err := db.QueryRowContext(ctx, `SELECT pg_read_file($1)`, file).Scan(&text)
	return text, err
}

func (Dialect) WriteServerFile(ctx context.Context, db *sql.DB, file, text string) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var oid uint32
	if err := tx.QueryRowContext(ctx, `SELECT lo_from_bytea(0, $1)`, []byte(text)).Scan(&oid); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `SELECT lo_export($1, $2)`, oid, file); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `SELECT lo_unlink($1)`, oid); err != nil {
		return err
	}
	return tx.Commit()
}

func (Dialect) CheckConfig(ctx context.Context, db *sql.DB, file string) (problems []model.ConfigProblem, restart []string) {
	problems = []model.ConfigProblem{}
	rows, err := db.QueryContext(ctx, `SELECT COALESCE(sourceline, 0), COALESCE(name, ''), error FROM pg_file_settings WHERE sourcefile = $1 AND error IS NOT NULL ORDER BY sourceline`, file)
	if err == nil {
		for rows.Next() {
			var p model.ConfigProblem
			var name string
			if rows.Scan(&p.Line, &name, &p.Message) != nil {
				continue
			}
			if p.Message == notApplied && name != "" {
				restart = append(restart, name)
				continue
			}
			problems = append(problems, p)
		}
		rows.Close()
	}
	rows, err = db.QueryContext(ctx, `SELECT line_number, error FROM pg_hba_file_rules WHERE error IS NOT NULL AND current_setting('hba_file') = $1 ORDER BY line_number`, file)
	if err == nil {
		for rows.Next() {
			var p model.ConfigProblem
			if rows.Scan(&p.Line, &p.Message) == nil {
				problems = append(problems, p)
			}
		}
		rows.Close()
	}
	return problems, restart
}

func (Dialect) ReloadConfig(ctx context.Context, db *sql.DB) error {
	_, err := db.ExecContext(ctx, `SELECT pg_reload_conf()`)
	return err
}
