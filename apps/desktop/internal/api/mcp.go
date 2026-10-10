package api

import (
	"os"
	"path/filepath"
)

func (a *App) MCPExecutable() string {
	exe, err := os.Executable()
	if err != nil {
		return "plugboard"
	}
	if resolved, err := filepath.EvalSymlinks(exe); err == nil {
		return resolved
	}
	return exe
}

func (a *App) MCPLog() string {
	return filepath.Join(DataDir(), "logs", "mcp.log")
}
