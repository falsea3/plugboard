package api

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"time"

	"github.com/relay-client/plugboard/apps/desktop/internal/db"
	"github.com/relay-client/plugboard/apps/desktop/internal/db/engines"
	"github.com/relay-client/plugboard/apps/desktop/internal/model"
	"github.com/relay-client/plugboard/apps/desktop/internal/sshtunnel"
	"github.com/relay-client/plugboard/apps/desktop/internal/store"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

func (a *App) ListConnections() ([]model.Connection, error) {
	return a.connections.List()
}

func (a *App) SaveConnection(c model.Connection) (model.Connection, error) {
	return a.connections.Save(c)
}

func (a *App) DeleteConnection(id string) error {
	if err := a.connections.Delete(id); err != nil {
		return err
	}
	return a.scripts.ForgetTabs(id)
}

func (a *App) TestConnection(c model.Connection) model.TestResult {
	if err := a.connections.FillSecrets(&c); err != nil {
		return model.TestResult{Error: describe(err).Message}
	}
	if c.Name == "" {
		c.Name = "test"
	}
	if err := store.Validate(c); err != nil {
		return model.TestResult{Error: describe(err).Message}
	}
	ctx, cancel := context.WithTimeout(a.context(), connectTimeout)
	defer cancel()
	start := time.Now()
	s, err := engines.Open(ctx, "test", c, a.openOptions(""))
	if err != nil {
		return model.TestResult{Error: describe(err).Message}
	}
	defer s.Close()
	return model.TestResult{
		Ok:            true,
		ServerVersion: s.ServerVersion(),
		LatencyMs:     float64(time.Since(start).Microseconds()) / 1000,
	}
}

func (a *App) Connect(id string, secrets model.ConnectSecrets) (model.ConnectResult, error) {
	c, err := a.connections.Get(id)
	if err != nil {
		return model.ConnectResult{}, err
	}
	override(&c.Password, secrets.Password)
	override(&c.SSH.Password, secrets.SSHPassword)
	override(&c.SSH.Passphrase, secrets.SSHPassphrase)
	info, err := a.open(model.NewID(), c)
	var changed *sshtunnel.HostKeyChangedError
	if errors.As(err, &changed) {
		a.mu.Lock()
		a.changedKeys[changed.Host] = changed
		a.mu.Unlock()
		return model.ConnectResult{HostKeyChange: &model.HostKeyChange{Host: changed.Host, Fingerprint: changed.Fingerprint}}, nil
	}
	if err != nil {
		return model.ConnectResult{}, err
	}
	return model.ConnectResult{Session: &info}, nil
}

func (a *App) TrustHostKey(host, fingerprint string) error {
	a.mu.Lock()
	changed := a.changedKeys[host]
	delete(a.changedKeys, host)
	a.mu.Unlock()
	if changed == nil || changed.Fingerprint != fingerprint {
		return errors.New("the server's key is not the one shown; connect again to see it")
	}
	return changed.Trust(knownHostsFile())
}

func (a *App) open(id string, c model.Connection) (model.SessionInfo, error) {
	ctx, cancel := context.WithTimeout(a.context(), connectTimeout)
	defer cancel()
	s, err := engines.Open(ctx, id, c, a.openOptions(id))
	if err != nil {
		return model.SessionInfo{}, err
	}
	info, err := s.Info(ctx)
	if err != nil {
		s.Close()
		return model.SessionInfo{}, err
	}
	a.mu.Lock()
	a.sessions[id] = s
	a.mu.Unlock()
	return info, nil
}

func (a *App) SetReadOnly(sessionID string, readOnly bool) (model.SessionInfo, error) {
	old, err := a.session(sessionID)
	if err != nil {
		return model.SessionInfo{}, err
	}
	c := old.Connection()
	c.ReadOnly = readOnly
	ctx, cancel := context.WithTimeout(a.context(), connectTimeout)
	defer cancel()
	s, err := engines.Open(ctx, sessionID, c, a.openOptions(sessionID))
	if err != nil {
		return model.SessionInfo{}, err
	}
	info, err := s.Info(ctx)
	if err != nil {
		s.Close()
		return model.SessionInfo{}, err
	}
	a.mu.Lock()
	current := a.sessions[sessionID]
	if current == old {
		a.sessions[sessionID] = s
	}
	a.mu.Unlock()
	if current != old {
		s.Close()
		return model.SessionInfo{}, errConnectionClosed
	}
	old.Close()
	return info, nil
}

func (a *App) Disconnect(sessionID string) error {
	a.mu.Lock()
	s, ok := a.sessions[sessionID]
	delete(a.sessions, sessionID)
	a.mu.Unlock()
	if !ok {
		return nil
	}
	return s.Close()
}

func (a *App) ChooseSQLiteFile() (string, error) {
	return runtime.OpenFileDialog(a.context(), runtime.OpenDialogOptions{
		Title: "Open SQLite database",
		Filters: []runtime.FileFilter{
			{DisplayName: "SQLite databases", Pattern: "*.db;*.sqlite;*.sqlite3;*.db3"},
			{DisplayName: "All files", Pattern: "*"},
		},
	})
}

func (a *App) ChooseSSHKeyFile() (string, error) {
	dir := ""
	if home, err := os.UserHomeDir(); err == nil {
		dir = filepath.Join(home, ".ssh")
	}
	return runtime.OpenFileDialog(a.context(), runtime.OpenDialogOptions{
		Title:                "Choose SSH private key",
		DefaultDirectory:     dir,
		ShowHiddenFiles:      true,
		CanCreateDirectories: false,
	})
}

const TunnelEvent = "session:tunnel"

func (a *App) openOptions(sessionID string) db.OpenOptions {
	return db.OpenOptions{
		KnownHostsFile: knownHostsFile(),
		OnTunnelState: func(st sshtunnel.State, err error) {
			if a.ctx == nil || sessionID == "" {
				return
			}
			ev := model.TunnelState{SessionID: sessionID, State: string(st)}
			if err != nil {
				ev.Error = describe(err).Message
			}
			runtime.EventsEmit(a.ctx, TunnelEvent, ev)
		},
	}
}

func override(dst *string, typed string) {
	if typed != "" {
		*dst = typed
	}
}
