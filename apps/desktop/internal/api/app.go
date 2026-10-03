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

	"github.com/relay-client/plugboard/apps/desktop/internal/apperr"
	"github.com/relay-client/plugboard/apps/desktop/internal/crash"
	"github.com/relay-client/plugboard/apps/desktop/internal/db"
	"github.com/relay-client/plugboard/apps/desktop/internal/db/engines"
	"github.com/relay-client/plugboard/apps/desktop/internal/model"
	"github.com/relay-client/plugboard/apps/desktop/internal/sshtunnel"
	"github.com/relay-client/plugboard/apps/desktop/internal/store"
	"github.com/relay-client/plugboard/apps/desktop/internal/update"
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

const oldDataDirName = "Relay DB"

func DataDir() string {
	if dir := os.Getenv("PLUGBOARD_DATA_DIR"); dir != "" {
		return dir
	}
	base, err := os.UserConfigDir()
	if err != nil {
		base = os.TempDir()
	}
	return filepath.Join(base, "Plugboard")
}

func knownHostsFile() string {
	return filepath.Join(DataDir(), "known_hosts")
}

func PrepareDataDir() string {
	dir := DataDir()
	if os.Getenv("PLUGBOARD_DATA_DIR") == "" {
		moveOldDataDir(dir)
	}
	return dir
}

func NoteCrash(a *App, path string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.crashLog = path
}

func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx
	moveOldSecrets(DataDir(), a.connections, store.OldSecrets())
}

func (a *App) LastCrash() string {
	a.mu.Lock()
	defer a.mu.Unlock()
	path := a.crashLog
	a.crashLog = ""
	return path
}

func (a *App) OpenLogs() error {
	return openFolder(crash.Dir(DataDir()))
}

func moveOldDataDir(dir string) bool {
	old := filepath.Join(filepath.Dir(dir), oldDataDirName)
	if _, err := os.Lstat(dir); !errors.Is(err, os.ErrNotExist) {
		return false
	}
	if info, err := os.Lstat(old); err != nil || !info.IsDir() {
		return false
	}
	return os.Rename(old, dir) == nil
}

func moveOldSecrets(dir string, connections *store.Connections, old store.Secrets) {
	if old == nil {
		return
	}
	done := filepath.Join(dir, ".secrets-moved")
	if _, err := os.Stat(done); err == nil {
		return
	}
	if err := connections.MoveSecrets(old); err != nil {
		return
	}
	if err := os.MkdirAll(dir, 0o700); err == nil {
		_ = os.WriteFile(done, nil, 0o600)
	}
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

func (a *App) AppInfo() model.AppInfo {
	return model.AppInfo{
		Name:      "Plugboard",
		Version:   appVersion,
		GoVersion: goruntime.Version(),
		Platform:  goruntime.GOOS + "/" + goruntime.GOARCH,
		DataDir:   DataDir(),
		Copyright: "© 2026 Relay Client",
	}
}

func devBuild() bool {
	v := strings.TrimSpace(appVersion)
	return v == "" || v == "dev" || strings.Contains(v, "-")
}

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

func (a *App) RestartApp() error {
	return update.Restart(func() { runtime.Quit(a.context()) })
}

func updateError(action string, err error) string {
	var dnsErr *net.DNSError
	switch {
	case errors.As(err, &dnsErr), errors.Is(err, context.DeadlineExceeded):
		return "Could not " + action + ": GitHub can't be reached. Check the internet connection."
	case errors.Is(err, update.ErrChecksum), errors.Is(err, update.ErrSignature), errors.Is(err, update.ErrUntrustedURL), errors.Is(err, update.ErrBundleNotPlugboard):
		return "Could not " + action + ": the download failed verification (" + err.Error() + "), so nothing was changed."
	case errors.Is(err, os.ErrPermission):
		return "Could not " + action + ": Plugboard may not replace itself where it is installed. Download the new version from GitHub instead."
	}
	return "Could not " + action + ": " + describe(err).Message
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

func (a *App) OpenDataFolder() error {
	return openFolder(DataDir())
}

func openFolder(dir string) error {
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

func (a *App) Diagram(sessionID, schema string) (model.Diagram, error) {
	r, err := a.relationReader(sessionID)
	if err != nil {
		return model.Diagram{}, err
	}
	ctx, cancel := context.WithTimeout(a.context(), catalogTimeout)
	defer cancel()
	return r.Diagram(ctx, schema)
}

func (a *App) Relations(sessionID, schema string) ([]model.Relation, error) {
	r, err := a.relationReader(sessionID)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(a.context(), catalogTimeout)
	defer cancel()
	return r.Relations(ctx, schema)
}

func (a *App) relationReader(id string) (db.RelationReader, error) {
	s, err := a.session(id)
	if err != nil {
		return nil, err
	}
	if r, ok := s.(db.RelationReader); ok {
		return r, nil
	}
	return nil, db.ErrNotSupported
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

func (a *App) CountRows(sessionID, queryID string, q model.TableQuery, exact bool) (model.RowCount, error) {
	s, err := a.session(sessionID)
	if err != nil {
		return model.RowCount{}, err
	}
	ctx, done := a.cancellable(queryID, 5*time.Minute)
	defer done()
	return s.CountRows(ctx, q, exact)
}

func (a *App) ApplyChanges(sessionID string, cs model.ChangeSet) model.ApplyResult {
	s, err := a.rowEditor(sessionID)
	if err != nil {
		return model.ApplyResult{Error: describe(err).Message, FailedIndex: -1}
	}
	ctx, cancel := context.WithTimeout(a.context(), catalogTimeout)
	defer cancel()
	n, err := s.ApplyChanges(ctx, cs)
	if err != nil {
		res := model.ApplyResult{Error: describe(err).Message, FailedIndex: -1}
		var ae *db.ApplyError
		if errors.As(err, &ae) {
			res.FailedIndex = ae.Index
			res.Error = describe(ae.Err).Message
		}
		return res
	}
	return model.ApplyResult{Applied: n, FailedIndex: -1}
}

func (a *App) ApplyStructure(sessionID, queryID string, sc model.StructureChange) model.ApplyResult {
	s, err := a.structureEditor(sessionID)
	if err != nil {
		return model.ApplyResult{Error: describe(err).Message, FailedIndex: -1}
	}
	ctx, done := a.cancellable(queryID, 0)
	defer done()
	n, partial, err := s.ApplyStructure(ctx, sc)
	if err != nil {
		res := model.ApplyResult{Applied: n, Partial: partial, Error: describe(err).Message, FailedIndex: -1, Cancelled: errors.Is(ctx.Err(), context.Canceled)}
		var ae *db.ApplyError
		if errors.As(err, &ae) {
			res.FailedIndex = ae.Index
			res.Error = describe(ae.Err).Message
		}
		return res
	}
	return model.ApplyResult{Applied: n, FailedIndex: -1}
}

func (a *App) PreviewStructure(sessionID string, sc model.StructureChange) ([]string, error) {
	s, err := a.structureEditor(sessionID)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(a.context(), catalogTimeout)
	defer cancel()
	return s.PreviewStructure(ctx, sc)
}

func (a *App) PreviewChanges(sessionID string, cs model.ChangeSet) ([]string, error) {
	s, err := a.rowEditor(sessionID)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(a.context(), catalogTimeout)
	defer cancel()
	return s.PreviewChanges(ctx, cs)
}

func (a *App) RunQuery(sessionID, queryID, script string) model.QueryRun {
	s, err := a.session(sessionID)
	if err != nil {
		return model.QueryRun{Results: []model.ResultSet{}, Error: describe(err).Message, ErrorIndex: -1, ErrorPosition: -1}
	}
	ctx, done := a.cancellable(queryID, 0)
	defer done()

	results, err := s.Run(ctx, script)
	run := model.QueryRun{Results: results, ErrorIndex: -1, ErrorPosition: -1}
	if run.Results == nil {
		run.Results = []model.ResultSet{}
	}
	if err != nil {
		run.Error = describe(err).Message
		var se *db.StatementError
		if errors.As(err, &se) {
			run.ErrorIndex = se.Index
			run.RolledBack = se.RolledBack
			run.ErrorPosition = se.Position
		}
		run.Cancelled = errors.Is(ctx.Err(), context.Canceled)
	}
	return run
}

func (a *App) CheckSyntax(sessionID, script string) ([]model.SyntaxProblem, error) {
	s, err := a.session(sessionID)
	if err != nil {
		return nil, err
	}
	checker, ok := s.(db.SyntaxChecker)
	if !ok {
		return []model.SyntaxProblem{}, nil
	}
	ctx, cancel := context.WithTimeout(a.context(), 5*time.Second)
	defer cancel()
	return checker.CheckSyntax(ctx, script)
}

func (a *App) WriteStatements(sessionID, script string) ([]string, error) {
	s, err := a.session(sessionID)
	if err != nil {
		return nil, err
	}
	return s.WriteStatements(script), nil
}

func (a *App) RunMore(sessionID, queryID, statement string, offset int) (model.ResultSet, error) {
	s, err := a.session(sessionID)
	if err != nil {
		return model.ResultSet{}, err
	}
	ctx, done := a.cancellable(queryID, 0)
	defer done()
	return s.RunMore(ctx, statement, offset)
}

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
