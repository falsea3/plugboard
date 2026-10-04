package api

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	goruntime "runtime"

	"github.com/relay-client/plugboard/apps/desktop/internal/crash"
	"github.com/relay-client/plugboard/apps/desktop/internal/store"
)

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
