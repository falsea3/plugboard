package sqlcore

import (
	"context"
	"database/sql"
	"io"
	"sync"
	"time"

	"github.com/relay-client/plugboard/apps/desktop/internal/db"
	"github.com/relay-client/plugboard/apps/desktop/internal/db/sql/dialect"
	"github.com/relay-client/plugboard/apps/desktop/internal/model"
)

type Session struct {
	ID      string
	Conn    model.Connection
	Dialect dialect.Dialect
	DB      *sql.DB
	Version string
	tunnel  io.Closer

	editorMu sync.Mutex
	editor   *sql.Conn
	editorTx bool
}

var (
	_ db.Session         = (*Session)(nil)
	_ db.RowEditor       = (*Session)(nil)
	_ db.StructureEditor = (*Session)(nil)
	_ db.SyntaxChecker   = (*Session)(nil)
)

func Open(ctx context.Context, id string, c model.Connection, d dialect.Dialect, opts db.OpenOptions) (*Session, error) {
	dial, tunnel, err := db.OpenTunnel(ctx, c, d.DefaultPort(), opts)
	if err != nil {
		return nil, err
	}
	closeTunnel := func() {
		if tunnel != nil {
			tunnel.Close()
		}
	}
	pool, err := d.Open(dial)
	if err != nil {
		closeTunnel()
		return nil, err
	}
	pool.SetMaxOpenConns(4)
	pool.SetConnMaxIdleTime(5 * time.Minute)
	version, err := d.ServerVersion(ctx, pool)
	if err != nil {
		pool.Close()
		closeTunnel()
		return nil, err
	}
	if v, ok := d.(dialect.Versioned); ok {
		d = v.ForVersion(version)
	}
	return &Session{ID: id, Conn: c, Dialect: d, DB: pool, Version: version, tunnel: tunnel}, nil
}

func (s *Session) Connection() model.Connection { return s.Conn }

func (s *Session) ServerVersion() string { return s.Version }

func (s *Session) Close() error {
	s.editorMu.Lock()
	if s.editor != nil {
		s.editor.Close()
		s.editor = nil
	}
	s.editorMu.Unlock()
	err := s.DB.Close()
	if s.tunnel != nil {
		s.tunnel.Close()
	}
	return err
}

func (s *Session) Info(ctx context.Context) (model.SessionInfo, error) {
	schemas, err := s.Dialect.ListSchemas(ctx, s.DB)
	if err != nil {
		return model.SessionInfo{}, err
	}
	def, err := s.Dialect.DefaultSchema(ctx, s.DB, s.Conn)
	if err != nil {
		return model.SessionInfo{}, err
	}
	if def == "" && len(schemas) > 0 {
		def = schemas[0]
	}
	return model.SessionInfo{
		SessionID:     s.ID,
		Connection:    s.Conn.WithoutSecrets(),
		ServerVersion: s.Version,
		Schemas:       schemas,
		DefaultSchema: def,
		Engine:        Features(s.Dialect),
	}, nil
}

func Features(d dialect.Dialect) model.EngineFeatures {
	return model.EngineFeatures{
		Syntax:             d.Syntax(),
		BooleanType:        d.BooleanType(),
		CanUpdateToDefault: d.CanUpdateToDefault(),
		CanAlterColumns:    d.CanAlterColumns(),
		TransactionalDDL:   d.TransactionalDDL(),
		ColumnTypes:        d.ColumnTypes(),
		Editable:           d.Editable(),
	}
}

func (s *Session) Columns(ctx context.Context, schema, table string) ([]model.Column, error) {
	return retry(ctx, s, func() ([]model.Column, error) { return s.Dialect.ListColumns(ctx, s.DB, schema, table) })
}

func (s *Session) Tables(ctx context.Context, schema string) ([]model.TableInfo, error) {
	return retry(ctx, s, func() ([]model.TableInfo, error) { return s.Dialect.ListTables(ctx, s.DB, schema) })
}

func (s *Session) qualified(schema, table string) string {
	return dialect.Qualified(s.Dialect, schema, table)
}
