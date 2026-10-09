package mcp

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/relay-client/plugboard/apps/desktop/internal/apperr"
	"github.com/relay-client/plugboard/apps/desktop/internal/db"
	"github.com/relay-client/plugboard/apps/desktop/internal/db/engines"
	"github.com/relay-client/plugboard/apps/desktop/internal/model"
)

type Store interface {
	List() ([]model.Connection, error)
	Get(id string) (model.Connection, error)
}

type Options struct {
	Envs  []string
	Write bool
}

type Agent struct {
	store Store
	opts  Options
	dir   string

	mu       sync.Mutex
	sessions map[string]db.Session
	logMu    sync.Mutex
}

func NewAgent(store Store, dir string, opts Options) *Agent {
	return &Agent{store: store, dir: dir, opts: opts, sessions: map[string]db.Session{}}
}

var errNoAccess = apperr.New("no_access", "no such connection, or it isn't open to AI agents — turn on “AI agents (MCP)” for it in Plugboard")

func (a *Agent) allowed(c model.Connection) bool {
	if !c.AIAccess {
		return false
	}
	return len(a.opts.Envs) == 0 || slices.Contains(a.opts.Envs, strings.ToLower(c.Env))
}

func (a *Agent) Connections() ([]model.Connection, error) {
	all, err := a.store.List()
	if err != nil {
		return nil, err
	}
	out := []model.Connection{}
	for _, c := range all {
		if a.allowed(c) {
			out = append(out, c)
		}
	}
	return out, nil
}

func (a *Agent) find(name string) (model.Connection, error) {
	list, err := a.Connections()
	if err != nil {
		return model.Connection{}, err
	}
	for _, c := range list {
		if c.ID == name || strings.EqualFold(c.Name, strings.TrimSpace(name)) {
			return c, nil
		}
	}
	return model.Connection{}, errNoAccess
}

func (a *Agent) canWrite(c model.Connection) bool {
	return a.opts.Write && !c.ReadOnly && c.Env != "prod"
}

func (a *Agent) session(ctx context.Context, name string) (db.Session, model.Connection, error) {
	listed, err := a.find(name)
	if err != nil {
		return nil, model.Connection{}, err
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if s, ok := a.sessions[listed.ID]; ok {
		return s, listed, nil
	}
	c, err := a.store.Get(listed.ID)
	if err != nil {
		return nil, listed, err
	}
	c.ReadOnly = !a.canWrite(c)
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	s, err := engines.Open(ctx, "mcp-"+c.ID, c, db.OpenOptions{KnownHostsFile: filepath.Join(a.dir, "known_hosts")})
	if err != nil {
		if c.Password == "" && !c.SavePassword && c.File == "" {
			err = fmt.Errorf("%w (Plugboard doesn't keep this connection's password — turn on “Save in Keychain” for it)", err)
		}
		return nil, listed, err
	}
	a.sessions[c.ID] = s
	return s, listed, nil
}

func (a *Agent) Close() error {
	a.mu.Lock()
	defer a.mu.Unlock()
	var errs []error
	for id, s := range a.sessions {
		errs = append(errs, s.Close())
		delete(a.sessions, id)
	}
	return errors.Join(errs...)
}

func (a *Agent) audit(tool, conn, detail string) {
	a.logMu.Lock()
	defer a.logMu.Unlock()
	dir := filepath.Join(a.dir, "logs")
	if os.MkdirAll(dir, 0o700) != nil {
		return
	}
	f, err := os.OpenFile(filepath.Join(dir, "mcp.log"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return
	}
	defer f.Close()
	fmt.Fprintf(f, "%s %s %q %s\n", time.Now().Format(time.RFC3339), tool, conn, strings.Join(strings.Fields(detail), " "))
}
