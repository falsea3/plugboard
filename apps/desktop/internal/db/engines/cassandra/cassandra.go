package cassandra

import (
	"context"
	"crypto/tls"
	"io"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	gocql "github.com/apache/cassandra-gocql-driver/v2"

	"github.com/relay-client/plugboard/apps/desktop/internal/apperr"
	"github.com/relay-client/plugboard/apps/desktop/internal/db"
	"github.com/relay-client/plugboard/apps/desktop/internal/model"
)

const DefaultPort = 9042

var errNoSort = apperr.New("not_supported", "Cassandra returns rows in key order and can't sort a whole table")

type Session struct {
	id      string
	conn    model.Connection
	cluster gocql.ClusterConfig
	version string
	tunnel  io.Closer

	mu       sync.Mutex
	sessions map[string]*gocql.Session

	editorMu sync.Mutex
	keyspace string

	pagesMu sync.Mutex
	pages   map[string][]byte
}

func Open(ctx context.Context, id string, c model.Connection, opts db.OpenOptions) (*Session, error) {
	dial, tunnel, err := db.OpenTunnel(ctx, c, DefaultPort, opts)
	if err != nil {
		return nil, err
	}
	s := &Session{id: id, conn: c, tunnel: tunnel, sessions: map[string]*gocql.Session{}, pages: map[string][]byte{}, keyspace: strings.TrimSpace(c.Database)}
	s.cluster, err = cluster(c, dial)
	if err == nil {
		err = s.start(ctx)
	}
	if err != nil {
		s.Close()
		return nil, err
	}
	return s, nil
}

func cluster(c, dial model.Connection) (gocql.ClusterConfig, error) {
	port := dial.Port
	if port == 0 {
		port = DefaultPort
	}
	ip := net.ParseIP(dial.Host)
	if ip == nil {
		addrs, err := net.LookupIP(dial.Host)
		if err != nil || len(addrs) == 0 {
			return gocql.ClusterConfig{}, err
		}
		ip = addrs[0]
	}
	cfg := gocql.NewCluster(net.JoinHostPort(ip.String(), strconv.Itoa(port)))
	cfg.AddressTranslator = gocql.AddressTranslatorFunc(func(net.IP, int) (net.IP, int) { return ip, port })
	cfg.DisableInitialHostLookup = true
	cfg.ConnectTimeout = 10 * time.Second
	cfg.Timeout = 30 * time.Second
	cfg.NumConns = 1
	cfg.Consistency = gocql.LocalOne
	if c.User != "" {
		cfg.Authenticator = gocql.PasswordAuthenticator{Username: c.User, Password: c.Password}
	}
	switch c.SSLMode {
	case "require":
		cfg.SslOpts = &gocql.SslOptions{Config: &tls.Config{InsecureSkipVerify: true}}
	case "verify-ca", "verify-full":
		cfg.SslOpts = &gocql.SslOptions{Config: &tls.Config{ServerName: c.Host}, EnableHostVerification: true}
	}
	return *cfg, nil
}

func (s *Session) start(ctx context.Context) error {
	main, err := s.session("")
	if err != nil {
		return err
	}
	var version string
	if err := main.Query(`SELECT release_version FROM system.local`).WithContext(ctx).Scan(&version); err != nil {
		return err
	}
	s.version = "Cassandra " + version
	if s.keyspace != "" {
		if _, err := s.session(s.keyspace); err != nil {
			return err
		}
	}
	return nil
}

func (s *Session) session(keyspace string) (*gocql.Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if gs, ok := s.sessions[keyspace]; ok {
		return gs, nil
	}
	cfg := s.cluster
	cfg.Keyspace = keyspace
	gs, err := cfg.CreateSession()
	if err != nil {
		return nil, describeConnect(err)
	}
	s.sessions[keyspace] = gs
	return gs, nil
}

func describeConnect(err error) error {
	msg := err.Error()
	switch {
	case strings.Contains(msg, "Provided username") || strings.Contains(msg, "authentication"):
		return apperr.New("auth", "the server refused the user name or password")
	case strings.Contains(msg, "Keyspace") && strings.Contains(msg, "does not exist"):
		return apperr.New("bad_database", "that keyspace doesn't exist")
	}
	return err
}

func (s *Session) main() (*gocql.Session, error) { return s.session("") }

func (s *Session) Info(ctx context.Context) (model.SessionInfo, error) {
	keyspaces, err := s.keyspaces(ctx)
	if err != nil {
		return model.SessionInfo{}, err
	}
	def := s.keyspace
	if def == "" && len(keyspaces) > 0 {
		def = keyspaces[0]
	}
	return model.SessionInfo{
		SessionID:     s.id,
		Connection:    s.conn.WithoutSecrets(),
		ServerVersion: s.version,
		Schemas:       keyspaces,
		DefaultSchema: def,
		Engine:        Features(),
	}, nil
}

func Features() model.EngineFeatures {
	return model.EngineFeatures{
		Syntax:      model.SQLSyntax{DollarQuotes: true},
		ColumnTypes: columnTypes,
		Editable:    true,
	}
}

func (s *Session) Connection() model.Connection { return s.conn }

func (s *Session) ServerVersion() string { return s.version }

func (s *Session) Close() error {
	s.mu.Lock()
	for k, gs := range s.sessions {
		gs.Close()
		delete(s.sessions, k)
	}
	s.mu.Unlock()
	if s.tunnel != nil {
		return s.tunnel.Close()
	}
	return nil
}

func (s *Session) WriteStatements(script string) []string {
	var out []string
	for _, st := range split(script) {
		if readOnlyReason(st) != "" {
			out = append(out, st)
		}
	}
	return out
}
