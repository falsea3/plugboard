package api

import (
	"context"
	"sync"
	"time"

	"github.com/relay-client/plugboard/apps/desktop/internal/apperr"
	"github.com/relay-client/plugboard/apps/desktop/internal/db"
	"github.com/relay-client/plugboard/apps/desktop/internal/sshtunnel"
	"github.com/relay-client/plugboard/apps/desktop/internal/store"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

var appVersion = "dev"

const (
	connectTimeout = 15 * time.Second
	catalogTimeout = 30 * time.Second
)

type App struct {
	ctx         context.Context
	connections *store.Connections
	settings    *store.Settings
	menu        *appMenu

	mu          sync.Mutex
	sessions    map[string]db.Session
	queries     map[string]context.CancelFunc
	changedKeys map[string]*sshtunnel.HostKeyChangedError
	crashLog    string
}

func NewApp() *App {
	dir := DataDir()
	return &App{
		connections: store.NewConnections(dir, store.NewSecrets(dir)),
		settings:    store.NewSettings(dir),
		sessions:    map[string]db.Session{},
		queries:     map[string]context.CancelFunc{},
		changedKeys: map[string]*sshtunnel.HostKeyChangedError{},
	}
}

func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx
	moveOldSecrets(DataDir(), a.connections, store.OldSecrets())
}

func (a *App) Shutdown(context.Context) {
	a.mu.Lock()
	defer a.mu.Unlock()
	for _, cancel := range a.queries {
		cancel()
	}
	for id, s := range a.sessions {
		s.Close()
		delete(a.sessions, id)
	}
}

func ShowWindow(a *App) {
	if a.ctx == nil {
		return
	}
	runtime.WindowUnminimise(a.ctx)
	runtime.Show(a.ctx)
}

func (a *App) schemaReader(id string) (db.SchemaReader, error) {
	s, err := a.session(id)
	if err != nil {
		return nil, err
	}
	if r, ok := s.(db.SchemaReader); ok {
		return r, nil
	}
	return nil, db.ErrNotSupported
}

func (a *App) session(id string) (db.Session, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	s, ok := a.sessions[id]
	if !ok {
		return nil, errConnectionClosed
	}
	return s, nil
}

func (a *App) rowEditor(id string) (db.RowEditor, error) {
	s, err := a.session(id)
	if err != nil {
		return nil, err
	}
	if ed, ok := s.(db.RowEditor); ok {
		return ed, nil
	}
	return nil, db.ErrNotSupported
}

func (a *App) structureEditor(id string) (db.StructureEditor, error) {
	s, err := a.session(id)
	if err != nil {
		return nil, err
	}
	if ed, ok := s.(db.StructureEditor); ok {
		return ed, nil
	}
	return nil, db.ErrNotSupported
}

var errConnectionClosed = apperr.New("closed", "this connection is closed — connect again")

func (a *App) context() context.Context {
	if a.ctx != nil {
		return a.ctx
	}
	return context.Background()
}
