package api

import (
	"context"
	"errors"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	goruntime "runtime"
	"strings"
	"sync"
	"time"

	"github.com/relay-client/relay-db/apps/desktop/internal/db"
	"github.com/relay-client/relay-db/apps/desktop/internal/model"
	"github.com/relay-client/relay-db/apps/desktop/internal/sshtunnel"
	"github.com/relay-client/relay-db/apps/desktop/internal/store"
	"github.com/relay-client/relay-db/apps/desktop/internal/update"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

var appVersion = "dev"

const (
	connectTimeout = 15 * time.Second
	catalogTimeout = 30 * time.Second
)

// App is the single object bound to the frontend. Every exported method shows
// up as window.go.api.App.<Method> in the webview.
type App struct {
	ctx         context.Context
	connections *store.Connections
	settings    *store.Settings
	menu        *appMenu

	mu       sync.Mutex
	sessions map[string]*db.Session
	queries  map[string]context.CancelFunc
	// changedKeys holds SSH host keys that differed from the remembered
	// ones, by host, until the user trusts them or not.
	changedKeys map[string]*sshtunnel.HostKeyChangedError
}

func NewApp() *App {
	dir := DataDir()
	return &App{
		connections: store.NewConnections(dir, store.NewSecrets(dir)),
		settings:    store.NewSettings(dir),
		sessions:    map[string]*db.Session{},
		queries:     map[string]context.CancelFunc{},
		changedKeys: map[string]*sshtunnel.HostKeyChangedError{},
	}
}

// DataDir is where Relay DB keeps its profile: ~/Library/Application Support/Relay DB on macOS.
func DataDir() string {
	if dir := os.Getenv("RELAYDB_DATA_DIR"); dir != "" {
		return dir
	}
	base, err := os.UserConfigDir()
	if err != nil {
		base = os.TempDir()
	}
	return filepath.Join(base, "Relay DB")
}

// knownHostsFile is Relay DB's own record of SSH host keys.
func knownHostsFile() string {
	return filepath.Join(DataDir(), "known_hosts")
}

func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx
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

// ShowWindow brings the window forward when the app is launched a second
// time. A plain function rather than a method, so Wails doesn't bind it.
func ShowWindow(a *App) {
	if a.ctx == nil {
		return
	}
	runtime.WindowUnminimise(a.ctx)
	runtime.Show(a.ctx)
}

func (a *App) AppInfo() model.AppInfo {
	return model.AppInfo{
		Name:      "Relay DB",
		Version:   appVersion,
		GoVersion: goruntime.Version(),
		Platform:  goruntime.GOOS + "/" + goruntime.GOARCH,
		DataDir:   DataDir(),
		Copyright: "© 2026 Relay Client",
	}
}

// devBuild reports a build that isn't a release — make dev, or make build
// between tags, which git describe versions like 0.2.0-3-gabc1234. Those
// never update themselves.
func devBuild() bool {
	v := strings.TrimSpace(appVersion)
	return v == "" || v == "dev" || strings.Contains(v, "-")
}

// CheckForUpdate looks for a newer release on GitHub.
func (a *App) CheckForUpdate() model.UpdateCheck {
	if devBuild() {
		return model.UpdateCheck{Error: "This is a development build (" + appVersion + "); it doesn't update itself."}
	}
	info, err := update.Check(a.context(), appVersion)
	if err != nil {
		return model.UpdateCheck{Error: updateError("check for updates", err)}
	}
	return model.UpdateCheck{Available: info}
}

// InstallUpdate downloads, verifies and installs the newest release, and
// returns its version. It takes effect when the app restarts (RestartApp).
func (a *App) InstallUpdate() (string, error) {
	if devBuild() {
		return "", errors.New("development builds don't update themselves")
	}
	version, err := update.Install(a.context(), appVersion)
	if err != nil {
		return "", errors.New(updateError("install the update", err))
	}
	return version, nil
}

// RestartApp quits and opens the app again, to switch to an installed update.
func (a *App) RestartApp() error {
	return update.Restart(func() { runtime.Quit(a.context()) })
}

// updateError says what went wrong in words the user can act on.
func updateError(action string, err error) string {
	var dnsErr *net.DNSError
	switch {
	case errors.As(err, &dnsErr), errors.Is(err, context.DeadlineExceeded):
		return "Could not " + action + ": GitHub can't be reached. Check the internet connection."
	case errors.Is(err, update.ErrChecksum), errors.Is(err, update.ErrSignature), errors.Is(err, update.ErrUntrustedURL), errors.Is(err, update.ErrBundleNotRelayDB):
		return "Could not " + action + ": the download failed verification (" + err.Error() + "), so nothing was changed."
	case errors.Is(err, os.ErrPermission):
		return "Could not " + action + ": Relay DB may not replace itself where it is installed. Download the new version from GitHub instead."
	}
	return "Could not " + action + ": " + err.Error()
}

func (a *App) GetSettings() model.Settings {
	return a.settings.Load()
}

func (a *App) SaveSettings(v model.Settings) (model.Settings, error) {
	saved, err := a.settings.Save(v)
	if err != nil {
		return saved, err
	}
	a.menu.setTheme(a.ctx, saved.Theme)
	return saved, nil
}

// OpenDataFolder shows the profile directory (connections, settings) in Finder.
func (a *App) OpenDataFolder() error {
	dir := DataDir()
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	var cmd *exec.Cmd
	switch goruntime.GOOS {
	case "darwin":
		cmd = exec.Command("open", dir)
	case "windows":
		cmd = exec.Command("explorer", dir)
	default:
		cmd = exec.Command("xdg-open", dir)
	}
	return cmd.Start()
}

func (a *App) ListConnections() ([]model.Connection, error) {
	return a.connections.List()
}

func (a *App) SaveConnection(c model.Connection) (model.Connection, error) {
	return a.connections.Save(c)
}

func (a *App) DeleteConnection(id string) error {
	return a.connections.Delete(id)
}

// TestConnection dials c without saving it. When the form is editing a saved
// profile and a secret field was left blank, the stored secret is used, as
// long as the profile still points at the same server.
func (a *App) TestConnection(c model.Connection) model.TestResult {
	if err := a.connections.FillSecrets(&c); err != nil {
		return model.TestResult{Error: err.Error()}
	}
	if c.Name == "" {
		c.Name = "test" // the form names unnamed profiles on save
	}
	if err := store.Validate(c); err != nil {
		return model.TestResult{Error: err.Error()}
	}
	ctx, cancel := context.WithTimeout(a.context(), connectTimeout)
	defer cancel()
	start := time.Now()
	s, err := db.Open(ctx, "test", c, a.openOptions(""))
	if err != nil {
		return model.TestResult{Error: err.Error()}
	}
	defer s.Close()
	return model.TestResult{
		Ok:            true,
		ServerVersion: s.Version,
		LatencyMs:     float64(time.Since(start).Microseconds()) / 1000,
	}
}

// Connect opens a saved profile. Non-empty secrets override stored ones. An
// SSH server whose key changed doesn't fail the call: the result carries the
// change for the user to look at, and TrustHostKey to accept it.
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

// TrustHostKey records the new key of an SSH host whose key changed, provided
// it is still the key with the fingerprint the user was shown.
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
	s, err := db.Open(ctx, id, c, a.openOptions(id))
	if err != nil {
		return model.SessionInfo{}, err
	}
	info, err := s.Info(ctx)
	if err != nil {
		s.Close()
		return model.SessionInfo{}, err
	}
	a.mu.Lock()
	a.sessions[s.ID] = s
	a.mu.Unlock()
	return info, nil
}

// SetReadOnly reopens an open session with read-only switched on or off. The
// server-side read-only mode is fixed per connection, so the pool (and the
// SQL editor's pinned connection, with any open transaction) is replaced.
func (a *App) SetReadOnly(sessionID string, readOnly bool) (model.SessionInfo, error) {
	old, err := a.session(sessionID)
	if err != nil {
		return model.SessionInfo{}, err
	}
	c := old.Conn
	c.ReadOnly = readOnly
	ctx, cancel := context.WithTimeout(a.context(), connectTimeout)
	defer cancel()
	s, err := db.Open(ctx, sessionID, c, a.openOptions(sessionID))
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
		// Disconnected or switched again while this one was opening.
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

func (a *App) ListTables(sessionID, schema string) ([]model.TableInfo, error) {
	s, err := a.session(sessionID)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(a.context(), catalogTimeout)
	defer cancel()
	return s.Tables(ctx, schema)
}

func (a *App) DescribeTable(sessionID, schema, table string) ([]model.Column, error) {
	s, err := a.session(sessionID)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(a.context(), catalogTimeout)
	defer cancel()
	return s.Columns(ctx, schema, table)
}

func (a *App) FetchTablePage(sessionID string, q model.TableQuery) (model.TablePage, error) {
	s, err := a.session(sessionID)
	if err != nil {
		return model.TablePage{}, err
	}
	ctx, cancel := context.WithTimeout(a.context(), catalogTimeout)
	defer cancel()
	return s.TablePage(ctx, q)
}

// CountRows returns a table's row count: the engine's estimate (instant), or
// the exact count when exact is set. queryID lets the UI cancel a slow count.
func (a *App) CountRows(sessionID, queryID string, q model.TableQuery, exact bool) (model.RowCount, error) {
	s, err := a.session(sessionID)
	if err != nil {
		return model.RowCount{}, err
	}
	ctx, done := a.cancellable(queryID, 5*time.Minute)
	defer done()
	return s.CountRows(ctx, q, exact)
}

// ApplyChanges commits edits from a table tab in one transaction.
func (a *App) ApplyChanges(sessionID string, cs model.ChangeSet) model.ApplyResult {
	s, err := a.session(sessionID)
	if err != nil {
		return model.ApplyResult{Error: err.Error(), FailedIndex: -1}
	}
	ctx, cancel := context.WithTimeout(a.context(), catalogTimeout)
	defer cancel()
	n, err := s.ApplyChanges(ctx, cs)
	if err != nil {
		res := model.ApplyResult{Error: err.Error(), FailedIndex: -1}
		var ae *db.ApplyError
		if errors.As(err, &ae) {
			res.FailedIndex = ae.Index
			res.Error = ae.Err.Error()
		}
		return res
	}
	return model.ApplyResult{Applied: n, FailedIndex: -1}
}

// PreviewChanges returns the SQL ApplyChanges would run, for display only.
func (a *App) PreviewChanges(sessionID string, cs model.ChangeSet) ([]string, error) {
	s, err := a.session(sessionID)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(a.context(), catalogTimeout)
	defer cancel()
	return s.PreviewChanges(ctx, cs)
}

// RunQuery executes a script. Results of the statements that succeeded are
// returned even when a later one fails, so the UI can show both.
func (a *App) RunQuery(sessionID, queryID, script string) model.QueryRun {
	s, err := a.session(sessionID)
	if err != nil {
		return model.QueryRun{Results: []model.ResultSet{}, Error: err.Error(), ErrorIndex: -1}
	}
	ctx, done := a.cancellable(queryID, 0)
	defer done()

	results, err := s.Run(ctx, script)
	run := model.QueryRun{Results: results, ErrorIndex: -1}
	if run.Results == nil {
		run.Results = []model.ResultSet{}
	}
	if err != nil {
		run.Error = err.Error()
		var se *db.StatementError
		if errors.As(err, &se) {
			run.ErrorIndex = se.Index
			run.RolledBack = se.RolledBack
		}
		run.Cancelled = errors.Is(ctx.Err(), context.Canceled)
	}
	return run
}

// WriteStatements lists the statements of script that may change data, for
// the editor to confirm before running them on Production. It is the same
// check a read-only session refuses scripts by.
func (a *App) WriteStatements(sessionID, script string) ([]string, error) {
	s, err := a.session(sessionID)
	if err != nil {
		return nil, err
	}
	return s.WriteStatements(script), nil
}

// RunMore fetches the next chunk of an editor result (see db.EditorChunk).
func (a *App) RunMore(sessionID, queryID, statement string, offset int) (model.ResultSet, error) {
	s, err := a.session(sessionID)
	if err != nil {
		return model.ResultSet{}, err
	}
	ctx, done := a.cancellable(queryID, 0)
	defer done()
	return s.RunMore(ctx, statement, offset)
}

// cancellable returns a context CancelQuery(queryID) can cancel, with a
// timeout unless it is 0, and the func to call when the query is over.
func (a *App) cancellable(queryID string, timeout time.Duration) (context.Context, func()) {
	var ctx context.Context
	var cancel context.CancelFunc
	if timeout > 0 {
		ctx, cancel = context.WithTimeout(a.context(), timeout)
	} else {
		ctx, cancel = context.WithCancel(a.context())
	}
	a.mu.Lock()
	a.queries[queryID] = cancel
	a.mu.Unlock()
	return ctx, func() {
		a.mu.Lock()
		delete(a.queries, queryID)
		a.mu.Unlock()
		cancel()
	}
}

func (a *App) CancelQuery(queryID string) {
	a.mu.Lock()
	cancel := a.queries[queryID]
	a.mu.Unlock()
	if cancel != nil {
		cancel()
	}
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

// TunnelEvent is emitted with a model.TunnelState when an open session's
// SSH tunnel drops or comes back.
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
				ev.Error = err.Error()
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

func (a *App) session(id string) (*db.Session, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	s, ok := a.sessions[id]
	if !ok {
		return nil, errConnectionClosed
	}
	return s, nil
}

var errConnectionClosed = errors.New("connection is closed")

func (a *App) context() context.Context {
	if a.ctx != nil {
		return a.ctx
	}
	return context.Background()
}
