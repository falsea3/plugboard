package redis

import (
	"context"
	"crypto/tls"
	"errors"
	"io"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	goredis "github.com/redis/go-redis/v9"

	"github.com/relay-client/plugboard/apps/desktop/internal/apperr"
	"github.com/relay-client/plugboard/apps/desktop/internal/db"
	"github.com/relay-client/plugboard/apps/desktop/internal/model"
)

const DefaultPort = 6379

var errNoTables = apperr.New("not_supported", "Redis has keys, not tables")

type Session struct {
	id        string
	conn      model.Connection
	opts      goredis.Options
	version   string
	databases int
	tunnel    io.Closer

	mu      sync.Mutex
	clients map[int]*goredis.Client

	consoleMu sync.Mutex
	console   *goredis.Conn
	consoleDB int

	flagsMu sync.Mutex
	flags   map[string]cmdFlags
}

func Open(ctx context.Context, id string, c model.Connection, opts db.OpenOptions) (*Session, error) {
	dial, tunnel, err := db.OpenTunnel(ctx, c, DefaultPort, opts)
	if err != nil {
		return nil, err
	}
	s := &Session{id: id, conn: c, tunnel: tunnel, consoleDB: -1, clients: map[int]*goredis.Client{}, flags: map[string]cmdFlags{}}
	s.opts = options(c, dial)
	if err := s.start(ctx); err != nil {
		s.Close()
		return nil, err
	}
	return s, nil
}

func options(c, dial model.Connection) goredis.Options {
	port := dial.Port
	if port == 0 {
		port = DefaultPort
	}
	o := goredis.Options{
		Addr:            net.JoinHostPort(dial.Host, strconv.Itoa(port)),
		Username:        c.User,
		Password:        c.Password,
		DialTimeout:     10 * time.Second,
		ReadTimeout:     -1,
		PoolSize:        4,
		MaxRetries:      1,
		DisableIdentity: true,
	}
	switch c.SSLMode {
	case "require":
		o.TLSConfig = &tls.Config{InsecureSkipVerify: true}
	case "verify-ca", "verify-full":
		o.TLSConfig = &tls.Config{ServerName: c.Host}
	}
	return o
}

func (s *Session) start(ctx context.Context) error {
	first, err := dbIndex(s.conn.Database)
	if err != nil {
		return err
	}
	c, err := s.client(first, true)
	if err != nil {
		return err
	}
	info, err := c.Info(ctx, "server", "keyspace").Result()
	if err != nil {
		return classify(err)
	}
	s.version = versionOf(info)
	s.databases = databasesOf(ctx, c, info)
	if first >= s.databases {
		return apperr.New("bad_database", "this server has databases 0–"+strconv.Itoa(s.databases-1))
	}
	return nil
}

func dbIndex(s string) (int, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, nil
	}
	n, err := strconv.Atoi(s)
	if err != nil || n < 0 {
		return 0, apperr.New("bad_database", "a Redis database is a number, like 0")
	}
	return n, nil
}

func versionOf(info string) string {
	name, version := "Redis", ""
	for line := range strings.Lines(info) {
		k, v, ok := strings.Cut(strings.TrimSpace(line), ":")
		switch {
		case !ok:
		case k == "redis_version" && version == "":
			version = v
		case k == "valkey_version":
			name, version = "Valkey", v
		case k == "server_name" && v == "valkey":
			name = "Valkey"
		}
	}
	return strings.TrimSpace(name + " " + version)
}

func databasesOf(ctx context.Context, c *goredis.Client, info string) int {
	if v, err := c.ConfigGet(ctx, "databases").Result(); err == nil {
		if n, err := strconv.Atoi(v["databases"]); err == nil && n > 0 {
			return n
		}
	}
	n := 16
	for line := range strings.Lines(info) {
		k, _, ok := strings.Cut(line, ":")
		if i, err := strconv.Atoi(strings.TrimPrefix(k, "db")); ok && strings.HasPrefix(k, "db") && err == nil && i+1 > n {
			n = i + 1
		}
	}
	return n
}

func (s *Session) client(n int, any bool) (*goredis.Client, error) {
	if !any && (n < 0 || n >= s.databases) {
		return nil, apperr.New("bad_database", "no such database: "+strconv.Itoa(n))
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if c, ok := s.clients[n]; ok {
		return c, nil
	}
	o := s.opts
	o.DB = n
	c := goredis.NewClient(&o)
	s.clients[n] = c
	return c, nil
}

func (s *Session) clientFor(schema string) (*goredis.Client, error) {
	n, err := dbIndex(schema)
	if err != nil {
		return nil, err
	}
	return s.client(n, false)
}

func (s *Session) Info(ctx context.Context) (model.SessionInfo, error) {
	schemas := make([]string, s.databases)
	for i := range schemas {
		schemas[i] = strconv.Itoa(i)
	}
	def, _ := dbIndex(s.conn.Database)
	return model.SessionInfo{
		SessionID:     s.id,
		Connection:    s.conn.WithoutSecrets(),
		ServerVersion: s.version,
		Schemas:       schemas,
		DefaultSchema: strconv.Itoa(def),
		Engine:        Features(),
	}, nil
}

func (s *Session) Connection() model.Connection { return s.conn }

func (s *Session) ServerVersion() string { return s.version }

func (s *Session) Close() error {
	s.consoleMu.Lock()
	if s.console != nil {
		s.console.Close()
		s.console = nil
	}
	s.consoleMu.Unlock()
	s.mu.Lock()
	var errs []error
	for n, c := range s.clients {
		errs = append(errs, c.Close())
		delete(s.clients, n)
	}
	s.mu.Unlock()
	if s.tunnel != nil {
		errs = append(errs, s.tunnel.Close())
	}
	return errors.Join(errs...)
}

func (s *Session) Tables(context.Context, string) ([]model.TableInfo, error) { return nil, errNoTables }

func (s *Session) Columns(context.Context, string, string) ([]model.Column, error) {
	return nil, errNoTables
}

func (s *Session) TablePage(context.Context, model.TableQuery) (model.TablePage, error) {
	return model.TablePage{}, errNoTables
}

func (s *Session) CountRows(context.Context, model.TableQuery, bool) (model.RowCount, error) {
	return model.RowCount{}, errNoTables
}

func (s *Session) RunMore(context.Context, string, int) (model.ResultSet, error) {
	return model.ResultSet{}, db.ErrNotSupported
}

func Features() model.EngineFeatures {
	return model.EngineFeatures{KeyValue: true, ColumnTypes: []string{}}
}
